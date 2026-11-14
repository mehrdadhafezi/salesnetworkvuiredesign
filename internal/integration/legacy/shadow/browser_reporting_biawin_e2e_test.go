//go:build e2e

package shadow_test

// R7-7 full-stack browser E2E for the Reporting / MIS read surface (OP-26)
// and the Biawin read surface (OP-21), on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// Both orchestrators seed the R7-5 integrated SYNTHETIC dataset through the
// existing import path (staging → module imports → finance / commission
// shadow → reporting projections) into a disposable database cloned from the
// CI-PERF-01 template. Every value the suites check comes from that dataset.
//
// Two evidence classes, never mixed:
//
// A. PRODUCTION-EQUIVALENT (TestBrowserReportingBiawinFullStack). crm-api
//    runs with CRM_ENV=test, real R1 sessions and the production subject
//    resolver; two synthetic credentials for imported synthetic users. A
//    subject holds only ACTIVE role grants and none exist (D-42; D-34 and
//    R5-SEC-01 open), and no role bundle carries a reporting or Biawin read
//    capability, so every read is refused at the capability gate. The data
//    is present, so a refusal is never an artefact of an empty database.
//
// B. TEST-ONLY OVERLAY (TestBrowserReportingBiawinReadOverlay). crm-api runs
//    with CRM_ENV=test and the EXISTING development auth stub (CRM_DEV_AUTH=1,
//    refused in production; never combined with sessions). The browser
//    context adds the stub header itself; the production SPA bundle carries no
//    dev identity. authorization.DevSubjects gives dev-user-001 synthetic
//    in-memory reporting / Biawin read capabilities (no row, no ACTIVE grant
//    in the database) and dev-user-002 nothing. This class is BACKEND-ONLY
//    authorization evidence: it proves what the pages render from the real
//    API and database, never that production authorizes it.
//
// The orchestrator passes database-derived expectations (counts, sums, ids,
// parity) to the browser so the suite checks rendered UI == API answer ==
// database. After each run it proves no row of any domain schema changed
// (only identity session / auth-event / credential rows of a real login may),
// no projection was rebuilt, no import or parity run was added, and no
// activation, provider operation, financial posting, commission credit or
// write-back exists.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
)

// rbFlags are every flag behind the reporting / MIS and Biawin read surfaces.
var rbFlags = []string{"reporting.read_shadow", "biawin_fulfillment.read_shadow"}

// rbGuarded are snapshotted before and after the browser runs (every domain
// schema of the integrated dataset; feature flags are switched by the
// orchestrator itself and are not part of the snapshot).
var rbGuarded = []string{"integration", "identity", "organization", "cases", "custody", "customers", "billing", "payments", "biawin",
	"fulfillment", "finance", "commission", "reporting", "catalog", "audit"}

// rbLogins are the restricted per-module logins crm-api receives.
func rbLogins(sessions bool) map[string]owner.Module {
	m := map[string]owner.Module{
		"CRM_IDENTITY_READ_DATABASE_URL":     identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL": organization.Module,
		"CRM_REPORTING_READ_DATABASE_URL":    reporting.Module,
		"CRM_BIAWIN_READ_DATABASE_URL":       biawin.Module,
		"CRM_FULFILLMENT_READ_DATABASE_URL":  fulfillment.Module,
	}
	if sessions {
		m["CRM_IDENTITY_AUTH_DATABASE_URL"] = identity.Module
	}
	return m
}

// seedR77 seeds the R7-5 integrated dataset and fails when a fixture the
// suites rely on is missing (an incomplete dataset must never pass).
func seedR77(t *testing.T) *integratedEnv {
	t.Helper()
	e := setupIntegrated(t) // disposable DB cloned from the CI-PERF-01 template (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)
	_, steps := e.seedIntegrated(t)
	for _, s := range steps {
		if s.Outcome.String() == "FAILED" {
			t.Fatalf("seed step %s failed", s.Step)
		}
	}
	unresolved := e.one(t, `SELECT coalesce(min(invoice_key), '') FROM biawin.subscription WHERE invoice_link_state = 'UNKNOWN'`)
	known := false
	for _, v := range legacytest.IntegratedKnownUnresolved["invoice"] {
		known = known || v == unresolved
	}
	for _, c := range []struct {
		what string
		ok   bool
	}{
		{"one reporting projection run", e.count(t, `SELECT count(*) FROM reporting.projection_run`) == 1},
		{"one Biawin import run", e.count(t, `SELECT count(*) FROM biawin.import_run`) == 1},
		{"one Biawin parity check", e.count(t, `SELECT count(*) FROM biawin.parity_check`) == 1},
		{"case facts", e.count(t, `SELECT count(*) FROM reporting.case_fact`) > 0},
		{"custody facts", e.count(t, `SELECT count(*) FROM reporting.custody_fact`) > 0},
		{"invoice facts", e.count(t, `SELECT count(*) FROM reporting.invoice_fact`) > 0},
		{"an unmapped payment state", e.count(t, `SELECT count(*) FROM reporting.payment_fact WHERE state_target IS NULL`) > 0},
		{"an unknown case state", e.count(t, `SELECT count(*) FROM reporting.case_fact WHERE state_mapping = 'UNKNOWN'`) > 0},
		{"a custody CONFLICT", e.count(t, `SELECT count(*) FROM reporting.custody_fact WHERE parity = 'CONFLICT'`) > 0},
		{"a NOT_FINAL metric contract", e.count(t, `SELECT count(*) FROM reporting.metric_contract WHERE status = 'NOT_FINAL'`) > 0},
		{"a subscription with a known-unresolved invoice", known},
		{"a subscription with an unmapped state", e.count(t, `SELECT count(*) FROM biawin.subscription WHERE state_mapped = 'UNKNOWN'`) > 0},
		{"a subscription with more than one item", e.count(t, `SELECT count(*) FROM biawin.subscription WHERE item_count > 1`) > 0},
	} {
		if !c.ok {
			t.Fatalf("the integrated dataset is incomplete: missing %s", c.what)
		}
	}
	return e
}

