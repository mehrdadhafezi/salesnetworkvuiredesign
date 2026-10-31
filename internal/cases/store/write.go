// Package store holds the cases module's SQL. It is private to cases
// (architecture lint: foreign-store). Write functions require a writable
// cases-owned transaction (import path only); read functions require a
// read-only one. There is no delete, no Case-to-Case rewrite and no UPDATE
// of state history or attribution snapshots.
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by cases.
	ErrWrongOwner = errors.New("cases/store: transaction is not owned by cases")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("cases/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("cases/store: write path requires a writable transaction")
)

func writable(tx db.Tx) error {
	if tx.Module() != cases.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != cases.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises cases import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.cases.import'))`)
	return err
}

// Key hashes parts into a 32-byte deterministic key.
func Key(parts ...string) []byte {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return h[:]
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

// UpsertCase mints (once) or refreshes the Case of one source row. A changed
// source row refreshes the same Case (mutable projection); it never forks.
func UpsertCase(ctx context.Context, tx db.Tx, c api.CaseRecord, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	var id, batch, presence string
	var sha []byte
	err := tx.QueryRow(ctx, `SELECT s.case_id::text, s.last_batch_id::text, s.row_sha256, s.presence FROM cases.case_source s
		WHERE s.source_system = $1 AND s.source_spec = $2 AND s.source_key = $3 FOR UPDATE`,
		c.Source.System, c.Source.Spec, c.Source.Key).Scan(&id, &batch, &sha, &presence)
	link, linkState, linkReason := any(nil), string(api.Unknown), "no explicit customer reference in the staged source"
	if c.Customer != nil {
		link, linkState, linkReason = c.Customer.CustomerID, string(c.Customer.State), c.Customer.Reason
	}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cases.work_case (case_id, workflow_type, state_namespace, state_raw, state_mapping,
			customer_id, customer_link_state, customer_link_reason, presence, imported_at, refreshed_at, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PRESENT', $9, $9, 1)`,
			id, string(c.Workflow), c.StateNamespace, str(c.StateRaw), string(c.StateMapping), link, linkState, linkReason, now); err != nil {
			return "", "", err
		}
		sid, err := ids.NewUUID()
		if err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cases.case_source (case_source_id, case_id, source_system, source_spec, source_key,
			channel_raw, origin_ref_raw, first_batch_id, last_batch_id, row_sha256, presence, first_seen_at, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9, 'PRESENT', $10, $10)`,
			sid, id, c.Source.System, c.Source.Spec, c.Source.Key, str(c.ChannelRaw), str(c.OriginRefRaw), c.Source.BatchID, c.Source.RowSHA256, now); err != nil {
			return "", "", err
		}
		if err := upsertLead(ctx, tx, id, c.Lead, now); err != nil {
			return "", "", err
		}
		return id, Inserted, nil
	case err != nil:
		return "", "", err
	}
	if !bytes.Equal(sha, c.Source.RowSHA256) {
		if _, err := tx.Exec(ctx, `UPDATE cases.work_case SET state_namespace = $2, state_raw = $3, state_mapping = $4, customer_id = $5,
			customer_link_state = $6, customer_link_reason = $7, presence = 'PRESENT', refreshed_at = $8, version = version + 1 WHERE case_id = $1`,
			id, c.StateNamespace, str(c.StateRaw), string(c.StateMapping), link, linkState, linkReason, now); err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE cases.case_source SET channel_raw = $2, origin_ref_raw = $3, last_batch_id = $4, row_sha256 = $5,
			presence = 'PRESENT', last_seen_at = $6 WHERE case_id = $1`,
			id, str(c.ChannelRaw), str(c.OriginRefRaw), c.Source.BatchID, c.Source.RowSHA256, now); err != nil {
			return "", "", err
		}
		return id, Refreshed, upsertLead(ctx, tx, id, c.Lead, now)
	}
	if presence != "PRESENT" {
		if _, err := tx.Exec(ctx, `UPDATE cases.work_case SET presence = 'PRESENT', refreshed_at = $2 WHERE case_id = $1`, id, now); err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE cases.lead SET presence = 'PRESENT' WHERE case_id = $1`, id); err != nil {
			return "", "", err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE cases.case_source SET last_batch_id = $2, presence = 'PRESENT', last_seen_at = $3 WHERE case_id = $1
		AND (last_batch_id, presence) IS DISTINCT FROM ($2::uuid, 'PRESENT')`, id, c.Source.BatchID, now); err != nil {
		return "", "", err
	}
	switch {
	case presence != "PRESENT":
		return id, Reappeared, nil
	case batch != c.Source.BatchID:
		return id, Rebatched, nil
	}
	return id, Unchanged, nil
}

