package relay_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
)

// R2 outbox / inbox reliability proofs over the synthetic producer
// (syn_alpha) and consumer (syn_beta) modules.

type clock struct {
	mu sync.Mutex
	at time.Time
}

func (c *clock) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *clock) add(d time.Duration) { c.mu.Lock(); c.at = c.at.Add(d); c.mu.Unlock() }

const widgetType = "syn_alpha.widget_created"

type env struct {
	*pgtest.Env
	alpha, beta *db.Runner
	clock       *clock
	calls       atomic.Int64
}

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{Env: pgtest.New(t), clock: &clock{at: time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)}}
	e.alpha, e.beta = e.Runner(t, pgtest.Alpha), e.Runner(t, pgtest.Beta)
	return e
}

func bg(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return c
}

// produce commits a widget and its event atomically (rollback: neither).
func (e *env) produce(t *testing.T, typ string, fail bool) string {
	t.Helper()
	id, _ := outbox.NewEventID()
	payload, _ := json.Marshal(map[string]string{"label": "synthetic"})
	boom := errors.New("command failed after append")
	err := e.alpha.InTx(bg(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if _, err := tx.Exec(c, "INSERT INTO widget (label) VALUES ('synthetic')"); err != nil {
			return err
		}
		if err := outbox.Append(c, tx, outbox.Event{ID: id, Type: typ, Version: 1, Payload: payload, OccurredAt: e.clock.Now()}, e.clock.Now()); err != nil {
			return err
		}
		if fail {
			return boom
		}
		return nil
	})
	if fail != errors.Is(err, boom) || !fail && err != nil {
		t.Fatalf("produce: %v", err)
	}
	return id
}

// consume is the consumer: a NON-idempotent effect (seen = seen + 1)
// guarded only by the inbox, in the consumer's own transaction.
func (e *env) consume(ctx context.Context, rec outbox.Record) error {
	e.calls.Add(1)
	return e.beta.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(c context.Context, tx db.Tx) error {
		_, err := inbox.Consume(c, tx, "syn_beta.widgets", rec.ID, e.clock.Now(), func(c context.Context) error {
			_, err := tx.Exec(c, `INSERT INTO widget_seen (event_id, label) VALUES ($1, 'synthetic')
				ON CONFLICT (event_id) DO UPDATE SET seen = widget_seen.seen + 1`, rec.ID)
			return err
		})
		return err
	})
}

func (e *env) route(h relay.Handler) relay.Route {
	return relay.Route{Name: "syn_beta.widgets", Handlers: map[string]relay.Handler{widgetType: h}, Backoff: &outbox.Backoff{}}
}

func (e *env) q(t *testing.T, r *db.Runner, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := r.InTx(bg(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, sql, args...).Scan(&s)
	}); err != nil {
		t.Fatal(err)
	}
	return s
}

func (e *env) state(t *testing.T, id string) string {
	return e.q(t, e.alpha, `SELECT dispatch_state || ':' || attempts FROM outbox_event WHERE event_id = $1`, id)
}

func (e *env) seen(t *testing.T, id string) string {
	return e.q(t, e.beta, `SELECT coalesce((SELECT seen::text FROM widget_seen WHERE event_id = $1), 'none')`, id)
}

// 1 + 3 + 5: atomic append; dispatched only after the consumer committed;
// a duplicate delivery is a no-op.
func TestAtomicAppendAndDispatchAfterAcceptance(t *testing.T) {
	e := setup(t)
	ok := e.produce(t, widgetType, false)
	e.produce(t, widgetType, true) // rolled back with its aggregate
	if n := e.q(t, e.alpha, `SELECT count(*)::text || '/' || (SELECT count(*) FROM widget)::text FROM outbox_event`); n != "1/1" {
		t.Fatalf("events/widgets after a rollback: %s", n)
	}
	// The consumer refuses: the event is NOT dispatched.
	refuse := e.route(func(context.Context, outbox.Record) error { return errors.New("consumer unavailable") })
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, refuse); err != nil || rep.Failed != 1 || rep.Dispatched != 0 {
		t.Fatalf("refused: %+v %v", rep, err)
	}
	if st := e.state(t, ok); st != "failed:1" || e.seen(t, ok) != "none" {
		t.Fatalf("a refused event must stay undispatched: %s", st)
	}
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, e.route(e.consume)); err != nil || rep.Dispatched != 1 {
		t.Fatalf("deliver: %+v %v", rep, err)
	}
	if e.state(t, ok) != "dispatched:2" || e.seen(t, ok) != "1" {
		t.Fatalf("delivered: %s seen=%s", e.state(t, ok), e.seen(t, ok))
	}
	// 5: the same event delivered again (e.g. a replayed source): no effect.
	rec := outbox.Record{Event: outbox.Event{ID: ok, Type: widgetType}}
	if err := e.consume(bg(t), rec); err != nil || e.seen(t, ok) != "1" {
		t.Fatalf("duplicate delivery duplicated the effect: seen=%s err=%v", e.seen(t, ok), err)
	}
}

