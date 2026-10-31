// Package store holds the customers module's SQL. It is private to customers
// (architecture lint: foreign-store). Write functions require a writable
// customers-owned transaction (import path only); read functions require a
// read-only one. There is no delete and no customer-to-customer rewrite.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by customers.
	ErrWrongOwner = errors.New("customers/store: transaction is not owned by customers")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("customers/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("customers/store: write path requires a writable transaction")
)

func writable(tx db.Tx) error {
	if tx.Module() != customers.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != customers.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises customers import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.customers.import'))`)
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

// UpsertCustomer mints (once) or refreshes the customer of one subject. A
// changed source row refreshes the same customer; it never forks one.
func UpsertCustomer(ctx context.Context, tx db.Tx, s api.SubjectRecord, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	var (
		id, batch, presence string
		sha                 []byte
	)
	err := tx.QueryRow(ctx, `SELECT customer_id::text, source_row_sha256, source_batch_id::text, presence FROM customers.customer
		WHERE source_system = $1 AND subject_kind = $2 AND source_key = $3 FOR UPDATE`,
		s.Source.System, string(s.Kind), s.Source.Key).Scan(&id, &sha, &batch, &presence)
	if errors.Is(err, pgx.ErrNoRows) {
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO customers.customer (customer_id, subject_kind, source_system, source_spec, source_key,
			first_batch_id, source_batch_id, source_row_sha256, imported_at, refreshed_at, presence, name_evidence, observed_raw,
			recoverable, resolution_state, resolution_reason, version)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $8, 'PRESENT', $9, $10, $11, 'UNKNOWN', 'not yet resolved', 1)`,
			id, string(s.Kind), s.Source.System, s.Source.Spec, s.Source.Key, s.Source.BatchID, s.Source.RowSHA256, now,
			s.Name, s.ObservedRaw, s.Recoverable)
		return id, Inserted, err
	}
	if err != nil {
		return "", "", err
	}
	switch {
	case !bytes.Equal(sha, s.Source.RowSHA256):
		_, err = tx.Exec(ctx, `UPDATE customers.customer SET source_spec = $2, source_batch_id = $3, source_row_sha256 = $4,
			refreshed_at = $5, presence = 'PRESENT', name_evidence = $6, observed_raw = $7, recoverable = $8, version = version + 1
			WHERE customer_id = $1`, id, s.Source.Spec, s.Source.BatchID, s.Source.RowSHA256, now, s.Name, s.ObservedRaw, s.Recoverable)
		return id, Refreshed, err
	case presence != "PRESENT":
		_, err = tx.Exec(ctx, `UPDATE customers.customer SET presence = 'PRESENT', source_batch_id = $2, refreshed_at = $3,
			version = version + 1 WHERE customer_id = $1`, id, s.Source.BatchID, now)
		return id, Reappeared, err
	case batch != s.Source.BatchID:
		_, err = tx.Exec(ctx, `UPDATE customers.customer SET source_batch_id = $2, refreshed_at = $3 WHERE customer_id = $1`, id, s.Source.BatchID, now)
		return id, Rebatched, err
	}
	return id, Unchanged, nil
}

// MarkAbsentCustomers flags customers of a COMPLETE source missing from it.
func MarkAbsentCustomers(ctx context.Context, tx db.Tx, system string, kind api.SubjectKind, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if present == nil {
		present = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.customer SET presence = 'ABSENT_IN_LATEST', refreshed_at = $4, version = version + 1
		WHERE source_system = $1 AND subject_kind = $2 AND presence = 'PRESENT' AND NOT (source_key = ANY($3))`,
		system, string(kind), present, now)
	return tag.RowsAffected(), err
}

// ContactInput is one normalized contact ready to store.
type ContactInput struct {
	CustomerID    string
	Contact       api.ContactRecord
	Normalized    *string
	Normalization string
	ObservedRaw   *string
	Source        api.SourceRef
}

