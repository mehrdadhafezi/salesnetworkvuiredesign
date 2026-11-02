package api

import "time"

// Slice 11 controlled write contract of the cases module. Only the contact
// outcome command exists; ready, invoice claim, case creation, aliasing and
// every custody move stay unavailable (see docs/implementation/
// slice-11-command-matrix.md).

// Contact outcomes. NoAnswer, Callback and NotPurchased are the only values
// a target command may record. Duplicate (D-29 / D-31: the seller duplicate
// intent is unresolved) and PreInvoice (only through invoice issuance,
// ADR-01) are named so they can be refused explicitly.
const (
	OutcomeNoAnswer     = "NO_ANSWER"
	OutcomeCallback     = "CALLBACK"
	OutcomeNotPurchased = "NOT_PURCHASED"
	OutcomeDuplicate    = "DUPLICATE"
	OutcomePreInvoice   = "PRE_INVOICE"
)

// RecordableOutcome reports an outcome a target command may record.
func RecordableOutcome(o string) bool {
	return o == OutcomeNoAnswer || o == OutcomeCallback || o == OutcomeNotPurchased
}

// EventContactOutcomeRecorded is the outbox event type of an accepted
// contact outcome command (applied or shadow-evaluated).
const EventContactOutcomeRecorded = "cases.contact_outcome_recorded"

// ContactOutcomeRecorded is the published payload of the event. It carries
// the legacy source reference of the case (spec + native key), never a phone
// or a legacy table name; only integration maps it to legacy storage.
type ContactOutcomeRecorded struct {
	CommandID       string     `json:"command_id"`
	IdempotencyKey  string     `json:"idempotency_key"`
	CorrelationID   string     `json:"correlation_id"`
	CaseID          string     `json:"case_id"`
	SourceSystem    string     `json:"source_system"`
	SourceSpec      string     `json:"source_spec"`
	SourceKey       string     `json:"source_key"`
	Outcome         string     `json:"outcome"`
	Reason          *string    `json:"reason,omitempty"`
	CallbackAt      *time.Time `json:"callback_at,omitempty"`
	PreviousOutcome *string    `json:"previous_outcome,omitempty"`
	ActorUserID     string     `json:"actor_user_id"`
	RolloutMode     string     `json:"rollout_mode"`
	Applied         bool       `json:"applied"`
	BeforeVersion   int64      `json:"before_version"`
	AfterVersion    *int64     `json:"after_version,omitempty"`
	OccurredAt      time.Time  `json:"occurred_at"`
}
