package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// NT-17 (RM-E04B / A2, F-07): every /api response of the full handler chain
// is `Cache-Control: no-store` with `Vary: Cookie`, for every OpenAPI
// operation, success and failure, handler- and middleware-generated.

const (
	goodToken = "cache-test-good-session-token"
	badToken  = "cache-test-revoked-session-token"
	csrfToken = "cache-test-csrf"
	origin    = "https://crm.example.test"
)

var fixedTime = time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)

// cacheAuth is a synthetic session surface: one valid token, login sets a
// cookie, logout clears it. No business behaviour.
type cacheAuth struct{ unconfiguredAuth }

func (cacheAuth) LoginEnabled(context.Context) bool { return true }

func (cacheAuth) Authenticate(_ context.Context, token string) (websession.Principal, bool, error) {
	if token != goodToken {
		return websession.Principal{}, false, nil
	}
	return websession.Principal{SessionID: "s1", UserID: someID, DisplayName: "Synthetic", CSRF: csrfToken,
		IdleExpiresAt: fixedTime.Add(time.Hour), AbsoluteExpiresAt: fixedTime.Add(8 * time.Hour)}, true, nil
}

func (cacheAuth) LoginWithPassword(ctx context.Context, _ oapi.LoginWithPasswordRequestObject) (oapi.LoginWithPasswordResponseObject, error) {
	c := websession.Cookie(goodToken, fixedTime.Add(8*time.Hour), fixedTime).String()
	return oapi.LoginWithPassword200JSONResponse{Body: oapi.AuthEnvelope{Outcome: "SUCCESS", CorrelationId: httpx.CorrelationID(ctx)},
		Headers: oapi.LoginWithPassword200ResponseHeaders{SetCookie: &c}}, nil
}

func (cacheAuth) Logout(ctx context.Context, _ oapi.LogoutRequestObject) (oapi.LogoutResponseObject, error) {
	c := websession.ClearCookie().String()
	return oapi.Logout200JSONResponse{Body: oapi.AuthEnvelope{Outcome: "SUCCESS", CorrelationId: httpx.CorrelationID(ctx)},
		Headers: oapi.Logout200ResponseHeaders{SetCookie: &c}}, nil
}

// failingRead makes one read operation fail inside the handler (500) and
// one panic (recovered 500).
type failingRead struct{ unconfigured }

func (failingRead) GetEffectiveAccess(context.Context, oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error) {
	return nil, errors.New("synthetic infrastructure failure")
}

func (failingRead) ListIdentityUsers(context.Context, oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error) {
	panic("synthetic panic")
}

type cacheReq struct {
	method, path, body string
	headers            map[string]string
}

