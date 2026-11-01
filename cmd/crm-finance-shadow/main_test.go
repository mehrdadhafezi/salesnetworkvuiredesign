package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func call(getenv func(string) string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, getenv, &out, &errb)
	return code, out.String(), errb.String()
}

func TestUsageAndDSNRules(t *testing.T) {
	for _, args := range [][]string{nil, {"post"}, {"rebuild", "--snapshot-id"}, {"approve"}, {"rebuild", "--force"}} {
		if code, _, stderr := call(env(nil), args...); code != 2 || !strings.Contains(stderr, "usage") {
			t.Fatalf("%v: %d %q", args, code, stderr)
		}
	}
	if code, _, stderr := call(env(nil), "rebuild"); code != 2 || !strings.Contains(stderr, EnvFinanceURL) {
		t.Fatalf("missing finance DSN: %d %q", code, stderr)
	}
	same := "postgres://x@127.0.0.1:1/x?sslmode=disable"
	if code, _, stderr := call(env(map[string]string{EnvFinanceURL: same, EnvBillingURL: same}), "rebuild"); code != 2 || !strings.Contains(stderr, "must not reuse") {
		t.Fatalf("DSN reuse: %d %q", code, stderr)
	}
	if code, _, stderr := call(env(map[string]string{EnvFinanceURL: same}), "rebuild", "--snapshot-id", "syn"); code != 2 || !strings.Contains(stderr, EnvStagingURL) {
		t.Fatalf("snapshot without staging: %d %q", code, stderr)
	}
}

// With every source missing, the run is PARTIAL (exit 3), never SUCCESS.
func TestRebuildWithMissingSourcesIsPartial(t *testing.T) {
	e := pgtest.New(t)
	if _, err := migrations.Up(context.Background(), e.DSN, financemigrations.Source()); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := call(env(map[string]string{EnvFinanceURL: e.LoginDSN(t, finance.Module)}), "rebuild")
	if code != 3 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"status": "MISSING"`) || !strings.Contains(out, `"record_kind": "SHADOW"`) {
		t.Fatalf("rebuild: %d %s %s", code, out, stderr)
	}
	// Same (empty) inputs again: a replay of the same run, nothing written.
	code2, out2, _ := call(env(map[string]string{EnvFinanceURL: e.LoginDSN(t, finance.Module)}), "rebuild")
	if code2 != 3 || !strings.Contains(out2, `"replayed": true`) {
		t.Fatalf("replay: %d %s", code2, out2)
	}
}
