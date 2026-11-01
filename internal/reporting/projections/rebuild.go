// Package projections is the reporting module's OFFLINE projection rebuild
// (a job command, never reachable from a read path: architecture lint
// read-path-import). It reads each owner through its PUBLISHED read API in
// that owner's own read-only snapshot — never another schema — and writes
// one immutable run with all its facts in one reporting transaction.
//
// A missing or failing source is recorded per source and makes the run
// PARTIAL, never SUCCESS; facts of a missing source are absent, not zero.
package projections

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	custodyapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	customersapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
)

// ErrContractChanged reports a registered metric version whose meaning
// changed: bump the version instead of rewriting history.
var ErrContractChanged = errors.New("reporting: metric contract changed without a version bump")

// Source names (published owner APIs).
const (
	SourceCases     = "cases"
	SourceCustody   = "custody"
	SourceBilling   = "billing"
	SourcePayments  = "payments"
	SourceCustomers = "customers"
)

// AllSources lists every source a complete run needs.
func AllSources() []string {
	return []string{SourceCases, SourceCustody, SourceBilling, SourcePayments, SourceCustomers}
}

// Sources are the published owner read APIs; a nil source is MISSING.
type Sources struct {
	Cases     casesapi.FactSource
	Custody   custodyapi.FactSource
	Billing   billingapi.FactSource
	Payments  paymentsapi.FactSource
	PayReader paymentsapi.Reader
	Customers customersapi.Reader
}

// Run is the outcome of one rebuild.
type Run struct {
	RunID     string
	Outcome   result.Status
	Reasons   []reasoncodes.Code
	Sources   map[string]SourceState
	Counts    map[string]int64
	FactsSHA  []byte
	Cutoff    time.Time
	Completed time.Time
}

// SourceState is one source's contribution.
type SourceState struct {
	Status string     `json:"status"`
	Rows   *int64     `json:"rows_read"`
	ReadAt *time.Time `json:"read_at"`
	Reason string     `json:"reason"`
}

// Command rebuilds the projections.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a reporting-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Command, error) {
	if runner.Module() != reporting.Module {
		return nil, errors.New("reporting: rebuild runner must be owned by reporting")
	}
	return &Command{runner: runner, clock: clock}, nil
}

type table struct {
	name string
	cols []string
	rows [][]any
}

