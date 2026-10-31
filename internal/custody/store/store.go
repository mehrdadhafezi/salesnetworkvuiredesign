// Package store holds the custody module's SQL. It is private to custody
// (architecture lint: foreign-store). Write functions require a writable
// custody-owned transaction (import path only); read functions require a
// read-only one. CustodyEvent rows are INSERT-only; there is no UPDATE or
// DELETE of an event anywhere (and the database rejects both).
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by custody.
	ErrWrongOwner = errors.New("custody/store: transaction is not owned by custody")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("custody/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("custody/store: write path requires a writable transaction")
	// ErrNotFound reports an unknown case.
	ErrNotFound = errors.New("custody/store: not found")
)

func writable(tx db.Tx) error {
	if tx.Module() != custody.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != custody.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises custody import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.custody.import'))`)
	return err
}

func key(parts ...string) []byte {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return h[:]
}

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func ref(u *api.UserRef) (any, any) {
	if u == nil {
		return nil, nil
	}
	return u.LegacyKey, str(u.UserID)
}

// AppendEvent inserts one custody event once. A changed legacy row with the
// same source key is never rewritten: it is reported as a conflict.
func AppendEvent(ctx context.Context, tx db.Tx, caseID string, e api.EventRecord, now time.Time) (inserted, conflict bool, err error) {
	if err := writable(tx); err != nil {
		return false, false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return false, false, err
	}
	ak, au := ref(e.Actor)
	fk, fu := ref(e.From)
	tk, tu := ref(e.To)
	tag, err := tx.Exec(ctx, `INSERT INTO custody.custody_event (event_id, case_id, source_system, source_spec, source_key, source_ordinal,
		legacy_action, kind, context, actor_legacy_key, actor_user_id, from_legacy_key, from_user_id, to_legacy_key, to_user_id,
		from_position_raw, to_position_raw, step_order_raw, occurred_raw, row_sha256, first_batch_id, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $22::numeric, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (source_system, source_spec, source_key) DO NOTHING`,
		id, caseID, e.Source.System, e.Source.Spec, e.Source.Key, e.Action, e.Kind, e.Context, ak, au, fk, fu, tk, tu,
		str(e.FromPositionRaw), str(e.ToPositionRaw), str(e.StepOrderRaw), str(e.OccurredRaw), e.Source.RowSHA256, e.Source.BatchID, now, e.Source.Key)
	if err != nil {
		return false, false, err
	}
	if tag.RowsAffected() == 1 {
		return true, false, nil
	}
	var sha []byte
	if err := tx.QueryRow(ctx, `SELECT row_sha256 FROM custody.custody_event WHERE source_system = $1 AND source_spec = $2 AND source_key = $3`,
		e.Source.System, e.Source.Spec, e.Source.Key).Scan(&sha); err != nil {
		return false, false, err
	}
	return false, !bytes.Equal(sha, e.Source.RowSHA256), nil
}

