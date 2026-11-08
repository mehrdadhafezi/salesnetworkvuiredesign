// Command crm-worker runs the CRM background jobs (R2): outbox relays,
// receipt finalisation, unbound-upload expiry reporting, the legacy
// write-back intake / delivery and provider-operation reconciliation.
//
// Usage:
//
//	crm-worker check         validate configuration and attestation; print the redacted plan
//	crm-worker run           run the enabled jobs until SIGINT / SIGTERM
//	crm-worker once <job>    run one pass of one job (rehearsal / diagnostics)
//
// Safety model (fail closed):
//
//   - CRM_WORKER_ENV must be declared (development | staging | production)
//     and the feature-flag database must attest exactly that environment
//     (flag worker.environment.<env>); nothing is inferred from host names.
//   - Every job is off unless listed in CRM_WORKER_JOBS AND its runtime
//     switch worker.job.<job> is on (absent = paused). A paused or blocked
//     job leaves its work queued and visible; it never reports success.
//   - Each job uses dedicated owner logins (CRM_WORKER_<MODULE>_DATABASE_URL),
//     never a shared or superuser connection; DSNs are never printed.
//   - External adapters default to disabled. The legacy write-back adapter
//     "compat" refuses to start (no approved contract, E-3); no verified
//     provider adapter exists. Delivery and reconciliation jobs are BLOCKED
//     and a backlog job reports what waits. No synthetic adapter is linked
//     into this binary.
//   - Unbound-upload purge (irreversible) is BLOCKED by D-44 outside
//     development; the default report mode changes nothing.
//   - Workers never approve, reject, post, refund or correct anything, and
//     never dispatch a provider order.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

// pools opens one pool per owner module plus the feature-flag pool.
type pools struct {
	runners map[owner.Module]*db.Runner
	flags   *db.Runner
	all     []*db.Pool
	checks  []worker.Check
}

func (p *pools) close() {
	for _, x := range p.all {
		x.Close()
	}
}

func openPools(ctx context.Context, cfg Config) (*pools, error) {
	p := &pools{runners: map[owner.Module]*db.Runner{}}
	open := func(m owner.Module, url string) (*db.Runner, error) {
		dc := db.DefaultConfig(url)
		dc.ApplicationName = "crm-worker"
		dc.MaxConns = int32(cfg.MaxParallel + 2)
		pool, err := db.Open(ctx, dc)
		if err != nil {
			// The DSN is never echoed: only the owner it was meant for.
			return nil, fmt.Errorf("crm-worker: %s database unavailable", m)
		}
		p.all = append(p.all, pool)
		p.checks = append(p.checks, worker.Check{Name: "db_" + string(m), Fn: pool.Ping})
		return pool.Runner(m)
	}
	var err error
	if p.flags, err = open(owner.FeatureFlags, cfg.FlagsURL); err != nil {
		p.close()
		return nil, err
	}
	mods := make([]owner.Module, 0, len(cfg.ModuleURLs))
	for m := range cfg.ModuleURLs {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(a, b int) bool { return mods[a] < mods[b] })
	for _, m := range mods {
		if p.runners[m], err = open(m, cfg.ModuleURLs[m]); err != nil {
			p.close()
			return nil, err
		}
	}
	return p, nil
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	log := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if len(args) == 0 || (args[0] != "check" && args[0] != "run" && args[0] != "once") || (args[0] == "once") != (len(args) == 2) ||
		args[0] != "once" && len(args) != 1 {
		fmt.Fprintln(stderr, "usage: crm-worker check | run | once <job>")
		return 2
	}
	cfg, err := ConfigFromEnv(getenv)
	if err != nil {
		log.Error("configuration refused", "error", err.Error())
		return 1
	}
	p, err := openPools(ctx, cfg)
	if err != nil {
		log.Error("startup refused", "error", err.Error())
		return 1
	}
	defer p.close()
	comp, err := compose(ctx, cfg, p.runners, p.flags, clock.System{}, log, overrides{})
	if err != nil {
		log.Error("startup refused", "error", err.Error())
		return 1
	}
	rt, err := worker.New(worker.Config{Logger: log, Specs: comp.specs, Checks: p.checks, MaxParallel: cfg.MaxParallel, Jitter: 0.2,
		Grace: cfg.Grace})
	if err != nil {
		log.Error("startup refused", "error", err.Error())
		return 1
	}
	switch args[0] {
	case "check":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(comp.plan); err != nil {
			return 1
		}
		return 0
	case "once":
		st, err := rt.RunOnce(ctx, args[1])
		if err != nil {
			log.Error("once refused", "error", err.Error())
			return 2
		}
		_ = json.NewEncoder(stdout).Encode(map[string]any{"job": st.Job, "state": st.State, "reason": st.Reason, "last_error": st.LastError})
		if st.State != "idle" {
			return 1
		}
		return 0
	}
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Error("health listener refused", "error", err.Error())
		return 1
	}
	srv := &http.Server{Handler: rt.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("health listener failed", "error", err.Error())
		}
	}()
	log.Info("crm-worker running", "environment", string(cfg.Environment), "jobs", cfg.Jobs, "health_addr", ln.Addr().String(),
		"instance", rt.Instance())
	_ = rt.Run(ctx)
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(sctx)
	return 0
}
