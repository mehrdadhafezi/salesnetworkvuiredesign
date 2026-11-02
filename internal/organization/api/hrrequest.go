package api

import "time"

// Slice 11 controlled write contract of HR change requests. Only the
// evidenced legacy types and transitions exist: request creation, a step
// approval by the current reviewer and a rejection. Final apply (personnel
// change, termination, handover) is unavailable (OPD-05, SD-04; HR apply is
// IMPLEMENTATION BLOCKED). No request ever changes identity, credentials or
// capabilities (D-34, D-42).

// HR request types (legacy `request_type`).
const (
	HRTransferSeller  = "transfer_seller"
	HRTerminateSeller = "terminate_seller"
)

// HR request states (legacy namespace, preserved).
const (
	HRPendingReview = "pending_review"
	HRPendingHR     = "pending_hr"
	HRApproved      = "approved"
	HRRejected      = "rejected"
	HRFailed        = "failed"
	HRCancelled     = "cancelled"
)

// HR decisions.
const (
	HRDecisionApprove = "approve"
	HRDecisionReject  = "reject"
)

// Outbox event types.
const (
	EventHRRequestCreated = "organization.hr_request_created"
	EventHRRequestDecided = "organization.hr_request_decided"
)

// HRRequestCreated is the published payload of a created request (applied or
// shadow-evaluated). User references are target identity ids; integration
// maps them to legacy keys through the identity published API.
type HRRequestCreated struct {
	CommandID           string    `json:"command_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CorrelationID       string    `json:"correlation_id"`
	RequestID           string    `json:"request_id"`
	RequestType         string    `json:"request_type"`
	SubjectUserID       string    `json:"subject_user_id"`
	SubjectParentUserID *string   `json:"subject_parent_user_id,omitempty"`
	TargetParentUserID  *string   `json:"target_parent_user_id,omitempty"`
	RequestedByUserID   string    `json:"requested_by_user_id"`
	CurrentReviewerUser *string   `json:"current_reviewer_user_id,omitempty"`
	Status              string    `json:"status"`
	Reason              string    `json:"reason"`
	RolloutMode         string    `json:"rollout_mode"`
	Applied             bool      `json:"applied"`
	Version             int64     `json:"version"`
	OccurredAt          time.Time `json:"occurred_at"`
}

// HRRequestDecided is the published payload of a step approval or rejection.
type HRRequestDecided struct {
	CommandID          string    `json:"command_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	CorrelationID      string    `json:"correlation_id"`
	RequestID          string    `json:"request_id"`
	Decision           string    `json:"decision"`
	FromStatus         string    `json:"from_status"`
	ToStatus           string    `json:"to_status"`
	NextReviewerUserID *string   `json:"next_reviewer_user_id,omitempty"`
	Notes              *string   `json:"notes,omitempty"`
	ActorUserID        string    `json:"actor_user_id"`
	RolloutMode        string    `json:"rollout_mode"`
	Applied            bool      `json:"applied"`
	BeforeVersion      int64     `json:"before_version"`
	AfterVersion       *int64    `json:"after_version,omitempty"`
	OccurredAt         time.Time `json:"occurred_at"`
}
