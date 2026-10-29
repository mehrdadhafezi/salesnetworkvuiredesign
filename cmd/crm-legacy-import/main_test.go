package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

const t0 = "2026-10-29T08:00:00Z"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func call(t *testing.T, e func(string) string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, e, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"nope"}, {"specs", "x"}, {"validate"}, {"load", "--force", "x"}, {"reconcile"}} {
		if code, _, stderr := call(t, env(nil), args...); code != 2 || !strings.Contains(stderr, "usage") {
			t.Errorf("%v: code %d stderr %q", args, code, stderr)
		}
	}
}

func TestSpecsInspectValidate(t *testing.T) {
	code, out, _ := call(t, env(nil), "specs")
	var specs []spec.Spec
	if code != 0 || json.Unmarshal([]byte(out), &specs) != nil || len(specs) != len(spec.Default().IDs()) {
		t.Fatalf("specs: code %d, %d specs", code, len(specs))
	}
	dir := legacytest.Write(t, "synthetic-snapshot-cli-1", t0, legacytest.Rows())
	if code, out, _ := call(t, env(nil), "inspect", dir); code != 0 || !strings.Contains(out, `"classification": "SENSITIVE"`) {
		t.Fatalf("inspect: %d %s", code, out)
	}
	code, out, _ = call(t, env(nil), "validate", dir)
	if code != 0 || !strings.Contains(out, `"outcome": "SUCCESS"`) {
		t.Fatalf("validate: %d %s", code, out)
	}
	if strings.Contains(out, "Dev User") || strings.Contains(out, "example.invalid") {
		t.Fatal("reports must not contain row values")
	}
	f := filepath.Join(dir, snapshot.EntityFile("legacy.hr_team.v1"))
	b, _ := os.ReadFile(f)
	_ = os.WriteFile(f, bytes.Replace(b, []byte("syn-"), []byte("syx-"), 1), 0o600)
	// Frozen result contract: a mix of FAILED and SUCCESS aggregates to
	// PARTIAL with the per-entity FAILED preserved (never SUCCESS).
	code, out, _ = call(t, env(nil), "validate", dir)
	if code != 3 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"status": "FAILED"`) {
		t.Fatalf("tampered validate: %d %s", code, out)
	}
	if code, out, _ := call(t, env(nil), "inspect", t.TempDir()); code != 3 || !strings.Contains(out, "CONTRACT_INVALID") {
		t.Fatalf("inspect empty dir: %d %s", code, out)
	}
}

func TestStagingDSNRules(t *testing.T) {
	dir := legacytest.Write(t, "synthetic-snapshot-cli-2", t0, legacytest.Rows())
	if code, _, stderr := call(t, env(nil), "load", dir); code != 2 || !strings.Contains(stderr, EnvStagingURL) {
		t.Fatalf("no DSN: %d %q", code, stderr)
	}
	same := "postgres://x@127.0.0.1:1/x?sslmode=disable"
	for _, other := range []string{db.EnvDatabaseURL, migrations.EnvMigrateURL} {
		if code, _, stderr := call(t, env(map[string]string{EnvStagingURL: same, other: same}), "reconcile", "synthetic-snapshot-cli-2"); code != 2 || !strings.Contains(stderr, "must not reuse") {
			t.Fatalf("reuse of %s: %d %q", other, code, stderr)
		}
	}
	if code, _, _ := call(t, env(map[string]string{EnvStagingURL: "postgres://nobody@127.0.0.1:1/x?sslmode=disable&connect_timeout=1"}), "load", dir); code != 1 {
		t.Fatalf("unreachable: %d", code)
	}
}

func TestLoadAndReconcileAgainstDisposablePostgres(t *testing.T) {
	e := pgtest.New(t)
	if _, err := migrations.Up(context.Background(), e.DSN, integrationmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	dir := legacytest.Write(t, "synthetic-snapshot-cli-3", t0, legacytest.Rows())

	// The provisioning superuser is not an integration-dedicated login: refused.
	if code, _, stderr := call(t, env(map[string]string{EnvStagingURL: e.DSN}), "load", dir); code != 1 || !strings.Contains(stderr, "session role") {
		t.Fatalf("superuser DSN: %d %q", code, stderr)
	}
	login := env(map[string]string{EnvStagingURL: e.LoginDSN(t, integration.Module)})
	code, out, stderr := call(t, login, "load", dir)
	if code != 0 || !strings.Contains(out, `"outcome": "SUCCESS"`) || !strings.Contains(out, `"batch_state": "LOADED"`) {
		t.Fatalf("load: %d %s %s", code, out, stderr)
	}
	if strings.Contains(out, "Dev User") || strings.Contains(out, "TEST-PHONE") {
		t.Fatal("load report must not contain row values")
	}
	if code, out, _ := call(t, login, "load", dir); code != 0 || !strings.Contains(out, `"replay": true`) {
		t.Fatalf("replay: %d %s", code, out)
	}
	if code, out, _ := call(t, login, "reconcile", "synthetic-snapshot-cli-3"); code != 0 || !strings.Contains(out, `"outcome": "SUCCESS"`) {
		t.Fatalf("reconcile: %d %s", code, out)
	}
	if code, out, _ := call(t, login, "reconcile", "synthetic-snapshot-missing"); code != 3 || !strings.Contains(out, `"outcome": "FAILED"`) {
		t.Fatalf("reconcile missing: %d %s", code, out)
	}

	// Unclassified data needs the explicit acknowledgement.
	udir := filepath.Join(t.TempDir(), "p")
	h := legacytest.Header("synthetic-snapshot-cli-4", t0)
	h.DataOrigin = snapshot.OriginUnclassified
	if _, err := snapshot.Write(udir, spec.Default(), h, legacytest.Rows()[:2]); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := call(t, login, "load", udir); code != 3 || !strings.Contains(out, `"outcome": "BLOCKED"`) {
		t.Fatalf("unclassified: %d %s", code, out)
	}
	if code, out, _ := call(t, login, "load", "--allow-unclassified-data", udir); code != 0 || !strings.Contains(out, `"outcome": "SUCCESS"`) {
		t.Fatalf("acknowledged: %d %s", code, out)
	}
}
