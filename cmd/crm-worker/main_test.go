package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	catalogmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/migrations"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

func getenv(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func baseEnv() map[string]string {
	return map[string]string{EnvEnvironment: "staging", EnvJobs: "receipt_finalise", EnvFeatureFlagsURL: "postgres://flags", EnvEvidenceRoot: "/srv/evidence",
		"CRM_WORKER_PAYMENTS_DATABASE_URL": "postgres://payments"}
}

// 22 + configuration mistakes fail startup; external adapters default to
// disabled and cannot be switched to anything unverified.
func TestConfigFailsClosed(t *testing.T) {
	c, err := ConfigFromEnv(getenv(baseEnv()))
	if err != nil || c.Environment != Staging || c.WritebackAdapter != AdapterDisabled || c.ProviderAdapter != AdapterDisabled ||
		c.UploadExpiryMode != ExpiryReport || c.Batch != 50 || c.Interval != 30*time.Second {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	bad := map[string]func(map[string]string){
		"no environment":            func(m map[string]string) { delete(m, EnvEnvironment) },
		"inferred environment":      func(m map[string]string) { m[EnvEnvironment] = "localhost" },
		"no job":                    func(m map[string]string) { m[EnvJobs] = " , " },
		"unknown job":               func(m map[string]string) { m[EnvJobs] = "post_ledger" },
		"missing owner login":       func(m map[string]string) { delete(m, "CRM_WORKER_PAYMENTS_DATABASE_URL") },
		"shared login":              func(m map[string]string) { m["CRM_WORKER_PAYMENTS_DATABASE_URL"] = m[EnvFeatureFlagsURL] },
		"no flag database":          func(m map[string]string) { delete(m, EnvFeatureFlagsURL) },
		"no evidence root":          func(m map[string]string) { delete(m, EnvEvidenceRoot) },
		"purge outside development": func(m map[string]string) { m[EnvJobs] = "upload_expiry"; m[EnvUploadExpiryMode] = "purge" },
		"unknown expiry mode":       func(m map[string]string) { m[EnvUploadExpiryMode] = "delete" },
		"provider adapter":          func(m map[string]string) { m[EnvProviderAdapter] = "biawin" },
		"synthetic provider":        func(m map[string]string) { m[EnvProviderAdapter] = "synthetic" },
		"unknown writeback adapter": func(m map[string]string) { m[EnvWritebackAdapter] = "wordpress" },
		"interval out of bounds":    func(m map[string]string) { m[EnvInterval] = "1ms" },
		"batch out of bounds":       func(m map[string]string) { m[EnvBatch] = "100000" },
		"bad address":               func(m map[string]string) { m[EnvAddr] = "nonsense" },
	}
	for name, mutate := range bad {
		m := baseEnv()
		mutate(m)
		if _, err := ConfigFromEnv(getenv(m)); !errors.Is(err, ErrConfig) {
			t.Fatalf("%s accepted", name)
		}
	}
	// A blocked external job always brings its backlog job.
	m := baseEnv()
	m[EnvJobs] = "writeback_deliver,provider_reconcile"
	m["CRM_WORKER_INTEGRATION_DATABASE_URL"] = "postgres://integration"
	c, err = ConfigFromEnv(getenv(m))
	if err != nil || strings.Join(c.Jobs, ",") != "provider_backlog,provider_reconcile,writeback_backlog,writeback_deliver" {
		t.Fatalf("backlog companions: %v %v", c.Jobs, err)
	}
	// The printable plan never contains a DSN.
	p, _ := json.Marshal(c.Plan())
	if strings.Contains(string(p), "postgres://") {
		t.Fatalf("plan leaks a DSN: %s", p)
	}
	// Purge is only for development.
	m = baseEnv()
	m[EnvEnvironment], m[EnvJobs], m[EnvUploadExpiryMode] = "development", "upload_expiry", "purge"
	if _, err := ConfigFromEnv(getenv(m)); err != nil {
		t.Fatalf("development purge: %v", err)
	}
}

type dbEnv struct {
	*pgtest.Env
	urls map[string]string
}

func setupDB(t *testing.T, attest Environment) *dbEnv {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	for _, s := range []migrations.Source{integrationmigrations.Source(), organizationmigrations.Source(), casesmigrations.Source(), billingmigrations.Source(),
		paymentsmigrations.Source(), biawinmigrations.Source(), fulfillmentmigrations.Source(), catalogmigrations.Source()} {
		if _, err := migrations.Up(ctx, e.DSN, s); err != nil {
			t.Fatal(err)
		}
	}
	d := &dbEnv{Env: e, urls: map[string]string{}}
	for _, m := range []owner.Module{"payments", "integration", "cases", "organization", "biawin", "fulfillment", "catalog"} {
		d.urls[moduleEnv(m)] = e.LoginDSN(t, m)
	}
	d.urls[EnvFeatureFlagsURL] = e.LoginDSN(t, owner.FeatureFlags)
	if attest != "" {
		d.flag(t, EnvironmentFlag(attest), true)
	}
	return d
}

func (d *dbEnv) flag(t *testing.T, key string, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-08T09:00:00Z")
	if err := d.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: key, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

func (d *dbEnv) env(extra map[string]string) func(string) string {
	m := map[string]string{}
	for k, v := range d.urls {
		m[k] = v
	}
	for k, v := range extra {
		m[k] = v
	}
	return getenv(m)
}

// 22: the database must attest exactly the declared environment.
func TestAttestationAndCheck(t *testing.T) {
	d := setupDB(t, Staging)
	root := t.TempDir()
	jobs := "receipt_finalise,upload_expiry,writeback_intake_cases,writeback_intake_organization,writeback_intake_payments,writeback_deliver," +
		"provider_reconcile,fulfillment_intake"
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"check"}, d.env(map[string]string{EnvEnvironment: "production", EnvJobs: jobs, EnvEvidenceRoot: root}),
		&out, &errOut); code != 1 || !strings.Contains(errOut.String(), "attestation failed") {
		t.Fatalf("declared production on a staging database: code=%d %s", code, errOut.String())
	}
	// Two attested environments are as wrong as none.
	d.flag(t, EnvironmentFlag(Production), true)
	out.Reset()
	errOut.Reset()
	if code := run(context.Background(), []string{"check"}, d.env(map[string]string{EnvEnvironment: "staging", EnvJobs: jobs, EnvEvidenceRoot: root}),
		&out, &errOut); code != 1 {
		t.Fatalf("ambiguous attestation accepted: %s", errOut.String())
	}
	d.flag(t, EnvironmentFlag(Production), false)
	out.Reset()
	errOut.Reset()
	if code := run(context.Background(), []string{"check"}, d.env(map[string]string{EnvEnvironment: "staging", EnvJobs: jobs, EnvEvidenceRoot: root}),
		&out, &errOut); code != 0 {
		t.Fatalf("check: %s", errOut.String())
	}
	var plan Plan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Gates[JobWritebackDeliver] != ReasonWritebackDisabled || plan.Gates[JobProviderReconcile] != ReasonProviderDisabled ||
		!strings.Contains(plan.Gates[JobUploadExpiry], "D-44") || len(plan.Jobs) != 10 {
		t.Fatalf("plan: %+v", plan)
	}
	if strings.Contains(out.String(), "postgres://") || strings.Contains(errOut.String(), "postgres://") {
		t.Fatal("a DSN was printed")
	}
	// 21 / E-3: the compat adapter refuses to start (no approved contract).
	errOut.Reset()
	if code := run(context.Background(), []string{"check"}, d.env(map[string]string{EnvEnvironment: "staging", EnvJobs: "writeback_deliver",
		EnvWritebackAdapter: "compat"}), &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "no approved legacy write-back contract") {
		t.Fatalf("compat adapter: code=%d %s", code, errOut.String())
	}
}

// 19 + 20 + health: jobs are paused until switched on; blocked jobs stay
// blocked; the process serves health and stops gracefully.
func TestRunLifecycle(t *testing.T) {
	d := setupDB(t, Staging)
	getenv := d.env(map[string]string{EnvEnvironment: "staging", EnvJobs: "writeback_intake_cases,writeback_deliver", EnvAddr: "127.0.0.1:0",
		EnvInterval: "1s"})
	var out, errOut bytes.Buffer
	// once: the switch is off → paused (non-zero: nothing was done).
	if code := run(context.Background(), []string{"once", JobWritebackIntakeCases}, getenv, &out, &errOut); code != 1 ||
		!strings.Contains(out.String(), `"state":"paused"`) {
		t.Fatalf("paused once: code=%d %s %s", code, out.String(), errOut.String())
	}
	d.flag(t, JobFlag(JobWritebackIntakeCases), true)
	d.flag(t, JobFlag(JobWritebackDeliver), true)
	d.flag(t, JobFlag(JobWritebackBacklog), true)
	out.Reset()
	if code := run(context.Background(), []string{"once", JobWritebackIntakeCases}, getenv, &out, &errOut); code != 0 {
		t.Fatalf("enabled once: %s %s", out.String(), errOut.String())
	}
	out.Reset()
	if code := run(context.Background(), []string{"once", JobWritebackDeliver}, getenv, &out, &errOut); code != 1 ||
		!strings.Contains(out.String(), ReasonWritebackDisabled) {
		t.Fatalf("a switched-on delivery without an adapter must stay blocked: %s", out.String())
	}
	// run: serve, then stop on cancellation.
	ctx, stop := context.WithCancel(context.Background())
	logs := &syncBuf{}
	done := make(chan int, 1)
	go func() { done <- run(ctx, []string{"run"}, getenv, io.Discard, logs) }()
	addr := ""
	for i := 0; i < 100 && addr == ""; i++ {
		time.Sleep(50 * time.Millisecond)
		for _, line := range strings.Split(logs.String(), "\n") {
			var rec map[string]any
			if json.Unmarshal([]byte(line), &rec) == nil && rec["msg"] == "crm-worker running" {
				addr, _ = rec["health_addr"].(string)
			}
		}
	}
	if addr == "" {
		t.Fatalf("worker did not start: %s", logs.String())
	}
	resp, err := http.Get("http://" + addr + "/health/live")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("live: %v", err)
	}
	resp.Body.Close()
	time.Sleep(1500 * time.Millisecond)
	resp, err = http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `crm_worker_job_state_blocked{job="writeback_deliver"} 1`) ||
		!strings.Contains(string(body), `crm_worker_writeback_pending{job="writeback_backlog"} 0`) {
		t.Fatalf("metrics:\n%s", body)
	}
	stop()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("shutdown code %d", code)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("graceful shutdown timed out")
	}
	if strings.Contains(logs.String(), "postgres://") || strings.Contains(logs.String(), "password") {
		t.Fatal("logs must not contain credentials")
	}
}

