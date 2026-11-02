package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	stagingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func call(getenv func(string) string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, getenv, &out, &errb)
	return code, out.String(), errb.String()
}

// No activation, provider, shipping or completion command exists; every
// unknown command or flag is a usage error.
func TestUsageAndDSNRules(t *testing.T) {
	for _, args := range [][]string{nil, {"import"}, {"import", "--snapshot-id"}, {"activate", "--snapshot-id", "s"}, {"charge_wallet", "--snapshot-id", "s"},
		{"create_order", "--snapshot-id", "s"}, {"suspend", "--snapshot-id", "s"}, {"import", "--snapshot-id", "s", "--run-id", "00000000-0000-0000-0000-000000000001"},
		{"parity", "--snapshot-id", "s", "--run-id", "not-a-uuid"}, {"import", "--snapshot-id", "s", "--force", "x"}, {"import", "--snapshot-id", "s", "--snapshot-id", "t"}} {
		if code, _, stderr := call(env(nil), args...); code != 2 || !strings.Contains(stderr, "usage") {
			t.Fatalf("%v: %d %q", args, code, stderr)
		}
	}
	if code, _, stderr := call(env(nil), "import", "--snapshot-id", "s"); code != 2 || !strings.Contains(stderr, EnvBiawinURL) {
		t.Fatalf("missing biawin DSN: %d %q", code, stderr)
	}
	base := map[string]string{EnvBiawinURL: "postgres://b@127.0.0.1:1/x", EnvStagingURL: "postgres://s@127.0.0.1:1/x", EnvIdentityURL: "postgres://i@127.0.0.1:1/x",
		EnvBillingURL: "postgres://bi@127.0.0.1:1/x"}
	if code, _, stderr := call(env(base), "import", "--snapshot-id", "s"); code != 2 || !strings.Contains(stderr, EnvFulfillmentURL) {
		t.Fatalf("import without fulfillment DSN: %d %q", code, stderr)
	}
	reuse := map[string]string{}
	for k, v := range base {
		reuse[k] = v
	}
	reuse[EnvFulfillmentURL] = base[EnvBiawinURL]
	if code, _, stderr := call(env(reuse), "import", "--snapshot-id", "s"); code != 2 || !strings.Contains(stderr, "must not reuse") {
		t.Fatalf("DSN reuse: %d %q", code, stderr)
	}
}

// A snapshot that was never staged fails (exit 1) and records nothing:
// no run, no parity, no fulfillment fact — never an empty SUCCESS.
func TestUnknownSnapshotRecordsNothing(t *testing.T) {
	e := pgtest.New(t)
	ctx := context.Background()
	for _, s := range []migrations.Source{stagingmigrations.Source(), identitymigrations.Source(), billingmigrations.Source(), biawinmigrations.Source(),
		fulfillmentmigrations.Source()} {
		if _, err := migrations.Up(ctx, e.DSN, s); err != nil {
			t.Fatal(err)
		}
	}
	m := map[string]string{EnvBiawinURL: e.LoginDSN(t, biawin.Module), EnvFulfillmentURL: e.LoginDSN(t, fulfillment.Module),
		EnvStagingURL: e.LoginDSN(t, integration.Module), EnvIdentityURL: e.LoginDSN(t, identity.Module), EnvBillingURL: e.LoginDSN(t, billing.Module)}
	for _, args := range [][]string{{"import", "--snapshot-id", "syn-missing"}, {"parity", "--snapshot-id", "syn-missing"}} {
		if code, out, _ := call(env(m), args...); code != 1 || out != "" {
			t.Fatalf("%v: %d %q", args, code, out)
		}
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	for _, tbl := range []string{"biawin.import_run", "biawin.parity_check", "fulfillment.import_run"} {
		var n int
		if err := admin.QueryRow(ctx, "SELECT count(*) FROM "+tbl).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d %v", tbl, n, err)
		}
	}
}
