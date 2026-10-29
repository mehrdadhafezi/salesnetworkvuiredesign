package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

var at, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")

func event(t *testing.T, label string) outbox.Event {
	t.Helper()
	id, err := outbox.NewEventID()
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"label": label})
	return outbox.Event{ID: id, Type: "syn_alpha.widget_created", Version: 1, Payload: payload, OccurredAt: at}
}

func TestEventValidation(t *testing.T) {
	good := event(t, "a")
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	subject := refs.NativeRef{Domain: refs.DomainEvent, ID: "7"}
	withSubject := good
	withSubject.Subject = &subject
	if err := withSubject.Validate(); err != nil {
		t.Fatal(err)
	}
	empty := ""
	long := strings.Repeat("c", 129)
	bad := []func(*outbox.Event){
		func(e *outbox.Event) { e.ID = "not-a-uuid" },
		func(e *outbox.Event) { e.Type = "NoDot" },
		func(e *outbox.Event) { e.Type = "single" },
		func(e *outbox.Event) { e.Version = 0 },
		func(e *outbox.Event) { e.Payload = nil },
		func(e *outbox.Event) { e.Payload = json.RawMessage(`[1,2]`) },
		func(e *outbox.Event) { e.Payload = json.RawMessage(`null`) },
		func(e *outbox.Event) {
			e.Payload = json.RawMessage(`{"x":` + strings.Repeat("1", outbox.MaxPayloadBytes) + `}`)
		},
		func(e *outbox.Event) { e.OccurredAt = timeutil.Instant{} },
		func(e *outbox.Event) { e.CorrelationID = &empty },
		func(e *outbox.Event) { e.CorrelationID = &long },
		func(e *outbox.Event) { e.Subject = &refs.NativeRef{Domain: "PHONE", ID: "1"} },
	}
	for i, m := range bad {
		e := good
		m(&e)
		if err := e.Validate(); !errors.Is(err, outbox.ErrInvalidEvent) {
			t.Fatalf("mutation %d accepted: %v", i, err)
		}
	}
}

func counts(t *testing.T, env *pgtest.Env) (widgets, events int) {
	t.Helper()
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, "SELECT (SELECT count(*) FROM widget), (SELECT count(*) FROM outbox_event)").Scan(&widgets, &events)
	})
	if err != nil {
		t.Fatal(err)
	}
	return widgets, events
}

func produce(c context.Context, tx db.Tx, e outbox.Event) error {
	if _, err := tx.Exec(c, "INSERT INTO widget (label) VALUES ($1)", "synthetic"); err != nil {
		return err
	}
	return outbox.Append(c, tx, e, at)
}

func TestAppendIsAtomicWithOwnerState(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	e := event(t, "first")
	if err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return produce(c, tx, e) }); err != nil {
		t.Fatal(err)
	}
	if w, ev := counts(t, env); w != 1 || ev != 1 {
		t.Fatalf("after commit widgets=%d events=%d", w, ev)
	}
	boom := errors.New("command failed after append")
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := produce(c, tx, event(t, "second")); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if w, ev := counts(t, env); w != 1 || ev != 1 {
		t.Fatalf("rollback left widgets=%d events=%d", w, ev)
	}
	// The same event identity cannot be appended twice.
	err = alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return outbox.Append(c, tx, e, at) })
	if err == nil {
		t.Fatal("duplicate event id must be rejected")
	}
	if err := alpha.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return outbox.Append(c, tx, event(t, "ro"), at)
	}); !errors.Is(err, outbox.ErrReadOnly) {
		t.Fatalf("read-only append err = %v", err)
	}
	if err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return outbox.Append(c, tx, event(t, "x"), timeutil.Instant{})
	}); !errors.Is(err, outbox.ErrInvalidEvent) {
		t.Fatalf("zero recorded_at err = %v", err)
	}
}

func TestAppendTargetsOnlyTheOwnersSchema(t *testing.T) {
	env := pgtest.New(t)
	// Beta's transaction appends to beta's outbox; alpha's outbox is untouched.
	if err := env.Runner(t, pgtest.Beta).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return outbox.Append(c, tx, outbox.Event{ID: event(t, "b").ID, Type: "syn_beta.noted", Version: 1, Payload: json.RawMessage(`{}`), OccurredAt: at}, at)
	}); err != nil {
		t.Fatal(err)
	}
	if _, ev := counts(t, env); ev != 0 {
		t.Fatalf("alpha outbox has %d rows; appends must stay in the owner's schema", ev)
	}
}

const consumer = "syn_beta.widget_projection"

