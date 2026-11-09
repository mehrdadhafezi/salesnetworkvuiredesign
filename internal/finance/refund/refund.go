// Package refund is the R4 refund STATE MACHINE contract (Gate-0 INV-015;
// Cross-Role §9 "Refund": request / report / proof / financial execution
// are distinct; the original invoice is preserved). It models and checks
// refund transitions and projects an append-only refund history. It
// executes nothing, has no route, no store and no refund authority: OPD-04
// (refund object, proof and authority) is OPEN, so every transition is
// BLOCKED even when it is structurally legal.
//
// Never inferred:
//   - an invoice cancellation is not money returned;
//   - an HTTP success from a bank or provider is not a refund;
//   - a refund never edits or deletes the original invoice, payment or
//     ledger records; compensation is its own approved transaction.
package refund

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// State is a refund state. Every state is distinct; none implies another.
type State string

const (
	Requested          State = "REQUESTED"
	Reviewed           State = "REVIEWED"
	Authorized         State = "AUTHORIZED"
	ExternallyExecuted State = "EXTERNALLY_EXECUTED"
	EvidenceReceived   State = "EVIDENCE_RECEIVED"
	Reconciled         State = "RECONCILED"
	Completed          State = "COMPLETED"
	Rejected           State = "REJECTED"
	Unknown            State = "UNKNOWN"
)

// States lists every state.
func States() []State {
	return []State{Requested, Reviewed, Authorized, ExternallyExecuted, EvidenceReceived, Reconciled, Completed, Rejected, Unknown}
}

// transitions is the structural refund graph. UNKNOWN (an external
// outcome that cannot be determined) is resolved only by evidence; there
// is no path from UNKNOWN back to AUTHORIZED (no blind re-execution).
var transitions = map[State][]State{
	Requested:          {Reviewed, Rejected},
	Reviewed:           {Authorized, Rejected},
	Authorized:         {ExternallyExecuted, Unknown},
	ExternallyExecuted: {EvidenceReceived, Unknown},
	Unknown:            {EvidenceReceived},
	EvidenceReceived:   {Reconciled},
	Reconciled:         {Completed},
}

// Terminal reports a state with no outgoing transition.
func Terminal(s State) bool { return s == Completed || s == Rejected }

// Legal reports whether from -> to is a structural transition.
func Legal(from, to State) bool { return slices.Contains(transitions[from], to) }

// EvidenceKind is the kind of evidence offered for a transition.
type EvidenceKind string

const (
	// EvidenceBankStatement / EvidenceProviderConfirmation are external
	// proofs of a money movement (with a reference and a document digest).
	EvidenceBankStatement        EvidenceKind = "BANK_STATEMENT"
	EvidenceProviderConfirmation EvidenceKind = "PROVIDER_CONFIRMATION"
	// EvidenceHTTPSuccess is an HTTP 2xx from a bank or provider API. It is
	// NEVER proof that money moved.
	EvidenceHTTPSuccess EvidenceKind = "HTTP_SUCCESS"
	// EvidenceInvoiceCancelled is an invoice cancellation. It is NEVER
	// proof that money was returned.
	EvidenceInvoiceCancelled EvidenceKind = "INVOICE_CANCELLED"
	// EvidenceReconciliation is a reconciliation record matching the
	// external proof to the refund.
	EvidenceReconciliation EvidenceKind = "RECONCILIATION_RECORD"
	// EvidenceNone: no evidence offered.
	EvidenceNone EvidenceKind = ""
)

// Evidence is what a transition offers.
type Evidence struct {
	Kind EvidenceKind
	// ExternalRef is the bank / provider reference of the movement.
	ExternalRef string
	// DocumentSHA256 is the digest of the stored proof document.
	DocumentSHA256 string
	// Amount is the amount the proof shows (exact, explicit unit).
	Amount *money.Money
}

// PersonLink is identity evidence about whether the actor and the other
// participants of the refund (requester, reviewer) are different HUMANS.
// Distinct account ids alone never prove that: one person may hold several
// accounts. The zero value means no person-level evidence exists.
type PersonLink string

const (
	// PersonDistinct: authoritative evidence proves different people.
	PersonDistinct PersonLink = "DISTINCT"
	// PersonSameHuman: evidence links the accounts to one person.
	PersonSameHuman PersonLink = "SAME_HUMAN"
)

// Actor is the server-derived actor of a transition.
type Actor struct {
	UserID   string
	Verified bool
	// Independence is the person-level evidence for the separation of
	// duties of this transition (review, authorization).
	Independence PersonLink
}

// Refund is the refund aggregate as a transition sees it. Original*
// identify the preserved invoice and payment; they are never mutated.
type Refund struct {
	RefundID          string
	OriginalInvoiceID string
	OriginalPaymentID string
	State             State
	Amount            money.Money
	// OriginalPaid is the proven paid amount of the original payment and
	// PreviouslyRefunded the proven sum already refunded (same unit).
	OriginalPaid       money.Money
	PreviouslyRefunded money.Money
	RequestedBy        string
	ReviewedBy         string
}