// UpsertContact stores one contact point; returns its id and whether it changed.
func UpsertContact(ctx context.Context, tx db.Tx, c ContactInput, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", false, err
	}
	var gotID string
	var inserted bool
	err = tx.QueryRow(ctx, `INSERT INTO customers.contact_point (contact_point_id, customer_id, kind, role, raw_value, normalized_value,
		normalization, valid_from_raw, valid_to_raw, observed_raw, source_system, source_spec, source_key, first_batch_id, source_batch_id,
		imported_at, refreshed_at, presence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14, $15, $15, 'PRESENT')
		ON CONFLICT (customer_id, kind, role, raw_value) DO UPDATE SET
			valid_from_raw = EXCLUDED.valid_from_raw, valid_to_raw = EXCLUDED.valid_to_raw, observed_raw = EXCLUDED.observed_raw,
			source_spec = EXCLUDED.source_spec, source_batch_id = EXCLUDED.source_batch_id, refreshed_at = EXCLUDED.refreshed_at,
			presence = 'PRESENT'
		WHERE (customers.contact_point.valid_from_raw, customers.contact_point.valid_to_raw, customers.contact_point.observed_raw,
			customers.contact_point.source_batch_id, customers.contact_point.presence)
			IS DISTINCT FROM (EXCLUDED.valid_from_raw, EXCLUDED.valid_to_raw, EXCLUDED.observed_raw, EXCLUDED.source_batch_id, 'PRESENT')
		RETURNING contact_point_id::text, (xmax = 0)`,
		id, c.CustomerID, string(c.Contact.Kind), string(c.Contact.Role), c.Contact.Raw, c.Normalized, c.Normalization,
		c.Contact.ValidFromRaw, c.Contact.ValidToRaw, c.ObservedRaw, c.Source.System, c.Source.Spec, c.Source.Key, c.Source.BatchID, now).
		Scan(&gotID, &inserted)
	if errors.Is(err, pgx.ErrNoRows) { // identical row: nothing to update
		err = tx.QueryRow(ctx, `SELECT contact_point_id::text FROM customers.contact_point
			WHERE customer_id = $1 AND kind = $2 AND role = $3 AND raw_value = $4`,
			c.CustomerID, string(c.Contact.Kind), string(c.Contact.Role), c.Contact.Raw).Scan(&gotID)
		return gotID, false, err
	}
	return gotID, err == nil, err
}

