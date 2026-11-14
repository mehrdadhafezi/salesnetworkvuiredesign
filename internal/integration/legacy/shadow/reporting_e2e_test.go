package shadow_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/audit"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	reportinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/http"
	reportingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/projections"
	reportingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/query"
)

const reportingFlag = "reporting.read_shadow"

type reportingEnv struct {
	*billingEnv
	rebuild *projections.Command
	q       *reportingquery.Service
}

func setupReporting(t *testing.T) *reportingEnv {
	t.Helper()
	e := setupBilling(t, reportingmigrations.Source())
	if _, err := migrations.Up(context.Background(), e.DSN, reportingmigrations.Source()); err != nil {
		t.Fatalf("reporting migrations: %v", err)
	}
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	r := e.Runner(t, reporting.Module)
	cmd, err := projections.New(r, clock)
	if err != nil {
		t.Fatal(err)
	}
	return &reportingEnv{billingEnv: e, rebuild: cmd, q: reportingquery.New(r, clock)}
}

// sources are the REAL owner published APIs (each its own module runner).
func (e *reportingEnv) sources() projections.Sources {
	return projections.Sources{Cases: e.casesQ, Custody: e.custodyQ, Billing: e.billingQ, Payments: e.paymentsQ, PayReader: e.paymentsQ,
		Customers: e.custQ}
}

func (e *reportingEnv) server(t *testing.T, auditor reportinghttp.ReadAuditor) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, reportingFlag)
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
	clock := timeutil.FixedClock{At: at}
	h, err := apiserver.NewHandlerWithSurfaces(cfg, clock, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Reporting: reportinghttp.New(guard, e.q, auditor, clock)})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *reportingEnv) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-01T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: reportingFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

func (e *reportingEnv) auditor(t *testing.T) reportinghttp.ReadAuditor {
	r := e.Runner(t, owner.Audit)
	return func(ctx context.Context, ev audit.Event) error {
		return r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error { return audit.Append(ctx, tx, ev) })
	}
}

var reportingTables = []string{"reporting.projection_run", "reporting.projection_source", "reporting.source_aggregate", "reporting.metric_contract",
	"reporting.case_fact", "reporting.custody_fact", "reporting.invoice_fact", "reporting.invoice_unit_fact", "reporting.payment_fact"}

var ownerTables = []string{"cases.work_case", "cases.case_source", "cases.attribution_snapshot", "custody.assignment", "custody.custody_event",
	"customers.customer", "billing.invoice", "billing.invoice_state_evidence", "billing.invoice_item", "billing.payment_stage",
	"payments.payment_attempt", "payments.receipt", "identity.user_account", "organization.org_assignment"}

func num(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}

