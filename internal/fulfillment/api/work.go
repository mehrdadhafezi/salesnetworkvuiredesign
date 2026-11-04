package api

// Work case states (operational_task_v1). COMPLETED / NOT_EXECUTED close
// only the ONE service case: other services of the same subscription stay
// independent and open.
const (
	WorkQueued      = "QUEUED"
	WorkAssigned    = "ASSIGNED"
	WorkInProgress  = "IN_PROGRESS"
	WorkCompleted   = "COMPLETED"
	WorkNotExecuted = "NOT_EXECUTED"
)

// Operational actions (legacy operations execution: no_answer, follow_up;
// projects: add_note).
const (
	ActionNoAnswer = "NO_ANSWER"
	ActionFollowUp = "FOLLOW_UP"
	ActionNote     = "NOTE"
)

// Decisions / causes named by a refusal.
const (
	BlockD35                  = "D-35"
	BlockD38                  = "D-38"
	BlockOPD04                = "OPD-04"
	BlockOPD05                = "OPD-05"
	BlockOPD08                = "OPD-08"
	BlockServiceMissing       = "SERVICE_DEFINITION_MISSING"
	BlockEvidenceMissing      = "REQUIRED_EVIDENCE_MISSING"
	BlockProviderConfirmation = "PROVIDER_CONFIRMATION_REQUIRED"
	BlockUnprovenPath         = "UNPROVEN_LEGACY_PATH"
)

// Outbox event types.
const (
	EventWorkCaseCreated   = "fulfillment.work_case_created"
	EventWorkCaseCompleted = "fulfillment.work_case_completed"
)
