// Package api is the published fulfillment contract (Slice 10, READ /
// SHADOW). Integration normalizes legacy operational evidence into these
// neutral records; the offline shadow import consumes them. No assignment,
// completion, shipping, upgrade or evidence command exists.
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Case kinds stay explicit: one source's completion never stands for
// another's.
const (
	KindServiceFollowup = "SERVICE_FOLLOWUP"  // a subscription item's follow-up workflow
	KindExecution       = "EXECUTION"         // operations execution case (subtype = execution type)
	KindUpgradeRequest  = "UPGRADE_EXECUTION" // upgrade / upsell request of an item
	KindShipping        = "SHIPPING"          // physical shipping order of an invoice
)

// Mapped statuses (only where a legacy writer proves the meaning; anything
// else is UNKNOWN with the raw value kept, never a pending / completed /
// failed fallback). COMPLETED is always qualified by its kind and
// namespace: it is an operational completion only — never invoice paid,
// subscription active, sales completed or finance settled.
const (
	StatusNotStarted       = "NOT_STARTED"
	StatusInProgress       = "IN_PROGRESS"
	StatusAwaitingPayment  = "AWAITING_PAYMENT"
	StatusAwaitingProvider = "AWAITING_PROVIDER"
	StatusProviderRecorded = "PROVIDER_ACTION_RECORDED"
	StatusOutcomeRecorded  = "OUTCOME_RECORDED"
	StatusShipped          = "SHIPPED"
	StatusDelivered        = "DELIVERED"
	StatusReturned         = "RETURNED"
	StatusNotExecuted      = "NOT_EXECUTED"
	StatusCancelled        = "CANCELLED"
	StatusCompleted        = "COMPLETED"
	StatusFailedAtProvider = "FAILED_AT_PROVIDER"
	StatusUnknown          = "UNKNOWN"
	StatusEmpty            = "EMPTY"
)

// Service outcomes (form final decision).
const (
	OutcomeExecuted    = "EXECUTED"
	OutcomeNotExecuted = "NOT_EXECUTED"
	OutcomeUnknown     = "UNKNOWN"
	OutcomeNone        = "NONE"
)

// Assignment roles (current assignee evidence; never an owner, credit
// owner, customer or subscription owner).
const (
	RoleManager = "MANAGER"
	RoleExpert  = "EXPERT"
	RoleHandler = "HANDLER"
)

// Source coverage.
const (
	SourceAvailable = "AVAILABLE"
	SourcePartial   = "PARTIAL"
	SourceMissing   = "MISSING"
	SourceUnknown   = "UNKNOWN"
)

// UserRef is a legacy staff reference with its identity resolution.
type UserRef struct {
	LegacyKey string
	UserID    *string
}

// Assignment is one current assignee of a case.
type Assignment struct {
	Role          string
	User          UserRef
	AssignedAtRaw *string
}

// Link is a cross-module link (ID only when RESOLVED).
type Link struct {
	State     string
	ID        *string
	LegacyKey *string
}

// CaseRecord is one normalized fulfillment case.
type CaseRecord struct {
	Kind            string
	SourceSpec      string
	SourceKey       string
	SubtypeRaw      *string
	SubscriptionID  *string
	ItemID          *string
	Invoice         Link
	StatusNamespace string
	StatusRaw       *string
	StatusMapped    string
	// Secondary raw statuses kept verbatim (never merged into the status).
	ShippingStatusRaw  *string
	ProviderStatusRaw  *string
	ContactStatusRaw   *string
	Assignments        []Assignment
	CompletedAtRaw     *string
	CompletedBy        *UserRef
	CancelledAtRaw     *string
	ShippedAtRaw       *string
	DeliveredAtRaw     *string
	StatusUpdatedAtRaw *string
	CarrierRaw         *string
	// External references are namespaced and opaque: only a SHA-256 of the
	// raw value is kept (never a global id, never shown raw).
	TrackingRefSHA    []byte
	ProviderRefSHA    []byte
	ProviderRefScheme *string
	CreatedAtRaw      *string
	UpdatedAtRaw      *string
}

// EvidenceRecord is one service evidence METADATA record (no content).
type EvidenceRecord struct {
	SourceKey, CaseSourceSpec, CaseSourceKey string
	FormDefinitionKey                        *string
	SubmittedBy                              *UserRef
	SubmittedAtRaw                           *string
	OutcomeRaw                               *string
	OutcomeMapped                            string
	DecidedBy                                *UserRef
	DecidedAtRaw                             *string
}

// EventRecord is one append-only case history event.
type EventRecord struct {
	SourceSpec, SourceKey         string
	CaseSourceSpec, CaseSourceKey string
	Actor                         *UserRef
	EventKeyRaw                   string
	OldStatusRaw, NewStatusRaw    *string
	OccurredAtRaw                 *string
}

// Import is one normalized staged snapshot.
type Import struct {
	BatchID, SnapshotID, SourceSystem string
	Cutoff                            time.Time
	Sources                           map[string]string
	SourceRows                        map[string]int64
	Cases                             []CaseRecord
	Evidence                          []EvidenceRecord
	Events                            []EventRecord
	Skipped                           map[string]int64
}

// CaseID is the stable, deterministic id of a case: system + source spec
// (kind of source) + native key.
func CaseID(system, spec, key string) string {
	h := sha256.New()
	for _, p := range []string{"fulfillment", "case", system, spec, key} {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	b := h.Sum(nil)
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b[:16])
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}
