package command

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// UnboundTTL is how long an unbound upload is kept (D-44 safe default).
const UnboundTTL = 7 * 24 * time.Hour

// Accepted receipt media types (2.0.160 upload_single_receipt), decided by
// content sniffing, never by a client-supplied name or type.
var mediaTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "application/pdf": true}

// UploadResult is the outcome of Upload.
type UploadResult struct {
	Status        result.Status      `json:"-"`
	StatusName    string             `json:"status"`
	Reasons       []reasoncodes.Code `json:"reasons"`
	BlockedBy     []string           `json:"blocked_by,omitempty"`
	Mode          rollout.Mode       `json:"rollout_mode"`
	UploadID      string             `json:"upload_id,omitempty"`
	ContentSHA256 string             `json:"content_sha256,omitempty"`
	SizeBytes     int                `json:"size_bytes,omitempty"`
	MediaType     string             `json:"media_type,omitempty"`
	State         string             `json:"state,omitempty"`
	ExpiresAt     *time.Time         `json:"expires_at,omitempty"`
	Existing      bool               `json:"existing"`
}

func uploadRefusal(status result.Status, mode rollout.Mode, code reasoncodes.Code, blockedBy ...string) UploadResult {
	return UploadResult{Status: status, StatusName: status.String(), Reasons: []reasoncodes.Code{code}, BlockedBy: blockedBy, Mode: mode}
}

// modeRefusal answers the rollout gate shared by every payments write:
// only CONTROLLED_WRITEBACK accepts a payments write. SHADOW_COMPARE is not
// supported for payments (a target submission shares no identity with a
// legacy row; there is nothing to compare).
func modeRefusal(mode rollout.Mode) (result.Status, reasoncodes.Code, []string, bool) {
	switch {
	case mode == rollout.Controlled:
		return 0, "", nil, false
	case mode == rollout.Shadow:
		return result.Blocked, reasoncodes.PolicyUnresolved, []string{ShadowUnsupported}, true
	default:
		return result.Blocked, reasoncodes.FeatureDisabled, nil, true
	}
}

// Upload stages one receipt file. Invariants:
//
//   - every UNBOUND / BOUND upload row references bytes that are durable and
//     hash-verified: bytes are written (fsync + hash check) BEFORE any row
//     references them;
//   - every write goes to a FRESH object key owned by this request alone; a
//     key is adopted only by a conditional database change against the exact
//     state this request observed, so two concurrent requests can never
//     share a key;
//   - a request deletes only its own fresh copy, and only when the database
//     proved it was not adopted (a unique violation or a conditional update
//     that matched nothing). An ambiguous database error never deletes bytes;
//   - re-sending the same bytes keeps the upload's identity: an unexpired
//     UNBOUND / BOUND upload is returned as is; an expired UNBOUND one is
//     RENEWED (intact bytes verified) or RESTAGED (fresh bytes); an
//     EXPIRING / PURGED one is REVIVED with fresh bytes. An expired upload is
//     never handed back as usable, and renewal never waits for the cleanup
//     worker (D-44 safe default: unbound uploads live 7 days).
//
// A storage failure records nothing (FAILED, safe to retry).
func (s *Service) Upload(ctx context.Context, mode rollout.Mode, actor Actor, body []byte) (UploadResult, error) {
	if st, code, blocked, refused := modeRefusal(mode); refused {
		if mode != rollout.Shadow {
			mode = rollout.Disabled
		}
		return uploadRefusal(st, mode, code, blocked...), nil
	}
	if actor.UserID == "" || actor.Principal == "" {
		return uploadRefusal(result.Unauthorized, mode, reasoncodes.RowScopeDenied), nil
	}
	if len(body) == 0 || len(body) > MaxUploadBytes {
		return UploadResult{}, invalid("file size (1 byte to 5 MiB)")
	}
	media := http.DetectContentType(body)
	if i := bytes.IndexByte([]byte(media), ';'); i >= 0 {
		media = media[:i]
	}
	if !mediaTypes[media] {
		return UploadResult{}, invalid("file type (jpeg, png, webp or pdf by content)")
	}
	sum := sha256.Sum256(body)
	// A lost race is resolved by re-reading the winner's state; a bounded
	// number of rounds covers every interleaving of the finite lifecycle.
	for round := 0; round < 4; round++ {
		out, done, err := s.uploadRound(ctx, actor, body, media, sum)
		if err != nil || done {
			out.Mode = mode
			return out, err
		}
	}
	return uploadRefusal(result.Conflict, mode, reasoncodes.StaleState), nil
}

