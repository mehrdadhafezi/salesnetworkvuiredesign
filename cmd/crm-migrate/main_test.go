package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"down"}, {"up", "extra"}} {
		var out, errOut bytes.Buffer
		if code := run(context.Background(), args, env(nil), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Fatalf("args %v: code=%d stderr=%q", args, code, errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"up"}, env(nil), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), migrations.EnvMigrateURL) {
		t.Fatalf("missing DSN: code=%d stderr=%q", code, errOut.String())
	}
	if code := run(context.Background(), []string{"status"}, env(map[string]string{migrations.EnvMigrateURL: "postgres://nobody@127.0.0.1:1/x?sslmode=disable&connect_timeout=1"}), &out, &errOut); code != 1 {
		t.Fatalf("unreachable status code = %d", code)
	}
	if code := run(context.Background(), []string{"up"}, env(map[string]string{migrations.EnvMigrateURL: "postgres://nobody@127.0.0.1:1/x?sslmode=disable&connect_timeout=1"}), &out, &errOut); code != 1 {
		t.Fatalf("unreachable up code = %d", code)
	}
}

func TestUpThenStatus(t *testing.T) {
	dsn := pgtest.CreateDatabase(t)
	e := env(map[string]string{migrations.EnvMigrateURL: dsn})
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"up"}, e, &out, &errOut); code != 0 {
		t.Fatalf("up code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "applied 00001 00001_platform_audit.sql") || !strings.Contains(out.String(), "applied 00002 00002_platform_featureflags.sql") ||
		!strings.Contains(out.String(), "applied 00001 00001_integration_staging.sql") || !strings.Contains(out.String(), "applied 00002 00002_integration_staging_guards.sql") ||
		!strings.Contains(out.String(), "applied 00001 00001_identity.sql") || !strings.Contains(out.String(), "applied 00001 00001_organization.sql") {
		t.Fatalf("up output = %q", out.String())
	}
	out.Reset()
	if code := run(context.Background(), []string{"up"}, e, &out, &errOut); code != 0 || !strings.Contains(out.String(), "no pending migrations") {
		t.Fatalf("second up code=%d out=%q", code, out.String())
	}
	out.Reset()
	if code := run(context.Background(), []string{"status"}, e, &out, &errOut); code != 0 || strings.Count(out.String(), "applied") != 6 || strings.Contains(out.String(), "pending") {
		t.Fatalf("status code=%d out=%q", code, out.String())
	}
}