// UpsertIdentifier stores one identifier claim of a customer.
func UpsertIdentifier(ctx context.Context, tx db.Tx, customerID string, idr api.IdentifierRecord, src api.SourceRef, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", false, err
	}
	var gotID string
	err = tx.QueryRow(ctx, `INSERT INTO customers.customer_identifier (identifier_id, customer_id, scheme, external_value, derivation,
		status, source_system, source_spec, source_key, first_batch_id, source_batch_id, imported_at, refreshed_at, presence)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6, $7, $8, $9, $9, $10, $10, 'PRESENT')
		ON CONFLICT (customer_id, scheme, external_value) DO UPDATE SET
			derivation = EXCLUDED.derivation, source_spec = EXCLUDED.source_spec, source_batch_id = EXCLUDED.source_batch_id,
			refreshed_at = EXCLUDED.refreshed_at, presence = 'PRESENT'
		WHERE (customers.customer_identifier.derivation, customers.customer_identifier.source_batch_id, customers.customer_identifier.presence)
			IS DISTINCT FROM (EXCLUDED.derivation, EXCLUDED.source_batch_id, 'PRESENT')
		RETURNING identifier_id::text`,
		id, customerID, string(idr.Scheme), idr.Value, string(idr.Derivation), src.System, src.Spec, src.Key, src.BatchID, now).Scan(&gotID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT identifier_id::text FROM customers.customer_identifier
			WHERE customer_id = $1 AND scheme = $2 AND external_value = $3`, customerID, string(idr.Scheme), idr.Value).Scan(&gotID)
		return gotID, false, err
	}
	return gotID, err == nil, err
}

// MarkAbsentChildren flags contacts / identifiers of the given customers
// that the latest record of those customers no longer carries.
func MarkAbsentChildren(ctx context.Context, tx db.Tx, customers, contacts, identifiers []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	for _, s := range []*[]string{&customers, &contacts, &identifiers} {
		if *s == nil {
			*s = []string{}
		}
	}
	a, err := tx.Exec(ctx, `UPDATE customers.contact_point SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3
		WHERE customer_id = ANY($1::uuid[]) AND presence = 'PRESENT' AND NOT (contact_point_id = ANY($2::uuid[]))`, customers, contacts, now)
	if err != nil {
		return 0, err
	}
	b, err := tx.Exec(ctx, `UPDATE customers.customer_identifier SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3
		WHERE customer_id = ANY($1::uuid[]) AND presence = 'PRESENT' AND NOT (identifier_id = ANY($2::uuid[]))`, customers, identifiers, now)
	return a.RowsAffected() + b.RowsAffected(), err
}

// --- derivation input ------------------------------------------------------------------

// Snapshot is the present model the derivation works on.
type Snapshot struct {
	Customers   map[string]bool // customer id → recoverable
	Contacts    []Contact
	Identifiers []Identifier
}

// Contact is one present contact of a present customer.
type Contact struct {
	ID, CustomerID, Kind string
	Normalized           *string
	ValidFrom, ValidTo   *string
	Raw                  string
}

// Identifier is one present identifier of a present customer.
type Identifier struct {
	ID, CustomerID, Scheme, Value, Derivation string
}

// LoadSnapshot reads the present customers, contacts and identifiers.
func LoadSnapshot(ctx context.Context, tx db.Tx) (Snapshot, error) {
	s := Snapshot{Customers: map[string]bool{}}
	rows, err := tx.Query(ctx, `SELECT customer_id::text, recoverable FROM customers.customer WHERE presence = 'PRESENT'`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var id string
		var rec bool
		if err := rows.Scan(&id, &rec); err != nil {
			rows.Close()
			return s, err
		}
		s.Customers[id] = rec
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT c.contact_point_id::text, c.customer_id::text, c.kind, c.normalized_value, c.valid_from_raw, c.valid_to_raw, c.raw_value
		FROM customers.contact_point c JOIN customers.customer k USING (customer_id)
		WHERE c.presence = 'PRESENT' AND k.presence = 'PRESENT' ORDER BY c.contact_point_id`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.CustomerID, &c.Kind, &c.Normalized, &c.ValidFrom, &c.ValidTo, &c.Raw); err != nil {
			rows.Close()
			return s, err
		}
		s.Contacts = append(s.Contacts, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT i.identifier_id::text, i.customer_id::text, i.scheme, i.external_value, i.derivation
		FROM customers.customer_identifier i JOIN customers.customer k USING (customer_id)
		WHERE i.presence = 'PRESENT' AND k.presence = 'PRESENT' ORDER BY i.identifier_id`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var i Identifier
		if err := rows.Scan(&i.ID, &i.CustomerID, &i.Scheme, &i.Value, &i.Derivation); err != nil {
			return s, err
		}
		s.Identifiers = append(s.Identifiers, i)
	}
	return s, rows.Err()
}

// SetIdentifierStatus marks identifiers ACTIVE or CONFLICT.
func SetIdentifierStatus(ctx context.Context, tx db.Tx, conflicting []string) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if conflicting == nil {
		conflicting = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.customer_identifier SET status = CASE WHEN identifier_id = ANY($1::uuid[]) THEN 'CONFLICT' ELSE 'ACTIVE' END
		WHERE status <> CASE WHEN identifier_id = ANY($1::uuid[]) THEN 'CONFLICT' ELSE 'ACTIVE' END`, conflicting)
	return tag.RowsAffected(), err
}

// --- evidence (append-only) -------------------------------------------------------------

// EvidenceInput is one evidence item to record.
type EvidenceInput struct {
	Key         []byte
	Kind        string
	Strength    string
	Polarity    string
	Origin      string // DERIVED | SUPPLIED
	Reference   string
	ObservedRaw *string
	System      string
	Spec        string
	SourceKey   *string
	Subjects    []string // customer ids
	// Contact is the normalized contact of a supplied OTP record (internal).
	Contact *string
}

