package api

import (
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// R4: the shared evaluation shape of the sales inventory gates (invoice
// claim in cases, invoice issuance in billing, return / reset / release in
// custody). Each gate is a pure evaluator: it executes nothing, and while
// ADR-01 / F01 / OPD-03 are unresolved it can never answer SUCCESS. When
// every precondition holds it answers BLOCKED with the exact decisions.

// Sales-gate decision ids.
const (
	DecisionADR01 = "ADR-01"
	DecisionD28   = "D-28"
)

// IdentityResolution is how the customer of a case is known. A customer
// inferred only from a similarity (contact, name, time) is never an
// identity (Gate-0 phone invariants): it is DISCOVERY_ONLY.
type IdentityResolution string

const (
	IdentityResolved      IdentityResolution = "RESOLVED"
	IdentityDiscoveryOnly IdentityResolution = "DISCOVERY_ONLY"
	IdentityConflict      IdentityResolution = "CONFLICT"
	IdentityUnknown       IdentityResolution = "UNKNOWN"
)

// Lineage is how the case's source lineage is known. Ambiguous or unknown
// lineage stays UNKNOWN; it is never guessed.
type Lineage string

const (
	LineageKnown     Lineage = "KNOWN"
	LineageAmbiguous Lineage = "AMBIGUOUS"
	LineageConflict  Lineage = "CONFLICT"
	LineageUnknown   Lineage = "UNKNOWN"
)

// GateEvaluation is one gate's answer. Status precedence, most specific
// first: UNAUTHORIZED (actor / row scope), CONFLICT (fresh facts contradict
// the intent), UNKNOWN (facts cannot be established), FAILED (the request
// itself is invalid), BLOCKED (policy).
type GateEvaluation struct {
	Action string
	Status result.Status
	// PreconditionsMet is true when only decision blockers remain: the
	// gate would still never execute.
	PreconditionsMet bool
	Reasons          []reasoncodes.Code
	BlockedBy        []string
	Findings         []string
}

// GateBuilder accumulates one gate's findings.
type GateBuilder struct {
	action                                string
	unauthorized, conflict, unknown, fail bool
	precondition                          bool
	reasons                               map[reasoncodes.Code]bool
	blockedBy                             []string
	findings                              []string
}

// NewGate starts the evaluation of action.
func NewGate(action string) *GateBuilder {
	return &GateBuilder{action: action, reasons: map[reasoncodes.Code]bool{}}
}

func (g *GateBuilder) note(code reasoncodes.Code, finding string) {
	g.reasons[code] = true
	g.findings = append(g.findings, finding)
	g.precondition = true
}

// Deny records an actor / row-scope denial.
func (g *GateBuilder) Deny(code reasoncodes.Code, finding string) {
	g.unauthorized = true
	g.note(code, finding)
}

// Conflict records fresh facts contradicting the intent.
func (g *GateBuilder) Conflict(code reasoncodes.Code, finding string) {
	g.conflict = true
	g.note(code, finding)
}

// Unknown records facts that cannot be established.
func (g *GateBuilder) Unknown(code reasoncodes.Code, finding string) {
	g.unknown = true
	g.note(code, finding)
}

// Fail records an invalid request.
func (g *GateBuilder) Fail(code reasoncodes.Code, finding string) {
	g.fail = true
	g.note(code, finding)
}

// Precondition records a known precondition barrier (BLOCKED).
func (g *GateBuilder) Precondition(code reasoncodes.Code, finding string, blockedBy ...string) {
	g.note(code, finding)
	g.blockedBy = append(g.blockedBy, blockedBy...)
}

// Dependencies merges a dependency verdict: anything other than
// NO_FINANCIAL_DEPENDENCY is a precondition finding with the verdict's
// status, reasons and decisions.
func (g *GateBuilder) Dependencies(v DependencyVerdict) {
	if v.Status == result.Success {
		return
	}
	for _, r := range v.Reasons {
		g.reasons[r] = true
	}
	g.blockedBy = append(g.blockedBy, v.BlockedBy...)
	g.findings = append(g.findings, "financial dependency: "+string(v.Outcome))
	g.precondition = true
	switch v.Status {
	case result.Conflict:
		g.conflict = true
	case result.Unknown:
		g.unknown = true
	}
}

// Identity applies the customer-identity and lineage rules.
func (g *GateBuilder) Identity(id IdentityResolution, lin Lineage) {
	switch id {
	case IdentityResolved:
	case IdentityConflict:
		g.Conflict(reasoncodes.LinkageConflict, "customer identity is contradicted by its sources")
	case IdentityDiscoveryOnly:
		g.Unknown(reasoncodes.DependencyUnknown, "customer known only from a similarity (discovery only, never identity)")
	default:
		g.Unknown(reasoncodes.DependencyUnknown, "customer identity unknown")
	}
	switch lin {
	case LineageKnown:
	case LineageConflict:
		g.Conflict(reasoncodes.LinkageConflict, "case lineage is contradictory")
	default:
		g.Unknown(reasoncodes.DependencyUnknown, "case lineage is ambiguous or unknown (stays UNKNOWN)")
	}
}

// Finish closes the evaluation. policy lists the decisions that block the
// action even when every precondition holds; it is never empty for an
// R4 sales gate.
func (g *GateBuilder) Finish(policy ...string) GateEvaluation {
	e := GateEvaluation{Action: g.action, PreconditionsMet: !g.precondition, Findings: g.findings}
	switch {
	case g.unauthorized:
		e.Status = result.Unauthorized
	case g.conflict:
		e.Status = result.Conflict
	case g.unknown:
		e.Status = result.Unknown
	case g.fail:
		e.Status = result.Failed
	default:
		e.Status = result.Blocked
	}
	if len(policy) > 0 {
		g.reasons[reasoncodes.PolicyUnresolved] = true
	}
	for _, c := range reasoncodes.Codes() {
		if g.reasons[c] {
			e.Reasons = append(e.Reasons, c)
		}
	}
	seen := map[string]bool{}
	for _, d := range append(g.blockedBy, policy...) {
		if !seen[d] {
			seen[d] = true
			e.BlockedBy = append(e.BlockedBy, d)
		}
	}
	return e
}
