package api

import (
	"context"
	"time"
)

// Slice 12 outbox events of the payments module.
const (
	// EventBatchSubmitted records a committed manual payment batch (audit).
	EventBatchSubmitted = "payments.batch_submitted"
	// EventAttemptEvidenceReady is emitted when every receipt of a target
	// attempt is READY; it is the only payments event integration writes
	// back to legacy (legacy needs the receipt files).
	EventAttemptEvidenceReady = "payments.attempt_evidence_ready"
)

// Target attempt and evidence vocabulary (Slice 12).
const (
	AttemptPendingReview = "PENDING_REVIEW"
	ReviewNotReviewed    = "NOT_REVIEWED"

	EvidencePendingFinalisation = "PENDING_FINALISATION"
	EvidenceReady               = "READY"
	EvidenceNotRecoverable      = "NOT_RECOVERABLE"

	ReceiptQuarantined    = "QUARANTINED"
	ReceiptFinalizing     = "FINALIZING"
	ReceiptReady          = "READY"
	ReceiptStorageError   = "STORAGE_ERROR"
	ReceiptNotRecoverable = "NOT_RECOVERABLE"

	UploadUnbound = "UNBOUND"
	UploadBound   = "BOUND"
	UploadPurged  = "PURGED"
)

// BatchSubmitted is the payload of EventBatchSubmitted.
type BatchSubmitted struct {
	BatchID         string    `json:"batch_id"`
	IdempotencyKey  string    `json:"idempotency_key"`
	CorrelationID   string    `json:"correlation_id"`
	InvoiceID       string    `json:"invoice_id"`
	StageNo         int       `json:"stage_no"`
	AttemptIDs      []string  `json:"attempt_ids"`
	TotalAmount     int64     `json:"total_amount"`
	Unit            string    `json:"unit"`
	ActorUserID     string    `json:"actor_user_id"`
	RolloutMode     string    `json:"rollout_mode"`
	CollectionAfter int64     `json:"collection_version_after"`
	OccurredAt      time.Time `json:"occurred_at"`
}

// EvidenceRef is one READY receipt as written back to legacy.
type EvidenceRef struct {
	ReceiptID     string `json:"receipt_id"`
	EvidenceRef   string `json:"evidence_ref"`
	ContentSHA256 string `json:"content_sha256"`
	MediaType     string `json:"media_type"`
}

// AttemptEvidenceReady is the payload of EventAttemptEvidenceReady. The
// command id of the write-back is the attempt id.
type AttemptEvidenceReady struct {
	CommandID        string        `json:"command_id"`
	IdempotencyKey   string        `json:"idempotency_key"`
	CorrelationID    string        `json:"correlation_id"`
	AttemptID        string        `json:"attempt_id"`
	BatchID          string        `json:"batch_id"`
	ItemKey          string        `json:"item_key"`
	InvoiceID        string        `json:"invoice_id"`
	InvoiceSystem    string        `json:"invoice_source_system"`
	InvoiceSpec      string        `json:"invoice_source_spec"`
	InvoiceSourceKey string        `json:"invoice_source_key"`
	StageNo          int           `json:"stage_no"`
	Amount           int64         `json:"amount"`
	Unit             string        `json:"unit"`
	ActorUserID      string        `json:"actor_user_id"`
	Receipts         []EvidenceRef `json:"receipts"`
	RolloutMode      string        `json:"rollout_mode"`
	OccurredAt       time.Time     `json:"occurred_at"`
}

// ReviewReceipt is one receipt of an attempt awaiting review.
type ReviewReceipt struct {
	ReceiptID string
	UploadID  string
	State     string
	MediaType string
	SizeBytes int
	UpdatedAt time.Time
}

// ReviewAttempt is a target-submitted attempt as the Finance review surface
// reads it (payments published read; finance never reads payments tables).
type ReviewAttempt struct {
	AttemptID         string
	BatchID           string
	InvoiceID         string
	StageNo           int
	Amount            int64
	Unit              string
	State             string
	ReviewProvenance  string
	EvidenceState     string
	SubmittedByUserID string
	SubmittedAt       time.Time
	Receipts          []ReviewReceipt
}

// ReviewSource is the published read the Finance review surface uses.
type ReviewSource interface {
	// ReviewQueue lists target attempts awaiting review, oldest first, after
	// the keyset cursor (attempt id), at most limit+1 rows.
	ReviewQueue(ctx context.Context, after *string, limit int) ([]ReviewAttempt, error)
	// ReviewAttempt reads one attempt; ok is false when unknown.
	ReviewAttempt(ctx context.Context, attemptID string) (a ReviewAttempt, ok bool, err error)
}
