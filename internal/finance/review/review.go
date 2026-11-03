// Package review is the Finance review gate of Slice 12 (Architecture V1
// §15.6, §16, §17). It is an evaluator over the payments published review
// read; it has no command. Finance approve / reject stay BLOCKED in Slice
// 12: approval posts to the ledger in the same finance transaction (§16)
// and the posting basis (D-41), the approval policy and maker/checker rules
// (OPD-08) and entitlement routing (OPD-09) are unresolved; the billing
// transition for a rejection is not defined by any authority.
//
// The gate still evaluates every safety rule so that, when a decision is
// approved by the owner, the conditions it must honour are already
// enforced and visible:
//   - D-43 safe default: no decision while any required receipt is not READY;
//   - maker ≠ checker (fail closed): the submitter of an attempt is never an
//     independent reviewer, whatever roles or capabilities it holds;
//   - a decision needs a decision capability; Slice 12 defines none.
package review

import (
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// Review actions.
const (
	ActionApprove = "finance.attempt.approve"
	ActionReject  = "finance.attempt.reject"
)

// Decision and policy ids.
const (
	DecisionD41                = "D-41"
	DecisionD43                = "D-43"
	DecisionOPD08              = "OPD-08"
	DecisionOPD09              = "OPD-09"
	RejectionContractUndefined = "BILLING_REJECTION_TRANSITION_UNDEFINED"
	NoDecisionCapability       = "NO_DECISION_CAPABILITY"
)

// Blocked is one action's evaluation. Every applicable reason is listed.
type Blocked struct {
	Action    string
	Status    result.Status
	Reasons   []reasoncodes.Code
	BlockedBy []string
}

// Gate is the evaluation of one attempt for one reviewer.
type Gate struct {
	EvidenceReady   bool
	ReviewerIsMaker bool
	Actions         []Blocked
}

// EvidenceReady reports the D-43 condition: the attempt has at least one
// receipt and every receipt is READY.
func EvidenceReady(a paymentsapi.ReviewAttempt) bool {
	if a.EvidenceState != paymentsapi.EvidenceReady || len(a.Receipts) == 0 {
		return false
	}
	for _, r := range a.Receipts {
		if r.State != paymentsapi.ReceiptReady {
			return false
		}
	}
	return true
}

// Evaluate returns the gate of attempt a for reviewerUserID. No action is
// ever allowed in Slice 12; the result says every reason why.
func Evaluate(a paymentsapi.ReviewAttempt, reviewerUserID string) Gate {
	g := Gate{EvidenceReady: EvidenceReady(a), ReviewerIsMaker: reviewerUserID == "" || reviewerUserID == a.SubmittedByUserID}
	for _, action := range []string{ActionApprove, ActionReject} {
		b := Blocked{Action: action, Status: result.Blocked}
		if g.ReviewerIsMaker {
			b.Status = result.Unauthorized
			b.Reasons = append(b.Reasons, reasoncodes.ActionDenied)
			b.BlockedBy = append(b.BlockedBy, DecisionOPD08)
		}
		if !g.EvidenceReady {
			b.Reasons = append(b.Reasons, reasoncodes.PolicyUnresolved)
			b.BlockedBy = append(b.BlockedBy, DecisionD43)
		}
		if a.State != paymentsapi.AttemptPendingReview {
			b.Reasons = append(b.Reasons, reasoncodes.StaleState)
		}
		switch action {
		case ActionApprove:
			b.BlockedBy = append(b.BlockedBy, DecisionD41, DecisionOPD08, DecisionOPD09)
		case ActionReject:
			b.BlockedBy = append(b.BlockedBy, DecisionOPD08, RejectionContractUndefined)
		}
		b.BlockedBy = append(b.BlockedBy, NoDecisionCapability)
		b.Reasons = append(b.Reasons, reasoncodes.PolicyUnresolved)
		b.Reasons, b.BlockedBy = dedupe(b.Reasons), dedupeS(b.BlockedBy)
		g.Actions = append(g.Actions, b)
	}
	return g
}

func dedupe(in []reasoncodes.Code) []reasoncodes.Code {
	seen := map[reasoncodes.Code]bool{}
	var out []reasoncodes.Code
	for _, c := range in {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

func dedupeS(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range in {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}
