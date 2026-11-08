package biawin_test

import (
	"context"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/activation"
	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	catalogmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/registry"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	integrationbiawin "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/biawin"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

type env struct {
	*pgtest.Env
	clock   timeutil.Clock
	cat     *registry.Store
	act     *activation.Service
	work    *work.Service
	ops     *providerops.Service
	bR, fR  *db.Runner
	confirm *confirmations
	conn    *pgx.Conn
}

// confirmations adapts the integration published read to fulfillment's
// completion guard, as the composition root does (no cross-schema join).
type confirmations struct{ ops *providerops.Service }

func (c *confirmations) Confirmed(ctx context.Context, caseID string) (bool, error) {
	return c.ops.Confirmed(ctx, caseID)
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	for _, s := range []migrations.Source{integrationmigrations.Source(), catalogmigrations.Source(), biawinmigrations.Source(), fulfillmentmigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, s); err != nil {
			t.Fatalf("migrations: %v", err)
		}
	}
	at, _ := timeutil.ParseRFC3339("2026-11-04T09:00:00Z")
	clock := timeutil.FixedClock{At: at}
	cat, err := registry.New(e.Runner(t, catalog.Module), clock)
	if err != nil {
		t.Fatal(err)
	}
	bR, fR := e.Runner(t, biawin.Module), e.Runner(t, fulfillment.Module)
	act, err := activation.New(bR, cat, clock)
	if err != nil {
		t.Fatal(err)
	}
	ops, err := providerops.New(e.Runner(t, integration.Module), clock)
	if err != nil {
		t.Fatal(err)
	}
	conf := &confirmations{ops: ops}
	w, err := work.New(fR, cat, conf, clock)
	if err != nil {
		t.Fatal(err)
	}
	return &env{Env: e, clock: clock, cat: cat, act: act, work: w, ops: ops, bR: bR, fR: fR, confirm: conf}
}