// uploadRound makes one attempt; done=false means a concurrent request
// changed the upload first and the caller must re-read.
func (s *Service) uploadRound(ctx context.Context, actor Actor, body []byte, media string, sum [32]byte) (UploadResult, bool, error) {
	existing, ok, err := s.findUpload(ctx, actor.UserID, sum[:])
	if err != nil {
		return UploadResult{}, true, err
	}
	now := s.clock.Now().Time().UTC()
	if ok {
		switch {
		case existing.State == api.UploadBound,
			existing.State == api.UploadUnbound && existing.ExpiresAt.After(now):
			existing.Existing = true
			return existing.UploadResult, true, nil
		case existing.State == api.UploadUnbound:
			// Expired before any cleanup run: renew under the same identity
			// when its bytes are intact (verified), else restage below.
			if info, err := s.store.Stat(ctx, existing.ref); err == nil && info.SHA256 == sum {
				renewed, applied, err := s.adopt(ctx, actor, existing, existing.ref, "RENEWED", now)
				if err != nil || !applied {
					return UploadResult{}, err != nil, err
				}
				return renewed, true, nil
			}
		}
	}
	// Fresh key, owned by this request alone until a conditional change adopts it.
	id, err := ids.NewUUID()
	if err != nil {
		return UploadResult{}, true, err
	}
	fresh := "quarantine/" + id
	if err := s.store.Put(ctx, fresh, bytes.NewReader(body), sum); err != nil {
		return uploadRefusal(result.Failed, "", reasoncodes.SourceUnavailable), true, nil
	}
	if !ok {
		out, err := s.insertUpload(ctx, actor, id, fresh, len(body), media, sum[:], now)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// The insert rolled back: our copy is provably unreferenced.
			_ = s.store.Delete(ctx, fresh)
			return UploadResult{}, false, nil
		}
		return out, true, err // an ambiguous error keeps the bytes
	}
	kind := "REVIVED"
	if existing.State == api.UploadUnbound {
		kind = "RESTAGED"
	}
	out, applied, err := s.adopt(ctx, actor, existing, fresh, kind, now)
	if err != nil {
		return UploadResult{}, true, err // ambiguous: never delete
	}
	if !applied {
		// Lost the race: the conditional update matched nothing, so our
		// copy was never referenced. The winner's key is untouched.
		_ = s.store.Delete(ctx, fresh)
		return UploadResult{}, false, nil
	}
	if kind == "RESTAGED" && existing.ref != fresh {
		// The replaced key held missing / mismatched bytes and is no longer
		// referenced by anything (an unbound upload has no receipt).
		_ = s.store.Delete(ctx, existing.ref)
	}
	return out, true, nil
}

func (s *Service) insertUpload(ctx context.Context, actor Actor, id, key string, size int, media string, sum []byte, now time.Time) (UploadResult, error) {
	expires := now.Add(UnboundTTL)
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO payments.upload_object (upload_id, uploader_user_id, content_sha256, size_bytes, media_type,
			quarantine_ref, state, created_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6, 'UNBOUND', $7, $8)`,
			id, actor.UserID, sum, size, media, key, now, expires); err != nil {
			return err
		}
		return uploadEvent(ctx, tx, id, "UPLOADED", key, "", actor.Principal, now)
	})
	if err != nil {
		return UploadResult{}, err
	}
	return UploadResult{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, UploadID: id,
		ContentSHA256: hexOf(sum), SizeBytes: size, MediaType: media, State: api.UploadUnbound, ExpiresAt: &expires}, nil
}

// adopt makes key the upload's object and starts a new unbound lifetime,
// conditionally on the exact state and key observed (optimistic): applied is
// false when a concurrent request or the expiry worker changed it first.
func (s *Service) adopt(ctx context.Context, actor Actor, u foundUpload, key, kind string, now time.Time) (UploadResult, bool, error) {
	expires := now.Add(UnboundTTL)
	applied := false
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE payments.upload_object SET state = 'UNBOUND', quarantine_ref = $4, expires_at = $5, purged_at = NULL,
			expiring_at = NULL, version = version + 1 WHERE upload_id = $1 AND state = $2 AND quarantine_ref = $3`,
			u.UploadID, u.State, u.ref, key, expires)
		if err != nil || tag.RowsAffected() != 1 {
			return err
		}
		applied = true
		return uploadEvent(ctx, tx, u.UploadID, kind, key, "from "+u.State, actor.Principal, now)
	})
	if err != nil || !applied {
		return UploadResult{}, false, err
	}
	out := u.UploadResult
	out.State, out.ExpiresAt, out.Existing = api.UploadUnbound, &expires, true
	return out, true, nil
}

func uploadEvent(ctx context.Context, tx db.Tx, uploadID, kind, key, detail, actor string, at time.Time) error {
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	var d *string
	if detail != "" {
		t := truncate300(detail)
		d = &t
	}
	_, err = tx.Exec(ctx, `INSERT INTO payments.upload_event (event_id, upload_id, kind, quarantine_ref, detail, actor_principal, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, uploadID, kind, key, d, actor, at)
	return err
}

// foundUpload is an upload as read, with the object key it references.
type foundUpload struct {
	UploadResult
	ref string
}

func (s *Service) findUpload(ctx context.Context, uploader string, sum []byte) (foundUpload, bool, error) {
	var out foundUpload
	found := false
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		var hash []byte
		var exp time.Time
		err := tx.QueryRow(ctx, `SELECT upload_id::text, content_sha256, size_bytes, media_type, state, expires_at, quarantine_ref
			FROM payments.upload_object WHERE uploader_user_id = $1 AND content_sha256 = $2`, uploader, sum).
			Scan(&out.UploadID, &hash, &out.SizeBytes, &out.MediaType, &out.State, &exp, &out.ref)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		out.ContentSHA256 = hexOf(hash)
		e := exp.UTC()
		out.ExpiresAt = &e
		return nil
	})
	out.Status, out.StatusName, out.Reasons = result.Success, result.Success.String(), []reasoncodes.Code{}
	return out, found, err
}