// 4: crash after the consumer committed, before the source acknowledged.
func TestCrashAfterConsumerCommitIsSafeOnReplay(t *testing.T) {
	e := setup(t)
	id := e.produce(t, widgetType, false)
	ctx, crash := context.WithCancel(bg(t))
	crashy := e.route(func(c context.Context, rec outbox.Record) error {
		if err := e.consume(context.WithoutCancel(c), rec); err != nil {
			return err
		}
		crash() // the relay process dies before marking the event dispatched
		return nil
	})
	if _, err := relay.Drain(ctx, e.alpha, e.clock, 10, crashy); !errors.Is(err, context.Canceled) {
		t.Fatalf("crash: %v", err)
	}
	if st := e.state(t, id); st != "pending:0" || e.seen(t, id) != "1" {
		t.Fatalf("after the crash the source must still hold the event: %s seen=%s", st, e.seen(t, id))
	}
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, e.route(e.consume)); err != nil || rep.Dispatched != 1 {
		t.Fatalf("replay: %+v %v", rep, err)
	}
	if e.state(t, id) != "dispatched:1" || e.seen(t, id) != "1" || e.calls.Load() != 2 {
		t.Fatalf("replay must be a no-op at the consumer: %s seen=%s calls=%d", e.state(t, id), e.seen(t, id), e.calls.Load())
	}
}

// 2 + 6: concurrent relays never deliver one event at the same time, and a
// non-idempotent effect is applied once.
func TestConcurrentRelaysDeliverEachEventOnce(t *testing.T) {
	e := setup(t)
	var ids []string
	for i := 0; i < 12; i++ {
		ids = append(ids, e.produce(t, widgetType, false))
	}
	var inFlight sync.Map
	var overlap atomic.Bool
	slow := e.route(func(c context.Context, rec outbox.Record) error {
		if _, loaded := inFlight.LoadOrStore(rec.ID, true); loaded {
			overlap.Store(true)
		}
		time.Sleep(20 * time.Millisecond)
		defer inFlight.Delete(rec.ID)
		return e.consume(c, rec)
	})
	var wg sync.WaitGroup
	var total atomic.Int64
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rep, err := relay.Drain(bg(t), e.alpha, e.clock, 100, slow)
			if err != nil {
				t.Error(err)
			}
			total.Add(int64(rep.Dispatched))
		}()
	}
	wg.Wait()
	if overlap.Load() || total.Load() != 12 || e.calls.Load() != 12 {
		t.Fatalf("overlap=%v dispatched=%d calls=%d", overlap.Load(), total.Load(), e.calls.Load())
	}
	for _, id := range ids {
		if e.seen(t, id) != "1" || e.state(t, id) != "dispatched:1" {
			t.Fatalf("%s: seen=%s state=%s", id, e.seen(t, id), e.state(t, id))
		}
	}
}

