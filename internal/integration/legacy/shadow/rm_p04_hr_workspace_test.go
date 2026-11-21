package shadow_test

// RM-P04 HR operational workspace: the two narrow read additions.
//
//   - GET /api/v1/organization/persons?user_id= returns every person whose
//     employment links that identity user (a shared user stays a visible
//     conflict, never a merge). The link is an employment field, so the
//     filter is refused without org.employment.read.
//   - GET /api/v1/hr/requests?status=&subject_user_id= only narrows the
//     caller's existing row scope (requester / current reviewer / history
//     actor); it never widens it.
//
// Both run on a disposable PostgreSQL with synthetic fixtures and prove that
// nothing is written.

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	organizationhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// p04Subjects is a TEST-ONLY resolver: the synthetic actor header selects a
// direct synthetic capability grant. It carries no user id (the HR read
// surface does not use one).
type p04Subjects map[string][]identityapi.Capability

func (s p04Subjects) Subject(ctx context.Context) authorization.Subject {
	key, _ := ctx.Value(actorKey{}).(string)
	caps := s[key]
	if key == "" || len(caps) == 0 {
		return authorization.Subject{}
	}
	return authorization.Subject{PrincipalID: "synthetic:" + key, Grants: []identityapi.Grant{{UserID: key, LegacyValue: "synthetic",
		Status: identityapi.GrantActive, Basis: authorization.DevGrantBasis, Capabilities: caps}}}
}

// hrReadServer composes only the HR read surface with the given resolver.
func (e *env) hrReadServer(t *testing.T, subjects authorization.SubjectResolver) http.Handler {
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
	guard, err := authorization.NewGuard(enabled, subjects)
	if err != nil {
		t.Fatal(err)
	}
	available := func(ctx context.Context) (map[identityapi.Capability]bool, error) {
		on, err := enabled(ctx)
		if err != nil {
			return nil, err
		}
		out := map[identityapi.Capability]bool{}
		for _, c := range identityapi.HRReadCapabilities() {
			out[c.Key] = on
		}
		return out, nil
	}
	read := readAPI{authorizationHandlers: authzhttp.New(guard, available),
		identityHandlers:     identityhttp.New(guard, identityquery.New(e.Runner(t, identity.Module))),
		organizationHandlers: organizationhttp.New(guard, organizationquery.New(e.Runner(t, organization.Module))), guard: guard}
	cfg := apiserver.DefaultConfig()
	cfg.Environment = authstub.Test
	at, _ := timeutil.ParseRFC3339("2026-11-21T10:00:00Z")
	h, err := apiserver.NewHandlerWith(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)), read, nil)
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, r.Header.Get(actorHeader))))
	})
}