// RecordEvidence appends evidence once (by key) and marks it seen; existing
// content is never rewritten. It returns the id and whether it was new or
// reappeared.
func RecordEvidence(ctx context.Context, tx db.Tx, e EvidenceInput, batchID string, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO customers.identity_evidence (evidence_id, evidence_key, kind, strength, polarity, origin, reference,
		observed_raw, source_system, source_spec, source_key, first_batch_id, recorded_at, last_seen_batch_id, last_seen_at, presence, contact_normalized)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $12, $13, 'PRESENT', $14)
		ON CONFLICT (evidence_key) DO NOTHING`,
		id, e.Key, e.Kind, e.Strength, e.Polarity, e.Origin, e.Reference, e.ObservedRaw, e.System, e.Spec, e.SourceKey, batchID, now, e.Contact)
	if err != nil {
		return "", false, err
	}
	if tag.RowsAffected() == 1 {
		for _, s := range e.Subjects {
			if _, err := tx.Exec(ctx, `INSERT INTO customers.evidence_subject (evidence_id, customer_id) VALUES ($1, $2)`, id, s); err != nil {
				return "", false, err
			}
		}
		return id, true, nil
	}
	var presence string
	if err := tx.QueryRow(ctx, `SELECT evidence_id::text, presence FROM customers.identity_evidence WHERE evidence_key = $1`, e.Key).Scan(&id, &presence); err != nil {
		return "", false, err
	}
	_, err = tx.Exec(ctx, `UPDATE customers.identity_evidence SET last_seen_batch_id = $2, last_seen_at = $3, presence = 'PRESENT'
		WHERE evidence_id = $1 AND (last_seen_batch_id, presence) IS DISTINCT FROM ($2::uuid, 'PRESENT')`, id, batchID, now)
	return id, presence != "PRESENT", err
}

// MarkAbsentSupplied flags supplied evidence of the given source specs that
// a COMPLETE command did not carry. Specs the command does not carry are
// left untouched: omission from another source is not withdrawal.
func MarkAbsentSupplied(ctx context.Context, tx db.Tx, system string, specs, seen []string) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if len(specs) == 0 {
		return 0, nil
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.identity_evidence SET presence = 'ABSENT_IN_LATEST'
		WHERE origin = 'SUPPLIED' AND presence = 'PRESENT' AND source_system = $1 AND source_spec = ANY($2::text[])
		AND NOT (evidence_id = ANY($3::uuid[]))`, system, specs, seen)
	return tag.RowsAffected(), err
}

// SuppliedRow is one PRESENT supplied evidence record with its subjects.
type SuppliedRow struct {
	ID, Kind, System, Spec string
	SourceKey              *string
	Contact                *string
	Subjects               []string // customer ids, sorted
}

// PresentSupplied returns every PRESENT supplied evidence record, so the
// case projection never depends on which records one command carried.
func PresentSupplied(ctx context.Context, tx db.Tx) ([]SuppliedRow, error) {
	rows, err := tx.Query(ctx, `SELECT e.evidence_id::text, e.kind, e.source_system, e.source_spec, e.source_key, e.contact_normalized,
		array_agg(s.customer_id::text ORDER BY s.customer_id::text)
		FROM customers.identity_evidence e JOIN customers.evidence_subject s USING (evidence_id)
		WHERE e.origin = 'SUPPLIED' AND e.presence = 'PRESENT'
		GROUP BY e.evidence_id ORDER BY e.evidence_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SuppliedRow
	for rows.Next() {
		var r SuppliedRow
		if err := rows.Scan(&r.ID, &r.Kind, &r.System, &r.Spec, &r.SourceKey, &r.Contact, &r.Subjects); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkAbsentEvidence flags evidence of an origin not seen in this run.
func MarkAbsentEvidence(ctx context.Context, tx db.Tx, origin string, seen []string) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.identity_evidence SET presence = 'ABSENT_IN_LATEST'
		WHERE origin = $1 AND presence = 'PRESENT' AND NOT (evidence_id = ANY($2::uuid[]))`, origin, seen)
	return tag.RowsAffected(), err
}

// --- merge cases (queue projection) --------------------------------------------------------

// CaseInput is one derived candidate case.
type CaseInput struct {
	Key         []byte
	Reason      string
	State       string
	StateReason string
	MaxStrength string
	Members     []string
	Evidence    []string
}

