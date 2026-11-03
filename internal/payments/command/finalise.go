package command

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
)

// ServiceActor is the technical actor of every finalisation / expiry step.
const ServiceActor = "service:payments.evidence"

// FinalisingLease is how long a FINALIZING claim is honoured before another
// run may take the receipt over (a crashed worker never strands evidence).
const FinalisingLease = 5 * time.Minute

// FinaliseReport counts what one Finalise run did.
type FinaliseReport struct {
	Ready, StorageError, NotRecoverable, AttemptsReady int
}

type receiptWork struct {
	id, attemptID, ref, state string
	sum                       []byte
	version                   int64
}

// Finalise finalises up to limit receipts (Architecture V1 §15.4–15.5):
// QUARANTINED, STORAGE_ERROR and lease-expired FINALIZING receipts are
// claimed (write-ahead FINALIZING), copied to the content-addressed key
// evidence/<sha256> if absent, verified and set READY. A storage error sets
// STORAGE_ERROR (the quarantine object is kept) and opens a reconciliation
// record; missing or mismatched bytes are NOT_RECOVERABLE with a
// reconciliation record for the Finance owner. The financial transaction
// (the attempt) is never rolled back here. A re-run finds the object,
// verifies its hash and only updates state: no duplicate file.
func (s *Service) Finalise(ctx context.Context, limit int) (FinaliseReport, error) {
	var rep FinaliseReport
	var work []receiptWork
	now := s.clock.Now().Time().UTC()
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT receipt_id::text, attempt_id::text, quarantine_ref, state, content_sha256, version FROM payments.receipt_evidence
			WHERE state IN ('QUARANTINED', 'STORAGE_ERROR') OR (state = 'FINALIZING' AND updated_at < $1)
			ORDER BY updated_at, receipt_id LIMIT $2`, now.Add(-FinalisingLease), limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var w receiptWork
			if err := rows.Scan(&w.id, &w.attemptID, &w.ref, &w.state, &w.sum, &w.version); err != nil {
				return err
			}
			work = append(work, w)
		}
		return rows.Err()
	})
	if err != nil {
		return rep, err
	}
	for _, w := range work {
		claimed, version, err := s.claimReceipt(ctx, w)
		if err != nil {
			return rep, err
		}
		if !claimed {
			continue
		}
		w.version = version
		var want [32]byte
		copy(want[:], w.sum)
		dst := "evidence/" + hexOf(w.sum)
		cerr := s.store.CopyIfAbsent(ctx, w.ref, dst, want)
		if cerr == nil {
			info, serr := s.store.Stat(ctx, dst)
			switch {
			case serr == nil && info.SHA256 != want:
				cerr = objectstore.ErrHashMismatch
			case serr != nil:
				cerr = serr
			}
		}
		switch {
		case cerr == nil:
			ready, err := s.markReady(ctx, w, dst)
			if err != nil {
				return rep, err
			}
			rep.Ready++
			if ready {
				rep.AttemptsReady++
			}
		case errors.Is(cerr, objectstore.ErrNotFound) || errors.Is(cerr, objectstore.ErrHashMismatch):
			if err := s.markNotRecoverable(ctx, w, cerr.Error()); err != nil {
				return rep, err
			}
			rep.NotRecoverable++
		default:
			if err := s.markStorageError(ctx, w, cerr.Error()); err != nil {
				return rep, err
			}
			rep.StorageError++
		}
	}
	return rep, nil
}

// claimReceipt durably moves a receipt to FINALIZING before any storage
// call (write-ahead), with a version check against concurrent workers.
func (s *Service) claimReceipt(ctx context.Context, w receiptWork) (bool, int64, error) {
	claimed := false
	var version int64
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		err := tx.QueryRow(ctx, `UPDATE payments.receipt_evidence SET state = 'FINALIZING', finalise_attempts = finalise_attempts + 1, updated_at = $3,
			version = version + 1 WHERE receipt_id = $1 AND version = $2 RETURNING version`, w.id, w.version, at).Scan(&version)
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		claimed = true
		return s.history(ctx, tx, w.id, w.state, api.ReceiptFinalizing, "finalisation claimed", at, version)
	})
	return claimed, version, err
}

func (s *Service) history(ctx context.Context, tx db.Tx, receiptID, from, to, reason string, at time.Time, version int64) error {
	_, err := tx.Exec(ctx, `INSERT INTO payments.receipt_state_history (history_id, receipt_id, from_state, to_state, reason, actor_principal,
		occurred_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ids.Stable("payments.receipt_history", receiptID, to, timeKey(at), itoa(version)), receiptID, from, to, truncate300(reason), ServiceActor, at)
	return err
}

