package compat

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// R2 review P2: evidence writes are detached from the request's
// cancellation but bounded by their own deadline; an unavailable or stalled
// evidence store is UNKNOWN, never a proven result, and its error text never
// leaks into the outcome.

// ctlRecorder observes the context it is given and can fail or block.
type ctlRecorder struct {
	mu        sync.Mutex
	got       []Exchange
	failNext  int  // fail this many writes with a store error
	block     bool // block until the record context ends
	sawCancel bool // the record context was already cancelled on entry
	deadline  time.Duration
}

var errStore = errors.New("store down: host=db-staging-7 role=crm_owner_integration password=hunter2-synthetic")

func (r *ctlRecorder) Record(ctx context.Context, e Exchange) error {
	r.mu.Lock()
	if ctx.Err() != nil {
		r.sawCancel = true
	}
	if d, ok := ctx.Deadline(); ok {
		r.deadline = time.Until(d)
	}
	fail, block := r.failNext > 0, r.block
	if fail {
		r.failNext--
	}
	r.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	if fail {
		return errStore
	}
	r.mu.Lock()
	r.got = append(r.got, e)
	r.mu.Unlock()
	return nil
}

func applied200(req request) (int, string, bool) {
	return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "77001"}), true
}

func noLeak(t *testing.T, err error) {
	t.Helper()
	for _, s := range []string{"hunter2", "db-staging-7", "crm_owner_integration", "password", "77001", clientKeyString(), "جواب"} {
		if strings.Contains(err.Error(), s) {
			t.Fatalf("diagnostic leaks %q: %v", s, err)
		}
	}
}

func clientKeyString() string { return string(clientKey) }

// Parent cancellation: the request is cut, the evidence is still written
// (best effort, not cancelled), the outcome is UNKNOWN.
func TestRecordSurvivesParentCancellation(t *testing.T) {
	rec := &ctlRecorder{}
	release := make(chan struct{})
	c, _ := setupWith(t, func(req request) (int, string, bool) {
		<-release
		return applied200(req)
	}, rec, time.Second)
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	_, err := c.ApplyLeadOutcome(ctx, lead())
	if !isUnknown(err) {
		t.Fatalf("a cancelled request must be UNKNOWN: %v", err)
	}
	if rec.sawCancel || len(rec.got) != 1 || rec.got[0].Verdict != VerdictTransport {
		t.Fatalf("evidence must be written after cancellation: saw_cancel=%v got=%+v", rec.sawCancel, rec.got)
	}
	if rec.deadline <= 0 || rec.deadline > time.Second {
		t.Fatalf("the evidence write must carry its own bounded deadline: %s", rec.deadline)
	}
}

// Parent deadline expiration: same, with the parent's deadline replaced by
// the record deadline (not inherited, not unbounded).
func TestRecordAfterParentDeadline(t *testing.T) {
	rec := &ctlRecorder{}
	release := make(chan struct{})
	c, _ := setupWith(t, func(req request) (int, string, bool) {
		<-release
		return applied200(req)
	}, rec, 700*time.Millisecond)
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.ApplyLeadOutcome(ctx, lead())
	if !isUnknown(err) || len(rec.got) != 1 || rec.sawCancel {
		t.Fatalf("deadline: %v got=%d saw_cancel=%v", err, len(rec.got), rec.sawCancel)
	}
	if v := rec.got[0].Verdict; v != VerdictTimeout && v != VerdictTransport {
		t.Fatalf("verdict: %s", v)
	}
	if rec.deadline < 500*time.Millisecond || rec.deadline > 700*time.Millisecond {
		t.Fatalf("record deadline must be its own (~700ms), not the expired parent's: %s", rec.deadline)
	}
}

// Recorder temporarily unavailable: UNKNOWN (no false success, no leak);
// the next exchange, once the store is back, is recorded and proven.
func TestRecorderTemporarilyUnavailable(t *testing.T) {
	rec := &ctlRecorder{failNext: 1}
	c, f := setupWith(t, applied200, rec, time.Second)
	res, err := c.ApplyLeadOutcome(context.Background(), lead())
	if !isUnknown(err) || res.Outcome == result.Success || res.Ref != "" {
		t.Fatalf("an unrecorded APPLIED must stay UNKNOWN: %+v %v", res, err)
	}
	if !strings.Contains(err.Error(), "evidence store unavailable") {
		t.Fatalf("diagnostic: %v", err)
	}
	noLeak(t, err)
	res, err = c.ApplyLeadOutcome(context.Background(), lead())
	if err != nil || res.Outcome != result.Success || len(rec.got) != 1 || f.calls != 2 {
		t.Fatalf("recovered store: %+v %v", res, err)
	}
}

// Recorder blocked beyond its own deadline: the call returns within the
// record deadline (bounded), as UNKNOWN, even though legacy acknowledged.
func TestRecorderBlockedBeyondDeadline(t *testing.T) {
	rec := &ctlRecorder{block: true}
	c, _ := setupWith(t, applied200, rec, 150*time.Millisecond)
	start := time.Now()
	res, err := c.ApplyLeadOutcome(context.Background(), lead())
	elapsed := time.Since(start)
	if !isUnknown(err) || res.Outcome == result.Success || !strings.Contains(err.Error(), "exceeded its deadline") {
		t.Fatalf("blocked store: %+v %v", res, err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("a stalled evidence store must be bounded: %s", elapsed)
	}
	noLeak(t, err)
	// Rejections and observations obey the same rule.
	rc, _ := setupWith(t, func(req request) (int, string, bool) {
		return 200, envelope(req, "REJECTED", map[string]any{"reason": "STATE_CONFLICT"}), true
	}, &ctlRecorder{failNext: 1}, time.Second)
	if res, err := rc.ApplyLeadOutcome(context.Background(), lead()); !isUnknown(err) || res.Outcome == result.Conflict {
		t.Fatalf("an unrecorded REJECTED must stay UNKNOWN: %+v %v", res, err)
	}
}

func TestRecordTimeoutBounds(t *testing.T) {
	creds, _ := NewHMACCredentials("syn-key-01", clientKey, serverKey)
	for _, d := range []time.Duration{-time.Second, MaxRecordTimeout + time.Second} {
		if _, err := newClient(Config{Endpoint: "https://legacy.example.test/x", RecordTimeout: d, Credentials: creds, Recorder: &memRecorder{},
			Clock: fixedClock()}); !errors.Is(err, ErrConfig) {
			t.Fatalf("record timeout %s accepted", d)
		}
	}
	c, err := newClient(Config{Endpoint: "https://legacy.example.test/x", Credentials: creds, Recorder: &memRecorder{}, Clock: fixedClock()})
	if err != nil || c.recordTO != DefaultRecordTimeout {
		t.Fatalf("default record timeout: %v", err)
	}
}

func fixedClock() timeutil.FixedClock { return timeutil.FixedClock{At: mustInstant()} }