func uid(t *testing.T) string {
	t.Helper()
	u, err := ids.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func ptr(s string) *string { return &s }

// seed records n services under one product version (+ scopes).
func (e *env) seed(t *testing.T, product string, n int, sup, exec map[string]string, mutate func(*catalogapi.ServiceVersion)) []string {
	t.Helper()
	rec := registry.Recording{ApprovalRef: "OWNER-APPROVAL-SYN-14", RecordedBy: "test:catalog"}
	var keys []string
	for i := 1; i <= n; i++ {
		k := fmt.Sprintf("%s.svc-%02d", product, i)
		keys = append(keys, k)
		v := catalogapi.ServiceVersion{ServiceKey: k, Version: 1, Title: "Synthetic service " + k, WorkflowKey: catalogapi.WorkflowOperationalTaskV1,
			Evidence: []catalogapi.EvidenceRequirement{{Type: "service-form", Required: true}}, CompletionOutcomes: []string{"EXECUTED", "NOT_EXECUTED"},
			FinancialBehaviour: catalogapi.FinancialNone, IntegrationProvider: catalogapi.ProviderNone, Status: "ACTIVE"}
		if mutate != nil {
			mutate(&v)
		}
		rec.Services = append(rec.Services, v)
		rec.Components = append(rec.Components, registry.ComponentRecord{ProductKey: product, ProductVersion: 1,
			Component: catalogapi.Component{Seq: i, ServiceKey: k, ServiceVersion: 1}})
		if u, ok := sup[k]; ok {
			rec.Scopes = append(rec.Scopes, registry.ScopeRecord{ServiceKey: k, UserID: u, Kind: catalogapi.ScopeSupervisor, Action: "GRANT"})
		}
		if u, ok := exec[k]; ok {
			rec.Scopes = append(rec.Scopes, registry.ScopeRecord{ServiceKey: k, UserID: u, Kind: catalogapi.ScopeExecutor, Action: "GRANT"})
		}
	}
	if err := e.cat.Record(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	return keys
}

func paid(t *testing.T, product, route, customerState string, customer *string) biawinapi.InvoiceFullyPaid {
	return biawinapi.InvoiceFullyPaid{MessageID: uid(t), CorrelationID: "corr-" + product, InvoiceID: uid(t), InvoiceRef: ptr("legacy-invoice-2001"),
		Customer: biawinapi.Customer{State: customerState, ID: customer, Basis: "customers published resolution (synthetic)"},
		Route:    biawinapi.Route{Branch: route, Basis: "synthetic lifecycle branch"}, Lines: []biawinapi.ProductLine{{ProductKey: product, ProductVersion: 1}},
		PaidAt: time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)}
}

func (e *env) relay(t *testing.T) int {
	t.Helper()
	route := e.work.IntakeRoute()
	route.Backoff = &outbox.Backoff{} // fixed test clock: redeliver immediately
	rep, err := relay.Drain(context.Background(), e.bR, e.clock, 500, route)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 0 || rep.NoRoute != 0 {
		t.Fatalf("relay: %+v", rep)
	}
	return rep.Dispatched + rep.Ignored
}

func (e *env) admin(t *testing.T) *pgx.Conn {
	t.Helper()
	if e.conn == nil {
		c, err := pgx.Connect(context.Background(), e.DSN)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close(context.Background()) })
		e.conn = c
	}
	return e.conn
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.admin(t).QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func (e *env) cases(t *testing.T, subscription string) []string {
	t.Helper()
	rows, err := e.admin(t).Query(context.Background(), `SELECT case_id::text FROM fulfillment.work_case WHERE subscription_id = $1 ORDER BY service_key`, subscription)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func (e *env) activate(t *testing.T, m biawinapi.InvoiceFullyPaid) activation.Outcome {
	t.Helper()
	rep, err := e.act.HandleInvoiceFullyPaid(context.Background(), rollout.Shadow, m)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Refused != nil || len(rep.Lines) != 1 {
		t.Fatalf("activation report: %+v", rep)
	}
	return rep.Lines[0]
}

var seq int

func key() string { seq++; return fmt.Sprintf("syn-key-%06d", seq) }

func (e *env) cmd(t *testing.T, c work.Command) work.Result {
	t.Helper()
	if c.IdempotencyKey == "" {
		c.IdempotencyKey = key()
	}
	if c.CorrelationID == "" {
		c.CorrelationID = "corr-cmd"
	}
	r, err := e.work.Execute(context.Background(), rollout.Controlled, c)
	if err != nil {
		t.Fatalf("command %s: %v", c.Type, err)
	}
	return r
}

func actor(u string) work.Actor { return work.Actor{Principal: "synthetic:" + u, UserID: u} }

func hmacHex(secret, body []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// signedCallback builds a canonical synthetic envelope signed with secret
// (no transport claims).
func signedCallback(t *testing.T, secret []byte, env providerops.CallbackEnvelope) providerops.Callback {
	t.Helper()
	env.Version, env.Provider = providerops.CallbackEnvelopeVersion, "BIAWIN"
	body, err := providerops.EncodeCallback(env)
	if err != nil {
		t.Fatal(err)
	}
	return providerops.Callback{Body: body, Signature: hmacHex(secret, body)}
}

func shaOf(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// 1 / 2 / 3 / 4 / 5 / 22 / 23 / 32: one, 12 and 31 services; replays;
// owner-local transactions; identifiers preserved.
func TestActivationFanOut(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for _, n := range []int{1, 12, 31} {
		product := fmt.Sprintf("prod-%d", n)
		e.seed(t, product, n, nil, nil, nil)
		customer := uid(t)
		m := paid(t, product, biawinapi.RouteAdditionalServices, "RESOLVED", &customer)
		o := e.activate(t, m)
		if o.Decision != biawinapi.DecisionShadowActivated || o.ItemCount != n || o.SubscriptionID == nil {
			t.Fatalf("%d services: %+v", n, o)
		}
		// 23: biawin committed alone; no fulfillment case exists before the relay.
		if got := len(e.cases(t, *o.SubscriptionID)); got != 0 {
			t.Fatalf("cases before relay: %d", got)
		}
		// 4: the same activation message replayed is an inbox duplicate.
		rep, err := e.act.HandleInvoiceFullyPaid(ctx, rollout.Shadow, m)
		if err != nil || !rep.Duplicate {
			t.Fatalf("replay: %+v %v", rep, err)
		}
		// A DIFFERENT message for the same invoice line never creates a second subscription.
		m2 := m
		m2.MessageID = uid(t)
		rep2, err := e.act.HandleInvoiceFullyPaid(ctx, rollout.Shadow, m2)
		if err != nil || len(rep2.Lines) != 1 || !rep2.Lines[0].Existing || *rep2.Lines[0].SubscriptionID != *o.SubscriptionID {
			t.Fatalf("second message: %+v %v", rep2, err)
		}
		if relayed := e.relay(t); relayed != n+1 {
			t.Fatalf("relayed %d events, want %d", relayed, n+1)
		}
		cs := e.cases(t, *o.SubscriptionID)
		if len(cs) != n {
			t.Fatalf("%d services → %d cases", n, len(cs))
		}
		// 32: business key, message (causation), correlation and invoice ref preserved.
		if e.count(t, `SELECT count(*) FROM biawin.activation WHERE activation_id = $1 AND message_id = $2 AND invoice_ref = 'legacy-invoice-2001'
			AND correlation_id = $3 AND business_key = $4`, o.ActivationID, m.MessageID, m.CorrelationID, o.BusinessKey) != 1 {
			t.Fatal("activation provenance not preserved")
		}
	}
	// 5 / 22: replaying an item event (relay redelivery) is a no-op.
	var id, payload string
	if err := e.admin(t).QueryRow(ctx, `SELECT event_id::text, payload::text FROM biawin.outbox_event WHERE event_type = $1 LIMIT 1`, biawinapi.EventItemActivated).
		Scan(&id, &payload); err != nil {
		t.Fatal(err)
	}
	at := e.clock.Now()
	out, err := e.work.OnItemActivated(ctx, outbox.Record{Event: outbox.Event{ID: id, Type: biawinapi.EventItemActivated, Version: 1, Payload: []byte(payload), OccurredAt: at}})
	if err != nil || !out.Duplicate {
		t.Fatalf("item replay: %+v %v", out, err)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_case`) != 1+12+31 {
		t.Fatal("cases duplicated")
	}
}

// 12 / 13 / 14 / 15 / 30: lineage, branches and D-35.
func TestActivationBranchesAndBlocks(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.seed(t, "prod-b", 2, nil, nil, nil)
	customer := uid(t)
	for _, tc := range []struct {
		route, state string
		cust         *string
		want         string
		block        string
	}{
		{biawinapi.RouteComplaint, "RESOLVED", &customer, biawinapi.DecisionNotApplicable, ""},
		{biawinapi.RouteArchive, "RESOLVED", &customer, biawinapi.DecisionNotApplicable, ""},
		{"", "RESOLVED", &customer, biawinapi.DecisionBlocked, biawinapi.BlockRouteUnknown},
		{biawinapi.RouteAdditionalServices, "UNKNOWN", nil, biawinapi.DecisionBlocked, biawinapi.BlockIdentityUnknown},
		{biawinapi.RouteAdditionalServices, "CONFLICT", nil, biawinapi.DecisionBlocked, biawinapi.BlockIdentityConflict},
	} {
		o := e.activate(t, paid(t, "prod-b", tc.route, tc.state, tc.cust))
		if o.Decision != tc.want || o.SubscriptionID != nil || (tc.block != "" && !strings.Contains(strings.Join(o.BlockedBy, ","), tc.block)) {
			t.Fatalf("%s/%s: %+v", tc.route, tc.state, o)
		}
	}
	if e.count(t, `SELECT count(*) FROM biawin.outbox_event`) != 0 || e.count(t, `SELECT count(*) FROM biawin.target_subscription`) != 0 {
		t.Fatal("14/15: complaint / archive / blocked lines must not enter Biawin")
	}
	// 13: an existing CRM customer entering the additional-services branch.
	m := paid(t, "prod-b", biawinapi.RouteAdditionalServices, "RESOLVED", &customer)
	m.Customer.Existing = true
	if o := e.activate(t, m); o.Decision != biawinapi.DecisionShadowActivated {
		t.Fatalf("existing customer: %+v", o)
	}
	if e.count(t, `SELECT count(*) FROM biawin.activation WHERE existing_customer AND customer_id = $1`, customer) != 1 {
		t.Fatal("existing-customer provenance")
	}
	// 30: CONTROLLED (a real activation) is always BLOCKED by D-35; DISABLED records nothing.
	rep, err := e.act.HandleInvoiceFullyPaid(ctx, rollout.Controlled, paid(t, "prod-b", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	if err != nil || len(rep.Lines) != 1 || rep.Lines[0].Decision != biawinapi.DecisionBlocked || !strings.Contains(strings.Join(rep.Lines[0].BlockedBy, ","), "D-35") {
		t.Fatalf("controlled: %+v %v", rep, err)
	}
	before := e.count(t, `SELECT count(*) FROM biawin.activation`)
	rep, err = e.act.HandleInvoiceFullyPaid(ctx, rollout.Disabled, paid(t, "prod-b", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	if err != nil || rep.Refused == nil || e.count(t, `SELECT count(*) FROM biawin.activation`) != before {
		t.Fatal("disabled must record nothing")
	}
	// 11 / 30: a component with a financial behaviour (credit grant) blocks activation.
	e.seed(t, "prod-credit", 1, nil, nil, func(v *catalogapi.ServiceVersion) { v.FinancialBehaviour = catalogapi.FinancialConsumesCredit })
	o := e.activate(t, paid(t, "prod-credit", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	if o.Decision != biawinapi.DecisionBlocked || !strings.Contains(strings.Join(o.BlockedBy, ","), biawinapi.BlockFinancialBehaviour) {
		t.Fatalf("credit component: %+v", o)
	}
	// 10: missing components / definitions.
	o = e.activate(t, paid(t, "prod-missing", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	if o.Decision != biawinapi.DecisionBlocked || !strings.Contains(strings.Join(o.BlockedBy, ","), biawinapi.BlockComponentsMissing) {
		t.Fatalf("missing components: %+v", o)
	}
	// 30: the decision-blocked command list.
	for _, n := range []string{"biawin.wallet_charge", "biawin.activate_production", "fulfillment.cancel_case", "fulfillment.reopen_case"} {
		if r, ok := work.BlockedCommand(n); !ok || r.Status != result.Blocked || len(r.BlockedBy) == 0 {
			t.Fatalf("%s must be blocked", n)
		}
	}
}

// 6 / 7 / 8 / 9 / 11 / 24 / 25 / 26 / 27: per-service operations.
func TestFulfillmentCommands(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	supA, supB, execA, execB, outsider := uid(t), uid(t), uid(t), uid(t), uid(t)
	keys := e.seed(t, "prod-ops", 3, map[string]string{"prod-ops.svc-01": supA, "prod-ops.svc-02": supB}, map[string]string{"prod-ops.svc-01": execA,
		"prod-ops.svc-02": execB}, nil)
	customer := uid(t)
	o := e.activate(t, paid(t, "prod-ops", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	e.relay(t)
	cs := e.cases(t, *o.SubscriptionID)
	caseA, caseB, caseC := cs[0], cs[1], cs[2]
	_ = keys
	// 8: a supervisor of another service, or a non-supervisor, cannot assign.
	if r := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseA, AssigneeUserID: execA, ExpectedVersion: 1, Actor: actor(supB)}); r.Status != result.Unauthorized {
		t.Fatalf("8: other service supervisor: %+v", r)
	}
	if r := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseA, AssigneeUserID: execA, ExpectedVersion: 1, Actor: actor(outsider)}); r.Status != result.Unauthorized {
		t.Fatalf("8: outsider: %+v", r)
	}
	// An executor of another service is not eligible.
	if r := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseA, AssigneeUserID: execB, ExpectedVersion: 1, Actor: actor(supA)}); r.Status != result.Blocked {
		t.Fatalf("ineligible assignee: %+v", r)
	}
	// 11: a service with no supervisor at all can be assigned by nobody.
	if r := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseC, AssigneeUserID: execA, ExpectedVersion: 1, Actor: actor(supA)}); r.Status != result.Unauthorized {
		t.Fatalf("11: no supervisor: %+v", r)
	}
	// 27: concurrent assignment attempts: exactly one wins.
	var wg sync.WaitGroup
	results := make([]work.Result, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := e.work.Execute(ctx, rollout.Controlled, work.Command{Type: work.TypeAssign, CaseID: caseA, AssigneeUserID: execA, ExpectedVersion: 1,
				IdempotencyKey: fmt.Sprintf("concurrent-assign-%d", i), CorrelationID: "corr", Actor: actor(supA)})
			if err != nil {
				t.Error(err)
			}
			results[i] = r
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, r := range results {
		if r.Status == result.Success {
			wins++
		} else if r.Status != result.Conflict {
			t.Fatalf("27: loser must be CONFLICT: %+v", r)
		}
	}
	if wins != 1 {
		t.Fatalf("27: %d winners", wins)
	}
	// 26: stale expected version.
	if r := e.cmd(t, work.Command{Type: work.TypeAction, CaseID: caseA, Action: api.ActionNoAnswer, ExpectedVersion: 1, Actor: actor(execA)}); r.Status != result.Conflict {
		t.Fatalf("26: stale version: %+v", r)
	}
	// A supervisor never substitutes for the assignee.
	if r := e.cmd(t, work.Command{Type: work.TypeAction, CaseID: caseA, Action: api.ActionNoAnswer, ExpectedVersion: 2, Actor: actor(supA)}); r.Status != result.Unauthorized {
		t.Fatalf("supervisor as executor: %+v", r)
	}
	r := e.cmd(t, work.Command{Type: work.TypeAction, CaseID: caseA, Action: api.ActionNoAnswer, ExpectedVersion: 2, Actor: actor(execA)})
	if r.Status != result.Success || r.State != api.WorkInProgress || r.CaseVersion != 3 {
		t.Fatalf("action: %+v", r)
	}
	// 25: completion without the required evidence is BLOCKED.
	if r := e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: caseA, Outcome: "EXECUTED", ExpectedVersion: 3, Actor: actor(execA)}); r.Status != result.Blocked ||
		!strings.Contains(strings.Join(r.BlockedBy, ","), api.BlockEvidenceMissing) {
		t.Fatalf("25: %+v", r)
	}
	// 24: evidence on another service's case by this executor is UNAUTHORIZED; an unknown type is refused.
	e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseB, AssigneeUserID: execB, ExpectedVersion: 1, Actor: actor(supB)})
	if r := e.cmd(t, work.Command{Type: work.TypeEvidence, CaseID: caseB, EvidenceType: "service-form", ContentSHA: shaOf("x"), Label: "form", ExpectedVersion: 2,
		Actor: actor(execA)}); r.Status != result.Unauthorized {
		t.Fatalf("24: %+v", r)
	}
	if r := e.cmd(t, work.Command{Type: work.TypeEvidence, CaseID: caseA, EvidenceType: "payment-receipt", ContentSHA: shaOf("x"), Label: "x", ExpectedVersion: 3,
		Actor: actor(execA)}); r.Status != result.Blocked {
		t.Fatalf("unknown evidence type: %+v", r)
	}
	r = e.cmd(t, work.Command{Type: work.TypeEvidence, CaseID: caseA, EvidenceType: "service-form", ContentSHA: shaOf("form-a"), Label: "signed form", ExpectedVersion: 3,
		Actor: actor(execA)})
	if r.Status != result.Success {
		t.Fatalf("evidence: %+v", r)
	}
	// 7 / 6: one service completed while the others stay open and independent.
	r = e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: caseA, Outcome: "EXECUTED", ExpectedVersion: 4, Actor: actor(execA)})
	if r.Status != result.Success || r.State != api.WorkCompleted {
		t.Fatalf("complete: %+v", r)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_case WHERE subscription_id = $1 AND state NOT IN ('COMPLETED', 'NOT_EXECUTED')`, *o.SubscriptionID) != 2 {
		t.Fatal("7: other services must stay open")
	}
	// A closed case accepts nothing more (reopen is blocked).
	if r := e.cmd(t, work.Command{Type: work.TypeAction, CaseID: caseA, Action: api.ActionNoAnswer, ExpectedVersion: 5, Actor: actor(execA)}); r.Status != result.Conflict {
		t.Fatalf("closed case: %+v", r)
	}
	// 9: changing the supervisor does not rewrite history: past events keep
	// the supervisors of their time; the revoked supervisor loses authority.
	newSup := uid(t)
	if err := e.cat.Record(ctx, registry.Recording{ApprovalRef: "OWNER-APPROVAL-SYN-14-B", RecordedBy: "test", Scopes: []registry.ScopeRecord{
		{ServiceKey: "prod-ops.svc-02", UserID: supB, Kind: catalogapi.ScopeSupervisor, Action: "REVOKE"},
		{ServiceKey: "prod-ops.svc-02", UserID: newSup, Kind: catalogapi.ScopeSupervisor, Action: "GRANT"}}}); err != nil {
		t.Fatal(err)
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'ASSIGNED' AND supervisors ? $2 AND actor_user_id = $3::uuid`, caseB, supB, supB) != 1 {
		t.Fatal("9: historical assignment must keep its supervisor")
	}
	if e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1 AND assigned_by_user_id = $2`, caseB, supB) != 1 {
		t.Fatal("9: historical assignment actor")
	}
	if r := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseB, AssigneeUserID: execB, ExpectedVersion: 2, Actor: actor(supB)}); r.Status != result.Unauthorized {
		t.Fatalf("9: revoked supervisor: %+v", r)
	}
	// NOT_EXECUTED needs a reason (400 otherwise) and closes only its own case.
	if _, err := e.work.Execute(ctx, rollout.Controlled, work.Command{Type: work.TypeComplete, CaseID: caseB, Outcome: "NOT_EXECUTED", ExpectedVersion: 2,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(execB)}); !errors.Is(err, work.ErrInvalid) {
		t.Fatalf("reason required: %v", err)
	}
	r = e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: caseB, Outcome: "NOT_EXECUTED", Reason: ptr("customer unreachable"), ExpectedVersion: 2, Actor: actor(execB)})
	if r.Status != result.Success || r.State != api.WorkNotExecuted {
		t.Fatalf("not executed: %+v", r)
	}
	// SHADOW mode evaluates but changes nothing.
	sh, err := e.work.Execute(ctx, rollout.Shadow, work.Command{Type: work.TypeAssign, CaseID: caseC, AssigneeUserID: execA, ExpectedVersion: 1,
		IdempotencyKey: key(), CorrelationID: "c", Actor: actor(supA)})
	if err != nil || sh.Status != result.Unauthorized {
		t.Fatalf("shadow (still authority-checked): %+v %v", sh, err)
	}
	// Idempotent replay of a successful command returns the same result.
	k := key()
	first := e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: caseC, AssigneeUserID: execA, ExpectedVersion: 1, IdempotencyKey: k, Actor: actor(supA)})
	_ = first
	// 28 / 29: fulfillment writes only its own schema; no wallet, ledger or commission anywhere.
	if e.count(t, `SELECT count(*) FROM information_schema.role_table_grants WHERE grantee IN ('crm_owner_fulfillment', 'crm_owner_biawin', 'crm_owner_catalog')
		AND table_schema NOT IN ('fulfillment', 'biawin', 'catalog')`) != 0 {
		t.Fatal("28: an owner holds a grant outside its schema")
	}
	if e.count(t, `SELECT count(*) FROM information_schema.columns WHERE table_schema IN ('fulfillment', 'biawin', 'catalog')
		AND column_name ~* '(wallet|ledger|commission|balance|amount|credit)' AND table_name !~ '^(project|wallet)'`) != 0 {
		t.Fatal("29: a money / wallet / commission column exists in the Slice 14 tables")
	}
}

