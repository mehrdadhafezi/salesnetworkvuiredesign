package biawin_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	integrationbiawin "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

type moving struct {
	mu sync.Mutex
	at time.Time
}

func (c *moving) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *moving) add(d time.Duration) { c.mu.Lock(); c.at = c.at.Add(d); c.mu.Unlock() }

// crashPort is the synthetic port of a dispatcher that dies after the
// DISPATCHING state was committed and before the provider answered.
type crashPort struct {
	*integrationbiawin.SyntheticPort
}

func (crashPort) Submit(context.Context, providerops.Request) (providerops.Response, error) {
	panic("synthetic dispatcher crash")
}

func runtimeFor(t *testing.T, jobs ...worker.Job) *worker.Runtime {
	t.Helper()
	var specs []worker.Spec
	for _, j := range jobs {
		specs = append(specs, worker.Spec{Job: j, Gate: worker.Always, Interval: time.Second, Timeout: 10 * time.Second, MaxBackoff: time.Minute})
	}
	rt, err := worker.New(worker.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Specs: specs, MaxParallel: 1})
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

// R2 13 / 14 + crash recovery: the reconciliation job resolves UNKNOWN
// through the provider lookup only, never redispatches, recovers a crashed
// dispatch, keeps conflicting references visible, and refuses a disabled
// port (which would escalate everything without asking the provider).
func TestProviderReconcileJob(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	clk := &moving{at: time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)}
	ops, err := providerops.New(e.Runner(t, integration.Module), clk)
	if err != nil {
		t.Fatal(err)
	}
	port := integrationbiawin.NewSynthetic()
	caseID := uid(t)
	req := func(k string) providerops.OperationRequest {
		return providerops.OperationRequest{OperationKey: "case:" + k + "|op:SERVICE_ORDER", Provider: "BIAWIN", OpType: providerops.OpServiceOrder,
			SubjectCaseID: &caseID, CorrelationID: "corr-r2", CausationID: "cmd-" + k, Payload: map[string]string{"service_key": "svc"}}
	}
	if _, err := ops.ReconcileJob("provider_reconcile", integrationbiawin.DisabledPort{}, 10); !errors.Is(err, providerops.ErrPortDisabled) {
		t.Fatal("a disabled port must not be schedulable for reconciliation")
	}
	// 13: provider UNKNOWN (created at the provider, answer lost).
	created, _, _ := ops.RequestOperation(ctx, req("created"))
	port.Script(created.Reference, integrationbiawin.BehaveTimeoutCreated)
	if o, _ := ops.Dispatch(ctx, port, created.OperationID); o.State != providerops.StateUnknown {
		t.Fatalf("timeout: %+v", o)
	}
	// Proven absent at the provider.
	absent, _, _ := ops.RequestOperation(ctx, req("absent"))
	port.Script(absent.Reference, integrationbiawin.BehaveTimeoutAbsent)
	_, _ = ops.Dispatch(ctx, port, absent.OperationID)
	// No lookup offered: human evidence required.
	nolookup, _, _ := ops.RequestOperation(ctx, req("nolookup"))
	port.Script(nolookup.Reference, integrationbiawin.BehaveTimeoutNoLookup)
	_, _ = ops.Dispatch(ctx, port, nolookup.OperationID)
	// A crashed dispatcher leaves DISPATCHING behind.
	crashed, _, _ := ops.RequestOperation(ctx, req("crashed"))
	func() {
		defer func() { _ = recover() }()
		_, _ = ops.Dispatch(ctx, crashPort{port}, crashed.OperationID)
	}()
	if o, _ := ops.Get(ctx, crashed.OperationID); o.State != providerops.StateDispatching {
		t.Fatalf("crash before the provider answer: %+v", o)
	}
	// A PREPARED operation is never dispatched by the job.
	prepared, _, _ := ops.RequestOperation(ctx, req("prepared"))

	job, err := ops.ReconcileJob("provider_reconcile", port, 50)
	if err != nil {
		t.Fatal(err)
	}
	rt := runtimeFor(t, job)
	if st, _ := rt.RunOnce(ctx, "provider_reconcile"); st.State != "idle" {
		t.Fatalf("pass: %+v", st)
	}
	state := func(id string) string { o, _ := ops.Get(ctx, id); return o.State }
	if state(created.OperationID) != providerops.StateConfirmed || state(absent.OperationID) != providerops.StateAbsent ||
		state(nolookup.OperationID) != providerops.StateReconciliation || state(crashed.OperationID) != providerops.StateDispatching ||
		state(prepared.OperationID) != providerops.StatePrepared {
		t.Fatalf("after one pass: created=%s absent=%s nolookup=%s crashed=%s prepared=%s", state(created.OperationID), state(absent.OperationID),
			state(nolookup.OperationID), state(crashed.OperationID), state(prepared.OperationID))
	}
	// Past the dispatch lease the crashed dispatch becomes UNKNOWN and is
	// then reconciled by lookup — never re-sent.
	clk.add(providerops.DispatchLease + time.Minute)
	_, _ = rt.RunOnce(ctx, "provider_reconcile")
	if state(crashed.OperationID) != providerops.StateAbsent {
		t.Fatalf("crashed dispatch: %s", state(crashed.OperationID))
	}
	for _, o := range []providerops.Operation{created, absent, nolookup, crashed, prepared} {
		want := 1
		if o.OperationID == crashed.OperationID || o.OperationID == prepared.OperationID {
			want = 0
		}
		if port.Calls(o.Reference) != want || port.Created(created.Reference) != 1 {
			t.Fatalf("the job must never dispatch: %s calls=%d", o.Reference, port.Calls(o.Reference))
		}
	}
	if v, _ := rt.Metrics().Value("provider_reconciliation_required", "provider_reconcile"); v != 1 {
		t.Fatalf("reconciliation backlog metric: %v", v)
	}
	if v, _ := rt.Metrics().Value("provider_stale_dispatch_unknown_total", "provider_reconcile"); v != 1 {
		t.Fatalf("stale dispatch metric: %v", v)
	}
	// 14: a duplicate callback is recorded once; a conflicting reference
	// stays visible in the backlog.
	secret := []byte("synthetic-callback-secret")
	cb := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-r2-1", Reference: created.Reference, ExternalRef: "syn-order-" + created.Reference,
		Status: "CONFIRMED"})
	d1, _ := ops.HandleCallback(ctx, secret, cb)
	d2, _ := ops.HandleCallback(ctx, secret, cb)
	conflict := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-r2-2", Reference: created.Reference, ExternalRef: "syn-order-OTHER",
		Status: "CONFIRMED"})
	d3, _ := ops.HandleCallback(ctx, secret, conflict)
	if d1 != "ALREADY_FINAL" || d2 != "DUPLICATE" || d3 != "CONFLICT" {
		t.Fatalf("callbacks: %s %s %s", d1, d2, d3)
	}
	_, _ = rt.RunOnce(ctx, "provider_reconcile")
	if v, _ := rt.Metrics().Value("provider_conflicting_callbacks", "provider_reconcile"); v != 1 {
		t.Fatalf("conflicting reference must stay visible: %v", v)
	}
	// Human resolution remains evidence-based and attributed.
	fixed, err := ops.Resolve(ctx, providerops.Resolution{OperationID: nolookup.OperationID, Present: false, EvidenceNote: "provider portal checked (synthetic)",
		ExpectedVersion: mustVersion(t, ops, nolookup.OperationID), ActorUserID: uid(t)})
	if err != nil || fixed.State != providerops.StateAbsent {
		t.Fatalf("resolve: %+v %v", fixed, err)
	}
}

func mustVersion(t *testing.T, ops *providerops.Service, id string) int64 {
	t.Helper()
	o, err := ops.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return o.Version
}