// Synthetic adapters only in an attested development environment.
func TestSyntheticAdaptersOnlyInDevelopment(t *testing.T) {
	d := setupDB(t, Staging)
	cfg, err := ConfigFromEnv(d.env(map[string]string{EnvEnvironment: "staging", EnvJobs: "writeback_deliver"}))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := lwb.NewConnector(legacytest.NewSyntheticLegacy(), noUsers{})
	if err != nil {
		t.Fatal(err)
	}
	runners := map[owner.Module]*db.Runner{"integration": d.Runner(t, "integration")}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := compose(context.Background(), cfg, runners, d.Runner(t, owner.FeatureFlags), timeutil.FixedClock{}, log,
		overrides{connector: conn}); !errors.Is(err, ErrConfig) {
		t.Fatalf("a synthetic adapter in staging must be refused: %v", err)
	}
	dev := setupDB(t, Development)
	cfg.Environment = Development
	comp, err := compose(context.Background(), cfg, map[owner.Module]*db.Runner{"integration": dev.Runner(t, "integration")},
		dev.Runner(t, owner.FeatureFlags), timeutil.FixedClock{}, log, overrides{connector: conn})
	if err != nil {
		t.Fatal(err)
	}
	if comp.plan.Gates[JobWritebackDeliver] != "synthetic adapter (development only)" {
		t.Fatalf("plan: %+v", comp.plan.Gates)
	}
	if _, err := worker.New(worker.Config{Logger: log, Specs: comp.specs, MaxParallel: 1}); err != nil {
		t.Fatal(err)
	}
}

type noUsers struct{}

func (noUsers) UserKeys(context.Context, string, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (noUsers) ResolveUsers(context.Context, string, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}
func (s *syncBuf) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.buf.String() }
