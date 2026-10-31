package shadow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/imports"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/staging"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/imports"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

const t0 = "2026-10-29T08:00:00Z"

type env struct {
	*pgtest.Env
	admin   *pgx.Conn
	loader  *staging.Loader
	staged  *db.Runner
	ident   *identityquery.Service
	org     *organizationquery.Service
	identIn *identityimports.Command
	orgIn   *organizationimports.Command
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	for _, src := range []migrations.Source{integrationmigrations.Source(), identitymigrations.Source(), organizationmigrations.Source()} {
		if _, err := migrations.Up(ctx, e.DSN, src); err != nil {
			t.Fatalf("migrations: %v", err)
		}
	}
	at, _ := timeutil.ParseRFC3339("2026-10-29T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	staged := e.Runner(t, integration.Module)
	l, err := staging.NewLoader(staged, clock, "test/1")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	identRunner, orgRunner := e.Runner(t, identity.Module), e.Runner(t, organization.Module)
	iq := identityquery.New(identRunner)
	return &env{Env: e, admin: admin, loader: l, staged: staged, ident: iq, org: organizationquery.New(orgRunner),
		identIn: identityimports.New(identRunner, clock), orgIn: organizationimports.New(orgRunner, iq, clock)}
}

func (e *env) stage(t *testing.T, id string, ents []snapshot.EntityRows) *shadow.Batch {
	t.Helper()
	ctx := context.Background()
	p, err := snapshot.Open(legacytest.Write(t, id, t0, ents), spec.Default())
	if err != nil {
		t.Fatal(err)
	}
	if rep, err := e.loader.Load(ctx, p); err != nil || rep.Outcome != result.Success {
		t.Fatalf("stage: %+v %v", rep, err)
	}
	b, err := shadow.ReadBatch(ctx, e.staged, spec.Default(), id)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (e *env) importAll(t *testing.T, b *shadow.Batch) (identityRep, orgRep any, replay bool) {
	t.Helper()
	ctx := context.Background()
	in, _ := shadow.IdentityCommand(b)
	ir, err := e.identIn.ImportIdentity(ctx, in)
	if err != nil {
		t.Fatalf("identity import: %v", err)
	}
	on, _ := shadow.OrganizationCommand(b)
	or, err := e.orgIn.ImportOrganization(ctx, on)
	if err != nil {
		t.Fatalf("organization import: %v", err)
	}
	return ir, or, ir.Replay && or.Replay
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func TestShadowImportReconcileAndDiff(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	b := e.stage(t, "synthetic-hr-scenario-01", legacytest.HRScenario())
	ir, or, _ := e.importAll(t, b)
	t.Logf("identity: %+v", ir)
	t.Logf("organization: %+v", or)

	// Identity: one account per legacy key, never login-enabled.
	if n := e.count(t, `SELECT count(*) FROM identity.user_account`); n != 13 {
		t.Fatalf("users = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM identity.user_account WHERE login_enabled OR account_state <> 'SHADOW_IMPORTED'`); n != 0 {
		t.Fatal("an account is login-enabled")
	}
	// 14 profiles carry a role label; the one whose account is missing is reported, not invented.
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant`); n != 13 {
		t.Fatalf("grants = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`); n != 0 {
		t.Fatal("non-shadow grant materialised")
	}
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE mapping_state = 'UNMAPPED'`); n != 2 {
		t.Fatalf("unmapped grants = %d", n)
	}
	// A position slug stored as a role label is not a role (Position ≠ Role).
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE legacy_role_value = 'seller' AND role_key IS NULL`); n != 1 {
		t.Fatalf("position-slug label mapped: %d", n)
	}

	// Organization: profile → Person + Employment, User stays a reference.
	for q, want := range map[string]int64{
		`SELECT count(*) FROM organization.person`:                                                          14,
		`SELECT count(*) FROM organization.employment`:                                                      14,
		`SELECT count(*) FROM organization.employment WHERE user_link_state = 'USER_NOT_IMPORTED'`:          1,
		`SELECT count(*) FROM organization.employment WHERE user_link_state = 'SHARED_USER'`:                2,
		`SELECT count(*) FROM organization.employment WHERE employment_status = 'UNMAPPED'`:                 1,
		`SELECT count(*) FROM organization.employment WHERE employment_status_raw = 'synthetic_on_leave_x'`: 1,
		`SELECT count(*) FROM organization.org_unit`:                                                        7,
		`SELECT count(*) FROM organization.org_assignment`:                                                  14,
		`SELECT count(*) FROM organization.org_assignment_history`:                                          3,
		`SELECT count(*) FROM organization.position_role_default WHERE mapping_state = 'MAPPED'`:            2,
		`SELECT count(*) FROM organization.org_unit WHERE parent_state = 'MISSING'`:                         2,
		`SELECT count(*) FROM organization.org_unit WHERE parent_state = 'NOT_RECORDED'`:                    1,
		`SELECT count(*) FROM organization.employment WHERE hire_date = DATE '2025-01-15'`:                  14,
	} {
		if n := e.count(t, q); n != want {
			t.Fatalf("%s = %d, want %d", q, n, want)
		}
	}

	// Every required hierarchy conflict is reported (never auto-fixed).
	for _, kind := range []string{"MISSING_PARENT", "SELF_PARENT", "CYCLE", "DUPLICATE_CURRENT", "MULTIPLE_CURRENT_PARENTS",
		"INTERVAL_OVERLAP", "HISTORY_GAP", "MISSING_CHILD", "ORG_UNIT_PARENT_MISSING", "ORG_UNIT_PARENT_NOT_RECORDED",
		"MEMBERSHIP_MISMATCH", "CURRENT_FLAG_INCONSISTENT", "HISTORY_CURRENT_DISAGREE", "UNREADABLE_DATE",
		"UNMAPPED_RELATIONSHIP", "SHARED_USER", "USER_NOT_IMPORTED"} {
		if n := e.count(t, `SELECT count(*) FROM organization.hierarchy_finding WHERE kind = $1`, kind); n == 0 {
			t.Errorf("finding %s not reported", kind)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM organization.hierarchy_finding WHERE kind = 'CYCLE'`); n != 2 {
		t.Errorf("cycle findings = %d", n)
	}
	// Missing parents are never invented: the cycle and self-parent rows stay as recorded.
	if n := e.count(t, `SELECT count(*) FROM organization.org_assignment WHERE parent_ref_state = 'MISSING' AND parent_employment_id IS NULL`); n != 1 {
		t.Errorf("missing parent rows = %d", n)
	}

	// Sensitive legacy values never reach a target table.
	for _, marker := range []string{"SYNTHETIC-NID", "TEST-PHONE", "999000001", "synthetic sensitive note", "synthetic reason text",
		"example.invalid", "synthetic_contract", "synthetic_training", `"synthetic":true`, `s:5:"sn_hr"`} {
		for _, tbl := range []string{"identity.user_account", "identity.role_grant", "organization.person", "organization.employment",
			"organization.org_assignment", "organization.org_assignment_history", "organization.position_role_default",
			"organization.hierarchy_finding", "organization.org_unit", "organization.position"} {
			if n := e.count(t, `SELECT count(*) FROM `+tbl+` x WHERE row_to_json(x)::text LIKE '%' || $1 || '%'`, marker); n != 0 {
				t.Errorf("%s leaked into %s", marker, tbl)
			}
		}
	}

	rep, err := shadow.Reconcile(ctx, b, e.ident, e.org)
	if err != nil {
		t.Fatal(err)
	}
	az := rep.Authorization
	want := map[authorization.DiffClass]int64{
		authorization.DiffMatch: 6, authorization.DiffTargetDeniesLegacyAllows: 6, authorization.DiffConflict: 6,
		authorization.DiffUnmappedRole: 12, authorization.DiffUnknownLegacy: 48, authorization.DiffTargetAllowsLegacyDenies: 0,
		authorization.DiffUnknownTarget: 0,
	}
	for c, n := range want {
		if az.Comparison[c] != n {
			t.Errorf("comparison %s = %d, want %d (all %v)", c, az.Comparison[c], n, az.Comparison)
		}
	}
	if az.TargetMorePermissive != 6 || len(az.SecurityRows) != 6 || az.SecurityRows[0].UserKey != "13" {
		t.Errorf("target-more-permissive = %d %+v", az.TargetMorePermissive, az.SecurityRows)
	}
	if az.Coverage != "INCOMPLETE" || az.Staged["wp_usermeta_capability_rows"] != 2 || az.Staged["wp_usermeta_rows"] != 3 {
		t.Errorf("coverage = %s staged=%v", az.Coverage, az.Staged)
	}
	if rep.Identity.Outcome != result.Conflict || rep.Organization.Outcome != result.Conflict ||
		rep.Authorization.Outcome != result.Conflict || rep.Outcome != result.Conflict {
		t.Errorf("outcomes identity=%s organization=%s overall=%s", rep.Identity.Outcome, rep.Organization.Outcome, rep.Outcome)
	}
	if rep.Identity.Staged["users_without_hr_profile"] != 1 || rep.Identity.Staged["hr_profiles_referencing_missing_user"] != 1 {
		t.Errorf("orphans %v", rep.Identity.Staged)
	}
}

func TestReplayAndRefreshAreDeterministic(t *testing.T) {
	e := setup(t)
	b := e.stage(t, "synthetic-hr-scenario-02", legacytest.HRScenario())
	e.importAll(t, b)
	var firstID string
	if err := e.admin.QueryRow(context.Background(), `SELECT user_id::text FROM identity.user_account WHERE source_key = '3'`).Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	snapshotRows := func() string {
		var s string
		if err := e.admin.QueryRow(context.Background(), `SELECT md5(string_agg(row_to_json(x)::text, ',' ORDER BY x.source_key))
			FROM (SELECT user_id, source_key, version, presence, display_name FROM identity.user_account) x`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := snapshotRows()
	// Same batch again: replay writes nothing and duplicates nothing.
	if _, _, replay := e.importAll(t, b); !replay {
		t.Fatal("same batch was not a replay")
	}
	if snapshotRows() != before {
		t.Fatal("replay changed identity rows")
	}
	if n := e.count(t, `SELECT count(*) FROM organization.person`); n != 14 {
		t.Fatalf("replay duplicated persons: %d", n)
	}

	// A newer batch: changed content refreshes in place; a missing account is
	// marked absent (never deleted); a changed history row is kept and reported.
	ents := legacytest.HRScenario()
	for i := range ents {
		switch ents[i].SpecID {
		case "legacy.wp_user.v1":
			ents[i].Rows = append(ents[i].Rows[:5], ents[i].Rows[6:]...) // drop account 6
			ents[i].Rows[2][3] = legacytest.T("Synthetic Account 003 renamed")
		case "legacy.hr_assignment_history.v1":
			ents[i].Rows[0][6] = legacytest.T("synthetic_changed_action")
		}
	}
	b2 := e.stage(t, "synthetic-hr-scenario-03", ents)
	ir, or, _ := e.importAll(t, b2)
	t.Logf("identity: %+v organization: %+v", ir, or)
	var id string
	var version int64
	if err := e.admin.QueryRow(context.Background(), `SELECT user_id::text, version FROM identity.user_account WHERE source_key = '3'`).Scan(&id, &version); err != nil {
		t.Fatal(err)
	}
	if id != firstID || version != 2 {
		t.Fatalf("refresh forked or skipped: id %s→%s version %d", firstID, id, version)
	}
	if n := e.count(t, `SELECT count(*) FROM identity.user_account`); n != 13 {
		t.Fatalf("users = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM identity.user_account WHERE presence = 'ABSENT_IN_LATEST' AND source_key = '6'`); n != 1 {
		t.Fatal("absent account not marked")
	}
	if n := e.count(t, `SELECT count(*) FROM organization.org_assignment_history WHERE action_raw = 'synthetic_changed_action'`); n != 0 {
		t.Fatal("history was rewritten")
	}
	if n := e.count(t, `SELECT count(*) FROM organization.hierarchy_finding WHERE kind = 'HISTORY_CHANGED'`); n != 1 {
		t.Fatal("changed history not reported")
	}
}

// Ownership: each module writes only its own schema; integration cannot
// write identity or organization; read transactions cannot write.
func TestModuleOwnershipAndReadOnly(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	b := e.stage(t, "synthetic-hr-scenario-04", legacytest.HRScenario())
	e.importAll(t, b)
	writes := map[string]string{
		"identity":     `UPDATE identity.user_account SET display_name = 'x'`,
		"organization": `UPDATE organization.person SET display_name = 'x'`,
	}
	runners := map[string]*db.Runner{"identity": e.Runner(t, identity.Module), "organization": e.Runner(t, organization.Module), "integration": e.staged}
	for owner, r := range runners {
		for target, q := range writes {
			err := r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error { _, err := tx.Exec(ctx, q); return err })
			if owner == target && err != nil {
				t.Errorf("%s cannot write its own schema: %v", owner, err)
			}
			if owner != target && (err == nil || !strings.Contains(err.Error(), "permission denied")) {
				t.Errorf("%s wrote %s: %v", owner, target, err)
			}
			// Read-only transactions never write, even their own schema.
			err = r.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error { _, err := tx.Exec(ctx, q); return err })
			if err == nil {
				t.Errorf("read-only %s transaction wrote %s", owner, target)
			}
		}
	}
	// History is append-only even for its owner.
	err := runners["organization"].InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE organization.org_assignment_history SET action_raw = 'x'`)
		return err
	})
	if err == nil {
		t.Error("assignment history is updatable")
	}
	// Shadow grants cannot be turned into active grants by the owner either.
	err = runners["identity"].InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity.role_grant SET status = 'ACTIVE'`)
		return err
	})
	if err == nil {
		t.Error("a role grant became ACTIVE")
	}
}

func TestBlockedWhenBatchMissing(t *testing.T) {
	e := setup(t)
	if _, err := shadow.ReadBatch(context.Background(), e.staged, spec.Default(), "synthetic-missing-01"); err == nil || !strings.Contains(err.Error(), "no staged batch") {
		t.Fatalf("missing batch: %v", err)
	}
}
