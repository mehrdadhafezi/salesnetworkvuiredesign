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

// Actor is the server-derived actor of a transition.
type Actor struct {
	UserID   string
	Verified bool
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
	DecisionOPD04          = "OPD-04"
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
	// Exact money: positive, one unit, never above what is refundable.
	if err := r.Amount.Validate(); err != nil || r.Amount.Sign() <= 0 {
		fail = true
		note(reasoncodes.ContractInvalid, "the refund amount must be positive in an explicit unit")
	} else if refundable, err := money.Sub(r.OriginalPaid, r.PreviouslyRefunded); err != nil {
		fail = true
		note(reasoncodes.ContractInvalid, "refundable amount cannot be computed exactly (unit mismatch or overflow)")
	} else if c, err := money.Compare(r.Amount, refundable); err != nil || c > 0 {
		conflict = true
		note(reasoncodes.DependencyFinancialActive, "the refund exceeds the proven refundable amount")
	}
	switch ev.Kind {
	case EvidenceInvoiceCancelled:
		fail = true
		note(reasoncodes.ContractInvalid, "an invoice cancellation is never evidence that money was returned")
	case EvidenceHTTPSuccess:
		fail = true
		note(reasoncodes.ContractInvalid, "an HTTP success is never evidence that a refund was executed")
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
	e.BlockedBy = []string{DecisionOPD04, NoRefundAuthority, CompensationUnapproved}
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
	State    State
	// Valid is false when the history contains a non-transition; the
	// projection then stops at the last valid state and reports it.
	Valid    bool
	Findings []string
}

// Project derives the current state from the ordered history. It never
// repairs or skips: an illegal step is a finding and the state stays at
// the last valid step. The first event must be REQUESTED.
func Project(refundID string, events []Event) Projection {
	p := Projection{RefundID: refundID, Valid: true}
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
	for n, e := range evs {
		if e.RefundID != refundID {
			p.Valid = false
			p.Findings = append(p.Findings, "event of another refund")
			continue
		}
		if n == 0 {
			if e.To != Requested {
				p.Valid = false
				p.Findings = append(p.Findings, "history does not start with REQUESTED")
				return p
			}
			p.State = Requested
			continue
		}
		if !Legal(p.State, e.To) {
			p.Valid = false
			p.Findings = append(p.Findings, "illegal step "+string(p.State)+" -> "+string(e.To))
			return p
		}
		p.State = e.To
	}
	return p
}
