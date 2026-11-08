// Package worker is the shared background-job runtime (R2).
//
// A Job is one bounded pass over durable work (claim a batch, process it,
// record outcomes). The runtime schedules each registered job on its own
// loop with a bounded interval and jitter, a per-run deadline, a gate that
// is re-evaluated before every pass (disabled / paused / blocked), bounded
// error backoff, a cap on concurrently running passes, health and readiness
// endpoints, Prometheus-text metrics and graceful shutdown.
//
// Durability, idempotency and exclusivity are NOT the runtime's job: every
// job claims its work in PostgreSQL (row locks, write-ahead states, inbox
// deduplication), so a pass may be interrupted at any instant and two
// instances may run the same job concurrently. The runtime never retries a
// failed pass immediately and never reports a pass it did not complete.
package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Errors.
var (
	ErrInvalid = errors.New("worker: invalid configuration")
)

// GateState is the result of a job's gate.
type GateState uint8

// Gate states.
const (
	// Enabled: the pass runs.
	Enabled GateState = iota + 1
	// Paused: the job's runtime switch is off; queued work stays queued and
	// visible. Nothing is reported as done.
	Paused
	// Blocked: a prerequisite is missing (no approved adapter, an open
	// decision, an unreadable flag). The pass does not run.
	Blocked
)

func (g GateState) String() string {
	switch g {
	case Enabled:
		return "enabled"
	case Paused:
		return "paused"
	case Blocked:
		return "blocked"
	}
	return "invalid"
}

// Gate decides, before every pass, whether the pass may run. reason is a
// short operator-facing code (never a secret).
type Gate func(ctx context.Context) (state GateState, reason string)

// Always is the gate of a job with no runtime switch.
func Always(context.Context) (GateState, string) { return Enabled, "" }

// BlockedBy is a gate that never opens.
func BlockedBy(reason string) Gate {
	return func(context.Context) (GateState, string) { return Blocked, reason }
}

// Job is one bounded pass. Run must return when ctx is done; it reports its
// counts and backlog through r.
type Job interface {
	Name() string
	Run(ctx context.Context, r *Run) error
}

// JobFunc adapts a function to Job.
type JobFunc struct {
	JobName string
	Fn      func(ctx context.Context, r *Run) error
}

// Name implements Job.
func (j JobFunc) Name() string { return j.JobName }

// Run implements Job.
func (j JobFunc) Run(ctx context.Context, r *Run) error { return j.Fn(ctx, r) }

// Spec registers a job.
type Spec struct {
	Job      Job
	Gate     Gate
	Interval time.Duration // between passes (with ±Jitter)
	Timeout  time.Duration // per-pass deadline
	// MaxBackoff caps the delay after consecutive failed passes.
	MaxBackoff time.Duration
}

