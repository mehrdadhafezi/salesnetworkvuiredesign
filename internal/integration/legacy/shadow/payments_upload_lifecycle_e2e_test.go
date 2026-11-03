package shadow_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// activeUploadDurable asserts the evidence invariant for one upload: an
// UNBOUND or BOUND upload references bytes that exist and hash to its
// recorded content hash.
func (e *paymentsEnv) activeUploadDurable(t *testing.T, uploadID string) {
	t.Helper()
	var ref, state string
	var sum []byte
	if err := e.admin.QueryRow(context.Background(), `SELECT quarantine_ref, state, content_sha256 FROM payments.upload_object WHERE upload_id = $1`,
		uploadID).Scan(&ref, &state, &sum); err != nil {
		t.Fatal(err)
	}
	if state != paymentsapi.UploadUnbound && state != paymentsapi.UploadBound {
		t.Fatalf("upload %s is %s, not active", uploadID, state)
	}
	body, err := os.ReadFile(filepath.Join(e.root, filepath.FromSlash(ref)))
	if err != nil {
		t.Fatalf("active upload %s (%s) references missing bytes %s: %v", uploadID, state, ref, err)
	}
	if got := sha256.Sum256(body); !bytes.Equal(got[:], sum) {
		t.Fatalf("active upload %s references bytes with another hash", uploadID)
	}
}