func (s *Service) reconciliation(ctx context.Context, tx db.Tx, w receiptWork, kind, reason, action string, evidence map[string]any, at time.Time) error {
	ev, _ := json.Marshal(evidence)
	_, err := tx.Exec(ctx, `INSERT INTO payments.evidence_reconciliation (reconciliation_id, receipt_id, attempt_id, kind, reason, domain_owner,
		next_safe_action, evidence, opened_at) VALUES ($1, $2, $3, $4, $5, 'finance', $6, $7, $8) ON CONFLICT (reconciliation_id) DO NOTHING`,
		ids.Stable("payments.evidence_reconciliation", w.id, kind, reason), w.id, w.attemptID, kind, reason, action, ev, at)
	return err
}

// markReady sets the receipt READY; when every receipt of the attempt is
// READY it marks the attempt's evidence READY and emits the write-back
// event in the same transaction. It reports whether the attempt became ready.
func (s *Service) markReady(ctx context.Context, w receiptWork, dst string) (bool, error) {
	attemptReady := false
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		at := now.Time().UTC()
		var version int64
		err := tx.QueryRow(ctx, `UPDATE payments.receipt_evidence SET state = 'READY', evidence_ref = $3, updated_at = $4, version = version + 1
			WHERE receipt_id = $1 AND version = $2 AND state = 'FINALIZING' RETURNING version`, w.id, w.version, dst, at).Scan(&version)
		if err != nil {
			if isNoRows(err) {
				return nil // another worker took the receipt over; it will record its own outcome
			}
			return err
		}
		if err := s.history(ctx, tx, w.id, api.ReceiptFinalizing, api.ReceiptReady, "content-addressed copy verified", at, version); err != nil {
			return err
		}
		var hadError bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payments.evidence_reconciliation WHERE receipt_id = $1 AND reason = 'STORAGE_ERROR')`,
			w.id).Scan(&hadError); err != nil {
			return err
		}
		if hadError {
			if err := s.reconciliation(ctx, tx, w, "RESOLVED", "RECOVERED", "NO_AUTOMATIC_ACTION", map[string]any{"evidence_ref": dst}, at); err != nil {
				return err
			}
		}
		var a struct {
			batchID, itemKey, invoiceID, system, spec, key, unit, actor, idemKey, corr, evidence string
			stageNo                                                                              int
			amount, version                                                                      int64
		}
		if err := tx.QueryRow(ctx, `SELECT t.batch_id::text, t.item_key, t.invoice_id::text, b.invoice_source_system, b.invoice_source_spec,
			b.invoice_source_key, t.unit, t.submitted_by_user_id::text, b.idempotency_key, b.correlation_id, t.evidence_state, t.stage_no, t.amount, t.version
			FROM payments.submitted_attempt t JOIN payments.payment_batch b USING (batch_id) WHERE t.attempt_id = $1 FOR UPDATE OF t`, w.attemptID).
			Scan(&a.batchID, &a.itemKey, &a.invoiceID, &a.system, &a.spec, &a.key, &a.unit, &a.actor, &a.idemKey, &a.corr, &a.evidence, &a.stageNo,
				&a.amount, &a.version); err != nil {
			return err
		}
		if a.evidence != api.EvidencePendingFinalisation {
			return nil
		}
		var notReady int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM payments.receipt_evidence WHERE attempt_id = $1 AND state <> 'READY'`, w.attemptID).Scan(&notReady); err != nil {
			return err
		}
		if notReady > 0 {
			return nil
		}
		refs := []api.EvidenceRef{}
		rows, err := tx.Query(ctx, `SELECT receipt_id::text, evidence_ref, content_sha256, media_type FROM payments.receipt_evidence WHERE attempt_id = $1
			ORDER BY receipt_id`, w.attemptID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var r api.EvidenceRef
			var sum []byte
			if err := rows.Scan(&r.ReceiptID, &r.EvidenceRef, &sum, &r.MediaType); err != nil {
				rows.Close()
				return err
			}
			r.ContentSHA256 = hexOf(sum)
			refs = append(refs, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE payments.submitted_attempt SET evidence_state = 'READY', updated_at = $2, version = version + 1
			WHERE attempt_id = $1`, w.attemptID, at); err != nil {
			return err
		}
		detail, _ := json.Marshal(map[string]any{"receipt_count": len(refs)})
		if _, err := tx.Exec(ctx, `INSERT INTO payments.submitted_attempt_event (event_id, attempt_id, kind, detail, actor_principal, correlation_id,
			occurred_at) VALUES ($1, $2, 'EVIDENCE_READY', $3, $4, $5, $6)`,
			ids.Stable("payments.attempt_event", w.attemptID, "EVIDENCE_READY"), w.attemptID, detail, ServiceActor, a.corr, at); err != nil {
			return err
		}
		payload, err := json.Marshal(api.AttemptEvidenceReady{CommandID: w.attemptID, IdempotencyKey: a.idemKey + "." + a.itemKey,
			CorrelationID: a.corr, AttemptID: w.attemptID, BatchID: a.batchID, ItemKey: a.itemKey, InvoiceID: a.invoiceID, InvoiceSystem: a.system,
			InvoiceSpec: a.spec, InvoiceSourceKey: a.key, StageNo: a.stageNo, Amount: a.amount, Unit: a.unit, ActorUserID: a.actor, Receipts: refs,
			RolloutMode: "CONTROLLED_WRITEBACK", OccurredAt: at})
		if err != nil {
			return err
		}
		corr := a.corr
		if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("payments.outbox", w.attemptID, "evidence_ready"),
			Type: api.EventAttemptEvidenceReady, Version: 1, Payload: payload, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
			return err
		}
		attemptReady = true
		return nil
	})
	return attemptReady, err
}

func (s *Service) markStorageError(ctx context.Context, w receiptWork, detail string) error {
	return s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		var version int64
		err := tx.QueryRow(ctx, `UPDATE payments.receipt_evidence SET state = 'STORAGE_ERROR', updated_at = $3, version = version + 1
			WHERE receipt_id = $1 AND version = $2 AND state = 'FINALIZING' RETURNING version`, w.id, w.version, at).Scan(&version)
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		if err := s.history(ctx, tx, w.id, api.ReceiptFinalizing, api.ReceiptStorageError, "storage error: "+detail, at, version); err != nil {
			return err
		}
		return s.reconciliation(ctx, tx, w, "OPENED", "STORAGE_ERROR", "RETRY_FROM_QUARANTINE", map[string]any{"quarantine_ref": w.ref}, at)
	})
}

func (s *Service) markNotRecoverable(ctx context.Context, w receiptWork, detail string) error {
	return s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		var version int64
		err := tx.QueryRow(ctx, `UPDATE payments.receipt_evidence SET state = 'NOT_RECOVERABLE', updated_at = $3, version = version + 1
			WHERE receipt_id = $1 AND version = $2 AND state = 'FINALIZING' RETURNING version`, w.id, w.version, at).Scan(&version)
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		if err := s.history(ctx, tx, w.id, api.ReceiptFinalizing, api.ReceiptNotRecoverable, "bytes missing or hash mismatch: "+detail, at, version); err != nil {
			return err
		}
		if err := s.reconciliation(ctx, tx, w, "OPENED", "NOT_RECOVERABLE", "DOMAIN_OWNER_REVIEW",
			map[string]any{"quarantine_ref": w.ref, "content_sha256": hexOf(w.sum)}, at); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE payments.submitted_attempt SET evidence_state = 'NOT_RECOVERABLE', updated_at = $2, version = version + 1
			WHERE attempt_id = $1 AND evidence_state = 'PENDING_FINALISATION'`, w.attemptID, at)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		detailJSON, _ := json.Marshal(map[string]any{"receipt_id": w.id})
		_, err = tx.Exec(ctx, `INSERT INTO payments.submitted_attempt_event (event_id, attempt_id, kind, detail, actor_principal, correlation_id,
			occurred_at) VALUES ($1, $2, 'EVIDENCE_NOT_RECOVERABLE', $3, $4, 'evidence-finaliser', $5)`,
			ids.Stable("payments.attempt_event", w.attemptID, "EVIDENCE_NOT_RECOVERABLE"), w.attemptID, detailJSON, ServiceActor, at)
		return err
	})
}