func (q cacheReq) do(h http.Handler) *http.Response {
	var req *http.Request
	if q.body != "" {
		req = httptest.NewRequest(q.method, q.path, strings.NewReader(q.body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(q.method, q.path, nil)
	}
	for k, v := range q.headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func authed(extra map[string]string) map[string]string {
	m := map[string]string{"Cookie": websession.CookieName + "=" + goodToken, websession.CSRFHeader: csrfToken, "Origin": origin, "Sec-Fetch-Site": "same-origin"}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func assertNoStore(t *testing.T, label string, res *http.Response) {
	t.Helper()
	if v := res.Header.Values("Cache-Control"); len(v) != 1 || v[0] != "no-store" {
		t.Fatalf("%s (%d): Cache-Control = %q", label, res.StatusCode, v)
	}
	if v := res.Header.Values("Pragma"); len(v) != 1 || v[0] != "no-cache" {
		t.Fatalf("%s (%d): Pragma = %q", label, res.StatusCode, v)
	}
	v := res.Header.Values("Vary")
	cookies := 0
	if len(v) == 1 {
		for _, tok := range strings.Split(v[0], ",") {
			if strings.EqualFold(strings.TrimSpace(tok), "Cookie") {
				cookies++
			}
		}
	}
	if len(v) != 1 || cookies != 1 {
		t.Fatalf("%s (%d): Vary = %q", label, res.StatusCode, v)
	}
	if !httpx.ValidCorrelationID(res.Header.Get(httpx.CorrelationHeader)) {
		t.Fatalf("%s (%d): correlation id lost", label, res.StatusCode)
	}
}

func cacheHandlers(t *testing.T) map[string]http.Handler {
	t.Helper()
	build := func(env authstub.Environment, devAuth bool, sf Surfaces, origins ...string) http.Handler {
		cfg := DefaultConfig()
		cfg.Environment, cfg.DevAuth, cfg.AllowedOrigins = env, devAuth, origins
		h, err := NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: fixedAt}, quiet, sf)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	return map[string]http.Handler{
		"production/anonymous": build(authstub.Production, false, Surfaces{}),
		"production/sessions":  build(authstub.Production, false, Surfaces{Auth: cacheAuth{}, Read: failingRead{}, Metrics: NewMetrics()}, origin),
		"development/dev-stub": build(authstub.Development, true, Surfaces{}),
	}
}

// operations returns every (method, template) of the bundled contract.
func operations(t *testing.T) [][2]string {
	t.Helper()
	var ops [][2]string
	for path, item := range loadContract(t).Paths.Map() {
		for method := range item.Operations() {
			ops = append(ops, [2]string{method, path})
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i][1]+ops[i][0] < ops[j][1]+ops[j][0] })
	return ops
}

// TestNT17EveryOperationIsNoStore drives all contract operations through
// every handler configuration with an anonymous, an authenticated, a
// revoked-cookie, an unknown-query and a wrong-method request.
func TestNT17EveryOperationIsNoStore(t *testing.T) {
	ops := operations(t)
	if len(ops) != 105 {
		t.Fatalf("contract has %d operations; NT-17 coverage was established for 105 — review the new operations' classification", len(ops))
	}
	statuses := map[int]int{}
	apiOps := 0
	for name, h := range cacheHandlers(t) {
		for _, op := range ops {
			method, path := op[0], concrete(op[1])
			body := ""
			if method != http.MethodGet {
				body = "{}"
			}
			if !httpx.APIPath(op[1]) {
				// /health/live and /health/ready: unauthenticated liveness
				// documents, outside the /api policy (see the health test).
				continue
			}
			if name == "production/anonymous" {
				apiOps++
			}
			wrong := http.MethodGet
			if method == http.MethodGet {
				wrong = http.MethodDelete
			}
			for label, q := range map[string]cacheReq{
				"anonymous":     {method, path, body, nil},
				"authenticated": {method, path, body, authed(map[string]string{authstub.Header: "dev-user-001"})},
				"revoked":       {method, path, body, map[string]string{"Cookie": websession.CookieName + "=" + badToken}},
				"unknown query": {method, path + "?zz_unknown=1", body, authed(nil)},
				"wrong method":  {wrong, path, "", authed(nil)},
				"csrf mismatch": {method, path, body, authed(map[string]string{websession.CSRFHeader: "wrong"})},
				"unknown child": {method, path + "/zz-unknown", body, authed(nil)},
			} {
				res := q.do(h)
				statuses[res.StatusCode]++
				assertNoStore(t, name+" "+label+" "+method+" "+op[1], res)
			}
		}
	}
	if apiOps != 103 {
		t.Fatalf("/api operations = %d, want 103 (105 minus 2 health)", apiOps)
	}
	for _, want := range []int{200, 400, 401, 403, 404, 405, 500} {
		if statuses[want] == 0 {
			t.Fatalf("status %d never exercised: %v", want, statuses)
		}
	}
	t.Logf("NT-17 statuses across all operations: %v", statuses)
}

// TestNT17SpecificPaths covers the NT-17 rows that need a particular path:
// session, login (Set-Cookie), logout (clear cookie), 500s, early refusals
// generated by middleware, and the NT-17 named reads.
func TestNT17SpecificPaths(t *testing.T) {
	hs := cacheHandlers(t)
	sess, dev := hs["production/sessions"], hs["development/dev-stub"]
	big := `{"login":"` + strings.Repeat("x", int(DefaultConfig().MaxBodyBytes)+1) + `"}`
	for _, tc := range []struct {
		label  string
		h      http.Handler
		q      cacheReq
		status int
		cookie string // expected Set-Cookie substring ("" = none)
	}{
		{"session anonymous", sess, cacheReq{"GET", "/api/v1/session", "", nil}, 200, ""},
		{"session authenticated", sess, cacheReq{"GET", "/api/v1/session", "", authed(nil)}, 200, ""},
		{"session revoked cookie cleared", sess, cacheReq{"GET", "/api/v1/session", "", map[string]string{"Cookie": websession.CookieName + "=" + badToken}}, 200, "Max-Age=0"},
		{"login sets cookie", sess, cacheReq{"POST", "/api/v1/auth/login", `{"login":"a","password":"b"}`, map[string]string{"Origin": origin}}, 200, goodToken},
		{"logout clears cookie", sess, cacheReq{"POST", "/api/v1/auth/logout", "", authed(nil)}, 200, "Max-Age=0"},
		{"csrf refused (middleware 403)", sess, cacheReq{"POST", "/api/v1/auth/logout", "", authed(map[string]string{websession.CSRFHeader: "wrong"})}, 403, ""},
		{"cross-site refused (middleware 403)", sess, cacheReq{"POST", "/api/v1/auth/login", `{"login":"a","password":"b"}`, map[string]string{"Sec-Fetch-Site": "cross-site"}}, 403, ""},
		{"foreign origin refused (middleware 403)", sess, cacheReq{"POST", "/api/v1/auth/login", `{"login":"a","password":"b"}`, map[string]string{"Origin": "https://evil.test"}}, 403, ""},
		{"handler error (500)", sess, cacheReq{"GET", "/api/v1/authorization/effective-access", "", authed(nil)}, 500, ""},
		{"handler panic (recovered 500)", sess, cacheReq{"GET", "/api/v1/identity/users", "", authed(nil)}, 500, ""},
		{"body too large (middleware 413)", sess, cacheReq{"POST", "/api/v1/auth/login", big, map[string]string{"Origin": origin}}, 413, ""},
		{"invalid JSON (400)", sess, cacheReq{"POST", "/api/v1/auth/login", `{`, map[string]string{"Origin": origin}}, 400, ""},
		{"unknown body field (middleware 400)", sess, cacheReq{"POST", "/api/v1/auth/login", `{"unexpected":1}`, map[string]string{"Origin": origin}}, 400, ""},
		{"invalid path id (400)", sess, cacheReq{"GET", "/api/v1/identity/users/not-a-uuid", "", authed(nil)}, 400, ""},
		{"unknown route (404)", sess, cacheReq{"GET", "/api/v1/ledger/entries", "", authed(nil)}, 404, ""},
		{"api root (404)", sess, cacheReq{"GET", "/api", "", nil}, 404, ""},
		{"wrong method (405)", sess, cacheReq{"PUT", "/api/v1/session", "", nil}, 405, ""},
		{"unknown dev identity (middleware 400)", dev, cacheReq{"GET", "/api/v1/session", "", map[string]string{authstub.Header: "nobody"}}, 400, ""},
		{"dev authenticated read", dev, cacheReq{"GET", "/api/v1/identity/users", "", map[string]string{authstub.Header: "dev-user-001"}}, 200, ""},
		{"commission me earnings", sess, cacheReq{"GET", "/api/v1/commission/me/earnings", "", authed(nil)}, 0, ""},
		{"payment batch", sess, cacheReq{"GET", concrete("/api/v1/payments/batches/{batch_id}"), "", authed(nil)}, 0, ""},
	} {
		res := tc.q.do(tc.h)
		if tc.status != 0 && res.StatusCode != tc.status {
			t.Fatalf("%s: status %d, want %d", tc.label, res.StatusCode, tc.status)
		}
		assertNoStore(t, tc.label, res)
		sc := strings.Join(res.Header.Values("Set-Cookie"), "|")
		if (tc.cookie == "") != (sc == "") || !strings.Contains(sc, tc.cookie) {
			t.Fatalf("%s: Set-Cookie = %q, want %q", tc.label, sc, tc.cookie)
		}
	}
}

// Health is not under /api: liveness/readiness carry no identity data and
// keep their existing (header-free) caching, except that a response that
// clears a stale session cookie is private wherever it is served.
func TestNT17HealthClassification(t *testing.T) {
	sess := cacheHandlers(t)["production/sessions"]
	for _, p := range []string{"/health/live", "/health/ready"} {
		res := cacheReq{"GET", p, "", nil}.do(sess)
		if res.StatusCode != 200 || len(res.Header.Values("Cache-Control")) != 0 || len(res.Header.Values("Vary")) != 0 {
			t.Fatalf("%s: %d %v", p, res.StatusCode, res.Header)
		}
		res = cacheReq{"GET", p, "", map[string]string{"Cookie": websession.CookieName + "=" + badToken}}.do(sess)
		if res.StatusCode != 200 || !slices.ContainsFunc(res.Header.Values("Set-Cookie"), func(s string) bool { return strings.Contains(s, "Max-Age=0") }) {
			t.Fatalf("%s with revoked cookie: %d %v", p, res.StatusCode, res.Header)
		}
		assertNoStore(t, p+" clearing cookie", res)
	}
}
