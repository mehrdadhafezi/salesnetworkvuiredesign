package apiserver

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
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

// R3 review remediation (PR #43, Codex P2): several API surfaces open
// separate pools for the SAME owner. The exposition must still carry each
// (metric, labels) identity exactly once, with the owner's pools summed.
func TestMetricsExpositionHasNoDuplicateSeriesWithSharedOwnerPools(t *testing.T) {
	dsn := pgtest.AdminURL(t)
	before := map[string]int64{}
	for _, s := range db.Stats() {
		before[s.Owner] = int64(s.Max)
	}
	owners := []owner.Module{"identity", "identity", "platform_featureflags", "platform_featureflags", "platform_featureflags"}
	for _, m := range owners {
		if _, err := pgtest.OpenPool(t, dsn).Runner(m); err != nil {
			t.Fatal(err)
		}
	}
	m := NewMetrics()
	var b strings.Builder
	m.Write(&b)
	seen := map[string]int{}
	values := map[string]string{}
	for _, line := range strings.Split(b.String(), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.LastIndexByte(line, ' ')
		id := line[:i]
		seen[id]++
		values[id] = line[i+1:]
	}
	for id, n := range seen {
		if n > 1 {
			t.Fatalf("duplicate sample identity %s (%d times):\n%s", id, n, b.String())
		}
	}
	per := db.DefaultConfig(dsn).MaxConns
	for owner, pools := range map[string]int64{"identity": 2, "platform_featureflags": 3} {
		id := `crm_db_pool_max_conns{owner="` + owner + `"}`
		want := strconv.FormatInt(before[owner]+pools*int64(per), 10)
		if values[id] != want {
			t.Fatalf("%s = %q, want %s (the owner's pools summed)", id, values[id], want)
		}
	}
}
