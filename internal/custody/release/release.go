// Package release is the R4 evaluator of the custody return / reset /
// release family (F01: the all-link protected return contract). It is a
// pure evaluator: no SQL, no command, no route. ADR-01 and F01 are not
// approved and reset / release also need OPD-03, so the answer is never
// SUCCESS. A return never erases history: IDs, links, attribution and
// custody events stay as they are whatever the outcome.
package release

import (
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
)

// Kind is the evaluated inventory operation.
type Kind string

const (
	KindReturn  Kind = "RETURN"
	KindReset   Kind = "RESET"
	KindRelease Kind = "RELEASE"
)

// Input is everything the return gate considers.
type Input struct {
	Kind            Kind
	ActorUserID     string
	ActorVerified   bool
	HolderUserID    *string
	ExpectedVersion int64
	CurrentVersion  int64
	// ActiveClaim: an invoice claim is open on the case (issuance in
	// flight); returning now would race the issuance (INV-009).
	ActiveClaim  bool
	Dependencies casesapi.DependencyVerdict
	Lineage      casesapi.Lineage
}

// Evaluate returns the gate of one return / reset / release.
func Evaluate(in Input) casesapi.GateEvaluation {
	g := casesapi.NewGate("custody." + string(in.Kind))
	switch {
	case in.Kind != KindReturn && in.Kind != KindReset && in.Kind != KindRelease:
		g.Fail(reasoncodes.ContractInvalid, "unknown inventory operation")
	case !in.ActorVerified || in.ActorUserID == "":
		g.Deny(reasoncodes.ActionDenied, "actor is not a verified authenticated user")
	case in.HolderUserID == nil:
		g.Unknown(reasoncodes.DependencyUnknown, "current holder cannot be established")
	case *in.HolderUserID != in.ActorUserID:
		g.Deny(reasoncodes.RowScopeDenied, "only the current holder may return the case (no rank shortcut)")
	}
	if in.ExpectedVersion != in.CurrentVersion {
		g.Conflict(reasoncodes.StaleState, "case version changed (stale aggregate)")
	}
	if in.ActiveClaim {
		g.Conflict(reasoncodes.DependencyFinancialActive, "an invoice claim is active: the return would race the issuance (INV-009)")
	}
	g.Dependencies(in.Dependencies)
	switch in.Lineage {
	case casesapi.LineageKnown:
	case casesapi.LineageConflict:
		g.Conflict(reasoncodes.LinkageConflict, "case lineage is contradictory")
	default:
		g.Unknown(reasoncodes.DependencyUnknown, "case lineage is ambiguous or unknown (stays UNKNOWN)")
	}
	policy := []string{casesapi.DecisionADR01, casesapi.DecisionF01}
	if in.Kind != KindReturn {
		policy = append(policy, casesapi.DecisionOPD03)
	}
	return g.Finish(policy...)
}