// 7 + 8 + 10: failures stay visible, attempts are bounded, an event type
// without a route fails instead of disappearing, and ignoring needs a
// documented reason.
func TestFailuresVisibleBoundedAndNoSilentSkip(t *testing.T) {
	e := setup(t)
	failing := e.produce(t, widgetType, false)
	unrouted := e.produce(t, "syn_alpha.widget_renamed", false)
	broken := e.route(func(context.Context, outbox.Record) error { return errors.New("consumer down") })
	for i := 0; i < relay.MaxAttempts+3; i++ {
		if _, err := relay.Drain(bg(t), e.alpha, e.clock, 10, broken); err != nil {
			t.Fatal(err)
		}
	}
	if st := e.state(t, failing); st != "failed:10" {
		t.Fatalf("attempts must stop at the bound: %s", st)
	}
	if st := e.state(t, unrouted); st != "failed:10" ||
		!strings.Contains(e.q(t, e.alpha, `SELECT last_error FROM outbox_event WHERE event_id = $1`, unrouted), "no consumer route") {
		t.Fatalf("an unrouted event must fail visibly: %s", st)
	}
	var stats outbox.Stats
	if err := e.alpha.InTx(bg(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		var err error
		stats, err = outbox.ReadStats(c, tx, relay.MaxAttempts, e.clock.Now())
		return err
	}); err != nil || stats.Exhausted != 2 || stats.Pending != 0 {
		t.Fatalf("stats: %+v %v", stats, err)
	}
	// Ignoring requires an explicit, documented reason.
	bad := relay.Route{Name: "x", Ignore: map[string]string{widgetType: ""}}
	if err := bad.Validate(); !errors.Is(err, relay.ErrInvalidRoute) {
		t.Fatal("an undocumented ignore must be refused")
	}
	both := relay.Route{Name: "x", Handlers: map[string]relay.Handler{widgetType: e.consume}, Ignore: map[string]string{widgetType: "documented reason text"}}
	if err := both.Validate(); !errors.Is(err, relay.ErrInvalidRoute) {
		t.Fatal("handled and ignored at once must be refused")
	}
	// A documented non-actionable type is dispatched without delivery.
	note := e.produce(t, "syn_alpha.widget_noted", false)
	ign := e.route(e.consume)
	ign.Ignore = map[string]string{"syn_alpha.widget_noted": "audit-only synthetic event, no consumer effect"}
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, ign); err != nil || rep.Ignored != 1 || e.state(t, note) != "dispatched:1" || e.seen(t, note) != "none" {
		t.Fatalf("ignored: %+v %v", rep, err)
	}
}

// Backoff: a failed event is not claimed again inside its window; one pass
// attempts an event at most once.
func TestBackoffAndOneAttemptPerPass(t *testing.T) {
	e := setup(t)
	id := e.produce(t, widgetType, false)
	r := e.route(func(context.Context, outbox.Record) error { return errors.New("down") })
	r.Backoff = &outbox.Backoff{Base: time.Minute, Max: 10 * time.Minute}
	if rep, _ := relay.Drain(bg(t), e.alpha, e.clock, 10, r); rep.Failed != 1 {
		t.Fatalf("first pass: %+v", rep)
	}
	if rep, _ := relay.Drain(bg(t), e.alpha, e.clock, 10, r); rep.Total() != 0 {
		t.Fatalf("inside the backoff window nothing is claimed: %+v", rep)
	}
	e.clock.add(61 * time.Second)
	if rep, _ := relay.Drain(bg(t), e.alpha, e.clock, 10, r); rep.Failed != 1 || e.state(t, id) != "failed:2" {
		t.Fatalf("after the window: %+v %s", rep, e.state(t, id))
	}
	e.clock.add(61 * time.Second) // second failure waits 2 minutes
	if rep, _ := relay.Drain(bg(t), e.alpha, e.clock, 10, r); rep.Total() != 0 {
		t.Fatalf("exponential backoff: %+v", rep)
	}
}

// 9: a claim held by a crashed relay is released with its connection.
func TestStaleClaimIsRecoverable(t *testing.T) {
	e := setup(t)
	id := e.produce(t, widgetType, false)
	holding, release := make(chan struct{}), make(chan struct{})
	ctx, crash := context.WithCancel(bg(t))
	done := make(chan error, 1)
	go func() {
		_, err := relay.Drain(ctx, e.alpha, e.clock, 10, e.route(func(c context.Context, rec outbox.Record) error {
			close(holding)
			<-release
			return errors.New("never reached")
		}))
		done <- err
	}()
	<-holding
	// While the first relay holds the claim, a second one skips the row.
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, e.route(e.consume)); err != nil || rep.Total() != 0 {
		t.Fatalf("a held claim must be skipped: %+v %v", rep, err)
	}
	crash()
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("crashed relay: %v", err)
	}
	if rep, err := relay.Drain(bg(t), e.alpha, e.clock, 10, e.route(e.consume)); err != nil || rep.Dispatched != 1 || e.state(t, id) != "dispatched:1" {
		t.Fatalf("the released claim must be recoverable: %+v %v", rep, err)
	}
}