func TestRMP04PersonUserFilter(t *testing.T) {
	e := setup(t)
	b := e.stage(t, "synthetic-hr-rm-p04-0001", legacytest.HRScenario())
	e.importAll(t, b)
	e.setFlag(t, true)
	h := e.hrReadServer(t, p04Subjects{
		"hr":      hrReadKeys(),
		"person":  {identityapi.CapPersonRead},
		"no-emp":  {identityapi.CapPersonRead, identityapi.CapPositionRead, identityapi.CapUnitRead, identityapi.CapAssignmentRead},
		"unknown": {identityapi.CapCaseRead},
	})
	before := e.fingerprint(t)
	userOf := func(key string) string {
		var id string
		if err := e.admin.QueryRow(context.Background(), `SELECT user_id::text FROM identity.user_account WHERE source_key = $1`, key).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	get := func(target, actor string) wresp { return call(t, h, http.MethodGet, target, actor, "", nil) }

	// Legacy user 7 is shared by two HR profiles: both are returned, each still marked SHARED_USER.
	shared := userOf("7")
	r := get("/api/v1/organization/persons?user_id="+shared, "hr")
	if r.code != 200 || r.json["outcome"] != "SUCCESS" {
		t.Fatalf("user filter: %d %s", r.code, r.body)
	}
	got := r.json["items"].([]any)
	if len(got) != 2 {
		t.Fatalf("shared user must return both persons, got %d: %s", len(got), r.body)
	}
	for _, it := range got {
		emp := it.(map[string]any)["employment"].(map[string]any)
		u := emp["user"].(map[string]any)
		if u["user_id"] != shared || u["link_state"] != "SHARED_USER" {
			t.Fatalf("shared user person: %v", u)
		}
	}
	// A linked single user returns exactly the persons whose employment carries it.
	var single string
	if err := e.admin.QueryRow(context.Background(), `SELECT user_id::text FROM organization.employment WHERE user_link_state = 'LINKED'
		GROUP BY user_id HAVING count(*) = 1 ORDER BY 1 LIMIT 1`).Scan(&single); err != nil {
		t.Fatal(err)
	}
	if r := get("/api/v1/organization/persons?user_id="+single, "hr"); len(r.json["items"].([]any)) != 1 {
		t.Fatalf("single user: %s", r.body)
	}
	// A user with no employment: an empty, successful answer (no guess, no fallback).
	if r := get("/api/v1/organization/persons?user_id=00000000-0000-4000-8000-000000000000", "hr"); r.code != 200 || len(r.json["items"].([]any)) != 0 {
		t.Fatalf("unknown user: %d %s", r.code, r.body)
	}
	// Combined with another filter, both apply.
	if r := get("/api/v1/organization/persons?employment_status=TERMINATED&user_id="+shared, "hr"); len(r.json["items"].([]any)) != 0 {
		t.Fatalf("combined filter: %s", r.body)
	}
	// The filter is on a masked employment field: refused without org.employment.read, never silently answered.
	for _, actor := range []string{"person", "no-emp"} {
		if r := get("/api/v1/organization/persons?user_id="+shared, actor); r.code != 403 || r.json["items"] != nil {
			t.Fatalf("%s: %d %s", actor, r.code, r.body)
		}
	}
	// Unrelated capability / anonymous: denied before any filter is read.
	for _, actor := range []string{"unknown", ""} {
		if r := get("/api/v1/organization/persons?user_id="+shared, actor); r.code != 403 || !strings.Contains(r.body, "UNAUTHORIZED") {
			t.Fatalf("%q: %d %s", actor, r.code, r.body)
		}
	}
	for _, bad := range []string{"?user_id=7", "?user_id=not-a-uuid", "?user_id=09.123"} {
		if r := get("/api/v1/organization/persons"+bad, "hr"); r.code != 400 || !strings.Contains(r.body, "CONTRACT_INVALID") {
			t.Fatalf("%s: %d %s", bad, r.code, r.body)
		}
	}
	// Flag off: BLOCKED, no data, whatever the filter.
	e.setFlag(t, false)
	if r := get("/api/v1/organization/persons?user_id="+shared, "hr"); r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setFlag(t, true)
	if after := e.fingerprint(t); after != before {
		t.Fatal("a read changed identity or organization data")
	}
}

func TestRMP04HRRequestListFilters(t *testing.T) {
	e := setupWrites(t)
	ctx := context.Background()
	u := e.users
	// Supervisor A (4) terminates seller 7 (reviewer = senior 3); senior (3) transfers seller 6 to B (5).
	term, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrrequest.Create{RequestType: api.HRTerminateSeller, SubjectUserID: u["7"],
		Reason: "synthetic rm-p04 terminate", IdempotencyKey: "rm-p04-seed-term-0001", CorrelationID: "corr-rm-p04", Actor: e.hrActor("4")})
	if err != nil || term.RequestStatus != api.HRPendingReview {
		t.Fatalf("seed terminate: %+v %v", term, err)
	}
	target := u["5"]
	tr, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrrequest.Create{RequestType: api.HRTransferSeller, SubjectUserID: u["6"],
		TargetParentUserID: &target, Reason: "synthetic rm-p04 transfer", IdempotencyKey: "rm-p04-seed-tr-0001", CorrelationID: "corr-rm-p04",
		Actor: e.hrActor("3")})
	if err != nil || tr.RequestStatus != api.HRPendingReview {
		t.Fatalf("seed transfer: %+v %v", tr, err)
	}
	read := []identityapi.Capability{identityapi.CapHRRequestRead}
	srv := e.writesServer(t, testSubjects{users: u, caps: map[string][]identityapi.Capability{"3": read, "4": read, "8": read, "5": {identityapi.CapCaseRead}}})
	e.setFlags(t, map[string]bool{"organization.hr_request_write": true, "organization.hr_request_writeback": true})
	g := (&hrHarness{writesEnv: e, t: t}).hrGuard(t)
	tables := e.md5(t, hrRequestTables...)

	ids := func(r wresp) map[string]bool {
		out := map[string]bool{}
		items, _ := r.json["items"].([]any)
		for _, it := range items {
			out[it.(map[string]any)["request_id"].(string)] = true
		}
		return out
	}
	list := func(actor, query string) wresp {
		r := call(t, srv, http.MethodGet, "/api/v1/hr/requests"+query, actor, "", nil)
		if r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("%s %s: %d %s", actor, query, r.code, r.body)
		}
		return r
	}
	// Senior (3): requester of the transfer and reviewer of the terminate → both visible unfiltered.
	if got := ids(list("3", "")); len(got) != 2 || !got[term.RequestID] || !got[tr.RequestID] {
		t.Fatalf("senior unfiltered: %v", got)
	}
	// subject_user_id narrows within the scope.
	if got := ids(list("3", "?subject_user_id="+u["7"])); len(got) != 1 || !got[term.RequestID] {
		t.Fatalf("senior subject 7: %v", got)
	}
	if got := ids(list("3", "?status=pending_review&subject_user_id="+u["6"])); len(got) != 1 || !got[tr.RequestID] {
		t.Fatalf("senior pending_review subject 6: %v", got)
	}
	for _, st := range []string{"pending_hr", "approved", "rejected", "cancelled", "failed"} {
		if got := ids(list("3", "?status="+st)); len(got) != 0 {
			t.Fatalf("senior status %s: %v", st, got)
		}
	}
	// Supervisor A (4) sees only its own request; a filter naming seller 6 never reveals the senior's transfer.
	if got := ids(list("4", "")); len(got) != 1 || !got[term.RequestID] {
		t.Fatalf("supervisor A unfiltered: %v", got)
	}
	if got := ids(list("4", "?subject_user_id="+u["6"])); len(got) != 0 {
		t.Fatalf("a filter widened supervisor A's scope: %v", got)
	}
	// Supervisor C (8) holds the capability but no row: filters never widen to other requests.
	for _, q := range []string{"", "?status=pending_review", "?subject_user_id=" + u["7"]} {
		if got := ids(list("8", q)); len(got) != 0 {
			t.Fatalf("supervisor C %q: %v", q, got)
		}
	}
	// Validation: an unknown status or a malformed subject is 400; no capability is 403.
	for _, bad := range []string{"?status=applied", "?status=PENDING_REVIEW", "?subject_user_id=7"} {
		if r := call(t, srv, http.MethodGet, "/api/v1/hr/requests"+bad, "3", "", nil); r.code != 400 {
			t.Fatalf("%s: %d %s", bad, r.code, r.body)
		}
	}
	for _, actor := range []string{"5", ""} {
		if r := call(t, srv, http.MethodGet, "/api/v1/hr/requests?status=pending_review", actor, "", nil); r.code != 403 || r.json["items"] != nil {
			t.Fatalf("%q: %d %s", actor, r.code, r.body)
		}
	}
	// Flag off: BLOCKED with no items.
	e.setFlags(t, map[string]bool{"organization.hr_request_write": false, "organization.hr_request_writeback": false})
	if r := call(t, srv, http.MethodGet, "/api/v1/hr/requests?status=pending_review", "3", "", nil); r.json["outcome"] != "BLOCKED" || r.json["items"] != nil {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	if e.md5(t, hrRequestTables...) != tables {
		t.Fatal("a filtered read wrote an HR request table")
	}
	(&hrHarness{writesEnv: e, t: t}).noSideEffects(t, "rm-p04 filters", g)
}

func hrReadKeys() []identityapi.Capability {
	var out []identityapi.Capability
	for _, c := range identityapi.HRReadCapabilities() {
		out = append(out, c.Key)
	}
	return out
}
