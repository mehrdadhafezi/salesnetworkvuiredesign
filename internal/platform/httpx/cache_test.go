package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// NT-17 (RM-E04B / A2) unit level: the cache policy itself.

func serve(h http.Handler, method, path string) *http.Response {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec.Result()
}

// assertPrivate: exactly one Cache-Control (no-store), one Pragma
// (no-cache) and one Vary field with Cookie exactly once.
func assertPrivate(t *testing.T, label string, res *http.Response) {
	t.Helper()
	if v := res.Header.Values("Cache-Control"); len(v) != 1 || v[0] != "no-store" {
		t.Fatalf("%s: Cache-Control = %q", label, v)
	}
	if v := res.Header.Values("Pragma"); len(v) != 1 || v[0] != "no-cache" {
		t.Fatalf("%s: Pragma = %q", label, v)
	}
	v := res.Header.Values("Vary")
	if len(v) != 1 {
		t.Fatalf("%s: Vary fields = %q", label, v)
	}
	n := 0
	for _, tok := range strings.Split(v[0], ",") {
		if strings.EqualFold(strings.TrimSpace(tok), "Cookie") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%s: Vary = %q (Cookie %d times)", label, v[0], n)
	}
}

func TestAPIPath(t *testing.T) {
	for p, want := range map[string]bool{
		"/api": true, "/api/": true, "/api/v1/session": true, "/api/v1/auth/login": true,
		"/apix": false, "/apiv1/session": false, "/": false, "/index.html": false,
		"/assets/app.js": false, "/health/live": false, "/health/ready": false, "/metrics": false,
	} {
		if APIPath(p) != want {
			t.Fatalf("APIPath(%q) = %v", p, !want)
		}
	}
}

func TestCachePolicyAllStatusesAndBodiesUnchanged(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 301, 302, 304, 400, 401, 403, 404, 405, 409, 413, 429, 500, 503} {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set(CorrelationHeader, "corr-0000-0001")
			w.Header().Set("Strict-Transport-Security", "max-age=1")
			if code >= 300 && code < 400 && code != 304 {
				w.Header().Set("Location", "/api/v1/session")
			}
			w.WriteHeader(code)
			if code != 204 && code != 304 {
				_, _ = io.WriteString(w, `{"k":"v"}`)
			}
		})
		res := serve(CachePolicy(inner), http.MethodGet, "/api/v1/x")
		label := http.StatusText(code)
		assertPrivate(t, label, res)
		body, _ := io.ReadAll(res.Body)
		wantBody := `{"k":"v"}`
		if code == 204 || code == 304 {
			wantBody = ""
		}
		if res.StatusCode != code || string(body) != wantBody {
			t.Fatalf("%s: status/body changed: %d %q", label, res.StatusCode, body)
		}
		if res.Header.Get(CorrelationHeader) != "corr-0000-0001" || res.Header.Get("Strict-Transport-Security") != "max-age=1" || res.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("%s: unrelated headers changed: %v", label, res.Header)
		}
		if code >= 300 && code < 400 && code != 304 && res.Header.Get("Location") != "/api/v1/session" {
			t.Fatalf("%s: Location lost", label)
		}
	}
}

func TestCachePolicyImplicit200AndProblem(t *testing.T) {
	// Write without WriteHeader (implicit 200) and the problem writer.
	res := serve(CachePolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("x")) })), http.MethodGet, "/api/v1/x")
	assertPrivate(t, "implicit 200", res)
	res = serve(CachePolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { WriteProblem(w, r, ProblemNotFound) })), http.MethodGet, "/api/v1/x")
	assertPrivate(t, "problem", res)
	// A handler that writes nothing at all.
	res = serve(CachePolicy(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "/api/v1/x")
	assertPrivate(t, "empty handler", res)
}

func TestCachePolicyOverridesConflictingCacheControl(t *testing.T) {
	for name, set := range map[string]func(h http.Header){
		"public max-age":  func(h http.Header) { h.Set("Cache-Control", "public, max-age=600") },
		"duplicate lines": func(h http.Header) { h.Add("Cache-Control", "no-store"); h.Add("Cache-Control", "max-age=60") },
		"private":         func(h http.Header) { h.Set("Cache-Control", "private"); h.Set("Pragma", "x") },
		"no-store twice":  func(h http.Header) { h.Add("Cache-Control", "no-store"); h.Add("Cache-Control", "no-store") },
	} {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			set(w.Header())
			w.WriteHeader(200)
		})
		assertPrivate(t, name, serve(CachePolicy(inner), http.MethodGet, "/api/v1/x"))
	}
}