// Decision ids.
const (
	DecisionOPD04 = "OPD-04"
	// PersonLinkUnmodelled: no authoritative person linkage exists, so the
	// separation of duties is proven only at the account level.
	PersonLinkUnmodelled   = "IDENTITY_PERSON_LINK_UNMODELLED"
	NoRefundAuthority      = "NO_REFUND_AUTHORITY"
	CompensationUnapproved = "FINANCIAL_COMPENSATION_UNAPPROVED"
)

// Evaluation is the answer for one transition.
type Evaluation struct {
	From, To  State
	Legal     bool
	Status    result.Status
	Reasons   []reasoncodes.Code
	BlockedBy []string
	Findings  []string
}

var sha = regexp.MustCompile(`^[0-9a-f]{64}$`)

// EvaluateTransition checks one proposed transition. It is BLOCKED by
// OPD-04 whenever it is structurally sound; otherwise it reports why not.
func EvaluateTransition(r Refund, to State, ev Evidence, actor Actor) Evaluation {
	e := Evaluation{From: r.State, To: to, Legal: Legal(r.State, to)}
	reasons := map[reasoncodes.Code]bool{}
	unauthorized, conflict, fail := false, false, false
	note := func(c reasoncodes.Code, f string) { reasons[c] = true; e.Findings = append(e.Findings, f) }
	if !e.Legal {
		conflict = true
		note(reasoncodes.StaleState, "not a refund transition: "+string(r.State)+" -> "+string(to))
	}
	if !actor.Verified || actor.UserID == "" {
		unauthorized = true
		note(reasoncodes.ActionDenied, "actor is not a verified authenticated user")
	}
	if r.OriginalInvoiceID == "" || r.OriginalPaymentID == "" {
		fail = true
		note(reasoncodes.ContractInvalid, "a refund must reference the preserved original invoice and payment")
	}
	// Exact money. Every fact is validated here, never assumed: the refund
	// amount is positive; the original paid and previously refunded amounts
	// are valid and nonnegative; all three share the refund's explicit unit;
	// previously refunded never exceeds original paid; the amount never
	// exceeds the remaining refundable amount. Invalid facts fail closed.
	amountOK := r.Amount.Validate() == nil && r.Amount.Sign() > 0
	paidOK := r.OriginalPaid.Validate() == nil && r.OriginalPaid.Sign() >= 0
	prevOK := r.PreviouslyRefunded.Validate() == nil && r.PreviouslyRefunded.Sign() >= 0
	if !amountOK {
		fail = true
		note(reasoncodes.ContractInvalid, "the refund amount must be positive in an explicit unit")
	}
	if !paidOK {
		fail = true
		note(reasoncodes.ContractInvalid, "the original paid amount must be a valid nonnegative amount in an explicit unit")
	}
	if !prevOK {
		fail = true
		note(reasoncodes.ContractInvalid, "the previously refunded amount must be a valid nonnegative amount in an explicit unit")
	}
	if amountOK && paidOK && prevOK {
		if r.OriginalPaid.Unit != r.Amount.Unit || r.PreviouslyRefunded.Unit != r.Amount.Unit {
			fail = true
			note(reasoncodes.ContractInvalid, "original paid, previously refunded and the refund must share one explicit unit (no conversion)")
		} else if c, err := money.Compare(r.PreviouslyRefunded, r.OriginalPaid); err != nil || c > 0 {
			conflict = true
			note(reasoncodes.LinkageConflict, "previously refunded exceeds the original paid amount")
		} else if remaining, err := money.Sub(r.OriginalPaid, r.PreviouslyRefunded); err != nil {
			fail = true
			note(reasoncodes.ContractInvalid, "the remaining refundable amount cannot be computed exactly")
		} else if c, err := money.Compare(r.Amount, remaining); err != nil || c > 0 {
			conflict = true
			note(reasoncodes.DependencyFinancialActive, "the refund exceeds the remaining refundable amount")
		}
	}
	switch ev.Kind {
	case EvidenceInvoiceCancelled:
		fail = true
		note(reasoncodes.ContractInvalid, "an invoice cancellation is never evidence that money was returned")
	case EvidenceHTTPSuccess:
		fail = true
		note(reasoncodes.ContractInvalid, "an HTTP success is never evidence that a refund was executed")
	}
	var blockedBy []string
	if to == Reviewed || to == Authorized {
		// Account inequality (below) is necessary, not sufficient: the
		// separation of duties needs person-level evidence.
		switch actor.Independence {
		case PersonDistinct:
		case PersonSameHuman:
			unauthorized = true
			note(reasoncodes.ActionDenied, "the actor is the same person as another participant (several accounts)")
		default:
			blockedBy = append(blockedBy, PersonLinkUnmodelled)
			note(reasoncodes.CoverageIncomplete, "independence is proven only at the account level (no person linkage)")
		}
	}
	switch to {
	case Reviewed:
		if actor.UserID != "" && actor.UserID == r.RequestedBy {
			unauthorized = true
			note(reasoncodes.ActionDenied, "the requester never reviews the same refund")
		}
	case Authorized:
		if actor.UserID != "" && (actor.UserID == r.RequestedBy || actor.UserID == r.ReviewedBy) {
			unauthorized = true
			note(reasoncodes.ActionDenied, "the authorizer is independent of requester and reviewer")
		}
	case ExternallyExecuted, EvidenceReceived:
		if ev.Kind != EvidenceBankStatement && ev.Kind != EvidenceProviderConfirmation || ev.ExternalRef == "" || !sha.MatchString(ev.DocumentSHA256) {
			fail = true
			note(reasoncodes.ContractInvalid, "execution needs an external proof (reference + document digest)")
		}
		if ev.Amount == nil || !money.Equal(*ev.Amount, r.Amount) {
			conflict = true
			note(reasoncodes.LinkageConflict, "the proven amount does not equal the refund amount exactly")
		}
	case Reconciled:
		if ev.Kind != EvidenceReconciliation || !sha.MatchString(ev.DocumentSHA256) {
			fail = true
			note(reasoncodes.ContractInvalid, "reconciliation needs a reconciliation record")
		}
	case Unknown:
		reasons[reasoncodes.OutcomeUnknown] = true
	}
	reasons[reasoncodes.PolicyUnresolved] = true
	e.BlockedBy = append([]string{DecisionOPD04, NoRefundAuthority, CompensationUnapproved}, blockedBy...)
	switch {
	case unauthorized:
		e.Status = result.Unauthorized
	case conflict:
		e.Status = result.Conflict
	case fail:
		e.Status = result.Failed
	default:
		e.Status = result.Blocked
	}
	for _, c := range reasoncodes.Codes() {
		if reasons[c] {
			e.Reasons = append(e.Reasons, c)
		}
	}
	return e
}