func upsertLead(ctx context.Context, tx db.Tx, caseID string, l *api.LeadFacts, _ time.Time) error {
	if l == nil {
		return nil
	}
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO cases.lead (lead_id, case_id, status_raw, lead_status_raw, destination_panel_raw,
		imported_at_raw, assigned_at_raw, updated_at_raw, presence) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PRESENT')
		ON CONFLICT (case_id) DO UPDATE SET status_raw = EXCLUDED.status_raw, lead_status_raw = EXCLUDED.lead_status_raw,
		destination_panel_raw = EXCLUDED.destination_panel_raw, imported_at_raw = EXCLUDED.imported_at_raw,
		assigned_at_raw = EXCLUDED.assigned_at_raw, updated_at_raw = EXCLUDED.updated_at_raw, presence = 'PRESENT'`,
		id, caseID, str(l.StatusRaw), str(l.LeadStatusRaw), str(l.DestinationPanelRaw), str(l.ImportedAtRaw), str(l.AssignedAtRaw), str(l.UpdatedAtRaw))
	return err
}

// MarkAbsentCases flags Cases of a COMPLETE spec whose source row is gone.
// Nothing is deleted; history and snapshots stay.
func MarkAbsentCases(ctx context.Context, tx db.Tx, system, spec string, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if present == nil {
		present = []string{}
	}
	rows, err := tx.Query(ctx, `UPDATE cases.case_source SET presence = 'ABSENT_IN_LATEST', last_seen_at = $4
		WHERE source_system = $1 AND source_spec = $2 AND presence = 'PRESENT' AND NOT (source_key = ANY($3::text[]))
		RETURNING case_id::text`, system, spec, present, now)
	if err != nil {
		return 0, err
	}
	var gone []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		gone = append(gone, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(gone) == 0 {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE cases.work_case SET presence = 'ABSENT_IN_LATEST', refreshed_at = $2 WHERE case_id = ANY($1::uuid[])`, gone, now); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE cases.lead SET presence = 'ABSENT_IN_LATEST' WHERE case_id = ANY($1::uuid[])`, gone); err != nil {
		return 0, err
	}
	return int64(len(gone)), nil
}

// CaseOf returns the Case id of one source row (any presence).
func CaseOf(ctx context.Context, tx db.Tx, system, spec, key string) (string, bool, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT case_id::text FROM cases.case_source WHERE source_system = $1 AND source_spec = $2 AND source_key = $3`,
		system, spec, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