func TestReportingEndToEnd(t *testing.T) {
	e := setupReporting(t)
	ctx := context.Background()
	b := e.prepare(t, "syn-rep-001", billingSource(snapshot.Complete))
	e.importBillingOnly(t, b)
	e.importPaymentsOnly(t, b)
	run, err := e.rebuild.Rebuild(ctx, e.sources())
	if err != nil {
		t.Fatal(err)
	}
	if run.Outcome != result.Success || run.Counts["case_fact"] != 6 || run.Counts["invoice_fact"] != 4 || run.Counts["payment_fact"] != 5 {
		t.Fatalf("rebuild from published APIs: %+v", run)
	}
	// Same sources + same projection version: identical facts.
	again, err := e.rebuild.Rebuild(ctx, e.sources())
	if err != nil || string(again.FactsSHA) != string(run.FactsSHA) {
		t.Fatalf("rebuild must be deterministic: %v", err)
	}
	// Historical attribution is immutable: a current-org change and a
	// rebuild do not rewrite it.
	before := e.md5(t, "reporting.case_fact")
	if _, err := e.admin.Exec(ctx, `UPDATE organization.org_assignment SET is_current = NOT COALESCE(is_current, false)`); err != nil {
		t.Fatal(err)
	}
	after, _ := e.rebuild.Rebuild(ctx, e.sources())
	if string(after.FactsSHA) != string(run.FactsSHA) {
		t.Fatal("a current-org change rewrote reporting attribution facts")
	}
	_ = before

	h := e.server(t, e.auditor(t))
	if r := get(t, h, "/api/v1/reporting/metrics", "dev-user-001"); !strings.Contains(r.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off: %s", r.body)
	}
	e.setFlag(t, true)
	fpReporting, fpOwners := e.md5(t, reportingTables...), e.md5(t, ownerTables...)

	// INV-021 over HTTP: metric card == report summary == union of pages; detail explains membership.
	m := get(t, h, "/api/v1/reporting/metrics/invoices.unknown_status", "dev-user-001")
	data, _ := m.json["data"].(map[string]any)
	if m.code != 200 || data["value_state"] != "VALUE" {
		t.Fatalf("metric: %d %s", m.code, m.body)
	}
	card := num(data["summary"].(map[string]any)["total"])
	q := data["meta"].(map[string]any)["query"].(map[string]any)
	runID := q["run_id"].(string)
	base := "/api/v1/reports/invoices?presence=PRESENT&has_unknown_status=true&run_id=" + runID
	var ids []string
	next := base + "&limit=1"
	for i := 0; next != "" && i < 20; i++ {
		r := get(t, h, next, "dev-user-001")
		if r.code != 200 || num(r.json["summary"].(map[string]any)["total"]) != card {
			t.Fatalf("page total must equal card %d: %d %s", card, r.code, r.body)
		}
		for _, it := range items(r) {
			ids = append(ids, it["invoice_id"].(string))
		}
		next = ""
		if pg, _ := r.json["page"].(map[string]any); pg != nil && pg["has_more"] == true {
			next = base + "&limit=1&cursor=" + url.QueryEscape(pg["next_cursor"].(string))
		}
	}
	if int64(len(ids)) != card || card != 1 {
		t.Fatalf("table rows %d vs card %d", len(ids), card)
	}
	for _, id := range ids {
		d := get(t, h, "/api/v1/reports/invoices/"+id+"?presence=PRESENT&has_unknown_status=true&run_id="+runID, "dev-user-001")
		if d.json["member"] != true {
			t.Fatalf("detail must confirm membership: %s", d.body)
		}
	}
	// Every GET endpoint answers.
	bodies := []string{m.body}
	for _, p := range []string{"/api/v1/reporting/metrics", "/api/v1/reports/cases?limit=2", "/api/v1/reports/custody", "/api/v1/reports/payments",
		"/api/v1/reports/payments?state=UNMAPPED", "/api/v1/mis/overview", "/api/v1/mis/source-quality", "/api/v1/mis/reconciliation",
		"/api/v1/reporting/metrics/invoices.open_preinvoices", "/api/v1/reporting/metrics/payments.accepted_attempt_amount",
		"/api/v1/reporting/metrics/cases.first_observed?time_basis=EVENT_RANGE&from=2026-10-01T00:00:00Z&to=2026-12-01T00:00:00Z"} {
		r := get(t, h, p, "dev-user-001")
		if r.code != 200 {
			t.Fatalf("GET %s = %d %s", p, r.code, r.body)
		}
		bodies = append(bodies, r.body)
	}
	// F02 not evaluated; IRT/IRR separate; unknown attempt state visible;
	// phone-only candidates reported, never merged.
	f02 := get(t, h, "/api/v1/reporting/metrics/invoices.open_preinvoices", "dev-user-001")
	if !strings.Contains(f02.body, `"value_state":"NOT_EVALUATED"`) || strings.Contains(f02.body, `"summary"`) {
		t.Fatalf("F02: %s", f02.body)
	}
	pay := get(t, h, "/api/v1/reports/payments", "dev-user-001")
	if !strings.Contains(pay.body, `"UNMAPPED":1`) || !strings.Contains(pay.body, `"unit":"IRT"`) {
		t.Fatalf("payments report: %s", pay.body)
	}
	sq := get(t, h, "/api/v1/mis/source-quality", "dev-user-001")
	if !strings.Contains(sq.body, `"duplicate_candidates"`) || !strings.Contains(sq.body, "NEVER merged") {
		t.Fatalf("source quality: %s", sq.body)
	}
	// Invoice Review: audited read; fails closed when the audit fails.
	var auditBefore int64
	_ = e.admin.QueryRow(ctx, `SELECT count(*) FROM audit.audit_event WHERE action = 'reporting.invoice_review.read'`).Scan(&auditBefore)
	ir := get(t, h, "/api/v1/invoice-review?limit=2", "dev-user-001")
	if ir.code != 200 || len(items(ir)) != 2 || ir.json["outcome"] != "SUCCESS" {
		t.Fatalf("invoice review: %d %s", ir.code, ir.body)
	}
	one := get(t, h, "/api/v1/invoice-review/"+ids[0], "dev-user-001")
	if one.code != 200 || one.json["data"] == nil {
		t.Fatalf("invoice review detail: %s", one.body)
	}
	var auditAfter int64
	_ = e.admin.QueryRow(ctx, `SELECT count(*) FROM audit.audit_event WHERE action = 'reporting.invoice_review.read'`).Scan(&auditAfter)
	if auditAfter-auditBefore != 2 {
		t.Fatalf("every Invoice Review read must be audited: %d", auditAfter-auditBefore)
	}
	bodies = append(bodies, ir.body, one.body)
	failing := e.server(t, func(context.Context, audit.Event) error { return errors.New("synthetic audit outage") })
	for _, p := range []string{"/api/v1/invoice-review", "/api/v1/invoice-review/" + ids[0]} {
		r := get(t, failing, p, "dev-user-001")
		if r.json["outcome"] != "BLOCKED" || r.json["items"] != nil || r.json["data"] != nil || !strings.Contains(r.body, "POLICY_UNRESOLVED") {
			t.Fatalf("audit failure must fail closed with no row: %s", r.body)
		}
	}
	noAudit := e.server(t, nil)
	if r := get(t, noAudit, "/api/v1/invoice-review", "dev-user-001"); r.json["outcome"] != "BLOCKED" || r.json["items"] != nil {
		t.Fatalf("no auditor must fail closed: %s", r.body)
	}
	// Aggregate reporting keeps working without an auditor.
	if r := get(t, noAudit, "/api/v1/reporting/metrics/cases.staged_total", "dev-user-001"); r.code != 200 || !strings.Contains(r.body, `"value_state":"VALUE"`) {
		t.Fatalf("aggregate metrics must not depend on the Invoice Review audit: %s", r.body)
	}
	// No export of any kind; no guessed export URL works; mutating methods are 405.
	for _, p := range []string{"/api/v1/invoice-review/export", "/api/v1/invoice-review.csv", "/api/v1/invoice-review/export.xlsx",
		"/api/v1/reports/invoices/export", "/api/v1/reports/export.csv", "/api/v1/reporting/export", "/api/v1/invoice-review/download"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			req := httptest.NewRequest(method, p, nil)
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code == 200 || strings.Contains(rec.Header().Get("Content-Type"), "csv") || strings.Contains(rec.Header().Get("Content-Type"), "spreadsheet") {
				t.Fatalf("%s %s must not exist: %d %s", method, p, rec.Code, rec.Header().Get("Content-Type"))
			}
		}
	}
	if r := get(t, h, "/api/v1/invoice-review?format=csv", "dev-user-001"); r.code != 400 {
		t.Fatalf("format parameter must be rejected: %d", r.code)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, p := range []string{"/api/v1/reporting/metrics", "/api/v1/reports/cases", "/api/v1/mis/overview", "/api/v1/invoice-review"} {
			req := httptest.NewRequest(method, p, nil)
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 405 {
				t.Fatalf("%s %s = %d", method, p, rec.Code)
			}
		}
	}
	// Authorization: the viewer has none; filters/scope cannot broaden.
	for _, p := range []string{"/api/v1/reporting/metrics", "/api/v1/mis/overview", "/api/v1/invoice-review"} {
		if r := get(t, h, p, "dev-user-002"); r.code != 403 {
			t.Fatalf("viewer %s: %d", p, r.code)
		}
	}
	for _, p := range []string{"/api/v1/reports/cases?phone=1", "/api/v1/reports/cases?scope=ALL", "/api/v1/reports/cases?time_basis=EVENT_RANGE",
		"/api/v1/reports/cases?as_of=2026-01-01T00:00:00Z"} {
		if r := get(t, h, p, "dev-user-001"); r.code != 400 {
			t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
		}
	}
	for _, body := range bodies {
		for _, leak := range []string{"phone\":", "customer_name", "address", "postcode", "national", "SYN-AUTH", "http://", "https://", "receipt-2003",
			"sn_", "wp_", "stg_", "token", "authority_sha256"} {
			if strings.Contains(body, leak) {
				t.Fatalf("leak %q in %s", leak, body)
			}
		}
	}
	if e.md5(t, reportingTables...) != fpReporting || e.md5(t, ownerTables...) != fpOwners {
		t.Fatal("a GET changed reporting or owner state (only the audit module records Invoice Review reads)")
	}
}

