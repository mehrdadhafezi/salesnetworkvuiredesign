// Package store holds the payments module's SQL. It is private to payments
// (architecture lint: foreign-store). Write functions require a writable
// payments-owned transaction (import path only); read functions require a
// read-only one. There is no delete, no billing write and no UPDATE of
// attempt_state_history.
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by payments.
	ErrWrongOwner = errors.New("payments/store: transaction is not owned by payments")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("payments/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("payments/store: write path requires a writable transaction")
	// ErrNotFound reports an unknown attempt or receipt.
	ErrNotFound = errors.New("payments/store: not found")
)

func writable(tx db.Tx) error {
	if tx.Module() != payments.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != payments.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises payments import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.payments.import'))`)
	return err
}

// Change classifies what an upsert did.
type Change string

const (
	Inserted   Change = "inserted"
	Refreshed  Change = "refreshed"
	Reappeared Change = "reappeared"
	Rebatched  Change = "rebatched"
	Unchanged  Change = "unchanged"
)

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func moneyCols(m api.RawMoney) (raw, amount, unit any, repr string) {
	a := domain.ParseAmount(m.Raw, m.Unit)
	raw = str(m.Raw)
	if a.Representation == domain.Representable {
		return raw, a.Money.Amount, a.Money.Unit.String(), a.Representation
	}
	return raw, nil, nil, a.Representation
}

func mapping(raw, target *string) (string, any) {
	switch {
	case raw == nil || *raw == "":
		return "EMPTY", nil
	case target != nil:
		return "MAPPED", *target
	}
	return "UNKNOWN", nil
}