// AppendHistory records one legacy history row once. A changed row with the
// same source key is never rewritten: it is reported as a conflict.
func AppendHistory(ctx context.Context, tx db.Tx, caseID string, h api.HistoryRecord, now time.Time) (inserted, conflict bool, err error) {
	if err := writable(tx); err != nil {
		return false, false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return false, false, err
	}
	var actorKey, actorUser any
	if h.Actor != nil {
		actorKey, actorUser = h.Actor.LegacyKey, str(h.Actor.UserID)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO cases.case_state_history (history_id, case_id, source_system, source_spec, source_key,
		source_ordinal, old_state_raw, new_state_raw, actor_legacy_key, actor_user_id, occurred_raw, row_sha256, first_batch_id, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $14::numeric, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (source_system, source_spec, source_key) DO NOTHING`,
		id, caseID, h.Source.System, h.Source.Spec, h.Source.Key, str(h.OldRaw), str(h.NewRaw), actorKey, actorUser, str(h.OccurredRaw),
		h.Source.RowSHA256, h.Source.BatchID, now, h.Source.Key)
	if err != nil {
		return false, false, err
	}
	if tag.RowsAffected() == 1 {
		return true, false, nil
	}
	var sha []byte
	if err := tx.QueryRow(ctx, `SELECT row_sha256 FROM cases.case_state_history WHERE source_system = $1 AND source_spec = $2 AND source_key = $3`,
		h.Source.System, h.Source.Spec, h.Source.Key).Scan(&sha); err != nil {
		return false, false, err
	}
	return false, !bytes.Equal(sha, h.Source.RowSHA256), nil
}

// AliasInput is one resolved alias row.
type AliasInput struct {
	CaseID, RelatedSpec, RelatedKey string
	RelatedCaseID                   *string
	Relation                        string
	Resolution                      string
	Basis                           string
}

// UpsertAlias records an explicit link once and keeps its target current.
func UpsertAlias(ctx context.Context, tx db.Tx, a AliasInput, batchID string, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", false, err
	}
	var rel any
	if a.RelatedCaseID != nil {
		rel = *a.RelatedCaseID
	}
	tag, err := tx.Exec(ctx, `INSERT INTO cases.case_alias (alias_id, case_id, related_case_id, related_source_spec, related_source_key,
		relation, resolution, basis, first_batch_id, recorded_at, presence) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'PRESENT')
		ON CONFLICT (case_id, relation, related_source_spec, related_source_key) DO NOTHING`,
		id, a.CaseID, rel, a.RelatedSpec, a.RelatedKey, a.Relation, a.Resolution, a.Basis, batchID, now)
	if err != nil {
		return "", false, err
	}
	if tag.RowsAffected() == 1 {
		return id, true, nil
	}
	if err := tx.QueryRow(ctx, `SELECT alias_id::text FROM cases.case_alias WHERE case_id = $1 AND relation = $2
		AND related_source_spec = $3 AND related_source_key = $4`, a.CaseID, a.Relation, a.RelatedSpec, a.RelatedKey).Scan(&id); err != nil {
		return "", false, err
	}
	tag, err = tx.Exec(ctx, `UPDATE cases.case_alias SET related_case_id = $2, resolution = $3, basis = $4, presence = 'PRESENT'
		WHERE alias_id = $1 AND (related_case_id, resolution, basis, presence) IS DISTINCT FROM ($2::uuid, $3, $4, 'PRESENT')`,
		id, rel, a.Resolution, a.Basis)
	if err != nil {
		return "", false, err
	}
	return id, tag.RowsAffected() > 0, nil
}

// MarkAbsentAliases flags aliases from cases of a COMPLETE spec not seen.
func MarkAbsentAliases(ctx context.Context, tx db.Tx, system, spec string, seen []string) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE cases.case_alias a SET presence = 'ABSENT_IN_LATEST' FROM cases.case_source s
		WHERE s.case_id = a.case_id AND s.source_system = $1 AND s.source_spec = $2 AND a.presence = 'PRESENT'
		AND NOT (a.alias_id = ANY($3::uuid[]))`, system, spec, seen)
	return tag.RowsAffected(), err
}

// SnapshotInput is one attribution snapshot to record.
type SnapshotInput struct {
	CaseID        string
	Facet         string
	Resolution    string
	SubjectKey    *string
	SubjectUserID *string
	PositionRaw   *string
	Basis         string
	SourceTimeRaw *string
	EvidenceSpec  *string
	EvidenceKey   *string
}

// RecordSnapshot inserts an immutable snapshot once per distinct evidence.
// Identical evidence is a no-op; different evidence is a new row. Existing
// snapshots are never updated (no grant; trigger-enforced).
func RecordSnapshot(ctx context.Context, tx db.Tx, s SnapshotInput, batchID string, now time.Time) (bool, error) {
	if err := writable(tx); err != nil {
		return false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return false, err
	}
	v := func(p *string) string {
		if p == nil {
			return "\x00"
		}
		return *p
	}
	key := Key("ATTRIBUTION", s.CaseID, s.Facet, s.Resolution, v(s.SubjectKey), v(s.SubjectUserID), v(s.PositionRaw), s.Basis,
		v(s.SourceTimeRaw), v(s.EvidenceSpec), v(s.EvidenceKey))
	tag, err := tx.Exec(ctx, `INSERT INTO cases.attribution_snapshot (snapshot_id, snapshot_key, case_id, facet, resolution,
		subject_legacy_key, subject_user_id, position_raw, basis, source_time_raw, evidence_spec, evidence_key, captured_batch_id, captured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (snapshot_key) DO NOTHING`,
		id, key, s.CaseID, s.Facet, s.Resolution, str(s.SubjectKey), str(s.SubjectUserID), str(s.PositionRaw), s.Basis,
		str(s.SourceTimeRaw), str(s.EvidenceSpec), str(s.EvidenceKey), batchID, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// PriorRun reports whether a batch was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM cases.import_run WHERE batch_id = $1`, batchID).Scan(&n)
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
	_, err = tx.Exec(ctx, `INSERT INTO cases.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}
