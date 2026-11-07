package http_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	platformmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// All accounts, passwords and origins are SYNTHETIC.

const origin = "https://crm.example.test"

type clock struct {
	mu sync.Mutex
	at time.Time
}

func (c *clock) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *clock) advance(d time.Duration) { c.mu.Lock(); c.at = c.at.Add(d); c.mu.Unlock() }

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}
func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

type env struct {
	*pgtest.Env
	conn *pgx.Conn
	clk  *clock
	flag atomic.Bool
	logs *lockedBuf
	h    http.Handler
}

var fast = password.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}
var seq atomic.Int64

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{Env: pgtest.New(t), clk: &clock{at: time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)}, logs: &lockedBuf{}}
	if _, err := platformmigrations.Up(context.Background(), e.DSN, migrations.Source()); err != nil {
		t.Fatal(err)
	}
	c, err := pgx.Connect(context.Background(), e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	e.conn = c
	e.flag.Store(true)
	svc, err := session.New(e.Runner(t, identity.Module), e.clk, session.DefaultConfig, func(context.Context) (bool, error) { return e.flag.Load(), nil })
	if err != nil {
		t.Fatal(err)
	}
	auth, err := identityhttp.NewAuth(svc, e.clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig() // production
	cfg.AllowedOrigins = []string{origin}
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clk, slog.New(slog.NewJSONHandler(e.logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		apiserver.Surfaces{Auth: auth})
	if err != nil {
		t.Fatal(err)
	}
	e.h = h
	return e
}

func (e *env) account(t *testing.T, login, pw string) string {
	t.Helper()
	ctx := context.Background()
	uid, _ := ids.NewUUID()
	batch, _ := ids.NewUUID()
	sha := sha256.Sum256([]byte(uid))
	n := seq.Add(1)
	if _, err := e.conn.Exec(ctx, `INSERT INTO identity.user_account (user_id, source_system, source_spec, source_key, first_batch_id, source_batch_id,
		source_row_sha256, imported_at, refreshed_at, presence, display_name, account_state, login_enabled, version)
		VALUES ($1, 'synthetic', 'synthetic.account.v1', $2, $3, $3, $4, now(), now(), 'PRESENT', $5, 'SHADOW_IMPORTED', false, 1)`,
		uid, fmt.Sprint(2000+n), batch, sha[:], "کاربر آزمایشی"); err != nil {
		t.Fatal(err)
	}
	hash, err := password.Hash(pw, fast)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := ids.NewUUID()
	if _, err := e.conn.Exec(ctx, `INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE', now(), now(), 1)`, cid, uid, login, hash); err != nil {
		t.Fatal(err)
	}
	return uid
}

type resp struct {
	code    int
	body    string
	json    map[string]any
	cookies []*http.Cookie
	setRaw  []string
}

type req struct {
	method, path string
	body         any
	cookie       string
	csrf         string
	origin       string
	fetchSite    string
	headers      map[string]string
}

func (e *env) do(t *testing.T, r req) resp {
	t.Helper()
	var rd io.Reader
	if r.body != nil {
		b, _ := json.Marshal(r.body)
		rd = bytes.NewReader(b)
	}
	q := httptest.NewRequest(r.method, r.path, rd)
	q.Host = "crm.example.test"
	if r.body != nil {
		q.Header.Set("Content-Type", "application/json")
	}
	if r.cookie != "" {
		q.AddCookie(&http.Cookie{Name: websession.CookieName, Value: r.cookie})
	}
	if r.csrf != "" {
		q.Header.Set(websession.CSRFHeader, r.csrf)
	}
	if r.origin != "" {
		q.Header.Set("Origin", r.origin)
	}
	if r.fetchSite != "" {
		q.Header.Set("Sec-Fetch-Site", r.fetchSite)
	}
	for k, v := range r.headers {
		q.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, q)
	out := resp{code: rec.Code, body: rec.Body.String(), cookies: rec.Result().Cookies(), setRaw: rec.Result().Header.Values("Set-Cookie")}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.json)
	return out
}

func (e *env) login(t *testing.T, login, pw string, extra ...func(*req)) resp {
	t.Helper()
	r := req{method: http.MethodPost, path: "/api/v1/auth/login", body: map[string]string{"login": login, "password": pw}, origin: origin, fetchSite: "same-origin"}
	for _, f := range extra {
		f(&r)
	}
	return e.do(t, r)
}

func sessionCookie(t *testing.T, r resp) *http.Cookie {
	t.Helper()
	for _, c := range r.cookies {
		if c.Name == websession.CookieName {
			return c
		}
	}
	t.Fatalf("no session cookie in %v", r.setRaw)
	return nil
}

func (e *env) session(t *testing.T, cookie string) map[string]any {
	t.Helper()
	r := e.do(t, req{method: http.MethodGet, path: "/api/v1/session", cookie: cookie})
	data, _ := r.json["data"].(map[string]any)
	return data
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.conn.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// 1 / 14 / 18: login over HTTP, cookie security, session restore, no secrets.
func TestHTTPLoginCookieAndSession(t *testing.T) {
	e := setup(t)
	uid := e.account(t, "seller.http", "synthetic-Pass-1")
	if s := e.session(t, ""); s["auth_mode"] != "session" || s["authenticated"] != false || s["session_state"] != "none" || s["login_enabled"] != true {
		t.Fatalf("anonymous session: %v", s)
	}
	r := e.login(t, "seller.http", "synthetic-Pass-1")
	if r.code != 200 || r.json["outcome"] != "SUCCESS" {
		t.Fatalf("login: %d %s", r.code, r.body)
	}
	c := sessionCookie(t, r)
	raw := strings.Join(r.setRaw, "\n")
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" || c.MaxAge <= 0 ||
		!strings.HasPrefix(raw, "__Host-crm_session=") {
		t.Fatalf("cookie attributes: %+v / %s", c, raw)
	}
	if strings.Contains(r.body, c.Value) {
		t.Fatal("the session token must never appear in a response body")
	}
	sess, _ := r.json["session"].(map[string]any)
	if sess["user_id"] != uid || sess["csrf_token"] != websession.CSRFToken(c.Value) {
		t.Fatalf("login body: %v", sess)
	}
	// 14: browser refresh → the session restores from the cookie alone.
	s := e.session(t, c.Value)
	inner, _ := s["session"].(map[string]any)
	if s["authenticated"] != true || s["session_state"] != "active" || inner["user_id"] != uid || inner["csrf_token"] != sess["csrf_token"] {
		t.Fatalf("restored session: %v", s)
	}
	// 18: no password, token, CSRF token or raw login in the logs.
	logs := e.logs.String()
	for _, secret := range []string{"synthetic-Pass-1", c.Value, websession.CSRFToken(c.Value), "seller.http"} {
		if strings.Contains(logs, secret) {
			t.Fatalf("secret %q in logs", secret)
		}
	}
}

// 2 / 9 / 8 / 12: uniform failures, throttle, flag OFF.
func TestHTTPLoginFailures(t *testing.T) {
	e := setup(t)
	e.account(t, "seller.fail", "synthetic-Pass-1")
	wrong := e.login(t, "seller.fail", "nope")
	unknown := e.login(t, "nobody.fail", "nope")
	for _, r := range []resp{wrong, unknown} {
		if r.code != 401 || r.json["outcome"] != "UNAUTHORIZED" || len(r.cookies) != 0 {
			t.Fatalf("failure: %d %s", r.code, r.body)
		}
	}
	strip := func(s string) string { return strings.Split(s, `"correlation_id"`)[0] }
	if strip(wrong.body) != strip(unknown.body) {
		t.Fatalf("wrong password and unknown login must be indistinguishable:\n%s\n%s", wrong.body, unknown.body)
	}
	if r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/login", body: map[string]any{"login": "a", "password": "b", "remember": true}, origin: origin}); r.code != 400 {
		t.Fatalf("unknown body field: %d", r.code)
	}
	for i := 0; i < session.DefaultConfig.MaxFailures-1; i++ {
		e.login(t, "seller.fail", "nope")
	}
	if r := e.login(t, "seller.fail", "synthetic-Pass-1"); r.code != 429 || !strings.Contains(r.body, "too-many-attempts") || len(r.cookies) != 0 {
		t.Fatalf("throttle: %d %s", r.code, r.body)
	}
	e.flag.Store(false)
	if r := e.login(t, "seller.fail", "synthetic-Pass-1"); r.code != 200 || r.json["outcome"] != "BLOCKED" || len(r.cookies) != 0 {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	if s := e.session(t, ""); s["login_enabled"] != false {
		t.Fatalf("login_enabled with flag off: %v", s)
	}
}

// 7: CSRF and cross-site checks.
func TestHTTPCSRF(t *testing.T) {
	e := setup(t)
	e.account(t, "seller.csrf", "synthetic-Pass-1")
	// Login from a foreign origin or a cross-site context is refused before any check.
	before := e.count(t, `SELECT count(*) FROM identity.auth_event`)
	for _, mut := range []func(*req){
		func(r *req) { r.origin = "https://evil.example.test" },
		func(r *req) { r.fetchSite = "cross-site" },
		func(r *req) { r.origin = "null" },
	} {
		if r := e.login(t, "seller.csrf", "synthetic-Pass-1", mut); r.code != 403 || !strings.Contains(r.body, "cross-site-request-rejected") || len(r.cookies) != 0 {
			t.Fatalf("cross-site login: %d %s", r.code, r.body)
		}
	}
	if e.count(t, `SELECT count(*) FROM identity.auth_event`) != before {
		t.Fatal("a rejected cross-site login must not even be evaluated")
	}
	c := sessionCookie(t, e.login(t, "seller.csrf", "synthetic-Pass-1"))
	csrf := websession.CSRFToken(c.Value)
	// Missing, wrong and another session's token are all refused; the session survives.
	other := sessionCookie(t, e.login(t, "seller.csrf", "synthetic-Pass-1")).Value
	// A login on another device (no cookie held) does not touch the first session.
	if e.session(t, c.Value)["authenticated"] != true {
		t.Fatal("an independent login revoked another session")
	}
	for _, bad := range []string{"", "x", websession.CSRFToken(other), csrf + "A"} {
		r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/logout", cookie: c.Value, csrf: bad, origin: origin})
		if r.code != 403 {
			t.Fatalf("csrf %q: %d %s", bad, r.code, r.body)
		}
	}
	if e.session(t, c.Value)["authenticated"] != true {
		t.Fatal("a refused logout must not revoke the session")
	}
	if e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'CSRF_REJECTED'`) != 4 {
		t.Fatal("CSRF rejections must be recorded")
	}
	// Right token, same origin → logout succeeds, cookie cleared, session revoked.
	r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/logout", cookie: c.Value, csrf: csrf, origin: origin})
	if r.code != 200 || r.json["outcome"] != "SUCCESS" {
		t.Fatalf("logout: %d %s", r.code, r.body)
	}
	if cleared := sessionCookie(t, r); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("cookie not cleared: %+v", cleared)
	}
	if s := e.session(t, c.Value); s["authenticated"] != false || s["session_state"] != "invalid" {
		t.Fatalf("after logout: %v", s)
	}
}

// 3 / 4 / 5 / 6 / 15: expiry on refresh, revocation, malformed tokens, fixation.
func TestHTTPSessionStates(t *testing.T) {
	e := setup(t)
	e.account(t, "seller.state", "synthetic-Pass-1")
	// 5: malformed / forged cookies are invalid and cleared.
	for _, bad := range []string{"garbage", strings.Repeat("A", 43), "<script>"} {
		r := e.do(t, req{method: http.MethodGet, path: "/api/v1/session", cookie: bad})
		data, _ := r.json["data"].(map[string]any)
		if data["session_state"] != "invalid" || data["authenticated"] != false || len(r.cookies) == 0 || r.cookies[0].MaxAge >= 0 {
			t.Fatalf("forged cookie %q: %s %v", bad, r.body, r.setRaw)
		}
	}
	// 6: fixation. An attacker-chosen cookie presented at login is not
	// adopted: login mints a different token.
	chosen, _ := websession.NewToken()
	r := e.login(t, "seller.state", "synthetic-Pass-1", func(q *req) { q.cookie = chosen })
	c := sessionCookie(t, r)
	if c.Value == chosen || e.session(t, chosen)["authenticated"] != false {
		t.Fatal("client-chosen session token adopted")
	}
	// Logging in again while holding a valid session retires it.
	r2 := e.login(t, "seller.state", "synthetic-Pass-1", func(q *req) { q.cookie = c.Value })
	c2 := sessionCookie(t, r2)
	if c2.Value == c.Value || e.session(t, c.Value)["authenticated"] != false || e.session(t, c2.Value)["authenticated"] != true {
		t.Fatal("re-login must replace the held session")
	}
	// 15: refresh after expiry → invalid, cookie cleared.
	e.clk.advance(session.DefaultConfig.IdleTimeout + time.Minute)
	if s := e.session(t, c2.Value); s["authenticated"] != false || s["session_state"] != "invalid" {
		t.Fatalf("expired session: %v", s)
	}
	// 4: revocation through logout-all reaches every device.
	d1 := sessionCookie(t, e.login(t, "seller.state", "synthetic-Pass-1")).Value
	d2 := sessionCookie(t, e.login(t, "seller.state", "synthetic-Pass-1")).Value
	if r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/logout-all", cookie: d1, csrf: websession.CSRFToken(d1), origin: origin}); r.code != 200 {
		t.Fatalf("logout-all: %d %s", r.code, r.body)
	}
	if e.session(t, d2)["authenticated"] != false {
		t.Fatal("logout-all left another device signed in")
	}
	if r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/logout-all", origin: origin}); r.code != 401 {
		t.Fatalf("logout-all without a session: %d", r.code)
	}
}

// 13: the development stub can never act in production, and it can never
// be combined with real sessions.
func TestHTTPNoDevBypassInProduction(t *testing.T) {
	e := setup(t)
	r := e.do(t, req{method: http.MethodGet, path: "/api/v1/session", headers: map[string]string{authstub.Header: "dev-user-001"}})
	data, _ := r.json["data"].(map[string]any)
	if data["authenticated"] != false || data["auth_mode"] != "session" || data["principal"] != nil {
		t.Fatalf("dev header honoured in production: %s", r.body)
	}
	cfg := apiserver.DefaultConfig()
	cfg.DevAuth = true
	if _, err := apiserver.NewHandlerWithSurfaces(cfg, e.clk, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{}); err == nil {
		t.Fatal("dev auth accepted in production")
	}
	cfg.Environment = authstub.Test
	auth := struct{ apiserver.AuthAPI }{}
	if _, err := apiserver.NewHandlerWithSurfaces(cfg, e.clk, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{Auth: auth}); !errors.Is(err, apiserver.ErrAuthConflict) {
		t.Fatalf("dev stub + sessions must conflict: %v", err)
	}
	prod := apiserver.DefaultConfig()
	if _, err := apiserver.NewHandlerWithSurfaces(prod, e.clk, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{Auth: auth}); err == nil {
		t.Fatal("sessions in production without allowed origins must be refused")
	}
	if _, err := apiserver.ParseAllowedOrigins("http://crm.example.test", authstub.Production); err == nil {
		t.Fatal("plain-http origin accepted in production")
	}
	if _, err := apiserver.ParseAllowedOrigins("https://crm.example.test/path", authstub.Production); err == nil {
		t.Fatal("origin with a path accepted")
	}
}

// 16: concurrent logouts and requests on one session over HTTP.
func TestHTTPConcurrentLogout(t *testing.T) {
	e := setup(t)
	e.account(t, "seller.race", "synthetic-Pass-1")
	c := sessionCookie(t, e.login(t, "seller.race", "synthetic-Pass-1")).Value
	csrf := websession.CSRFToken(c)
	var wg sync.WaitGroup
	var ok200 atomic.Int64
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if r := e.do(t, req{method: http.MethodPost, path: "/api/v1/auth/logout", cookie: c, csrf: csrf, origin: origin}); r.code == 200 {
				ok200.Add(1)
			}
		}()
		go func() { defer wg.Done(); e.session(t, c) }()
	}
	wg.Wait()
	if e.count(t, `SELECT count(*) FROM identity.auth_event WHERE kind = 'LOGOUT'`) != 1 || e.session(t, c)["authenticated"] != false {
		t.Fatal("concurrent logout must revoke exactly once")
	}
	if ok200.Load() == 0 {
		t.Fatal("no logout succeeded")
	}
}
