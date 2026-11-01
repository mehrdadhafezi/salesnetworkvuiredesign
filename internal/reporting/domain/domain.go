// Package domain holds the reporting module's pure contracts: versioned
// metric contracts (the only place a metric formula is declared), temporal
// modes, freshness and coverage classification. No I/O.
//
// A metric missing any required metadata, or depending on an unresolved
// decision (OPD-07, TAD #20), is NOT_FINAL and never yields a value. UNKNOWN,
// INCOMPLETE, DENIED, NOT_RECOVERABLE, STALE and ERROR are never zero.
package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// ProjectionVersion is the version of the projection fact model. Changing
// any fact's meaning requires a new version; runs of different versions
// coexist and are never rewritten.
const ProjectionVersion = 1

// TimeBasis is an explicit temporal mode (Cross-Role §13, INV-022).
type TimeBasis string

const (
	// CurrentSnapshot evaluates the facts of the latest run (state at its
	// source cutoff).
	CurrentSnapshot TimeBasis = "CURRENT_SNAPSHOT"
	// EventRange selects a cohort by an event time inside [from, to); the
	// only trustworthy UTC event time in Slice 8 is the target's first
	// observation of a source row (legacy business times carry no asserted
	// zone).
	EventRange TimeBasis = "EVENT_RANGE"
	// HistoricalAsOf answers from the run whose source cutoff is the latest
	// at or before as_of: the state as it was RECORDED then. Without such a
	// run the answer is NOT_AVAILABLE; nothing is reconstructed from the
	// current state or hierarchy.
	HistoricalAsOf TimeBasis = "HISTORICAL_AS_OF"
	// NotDefined marks a contract whose time basis is not frozen.
	NotDefined TimeBasis = "NOT_DEFINED"
)

// Aggregation kinds a metric may declare (no free-form formula DSL).
const (
	AggCount     = "COUNT"
	AggCountBy   = "COUNT_BY"
	AggSumByUnit = "SUM_BY_UNIT"
	AggUnitClass = "UNIT_CLASS"
)

// Reports a metric may be evaluated on (the shared report query specs).
const (
	ReportCases    = "cases"
	ReportCustody  = "custody"
	ReportInvoices = "invoices"
	ReportPayments = "payments"
)

// Contract is one versioned metric contract.
type Contract struct {
	ID        string      `json:"metric_id"`
	Version   int         `json:"version"`
	Name      string      `json:"name"`
	Grain     string      `json:"grain"`
	Source    string      `json:"source"`
	Cohort    string      `json:"cohort"`
	TimeBases []TimeBasis `json:"time_bases"`
	Scope     string      `json:"scope"`
	Formula   string      `json:"formula"`
	Unit      string      `json:"unit"`
	// FreshnessSLA is how old the run may be before results are STALE.
	FreshnessSLA time.Duration     `json:"freshness_sla"`
	Coverage     string            `json:"coverage"`
	Report       string            `json:"report"`
	Filters      map[string]string `json:"filters"`
	Aggregation  string            `json:"aggregation"`
	Dimension    string            `json:"dimension"`
	// Blocked names an unresolved decision the metric depends on.
	Blocked string `json:"blocked"`
	Finding string `json:"finding"`
}

// Status is FINAL only when every required field is present and nothing
// blocks the metric; otherwise NOT_FINAL with the reason.
func (c Contract) Status() (string, string) {
	if c.Blocked != "" {
		return "NOT_FINAL", c.Blocked
	}
	var missing []string
	for name, v := range map[string]string{"name": c.Name, "grain": c.Grain, "source": c.Source, "cohort": c.Cohort, "scope": c.Scope,
		"formula": c.Formula, "unit": c.Unit, "coverage": c.Coverage, "report": c.Report, "aggregation": c.Aggregation} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(c.TimeBases) == 0 {
		missing = append(missing, "time_basis")
	}
	if c.FreshnessSLA <= 0 {
		missing = append(missing, "freshness")
	}
	if c.Version <= 0 {
		missing = append(missing, "version")
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "NOT_FINAL", fmt.Sprintf("missing required metadata: %v", missing)
	}
	return "FINAL", ""
}

// Allows reports whether the contract may be evaluated in time basis tb.
func (c Contract) Allows(tb TimeBasis) bool {
	for _, b := range c.TimeBases {
		if b == tb {
			return true
		}
	}
	return false
}

// PrimaryBasis is the first declared time basis (or NOT_DEFINED).
func (c Contract) PrimaryBasis() TimeBasis {
	if len(c.TimeBases) == 0 {
		return NotDefined
	}
	return c.TimeBases[0]
}

// SHA256 fingerprints the contract's meaning; a registered version whose
// fingerprint changes is a contract violation (bump the version instead).
func (c Contract) SHA256() []byte {
	b, _ := json.Marshal(c) // map keys are sorted by encoding/json
	h := sha256.Sum256(b)
	return h[:]
}

const day = 24 * time.Hour

