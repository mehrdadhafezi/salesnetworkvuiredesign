package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func spec(j Job, g Gate) Spec {
	return Spec{Job: j, Gate: g, Interval: 100 * time.Millisecond, Timeout: time.Second, MaxBackoff: time.Second}
}

func counter(name string, n *atomic.Int64, err error) Job {
	return JobFunc{JobName: name, Fn: func(ctx context.Context, r *Run) error {
		n.Add(1)
		r.Count("things_done", 2)
		r.Gauge("backlog", 7)
		return err
	}}
}

func TestConfigValidation(t *testing.T) {
	var n atomic.Int64
	good := Config{Logger: quiet, Specs: []Spec{spec(counter("good_job", &n, nil), Always)}, MaxParallel: 1}
	if _, err := New(good); err != nil {
		t.Fatal(err)
	}
	bad := []func(*Config){
		func(c *Config) { c.Logger = nil },
		func(c *Config) { c.Specs = nil },
		func(c *Config) { c.MaxParallel = 0 },
		func(c *Config) { c.MaxParallel = 99 },
		func(c *Config) { c.Jitter = 0.9 },
		func(c *Config) { c.Specs = []Spec{spec(counter("Bad-Name", &n, nil), Always)} },
		func(c *Config) {
			c.Specs = []Spec{spec(counter("dup_job", &n, nil), Always), spec(counter("dup_job", &n, nil), Always)}
		},
		func(c *Config) { s := spec(counter("x_job", &n, nil), nil); c.Specs = []Spec{s} },
		func(c *Config) {
			s := spec(counter("x_job", &n, nil), Always)
			s.Interval = time.Millisecond
			c.Specs = []Spec{s}
		},
		func(c *Config) { s := spec(counter("x_job", &n, nil), Always); s.Timeout = 0; c.Specs = []Spec{s} },
		func(c *Config) {
			s := spec(counter("x_job", &n, nil), Always)
			s.MaxBackoff = time.Millisecond
			c.Specs = []Spec{s}
		},
		func(c *Config) { c.Checks = []Check{{Name: "db"}} },
	}
	for i, m := range bad {
		c := good
		m(&c)
		if _, err := New(c); !errors.Is(err, ErrInvalid) {
			t.Fatalf("case %d accepted", i)
		}
	}
}

