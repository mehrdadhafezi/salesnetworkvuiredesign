//go:build e2e

package shadow_test

// RM-P02 browser E2E: the invoice 360° workspace, on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the R7-7 class B
// pattern): crm-api runs with CRM_ENV=test and the existing development auth
// stub (CRM_DEV_AUTH=1, refused in production). dev-user-001 holds the
// existing synthetic in-memory read capabilities; dev-user-002 holds none.
// No grant, capability, role or flag default is added.
//
// Data: the R7-2 synthetic payments scenario through the real import path
// (invoices, items, stages, legacy attempts with gateway evidence, receipt
// references). Before the browser starts, three target payments are
// submitted through the REAL Slice 12 submission command (two on invoice
// 3001, one on 3005) so the Finance review section has rows to filter. The
// browser itself never writes: the financial snapshot taken after that
// setup must be identical after the run, and the evidence store crm-api
// receives stays empty.

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

func TestBrowserInvoiceWorkspaceReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := setupPayments(t)
	requireLoopbackDSN(t, e.DSN)

	seller := e.actor("6")
	if r := e.submit(t, batch("p02-browser-0001", e.invoices["3001"], 0, seller, item("a", 300, e.upload(t, "6", 91)), item("b", 200, e.upload(t, "6", 92)))); r.Status != result.Success {
		t.Fatalf("submit 3001: %+v", r)
	}
	if r := e.submit(t, batch("p02-browser-0002", e.invoices["3005"], 0, seller, item("a", 100, e.upload(t, "6", 93)))); r.Status != result.Success {
		t.Fatalf("submit 3005: %+v", r)
	}

	// The invoice with the most legacy attempts, and one attempt with history.
	attInv := e.one(t, `SELECT invoice_id::text FROM payments.payment_attempt WHERE invoice_id IS NOT NULL GROUP BY invoice_id
		ORDER BY count(*) DESC, invoice_id LIMIT 1`)
	histAttempt := e.one(t, `SELECT coalesce(min(h.attempt_id::text), '') FROM payments.attempt_state_history h
		JOIN payments.payment_attempt a USING (attempt_id) WHERE a.invoice_id = $1`, attInv)

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
		"E2E_WS_REVIEW_INVOICE":  inv,
		"E2E_WS_REVIEW_KEY":      e.one(t, `SELECT source_spec || ' #' || source_key FROM billing.invoice WHERE invoice_id = $1`, inv),
		"E2E_WS_REVIEW_N":        itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE invoice_id = $1 AND state = 'PENDING_REVIEW'`, inv)),
		"E2E_WS_REVIEW_ALL":      itoa(e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW'`)),
		"E2E_WS_REVIEW_CUSTOMER": e.one(t, `SELECT customer_link_state FROM billing.invoice WHERE invoice_id = $1`, inv),
		"E2E_WS_REVIEW_CASE":     e.one(t, `SELECT case_link_state FROM billing.invoice WHERE invoice_id = $1`, inv),
		"E2E_WS_ATT_INVOICE":     attInv,
		"E2E_WS_ATT_KEY":         e.one(t, `SELECT source_spec || ' #' || source_key FROM billing.invoice WHERE invoice_id = $1`, attInv),
		"E2E_WS_ATT_N":           itoa(e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE invoice_id = $1 AND presence = 'PRESENT'`, attInv)),
		"E2E_WS_ATT_HISTORY":     histAttempt,
		"E2E_WS_ATT_STAGES":      itoa(e.count(t, `SELECT count(*) FROM billing.payment_stage WHERE invoice_id = $1 AND presence = 'PRESENT'`, attInv)),
		"E2E_WS_ATT_ITEMS":       itoa(e.count(t, `SELECT count(*) FROM billing.invoice_item WHERE invoice_id = $1 AND presence = 'PRESENT'`, attInv)),
	}
	if dir := os.Getenv("E2E_SCREENSHOT_DIR"); dir != "" {
		env["E2E_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "invoice-workspace-read-overlay", env)

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
