package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func call(getenv func(string) string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, getenv, &out, &errb)
	return code, out.String(), errb.String()
}

// No approve, post, credit, settle, reverse or recalculate command exists;
// there is no default rounding; every unknown command or flag is a usage
// error.
func TestUsageAndDSNRules(t *testing.T) {
	for _, args := range [][]string{nil, {"run"}, {"run", "--snapshot-id", "s"}, {"run", "--snapshot-id", "s", "--rounding", "BANKERS"},
		{"approve", "--snapshot-id", "s", "--rounding", "NONE"}, {"post", "--snapshot-id", "s", "--rounding", "NONE"},
		{"credit", "--snapshot-id", "s", "--rounding", "NONE"}, {"settle", "--snapshot-id", "s", "--rounding", "NONE"},
		{"reverse", "--snapshot-id", "s", "--rounding", "NONE"}, {"backfill", "--snapshot-id", "s", "--rounding", "NONE"},
		{"run", "--snapshot-id", "s", "--rounding", "NONE", "--force"}, {"run", "--snapshot-id", "", "--rounding", "NONE"},
		{"run", "--snapshot-id", "s", "--post", "yes"}} {
		if code, _, stderr := call(env(nil), args...); code != 2 || !strings.Contains(stderr, "usage") {
			t.Fatalf("%v: %d %q", args, code, stderr)
		}
	}
	if code, _, stderr := call(env(nil), "run", "--snapshot-id", "s", "--rounding", "NONE"); code != 2 || !strings.Contains(stderr, EnvCommissionURL) {
		t.Fatalf("missing commission DSN: %d %q", code, stderr)
	}
	base := map[string]string{EnvCommissionURL: "postgres://c@127.0.0.1:1/x", EnvStagingURL: "postgres://s@127.0.0.1:1/x",
		EnvIdentityURL: "postgres://i@127.0.0.1:1/x", EnvBillingURL: "postgres://c@127.0.0.1:1/x"}
	if code, _, stderr := call(env(base), "run", "--snapshot-id", "s", "--rounding", "NONE"); code != 2 || !strings.Contains(stderr, "must not reuse") {
		t.Fatalf("DSN reuse: %d %q", code, stderr)
	}
}