func (t *table) add(row ...any) { t.rows = append(t.rows, row) }

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// Rebuild runs one projection rebuild. Owner reads happen first (each in the
// owner's own read-only snapshot); the reporting write is one transaction.
func (c *Command) Rebuild(ctx context.Context, src Sources) (Run, error) {
	cutoff := c.clock.Now().Time()
	run := Run{Sources: map[string]SourceState{}, Counts: map[string]int64{}, Cutoff: cutoff}
	var err error
	if run.RunID, err = ids.NewUUID(); err != nil {
		return run, err
	}
	tables := map[string]*table{
		"case_fact": {name: "reporting.case_fact", cols: []string{"case_id", "workflow_type", "state_namespace", "state_raw", "state_mapping",
			"customer_link_state", "presence", "source_spec", "source_key", "alias_count", "first_seen_at", "original_owner_resolution",
			"original_owner_key", "original_seller_resolution", "original_seller_key", "credit_owner_resolution"}},
		"custody_fact": {name: "reporting.custody_fact", cols: []string{"case_id", "derivation", "parity", "derived_key", "legacy_key",
			"event_count", "finding_count", "presence"}},
		"invoice_fact": {name: "reporting.invoice_fact", cols: []string{"invoice_id", "source_spec", "source_key", "code_raw", "presence",
			"customer_link_state", "case_link_state", "seller_key", "main_raw", "main_mapping", "invoice_raw", "invoice_mapping", "payment_raw",
			"payment_mapping", "workflow_raw", "workflow_mapping", "return_raw", "return_mapping", "unknown_state_count", "item_count",
			"stage_count", "items_complete", "stages_complete", "first_seen_at"}},
		"invoice_unit_fact": {name: "reporting.invoice_unit_fact", cols: []string{"invoice_id", "unit", "declared_final", "declared_payment",
			"item_sum", "item_state", "stage_sum", "stage_state", "payment_sum", "payment_state", "total_class"}},
		"payment_fact": {name: "reporting.payment_fact", cols: []string{"attempt_id", "invoice_id", "invoice_link_state", "channel", "state_raw",
			"state_mapping", "state_target", "approval_provenance", "amount_raw", "amount", "unit", "amount_repr", "gateway_state", "presence",
			"first_seen_at"}},
		"source_aggregate": {name: "reporting.source_aggregate", cols: []string{"source", "key", "value"}},
	}
	note := func(name string, rows int64, status, reason string) {
		at := c.clock.Now().Time()
		r := rows
		run.Sources[name] = SourceState{Status: status, Rows: &r, ReadAt: &at, Reason: reason}
	}
	missing := func(name string) {
		run.Sources[name] = SourceState{Status: domain.Missing, Reason: "published source not configured for this run"}
	}
	failed := func(name string, err error) {
		at := c.clock.Now().Time()
		status := domain.Unknown
		if errors.Is(err, casesapi.ErrTooManyFacts) || errors.Is(err, custodyapi.ErrTooManyFacts) || errors.Is(err, billingapi.ErrTooManyFacts) ||
			errors.Is(err, paymentsapi.ErrTooManyFacts) {
			status = domain.Partial
		}
		run.Sources[name] = SourceState{Status: status, ReadAt: &at, Reason: "source read failed: " + err.Error()}
	}

	// cases
	if src.Cases == nil {
		missing(SourceCases)
	} else {
		var rows []casesapi.CaseFact
		if err := src.Cases.CaseFacts(ctx, func(f casesapi.CaseFact) error { rows = append(rows, f); return nil }); err != nil {
			failed(SourceCases, err)
		} else {
			t := tables["case_fact"]
			for _, f := range rows {
				facet := func(name string) (any, any) {
					a, ok := f.Attribution[name]
					if !ok {
						return "NOT_CAPTURED", nil
					}
					return a.Resolution, str(a.SubjectKey)
				}
				oo, ook := facet("ORIGINAL_OWNER")
				os, osk := facet("ORIGINAL_SELLER")
				co, _ := facet("CREDIT_OWNER")
				t.add(f.CaseID, f.Workflow, f.StateNamespace, str(f.StateRaw), f.StateMapping, f.CustomerLinkState, f.Presence, f.SourceSpec,
					f.SourceKey, f.AliasCount, f.FirstSeenAt.UTC(), oo, ook, os, osk, co)
			}
			note(SourceCases, int64(len(rows)), domain.Available, "published cases FactSource read in one snapshot")
		}
	}
	// custody
	if src.Custody == nil {
		missing(SourceCustody)
	} else {
		var rows []custodyapi.AssignmentFact
		if err := src.Custody.AssignmentFacts(ctx, func(f custodyapi.AssignmentFact) error { rows = append(rows, f); return nil }); err != nil {
			failed(SourceCustody, err)
		} else {
			t := tables["custody_fact"]
			for _, f := range rows {
				t.add(f.CaseID, f.Derivation, f.Parity, str(f.DerivedKey), str(f.LegacyKey), f.EventCount, f.FindingCount, f.Presence)
			}
			note(SourceCustody, int64(len(rows)), domain.Available, "published custody FactSource read in one snapshot")
		}
	}
	// payments (attempt facts; accepted sums feed billing unit totals)
	var evidence map[string]billingapi.PaymentEvidence
	if src.Payments == nil || src.PayReader == nil {
		missing(SourcePayments)
	} else {
		var rows []paymentsapi.AttemptFact
		if err := src.Payments.AttemptFacts(ctx, func(f paymentsapi.AttemptFact) error { rows = append(rows, f); return nil }); err != nil {
			failed(SourcePayments, err)
		} else {
			t := tables["payment_fact"]
			for _, f := range rows {
				t.add(f.AttemptID, str(f.InvoiceID), f.InvoiceLink, f.Channel, str(f.StateRaw), f.StateMapping, str(f.StateTarget), f.Provenance,
					str(f.AmountRaw), f.Amount, str(f.Unit), f.AmountRepr, str(f.GatewayState), f.Presence, f.FirstSeenAt.UTC())
			}
			note(SourcePayments, int64(len(rows)), domain.Available, "published payments FactSource read in one snapshot")
			evidence = map[string]billingapi.PaymentEvidence{}
		}
	}
	// billing (+ per-unit total evidence using the payments published sums)
	if src.Billing == nil {
		missing(SourceBilling)
	} else {
		var rows []billingapi.InvoiceFact
		if err := src.Billing.InvoiceFacts(ctx, func(f billingapi.InvoiceFact) error { rows = append(rows, f); return nil }); err != nil {
			failed(SourceBilling, err)
		} else {
			ids := make([]string, 0, len(rows))
			for _, f := range rows {
				ids = append(ids, f.InvoiceID)
			}
			status, reason := domain.Available, "published billing FactSource read in one snapshot"
			if evidence != nil && len(ids) > 0 {
				sums, err := src.PayReader.AcceptedSums(ctx, ids)
				if err != nil {
					status, reason = domain.Partial, "payment evidence for unit totals unavailable: "+err.Error()
				} else {
					for id, p := range sums {
						e := billingapi.PaymentEvidence{Units: map[string]billingapi.PaymentUnitSum{}, Complete: p.Complete}
						for u, s := range p.Units {
							e.Units[u] = billingapi.PaymentUnitSum{Amount: s.Amount, State: s.State, Rows: s.Accepted}
						}
						evidence[id] = e
					}
				}
			} else if evidence == nil {
				reason += "; payments source missing: payment sums are INCOMPLETE"
			}
			totals, err := src.Billing.InvoiceUnitTotals(ctx, ids, evidence)
			if err != nil {
				status, reason = domain.Partial, "unit totals unavailable: "+err.Error()
			}
			t, u := tables["invoice_fact"], tables["invoice_unit_fact"]
			for _, f := range rows {
				st := func(field billingapi.StateField) (any, any) {
					s, ok := f.States[field]
					if !ok {
						return nil, "EMPTY"
					}
					return str(s.Raw), s.Mapping
				}
				mr, mm := st(billingapi.FieldMain)
				ir, im := st(billingapi.FieldInvoice)
				pr, pm := st(billingapi.FieldPayment)
				wr, wm := st(billingapi.FieldWorkflow)
				rr, rm := st(billingapi.FieldReturn)
				t.add(f.InvoiceID, f.SourceSpec, f.SourceKey, str(f.CodeRaw), f.Presence, f.CustomerLink, f.CaseLink, str(f.SellerKey),
					mr, mm, ir, im, pr, pm, wr, wm, rr, rm, f.UnknownStates, f.ItemCount, f.StageCount, f.ItemsComplete, f.StagesComplete,
					f.FirstSeenAt.UTC())
				if f.Presence == "PRESENT" && (!f.ItemsComplete || !f.StagesComplete) && status == domain.Available {
					status, reason = domain.Partial, "billing reports an incomplete item or stage source for present invoices"
				}
				for _, x := range totals[f.InvoiceID] {
					u.add(f.InvoiceID, x.Unit, x.Declared["FINAL_TOTAL"], x.Declared["PAYMENT_TOTAL_AMOUNT"], x.ItemSum.Amount, x.ItemSum.State,
						x.StageSum.Amount, x.StageSum.State, x.PaySum.Amount, x.PaySum.State, x.Class)
				}
			}
			note(SourceBilling, int64(len(rows)), status, reason)
		}
	}
	// customers (aggregate evidence only: resolution states and phone-only
	// candidates — never a merge)
	if src.Customers == nil {
		missing(SourceCustomers)
	} else if r, err := src.Customers.ResolutionReport(ctx); err != nil {
		failed(SourceCustomers, err)
	} else {
		t := tables["source_aggregate"]
		agg := map[string]int64{"customers_present": r.CustomersPresent, "subjects_present": r.SubjectsPresent,
			"phone_only_candidates": r.PhoneOnlyCandidates, "overlapping_phone_conflicts": r.OverlappingPhoneConflict,
			"duplicate_external_id_conflicts": r.DuplicateExternalID, "phone_only_resolved": r.PhoneOnlyResolved, "merges": r.Merges}
		for st, n := range r.ByState {
			agg["state_"+st] = n
		}
		for k, v := range agg {
			t.add(SourceCustomers, k, v)
		}
		note(SourceCustomers, r.SubjectsPresent, domain.Available, "published customers resolution report")
	}

	run.Outcome = result.Success
	for _, name := range AllSources() {
		if s := run.Sources[name]; s.Status != domain.Available {
			run.Outcome = result.Partial
		}
	}
	if run.Outcome == result.Partial {
		run.Reasons = []reasoncodes.Code{reasoncodes.CoverageIncomplete}
	} else {
		run.Reasons = []reasoncodes.Code{}
	}
	names := make([]string, 0, len(tables))
	for n := range tables {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		t := tables[n]
		run.Counts[n] = int64(len(t.rows))
		enc := make([]string, 0, len(t.rows))
		for _, r := range t.rows {
			b, err := json.Marshal(r)
			if err != nil {
				return run, err
			}
			enc = append(enc, string(b))
		}
		sort.Strings(enc)
		fmt.Fprintf(h, "%s\n%s\n", n, strings.Join(enc, "\n"))
	}
	run.FactsSHA = h.Sum(nil)
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if err := registerContracts(ctx, tx, c.clock.Now().Time()); err != nil {
			return err
		}
		run.Completed = c.clock.Now().Time()
		reasons, _ := json.Marshal(run.Reasons)
		counts, _ := json.Marshal(run.Counts)
		if _, err := tx.Exec(ctx, `INSERT INTO reporting.projection_run (run_id, projection_version, source_cutoff, started_at, completed_at,
			outcome, reasons, fact_counts, facts_sha256) VALUES ($1, $2, $3, $3, $4, $5, $6, $7, $8)`,
			run.RunID, domain.ProjectionVersion, cutoff, run.Completed, run.Outcome.String(), reasons, counts, run.FactsSHA); err != nil {
			return err
		}
		for _, name := range AllSources() {
			s := run.Sources[name]
			if _, err := tx.Exec(ctx, `INSERT INTO reporting.projection_source (run_id, source, status, rows_read, read_at, reason)
				VALUES ($1, $2, $3, $4, $5, $6)`, run.RunID, name, s.Status, s.Rows, s.ReadAt, s.Reason); err != nil {
				return err
			}
		}
		for _, n := range names {
			if err := insert(ctx, tx, run.RunID, tables[n]); err != nil {
				return err
			}
		}
		return nil
	})
	return run, err
}

