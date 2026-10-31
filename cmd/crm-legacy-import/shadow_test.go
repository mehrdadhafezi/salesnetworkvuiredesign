package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func TestShadowDSNRules(t *testing.T) {
	if code, _, stderr := call(t, env(nil), "import-identity", "synthetic-x-0001"); code != 2 || !strings.Contains(stderr, "is not set") {
		t.Fatalf("no DSN: %d %q", code, stderr)
	}
	same := "postgres://x@127.0.0.1:1/x?sslmode=disable"
	e := env(map[string]string{EnvStagingURL: same, EnvIdentityURL: same, EnvOrganizationURL: "postgres://y@127.0.0.1:1/y"})
	if code, _, stderr := call(t, e, "import-organization", "synthetic-x-0001"); code != 2 || !strings.Contains(stderr, "must not reuse") {
		t.Fatalf("DSN reuse: %d %q", code, stderr)
	}
	for _, args := range [][]string{{"import-identity"}, {"import-organization", "a", "b"}, {"reconcile-identity-org"}} {
		if code, _, stderr := call(t, env(nil), args...); code != 2 || !strings.Contains(stderr, "usage") {
			t.Fatalf("%v: %d %q", args, code, stderr)
		}
	}
}

func TestShadowCommandsAgainstDisposablePostgres(t *testing.T) {
	e := pgtest.New(t)
	for _, src := range []migrations.Source{integrationmigrations.Source(), identitymigrations.Source(), organizationmigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, src); err != nil {
			t.Fatal(err)
		}
	}
	vars := map[string]string{
		EnvStagingURL:      e.LoginDSN(t, integration.Module),
		EnvIdentityURL:     e.LoginDSN(t, identity.Module),
		EnvOrganizationURL: e.LoginDSN(t, organization.Module),
	}
	dir := legacytest.Write(t, "synthetic-hr-cli-0001", t0, legacytest.HRScenario())
	if code, out, stderr := call(t, env(vars), "load", dir); code != 0 {
		t.Fatalf("load: %d %s %s", code, out, stderr)
	}
	// Organization before identity: every account is unresolved, honestly.
	code, out, _ := call(t, env(vars), "import-organization", "synthetic-hr-cli-0001")
	if code != 3 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"employments_user_not_imported": 14`) {
		t.Fatalf("org before identity: %d %s", code, out)
	}
	code, out, _ = call(t, env(vars), "import-identity", "synthetic-hr-cli-0001")
	if code != 3 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"users_inserted": 13`) {
		t.Fatalf("identity: %d %s", code, out)
	}
	code, out, _ = call(t, env(vars), "import-organization", "synthetic-hr-cli-0001")
	if code != 3 || !strings.Contains(out, `"employments_user_not_imported": 1`) || !strings.Contains(out, `"CYCLE": 2`) {
		t.Fatalf("organization: %d %s", code, out)
	}
	code, out, _ = call(t, env(vars), "reconcile-identity-org", "synthetic-hr-cli-0001")
	if code != 3 || !strings.Contains(out, `"outcome": "CONFLICT"`) || !strings.Contains(out, `"TARGET_ALLOWS_LEGACY_DENIED": 0`) ||
		!strings.Contains(out, `"target_more_permissive": 6`) || !strings.Contains(out, `"coverage": "INCOMPLETE"`) {
		t.Fatalf("reconcile: %d %s", code, out)
	}
	for _, marker := range []string{"SYNTHETIC-NID", "TEST-PHONE", "example.invalid", "Synthetic Account", "Synthetic HR Lead", "999000001"} {
		if strings.Contains(out, marker) {
			t.Fatalf("report leaks %q", marker)
		}
	}
	// Replay.
	if code, out, _ := call(t, env(vars), "import-identity", "synthetic-hr-cli-0001"); code != 3 || !strings.Contains(out, `"replay": true`) {
		t.Fatalf("identity replay: %d %s", code, out)
	}
	if code, out, _ := call(t, env(vars), "import-identity", "synthetic-missing-0001"); code != 3 || !strings.Contains(out, `"SOURCE_UNAVAILABLE"`) {
		t.Fatalf("missing batch: %d %s", code, out)
	}
}