// deliver runs one synthetic dispatch round: claim alpha's pending events in
// an alpha transaction, hand each to beta's consumer in beta's own
// transaction, then mark it dispatched in alpha's transaction.
func deliver(t *testing.T, env *pgtest.Env, handled *atomic.Int32) int {
	t.Helper()
	alpha, beta := env.Runner(t, pgtest.Alpha), env.Runner(t, pgtest.Beta)
	n := 0
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		recs, err := outbox.Claim(c, tx, 10, 5)
		if err != nil {
			return err
		}
		for _, r := range recs {
			var p struct{ Label string }
			if err := json.Unmarshal(r.Payload, &p); err != nil {
				return err
			}
			if err := consume(t, beta, r.ID, p.Label, handled); err != nil {
				return err
			}
			if err := outbox.MarkDispatched(c, tx, r.ID, at); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func consume(t *testing.T, beta *db.Runner, id, label string, handled *atomic.Int32) error {
	return beta.InTx(context.Background(), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		_, err := inbox.Consume(c, tx, consumer, id, at, func(c context.Context) error {
			handled.Add(1)
			_, err := tx.Exec(c, `INSERT INTO widget_seen (event_id, label) VALUES ($1, $2)
				ON CONFLICT (event_id) DO UPDATE SET seen = widget_seen.seen + 1`, id, label)
			return err
		})
		return err
	})
}

func TestDispatchEndToEndAndRedelivery(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	e := event(t, "delivered")
	if err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return produce(c, tx, e) }); err != nil {
		t.Fatal(err)
	}
	var handled atomic.Int32
	if n := deliver(t, env, &handled); n != 1 {
		t.Fatalf("delivered %d events, want 1", n)
	}
	if n := deliver(t, env, &handled); n != 0 {
		t.Fatalf("dispatched event claimed again (%d)", n)
	}
	// At-least-once: a redelivery of the same event must not run the handler again.
	if err := consume(t, env.Runner(t, pgtest.Beta), e.ID, "delivered", &handled); err != nil {
		t.Fatal(err)
	}
	if handled.Load() != 1 {
		t.Fatalf("handler ran %d times, want 1", handled.Load())
	}
	var seen int
	err := env.Runner(t, pgtest.Beta).InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, "SELECT seen FROM widget_seen WHERE event_id = $1", e.ID).Scan(&seen)
	})
	if err != nil || seen != 1 {
		t.Fatalf("projection seen=%d err=%v", seen, err)
	}
}

func TestClaimFailureAttemptsAndSkipLocked(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	var ids []string
	for i := 0; i < 4; i++ {
		e := event(t, "e")
		ids = append(ids, e.ID)
		if err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return produce(c, tx, e) }); err != nil {
			t.Fatal(err)
		}
	}
	// Two concurrent dispatchers never claim the same row.
	var mu sync.Mutex
	claimed := map[string]int{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = alpha.InTx(context.Background(), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
				<-start
				recs, err := outbox.Claim(c, tx, 2, 5)
				if err != nil {
					return err
				}
				mu.Lock()
				for _, r := range recs {
					claimed[r.ID]++
				}
				mu.Unlock()
				time.Sleep(200 * time.Millisecond) // hold the row locks
				return errors.New("release without marking")
			})
		}()
	}
	close(start)
	wg.Wait()
	for id, n := range claimed {
		if n != 1 {
			t.Fatalf("event %s claimed %d times concurrently", id, n)
		}
	}
	if len(claimed) != 4 {
		t.Fatalf("claimed %d distinct events, want 4", len(claimed))
	}
	// Failed attempts count up; at maxAttempts the event stops being claimable.
	for i := 0; i < 2; i++ {
		if err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
			return outbox.MarkFailed(c, tx, ids[0], at, strings.Repeat("x", 2000))
		}); err != nil {
			t.Fatal(err)
		}
	}
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		recs, err := outbox.Claim(c, tx, 10, 2)
		if err != nil {
			return err
		}
		for _, r := range recs {
			if r.ID == ids[0] {
				t.Errorf("event at maxAttempts was claimed (attempts=%d)", r.Attempts)
			}
		}
		if len(recs) != 3 {
			t.Errorf("claimed %d, want 3", len(recs))
		}
		if err := outbox.MarkDispatched(c, tx, ids[1], at); err != nil {
			return err
		}
		if err := outbox.MarkDispatched(c, tx, ids[1], at); !errors.Is(err, outbox.ErrNotFound) {
			t.Errorf("second MarkDispatched err = %v", err)
		}
		if err := outbox.MarkFailed(c, tx, ids[1], at, "late"); !errors.Is(err, outbox.ErrNotFound) {
			t.Errorf("MarkFailed after dispatch err = %v", err)
		}
		if _, err := outbox.Claim(c, tx, 0, 1); !errors.Is(err, outbox.ErrInvalidEvent) {
			t.Errorf("bad claim bounds err = %v", err)
		}
		if err := outbox.MarkDispatched(c, tx, "nope", at); !errors.Is(err, outbox.ErrInvalidEvent) {
			t.Errorf("bad id err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = alpha.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		if _, err := outbox.Claim(c, tx, 1, 1); !errors.Is(err, outbox.ErrReadOnly) {
			t.Errorf("read-only claim err = %v", err)
		}
		if err := outbox.MarkDispatched(c, tx, ids[2], at); !errors.Is(err, outbox.ErrReadOnly) {
			t.Errorf("read-only mark err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubjectRoundTrip(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	e := event(t, "with-subject")
	subject := refs.NativeRef{Domain: refs.DomainEvent, ID: "42"}
	corr := "corr-1"
	e.Subject, e.CorrelationID = &subject, &corr
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := outbox.Append(c, tx, e, at); err != nil {
			return err
		}
		recs, err := outbox.Claim(c, tx, 1, 1)
		if err != nil {
			return err
		}
		if len(recs) != 1 || recs[0].Subject == nil || !recs[0].Subject.Equal(subject) || *recs[0].CorrelationID != corr ||
			!recs[0].OccurredAt.Equal(at) || recs[0].Type != e.Type || recs[0].Version != 1 {
			t.Errorf("round trip = %+v", recs)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