// Check is a readiness dependency (a database ping).
type Check struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Config is the runtime configuration.
type Config struct {
	Instance string // empty: hostname + random suffix
	Logger   *slog.Logger
	Specs    []Spec
	Checks   []Check
	// MaxParallel bounds passes running at the same time (all jobs).
	MaxParallel int
	// Jitter is the fraction of Interval randomly added or removed (0..0.5).
	Jitter float64
	// Grace is how long Run waits for in-flight passes after shutdown
	// before cancelling them.
	Grace time.Duration
	// Now is the wall clock (tests); nil means time.Now.
	Now func() time.Time
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,47}$`)

// Bounds (engineering limits, not SLA values).
const (
	MinInterval = 100 * time.Millisecond
	MaxInterval = 24 * time.Hour
	MaxTimeout  = 30 * time.Minute
	MaxParallel = 16
)

func (c Config) validate() error {
	if c.Logger == nil {
		return fmt.Errorf("%w: logger required", ErrInvalid)
	}
	if len(c.Specs) == 0 {
		return fmt.Errorf("%w: no job registered", ErrInvalid)
	}
	if c.MaxParallel < 1 || c.MaxParallel > MaxParallel {
		return fmt.Errorf("%w: MaxParallel must be 1..%d", ErrInvalid, MaxParallel)
	}
	if c.Jitter < 0 || c.Jitter > 0.5 || c.Grace < 0 || c.Grace > MaxTimeout {
		return fmt.Errorf("%w: jitter / grace", ErrInvalid)
	}
	seen := map[string]bool{}
	for _, s := range c.Specs {
		if s.Job == nil || s.Gate == nil {
			return fmt.Errorf("%w: job and gate required", ErrInvalid)
		}
		n := s.Job.Name()
		if !namePattern.MatchString(n) || seen[n] {
			return fmt.Errorf("%w: job name %q", ErrInvalid, n)
		}
		seen[n] = true
		if s.Interval < MinInterval || s.Interval > MaxInterval || s.Timeout <= 0 || s.Timeout > MaxTimeout ||
			s.MaxBackoff < s.Interval || s.MaxBackoff > MaxInterval {
			return fmt.Errorf("%w: job %s interval / timeout / backoff out of bounds", ErrInvalid, n)
		}
	}
	for _, ch := range c.Checks {
		if ch.Fn == nil || !namePattern.MatchString(ch.Name) {
			return fmt.Errorf("%w: readiness check %q", ErrInvalid, ch.Name)
		}
	}
	return nil
}

// Status is the observable state of one job.
type Status struct {
	Job                 string
	State               string // idle, running, paused, blocked, failing, stopped
	Reason              string
	Runs, Failures      int64
	ConsecutiveFailures int
	LastStart, LastEnd  time.Time
	LastSuccess         time.Time
	LastError           string
}

type jobState struct {
	spec Spec
	mu   sync.Mutex
	st   Status
}

// Runtime runs registered jobs.
type Runtime struct {
	cfg     Config
	jobs    []*jobState
	metrics *Metrics
	sem     chan struct{}
	now     func() time.Time
	started time.Time
	mu      sync.Mutex
	running bool
	stopped bool
}

// New validates cfg and returns a runtime.
func New(cfg Config) (*Runtime, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if cfg.Instance == "" {
		cfg.Instance = defaultInstance()
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	rt := &Runtime{cfg: cfg, metrics: NewMetrics(), sem: make(chan struct{}, cfg.MaxParallel), now: now}
	for _, s := range cfg.Specs {
		rt.jobs = append(rt.jobs, &jobState{spec: s, st: Status{Job: s.Job.Name(), State: "idle"}})
	}
	return rt, nil
}

func defaultInstance() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		h = "worker"
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	h = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(h))
	if len(h) > 40 {
		h = h[:40]
	}
	return h + "-" + hex.EncodeToString(b[:])
}

// Instance returns the worker instance id.
func (rt *Runtime) Instance() string { return rt.cfg.Instance }

// Metrics returns the metric registry.
func (rt *Runtime) Metrics() *Metrics { return rt.metrics }

// Statuses returns a snapshot of every job.
func (rt *Runtime) Statuses() []Status {
	out := make([]Status, 0, len(rt.jobs))
	for _, j := range rt.jobs {
		j.mu.Lock()
		out = append(out, j.st)
		j.mu.Unlock()
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Job < out[b].Job })
	return out
}

// Run runs every job until ctx is done, then waits up to Grace for passes in
// flight (cancelling them after it). It returns ctx's error.
func (rt *Runtime) Run(ctx context.Context) error {
	rt.mu.Lock()
	if rt.running || rt.stopped {
		rt.mu.Unlock()
		return fmt.Errorf("%w: runtime already started", ErrInvalid)
	}
	rt.running = true
	rt.started = rt.now()
	rt.mu.Unlock()
	rt.cfg.Logger.Info("worker started", "instance", rt.cfg.Instance, "jobs", len(rt.jobs))

	// runCtx outlives ctx by Grace so in-flight passes can finish.
	runCtx, cancelRuns := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelRuns()
	var wg sync.WaitGroup
	for _, j := range rt.jobs {
		wg.Add(1)
		go func(j *jobState) {
			defer wg.Done()
			rt.loop(ctx, runCtx, j)
		}(j)
	}
	<-ctx.Done()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(rt.cfg.Grace):
		rt.cfg.Logger.Warn("worker grace period elapsed; cancelling in-flight passes", "instance", rt.cfg.Instance)
		cancelRuns()
		<-done
	}
	rt.mu.Lock()
	rt.running, rt.stopped = false, true
	rt.mu.Unlock()
	for _, j := range rt.jobs {
		j.mu.Lock()
		j.st.State = "stopped"
		j.mu.Unlock()
		rt.metrics.setState(j.spec.Job.Name(), "stopped")
	}
	rt.cfg.Logger.Info("worker stopped", "instance", rt.cfg.Instance)
	return ctx.Err()
}

func (rt *Runtime) delay(j *jobState) time.Duration {
	j.mu.Lock()
	fails := j.st.ConsecutiveFailures
	j.mu.Unlock()
	d := j.spec.Interval
	for i := 0; i < fails && d < j.spec.MaxBackoff; i++ {
		d *= 2
	}
	if d > j.spec.MaxBackoff {
		d = j.spec.MaxBackoff
	}
	if rt.cfg.Jitter > 0 {
		span := float64(d) * rt.cfg.Jitter
		d += time.Duration((mrand.Float64()*2 - 1) * span)
	}
	if d < MinInterval {
		d = MinInterval
	}
	return d
}

func (rt *Runtime) loop(stop, runCtx context.Context, j *jobState) {
	// The first pass waits a jittered fraction of the interval so several
	// instances started together do not poll in lockstep.
	first := time.Duration(0)
	if rt.cfg.Jitter > 0 {
		first = time.Duration(mrand.Float64() * rt.cfg.Jitter * float64(j.spec.Interval))
	}
	timer := time.NewTimer(first)
	defer timer.Stop()
	for {
		select {
		case <-stop.Done():
			return
		case <-timer.C:
		}
		rt.pass(stop, runCtx, j)
		timer.Reset(rt.delay(j))
	}
}

// RunOnce runs one pass of the named job synchronously (tests and the
// operator "once" command). It honours the gate.
func (rt *Runtime) RunOnce(ctx context.Context, name string) (Status, error) {
	for _, j := range rt.jobs {
		if j.spec.Job.Name() == name {
			rt.pass(ctx, ctx, j)
			j.mu.Lock()
			defer j.mu.Unlock()
			return j.st, nil
		}
	}
	return Status{}, fmt.Errorf("%w: unknown job %q", ErrInvalid, name)
}

func runID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "wrk-" + hex.EncodeToString(b[:])
}

const maxErrLen = 300

func sanitize(err error) string {
	s := err.Error()
	if len(s) > maxErrLen {
		s = s[:maxErrLen]
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 {
			return ' '
		}
		return r
	}, s)
}

func (rt *Runtime) pass(stop, runCtx context.Context, j *jobState) {
	name := j.spec.Job.Name()
	log := rt.cfg.Logger.With("job", name, "instance", rt.cfg.Instance)
	gctx, gcancel := context.WithTimeout(stop, j.spec.Timeout)
	state, reason := j.spec.Gate(gctx)
	gcancel()
	if state != Enabled {
		label := state.String()
		if state != Paused && state != Blocked {
			label, reason = "blocked", "INVALID_GATE_STATE"
		}
		j.mu.Lock()
		changed := j.st.State != label || j.st.Reason != reason
		j.st.State, j.st.Reason = label, reason
		j.mu.Unlock()
		rt.metrics.setState(name, label)
		if changed {
			log.Warn("job not running", "state", label, "reason", reason)
		}
		return
	}
	select {
	case rt.sem <- struct{}{}:
	case <-stop.Done():
		return
	}
	defer func() { <-rt.sem }()
	id := runID()
	start := rt.now()
	j.mu.Lock()
	j.st.State, j.st.Reason, j.st.LastStart = "running", "", start
	j.mu.Unlock()
	rt.metrics.setState(name, "running")
	rt.metrics.Inc("job_running", name, 1)

	ctx, cancel := context.WithTimeout(runCtx, j.spec.Timeout)
	r := &Run{ID: id, job: name, metrics: rt.metrics}
	err := safeRun(ctx, j.spec.Job, r)
	cancel()
	end := rt.now()
	rt.metrics.Inc("job_running", name, -1)

	j.mu.Lock()
	j.st.Runs++
	j.st.LastEnd = end
	if err != nil {
		j.st.Failures++
		j.st.ConsecutiveFailures++
		j.st.LastError = sanitize(err)
		j.st.State = "failing"
	} else {
		j.st.ConsecutiveFailures = 0
		j.st.LastError = ""
		j.st.LastSuccess = end
		j.st.State = "idle"
	}
	st := j.st
	j.mu.Unlock()
	rt.metrics.setState(name, st.State)
	rt.metrics.Set("job_last_duration_seconds", name, end.Sub(start).Seconds())
	if err != nil {
		rt.metrics.Add("job_runs_failed_total", name, 1)
		if errors.Is(err, context.DeadlineExceeded) {
			rt.metrics.Add("job_timeouts_total", name, 1)
		}
		log.Error("job pass failed", "run_id", id, "consecutive_failures", st.ConsecutiveFailures, "error", st.LastError)
		return
	}
	rt.metrics.Add("job_runs_succeeded_total", name, 1)
	rt.metrics.Set("job_last_success_timestamp_seconds", name, float64(end.Unix()))
	log.Debug("job pass completed", "run_id", id, "duration_ms", end.Sub(start).Milliseconds())
}

// errPanic reports a recovered panic (a programming error); the pass fails.
var errPanic = errors.New("worker: job panicked")

func safeRun(ctx context.Context, j Job, r *Run) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%w: %v", errPanic, p)
		}
	}()
	return j.Run(ctx, r)
}

// Run is the per-pass handle a job reports through.
type Run struct {
	ID      string
	job     string
	metrics *Metrics
}

// Job returns the job name.
func (r *Run) Job() string { return r.job }

// Count adds n to the job counter crm_worker_<name>_total{job}.
func (r *Run) Count(name string, n int) {
	if n != 0 {
		r.metrics.Add(name+"_total", r.job, float64(n))
	}
}

// Gauge sets the job gauge crm_worker_<name>{job}.
func (r *Run) Gauge(name string, v float64) { r.metrics.Set(name, r.job, v) }
