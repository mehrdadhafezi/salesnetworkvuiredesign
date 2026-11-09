package apiserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
)

// R3: request telemetry is labelled by route TEMPLATE, method and status
// class only; no concrete id, query value, login name or credential can
// become a label or a sample.
func TestRequestMetricsLowCardinalityAndRedacted(t *testing.T) {
	m := NewMetrics()
	cfg := DefaultConfig()
	cfg.Environment = authstub.Production
	h, err := NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: fixedAt}, quiet, Surfaces{Metrics: m})
	if err != nil {
		t.Fatal(err)
	}
	secretID := "7a1c9a7e-5f2b-4b8e-9d11-0c0ffee00001"
	for _, p := range []string{"/health/live", "/health/ready", "/api/v1/identity/users/" + secretID, "/api/v1/identity/users/" + secretID,
		"/wp-admin/admin-ajax.php?phone=09123456789"} {
		do(h, http.MethodGet, p, nil)
	}
	doBody(h, http.MethodPost, "/api/v1/auth/login", `{"login":"seller.secret","password":"synthetic-Pass-SECRET"}`,
		map[string]string{"Content-Type": "application/json"})
	var b strings.Builder
	m.Write(&b)
	out := b.String()
	for _, want := range []string{
		`crm_api_requests_total{method="GET",route="/health/live",class="2xx"} 1`,
		`crm_api_requests_total{method="GET",route="/api/v1/identity/users/{user_id}",class=`,
		`crm_api_requests_total{method="GET",route="unmatched",class="4xx"} 1`,
		`crm_api_login_outcomes_total{outcome="ok_or_blocked"} 1`,
		`crm_api_request_duration_seconds_bucket{method="GET",route="/health/live",class="2xx",le="+Inf"} 1`,
		`crm_build_info{component="crm-api"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, leak := range []string{secretID, "09123456789", "admin-ajax", "seller.secret", "SECRET", "phone"} {
		if strings.Contains(out, leak) {
			t.Fatalf("telemetry leaks %q", leak)
		}
	}
	// Served only by the private handler.
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("metrics handler: %d", rec.Code)
	}
	if r := do(h, http.MethodGet, "/metrics", nil); r.Code == 200 {
		t.Fatal("/metrics must not be served on the public listener")
	}
}

func TestMetricsAddrMustDifferFromPublic(t *testing.T) {
	env := map[string]string{EnvAddr: "127.0.0.1:8080", EnvMetricsAddr: "127.0.0.1:8080"}
	if _, err := ConfigFromEnv(func(k string) string { return env[k] }); err == nil {
		t.Fatal("a metrics listener on the public address must be refused")
	}
	env[EnvMetricsAddr] = "127.0.0.1:9090"
	if c, err := ConfigFromEnv(func(k string) string { return env[k] }); err != nil || c.MetricsAddr != "127.0.0.1:9090" {
		t.Fatalf("metrics addr: %+v %v", c, err)
	}
}
