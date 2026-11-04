// Package relay moves one module's outbox events to a consumer, never in a
// shared transaction: the source claims events in its own transaction, the
// consumer records each in ITS own transaction (inbox-deduplicated by the
// consumer), and the source then marks the event dispatched in another
// source transaction. A consumer failure marks the event failed (retried
// later, bounded); a crash between consume and mark redelivers the event,
// which the consumer's inbox turns into a no-op.
package relay

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
)

// MaxAttempts bounds redelivery of one event.
const MaxAttempts = 10

// Deliver hands one event to a consumer. ErrSkip marks an event the
// consumer deliberately does not handle (it is still marked dispatched).
type Deliver func(ctx context.Context, rec outbox.Record) error

// ErrSkip reports an event type the consumer ignores.
var ErrSkip = errors.New("relay: event not handled by this consumer")

// Drain relays up to limit events of source.
func Drain(ctx context.Context, source *db.Runner, clock timeutil.Clock, limit int, deliver Deliver) (int, error) {
	if source == nil || clock == nil || deliver == nil {
		return 0, errors.New("relay: source, clock and deliver are required")
	}
	var recs []outbox.Record
	if err := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		var err error
		recs, err = outbox.Claim(ctx, tx, limit, MaxAttempts)
		return err
	}); err != nil {
		return 0, err
	}
	n := 0
	for _, rec := range recs {
		if err := deliver(ctx, rec); err != nil && !errors.Is(err, ErrSkip) {
			reason := err.Error()
			if mErr := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
				return outbox.MarkFailed(ctx, tx, rec.ID, clock.Now(), reason)
			}); mErr != nil {
				return n, mErr
			}
			continue
		}
		if err := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
			return outbox.MarkDispatched(ctx, tx, rec.ID, clock.Now())
		}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
