package worker

import (
	"context"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
)

// RelayJob relays one source module's outbox through route (one pass of at
// most batch events) and reports the source backlog. The source runner is
// owned by the source module: the relay never writes another schema.
func RelayJob(name string, source *db.Runner, clock timeutil.Clock, batch int, route relay.Route) (Job, error) {
	if source == nil || clock == nil || batch < 1 || batch > 1000 {
		return nil, fmt.Errorf("%w: relay job %s", ErrInvalid, name)
	}
	if err := route.Validate(); err != nil {
		return nil, err
	}
	return JobFunc{JobName: name, Fn: func(ctx context.Context, r *Run) error {
		rep, err := relay.Drain(ctx, source, clock, batch, route)
		r.Count("relay_dispatched", rep.Dispatched)
		r.Count("relay_ignored", rep.Ignored)
		r.Count("relay_delivery_failed", rep.Failed)
		r.Count("relay_no_route", rep.NoRoute)
		if err != nil {
			return err
		}
		return OutboxBacklog(ctx, source, clock, r)
	}}, nil
}

// OutboxBacklog reports the backlog gauges of source's outbox.
func OutboxBacklog(ctx context.Context, source *db.Runner, clock timeutil.Clock, r *Run) error {
	return source.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		s, err := outbox.ReadStats(ctx, tx, relay.MaxAttempts, clock.Now())
		if err != nil {
			return err
		}
		r.Gauge("outbox_pending", float64(s.Pending))
		r.Gauge("outbox_failed_retryable", float64(s.Failed))
		r.Gauge("outbox_exhausted", float64(s.Exhausted))
		r.Gauge("outbox_oldest_pending_age_seconds", s.OldestUndispatched.Seconds())
		return nil
	})
}
