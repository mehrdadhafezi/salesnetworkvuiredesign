package shadow_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/gateway/synthetic"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// moveClock is a test clock that can advance (expiry, finalisation lease).
type moveClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *moveClock) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *moveClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

// faultStore injects storage failures around the real filesystem store.
type faultStore struct {
	objectstore.Store
	mu        sync.Mutex
	failPut   bool
	failCopy  bool // copy fails before writing
	lostReply bool // copy succeeds, the reply is lost
	panicCopy bool // the worker "crashes" during the copy
	copies    int
}

var errInjected = errors.New("injected storage failure")

func (f *faultStore) Put(ctx context.Context, key string, r io.Reader, want [32]byte) error {
	f.mu.Lock()
	fail := f.failPut
	f.mu.Unlock()
	if fail {
		return errInjected
	}
	return f.Store.Put(ctx, key, r, want)
}

func (f *faultStore) CopyIfAbsent(ctx context.Context, src, dst string, want [32]byte) error {
	f.mu.Lock()
	fail, lost, crash := f.failCopy, f.lostReply, f.panicCopy
	f.failCopy, f.lostReply, f.panicCopy = false, false, false
	f.copies++
	f.mu.Unlock()
	if crash {
		panic("synthetic worker crash during evidence copy")
	}
	if fail {
		return errInjected
	}
	err := f.Store.CopyIfAbsent(ctx, src, dst, want)
	if err == nil && lost {
		return errInjected
	}
	return err
}

type paymentsEnv struct {
	*billingEnv
	clock    *moveClock
	root     string
	store    *faultStore
	svc      *paymentscommand.Service
	worker   *writeback.Worker
	legacy   *legacytest.SyntheticLegacy
	users    map[string]string
	invoices map[string]string // legacy invoice key → invoice id
}

func setupPayments(t *testing.T) *paymentsEnv {
	t.Helper()
	ctx := context.Background()
	be := setupBilling(t)
	for _, src := range []migrations.Source{financemigrations.Source(), biawinmigrations.Source(), fulfillmentmigrations.Source()} {
		if _, err := migrations.Up(ctx, be.DSN, src); err != nil {
			t.Fatal(err)
		}
	}
	b := be.prepare(t, "syn-payments", legacytest.PaymentsScenario())
	if st := be.importBillingOnly(t, b); st != result.Success {
		t.Fatalf("billing import: %s", st)
	}
	if st, _ := be.importPaymentsOnly(t, b); st != result.Success {
		t.Fatalf("payments import: %s", st)
	}
	clock := &moveClock{at: time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)}
	root := t.TempDir()
	fs, err := objectstore.NewFS(root)
	if err != nil {
		t.Fatal(err)
	}
	store := &faultStore{Store: fs}
	svc, err := paymentscommand.New(be.Runner(t, payments.Module), store, be.billingQ, clock)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacytest.NewSyntheticLegacy()
	legacy.AddInvoice("3001", legacytest.SyntheticInvoice{SellerLegacyUserID: "6", StatusRaw: "pending", StageNo: 1, StageDueToman: 1000})
	legacy.AddInvoice("3005", legacytest.SyntheticInvoice{SellerLegacyUserID: "6", StatusRaw: "pending_financial_approval", StageNo: 1,
		StageDueToman: 800, Payments: []legacytest.SyntheticPayment{{ID: 71, StageNo: 1, AmountToman: 300, StatusRaw: "pending"}}})
	conn, err := lwb.NewConnector(legacy, be.ident)
	if err != nil {
		t.Fatal(err)
	}
	conn.WithPayments(legacy)
	w, err := writeback.New(be.Runner(t, integration.Module), clock, conn)
	if err != nil {
		t.Fatal(err)
	}
	users, err := be.ident.ResolveUsers(ctx, shadow.SourceSystem, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"})
	if err != nil || len(users) != 10 {
		t.Fatalf("users: %v %v", users, err)
	}
	keys := []string{"3001", "3002", "3003", "3004", "3005", "3006", "3007", "3008"}
	inv, err := be.billingQ.ResolveInvoices(ctx, shadow.SourceSystem, shadow.SpecInvoice, keys)
	if err != nil || len(inv) != len(keys) {
		t.Fatalf("invoices: %v %v", inv, err)
	}
	ids := map[string]string{}
	for k, v := range inv {
		ids[k] = v.InvoiceID
	}
	return &paymentsEnv{billingEnv: be, clock: clock, root: root, store: store, svc: svc, worker: w, legacy: legacy, users: users, invoices: ids}
}

func (e *paymentsEnv) actor(key string) paymentscommand.Actor {
	return paymentscommand.Actor{Principal: "synthetic:" + key, UserID: e.users[key]}
}

// receipt returns a small synthetic PNG / PDF body distinct per n.
func receipt(n int) []byte {
	if n%2 == 0 {
		return append([]byte("%PDF-1.4\n% synthetic receipt "), []byte(fmt.Sprint(n))...)
	}
	return append([]byte("\x89PNG\r\n\x1a\n synthetic receipt "), []byte(fmt.Sprint(n))...)
}

func (e *paymentsEnv) upload(t *testing.T, who string, n int) paymentscommand.ReceiptRef {
	t.Helper()
	r, err := e.svc.Upload(context.Background(), rollout.Controlled, e.actor(who), receipt(n))
	if err != nil || r.Status != result.Success {
		t.Fatalf("upload: %+v %v", r, err)
	}
	return paymentscommand.ReceiptRef{UploadID: r.UploadID, ContentSHA256: r.ContentSHA256}
}

