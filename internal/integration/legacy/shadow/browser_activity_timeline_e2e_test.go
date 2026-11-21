//go:build e2e

package shadow_test

// RM-P03 browser E2E: the unified, read-only activity timeline on the Case
// and Customer pages, on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the R7-7 class B
// pattern, as RM-P01): crm-api runs with CRM_ENV=test and the existing
// development auth stub (CRM_DEV_AUTH=1, refused in production).
// dev-user-001 holds the existing synthetic in-memory read capabilities;
// dev-user-002 holds none. No grant, capability, role or flag default is
// added; the contact outcome write flag stays off, so the page must not
// request contact outcomes at all.
//
// Data: the R7-5 integrated SYNTHETIC dataset through the real import path,
// unchanged. Every expected value is the orchestrator's own SQL over the
// disposable database (rendered UI == API answer == database), and the
// whole run must leave every guarded schema unchanged.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserActivityTimelineReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	// The Case with the most state history rows (ties: lowest id).
	kase := e.one(t, `SELECT coalesce((SELECT case_id::text FROM cases.case_state_history GROUP BY case_id
		ORDER BY count(*) DESC, case_id LIMIT 1), '')`)
	if kase == "" {
		t.Fatal("the integrated dataset needs a Case with state history")
	}
	// The customer with the most invoices RESOLVED to it (ties: lowest id).
	cust := e.one(t, `SELECT coalesce((SELECT customer_id::text FROM billing.invoice WHERE customer_link_state = 'RESOLVED'
		GROUP BY customer_id ORDER BY count(*) DESC, customer_id LIMIT 1), '')`)
	if cust == "" {
		t.Fatal("the integrated dataset needs an invoice RESOLVED to a customer")
	}
	// A RESOLVED invoice of another customer must never appear on cust's timeline.
	foreign := e.one(t, `SELECT coalesce(min(source_spec || ' #' || source_key), '') FROM billing.invoice
		WHERE customer_link_state = 'RESOLVED' AND customer_id <> $1`, cust)

	on := map[string]bool{}
	for _, f := range c360Flags {
		on[f] = true
	}
	setFlagsIn(t, e.env, on)
	apiURL := e.startCRMAPI(t, root, c360Logins(), map[string]string{"CRM_DEV_AUTH": "1"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, rbGuarded)
	env := map[string]string{
		"E2E_BASE_URL":            web,
		"E2E_TL_CASE":             kase,
		"E2E_TL_CASE_KEY":         e.one(t, `SELECT source_spec || ' #' || source_key FROM cases.case_source WHERE case_id = $1`, kase),
		"E2E_TL_CASE_HISTORY":     itoa(e.count(t, `SELECT count(*) FROM cases.case_state_history WHERE case_id = $1`, kase)),
		"E2E_TL_CASE_CUSTODY":     itoa(e.count(t, `SELECT count(*) FROM custody.custody_event WHERE case_id = $1`, kase)),
		"E2E_TL_CASE_ATTRIBUTION": itoa(e.count(t, `SELECT count(*) FROM cases.attribution_snapshot WHERE case_id = $1`, kase)),
		"E2E_TL_CASE_FIRST_RAW": e.one(t, `SELECT coalesce(nullif(old_state_raw, ''), '—') || ' → ' || coalesce(nullif(new_state_raw, ''), '—') FROM cases.case_state_history
			WHERE case_id = $1 ORDER BY source_ordinal LIMIT 1`, kase),
		"E2E_TL_CUSTOMER":          cust,
		"E2E_TL_CUSTOMER_INVOICES": itoa(e.count(t, `SELECT count(*) FROM billing.invoice WHERE customer_link_state = 'RESOLVED' AND customer_id = $1`, cust)),
		"E2E_TL_FOREIGN_INVOICE":   foreign,
	}
	if dir := os.Getenv("E2E_SCREENSHOT_DIR"); dir != "" {
		env["E2E_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "activity-timeline-read-overlay", env)

	assertReadOnly(t, e, before)
}
