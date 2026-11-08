package writeback

import (
	"context"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

// Stats is the write-back backlog (technical delivery state).
type Stats struct {
	Pending, Waiting, Unknown, Reconciliation, Rejected int
	// OpenReconciliations are reconciliation records of commands not yet
	// APPLIED (evidence awaiting a domain owner).
	OpenReconciliations int
	// ParityConflicts are commands whose parity is MISMATCH or CONFLICT.
	ParityConflicts int
	// OldestPending is the age of the oldest PENDING / WAITING / UNKNOWN command.
	OldestPending time.Duration
}

// ReadStats reads the write-back backlog.
func (w *Worker) ReadStats(ctx context.Context) (Stats, error) {
	var s Stats
	var oldest *time.Time
	err := w.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT
				count(*) FILTER (WHERE state = 'PENDING'),
				count(*) FILTER (WHERE state = 'WAITING_DEPENDENCY'),
				count(*) FILTER (WHERE state = 'UNKNOWN'),
				count(*) FILTER (WHERE state = 'RECONCILIATION_REQUIRED'),
				count(*) FILTER (WHERE state = 'REJECTED'),
				count(*) FILTER (WHERE parity IN ('MISMATCH', 'CONFLICT')),
				min(received_at) FILTER (WHERE state IN ('PENDING', 'WAITING_DEPENDENCY', 'UNKNOWN'))
			FROM integration.writeback_command`).Scan(&s.Pending, &s.Waiting, &s.Unknown, &s.Reconciliation, &s.Rejected,
			&s.ParityConflicts, &oldest); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT count(*) FROM integration.writeback_reconciliation r JOIN integration.writeback_command c USING (command_id)
			WHERE c.state <> 'APPLIED' OR r.reason = 'PARITY_MISMATCH'`).Scan(&s.OpenReconciliations)
	})
	if err == nil && oldest != nil {
		if d := w.clock.Now().Time().Sub(*oldest); d > 0 {
			s.OldestPending = d
		}
	}
	return s, err
}

// Report reports s as job gauges.
func (s Stats) Report(r *worker.Run) {
	r.Gauge("writeback_pending", float64(s.Pending))
	r.Gauge("writeback_waiting_dependency", float64(s.Waiting))
	r.Gauge("writeback_unknown", float64(s.Unknown))
	r.Gauge("writeback_reconciliation_required", float64(s.Reconciliation))
	r.Gauge("writeback_rejected", float64(s.Rejected))
	r.Gauge("writeback_open_reconciliations", float64(s.OpenReconciliations))
	r.Gauge("writeback_parity_conflicts", float64(s.ParityConflicts))
	r.Gauge("writeback_oldest_pending_age_seconds", s.OldestPending.Seconds())
}

// DeliverJob is one bounded delivery pass (Step over at most batch commands)
// followed by the backlog report. The worker must be bounded (WithOptions)
// and its connector must be an approved or synthetic adapter: the caller's
// gate decides whether the pass may run at all.
func (w *Worker) DeliverJob(name string, batch int) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		rep, err := w.Step(ctx, batch)
		r.Count("writeback_applied", rep.Applied)
		r.Count("writeback_outcome_unknown", rep.Unknown)
		r.Count("writeback_rejected", rep.Rejected)
		r.Count("writeback_reconciliation_opened", rep.Reconciliation)
		r.Count("writeback_compared", rep.Compared)
		r.Count("writeback_retried_after_absence", rep.Retried)
		if err != nil {
			return err
		}
		s, err := w.ReadStats(ctx)
		if err != nil {
			return err
		}
		s.Report(r)
		return nil
	}}
}

// BacklogJob reports the backlog only (no legacy call): it runs while the
// delivery job is blocked, so a stopped delivery is visible, never silent.
func (w *Worker) BacklogJob(name string) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		s, err := w.ReadStats(ctx)
		if err != nil {
			return err
		}
		s.Report(r)
		return nil
	}}
}
