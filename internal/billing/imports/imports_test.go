package imports_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/imports"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// All data is SYNTHETIC.
const (
	b1        = "11111111-1111-4111-8111-111111111111"
	b2        = "22222222-2222-4222-8222-222222222222"
	invSpec   = "synthetic.invoice.v2"
	itemSpec  = "synthetic.invoice_item.v1"
	stageSpec = "synthetic.invoice_payment_stage.v1"
)

type env struct {
	*pgtest.Env
	admin *pgx.Conn
	cmd   *imports.Command
	q     *query.Service
	r     *db.Runner
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, billingmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	r := e.Runner(t, billing.Module)
	return &env{Env: e, admin: admin, cmd: imports.New(r, timeutil.FixedClock{At: at}), q: query.New(r), r: r}
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func (e *env) fingerprint(t *testing.T) string {
	t.Helper()
	var s string
	if err := e.admin.QueryRow(context.Background(), `SELECT md5(
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY invoice_id), '') FROM billing.invoice t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY invoice_id, field), '') FROM billing.invoice_state_evidence t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY invoice_id, kind), '') FROM billing.amount_evidence t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY item_id), '') FROM billing.invoice_item t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY stage_id), '') FROM billing.payment_stage t))`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func src(batch, spec, key, content string) api.SourceRef {
	sum := sha256.Sum256([]byte(spec + "/" + key + "/" + content))
	return api.SourceRef{System: "wp", Spec: spec, Key: key, BatchID: batch, RowSHA256: sum[:]}
}

func str(s string) *string { return &s }

func irt(raw string) api.RawMoney { return api.RawMoney{Raw: str(raw), Unit: "IRT"} }

func state(f api.StateField, ns, raw string, m api.Vocabulary) api.StateEvidence {
	return api.StateEvidence{Field: f, Namespace: ns, Raw: str(raw), Mapping: m}
}

func invoice(batch, key, final string, main string) api.InvoiceRecord {
	mainMap := api.KnownVocabulary
	if main == "mystery_status" {
		mainMap = api.UnknownValue
	}
	return api.InvoiceRecord{Source: src(batch, invSpec, key, final+main), CodeRaw: str("INV-" + key),
		Seller: &api.UserRef{LegacyKey: "7"},
		States: []api.StateEvidence{
			state(api.FieldMain, "legacy_invoice.status", main, mainMap),
			state(api.FieldInvoice, "legacy_invoice.invoice_status", "pre_invoice", api.KnownVocabulary),
			state(api.FieldPayment, "legacy_invoice.payment_status", "awaiting_payment", api.KnownVocabulary),
			state(api.FieldWorkflow, "legacy_invoice.payment_workflow_status", "awaiting_payment", api.KnownVocabulary),
			{Field: api.FieldReturn, Namespace: "legacy_invoice.financial_return_state", Mapping: api.EmptyValue},
			state(api.FieldLegacyDisplay, "legacy_invoice.display_status", main, api.LegacyDerived),
		},
		Declared: map[api.AmountKind]api.RawMoney{"FINAL_TOTAL": irt(final), "PAYMENT_TOTAL_AMOUNT": irt(final)},
		Customer: api.Link{State: "UNKNOWN", Basis: "no published customer reference"},
		Case:     api.Link{State: "UNKNOWN", Basis: "no explicit lead reference"},
	}
}

func item(batch, key, inv string, total api.RawMoney) api.ItemRecord {
	return api.ItemRecord{Source: src(batch, itemSpec, key, *total.Raw+total.Unit), Invoice: api.RowRef{Spec: invSpec, Key: inv},
		TitleSnapshot: str("Synthetic product"), LineTotal: total, UnitPrice: total}
}

func stage(batch, key, inv string, req api.RawMoney, raw string, target *string) api.StageRecord {
	return api.StageRecord{Source: src(batch, stageSpec, key, *req.Raw+raw), Invoice: api.RowRef{Spec: invSpec, Key: inv},
		StageNoRaw: str(key), Requested: req, StateNamespace: "legacy_invoice_payment_stage.status", StateRaw: str(raw), StateTarget: target}
}

func base(batch string) api.BillingImport {
	return api.BillingImport{BatchID: batch, Complete: map[string]bool{invSpec: true, itemSpec: true, stageSpec: true},
		ItemsComplete: true, StagesComplete: true,
		Invoices: []api.InvoiceRecord{
			invoice(batch, "1", "1000.00", "pending"),        // all equal
			invoice(batch, "2", "1000.00", "mystery_status"), // mismatching items, unknown status
			invoice(batch, "3", "500", "paid"),               // mixed units
		},
		Items: []api.ItemRecord{
			item(batch, "11", "1", irt("1000.00")),
			item(batch, "21", "2", irt("900")),
			item(batch, "31", "3", irt("500")),
			item(batch, "32", "3", api.RawMoney{Raw: str("5000"), Unit: "IRR"}),
		},
		Stages: []api.StageRecord{
			stage(batch, "1", "1", irt("1000"), "approved", str("ACCEPTED")),
			stage(batch, "2", "2", irt("1000"), "weird_stage", nil),
			stage(batch, "3", "3", irt("500"), "pending", str("PENDING")),
		},
	}
}

func (e *env) run(t *testing.T, in api.BillingImport) api.ImportReport {
	t.Helper()
	in.SnapshotID, in.SourceSystem = "synthetic-billing-01", "wp"
	rep, err := e.cmd.ImportBilling(context.Background(), in)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return rep
}

func (e *env) invoiceID(t *testing.T, key string) string {
	t.Helper()
	var id string
	if err := e.admin.QueryRow(context.Background(), `SELECT invoice_id::text FROM billing.invoice WHERE source_key = $1`, key).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *env) totals(t *testing.T, key string, pay map[string]domain.Sum, complete bool) map[string]query.UnitTotals {
	t.Helper()
	ts, err := e.q.Totals(context.Background(), e.invoiceID(t, key), pay, complete)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]query.UnitTotals{}
	for _, u := range ts {
		out[u.Unit] = u
	}
	return out
}

func amt(n int64) *int64 { return &n }

// Unknown statuses are preserved raw with an exact count; every legacy state
// field stays separate; the invoice lifecycle is never claimed (TAD #20).
func TestStatesPreservedAndSeparate(t *testing.T) {
	e := setup(t)
	if rep := e.run(t, base(b1)); rep.Outcome != result.Success {
		t.Fatalf("import: %+v", rep)
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice_state_evidence WHERE mapping = 'UNKNOWN' AND raw_value = 'mystery_status'`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.invoice_state_evidence WHERE mapping = 'UNKNOWN'`) != 1 {
		t.Fatal("unknown invoice state must be preserved raw, exactly once")
	}
	if e.count(t, `SELECT count(*) FROM billing.payment_stage WHERE state_mapping = 'UNKNOWN' AND state_raw = 'weird_stage' AND state_target IS NULL`) != 1 {
		t.Fatal("unknown stage state must be preserved raw with no target")
	}
	if e.count(t, `SELECT count(DISTINCT field) FROM billing.invoice_state_evidence`) != 6 ||
		e.count(t, `SELECT count(*) FROM billing.invoice_state_evidence WHERE mapping = 'LEGACY_DERIVED' AND field <> 'LEGACY_DISPLAY'`) != 0 {
		t.Fatal("state fields must stay separate; LEGACY_DERIVED only for the display field")
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice WHERE lifecycle_state <> 'UNKNOWN'`) != 0 {
		t.Fatal("no canonical invoice status may be claimed")
	}
	// The DB itself refuses a claimed lifecycle.
	if _, err := e.admin.Exec(context.Background(), `UPDATE billing.invoice SET lifecycle_state = 'PAID'`); err == nil {
		t.Fatal("lifecycle CHECK must reject a claimed state")
	}
	cov, err := e.q.Coverage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cov.UnknownRawStates["MAIN"]["mystery_status"] != 1 || cov.UnknownStageRaw["weird_stage"] != 1 || cov.PreInvoiceRawSeen != 3 {
		t.Fatalf("coverage: %+v", cov)
	}
	v, err := e.q.GetInvoice(context.Background(), e.invoiceID(t, "2"))
	if err != nil || v.UnknownStates != 2 || len(v.States) != 6 {
		t.Fatalf("invoice view: %+v %v", v, err)
	}
}

// Totals per unit: all equal → MATCH; one mismatch → MISMATCH; mixed units
// are never summed together; IRR and IRT stay separate; incomplete sources
// and unknown payment evidence are INCOMPLETE, never zero.
func TestTotalsPerUnit(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	paid := map[string]domain.Sum{"IRT": {Unit: "IRT", Amount: amt(1000), State: domain.SumComplete, Rows: 1}}
	t1 := e.totals(t, "1", paid, true)
	if len(t1) != 1 || t1["IRT"].Class != domain.Match || *t1["IRT"].ItemSum.Amount != 1000 || *t1["IRT"].StageSum.Amount != 1000 {
		t.Fatalf("all equal: %+v", t1)
	}
	t2 := e.totals(t, "2", map[string]domain.Sum{}, true)
	if t2["IRT"].Class != domain.Mismatch || t2["IRT"].PaymentSum.State != domain.SumNone {
		t.Fatalf("mismatch: %+v", t2)
	}
	t3 := e.totals(t, "3", map[string]domain.Sum{}, true)
	if len(t3) != 2 || *t3["IRR"].ItemSum.Amount != 5000 || *t3["IRT"].ItemSum.Amount != 500 {
		t.Fatalf("units must stay separate: %+v", t3)
	}
	if t3["IRR"].Declared != nil || t3["IRR"].Class != domain.Unknown {
		t.Fatalf("IRR has a single evidence: %+v", t3["IRR"])
	}
	// Incomplete payment evidence: INCOMPLETE, never NONE or zero.
	t4 := e.totals(t, "1", paid, false)
	if t4["IRT"].PaymentSum.State != domain.SumIncomplete || t4["IRT"].Class != domain.Incomplete {
		t.Fatalf("incomplete payments: %+v", t4)
	}
	// Incomplete item source → INCOMPLETE.
	in := base(b2)
	in.ItemsComplete = false
	in.Complete[itemSpec] = false
	if rep := e.run(t, in); rep.Outcome != result.Partial {
		t.Fatalf("incomplete source must be PARTIAL: %+v", rep)
	}
	if t5 := e.totals(t, "1", paid, true); t5["IRT"].ItemSum.State != domain.SumIncomplete || t5["IRT"].Class != domain.Incomplete {
		t.Fatalf("incomplete items: %+v", t5)
	}
}

// A fractional toman amount is UNREPRESENTABLE (never rounded) and makes
// the comparison INCOMPLETE; overflow is never wrapped.
func TestUnrepresentableAndOverflow(t *testing.T) {
	e := setup(t)
	in := base(b1)
	in.Items[0] = item(b1, "11", "1", irt("999.50"))
	// Ten 18-digit lines exceed int64 together (each is representable).
	in.Items[1] = item(b1, "21", "2", irt("999999999999999999"))
	for k := 22; k < 31; k++ {
		in.Items = append(in.Items, item(b1, strconv.Itoa(k), "2", irt("999999999999999999")))
	}
	e.run(t, in)
	if e.count(t, `SELECT count(*) FROM billing.invoice_item WHERE line_total_repr = 'UNREPRESENTABLE' AND line_total_raw = '999.50' AND line_total IS NULL`) != 1 {
		t.Fatal("fractional amount must stay raw and unrepresentable")
	}
	if t1 := e.totals(t, "1", map[string]domain.Sum{}, true); t1["IRT"].Class != domain.Incomplete {
		t.Fatalf("unrepresentable: %+v", t1)
	}
	if t2 := e.totals(t, "2", map[string]domain.Sum{}, true); t2["IRT"].ItemSum.State != domain.SumOverflow || t2["IRT"].ItemSum.Amount != nil {
		t.Fatalf("overflow: %+v", t2)
	}
}

// Replay creates nothing; a changed row refreshes with a version bump;
// only a COMPLETE source marks rows absent; nothing is deleted.
func TestReplayRefreshAbsence(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	fp := e.fingerprint(t)
	if r := e.run(t, base(b1)); !r.Replay || e.fingerprint(t) != fp {
		t.Fatalf("replay changed state: %+v", r)
	}
	in := base(b2)
	in.Invoices[0] = invoice(b2, "1", "1200", "pending")
	in.Items = in.Items[1:] // item 11 gone
	r := e.run(t, in)
	if r.Replay || r.Counts["invoices_refreshed"] != 1 || r.Counts["items_marked_absent"] != 1 {
		t.Fatalf("refresh/absence: %+v", r)
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice WHERE source_key = '1' AND version = 2`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.amount_evidence a JOIN billing.invoice i USING (invoice_id) WHERE i.source_key = '1' AND a.kind = 'FINAL_TOTAL' AND a.amount = 1200`) != 1 {
		t.Fatal("refresh must update evidence and bump version")
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice_item WHERE source_key = '11' AND presence = 'ABSENT_IN_LATEST'`) != 1 {
		t.Fatal("absent item must be kept and flagged")
	}
	// An incomplete source never marks absence.
	in3 := base("33333333-3333-4333-8333-333333333333")
	in3.Complete = map[string]bool{invSpec: false, itemSpec: false, stageSpec: false}
	in3.Invoices, in3.Items, in3.Stages = in3.Invoices[:1], nil, nil
	if r := e.run(t, in3); r.Outcome != result.Partial || e.count(t, `SELECT count(*) FROM billing.invoice WHERE presence = 'ABSENT_IN_LATEST'`) != 0 {
		t.Fatalf("incomplete must not mark absence: %+v", r)
	}
	if _, err := e.admin.Exec(context.Background(), `SET ROLE crm_owner_billing; DELETE FROM billing.invoice_item`); err == nil {
		t.Fatal("billing owner must not hold DELETE")
	}
}

// Read paths run read-only; writes from a read-only transaction fail.
func TestReadPathIsReadOnly(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	err := e.r.InTx(context.Background(), db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		_, err := store.UpsertStage(ctx, tx, e.invoiceID(t, "1"), base(b1).Stages[0], timeNow())
		return err
	})
	if !errors.Is(err, store.ErrReadOnly) {
		t.Fatalf("read-only tx must refuse writes: %v", err)
	}
}

// Pagination continues beyond one page for invoices, items and stages.
func TestPaginationBeyondPageSize(t *testing.T) {
	e := setup(t)
	in := base(b1)
	for i := 0; i < 5; i++ {
		k := string(rune('4' + i))
		in.Invoices = append(in.Invoices, invoice(b1, k, "10", "pending"))
		in.Items = append(in.Items, item(b1, "1"+k+"0", "1", irt("0")))
		in.Stages = append(in.Stages, stage(b1, "1"+k+"0", "1", irt("0"), "pending", str("PENDING")))
	}
	e.run(t, in)
	two := 2
	seen := 0
	var cur *string
	for {
		p, err := e.q.ListInvoices(context.Background(), store.InvoiceFilter{}, cur, &two)
		if err != nil {
			t.Fatal(err)
		}
		seen += len(p.Invoices)
		if p.NextCursor == nil {
			break
		}
		cur = p.NextCursor
	}
	if seen != 8 {
		t.Fatalf("invoices seen %d", seen)
	}
	id := e.invoiceID(t, "1")
	items, stages := 0, 0
	cur = nil
	for {
		p, err := e.q.Items(context.Background(), id, cur, &two)
		if err != nil {
			t.Fatal(err)
		}
		items += len(p.Items)
		if cur = p.NextCursor; cur == nil {
			break
		}
	}
	for {
		p, err := e.q.Stages(context.Background(), id, cur, &two)
		if err != nil {
			t.Fatal(err)
		}
		stages += len(p.Stages)
		if cur = p.NextCursor; cur == nil {
			break
		}
	}
	if items != 6 || stages != 6 {
		t.Fatalf("items %d stages %d", items, stages)
	}
	unknown := true
	p, err := e.q.ListInvoices(context.Background(), store.InvoiceFilter{HasUnknownStatus: &unknown}, nil, nil)
	if err != nil || len(p.Invoices) != 1 {
		t.Fatalf("has_unknown_status filter: %v %v", p, err)
	}
}

func timeNow() time.Time { return time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC) }

// A link resolved after the first import (customers/cases imported later)
// is refreshed by a replay even though the source row is unchanged.
func TestReplayRefreshesResolvedLinks(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	in := base(b1)
	cust, cs := "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	in.Invoices[0].Customer = api.Link{ID: &cust, State: "RESOLVED", Basis: "subject customer"}
	in.Invoices[0].Case = api.Link{ID: &cs, State: "RESOLVED", Basis: "lead case"}
	r := e.run(t, in)
	if r.Replay || r.Counts["invoices_refreshed"] != 1 {
		t.Fatalf("link refresh: %+v", r)
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice WHERE source_key = '1' AND customer_id = $1 AND case_id = $2 AND customer_link_state = 'RESOLVED' AND version = 2`, cust, cs) != 1 {
		t.Fatal("resolved links must be refreshed on an unchanged row")
	}
	fp := e.fingerprint(t)
	if r := e.run(t, in); !r.Replay || e.fingerprint(t) != fp {
		t.Fatalf("second replay must change nothing: %+v", r)
	}
}
