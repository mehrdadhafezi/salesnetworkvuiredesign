package shadow_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

// R2 fault-injection proofs over PostgreSQL, the real filesystem object
// store (a temporary directory) and the deterministic synthetic legacy.
// Every job runs through the shared worker runtime exactly as crm-worker
// composes it. Nothing leaves the test process.

// panicConn is the connector of a write-back worker that crashes after its
// write-ahead claim and BEFORE any legacy request.
type panicConn struct{ writeback.Connector }

func (panicConn) Apply(context.Context, writeback.Intent) (writeback.Applied, error) {
	panic("synthetic worker crash before the legacy request")
}

type r2 struct {
	*paymentsEnv
	rt         *worker.Runtime
	switchOn   atomic.Bool // the write-back delivery runtime switch
	intakeDown atomic.Bool
}

func setupR2(t *testing.T) *r2 {
	t.Helper()
	e := &r2{paymentsEnv: setupPayments(t)}
	route, err := e.worker.IntakeRoute(payments.Module)
	if err != nil {
		t.Fatal(err)
	}
	for typ, h := range route.Handlers {
		h := h
		route.Handlers[typ] = func(ctx context.Context, rec outbox.Record) error {
			if e.intakeDown.Load() {
				return errors.New("integration consumer unavailable")
			}
			return h(ctx, rec)
		}
	}
	intake, err := worker.RelayJob("writeback_intake_payments", e.Runner(t, payments.Module), e.clock, 50, route)
	if err != nil {
		t.Fatal(err)
	}
	// Test-only: the payments type is delivered here to exercise the
	// UNKNOWN machinery against the synthetic legacy. crm-worker excludes it.
	fin, err := e.worker.WithOptions(writeback.Options{Types: []string{writeback.TypePaymentAttempt}, ObserveInterval: time.Minute, ObservationBudget: 3})
	if err != nil {
		t.Fatal(err)
	}
	nonFin, err := e.worker.WithOptions(writeback.Options{Types: []string{writeback.TypeContactOutcome, writeback.TypeHRCreate, writeback.TypeHRDecide},
		ObserveInterval: time.Minute, ObservationBudget: 3})
	if err != nil {
		t.Fatal(err)
	}
	gate := func(context.Context) (worker.GateState, string) {
		if !e.switchOn.Load() {
			return worker.Paused, "JOB_SWITCH_OFF"
		}
		return worker.Enabled, ""
	}
	spec := func(j worker.Job, g worker.Gate) worker.Spec {
		return worker.Spec{Job: j, Gate: g, Interval: time.Second, Timeout: 30 * time.Second, MaxBackoff: time.Minute}
	}
	e.rt, err = worker.New(worker.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), MaxParallel: 2, Specs: []worker.Spec{
		spec(intake, worker.Always),
		spec(e.svc.FinaliseJob("receipt_finalise", 20), worker.Always),
		spec(e.svc.ExpiryReportJob("upload_expiry"), worker.Always),
		spec(fin.DeliverJob("writeback_deliver", 20), gate),
		spec(nonFin.DeliverJob("writeback_deliver_nonfinancial", 20), worker.Always),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func (e *r2) pass(t *testing.T, job string) worker.Status {
	t.Helper()
	st, err := e.rt.RunOnce(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func (e *r2) metric(name, job string) float64 {
	v, _ := e.rt.Metrics().Value(name, job)
	return v
}

func (e *r2) cmd(t *testing.T, attempt string) string {
	t.Helper()
	return e.q(t, `SELECT coalesce((SELECT state || ':' || coalesce(parity, '-') FROM integration.writeback_command WHERE command_id = $1), 'none')`, attempt)
}

func (e *r2) receiptStates(t *testing.T, attempt string) string {
	t.Helper()
	return e.q(t, `SELECT string_agg(state, ',' ORDER BY state) FROM payments.receipt_evidence WHERE attempt_id = $1`, attempt)
}

// Scenarios 3, 4, 5, 8, 13, 15, 18 (budget), 19, 24, 25: receipt
// finalisation, intake relay and guarded write-back delivery, end to end.
func TestR2WorkerFaultInjection(t *testing.T) {
	e := setupR2(t)
	ctx := context.Background()
	seller := e.actor("6")
	before := e.legacy.Calls["payment"]

	// 15: copy succeeds, the reply is lost (state not updated) → STORAGE_ERROR;
	// the next pass verifies the existing object and finalises without a
	// second copy of the bytes.
	a, b := e.upload(t, "6", 41), e.upload(t, "6", 42)
	res := e.submit(t, batch("r2-key-0001", e.invoices["3001"], 0, seller, item("a", 600, a), item("b", 400, b)))
	first, second := res.Items[0].AttemptID, res.Items[1].AttemptID
	e.store.lostReply = true
	if st := e.pass(t, "receipt_finalise"); st.State != "idle" {
		t.Fatalf("finalise pass: %+v", st)
	}
	if got := e.receiptStates(t, first) + "|" + e.receiptStates(t, second); got != "STORAGE_ERROR|READY" && got != "READY|STORAGE_ERROR" {
		t.Fatalf("lost reply: %s", got)
	}
	if e.metric("receipt_storage_error", "receipt_finalise") != 1 {
		t.Fatal("STORAGE_ERROR must be visible in metrics")
	}
	e.pass(t, "receipt_finalise")
	if e.receiptStates(t, first) != "READY" || e.receiptStates(t, second) != "READY" {
		t.Fatalf("recovered: %s %s", e.receiptStates(t, first), e.receiptStates(t, second))
	}
	if n, _ := filepath.Glob(filepath.Join(e.root, "evidence", "*")); len(n) != 2 {
		t.Fatalf("content-addressed evidence must hold exactly one object per receipt: %v", n)
	}
	if e.count(t, `SELECT count(*) FROM payments.evidence_reconciliation WHERE reason = 'RECOVERED'`) != 1 {
		t.Fatal("the recovery must close the STORAGE_ERROR reconciliation")
	}

	// 8: the integration consumer is unavailable: the source events stay
	// undispatched and visible; after the backoff they are delivered.
	e.intakeDown.Store(true)
	e.pass(t, "writeback_intake_payments")
	if e.cmd(t, first) != "none" || e.metric("outbox_failed_retryable", "writeback_intake_payments") == 0 {
		t.Fatalf("consumer down: %s", e.cmd(t, first))
	}
	e.intakeDown.Store(false)
	e.pass(t, "writeback_intake_payments")
	if e.cmd(t, first) != "none" {
		t.Fatal("redelivery must wait for the backoff window")
	}
	e.clock.add(relay.DefaultBackoff.Base + time.Second)
	e.pass(t, "writeback_intake_payments")
	if e.cmd(t, first) != "PENDING:-" || e.cmd(t, second) != "PENDING:-" || e.metric("outbox_pending", "writeback_intake_payments") != 0 {
		t.Fatalf("intake: %s %s", e.cmd(t, first), e.cmd(t, second))
	}
	// 5: the same event delivered twice adds nothing.
	var rec outbox.Record
	rec.ID = e.q(t, `SELECT event_id::text FROM payments.outbox_event WHERE event_type = $1 LIMIT 1`, paymentsapi.EventAttemptEvidenceReady)
	rec.Type = paymentsapi.EventAttemptEvidenceReady
	rec.Payload = []byte(e.q(t, `SELECT payload::text FROM payments.outbox_event WHERE event_id = $1`, rec.ID))
	if out, err := e.worker.Receive(ctx, payments.Module, rec); err != nil || out.String() != "duplicate" {
		t.Fatalf("duplicate delivery: %v %v", out, err)
	}

	// 24: the delivery composed like crm-worker (no financial type) leaves
	// the payment commands untouched and sends nothing.
	e.pass(t, "writeback_deliver_nonfinancial")
	if e.cmd(t, first) != "PENDING:-" || e.legacy.Calls["payment"] != before {
		t.Fatal("a financial write-back must not run without an approved contract")
	}
	// 19: the runtime switch is off while commands are queued: nothing runs,
	// nothing is reported done.
	if st := e.pass(t, "writeback_deliver"); st.State != "paused" || e.cmd(t, first) != "PENDING:-" || e.legacy.Calls["payment"] != before {
		t.Fatalf("paused delivery: %+v %s", st, e.cmd(t, first))
	}
	e.switchOn.Store(true)

	// 3 + 13: legacy commits, the answer is lost → UNKNOWN; no blind retry;
	// after the observe interval the lookup proves the commit.
	e.legacy.Inject("payment", legacytest.TimeoutAfterCommit)
	e.pass(t, "writeback_deliver")
	st1, st2 := e.cmd(t, first), e.cmd(t, second)
	if (st1 != "UNKNOWN:-") == (st2 != "UNKNOWN:-") {
		t.Fatalf("exactly one command must be UNKNOWN: %s %s", st1, st2)
	}
	calls := e.legacy.Calls["payment"]
	e.pass(t, "writeback_deliver") // inside the observe interval: not observed, not retried
	if e.legacy.Calls["payment"] != calls {
		t.Fatal("an UNKNOWN command must never be re-applied")
	}
	e.clock.add(time.Minute + time.Second)
	e.pass(t, "writeback_deliver")
	if e.cmd(t, first) != "APPLIED:MATCH" || e.cmd(t, second) != "APPLIED:MATCH" || e.legacy.Calls["payment"] != before+2 {
		t.Fatalf("reconciled: %s %s calls=%d", e.cmd(t, first), e.cmd(t, second), e.legacy.Calls["payment"]-before)
	}

	// 4: the worker crashes after the write-ahead claim, before the legacy
	// request: UNKNOWN, observed ABSENT, retried once, applied once.
	c := e.upload(t, "6", 43)
	res = e.submit(t, batch("r2-key-0002", e.invoices["3005"], 0, seller, item("c", 200, c)))
	crashing := res.Items[0].AttemptID
	e.pass(t, "receipt_finalise")
	e.clock.add(relay.DefaultBackoff.Base + time.Second)
	e.pass(t, "writeback_intake_payments")
	conn, err := lwb.NewConnector(e.legacy, e.ident)
	if err != nil {
		t.Fatal(err)
	}
	conn.WithPayments(e.legacy)
	crashy, err := writeback.New(e.Runner(t, integration.Module), e.clock, panicConn{conn})
	if err != nil {
		t.Fatal(err)
	}
	crashy, _ = crashy.WithOptions(writeback.Options{Types: []string{writeback.TypePaymentAttempt}, ObserveInterval: time.Minute, ObservationBudget: 3})
	crt, _ := worker.New(worker.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), MaxParallel: 1, Specs: []worker.Spec{{
		Job: crashy.DeliverJob("writeback_deliver", 20), Gate: worker.Always, Interval: time.Second, Timeout: 30 * time.Second, MaxBackoff: time.Minute}}})
	calls = e.legacy.Calls["payment"]
	if st, _ := crt.RunOnce(ctx, "writeback_deliver"); st.State != "failing" {
		t.Fatalf("crashed pass must fail: %+v", st)
	}
	if e.cmd(t, crashing) != "UNKNOWN:-" || e.legacy.Calls["payment"] != calls {
		t.Fatalf("after the crash: %s", e.cmd(t, crashing))
	}
	e.clock.add(time.Minute + time.Second)
	e.pass(t, "writeback_deliver") // observe → ABSENT → PENDING
	e.pass(t, "writeback_deliver") // apply
	if e.cmd(t, crashing) != "APPLIED:MATCH" || e.legacy.Calls["payment"] != calls+1 {
		t.Fatalf("crash before the request: %s calls=%d", e.cmd(t, crashing), e.legacy.Calls["payment"]-calls)
	}

	// Observation budget: legacy stays unreadable → RECONCILIATION_REQUIRED
	// for human review; never retried, never assumed.
	d := e.upload(t, "6", 44)
	res = e.submit(t, batch("r2-key-0003", e.invoices["3005"], 1, seller, item("d", 100, d)))
	stuck := res.Items[0].AttemptID
	e.pass(t, "receipt_finalise")
	e.clock.add(relay.DefaultBackoff.Base + time.Second)
	e.pass(t, "writeback_intake_payments")
	e.legacy.Inject("payment", legacytest.TimeoutBeforeCommit)
	e.pass(t, "writeback_deliver")
	calls = e.legacy.Calls["payment"]
	for i := 0; i < 3; i++ {
		e.clock.add(time.Minute + time.Second)
		e.legacy.Inject("payment_observe", legacytest.ObserveDown)
		e.pass(t, "writeback_deliver")
	}
	if e.cmd(t, stuck) != "RECONCILIATION_REQUIRED:-" || e.legacy.Calls["payment"] != calls ||
		e.count(t, `SELECT count(*) FROM integration.writeback_reconciliation WHERE command_id = $1 AND reason = 'WRITEBACK_UNKNOWN'
			AND next_safe_action = 'DOMAIN_OWNER_REVIEW'`, stuck) != 1 {
		t.Fatalf("budget: %s calls=%d", e.cmd(t, stuck), e.legacy.Calls["payment"]-calls)
	}
	if e.metric("writeback_reconciliation_required", "writeback_deliver") != 1 {
		t.Fatal("the reconciliation backlog must be visible")
	}

	// 25: every attempt, observation and reconciliation is retained, as the
	// service actor; nothing was deleted across the crashes.
	if e.count(t, `SELECT count(*) FROM integration.writeback_attempt WHERE command_id = $1`, crashing) != 2 ||
		e.count(t, `SELECT count(*) FROM integration.writeback_attempt WHERE command_id = $1`, stuck) != 4 ||
		e.count(t, `SELECT count(*) FROM integration.writeback_attempt WHERE service_actor <> 'service:integration.writeback'`) != 0 ||
		e.count(t, `SELECT count(*) FROM payments.outbox_event WHERE dispatch_state <> 'dispatched'`) != 0 {
		t.Fatal("attempt / event evidence must be retained across recovery")
	}
	// 24: nothing was approved, rejected or posted, in the target or legacy.
	if e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state <> $1`, paymentsapi.AttemptPendingReview) != 0 ||
		e.count(t, `SELECT count(*) FROM finance.ledger_entry`) != 0 {
		t.Fatal("workers must never approve, reject or post")
	}
	for _, p := range e.legacy.Invoice("3001").Payments {
		if p.StatusRaw != "pending" {
			t.Fatalf("legacy payment written as %s", p.StatusRaw)
		}
	}
}

// Scenarios 16, 18 (finalisation budget), 7 and 17 (report-only expiry).
func TestR2EvidenceFaults(t *testing.T) {
	e := setupR2(t)
	ctx := context.Background()
	seller := e.actor("6")
	// 16: the quarantined bytes no longer match their hash.
	a := e.upload(t, "6", 51)
	res := e.submit(t, batch("r2-key-0101", e.invoices["3001"], 0, seller, item("a", 100, a)))
	if err := os.WriteFile(filepath.Join(e.root, "quarantine", a.UploadID), []byte("tampered synthetic bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.pass(t, "receipt_finalise")
	if e.receiptStates(t, res.Items[0].AttemptID) != "NOT_RECOVERABLE" ||
		e.q(t, `SELECT evidence_state FROM payments.submitted_attempt WHERE attempt_id = $1`, res.Items[0].AttemptID) != "NOT_RECOVERABLE" ||
		e.q(t, `SELECT state FROM payments.submitted_attempt WHERE attempt_id = $1`, res.Items[0].AttemptID) != paymentsapi.AttemptPendingReview {
		t.Fatal("a hash mismatch is NOT_RECOVERABLE and never approves or rejects the payment")
	}
	if e.metric("receipt_not_recoverable", "receipt_finalise") != 1 {
		t.Fatal("NOT_RECOVERABLE must be visible")
	}
	// 18: storage keeps failing: the receipt is retried at most
	// MaxFinaliseAttempts times, then left visible (exhausted), never READY.
	b := e.upload(t, "6", 52)
	res = e.submit(t, batch("r2-key-0102", e.invoices["3005"], 0, seller, item("b", 100, b)))
	for i := 0; i < paymentscommand.MaxFinaliseAttempts+3; i++ {
		e.store.failCopy = true
		e.pass(t, "receipt_finalise")
	}
	e.store.failCopy = false
	if e.receiptStates(t, res.Items[0].AttemptID) != "STORAGE_ERROR" ||
		e.q(t, `SELECT finalise_attempts::text FROM payments.receipt_evidence WHERE attempt_id = $1`, res.Items[0].AttemptID) != "20" ||
		e.metric("receipt_retry_exhausted", "receipt_finalise") != 1 {
		t.Fatalf("finalisation budget: %s", e.receiptStates(t, res.Items[0].AttemptID))
	}
	// 7: the database is unavailable: the pass fails, readiness fails,
	// nothing is reported done.
	pool := pgtest.OpenPool(t, e.LoginDSN(t, payments.Module))
	runner, _ := pool.Runner(payments.Module)
	svc, err := paymentscommand.New(runner, e.store, e.billingQ, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	drt, _ := worker.New(worker.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), MaxParallel: 1,
		Specs:  []worker.Spec{{Job: svc.FinaliseJob("receipt_finalise", 10), Gate: worker.Always, Interval: time.Second, Timeout: 5 * time.Second, MaxBackoff: time.Minute}},
		Checks: []worker.Check{{Name: "db_payments", Fn: pool.Ping}}})
	if st, _ := drt.RunOnce(ctx, "receipt_finalise"); st.State != "failing" || st.LastSuccess != (time.Time{}) {
		t.Fatalf("database down: %+v", st)
	}
	if ok, _ := drt.Ready(ctx); ok {
		t.Fatal("a worker without its database must not be ready")
	}
	// 17 (D-44 open): expiry in report mode counts eligible uploads and
	// changes nothing: no state change, no deletion.
	u := e.upload(t, "6", 53)
	e.clock.add(paymentscommand.UnboundTTL + time.Hour)
	fp := e.financial(t)
	e.pass(t, "upload_expiry")
	if e.q(t, `SELECT state FROM payments.upload_object WHERE upload_id = $1`, u.UploadID) != "UNBOUND" || e.financial(t) != fp ||
		e.metric("upload_unbound_expired", "upload_expiry") < 1 {
		t.Fatal("report-only expiry must not change anything")
	}
	if _, err := os.Stat(filepath.Join(e.root, "quarantine", u.UploadID)); err != nil {
		t.Fatal("report-only expiry must not delete bytes")
	}
}
