package query_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	custodyapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	customersapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
	reportingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/projections"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/query"
)

// All data is SYNTHETIC.

func id(n int) string    { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
func s(v string) *string { return &v }
func i(v int64) *int64   { return &v }

type clock struct{ at time.Time }

func (c *clock) Now() timeutil.Instant { v, _ := timeutil.NewInstant(c.at); return v }

// stubs are the published owner FactSources (no owner schema involved).
type stubs struct {
	cases    []casesapi.CaseFact
	custody  []custodyapi.AssignmentFact
	invoices []billingapi.InvoiceFact
	totals   map[string][]billingapi.UnitTotal
	attempts []paymentsapi.AttemptFact
}

func (x *stubs) CaseFacts(_ context.Context, fn func(casesapi.CaseFact) error) error {
	for _, f := range x.cases {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}
func (x *stubs) AssignmentFacts(_ context.Context, fn func(custodyapi.AssignmentFact) error) error {
	for _, f := range x.custody {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}
func (x *stubs) InvoiceFacts(_ context.Context, fn func(billingapi.InvoiceFact) error) error {
	for _, f := range x.invoices {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}
func (x *stubs) InvoiceUnitTotals(_ context.Context, _ []string, _ map[string]billingapi.PaymentEvidence) (map[string][]billingapi.UnitTotal, error) {
	return x.totals, nil
}
func (x *stubs) AttemptFacts(_ context.Context, fn func(paymentsapi.AttemptFact) error) error {
	for _, f := range x.attempts {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}
func (x *stubs) AcceptedSums(context.Context, []string) (map[string]paymentsapi.InvoicePayments, error) {
	return map[string]paymentsapi.InvoicePayments{}, nil
}
func (x *stubs) RowProvenance(context.Context, string, string, []string) (map[string]paymentsapi.RowProvenance, error) {
	return nil, nil
}
func (x *stubs) BatchImported(context.Context, string) (bool, error) { return false, nil }
func (x *stubs) Coverage(context.Context) (paymentsapi.Coverage, error) {
	return paymentsapi.Coverage{}, nil
}
func (x *stubs) ResolutionReport(context.Context) (customersapi.Resolution, error) {
	return customersapi.Resolution{CustomersPresent: 3, SubjectsPresent: 3, PhoneOnlyCandidates: 1, ByState: map[string]int64{"RESOLVED": 1, "UNKNOWN": 2}}, nil
}
func (x *stubs) SubjectProvenance(context.Context, string, customersapi.SubjectKind, []string) (map[string]customersapi.Provenance, error) {
	return nil, nil
}

var t0 = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func scenario() *stubs {
	x := &stubs{totals: map[string][]billingapi.UnitTotal{}}
	for n := 1; n <= 7; n++ {
		mapping, wf := "KNOWN", "DISTRIBUTION_ITEM"
		if n%3 == 0 {
			mapping = "UNKNOWN"
		}
		if n <= 2 {
			wf = "LEGACY_LEAD"
		}
		x.cases = append(x.cases, casesapi.CaseFact{CaseID: id(n), Workflow: wf, StateNamespace: "ns", StateRaw: s("raw"), StateMapping: mapping,
			CustomerLinkState: "UNKNOWN", Presence: "PRESENT", SourceSpec: "synthetic.case.v1", SourceKey: fmt.Sprint(n),
			FirstSeenAt: t0.Add(-time.Duration(n) * 24 * time.Hour),
			Attribution: map[string]casesapi.AttributionFact{"ORIGINAL_OWNER": {Resolution: "RESOLVED", SubjectKey: s("4")}}})
		x.custody = append(x.custody, custodyapi.AssignmentFact{CaseID: id(n), Derivation: "DERIVED", Parity: []string{"MATCH", "CONFLICT", "NOT_RECOVERABLE"}[n%3],
			Presence: "PRESENT"})
	}
	x.invoices = []billingapi.InvoiceFact{
		{InvoiceID: id(101), SourceSpec: "synthetic.invoice.v2", SourceKey: "1", Presence: "PRESENT", CustomerLink: "UNKNOWN", CaseLink: "UNKNOWN",
			States:        map[billingapi.StateField]billingapi.StateFact{billingapi.FieldMain: {Raw: s("synthetic_unfamiliar"), Mapping: "UNKNOWN"}},
			UnknownStates: 1, ItemsComplete: true, StagesComplete: true, FirstSeenAt: t0},
		{InvoiceID: id(102), SourceSpec: "synthetic.invoice.v2", SourceKey: "2", Presence: "PRESENT", CustomerLink: "CONFLICT", CaseLink: "UNKNOWN",
			ItemsComplete: true, StagesComplete: true, FirstSeenAt: t0},
	}
	x.totals[id(101)] = []billingapi.UnitTotal{{Unit: "IRT", Class: "MATCH", ItemSum: billingapi.PaymentUnitSum{Amount: i(1000), State: "COMPLETE"},
		StageSum: billingapi.PaymentUnitSum{Amount: i(1000), State: "COMPLETE"}, PaySum: billingapi.PaymentUnitSum{Amount: i(1000), State: "COMPLETE"},
		Declared: map[string]*int64{"FINAL_TOTAL": i(1000)}}}
	x.totals[id(102)] = []billingapi.UnitTotal{
		{Unit: "IRT", Class: "INCOMPLETE", ItemSum: billingapi.PaymentUnitSum{State: "INCOMPLETE"}, StageSum: billingapi.PaymentUnitSum{State: "NONE"},
			PaySum: billingapi.PaymentUnitSum{State: "NONE"}},
		{Unit: "IRR", Class: "UNKNOWN", ItemSum: billingapi.PaymentUnitSum{Amount: i(5000), State: "COMPLETE"}, StageSum: billingapi.PaymentUnitSum{State: "NONE"},
			PaySum: billingapi.PaymentUnitSum{State: "NONE"}}}
	acc, irt, irr := "ACCEPTED", "IRT", "IRR"
	x.attempts = []paymentsapi.AttemptFact{
		{AttemptID: id(201), InvoiceID: s(id(101)), InvoiceLink: "RESOLVED", Channel: "GATEWAY", StateRaw: s("approved"), StateMapping: "MAPPED",
			StateTarget: &acc, Provenance: "GATEWAY_AUTO", Amount: i(1000), Unit: &irt, AmountRepr: "REPRESENTABLE", Presence: "PRESENT", FirstSeenAt: t0},
		{AttemptID: id(202), InvoiceID: s(id(102)), InvoiceLink: "RESOLVED", Channel: "MANUAL_CARD", StateRaw: s("approved"), StateMapping: "MAPPED",
			StateTarget: &acc, Provenance: "LEGACY_MANUAL_APPROVAL", Amount: i(5000), Unit: &irr, AmountRepr: "REPRESENTABLE", Presence: "PRESENT", FirstSeenAt: t0},
		{AttemptID: id(203), InvoiceLink: "UNKNOWN", Channel: "GATEWAY", StateRaw: s("verified"), StateMapping: "UNKNOWN", Provenance: "UNKNOWN",
			Amount: i(70), Unit: &irt, AmountRepr: "REPRESENTABLE", Presence: "PRESENT", FirstSeenAt: t0},
	}
	return x
}

func (x *stubs) sources() projections.Sources {
	return projections.Sources{Cases: x, Custody: x, Billing: x, Payments: x, PayReader: x, Customers: x}
}

type env struct {
	*pgtest.Env
	admin *pgx.Conn
	clock *clock
	cmd   *projections.Command
	q     *query.Service
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, reportingmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	c := &clock{at: t0}
	r := e.Runner(t, reporting.Module)
	cmd, err := projections.New(r, c)
	if err != nil {
		t.Fatal(err)
	}
	return &env{Env: e, admin: admin, clock: c, cmd: cmd, q: query.New(r, c)}
}

func (e *env) rebuild(t *testing.T, src projections.Sources) projections.Run {
	t.Helper()
	run, err := e.cmd.Rebuild(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func (e *env) fingerprint(t *testing.T) string {
	t.Helper()
	var out string
	if err := e.admin.QueryRow(context.Background(), `SELECT md5(string_agg(t, '|' ORDER BY t)) FROM (
		SELECT 'r' || row_to_json(x)::text t FROM reporting.projection_run x UNION ALL SELECT 'c' || row_to_json(x)::text FROM reporting.case_fact x
		UNION ALL SELECT 'p' || row_to_json(x)::text FROM reporting.payment_fact x UNION ALL SELECT 'm' || row_to_json(x)::text FROM reporting.metric_contract x) y`).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func allRows(t *testing.T, q *query.Service, spec query.Spec, limit int) ([]any, []query.Result) {
	t.Helper()
	var rows []any
	var results []query.Result
	var cursor *string
	for {
		res, page, err := q.Report(context.Background(), spec, cursor, &limit)
		if err != nil {
			t.Fatal(err)
		}
		rows, results = append(rows, page.Rows...), append(results, res)
		if cursor = page.NextCursor; cursor == nil {
			return rows, results
		}
	}
}

func TestRebuildIdempotentAndComplete(t *testing.T) {
	e := setup(t)
	x := scenario()
	r1 := e.rebuild(t, x.sources())
	if r1.Outcome != result.Success || r1.Counts["case_fact"] != 7 || r1.Counts["invoice_unit_fact"] != 3 || r1.Counts["payment_fact"] != 3 {
		t.Fatalf("run: %+v", r1)
	}
	r2 := e.rebuild(t, x.sources())
	if r1.RunID == r2.RunID || string(r1.FactsSHA) != string(r2.FactsSHA) {
		t.Fatal("same sources and projection version must rebuild identical facts in a new run")
	}
	var dup int64
	if err := e.admin.QueryRow(context.Background(), `SELECT count(*) - count(DISTINCT (run_id, case_id)) FROM reporting.case_fact`).Scan(&dup); err != nil || dup != 0 {
		t.Fatalf("duplicate facts: %d %v", dup, err)
	}
	// Append-only even for the table owner.
	if _, err := e.admin.Exec(context.Background(), `UPDATE reporting.case_fact SET presence = 'X'`); err == nil {
		t.Fatal("facts must be append-only")
	}
	if _, err := e.admin.Exec(context.Background(), `DELETE FROM reporting.projection_run`); err == nil {
		t.Fatal("runs must be append-only")
	}
}

// INV-021: card total = table aggregate = union of table pages; every page
// row is a member in detail; pagination never changes the summary.
func TestCardTableDetailParity(t *testing.T) {
	e := setup(t)
	e.rebuild(t, scenario().sources())
	ctx := context.Background()
	mv, err := e.q.Metric(ctx, "cases.unknown_state", "", nil, nil, nil, nil)
	if err != nil || mv.ValueState != "VALUE" {
		t.Fatalf("metric: %+v %v", mv, err)
	}
	card := mv.Result.Summary.Total
	spec := mv.Result.Spec // the metric's own spec, pinned to its run (cutoff)
	var totals []int64
	var all []any
	for _, limit := range []int{1, 2, 50} {
		rows, results := allRows(t, e.q, spec, limit)
		for _, r := range results {
			totals = append(totals, r.Summary.Total)
		}
		all = rows
		if int64(len(rows)) != card {
			t.Fatalf("limit %d: %d rows, card %d", limit, len(rows), card)
		}
		seen := map[string]bool{}
		for _, r := range rows {
			k := r.(query.CaseRow).CaseID
			if seen[k] {
				t.Fatalf("duplicate row across pages: %s", k)
			}
			seen[k] = true
		}
	}
	for _, n := range totals {
		if n != card {
			t.Fatalf("pagination changed the total: %v vs card %d", totals, card)
		}
	}
	if card != 2 {
		t.Fatalf("expected 2 unknown-state cases, got %d", card)
	}
	for _, r := range all {
		_, row, member, err := e.q.Detail(ctx, spec, r.(query.CaseRow).CaseID)
		if err != nil || !member || row.(query.CaseRow).StateMapping != "UNKNOWN" {
			t.Fatalf("detail must explain card membership: %v %v", member, err)
		}
	}
	if _, _, member, err := e.q.Detail(ctx, spec, id(1)); err != nil || member {
		t.Fatalf("a non-member must say so in detail: %v %v", member, err)
	}
	// Filter parity: the table queried with the card's filters equals the card.
	res, _, err := e.q.Report(ctx, query.Spec{Report: "cases", Filters: map[string]string{"presence": "PRESENT", "state_mapping": "UNKNOWN"}}, nil, nil)
	if err != nil || res.Summary.Total != card {
		t.Fatalf("filter parity: %+v %v", res.Summary, err)
	}
}

// Cutoff: a pinned older run keeps its result; the current answer moves.
// EVENT_RANGE (first observation) is not conflated with current state;
// HISTORICAL_AS_OF needs a run at/before as_of; nothing is reconstructed.
func TestCutoffAndTemporalModes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	x := scenario()
	r1 := e.rebuild(t, x.sources())
	e.clock.at = t0.Add(2 * time.Hour)
	x.cases[0].StateMapping = "UNKNOWN" // the state of case 1 changes later
	r2 := e.rebuild(t, x.sources())
	cur, _, _ := e.q.Report(ctx, query.Spec{Report: "cases", Filters: map[string]string{"state_mapping": "UNKNOWN"}}, nil, nil)
	old, _, _ := e.q.Report(ctx, query.Spec{Report: "cases", RunID: &r1.RunID, Filters: map[string]string{"state_mapping": "UNKNOWN"}}, nil, nil)
	if cur.Summary.Total != 3 || old.Summary.Total != 2 || *cur.Spec.RunID != r2.RunID || *old.Spec.RunID != r1.RunID {
		t.Fatalf("cutoff: current %d old %d", cur.Summary.Total, old.Summary.Total)
	}
	// EVENT_RANGE: cohort by first observation; unchanged by the later state change.
	from, to := t0.Add(-36*time.Hour), t0
	ev, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.EventRange, From: &from, To: &to}, nil, nil)
	if err != nil || ev.Summary.Total != 1 {
		t.Fatalf("event range: %+v %v", ev.Summary, err)
	}
	// HISTORICAL_AS_OF: between the runs → r1; before any run → not available.
	asOf := t0.Add(time.Hour)
	h, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.HistoricalAsOf, AsOf: &asOf, Filters: map[string]string{"state_mapping": "UNKNOWN"}}, nil, nil)
	if err != nil || !h.Available || *h.Spec.RunID != r1.RunID || h.Summary.Total != 2 {
		t.Fatalf("as-of: %+v %v", h, err)
	}
	before := t0.Add(-time.Hour)
	if h, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.HistoricalAsOf, AsOf: &before}, nil, nil); err != nil || h.Available || h.Summary != nil {
		t.Fatalf("as-of before any run must be NOT_AVAILABLE, not zero: %+v %v", h, err)
	}
	// Custody has no trustworthy event time: EVENT_RANGE is NOT_AVAILABLE.
	if c, _, err := e.q.Report(ctx, query.Spec{Report: "custody", TimeBasis: domain.EventRange, From: &from, To: &to}, nil, nil); err != nil || c.Available {
		t.Fatalf("custody event range must be not available: %+v %v", c, err)
	}
	// Invalid combinations are rejected.
	if _, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.CurrentSnapshot, AsOf: &asOf}, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatal("as_of with CURRENT_SNAPSHOT must be invalid")
	}
	if _, _, err := e.q.Report(ctx, query.Spec{Report: "cases", Filters: map[string]string{"phone": "1"}}, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatal("non-allowlisted filter must be invalid")
	}
	if _, _, err := e.q.Report(ctx, query.Spec{Report: "cases", Scope: "ALL"}, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatal("scope cannot be broadened")
	}
}