const (
	scopeOrg    = "ORGANIZATION (evaluator-granted; no frozen row scope narrower than the organisation exists)"
	srcCases    = "cases published FactSource (Slice 6)"
	srcCustody  = "custody published FactSource (Slice 6)"
	srcBilling  = "billing published FactSource (Slice 7)"
	srcPayments = "payments published FactSource (Slice 7)"
	snapCohort  = "facts present in the projection run selected by the time basis"
)

var both = []TimeBasis{CurrentSnapshot, HistoricalAsOf}

// registry is the only place metric formulas are declared.
var registry = []Contract{
	{ID: "cases.staged_total", Version: 1, Name: "Cases in staged sources (F03: not a proven cross-source Total Cases)", Grain: "Case (one per staged source row)",
		Source: srcCases, Cohort: snapCohort + "; presence = PRESENT", TimeBases: both, Scope: scopeOrg,
		Formula: "count(case facts) where presence = PRESENT", Unit: "count", FreshnessSLA: day,
		Coverage: "PARTIAL by definition: only legacy leads and distribution items are staged; explicit links are unproven aliases (OPD-01), nothing is deduplicated",
		Report:   ReportCases, Filters: map[string]string{"presence": "PRESENT"}, Aggregation: AggCount, Finding: "F03"},
	{ID: "cases.legacy_leads_only", Version: 1, Name: "Legacy Leads Only (F03; not Total Cases)", Grain: "legacy lead fact",
		Source: srcCases, Cohort: snapCohort + "; presence = PRESENT; workflow = LEGACY_LEAD", TimeBases: both, Scope: scopeOrg,
		Formula: "count(case facts) where presence = PRESENT and workflow_type = LEGACY_LEAD", Unit: "count", FreshnessSLA: day,
		Coverage: "legacy lead source as staged", Report: ReportCases, Filters: map[string]string{"presence": "PRESENT", "workflow_type": "LEGACY_LEAD"},
		Aggregation: AggCount, Finding: "F03"},
	{ID: "cases.unknown_state", Version: 1, Name: "Cases with an unmapped source state", Grain: "Case", Source: srcCases,
		Cohort: snapCohort + "; presence = PRESENT; state_mapping = UNKNOWN", TimeBases: both, Scope: scopeOrg,
		Formula: "count(case facts) where presence = PRESENT and state_mapping = UNKNOWN", Unit: "count", FreshnessSLA: day,
		Coverage: "staged sales sources", Report: ReportCases, Filters: map[string]string{"presence": "PRESENT", "state_mapping": "UNKNOWN"},
		Aggregation: AggCount},
	{ID: "cases.first_observed", Version: 1, Name: "Cases first observed by the target import in the range", Grain: "Case",
		Source: srcCases, Cohort: "cases whose source row the target import first observed in [from, to) (UTC); NOT a legacy business event time",
		TimeBases: []TimeBasis{EventRange}, Scope: scopeOrg, Formula: "count(case facts of the latest run) where from <= first_seen_at < to",
		Unit: "count", FreshnessSLA: day, Coverage: "staged sales sources", Report: ReportCases, Filters: map[string]string{},
		Aggregation: AggCount},
	{ID: "custody.by_parity", Version: 1, Name: "Current custody parity (derived holder vs legacy holder evidence)", Grain: "Case custody projection",
		Source: srcCustody, Cohort: snapCohort + "; presence = PRESENT", TimeBases: both, Scope: scopeOrg,
		Formula: "count(custody facts) where presence = PRESENT, grouped by parity", Unit: "count", FreshnessSLA: day,
		Coverage: "distribution items with logs; legacy leads have no custody log (NOT_RECOVERABLE)", Report: ReportCustody,
		Filters: map[string]string{"presence": "PRESENT"}, Aggregation: AggCountBy, Dimension: "parity"},
	{ID: "invoices.staged_total", Version: 1, Name: "Invoices in the staged billing source", Grain: "Invoice", Source: srcBilling,
		Cohort: snapCohort + "; presence = PRESENT", TimeBases: both, Scope: scopeOrg, Formula: "count(invoice facts) where presence = PRESENT",
		Unit: "count", FreshnessSLA: day, Coverage: "staged billing source", Report: ReportInvoices, Filters: map[string]string{"presence": "PRESENT"},
		Aggregation: AggCount},
	{ID: "invoices.unknown_status", Version: 1, Name: "Invoices with an unknown raw state value", Grain: "Invoice", Source: srcBilling,
		Cohort: snapCohort + "; presence = PRESENT; any invoice or stage state UNKNOWN", TimeBases: both, Scope: scopeOrg,
		Formula: "count(invoice facts) where presence = PRESENT and unknown_state_count > 0", Unit: "count", FreshnessSLA: day,
		Coverage: "staged billing source", Report: ReportInvoices, Filters: map[string]string{"presence": "PRESENT", "has_unknown_status": "true"},
		Aggregation: AggCount},
	{ID: "invoices.total_evidence_class", Version: 1, Name: "Invoice total evidence agreement per unit (no canonical total; TAD #19)",
		Grain: "Invoice x unit", Source: srcBilling + " + " + srcPayments, Cohort: snapCohort + "; presence = PRESENT", TimeBases: both,
		Scope: scopeOrg, Formula: "count(invoice unit facts) grouped by unit and class (MATCH/MISMATCH/UNKNOWN/INCOMPLETE/CONFLICT)",
		Unit: "count per unit", FreshnessSLA: day, Coverage: "staged billing and payments sources", Report: ReportInvoices,
		Filters: map[string]string{"presence": "PRESENT"}, Aggregation: AggUnitClass},
	{ID: "invoices.open_preinvoices", Version: 1, Name: "Open Pre-invoices (F02)", Grain: "Invoice", Source: srcBilling,
		Cohort: "current normalized invoice state = pre_invoice without completion/cancel", TimeBases: []TimeBasis{CurrentSnapshot},
		Scope: scopeOrg, Unit: "count", FreshnessSLA: day, Coverage: "raw invoice status fields only", Report: ReportInvoices,
		Aggregation: AggCount, Blocked: "NOT_EVALUATED: needs the canonical current invoice state (TAD #20 deferred)", Finding: "F02"},
	{ID: "payments.by_state", Version: 1, Name: "Payment attempts by explicitly mapped attempt state", Grain: "Payment attempt",
		Source: srcPayments, Cohort: snapCohort + "; presence = PRESENT", TimeBases: both, Scope: scopeOrg,
		Formula: "count(payment facts) where presence = PRESENT grouped by state (UNMAPPED when no explicit target)", Unit: "count",
		FreshnessSLA: day, Coverage: "staged payments source", Report: ReportPayments, Filters: map[string]string{"presence": "PRESENT"},
		Aggregation: AggCountBy, Dimension: "state"},
	{ID: "payments.accepted_attempt_amount", Version: 1, Name: "Amount of attempts explicitly mapped ACCEPTED (not Finance-verified collection)",
		Grain: "Payment attempt", Source: srcPayments, Cohort: snapCohort + "; presence = PRESENT; state = ACCEPTED", TimeBases: both,
		Scope: scopeOrg, Formula: "sum(amount) per unit where presence = PRESENT and state = ACCEPTED; unrepresentable amounts make it INCOMPLETE",
		Unit: "money per unit (IRT and IRR separate; never converted)", FreshnessSLA: day,
		Coverage: "staged payments source; 'approved' is not Finance-reviewed (D-25 open)", Report: ReportPayments,
		Filters: map[string]string{"presence": "PRESENT", "state": "ACCEPTED"}, Aggregation: AggSumByUnit},
	{ID: "cases.completion_rate", Version: 1, Name: "Completion Rate", Grain: "Case cohort", Source: srcCases + " + " + srcBilling,
		TimeBases: nil, Scope: scopeOrg, Unit: "ratio", Report: ReportCases,
		Blocked: "NOT_FINAL: OPD-07 (cohort, outcome window and attribution unresolved); not evaluated"},
	{ID: "cases.conversion_rate", Version: 1, Name: "Conversion Rate", Grain: "Case cohort", Source: srcCases + " + " + srcBilling,
		TimeBases: nil, Scope: scopeOrg, Unit: "ratio", Report: ReportCases,
		Blocked: "NOT_FINAL: OPD-07 (assigned cohort/window unresolved); not evaluated"},
}

