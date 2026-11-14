package shadow_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	biawinquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/query"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	fulfillmentquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

const biawinFlag = "biawin_fulfillment.read_shadow"

// biawinSurface composes both module handler sets behind one flag (as
// cmd/crm-api does).
type (
	bwHandlers = biawinhttp.Handlers
	fwHandlers = fulfillmenthttp.Handlers
	bwSurface  struct {
		*bwHandlers
		*fwHandlers
	}
)

// server builds the API over the real owners; now is the read clock (the
// freshness of a run is judged against it).
func (e *biawinEnv) server(t *testing.T, now time.Time) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, biawinFlag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	at, err := timeutil.NewInstant(now)
	if err != nil {
		t.Fatal(err)
	}
	clock := timeutil.FixedClock{At: at}
	bq, err := biawinquery.New(e.bR, clock)
	if err != nil {
		t.Fatal(err)
	}
	fq, err := fulfillmentquery.New(e.fR, clock)
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	h, err := apiserver.NewHandlerWithSurfaces(cfg, clock, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Biawin: bwSurface{biawinhttp.New(guard, bq), fulfillmenthttp.New(guard, fq)}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *biawinEnv) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-02T08:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: biawinFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

var biawinGets = []string{"/api/v1/biawin/subscriptions", "/api/v1/biawin/parity", "/api/v1/fulfillment/cases"}

func findItem(t *testing.T, r resp, key, val string) map[string]any {
	t.Helper()
	for _, it := range items(r) {
		if it[key] == val {
			return it
		}
	}
	t.Fatalf("no item with %s=%s in %s", key, val, r.body[:min(400, len(r.body))])
	return nil
}

func summary(r resp) map[string]float64 {
	out := map[string]float64{}
	raw, _ := r.json["summary"].([]any)
	for _, x := range raw {
		m := x.(map[string]any)
		k, _ := m["key"].(string)
		if k == "" {
			k = m["kind"].(string) + "/" + m["status"].(string)
		}
		out[k] = m["count"].(float64)
	}
	return out
}

