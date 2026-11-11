//go:build e2e

package shadow_test

// R7-2 full-stack browser E2E for invoices and payment submission (build tag
// e2e; run by .github/workflows/e2e-ci.yml), on the R7-1 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// The disposable database and the SYNTHETIC Slice 12 payments scenario come
// from the existing test environment (setupPayments: synthetic invoices
// 3001–3008, imported billing and payments). Nothing here adds an endpoint,
// capability, grant, migration or business rule:
//
//   - two synthetic credentials are inserted for synthetic sellers X (the
//     seller of invoice 3001) and Y (another seller), exactly as R7-1 does;
//   - flags are switched only inside this disposable database;
//   - crm-api runs with CRM_ENV=test, real R1 sessions and the production
//     subject resolver: a subject holds only ACTIVE role grants and none exist
//     (D-42; D-34 and the seller capability / read-scope decisions are open).
//     Every invoice read and payment command is therefore refused, and the
//     suite asserts those refusals; it never records a payment.
//
// After both Playwright projects the orchestrator proves that no row of any
// billing, payments or finance table changed, that no write-back command was
// queued and that no receipt byte reached the evidence store.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// paymentFlags are every flag behind the surfaces this suite exercises.
var paymentFlags = []string{"billing_payments.read_shadow", "payments.manual_write", "payments.manual_writeback", "finance.review_read"}

func TestBrowserPaymentsFullStack(t *testing.T) {
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "web", "dist", "index.html")); err != nil {
		t.Fatalf("web/dist is missing (run `npm run build` in web/ first): %v", err)
	}

	e := setupPayments(t) // disposable DB + synthetic payments scenario (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)

	x := e.syntheticCredential(t, "e2e-pay-seller-x", e.users["6"]) // seller of invoice 3001
	y := e.syntheticCredential(t, "e2e-pay-seller-y", e.users["7"]) // another seller

	// Every surface ON (controlled mode: manual_write + manual_writeback),
	// only inside this disposable database.
	on := map[string]bool{"identity.session_auth": true}
	for _, f := range paymentFlags {
		on[f] = true
	}
	e.setFlags(t, on)

	// The evidence store is an empty directory of its own; it must stay empty.
	objects := t.TempDir()
	api := e.startCRMAPI(t, root, map[string]owner.Module{
		"CRM_IDENTITY_AUTH_DATABASE_URL":         identity.Module,
		"CRM_IDENTITY_READ_DATABASE_URL":         identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL":     organization.Module,
		"CRM_BILLING_READ_DATABASE_URL":          billing.Module,
		"CRM_PAYMENTS_READ_DATABASE_URL":         payments.Module,
		"CRM_PAYMENTS_WRITE_DATABASE_URL":        payments.Module,
		"CRM_PAYMENTS_BILLING_READ_DATABASE_URL": billing.Module,
		"CRM_WRITEBACK_READ_DATABASE_URL":        integration.Module,
	}, map[string]string{"CRM_PAYMENTS_OBJECT_ROOT": objects})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), api)

	before := e.financialRows(t)
	if len(before) == 0 {
		t.Fatal("no billing / payments / finance table found: the scenario did not migrate")
	}
	if before["billing.invoice"] == 0 || before["payments.payment_attempt"] == 0 {
		t.Fatalf("the synthetic scenario imported no invoice or payment attempt: %v", before)
	}

	env := map[string]string{
		"E2E_BASE_URL":          web,
		"E2E_SELLER_X_LOGIN":    x.login,
		"E2E_SELLER_X_PASSWORD": x.password,
		"E2E_SELLER_X_USER_ID":  x.userID,
		"E2E_SELLER_Y_LOGIN":    y.login,
		"E2E_SELLER_Y_PASSWORD": y.password,
		"E2E_SELLER_Y_USER_ID":  y.userID,
		"E2E_INVOICE_OF_X":      e.invoices["3001"],
	}
	runPlaywright(t, root, "payments-flags-on", env, x, y)

	// Every payment surface OFF (sessions stay ON), then prove the UI and the
	// API answer the disabled contract.
	off := map[string]bool{}
	for _, f := range paymentFlags {
		off[f] = false
	}
	e.setFlags(t, off)
	runPlaywright(t, root, "payments-flags-off", env, x, y)

	// No unauthorized write: every billing, payments and finance table holds
	// exactly the rows it held before the browser ran.
	after := e.financialRows(t)
	for table, n := range before {
		if after[table] != n {
			t.Errorf("%s: %d rows before the browser suite, %d after (no write may have been applied)", table, n, after[table])
		}
	}
	for table := range after {
		if _, ok := before[table]; !ok {
			t.Errorf("%s appeared during the browser suite", table)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM integration.writeback_command`); n != 0 {
		t.Errorf("integration.writeback_command = %d, want 0 (no legacy write-back may be queued)", n)
	}
	if n := countFiles(t, objects); n != 0 {
		t.Errorf("evidence store holds %d files, want 0 (no receipt may have been stored)", n)
	}
}

// financialRows counts the rows of every base table in the billing, payments
// and finance schemas.
func (e *paymentsEnv) financialRows(t *testing.T) map[string]int64 {
	t.Helper()
	ctx := context.Background()
	rows, err := e.admin.Query(ctx, `SELECT table_schema || '.' || table_name FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND table_schema IN ('billing', 'payments', 'finance') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	out := make(map[string]int64, len(tables))
	for _, n := range tables {
		// Names come from information_schema of the disposable database.
		out[n] = e.count(t, `SELECT count(*) FROM `+n)
	}
	return out
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	if err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return n
}
