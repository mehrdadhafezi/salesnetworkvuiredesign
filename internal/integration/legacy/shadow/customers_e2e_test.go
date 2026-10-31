package shadow_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customershttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/http"
	customersimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/imports"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

const customersFlag = "customers.identity_shadow_read"

type custEnv struct {
	*env
	runner *db.Runner
	in     *customersimports.Command
	q      *customersquery.Service
}

func setupCustomers(t *testing.T) *custEnv {
	t.Helper()
	e := setup(t)
	if _, err := migrations.Up(context.Background(), e.DSN, customersmigrations.Source()); err != nil {
		t.Fatalf("customers migrations: %v", err)
	}
	at, _ := timeutil.ParseRFC3339("2026-10-29T09:00:00Z")
	r := e.Runner(t, customers.Module)
	return &custEnv{env: e, runner: r, in: customersimports.New(r, timeutil.FixedClock{At: at}), q: customersquery.New(r)}
}

func (e *custEnv) importBatch(t *testing.T, b *shadow.Batch) (result.Status, bool) {
	t.Helper()
	in, _ := shadow.CustomersCommand(b)
	rep, err := e.in.ImportCustomers(context.Background(), in)
	if err != nil {
		t.Fatalf("customers import: %v", err)
	}
	return rep.Outcome, rep.Replay
}

func (e *custEnv) server(t *testing.T) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, customersFlag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	at, _ := timeutil.ParseRFC3339("2026-10-29T10:00:00Z")
	h, err := apiserver.NewHandlerWith(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil,
		customershttp.New(guard, e.q))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *custEnv) setCustomersFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-10-29T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: customersFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

func (e *custEnv) customersFingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, tbl := range []string{"customer", "contact_point", "customer_identifier", "identity_evidence", "evidence_subject", "merge_case",
		"merge_case_member", "merge_case_evidence", "merge_case_state_history", "customer_resolution_history"} {
		var s *string
		if err := e.admin.QueryRow(context.Background(), `SELECT md5(string_agg(row_to_json(x)::text, '|' ORDER BY row_to_json(x)::text)) FROM customers.`+tbl+` x`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		if s != nil {
			parts = append(parts, *s)
		}
	}
	return strings.Join(parts, ",")
}

