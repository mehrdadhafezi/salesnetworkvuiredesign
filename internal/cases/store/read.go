package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// ErrNotFound reports an unknown Case.
var ErrNotFound = errors.New("cases/store: not found")

// Source is a Case's lineage.
type Source struct {
	System, Spec, Key        string
	ChannelRaw, OriginRefRaw *string
	FirstBatchID, LastBatch  string
	Presence                 string
	FirstSeen, LastSeen      time.Time
}

// Case is the read view of one Case (no phone, no names: none are stored).
type Case struct {
	CaseID, Workflow, StateNamespace string
	StateRaw                         *string
	StateMapping                     string
	CustomerID                       *string
	CustomerLinkState                string
	CustomerLinkReason               string
	Presence                         string
	ImportedAt, RefreshedAt          time.Time
	Source                           Source
	AliasCount, HistoryCount         int64
}

const caseSelect = `SELECT c.case_id::text, c.workflow_type, c.state_namespace, c.state_raw, c.state_mapping, c.customer_id::text,
	c.customer_link_state, c.customer_link_reason, c.presence, c.imported_at, c.refreshed_at,
	s.source_system, s.source_spec, s.source_key, s.channel_raw, s.origin_ref_raw, s.first_batch_id::text, s.last_batch_id::text,
	s.presence, s.first_seen_at, s.last_seen_at,
	(SELECT count(*) FROM cases.case_alias a WHERE (a.case_id = c.case_id OR a.related_case_id = c.case_id) AND a.presence = 'PRESENT'),
	(SELECT count(*) FROM cases.case_state_history h WHERE h.case_id = c.case_id)
	FROM cases.work_case c JOIN cases.case_source s USING (case_id)`

func scanCase(r pgx.Row) (Case, error) {
	var c Case
	err := r.Scan(&c.CaseID, &c.Workflow, &c.StateNamespace, &c.StateRaw, &c.StateMapping, &c.CustomerID, &c.CustomerLinkState,
		&c.CustomerLinkReason, &c.Presence, &c.ImportedAt, &c.RefreshedAt,
		&c.Source.System, &c.Source.Spec, &c.Source.Key, &c.Source.ChannelRaw, &c.Source.OriginRefRaw, &c.Source.FirstBatchID,
		&c.Source.LastBatch, &c.Source.Presence, &c.Source.FirstSeen, &c.Source.LastSeen, &c.AliasCount, &c.HistoryCount)
	return c, err
}

// CaseFilter is the allowlisted list filter.
type CaseFilter struct {
	Workflow, StateMapping, CustomerLinkState, Presence *string
}

// ListCases lists Cases by case id (keyset).
func ListCases(ctx context.Context, tx db.Tx, f CaseFilter, after *string, limit int) ([]Case, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, caseSelect+`
		WHERE ($1::text IS NULL OR c.workflow_type = $1) AND ($2::text IS NULL OR c.state_mapping = $2)
		  AND ($3::text IS NULL OR c.customer_link_state = $3) AND ($4::text IS NULL OR c.presence = $4)
		  AND ($5::uuid IS NULL OR c.case_id > $5)
		ORDER BY c.case_id LIMIT $6`, f.Workflow, f.StateMapping, f.CustomerLinkState, f.Presence, after, limit+1)
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

