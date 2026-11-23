package shadow_test

// RM-P05 MIS operational workspace: the narrow read additions.
//
//   - GET /api/v1/custody/assignments?case_id=…(repeat) returns the custody
//     projections of one page of Cases (the inventory's custody column).
//   - GET /api/v1/custody/assignments?finding_kind= keeps projections with a
//     PRESENT custody finding of that kind (exception queue).
//   - GET /api/v1/custody/assignments?holder_link=UNRESOLVED keeps
//     projections whose derived or legacy holder reference has no resolved
//     identity user (the legacy key is kept; nothing is inferred).
//   - GET /api/v1/cases?source_key= is an exact provenance lookup.
//
// Every expectation is checked against the stored rows of a disposable
// PostgreSQL loaded with synthetic fixtures, and every read is proven to
// write nothing. ABSENT_IN_LATEST never becomes "eligible": no operation,
// route or field of that kind exists.

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	custodyhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// salesServerWith composes only the sales read surface with a TEST-ONLY
// synthetic subject resolver (actor header → direct capability grant).
func (e *salesEnv) salesServerWith(t *testing.T, subjects authorization.SubjectResolver) http.Handler {
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
	}, subjects)
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment = authstub.Test
	at, _ := timeutil.ParseRFC3339("2026-11-21T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Sales: salesSurface{caseshttp.New(guard, e.casesQ), custodyhttp.New(guard, e.custodyQ)}})
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, r.Header.Get(actorHeader))))
	})
}

func p05Get(t *testing.T, h http.Handler, target, actor string) resp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set(actorHeader, actor)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := resp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

func caseIDsOf(r resp) []string {
	var out []string
	for _, it := range items(r) {
		out = append(out, it["case_id"].(string))
	}
	sort.Strings(out)
	return out
}