// EventsOf loads a case's events for derivation, with the user ids the
// events carried for each legacy key.
func EventsOf(ctx context.Context, tx db.Tx, caseID string) ([]domain.Event, map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT event_id::text, source_ordinal::text, kind, legacy_action, from_legacy_key, to_legacy_key,
		to_position_raw, occurred_raw, to_user_id::text FROM custody.custody_event WHERE case_id = $1 ORDER BY source_ordinal`, caseID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []domain.Event
	users := map[string]string{}
	for rows.Next() {
		var e domain.Event
		var toUser *string
		if err := rows.Scan(&e.ID, &e.Ordinal, &e.Kind, &e.Action, &e.From, &e.To, &e.ToPosition, &e.OccurredRaw, &toUser); err != nil {
			return nil, nil, err
		}
		if e.To != nil && toUser != nil {
			users[*e.To] = *toUser
		}
		out = append(out, e)
	}
	return out, users, rows.Err()
}

// AssignmentInput is the derived projection of one case.
type AssignmentInput struct {
	CaseID                    string
	Derivation                string
	DerivedKey, DerivedUserID *string
	DerivedPosition           *string
	LegacyKey, LegacyUserID   *string
	LegacyPosition            *string
	LegacySpec                string
	Parity                    string
	EventCount                int
	LastEventID               *string
}

// UpsertAssignment refreshes the derived projection of one case.
func UpsertAssignment(ctx context.Context, tx db.Tx, a AssignmentInput, batchID string, now time.Time) (bool, error) {
	if err := writable(tx); err != nil {
		return false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO custody.assignment (assignment_id, case_id, derivation, derived_legacy_key, derived_user_id,
		derived_position_raw, legacy_holder_key, legacy_holder_user_id, legacy_position_raw, legacy_source_spec, parity, event_count,
		last_event_id, presence, refreshed_batch_id, refreshed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'PRESENT', $14, $15)
		ON CONFLICT (case_id) DO UPDATE SET derivation = EXCLUDED.derivation, derived_legacy_key = EXCLUDED.derived_legacy_key,
		derived_user_id = EXCLUDED.derived_user_id, derived_position_raw = EXCLUDED.derived_position_raw,
		legacy_holder_key = EXCLUDED.legacy_holder_key, legacy_holder_user_id = EXCLUDED.legacy_holder_user_id,
		legacy_position_raw = EXCLUDED.legacy_position_raw, legacy_source_spec = EXCLUDED.legacy_source_spec, parity = EXCLUDED.parity,
		event_count = EXCLUDED.event_count, last_event_id = EXCLUDED.last_event_id, presence = 'PRESENT',
		refreshed_batch_id = EXCLUDED.refreshed_batch_id, refreshed_at = EXCLUDED.refreshed_at
		WHERE (custody.assignment.derivation, custody.assignment.derived_legacy_key, custody.assignment.derived_user_id,
			custody.assignment.derived_position_raw, custody.assignment.legacy_holder_key, custody.assignment.legacy_holder_user_id,
			custody.assignment.legacy_position_raw, custody.assignment.parity, custody.assignment.event_count,
			custody.assignment.last_event_id, custody.assignment.presence)
		IS DISTINCT FROM (EXCLUDED.derivation, EXCLUDED.derived_legacy_key, EXCLUDED.derived_user_id, EXCLUDED.derived_position_raw,
			EXCLUDED.legacy_holder_key, EXCLUDED.legacy_holder_user_id, EXCLUDED.legacy_position_raw, EXCLUDED.parity,
			EXCLUDED.event_count, EXCLUDED.last_event_id, 'PRESENT')`,
		id, a.CaseID, a.Derivation, str(a.DerivedKey), str(a.DerivedUserID), str(a.DerivedPosition), str(a.LegacyKey), str(a.LegacyUserID),
		str(a.LegacyPosition), a.LegacySpec, a.Parity, a.EventCount, str(a.LastEventID), batchID, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// MarkAbsentAssignments flags projections of a COMPLETE holder spec whose
// source row is gone. Events and findings history stay.
func MarkAbsentAssignments(ctx context.Context, tx db.Tx, spec string, seen []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE custody.assignment SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3
		WHERE legacy_source_spec = $1 AND presence = 'PRESENT' AND NOT (case_id = ANY($2::uuid[]))`, spec, seen, now)
	return tag.RowsAffected(), err
}

// RecordFinding inserts a finding once (by content key) and marks it seen.
func RecordFinding(ctx context.Context, tx db.Tx, caseID string, f domain.Finding, batchID string, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", false, err
	}
	k := key("FINDING", caseID, f.Kind, f.EventID, f.Detail)
	var ev any
	if f.EventID != "" {
		ev = f.EventID
	}
	tag, err := tx.Exec(ctx, `INSERT INTO custody.finding (finding_id, finding_key, case_id, kind, detail, event_id, presence,
		first_batch_id, last_seen_batch_id, recorded_at) VALUES ($1, $2, $3, $4, $5, $6, 'PRESENT', $7, $7, $8)
		ON CONFLICT (finding_key) DO NOTHING`, id, k, caseID, f.Kind, f.Detail, ev, batchID, now)
	if err != nil {
		return "", false, err
	}
	if tag.RowsAffected() == 1 {
		return id, true, nil
	}
	var presence string
	if err := tx.QueryRow(ctx, `SELECT finding_id::text, presence FROM custody.finding WHERE finding_key = $1`, k).Scan(&id, &presence); err != nil {
		return "", false, err
	}
	_, err = tx.Exec(ctx, `UPDATE custody.finding SET presence = 'PRESENT', last_seen_batch_id = $2 WHERE finding_id = $1
		AND (presence, last_seen_batch_id) IS DISTINCT FROM ('PRESENT', $2::uuid)`, id, batchID)
	return id, presence != "PRESENT", err
}

// MarkAbsentFindings flags findings of the processed cases not seen now.
func MarkAbsentFindings(ctx context.Context, tx db.Tx, cases, seen []string) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if cases == nil {
		cases = []string{}
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE custody.finding SET presence = 'ABSENT_IN_LATEST' WHERE presence = 'PRESENT'
		AND case_id = ANY($1::uuid[]) AND NOT (finding_id = ANY($2::uuid[]))`, cases, seen)
	return tag.RowsAffected(), err
}

// PriorRun reports whether a batch was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM custody.import_run WHERE batch_id = $1`, batchID).Scan(&n)
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
	_, err = tx.Exec(ctx, `INSERT INTO custody.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}