// insert writes a table's rows in bounded multi-row statements.
func insert(ctx context.Context, tx db.Tx, runID string, t *table) error {
	const batch = 200
	cols := append([]string{"run_id"}, t.cols...)
	for start := 0; start < len(t.rows); start += batch {
		end := min(start+batch, len(t.rows))
		var ph []string
		args := []any{}
		for _, r := range t.rows[start:end] {
			row := append([]any{runID}, r...)
			p := make([]string, len(row))
			for i, v := range row {
				args = append(args, v)
				p[i] = fmt.Sprintf("$%d", len(args))
			}
			ph = append(ph, "("+strings.Join(p, ", ")+")")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO `+t.name+` (`+strings.Join(cols, ", ")+`) VALUES `+strings.Join(ph, ", "), args...); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
	}
	return nil
}

// registerContracts records every registry contract version once; a version
// whose meaning changed is refused (history is never rewritten).
func registerContracts(ctx context.Context, tx db.Tx, now time.Time) error {
	for _, c := range domain.Registry() {
		var sha []byte
		err := tx.QueryRow(ctx, `SELECT contract_sha256 FROM reporting.metric_contract WHERE metric_id = $1 AND version = $2`, c.ID, c.Version).Scan(&sha)
		if err == nil {
			if string(sha) != string(c.SHA256()) {
				return fmt.Errorf("%w: %s v%d", ErrContractChanged, c.ID, c.Version)
			}
			continue
		}
		status, reason := c.Status()
		var r any
		if reason != "" {
			r = reason
		}
		if _, err := tx.Exec(ctx, `INSERT INTO reporting.metric_contract (metric_id, version, name, grain, source, cohort, time_basis, scope,
			formula, unit, freshness_sla_seconds, coverage, status, status_reason, contract_sha256, registered_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
			c.ID, c.Version, c.Name, c.Grain, c.Source, c.Cohort, string(c.PrimaryBasis()), c.Scope, c.Formula, c.Unit,
			int(c.FreshnessSLA/time.Second), c.Coverage, status, r, c.SHA256(), now); err != nil {
			return err
		}
	}
	return nil
}