func (e *paymentsEnv) expire(t *testing.T) paymentscommand.ExpireReport {
	t.Helper()
	rep, err := e.svc.ExpireUnbound(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func (e *paymentsEnv) uploadState(t *testing.T, id string) string {
	t.Helper()
	return e.q(t, `SELECT state FROM payments.upload_object WHERE upload_id = $1`, id)
}

func quarantineExists(e *paymentsEnv, t *testing.T, id string) bool {
	t.Helper()
	ref := e.q(t, `SELECT quarantine_ref FROM payments.upload_object WHERE upload_id = $1`, id)
	_, err := os.Stat(filepath.Join(e.root, filepath.FromSlash(ref)))
	return err == nil
}

// Finding 1: two identical uploads reviving the same PURGED upload at the
// same time. Both pass the lookup before either commits (barrier on Put).
// The loser must never delete the winner's bytes.
func TestUploadRevivalRace(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	first := e.upload(t, "6", 71)
	e.clock.add(paymentscommand.UnboundTTL + time.Minute)
	e.expire(t)
	if st := e.uploadState(t, first.UploadID); st != paymentsapi.UploadPurged {
		t.Fatalf("setup: upload is %s", st)
	}
	var arrived sync.WaitGroup
	arrived.Add(2)
	release := make(chan struct{})
	e.store.mu.Lock()
	e.store.putHook = func(string) { arrived.Done(); <-release }
	e.store.mu.Unlock()
	results := make([]paymentscommand.UploadResult, 2)
	errs := make([]error, 2)
	var done sync.WaitGroup
	for i := range 2 {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			results[i], errs[i] = e.svc.Upload(ctx, rollout.Controlled, seller, receipt(71))
		}(i)
	}
	arrived.Wait() // both requests looked the upload up and reached storage
	close(release)
	done.Wait()
	e.store.mu.Lock()
	e.store.putHook = nil
	e.store.mu.Unlock()
	for i := range 2 {
		if errs[i] != nil || results[i].Status != result.Success || results[i].UploadID != first.UploadID {
			t.Fatalf("request %d: %+v %v", i, results[i], errs[i])
		}
	}
	if n := e.count(t, `SELECT count(*) FROM payments.upload_object WHERE uploader_user_id = $1`, e.users["6"]); n != 1 {
		t.Fatalf("upload identity not preserved: %d rows", n)
	}
	e.activeUploadDurable(t, first.UploadID)
	// The revived upload is usable as evidence end to end.
	r := e.submit(t, batch("revival-race-01", e.invoices["3001"], 0, seller, item("a", 100, first)))
	if r.Status != result.Success {
		t.Fatalf("revived upload not bindable: %+v", r)
	}
	if rep, err := e.svc.Finalise(ctx, 10); err != nil || rep.Ready != 1 || rep.NotRecoverable != 0 {
		t.Fatalf("revived evidence must finalise: %+v %v", rep, err)
	}
	// Exactly one revival is recorded; the losing copy left no referenced bytes behind.
	if n := e.count(t, `SELECT count(*) FROM payments.upload_event WHERE upload_id = $1 AND kind = 'REVIVED'`, first.UploadID); n != 1 {
		t.Fatalf("revivals recorded: %d", n)
	}
	files, _ := filepath.Glob(filepath.Join(e.root, "quarantine", "*"))
	if len(files) != 1 {
		t.Fatalf("quarantine objects: %v (the losing copy must be removed, the winner's kept)", files)
	}
}

// Finding 2: the expiry lifecycle is durable and retryable: a failed
// deletion, a crash before or after deletion, and repeated runs all
// converge on PURGED without ever touching bound evidence.
func TestUnboundExpiryLifecycle(t *testing.T) {
	e := setupPayments(t)
	seller := e.actor("6")
	ok, failing, crashBefore, crashAfter := e.upload(t, "6", 81), e.upload(t, "6", 82), e.upload(t, "6", 83), e.upload(t, "6", 84)
	bound := e.upload(t, "6", 85)
	if r := e.submit(t, batch("expiry-bound-01", e.invoices["3001"], 0, seller, item("a", 100, bound))); r.Status != result.Success {
		t.Fatal(r)
	}
	e.clock.add(paymentscommand.UnboundTTL + time.Minute)

	// 1 + 2: one deletion fails; the others succeed. The failed one stays
	// eligible (EXPIRING) with the failure recorded.
	e.store.mu.Lock()
	e.store.deleteFails = 1
	e.store.mu.Unlock()
	rep := e.expire(t)
	if rep.Purged != 3 || rep.DeleteFailed != 1 {
		t.Fatalf("first run: %+v", rep)
	}
	expiring := e.q(t, `SELECT upload_id::text FROM payments.upload_object WHERE state = 'EXPIRING'`)
	if e.count(t, `SELECT purge_attempts FROM payments.upload_object WHERE upload_id = $1`, expiring) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.upload_event WHERE upload_id = $1 AND kind = 'PURGE_FAILED'`, expiring) != 1 {
		t.Fatal("a failed deletion must stay eligible and be recorded")
	}
	rep = e.expire(t)
	if rep.Purged != 1 || rep.DeleteFailed != 0 {
		t.Fatalf("retry: %+v", rep)
	}
	for _, u := range []paymentscommand.ReceiptRef{ok, failing, crashBefore, crashAfter} {
		if e.uploadState(t, u.UploadID) != paymentsapi.UploadPurged {
			t.Fatalf("upload %s not purged", u.UploadID)
		}
	}
	if files, _ := filepath.Glob(filepath.Join(e.root, "quarantine", "*")); len(files) != 1 {
		t.Fatalf("only the bound upload's bytes may remain: %v", files)
	}

	// 3 + 4: crash before deletion, crash after deletion before the
	// acknowledgement. Both converge on the next run; deletion is idempotent.
	a, b := e.upload(t, "6", 86), e.upload(t, "6", 87)
	e.clock.add(paymentscommand.UnboundTTL + time.Minute)
	e.store.mu.Lock()
	e.store.crashBeforeDelete = true
	e.store.mu.Unlock()
	func() { defer func() { _ = recover() }(); _, _ = e.svc.ExpireUnbound(context.Background(), 1) }()
	crashed := e.q(t, `SELECT upload_id::text FROM payments.upload_object WHERE state = 'EXPIRING'`)
	if !quarantineExists(e, t, crashed) {
		t.Fatal("crash before deletion: bytes must still exist")
	}
	e.store.mu.Lock()
	e.store.crashAfterDelete = true
	e.store.mu.Unlock()
	func() { defer func() { _ = recover() }(); _, _ = e.svc.ExpireUnbound(context.Background(), 50) }()
	if n := e.count(t, `SELECT count(*) FROM payments.upload_object WHERE state = 'EXPIRING'`); n < 1 {
		t.Fatal("crash after deletion: the acknowledgement was lost, the upload must stay EXPIRING")
	}
	rep = e.expire(t)
	if e.uploadState(t, a.UploadID) != paymentsapi.UploadPurged || e.uploadState(t, b.UploadID) != paymentsapi.UploadPurged || rep.DeleteFailed != 0 {
		t.Fatalf("crash recovery: %+v", rep)
	}

	// 6: repeated cleanup never deletes bound evidence.
	for range 3 {
		e.clock.add(paymentscommand.UnboundTTL)
		e.expire(t)
	}
	if e.uploadState(t, bound.UploadID) != paymentsapi.UploadBound {
		t.Fatal("bound upload changed state")
	}
	e.activeUploadDurable(t, bound.UploadID)
	if n := e.count(t, `SELECT count(*) FROM payments.upload_event WHERE kind IN ('EXPIRING', 'PURGED') AND upload_id = $1`, bound.UploadID); n != 0 {
		t.Fatal("bound evidence entered the expiry lifecycle")
	}
	// The expiry history is append-only and complete for each purged upload.
	if n := e.count(t, `SELECT count(*) FROM payments.upload_event WHERE upload_id = $1 AND kind IN ('EXPIRING', 'PURGED')`, ok.UploadID); n != 2 {
		t.Fatalf("expiry history: %d", n)
	}
	if _, err := e.admin.Exec(context.Background(), `DELETE FROM payments.upload_event`); err == nil {
		t.Fatal("upload history must be append-only")
	}
}

// Finding 2 (5): binding and expiration racing on the same upload, from two
// services whose clocks disagree about the expiry. The upload row lock
// serialises them; either the binding wins (bytes kept, never purged) or
// the expiry wins (the batch is refused, nothing bound) — never both.
func TestBindingVersusExpiry(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	late := &moveClock{at: e.clock.at.Add(paymentscommand.UnboundTTL + time.Minute)}
	expirer, err := paymentscommand.New(e.Runner(t, payments.Module), e.store, e.billingQ, late)
	if err != nil {
		t.Fatal(err)
	}
	for round, order := range []string{"bind-first", "expire-first", "race"} {
		u := e.upload(t, "6", 90+round)
		key := "bind-expire-" + order
		var br paymentscommand.BatchResult
		// Read before any goroutine starts: the admin connection is not
		// safe for concurrent use.
		version := e.count(t, `SELECT coalesce(max(version), 0) FROM payments.collection_account WHERE invoice_id = $1`, e.invoices["3007"])
		bind := func() {
			var err error
			br, err = e.svc.SubmitBatch(ctx, rollout.Controlled, batch(key, e.invoices["3007"], version, seller, item("a", 1, u)))
			if err != nil {
				t.Error(err)
			}
		}
		exp := func() {
			if _, err := expirer.ExpireUnbound(ctx, 50); err != nil {
				t.Error(err)
			}
		}
		switch order {
		case "bind-first":
			bind()
			exp()
		case "expire-first":
			exp()
			bind()
		default:
			// Hold the upload row so both block, then release them together.
			tx, err := e.admin.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `SELECT 1 FROM payments.upload_object WHERE upload_id = $1 FOR UPDATE`, u.UploadID); err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); bind() }()
			go func() { defer wg.Done(); exp() }()
			time.Sleep(200 * time.Millisecond)
			_ = tx.Rollback(ctx)
			wg.Wait()
			exp() // a skipped (locked) expiry is retried by the next run
		}
		st := e.uploadState(t, u.UploadID)
		switch {
		case br.Status == result.Success:
			if st != paymentsapi.UploadBound {
				t.Fatalf("%s: bound batch but upload %s", order, st)
			}
			e.activeUploadDurable(t, u.UploadID)
		case st == paymentsapi.UploadPurged:
			if br.Status != result.Blocked || !has(br.Reasons, reasoncodes.StaleState) ||
				e.count(t, `SELECT count(*) FROM payments.receipt_evidence WHERE upload_id = $1`, u.UploadID) != 0 {
				t.Fatalf("%s: purged upload but batch %+v", order, br)
			}
		default:
			t.Fatalf("%s: upload %s, batch %+v", order, st, br)
		}
		if order == "bind-first" && br.Status != result.Success {
			t.Fatalf("bind-first must bind: %+v", br)
		}
		if order == "expire-first" && st != paymentsapi.UploadPurged {
			t.Fatalf("expire-first must purge: %s", st)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM payments.receipt_evidence r JOIN payments.upload_object u USING (upload_id) WHERE u.state <> 'BOUND'`); n != 0 {
		t.Fatal("a receipt references an upload that is not bound")
	}
}