// UpsertAttempt mints (once) or refreshes one attempt. A changed raw state
// of an existing attempt appends to attempt_state_history (regression =
// it was ACCEPTED and is no longer); the change is never silent.
func UpsertAttempt(ctx context.Context, tx db.Tx, invoiceID *string, a api.AttemptRecord, now time.Time) (string, Change, bool, error) {
	if err := writable(tx); err != nil {
		return "", "", false, err
	}
	raw, amount, unit, repr := moneyCols(a.Amount)
	stMap, stTarget := mapping(a.StateRaw, a.StateTarget)
	link := "UNKNOWN"
	if invoiceID != nil {
		link = "RESOLVED"
	}
	var upKey, upUser any
	if a.UploadedBy != nil {
		upKey, upUser = a.UploadedBy.LegacyKey, str(a.UploadedBy.UserID)
	}
	var id, batch, presence string
	var sha []byte
	var oldRaw, oldTarget *string
	err := tx.QueryRow(ctx, `SELECT attempt_id::text, last_batch_id::text, row_sha256, presence, state_raw, state_target FROM payments.payment_attempt
		WHERE source_system = $1 AND source_spec = $2 AND source_key = $3 FOR UPDATE`, a.Source.System, a.Source.Spec, a.Source.Key).
		Scan(&id, &batch, &sha, &presence, &oldRaw, &oldTarget)
	vals := []any{str(invoiceID), link, str(a.StageNoRaw), a.Channel, str(a.PayMethodRaw), str(a.PaymentSourceRaw), raw, amount, unit, repr,
		a.StateNamespace, str(a.StateRaw), stMap, stTarget, a.ApprovalProvenance, str(a.UploadedByTypeRaw), upKey, upUser,
		str(a.CreatedAtRaw), str(a.UpdatedAtRaw)}
	regression := false
	ch := Unchanged
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if id, err = ids.NewUUID(); err != nil {
			return "", "", false, err
		}
		args := append([]any{id, a.Source.System, a.Source.Spec, a.Source.Key, a.Source.BatchID, a.Source.RowSHA256, now}, vals...)
		if _, err := tx.Exec(ctx, `INSERT INTO payments.payment_attempt (attempt_id, source_system, source_spec, source_key, first_batch_id,
			last_batch_id, row_sha256, first_seen_at, last_seen_at, presence, invoice_id, invoice_link_state, stage_no_raw, channel, pay_method_raw,
			payment_source_raw, amount_raw, amount, unit, amount_repr, state_namespace, state_raw, state_mapping, state_target,
			approval_provenance, uploaded_by_type_raw, uploaded_by_legacy_key, uploaded_by_user_id, created_at_raw, updated_at_raw, version)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $7, 'PRESENT', $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, 1)`, args...); err != nil {
			return "", "", false, err
		}
		ch = Inserted
	case err != nil:
		return "", "", false, err
	case !bytes.Equal(sha, a.Source.RowSHA256):
		args := append([]any{id, a.Source.BatchID, a.Source.RowSHA256, now}, vals...)
		if _, err := tx.Exec(ctx, `UPDATE payments.payment_attempt SET last_batch_id = $2, row_sha256 = $3, last_seen_at = $4, presence = 'PRESENT',
			invoice_id = $5, invoice_link_state = $6, stage_no_raw = $7, channel = $8, pay_method_raw = $9, payment_source_raw = $10,
			amount_raw = $11, amount = $12, unit = $13, amount_repr = $14, state_namespace = $15, state_raw = $16, state_mapping = $17,
			state_target = $18, approval_provenance = $19, uploaded_by_type_raw = $20, uploaded_by_legacy_key = $21, uploaded_by_user_id = $22,
			created_at_raw = $23, updated_at_raw = $24, version = version + 1 WHERE attempt_id = $1`, args...); err != nil {
			return "", "", false, err
		}
		if !eq(oldRaw, a.StateRaw) {
			regression = oldTarget != nil && *oldTarget == "ACCEPTED" && (a.StateTarget == nil || *a.StateTarget != "ACCEPTED")
			hid, err := ids.NewUUID()
			if err != nil {
				return "", "", false, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO payments.attempt_state_history (history_id, attempt_id, old_state_raw, new_state_raw, regression,
				batch_id, recorded_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`, hid, id, str(oldRaw), str(a.StateRaw), regression, a.Source.BatchID, now); err != nil {
				return "", "", false, err
			}
		}
		ch = Refreshed
	default:
		// Unchanged source row: refresh only the externally resolved fields
		// (invoice link via the billing API, uploader user id via identity),
		// so a later replay repairs links resolved since the last import.
		tag, err := tx.Exec(ctx, `UPDATE payments.payment_attempt SET invoice_id = $2, invoice_link_state = $3, uploaded_by_user_id = $4,
			last_batch_id = $5, last_seen_at = $6, presence = 'PRESENT', version = version + 1
			WHERE attempt_id = $1 AND (invoice_id IS DISTINCT FROM $2::uuid OR uploaded_by_user_id IS DISTINCT FROM $4::uuid)`,
			id, str(invoiceID), link, upUser, a.Source.BatchID, now)
		if err != nil {
			return "", "", false, err
		}
		if tag.RowsAffected() > 0 {
			ch = Refreshed
			break
		}
		if _, err := tx.Exec(ctx, `UPDATE payments.payment_attempt SET last_batch_id = $2, last_seen_at = $3, presence = 'PRESENT'
			WHERE attempt_id = $1 AND (last_batch_id, presence) IS DISTINCT FROM ($2::uuid, 'PRESENT')`, id, a.Source.BatchID, now); err != nil {
			return "", "", false, err
		}
		switch {
		case presence != "PRESENT":
			ch = Reappeared
		case batch != a.Source.BatchID:
			ch = Rebatched
		}
	}
	if g := a.Gateway; g != nil && (ch == Inserted || ch == Refreshed) {
		gMap, gTarget := mapping(g.StateRaw, g.StateTarget)
		gid, err := ids.NewUUID()
		if err != nil {
			return "", "", false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO payments.gateway_attempt (gateway_attempt_id, attempt_id, provider_raw, authority_sha256, ref_sha256,
			state_raw, state_mapping, state_target) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (attempt_id) DO UPDATE SET provider_raw = EXCLUDED.provider_raw, authority_sha256 = EXCLUDED.authority_sha256,
			ref_sha256 = EXCLUDED.ref_sha256, state_raw = EXCLUDED.state_raw, state_mapping = EXCLUDED.state_mapping, state_target = EXCLUDED.state_target`,
			gid, id, str(g.ProviderRaw), g.AuthoritySHA256, g.RefSHA256, str(g.StateRaw), gMap, gTarget); err != nil {
			return "", "", false, err
		}
	}
	return id, ch, regression, nil
}

func eq(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ReceiptKey is the identity of one receipt reference.
func ReceiptKey(r api.ReceiptRecord) []byte {
	h := sha256.Sum256([]byte(r.Source.System + "\x1f" + r.Source.Spec + "\x1f" + r.Source.Key + "\x1f" + strconv.Itoa(r.RefIndex)))
	return h[:]
}

// UpsertReceipt records one receipt reference once and keeps it current.
func UpsertReceipt(ctx context.Context, tx db.Tx, invoiceID *string, r api.ReceiptRecord, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	key := ReceiptKey(r)
	var id, batch, presence string
	err := tx.QueryRow(ctx, `SELECT receipt_id::text, last_batch_id::text, presence FROM payments.receipt WHERE receipt_key = $1 FOR UPDATE`, key).
		Scan(&id, &batch, &presence)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO payments.receipt (receipt_id, receipt_key, invoice_id, stage_no_raw, owner_kind, source_system, source_spec,
			source_key, ref_index, ref_sha256, host, extension, state, first_batch_id, last_batch_id, first_seen_at, last_seen_at, presence)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14, $15, $15, 'PRESENT')`,
			id, key, str(invoiceID), str(r.StageNoRaw), r.OwnerKind, r.Source.System, r.Source.Spec, r.Source.Key, r.RefIndex, r.RefSHA256,
			str(r.Host), str(r.Extension), r.State, r.Source.BatchID, now)
		return id, Inserted, err
	case err != nil:
		return "", "", err
	}
	// The projection is refreshed when the reference OR its ownership
	// (invoice link, stage, owner kind) or metadata changed: an unchanged
	// reference hash is not an unchanged receipt.
	tag, err := tx.Exec(ctx, `UPDATE payments.receipt SET invoice_id = $2, stage_no_raw = $3, ref_sha256 = $4, host = $5, extension = $6, state = $7,
		owner_kind = $10, last_batch_id = $8, last_seen_at = $9, presence = 'PRESENT' WHERE receipt_id = $1
		AND (invoice_id IS DISTINCT FROM $2::uuid OR stage_no_raw IS DISTINCT FROM $3 OR ref_sha256 IS DISTINCT FROM $4 OR host IS DISTINCT FROM $5
		     OR extension IS DISTINCT FROM $6 OR state IS DISTINCT FROM $7 OR owner_kind IS DISTINCT FROM $10)`,
		id, str(invoiceID), str(r.StageNoRaw), r.RefSHA256, str(r.Host), str(r.Extension), r.State, r.Source.BatchID, now, r.OwnerKind)
	if err != nil {
		return "", "", err
	}
	if tag.RowsAffected() > 0 {
		return id, Refreshed, nil
	}
	_, err = tx.Exec(ctx, `UPDATE payments.receipt SET last_batch_id = $2, last_seen_at = $3, presence = 'PRESENT' WHERE receipt_id = $1
		AND (last_batch_id, presence) IS DISTINCT FROM ($2::uuid, 'PRESENT')`, id, r.Source.BatchID, now)
	switch {
	case presence != "PRESENT":
		return id, Reappeared, err
	case batch != r.Source.BatchID:
		return id, Rebatched, err
	}
	return id, Unchanged, err
}

// MarkAbsentAttempts flags attempts of a COMPLETE spec whose row is gone.
func MarkAbsentAttempts(ctx context.Context, tx db.Tx, system, spec string, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if present == nil {
		present = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE payments.payment_attempt SET presence = 'ABSENT_IN_LATEST', last_seen_at = $4
		WHERE source_system = $1 AND source_spec = $2 AND presence = 'PRESENT' AND NOT (source_key = ANY($3::text[]))`, system, spec, present, now)
	return tag.RowsAffected(), err
}

// MarkAbsentReceipts flags receipt references not seen in a COMPLETE batch.
func MarkAbsentReceipts(ctx context.Context, tx db.Tx, system string, seen []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE payments.receipt SET presence = 'ABSENT_IN_LATEST', last_seen_at = $3
		WHERE source_system = $1 AND presence = 'PRESENT' AND NOT (receipt_id = ANY($2::uuid[]))`, system, seen, now)
	return tag.RowsAffected(), err
}

// PriorRun reports whether a batch was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM payments.import_run WHERE batch_id = $1`, batchID).Scan(&n)
	return n > 0, err
}

// InsertRun records one import run.
func InsertRun(ctx context.Context, tx db.Tx, runID, batchID, snapshotID, outcome string, replay bool, counts map[string]int64, now time.Time) error {
	if err := writable(tx); err != nil {
		return err
	}
	b, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO payments.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}

// AttemptSpecs lists the attempt source specs known for a source system.
func AttemptSpecs(ctx context.Context, tx db.Tx, system string) ([]string, error) {
	if err := writable(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT source_spec FROM payments.payment_attempt WHERE source_system = $1 ORDER BY 1`, system)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