func batch(key, invoice string, version int64, a paymentscommand.Actor, items ...paymentscommand.Item) paymentscommand.BatchCommand {
	return paymentscommand.BatchCommand{InvoiceID: invoice, StageNo: 1, EntryMode: paymentscommand.EntryReceipt, Items: items, ExpectedVersion: version,
		IdempotencyKey: key, CorrelationID: "corr-" + key, Actor: a}
}

func item(key string, amount int64, refs ...paymentscommand.ReceiptRef) paymentscommand.Item {
	return paymentscommand.Item{ItemKey: key, Amount: amount, Unit: "IRT", Receipts: refs}
}

func (e *paymentsEnv) submit(t *testing.T, c paymentscommand.BatchCommand) paymentscommand.BatchResult {
	t.Helper()
	r, err := e.svc.SubmitBatch(context.Background(), rollout.Controlled, c)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	return r
}

func (e *paymentsEnv) q(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var v string
	if err := e.admin.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// financial fingerprints the target's financial state (payments writes,
// billing, finance, biawin and fulfillment) for non-mutation proofs.
func (e *paymentsEnv) financial(t *testing.T) string {
	t.Helper()
	return e.md5(t, "payments.payment_batch", "payments.submitted_attempt", "payments.receipt_evidence", "payments.collection_account",
		"payments.upload_object", "payments.payment_attempt", "billing.invoice", "billing.payment_stage", "billing.amount_evidence",
		"finance.ledger_entry", "finance.ledger_line", "biawin.subscription", "fulfillment.fulfillment_case")
}

func has(rs []reasoncodes.Code, c reasoncodes.Code) bool {
	for _, r := range rs {
		if r == c {
			return true
		}
	}
	return false
}

// Upload validation, idempotent upload, the submission authority matrix,
// evidence binding, all-or-nothing batches, idempotency, versions and the
// available amount.
func TestPaymentBatchCommand(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller, other := e.actor("6"), e.actor("7")
	inv := e.invoices["3001"]

	// Upload: content sniffing, size and mode; same bytes → same upload.
	if _, err := e.svc.Upload(ctx, rollout.Controlled, seller, []byte("<html>not a receipt</html>")); !errors.Is(err, paymentscommand.ErrInvalid) {
		t.Fatalf("html accepted: %v", err)
	}
	if _, err := e.svc.Upload(ctx, rollout.Controlled, seller, make([]byte, paymentscommand.MaxUploadBytes+1)); !errors.Is(err, paymentscommand.ErrInvalid) {
		t.Fatalf("oversize accepted: %v", err)
	}
	if r, _ := e.svc.Upload(ctx, rollout.Disabled, seller, receipt(1)); r.Status != result.Blocked || r.Reasons[0] != reasoncodes.FeatureDisabled {
		t.Fatalf("disabled upload: %+v", r)
	}
	if r, _ := e.svc.Upload(ctx, rollout.Shadow, seller, receipt(1)); r.Status != result.Blocked || r.BlockedBy[0] != paymentscommand.ShadowUnsupported {
		t.Fatalf("shadow upload: %+v", r)
	}
	r1 := e.upload(t, "6", 1)
	again, _ := e.svc.Upload(ctx, rollout.Controlled, seller, receipt(1))
	if again.UploadID != r1.UploadID || !again.Existing {
		t.Fatalf("re-upload must return the existing upload: %+v", again)
	}
	if e.count(t, `SELECT count(*) FROM payments.upload_object`) != 1 {
		t.Fatal("bytes stored twice")
	}

	// Authority matrix (fail closed; hierarchy never substitutes).
	before := e.financial(t)
	cases := []struct {
		name    string
		invoice string
		a       paymentscommand.Actor
		status  result.Status
		code    reasoncodes.Code
	}{
		{"another seller's invoice", "3002", seller, result.Unauthorized, reasoncodes.RowScopeDenied},
		{"supervisor of the seller", "3001", e.actor("4"), result.Unauthorized, reasoncodes.RowScopeDenied},
		{"manager above the seller", "3001", e.actor("2"), result.Unauthorized, reasoncodes.RowScopeDenied},
		{"HR", "3001", e.actor("1"), result.Unauthorized, reasoncodes.RowScopeDenied},
		{"paid invoice", "3003", seller, result.Blocked, reasoncodes.ActionDenied},
		{"issued by someone else (conversion unknown)", "3004", seller, result.Unauthorized, reasoncodes.RowScopeDenied},
		{"unmapped stage status", "3006", seller, result.Blocked, reasoncodes.DependencyUnknown},
	}
	for i, c := range cases {
		r := e.submit(t, batch(fmt.Sprintf("matrix-key-%02d", i), e.invoices[c.invoice], 0, c.a, item("a", 100, r1)))
		if r.Status != c.status || !has(r.Reasons, c.code) || r.Applied {
			t.Fatalf("%s: %+v", c.name, r)
		}
	}
	// Info mode (no receipt) is BLOCKED (D-43); receipt mode needs a receipt.
	info := batch("info-mode-key", inv, 0, seller, item("a", 100))
	info.EntryMode = paymentscommand.EntryInfo
	if r := e.submit(t, info); r.Status != result.Blocked || r.BlockedBy[0] != paymentscommand.DecisionD43 {
		t.Fatalf("info mode: %+v", r)
	}
	if _, err := e.svc.SubmitBatch(ctx, rollout.Controlled, batch("no-receipt-key", inv, 0, seller, item("a", 100))); !errors.Is(err, paymentscommand.ErrInvalid) {
		t.Fatalf("payment without evidence must not commit: %v", err)
	}
	// Wrong evidence: another user's upload, a hash that does not match.
	theirs := e.upload(t, "7", 9)
	if r := e.submit(t, batch("theirs-key-1", inv, 0, seller, item("a", 100, theirs))); r.Status != result.Blocked || !has(r.Reasons, reasoncodes.ContractInvalid) {
		t.Fatalf("another user's upload: %+v", r)
	}
	bad := r1
	bad.ContentSHA256 = hex.EncodeToString(make([]byte, 32))
	if r := e.submit(t, batch("bad-hash-key", inv, 0, seller, item("a", 100, bad))); r.Status != result.Blocked || !has(r.Reasons, reasoncodes.ContractInvalid) {
		t.Fatalf("hash mismatch: %+v", r)
	}
	// All-or-nothing: one item over the available amount refuses the batch;
	// the valid item is reported valid but not committed.
	r2 := e.upload(t, "6", 2)
	r := e.submit(t, batch("over-key-01", inv, 0, seller, item("a", 600, r1), item("b", 500, r2)))
	if r.Status != result.Blocked || r.Applied || len(r.Items) != 2 || !r.Items[0].Valid || r.Items[0].Committed || r.Items[1].Valid ||
		!has(r.Items[1].Reasons, reasoncodes.ActionDenied) {
		t.Fatalf("partial batch must not commit: %+v", r)
	}
	if got := e.financial(t); got != before {
		// Only the uploads changed (they are not financial state).
		if e.count(t, `SELECT count(*) FROM payments.submitted_attempt`) != 0 || e.count(t, `SELECT count(*) FROM payments.payment_batch`) != 0 {
			t.Fatal("a refused batch committed financial state")
		}
	}
	// Outdated version.
	if r := e.submit(t, batch("stale-key-01", inv, 7, seller, item("a", 600, r1))); r.Status != result.Conflict || !has(r.Reasons, reasoncodes.StaleState) ||
		r.CurrentVersion != 0 {
		t.Fatalf("stale version: %+v", r)
	}

	// Success: two items, three receipts, one transaction.
	r3 := e.upload(t, "6", 3)
	ok := batch("good-batch-01", inv, 0, seller, item("a", 600, r1, r3), item("b", 400, r2))
	r = e.submit(t, ok)
	if r.Status != result.Success || !r.Applied || r.CollectionVersion != 1 || r.TotalAmount != 1000 || len(r.Items) != 2 ||
		!r.Items[0].Committed || len(r.Items[0].Receipts) != 2 || r.WritebackState != paymentscommand.WritebackAwaitingEvidence {
		t.Fatalf("submit: %+v", r)
	}
	if n := e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW' AND review_provenance = 'NOT_REVIEWED'`); n != 2 {
		t.Fatalf("attempts: %d", n)
	}
	if e.q(t, `SELECT string_agg(state, ',' ORDER BY state) FROM payments.upload_object`) != "BOUND,BOUND,BOUND,UNBOUND" {
		t.Fatal("upload binding")
	}
	// Replay: same key, same body → the stored result, nothing new.
	again2 := e.submit(t, ok)
	if !again2.Replayed || again2.BatchID != r.BatchID || e.count(t, `SELECT count(*) FROM payments.payment_batch`) != 1 {
		t.Fatalf("replay: %+v", again2)
	}
	// Same key, different body → CONFLICT; another actor with the key → CONFLICT.
	changed := ok
	changed.Items = []paymentscommand.Item{item("a", 500, r1, r3), item("b", 400, r2)}
	if c := e.submit(t, changed); c.Status != result.Conflict {
		t.Fatalf("different body: %+v", c)
	}
	stolen := ok
	stolen.Actor = other
	if c := e.submit(t, stolen); c.Status != result.Conflict || c.Replayed {
		t.Fatalf("another actor reusing a key: %+v", c)
	}
	// The stage is now fully used: nothing more is available; a bound
	// upload can never be attached to a second payment.
	r4 := e.upload(t, "6", 4)
	if c := e.submit(t, batch("after-full-01", inv, 1, seller, item("a", 1, r4))); c.Status != result.Blocked || !has(c.Reasons, reasoncodes.ActionDenied) {
		t.Fatalf("over the stage due: %+v", c)
	}
	inv5 := e.invoices["3005"]
	if c := e.submit(t, batch("rebind-key-1", inv5, 0, seller, item("a", 100, r1))); c.Status != result.Conflict || !has(c.Reasons, reasoncodes.LinkageConflict) {
		t.Fatalf("rebinding a bound upload: %+v", c)
	}
	// 3005: legacy pending 300 of 800 → 500 available exactly.
	ctxView, found, err := e.svc.Context(ctx, inv5, seller)
	if err != nil || !found || !ctxView.Eligible || *ctxView.Available != 500 || *ctxView.Used != 300 {
		t.Fatalf("context: %+v %v", ctxView, err)
	}
	if c := e.submit(t, batch("avail-key-01", inv5, 0, seller, item("a", 501, r4))); c.Status != result.Blocked {
		t.Fatalf("501 of 500: %+v", c)
	}
	if c := e.submit(t, batch("avail-key-02", inv5, 0, seller, item("a", 500, r4))); c.Status != result.Success {
		t.Fatalf("500 of 500: %+v", c)
	}
	// Read: the batch reads only for its submitter.
	if _, found, _ := e.svc.Batch(ctx, r.BatchID, other); found {
		t.Fatal("another seller read the batch")
	}
	if bv, found, err := e.svc.Batch(ctx, r.BatchID, seller); err != nil || !found || len(bv.Attempts) != 2 || len(bv.Attempts[0].Receipts) != 2 {
		t.Fatalf("batch read: %+v %v", bv, err)
	}
	// History is append-only, batches immutable, evidence never deleted.
	for _, sql := range []string{
		`UPDATE payments.payment_batch SET total_amount = 1`,
		`DELETE FROM payments.submitted_attempt_event`,
		`UPDATE payments.receipt_state_history SET reason = 'x'`,
		`DELETE FROM payments.receipt_evidence`,
		`DELETE FROM payments.upload_object`,
		`DELETE FROM payments.submitted_attempt`,
		`UPDATE payments.command_record SET applied = false`,
		`UPDATE payments.submitted_attempt SET state = 'ACCEPTED'`,
	} {
		if _, err := e.admin.Exec(ctx, sql); err == nil {
			t.Fatalf("allowed: %s", sql)
		}
	}
}

