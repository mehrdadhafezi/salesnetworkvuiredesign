package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Provenance is read-back provenance.
type Provenance struct {
	System, Spec, Key, BatchID string
	ImportedAt, RefreshedAt    time.Time
	Presence                   string
}

// Customer is one customer read model. Contact and identifier values are
// summarised; raw values never leave the store unmasked.
type Customer struct {
	CustomerID      string
	SubjectKind     string
	State           string
	StateReason     string
	ObservedRaw     *string
	NameMasked      *string
	ContactCount    int64
	IdentifierCount int64
	EvidenceCount   int64
	OpenCaseCount   int64
	ConflictCases   int64
	Source          Provenance
}

const customerSelect = `SELECT c.customer_id::text, c.subject_kind, c.resolution_state, c.resolution_reason, c.observed_raw, c.name_evidence,
	(SELECT count(*) FROM customers.contact_point p WHERE p.customer_id = c.customer_id AND p.presence = 'PRESENT'),
	(SELECT count(*) FROM customers.customer_identifier i WHERE i.customer_id = c.customer_id AND i.presence = 'PRESENT'),
	(SELECT count(*) FROM customers.evidence_subject s JOIN customers.identity_evidence e USING (evidence_id)
		WHERE s.customer_id = c.customer_id AND e.presence = 'PRESENT'),
	(SELECT count(*) FROM customers.merge_case_member m JOIN customers.merge_case k USING (merge_case_id)
		WHERE m.customer_id = c.customer_id AND m.presence = 'PRESENT' AND k.presence = 'PRESENT'),
	(SELECT count(*) FROM customers.merge_case_member m JOIN customers.merge_case k USING (merge_case_id)
		WHERE m.customer_id = c.customer_id AND m.presence = 'PRESENT' AND k.presence = 'PRESENT' AND k.state = 'CONFLICT'),
	c.source_system, c.source_spec, c.source_key, c.source_batch_id::text, c.imported_at, c.refreshed_at, c.presence
	FROM customers.customer c`

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	var name *string
	err := row.Scan(&c.CustomerID, &c.SubjectKind, &c.State, &c.StateReason, &c.ObservedRaw, &name, &c.ContactCount,
		&c.IdentifierCount, &c.EvidenceCount, &c.OpenCaseCount, &c.ConflictCases, &c.Source.System, &c.Source.Spec, &c.Source.Key,
		&c.Source.BatchID, &c.Source.ImportedAt, &c.Source.RefreshedAt, &c.Source.Presence)
	if name != nil {
		m := domain.Mask(*name)
		c.NameMasked = &m
	}
	c.Source.ImportedAt, c.Source.RefreshedAt = c.Source.ImportedAt.UTC(), c.Source.RefreshedAt.UTC()
	return c, err
}

// ResolutionStates is the allowlist of the state filter.
var ResolutionStates = map[string]bool{domain.Resolved: true, domain.Related: true, domain.Unknown: true, domain.Conflict: true, domain.NotRecoverable: true}