// A missing source: run PARTIAL, metric UNKNOWN (no value), never zero; a
// metric on an available source still answers. Freshness turns STALE.
func TestPartialUnknownStale(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	x := scenario()
	src := x.sources()
	src.Custody = nil
	run := e.rebuild(t, src)
	if run.Outcome != result.Partial || run.Sources["custody"].Status != domain.Missing {
		t.Fatalf("missing source must be PARTIAL: %+v", run)
	}
	mv, err := e.q.Metric(ctx, "custody.by_parity", "", nil, nil, nil, nil)
	if err != nil || mv.ValueState != "UNKNOWN" || mv.Result.Summary != nil {
		t.Fatalf("missing source metric must be UNKNOWN with no value: %+v %v", mv, err)
	}
	if mv, _ := e.q.Metric(ctx, "cases.staged_total", "", nil, nil, nil, nil); mv.ValueState != "VALUE" || mv.Result.Summary.Total != 7 {
		t.Fatalf("available source metric: %+v", mv)
	}
	e.clock.at = t0.Add(25 * time.Hour)
	if mv, _ := e.q.Metric(ctx, "cases.staged_total", "", nil, nil, nil, nil); mv.Result.Freshness != domain.Stale || mv.Result.Summary.Total != 7 {
		t.Fatalf("old run must be STALE (value labelled, not hidden): %+v", mv.Result)
	}
	// No run at all → UNKNOWN for every metric.
	e2 := setup(t)
	if mv, _ := e2.q.Metric(ctx, "cases.staged_total", "", nil, nil, nil, nil); mv.ValueState != "UNKNOWN" || mv.Result.Summary != nil || mv.Result.Freshness != domain.Unknown {
		t.Fatalf("no run must be UNKNOWN: %+v", mv)
	}
}

