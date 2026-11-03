package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/idempotency"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// ScopeBatch is the idempotency scope and command type of SubmitBatch.
const ScopeBatch = "payments.submit_batch"

// Entry modes. Only RECEIPT is accepted; INFO (transfer details without a
// receipt) is BLOCKED: whether evidence may be absent is D-43 territory.
const (
	EntryReceipt = "RECEIPT"
	EntryInfo    = "INFO"
)

// ReceiptRef binds one staged upload to an item; the declared content hash
// must equal the stored one (no evidence is attached to the wrong payment).
type ReceiptRef struct {
	UploadID      string
	ContentSHA256 string
}

// Item is one payment transaction of a batch.
type Item struct {
	ItemKey  string
	Amount   int64
	Unit     string
	Receipts []ReceiptRef
}

// BatchCommand is one manual payment batch for an invoice's current stage.
type BatchCommand struct {
	InvoiceID       string
	StageNo         int
	EntryMode       string
	Items           []Item
	ExpectedVersion int64
	IdempotencyKey  string
	CorrelationID   string
	Actor           Actor
}

// ReceiptOutcome is a bound receipt in an item result.
type ReceiptOutcome struct {
	ReceiptID string `json:"receipt_id"`
	UploadID  string `json:"upload_id"`
	State     string `json:"state"`
}

// ItemResult is one item's own outcome. Valid tells whether the item itself
// passed validation; Committed is true only when the whole batch committed.
type ItemResult struct {
	ItemKey    string             `json:"item_key"`
	Status     result.Status      `json:"-"`
	StatusName string             `json:"status"`
	Reasons    []reasoncodes.Code `json:"reasons"`
	BlockedBy  []string           `json:"blocked_by,omitempty"`
	Valid      bool               `json:"valid"`
	Committed  bool               `json:"committed"`
	AttemptID  string             `json:"attempt_id,omitempty"`
	Amount     int64              `json:"amount"`
	Unit       string             `json:"unit"`
	Receipts   []ReceiptOutcome   `json:"receipts"`
}

// BatchResult is the batch outcome; it is also what an idempotent replay
// returns. A batch-level SUCCESS means every item committed; anything else
// means nothing committed (all-or-nothing, Architecture V1 §15.2).
type BatchResult struct {
	Status            result.Status      `json:"-"`
	StatusName        string             `json:"status"`
	Reasons           []reasoncodes.Code `json:"reasons"`
	BlockedBy         []string           `json:"blocked_by,omitempty"`
	Mode              rollout.Mode       `json:"rollout_mode"`
	BatchID           string             `json:"batch_id,omitempty"`
	InvoiceID         string             `json:"invoice_id"`
	StageNo           int                `json:"stage_no"`
	Applied           bool               `json:"applied"`
	CollectionVersion int64              `json:"collection_version,omitempty"`
	CurrentVersion    int64              `json:"-"`
	TotalAmount       int64              `json:"total_amount"`
	Unit              string             `json:"unit,omitempty"`
	Items             []ItemResult       `json:"items"`
	EvidenceState     string             `json:"evidence_state,omitempty"`
	WritebackState    string             `json:"writeback_state"`
	Replayed          bool               `json:"-"`
}

// Write-back states the command can report at commit time.
const (
	WritebackAwaitingEvidence = "AWAITING_EVIDENCE_FINALISATION"
	WritebackNotRequested     = "NOT_REQUESTED"
)

