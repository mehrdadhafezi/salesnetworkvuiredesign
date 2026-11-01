package shadow_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	billinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/http"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/imports"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	billingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customersapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	customersimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/imports"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/imports"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	paymentsquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

type billingEnv struct {
	*salesEnv
	custIn    *customersimports.Command
	custQ     *customersquery.Service
	billingQ  *billingquery.Service
	paymentsQ *paymentsquery.Service
	billIn    *billingimports.Command
	payIn     *paymentsimports.Command
}

func setupBilling(t *testing.T) *billingEnv {
	t.Helper()
	e := setupSales(t)
	for _, s := range []migrations.Source{customersmigrations.Source(), billingmigrations.Source(), paymentsmigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, s); err != nil {
			t.Fatalf("billing migrations: %v", err)
		}
	}
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	cr, br, pr := e.Runner(t, customers.Module), e.Runner(t, billing.Module), e.Runner(t, payments.Module)
	bq := billingquery.New(br)
	return &billingEnv{salesEnv: e, custIn: customersimports.New(cr, clock), custQ: customersquery.New(cr), billingQ: bq,
		paymentsQ: paymentsquery.New(pr), billIn: billingimports.New(br, clock), payIn: paymentsimports.New(pr, bq, clock)}
}

// importBillingOnly resolves users, customer and Case links through the
// published identity, customers and cases APIs, then imports billing in a
// billing-only transaction.
func (e *billingEnv) importBillingOnly(t *testing.T, b *shadow.Batch) result.Status {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.BillingUserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	prov, err := e.custQ.SubjectProvenance(ctx, shadow.SourceSystem, customersapi.IssuedTo, shadow.InvoiceKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	custs := map[string]string{}
	for k, p := range prov {
		if p.Present {
			custs[k] = p.CustomerID
		}
	}
	srcs, err := e.casesQ.ResolveSources(ctx, shadow.SourceSystem, shadow.SpecLead, shadow.InvoiceLeadKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{}
	for k, s := range srcs {
		cases[k] = s.CaseID
	}
	in, _ := shadow.BillingCommand(b, users, custs, cases)
	rep, err := e.billIn.ImportBilling(ctx, in)
	if err != nil {
		t.Fatalf("billing import: %v", err)
	}
	return rep.Outcome
}

// importPaymentsOnly imports payments; invoices resolve via billing's API.
func (e *billingEnv) importPaymentsOnly(t *testing.T, b *shadow.Batch) (result.Status, bool) {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.BillingUserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	in, _ := shadow.PaymentsCommand(b, users)
	rep, err := e.payIn.ImportPayments(ctx, in)
	if err != nil {
		t.Fatalf("payments import: %v", err)
	}
	return rep.Outcome, rep.Replay
}

func billingSource(c snapshot.Completeness) []snapshot.EntityRows {
	ents := append(legacytest.HRScenario(), legacytest.SalesScenario(c)...)
	cust := legacytest.CustomerScenario(c)
	cust[0].Rows = nil
	cust = legacytest.With(cust, "legacy.invoice_issued_to.v1", []legacytest.Rec{
		legacytest.IssuedTo("2001", "", "Synthetic Customer D", "09000000901", "", "2025-03-01 10:00:00")}, c)
	ents = append(ents, cust...)
	return append(ents, legacytest.BillingScenario(c)...)
}

func (e *billingEnv) prepare(t *testing.T, id string, ents []snapshot.EntityRows) *shadow.Batch {
	t.Helper()
	b := e.stage(t, id, ents)
	e.importAll(t, b)
	e.importSales(t, b)
	in, _ := shadow.CustomersCommand(b)
	if _, err := e.custIn.ImportCustomers(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBillingShadowEndToEnd(t *testing.T) {
	e := setupBilling(t)
	ctx := context.Background()
	b := e.prepare(t, "syn-billing-001", billingSource(snapshot.Complete))
	if o := e.importBillingOnly(t, b); o != result.Success {
		t.Fatalf("billing import: %s", o)
	}
	if o, _ := e.importPaymentsOnly(t, b); o != result.Partial {
		t.Fatalf("payments import must be PARTIAL (payment 9 names an unstaged invoice): %s", o)
	}
	// Unknown statuses preserved raw, exact counts; fields separate.
	if e.count(t, `SELECT count(*) FROM billing.invoice_state_evidence WHERE field = 'MAIN' AND mapping = 'UNKNOWN' AND raw_value = 'synthetic_unfamiliar_status'`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.payment_stage WHERE state_mapping = 'UNKNOWN' AND state_raw = 'synthetic_stage_status'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_mapping = 'UNKNOWN' AND state_raw = 'verified'`) != 1 {
		t.Fatal("unknown states must be preserved raw")
	}
	if e.count(t, `SELECT count(*) FROM billing.invoice WHERE lifecycle_state <> 'UNKNOWN'`) != 0 ||
		e.count(t, `SELECT count(*) FROM billing.invoice_state_evidence WHERE field = 'LEGACY_DISPLAY' AND mapping = 'LEGACY_DERIVED'`) != 4 {
		t.Fatal("no canonical invoice status; legacy display is LEGACY_DERIVED evidence")
	}
	// Links: customer only via the Slice 5 subject; Case via lead_id.
	if e.count(t, `SELECT count(*) FROM billing.invoice WHERE source_key = '2001' AND customer_link_state = 'RESOLVED' AND customer_id IS NOT NULL
		AND case_link_state = 'RESOLVED' AND case_id IS NOT NULL`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.invoice WHERE customer_link_state = 'UNKNOWN' AND customer_id IS NULL`) != 3 {
		t.Fatal("explicit links only")
	}
	// Money: Toman int64, fractional unrepresentable, never converted.
	if e.count(t, `SELECT count(*) FROM billing.amount_evidence WHERE kind = 'FINAL_TOTAL' AND unit = 'IRT' AND amount = 1000`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.amount_evidence WHERE kind = 'FINAL_TOTAL' AND representation = 'UNREPRESENTABLE' AND raw_value = '300.50'`) != 1 ||
		e.count(t, `SELECT count(*) FROM billing.amount_evidence WHERE unit = 'IRR'`) != 0 {
		t.Fatal("money evidence")
	}
	// Payment semantics.
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'approved' AND approval_provenance = 'GATEWAY_AUTO' AND channel = 'GATEWAY'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'approved' AND approval_provenance = 'LEGACY_MANUAL_APPROVAL'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'reconcile_error' AND state_target = 'RECONCILIATION_REQUIRED'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE invoice_link_state = 'UNKNOWN'`) != 1 {
		t.Fatal("payment semantics")
	}
	// Receipts: metadata only.
	if e.count(t, `SELECT count(*) FROM payments.receipt WHERE state = 'REFERENCE_ONLY' AND host = 'files.example.test'`) != 3 ||
		e.count(t, `SELECT count(*) FROM payments.receipt WHERE state = 'UNKNOWN' AND host IS NULL`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.receipt WHERE state = 'AVAILABLE'`) != 0 {
		t.Fatal("receipt states")
	}
	// Totals classes per invoice/unit.
	rep, err := shadow.ReconcileBilling(ctx, b, e.billingQ, e.paymentsQ)
	if err != nil {
		t.Fatal(err)
	}
	// 2001 MATCH; 2002 (an UNKNOWN 'verified' attempt) INCOMPLETE; 2003
	// (every amount fractional: no attributable unit) INCOMPLETE under
	// UNATTRIBUTED; 2004 (declared only) UNKNOWN. Never MISMATCH-by-zero.
	if rep.Totals["IRT"]["MATCH"] != 1 || rep.Totals["IRT"]["INCOMPLETE"] != 1 || rep.Totals["IRT"]["UNKNOWN"] != 1 ||
		rep.Totals["UNATTRIBUTED"]["INCOMPLETE"] != 1 {
		t.Fatalf("totals = %+v", rep.Totals)
	}
	if rep.F02.Status != "NOT_EVALUATED" || rep.F02.Value != nil {
		t.Fatalf("F02: %+v", rep.F02)
	}
	for k, ok := range rep.Invariants {
		if !ok {
			t.Fatalf("invariant %s violated: %+v", k, rep)
		}
	}
	if rep.Outcome != result.Partial {
		t.Fatalf("an attempt without invoice must keep reconciliation PARTIAL: %s", rep.Outcome)
	}
	// Replay creates nothing (model tables unchanged; only a run row is added).
	model := []string{"billing.invoice", "billing.invoice_state_evidence", "billing.amount_evidence", "billing.invoice_item",
		"billing.payment_stage", "payments.payment_attempt", "payments.gateway_attempt", "payments.attempt_state_history", "payments.receipt"}
	fp := e.md5(t, model...)
	if o := e.importBillingOnly(t, b); o != result.Success {
		t.Fatal(o)
	}
	if _, replay := e.importPaymentsOnly(t, b); !replay {
		t.Fatal("payments replay")
	}
	if e.md5(t, model...) != fp {
		t.Fatal("replay changed the billing/payments model")
	}
}

func cleanBilling(c snapshot.Completeness, invoice2001Final string) []snapshot.EntityRows {
	ents := billingSource(c)
	ents = legacytest.With(ents, shadow.SpecPayment, []legacytest.Rec{
		legacytest.Payment("1", "2001", "1000.00", "approved", "gateway", "SYN-AUTH-0001", "zibal")}, c)
	ents = legacytest.With(ents, shadow.SpecInvoiceItem, []legacytest.Rec{legacytest.InvoiceItem("31", "2001", "1000.00")}, c)
	ents = legacytest.With(ents, shadow.SpecPaymentStage, []legacytest.Rec{legacytest.Stage("41", "2001", "1", "1000", "approved", "")}, c)
	return legacytest.With(ents, shadow.SpecInvoice, []legacytest.Rec{
		legacytest.Invoice("2001", "101", "pending", invoice2001Final, "1000.00"),
		legacytest.Invoice("2004", "", "pre_invoice", "50", "50")}, c)
}

// Stale batch: import A; stage B with a changed row; import only one
// module → never SUCCESS; import both → SUCCESS.
func TestReconcileRejectsStaleBatch(t *testing.T) {
	e := setupBilling(t)
	ctx := context.Background()
	a := e.prepare(t, "syn-bill-a", cleanBilling(snapshot.Complete, "1000.00"))
	e.importBillingOnly(t, a)
	e.importPaymentsOnly(t, a)
	rep, err := shadow.ReconcileBilling(ctx, a, e.billingQ, e.paymentsQ)
	if err != nil || rep.Outcome != result.Success {
		t.Fatalf("batch A: %s %+v %v", rep.Outcome, rep.Provenance, err)
	}
	b := e.prepare(t, "syn-bill-b", cleanBilling(snapshot.Complete, "1000"))
	// Only billing imported B: payments rows still carry A.
	e.importBillingOnly(t, b)
	rep, err = shadow.ReconcileBilling(ctx, b, e.billingQ, e.paymentsQ)
	if err != nil || rep.Outcome == result.Success || rep.Provenance[shadow.SpecPayment]["other_batch_or_changed"] == 0 {
		t.Fatalf("billing-only import of B must not be SUCCESS: %s %+v", rep.Outcome, rep.Provenance)
	}
	// Only payments imported C: billing rows still carry B.
	c := e.prepare(t, "syn-bill-c", cleanBilling(snapshot.Complete, "1000.0"))
	e.importPaymentsOnly(t, c)
	rep, err = shadow.ReconcileBilling(ctx, c, e.billingQ, e.paymentsQ)
	if err != nil || rep.Outcome == result.Success || rep.Provenance[shadow.SpecInvoice]["other_batch_or_changed"] == 0 {
		t.Fatalf("payments-only import of C must not be SUCCESS: %s %+v", rep.Outcome, rep.Provenance)
	}
	e.importBillingOnly(t, c)
	e.importPaymentsOnly(t, c)
	rep, err = shadow.ReconcileBilling(ctx, c, e.billingQ, e.paymentsQ)
	if err != nil || rep.Outcome != result.Success {
		t.Fatalf("both modules on C: %s %+v %v", rep.Outcome, rep.Provenance, err)
	}
}

const billingFlag = "billing_payments.read_shadow"

// Same composition as cmd/crm-api (which cannot be imported from a test).
type (
	billingHandlers  = billinghttp.Handlers
	paymentsHandlers = paymentshttp.Handlers
)

type billingSurface struct {
	*billingHandlers
	*paymentsHandlers
}

func (e *billingEnv) server(t *testing.T) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, billingFlag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	at, _ := timeutil.ParseRFC3339("2026-11-01T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Billing: billingSurface{billinghttp.New(guard, e.billingQ, e.paymentsQ), paymentshttp.New(guard, e.paymentsQ)}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *billingEnv) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: billingFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

var billingModel = []string{"billing.invoice", "billing.invoice_state_evidence", "billing.amount_evidence", "billing.invoice_item",
	"billing.payment_stage", "billing.import_run", "payments.payment_attempt", "payments.gateway_attempt", "payments.attempt_state_history",
	"payments.receipt", "payments.import_run"}

// HTTP: flag off → BLOCKED; every GET leaves state unchanged; mutating
// methods are 405; a viewer is 403; nothing sensitive leaks; pagination
// continues beyond one page.
func TestBillingHTTPReadOnly(t *testing.T) {
	e := setupBilling(t)
	b := e.prepare(t, "syn-bill-http", billingSource(snapshot.Complete))
	e.importBillingOnly(t, b)
	e.importPaymentsOnly(t, b)
	h := e.server(t)
	if r := get(t, h, "/api/v1/invoices", "dev-user-001"); !strings.Contains(r.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off: %s", r.body)
	}
	e.setFlag(t, true)
	fp := e.md5(t, billingModel...)
	// Pagination beyond the page size, following next_cursor.
	var ids []string
	next := "/api/v1/invoices?limit=1"
	for i := 0; next != "" && i < 10; i++ {
		r := get(t, h, next, "dev-user-001")
		if r.code != 200 {
			t.Fatalf("list: %d %s", r.code, r.body)
		}
		for _, it := range items(r) {
			ids = append(ids, it["invoice_id"].(string))
		}
		next = ""
		if pg, _ := r.json["page"].(map[string]any); pg != nil && pg["has_more"] == true {
			next = "/api/v1/invoices?limit=1&cursor=" + url.QueryEscape(pg["next_cursor"].(string))
		}
	}
	if len(ids) != 4 {
		t.Fatalf("paged invoices = %d", len(ids))
	}
	bodies := []string{}
	for _, id := range ids {
		for _, p := range []string{"", "/items?limit=1", "/stages", "/totals", "/payments", "/receipts?limit=1"} {
			r := get(t, h, "/api/v1/invoices/"+id+p, "dev-user-001")
			if r.code != 200 {
				t.Fatalf("GET %s%s = %d %s", id, p, r.code, r.body)
			}
			bodies = append(bodies, r.body)
		}
	}
	att := get(t, h, "/api/v1/payments/attempts?limit=200", "dev-user-001")
	if att.code != 200 || len(items(att)) != 5 {
		t.Fatalf("attempts: %d %s", att.code, att.body)
	}
	bodies = append(bodies, att.body)
	for _, it := range items(att) {
		id := it["attempt_id"].(string)
		for _, p := range []string{"/api/v1/payments/attempts/" + id, "/api/v1/payments/attempts/" + id + "/history"} {
			r := get(t, h, p, "dev-user-001")
			if r.code != 200 {
				t.Fatalf("GET %s = %d %s", p, r.code, r.body)
			}
			bodies = append(bodies, r.body)
		}
	}
	rep := get(t, h, "/api/v1/billing/reconciliation-report", "dev-user-001")
	for _, want := range []string{`"code":"F02"`, `"status":"NOT_EVALUATED"`, `"count":null`, `"approved_raw_count":2`, `"reconcile_error_raw_count":1`} {
		if !strings.Contains(rep.body, want) {
			t.Fatalf("report lacks %s: %s", want, rep.body)
		}
	}
	bodies = append(bodies, rep.body)
	// States stay separate; raw unknowns visible; money with explicit units.
	gw := get(t, h, "/api/v1/payments/attempts?channel=GATEWAY&state=ACCEPTED", "dev-user-001")
	if len(items(gw)) != 1 || !strings.Contains(gw.body, `"approval_provenance":"GATEWAY_AUTO"`) || !strings.Contains(gw.body, `"state_raw":"approved"`) ||
		!strings.Contains(gw.body, `"has_authority_reference":true`) {
		t.Fatalf("gateway attempt: %s", gw.body)
	}
	unk := get(t, h, "/api/v1/invoices?has_unknown_status=true", "dev-user-001")
	if len(items(unk)) != 1 {
		t.Fatalf("unknown filter: %s", unk.body)
	}
	if r := get(t, h, "/api/v1/invoices/"+ids[0]+"/totals", "dev-user-001"); !strings.Contains(r.body, `"unit":"`) {
		t.Fatalf("totals: %s", r.body)
	}
	for _, p := range []string{"/api/v1/invoices?phone=1", "/api/v1/payments/attempts?customer_phone=1", "/api/v1/payments/attempts?channel=CASH"} {
		if r := get(t, h, p, "dev-user-001"); r.code != 400 {
			t.Fatalf("%s must be 400: %d", p, r.code)
		}
	}
	if r := get(t, h, "/api/v1/invoices", "dev-user-002"); r.code != 403 {
		t.Fatalf("viewer: %d", r.code)
	}
	for _, body := range bodies {
		for _, leak := range []string{"phone", "national_id", "address", "SYN-AUTH", "http://", "https://", "/uploads/", "receipt-2003",
			"proof_data", "submission_key", "authority_sha256", "ref_sha256", "sn_", "wp_", "stg_", "password", "token"} {
			if strings.Contains(body, leak) {
				t.Fatalf("leak %q in %s", leak, body)
			}
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, p := range []string{"/api/v1/invoices", "/api/v1/invoices/" + ids[0], "/api/v1/payments/attempts", "/api/v1/billing/reconciliation-report"} {
			req := httptest.NewRequest(m, p, nil)
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 405 {
				t.Fatalf("%s %s = %d", m, p, rec.Code)
			}
		}
	}
	if e.md5(t, billingModel...) != fp {
		t.Fatal("a GET changed billing/payments state")
	}
}