// Units stay separate; unknown states stay unknown; NOT_FINAL metrics are
// never evaluated; GETs never write.
func TestMoneyStatesNotFinalReadOnly(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.rebuild(t, scenario().sources())
	fp := e.fingerprint(t)
	mv, err := e.q.Metric(ctx, "payments.accepted_attempt_amount", "", nil, nil, nil, nil)
	if err != nil || mv.ValueState != "VALUE" || len(mv.Result.Summary.Amounts) != 2 {
		t.Fatalf("accepted amount: %+v %v", mv, err)
	}
	got := map[string]int64{}
	for _, a := range mv.Result.Summary.Amounts {
		got[a.Unit] = *a.Sum
	}
	if got["IRT"] != 1000 || got["IRR"] != 5000 {
		t.Fatalf("units must stay separate and unconverted: %v", got)
	}
	st, _ := e.q.Metric(ctx, "payments.by_state", "", nil, nil, nil, nil)
	if st.Result.Summary.Counts["state"]["UNMAPPED"] != 1 || st.Result.Summary.Counts["state"]["ACCEPTED"] != 2 {
		t.Fatalf("unknown attempt state must stay visible: %+v", st.Result.Summary.Counts)
	}
	inv, _ := e.q.Metric(ctx, "invoices.total_evidence_class", "", nil, nil, nil, nil)
	if inv.Result.Summary.UnitClasses["IRT"]["MATCH"] != 1 || inv.Result.Summary.UnitClasses["IRR"]["UNKNOWN"] != 1 {
		t.Fatalf("unit classes: %+v", inv.Result.Summary.UnitClasses)
	}
	for _, m := range []string{"invoices.open_preinvoices", "cases.completion_rate", "cases.conversion_rate"} {
		mv, err := e.q.Metric(ctx, m, "", nil, nil, nil, nil)
		if err != nil || mv.Status != "NOT_FINAL" || mv.ValueState != "NOT_EVALUATED" || mv.Result.Summary != nil {
			t.Fatalf("%s must be NOT_FINAL and unevaluated: %+v %v", m, mv, err)
		}
	}
	if e.fingerprint(t) != fp {
		t.Fatal("a read changed reporting state")
	}
}

