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
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	casesimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/imports"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodyhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/http"
	custodyimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/imports"
	custodymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/migrations"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
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

const salesFlag = "sales.read_shadow"

type salesEnv struct {
	*env
	casesQ   *casesquery.Service
	custodyQ *custodyquery.Service
	casesIn  *casesimports.Command
	custIn   *custodyimports.Command
}

func setupSales(t *testing.T) *salesEnv {
	t.Helper()
	e := setup(t)
	for _, s := range []migrations.Source{casesmigrations.Source(), custodymigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, s); err != nil {
			t.Fatalf("sales migrations: %v", err)
		}
	}
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	cr, sr := e.Runner(t, cases.Module), e.Runner(t, custody.Module)
	cq := casesquery.New(cr)
	return &salesEnv{env: e, casesQ: cq, custodyQ: custodyquery.New(sr), casesIn: casesimports.New(cr, clock), custIn: custodyimports.New(sr, cq, clock)}
}

// importSales runs the frozen cross-module sequence: identity resolves
// users, cases imports (cases tx), custody resolves Cases through the
// published cases API and imports (custody tx). No transaction spans two
// modules.
func (e *salesEnv) importSales(t *testing.T, b *shadow.Batch) (rc, rs result.Status, replay bool) {
	t.Helper()
	ctx := context.Background()
	users, err := e.ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.UserKeys(b))
	if err != nil {
		t.Fatal(err)
	}
	ci, _ := shadow.CasesCommand(b, users)
	cr, err := e.casesIn.ImportCases(ctx, ci)
	if err != nil {
		t.Fatalf("cases import: %v", err)
	}
	si, _ := shadow.CustodyCommand(b, users)
	sr, err := e.custIn.ImportCustody(ctx, si)
	if err != nil {
		t.Fatalf("custody import: %v", err)
	}
	return cr.Outcome, sr.Outcome, cr.Replay && sr.Replay
}

func withSales(ents []snapshot.EntityRows, c snapshot.Completeness) []snapshot.EntityRows {
	return append(ents, legacytest.SalesScenario(c)...)
}

func (e *salesEnv) parityOf(t *testing.T, spec, key string) string {
	t.Helper()
	var p string
	if err := e.admin.QueryRow(context.Background(), `SELECT a.parity FROM custody.assignment a JOIN cases.case_source s USING (case_id)
		WHERE s.source_spec = $1 AND s.source_key = $2`, spec, key).Scan(&p); err != nil {
		t.Fatalf("%s/%s: %v", spec, key, err)
	}
	return p
}