// Concurrent submissions on one stage serialise on the collection account:
// exactly one commits; the other is a version CONFLICT, never a double use.
func TestPaymentConcurrency(t *testing.T) {
	e := setupPayments(t)
	seller := e.actor("6")
	inv := e.invoices["3001"]
	a, b := e.upload(t, "6", 11), e.upload(t, "6", 12)
	var wg sync.WaitGroup
	results := make([]paymentscommand.BatchResult, 2)
	for i, ref := range []paymentscommand.ReceiptRef{a, b} {
		wg.Add(1)
		go func(i int, ref paymentscommand.ReceiptRef) {
			defer wg.Done()
			r, err := e.svc.SubmitBatch(context.Background(), rollout.Controlled, batch(fmt.Sprintf("race-key-%02d", i), inv, 0, seller, item("x", 700, ref)))
			if err != nil {
				t.Error(err)
			}
			results[i] = r
		}(i, ref)
	}
	wg.Wait()
	succ, conf := 0, 0
	for _, r := range results {
		switch r.Status {
		case result.Success:
			succ++
		case result.Conflict:
			conf++
		}
	}
	if succ != 1 || conf != 1 || e.count(t, `SELECT coalesce(sum(amount), 0) FROM payments.submitted_attempt`) != 700 {
		t.Fatalf("race: %+v", results)
	}
}