// Read-only proof over HTTP against the real owners: flag, capability,
// run pinning, explicit states, filters, 405 / guessed routes, no GET write.
func TestBiawinHTTPReadOnly(t *testing.T) {
	e := setupBiawin(t, financemigrations.Source())
	if _, err := migrations.Up(context.Background(), e.DSN, financemigrations.Source()); err != nil {
		t.Fatal(err)
	}
	b := e.loadedBiawin(t, "syn-bw-http")
	now, _ := time.Parse(time.RFC3339, "2026-11-02T09:00:00Z")
	h := e.server(t, now)

	// Flag off (default): BLOCKED / FEATURE_DISABLED, no data.
	for _, p := range biawinGets {
		if r := get(t, h, p, "dev-user-001"); r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil || r.json["check"] != nil {
			t.Fatalf("%s with the flag off: %d %s", p, r.code, r.body)
		}
	}
	e.setFlag(t, true)
	// No run yet: UNKNOWN, never an empty zero answer.
	for _, p := range biawinGets {
		if r := get(t, h, p, "dev-user-001"); r.json["outcome"] != "UNKNOWN" || r.json["items"] != nil || r.json["summary"] != nil {
			t.Fatalf("no run %s: %s", p, r.body)
		}
	}
	br, fr := e.importBoth(t, b)

	fp := e.md5(t, append(append(append(biawinTables, fulfillmentTables...), ownerTables...), financeTables...)...)
	stagedTables := []string{"integration.import_batch", "integration.import_entity", "integration.stg_project_membership_v1", "integration.stg_project_membership_item_v1",
		"integration.stg_operations_execution_case_v1", "integration.stg_shipping_order_v1"}
	staged := e.md5(t, stagedTables...)
	audit := e.md5(t, "audit.audit_event")

	// Every GET answers SHADOW, pinned to its run, repeatedly.
	for i := 0; i < 3; i++ {
		for _, p := range biawinGets {
			r := get(t, h, p, "dev-user-001")
			want := br.RunID
			if strings.HasPrefix(p, "/api/v1/fulfillment") {
				want = fr.RunID
			}
			if r.code != 200 || (r.json["outcome"] != "SUCCESS" && r.json["outcome"] != "PARTIAL") || !strings.Contains(r.body, `"SHADOW"`) ||
				!strings.Contains(r.body, want) || !strings.Contains(r.body, "Not live provider state") {
				t.Fatalf("%s: %d %s", p, r.code, r.body[:min(400, len(r.body))])
			}
		}
	}

	// Subscriptions: raw states kept; UNKNOWN is explicit; summary is the
	// full filtered set (pagination never changes it).
	subs := get(t, h, "/api/v1/biawin/subscriptions?limit=200", "dev-user-001")
	if len(items(subs)) != 5 {
		t.Fatalf("subscriptions: %s", subs.body)
	}
	sum := summary(subs)
	if sum["ACTIVE"] != 3 || sum["SUSPENDED"] != 1 || sum["UNKNOWN"] != 1 {
		t.Fatalf("summary: %v", sum)
	}
	if s503 := findItem(t, subs, "source_key", "503"); s503["state_mapped"] != "UNKNOWN" || s503["state_raw"] != "synthetic_paused" {
		t.Fatalf("unknown raw kept: %v", s503)
	}
	s504 := findItem(t, subs, "source_key", "504")
	if c := s504["customer"].(map[string]any); c["state"] != "UNKNOWN" || c["id"] != nil {
		t.Fatalf("unresolved customer stays UNKNOWN: %v", c)
	}
	if c := s504["case"].(map[string]any); c["state"] != "UNKNOWN" || c["legacy_key"] != "77" {
		t.Fatalf("Dot ref kept raw: %v", c)
	}
	page1 := get(t, h, "/api/v1/biawin/subscriptions?limit=2", "dev-user-001")
	pg := page1.json["page"].(map[string]any)
	if len(items(page1)) != 2 || pg["has_more"] != true || summary(page1)["ACTIVE"] != 3 {
		t.Fatalf("page 1: %s", page1.body)
	}
	page2 := get(t, h, "/api/v1/biawin/subscriptions?limit=2&cursor="+pg["next_cursor"].(string), "dev-user-001")
	if len(items(page2)) != 2 || items(page2)[0]["subscription_id"] == items(page1)[1]["subscription_id"] {
		t.Fatalf("page 2: %s", page2.body)
	}
	if r := get(t, h, "/api/v1/biawin/subscriptions?state=ACTIVE", "dev-user-001"); len(items(r)) != 3 {
		t.Fatalf("state filter: %s", r.body)
	}

	// Detail + items: 12 distinct items; identical titles never merged.
	s501 := findItem(t, subs, "source_key", "501")
	id501 := s501["subscription_id"].(string)
	if s501["product_version_state"] != "UNKNOWN" || s501["item_count"] != float64(12) {
		t.Fatalf("501: %v", s501)
	}
	if r := get(t, h, "/api/v1/biawin/subscriptions/"+id501, "dev-user-001"); r.code != 200 || r.json["history"] == nil {
		t.Fatalf("501 detail: %d %s", r.code, r.body)
	}
	its := get(t, h, "/api/v1/biawin/subscriptions/"+id501+"/items?limit=200", "dev-user-001")
	same := 0
	for _, it := range items(its) {
		if it["service_title_snapshot"] == "Synthetic Service A" {
			same++
		}
	}
	if len(items(its)) != 12 || its.json["item_count"] != float64(12) || same != 2 {
		t.Fatalf("items: %d %v %d", len(items(its)), its.json["item_count"], same)
	}
	if r := get(t, h, "/api/v1/biawin/subscriptions/"+id501+"/items?limit=5", "dev-user-001"); r.json["item_count"] != float64(12) || len(items(r)) != 5 {
		t.Fatalf("item count independent of page: %s", r.body)
	}
	// No cross-run drill-down: another run id is not found.
	other := "00000000-0000-4000-8000-0000000000ff"
	for _, p := range []string{"/api/v1/biawin/subscriptions/" + id501 + "?run_id=" + other, "/api/v1/biawin/subscriptions/" + id501 + "/items?run_id=" + other,
		"/api/v1/biawin/subscriptions/" + other} {
		if r := get(t, h, p, "dev-user-001"); r.code != 404 {
			t.Fatalf("%s: %d", p, r.code)
		}
	}

	// Parity: recorded offline, read here; never MATCH with an UNKNOWN row.
	par := get(t, h, "/api/v1/biawin/parity?limit=200", "dev-user-001")
	chk, _ := par.json["check"].(map[string]any)
	if chk == nil || chk["outcome"] != "UNKNOWN" || chk["matched"] != float64(3) || chk["unknown"] != float64(1) || chk["legacy_count"] != float64(3) ||
		chk["target_count"] != float64(3) || chk["cutoff_mismatch"] != false {
		t.Fatalf("parity: %s", par.body)
	}
	if r := get(t, h, "/api/v1/biawin/parity?result=UNKNOWN", "dev-user-001"); len(r.json["check"].(map[string]any)["rows"].([]any)) != 1 ||
		r.json["check"].(map[string]any)["matched"] != float64(3) {
		t.Fatalf("parity rows filter keeps the counts: %s", r.body)
	}

	// Fulfillment: kinds explicit; D-35 — a COMPLETED case does not make
	// its subscription active.
	cases := get(t, h, "/api/v1/fulfillment/cases?limit=200", "dev-user-001")
	if len(items(cases)) != 22 {
		t.Fatalf("cases: %d", len(items(cases)))
	}
	if r := get(t, h, "/api/v1/fulfillment/cases?kind=SHIPPING", "dev-user-001"); len(items(r)) != 2 {
		t.Fatalf("shipping cases: %s", r.body)
	}
	c1004 := findItem(t, cases, "source_key", "1004")
	if c1004["status_mapped"] != "COMPLETED" || c1004["kind"] != "EXECUTION" {
		t.Fatalf("1004: %v", c1004)
	}
	if s := findItem(t, subs, "source_key", "502"); s["state_mapped"] != "SUSPENDED" {
		t.Fatalf("D-35: a completed case never activates: %v", s)
	}
	// D-38 open: a legacy wallet charge is provider evidence only — a hashed
	// external reference, never a finance posting, never a balance.
	if c1004["subtype_raw"] != "wallet_charge" || c1004["provider_ref_scheme"] != "biawin.wallet_external_id" || c1004["provider_ref_sha256"] == nil {
		t.Fatalf("D-38 provider evidence: %v", c1004)
	}
	for _, tbl := range financeTables {
		if n := e.q1(t, `SELECT count(*)::text FROM `+tbl); n != "0" {
			t.Fatalf("D-38: a Biawin wallet charge must never reach finance: %s = %s", tbl, n)
		}
	}
	if strings.Contains(cases.body, "balance") || strings.Contains(subs.body, "balance") {
		t.Fatal("D-38: no Biawin wallet balance is modelled")
	}
	cid := c1004["case_id"].(string)
	if r := get(t, h, "/api/v1/fulfillment/cases/"+cid, "dev-user-001"); r.code != 200 || r.json["case"] == nil {
		t.Fatalf("case detail: %d %s", r.code, r.body)
	}
	if r := get(t, h, "/api/v1/fulfillment/cases/"+cid+"/history", "dev-user-001"); r.code != 200 || r.json["items"] == nil {
		t.Fatalf("case history: %d %s", r.code, r.body)
	}
	if r := get(t, h, "/api/v1/fulfillment/cases/"+cid+"?run_id="+other, "dev-user-001"); r.code != 404 {
		t.Fatalf("cross-run case: %d", r.code)
	}
	// External references are digests only.
	if strings.Contains(cases.body, "SYN-EXT") || strings.Contains(cases.body, "SYN-TRK") {
		t.Fatal("raw external refs leaked over HTTP")
	}

	// Unknown / phone / invalid filters are 400; no arbitrary query.
	for _, p := range []string{"/api/v1/biawin/subscriptions?phone=09000000000", "/api/v1/biawin/subscriptions?mobile=1", "/api/v1/fulfillment/cases?phone=1",
		"/api/v1/biawin/subscriptions?state=LIVE", "/api/v1/biawin/subscriptions?product=Synthetic", "/api/v1/fulfillment/cases?kind=PORTAL",
		"/api/v1/fulfillment/cases?status=DONE", "/api/v1/fulfillment/cases?assignee=09000000000", "/api/v1/biawin/parity?result=OK",
		"/api/v1/biawin/subscriptions?cursor=%%%", "/api/v1/biawin/subscriptions?customer_name=x", "/api/v1/fulfillment/cases?sql=select",
		"/api/v1/biawin/subscriptions?run_id=latest", "/api/v1/biawin/subscriptions?limit=500"} {
		if r := get(t, h, p, "dev-user-001"); r.code != 400 {
			t.Fatalf("%s: %d %s", p, r.code, r.body)
		}
	}
	// Mutating methods: 405. Guessed command routes: never 2xx.
	for _, p := range append(append([]string(nil), biawinGets...), "/api/v1/biawin/subscriptions/"+id501, "/api/v1/biawin/subscriptions/"+id501+"/items",
		"/api/v1/fulfillment/cases/"+cid, "/api/v1/fulfillment/cases/"+cid+"/history") {
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			req := httptest.NewRequest(m, p, strings.NewReader(`{}`))
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: %d", m, p, rec.Code)
			}
		}
	}
	for _, p := range []string{"/api/v1/biawin/subscriptions/" + id501 + "/activate", "/api/v1/biawin/subscriptions/" + id501 + "/suspend",
		"/api/v1/biawin/subscriptions/" + id501 + "/terminate", "/api/v1/biawin/orders", "/api/v1/biawin/charge-wallet", "/api/v1/biawin/wallet",
		"/api/v1/biawin/sync", "/api/v1/biawin/import", "/api/v1/fulfillment/cases/" + cid + "/assign", "/api/v1/fulfillment/cases/" + cid + "/complete",
		"/api/v1/fulfillment/cases/" + cid + "/ship", "/api/v1/fulfillment/cases/" + cid + "/evidence", "/api/v1/fulfillment/upgrades",
		"/api/v1/portal", "/api/v1/biawin/memberships"} {
		for _, m := range []string{http.MethodGet, http.MethodPost} {
			req := httptest.NewRequest(m, p, strings.NewReader(`{}`))
			req.Header.Set(authstub.Header, "dev-user-001")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code >= 200 && rec.Code < 300 {
				t.Fatalf("hidden command route %s %s answered %d", m, p, rec.Code)
			}
		}
	}
	// A principal without the read capabilities is denied (no real role
	// carries them; view is never implied).
	for _, p := range append(append([]string(nil), biawinGets...), "/api/v1/biawin/subscriptions/"+id501, "/api/v1/fulfillment/cases/"+cid) {
		if r := get(t, h, p, "dev-user-002"); r.code != 403 {
			t.Fatalf("%s for a viewer: %d", p, r.code)
		}
	}

	// A stale snapshot is labelled: PARTIAL with STALE_STATE, freshness STALE.
	stale := e.server(t, now.Add(30*24*time.Hour))
	if r := get(t, stale, "/api/v1/biawin/subscriptions", "dev-user-001"); r.json["outcome"] != "PARTIAL" || !strings.Contains(r.body, `"STALE_STATE"`) ||
		!strings.Contains(r.body, `"freshness":"STALE"`) {
		t.Fatalf("stale: %s", r.body[:min(400, len(r.body))])
	}

	// No GET wrote anything: biawin, fulfillment, owners, finance, staging
	// and audit are unchanged; no new run, no parity check, no case event.
	if e.md5(t, append(append(append(biawinTables, fulfillmentTables...), ownerTables...), financeTables...)...) != fp {
		t.Fatal("biawin / fulfillment / owner / finance state changed on read")
	}
	if e.md5(t, stagedTables...) != staged {
		t.Fatal("staged legacy evidence changed on read")
	}
	if e.md5(t, "audit.audit_event") != audit {
		t.Fatal("audit changed on read")
	}
	for tbl, want := range map[string]string{"biawin.import_run": "1", "biawin.parity_check": "1", "fulfillment.import_run": "1"} {
		if n := e.q1(t, `SELECT count(*)::text FROM `+tbl); n != want {
			t.Fatalf("a GET must never import or record parity: %s = %s", tbl, n)
		}
	}
}