// TestCustomersShadowEndToEnd: stage synthetic issued-to rows, import them
// through the published command, reconcile, and read them over HTTP.
func TestCustomersShadowEndToEnd(t *testing.T) {
	e := setupCustomers(t)
	ctx := context.Background()
	b := e.stage(t, "syn-customers-001", legacytest.CustomerScenario(snapshot.Complete))
	if out, replay := e.importBatch(t, b); replay || (out != result.Success && out != result.Partial) {
		t.Fatalf("first import: %s replay=%v", out, replay)
	}

	// One staged row = one customer; nothing collapsed, phone is never a key.
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE presence = 'PRESENT'`); n != 5 {
		t.Fatalf("customers = %d, want 5 (one per staged subject)", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'RESOLVED'`); n != 0 {
		t.Fatalf("legacy evidence produced %d RESOLVED customers", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE state = 'RESOLVED'`); n != 0 {
		t.Fatalf("legacy evidence produced %d RESOLVED merge cases", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'NOT_RECOVERABLE'`); n != 1 {
		t.Fatalf("not-recoverable customers = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE reason = 'SHARED_PHONE'`); n < 1 {
		t.Fatal("shared phone produced no candidate")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE reason = 'SHARED_ACCOUNT_REF' AND state = 'RELATED'`); n != 1 {
		t.Fatalf("phone-derived shared account cases = %d, want 1 RELATED", n)
	}

	// Reconcile: invariants hold, report is complete.
	rep, err := shadow.ReconcileCustomers(ctx, b, e.q)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != result.Success || rep.Resolution.PhoneOnlyResolved != 0 || rep.Resolution.PhoneOnlyMerges != 0 || rep.Resolution.Merges != 0 {
		t.Fatalf("reconcile = %+v", rep)
	}
	for k, ok := range rep.Invariants {
		if !ok {
			t.Fatalf("invariant %s violated", k)
		}
	}

	// Replay is a no-op.
	before := e.customersFingerprint(t)
	if _, replay := e.importBatch(t, b); !replay {
		t.Fatal("replay not detected")
	}
	if e.customersFingerprint(t) != before {
		t.Fatal("replay changed customers state")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.import_run WHERE replay`); n != 1 {
		t.Fatalf("replay audit runs = %d", n)
	}

	// HTTP: flag off → BLOCKED.
	h := e.server(t)
	if r := get(t, h, "/api/v1/customers", "dev-user-001"); r.code != 200 || !strings.Contains(r.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setCustomersFlag(t, true)
	before = e.customersFingerprint(t)
	list := get(t, h, "/api/v1/customers?limit=2", "dev-user-001")
	if list.code != 200 || len(items(list)) != 2 || list.json["page"].(map[string]any)["next_cursor"] == nil {
		t.Fatalf("list: %d %s", list.code, list.body)
	}
	all := get(t, h, "/api/v1/customers?limit=200", "dev-user-001")
	var all5 []string
	for _, it := range items(all) {
		all5 = append(all5, it["customer_id"].(string))
	}
	if len(all5) != 5 {
		t.Fatalf("all: %s", all.body)
	}
	// No raw phone, raw name, staging or source payload leaks anywhere.
	bodies := []string{list.body, all.body}
	for _, id := range all5 {
		for _, p := range []string{"", "/contact-points", "/evidence"} {
			r := get(t, h, "/api/v1/customers/"+id+p, "dev-user-001")
			if r.code != 200 {
				t.Fatalf("GET %s%s = %d %s", id, p, r.code, r.body)
			}
			bodies = append(bodies, r.body)
		}
	}
	mc := get(t, h, "/api/v1/customer-identity/merge-cases?limit=200", "dev-user-001")
	if mc.code != 200 || len(items(mc)) == 0 {
		t.Fatalf("merge cases: %s", mc.body)
	}
	bodies = append(bodies, mc.body)
	for _, it := range items(mc) {
		r := get(t, h, "/api/v1/customer-identity/merge-cases/"+it["merge_case_id"].(string), "dev-user-001")
		if r.code != 200 || strings.Contains(r.body, `"state":"RESOLVED"`) {
			t.Fatalf("merge case: %s", r.body)
		}
		bodies = append(bodies, r.body)
	}
	report := get(t, h, "/api/v1/customer-identity/resolution-report", "dev-user-001")
	if report.code != 200 || !strings.Contains(report.body, `"phone_only_resolved_count":0`) || !strings.Contains(report.body, `"phone_only_merge_count":0`) {
		t.Fatalf("report: %s", report.body)
	}
	bodies = append(bodies, report.body)
	for _, body := range bodies {
		for _, leak := range []string{"09000000101", "09000000404", "09000000405", "+98 900 000 0101", "900 000 0101", "Synthetic Customer",
			"SYN-INV-", "stg_", "raw_", "row_sha256", "wp_", "sn_"} {
			if strings.Contains(body, leak) {
				t.Fatalf("leak %q in %s", leak, body)
			}
		}
	}
	// Filters are server-side and allowlisted.
	if r := get(t, h, "/api/v1/customers?state=NOT_RECOVERABLE", "dev-user-001"); len(items(r)) != 1 {
		t.Fatalf("state filter: %s", r.body)
	}
	if r := get(t, h, "/api/v1/customers?phone=09000000101", "dev-user-001"); r.code != 400 {
		t.Fatalf("phone filter must be rejected: %d", r.code)
	}
	// Viewer is denied; anonymous is denied.
	if r := get(t, h, "/api/v1/customers", "dev-user-002"); r.code != 403 {
		t.Fatalf("viewer: %d %s", r.code, r.body)
	}
	// Writes do not exist.
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(m, "/api/v1/customers/"+all5[0], nil)
		req.Header.Set(authstub.Header, "dev-user-001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 405 {
			t.Fatalf("%s = %d", m, rec.Code)
		}
	}
	// Every GET above left customers state (and the run audit) untouched.
	if n := e.count(t, `SELECT count(*) FROM customers.import_run`); e.customersFingerprint(t) != before || n != 2 {
		t.Fatal("GET changed customers state")
	}
}

// An INCOMPLETE batch marks nothing absent; a later COMPLETE empty batch
// marks absent but deletes nothing.
func TestCustomersAbsenceOnlyFromCompleteBatch(t *testing.T) {
	e := setupCustomers(t)
	e.importBatch(t, e.stage(t, "syn-customers-a", legacytest.CustomerScenario(snapshot.Complete)))
	empty := legacytest.CustomerScenario(snapshot.Incomplete)
	empty[0].Rows = nil
	e.importBatch(t, e.stage(t, "syn-customers-b", empty))
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE presence = 'PRESENT'`); n != 5 {
		t.Fatalf("incomplete batch changed presence: %d present", n)
	}
	empty[0].Completeness = snapshot.Complete
	e.importBatch(t, e.stage(t, "syn-customers-c", empty))
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 5 {
		t.Fatalf("customers deleted: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE presence = 'PRESENT'`); n != 0 {
		t.Fatalf("complete empty batch left %d present", n)
	}
}