// Removing one required source: the run is PARTIAL and metrics on that
// source are UNKNOWN, never zero; other metrics still answer.
func TestReportingMissingSourceIsPartial(t *testing.T) {
	e := setupReporting(t)
	ctx := context.Background()
	b := e.prepare(t, "syn-rep-002", billingSource(snapshot.Complete))
	e.importBillingOnly(t, b)
	e.importPaymentsOnly(t, b)
	src := e.sources()
	src.Payments, src.PayReader = nil, nil
	run, err := e.rebuild.Rebuild(ctx, src)
	if err != nil || run.Outcome != result.Partial || run.Sources["payments"].Status != "MISSING" {
		t.Fatalf("missing payments must be PARTIAL: %+v %v", run, err)
	}
	e.setFlag(t, true)
	h := e.server(t, e.auditor(t))
	if r := get(t, h, "/api/v1/reporting/metrics/payments.by_state", "dev-user-001"); !strings.Contains(r.body, `"value_state":"UNKNOWN"`) ||
		strings.Contains(r.body, `"summary"`) {
		t.Fatalf("missing source metric must be UNKNOWN without a value: %s", r.body)
	}
	if r := get(t, h, "/api/v1/reporting/metrics/cases.staged_total", "dev-user-001"); !strings.Contains(r.body, `"value_state":"VALUE"`) {
		t.Fatalf("unaffected metric: %s", r.body)
	}
	if r := get(t, h, "/api/v1/mis/overview", "dev-user-001"); r.json["outcome"] != "PARTIAL" || !strings.Contains(r.body, `"coverage":"MISSING"`) {
		t.Fatalf("overview must show the gap: %s", r.body)
	}
}
