//go:build e2e

package shadow_test

// R7-6 full-stack browser E2E for the HR read surface (OP-04) and the HR
// change-request workflow (OP-05), on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// Two evidence classes, never mixed:
//
// A. PRODUCTION-EQUIVALENT (TestBrowserHRFullStack). crm-api runs with
//    CRM_ENV=test, real R1 sessions and the production subject resolver; the
//    synthetic Slice 11 organisation (setupWrites) and two synthetic
//    credentials (supervisor A = legacy user 4, senior = legacy user 3). A
//    subject holds only ACTIVE role grants and none exist (D-42; D-34,
//    R5-SEC-01 open), and no role bundle carries an HR read or HR request
//    capability, so every HR read and command is refused. Two requests are
//    created BEFORE the browser runs through the domain service (backend
//    setup, never through the browser or the API), so the suite can prove
//    that even the recorded reviewer cannot decide them over HTTP.
//
// B. TEST-ONLY OVERLAY (TestBrowserHRReadOverlay). crm-api runs with
//    CRM_ENV=test and the EXISTING development auth stub (CRM_DEV_AUTH=1,
//    refused in production by apiserver.Config.Validate and
//    TestRunRefusesDevAuthInProduction; never combined with sessions). The
//    browser context adds the stub header (X-Dev-User) itself: the
//    production SPA bundle contains no dev identity. authorization.DevSubjects
//    gives dev-user-001 a synthetic in-memory HR role (no row, no ACTIVE
//    grant in the database) and dev-user-002 nothing. The data is the R7-5
//    integrated dataset (imported SHADOW organisation evidence, including
//    cycles, self-parent lines and multiple parents). This class is
//    BACKEND-ONLY authorization evidence: it shows what the HR read pages
//    render, never that production authorizes it.
//
// Positive create / approve / reject in a browser is BLOCKED: no existing
// isolated overlay maps a browser identity to a synthetic organisation user
// (DevSubjects carries no user id, so the service answers ROW_SCOPE_DENIED).
// The positive workflow, wrong reviewer, stale version, idempotent replay,
// out-of-scope seller, ineligible destination, malformed hierarchy and the
// DEF-HR-01 self-parent / cycle refusals stay covered by the backend tests
// (r7_4_hrrequest_hardening_test.go, r7_4_def_hr_01_test.go).
//
// After each run the orchestrator proves no HR request, decision, command
// record, outbox or idempotency row changed and no employment, hierarchy,
// identity grant, custody, case, customer, billing, payment, finance,
// commission, integration, fulfillment, biawin or catalog row changed.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// hrFlags are every flag behind the HR read and HR request surfaces
// (HR request write + writeback ON = CONTROLLED_WRITEBACK mode).
var hrFlags = []string{"organization.hr_read_shadow", "organization.hr_request_write", "organization.hr_request_writeback"}

// hrGuarded are snapshotted before and after the browser runs.
var hrGuarded = []string{"organization", "identity", "cases", "custody", "customers", "billing", "payments", "finance", "commission",
	"integration", "fulfillment", "biawin", "catalog"}

// hrAuthRuntime are the only rows a real login / logout / expiry may write.
var hrAuthRuntime = map[string]bool{`identity.session`: true, `identity.auth_event`: true, `identity.credential`: true}

func requireDist(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "web", "dist", "index.html")); err != nil {
		t.Fatalf("web/dist is missing (run `npm run build` in web/ first): %v", err)
	}
}

// diffOutsideAuth reports every snapshot difference except the auth runtime
// tables of a real session.
func diffOutsideAuth(t *testing.T, before, after map[string]tableState) {
	t.Helper()
	for _, d := range diffFinancial(before, after) {
		table := strings.Fields(d)[0]
		table = strings.TrimSuffix(table, ":")
		if hrAuthRuntime[table] {
			continue
		}
		t.Errorf("%s (the browser suite may not change it)", d)
	}
}