// Finding 3: an UNBOUND upload past expires_at, before any cleanup run,
// is never handed back as usable. Re-sending the same bytes renews it under
// the same identity, with durable bytes and a new expiry.
func TestExpiredUploadRenewal(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	seller := e.actor("6")
	u := e.upload(t, "6", 95)
	e.clock.add(paymentscommand.UnboundTTL + time.Minute)
	again, err := e.svc.Upload(ctx, rollout.Controlled, seller, receipt(95))
	if err != nil || again.Status != result.Success || again.UploadID != u.UploadID || again.State != paymentsapi.UploadUnbound {
		t.Fatalf("renewal: %+v %v", again, err)
	}
	now := e.clock.Now().Time()
	if again.ExpiresAt == nil || !again.ExpiresAt.After(now) {
		t.Fatalf("an expired upload was returned as usable: expires_at %v, now %v", again.ExpiresAt, now)
	}
	e.activeUploadDurable(t, u.UploadID)
	if n := e.count(t, `SELECT count(*) FROM payments.upload_event WHERE upload_id = $1 AND kind = 'RENEWED'`, u.UploadID); n != 1 {
		t.Fatalf("renewal history: %d", n)
	}
	// The renewed upload binds without any cleanup run in between.
	if r := e.submit(t, batch("renewed-bind-01", e.invoices["3001"], 0, seller, item("a", 100, u))); r.Status != result.Success {
		t.Fatalf("renewed upload must bind: %+v", r)
	}
	// Renewal when the expired bytes are missing restages fresh bytes under
	// the same identity (never references missing bytes).
	v := e.upload(t, "6", 96)
	e.clock.add(paymentscommand.UnboundTTL + time.Minute)
	if err := os.Remove(filepath.Join(e.root, filepath.FromSlash(e.q(t, `SELECT quarantine_ref FROM payments.upload_object WHERE upload_id = $1`,
		v.UploadID)))); err != nil {
		t.Fatal(err)
	}
	r2, err := e.svc.Upload(ctx, rollout.Controlled, seller, receipt(96))
	if err != nil || r2.UploadID != v.UploadID || r2.ExpiresAt == nil || !r2.ExpiresAt.After(e.clock.Now().Time()) {
		t.Fatalf("restage: %+v %v", r2, err)
	}
	e.activeUploadDurable(t, v.UploadID)
}