func (e *salesEnv) ids(t *testing.T, q string, args ...any) []string {
	t.Helper()
	rows, err := e.admin.Query(context.Background(), q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func p05Subjects() p04Subjects {
	return p04Subjects{
		"mis":        {identityapi.CapCaseRead, identityapi.CapCustodyRead, identityapi.CapSalesReportRead},
		"cases-only": {identityapi.CapCaseRead},
		"custody":    {identityapi.CapCustodyRead},
		"unrelated":  {identityapi.CapPersonRead},
	}
}

func TestRMP05CustodyFilters(t *testing.T) {
	e := setupSales(t)
	b := e.stage(t, "syn-mis-p05-001", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	e.importSales(t, b)
	h := e.salesServerWith(t, p05Subjects())

	// Flag off: BLOCKED, never an empty page dressed as zero.
	if r := p05Get(t, h, "/api/v1/custody/assignments?holder_link=UNRESOLVED", "mis"); r.code != 200 || !strings.Contains(r.body, `"FEATURE_DISABLED"`) || strings.Contains(r.body, `"items"`) {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setSalesFlag(t, true)
	fp := e.md5(t, salesTables...)

	all := e.ids(t, `SELECT case_id::text FROM custody.assignment`)
	if len(all) != 6 {
		t.Fatalf("fixture: %d projections", len(all))
	}

	// case_id: exactly the named projections; keyset pages stay inside the set.
	pick := []string{all[0], all[2], all[4]}
	q := url.Values{}
	for _, id := range pick {
		q.Add("case_id", id)
	}
	if r := p05Get(t, h, "/api/v1/custody/assignments?"+q.Encode(), "mis"); r.code != 200 || !slices.Equal(caseIDsOf(r), pick) {
		t.Fatalf("case_id: %d %v %s", r.code, caseIDsOf(r), r.body)
	}
	q.Set("limit", "2")
	first := p05Get(t, h, "/api/v1/custody/assignments?"+q.Encode(), "mis")
	page, _ := first.json["page"].(map[string]any)
	next, _ := page["next_cursor"].(string)
	if len(items(first)) != 2 || page["has_more"] != true || next == "" {
		t.Fatalf("case_id page 1: %s", first.body)
	}
	q.Set("cursor", next)
	second := p05Get(t, h, "/api/v1/custody/assignments?"+q.Encode(), "mis")
	if got := append(caseIDsOf(first), caseIDsOf(second)...); !slices.Equal(got, pick) || second.json["page"].(map[string]any)["has_more"] != false {
		t.Fatalf("case_id keyset: %v", got)
	}
	// An unknown case id is an empty page, never a fabricated holder.
	if r := p05Get(t, h, "/api/v1/custody/assignments?case_id=00000000-0000-4000-8000-000000000000", "mis"); r.code != 200 || len(items(r)) != 0 {
		t.Fatalf("unknown case id: %s", r.body)
	}
	tooMany := url.Values{}
	for i := 0; i < 201; i++ {
		tooMany.Add("case_id", all[i%len(all)])
	}
	for _, bad := range []string{"case_id=not-a-uuid", tooMany.Encode(), "finding_kind=FIXED", "holder_link=RESOLVED", "holder_link=", "parity=ELIGIBLE",
		"presence=ELIGIBLE", "derivation=REUSABLE"} {
		if r := p05Get(t, h, "/api/v1/custody/assignments?"+bad, "mis"); r.code != 400 {
			t.Fatalf("%s must be 400, got %d %s", bad[:min(len(bad), 40)], r.code, r.body)
		}
	}

	// finding_kind: the API set equals the stored PRESENT findings, per kind.
	nonEmpty := 0
	for _, kind := range []string{"MISSING_PRIOR_OWNER", "MISSING_TARGET_OWNER", "HISTORY_GAP", "MULTIPLE_CURRENT", "ORDERING_CONTRADICTION",
		"MISSING_EVENT_HISTORY", "SOURCE_CURRENT_MISMATCH", "UNKNOWN_ACTION"} {
		want := e.ids(t, `SELECT DISTINCT case_id::text FROM custody.finding WHERE presence = 'PRESENT' AND kind = $1`, kind)
		r := p05Get(t, h, "/api/v1/custody/assignments?limit=200&finding_kind="+kind, "mis")
		if r.code != 200 || !slices.Equal(caseIDsOf(r), want) {
			t.Fatalf("finding_kind=%s: got %v want %v", kind, caseIDsOf(r), want)
		}
		if len(want) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		t.Fatal("fixture has no custody finding: the filter would be untested")
	}

	// holder_link=UNRESOLVED: a holder key exists without a resolved user.
	wantUnresolved := e.ids(t, `SELECT case_id::text FROM custody.assignment
		WHERE (derived_legacy_key IS NOT NULL AND derived_user_id IS NULL) OR (legacy_holder_key IS NOT NULL AND legacy_holder_user_id IS NULL)`)
	unresolved := p05Get(t, h, "/api/v1/custody/assignments?limit=200&holder_link=UNRESOLVED", "mis")
	if unresolved.code != 200 || !slices.Equal(caseIDsOf(unresolved), wantUnresolved) {
		t.Fatalf("holder_link: got %v want %v", caseIDsOf(unresolved), wantUnresolved)
	}
	for _, it := range items(unresolved) {
		ok := false
		for _, side := range []string{"derived_holder", "legacy_holder"} {
			if u, has := it[side].(map[string]any); has && u["user_id"] == nil && u["legacy_key"] != "" {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("holder_link row without an unresolved reference: %v", it)
		}
	}
	// Filters are conjunctive.
	wantAnd := e.ids(t, `SELECT case_id::text FROM custody.assignment WHERE parity = 'CONFLICT' AND
		((derived_legacy_key IS NOT NULL AND derived_user_id IS NULL) OR (legacy_holder_key IS NOT NULL AND legacy_holder_user_id IS NULL))`)
	if r := p05Get(t, h, "/api/v1/custody/assignments?parity=CONFLICT&holder_link=UNRESOLVED", "mis"); !slices.Equal(caseIDsOf(r), wantAnd) {
		t.Fatalf("AND: got %v want %v", caseIDsOf(r), wantAnd)
	}

	// source_key: exact provenance lookup of the Case's own source row.
	want501 := e.ids(t, `SELECT case_id::text FROM cases.case_source WHERE source_key = '501'`)
	if r := p05Get(t, h, "/api/v1/cases?source_key=501", "mis"); r.code != 200 || len(want501) != 1 || !slices.Equal(caseIDsOf(r), want501) {
		t.Fatalf("source_key: %v want %v", caseIDsOf(r), want501)
	}
	if r := p05Get(t, h, "/api/v1/cases?source_key=501&workflow_type=LEGACY_LEAD", "mis"); r.code != 200 || len(items(r)) != 0 {
		t.Fatalf("source_key AND workflow_type: %s", r.body)
	}
	for _, bad := range []string{"source_key=50", "source_key=0", "source_key=abc", "source_key=5%25", "phone=0912", "q=501"} {
		r := p05Get(t, h, "/api/v1/cases?"+bad, "mis")
		if bad == "source_key=50" {
			if r.code != 200 || len(items(r)) != 0 {
				t.Fatalf("source_key is exact, not a prefix: %s", r.body)
			}
			continue
		}
		if r.code != 400 {
			t.Fatalf("%s must be 400, got %d", bad, r.code)
		}
	}

	// Permission boundaries: each read keeps its own capability.
	for _, c := range []struct {
		actor, path string
		code        int
	}{
		{"cases-only", "/api/v1/cases?source_key=501", 200},
		{"cases-only", "/api/v1/custody/assignments?holder_link=UNRESOLVED", 403},
		{"cases-only", "/api/v1/sales/custody-parity", 403},
		{"custody", "/api/v1/cases?source_key=501", 403},
		{"custody", "/api/v1/custody/assignments?finding_kind=HISTORY_GAP", 200},
		{"unrelated", "/api/v1/custody/assignments?case_id=" + all[0], 403},
		{"", "/api/v1/custody/assignments", 403},
	} {
		r := p05Get(t, h, c.path, c.actor)
		if r.code != c.code {
			t.Fatalf("%s %s = %d, want %d", c.actor, c.path, r.code, c.code)
		}
		if c.code == 403 && strings.Contains(r.body, `"items"`) {
			t.Fatalf("denied body carries rows: %s", r.body)
		}
	}

	// No write exists on these paths and no operation route was added.
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, p := range []string{"/api/v1/custody/assignments", "/api/v1/custody/cases/" + all[0], "/api/v1/cases"} {
			req := httptest.NewRequest(m, p, nil)
			req.Header.Set(actorHeader, "mis")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s = %d", m, p, rec.Code)
			}
		}
	}
	for _, p := range []string{"/return", "/release", "/reset", "/reassign", "/recycle", "/assign"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/custody/cases/"+all[0]+p, nil)
		req.Header.Set(actorHeader, "mis")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST custody %s = %d (no operation route may exist)", p, rec.Code)
		}
	}
	if e.md5(t, salesTables...) != fp {
		t.Fatal("an RM-P05 read changed cases/custody state")
	}
}

// Cases missing from the latest COMPLETE import stay listed as
// ABSENT_IN_LATEST with their history, attribution and custody evidence.
// Absence is never turned into eligibility, reuse or release.
func TestRMP05AbsentInLatestNeverEligible(t *testing.T) {
	e := setupSales(t)
	b := e.stage(t, "syn-mis-p05-a", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	e.importSales(t, b)
	history := e.md5(t, "cases.attribution_snapshot", "custody.custody_event", "cases.case_state_history")
	parityBefore := e.ids(t, `SELECT case_id::text || ':' || parity || ':' || derivation || ':' || COALESCE(derived_legacy_key, '-') FROM custody.assignment`)

	empty := legacytest.SalesScenario(snapshot.Complete)
	for i := range empty {
		empty[i].Rows = nil
	}
	e.importSales(t, e.stage(t, "syn-mis-p05-b", empty))
	e.setSalesFlag(t, true)
	h := e.salesServerWith(t, p05Subjects())
	fp := e.md5(t, salesTables...)

	if e.md5(t, "cases.attribution_snapshot", "custody.custody_event", "cases.case_state_history") != history {
		t.Fatal("absence rewrote history or attribution")
	}
	if got := e.ids(t, `SELECT case_id::text || ':' || parity || ':' || derivation || ':' || COALESCE(derived_legacy_key, '-') FROM custody.assignment`); !slices.Equal(got, parityBefore) {
		t.Fatalf("absence changed custody evidence: %v -> %v", parityBefore, got)
	}
	absentCases := p05Get(t, h, "/api/v1/cases?presence=ABSENT_IN_LATEST&limit=200", "mis")
	absentCustody := p05Get(t, h, "/api/v1/custody/assignments?presence=ABSENT_IN_LATEST&limit=200", "mis")
	if len(items(absentCases)) != 6 || len(items(absentCustody)) != 6 {
		t.Fatalf("absent: cases %d custody %d", len(items(absentCases)), len(items(absentCustody)))
	}
	if r := p05Get(t, h, "/api/v1/custody/assignments?presence=PRESENT", "mis"); len(items(r)) != 0 {
		t.Fatalf("present after complete empty import: %s", r.body)
	}
	// The absent rows still carry their holder evidence (derived holder is
	// not cleared): "absent" is not "released", "returned" or "free".
	held := 0
	for _, it := range items(absentCustody) {
		if _, ok := it["derived_holder"].(map[string]any); ok {
			held++
		}
		if it["presence"] != "ABSENT_IN_LATEST" {
			t.Fatalf("row presence: %v", it)
		}
	}
	if held == 0 {
		t.Fatal("fixture: no absent projection keeps a derived holder")
	}
	for _, body := range []string{absentCases.body, absentCustody.body, p05Get(t, h, "/api/v1/sales/custody-parity", "mis").body} {
		for _, word := range []string{"ELIGIBLE", "eligible", "REUSABLE", "reusable", "RECYCLE", "AVAILABLE_FOR"} {
			if strings.Contains(body, word) {
				t.Fatalf("absence surfaced as %q: %s", word, body)
			}
		}
	}
	// allowed_actions never offers a write on a read page.
	for _, it := range []resp{absentCases, absentCustody} {
		acts, _ := it.json["allowed_actions"].(map[string]any)
		for k, v := range acts {
			a, _ := v.(map[string]any)
			if strings.HasPrefix(k, "custody.") || strings.HasPrefix(k, "cases.") {
				if !strings.HasSuffix(k, ".read") && a["allowed"] == true {
					t.Fatalf("allowed_actions offers %s on an absent read", k)
				}
			}
		}
	}
	if e.md5(t, salesTables...) != fp {
		t.Fatal("an absent-row read changed state")
	}
}