// ListCustomers returns one keyset page ordered by customer id.
func ListCustomers(ctx context.Context, tx db.Tx, state *string, hasConflict *bool, presence *string, after *string, limit int) ([]Customer, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, customerSelect+`
		WHERE ($1::text IS NULL OR c.resolution_state = $1)
		  AND ($2::boolean IS NULL OR EXISTS (SELECT 1 FROM customers.merge_case_member m JOIN customers.merge_case k USING (merge_case_id)
				WHERE m.customer_id = c.customer_id AND m.presence = 'PRESENT' AND k.presence = 'PRESENT' AND k.state = 'CONFLICT') = $2)
		  AND ($3::text IS NULL OR c.presence = $3)
		  AND ($4::uuid IS NULL OR c.customer_id > $4)
		ORDER BY c.customer_id LIMIT $5`, state, hasConflict, presence, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCustomer returns one customer.
func GetCustomer(ctx context.Context, tx db.Tx, id string) (Customer, bool, error) {
	if err := ReadTx(tx); err != nil {
		return Customer{}, false, err
	}
	c, err := scanCustomer(tx.QueryRow(ctx, customerSelect+` WHERE c.customer_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, false, nil
	}
	return c, err == nil, err
}

// ContactView is one masked contact point.
type ContactView struct {
	ID, Kind, Role, Masked, Normalization string
	SharedWith                            int64 // other present customers with the same normalized value
	ValidFromRaw, ValidToRaw, ObservedRaw *string
	Source                                Provenance
}

// ContactsOf lists a customer's contact points, masked.
func ContactsOf(ctx context.Context, tx db.Tx, customerID string) ([]ContactView, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT p.contact_point_id::text, p.kind, p.role, p.raw_value, p.normalization,
		CASE WHEN p.normalized_value IS NULL THEN 0 ELSE (SELECT count(DISTINCT o.customer_id) FROM customers.contact_point o
			JOIN customers.customer k ON k.customer_id = o.customer_id
			WHERE o.kind = p.kind AND o.normalized_value = p.normalized_value AND o.customer_id <> p.customer_id
			  AND o.presence = 'PRESENT' AND k.presence = 'PRESENT') END,
		p.valid_from_raw, p.valid_to_raw, p.observed_raw, p.source_system, p.source_spec, p.source_key, p.source_batch_id::text,
		p.imported_at, p.refreshed_at, p.presence
		FROM customers.contact_point p WHERE p.customer_id = $1 ORDER BY p.kind, p.role, p.contact_point_id`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContactView
	for rows.Next() {
		var v ContactView
		var raw string
		if err := rows.Scan(&v.ID, &v.Kind, &v.Role, &raw, &v.Normalization, &v.SharedWith, &v.ValidFromRaw, &v.ValidToRaw, &v.ObservedRaw,
			&v.Source.System, &v.Source.Spec, &v.Source.Key, &v.Source.BatchID, &v.Source.ImportedAt, &v.Source.RefreshedAt, &v.Source.Presence); err != nil {
			return nil, err
		}
		v.Masked = domain.Mask(raw)
		v.Source.ImportedAt, v.Source.RefreshedAt = v.Source.ImportedAt.UTC(), v.Source.RefreshedAt.UTC()
		out = append(out, v)
	}
	return out, rows.Err()
}

// IdentifierView is one identifier (value masked).
type IdentifierView struct {
	ID, Scheme, Masked, Derivation, Status string
	Source                                 Provenance
}

// IdentifiersOf lists a customer's identifiers.
func IdentifiersOf(ctx context.Context, tx db.Tx, customerID string) ([]IdentifierView, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT identifier_id::text, scheme, external_value, derivation, status, source_system, source_spec, source_key,
		source_batch_id::text, imported_at, refreshed_at, presence FROM customers.customer_identifier WHERE customer_id = $1
		ORDER BY scheme, identifier_id`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IdentifierView
	for rows.Next() {
		var v IdentifierView
		var raw string
		if err := rows.Scan(&v.ID, &v.Scheme, &raw, &v.Derivation, &v.Status, &v.Source.System, &v.Source.Spec, &v.Source.Key,
			&v.Source.BatchID, &v.Source.ImportedAt, &v.Source.RefreshedAt, &v.Source.Presence); err != nil {
			return nil, err
		}
		v.Masked = domain.Mask(raw)
		v.Source.ImportedAt, v.Source.RefreshedAt = v.Source.ImportedAt.UTC(), v.Source.RefreshedAt.UTC()
		out = append(out, v)
	}
	return out, rows.Err()
}

// EvidenceView is one evidence item (no raw values).
type EvidenceView struct {
	ID, Kind, Strength, Polarity, Origin, Reference, Presence string
	ObservedRaw                                               *string
	SubjectCount                                              int64
	Spec                                                      string
	RecordedAt                                                time.Time
}

// EvidenceOf lists the evidence about a customer, keyset by id.
func EvidenceOf(ctx context.Context, tx db.Tx, customerID string, strength *string, after *string, limit int) ([]EvidenceView, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT e.evidence_id::text, e.kind, e.strength, e.polarity, e.origin, e.reference, e.presence, e.observed_raw,
		(SELECT count(*) FROM customers.evidence_subject x WHERE x.evidence_id = e.evidence_id), e.source_spec, e.recorded_at
		FROM customers.identity_evidence e JOIN customers.evidence_subject s ON s.evidence_id = e.evidence_id AND s.customer_id = $1
		WHERE ($2::text IS NULL OR e.strength = $2) AND ($3::uuid IS NULL OR e.evidence_id > $3)
		ORDER BY e.evidence_id LIMIT $4`, customerID, strength, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EvidenceView
	for rows.Next() {
		var v EvidenceView
		if err := rows.Scan(&v.ID, &v.Kind, &v.Strength, &v.Polarity, &v.Origin, &v.Reference, &v.Presence, &v.ObservedRaw,
			&v.SubjectCount, &v.Spec, &v.RecordedAt); err != nil {
			return nil, err
		}
		v.RecordedAt = v.RecordedAt.UTC()
		out = append(out, v)
	}
	return out, rows.Err()
}

// Case is one merge case read model.
type Case struct {
	ID, Reason, State, StateReason, MaxStrength, Presence string
	Members                                               []string
	EvidenceKinds                                         []string
	CreatedAt, RefreshedAt                                time.Time
}

// CaseFilter is the allowlisted case filter set.
type CaseFilter struct {
	State, Strength, Reason, CustomerID *string
}

// CaseReasons and strengths allowlisted for filters.
var (
	CaseReasons = map[string]bool{"SHARED_PHONE": true, "SHARED_ACCOUNT_REF": true, "SHARED_EXCLUSIVE_IDENTIFIER": true, "SUPPLIED_EVIDENCE": true}
	Strengths   = map[string]bool{domain.Strong: true, domain.Medium: true, domain.Weak: true}
)

const caseSelect = `SELECT k.merge_case_id::text, k.reason, k.state, k.state_reason, k.max_strength, k.presence,
	COALESCE((SELECT array_agg(m.customer_id::text ORDER BY m.customer_id) FROM customers.merge_case_member m
		WHERE m.merge_case_id = k.merge_case_id AND m.presence = 'PRESENT'), '{}'),
	COALESCE((SELECT array_agg(DISTINCT e.kind ORDER BY e.kind) FROM customers.merge_case_evidence l
		JOIN customers.identity_evidence e USING (evidence_id) WHERE l.merge_case_id = k.merge_case_id AND l.presence = 'PRESENT'), '{}'),
	k.created_at, k.refreshed_at
	FROM customers.merge_case k`

func scanCase(row pgx.Row) (Case, error) {
	var c Case
	err := row.Scan(&c.ID, &c.Reason, &c.State, &c.StateReason, &c.MaxStrength, &c.Presence, &c.Members, &c.EvidenceKinds, &c.CreatedAt, &c.RefreshedAt)
	c.CreatedAt, c.RefreshedAt = c.CreatedAt.UTC(), c.RefreshedAt.UTC()
	return c, err
}

// ListCases returns one keyset page of present cases.
func ListCases(ctx context.Context, tx db.Tx, f CaseFilter, after *string, limit int) ([]Case, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, caseSelect+`
		WHERE k.presence = 'PRESENT'
		  AND ($1::text IS NULL OR k.state = $1) AND ($2::text IS NULL OR k.max_strength = $2) AND ($3::text IS NULL OR k.reason = $3)
		  AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM customers.merge_case_member m WHERE m.merge_case_id = k.merge_case_id
				AND m.customer_id = $4 AND m.presence = 'PRESENT'))
		  AND ($5::uuid IS NULL OR k.merge_case_id > $5)
		ORDER BY k.merge_case_id LIMIT $6`, f.State, f.Strength, f.Reason, f.CustomerID, after, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Case
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCase returns one case.
func GetCase(ctx context.Context, tx db.Tx, id string) (Case, bool, error) {
	if err := ReadTx(tx); err != nil {
		return Case{}, false, err
	}
	c, err := scanCase(tx.QueryRow(ctx, caseSelect+` WHERE k.merge_case_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Case{}, false, nil
	}
	return c, err == nil, err
}

// Report computes the resolution report. merge_count compares present
// source subjects with present customers: any merge would make them differ.
func Report(ctx context.Context, tx db.Tx) (api.Resolution, error) {
	if err := ReadTx(tx); err != nil {
		return api.Resolution{}, err
	}
	r := api.Resolution{ByState: map[string]int64{}, CasesByState: map[string]int64{}, CasesByReason: map[string]int64{}}
	for _, s := range []string{domain.Resolved, domain.Related, domain.Unknown, domain.Conflict, domain.NotRecoverable} {
		r.ByState[s], r.CasesByState[s] = 0, 0
	}
	scalar := map[string]*int64{
		`SELECT count(*) FROM customers.customer`:                                                                             &r.Customers,
		`SELECT count(*) FROM customers.customer WHERE presence = 'PRESENT'`:                                                  &r.CustomersPresent,
		`SELECT count(DISTINCT (source_system, subject_kind, source_key)) FROM customers.customer WHERE presence = 'PRESENT'`: &r.SubjectsPresent,
		`SELECT count(*) FROM customers.merge_case k WHERE k.presence = 'PRESENT' AND NOT EXISTS (SELECT 1 FROM customers.merge_case_evidence l
			JOIN customers.identity_evidence e USING (evidence_id) WHERE l.merge_case_id = k.merge_case_id AND l.presence = 'PRESENT'
			AND e.kind <> 'PHONE_MATCH')`: &r.PhoneOnlyCandidates,
		`SELECT count(*) FROM customers.identity_evidence WHERE presence = 'PRESENT' AND kind = 'OVERLAPPING_PHONE_OWNERSHIP'`: &r.OverlappingPhoneConflict,
		`SELECT count(*) FROM customers.merge_case WHERE presence = 'PRESENT' AND reason = 'SHARED_EXCLUSIVE_IDENTIFIER'`:      &r.DuplicateExternalID,
		`SELECT count(*) FROM customers.identity_evidence WHERE presence = 'PRESENT' AND strength = 'STRONG'`:                  &r.StrongEvidence,
		`SELECT count(*) FROM customers.customer c WHERE c.presence = 'PRESENT' AND EXISTS (SELECT 1 FROM customers.evidence_subject s
			JOIN customers.identity_evidence e USING (evidence_id) WHERE s.customer_id = c.customer_id AND e.presence = 'PRESENT')
			AND NOT EXISTS (SELECT 1 FROM customers.evidence_subject s JOIN customers.identity_evidence e USING (evidence_id)
			WHERE s.customer_id = c.customer_id AND e.presence = 'PRESENT' AND e.strength <> 'WEAK')`: &r.WeakOnlyCustomers,
		// RESOLVED customers/cases whose present evidence holds no STRONG item: must stay 0.
		`SELECT (SELECT count(*) FROM customers.merge_case k WHERE k.presence = 'PRESENT' AND k.state = 'RESOLVED' AND NOT EXISTS (
				SELECT 1 FROM customers.merge_case_evidence l JOIN customers.identity_evidence e USING (evidence_id)
				WHERE l.merge_case_id = k.merge_case_id AND l.presence = 'PRESENT' AND e.strength = 'STRONG'))
			+ (SELECT count(*) FROM customers.customer c WHERE c.presence = 'PRESENT' AND c.resolution_state = 'RESOLVED' AND NOT EXISTS (
				SELECT 1 FROM customers.evidence_subject s JOIN customers.identity_evidence e USING (evidence_id)
				WHERE s.customer_id = c.customer_id AND e.presence = 'PRESENT' AND e.strength = 'STRONG'))`: &r.PhoneOnlyResolved,
	}
	for q, dst := range scalar {
		if err := tx.QueryRow(ctx, q).Scan(dst); err != nil {
			return r, fmt.Errorf("report: %w", err)
		}
	}
	for q, m := range map[string]map[string]int64{
		`SELECT resolution_state, count(*) FROM customers.customer WHERE presence = 'PRESENT' GROUP BY 1`: r.ByState,
		`SELECT state, count(*) FROM customers.merge_case WHERE presence = 'PRESENT' GROUP BY 1`:          r.CasesByState,
		`SELECT reason, count(*) FROM customers.merge_case WHERE presence = 'PRESENT' GROUP BY 1`:         r.CasesByReason,
	} {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return r, err
		}
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				rows.Close()
				return r, err
			}
			m[k] = n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return r, err
		}
	}
	r.Merges = r.SubjectsPresent - r.CustomersPresent
	r.PhoneOnlyMerges = r.Merges // no merge path exists; any merge would be unproven by strong evidence
	r.Unresolved = r.ByState[domain.Unknown] + r.ByState[domain.Conflict] + r.ByState[domain.NotRecoverable]
	return r, nil
}
