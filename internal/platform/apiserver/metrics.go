package apiserver

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// EnvMetricsAddr is the private metrics listener (R3). Empty disables it.
// It must never be the public address: /metrics is not routed through the
// public handler and carries no business data, but it is operational
// information for the internal network only.
const EnvMetricsAddr = "CRM_API_METRICS_ADDR"

// latencyBuckets are the request-duration histogram upper bounds (s).
var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type reqKey struct{ method, route, class string }

// Metrics records low-cardinality request telemetry: the method, the
// OpenAPI route TEMPLATE (never the concrete path, so no id or value can
// become a label) and the status class. Login outcomes are counted by
// status only; no login name, password, token or cookie is ever observed.
type Metrics struct {
	mu       sync.Mutex
	requests map[reqKey]uint64
	buckets  map[reqKey][]uint64
	sums     map[reqKey]float64
	login    map[string]uint64
	inflight int64
	started  time.Time
}

// NewMetrics returns an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{requests: map[reqKey]uint64{}, buckets: map[reqKey][]uint64{}, sums: map[reqKey]float64{}, login: map[string]uint64{},
		started: time.Now()}
}

var knownMethods = map[string]bool{http.MethodGet: true, http.MethodPost: true, http.MethodPut: true, http.MethodPatch: true,
	http.MethodDelete: true, http.MethodHead: true, http.MethodOptions: true}

func routeLabel(r *http.Request) string {
	if t, ok := template(r.URL.Path); ok {
		return t
	}
	return "unmatched"
}

func statusClass(code int) string {
	if code < 100 || code > 599 {
		return "other"
	}
	return strconv.Itoa(code/100) + "xx"
}

func loginOutcome(code int) string {
	switch code {
	case http.StatusOK:
		return "ok_or_blocked" // 200 carries SUCCESS or BLOCKED (flag off)
	case http.StatusUnauthorized:
		return "denied"
	case http.StatusTooManyRequests:
		return "throttled"
	case http.StatusForbidden:
		return "cross_site_rejected"
	case http.StatusBadRequest:
		return "invalid_request"
	}
	return "error"
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.code == 0 {
		w.code = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Middleware observes every request handled by next.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.mu.Lock()
		m.inflight++
		m.mu.Unlock()
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			code := sw.code
			if code == 0 {
				code = http.StatusOK
			}
			method := r.Method
			if !knownMethods[method] {
				method = "OTHER"
			}
			k := reqKey{method: method, route: routeLabel(r), class: statusClass(code)}
			d := time.Since(start).Seconds()
			m.mu.Lock()
			defer m.mu.Unlock()
			m.inflight--
			m.requests[k]++
			m.sums[k] += d
			b := m.buckets[k]
			if b == nil {
				b = make([]uint64, len(latencyBuckets))
				m.buckets[k] = b
			}
			for i, ub := range latencyBuckets {
				if d <= ub {
					b[i]++
				}
			}
			if r.URL.Path == loginPath && r.Method == http.MethodPost {
				m.login[loginOutcome(code)]++
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

func esc(s string) string { return strconv.Quote(s) }

// buildInfo returns the module version and VCS revision of the binary.
func buildInfo() (version, revision string) {
	version, revision = "unknown", "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" {
			version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				revision = s.Value
			}
		}
	}
	return version, revision
}

// Write renders the registry plus database pool stats in Prometheus text.
func (m *Metrics) Write(b *strings.Builder) {
	m.mu.Lock()
	keys := make([]reqKey, 0, len(m.requests))
	for k := range m.requests {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, c := keys[i], keys[j]
		return a.route+a.method+a.class < c.route+c.method+c.class
	})
	b.WriteString("# TYPE crm_api_requests_total counter\n")
	for _, k := range keys {
		fmt.Fprintf(b, "crm_api_requests_total{method=%s,route=%s,class=%s} %d\n", esc(k.method), esc(k.route), esc(k.class), m.requests[k])
	}
	b.WriteString("# TYPE crm_api_request_duration_seconds histogram\n")
	for _, k := range keys {
		lbl := fmt.Sprintf("method=%s,route=%s,class=%s", esc(k.method), esc(k.route), esc(k.class))
		for i, ub := range latencyBuckets {
			fmt.Fprintf(b, "crm_api_request_duration_seconds_bucket{%s,le=%q} %d\n", lbl, strconv.FormatFloat(ub, 'g', -1, 64), m.buckets[k][i])
		}
		fmt.Fprintf(b, "crm_api_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", lbl, m.requests[k])
		fmt.Fprintf(b, "crm_api_request_duration_seconds_sum{%s} %s\n", lbl, strconv.FormatFloat(m.sums[k], 'g', -1, 64))
		fmt.Fprintf(b, "crm_api_request_duration_seconds_count{%s} %d\n", lbl, m.requests[k])
	}
	outcomes := make([]string, 0, len(m.login))
	for o := range m.login {
		outcomes = append(outcomes, o)
	}
	sort.Strings(outcomes)
	b.WriteString("# TYPE crm_api_login_outcomes_total counter\n")
	for _, o := range outcomes {
		fmt.Fprintf(b, "crm_api_login_outcomes_total{outcome=%s} %d\n", esc(o), m.login[o])
	}
	fmt.Fprintf(b, "# TYPE crm_api_inflight_requests gauge\ncrm_api_inflight_requests %d\n", m.inflight)
	fmt.Fprintf(b, "# TYPE crm_api_start_time_seconds gauge\ncrm_api_start_time_seconds %d\n", m.started.Unix())
	m.mu.Unlock()
	version, revision := buildInfo()
	fmt.Fprintf(b, "# TYPE crm_build_info gauge\ncrm_build_info{component=\"crm-api\",version=%s,revision=%s} 1\n", esc(version), esc(revision))
	WritePoolStats(b, "crm_db_pool")
}

// WritePoolStats renders db.Stats() under prefix, labelled by owner.
func WritePoolStats(b *strings.Builder, prefix string) {
	stats := db.Stats()
	for _, g := range []struct {
		name string
		val  func(db.PoolStat) string
		kind string
	}{
		{"max_conns", func(s db.PoolStat) string { return strconv.Itoa(int(s.Max)) }, "gauge"},
		{"total_conns", func(s db.PoolStat) string { return strconv.Itoa(int(s.Total)) }, "gauge"},
		{"idle_conns", func(s db.PoolStat) string { return strconv.Itoa(int(s.Idle)) }, "gauge"},
		{"acquired_conns", func(s db.PoolStat) string { return strconv.Itoa(int(s.Acquired)) }, "gauge"},
		{"empty_acquire_total", func(s db.PoolStat) string { return strconv.FormatInt(s.EmptyAcquire, 10) }, "counter"},
	} {
		fmt.Fprintf(b, "# TYPE %s_%s %s\n", prefix, g.name, g.kind)
		for _, s := range stats {
			fmt.Fprintf(b, "%s_%s{owner=%s} %s\n", prefix, g.name, esc(s.Owner), g.val(s))
		}
	}
}

// Handler serves GET /metrics only (private listener).
func (m *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		m.Write(&b)
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(b.String()))
	})
	return mux
}
