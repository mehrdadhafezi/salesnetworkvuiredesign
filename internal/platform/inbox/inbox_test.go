package inbox_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

var at, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")

const consumer = "syn_beta.test_consumer"

func msgID(t *testing.T) string {
	t.Helper()
	id, err := ids.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func consumeOnce(t *testing.T, beta *db.Runner, id string, handle func(context.Context) error) (inbox.Outcome, error) {
	var out inbox.Outcome
	err := beta.InTx(context.Background(), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		var err error
		out, err = inbox.Consume(c, tx, consumer, id, at, handle)
		return err
	})
	return out, err
}

func TestDuplicateDeliveryRunsHandlerOnce(t *testing.T) {
	env := pgtest.New(t)
	beta := env.Runner(t, pgtest.Beta)
	id := msgID(t)
	var runs atomic.Int32
	handle := func(context.Context) error { runs.Add(1); return nil }
	for i, want := range []inbox.Outcome{inbox.Processed, inbox.Duplicate, inbox.Duplicate} {
		got, err := consumeOnce(t, beta, id, handle)
		if err != nil || got != want {
			t.Fatalf("delivery %d = %v, %v; want %v", i, got, err, want)
		}
	}
	if runs.Load() != 1 {
		t.Fatalf("handler ran %d times", runs.Load())
	}
	if inbox.Processed.String() != "processed" || inbox.Duplicate.String() != "duplicate" || inbox.Outcome(0).String() != "invalid" {
		t.Fatal("outcome names")
	}
	// Another consumer of the same message is independent.
	err := beta.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		got, err := inbox.Consume(c, tx, "syn_beta.other_consumer", id, at, handle)
		if got != inbox.Processed {
			t.Errorf("other consumer outcome = %v", got)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerFailureLeavesMessageDeliverable(t *testing.T) {
	env := pgtest.New(t)
	beta := env.Runner(t, pgtest.Beta)
	id := msgID(t)
	boom := errors.New("technical failure")
	if _, err := consumeOnce(t, beta, id, func(context.Context) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("handler error not returned: %v", err)
	}
	// The failed attempt is recorded in its own transaction.
	if err := beta.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return inbox.RecordFailure(c, tx, consumer, id, at, strings.Repeat("e", 3000))
	}); err != nil {
		t.Fatal(err)
	}
	status := lookup(t, beta, id)
	if status.State != "failed" || status.Attempts != 1 {
		t.Fatalf("after failure = %+v", status)
	}
	got, err := consumeOnce(t, beta, id, func(context.Context) error { return nil })
	if err != nil || got != inbox.Processed {
		t.Fatalf("retry = %v, %v", got, err)
	}
	status = lookup(t, beta, id)
	if status.State != "processed" || status.Attempts != 2 {
		t.Fatalf("after retry = %+v", status)
	}
	// RecordFailure never downgrades a processed message.
	if err := beta.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return inbox.RecordFailure(c, tx, consumer, id, at, "late")
	}); err != nil {
		t.Fatal(err)
	}
	if lookup(t, beta, id).State != "processed" {
		t.Fatal("processed message was downgraded")
	}
}

func lookup(t *testing.T, beta *db.Runner, id string) inbox.Status {
	t.Helper()
	var s inbox.Status
	err := beta.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		var ok bool
		var err error
		s, ok, err = inbox.Lookup(c, tx, consumer, id)
		if !ok && err == nil {
			t.Errorf("message %s not found", id)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConcurrentDuplicatesProcessOnce(t *testing.T) {
	env := pgtest.New(t)
	beta := env.Runner(t, pgtest.Beta)
	id := msgID(t)
	var runs, processed, duplicates atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			out, err := consumeOnce(t, beta, id, func(context.Context) error {
				runs.Add(1)
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("concurrent consume: %v", err)
				return
			}
			switch out {
			case inbox.Processed:
				processed.Add(1)
			case inbox.Duplicate:
				duplicates.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if runs.Load() != 1 || processed.Load() != 1 || duplicates.Load() != 7 {
		t.Fatalf("runs=%d processed=%d duplicates=%d", runs.Load(), processed.Load(), duplicates.Load())
	}
}

func TestValidationAndReadOnly(t *testing.T) {
	env := pgtest.New(t)
	beta := env.Runner(t, pgtest.Beta)
	noop := func(context.Context) error { return nil }
	err := beta.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if _, err := inbox.Consume(c, tx, "Bad Consumer", msgID(t), at, noop); !errors.Is(err, inbox.ErrInvalidMessage) {
			t.Errorf("bad consumer err = %v", err)
		}
		if _, err := inbox.Consume(c, tx, consumer, "not-a-uuid", at, noop); !errors.Is(err, inbox.ErrInvalidMessage) {
			t.Errorf("bad id err = %v", err)
		}
		if _, err := inbox.Consume(c, tx, consumer, msgID(t), timeutil.Instant{}, noop); !errors.Is(err, inbox.ErrInvalidMessage) {
			t.Errorf("zero time err = %v", err)
		}
		if err := inbox.RecordFailure(c, tx, "x", msgID(t), at, ""); !errors.Is(err, inbox.ErrInvalidMessage) {
			t.Errorf("bad failure record err = %v", err)
		}
		if _, _, err := inbox.Lookup(c, tx, consumer, "nope"); !errors.Is(err, inbox.ErrInvalidMessage) {
			t.Errorf("bad lookup err = %v", err)
		}
		if _, ok, err := inbox.Lookup(c, tx, consumer, msgID(t)); ok || err != nil {
			t.Errorf("missing lookup = %v, %v", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = beta.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		if _, err := inbox.Consume(c, tx, consumer, msgID(t), at, noop); !errors.Is(err, inbox.ErrReadOnly) {
			t.Errorf("read-only consume err = %v", err)
		}
		if err := inbox.RecordFailure(c, tx, consumer, msgID(t), at, ""); !errors.Is(err, inbox.ErrReadOnly) {
			t.Errorf("read-only failure err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