// Registry returns the metric contracts in a stable order.
func Registry() []Contract { return append([]Contract(nil), registry...) }

// Lookup returns one contract by id (latest version).
func Lookup(id string) (Contract, bool) {
	var best Contract
	found := false
	for _, c := range registry {
		if c.ID == id && (!found || c.Version > best.Version) {
			best, found = c, true
		}
	}
	return best, found
}

// Freshness states.
const (
	Fresh   = "FRESH"
	Stale   = "STALE"
	Unknown = "UNKNOWN"
)

// FreshnessOf classifies a run's age against an SLA (no run = UNKNOWN).
func FreshnessOf(completed *time.Time, now time.Time, sla time.Duration) string {
	if completed == nil {
		return Unknown
	}
	if now.Sub(*completed) > sla {
		return Stale
	}
	return Fresh
}

// Source coverage states (per published owner source).
const (
	Available      = "AVAILABLE"
	Partial        = "PARTIAL"
	Missing        = "MISSING"
	NotRecoverable = "NOT_RECOVERABLE"
)

var coverageRank = map[string]int{Available: 0, Partial: 1, Stale: 2, Unknown: 3, Missing: 4, NotRecoverable: 5}

// WorstCoverage folds source statuses into one (AVAILABLE only if all are).
func WorstCoverage(states ...string) string {
	if len(states) == 0 {
		return Unknown
	}
	w := Available
	for _, s := range states {
		r, ok := coverageRank[s]
		if !ok {
			s, r = Unknown, coverageRank[Unknown]
		}
		if r > coverageRank[w] {
			w = s
		}
	}
	return w
}
