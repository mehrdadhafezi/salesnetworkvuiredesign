package biawin_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	provideropshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// R1 authentication × authorization against real PostgreSQL sessions.
// All accounts are SYNTHETIC.

const r1Origin = "https://crm.example.test"

// capabilityOverlay is TEST-ONLY: the session subject (identity proven by a
// real server-side session) plus synthetic capabilities, so the row-scope
// checks behind the capabilities can be exercised. No role bundle grants
// fulfillment capabilities and no production path builds this overlay.
type capabilityOverlay struct {
	inner authorization.SessionSubjects
	caps  map[string][]identityapi.Capability
}

func (o capabilityOverlay) Subject(ctx context.Context) authorization.Subject {
	s := o.inner.Subject(ctx)
	if c := o.caps[s.UserID]; s.UserID != "" && len(c) > 0 {
		s.Grants = append(s.Grants, identityapi.Grant{UserID: s.UserID, LegacyValue: "synthetic", Status: identityapi.GrantActive,
			Basis: authorization.DevGrantBasis, Capabilities: c})
	}
	return s
}

func (e *env) r1Account(t *testing.T, login, pw string) string {
	t.Helper()
	ctx := context.Background()
	uid, _ := ids.NewUUID()
	batch, _ := ids.NewUUID()
	sha := sha256.Sum256([]byte(uid))
	if _, err := e.admin(t).Exec(ctx, `INSERT INTO identity.user_account (user_id, source_system, source_spec, source_key, first_batch_id, source_batch_id,
		source_row_sha256, imported_at, refreshed_at, presence, display_name, account_state, login_enabled, version)
		VALUES ($1, 'synthetic', 'synthetic.account.v1', $2, $3, $3, $4, now(), now(), 'PRESENT', $5, 'SHADOW_IMPORTED', false, 1)`,
		uid, fmt.Sprint(3000+len(login)*7+int(sha[0])), batch, sha[:], login); err != nil {
		t.Fatal(err)
	}
	h, err := password.Hash(pw, password.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := ids.NewUUID()
	if _, err := e.admin(t).Exec(ctx, `INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE', now(), now(), 1)`, cid, uid, login, h); err != nil {
		t.Fatal(err)
	}
	return uid
}

func (e *env) r1Server(t *testing.T, f *opsFlags, subs authorization.SubjectResolver, auth apiserver.AuthAPI) http.Handler {
	t.Helper()
	guard := func(b *atomic.Bool) *authorization.Guard {
		g, err := authorization.NewGuard(func(context.Context) (bool, error) { return b.Load(), nil }, subs)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	mode, err := rollout.FromFlags("fulfillment.work", func(_ context.Context, key string) (bool, error) {
		switch key {
		case rollout.WriteFlag("fulfillment.work"):
			return f.write.Load(), nil
		case rollout.WritebackFlag("fulfillment.work"):
			return f.writeback.Load(), nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	read := guard(&f.read)
	surface := struct {
		*fulfillmenthttp.WorkHandlers
		*biawinhttp.ActivationHandlers
		*provideropshttp.Handlers
	}{fulfillmenthttp.NewWork(read, guard(&f.write), mode, e.work, e.cat), biawinhttp.NewActivation(read, e.act),
		provideropshttp.New(read, guard(&f.reconcile), e.ops)}
	cfg := apiserver.DefaultConfig() // production
	cfg.AllowedOrigins = []string{r1Origin}
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clock, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{Operations: surface, Auth: auth})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func r1Do(t *testing.T, h http.Handler, method, target, cookie string, body any, headers map[string]string) opsResp {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	q := httptest.NewRequest(method, target, rd)
	if body != nil {
		q.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		q.AddCookie(&http.Cookie{Name: websession.CookieName, Value: cookie})
	}
	if method != http.MethodGet {
		q.Header.Set("Origin", r1Origin)
	}
	for k, v := range headers {
		q.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, q)
	r := opsResp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	if method == http.MethodPost && strings.HasSuffix(target, "/auth/login") && rec.Code == 200 {
		for _, c := range rec.Result().Cookies() {
			if c.Name == websession.CookieName {
				r.body = c.Value // hand the token to the test only
			}
		}
	}
	return r
}

// 10 / 11 / 12 / Step 3: authentication establishes identity, never
// permission; row scope keys on the session-proven user id.
func TestSessionAuthorizationBoundaries(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, identitymigrations.Source()); err != nil {
		t.Fatal(err)
	}
	a := e.r1Account(t, "seller.alpha", "synthetic-Pass-A")
	b := e.r1Account(t, "seller.bravo", "synthetic-Pass-B")
	idRunner := e.Runner(t, identity.Module)
	var authOn atomic.Bool
	authOn.Store(true)
	svc, err := session.New(idRunner, e.clock, session.DefaultConfig, func(context.Context) (bool, error) { return authOn.Load(), nil })
	if err != nil {
		t.Fatal(err)
	}
	auth, err := identityhttp.NewAuth(svc, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	prodSubjects := authorization.SessionSubjects{Grants: identityquery.New(idRunner)}

	// Two services, each owned by a different seller.
	e.seed(t, "prod-r1a", 1, map[string]string{"prod-r1a.svc-01": a}, map[string]string{"prod-r1a.svc-01": a}, nil)
	e.seed(t, "prod-r1b", 1, map[string]string{"prod-r1b.svc-01": b}, map[string]string{"prod-r1b.svc-01": b}, nil)
	for _, p := range []string{"prod-r1a", "prod-r1b"} {
		cust := uid(t)
		e.activate(t, paid(t, p, biawinapi.RouteAdditionalServices, "RESOLVED", &cust))
	}
	e.relay(t)
	var caseA, caseB string
	_ = e.admin(t).QueryRow(ctx, `SELECT case_id::text FROM fulfillment.work_case WHERE service_key = 'prod-r1a.svc-01'`).Scan(&caseA)
	_ = e.admin(t).QueryRow(ctx, `SELECT case_id::text FROM fulfillment.work_case WHERE service_key = 'prod-r1b.svc-01'`).Scan(&caseB)
	history := func() string {
		var s string
		_ = e.admin(t).QueryRow(ctx, `SELECT coalesce(string_agg(event_id::text || ':' || coalesce(actor_user_id::text, '-'), ',' ORDER BY event_id), '')
			FROM fulfillment.work_event`).Scan(&s)
		return s
	}

	f := &opsFlags{}
	f.read.Store(true)
	prod := e.r1Server(t, f, prodSubjects, auth)
	tokA := r1Do(t, prod, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"login": "seller.alpha", "password": "synthetic-Pass-A"}, nil).body
	if len(tokA) != 43 {
		t.Fatalf("login A: %q", tokA)
	}

	// Production resolver: a real, authenticated seller has NO authority
	// (no ACTIVE grant exists; legacy role evidence is SHADOW only).
	if r := r1Do(t, prod, http.MethodGet, "/api/v1/fulfillment/work-cases", tokA, nil, nil); r.code != 403 || r.json["outcome"] != "UNAUTHORIZED" {
		t.Fatalf("authenticated without grants: %d %s", r.code, r.body)
	}
	// Anonymous is unauthorized too.
	if r := r1Do(t, prod, http.MethodGet, "/api/v1/fulfillment/work-cases", "", nil, nil); r.code != 403 {
		t.Fatalf("anonymous: %d %s", r.code, r.body)
	}

	// Test-only overlay: identical session identity, synthetic capabilities.
	overlay := capabilityOverlay{inner: prodSubjects, caps: map[string][]identityapi.Capability{
		a: {identityapi.CapWorkRead, identityapi.CapWorkAssign, identityapi.CapWorkExecute},
		b: {identityapi.CapWorkRead, identityapi.CapWorkAssign, identityapi.CapWorkExecute},
	}}
	h := e.r1Server(t, f, overlay, auth)
	tokB := r1Do(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"login": "seller.bravo", "password": "synthetic-Pass-B"}, nil).body

	// 11: seller A sees only their service; B's case is not found for A.
	r := r1Do(t, h, http.MethodGet, "/api/v1/fulfillment/work-cases", tokA, nil, nil)
	if r.code != 200 || !strings.Contains(r.body, caseA) || strings.Contains(r.body, caseB) {
		t.Fatalf("A's queue: %d %s", r.code, r.body)
	}
	if r := r1Do(t, h, http.MethodGet, "/api/v1/fulfillment/work-cases/"+caseB, tokA, nil, nil); r.code != 404 {
		t.Fatalf("A reading B's case: %d %s", r.code, r.body)
	}
	if r := r1Do(t, h, http.MethodGet, "/api/v1/fulfillment/work-cases/"+caseB, tokB, nil, nil); r.code != 200 {
		t.Fatalf("B reading own case: %d %s", r.code, r.body)
	}
	// Supervisor authority needs real per-service scope: A holds the assign
	// capability but supervises only svc A, so assigning B's case fails.
	f.write.Store(true)
	f.writeback.Store(true)
	assign := map[string]any{"assignee_user_id": a}
	csrfA := websession.CSRFToken(tokA)
	before := history()
	r = r1Do(t, h, http.MethodPost, "/api/v1/fulfillment/work-cases/"+caseB+"/assignment", tokA, assign,
		map[string]string{"Idempotency-Key": "r1-assign-0001", "If-Match": `"1"`, websession.CSRFHeader: csrfA})
	if r.code == 200 && r.json["outcome"] == "SUCCESS" {
		t.Fatalf("A assigned B's case: %s", r.body)
	}
	// The command needs the CSRF token; without it nothing runs.
	if r := r1Do(t, h, http.MethodPost, "/api/v1/fulfillment/work-cases/"+caseA+"/assignment", tokA, assign,
		map[string]string{"Idempotency-Key": "r1-assign-0002", "If-Match": `"1"`}); r.code != 403 || !strings.Contains(r.body, "cross-site-request-rejected") {
		t.Fatalf("command without CSRF: %d %s", r.code, r.body)
	}
	if history() != before {
		t.Fatal("a refused command changed history")
	}
	// With CSRF, A's own case: allowed, and the event actor is A's identity.
	r = r1Do(t, h, http.MethodPost, "/api/v1/fulfillment/work-cases/"+caseA+"/assignment", tokA, assign,
		map[string]string{"Idempotency-Key": "r1-assign-0003", "If-Match": `"1"`, websession.CSRFHeader: csrfA})
	if r.code != 200 || r.json["outcome"] != "SUCCESS" {
		t.Fatalf("A assigning own case: %d %s", r.code, r.body)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1::uuid AND kind = 'ASSIGNED' AND actor_user_id = $2::uuid`, caseA, a) != 1 {
		t.Fatal("event actor is not the session user")
	}

	// 12: a disabled surface flag stays effective for an authenticated user.
	f.read.Store(false)
	if r := r1Do(t, h, http.MethodGet, "/api/v1/fulfillment/work-cases", tokA, nil, nil); r.json["outcome"] != "BLOCKED" {
		t.Fatalf("flag off with a session: %s", r.body)
	}
	f.read.Store(true)

	// Identity changes never rewrite historical attribution: revoking every
	// session and disabling the credential leaves history untouched, and
	// the old token stops working.
	after := history()
	csrfB := websession.CSRFToken(tokB)
	if r := r1Do(t, h, http.MethodPost, "/api/v1/auth/logout-all", tokB, nil, map[string]string{websession.CSRFHeader: csrfB}); r.code != 200 {
		t.Fatalf("logout-all: %d %s", r.code, r.body)
	}
	_, _ = e.admin(t).Exec(ctx, `UPDATE identity.credential SET status = 'DISABLED' WHERE user_id = $1`, a)
	if r := r1Do(t, h, http.MethodGet, "/api/v1/fulfillment/work-cases", tokA, nil, nil); r.code != 403 {
		t.Fatalf("disabled credential still authorised: %d %s", r.code, r.body)
	}
	if history() != after {
		t.Fatal("identity changes rewrote work history")
	}
	// Sessions touch only identity.session / identity.auth_event.
	if e.count(t, `SELECT count(*) FROM information_schema.role_table_grants WHERE grantee = 'crm_owner_identity' AND table_schema <> 'identity'`) != 0 {
		t.Fatal("identity owner holds a grant outside its schema")
	}
}