// Event is one append-only refund history event.
type Event struct {
	Seq      int64
	RefundID string
	To       State
}

// Projection is the derived read of a refund history.
type Projection struct {
	RefundID string
	// State is the current state; it is set ONLY when the whole history is
	// valid. An invalid history never yields a current state.
	State State
	Valid bool
	// VerifiedState / VerifiedThrough describe the longest valid prefix
	// (diagnosis only; never a current state).
	VerifiedState   State
	VerifiedThrough int64
	Findings        []string
}

// Project derives the current state from a refund's append-only history.
//
// Sequence contract (an engineering contract of the event log, not a
// business policy): the events of ONE refund carry a per-refund sequence
// that starts at 1 and increases by exactly 1 per event (1..n), like the
// per-aggregate versions every target aggregate already uses. A strictly
// increasing but gapped sequence is NOT accepted: a gap cannot be told
// apart from a lost or hidden event, so it fails closed.
//
// Delivery order does not matter (events are ordered by sequence). Any
// empty history, foreign event, sequence below 1, duplicate, gap, start
// other than REQUESTED or illegal transition makes the history invalid:
// nothing is skipped, repaired or manufactured, and no current state is
// returned. The input slice is never modified.
func Project(refundID string, events []Event) Projection {
	p := Projection{RefundID: refundID}
	if len(events) == 0 {
		p.Findings = append(p.Findings, "no history")
		return p
	}
	evs := slices.Clone(events)
	slices.SortStableFunc(evs, func(a, b Event) int {
		switch {
		case a.Seq < b.Seq:
			return -1
		case a.Seq > b.Seq:
			return 1
		}
		return 0
	})
	broken := false
	for n, e := range evs {
		want := int64(n + 1)
		switch {
		case e.RefundID != refundID:
			p.Findings = append(p.Findings, fmt.Sprintf("seq %d: event of another refund", e.Seq))
		case e.Seq < 1:
			p.Findings = append(p.Findings, fmt.Sprintf("seq %d: sequence below 1", e.Seq))
		case n > 0 && e.Seq == evs[n-1].Seq:
			p.Findings = append(p.Findings, fmt.Sprintf("seq %d: duplicate sequence", e.Seq))
		case e.Seq != want:
			p.Findings = append(p.Findings, fmt.Sprintf("seq %d: expected %d (missing or out-of-contract sequence)", e.Seq, want))
		case n == 0 && e.To != Requested:
			p.Findings = append(p.Findings, "history does not start with REQUESTED")
		case n > 0 && !broken && !Legal(p.VerifiedState, e.To):
			p.Findings = append(p.Findings, fmt.Sprintf("seq %d: illegal step %s -> %s", e.Seq, p.VerifiedState, e.To))
		default:
			if !broken {
				p.VerifiedState, p.VerifiedThrough = e.To, e.Seq
			}
			continue
		}
		broken = true
	}
	if !broken {
		p.Valid, p.State = true, p.VerifiedState
	}
	return p
}
