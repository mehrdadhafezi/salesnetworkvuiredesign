package observability

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// R3: every crm_* metric an alert or dashboard uses is actually emitted by
// the code, and every alert carries the proposed label, a severity, an
// owner ROLE (never a person), containment and a runbook anchor that exists.
// promtool validation (config, rules, alerts.promtool-test.yml) and a real
// Prometheus scrape of the staging stack run in release-ci.yml.

type rule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type rules struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []rule `yaml:"rules"`
	} `yaml:"groups"`
}

var (
	metricRef = regexp.MustCompile(`\bcrm_[a-z0-9_]+`)
	gaugeCall = regexp.MustCompile(`\.Gauge\("([a-z0-9_]+)"`)
	countCall = regexp.MustCompile(`\.Count\("([a-z0-9_]+)"`)
)

// emitted collects the metric names the code can emit.
func emitted(t *testing.T) map[string]bool {
	t.Helper()
	m := map[string]bool{}
	add := func(names ...string) {
		for _, n := range names {
			m[n] = true
		}
	}
	add("crm_api_requests_total", "crm_api_request_duration_seconds_bucket", "crm_api_request_duration_seconds_sum",
		"crm_api_request_duration_seconds_count", "crm_api_login_outcomes_total", "crm_api_inflight_requests", "crm_api_start_time_seconds",
		"crm_build_info", "crm_worker_feature_flags_enabled", "crm_worker_environment_attested", "crm_worker_ready", "crm_worker_check_up",
		"crm_worker_job_running", "crm_worker_job_last_duration_seconds", "crm_worker_job_runs_failed_total", "crm_worker_job_timeouts_total",
		"crm_worker_job_runs_succeeded_total", "crm_worker_job_last_success_timestamp_seconds")
	for _, s := range []string{"idle", "running", "paused", "blocked", "failing", "stopped"} {
		add("crm_worker_job_state_" + s)
	}
	for _, p := range []string{"crm_db_pool", "crm_worker_db_pool"} {
		for _, s := range []string{"max_conns", "total_conns", "idle_conns", "acquired_conns", "empty_acquire_total"} {
			add(p + "_" + s)
		}
	}
	for _, op := range []string{"backup", "backup_verify", "restore_rehearsal"} {
		add("crm_"+op+"_last_run_success", "crm_"+op+"_last_success_timestamp_seconds", "crm_"+op+"_last_run_timestamp_seconds")
	}
	add("crm_backup_last_duration_seconds", "crm_restore_rehearsal_last_recovery_seconds")
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, g := range gaugeCall.FindAllStringSubmatch(string(src), -1) {
			add("crm_worker_" + g[1])
		}
		for _, c := range countCall.FindAllStringSubmatch(string(src), -1) {
			add("crm_worker_" + c[1] + "_total")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAlertRulesReferenceEmittedMetrics(t *testing.T) {
	raw, err := os.ReadFile("alerts.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var r rules
	if err := yaml.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "remediation", "r3-observability-and-alerts.md"))
	if err != nil {
		t.Fatal(err)
	}
	known := emitted(t)
	seen := map[string]bool{}
	n := 0
	for _, g := range r.Groups {
		for _, a := range g.Rules {
			n++
			if a.Alert == "" || a.Expr == "" || seen[a.Alert] {
				t.Fatalf("alert name / expr missing or duplicate: %+v", a)
			}
			seen[a.Alert] = true
			if a.Labels["proposed"] != "true" || a.Labels["severity"] == "" {
				t.Fatalf("%s: proposed=true and a severity are required", a.Alert)
			}
			for _, k := range []string{"summary", "owner_role", "containment", "runbook"} {
				if strings.TrimSpace(a.Annotations[k]) == "" {
					t.Fatalf("%s: annotation %s missing", a.Alert, k)
				}
			}
			if !regexp.MustCompile(`^[a-z][a-z-]+$`).MatchString(a.Annotations["owner_role"]) {
				t.Fatalf("%s: owner_role must be a role slug, never a person", a.Alert)
			}
			anchor := a.Annotations["runbook"][strings.Index(a.Annotations["runbook"], "#")+1:]
			if !strings.Contains(strings.ToLower(string(doc)), "### "+anchor) {
				t.Fatalf("%s: runbook section %q missing in r3-observability-and-alerts.md", a.Alert, anchor)
			}
			for _, ref := range metricRef.FindAllString(a.Expr, -1) {
				if !known[ref] {
					t.Fatalf("%s references %s, which no code emits", a.Alert, ref)
				}
			}
		}
	}
	if n < 20 {
		t.Fatalf("expected the full proposed alert set, got %d", n)
	}
	dash, err := os.ReadFile("dashboard.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range metricRef.FindAllString(string(dash), -1) {
		if !known[ref] {
			t.Fatalf("dashboard references %s, which no code emits", ref)
		}
	}
}

// R3: the worker labels per-job series with `job`; the scrape must honour it
// (otherwise Prometheus renames it exported_job and `on (job)` matching in
// the rules joins unrelated worker jobs).
func TestWorkerScrapeHonoursJobLabel(t *testing.T) {
	src, err := os.ReadFile("prometheus.yml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		ScrapeConfigs []struct {
			JobName     string `yaml:"job_name"`
			HonorLabels bool   `yaml:"honor_labels"`
		} `yaml:"scrape_configs"`
	}
	if err := yaml.Unmarshal(src, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, sc := range cfg.ScrapeConfigs {
		if sc.JobName == "crm-worker" {
			if !sc.HonorLabels {
				t.Fatal("crm-worker scrape must set honor_labels: true")
			}
			return
		}
	}
	t.Fatal("crm-worker scrape config missing")
}
