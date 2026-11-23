//go:build e2e

package shadow_test

// RM-P06 browser E2E: MIS case-to-custody evidence continuity on the R7-1
// harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the RM-P01..P05
// pattern): crm-api runs with CRM_ENV=test and the existing development auth
// stub (CRM_DEV_AUTH=1, refused in production). dev-user-001 holds the
// existing synthetic in-memory read capabilities; dev-user-002 holds none.
// The "cases denied, custody allowed" case is produced in the browser by
// answering GET /api/v1/cases with the server's own UNAUTHORIZED envelope
// (no grant, capability, role or flag is added for it).
//
// Data: the R7-5 integrated SYNTHETIC dataset, unchanged. Every expected
// value is the orchestrator's own SQL, and the run must leave every guarded
// schema unchanged.

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserMISEvidenceReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	// A present non-MATCH projection with events (ties: lowest id), and its own source row.
	exc := e.one(t, `SELECT coalesce((SELECT a.case_id::text FROM custody.assignment a WHERE a.presence = 'PRESENT' AND a.parity IN ('CONFLICT', 'MULTIPLE_CURRENT', 'TARGET_UNKNOWN', 'NOT_RECOVERABLE', 'HISTORY_GAP')
		AND a.event_count > 0 ORDER BY a.case_id LIMIT 1), '')`)
	if exc == "" {
		t.Fatal("the integrated dataset needs a present non-MATCH custody projection with events")
	}
	// The first custody page (keyset on case_id, 25 rows) as the database orders it.
	firstPage := e.one(t, `SELECT string_agg(case_id::text, ',' ORDER BY case_id) FROM (SELECT case_id FROM custody.assignment ORDER BY case_id LIMIT 25) p`)
	withCase := e.count(t, `SELECT count(*) FROM (SELECT case_id FROM custody.assignment ORDER BY case_id LIMIT 25) p
		WHERE EXISTS (SELECT 1 FROM cases.work_case c WHERE c.case_id = p.case_id)`)

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
		"E2E_BASE_URL":              web,
		"E2E_MIS_EXC_CASE":          exc,
		"E2E_MIS_EXC_PARITY":        e.one(t, `SELECT parity FROM custody.assignment WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_SYSTEM":        e.one(t, `SELECT source_system FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_SPEC":          e.one(t, `SELECT source_spec FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_KEY":           e.one(t, `SELECT source_key FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_SRC_PRESENCE":  e.one(t, `SELECT presence FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_FIRST_BATCH":   e.one(t, `SELECT first_batch_id::text FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_EXC_LAST_BATCH":    e.one(t, `SELECT last_batch_id::text FROM cases.case_source WHERE case_id = $1`, exc),
		"E2E_MIS_CUSTODY_PAGE":      firstPage,
		"E2E_MIS_CUSTODY_PAGE_CASE": itoa(withCase),
	}
	if dir := os.Getenv("E2E_SCREENSHOT_DIR"); dir != "" {
		env["E2E_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "mis-evidence-read-overlay", env)

	assertReadOnly(t, e, before)
}
