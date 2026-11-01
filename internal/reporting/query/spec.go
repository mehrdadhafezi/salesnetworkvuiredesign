// Package query is the reporting read service and the ONE shared report
// query specification (INV-021). A Spec — report, scope, filters, time basis
// and the projection run (cutoff) it resolves to — produces exactly one
// WHERE clause. The card summary, the table rows and the detail membership
// all use that same clause; pagination only limits table rows and never
// changes the summary. Metric values are summaries of their contract's
// Spec, so a card equals its table aggregate by construction.
//
// Every read runs in a reporting-owned READ ONLY transaction over the
// reporting schema only: it never refreshes, backfills, repairs or records
// anything, and it never reads an owner schema.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
)

var (
	// ErrInvalidQuery reports a filter, time basis or cursor outside the contract.
	ErrInvalidQuery = errors.New("reporting/query: invalid query")
	// ErrNotFound reports an unknown run, metric or row.
	ErrNotFound = errors.New("reporting/query: not found")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

// ScopeOrganization is the only scope with frozen authority in Slice 8; a
// narrower row scope has no resolver yet, so it is never broadened to this.
const ScopeOrganization = "ORGANIZATION"

// Spec is the shared report query contract.
type Spec struct {
	Report    string
	Scope     string
	TimeBasis domain.TimeBasis
	From, To  *time.Time
	AsOf      *time.Time
	RunID     *string
	Filters   map[string]string
}

type filterDef struct {
	expr    string
	values  map[string]bool
	boolean bool // "true" / "false" → expr > 0 / = 0
	exists  string
}

type reportDef struct {
	table, id, alias string
	eventCol         string
	sources          []string
	filters          map[string]filterDef
	dims             map[string]string
}

func set(vs ...string) map[string]bool {
	m := map[string]bool{}
	for _, v := range vs {
		m[v] = true
	}
	return m
}

var (
	presence   = filterDef{expr: "f.presence", values: set("PRESENT", "ABSENT_IN_LATEST")}
	linkStates = set("RESOLVED", "RELATED", "UNKNOWN", "CONFLICT", "NOT_RECOVERABLE")
)

// reports is the allowlist of reports, filters and dimensions.
var reports = map[string]reportDef{
	domain.ReportCases: {table: "reporting.case_fact", id: "case_id", eventCol: "f.first_seen_at", sources: []string{"cases"},
		filters: map[string]filterDef{
			"presence":                  presence,
			"workflow_type":             {expr: "f.workflow_type", values: set("LEGACY_LEAD", "DISTRIBUTION_ITEM")},
			"state_mapping":             {expr: "f.state_mapping", values: set("KNOWN", "UNKNOWN")},
			"customer_link_state":       {expr: "f.customer_link_state", values: linkStates},
			"original_owner_resolution": {expr: "f.original_owner_resolution", values: set("RESOLVED", "RELATED", "UNKNOWN", "CONFLICT", "NOT_RECOVERABLE", "NOT_CAPTURED")},
		},
		dims: map[string]string{"workflow_type": "f.workflow_type", "state_mapping": "f.state_mapping", "customer_link_state": "f.customer_link_state",
			"original_owner_resolution": "f.original_owner_resolution"}},
	domain.ReportCustody: {table: "reporting.custody_fact", id: "case_id", sources: []string{"custody"},
		filters: map[string]filterDef{
			"presence":   presence,
			"parity":     {expr: "f.parity", values: set("MATCH", "TARGET_UNKNOWN", "LEGACY_UNKNOWN", "CONFLICT", "MULTIPLE_CURRENT", "HISTORY_GAP", "NOT_RECOVERABLE")},
			"derivation": {expr: "f.derivation", values: set("DERIVED", "RELEASED", "UNKNOWN", "MULTIPLE", "NONE")},
		},
		dims: map[string]string{"parity": "f.parity", "derivation": "f.derivation"}},
	domain.ReportInvoices: {table: "reporting.invoice_fact", id: "invoice_id", eventCol: "f.first_seen_at", sources: []string{"billing", "payments"},
		filters: map[string]filterDef{
			"presence":            presence,
			"customer_link_state": {expr: "f.customer_link_state", values: linkStates},
			"case_link_state":     {expr: "f.case_link_state", values: linkStates},
			"has_unknown_status":  {expr: "f.unknown_state_count", boolean: true},
			"main_mapping":        {expr: "f.main_mapping", values: set("KNOWN_VOCABULARY", "UNKNOWN", "EMPTY")},
			"total_class": {values: set("MATCH", "MISMATCH", "UNKNOWN", "INCOMPLETE", "CONFLICT"),
				exists: "EXISTS (SELECT 1 FROM reporting.invoice_unit_fact u WHERE u.run_id = f.run_id AND u.invoice_id = f.invoice_id AND u.total_class = %s)"},
			"unit": {values: set("IRT", "IRR", "UNATTRIBUTED"),
				exists: "EXISTS (SELECT 1 FROM reporting.invoice_unit_fact u WHERE u.run_id = f.run_id AND u.invoice_id = f.invoice_id AND u.unit = %s)"},
		},
		dims: map[string]string{"main_mapping": "f.main_mapping", "customer_link_state": "f.customer_link_state", "case_link_state": "f.case_link_state"}},
	domain.ReportPayments: {table: "reporting.payment_fact", id: "attempt_id", eventCol: "f.first_seen_at", sources: []string{"payments"},
		filters: map[string]filterDef{
			"presence":            presence,
			"channel":             {expr: "f.channel", values: set("GATEWAY", "MANUAL_CARD", "UNKNOWN")},
			"state":               {expr: "COALESCE(f.state_target, 'UNMAPPED')", values: set("PENDING", "ACCEPTED", "REJECTED", "CANCELLED", "RECONCILIATION_REQUIRED", "UNMAPPED")},
			"state_mapping":       {expr: "f.state_mapping", values: set("MAPPED", "UNKNOWN", "EMPTY")},
			"approval_provenance": {expr: "f.approval_provenance", values: set("GATEWAY_AUTO", "LEGACY_MANUAL_APPROVAL", "NOT_APPROVED", "UNKNOWN")},
			"invoice_link_state":  {expr: "f.invoice_link_state", values: set("RESOLVED", "UNKNOWN")},
			"unit":                {expr: "COALESCE(f.unit, 'NONE')", values: set("IRT", "IRR", "NONE")},
		},
		dims: map[string]string{"state": "COALESCE(f.state_target, 'UNMAPPED')", "state_mapping": "f.state_mapping", "channel": "f.channel",
			"approval_provenance": "f.approval_provenance", "invoice_link_state": "f.invoice_link_state"}},
}

// Reports lists the report ids.
func Reports() []string {
	out := make([]string, 0, len(reports))
	for r := range reports {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// FilterNames lists a report's allowlisted filters.
func FilterNames(report string) []string {
	var out []string
	for f := range reports[report].filters {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// RequiredSources lists the published sources a report depends on.
func RequiredSources(report string) []string {
	return append([]string(nil), reports[report].sources...)
}

// Validate checks the spec against the contract and normalises defaults.
func (s *Spec) Validate() error {
	def, ok := reports[s.Report]
	if !ok {
		return invalid("report")
	}
	if s.Scope == "" {
		s.Scope = ScopeOrganization
	}
	if s.Scope != ScopeOrganization {
		return invalid("scope (only ORGANIZATION has frozen authority)")
	}
	if s.TimeBasis == "" {
		s.TimeBasis = domain.CurrentSnapshot
	}
	switch s.TimeBasis {
	case domain.CurrentSnapshot:
		if s.From != nil || s.To != nil || s.AsOf != nil {
			return invalid("CURRENT_SNAPSHOT takes no from/to/as_of")
		}
	case domain.EventRange:
		if s.From == nil || s.To == nil || !s.From.Before(*s.To) || s.AsOf != nil {
			return invalid("EVENT_RANGE needs from < to and no as_of")
		}
	case domain.HistoricalAsOf:
		// run_id may pin the run as_of resolved to (returned specs stay
		// reusable); resolveRun checks that its cutoff is at/before as_of.
		if s.AsOf == nil || s.From != nil || s.To != nil {
			return invalid("HISTORICAL_AS_OF needs as_of (and optionally the run it resolved to)")
		}
	default:
		return invalid("time_basis")
	}
	if s.RunID != nil && !uuidPattern.MatchString(*s.RunID) {
		return invalid("run_id")
	}
	for k, v := range s.Filters {
		f, ok := def.filters[k]
		if !ok {
			return invalid("filter %q is not allowed for %s", k, s.Report)
		}
		if f.boolean {
			if v != "true" && v != "false" {
				return invalid("filter %q", k)
			}
		} else if !f.values[v] {
			return invalid("filter %q value", k)
		}
	}
	return nil
}

// EventRangeSupported reports whether a report has a trustworthy event time.
func EventRangeSupported(report string) bool { return reports[report].eventCol != "" }

// where is THE single membership predicate of a spec and run.
func where(s Spec, runID string) (string, []any) {
	def := reports[s.Report]
	conds := []string{"f.run_id = $1"}
	args := []any{runID}
	keys := make([]string, 0, len(s.Filters))
	for k := range s.Filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f, v := def.filters[k], s.Filters[k]
		switch {
		case f.boolean && v == "true":
			conds = append(conds, f.expr+" > 0")
		case f.boolean:
			conds = append(conds, f.expr+" = 0")
		case f.exists != "":
			args = append(args, v)
			conds = append(conds, fmt.Sprintf(f.exists, fmt.Sprintf("$%d", len(args))))
		default:
			args = append(args, v)
			conds = append(conds, fmt.Sprintf("%s = $%d", f.expr, len(args)))
		}
	}
	if s.TimeBasis == domain.EventRange {
		args = append(args, s.From.UTC(), s.To.UTC())
		conds = append(conds, fmt.Sprintf("%s >= $%d AND %s < $%d", def.eventCol, len(args)-1, def.eventCol, len(args)))
	}
	return strings.Join(conds, " AND "), args
}

// Run is the projection run a spec resolved to (its cutoff).
type Run struct {
	RunID             string
	ProjectionVersion int
	SourceCutoff      time.Time
	CompletedAt       time.Time
	Outcome           string
	Sources           map[string]SourceStatus
}

// SourceStatus is one published source's contribution to a run.
type SourceStatus struct {
	Status   string
	RowsRead *int64
	ReadAt   *time.Time
	Reason   string
}

// ErrNotAvailable reports that no projection run can answer the spec (no
// run yet, or no run at/before as_of): the answer is UNKNOWN, never zero.
var ErrNotAvailable = errors.New("reporting/query: no projection run can answer this query")

func resolveRun(ctx context.Context, tx db.Tx, s Spec) (Run, error) {
	var r Run
	var err error
	switch {
	case s.RunID != nil:
		err = tx.QueryRow(ctx, `SELECT run_id::text, projection_version, source_cutoff, completed_at, outcome FROM reporting.projection_run
			WHERE run_id = $1`, *s.RunID).Scan(&r.RunID, &r.ProjectionVersion, &r.SourceCutoff, &r.CompletedAt, &r.Outcome)
		if errors.Is(err, pgx.ErrNoRows) {
			return r, ErrNotFound
		}
		if err == nil && s.TimeBasis == domain.HistoricalAsOf && r.SourceCutoff.After(s.AsOf.UTC()) {
			return r, invalid("run_id cutoff is after as_of")
		}
	case s.TimeBasis == domain.HistoricalAsOf:
		err = tx.QueryRow(ctx, `SELECT run_id::text, projection_version, source_cutoff, completed_at, outcome FROM reporting.projection_run
			WHERE projection_version = $1 AND source_cutoff <= $2 ORDER BY source_cutoff DESC, run_seq DESC LIMIT 1`,
			domain.ProjectionVersion, s.AsOf.UTC()).Scan(&r.RunID, &r.ProjectionVersion, &r.SourceCutoff, &r.CompletedAt, &r.Outcome)
	default:
		err = tx.QueryRow(ctx, `SELECT run_id::text, projection_version, source_cutoff, completed_at, outcome FROM reporting.projection_run
			WHERE projection_version = $1 ORDER BY run_seq DESC LIMIT 1`,
			domain.ProjectionVersion).Scan(&r.RunID, &r.ProjectionVersion, &r.SourceCutoff, &r.CompletedAt, &r.Outcome)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotAvailable
	}
	if err != nil {
		return r, err
	}
	r.Sources, err = sourcesOf(ctx, tx, r.RunID)
	return r, err
}

func sourcesOf(ctx context.Context, tx db.Tx, runID string) (map[string]SourceStatus, error) {
	rows, err := tx.Query(ctx, `SELECT source, status, rows_read, read_at, reason FROM reporting.projection_source WHERE run_id = $1`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]SourceStatus{}
	for rows.Next() {
		var name string
		var st SourceStatus
		if err := rows.Scan(&name, &st.Status, &st.RowsRead, &st.ReadAt, &st.Reason); err != nil {
			return nil, err
		}
		out[name] = st
	}
	return out, rows.Err()
}

// PrimaryUsable reports whether the report's primary source (whose facts the
// report rows are) contributed facts to the run. Without them a count would
// be a fabricated zero, so the answer is UNKNOWN instead.
func (r Run) PrimaryUsable(report string) bool {
	srcs := RequiredSources(report)
	if len(srcs) == 0 {
		return false
	}
	st, ok := r.Sources[srcs[0]]
	return ok && (st.Status == domain.Available || st.Status == domain.Partial)
}

// Coverage folds the statuses of a report's required sources.
func (r Run) Coverage(report string) (string, map[string]string) {
	per := map[string]string{}
	var states []string
	for _, s := range RequiredSources(report) {
		st, ok := r.Sources[s]
		status := domain.Unknown
		if ok {
			status = st.Status
		}
		per[s] = status
		states = append(states, status)
	}
	return domain.WorstCoverage(states...), per
}
