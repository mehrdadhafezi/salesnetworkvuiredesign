package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	reportingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/migrations"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func call(getenv func(string) string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, getenv, &out, &errb)
	return code, out.String(), errb.String()
}

func TestUsageAndDSNRules(t *testing.T) {
	if code, _, stderr := call(env(nil)); code != 2 || !strings.Contains(stderr, "usage") {
		t.Fatalf("usage: %d %q", code, stderr)
	}
	if code, _, stderr := call(env(nil), "rebuild"); code != 2 || !strings.Contains(stderr, EnvReportingURL) {
		t.Fatalf("missing reporting DSN: %d %q", code, stderr)
	}
	same := "postgres://x@127.0.0.1:1/x?sslmode=disable"
	if code, _, stderr := call(env(map[string]string{EnvReportingURL: same, EnvCasesURL: same}), "rebuild"); code != 2 || !strings.Contains(stderr, "must not reuse") {
		t.Fatalf("DSN reuse: %d %q", code, stderr)
	}
}

// With every owner source missing, the run is PARTIAL (exit 3), never SUCCESS.
func TestRebuildWithMissingSourcesIsPartial(t *testing.T) {
	e := pgtest.New(t)
	if _, err := migrations.Up(context.Background(), e.DSN, reportingmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := call(env(map[string]string{EnvReportingURL: e.LoginDSN(t, reporting.Module)}), "rebuild")
	if code != 3 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"status": "MISSING"`) {
		t.Fatalf("rebuild: %d %s %s", code, out, stderr)
	}
}