func (e *salesEnv) md5(t *testing.T, tables ...string) string {
	t.Helper()
	var parts []string
	for _, tbl := range tables {
		var s *string
		if err := e.admin.QueryRow(context.Background(), `SELECT md5(string_agg(row_to_json(x)::text, '|' ORDER BY row_to_json(x)::text)) FROM `+tbl+` x`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		if s != nil {
			parts = append(parts, *s)
		}
	}
	return strings.Join(parts, ",")
}

var salesTables = []string{"cases.work_case", "cases.case_source", "cases.lead", "cases.case_alias", "cases.case_state_history",
	"cases.attribution_snapshot", "cases.import_run", "custody.custody_event", "custody.assignment", "custody.finding", "custody.import_run"}

func TestSalesShadowEndToEnd(t *testing.T) {
	e := setupSales(t)
	ctx := context.Background()
	b := e.stage(t, "syn-sales-001", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	if rc, rs, _ := e.importSales(t, b); rc != result.Success || rs != result.Success {
		t.Fatalf("imports: cases %s custody %s", rc, rs)
	}

	// One staged row = one Case; the conversion link is an alias, not a merge.
	if e.count(t, `SELECT count(*) FROM cases.work_case`) != 6 || e.count(t, `SELECT count(*) FROM cases.lead`) != 2 {
		t.Fatal("expected 2 lead Cases and 4 item Cases")
	}
	if e.count(t, `SELECT count(*) FROM cases.case_alias WHERE resolution = 'RELATED' AND related_case_id IS NOT NULL`) != 1 {
		t.Fatal("conversion link must be one RELATED alias")
	}
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE customer_link_state <> 'UNKNOWN'`) != 0 {
		t.Fatal("no explicit customer reference exists; linkage must stay UNKNOWN")
	}
	// Unknown legacy states and actions are preserved raw, never defaulted.
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE state_mapping = 'UNKNOWN' AND state_raw = 'synthetic_unfamiliar_status'`) != 2 ||
		e.count(t, `SELECT count(*) FROM custody.custody_event WHERE kind = 'UNKNOWN' AND legacy_action = 'synthetic_unverified_action'`) != 1 {
		t.Fatal("unknown states/actions must be UNKNOWN with the raw value")
	}

	// Custody parity per evidence.
	for _, c := range []struct{ spec, key, want string }{
		{shadow.SpecDistItem, "501", "MATCH"}, {shadow.SpecDistItem, "502", "CONFLICT"}, {shadow.SpecDistItem, "503", "MATCH"},
		{shadow.SpecDistItem, "504", "TARGET_UNKNOWN"}, {shadow.SpecLead, "101", "TARGET_UNKNOWN"}, {shadow.SpecLead, "102", "NOT_RECOVERABLE"},
	} {
		if got := e.parityOf(t, c.spec, c.key); got != c.want {
			t.Errorf("%s/%s parity = %s, want %s", c.spec, c.key, got, c.want)
		}
	}
	// Attribution: original owner 4 and original seller 5 for item 501,
	// resolved to identity users; lead attribution NOT_RECOVERABLE.
	if e.count(t, `SELECT count(*) FROM cases.attribution_snapshot a JOIN cases.case_source s USING (case_id)
		WHERE s.source_key = '501' AND a.facet = 'ORIGINAL_OWNER' AND a.resolution = 'RESOLVED' AND a.subject_legacy_key = '4' AND a.subject_user_id IS NOT NULL`) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.attribution_snapshot a JOIN cases.case_source s USING (case_id)
		WHERE s.source_key = '501' AND a.facet = 'ORIGINAL_SELLER' AND a.subject_legacy_key = '5'`) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE facet = 'CREDIT_OWNER' AND resolution = 'RESOLVED'`) != 0 {
		t.Fatal("attribution snapshots")
	}

	// Mandatory: change the CURRENT organisation after import; historical
	// custody and attribution must not change, on re-read or re-import.
	before := e.md5(t, "cases.attribution_snapshot", "custody.custody_event", "custody.assignment")
	for _, q := range []string{
		`UPDATE organization.org_assignment SET is_current = NOT COALESCE(is_current, false)`,
		`UPDATE organization.employment SET employee_code = 'SYN-REORG-' || employment_id::text`,
		`UPDATE organization.person SET refreshed_at = refreshed_at + interval '1 day'`,
	} {
		if _, err := e.admin.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if _, _, replay := e.importSales(t, b); !replay {
		t.Fatal("re-import of the same batch must be a replay")
	}
	if e.md5(t, "cases.attribution_snapshot", "custody.custody_event", "custody.assignment") != before {
		t.Fatal("historical attribution/custody changed after a current-org change")
	}
	// A new batch carrying the same evidence plus one new lead appends only
	// the new lead's Case and snapshots: nothing existing is duplicated.
	ents := withSales(legacytest.HRScenario(), snapshot.Complete)
	for i := range ents {
		if ents[i].SpecID == shadow.SpecLead {
			ents[i] = legacytest.With(ents, shadow.SpecLead, []legacytest.Rec{legacytest.Lead("101", "assigned", "5", "2"),
				legacytest.Lead("102", "synthetic_unfamiliar_status", "", ""), legacytest.Lead("103", "unassigned", "", "")}, snapshot.Complete)[len(ents)-1]
		}
	}
	b2 := e.stage(t, "syn-sales-002", ents)
	e.importSales(t, b2)
	if e.count(t, `SELECT count(*) FROM cases.attribution_snapshot`) != 18 || e.count(t, `SELECT count(*) FROM custody.custody_event`) != 8 ||
		e.count(t, `SELECT count(*) FROM cases.case_state_history`) != 2 || e.count(t, `SELECT count(*) FROM cases.work_case`) != 7 {
		t.Fatal("duplicates after re-import")
	}

	// Reconciliation: honest, never full parity here.
	rep, err := shadow.ReconcileSales(ctx, b2, e.casesQ, e.custodyQ)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != result.Partial || rep.Custody.FullParity || rep.Staged["cases_matched_to_batch"] != 7 || rep.Cases.LegacyLeadsOnly != 3 {
		t.Fatalf("reconcile = %+v", rep)
	}
	for k, ok := range rep.Invariants {
		if !ok {
			t.Fatalf("invariant %s violated", k)
		}
	}

	// HTTP: flag off → BLOCKED; then every GET; no state change; 405s; 403.
	h := e.salesServer(t)
	if r := get(t, h, "/api/v1/cases", "dev-user-001"); !strings.Contains(r.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off: %s", r.body)
	}
	e.setSalesFlag(t, true)
	fp := e.md5(t, salesTables...)
	list := get(t, h, "/api/v1/cases?limit=200", "dev-user-001")
	if list.code != 200 || len(items(list)) != 7 {
		t.Fatalf("list: %d %s", list.code, list.body)
	}
	bodies := []string{list.body}
	for _, it := range items(list) {
		id := it["case_id"].(string)
		for _, p := range []string{"/api/v1/cases/" + id, "/api/v1/cases/" + id + "/history", "/api/v1/cases/" + id + "/attribution",
			"/api/v1/custody/cases/" + id} {
			r := get(t, h, p, "dev-user-001")
			if r.code != 200 {
				t.Fatalf("GET %s = %d %s", p, r.code, r.body)
			}
			bodies = append(bodies, r.body)
		}
	}
	for _, p := range []string{"/api/v1/custody/assignments?parity=CONFLICT", "/api/v1/sales/coverage-report", "/api/v1/sales/custody-parity",
		"/api/v1/cases?workflow_type=LEGACY_LEAD&state_mapping=UNKNOWN"} {
		r := get(t, h, p, "dev-user-001")
		if r.code != 200 {
			t.Fatalf("GET %s = %d %s", p, r.code, r.body)
		}
		bodies = append(bodies, r.body)
	}
	cov := get(t, h, "/api/v1/sales/coverage-report", "dev-user-001")
	for _, want := range []string{`"code":"F02"`, `"status":"NOT_EVALUATED"`, `"count":null`, `"label":"Legacy Leads Only (not Total Cases)"`, `"count":3`} {
		if !strings.Contains(cov.body, want) {
			t.Fatalf("coverage report lacks %s: %s", want, cov.body)
		}
	}
	if r := get(t, h, "/api/v1/sales/custody-parity", "dev-user-001"); !strings.Contains(r.body, `"full_parity":false`) || !strings.Contains(r.body, `"CONFLICT":1`) {
		t.Fatalf("parity report: %s", r.body)
	}
	if r := get(t, h, "/api/v1/cases?phone=1", "dev-user-001"); r.code != 400 {
		t.Fatalf("phone filter must be rejected: %d", r.code)
	}
	if r := get(t, h, "/api/v1/custody/assignments", "dev-user-002"); r.code != 403 {
		t.Fatalf("viewer: %d", r.code)
	}
	for _, body := range bodies {
		for _, leak := range []string{"phone", "customer_name", "national_id", "sn_", "wp_", "stg_", "raw_id", "user_login"} {
			if strings.Contains(body, leak) {
				t.Fatalf("leak %q in %s", leak, body)
			}
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, p := range []string{"/api/v1/cases", "/api/v1/custody/assignments", "/api/v1/sales/custody-parity"} {
			req := httptest.NewRequest(m, p, nil)
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 405 {
				t.Fatalf("%s %s = %d", m, p, rec.Code)
			}
		}
	}
	if e.md5(t, salesTables...) != fp {
		t.Fatal("a GET changed cases/custody state")
	}
}

// INCOMPLETE source marks nothing absent; a COMPLETE empty source marks
// projections absent and keeps every event, history row and snapshot.
func TestSalesAbsenceSemantics(t *testing.T) {
	e := setupSales(t)
	b := e.stage(t, "syn-sales-a", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	e.importSales(t, b)
	empty := func(c snapshot.Completeness) []snapshot.EntityRows {
		ents := legacytest.SalesScenario(c)
		for i := range ents {
			ents[i].Rows = nil
		}
		return ents
	}
	e.importSales(t, e.stage(t, "syn-sales-b", empty(snapshot.Incomplete)))
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE presence = 'PRESENT'`) != 6 ||
		e.count(t, `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT'`) != 6 {
		t.Fatal("incomplete source marked absence")
	}
	e.importSales(t, e.stage(t, "syn-sales-c", empty(snapshot.Complete)))
	if e.count(t, `SELECT count(*) FROM cases.work_case WHERE presence = 'ABSENT_IN_LATEST'`) != 6 ||
		e.count(t, `SELECT count(*) FROM custody.assignment WHERE presence = 'ABSENT_IN_LATEST'`) != 6 ||
		e.count(t, `SELECT count(*) FROM custody.custody_event`) != 8 || e.count(t, `SELECT count(*) FROM cases.case_state_history`) != 2 ||
		e.count(t, `SELECT count(*) FROM cases.attribution_snapshot`) != 16 {
		t.Fatal("complete empty source must mark absent and keep history")
	}
}

func (e *salesEnv) salesServer(t *testing.T) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, salesFlag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	at, _ := timeutil.ParseRFC3339("2026-10-31T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Sales: salesSurface{caseshttp.New(guard, e.casesQ), custodyhttp.New(guard, e.custodyQ)}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// Same composition as cmd/crm-api (which cannot be imported from a test).
type (
	casesHandlers   = caseshttp.Handlers
	custodyHandlers = custodyhttp.Handlers
)

type salesSurface struct {
	*casesHandlers
	*custodyHandlers
}

func (e *salesEnv) setSalesFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: salesFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}
