package shadow_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Same composition as cmd/crm-api (which cannot be imported from a test).
type (
	authorizationHandlers = authzhttp.Handlers
	identityHandlers      = identityhttp.Handlers
	organizationHandlers  = organizationhttp.Handlers
)

type readAPI struct {
	*authorizationHandlers
	*identityHandlers
	*organizationHandlers
	guard *authorization.Guard
}

func (r readAPI) HRReadEnabled(ctx context.Context) bool {
	on, err := r.guard.Enabled(ctx)
	return err == nil && on
}

const flagKey = "organization.hr_read_shadow"

func (e *env) server(t *testing.T, env authstub.Environment, devAuth bool) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	enabled := func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, flagKey)
			return err
		})
		return on, err
	}
	var subjects authorization.SubjectResolver = authorization.Anonymous{}
	if devAuth {
		subjects = authorization.DevSubjects{}
	}
	guard, err := authorization.NewGuard(enabled, subjects)
	if err != nil {
		t.Fatal(err)
	}
	read := readAPI{authorizationHandlers: authzhttp.New(guard),
		identityHandlers:     identityhttp.New(guard, identityquery.New(e.Runner(t, identity.Module))),
		organizationHandlers: organizationhttp.New(guard, organizationquery.New(e.Runner(t, organization.Module))), guard: guard}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = env, devAuth
	at, _ := timeutil.ParseRFC3339("2026-10-29T10:00:00Z")
	h, err := apiserver.NewHandlerWith(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)), read, nil)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (e *env) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-10-29T09:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: flagKey, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

type resp struct {
	code int
	body string
	json map[string]any
}