// 19 / gates: a paused or blocked job never runs and reports its state.
func TestGatesPausedAndBlocked(t *testing.T) {
	var a, b, c atomic.Int64
	paused := func(context.Context) (GateState, string) { return Paused, "JOB_SWITCH_OFF" }
	rt, err := New(Config{Logger: quiet, MaxParallel: 2, Specs: []Spec{
		spec(counter("paused_job", &a, nil), paused),
		spec(counter("blocked_job", &b, nil), BlockedBy("NO_ADAPTER")),
		spec(counter("open_job", &c, nil), Always),
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"paused_job", "blocked_job", "open_job"} {
		if _, err := rt.RunOnce(context.Background(), name); err != nil {
			t.Fatal(err)
		}
	}
	if a.Load() != 0 || b.Load() != 0 || c.Load() != 1 {
		t.Fatalf("runs: paused=%d blocked=%d open=%d", a.Load(), b.Load(), c.Load())
	}
	st := map[string]Status{}
	for _, s := range rt.Statuses() {
		st[s.Job] = s
	}
	if st["paused_job"].State != "paused" || st["blocked_job"].State != "blocked" || st["blocked_job"].Reason != "NO_ADAPTER" ||
		st["open_job"].State != "idle" || st["open_job"].LastSuccess.IsZero() {
		t.Fatalf("statuses: %+v", st)
	}
	if v, _ := rt.Metrics().Value("job_state_blocked", "blocked_job"); v != 1 {
		t.Fatal("blocked state must be visible in metrics")
	}
	if v, _ := rt.Metrics().Value("things_done_total", "open_job"); v != 2 {
		t.Fatal("job counters")
	}
}

// Failure: recorded, backoff grows, a panic fails the pass (never success).
func TestFailurePanicAndBackoff(t *testing.T) {
	var n atomic.Int64
	boom := counter("failing_job", &n, errors.New("db down"))
	panicky := JobFunc{JobName: "panic_job", Fn: func(context.Context, *Run) error { panic("bug") }}
	s := spec(boom, Always)
	s.MaxBackoff = 800 * time.Millisecond
	rt, err := New(Config{Logger: quiet, MaxParallel: 1, Specs: []Spec{s, spec(panicky, Always)}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		_, _ = rt.RunOnce(context.Background(), "failing_job")
	}
	st, _ := rt.RunOnce(context.Background(), "panic_job")
	if st.State != "failing" || !strings.Contains(st.LastError, "panicked") || !st.LastSuccess.IsZero() {
		t.Fatalf("panic: %+v", st)
	}
	j := rt.jobs[0]
	if d := rt.delay(j); d != 800*time.Millisecond {
		t.Fatalf("backoff must be capped at MaxBackoff: %s", d)
	}
	if v, _ := rt.Metrics().Value("job_runs_failed_total", "failing_job"); v != 5 {
		t.Fatalf("failed runs: %v", v)
	}
}

// Per-pass deadline: a stuck pass is cancelled and counted as a timeout.
func TestPassDeadline(t *testing.T) {
	stuck := JobFunc{JobName: "stuck_job", Fn: func(ctx context.Context, _ *Run) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	s := spec(stuck, Always)
	s.Timeout = 50 * time.Millisecond
	rt, _ := New(Config{Logger: quiet, MaxParallel: 1, Specs: []Spec{s}})
	st, _ := rt.RunOnce(context.Background(), "stuck_job")
	if st.State != "failing" {
		t.Fatalf("timed-out pass: %+v", st)
	}
	if v, _ := rt.Metrics().Value("job_timeouts_total", "stuck_job"); v != 1 {
		t.Fatal("timeouts metric")
	}
}

// 20: graceful shutdown waits for a pass in flight; after the grace period
// the pass is cancelled. Controlled concurrency is honoured.
func TestGracefulShutdownAndConcurrency(t *testing.T) {
	var active, peak atomic.Int64
	started := make(chan struct{}, 8)
	slow := func(name string, d time.Duration) Job {
		return JobFunc{JobName: name, Fn: func(ctx context.Context, _ *Run) error {
			cur := active.Add(1)
			for {
				p := peak.Load()
				if cur <= p || peak.CompareAndSwap(p, cur) {
					break
				}
			}
			defer active.Add(-1)
			started <- struct{}{}
			select {
			case <-time.After(d):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}}
	}
	rt, err := New(Config{Logger: quiet, MaxParallel: 1, Grace: 2 * time.Second,
		Specs: []Spec{spec(slow("one_job", 150*time.Millisecond), Always), spec(slow("two_job", 150*time.Millisecond), Always)}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- rt.Run(ctx) }()
	<-started
	stop() // shutdown while a pass is in flight
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for _, s := range rt.Statuses() {
		if s.State != "stopped" {
			t.Fatalf("after shutdown: %+v", s)
		}
	}
	if peak.Load() != 1 {
		t.Fatalf("MaxParallel=1 exceeded: %d", peak.Load())
	}
	var completed int64
	for _, s := range rt.Statuses() {
		completed += s.Runs - s.Failures
	}
	if completed < 1 {
		t.Fatal("the in-flight pass must complete within the grace period")
	}
	// Grace elapsed: the pass is cancelled, not abandoned.
	var cancelled atomic.Bool
	hang := JobFunc{JobName: "hang_job", Fn: func(ctx context.Context, _ *Run) error {
		started <- struct{}{}
		<-ctx.Done()
		cancelled.Store(true)
		return ctx.Err()
	}}
	hs := spec(hang, Always)
	hs.Timeout = time.Minute
	rt2, _ := New(Config{Logger: quiet, MaxParallel: 1, Grace: 100 * time.Millisecond, Specs: []Spec{hs}})
	ctx2, stop2 := context.WithCancel(context.Background())
	go func() { done <- rt2.Run(ctx2) }()
	<-started
	stop2()
	<-done
	if !cancelled.Load() {
		t.Fatal("a pass still running after the grace period must be cancelled")
	}
}

// Health, readiness and metrics endpoints; readiness fails on a down
// dependency and on a stale job; labels carry job names only.
func TestHealthReadinessAndMetrics(t *testing.T) {
	var n atomic.Int64
	var down atomic.Bool
	now := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	rt, _ := New(Config{Logger: quiet, MaxParallel: 1, Now: clock, Instance: "test-instance",
		Specs: []Spec{spec(counter("ready_job", &n, nil), Always)},
		Checks: []Check{{Name: "db_payments", Fn: func(context.Context) error {
			if down.Load() {
				return errors.New("down")
			}
			return nil
		}}}})
	h := rt.Handler()
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	if r := get("/health/live"); r.Code != 200 {
		t.Fatal("live")
	}
	if r := get("/health/ready"); r.Code != 503 {
		t.Fatal("a runtime that is not running is not ready")
	}
	rt.mu.Lock()
	rt.running, rt.started = true, now
	rt.mu.Unlock()
	_, _ = rt.RunOnce(context.Background(), "ready_job")
	if r := get("/health/ready"); r.Code != 200 {
		t.Fatalf("ready: %s", r.Body)
	}
	down.Store(true)
	if r := get("/health/ready"); r.Code != 503 || !strings.Contains(r.Body.String(), "unavailable") {
		t.Fatalf("database down must make the worker unready: %s", r.Body)
	}
	down.Store(false)
	mu.Lock()
	now = now.Add(time.Hour) // no pass completed for far longer than 3 intervals
	mu.Unlock()
	if r := get("/health/ready"); r.Code != 503 || !strings.Contains(r.Body.String(), `"stale":true`) {
		t.Fatalf("stale job must make the worker unready: %s", r.Body)
	}
	m := get("/metrics").Body.String()
	for _, want := range []string{`crm_worker_things_done_total{job="ready_job"} 2`, `crm_worker_backlog{job="ready_job"} 7`,
		`# TYPE crm_worker_job_runs_succeeded_total counter`} {
		if !strings.Contains(m, want) {
			t.Fatalf("metrics missing %q:\n%s", want, m)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(m), "\n") {
		if !strings.HasPrefix(line, "#") && !strings.Contains(line, `{job="ready_job"}`) {
			t.Fatalf("a series carries a label other than the job name: %s", line)
		}
	}
	// Invalid names never reach the exposition.
	rt.Metrics().Set("customer phone", "ready_job", 1)
	rt.Metrics().Set("ok_metric", "0912 345 6789", 1)
	if m := get("/metrics").Body.String(); strings.Contains(m, "phone") || strings.Contains(m, "0912") {
		t.Fatal("invalid metric names / labels must be dropped")
	}
}

// R2 review P2: shutdown is bounded even when a pass ignores cancellation
// (grace, then cancel, then at most HardStop).
func TestShutdownBoundedWhenPassIgnoresCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	stubborn := JobFunc{JobName: "stubborn_job", Fn: func(context.Context, *Run) error {
		started <- struct{}{}
		time.Sleep(10 * time.Second) // ignores ctx entirely
		return nil
	}}
	s := spec(stubborn, Always)
	s.Timeout = time.Minute
	rt, err := New(Config{Logger: quiet, MaxParallel: 1, Grace: 100 * time.Millisecond, HardStop: 200 * time.Millisecond, Specs: []Spec{s}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- rt.Run(ctx) }()
	<-started
	begin := time.Now()
	stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown must be bounded by Grace + HardStop")
	}
	if d := time.Since(begin); d > 2*time.Second {
		t.Fatalf("shutdown took %s", d)
	}
	if _, err := New(Config{Logger: quiet, MaxParallel: 1, HardStop: -time.Second, Specs: []Spec{s}}); !errors.Is(err, ErrInvalid) {
		t.Fatal("negative hard stop accepted")
	}
}
