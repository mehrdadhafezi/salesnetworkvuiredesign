package providerops

import (
	"context"
	"errors"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

// DispatchLease is how long a DISPATCHING operation may wait for its
// dispatcher to record the provider's answer. Past it, the dispatcher is
// presumed crashed: the outcome is UNKNOWN (never a retry, never success).
const DispatchLease = 15 * time.Minute

// ExpireStaleDispatches moves DISPATCHING operations older than lease to
// UNKNOWN (R2 crash recovery). UNKNOWN is never dispatched again; only a
// provider lookup or recorded human evidence resolves it.
func (s *Service) ExpireStaleDispatches(ctx context.Context, lease time.Duration, limit int) (int, error) {
	if lease < time.Minute || limit < 1 || limit > 500 {
		return 0, invalid("stale dispatch bounds")
	}
	n := 0
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		rows, err := tx.Query(ctx, `SELECT `+cols+` FROM integration.provider_operation WHERE state = 'DISPATCHING' AND updated_at < $1
			ORDER BY updated_at, operation_id LIMIT $2 FOR UPDATE SKIP LOCKED`, at.Add(-lease), limit)
		if err != nil {
			return err
		}
		var stale []Operation
		for rows.Next() {
			o, err := scan(rows)
			if err != nil {
				rows.Close()
				return err
			}
			stale = append(stale, o)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, o := range stale {
			if err := s.transition(ctx, tx, o, StateUnknown, "OUTCOME_UNKNOWN", nil,
				map[string]string{"reason": "dispatcher recorded no provider answer within the dispatch lease (presumed crash)"},
				"service:integration", at, false); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}

// Backlog is the provider-operation ledger backlog.
type Backlog struct {
	Unknown, Reconciliation, Dispatching, Prepared int
	ConflictCallbacks, UnmatchedCallbacks          int
	OldestUnresolved                               time.Duration
}

// ReadBacklog reads the ledger backlog.
func (s *Service) ReadBacklog(ctx context.Context) (Backlog, error) {
	var b Backlog
	var oldest *time.Time
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE state = 'UNKNOWN'), count(*) FILTER (WHERE state = 'RECONCILIATION_REQUIRED'),
				count(*) FILTER (WHERE state = 'DISPATCHING'), count(*) FILTER (WHERE state = 'PREPARED'),
				min(updated_at) FILTER (WHERE state IN ('UNKNOWN', 'RECONCILIATION_REQUIRED', 'DISPATCHING'))
			FROM integration.provider_operation`).Scan(&b.Unknown, &b.Reconciliation, &b.Dispatching, &b.Prepared, &oldest); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE disposition = 'CONFLICT'), count(*) FILTER (WHERE disposition = 'UNMATCHED')
			FROM integration.provider_callback`).Scan(&b.ConflictCallbacks, &b.UnmatchedCallbacks)
	})
	if err == nil && oldest != nil {
		if d := s.clock.Now().Time().Sub(*oldest); d > 0 {
			b.OldestUnresolved = d
		}
	}
	return b, err
}

func (b Backlog) report(r *worker.Run) {
	r.Gauge("provider_unknown", float64(b.Unknown))
	r.Gauge("provider_reconciliation_required", float64(b.Reconciliation))
	r.Gauge("provider_dispatching", float64(b.Dispatching))
	r.Gauge("provider_prepared", float64(b.Prepared))
	r.Gauge("provider_conflicting_callbacks", float64(b.ConflictCallbacks))
	r.Gauge("provider_unmatched_callbacks", float64(b.UnmatchedCallbacks))
	r.Gauge("provider_oldest_unresolved_age_seconds", b.OldestUnresolved.Seconds())
}

// unknownOldest lists UNKNOWN operations oldest first.
func (s *Service) unknownOldest(ctx context.Context, limit int) ([]string, error) {
	var out []string
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT operation_id::text FROM integration.provider_operation WHERE state = 'UNKNOWN'
			ORDER BY updated_at, operation_id LIMIT $1`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}

// ReconcileJob is one bounded reconciliation pass (R2): stale DISPATCHING
// operations become UNKNOWN, then up to batch UNKNOWN operations are looked
// up through port. It NEVER dispatches: a PREPARED or CONFIRMED_ABSENT
// operation is sent only by an explicit, authorised command. A port that is
// not enabled is refused here as well as by the caller's gate, because a
// disabled lookup would escalate every UNKNOWN to RECONCILIATION_REQUIRED
// without having asked the provider.
func (s *Service) ReconcileJob(name string, port Port, batch int) (worker.Job, error) {
	if port == nil || !port.Enabled() || batch < 1 || batch > 200 {
		return nil, ErrPortDisabled
	}
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		stale, err := s.ExpireStaleDispatches(ctx, DispatchLease, batch)
		r.Count("provider_stale_dispatch_unknown", stale)
		if err != nil {
			return err
		}
		ids, err := s.unknownOldest(ctx, batch)
		if err != nil {
			return err
		}
		for _, id := range ids {
			o, err := s.Reconcile(ctx, port, id)
			if errors.Is(err, ErrState) {
				continue // another instance (or a callback) moved it
			}
			if err != nil {
				return err
			}
			switch o.State {
			case StateConfirmed:
				r.Count("provider_lookup_confirmed", 1)
			case StateAbsent:
				r.Count("provider_lookup_absent", 1)
			case StateReconciliation:
				r.Count("provider_lookup_unavailable", 1)
			}
		}
		b, err := s.ReadBacklog(ctx)
		if err != nil {
			return err
		}
		b.report(r)
		return nil
	}}, nil
}

// BacklogJob reports the ledger backlog and recovers stale dispatches
// (both are local, provider-free): it runs while reconciliation is blocked
// for lack of a verified adapter.
func (s *Service) BacklogJob(name string, batch int) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		stale, err := s.ExpireStaleDispatches(ctx, DispatchLease, batch)
		r.Count("provider_stale_dispatch_unknown", stale)
		if err != nil {
			return err
		}
		b, err := s.ReadBacklog(ctx)
		if err != nil {
			return err
		}
		b.report(r)
		return nil
	}}
}