// GetCase returns one Case.
func GetCase(ctx context.Context, tx db.Tx, id string) (Case, error) {
	if err := ReadTx(tx); err != nil {
		return Case{}, err
	}
	c, err := scanCase(tx.QueryRow(ctx, caseSelect+` WHERE c.case_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Case{}, ErrNotFound
	}
	return c, err
}

// Lead is the raw legacy lead facts of a Case.
type Lead struct {
	StatusRaw, LeadStatusRaw, DestinationPanelRaw *string
	ImportedAtRaw, AssignedAtRaw, UpdatedAtRaw    *string
	Presence                                      string
}

// LeadOf returns the lead facts (nil for non-lead Cases).
func LeadOf(ctx context.Context, tx db.Tx, caseID string) (*Lead, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	var l Lead
	err := tx.QueryRow(ctx, `SELECT status_raw, lead_status_raw, destination_panel_raw, imported_at_raw, assigned_at_raw, updated_at_raw, presence
		FROM cases.lead WHERE case_id = $1`, caseID).Scan(&l.StatusRaw, &l.LeadStatusRaw, &l.DestinationPanelRaw, &l.ImportedAtRaw,
		&l.AssignedAtRaw, &l.UpdatedAtRaw, &l.Presence)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &l, err
}

// Alias is one explicit link touching a Case (either direction).
type Alias struct {
	AliasID, CaseID         string
	RelatedCaseID           *string
	RelatedSpec, RelatedKey string
	Relation, Resolution    string
	Basis, Presence         string
	RecordedAt              time.Time
}

// AliasesOf returns the links of a Case.
func AliasesOf(ctx context.Context, tx db.Tx, caseID string) ([]Alias, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT alias_id::text, case_id::text, related_case_id::text, related_source_spec, related_source_key,
		relation, resolution, basis, presence, recorded_at FROM cases.case_alias
		WHERE case_id = $1 OR related_case_id = $1 ORDER BY alias_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Alias
	for rows.Next() {
		var a Alias
		if err := rows.Scan(&a.AliasID, &a.CaseID, &a.RelatedCaseID, &a.RelatedSpec, &a.RelatedKey, &a.Relation, &a.Resolution,
			&a.Basis, &a.Presence, &a.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// History is one state history row.
type History struct {
	HistoryID             string
	SourceSpec, SourceKey string
	OldRaw, NewRaw        *string
	ActorKey, ActorUserID *string
	OccurredRaw           *string
	RecordedAt            time.Time
}

// HistoryOf pages a Case's history in source order (keyset on history id
// would not be source order, so the cursor is the source ordinal).
func HistoryOf(ctx context.Context, tx db.Tx, caseID string, afterOrdinal *string, limit int) ([]History, []string, error) {
	if err := ReadTx(tx); err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(ctx, `SELECT history_id::text, source_spec, source_key, old_state_raw, new_state_raw, actor_legacy_key,
		actor_user_id::text, occurred_raw, recorded_at, source_ordinal::text FROM cases.case_state_history
		WHERE case_id = $1 AND ($2::numeric IS NULL OR source_ordinal > $2::numeric) ORDER BY source_ordinal LIMIT $3`, caseID, afterOrdinal, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []History
	var ords []string
	for rows.Next() {
		var h History
		var ord string
		if err := rows.Scan(&h.HistoryID, &h.SourceSpec, &h.SourceKey, &h.OldRaw, &h.NewRaw, &h.ActorKey, &h.ActorUserID, &h.OccurredRaw,
			&h.RecordedAt, &ord); err != nil {
			return nil, nil, err
		}
		out = append(out, h)
		ords = append(ords, ord)
	}
	return out, ords, rows.Err()
}

// Snapshot is one immutable attribution snapshot.
type Snapshot struct {
	SnapshotID, Facet, Resolution string
	SubjectKey, SubjectUserID     *string
	PositionRaw                   *string
	Basis                         string
	SourceTimeRaw                 *string
	EvidenceSpec, EvidenceKey     *string
	CapturedBatchID               string
	CapturedAt                    time.Time
}

// SnapshotsOf returns every snapshot of a Case (oldest first per facet).
func SnapshotsOf(ctx context.Context, tx db.Tx, caseID string) ([]Snapshot, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT snapshot_id::text, facet, resolution, subject_legacy_key, subject_user_id::text, position_raw, basis,
		source_time_raw, evidence_spec, evidence_key, captured_batch_id::text, captured_at FROM cases.attribution_snapshot
		WHERE case_id = $1 ORDER BY facet, captured_at, snapshot_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.SnapshotID, &s.Facet, &s.Resolution, &s.SubjectKey, &s.SubjectUserID, &s.PositionRaw, &s.Basis,
			&s.SourceTimeRaw, &s.EvidenceSpec, &s.EvidenceKey, &s.CapturedBatchID, &s.CapturedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ResolveSources maps source keys of one spec to their Case.
func ResolveSources(ctx context.Context, tx db.Tx, system, spec string, keys []string) (map[string]api.SourceCase, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []string{}
	}
	rows, err := tx.Query(ctx, `SELECT source_key, case_id::text, last_batch_id::text, row_sha256, presence = 'PRESENT' FROM cases.case_source
		WHERE source_system = $1 AND source_spec = $2 AND source_key = ANY($3::text[])`, system, spec, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]api.SourceCase{}
	for rows.Next() {
		var k string
		var s api.SourceCase
		if err := rows.Scan(&k, &s.CaseID, &s.BatchID, &s.RowSHA256, &s.Present); err != nil {
			return nil, err
		}
		out[k] = s
	}
	return out, rows.Err()
}

// Coverage computes the cases coverage report.
func Coverage(ctx context.Context, tx db.Tx) (api.Coverage, error) {
	if err := ReadTx(tx); err != nil {
		return api.Coverage{}, err
	}
	r := api.Coverage{CasesBySource: map[string]int64{}, CasesByWorkflow: map[string]int64{}, AliasesByResolution: map[string]int64{},
		AttributionByFacet: map[string]map[string]int64{}, CustomerLinks: map[string]int64{}}
	for q, dst := range map[string]*int64{
		`SELECT count(*) FROM cases.work_case`:                                                                                    &r.CasesTotal,
		`SELECT count(*) FROM cases.work_case WHERE presence = 'PRESENT'`:                                                         &r.CasesPresent,
		`SELECT count(*) FROM cases.work_case WHERE state_mapping = 'UNKNOWN'`:                                                    &r.UnknownStates,
		`SELECT count(*) FROM cases.work_case c WHERE NOT EXISTS (SELECT 1 FROM cases.case_source s WHERE s.case_id = c.case_id)`: &r.SourcelessCases,
		`SELECT count(*) FROM cases.case_state_history`:                                                                           &r.StatusHistoryRows,
		`SELECT count(DISTINCT case_id) FROM cases.case_state_history`:                                                            &r.CasesWithHistory,
	} {
		if err := tx.QueryRow(ctx, q).Scan(dst); err != nil {
			return r, err
		}
	}
	// Legacy Leads Only (GATE-0 metric table): distinct present legacy leads.
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM cases.lead WHERE presence = 'PRESENT'`).Scan(&r.LegacyLeadsOnly); err != nil {
		return r, err
	}
	group := func(q string, into func(a, b string, n int64)) error {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a, b string
			var n int64
			if err := rows.Scan(&a, &b, &n); err != nil {
				return err
			}
			into(a, b, n)
		}
		return rows.Err()
	}
	for _, g := range []struct {
		q    string
		into func(a, b string, n int64)
	}{
		{`SELECT s.source_spec, '', count(*) FROM cases.work_case c JOIN cases.case_source s USING (case_id) WHERE c.presence = 'PRESENT' GROUP BY 1`,
			func(a, _ string, n int64) { r.CasesBySource[a] = n }},
		{`SELECT workflow_type, '', count(*) FROM cases.work_case WHERE presence = 'PRESENT' GROUP BY 1`, func(a, _ string, n int64) { r.CasesByWorkflow[a] = n }},
		{`SELECT resolution, '', count(*) FROM cases.case_alias WHERE presence = 'PRESENT' GROUP BY 1`, func(a, _ string, n int64) { r.AliasesByResolution[a] = n }},
		{`SELECT customer_link_state, '', count(*) FROM cases.work_case WHERE presence = 'PRESENT' GROUP BY 1`, func(a, _ string, n int64) { r.CustomerLinks[a] = n }},
		{`SELECT facet, resolution, count(DISTINCT case_id) FROM cases.attribution_snapshot GROUP BY 1, 2`, func(a, b string, n int64) {
			if r.AttributionByFacet[a] == nil {
				r.AttributionByFacet[a] = map[string]int64{}
			}
			r.AttributionByFacet[a][b] = n
		}},
	} {
		if err := group(g.q, g.into); err != nil {
			return r, err
		}
	}
	return r, nil
}

// CoverageView is the minimal input of the honest F02/F03 metrics.
type CoverageView struct {
	LegacyLeadsOnly, CasesPresent  int64
	AliasesRelated, AliasesUnknown int64
}
