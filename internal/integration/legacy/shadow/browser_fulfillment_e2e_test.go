//go:build e2e

package shadow_test

// R7-3 full-stack browser E2E for the Slice 14 fulfillment work cases and the
// provider-operation boundary (build tag e2e; run by
// .github/workflows/e2e-ci.yml), on the R7-1 / R7-2 harness:
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// The disposable database and the SYNTHETIC Slice 12 payments scenario come
// from setupPayments (so the R7-2 financial no-mutation guard applies
// unchanged). The work cases are created through the production path only:
// a synthetic catalog recording (registry.Record) → a SHADOW biawin
// activation of a synthetic fully-paid message → the relay → the
// fulfillment intake consumer. One provider operation is requested in the
// integration ledger for the provider-gated case and never dispatched: no
// port exists in this process, nothing is confirmed and no provider is
// called. Nothing here adds an endpoint, capability, grant, migration or
// business rule:
//
//   - two synthetic credentials are inserted for synthetic users X and Y,
//     exactly as R7-1 / R7-2 do; the catalog scopes make X the CURRENT
//     supervisor and executor of services A and P and Y those of service B;
//   - flags are switched only inside this disposable database;
//   - crm-api runs with CRM_ENV=test, real R1 sessions and the production
//     subject resolver: a subject holds only ACTIVE role grants and none exist
//     (D-42; D-34 and R5-SEC-01 are open), and no role bundle carries a
//     fulfillment capability. Every work-case read, command, provider read
//     and reconciliation is therefore refused, and the suite asserts those
//     refusals; it never assigns, records, attaches or completes anything.
//
// After the three Playwright projects (flags ON, mixed, OFF) the orchestrator proves that no billing,
// payments, finance, fulfillment, biawin, catalog or integration row changed
// (row count plus a SHA-256 fingerprint of every row; see
// financial_snapshot_test.go), that no write-back command was queued, that the
// provider operation is still un-dispatched and that no work case moved.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/activation"
	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	catalogmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/registry"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// operationsFlags are every flag behind the Slice 14 surfaces this suite
// exercises (read, work-case commands in CONTROLLED mode, reconciliation).
var operationsFlags = []string{"fulfillment.work_read", "fulfillment.work_write", "fulfillment.work_writeback", "integration.provider_reconcile"}

// guardedSchemas are snapshotted before and after the browser runs: the R7-2
// financial schemas plus every schema a work-case or provider command could
// write (cases, history, evidence, idempotency, inbox / outbox, ledger).
var guardedSchemas = []string{"billing", "payments", "finance", "fulfillment", "biawin", "catalog", "integration"}