func validateBatch(c BatchCommand) (*BatchResult, error) {
	if !uuidPattern.MatchString(c.InvoiceID) {
		return nil, invalid("invoice_id")
	}
	if !keyPattern.MatchString(c.IdempotencyKey) {
		return nil, invalid("idempotency_key")
	}
	if c.CorrelationID == "" || len(c.CorrelationID) > 128 {
		return nil, invalid("correlation_id")
	}
	if c.ExpectedVersion < 0 {
		return nil, invalid("expected_version")
	}
	if c.StageNo < 1 {
		return nil, invalid("stage_no")
	}
	if len(c.Items) < 1 || len(c.Items) > MaxItems {
		return nil, invalid("items (1 to 10)")
	}
	keys, uploads, receipts := map[string]bool{}, map[string]bool{}, 0
	for _, it := range c.Items {
		if !itemKeyPattern.MatchString(it.ItemKey) || keys[it.ItemKey] {
			return nil, invalid("item_key (unique per batch)")
		}
		keys[it.ItemKey] = true
		if it.Amount <= 0 {
			return nil, invalid("amount (positive integer)")
		}
		if it.Unit != "IRR" && it.Unit != "IRT" {
			return nil, invalid("unit")
		}
		for _, r := range it.Receipts {
			if !uuidPattern.MatchString(r.UploadID) || !shaPattern.MatchString(r.ContentSHA256) || uploads[r.UploadID] {
				return nil, invalid("receipts (unique upload_id and content_sha256)")
			}
			uploads[r.UploadID] = true
		}
		receipts += len(it.Receipts)
	}
	if receipts > MaxReceipts {
		return nil, invalid("receipts (at most 10 per submission)")
	}
	switch c.EntryMode {
	case EntryInfo:
		r := BatchResult{Status: result.Blocked, StatusName: result.Blocked.String(), Reasons: []reasoncodes.Code{reasoncodes.PolicyUnresolved},
			BlockedBy: []string{DecisionD43}, InvoiceID: c.InvoiceID, StageNo: c.StageNo, Items: []ItemResult{}, WritebackState: WritebackNotRequested}
		return &r, nil
	case EntryReceipt:
	default:
		return nil, invalid("entry_mode")
	}
	for _, it := range c.Items {
		if len(it.Receipts) == 0 {
			return nil, invalid("receipts (receipt mode needs at least one receipt per item)")
		}
	}
	return nil, nil
}

// canonicalBatch is the deterministic request encoding hashed for
// idempotency; the actor is part of it.
func canonicalBatch(c BatchCommand) []byte {
	type rr struct {
		U string `json:"upload_id"`
		H string `json:"content_sha256"`
	}
	type it struct {
		K string `json:"item_key"`
		A int64  `json:"amount"`
		U string `json:"unit"`
		R []rr   `json:"receipts"`
	}
	items := make([]it, 0, len(c.Items))
	for _, i := range c.Items {
		rs := make([]rr, 0, len(i.Receipts))
		for _, r := range i.Receipts {
			rs = append(rs, rr{r.UploadID, r.ContentSHA256})
		}
		items = append(items, it{i.ItemKey, i.Amount, i.Unit, rs})
	}
	b, _ := json.Marshal(struct {
		Invoice  string `json:"invoice_id"`
		Stage    int    `json:"stage_no"`
		Mode     string `json:"entry_mode"`
		Items    []it   `json:"items"`
		Expected int64  `json:"expected_version"`
		Actor    string `json:"actor_user_id"`
	}{c.InvoiceID, c.StageNo, c.EntryMode, items, c.ExpectedVersion, c.Actor.UserID})
	return b
}

type errBatchRefused struct{ r BatchResult }

func (e errBatchRefused) Error() string { return "refused: " + e.r.StatusName }

func (c BatchCommand) itemsWith(status result.Status) []ItemResult {
	out := make([]ItemResult, 0, len(c.Items))
	for _, it := range c.Items {
		out = append(out, ItemResult{ItemKey: it.ItemKey, Status: status, StatusName: status.String(), Reasons: []reasoncodes.Code{}, Valid: false,
			Amount: it.Amount, Unit: it.Unit, Receipts: []ReceiptOutcome{}})
	}
	return out
}

func (c BatchCommand) refuse(mode rollout.Mode, e Eligibility) BatchResult {
	return BatchResult{Status: e.Status, StatusName: e.Status.String(), Reasons: e.Reasons, BlockedBy: e.BlockedBy, Mode: mode, InvoiceID: c.InvoiceID,
		StageNo: c.StageNo, Items: c.itemsWith(e.Status), WritebackState: WritebackNotRequested}
}

type boundUpload struct {
	uploadID, ref, media string
	sum                  []byte
	size                 int
}

