package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// RunState is a commission run / item lifecycle state. The states are the
// architecture's (§19) vocabulary; the legacy raw values are preserved
// beside them and never overwritten.
type RunState string

const (
	StateCalculated       RunState = "CALCULATED" // a CRM shadow run (never approvable here)
	StateGenerated        RunState = "GENERATED"
	StateApproved         RunState = "APPROVED"
	StateRejected         RunState = "REJECTED"
	StatePostingRequested RunState = "POSTING_REQUESTED"
	StatePosted           RunState = "POSTED"
	StatePostingFailed    RunState = "POSTING_FAILED"
	StateReversed         RunState = "REVERSED"
	StateCorrected        RunState = "CORRECTED"
	StateSettled          RunState = "SETTLED"
	StateUnknown          RunState = "UNKNOWN"
)

// transitions are ONLY the ones evidenced in plugin/current 2.0.160:
//
//	generated -> approved | rejected        class-sn-plugin.php:33568-33580
//	rejected  -> approved                   :33568 (approve allowed unless already approved)
//	approved  -> posting requested (APPLY)  :33616-33620
//	posting requested -> posted | failed    :33641-33652
//	posted    -> reversed                   :29297-29331 (financial reject)
//	posted    -> settled                    sn_wallet_settlement_* (settlement batch)
//
// approved -> rejected is NOT evidenced (reject requires current != approved).
var transitions = map[RunState][]RunState{
	StateGenerated:        {StateApproved, StateRejected},
	StateRejected:         {StateApproved},
	StateApproved:         {StatePostingRequested},
	StatePostingRequested: {StatePosted, StatePostingFailed},
	StatePosted:           {StateReversed, StateSettled},
}

// ErrTransition reports a transition that is not evidenced.
var ErrTransition = errors.New("commission: transition not evidenced")

// ErrCommandBlocked reports a command that exists in the lifecycle but is
// decision-blocked in Slice 13 (approval / posting: OPD-09, D-41).
var ErrCommandBlocked = errors.New("commission: command is decision-blocked (OPD-09 / D-41); no approval or posting is authorised")

// CanTransition reports whether from -> to is an evidenced transition.
func CanTransition(from, to RunState) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// History validates a sequence of lifecycle states (e.g. a historical
// rejected-then-approved run): every step must be evidenced. It never
// collapses steps: generated, approved, posted and settled stay distinct.
func History(states ...RunState) error {
	for i := 1; i < len(states); i++ {
		if !CanTransition(states[i-1], states[i]) {
			return fmt.Errorf("%w: %s -> %s", ErrTransition, states[i-1], states[i])
		}
	}
	return nil
}

// Approve is the approval command. It is decision-blocked in Slice 13: it
// validates the transition and then always refuses.
func Approve(from RunState) error {
	if !CanTransition(from, StateApproved) {
		return fmt.Errorf("%w: %s -> %s", ErrTransition, from, StateApproved)
	}
	return ErrCommandBlocked
}

// RequestPosting is the posting command. It is decision-blocked in Slice 13.
func RequestPosting(from RunState) error {
	if !CanTransition(from, StatePostingRequested) {
		return fmt.Errorf("%w: %s -> %s", ErrTransition, from, StatePostingRequested)
	}
	return ErrCommandBlocked
}

// LegacyRunState maps a legacy dry run's raw approval_status. Unknown raw
// values stay UNKNOWN with the raw value kept by the caller.
func LegacyRunState(approvalRaw *string) RunState {
	if approvalRaw == nil {
		return StateUnknown
	}
	switch *approvalRaw {
	case "generated":
		return StateGenerated
	case "approved":
		return StateApproved
	case "rejected":
		return StateRejected
	}
	return StateUnknown
}

// LegacyItemPosting maps a legacy dry-run item's raw posting_status.
func LegacyItemPosting(raw *string) RunState {
	if raw == nil {
		return StateUnknown
	}
	switch *raw {
	case "pending":
		return StateGenerated // not posted; generated != approved != posted
	case "posted":
		return StatePosted
	case "error":
		return StatePostingFailed
	}
	return StateUnknown
}

// PostingRequested is the commission -> finance posting CONTRACT
// (CommissionPostingRequested -> Finance -> PostingCompleted / Rejected).
// It is designed and validated only: Slice 13 never constructs one from a
// run, never appends it to an outbox and never dispatches it (lint s13).
type PostingRequested struct {
	RequestKey     string // deterministic idempotency key
	EntitlementKey string
	PolicyKey      string
	PolicyVer      int
	RunID          string
	RunState       RunState
	RecipientKey   string
	Amount         Exact
	Reverses       *string // the ORIGINAL entitlement's request key for a reversal
}

// PostingResult is finance's answer to a PostingRequested.
type PostingResult string

const (
	PostingCompleted PostingResult = "POSTING_COMPLETED"
	PostingRejected  PostingResult = "POSTING_REJECTED"
	// PostingOutcomeUnknown must be reconciled before any retry.
	PostingOutcomeUnknown PostingResult = "OUTCOME_UNKNOWN"
)

// PostingRequestKey is the deterministic key: the same entitlement under the
// same policy version can only ever produce ONE request.
func PostingRequestKey(entitlementKey, policyKey string, policyVer int, reverses *string) string {
	parts := []string{"commission-posting", entitlementKey, policyKey, itoa(policyVer)}
	if reverses != nil {
		parts = append(parts, "reverses", *reverses)
	}
	h := sha256.Sum256([]byte(fmt.Sprint(parts)))
	return hex.EncodeToString(h[:])
}

// Validate checks the contract shape. An unapproved run, a non-positive
// amount, a missing recipient snapshot or a key mismatch is rejected.
func (p PostingRequested) Validate() error {
	if p.EntitlementKey == "" || p.PolicyKey == "" || p.RunID == "" || p.RecipientKey == "" {
		return errors.New("commission: posting request needs entitlement, policy, run and recipient snapshot")
	}
	if p.RunState != StateApproved {
		return fmt.Errorf("%w: run is %s, only an APPROVED snapshot may request posting", ErrTransition, p.RunState)
	}
	if !p.Amount.Unit.Valid() {
		return money.ErrInvalidUnit
	}
	if p.Amount.Hundredths <= 0 {
		return errors.New("commission: posting amount must be positive (a reversal references the original instead)")
	}
	if p.RequestKey != PostingRequestKey(p.EntitlementKey, p.PolicyKey, p.PolicyVer, p.Reverses) {
		return errors.New("commission: posting request key is not the deterministic key")
	}
	return nil
}

// PostingRequests is an in-memory de-duplicating set of posting requests
// (the contract's idempotency rule): a second request for the same key is
// refused, never sent twice.
type PostingRequests struct{ seen map[string]bool }

// ErrDuplicatePosting reports a second request for the same entitlement.
var ErrDuplicatePosting = errors.New("commission: duplicate posting request")

// Add validates and records a request.
func (s *PostingRequests) Add(p PostingRequested) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if s.seen == nil {
		s.seen = map[string]bool{}
	}
	if s.seen[p.RequestKey] {
		return ErrDuplicatePosting
	}
	s.seen[p.RequestKey] = true
	return nil
}