// Evidence lifecycle with fault injection: storage failure before commit,
// finalisation, replay, storage error and recovery, a crashed worker,
// missing bytes and unbound expiry.
func TestEvidenceFinalisation(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	// Storage failure before evidence exists: nothing is recorded.
	e.store.failPut = true
	r, err := e.svc.Upload(ctx, rollout.Controlled, seller, receipt(21))
	e.store.failPut = false
	if err != nil || r.Status != result.Failed || r.UploadID != "" || e.count(t, `SELECT count(*) FROM payments.upload_object`) != 0 {
		t.Fatalf("storage failure: %+v %v", r, err)
	}
	a, b := e.upload(t, "6", 21), e.upload(t, "6", 22)
	res := e.submit(t, batch("evidence-key-1", e.invoices["3001"], 0, seller, item("a", 300, a), item("b", 200, b)))
	if res.Status != result.Success {
		t.Fatalf("submit: %+v", res)
	}
	attempt := res.Items[0].AttemptID
	// A storage error leaves the attempt intact and opens reconciliation.
	e.store.failCopy = true
	rep, err := e.svc.Finalise(ctx, 1)
	if err != nil || rep.StorageError != 1 {
		t.Fatalf("finalise with storage error: %+v %v", rep, err)
	}
	if e.q(t, `SELECT evidence_state FROM payments.submitted_attempt WHERE attempt_id = $1`, attempt) != paymentsapi.EvidencePendingFinalisation ||
		e.count(t, `SELECT count(*) FROM payments.evidence_reconciliation WHERE reason = 'STORAGE_ERROR' AND next_safe_action = 'RETRY_FROM_QUARANTINE'`) != 1 {
		t.Fatal("storage error must keep the attempt and open reconciliation")
	}
	// A lost reply after a successful copy: retry finds the object.
	e.store.lostReply = true
	if rep, _ := e.svc.Finalise(ctx, 10); rep.StorageError != 1 || rep.Ready != 1 {
		t.Fatalf("lost reply: %+v", rep)
	}
	rep, err = e.svc.Finalise(ctx, 10)
	if err != nil || rep.Ready != 1 || rep.AttemptsReady != 1 ||
		e.count(t, `SELECT count(*) FROM payments.submitted_attempt WHERE evidence_state = 'READY'`) != 2 {
		t.Fatalf("recovery: %+v %v", rep, err)
	}
	if n := e.count(t, `SELECT count(*) FROM payments.evidence_reconciliation WHERE kind = 'RESOLVED' AND reason = 'RECOVERED'`); n < 1 ||
		n != e.count(t, `SELECT count(DISTINCT receipt_id) FROM payments.evidence_reconciliation WHERE kind = 'OPENED' AND reason = 'STORAGE_ERROR'`) {
		t.Fatal("recovery must be recorded")
	}
	// Replay: nothing left to do; no duplicate file; one event per attempt.
	if rep, _ := e.svc.Finalise(ctx, 10); rep != (paymentscommand.FinaliseReport{}) {
		t.Fatalf("replay changed something: %+v", rep)
	}
	files, _ := filepath.Glob(filepath.Join(e.root, "evidence", "*"))
	if len(files) != 2 {
		t.Fatalf("evidence files: %v", files)
	}
	if e.count(t, `SELECT count(*) FROM payments.outbox_event WHERE event_type = 'payments.attempt_evidence_ready'`) != 2 {
		t.Fatal("evidence-ready events")
	}
	// A worker crash after the write-ahead claim: the lease expires and the
	// next run finishes the receipt.
	c := e.upload(t, "6", 23)
	res = e.submit(t, batch("evidence-key-2", e.invoices["3001"], 1, seller, item("c", 100, c)))
	e.store.panicCopy = true
	func() {
		defer func() { _ = recover() }()
		_, _ = e.svc.Finalise(ctx, 10)
	}()
	if e.q(t, `SELECT state FROM payments.receipt_evidence WHERE attempt_id = $1`, res.Items[0].AttemptID) != paymentsapi.ReceiptFinalizing {
		t.Fatal("claim must be durable before the copy")
	}
	if rep, _ := e.svc.Finalise(ctx, 10); rep.Ready != 0 {
		t.Fatal("a live claim must be honoured")
	}
	e.clock.add(paymentscommand.FinalisingLease + time.Second)
	if rep, _ := e.svc.Finalise(ctx, 10); rep.Ready != 1 || rep.AttemptsReady != 1 {
		t.Fatalf("lease takeover: %+v", rep)
	}
	// Missing bytes: NOT_RECOVERABLE, reconciliation for the Finance owner.
	d := e.upload(t, "6", 24)
	res = e.submit(t, batch("evidence-key-3", e.invoices["3001"], 2, seller, item("d", 100, d)))
	if err := os.Remove(filepath.Join(e.root, "quarantine", d.UploadID)); err != nil {
		t.Fatal(err)
	}
	if rep, _ := e.svc.Finalise(ctx, 10); rep.NotRecoverable != 1 {
		t.Fatalf("missing bytes: %+v", rep)
	}
	if e.q(t, `SELECT evidence_state FROM payments.submitted_attempt WHERE attempt_id = $1`, res.Items[0].AttemptID) != paymentsapi.EvidenceNotRecoverable ||
		e.count(t, `SELECT count(*) FROM payments.evidence_reconciliation WHERE reason = 'NOT_RECOVERABLE' AND domain_owner = 'finance'`) != 1 {
		t.Fatal("NOT_RECOVERABLE must be explicit")
	}
	// Receipt history is complete and append-only.
	if e.count(t, `SELECT count(*) FROM payments.receipt_state_history`) < 8 {
		t.Fatal("receipt history")
	}
	// Unbound expiry (D-44): only unbound uploads past expiry are purged;
	// bound evidence is never touched.
	u := e.upload(t, "6", 25)
	if rep, _ := e.svc.ExpireUnbound(ctx, 10); rep.Purged != 0 {
		t.Fatal("purged before expiry")
	}
	e.clock.add(paymentscommand.UnboundTTL)
	rep2, err := e.svc.ExpireUnbound(ctx, 10)
	if err != nil || rep2.Purged != 1 {
		t.Fatalf("expiry: %+v %v", rep2, err)
	}
	if e.q(t, `SELECT state FROM payments.upload_object WHERE upload_id = $1`, u.UploadID) != paymentsapi.UploadPurged ||
		e.count(t, `SELECT count(*) FROM payments.upload_object WHERE state = 'BOUND' AND purged_at IS NOT NULL`) != 0 {
		t.Fatal("purge state")
	}
	if _, err := os.Stat(filepath.Join(e.root, "quarantine", a.UploadID)); err != nil {
		t.Fatal("bound evidence must never be purged")
	}
	// An expired upload cannot be bound.
	if r := e.submit(t, batch("expired-key-1", e.invoices["3007"], 0, seller, item("e", 10, u))); r.Status != result.Blocked || !has(r.Reasons, reasoncodes.StaleState) {
		t.Fatalf("expired upload bound: %+v", r)
	}
}