// ExpireReport counts what one ExpireUnbound run did.
type ExpireReport struct{ Purged, DeleteFailed int }

// ExpireUnbound purges unbound uploads past expires_at (D-44 safe default).
// The row is kept (state PURGED); only the quarantine bytes are deleted, and
// only after a conditional database update proves the upload was never
// bound. A bound upload is never purged.
func (s *Service) ExpireUnbound(ctx context.Context, limit int) (ExpireReport, error) {
	var rep ExpireReport
	var refs []string
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		rows, err := tx.Query(ctx, `UPDATE payments.upload_object SET state = 'PURGED', purged_at = $1, version = version + 1
			WHERE upload_id IN (SELECT upload_id FROM payments.upload_object WHERE state = 'UNBOUND' AND expires_at <= $1
			ORDER BY expires_at LIMIT $2 FOR UPDATE SKIP LOCKED) AND state = 'UNBOUND' RETURNING quarantine_ref`, at, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r string
			if err := rows.Scan(&r); err != nil {
				return err
			}
			refs = append(refs, r)
		}
		return rows.Err()
	})
	if err != nil {
		return rep, err
	}
	for _, r := range refs {
		if err := s.store.Delete(ctx, r); err != nil {
			rep.DeleteFailed++
			continue
		}
		rep.Purged++
	}
	return rep, nil
}