// Slice 5: staging and customers only; reconcile fails on any invariant.
func TestCustomerCommandsAgainstDisposablePostgres(t *testing.T) {
	e := pgtest.New(t)
	for _, src := range []migrations.Source{integrationmigrations.Source(), customersmigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, src); err != nil {
			t.Fatal(err)
		}
	}
	if code, _, stderr := call(t, env(map[string]string{EnvStagingURL: e.LoginDSN(t, integration.Module)}), "import-customers", "synthetic-cust-0001"); code != 2 ||
		!strings.Contains(stderr, EnvCustomersURL) {
		t.Fatalf("no customers DSN: %d %q", code, stderr)
	}
	vars := map[string]string{EnvStagingURL: e.LoginDSN(t, integration.Module), EnvCustomersURL: e.LoginDSN(t, customers.Module)}
	dir := legacytest.Write(t, "synthetic-cust-0001", t0, legacytest.CustomerScenario(snapshot.Complete))
	if code, out, stderr := call(t, env(vars), "load", dir); code != 0 && code != 3 {
		t.Fatalf("load: %d %s %s", code, out, stderr)
	}
	code, out, stderr := call(t, env(vars), "import-customers", "synthetic-cust-0001")
	if (code != 0 && code != 3) || !strings.Contains(out, `"phone_derived_account_refs": 2`) || !strings.Contains(out, `"subjects_not_recoverable": 1`) {
		t.Fatalf("import-customers: %d %s %s", code, out, stderr)
	}
	code, out, _ = call(t, env(vars), "reconcile-customers", "synthetic-cust-0001")
	if code != 0 || !strings.Contains(out, `"phone_only_resolved_count": 0`) || !strings.Contains(out, `"phone_only_merge_count": 0`) ||
		!strings.Contains(out, `"phone_only_merge_count_is_zero": true`) || !strings.Contains(out, `"outcome": "SUCCESS"`) {
		t.Fatalf("reconcile-customers: %d %s", code, out)
	}
	for _, marker := range []string{"09000000101", "Synthetic Customer", "SYN-INV-"} {
		if strings.Contains(out, marker) {
			t.Fatalf("report leaks %q", marker)
		}
	}
	if code, out, _ := call(t, env(vars), "import-customers", "synthetic-cust-0001"); !strings.Contains(out, `"replay": true`) {
		t.Fatalf("replay: %d %s", code, out)
	}
}

// Slice 6: staging, identity (read), cases then custody; reconcile is honest.
func TestSalesCommandsAgainstDisposablePostgres(t *testing.T) {
	e := pgtest.New(t)
	for _, src := range []migrations.Source{integrationmigrations.Source(), identitymigrations.Source(), organizationmigrations.Source(),
		casesmigrations.Source(), custodymigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, src); err != nil {
			t.Fatal(err)
		}
	}
	if code, _, stderr := call(t, env(map[string]string{EnvStagingURL: e.LoginDSN(t, integration.Module)}), "import-sales", "synthetic-sales-cli-0001"); code != 2 ||
		!strings.Contains(stderr, "is not set") {
		t.Fatalf("missing DSNs: %d %q", code, stderr)
	}
	vars := map[string]string{
		EnvStagingURL: e.LoginDSN(t, integration.Module), EnvIdentityURL: e.LoginDSN(t, identity.Module), EnvOrganizationURL: e.LoginDSN(t, organization.Module),
		EnvCasesURL: e.LoginDSN(t, cases.Module), EnvCustodyURL: e.LoginDSN(t, custody.Module),
	}
	dir := legacytest.Write(t, "synthetic-sales-cli-0001", t0, append(legacytest.HRScenario(), legacytest.SalesScenario(snapshot.Complete)...))
	if code, out, stderr := call(t, env(vars), "load", dir); code != 0 && code != 3 {
		t.Fatalf("load: %d %s %s", code, out, stderr)
	}
	call(t, env(vars), "import-identity", "synthetic-sales-cli-0001")
	code, out, stderr := call(t, env(vars), "import-sales", "synthetic-sales-cli-0001")
	if code != 0 || !strings.Contains(out, `"cases.cases_inserted": 6`) || !strings.Contains(out, `"custody.events_appended": 8`) ||
		!strings.Contains(out, `"custody.log_actions_unknown": 1`) {
		t.Fatalf("import-sales: %d %s %s", code, out, stderr)
	}
	code, out, _ = call(t, env(vars), "reconcile-sales", "synthetic-sales-cli-0001")
	// Imperfect parity (one CONFLICT, unknowns) is PARTIAL: non-zero exit.
	if code == 0 || !strings.Contains(out, `"outcome": "PARTIAL"`) || !strings.Contains(out, `"full_parity": false`) ||
		!strings.Contains(out, `"no_resolved_alias": true`) || !strings.Contains(out, `"cases_matched_to_batch": 6`) {
		t.Fatalf("reconcile-sales: %d %s", code, out)
	}
	if code, out, _ := call(t, env(vars), "import-sales", "synthetic-sales-cli-0001"); code != 0 || !strings.Contains(out, `"replay": true`) {
		t.Fatalf("replay: %d %s", code, out)
	}
}
