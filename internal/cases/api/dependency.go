package api

import (
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// R4: the shared financial-dependency contract of Cross-Role Architecture V1
// §8 ("Consumed is a dependency guard, not a sold status"). Every inventory
// writer (return, reset, release, reassignment, invoice claim) must consult
// the UNION of all valid links of a case; a lookup of the last invoice, or a
// role-specific easier predicate, is never a substitute.
//
// This is a pure, policy-independent classifier: it decides nothing about
// who may act (actor / row / scope / handoff / reset policy) and authorises
// no write. Every command that consumes a verdict stays decision-gated
// (ADR-01, F01, OPD-03, OPD-04).

// LinkSource is the proof source of one case-to-invoice link, in the proof
// hierarchy of §8. Similarity (name / contact / time) is discovery only and
// never proves or disproves a link; a compatibility id counts only when it
// was validated.
type LinkSource string

const (
	LinkLegacyInvoiceLead   LinkSource = "LEGACY_INVOICE_LEAD"  // the invoice's own case reference
	LinkFollowInvoice       LinkSource = "FOLLOW_INVOICE"       // follow-up invoice reference
	LinkFlowEventRef        LinkSource = "FLOW_EVENT_REF"       // flow state / event invoice refs with source context
	LinkDotRef              LinkSource = "DOT_REF"              // Dot case / payment refs
	LinkCompatibilityID     LinkSource = "COMPATIBILITY_ID"     // validated fallback only
	LinkDiscoverySimilarity LinkSource = "DISCOVERY_SIMILARITY" // discovery only: never proof
)

// LinkResolution is how far one link was resolved to its target.
type LinkResolution string

const (
	LinkResolved      LinkResolution = "RESOLVED"
	LinkMissingTarget LinkResolution = "MISSING_TARGET" // the referenced invoice is not present
	LinkConflicting   LinkResolution = "CONFLICTING"    // sources disagree about the target
	LinkUnresolved    LinkResolution = "UNRESOLVED"     // not (yet) resolvable
)

// InvoiceClass is the financial class of a resolved link's invoice, per the
// §8 dependency table. Mapping a raw billing status to a class is the
// billing owner's published read; an unmapped status is ClassUnknown.
type InvoiceClass string

const (
	// ClassCommitted: draft / pre-invoice / pending / receipt / review /
	// rejected-correctable. Not yet collected, still a financial commitment.
	ClassCommitted InvoiceClass = "COMMITTED"
	// ClassPaidOrProtected: partial / full paid / completed, or a protected
	// downstream materialization (fulfillment). The end of a sale never
	// frees history.
	ClassPaidOrProtected InvoiceClass = "PAID_OR_PROTECTED"
	// ClassCancelled: cancelled unpaid / cancelled paid / refund-reported.
	// Automatic release is forbidden (OPD-03 / OPD-04 + reconciliation).
	ClassCancelled InvoiceClass = "CANCELLED"
	// ClassUnknown: the invoice state could not be classified.
	ClassUnknown InvoiceClass = "UNKNOWN"
)

// DependencyLink is one link between the case and an invoice (or another
// protected record).
type DependencyLink struct {
	Source     LinkSource
	InvoiceRef string
	Resolution LinkResolution
	Class      InvoiceClass
	// Validated marks a COMPATIBILITY_ID that was validated against an
	// authoritative source; an unvalidated compatibility id is unresolved.
	Validated bool
}

// DependencyOutcome is the ordinary-operation eligibility of §8.
type DependencyOutcome string

const (
	// NoFinancialDependency: no financial / materialized / fulfillment
	// dependency with complete resolution. The operation is STILL subject
	// to actor / row / scope / handoff / reset policy; this is not consent.
	NoFinancialDependency DependencyOutcome = "NO_FINANCIAL_DEPENDENCY"
	// BlockedFinancialCommitment: a committed (unpaid) invoice exists.
	BlockedFinancialCommitment DependencyOutcome = "BLOCKED_FINANCIAL_COMMITMENT"
	// BlockedProtectedDownstream: a paid / completed / protected record.
	BlockedProtectedDownstream DependencyOutcome = "BLOCKED_PROTECTED_DOWNSTREAM"
	// BlockedCancelledNeedsPolicy: a cancelled / refund-reported invoice;
	// release needs OPD-03 / OPD-04 and reconciliation.
	BlockedCancelledNeedsPolicy DependencyOutcome = "BLOCKED_CANCELLED_NEEDS_POLICY"
	// FailClosedLinkage: a missing target, a conflicting or unresolved link,
	// an unclassified invoice or incomplete coverage. Fail closed with a
	// named reconciliation owner; never labelled unused or orphan.
	FailClosedLinkage DependencyOutcome = "FAIL_CLOSED_LINKAGE"
)

// Decision ids named by a dependency verdict.
const (
	DecisionOPD03 = "OPD-03"
	DecisionOPD04 = "OPD-04"
	DecisionF01   = "F01"
)

// DependencyVerdict is the classification of the union of a case's links.
type DependencyVerdict struct {
	Outcome DependencyOutcome
	// Status is SUCCESS only for NoFinancialDependency; CONFLICT for a
	// conflicting link; UNKNOWN for missing / unresolved / unclassified
	// links or incomplete coverage; otherwise BLOCKED.
	Status    result.Status
	Reasons   []reasoncodes.Code
	BlockedBy []string
	// Outcomes lists every outcome that applies (the union), most
	// restrictive first; Outcome is Outcomes[0].
	Outcomes []DependencyOutcome
	// Considered is the number of proof links considered; Discovery the
	// number of similarity links that were deliberately NOT used as proof.
	Considered int
	Discovery  int
}

// severity orders outcomes, most restrictive first.
var severity = map[DependencyOutcome]int{
	FailClosedLinkage:           0,
	BlockedProtectedDownstream:  1,
	BlockedFinancialCommitment:  2,
	BlockedCancelledNeedsPolicy: 3,
	NoFinancialDependency:       4,
}

// ClassifyDependencies classifies the union of links. coverageComplete
// states that every proof source of §8 was consulted for this case: absent
// links prove nothing while coverage is incomplete (a non-match in one
// source never negates links in another).
func ClassifyDependencies(links []DependencyLink, coverageComplete bool) DependencyVerdict {
	v := DependencyVerdict{}
	set := map[DependencyOutcome]bool{}
	reasons := map[reasoncodes.Code]bool{}
	blocked := map[string]bool{}
	conflict, unknown := false, false
	if !coverageComplete {
		set[FailClosedLinkage] = true
		reasons[reasoncodes.CoverageIncomplete] = true
		blocked[DecisionF01] = true
		unknown = true
	}
	for _, l := range links {
		if l.Source == LinkDiscoverySimilarity {
			v.Discovery++
			continue
		}
		v.Considered++
		resolution := l.Resolution
		if l.Source == LinkCompatibilityID && !l.Validated && resolution == LinkResolved {
			resolution = LinkUnresolved
		}
		switch resolution {
		case LinkResolved:
		case LinkConflicting:
			set[FailClosedLinkage], reasons[reasoncodes.LinkageConflict], blocked[DecisionF01], conflict = true, true, true, true
			continue
		default: // missing target, unresolved, or an unknown resolution value
			set[FailClosedLinkage], reasons[reasoncodes.DependencyUnknown], blocked[DecisionF01], unknown = true, true, true, true
			continue
		}
		switch l.Class {
		case ClassCommitted:
			set[BlockedFinancialCommitment], reasons[reasoncodes.DependencyFinancialActive] = true, true
		case ClassPaidOrProtected:
			set[BlockedProtectedDownstream], reasons[reasoncodes.DependencyProtectedDownstream] = true, true
		case ClassCancelled:
			set[BlockedCancelledNeedsPolicy], reasons[reasoncodes.PolicyUnresolved] = true, true
			blocked[DecisionOPD03], blocked[DecisionOPD04] = true, true
		default:
			set[FailClosedLinkage], reasons[reasoncodes.DependencyUnknown], blocked[DecisionF01], unknown = true, true, true, true
		}
	}
	if len(set) == 0 {
		set[NoFinancialDependency] = true
	}
	for o := range set {
		v.Outcomes = append(v.Outcomes, o)
	}
	sort.Slice(v.Outcomes, func(i, j int) bool { return severity[v.Outcomes[i]] < severity[v.Outcomes[j]] })
	v.Outcome = v.Outcomes[0]
	switch {
	case v.Outcome == NoFinancialDependency:
		v.Status = result.Success
	case conflict:
		v.Status = result.Conflict
	case unknown:
		v.Status = result.Unknown
	default:
		v.Status = result.Blocked
	}
	for _, c := range reasoncodes.Codes() {
		if reasons[c] {
			v.Reasons = append(v.Reasons, c)
		}
	}
	for _, d := range []string{DecisionF01, DecisionOPD03, DecisionOPD04} {
		if blocked[d] {
			v.BlockedBy = append(v.BlockedBy, d)
		}
	}
	return v
}