func TestBrowserFulfillmentFullStack(t *testing.T) {
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "web", "dist", "index.html")); err != nil {
		t.Fatalf("web/dist is missing (run `npm run build` in web/ first): %v", err)
	}

	e := setupPayments(t) // disposable DB + synthetic payments scenario (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)
	if _, err := migrations.Up(context.Background(), e.DSN, catalogmigrations.Source()); err != nil {
		t.Fatal(err)
	}

	x := e.syntheticCredential(t, "e2e-ops-user-x", e.users["6"]) // supervisor + executor of services A and P
	y := e.syntheticCredential(t, "e2e-ops-user-y", e.users["7"]) // supervisor + executor of service B
	sc := e.operationsScenario(t, x.userID, y.userID)

	on := map[string]bool{"identity.session_auth": true}
	for _, f := range operationsFlags {
		on[f] = true
	}
	e.setFlags(t, on)

	api := e.startCRMAPI(t, root, map[string]owner.Module{
		"CRM_IDENTITY_AUTH_DATABASE_URL":     identity.Module,
		"CRM_IDENTITY_READ_DATABASE_URL":     identity.Module,
		"CRM_ORGANIZATION_READ_DATABASE_URL": organization.Module,
		"CRM_OPS_FULFILLMENT_DATABASE_URL":   fulfillment.Module,
		"CRM_OPS_BIAWIN_DATABASE_URL":        biawin.Module,
		"CRM_OPS_CATALOG_DATABASE_URL":       catalog.Module,
		"CRM_OPS_INTEGRATION_DATABASE_URL":   integration.Module,
		"CRM_WRITEBACK_READ_DATABASE_URL":    integration.Module,
	}, map[string]string{
		// The smallest lifetime session.Config.Validate accepts, through the
		// supported R1 overrides: the browser suite observes a real server-side
		// expiry (no session row is edited). Every other test logs in afresh
		// and finishes well inside one minute.
		"CRM_AUTH_SESSION_IDLE":     "1m",
		"CRM_AUTH_SESSION_ABSOLUTE": "1m",
	})
	web := serveSPA(t, filepath.Join(root, "web", "dist"), api)

	before := schemaSnapshot(t, e.admin, guardedSchemas)
	if m := missingSeed(before, "billing.invoice", "payments.payment_attempt", "fulfillment.work_case", "fulfillment.work_event",
		"integration.provider_operation"); len(m) != 0 {
		t.Fatalf("the synthetic scenario is incomplete: %v", m)
	}
	cases := e.workCaseStates(t)

	env := map[string]string{
		"E2E_BASE_URL":           web,
		"E2E_SELLER_X_LOGIN":     x.login,
		"E2E_SELLER_X_PASSWORD":  x.password,
		"E2E_SELLER_X_USER_ID":   x.userID,
		"E2E_SELLER_Y_LOGIN":     y.login,
		"E2E_SELLER_Y_PASSWORD":  y.password,
		"E2E_SELLER_Y_USER_ID":   y.userID,
		"E2E_WORK_CASE_A":        sc.caseA,
		"E2E_WORK_CASE_B":        sc.caseB,
		"E2E_WORK_CASE_PROVIDER": sc.caseP,
		"E2E_PROVIDER_OPERATION": sc.operation,
		"E2E_WORK_SERVICE_A":     sc.serviceA,
		"E2E_WORK_SERVICE_B":     sc.serviceB,
		"E2E_WORK_SERVICE_P":     sc.serviceP,
	}
	runPlaywright(t, root, "fulfillment-flags-on", env, x, y)

	// Mixed: reads ON, the command flag OFF, the write-back flag ON on its own
	// (it must enable nothing) and reconciliation OFF. Proves the feature gate
	// and the capability gate answer independently per surface.
	e.setFlags(t, map[string]bool{"fulfillment.work_read": true, "fulfillment.work_write": false,
		"fulfillment.work_writeback": true, "integration.provider_reconcile": false})
	runPlaywright(t, root, "fulfillment-flags-mixed", env, x, y)

	// Every Slice 14 flag OFF (sessions stay ON), then prove the UI and the
	// API answer the disabled contract.
	off := map[string]bool{}
	for _, f := range operationsFlags {
		off[f] = false
	}
	e.setFlags(t, off)
	runPlaywright(t, root, "fulfillment-flags-off", env, x, y)

	// No unauthorized write: every guarded table holds exactly the rows it
	// held before the browser ran, with the same content. The report names
	// tables and counts only.
	after := schemaSnapshot(t, e.admin, guardedSchemas)
	for _, d := range diffFinancial(before, after) {
		t.Errorf("%s (no write may have been applied by the browser suite)", d)
	}
	for id, st := range e.workCaseStates(t) {
		if cases[id] != st {
			t.Errorf("work case %s moved from %q to %q (no command may have been applied)", id, cases[id], st)
		}
	}
	for _, q := range []string{
		`SELECT count(*) FROM integration.writeback_command`,
		`SELECT count(*) FROM fulfillment.work_command_record`,
		`SELECT count(*) FROM fulfillment.work_assignment`,
		`SELECT count(*) FROM fulfillment.work_evidence`,
		`SELECT count(*) FROM fulfillment.work_case WHERE state <> 'QUEUED' OR assignee_user_id IS NOT NULL OR version <> 1`,
		`SELECT count(*) FROM integration.provider_operation WHERE state <> 'PREPARED' OR attempts <> 0 OR external_ref IS NOT NULL`,
	} {
		if n := e.count(t, q); n != 0 {
			t.Errorf("%s = %d, want 0", q, n)
		}
	}
}

// opsScenario names the synthetic work cases, services and the operation.
type opsScenario struct {
	serviceA, serviceB, serviceP string
	caseA, caseB, caseP          string
	operation                    string
}