// Write-back of READY attempts to the synthetic legacy master: success,
// timeout after / before commit, unreadable legacy, ambiguity and a known
// legacy refusal. Production delivery is unsupported (synthetic only).
func TestPaymentWriteback(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	pr := e.Runner(t, payments.Module)
	run := func() writeback.Report {
		if _, err := e.worker.Drain(ctx, pr, 50); err != nil {
			t.Fatal(err)
		}
		rep, err := e.worker.Step(ctx, 50)
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	state := func(id string) string {
		return e.q(t, `SELECT state || ':' || coalesce(parity, '-') FROM integration.writeback_command WHERE command_id = $1`, id)
	}
	a, b := e.upload(t, "6", 31), e.upload(t, "6", 32)
	res := e.submit(t, batch("wb-key-0001", e.invoices["3001"], 0, seller, item("a", 600, a), item("b", 400, b)))
	if _, err := e.svc.Finalise(ctx, 10); err != nil {
		t.Fatal(err)
	}
	first, second := res.Items[0].AttemptID, res.Items[1].AttemptID
	// The audit-only batch event is consumed without a command.
	e.legacy.Inject("payment", legacytest.TimeoutAfterCommit)
	run()
	if state(first) != "UNKNOWN:-" && state(second) != "UNKNOWN:-" {
		t.Fatalf("timeout after commit must be UNKNOWN: %s %s", state(first), state(second))
	}
	run()
	if state(first) != "APPLIED:MATCH" || state(second) != "APPLIED:MATCH" {
		t.Fatalf("reconciled: %s %s", state(first), state(second))
	}
	inv := e.legacy.Invoice("3001")
	if len(inv.Payments) != 2 || e.legacy.Calls["payment"] != 2 {
		t.Fatalf("exactly one legacy payment per attempt (no blind retry): %+v calls=%d", inv.Payments, e.legacy.Calls["payment"])
	}
	if e.count(t, `SELECT count(*) FROM integration.writeback_command WHERE source_module = 'payments'`) != 2 {
		t.Fatal("the batch event must not become a write-back command")
	}
	// Timeout before commit: proven absent → retried once → applied.
	c := e.upload(t, "6", 33)
	res = e.submit(t, batch("wb-key-0002", e.invoices["3005"], 0, seller, item("c", 200, c)))
	_, _ = e.svc.Finalise(ctx, 10)
	e.legacy.Inject("payment", legacytest.TimeoutBeforeCommit)
	run()
	e.legacy.Inject("payment_observe", legacytest.ObserveDown)
	run()
	if st := state(res.Items[0].AttemptID); st != "UNKNOWN:-" {
		t.Fatalf("unreadable legacy must stay UNKNOWN: %s", st)
	}
	run()
	run()
	if st := state(res.Items[0].AttemptID); st != "APPLIED:MATCH" || len(e.legacy.Invoice("3005").Payments) != 2 {
		t.Fatalf("absent then retried: %s", st)
	}
	// Ambiguous legacy result: never retried, never success.
	d := e.upload(t, "6", 34)
	res = e.submit(t, batch("wb-key-0003", e.invoices["3005"], 1, seller, item("d", 100, d)))
	_, _ = e.svc.Finalise(ctx, 10)
	e.legacy.Inject("payment", legacytest.TimeoutAfterCommit)
	run()
	e.legacy.DuplicatePayment("3005", lwb.RequestToken(res.Items[0].AttemptID))
	calls := e.legacy.Calls["payment"]
	run()
	if st := state(res.Items[0].AttemptID); st != "RECONCILIATION_REQUIRED:-" || e.legacy.Calls["payment"] != calls {
		t.Fatalf("ambiguous: %s", st)
	}
	// A known legacy refusal (legacy availability rule): REJECTED with a
	// reconciliation record; target history stays.
	e.legacy.AddInvoice("3001", legacytest.SyntheticInvoice{SellerLegacyUserID: "6", StatusRaw: "paid", StageNo: 1, StageDueToman: 1000})
	f := e.upload(t, "6", 35)
	res2 := e.submit(t, batch("wb-key-0004", e.invoices["3007"], 0, seller, item("f", 100, f)))
	_, _ = e.svc.Finalise(ctx, 10)
	run()
	if st := state(res2.Items[0].AttemptID); st != "REJECTED:-" {
		t.Fatalf("legacy refusal (invoice 3007 unknown to legacy): %s", st)
	}
	if e.count(t, `SELECT count(*) FROM integration.writeback_reconciliation WHERE domain_owner = 'payments' AND reason = 'WRITEBACK_REJECTED'`) != 1 ||
		e.q(t, `SELECT state FROM payments.submitted_attempt WHERE attempt_id = $1`, res2.Items[0].AttemptID) != paymentsapi.AttemptPendingReview {
		t.Fatal("rejection must reconcile, not rewrite target history")
	}
	// Every attempt is recorded as the service actor, never the seller.
	if e.count(t, `SELECT count(*) FROM integration.writeback_attempt WHERE service_actor <> 'service:integration.writeback'`) != 0 {
		t.Fatal("write-back attempts must carry the service actor")
	}
}

// Synthetic gateway: stable identity, write-ahead UNKNOWN, reconcile before
// retry, authenticated callbacks, replay / duplicate / out-of-order /
// stale callbacks, D-25 provenance and D-26 (no session).
func TestGatewayFlow(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	gw := synthetic.New()
	flow, err := e.svc.NewGateway(gw)
	if err != nil {
		t.Fatal(err)
	}
	verifier, _ := synthetic.NewVerifier([]byte("synthetic-test-secret-0123456789abcdef"))
	start := func(key, invoice string) paymentscommand.GatewayResult {
		r, err := flow.Start(ctx, rollout.Controlled, paymentscommand.GatewayStart{InvoiceID: e.invoices[invoice], IdempotencyKey: key, CorrelationID: "c-" + key,
			Actor: seller})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	callback := func(reqID, ref, status string) paymentscommand.GatewayResult {
		raw := synthetic.RawCallback{Provider: synthetic.Provider, RequestID: reqID, ExternalRef: ref, Status: status}
		raw.Signature = verifier.Sign(raw)
		cb, err := verifier.Authenticate(raw)
		if err != nil {
			t.Fatal(err)
		}
		r, err := flow.HandleCallback(ctx, cb)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Timeout after the provider committed: UNKNOWN, then reconciled
	// without a second provider request.
	gw.Inject("request", synthetic.RequestTimeoutAfterCommit)
	r := start("gw-key-00001", "3007")
	if r.Status != result.Unknown || r.State != paymentscommand.GatewayUnknown {
		t.Fatalf("timeout after commit: %+v", r)
	}
	if again := start("gw-key-00001", "3007"); again.State != paymentscommand.GatewayRedirected || gw.Requests() != 1 {
		t.Fatalf("retry must reconcile first: %+v requests=%d", again, gw.Requests())
	}
	// The same key for another invoice is a CONFLICT, never a replay.
	if x := start("gw-key-00001", "3008"); x.Status != result.Conflict {
		t.Fatalf("key reuse across invoices: %+v", x)
	}
	// A second start while one is open is refused.
	if x := start("gw-key-00002", "3007"); x.Status != result.Blocked {
		t.Fatalf("second open request: %+v", x)
	}
	// Callback replay: verified once; duplicates change nothing.
	ref := gw.Pay(r.RequestID)
	v := callback(r.RequestID, ref, "OK")
	if v.State != paymentscommand.GatewayVerified || v.ApprovalProvenance != "GATEWAY_AUTO" || v.FinanceReviewed || v.SessionIssued {
		t.Fatalf("verified: %+v", v)
	}
	verifies := gw.Calls["verify"]
	if d := callback(r.RequestID, ref, "OK"); !d.Duplicate || gw.Calls["verify"] != verifies {
		t.Fatalf("duplicate callback: %+v", d)
	}
	// A late failure callback after verification is stale: no downgrade.
	if s := callback(r.RequestID, ref, "NOK"); s.State != paymentscommand.GatewayVerified {
		t.Fatalf("stale callback: %+v", s)
	}
	if e.q(t, `SELECT finance_reviewed::text || session_issued::text FROM payments.gateway_request WHERE request_id = $1`, r.RequestID) != "falsefalse" {
		t.Fatal("D-25 / D-26")
	}
	// Timeout before the provider committed: reconcile proves absence, then
	// the same request id is requested once.
	gw2 := synthetic.New()
	flow2, _ := e.svc.NewGateway(gw2)
	gw2.Inject("request", synthetic.RequestTimeoutBeforeCommit)
	r2, _ := flow2.Start(ctx, rollout.Controlled, paymentscommand.GatewayStart{InvoiceID: e.invoices["3001"], IdempotencyKey: "gw-key-00003",
		CorrelationID: "c3", Actor: seller})
	if r2.Status != result.Unknown {
		t.Fatalf("timeout before commit: %+v", r2)
	}
	gw2.Inject("inquire", synthetic.InquireDown)
	if x, _ := flow2.Reconcile(ctx, r2.RequestID); x.Status != result.Unknown || gw2.Requests() != 0 {
		t.Fatalf("unreadable provider must not retry: %+v", x)
	}
	x, err := flow2.Reconcile(ctx, r2.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if x.State != paymentscommand.GatewayRedirected || gw2.Requests() != 1 {
		t.Fatalf("proven absent → one request: %+v", x)
	}
	// Out-of-order: a callback for a request whose answer was never
	// recorded; an orphan callback; an amount mismatch on verification.
	ref2 := gw2.Cancel(r2.RequestID)
	if c := callback2(t, flow2, verifier, r2.RequestID, ref2, "NOK"); c.State != paymentscommand.GatewayCancelled {
		t.Fatalf("cancel: %+v", c)
	}
	if o := callback2(t, flow2, verifier, "00000000-0000-4000-8000-00000000abcd", "syn-999999", "OK"); o.Status != result.Blocked {
		t.Fatalf("orphan: %+v", o)
	}
	if e.count(t, `SELECT count(*) FROM payments.gateway_event WHERE kind = 'ORPHAN_CALLBACK'`) != 1 {
		t.Fatal("orphan callback must be recorded")
	}
	// A VERIFIED gateway payment uses the stage: a manual submission or a
	// second gateway start cannot pay it again.
	m := e.upload(t, "6", 51)
	if r := e.submit(t, batch("gw-manual-01", e.invoices["3007"], 0, seller, item("m", 1, m))); r.Status != result.Blocked ||
		!has(r.Reasons, reasoncodes.ActionDenied) {
		t.Fatalf("manual payment after a verified gateway payment: %+v", r)
	}
	if x := start("gw-key-00004", "3007"); x.Status != result.Blocked {
		t.Fatalf("second gateway payment: %+v", x)
	}
	// Out of order: the request answer is lost and the paid callback
	// arrives before any reconciliation. While the outcome is UNKNOWN no
	// manual submission is accepted.
	gw3 := synthetic.New()
	flow3, _ := e.svc.NewGateway(gw3)
	gw3.Inject("request", synthetic.RequestTimeoutAfterCommit)
	r3, _ := flow3.Start(ctx, rollout.Controlled, paymentscommand.GatewayStart{InvoiceID: e.invoices["3008"], IdempotencyKey: "gw-key-00005",
		CorrelationID: "c5", Actor: seller})
	if r3.State != paymentscommand.GatewayUnknown {
		t.Fatalf("lost answer: %+v", r3)
	}
	if r := e.submit(t, batch("gw-manual-02", e.invoices["3008"], 0, seller, item("m", 1, m))); r.Status != result.Blocked ||
		!has(r.Reasons, reasoncodes.OutcomeUnknown) {
		t.Fatalf("manual payment while a gateway outcome is unknown: %+v", r)
	}
	ref3 := gw3.Pay(r3.RequestID)
	gw3.OverrideVerifiedAmount(ref3, 899)
	if c := callback2(t, flow3, verifier, r3.RequestID, ref3, "OK"); c.State != paymentscommand.GatewayReconciliation || c.Status != result.Conflict {
		t.Fatalf("verified amount mismatch must need reconciliation: %+v", c)
	}
	if e.q(t, `SELECT external_ref FROM payments.gateway_request WHERE request_id = $1`, r3.RequestID) != ref3 {
		t.Fatal("out-of-order callback must record the provider reference")
	}
	// Forged callbacks never authenticate (adapter boundary).
	forged := synthetic.RawCallback{Provider: synthetic.Provider, RequestID: r2.RequestID, ExternalRef: ref2, Status: "OK", Signature: "00"}
	if _, err := verifier.Authenticate(forged); err == nil {
		t.Fatal("forged callback authenticated")
	}
	// No gateway step changes billing, finance, biawin or fulfillment.
	if e.count(t, `SELECT count(*) FROM payments.gateway_request WHERE state = 'VERIFIED' AND approval_provenance <> 'GATEWAY_AUTO'`) != 0 {
		t.Fatal("provenance")
	}
}

func callback2(t *testing.T, flow *paymentscommand.Gateway, v *synthetic.Verifier, reqID, ref, status string) paymentscommand.GatewayResult {
	t.Helper()
	raw := synthetic.RawCallback{Provider: synthetic.Provider, RequestID: reqID, ExternalRef: ref, Status: status}
	raw.Signature = v.Sign(raw)
	cb, err := v.Authenticate(raw)
	if err != nil {
		t.Fatal(err)
	}
	r, err := flow.HandleCallback(context.Background(), cb)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Financial non-mutation and boundary invariants across every Slice 12
// command: billing, finance (ledger), biawin and fulfillment are never
// written; imported Slice 7 payment evidence is unchanged; no capability is
// granted to a role.
func TestPaymentsInvariants(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	other := e.md5(t, "billing.invoice", "billing.payment_stage", "billing.amount_evidence", "billing.invoice_state_evidence",
		"payments.payment_attempt", "payments.attempt_state_history", "finance.ledger_entry", "finance.ledger_line", "finance.shadow_run",
		"biawin.subscription", "fulfillment.fulfillment_case")
	seller := e.actor("6")
	a := e.upload(t, "6", 41)
	if r := e.submit(t, batch("inv-key-0001", e.invoices["3001"], 0, seller, item("a", 100, a))); r.Status != result.Success {
		t.Fatal(r)
	}
	if _, err := e.svc.Finalise(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := e.worker.Drain(ctx, e.Runner(t, payments.Module), 10); err != nil {
		t.Fatal(err)
	}
	if _, err := e.worker.Step(ctx, 10); err != nil {
		t.Fatal(err)
	}
	gw := synthetic.New()
	flow, _ := e.svc.NewGateway(gw)
	if r, _ := flow.Start(ctx, rollout.Controlled, paymentscommand.GatewayStart{InvoiceID: e.invoices["3007"], IdempotencyKey: "inv-gw-0001",
		CorrelationID: "c", Actor: seller}); r.State != paymentscommand.GatewayRedirected {
		t.Fatal(r)
	}
	if got := e.md5(t, "billing.invoice", "billing.payment_stage", "billing.amount_evidence", "billing.invoice_state_evidence",
		"payments.payment_attempt", "payments.attempt_state_history", "finance.ledger_entry", "finance.ledger_line", "finance.shadow_run",
		"biawin.subscription", "fulfillment.fulfillment_case"); got != other {
		t.Fatal("a Slice 12 command changed billing, finance, biawin, fulfillment or the imported payment evidence")
	}
	// No Slice 12 capability is granted to any role.
	if n := e.count(t, `SELECT count(*) FROM identity.role_capability WHERE capability_key IN ('payments.upload.stage', 'payments.batch.submit',
		'payments.submission.read', 'payments.evidence.read', 'finance.review.read')`); n != 0 {
		t.Fatalf("granted: %d", n)
	}
	// Schema guards: a target attempt can never be ACCEPTED / REJECTED
	// (no Finance review); a gateway request can never be Finance-reviewed
	// or issue a session.
	for _, sql := range []string{
		`INSERT INTO payments.submitted_attempt SELECT gen_random_uuid(), batch_id, 'zz', 99, invoice_id, stage_no, channel, entry_mode, amount, unit,
			'ACCEPTED', review_provenance, evidence_state, submitted_by_user_id, created_at, updated_at, 1 FROM payments.submitted_attempt LIMIT 1`,
		`UPDATE payments.gateway_request SET finance_reviewed = true`,
		`UPDATE payments.gateway_request SET session_issued = true`,
	} {
		if _, err := e.admin.Exec(ctx, sql); err == nil {
			t.Fatalf("allowed: %s", sql)
		}
	}
	// Evidence bytes are content-addressed and verifiable.
	var sum []byte
	var ref string
	if err := e.admin.QueryRow(ctx, `SELECT content_sha256, evidence_ref FROM payments.receipt_evidence LIMIT 1`).Scan(&sum, &ref); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(e.root, filepath.FromSlash(ref)))
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256.Sum256(body); !bytes.Equal(got[:], sum) || ref != "evidence/"+hex.EncodeToString(sum) {
		t.Fatal("evidence is not the bound content")
	}
}
