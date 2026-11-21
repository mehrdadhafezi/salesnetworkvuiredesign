//go:build e2e

package shadow_test

// RM-P01 browser E2E: Customer ⇄ Case ⇄ Invoice read navigation, on the
// R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE (the R7-7 class B
// pattern): crm-api runs with CRM_ENV=test and the existing development auth
// stub (CRM_DEV_AUTH=1, refused in production). dev-user-001 holds the
// existing synthetic in-memory read capabilities; dev-user-002 holds none.
// No grant, capability, role or flag default is added.
//
// Data: the R7-5 integrated SYNTHETIC dataset through the real import path.
// That import links invoices to customers and to Cases (RESOLVED) but, by
// its own invariant no_resolved_customer_link, never links a Case to a
// customer. So that the direct Customer → Case step is exercised, ONE
// clearly synthetic Case (source spec synthetic.rm_p01.v1) RESOLVED to the
// chosen customer is inserted before the read-only snapshot; every other
// row comes from the import. Every expected value is the orchestrator's own
// SQL over the disposable database (rendered UI == API answer == database).

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

var c360Flags = []string{"customers.identity_shadow_read", "sales.read_shadow", "billing_payments.read_shadow"}

func c360Logins() map[string]owner.Module {
	return map[string]owner.Module{
		"CRM_IDENTITY_READ_DATABASE_URL":     identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL": organization.Module,
		"CRM_CUSTOMERS_READ_DATABASE_URL":    customers.Module,
		"CRM_CASES_READ_DATABASE_URL":        cases.Module,
		"CRM_CUSTODY_READ_DATABASE_URL":      custody.Module,
		"CRM_BILLING_READ_DATABASE_URL":      billing.Module,
		"CRM_PAYMENTS_READ_DATABASE_URL":     payments.Module,
	}
}

func TestBrowserCustomer360ReadOverlay(t *testing.T) {
	root := repoRoot(t)
	requireDist(t, root)
	e := seedR77(t)

	// An imported invoice RESOLVED to both a customer and a Case.
	inv := e.one(t, `SELECT coalesce(min(invoice_id::text), '') FROM billing.invoice
		WHERE customer_link_state = 'RESOLVED' AND case_link_state = 'RESOLVED' AND presence = 'PRESENT'`)
	if inv == "" {
		t.Fatal("the integrated dataset needs an invoice RESOLVED to a customer and a Case")
	}
	cust := e.one(t, `SELECT customer_id::text FROM billing.invoice WHERE invoice_id = $1`, inv)
	kase := e.one(t, `SELECT case_id::text FROM billing.invoice WHERE invoice_id = $1`, inv)
	if e.count(t, `SELECT count(*) FROM customers.customer WHERE customer_id = $1`, cust) != 1 ||
		e.count(t, `SELECT count(*) FROM cases.work_case WHERE case_id = $1`, kase) != 1 {
		t.Fatal("the invoice's RESOLVED customer / Case must exist in their own modules")
	}
	if n := e.count(t, `SELECT count(*) FROM cases.work_case WHERE customer_link_state = 'RESOLVED'`); n != 0 {
		t.Fatalf("the import's no_resolved_customer_link invariant changed (%d); revisit this fixture", n)
	}
	// An invoice whose customer link is not RESOLVED (shown, never linked).
	unlinked := e.one(t, `SELECT coalesce(min(invoice_id::text), '') FROM billing.invoice WHERE customer_link_state <> 'RESOLVED'`)
	if unlinked == "" {
		t.Fatal("the integrated dataset needs an invoice with an unresolved customer link")
	}

	// The one synthetic Case RESOLVED to cust (see the file comment).
	synth := uuid.NewString()
	sha := sha256.Sum256([]byte("rm-p01-browser"))
	batch := uuid.NewString()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO cases.work_case (case_id, workflow_type, state_namespace, state_raw, state_mapping, customer_id, customer_link_state,
			customer_link_reason, presence, imported_at, refreshed_at, version)
			VALUES ($1, 'LEGACY_LEAD', 'synthetic.rm_p01', 'new', 'KNOWN', $2, 'RESOLVED', 'synthetic RM-P01 browser fixture', 'PRESENT', now(), now(), 1)`,
			[]any{synth, cust}},
		{`INSERT INTO cases.case_source (case_source_id, case_id, source_system, source_spec, source_key, first_batch_id, last_batch_id, row_sha256,
			presence, first_seen_at, last_seen_at) VALUES ($1, $2, 'synthetic', 'synthetic.rm_p01.v1', '1', $3, $3, $4, 'PRESENT', now(), now())`,
			[]any{uuid.NewString(), synth, batch, sha[:]}},
	} {
		if _, err := e.admin.Exec(t.Context(), q.sql, q.args...); err != nil {
			t.Fatalf("synthetic fixture: %v", err)
		}
	}

	on := map[string]bool{}
	for _, f := range c360Flags {
		on[f] = true
	}
	setFlagsIn(t, e.env, on)
	apiURL := e.startCRMAPI(t, root, c360Logins(), map[string]string{"CRM_DEV_AUTH": "1"})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), apiURL)

	before := schemaSnapshot(t, e.admin, rbGuarded)
	env := map[string]string{
		"E2E_BASE_URL":         web,
		"E2E_C360_CUSTOMER":    cust,
		"E2E_C360_SYNTH_CASE":  synth,
		"E2E_C360_CASE":        kase,
		"E2E_C360_CASE_KEY":    e.one(t, `SELECT source_spec || ' #' || source_key FROM cases.case_source WHERE case_id = $1`, kase),
		"E2E_C360_CASE_LINK":   e.one(t, `SELECT customer_link_state FROM cases.work_case WHERE case_id = $1`, kase),
		"E2E_C360_INVOICE":     inv,
		"E2E_C360_INVOICE_KEY": e.one(t, `SELECT source_spec || ' #' || source_key FROM billing.invoice WHERE invoice_id = $1`, inv),
		"E2E_C360_CUSTOMER_INVOICES": itoa(e.count(t, `SELECT count(*) FROM billing.invoice WHERE customer_link_state = 'RESOLVED'
			AND customer_id = $1`, cust)),
		"E2E_C360_CASE_INVOICES":          itoa(e.count(t, `SELECT count(*) FROM billing.invoice WHERE case_link_state = 'RESOLVED' AND case_id = $1`, kase)),
		"E2E_C360_UNLINKED_INVOICE":       unlinked,
		"E2E_C360_UNLINKED_CUSTOMER_LINK": e.one(t, `SELECT customer_link_state FROM billing.invoice WHERE invoice_id = $1`, unlinked),
	}
	if dir := os.Getenv("E2E_SCREENSHOT_DIR"); dir != "" {
		env["E2E_SCREENSHOT_DIR"] = dir
	}
	runPlaywright(t, root, "customer-360-read-overlay", env)

	assertReadOnly(t, e, before)
}