// A registered metric version cannot change meaning silently.
func TestContractVersionImmutable(t *testing.T) {
	e := setup(t)
	e.rebuild(t, scenario().sources())
	ctx := context.Background()
	if _, err := e.admin.Exec(ctx, `ALTER TABLE reporting.metric_contract DISABLE TRIGGER ALL`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.admin.Exec(ctx, `UPDATE reporting.metric_contract SET contract_sha256 = decode(repeat('00', 32), 'hex') WHERE metric_id = 'cases.staged_total'`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.admin.Exec(ctx, `ALTER TABLE reporting.metric_contract ENABLE TRIGGER ALL`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.cmd.Rebuild(ctx, scenario().sources()); !errors.Is(err, projections.ErrContractChanged) {
		t.Fatalf("a changed contract without a version bump must be refused: %v", err)
	}
	for _, c := range domain.Registry() {
		if st, reason := c.Status(); st == "NOT_FINAL" && c.Blocked == "" {
			t.Errorf("%s is NOT_FINAL without a declared reason: %s", c.ID, reason)
		}
	}
}

// failingCustody is a custody source that exceeds its published bound.
type failingCustody struct{}

func (failingCustody) AssignmentFacts(_ context.Context, fn func(custodyapi.AssignmentFact) error) error {
	_ = fn(custodyapi.AssignmentFact{CaseID: id(1), Derivation: "DERIVED", Parity: "MATCH", Presence: "PRESENT"})
	return custodyapi.ErrTooManyFacts
}

// Review fixes: a discarded read is UNKNOWN (never a partial zero); a report
// on a source without facts has no summary; absent accepted amounts make the
// amount INCOMPLETE; a resolved HISTORICAL_AS_OF spec is reusable.
func TestReviewFixes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	x := scenario()
	src := x.sources()
	src.Custody = failingCustody{}
	acc, irt := "ACCEPTED", "IRT"
	x.attempts = append(x.attempts, paymentsapi.AttemptFact{AttemptID: id(204), InvoiceLink: "UNKNOWN", Channel: "MANUAL_CARD", StateRaw: s("approved"),
		StateMapping: "MAPPED", StateTarget: &acc, Provenance: "LEGACY_MANUAL_APPROVAL", Unit: &irt, AmountRepr: "ABSENT", Presence: "PRESENT", FirstSeenAt: t0})
	run := e.rebuild(t, src)
	if run.Outcome != result.Partial || run.Sources["custody"].Status != domain.Unknown || run.Counts["custody_fact"] != 0 {
		t.Fatalf("over-bound read must be UNKNOWN with no facts: %+v", run.Sources["custody"])
	}
	if mv, _ := e.q.Metric(ctx, "custody.by_parity", "", nil, nil, nil, nil); mv.ValueState != "UNKNOWN" || mv.Result.Summary != nil {
		t.Fatalf("metric on a discarded source must be UNKNOWN: %+v", mv)
	}
	res, page, err := e.q.Report(ctx, query.Spec{Report: "custody"}, nil, nil)
	if err != nil || res.Available || res.Summary != nil || len(page.Rows) != 0 || res.Run == nil || res.SourceCoverage["custody"] != domain.Unknown {
		t.Fatalf("report on a source without facts must be UNKNOWN, not total 0: %+v %v", res, err)
	}
	if mv, _ := e.q.Metric(ctx, "payments.accepted_attempt_amount", "", nil, nil, nil, nil); mv.ValueState != "INCOMPLETE" || mv.Result.Summary.Absent != 1 {
		t.Fatalf("absent accepted amount must make the amount INCOMPLETE: %+v", mv.ValueState)
	}
	asOf := t0.Add(time.Hour)
	h, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.HistoricalAsOf, AsOf: &asOf}, nil, nil)
	if err != nil || !h.Available {
		t.Fatalf("as-of: %+v %v", h, err)
	}
	again, _, err := e.q.Report(ctx, h.Spec, nil, nil)
	if err != nil || *again.Spec.RunID != *h.Spec.RunID || again.Summary.Total != h.Summary.Total {
		t.Fatalf("a returned HISTORICAL_AS_OF spec must be reusable: %v", err)
	}
	if _, row, _, err := e.q.Detail(ctx, h.Spec, id(1)); err != nil || row == nil {
		t.Fatalf("detail with the returned as-of spec: %v", err)
	}
	early := t0.Add(-time.Hour)
	if _, _, err := e.q.Report(ctx, query.Spec{Report: "cases", TimeBasis: domain.HistoricalAsOf, AsOf: &early, RunID: h.Spec.RunID}, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatalf("a pinned run after as_of must be invalid: %v", err)
	}
}
