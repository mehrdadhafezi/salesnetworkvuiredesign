package command

import (
	"context"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

// EvidenceBacklog is the receipt / upload backlog (technical state only;
// it says nothing about whether a payment is approved or posted).
type EvidenceBacklog struct {
	Quarantined, Finalizing, StorageError, NotRecoverable, Exhausted int
	OpenReconciliations                                              int
	OldestUnfinalised                                                time.Duration
	// Uploads.
	UnboundExpired, Expiring, PurgeFailures int
	OldestUnboundExpired                    time.Duration
}

// ReadEvidenceBacklog reads the backlog in one read-only transaction.
func (s *Service) ReadEvidenceBacklog(ctx context.Context) (EvidenceBacklog, error) {
	var b EvidenceBacklog
	var oldest, oldestUpload *time.Time
	now := s.clock.Now().Time().UTC()
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE state = 'QUARANTINED'), count(*) FILTER (WHERE state = 'FINALIZING'),
				count(*) FILTER (WHERE state = 'STORAGE_ERROR'), count(*) FILTER (WHERE state = 'NOT_RECOVERABLE'),
				count(*) FILTER (WHERE state IN ('QUARANTINED', 'STORAGE_ERROR', 'FINALIZING') AND finalise_attempts >= $1),
				min(created_at) FILTER (WHERE state IN ('QUARANTINED', 'STORAGE_ERROR', 'FINALIZING'))
			FROM payments.receipt_evidence`, MaxFinaliseAttempts).Scan(&b.Quarantined, &b.Finalizing, &b.StorageError, &b.NotRecoverable,
			&b.Exhausted, &oldest); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM payments.evidence_reconciliation o WHERE o.kind = 'OPENED'
			AND NOT EXISTS (SELECT 1 FROM payments.evidence_reconciliation r WHERE r.receipt_id = o.receipt_id AND r.kind = 'RESOLVED')`).
			Scan(&b.OpenReconciliations); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE state = 'UNBOUND' AND expires_at <= $1), count(*) FILTER (WHERE state = 'EXPIRING'),
				count(*) FILTER (WHERE state = 'EXPIRING' AND purge_attempts > 0),
				min(expires_at) FILTER (WHERE state = 'UNBOUND' AND expires_at <= $1)
			FROM payments.upload_object`, now).Scan(&b.UnboundExpired, &b.Expiring, &b.PurgeFailures, &oldestUpload)
	})
	if err != nil {
		return b, err
	}
	if oldest != nil && now.After(*oldest) {
		b.OldestUnfinalised = now.Sub(*oldest)
	}
	if oldestUpload != nil && now.After(*oldestUpload) {
		b.OldestUnboundExpired = now.Sub(*oldestUpload)
	}
	return b, nil
}

func (b EvidenceBacklog) report(r *worker.Run) {
	r.Gauge("receipt_quarantined", float64(b.Quarantined))
	r.Gauge("receipt_finalizing", float64(b.Finalizing))
	r.Gauge("receipt_storage_error", float64(b.StorageError))
	r.Gauge("receipt_not_recoverable", float64(b.NotRecoverable))
	r.Gauge("receipt_retry_exhausted", float64(b.Exhausted))
	r.Gauge("receipt_open_reconciliations", float64(b.OpenReconciliations))
	r.Gauge("receipt_oldest_unfinalised_age_seconds", b.OldestUnfinalised.Seconds())
	r.Gauge("upload_unbound_expired", float64(b.UnboundExpired))
	r.Gauge("upload_expiring", float64(b.Expiring))
	r.Gauge("upload_purge_failures", float64(b.PurgeFailures))
	r.Gauge("upload_oldest_unbound_expired_age_seconds", b.OldestUnboundExpired.Seconds())
}

// FinaliseJob is one bounded receipt-finalisation pass (R2). Finalisation
// moves bytes from quarantine to the content-addressed evidence key and
// marks receipts READY; it never approves, rejects or posts a payment.
func (s *Service) FinaliseJob(name string, batch int) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		rep, err := s.Finalise(ctx, batch)
		r.Count("receipt_ready", rep.Ready)
		r.Count("receipt_storage_error", rep.StorageError)
		r.Count("receipt_not_recoverable", rep.NotRecoverable)
		r.Count("attempt_evidence_ready", rep.AttemptsReady)
		if err != nil {
			return err
		}
		b, err := s.ReadEvidenceBacklog(ctx)
		if err != nil {
			return err
		}
		b.report(r)
		return nil
	}}
}

// ExpiryReportJob reports unbound-upload expiry eligibility WITHOUT changing
// anything (R2 default while D-44 retention is not approved): the bytes and
// rows stay as they are and the blocked cleanup is visible in metrics.
func (s *Service) ExpiryReportJob(name string) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		b, err := s.ReadEvidenceBacklog(ctx)
		if err != nil {
			return err
		}
		b.report(r)
		return nil
	}}
}

// ExpiryPurgeJob runs ExpireUnbound (irreversible deletion of unbound
// quarantine bytes). Its gate must stay closed until the D-44 retention
// decision is approved; the worker composition opens it only for a
// disposable development environment.
func (s *Service) ExpiryPurgeJob(name string, batch int) worker.Job {
	return worker.JobFunc{JobName: name, Fn: func(ctx context.Context, r *worker.Run) error {
		rep, err := s.ExpireUnbound(ctx, batch)
		r.Count("upload_expiry_claimed", rep.Claimed)
		r.Count("upload_purged", rep.Purged)
		r.Count("upload_purge_failed", rep.DeleteFailed)
		if err != nil {
			return err
		}
		b, err := s.ReadEvidenceBacklog(ctx)
		if err != nil {
			return err
		}
		b.report(r)
		return nil
	}}
}