// operationsScenario creates the synthetic work cases through the production
// path (catalog recording → SHADOW activation → relay → fulfillment intake)
// and requests, without dispatching, one provider operation for the
// provider-gated case.
func (e *paymentsEnv) operationsScenario(t *testing.T, x, y string) opsScenario {
	t.Helper()
	ctx := context.Background()
	clock := timeutil.FixedClock{At: mustInstant(t, "2026-11-04T09:00:00Z")}
	cat, err := registry.New(e.Runner(t, catalog.Module), clock)
	if err != nil {
		t.Fatal(err)
	}
	bR := e.Runner(t, biawin.Module)
	act, err := activation.New(bR, cat, clock)
	if err != nil {
		t.Fatal(err)
	}
	ops, err := providerops.New(e.Runner(t, integration.Module), clock)
	if err != nil {
		t.Fatal(err)
	}
	w, err := work.New(e.Runner(t, fulfillment.Module), cat, ops, clock)
	if err != nil {
		t.Fatal(err)
	}

	sc := opsScenario{serviceA: "e2e-ops-a.svc-01", serviceB: "e2e-ops-b.svc-01", serviceP: "e2e-ops-p.svc-01"}
	rec := registry.Recording{ApprovalRef: "SYNTHETIC-E2E-R7-3", RecordedBy: "test:e2e-r7-3"}
	for _, s := range []struct {
		product, key, user, provider string
	}{
		{"e2e-ops-a", sc.serviceA, x, catalogapi.ProviderNone},
		{"e2e-ops-b", sc.serviceB, y, catalogapi.ProviderNone},
		{"e2e-ops-p", sc.serviceP, x, catalogapi.ProviderBiawin},
	} {
		rec.Services = append(rec.Services, catalogapi.ServiceVersion{ServiceKey: s.key, Version: 1, Title: "Synthetic E2E service " + s.key,
			WorkflowKey: catalogapi.WorkflowOperationalTaskV1, Evidence: []catalogapi.EvidenceRequirement{{Type: "service-form", Required: true}},
			CompletionOutcomes: []string{catalogapi.OutcomeExecuted, catalogapi.OutcomeNotExecuted}, FinancialBehaviour: catalogapi.FinancialNone,
			IntegrationProvider: s.provider, Status: "ACTIVE"})
		rec.Components = append(rec.Components, registry.ComponentRecord{ProductKey: s.product, ProductVersion: 1,
			Component: catalogapi.Component{Seq: 1, ServiceKey: s.key, ServiceVersion: 1}})
		rec.Scopes = append(rec.Scopes,
			registry.ScopeRecord{ServiceKey: s.key, UserID: s.user, Kind: catalogapi.ScopeSupervisor, Action: "GRANT"},
			registry.ScopeRecord{ServiceKey: s.key, UserID: s.user, Kind: catalogapi.ScopeExecutor, Action: "GRANT"})
	}
	if err := cat.Record(ctx, rec); err != nil {
		t.Fatal(err)
	}

	for _, product := range []string{"e2e-ops-a", "e2e-ops-b", "e2e-ops-p"} {
		customer := newID(t)
		ref := "synthetic-e2e-invoice-" + product
		rep, err := act.HandleInvoiceFullyPaid(ctx, rollout.Shadow, biawinapi.InvoiceFullyPaid{MessageID: newID(t), CorrelationID: "corr-e2e-" + product,
			InvoiceID: newID(t), InvoiceRef: &ref,
			Customer: biawinapi.Customer{State: "RESOLVED", ID: &customer, Basis: "synthetic E2E customer resolution"},
			Route:    biawinapi.Route{Branch: biawinapi.RouteAdditionalServices, Basis: "synthetic E2E lifecycle branch"},
			Lines:    []biawinapi.ProductLine{{ProductKey: product, ProductVersion: 1}}, PaidAt: time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		if rep.Refused != nil || len(rep.Lines) != 1 {
			t.Fatalf("synthetic activation of %s was not recorded", product)
		}
	}
	route := w.IntakeRoute()
	route.Backoff = &outbox.Backoff{} // fixed clock: redeliver immediately
	rep, err := relay.Drain(ctx, bR, clock, 500, route)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 0 || rep.NoRoute != 0 {
		t.Fatalf("relay: %+v", rep)
	}

	caseOf := func(service string) string {
		var id string
		if err := e.admin.QueryRow(ctx, `SELECT case_id::text FROM fulfillment.work_case WHERE service_key = $1`, service).Scan(&id); err != nil {
			t.Fatalf("no work case for %s", service)
		}
		return id
	}
	sc.caseA, sc.caseB, sc.caseP = caseOf(sc.serviceA), caseOf(sc.serviceB), caseOf(sc.serviceP)

	// Requested only: PREPARED, never dispatched (no port), never confirmed.
	op, _, err := ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:" + sc.caseP + "|op:SERVICE_ORDER", Provider: "BIAWIN",
		OpType: providerops.OpServiceOrder, SubjectCaseID: &sc.caseP, CorrelationID: "corr-e2e-r7-3", CausationID: "e2e-r7-3",
		Payload: map[string]string{"service_key": sc.serviceP}})
	if err != nil {
		t.Fatal(err)
	}
	if op.State != providerops.StatePrepared {
		t.Fatalf("synthetic provider operation state %s, want %s", op.State, providerops.StatePrepared)
	}
	sc.operation = op.OperationID
	return sc
}

// workCaseStates maps each work case to its state, assignee and version.
func (e *paymentsEnv) workCaseStates(t *testing.T) map[string]string {
	t.Helper()
	rows, err := e.admin.Query(context.Background(), `SELECT case_id::text, state || '|' || coalesce(assignee_user_id::text, '-') || '|v' || version
		FROM fulfillment.work_case`)
	if err != nil {
		t.Fatal("read work cases failed")
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, st string
		if err := rows.Scan(&id, &st); err != nil {
			t.Fatal("read work cases failed")
		}
		out[id] = st
	}
	return out
}

func newID(t *testing.T) string {
	t.Helper()
	u, err := ids.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustInstant(t *testing.T, s string) timeutil.Instant {
	t.Helper()
	at, err := timeutil.ParseRFC3339(s)
	if err != nil {
		t.Fatal(err)
	}
	return at
}