// UpsertCase records a case, its membership and evidence links. State
// changes append a history row. It returns whether anything changed.
func UpsertCase(ctx context.Context, tx db.Tx, c CaseInput, runID, batchID string, now time.Time) (string, bool, error) {
	if err := writable(tx); err != nil {
		return "", false, err
	}
	var (
		id, state, presence string
		changed             bool
	)
	err := tx.QueryRow(ctx, `SELECT merge_case_id::text, state, presence FROM customers.merge_case WHERE case_key = $1 FOR UPDATE`, c.Key).
		Scan(&id, &state, &presence)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if id, err = ids.NewUUID(); err != nil {
			return "", false, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO customers.merge_case (merge_case_id, case_key, reason, state, state_reason, max_strength,
			first_batch_id, source_batch_id, created_at, refreshed_at, presence, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $8, 'PRESENT', 1)`,
			id, c.Key, c.Reason, c.State, c.StateReason, c.MaxStrength, batchID, now); err != nil {
			return "", false, err
		}
		changed = true
	case err != nil:
		return "", false, err
	default:
		tag, err := tx.Exec(ctx, `UPDATE customers.merge_case SET state = $2, state_reason = $3, max_strength = $4, source_batch_id = $5,
			refreshed_at = $6, presence = 'PRESENT', version = version + 1
			WHERE merge_case_id = $1 AND (state, state_reason, max_strength, presence) IS DISTINCT FROM ($2, $3, $4, 'PRESENT')`,
			id, c.State, c.StateReason, c.MaxStrength, batchID, now)
		if err != nil {
			return "", false, err
		}
		changed = tag.RowsAffected() > 0
	}
	if changed && (state != c.State || presence != "PRESENT") {
		if _, err := tx.Exec(ctx, `INSERT INTO customers.merge_case_state_history (merge_case_id, run_id, state, state_reason, recorded_at)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, id, runID, c.State, c.StateReason, now); err != nil {
			return "", false, err
		}
	}
	for _, link := range []struct {
		table, col string
		ids        []string
	}{{"customers.merge_case_member", "customer_id", c.Members}, {"customers.merge_case_evidence", "evidence_id", c.Evidence}} {
		ids := link.ids
		if ids == nil {
			ids = []string{}
		}
		tag, err := tx.Exec(ctx, `INSERT INTO `+link.table+` (merge_case_id, `+link.col+`, presence)
			SELECT $1, x, 'PRESENT' FROM unnest($2::uuid[]) AS x
			ON CONFLICT (merge_case_id, `+link.col+`) DO UPDATE SET presence = 'PRESENT' WHERE `+link.table+`.presence <> 'PRESENT'`, id, ids)
		if err != nil {
			return "", false, err
		}
		gone, err := tx.Exec(ctx, `UPDATE `+link.table+` SET presence = 'ABSENT_IN_LATEST'
			WHERE merge_case_id = $1 AND presence = 'PRESENT' AND NOT (`+link.col+` = ANY($2::uuid[]))`, id, ids)
		if err != nil {
			return "", false, err
		}
		changed = changed || tag.RowsAffected() > 0 || gone.RowsAffected() > 0
	}
	return id, changed, nil
}

// MarkAbsentCases flags cases not derived in this run.
func MarkAbsentCases(ctx context.Context, tx db.Tx, seen []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	if seen == nil {
		seen = []string{}
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.merge_case SET presence = 'ABSENT_IN_LATEST', refreshed_at = $2, version = version + 1
		WHERE presence = 'PRESENT' AND NOT (merge_case_id = ANY($1::uuid[]))`, seen, now)
	return tag.RowsAffected(), err
}

// SetCustomerState updates one customer's projection, appending history
// when it changes. The customer row itself is never removed or redirected.
func SetCustomerState(ctx context.Context, tx db.Tx, customerID, state, reason, runID string, now time.Time) (bool, error) {
	if err := writable(tx); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `UPDATE customers.customer SET resolution_state = $2, resolution_reason = $3
		WHERE customer_id = $1 AND (resolution_state, resolution_reason) IS DISTINCT FROM ($2, $3)`, customerID, state, reason)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO customers.customer_resolution_history (customer_id, run_id, state, reason, recorded_at)
		VALUES ($1, $2, $3, $4, $5)`, customerID, runID, state, reason, now)
	return true, err
}

// PriorRun reports whether batchID was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM customers.import_run WHERE batch_id = $1`, batchID).Scan(&n)
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
	_, err = tx.Exec(ctx, `INSERT INTO customers.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}
