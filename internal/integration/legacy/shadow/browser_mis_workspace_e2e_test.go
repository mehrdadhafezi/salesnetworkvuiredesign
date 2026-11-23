//go:build e2e

package shadow_test

// RM-P05 browser E2E: the MIS operational workspace (landing counters, case
// inventory, custody evidence, exceptions queue) on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the RM-P01..P03
// pattern): crm-api runs with CRM_ENV=test and the existing development auth
// stub (CRM_DEV_AUTH=1, refused in production). dev-user-001 holds the
// existing synthetic in-memory read capabilities; dev-user-002 holds none.
// The MIS role carries no capability: nothing here is real-account evidence.
// No grant, capability, role or flag default is added.
//
// Data: the R7-5 integrated SYNTHETIC dataset, unchanged. Every expected value
// is the orchestrator's own SQL (rendered UI == API answer == database), and
// the run must leave every guarded schema unchanged.

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserMISWorkspaceReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	// A present projection whose holder evidence conflicts or is unknown, with events (ties: lowest id).
	exc := e.one(t, `SELECT coalesce((SELECT a.case_id::text FROM custody.assignment a WHERE a.presence = 'PRESENT' AND a.parity IN ('CONFLICT', 'MULTIPLE_CURRENT', 'TARGET_UNKNOWN', 'NOT_RECOVERABLE', 'HISTORY_GAP')
		AND a.event_count > 0 ORDER BY a.case_id LIMIT 1), '')`)
	if exc == "" {
		t.Fatal("the integrated dataset needs a present non-MATCH custody projection with events")
	}
	parity := e.one(t, `SELECT parity FROM custody.assignment WHERE case_id = $1`, exc)
	key := e.one(t, `SELECT source_key FROM cases.case_source WHERE case_id = $1`, exc)

	on := map[string]bool{}
	for _, f := range append(append([]string{}, c360Flags...), rbFlags...) {
		on[f] = true
	}
	setFlagsIn(t, e.env, on)
	logins := c360Logins()
	maps.Copy(logins, rbLogins(false))
	apiURL := e.startCRMAPI(t, root, logins, map[string]string{"CRM_DEV_AUTH": "1"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, rbGuarded)
	env := map[string]string{
		"E2E_BASE_URL":             web,
		"E2E_MIS_CASES_TOTAL":      itoa(e.count(t, `SELECT count(*) FROM cases.work_case`)),
		"E2E_MIS_CASES_PRESENT":    itoa(e.count(t, `SELECT count(*) FROM cases.work_case WHERE presence = 'PRESENT'`)),
		"E2E_MIS_CUSTODY_EVENTS":   itoa(e.count(t, `SELECT count(*) FROM custody.custody_event`)),
		"E2E_MIS_EXC_CASE":         exc,
		"E2E_MIS_EXC_PARITY":       parity,
		"E2E_MIS_EXC_PARITY_COUNT": itoa(e.count(t, `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND parity = $1`, parity)),
		"E2E_MIS_EXC_KEY":          key,
		"E2E_MIS_EXC_KEY_CASES":    itoa(e.count(t, `SELECT count(*) FROM cases.case_source WHERE source_key = $1`, key)),
		"E2E_MIS_EXC_EVENTS":       itoa(e.count(t, `SELECT count(*) FROM custody.custody_event WHERE case_id = $1`, exc)),
		"E2E_MIS_EXC_ATTRIBUTION":  itoa(e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE case_id = $1`, exc)),
		"E2E_MIS_UNRESOLVED": itoa(e.count(t, `SELECT count(*) FROM custody.assignment
			WHERE (derived_legacy_key IS NOT NULL AND derived_user_id IS NULL) OR (legacy_holder_key IS NOT NULL AND legacy_holder_user_id IS NULL)`)),
	}
	if dir := os.Getenv("E2E_SCREENSHOT_DIR"); dir != "" {
		env["E2E_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "mis-workspace-read-overlay", env)

	assertReadOnly(t, e, before)
}
