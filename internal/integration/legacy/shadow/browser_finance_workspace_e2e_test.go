//go:build e2e

package shadow_test

// RM-P07 browser E2E: the Finance operational workspace (landing counters,
// review queue with server filters, exception queues, receipt evidence),
// on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the R7-7 class B
// pattern): crm-api runs with CRM_ENV=test and the existing development auth
// stub (CRM_DEV_AUTH=1, refused in production). dev-user-001 holds the
// existing synthetic in-memory read capabilities; dev-user-002 holds none.
// No grant, capability, role or flag default is added. The Finance role is
// NOT live verified: nothing here is real-account evidence.
//
// Data: the R7-2 synthetic payments scenario through the real import path.
// Before the browser starts, three target payments are submitted through
// the REAL Slice 12 submission command (two on 3001, finalised to READY;
// one on 3005, left PENDING_FINALISATION) and one synthetic regression
// history row is added to a legacy attempt. payments.manual_write stays
// OFF, so target receipt diagnostics are BLOCKED while the legacy receipt
// counts are readable (independent sources). The browser never writes: the
// financial snapshot taken after setup must be identical after the run.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

func TestBrowserFinanceWorkspaceReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := setupPayments(t)
	requireLoopbackDSN(t, e.DSN)

	seller := e.actor("6")
	if r := e.submit(t, batch("p07-browser-0001", e.invoices["3001"], 0, seller, item("a", 300, e.upload(t, "6", 94)), item("b", 200, e.upload(t, "6", 95)))); r.Status != result.Success {
		t.Fatalf("submit 3001: %+v", r)
	}
	if rep, err := e.svc.Finalise(t.Context(), 10); err != nil || rep.AttemptsReady < 1 {
		t.Fatalf("finalise: %+v %v", rep, err)
	}
	if r := e.submit(t, batch("p07-browser-0002", e.invoices["3005"], 0, seller, item("a", 100, e.upload(t, "6", 96)))); r.Status != result.Success {
		t.Fatalf("submit 3005: %+v", r)
	}
	legacyAttempt := e.q(t, `SELECT min(attempt_id::text) FROM payments.payment_attempt`)
	if _, err := e.admin.Exec(t.Context(), `INSERT INTO payments.attempt_state_history (history_id, attempt_id, old_state_raw, new_state_raw, regression, batch_id, recorded_at)
		VALUES ('70000000-0000-4000-8000-0000000000e1', $1, 'approved', 'pending', true, '70000000-0000-4000-8000-0000000000e2', '2026-11-03T08:00:00Z')`, legacyAttempt); err != nil {
		t.Fatal(err)
	}

	on := map[string]bool{"billing_payments.read_shadow": true, "finance.review_read": true, "customers.identity_shadow_read": true, "sales.read_shadow": true}
	setFlagsIn(t, e.env, on)
	objects := t.TempDir()
	apiURL := e.startCRMAPI(t, root, map[string]owner.Module{
		"CRM_IDENTITY_READ_DATABASE_URL":         identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL":     organization.Module,
		"CRM_CUSTOMERS_READ_DATABASE_URL":        customers.Module,
		"CRM_CASES_READ_DATABASE_URL":            cases.Module,
		"CRM_CUSTODY_READ_DATABASE_URL":          custody.Module,
		"CRM_BILLING_READ_DATABASE_URL":          billing.Module,
		"CRM_PAYMENTS_READ_DATABASE_URL":         payments.Module,
		"CRM_PAYMENTS_WRITE_DATABASE_URL":        payments.Module,
		"CRM_PAYMENTS_BILLING_READ_DATABASE_URL": billing.Module,
		"CRM_WRITEBACK_READ_DATABASE_URL":        integration.Module,
	}, map[string]string{"CRM_DEV_AUTH": "1", "CRM_PAYMENTS_OBJECT_ROOT": objects})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := financialSnapshot(t, e.admin)
	if m := missingSeed(before, "billing.invoice", "payments.payment_attempt", "payments.submitted_attempt"); len(m) != 0 {
		t.Fatalf("the synthetic scenario is incomplete: %v", m)
	}
	inv := e.invoices["3001"]
	env := map[string]string{
		"E2E_BASE_URL":           web,
		"E2E_FW_INVOICES":        itoa(e.count(t, `SELECT count(*) FROM billing.invoice WHERE presence = 'PRESENT'`)),
		"E2E_FW_ATTEMPTS":        itoa(e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE presence = 'PRESENT'`)),
		"E2E_FW_REVIEW_ALL":      itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW'`)),
		"E2E_FW_REVIEW_READY":    itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW' AND evidence_state = 'READY'`)),
		"E2E_FW_REVIEW_PENDING":  itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW' AND evidence_state = 'PENDING_FINALISATION'`)),
		"E2E_FW_REVIEW_INVOICE":  inv,
		"E2E_FW_REVIEW_INV_N":    itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW' AND invoice_id = $1`, inv)),
		"E2E_FW_REVIEW_KEY":      e.one(t, `SELECT source_spec || ' #' || source_key FROM billing.invoice WHERE invoice_id = $1`, inv),
		"E2E_FW_REGRESSION":      itoa(e.count(t, `SELECT count(*) FROM payments.payment_attempt a WHERE EXISTS (SELECT 1 FROM payments.attempt_state_history h WHERE h.attempt_id = a.attempt_id AND h.regression)`)),
		"E2E_FW_RECEIPTS_REF":    itoa(e.count(t, `SELECT count(*) FROM payments.receipt WHERE presence = 'PRESENT' AND state = 'REFERENCE_ONLY'`)),
		"E2E_FW_UNKNOWN_INVOICE": itoa(e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE presence = 'PRESENT' AND invoice_link_state = 'UNKNOWN'`)),
	}
	// RM-P07 evidence uses its own variable (never the generic E2E_SCREENSHOT_DIR the other suites share).
	if dir := os.Getenv("E2E_FINANCE_SCREENSHOT_DIR"); dir != "" {
		env["E2E_FINANCE_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "finance-workspace-read-overlay", env)

	after := financialSnapshot(t, e.admin)
	for _, d := range diffFinancial(before, after) {
		t.Errorf("%s (the browser suite must not write)", d)
	}
	if n := e.count(t, `SELECT count(*) FROM integration.writeback_command`); n != 0 {
		t.Errorf("integration.writeback_command = %d, want 0", n)
	}
	if n := countFiles(t, objects); n != 0 {
		t.Errorf("evidence store holds %d files, want 0", n)
	}
}