// SubmitBatch runs the command in mode.
func (s *Service) SubmitBatch(ctx context.Context, mode rollout.Mode, c BatchCommand) (BatchResult, error) {
	if pre, err := validateBatch(c); err != nil || pre != nil {
		if pre != nil {
			pre.Mode = mode
			return *pre, nil
		}
		return BatchResult{}, err
	}
	if st, code, blocked, refused := modeRefusal(mode); refused {
		if mode != rollout.Shadow {
			mode = rollout.Disabled
		}
		return c.refuse(mode, Eligibility{Status: st, Reasons: []reasoncodes.Code{code}, BlockedBy: blocked}), nil
	}
	if c.Actor.UserID == "" || c.Actor.Principal == "" {
		return c.refuse(mode, refusal(result.Unauthorized, reasoncodes.RowScopeDenied)), nil
	}
	sum := sha256.Sum256(canonicalBatch(c))
	req := idempotency.Request{Scope: ScopeBatch, Key: c.IdempotencyKey, Hash: idempotency.Hash(sum)}
	batchID := ids.Stable(ScopeBatch, c.IdempotencyKey)
	var out BatchResult
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		at := now.Time().UTC()
		acq, err := idempotency.Acquire(ctx, tx, req, now)
		if err != nil {
			return err
		}
		switch acq.State {
		case idempotency.Replay:
			if err := json.Unmarshal(acq.Result, &out); err != nil {
				return err
			}
			out.Replayed = true
			out.Status, _ = result.ParseStatus(out.StatusName)
			for i := range out.Items {
				out.Items[i].Status, _ = result.ParseStatus(out.Items[i].StatusName)
			}
			return nil
		case idempotency.Conflict:
			return errBatchRefused{c.refuse(mode, refusal(result.Conflict, reasoncodes.ContractInvalid))}
		case idempotency.InProgress:
			return errBatchRefused{c.refuse(mode, refusal(result.Unknown, reasoncodes.OutcomeUnknown))}
		}
		// Serialise on payments' own per-stage collection account, then
		// re-evaluate everything (apply-time recheck).
		version, err := lockAccount(ctx, tx, c.InvoiceID, c.StageNo, at)
		if err != nil {
			return err
		}
		e, found, err := s.evaluate(ctx, tx, c.InvoiceID, c.Actor)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if e.Status != result.Success {
			return errBatchRefused{c.refuse(mode, e)}
		}
		if e.StageNo != c.StageNo {
			return errBatchRefused{c.refuse(mode, refusal(result.Conflict, reasoncodes.StaleState))}
		}
		if version != c.ExpectedVersion {
			r := c.refuse(mode, refusal(result.Conflict, reasoncodes.StaleState))
			r.CurrentVersion = version
			return errBatchRefused{r}
		}
		items := make([]ItemResult, len(c.Items))
		bound := make([][]boundUpload, len(c.Items))
		remaining, totalBytes, total := e.Available, 0, int64(0)
		anyRefused := false
		for i, it := range c.Items {
			ir := ItemResult{ItemKey: it.ItemKey, Status: result.Success, Reasons: []reasoncodes.Code{}, Valid: true, Amount: it.Amount, Unit: it.Unit,
				Receipts: []ReceiptOutcome{}}
			fail := func(st result.Status, code reasoncodes.Code, blocked ...string) {
				if ir.Valid || severity[st] > severity[ir.Status] {
					ir.Status = st
				}
				ir.Valid = false
				ir.Reasons = append(ir.Reasons, code)
				ir.BlockedBy = append(ir.BlockedBy, blocked...)
			}
			if it.Unit != e.Unit {
				fail(result.Blocked, reasoncodes.PolicyUnresolved, DecisionOPD06)
			} else if it.Amount > remaining {
				fail(result.Blocked, reasoncodes.ActionDenied)
			} else {
				remaining -= it.Amount
				total += it.Amount
			}
			for _, r := range it.Receipts {
				var (
					uploader, state, ref, media string
					hash                        []byte
					size                        int
					expires                     time.Time
				)
				err := tx.QueryRow(ctx, `SELECT uploader_user_id::text, state, quarantine_ref, media_type, content_sha256, size_bytes, expires_at
					FROM payments.upload_object WHERE upload_id = $1 FOR UPDATE`, r.UploadID).Scan(&uploader, &state, &ref, &media, &hash, &size, &expires)
				switch {
				case errors.Is(err, pgx.ErrNoRows) || (err == nil && uploader != c.Actor.UserID):
					// Another user's upload is reported exactly like a missing one.
					fail(result.Blocked, reasoncodes.ContractInvalid)
					continue
				case err != nil:
					return err
				}
				want, _ := hex.DecodeString(r.ContentSHA256)
				switch {
				case string(want) != string(hash):
					fail(result.Blocked, reasoncodes.ContractInvalid)
				case state == api.UploadBound:
					fail(result.Conflict, reasoncodes.LinkageConflict)
				case state != api.UploadUnbound || !expires.After(at):
					fail(result.Blocked, reasoncodes.StaleState)
				default:
					totalBytes += size
					bound[i] = append(bound[i], boundUpload{uploadID: r.UploadID, ref: ref, media: media, sum: hash, size: size})
				}
			}
			if totalBytes > MaxBatchBytes {
				fail(result.Blocked, reasoncodes.ContractInvalid)
			}
			ir.Reasons = uniqueCodes(ir.Reasons)
			ir.BlockedBy = uniqueStrings(ir.BlockedBy)
			if !ir.Valid {
				anyRefused = true
			}
			ir.StatusName = ir.Status.String()
			items[i] = ir
		}
		if anyRefused {
			r := BatchResult{Mode: mode, InvoiceID: c.InvoiceID, StageNo: c.StageNo, Items: items, Unit: e.Unit, WritebackState: WritebackNotRequested}
			worst, codes, blocked := result.Blocked, []reasoncodes.Code{}, []string{}
			for j := range r.Items {
				it := &r.Items[j]
				if it.Valid {
					// All-or-nothing: a valid item of a refused batch is not committed.
					it.Status, it.StatusName = result.Blocked, result.Blocked.String()
					continue
				}
				if severity[it.Status] > severity[worst] {
					worst = it.Status
				}
				codes = append(codes, it.Reasons...)
				blocked = append(blocked, it.BlockedBy...)
			}
			r.Status, r.StatusName, r.Reasons, r.BlockedBy = worst, worst.String(), uniqueCodes(codes), uniqueStrings(blocked)
			return errBatchRefused{r}
		}
		// Commit: batch, items/attempts, receipts, bindings, history, the
		// collection account version, command record, outbox, idempotency.
		after := version + 1
		if _, err := tx.Exec(ctx, `INSERT INTO payments.payment_batch (batch_id, invoice_id, stage_no, stage_id, invoice_source_system,
			invoice_source_spec, invoice_source_key, idempotency_key, request_sha256, item_count, total_amount, unit, before_version, after_version,
			actor_principal, actor_user_id, correlation_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
			batchID, c.InvoiceID, c.StageNo, e.StageID, e.Target.SourceSystem, e.Target.SourceSpec, e.Target.SourceKey, c.IdempotencyKey, sum[:],
			len(c.Items), total, e.Unit, version, after, c.Actor.Principal, c.Actor.UserID, c.CorrelationID, at); err != nil {
			return err
		}
		attemptIDs := make([]string, 0, len(c.Items))
		for i, it := range c.Items {
			attemptID := ids.Stable("payments.attempt", batchID, it.ItemKey)
			attemptIDs = append(attemptIDs, attemptID)
			if _, err := tx.Exec(ctx, `INSERT INTO payments.submitted_attempt (attempt_id, batch_id, item_key, item_no, invoice_id, stage_no, channel,
				entry_mode, amount, unit, state, review_provenance, evidence_state, submitted_by_user_id, created_at, updated_at, version)
				VALUES ($1, $2, $3, $4, $5, $6, 'MANUAL_CARD', 'RECEIPT', $7, $8, $9, $10, $11, $12, $13, $13, 1)`,
				attemptID, batchID, it.ItemKey, i+1, c.InvoiceID, c.StageNo, it.Amount, it.Unit, api.AttemptPendingReview, api.ReviewNotReviewed,
				api.EvidencePendingFinalisation, c.Actor.UserID, at); err != nil {
				return err
			}
			detail, _ := json.Marshal(map[string]any{"amount": it.Amount, "unit": it.Unit, "receipt_count": len(bound[i]), "batch_id": batchID})
			if _, err := tx.Exec(ctx, `INSERT INTO payments.submitted_attempt_event (event_id, attempt_id, kind, detail, actor_principal,
				correlation_id, occurred_at) VALUES ($1, $2, 'SUBMITTED', $3, $4, $5, $6)`,
				ids.Stable("payments.attempt_event", attemptID, "SUBMITTED"), attemptID, detail, c.Actor.Principal, c.CorrelationID, at); err != nil {
				return err
			}
			items[i].AttemptID, items[i].Committed = attemptID, true
			for _, b := range bound[i] {
				receiptID := ids.Stable("payments.receipt_evidence", attemptID, b.uploadID)
				if _, err := tx.Exec(ctx, `INSERT INTO payments.receipt_evidence (receipt_id, attempt_id, batch_id, invoice_id, upload_id, content_sha256,
					size_bytes, media_type, quarantine_ref, state, created_at, updated_at, version)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'QUARANTINED', $10, $10, 1)`,
					receiptID, attemptID, batchID, c.InvoiceID, b.uploadID, b.sum, b.size, b.media, b.ref, at); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO payments.receipt_state_history (history_id, receipt_id, from_state, to_state, reason,
					actor_principal, occurred_at) VALUES ($1, $2, NULL, 'QUARANTINED', 'bound to attempt by batch submission', $3, $4)`,
					ids.Stable("payments.receipt_history", receiptID, "bind"), receiptID, c.Actor.Principal, at); err != nil {
					return err
				}
				tag, err := tx.Exec(ctx, `UPDATE payments.upload_object SET state = 'BOUND', bound_at = $2, version = version + 1
					WHERE upload_id = $1 AND state = 'UNBOUND'`, b.uploadID, at)
				if err != nil {
					return err
				}
				if tag.RowsAffected() != 1 {
					return errBatchRefused{c.refuse(mode, refusal(result.Conflict, reasoncodes.LinkageConflict))}
				}
				items[i].Receipts = append(items[i].Receipts, ReceiptOutcome{ReceiptID: receiptID, UploadID: b.uploadID, State: api.ReceiptQuarantined})
			}
		}
		tag, err := tx.Exec(ctx, `UPDATE payments.collection_account SET version = version + 1, updated_at = $3
			WHERE invoice_id = $1 AND stage_no = $2 AND version = $4`, c.InvoiceID, c.StageNo, at, version)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errBatchRefused{c.refuse(mode, refusal(result.Conflict, reasoncodes.StaleState))}
		}
		intent, _ := json.Marshal(map[string]any{"invoice_id": c.InvoiceID, "stage_no": c.StageNo, "items": len(c.Items), "total_amount": total,
			"unit": e.Unit, "attempt_ids": attemptIDs})
		if _, err := tx.Exec(ctx, `INSERT INTO payments.command_record (command_id, command_type, subject_id, idempotency_key, request_sha256,
			rollout_mode, applied, intent, actor_principal, actor_user_id, correlation_id, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8, $9, $10, $11)`,
			batchID, ScopeBatch, batchID, c.IdempotencyKey, sum[:], string(mode), intent, c.Actor.Principal, c.Actor.UserID, c.CorrelationID, at); err != nil {
			return err
		}
		payload, err := json.Marshal(api.BatchSubmitted{BatchID: batchID, IdempotencyKey: c.IdempotencyKey, CorrelationID: c.CorrelationID,
			InvoiceID: c.InvoiceID, StageNo: c.StageNo, AttemptIDs: attemptIDs, TotalAmount: total, Unit: e.Unit, ActorUserID: c.Actor.UserID,
			RolloutMode: string(mode), CollectionAfter: after, OccurredAt: at})
		if err != nil {
			return err
		}
		corr := c.CorrelationID
		if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("payments.outbox", batchID), Type: api.EventBatchSubmitted, Version: 1,
			Payload: payload, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
			return err
		}
		for i := range items {
			items[i].StatusName = result.Success.String()
		}
		out = BatchResult{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, Mode: mode, BatchID: batchID,
			InvoiceID: c.InvoiceID, StageNo: c.StageNo, Applied: true, CollectionVersion: after, TotalAmount: total, Unit: e.Unit, Items: items,
			EvidenceState: api.EvidencePendingFinalisation, WritebackState: WritebackAwaitingEvidence}
		stored, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return idempotency.Complete(ctx, tx, req, stored, now)
	})
	var refused errBatchRefused
	if errors.As(err, &refused) {
		return refused.r, nil
	}
	if err != nil {
		return BatchResult{}, err
	}
	return out, nil
}