func get(t *testing.T, h http.Handler, target, user string) resp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if user != "" {
		req.Header.Set(authstub.Header, user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := resp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

func items(r resp) []map[string]any {
	raw, _ := r.json["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, x := range raw {
		out = append(out, x.(map[string]any))
	}
	return out
}

func (e *env) fingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, tbl := range []string{"identity.user_account", "identity.role_grant", "identity.import_run", "organization.person", "organization.employment",
		"organization.org_unit", "organization.org_assignment", "organization.org_assignment_history", "organization.hierarchy_finding",
		"organization.position", "organization.level", "organization.position_role_default", "organization.import_run"} {
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

func TestReadAPIEndToEnd(t *testing.T) {
	e := setup(t)
	b := e.stage(t, "synthetic-hr-api-0001", legacytest.HRScenario())
	e.importAll(t, b)
	h := e.server(t, authstub.Test, true)
	before := e.fingerprint(t)

	// Flag off (default): BLOCKED / FEATURE_DISABLED, no data.
	r := get(t, h, "/api/v1/organization/persons", "dev-user-001")
	if r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil || !strings.Contains(r.body, "FEATURE_DISABLED") {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	if s := get(t, h, "/api/v1/session", ""); !strings.Contains(s.body, `"hr_read":false`) {
		t.Fatalf("session flag off: %s", s.body)
	}
	e.setFlag(t, true)
	before = e.fingerprint(t) // flag write happened in the flag store only
	if s := get(t, h, "/api/v1/session", ""); !strings.Contains(s.body, `"hr_read":true`) {
		t.Fatalf("session flag on: %s", s.body)
	}

	// Without the capability: UNAUTHORIZED (anonymous, and the synthetic viewer).
	for _, user := range []string{"", "dev-user-002"} {
		r := get(t, h, "/api/v1/organization/persons", user)
		if r.code != 403 || r.json["outcome"] != "UNAUTHORIZED" || !strings.Contains(r.body, "ACTION_DENIED") || r.json["items"] != nil {
			t.Fatalf("%q: %d %s", user, r.code, r.body)
		}
	}
	// Production: the dev header is never read, so even dev-user-001 is anonymous.
	prod := e.server(t, authstub.Production, false)
	if r := get(t, prod, "/api/v1/organization/persons", "dev-user-001"); r.code != 403 {
		t.Fatalf("production with dev header: %d %s", r.code, r.body)
	}

	// Allowed: cursor pagination over 14 persons, no duplicates.
	seen := map[string]bool{}
	target := "/api/v1/organization/persons?limit=5"
	for pages := 0; ; pages++ {
		r := get(t, h, target, "dev-user-001")
		if r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("persons page: %d %s", r.code, r.body)
		}
		for _, p := range items(r) {
			id := p["person_id"].(string)
			if seen[id] {
				t.Fatalf("duplicate %s across pages", id)
			}
			seen[id] = true
		}
		page := r.json["page"].(map[string]any)
		if page["has_more"] != true {
			break
		}
		target = "/api/v1/organization/persons?limit=5&cursor=" + url.QueryEscape(page["next_cursor"].(string))
		if pages > 5 {
			t.Fatal("pagination does not terminate")
		}
	}
	if len(seen) != 14 {
		t.Fatalf("paged %d persons", len(seen))
	}
	r = get(t, h, "/api/v1/organization/persons?limit=200", "dev-user-001")
	aa := r.json["allowed_actions"].(map[string]any)
	for _, c := range []string{"identity.user.read", "org.person.read", "org.employment.read", "org.position.read", "org.unit.read", "org.assignment.read"} {
		if aa[c].(map[string]any)["allowed"] != true {
			t.Fatalf("allowed_actions %s = %v", c, aa[c])
		}
	}
	// Server-side filters and sorts.
	if r := get(t, h, "/api/v1/organization/persons?employment_status=UNMAPPED", "dev-user-001"); len(items(r)) != 1 {
		t.Fatalf("status filter: %s", r.body)
	}
	positions := get(t, h, "/api/v1/organization/positions", "dev-user-001")
	var hrPos string
	for _, p := range items(positions) {
		if p["slug"] == "hr" {
			hrPos = p["id"].(string)
		}
	}
	if r := get(t, h, "/api/v1/organization/persons?position_id="+hrPos, "dev-user-001"); len(items(r)) != 3 {
		t.Fatalf("position filter: %d", len(items(r)))
	}
	desc := items(get(t, h, "/api/v1/organization/persons?sort=-display_name&limit=200", "dev-user-001"))
	if desc[0]["display_name"].(string) < desc[len(desc)-1]["display_name"].(string) {
		t.Fatal("descending sort not applied")
	}
	for _, bad := range []string{"?sort=national_id", "?sort=display_name;drop", "?employment_status=active", "?cursor=abc", "?limit=0", "?q="} {
		if r := get(t, h, "/api/v1/organization/persons"+bad, "dev-user-001"); r.code != 400 || !strings.Contains(r.body, "CONTRACT_INVALID") {
			t.Fatalf("%s: %d %s", bad, r.code, r.body)
		}
	}
	page1 := get(t, h, "/api/v1/organization/persons?limit=2", "dev-user-001")
	cur := page1.json["page"].(map[string]any)["next_cursor"].(string)
	if r := get(t, h, "/api/v1/organization/persons?sort=-display_name&cursor="+url.QueryEscape(cur), "dev-user-001"); r.code != 400 {
		t.Fatalf("cursor reused with another sort: %d", r.code)
	}

	// Person detail: conflicts are visible, history preserved, nothing fixed.
	var sellerB string
	for _, p := range items(r) {
		if p["display_name"] == "Synthetic Seller B" {
			sellerB = p["person_id"].(string)
		}
	}
	d := get(t, h, "/api/v1/organization/persons/"+sellerB, "dev-user-001")
	data := d.json["data"].(map[string]any)
	person := data["person"].(map[string]any)
	if person["manager_state"] != "MULTIPLE" || person["manager"] != nil || len(data["assignments"].([]any)) != 2 {
		t.Fatalf("seller B detail: %s", d.body)
	}
	if !strings.Contains(d.body, "MEMBERSHIP_MISMATCH") || !strings.Contains(d.body, "MULTIPLE_CURRENT_PARENTS") || !strings.Contains(d.body, `"interval_zone":"UNKNOWN"`) {
		t.Fatalf("seller B findings: %s", d.body)
	}
	emp := person["employment"].(map[string]any)
	if emp["status"] != "UNMAPPED" || emp["status_raw"] != "synthetic_on_leave_x" || emp["hire_date"] != "2025-01-15" {
		t.Fatalf("employment: %v", emp)
	}
	if r := get(t, h, "/api/v1/organization/persons/0b9c2f5e-1111-4111-8111-111111111111", "dev-user-001"); r.code != 404 {
		t.Fatalf("unknown person: %d", r.code)
	}

	// Org tree: orphans are not attached to a root.
	units := get(t, h, "/api/v1/organization/org-units", "dev-user-001")
	missing := 0
	for _, u := range items(units) {
		if u["parent_state"] == "MISSING" {
			missing++
			if u["parent_id"] != nil {
				t.Fatalf("orphan attached: %v", u)
			}
		}
	}
	if len(items(units)) != 7 || missing != 2 {
		t.Fatalf("org units: %s", units.body)
	}
	if r := get(t, h, "/api/v1/organization/hierarchy-findings?kind=CYCLE", "dev-user-001"); len(items(r)) != 2 {
		t.Fatalf("cycle findings: %s", r.body)
	}
	if r := get(t, h, "/api/v1/organization/assignments?is_current=true&relationship_kind=REPORTS_TO", "dev-user-001"); len(items(r)) != 12 {
		t.Fatalf("current assignments: %d", len(items(r)))
	}
	if r := get(t, h, "/api/v1/identity/users?limit=200", "dev-user-001"); len(items(r)) != 13 || strings.Contains(r.body, `"login_enabled":true`) {
		t.Fatalf("users: %s", r.body)
	}

	// Effective access: the same evaluator.
	ea := get(t, h, "/api/v1/authorization/effective-access", "dev-user-001")
	if strings.Count(ea.body, `"allowed":true`) != 9 || !strings.Contains(ea.body, `"matched_roles":["hr"]`) ||
		!strings.Contains(ea.body, `"capability":"customers.customer.read","matched_roles":["synthetic_dev_stub"]`) {
		t.Fatalf("effective access: %s", ea.body)
	}
	if ea := get(t, h, "/api/v1/authorization/effective-access", "dev-user-002"); strings.Contains(ea.body, `"allowed":true`) {
		t.Fatalf("viewer effective access: %s", ea.body)
	}

	// No sensitive value and no staging column ever leaves the API.
	all := []string{"/api/v1/organization/persons?limit=200", "/api/v1/identity/users?limit=200", "/api/v1/organization/assignments?limit=200",
		"/api/v1/organization/hierarchy-findings?limit=200", "/api/v1/organization/org-units", "/api/v1/organization/positions",
		"/api/v1/organization/levels", "/api/v1/organization/persons/" + sellerB}
	for _, target := range all {
		body := get(t, h, target, "dev-user-001").body
		for _, marker := range []string{"SYNTHETIC-NID", "TEST-PHONE", "999000001", "synthetic sensitive note", "synthetic reason text",
			"example.invalid", "dev-hr-user-", "synthetic_contract", "synthetic_training", "raw_", "user_pass", "sn_hr", "legacy_role"} {
			if strings.Contains(body, marker) {
				t.Fatalf("%s leaks %q", target, marker)
			}
		}
	}

	// Every read above wrote nothing.
	if after := e.fingerprint(t); after != before {
		t.Fatal("a GET changed identity/organization data")
	}
}
