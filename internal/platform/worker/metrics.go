package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Metrics is a minimal Prometheus-text registry. Every series carries
// exactly one label, job, whose values are registered job names: no
// identifier, customer data, token or receipt content can become a label.
type Metrics struct {
	mu     sync.Mutex
	values map[string]map[string]float64 // metric → job → value
	kinds  map[string]string
}

// NewMetrics returns an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{values: map[string]map[string]float64{}, kinds: map[string]string{}}
}

var metricPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// jobStates are the values of the crm_worker_job_state gauge.
var jobStates = []string{"idle", "running", "paused", "blocked", "failing", "stopped"}

func (m *Metrics) put(metric, kind, job string, f func(float64) float64) {
	if !metricPattern.MatchString(metric) || !namePattern.MatchString(job) {
		return // invalid names are dropped, never emitted
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.kinds[metric]; ok && k != kind {
		return
	}
	m.kinds[metric] = kind
	if m.values[metric] == nil {
		m.values[metric] = map[string]float64{}
	}
	m.values[metric][job] = f(m.values[metric][job])
}

// Add adds v to a counter (metric must end in _total).
func (m *Metrics) Add(metric, job string, v float64) {
	if !strings.HasSuffix(metric, "_total") || v < 0 {
		return
	}
	m.put(metric, "counter", job, func(x float64) float64 { return x + v })
}

// Set sets a gauge.
func (m *Metrics) Set(metric, job string, v float64) {
	m.put(metric, "gauge", job, func(float64) float64 { return v })
}

// Inc adds d to a gauge.
func (m *Metrics) Inc(metric, job string, d float64) {
	m.put(metric, "gauge", job, func(x float64) float64 { return x + d })
}

func (m *Metrics) setState(job, state string) {
	for _, s := range jobStates {
		v := 0.0
		if s == state {
			v = 1
		}
		m.Set("job_state_"+s, job, v)
	}
}

// Value returns one series value (tests, diagnostics).
func (m *Metrics) Value(metric, job string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[metric][job]
	return v, ok
}

// Write renders the registry in the Prometheus text format.
func (m *Metrics) Write(b *strings.Builder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.values))
	for n := range m.values {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		full := "crm_worker_" + n
		fmt.Fprintf(b, "# TYPE %s %s\n", full, m.kinds[n])
		jobs := make([]string, 0, len(m.values[n]))
		for j := range m.values[n] {
			jobs = append(jobs, j)
		}
		sort.Strings(jobs)
		for _, j := range jobs {
			fmt.Fprintf(b, "%s{job=%q} %s\n", full, j, strconv.FormatFloat(m.values[n][j], 'g', -1, 64))
		}
	}
}

// Handler serves /health/live, /health/ready and /metrics. It is meant for
// a private listener (loopback or the orchestrator network); it exposes no
// identifiers, payloads or configuration values.
func (rt *Runtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]any{"status": "live", "instance": rt.cfg.Instance})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ok, detail := rt.Ready(r.Context())
		code := http.StatusOK
		if !ok {
			code = http.StatusServiceUnavailable
		}
		write(w, code, detail)
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		rt.metrics.Write(&b)
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(b.String()))
	})
	return mux
}

func write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Ready reports readiness: every dependency check passes, the runtime is
// running, and no enabled job is stale (no completed pass for 3 intervals
// plus its timeout after the first such window). Paused and blocked jobs do
// not make the worker unready; they are reported per job and in metrics.
func (rt *Runtime) Ready(ctx context.Context) (bool, map[string]any) {
	ok := true
	checks := map[string]string{}
	for _, ch := range rt.cfg.Checks {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := ch.Fn(cctx)
		cancel()
		if err != nil {
			ok = false
			checks[ch.Name] = "unavailable"
		} else {
			checks[ch.Name] = "ok"
		}
	}
	rt.mu.Lock()
	running, started := rt.running, rt.started
	rt.mu.Unlock()
	if !running {
		ok = false
	}
	now := rt.now()
	jobs := map[string]any{}
	for _, j := range rt.jobs {
		j.mu.Lock()
		st := j.st
		j.mu.Unlock()
		window := 3*j.spec.Interval + j.spec.Timeout
		stale := running && (st.State == "idle" || st.State == "failing" || st.State == "running") &&
			now.Sub(started) > window && (st.LastSuccess.IsZero() || now.Sub(st.LastSuccess) > window)
		if stale {
			ok = false
		}
		jobs[st.Job] = map[string]any{"state": st.State, "reason": st.Reason, "stale": stale, "consecutive_failures": st.ConsecutiveFailures}
	}
	status := "ready"
	if !ok {
		status = "not_ready"
	}
	return ok, map[string]any{"status": status, "instance": rt.cfg.Instance, "checks": checks, "jobs": jobs}
}