// assertReadOnly proves the browser changed nothing outside a real login's
// session rows, and that no rebuild, import, activation, provider operation,
// posting, credit or write-back exists.
func assertReadOnly(t *testing.T, e *integratedEnv, before map[string]tableState) {
	t.Helper()
	diffOutsideAuth(t, before, schemaSnapshot(t, e.admin, rbGuarded))
	for q, want := range map[string]int64{
		`SELECT count(*) FROM reporting.projection_run`:                     1,
		`SELECT count(*) FROM biawin.import_run`:                            1,
		`SELECT count(*) FROM biawin.parity_check`:                          1,
		`SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`: 0,
		`SELECT count(*) FROM integration.writeback_command`:                0,
		`SELECT count(*) FROM integration.provider_operation`:               0,
		`SELECT count(*) FROM biawin.activation`:                            0,
		`SELECT count(*) FROM fulfillment.work_case`:                        0,
		`SELECT count(*) FROM payments.submitted_attempt`:                   0,
		`SELECT count(*) FROM audit.audit_event`:                            0,
	} {
		if n := e.count(t, q); n != want {
			t.Errorf("%s = %d, want %d", q, n, want)
		}
	}
}

func TestBrowserReportingBiawinFullStack(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	users := strings.Fields(e.one(t, `SELECT string_agg(user_id::text, ' ' ORDER BY source_key::bigint) FROM
		(SELECT user_id, source_key FROM identity.user_account ORDER BY source_key::bigint LIMIT 2) u`))
	if len(users) != 2 {
		t.Fatalf("the integrated dataset needs two synthetic users, got %d", len(users))
	}
	a := e.syntheticCredential(t, "e2e-rb-user-a", users[0])
	b := e.syntheticCredential(t, "e2e-rb-user-b", users[1])

	on := map[string]bool{"identity.session_auth": true}
	for _, f := range rbFlags {
		on[f] = true
	}
	setFlagsIn(t, e.env, on)
	apiURL := e.startCRMAPI(t, root, rbLogins(true), map[string]string{"CRM_AUTH_SESSION_IDLE": "1m", "CRM_AUTH_SESSION_ABSOLUTE": "1m"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, rbGuarded)
	env := map[string]string{
		"E2E_BASE_URL":         web,
		"E2E_RB_A_LOGIN":       a.login,
		"E2E_RB_A_PASSWORD":    a.password,
		"E2E_RB_A_USER_ID":     a.userID,
		"E2E_RB_B_LOGIN":       b.login,
		"E2E_RB_B_PASSWORD":    b.password,
		"E2E_RB_B_USER_ID":     b.userID,
		"E2E_RB_CASE_ID":       e.one(t, `SELECT min(case_id::text) FROM reporting.case_fact`),
		"E2E_RB_INVOICE_ID":    e.one(t, `SELECT min(invoice_id::text) FROM reporting.invoice_fact`),
		"E2E_RB_ATTEMPT_ID":    e.one(t, `SELECT min(attempt_id::text) FROM reporting.payment_fact`),
		"E2E_RB_SUBSCRIPTION":  e.one(t, `SELECT min(subscription_id::text) FROM biawin.subscription`),
		"E2E_RB_RP_RUN_ID":     e.one(t, `SELECT run_id::text FROM reporting.projection_run`),
		"E2E_RB_BW_RUN_ID":     e.one(t, `SELECT run_id::text FROM biawin.import_run`),
		"E2E_RB_SUBSCRIPTIONS": itoa(e.count(t, `SELECT count(*) FROM biawin.subscription`)),
	}
	runPlaywright(t, root, "reporting-biawin-flags-on", env, a, b)

	off := map[string]bool{}
	for _, f := range rbFlags {
		off[f] = false
	}
	setFlagsIn(t, e.env, off)
	runPlaywright(t, root, "reporting-biawin-flags-off", env, a, b)

	assertReadOnly(t, e, before)
}

func TestBrowserReportingBiawinReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	on := map[string]bool{}
	for _, f := range rbFlags {
		on[f] = true
	}
	setFlagsIn(t, e.env, on)
	apiURL := e.startCRMAPI(t, root, rbLogins(false), map[string]string{"CRM_DEV_AUTH": "1"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, rbGuarded)
	multi := e.one(t, `SELECT subscription_id::text FROM biawin.subscription ORDER BY item_count DESC, source_key LIMIT 1`)
	env := map[string]string{
		"E2E_BASE_URL": web,
		// Reporting projections (the one projection run of the dataset).
		"E2E_RP_RUN_ID":            e.one(t, `SELECT run_id::text FROM reporting.projection_run`),
		"E2E_RP_CASES":             itoa(e.count(t, `SELECT count(*) FROM reporting.case_fact`)),
		"E2E_RP_CASES_UNKNOWN":     itoa(e.count(t, `SELECT count(*) FROM reporting.case_fact WHERE state_mapping = 'UNKNOWN'`)),
		"E2E_RP_LEGACY_LEADS":      itoa(e.count(t, `SELECT count(*) FROM reporting.case_fact WHERE workflow_type = 'LEGACY_LEAD'`)),
		"E2E_RP_CUSTODY":           itoa(e.count(t, `SELECT count(*) FROM reporting.custody_fact`)),
		"E2E_RP_CUSTODY_CONFLICT":  itoa(e.count(t, `SELECT count(*) FROM reporting.custody_fact WHERE parity = 'CONFLICT'`)),
		"E2E_RP_INVOICES":          itoa(e.count(t, `SELECT count(*) FROM reporting.invoice_fact`)),
		"E2E_RP_PAYMENTS":          itoa(e.count(t, `SELECT count(*) FROM reporting.payment_fact`)),
		"E2E_RP_PAYMENTS_UNMAPPED": itoa(e.count(t, `SELECT count(*) FROM reporting.payment_fact WHERE state_target IS NULL`)),
		"E2E_RP_PAYMENTS_ACCEPTED": itoa(e.count(t, `SELECT count(*) FROM reporting.payment_fact WHERE state_target = 'ACCEPTED'`)),
		"E2E_RP_IRT_SUM":           e.one(t, `SELECT coalesce(sum(amount), 0)::text FROM reporting.payment_fact WHERE unit = 'IRT'`),
		"E2E_RP_ACCEPTED_IRT_SUM":  e.one(t, `SELECT coalesce(sum(amount), 0)::text FROM reporting.payment_fact WHERE unit = 'IRT' AND state_target = 'ACCEPTED'`),
		"E2E_RP_NOT_FINAL":         itoa(e.count(t, `SELECT count(*) FROM reporting.metric_contract WHERE status = 'NOT_FINAL'`)),
		"E2E_RP_CONTRACTS":         itoa(e.count(t, `SELECT count(*) FROM reporting.metric_contract`)),
		// Biawin shadow import (the one import run of the dataset).
		"E2E_BW_RUN_ID":        e.one(t, `SELECT run_id::text FROM biawin.import_run`),
		"E2E_BW_SUBSCRIPTIONS": itoa(e.count(t, `SELECT count(*) FROM biawin.subscription`)),
		"E2E_BW_STATE_COUNTS": e.one(t, `SELECT string_agg(s || '=' || (SELECT count(*) FROM biawin.subscription WHERE state_mapped = s), ',' ORDER BY s)
			FROM unnest(ARRAY['ACTIVE', 'SUSPENDED', 'UNKNOWN', 'EMPTY']) s`),
		"E2E_BW_UNRESOLVED_SUB": e.one(t, `SELECT subscription_id::text FROM biawin.subscription WHERE invoice_link_state = 'UNKNOWN' ORDER BY source_key LIMIT 1`),
		"E2E_BW_UNRESOLVED_KEY": e.one(t, `SELECT source_key || ':' || invoice_key FROM biawin.subscription WHERE invoice_link_state = 'UNKNOWN' ORDER BY source_key LIMIT 1`),
		"E2E_BW_UNRESOLVED_N":   itoa(e.count(t, `SELECT count(*) FROM biawin.subscription WHERE invoice_link_state = 'UNKNOWN'`)),
		"E2E_BW_UNMAPPED_KEY":   e.one(t, `SELECT source_key || ':' || coalesce(state_raw, '') FROM biawin.subscription WHERE state_mapped = 'UNKNOWN' ORDER BY source_key LIMIT 1`),
		"E2E_BW_MULTI_SUB":      multi,
		"E2E_BW_MULTI_KEY":      e.one(t, `SELECT source_key FROM biawin.subscription WHERE subscription_id = $1`, multi),
		"E2E_BW_MULTI_ITEMS":    itoa(e.count(t, `SELECT count(*) FROM biawin.subscription_item WHERE subscription_id = $1`, multi)),
		"E2E_BW_PARITY": e.one(t, `SELECT concat_ws(',', legacy_count, target_count, matched, missing_target, extra_target, unknown, conflict,
			not_recoverable, outcome) FROM biawin.parity_check`),
		"E2E_BW_PARITY_ROWS": itoa(e.count(t, `SELECT count(*) FROM biawin.parity_row`)),
	}
	runPlaywright(t, root, "reporting-biawin-read-overlay", env)

	assertReadOnly(t, e, before)
}
