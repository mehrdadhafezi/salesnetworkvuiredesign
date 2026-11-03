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

// Upload stages one receipt file. The bytes are made durable in quarantine
// storage BEFORE the upload row is written, so no row ever names bytes that
// do not exist. A storage failure records nothing (FAILED, safe to retry).
// Re-sending the same bytes returns the caller's existing upload.
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
	if existing, ok, err := s.findUpload(ctx, actor.UserID, sum[:]); err != nil {
		return UploadResult{}, err
	} else if ok && existing.State != api.UploadPurged {
		existing.Existing = true
		existing.Mode = mode
		return existing, nil
	}
	id, err := ids.NewUUID()
	if err != nil {
		return UploadResult{}, err
	}
	key := "quarantine/" + id
	// Durable bytes first. A purged upload of the same bytes is revived
	// under its original key.
	if existing, ok, err := s.findUpload(ctx, actor.UserID, sum[:]); err != nil {
		return UploadResult{}, err
	} else if ok {
		id, key = existing.UploadID, "quarantine/"+existing.UploadID
	}
	if err := s.store.Put(ctx, key, bytes.NewReader(body), sum); err != nil {
		return uploadRefusal(result.Failed, mode, reasoncodes.SourceUnavailable), nil
	}
	now := s.clock.Now().Time().UTC()
	expires := now.Add(UnboundTTL)
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE payments.upload_object SET state = 'UNBOUND', purged_at = NULL, created_at = $2, expires_at = $3,
			version = version + 1 WHERE upload_id = $1 AND state = 'PURGED'`, id, now, expires)
		if err != nil || tag.RowsAffected() == 1 {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO payments.upload_object (upload_id, uploader_user_id, content_sha256, size_bytes, media_type,
			quarantine_ref, state, created_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6, 'UNBOUND', $7, $8)`,
			id, actor.UserID, sum[:], len(body), media, key, now, expires)
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		// A concurrent identical upload won; ours is an orphan copy of the
		// same bytes and is removed (it was never evidence).
		_ = s.store.Delete(ctx, key)
		existing, ok, ferr := s.findUpload(ctx, actor.UserID, sum[:])
		if ferr != nil || !ok {
			return UploadResult{}, errors.Join(err, ferr)
		}
		existing.Existing, existing.Mode = true, mode
		return existing, nil
	}
	if err != nil {
		return UploadResult{}, err
	}
	return UploadResult{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, Mode: mode, UploadID: id,
		ContentSHA256: hexOf(sum[:]), SizeBytes: len(body), MediaType: media, State: api.UploadUnbound, ExpiresAt: &expires}, nil
}

func (s *Service) findUpload(ctx context.Context, uploader string, sum []byte) (UploadResult, bool, error) {
	var out UploadResult
	found := false
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		var hash []byte
		var exp time.Time
		err := tx.QueryRow(ctx, `SELECT upload_id::text, content_sha256, size_bytes, media_type, state, expires_at FROM payments.upload_object
			WHERE uploader_user_id = $1 AND content_sha256 = $2`, uploader, sum).Scan(&out.UploadID, &hash, &out.SizeBytes, &out.MediaType, &out.State, &exp)
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
