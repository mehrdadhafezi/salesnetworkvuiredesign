package command

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

var _ api.ReviewSource = (*Service)(nil)

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// ReceiptView is one receipt as the seller and Finance see it.
type ReceiptView struct {
	ReceiptID string    `json:"receipt_id"`
	UploadID  string    `json:"upload_id"`
	State     string    `json:"state"`
	MediaType string    `json:"media_type"`
	SizeBytes int       `json:"size_bytes"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AttemptView is one target-submitted attempt.
type AttemptView struct {
	AttemptID        string        `json:"attempt_id"`
	BatchID          string        `json:"batch_id"`
	ItemKey          string        `json:"item_key"`
	Amount           int64         `json:"amount"`
	Unit             string        `json:"unit"`
	State            string        `json:"state"`
	ReviewProvenance string        `json:"review_provenance"`
	EvidenceState    string        `json:"evidence_state"`
	SubmittedAt      time.Time     `json:"submitted_at"`
	Receipts         []ReceiptView `json:"receipts"`
}

// SubmissionContext is the read model of the seller payment surface: the
// current stage evidence, the same eligibility evaluation the command runs,
// and the actor's target attempts for the stage.
type SubmissionContext struct {
	InvoiceID         string             `json:"invoice_id"`
	StageNo           *int               `json:"stage_no"`
	Unit              *string            `json:"unit"`
	Due               *int64             `json:"due_amount"`
	Remaining         *int64             `json:"remaining_amount"`
	Used              *int64             `json:"pending_or_accepted_amount"`
	Available         *int64             `json:"available_amount"`
	CollectionVersion int64              `json:"collection_version"`
	Eligible          bool               `json:"eligible"`
	Status            string             `json:"evaluation_status"`
	Reasons           []reasoncodes.Code `json:"reasons"`
	BlockedBy         []string           `json:"blocked_by"`
	Attempts          []AttemptView      `json:"attempts"`
}

// Context evaluates the submission context for actor. ok is false for an
// unknown invoice. It runs in a payments READ ONLY transaction.
func (s *Service) Context(ctx context.Context, invoiceID string, actor Actor) (SubmissionContext, bool, error) {
	if !uuidPattern.MatchString(invoiceID) {
		return SubmissionContext{}, false, nil
	}
	out := SubmissionContext{InvoiceID: invoiceID, Reasons: []reasoncodes.Code{}, BlockedBy: []string{}, Attempts: []AttemptView{}}
	found := false
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		e, ok, err := s.evaluate(ctx, tx, invoiceID, actor)
		if err != nil || !ok {
			return err
		}
		found = true
		out.Status, out.Eligible, out.Reasons = e.Status.String(), e.Status == result.Success, e.Reasons
		if e.BlockedBy != nil {
			out.BlockedBy = e.BlockedBy
		}
		if e.StageNo > 0 {
			no, unit, due, rem := e.StageNo, e.Unit, e.Due, e.Remaining
			out.StageNo, out.Unit, out.Due, out.Remaining = &no, &unit, &due, &rem
			if out.CollectionVersion, err = accountVersion(ctx, tx, invoiceID, no); err != nil {
				return err
			}
			if e.Known {
				used, avail := e.Used, e.Available
				out.Used, out.Available = &used, &avail
			}
			// Only the owning seller sees the stage's attempts.
			if e.Status == result.Success || e.Status == result.Blocked {
				if out.Attempts, err = attempts(ctx, tx, `t.invoice_id = $1 AND t.stage_no = $2 AND t.submitted_by_user_id = $3`, invoiceID, no, actor.UserID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return out, found, err
}

func attempts(ctx context.Context, tx db.Tx, where string, args ...any) ([]AttemptView, error) {
	rows, err := tx.Query(ctx, `SELECT t.attempt_id::text, t.batch_id::text, t.item_key, t.amount, t.unit, t.state, t.review_provenance,
		t.evidence_state, t.created_at FROM payments.submitted_attempt t WHERE `+where+` ORDER BY t.created_at, t.item_no`, args...)
	if err != nil {
		return nil, err
	}
	out := []AttemptView{}
	for rows.Next() {
		var a AttemptView
		if err := rows.Scan(&a.AttemptID, &a.BatchID, &a.ItemKey, &a.Amount, &a.Unit, &a.State, &a.ReviewProvenance, &a.EvidenceState, &a.SubmittedAt); err != nil {
			rows.Close()
			return nil, err
		}
		a.SubmittedAt = a.SubmittedAt.UTC()
		out = append(out, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Receipts, err = receipts(ctx, tx, out[i].AttemptID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func receipts(ctx context.Context, tx db.Tx, attemptID string) ([]ReceiptView, error) {
	rows, err := tx.Query(ctx, `SELECT receipt_id::text, upload_id::text, state, media_type, size_bytes, updated_at FROM payments.receipt_evidence
		WHERE attempt_id = $1 ORDER BY created_at, receipt_id`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReceiptView{}
	for rows.Next() {
		var r ReceiptView
		if err := rows.Scan(&r.ReceiptID, &r.UploadID, &r.State, &r.MediaType, &r.SizeBytes, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.UpdatedAt = r.UpdatedAt.UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

// BatchView is a committed batch with its attempts.
type BatchView struct {
	BatchID       string        `json:"batch_id"`
	InvoiceID     string        `json:"invoice_id"`
	StageNo       int           `json:"stage_no"`
	ItemCount     int           `json:"item_count"`
	TotalAmount   int64         `json:"total_amount"`
	Unit          string        `json:"unit"`
	BeforeVersion int64         `json:"collection_version_before"`
	AfterVersion  int64         `json:"collection_version_after"`
	CorrelationID string        `json:"correlation_id"`
	CreatedAt     time.Time     `json:"created_at"`
	Attempts      []AttemptView `json:"attempts"`
}

// Batch reads one committed batch of the actor (row scope: the submitter).
// Another user's batch reads as not found.
func (s *Service) Batch(ctx context.Context, batchID string, actor Actor) (BatchView, bool, error) {
	if !uuidPattern.MatchString(batchID) || actor.UserID == "" {
		return BatchView{}, false, nil
	}
	var b BatchView
	found := false
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		err := tx.QueryRow(ctx, `SELECT batch_id::text, invoice_id::text, stage_no, item_count, total_amount, unit, before_version, after_version,
			correlation_id, created_at FROM payments.payment_batch WHERE batch_id = $1 AND actor_user_id = $2`, batchID, actor.UserID).
			Scan(&b.BatchID, &b.InvoiceID, &b.StageNo, &b.ItemCount, &b.TotalAmount, &b.Unit, &b.BeforeVersion, &b.AfterVersion, &b.CorrelationID, &b.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		b.CreatedAt = b.CreatedAt.UTC()
		b.Attempts, err = attempts(ctx, tx, `t.batch_id = $1`, batchID)
		return err
	})
	return b, found, err
}

// ReviewQueue implements api.ReviewSource.
func (s *Service) ReviewQueue(ctx context.Context, after *string, limit int) ([]api.ReviewAttempt, error) {
	var out []api.ReviewAttempt
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var cursor any
		if after != nil {
			cursor = *after
		}
		rows, err := tx.Query(ctx, `SELECT attempt_id::text FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW'
			AND ($1::uuid IS NULL OR attempt_id > $1::uuid) ORDER BY attempt_id LIMIT $2`, cursor, limit+1)
		if err != nil {
			return err
		}
		var list []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			list = append(list, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, id := range list {
			a, _, err := reviewAttempt(ctx, tx, id)
			if err != nil {
				return err
			}
			out = append(out, a)
		}
		return nil
	})
	return out, err
}

// ReviewAttempt implements api.ReviewSource.
func (s *Service) ReviewAttempt(ctx context.Context, attemptID string) (api.ReviewAttempt, bool, error) {
	if !uuidPattern.MatchString(attemptID) {
		return api.ReviewAttempt{}, false, nil
	}
	var a api.ReviewAttempt
	ok := false
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		a, ok, err = reviewAttempt(ctx, tx, attemptID)
		return err
	})
	return a, ok, err
}

func reviewAttempt(ctx context.Context, tx db.Tx, id string) (api.ReviewAttempt, bool, error) {
	var a api.ReviewAttempt
	err := tx.QueryRow(ctx, `SELECT attempt_id::text, batch_id::text, invoice_id::text, stage_no, amount, unit, state, review_provenance,
		evidence_state, submitted_by_user_id::text, created_at FROM payments.submitted_attempt WHERE attempt_id = $1`, id).
		Scan(&a.AttemptID, &a.BatchID, &a.InvoiceID, &a.StageNo, &a.Amount, &a.Unit, &a.State, &a.ReviewProvenance, &a.EvidenceState,
			&a.SubmittedByUserID, &a.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, false, nil
	}
	if err != nil {
		return a, false, err
	}
	a.SubmittedAt = a.SubmittedAt.UTC()
	rs, err := receipts(ctx, tx, id)
	if err != nil {
		return a, false, err
	}
	for _, r := range rs {
		a.Receipts = append(a.Receipts, api.ReviewReceipt{ReceiptID: r.ReceiptID, UploadID: r.UploadID, State: r.State, MediaType: r.MediaType,
			SizeBytes: r.SizeBytes, UpdatedAt: r.UpdatedAt})
	}
	return a, true, nil
}

// ReconciliationView is one evidence reconciliation record.
type ReconciliationView struct {
	ReconciliationID string    `json:"reconciliation_id"`
	ReceiptID        string    `json:"receipt_id"`
	AttemptID        string    `json:"attempt_id"`
	Kind             string    `json:"kind"`
	Reason           string    `json:"reason"`
	DomainOwner      string    `json:"domain_owner"`
	NextSafeAction   string    `json:"next_safe_action"`
	ReceiptState     string    `json:"receipt_state"`
	OpenedAt         time.Time `json:"opened_at"`
}

// EvidenceSummary counts receipts by state and reconciliations by reason.
type EvidenceSummary struct {
	Receipts        map[string]int64     `json:"receipts_by_state"`
	Uploads         map[string]int64     `json:"uploads_by_state"`
	Reconciliations []ReconciliationView `json:"reconciliations"`
}

// Evidence reads evidence diagnostics: counts and the latest
// reconciliation records (no object references beyond ids, no bytes).
func (s *Service) Evidence(ctx context.Context, limit int) (EvidenceSummary, error) {
	out := EvidenceSummary{Receipts: map[string]int64{}, Uploads: map[string]int64{}, Reconciliations: []ReconciliationView{}}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		for sql, m := range map[string]map[string]int64{
			`SELECT state, count(*) FROM payments.receipt_evidence GROUP BY state`: out.Receipts,
			`SELECT state, count(*) FROM payments.upload_object GROUP BY state`:    out.Uploads,
		} {
			rows, err := tx.Query(ctx, sql)
			if err != nil {
				return err
			}
			for rows.Next() {
				var k string
				var n int64
				if err := rows.Scan(&k, &n); err != nil {
					rows.Close()
					return err
				}
				m[k] = n
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `SELECT r.reconciliation_id::text, r.receipt_id::text, r.attempt_id::text, r.kind, r.reason, r.domain_owner,
			r.next_safe_action, c.state, r.opened_at FROM payments.evidence_reconciliation r JOIN payments.receipt_evidence c USING (receipt_id)
			ORDER BY r.opened_at DESC, r.reconciliation_id LIMIT $1`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v ReconciliationView
			if err := rows.Scan(&v.ReconciliationID, &v.ReceiptID, &v.AttemptID, &v.Kind, &v.Reason, &v.DomainOwner, &v.NextSafeAction, &v.ReceiptState,
				&v.OpenedAt); err != nil {
				return err
			}
			v.OpenedAt = v.OpenedAt.UTC()
			out.Reconciliations = append(out.Reconciliations, v)
		}
		return rows.Err()
	})
	return out, err
}
