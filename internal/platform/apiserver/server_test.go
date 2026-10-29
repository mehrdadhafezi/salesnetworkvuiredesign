package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

var (
	fixedAt, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")
	quiet      = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func handler(t *testing.T, env authstub.Environment, devAuth bool) http.Handler {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Environment, cfg.DevAuth = env, devAuth
	h, err := NewHandler(cfg, timeutil.FixedClock{At: fixedAt}, quiet)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func do(h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h := handler(t, authstub.Production, false)
	rec := do(h, http.MethodGet, "/health/live", nil)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" || strings.TrimSpace(rec.Body.String()) != `{"status":"live"}` {
		t.Fatalf("live = %d %q %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if !httpx.ValidCorrelationID(rec.Header().Get(httpx.CorrelationHeader)) {
		t.Fatal("health must carry a correlation id")
	}
	rec = do(h, http.MethodGet, "/health/ready", nil)
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"checks":[],"status":"ready"}` {
		t.Fatalf("ready = %d %s", rec.Code, rec.Body.String())
	}
	for _, leak := range []string{"postgres", "password", "dsn", "CRM_"} {
		if strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(leak)) {
			t.Fatalf("readiness leaks %q", leak)
		}
	}
}

func session(t *testing.T, h http.Handler, headers map[string]string) oapi.SessionEnvelope {
	t.Helper()
	rec := do(h, http.MethodGet, "/api/v1/session", headers)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("session = %d %s", rec.Code, rec.Body.String())
	}
	var env oapi.SessionEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.CorrelationId != rec.Header().Get(httpx.CorrelationHeader) {
		t.Fatalf("body correlation %q != header %q", env.CorrelationId, rec.Header().Get(httpx.CorrelationHeader))
	}
	if !strings.HasSuffix(rec.Body.String(), "\n") || !strings.Contains(rec.Body.String(), `"server_time":"2026-10-29T10:00:00Z"`) {
		t.Fatalf("server_time must be RFC 3339 UTC: %s", rec.Body.String())
	}
	return env
}

func TestSessionInProductionIgnoresDevHeader(t *testing.T) {
	h := handler(t, authstub.Production, false)
	env := session(t, h, map[string]string{authstub.Header: "dev-user-001", httpx.CorrelationHeader: "client-req-0001"})
	if env.Outcome != "SUCCESS" || env.Data.Authenticated || env.Data.AuthMode != "none" || env.Data.Principal != nil || env.Data.CapabilityHints != nil {
		t.Fatalf("production session = %+v", env)
	}
	if env.CorrelationId != "client-req-0001" || env.Data.DisplayZone != "Asia/Tehran" || len(env.Reasons) != 0 {
		t.Fatalf("session = %+v", env)
	}
	// Even an unknown identity header is simply ignored in production.
	if rec := do(h, http.MethodGet, "/api/v1/session", map[string]string{authstub.Header: "anything"}); rec.Code != 200 {
		t.Fatalf("production must not read %s (got %d)", authstub.Header, rec.Code)
	}
}

func TestSessionWithDevStub(t *testing.T) {
	h := handler(t, authstub.Development, true)
	anon := session(t, h, nil)
	if anon.Data.Authenticated || anon.Data.AuthMode != "dev_stub" {
		t.Fatalf("anonymous dev session = %+v", anon.Data)
	}
	env := session(t, h, map[string]string{authstub.Header: "dev-user-001"})
	p := env.Data.Principal
	if !env.Data.Authenticated || p == nil || p.PrincipalId != "dev-user-001" || p.DisplayName != "Dev Operator" || !bool(p.Synthetic) {
		t.Fatalf("dev session = %+v", env.Data)
	}
	if env.Data.CapabilityHints == nil || len(*env.Data.CapabilityHints) == 0 {
		t.Fatal("capability hints missing")
	}
	rec := do(h, http.MethodGet, "/api/v1/session", map[string]string{authstub.Header: "root"})
	if rec.Code != 400 || rec.Header().Get("Content-Type") != httpx.ProblemContentType {
		t.Fatalf("unknown dev identity = %d %s", rec.Code, rec.Body.String())
	}
}

func TestDevAuthCannotRunInProduction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DevAuth = true // Environment stays production
	if _, err := NewHandler(cfg, timeutil.FixedClock{At: fixedAt}, quiet); !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), "production") {
		t.Fatalf("NewHandler(prod, devAuth) err = %v", err)
	}
	if _, err := NewServer(cfg, timeutil.FixedClock{At: fixedAt}, quiet); err == nil {
		t.Fatal("NewServer must refuse dev auth in production")
	}
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for name, m := range map[string]map[string]string{
		"default env":      {EnvDevAuth: "1"},
		"explicit prod":    {EnvEnv: "production", EnvDevAuth: "1"},
		"bad dev value":    {EnvEnv: "development", EnvDevAuth: "true"},
		"unknown env":      {EnvEnv: "staging"},
		"unknown env auth": {EnvEnv: "staging", EnvDevAuth: "1"},
	} {
		if _, err := ConfigFromEnv(env(m)); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
	cfgOK, err := ConfigFromEnv(env(map[string]string{EnvEnv: "test", EnvDevAuth: "1", EnvAddr: "127.0.0.1:0"}))
	if err != nil || !cfgOK.DevAuth || cfgOK.Environment != authstub.Test || cfgOK.Addr != "127.0.0.1:0" {
		t.Fatalf("test config = %+v, %v", cfgOK, err)
	}
	def, err := ConfigFromEnv(env(nil))
	if err != nil || def.Environment != authstub.Production || def.DevAuth || def.Addr != "127.0.0.1:8080" {
		t.Fatalf("default config = %+v, %v", def, err)
	}
}

func TestConfigValidation(t *testing.T) {
	bad := []func(*Config){
		func(c *Config) { c.Addr = "" },
		func(c *Config) { c.ReadHeaderTimeout = 0 },
		func(c *Config) { c.WriteTimeout = -1 },
		func(c *Config) { c.MaxHeaderBytes = 0 },
		func(c *Config) { c.MaxBodyBytes = 0 },
	}
	for i, m := range bad {
		c := DefaultConfig()
		m(&c)
		if err := c.Validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("mutation %d accepted", i)
		}
	}
	if _, err := NewHandler(DefaultConfig(), nil, quiet); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("nil clock must be rejected")
	}
}

func TestCentralProblems(t *testing.T) {
	h := handler(t, authstub.Production, false)
	rec := do(h, http.MethodGet, "/api/v1/customers", nil)
	if rec.Code != 404 || rec.Header().Get("Content-Type") != httpx.ProblemContentType {
		t.Fatalf("unknown path = %d %s", rec.Code, rec.Body.String())
	}
	var p oapi.Problem
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if p.ReasonCode != "CONTRACT_INVALID" || p.Status != 404 || p.CorrelationId == "" {
		t.Fatalf("404 problem = %+v", p)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec := do(h, method, "/api/v1/session", nil)
		if rec.Code != 405 || rec.Header().Get("Allow") != "GET" || rec.Header().Get("Content-Type") != httpx.ProblemContentType {
			t.Fatalf("%s /api/v1/session = %d allow=%q", method, rec.Code, rec.Header().Get("Allow"))
		}
	}
}

func loadContract(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("../../../api/openapi/openapi.bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("bundled contract invalid: %v", err)
	}
	return doc
}

// TestRoutesMatchContract: the server's route table is exactly the contract.
func TestRoutesMatchContract(t *testing.T) {
	doc := loadContract(t)
	got := map[string][]string{}
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			got[path] = append(got[path], method)
		}
		sort.Strings(got[path])
	}
	if len(got) != len(ContractPaths) {
		t.Fatalf("contract paths %v != server paths %v", got, ContractPaths)
	}
	h := handler(t, authstub.Production, false)
	for path, methods := range ContractPaths {
		if strings.Join(got[path], ",") != strings.Join(methods, ",") {
			t.Fatalf("%s: contract %v != server %v", path, got[path], methods)
		}
		for _, m := range methods {
			if rec := do(h, m, path, nil); rec.Code == 404 || rec.Code == 405 {
				t.Fatalf("%s %s is in the contract but not served (%d)", m, path, rec.Code)
			}
		}
	}
}

func TestServeGracefulShutdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Addr = "127.0.0.1:0"
	cfg.ShutdownTimeout = 2 * time.Second
	srv, err := NewServer(cfg, timeutil.FixedClock{At: fixedAt}, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if srv.ReadHeaderTimeout != cfg.ReadHeaderTimeout || srv.MaxHeaderBytes != cfg.MaxHeaderBytes || srv.IdleTimeout == 0 || srv.WriteTimeout == 0 {
		t.Fatalf("server limits not applied: %+v", srv)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, srv, cfg, quiet) }()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v after shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("graceful shutdown did not finish")
	}
	bad := DefaultConfig()
	bad.Addr = "256.0.0.1:99999"
	srv2, _ := NewServer(bad, timeutil.FixedClock{At: fixedAt}, quiet)
	if err := Serve(context.Background(), srv2, bad, quiet); err == nil {
		t.Fatal("an unusable listen address must fail")
	}
}