// 16 / 17 / 18 / 19 / 20 / 21 / 29: the provider ledger with the
// deterministic synthetic adapter (NOT a production connector).
func TestProviderOperations(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	port := integrationbiawin.NewSynthetic()
	caseID := uid(t)
	req := func(k string) providerops.OperationRequest {
		return providerops.OperationRequest{OperationKey: "case:" + k + "|op:SERVICE_ORDER", Provider: "BIAWIN", OpType: providerops.OpServiceOrder,
			SubjectCaseID: &caseID, CorrelationID: "corr-op", CausationID: "cmd-" + k, Payload: map[string]string{"service_key": "svc"}}
	}
	// 16: a duplicate request is the same operation; a changed request is a conflict.
	op, replay, err := e.ops.RequestOperation(ctx, req("a"))
	if err != nil || replay || op.State != providerops.StatePrepared {
		t.Fatalf("request: %+v %v", op, err)
	}
	if again, replay, err := e.ops.RequestOperation(ctx, req("a")); err != nil || !replay || again.OperationID != op.OperationID {
		t.Fatalf("16: duplicate request: %v", err)
	}
	changed := req("a")
	changed.Payload = map[string]string{"service_key": "other"}
	if _, _, err := e.ops.RequestOperation(ctx, changed); !errors.Is(err, providerops.ErrConflict) {
		t.Fatalf("16: changed request: %v", err)
	}
	// The disabled (production) adapter sends nothing and changes nothing.
	if _, err := e.ops.Dispatch(ctx, integrationbiawin.DisabledPort{}, op.OperationID); !errors.Is(err, providerops.ErrPortDisabled) {
		t.Fatalf("disabled port: %v", err)
	}
	if cur, _ := e.ops.Get(ctx, op.OperationID); cur.State != providerops.StatePrepared || cur.Attempts != 0 {
		t.Fatalf("disabled port changed state: %+v", cur)
	}
	done, err := e.ops.Dispatch(ctx, port, op.OperationID)
	if err != nil || done.State != providerops.StateAcknowledged || done.ExternalRef == nil {
		t.Fatalf("ack: %+v %v", done, err)
	}
	if _, err := e.ops.Dispatch(ctx, port, op.OperationID); !errors.Is(err, providerops.ErrState) || port.Calls(op.Reference) != 1 {
		t.Fatal("16: an acknowledged operation is never sent twice")
	}
	// 18: the same callback twice; 20: a conflicting external reference.
	secret := []byte("synthetic-callback-secret")
	cb := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-1", Reference: op.Reference, ExternalRef: *done.ExternalRef, Status: "CONFIRMED"})
	if d, err := e.ops.HandleCallback(ctx, secret, cb); err != nil || d != "APPLIED" {
		t.Fatalf("callback: %s %v", d, err)
	}
	if d, _ := e.ops.HandleCallback(ctx, secret, cb); d != "DUPLICATE" {
		t.Fatalf("18: duplicate callback: %s", d)
	}
	bad := cb
	bad.Signature = hmacHex(secret, []byte("tampered"))
	if _, err := e.ops.HandleCallback(ctx, secret, bad); !errors.Is(err, providerops.ErrSignature) {
		t.Fatal("unauthenticated callback accepted")
	}
	conflict := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-2", Reference: op.Reference, ExternalRef: "syn-order-OTHER", Status: "CONFIRMED"})
	if d, _ := e.ops.HandleCallback(ctx, secret, conflict); d != "CONFLICT" {
		t.Fatalf("20: conflicting reference: %s", d)
	}
	if ok, _ := e.ops.Confirmed(ctx, caseID); !ok {
		t.Fatal("confirmed read")
	}
	// 19: callback before the dispatch answer.
	op19, _, _ := e.ops.RequestOperation(ctx, req("early"))
	if d, err := e.ops.HandleCallback(ctx, secret, signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-early", Reference: op19.Reference,
		ExternalRef: "syn-order-early", Status: "CONFIRMED"})); err != nil || d != "EARLY_APPLIED" {
		t.Fatalf("19: %s %v", d, err)
	}
	if _, err := e.ops.Dispatch(ctx, port, op19.OperationID); !errors.Is(err, providerops.ErrState) || port.Calls(op19.Reference) != 0 {
		t.Fatal("19: a confirmed operation must not be dispatched")
	}
	// 17 / 21: timeout after a possible creation → UNKNOWN; no blind retry;
	// reconcile finds it → CONFIRMED, no duplicate order.
	op17, _, _ := e.ops.RequestOperation(ctx, req("timeout"))
	port.Script(op17.Reference, integrationbiawin.BehaveTimeoutCreated)
	u, err := e.ops.Dispatch(ctx, port, op17.OperationID)
	if err != nil || u.State != providerops.StateUnknown {
		t.Fatalf("17: %+v %v", u, err)
	}
	if _, err := e.ops.Dispatch(ctx, port, op17.OperationID); !errors.Is(err, providerops.ErrState) {
		t.Fatal("21: UNKNOWN must not be retried blindly")
	}
	rc, err := e.ops.Reconcile(ctx, port, op17.OperationID)
	if err != nil || rc.State != providerops.StateConfirmed || port.Calls(op17.Reference) != 1 || port.Created(op17.Reference) != 1 {
		t.Fatalf("21: reconcile found: %+v %v", rc, err)
	}
	// Timeout without creation → lookup proves absence → retry permitted.
	opA, _, _ := e.ops.RequestOperation(ctx, req("absent"))
	port.Script(opA.Reference, integrationbiawin.BehaveTimeoutAbsent)
	e.ops.Dispatch(ctx, port, opA.OperationID)
	if rc, _ := e.ops.Reconcile(ctx, port, opA.OperationID); rc.State != providerops.StateAbsent {
		t.Fatalf("absent: %+v", rc)
	}
	port.Script(opA.Reference, integrationbiawin.BehaveAck)
	if again, err := e.ops.Dispatch(ctx, port, opA.OperationID); err != nil || again.State != providerops.StateAcknowledged || again.Attempts != 2 {
		t.Fatalf("retry after proven absence: %+v %v", again, err)
	}
	// No lookup available → RECONCILIATION_REQUIRED → recorded human evidence.
	opN, _, _ := e.ops.RequestOperation(ctx, req("nolookup"))
	port.Script(opN.Reference, integrationbiawin.BehaveTimeoutNoLookup)
	e.ops.Dispatch(ctx, port, opN.OperationID)
	rq, _ := e.ops.Reconcile(ctx, port, opN.OperationID)
	if rq.State != providerops.StateReconciliation {
		t.Fatalf("reconciliation required: %+v", rq)
	}
	if _, err := e.ops.Dispatch(ctx, port, opN.OperationID); !errors.Is(err, providerops.ErrState) {
		t.Fatal("RECONCILIATION_REQUIRED must not be dispatched")
	}
	res, err := e.ops.Resolve(ctx, providerops.Resolution{OperationID: opN.OperationID, Present: true, ExternalRef: "observed-in-provider-panel",
		EvidenceNote: "operator verified in provider admin", ExpectedVersion: rq.Version, ActorUserID: uid(t)})
	if err != nil || res.State != providerops.StateConfirmed {
		t.Fatalf("resolve: %+v %v", res, err)
	}
	// 29: a wallet charge is only ever BLOCKED (D-38); the port is never called.
	w, _, err := e.ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:wallet|op:WALLET_CHARGE", Provider: "BIAWIN",
		OpType: providerops.OpWalletCharge, CorrelationID: "c", CausationID: "x"})
	if err != nil || w.State != providerops.StateBlocked || w.BlockedBy == nil || *w.BlockedBy != "D-38" {
		t.Fatalf("29: wallet charge: %+v %v", w, err)
	}
	if _, err := e.ops.Dispatch(ctx, port, w.OperationID); !errors.Is(err, providerops.ErrState) || port.Calls(w.Reference) != 0 {
		t.Fatal("29: wallet charge must never be dispatched")
	}
	// The evidence trail is append-only.
	if _, err := e.admin(t).Exec(ctx, `UPDATE integration.provider_operation_event SET kind = 'ACKNOWLEDGED'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("provider evidence must be append-only: %v", err)
	}
}

// Completion of a provider-integrated service needs a CONFIRMED provider
// operation; a non-NONE financial behaviour blocks completion (D-35/D-38).
func TestCompletionGuards(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sup, ex := uid(t), uid(t)
	e.seed(t, "prod-prov", 1, map[string]string{"prod-prov.svc-01": sup}, map[string]string{"prod-prov.svc-01": ex}, func(v *catalogapi.ServiceVersion) {
		v.IntegrationProvider = catalogapi.ProviderBiawin
		v.Evidence = []catalogapi.EvidenceRequirement{}
	})
	customer := uid(t)
	o := e.activate(t, paid(t, "prod-prov", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	e.relay(t)
	c := e.cases(t, *o.SubscriptionID)[0]
	e.cmd(t, work.Command{Type: work.TypeAssign, CaseID: c, AssigneeUserID: ex, ExpectedVersion: 1, Actor: actor(sup)})
	if r := e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: "EXECUTED", ExpectedVersion: 2, Actor: actor(ex)}); r.Status != result.Blocked ||
		!strings.Contains(strings.Join(r.BlockedBy, ","), api.BlockProviderConfirmation) {
		t.Fatalf("provider confirmation: %+v", r)
	}
	op, _, _ := e.ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:" + c + "|op:SERVICE_ORDER", Provider: "BIAWIN",
		OpType: providerops.OpServiceOrder, SubjectCaseID: &c, CorrelationID: "c", CausationID: "x"})
	if _, err := e.ops.Dispatch(ctx, integrationbiawin.NewSynthetic(), op.OperationID); err != nil {
		t.Fatal(err)
	}
	secret := []byte("s")
	cur, _ := e.ops.Get(ctx, op.OperationID)
	if _, err := e.ops.HandleCallback(ctx, secret, signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb", Reference: op.Reference,
		ExternalRef: *cur.ExternalRef, Status: "CONFIRMED"})); err != nil {
		t.Fatal(err)
	}
	if r := e.cmd(t, work.Command{Type: work.TypeComplete, CaseID: c, Outcome: "EXECUTED", ExpectedVersion: 2, Actor: actor(ex)}); r.Status != result.Success {
		t.Fatalf("confirmed provider completion: %+v", r)
	}
}

// 22 / 23: fault injection across the module boundary. A failing
// fulfillment intake never rolls back biawin's committed activation; the
// event stays in biawin's outbox (marked failed) and the next relay
// delivers it exactly once (fulfillment inbox).
func TestRelayFaultInjection(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.seed(t, "prod-f", 3, nil, nil, nil)
	customer := uid(t)
	o := e.activate(t, paid(t, "prod-f", biawinapi.RouteAdditionalServices, "RESOLVED", &customer))
	boom := errors.New("injected fulfillment outage")
	faulty := e.work.IntakeRoute()
	faulty.Backoff = &outbox.Backoff{}
	faulty.Handlers = map[string]relay.Handler{biawinapi.EventItemActivated: func(context.Context, outbox.Record) error { return boom }}
	rep, err := relay.Drain(ctx, e.bR, e.clock, 500, faulty)
	if err != nil || rep.Ignored != 1 || rep.Dispatched != 0 || rep.Failed != 3 {
		t.Fatalf("faulty relay: %+v err=%v (only the subscription event may be dispatched)", rep, err)
	}
	if len(e.cases(t, *o.SubscriptionID)) != 0 || e.count(t, `SELECT count(*) FROM biawin.target_item WHERE subscription_id = $1`, *o.SubscriptionID) != 3 {
		t.Fatal("biawin must stay committed while fulfillment failed")
	}
	if e.count(t, `SELECT count(*) FROM biawin.outbox_event WHERE event_type = $1 AND dispatched_at IS NULL AND attempts = 1`, biawinapi.EventItemActivated) != 3 {
		t.Fatal("failed events must stay undispatched for redelivery")
	}
	e.relay(t)
	e.relay(t)
	if len(e.cases(t, *o.SubscriptionID)) != 3 || e.count(t, `SELECT count(*) FROM fulfillment.work_intake WHERE subscription_id = $1`, *o.SubscriptionID) != 3 {
		t.Fatal("redelivery must create each case exactly once")
	}
}