func TestCachePolicyMergesVary(t *testing.T) {
	for name, tc := range map[string]struct {
		pre  []string
		want string
	}{
		"none":              {nil, "Cookie"},
		"accept-encoding":   {[]string{"Accept-Encoding"}, "Accept-Encoding, Cookie"},
		"already cookie":    {[]string{"cookie"}, "cookie"},
		"multi lines + dup": {[]string{"Origin, Accept-Encoding", "accept-encoding", "Cookie, Origin"}, "Origin, Accept-Encoding, Cookie"},
		"empty tokens":      {[]string{" , Origin ,, "}, "Origin, Cookie"},
		"star":              {[]string{"Origin", "*"}, "*"},
	} {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, v := range tc.pre {
				w.Header().Add("Vary", v)
			}
			w.WriteHeader(200)
		})
		res := serve(CachePolicy(inner), http.MethodGet, "/api/v1/x")
		if v := res.Header.Values("Vary"); len(v) != 1 || v[0] != tc.want {
			t.Fatalf("%s: Vary = %q, want %q", name, v, tc.want)
		}
		if name != "star" {
			assertPrivate(t, name, res)
		}
	}
}

func TestCachePolicyIdempotent(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Set-Cookie", "a=b")
		w.WriteHeader(200)
	})
	once := serve(CachePolicy(inner), http.MethodGet, "/api/v1/x")
	thrice := serve(CachePolicy(CachePolicy(CachePolicy(inner))), http.MethodGet, "/api/v1/x")
	assertPrivate(t, "thrice", thrice)
	for _, k := range []string{"Cache-Control", "Pragma", "Vary", "Set-Cookie"} {
		if strings.Join(once.Header.Values(k), "|") != strings.Join(thrice.Header.Values(k), "|") {
			t.Fatalf("%s differs: %q vs %q", k, once.Header.Values(k), thrice.Header.Values(k))
		}
	}
	if v := thrice.Header.Values("Vary"); v[0] != "Origin, Cookie" {
		t.Fatalf("Vary = %q", v)
	}
}

func TestCachePolicyLeavesNonAPIAlone(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets/app.js" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("Vary", "Accept-Encoding")
		}
		w.WriteHeader(200)
	})
	h := CachePolicy(inner)
	res := serve(h, http.MethodGet, "/assets/app.js")
	if res.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" || strings.Join(res.Header.Values("Vary"), ",") != "Accept-Encoding" || res.Header.Get("Pragma") != "" {
		t.Fatalf("static asset caching altered: %v", res.Header)
	}
	for _, p := range []string{"/", "/index.html", "/health/live", "/apix"} {
		res := serve(h, http.MethodGet, p)
		if len(res.Header.Values("Cache-Control")) != 0 || len(res.Header.Values("Vary")) != 0 || len(res.Header.Values("Pragma")) != 0 {
			t.Fatalf("%s: non-API response altered: %v", p, res.Header)
		}
	}
}

func TestCachePolicySetCookieAnywhereIsPrivate(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		http.SetCookie(w, &http.Cookie{Name: "__Host-crm_session", Value: "", Path: "/", MaxAge: -1})
		w.WriteHeader(200)
	})
	res := serve(CachePolicy(inner), http.MethodGet, "/health/live")
	assertPrivate(t, "set-cookie on non-API", res)
	if len(res.Header.Values("Set-Cookie")) != 1 {
		t.Fatalf("Set-Cookie changed: %q", res.Header.Values("Set-Cookie"))
	}
}

func TestCachePolicyInformationalThenFinal(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.Header().Set("Cache-Control", "max-age=60") // set after the 1xx
		w.WriteHeader(200)
	})
	assertPrivate(t, "after 103", serve(CachePolicy(inner), http.MethodGet, "/api/v1/x"))
}

func TestCachePolicyCoversRecoverAndCorrelation(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	panics := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		panic("boom")
	})
	h := CachePolicy(Correlation(Recover(quiet)(panics)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
	req.Header.Set(CorrelationHeader, "corr-0000-0002")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	if res.StatusCode != 500 || res.Header.Get(CorrelationHeader) != "corr-0000-0002" {
		t.Fatalf("recover = %d %v", res.StatusCode, res.Header)
	}
	assertPrivate(t, "recovered panic", res)
}

func TestCachePolicyFlushAndUnwrap(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=5")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush: %v", err)
		}
	})
	rec := httptest.NewRecorder()
	CachePolicy(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/x", nil))
	if !rec.Flushed {
		t.Fatal("flush not forwarded")
	}
	assertPrivate(t, "flushed", rec.Result())
}
