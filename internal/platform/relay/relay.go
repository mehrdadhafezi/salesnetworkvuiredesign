// Package relay moves one module's outbox events to a consumer, never in a
// shared transaction.
//
// Each event is claimed in its own SOURCE transaction (FOR UPDATE SKIP
// LOCKED) and that transaction stays open while the consumer records the
// event in ITS own transaction (inbox-deduplicated by the consumer). The
// source then marks the event dispatched, or failed, in the same source
// transaction and commits. Consequences (R2):
//
//   - exclusive delivery: a concurrent relay skips a claimed row, so two
//     workers never deliver one event at the same time;
//   - the claim lease is the transaction: a crashed relay's lock disappears
//     with its connection and the event becomes claimable again;
//   - an event is marked dispatched only after the consumer committed;
//   - a crash after the consumer committed but before the source commit
//     redelivers the event, which the consumer's inbox turns into a no-op;
//   - a consumer failure marks the event failed; it is retried after a
//     backoff, at most MaxAttempts times, and then stays visible
//     (exhausted) — it is never discarded;
//   - an event type without a route FAILS (visible, bounded). Only types a
//     route explicitly declares as non-actionable, with a documented reason,
//     are marked dispatched without delivery.
package relay

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
)

// MaxAttempts bounds redelivery of one event.
const MaxAttempts = 10

// DefaultBackoff spaces redeliveries of a failed event (proposed
// engineering default, not an approved SLA).
var DefaultBackoff = outbox.Backoff{Base: 30 * time.Second, Max: 30 * time.Minute}

// Handler hands one event to a consumer. It must record the event in the
// consumer's own transaction (inbox) and return nil only after that commit.
type Handler func(ctx context.Context, rec outbox.Record) error

// ErrNoRoute is recorded on an event whose type the route does not know.
var ErrNoRoute = errors.New("relay: no consumer route for event type")

// ErrInvalidRoute reports a malformed route.
var ErrInvalidRoute = errors.New("relay: invalid route")

var reasonPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ,;:()./_-]{9,299}$`)

// Route is the complete delivery contract of one source outbox: every event
// type the source emits is either delivered (Handlers) or explicitly
// declared non-actionable (Ignore, type → documented reason). Anything else
// fails visibly.
type Route struct {
	Name     string
	Handlers map[string]Handler
	Ignore   map[string]string
	// Backoff overrides DefaultBackoff when set.
	Backoff *outbox.Backoff
	// DeliveryTimeout bounds one consumer call (0: the caller's context).
	DeliveryTimeout time.Duration
}

// Validate checks the route.
func (r Route) Validate() error {
	if r.Name == "" || len(r.Handlers)+len(r.Ignore) == 0 {
		return fmt.Errorf("%w: name and at least one type are required", ErrInvalidRoute)
	}
	for t, h := range r.Handlers {
		if h == nil {
			return fmt.Errorf("%w: nil handler for %s", ErrInvalidRoute, t)
		}
		if _, dup := r.Ignore[t]; dup {
			return fmt.Errorf("%w: %s is both handled and ignored", ErrInvalidRoute, t)
		}
	}
	for t, reason := range r.Ignore {
		if !reasonPattern.MatchString(reason) {
			return fmt.Errorf("%w: ignored type %s needs a documented reason", ErrInvalidRoute, t)
		}
	}
	if r.DeliveryTimeout < 0 || r.Backoff != nil && (r.Backoff.Base < 0 || r.Backoff.Max < r.Backoff.Base) {
		return fmt.Errorf("%w: timeouts", ErrInvalidRoute)
	}
	return nil
}

// Report counts what one Drain did.
type Report struct {
	Dispatched, Ignored, Failed, NoRoute int
}

// Total is the number of events Drain touched.
func (r Report) Total() int { return r.Dispatched + r.Ignored + r.Failed + r.NoRoute }

// Drain relays up to limit events of source through route.
func Drain(ctx context.Context, source *db.Runner, clock timeutil.Clock, limit int, route Route) (Report, error) {
	var rep Report
	if source == nil || clock == nil {
		return rep, errors.New("relay: source and clock are required")
	}
	if err := route.Validate(); err != nil {
		return rep, err
	}
	if limit < 1 || limit > 1000 {
		return rep, fmt.Errorf("%w: limit", ErrInvalidRoute)
	}
	backoff := DefaultBackoff
	if route.Backoff != nil {
		backoff = *route.Backoff
	}
	// seen keeps one pass from re-attempting an event it already handled
	// (with a zero backoff a failing event would otherwise burn its whole
	// attempt budget in one pass).
	seen := make([]string, 0, limit)
	for rep.Total() < limit {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		done := false
		err := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
			rec, ok, err := outbox.ClaimNext(ctx, tx, MaxAttempts, clock.Now(), backoff, seen)
			if err != nil {
				return err
			}
			if !ok {
				done = true
				return nil
			}
			seen = append(seen, rec.ID)
			if _, ignored := route.Ignore[rec.Type]; ignored {
				rep.Ignored++
				return outbox.MarkDispatched(ctx, tx, rec.ID, clock.Now())
			}
			h, known := route.Handlers[rec.Type]
			if !known {
				rep.NoRoute++
				return outbox.MarkFailed(ctx, tx, rec.ID, clock.Now(), fmt.Sprintf("%v: %s (route %s)", ErrNoRoute, rec.Type, route.Name))
			}
			dctx, cancel := ctx, context.CancelFunc(func() {})
			if route.DeliveryTimeout > 0 {
				dctx, cancel = context.WithTimeout(ctx, route.DeliveryTimeout)
			}
			derr := h(dctx, rec)
			cancel()
			if ctx.Err() != nil {
				// Shutdown during delivery: roll the claim back; the consumer's
				// inbox makes the redelivery safe whatever it committed.
				return ctx.Err()
			}
			if derr != nil {
				rep.Failed++
				return outbox.MarkFailed(ctx, tx, rec.ID, clock.Now(), derr.Error())
			}
			rep.Dispatched++
			return outbox.MarkDispatched(ctx, tx, rec.ID, clock.Now())
		})
		if err != nil {
			return rep, err
		}
		if done {
			return rep, nil
		}
	}
	return rep, nil
}
