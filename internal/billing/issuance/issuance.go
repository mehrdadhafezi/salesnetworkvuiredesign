// Package issuance is the R4 evaluator of invoice issuance (the billing
// half of the ADR-01 CANDIDATE reservation protocol: billing issues only
// against a fresh, active claim of the same seller and case, validated
// synchronously, and publishes InvoiceIssued to confirm the claim). It is a
// pure evaluator: no SQL, no command, no route, no invoice. ADR-01 is not
// approved, so the answer is never SUCCESS.
package issuance

import (
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
)

// Action is the evaluated action.
const Action = "billing.invoice_issue"

// Claim states of the candidate protocol.
const (
	ClaimActive                 = "ACTIVE"
	ClaimConfirmed              = "CONFIRMED"
	ClaimReconciliationRequired = "RECONCILIATION_REQUIRED"
)

// Claim is the claim the issuance relies on (cases published read).
type Claim struct {
	ClaimID      string
	CaseID       string
	SellerUserID string
	State        string
	Expired      bool
	CaseVersion  int64
}

// Idempotency is the command-record evidence for the request key.
type Idempotency struct {
	Key string
	// PriorRequestHash is the hash stored under Key, nil when unused.
	PriorRequestHash *string
	RequestHash      string
}

// Attribution compares the immutable snapshot with what the issuance
// would record. Issuance never changes original seller or credit owner.
type Attribution struct {
	SnapshotOriginalSeller, ProposedOriginalSeller *string
	SnapshotCreditOwner, ProposedCreditOwner       *string
}

// Input is everything the issuance gate considers.
type Input struct {
	CaseID              string
	ActorUserID         string
	ActorVerified       bool
	Claim               *Claim
	ExpectedCaseVersion int64
	// IssuedForClaim is the invoice already issued against the claim.
	IssuedForClaim *string
	Idempotency    Idempotency
	Attribution    Attribution
	Total          money.Money
	// AssessmentPartialPayment: the plan asks for a partial-payment
	// assessment invoice (D-28; 2.0.160 keeps the full-payment rule).
	AssessmentPartialPayment bool
	// Dependencies is the case's union-of-links verdict, excluding the
	// claim being issued.
	Dependencies casesapi.DependencyVerdict
	Customer     casesapi.IdentityResolution
	Lineage      casesapi.Lineage
}

func same(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

// Evaluate returns the issuance gate.
func Evaluate(in Input) casesapi.GateEvaluation {
	g := casesapi.NewGate(Action)
	if !in.ActorVerified || in.ActorUserID == "" {
		g.Deny(reasoncodes.ActionDenied, "actor is not a verified authenticated user")
	}
	switch {
	case in.Idempotency.Key == "":
		g.Fail(reasoncodes.ContractInvalid, "Idempotency-Key is required")
	case in.Idempotency.PriorRequestHash != nil && *in.Idempotency.PriorRequestHash != in.Idempotency.RequestHash:
		g.Conflict(reasoncodes.ContractInvalid, "idempotency key reused with a different request")
	}
	if c := in.Claim; c == nil {
		g.Precondition(reasoncodes.PolicyUnresolved, "no invoice claim (issuance requires a cases claim, ADR-01 candidate)", casesapi.DecisionADR01)
	} else {
		switch {
		case c.CaseID != in.CaseID:
			g.Conflict(reasoncodes.LinkageConflict, "the claim belongs to another case")
		case c.SellerUserID != in.ActorUserID:
			g.Deny(reasoncodes.RowScopeDenied, "only the claiming seller may issue against the claim")
		}
		switch {
		case c.State != ClaimActive:
			g.Conflict(reasoncodes.StaleState, "the claim is not ACTIVE ("+c.State+")")
		case c.Expired:
			g.Conflict(reasoncodes.StaleState, "the claim expired (it goes to reconciliation, never silent release)")
		}
		if c.CaseVersion != in.ExpectedCaseVersion {
			g.Conflict(reasoncodes.StaleState, "case version changed since the claim (stale aggregate)")
		}
	}
	if in.IssuedForClaim != nil {
		g.Conflict(reasoncodes.LinkageConflict, "an invoice was already issued against the claim (duplicate issuance)")
	}
	if !same(in.Attribution.SnapshotOriginalSeller, in.Attribution.ProposedOriginalSeller) ||
		!same(in.Attribution.SnapshotCreditOwner, in.Attribution.ProposedCreditOwner) {
		g.Conflict(reasoncodes.FieldDenied, "original seller / credit owner attribution is immutable")
	}
	if err := in.Total.Validate(); err != nil || in.Total.Sign() <= 0 {
		g.Fail(reasoncodes.ContractInvalid, "the invoice total must be a positive amount in an explicit unit")
	}
	if in.AssessmentPartialPayment {
		g.Precondition(reasoncodes.PolicyUnresolved, "partial-payment assessment invoice", casesapi.DecisionD28)
	}
	g.Dependencies(in.Dependencies)
	g.Identity(in.Customer, in.Lineage)
	return g.Finish(casesapi.DecisionADR01)
}
