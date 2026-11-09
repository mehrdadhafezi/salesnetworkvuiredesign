// Package claimgate is the R4 evaluator of the invoice claim (the cases
// half of the ADR-01 CANDIDATE reservation protocol: a seller reserves a
// case for invoice issuance in the cases module's own transaction). It is a
// pure evaluator: no SQL, no command, no route. ADR-01 is not approved, so
// the answer is never SUCCESS; when every precondition holds it is BLOCKED
// by ADR-01.
package claimgate

import (
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
)

// Action is the evaluated action.
const Action = "cases.invoice_claim"

// ActiveClaim is an existing unreleased claim on the case.
type ActiveClaim struct {
	ClaimID      string
	SellerUserID string
}

// Input is everything the claim gate considers. The caller supplies fresh
// facts read in the cases transaction that would hold the case row lock.
type Input struct {
	CaseID      string
	CasePresent bool
	// ActorUserID is the authenticated actor; ActorVerified states that it
	// came from a verified session (never a client-supplied id).
	ActorUserID   string
	ActorVerified bool
	// HolderUserID is the current custody holder (custody published read);
	// nil when it cannot be established.
	HolderUserID    *string
	ExpectedVersion int64
	CurrentVersion  int64
	ActiveClaim     *ActiveClaim
	// ReturnInProgress: a return / reset / release reservation is open.
	ReturnInProgress bool
	// Dependencies is the union-of-links verdict of the case.
	Dependencies casesapi.DependencyVerdict
	Customer     casesapi.IdentityResolution
	Lineage      casesapi.Lineage
}

// Evaluate returns the claim gate.
func Evaluate(in Input) casesapi.GateEvaluation {
	g := casesapi.NewGate(Action)
	switch {
	case !in.ActorVerified || in.ActorUserID == "":
		g.Deny(reasoncodes.ActionDenied, "actor is not a verified authenticated user")
	case in.HolderUserID == nil:
		g.Unknown(reasoncodes.DependencyUnknown, "current holder cannot be established")
	case *in.HolderUserID != in.ActorUserID:
		g.Deny(reasoncodes.RowScopeDenied, "only the current holder may claim the case (no rank shortcut, OPD-10)")
	}
	if !in.CasePresent {
		g.Unknown(reasoncodes.DependencyUnknown, "case is not present in target evidence")
	}
	if in.ExpectedVersion != in.CurrentVersion {
		g.Conflict(reasoncodes.StaleState, "case version changed (stale aggregate)")
	}
	if in.ActiveClaim != nil {
		g.Conflict(reasoncodes.LinkageConflict, "the case already carries an active invoice claim")
	}
	if in.ReturnInProgress {
		g.Conflict(reasoncodes.StaleState, "a return / reset / release is in progress on the case")
	}
	g.Dependencies(in.Dependencies)
	g.Identity(in.Customer, in.Lineage)
	return g.Finish(casesapi.DecisionADR01)
}