func TestBrowserHRFullStack(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := setupWrites(t) // disposable DB + synthetic Slice 11 organisation (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)

	sup := e.syntheticCredential(t, "e2e-hr-supervisor-a", e.users["4"]) // supervisor A: seller 6 and 7 in scope
	sen := e.syntheticCredential(t, "e2e-hr-senior", e.users["3"])       // senior: A and B in scope; reviewer of A's requests

	// Backend setup through the domain service (never the API or browser):
	// one pending terminate request (A → seller 7, reviewer = senior) and one
	// pending transfer request (senior → seller 6 to supervisor B, reviewer =
	// manager). These exist so the browser can prove that even the recorded
	// reviewer cannot read or decide them.
	ctx := context.Background()
	term, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrrequest.Create{RequestType: api.HRTerminateSeller, SubjectUserID: e.users["7"],
		Reason: "synthetic r7-6 terminate", IdempotencyKey: "r7-6-seed-terminate-0001", CorrelationID: "corr-r7-6-seed", Actor: e.hrActor("4")})
	if err != nil || term.RequestStatus != api.HRPendingReview {
		t.Fatalf("seed terminate: %+v %v", term, err)
	}
	target := e.users["5"]
	tr, err := e.hr.CreateRequest(ctx, rollout.Controlled, hrrequest.Create{RequestType: api.HRTransferSeller, SubjectUserID: e.users["6"],
		TargetParentUserID: &target, Reason: "synthetic r7-6 transfer", IdempotencyKey: "r7-6-seed-transfer-0001", CorrelationID: "corr-r7-6-seed",
		Actor: e.hrActor("3")})
	if err != nil || tr.RequestStatus != api.HRPendingReview {
		t.Fatalf("seed transfer: %+v %v", tr, err)
	}

	on := map[string]bool{"identity.session_auth": true}
	for _, f := range hrFlags {
		on[f] = true
	}
	e.setFlags(t, on)
	apiURL := e.startCRMAPI(t, root, map[string]owner.Module{
		"CRM_IDENTITY_AUTH_DATABASE_URL":      identity.Module,
		"CRM_IDENTITY_READ_DATABASE_URL":      identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL":  organization.Module,
		"CRM_ORGANIZATION_WRITE_DATABASE_URL": organization.Module,
	}, map[string]string{"CRM_AUTH_SESSION_IDLE": "1m", "CRM_AUTH_SESSION_ABSOLUTE": "1m"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, hrGuarded)
	if m := missingSeed(before, "organization.employment", "organization.org_assignment", "organization.hr_request",
		"organization.hr_request_event"); len(m) != 0 {
		t.Fatalf("the synthetic scenario is incomplete: %v", m)
	}
	env := map[string]string{
		"E2E_BASE_URL":               web,
		"E2E_HR_SUPERVISOR_LOGIN":    sup.login,
		"E2E_HR_SUPERVISOR_PASSWORD": sup.password,
		"E2E_HR_SUPERVISOR_USER_ID":  sup.userID,
		"E2E_HR_SENIOR_LOGIN":        sen.login,
		"E2E_HR_SENIOR_PASSWORD":     sen.password,
		"E2E_HR_SENIOR_USER_ID":      sen.userID,
		"E2E_HR_SELLER_IN_SCOPE":     e.users["7"],
		"E2E_HR_SELLER_OUT_OF_SCOPE": e.users["9"],
		"E2E_HR_TARGET_SUPERVISOR":   e.users["5"],
		"E2E_HR_REQUEST_TERMINATE":   term.RequestID,
		"E2E_HR_REQUEST_TRANSFER":    tr.RequestID,
		"E2E_HR_PERSON_ID":           e.q(t, `SELECT person_id::text FROM organization.employment WHERE user_id = $1`, e.users["7"]),
	}
	runPlaywright(t, root, "hr-flags-on", env, sup, sen)

	off := map[string]bool{}
	for _, f := range hrFlags {
		off[f] = false
	}
	e.setFlags(t, off)
	runPlaywright(t, root, "hr-flags-off", env, sup, sen)

	diffOutsideAuth(t, before, schemaSnapshot(t, e.admin, hrGuarded))
	for _, q := range []string{
		`SELECT count(*) FROM organization.hr_request WHERE status <> 'pending_review' OR version <> 1`,
		`SELECT count(*) FROM organization.hr_request_event WHERE action <> 'created'`,
		`SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`,
		`SELECT count(*) FROM integration.writeback_command`,
	} {
		if n := e.count(t, q); n != 0 {
			t.Errorf("%s = %d, want 0", q, n)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM organization.hr_request`); n != 2 {
		t.Errorf("hr_request rows = %d, want the 2 seeded requests", n)
	}
}

func TestBrowserHRReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := setupIntegrated(t) // disposable DB + R7-5 integrated synthetic dataset (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)
	_, steps := e.seedIntegrated(t)
	for _, s := range steps {
		if s.Outcome.String() == "FAILED" {
			t.Fatalf("seed step %s failed", s.Step)
		}
	}
	setFlagsIn(t, e.env, map[string]bool{"organization.hr_read_shadow": true,
		"organization.hr_request_write": true, "organization.hr_request_writeback": true})
	apiURL := e.startCRMAPI(t, root, map[string]owner.Module{
		"CRM_IDENTITY_READ_DATABASE_URL":      identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL":  organization.Module,
		"CRM_ORGANIZATION_WRITE_DATABASE_URL": organization.Module,
	}, map[string]string{"CRM_DEV_AUTH": "1"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, hrGuarded)
	persons := e.count(t, `SELECT count(*) FROM organization.person WHERE presence = 'PRESENT'`)
	cycles := e.count(t, `SELECT count(*) FROM organization.hierarchy_finding WHERE kind = 'CYCLE'`)
	selfParents := e.count(t, `SELECT count(*) FROM organization.hierarchy_finding WHERE kind = 'SELF_PARENT'`)
	if persons == 0 || cycles == 0 || selfParents == 0 {
		t.Fatalf("the integrated dataset is incomplete: persons %d, cycles %d, self-parent %d", persons, cycles, selfParents)
	}
	env := map[string]string{
		"E2E_BASE_URL":              web,
		"E2E_HR_PERSONS":            itoa(persons),
		"E2E_HR_CYCLE_FINDINGS":     itoa(cycles),
		"E2E_HR_SELF_PARENT":        itoa(selfParents),
		"E2E_HR_SELLER_USER_ID":     e.one(t, `SELECT user_id::text FROM identity.user_account WHERE source_key = '3'`),
		"E2E_HR_SELF_PARENT_PERSON": e.one(t, `SELECT p.person_id::text FROM organization.person p WHERE p.source_key = '110'`),
	}
	runPlaywright(t, root, "hr-read-overlay", env)

	diffOutsideAuth(t, before, schemaSnapshot(t, e.admin, hrGuarded))
	if n := e.count(t, `SELECT count(*) FROM organization.hr_request`); n != 0 {
		t.Errorf("hr_request rows = %d, want 0 (no request may be created through the overlay)", n)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// setFlagsIn switches flags inside the disposable database only.
func setFlagsIn(t *testing.T, e *env, kv map[string]bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-02T08:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		for k, v := range kv {
			if err := featureflags.Set(ctx, tx, featureflags.Flag{Key: k, Enabled: v, UpdatedAt: at}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func (e *env) one(t *testing.T, q string, args ...any) string {
	t.Helper()
	var v string
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}
