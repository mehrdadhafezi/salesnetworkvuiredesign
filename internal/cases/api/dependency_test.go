package api

import (
	"slices"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func link(src LinkSource, res LinkResolution, class InvoiceClass) DependencyLink {
	return DependencyLink{Source: src, InvoiceRef: "inv", Resolution: res, Class: class}
}

// R4 (Cross-Role §8): the union of every valid link decides; one source's
// non-match never negates another; similarity is never proof; unresolved
// linkage fails closed.
func TestClassifyDependenciesUnionOfLinks(t *testing.T) {
	cases := []struct {
		name     string
		links    []DependencyLink
		complete bool
		outcome  DependencyOutcome
		status   result.Status
		reason   reasoncodes.Code
		blocked  []string
	}{
		{"no links, full coverage", nil, true, NoFinancialDependency, result.Success, "", nil},
		{"no links, incomplete coverage", nil, false, FailClosedLinkage, result.Unknown, reasoncodes.CoverageIncomplete, []string{DecisionF01}},
		{"similarity only is not proof", []DependencyLink{link(LinkDiscoverySimilarity, LinkResolved, ClassPaidOrProtected)}, true,
			NoFinancialDependency, result.Success, "", nil},
		{"committed invoice", []DependencyLink{link(LinkLegacyInvoiceLead, LinkResolved, ClassCommitted)}, true,
			BlockedFinancialCommitment, result.Blocked, reasoncodes.DependencyFinancialActive, nil},
		{"paid invoice", []DependencyLink{link(LinkFollowInvoice, LinkResolved, ClassPaidOrProtected)}, true,
			BlockedProtectedDownstream, result.Blocked, reasoncodes.DependencyProtectedDownstream, nil},
		{"cancelled invoice", []DependencyLink{link(LinkFlowEventRef, LinkResolved, ClassCancelled)}, true,
			BlockedCancelledNeedsPolicy, result.Blocked, reasoncodes.PolicyUnresolved, []string{DecisionOPD03, DecisionOPD04}},
		{"missing target", []DependencyLink{link(LinkDotRef, LinkMissingTarget, ClassCommitted)}, true,
			FailClosedLinkage, result.Unknown, reasoncodes.DependencyUnknown, []string{DecisionF01}},
		{"conflicting link", []DependencyLink{link(LinkLegacyInvoiceLead, LinkConflicting, ClassCommitted)}, true,
			FailClosedLinkage, result.Conflict, reasoncodes.LinkageConflict, []string{DecisionF01}},
		{"unvalidated compatibility id", []DependencyLink{link(LinkCompatibilityID, LinkResolved, ClassCommitted)}, true,
			FailClosedLinkage, result.Unknown, reasoncodes.DependencyUnknown, []string{DecisionF01}},
		{"unclassified invoice", []DependencyLink{link(LinkLegacyInvoiceLead, LinkResolved, ClassUnknown)}, true,
			FailClosedLinkage, result.Unknown, reasoncodes.DependencyUnknown, []string{DecisionF01}},
		{"union: cancelled in one source, paid in another", []DependencyLink{link(LinkLegacyInvoiceLead, LinkResolved, ClassCancelled),
			link(LinkFollowInvoice, LinkResolved, ClassPaidOrProtected)}, true, BlockedProtectedDownstream, result.Blocked,
			reasoncodes.DependencyProtectedDownstream, []string{DecisionOPD03, DecisionOPD04}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := ClassifyDependencies(c.links, c.complete)
			if v.Outcome != c.outcome || v.Status != c.status {
				t.Fatalf("got %s/%s, want %s/%s (%+v)", v.Outcome, v.Status, c.outcome, c.status, v)
			}
			if c.reason != "" && !slices.Contains(v.Reasons, c.reason) {
				t.Fatalf("reasons %v lack %s", v.Reasons, c.reason)
			}
			for _, b := range c.blocked {
				if !slices.Contains(v.BlockedBy, b) {
					t.Fatalf("blocked-by %v lacks %s", v.BlockedBy, b)
				}
			}
		})
	}
	v := ClassifyDependencies([]DependencyLink{link(LinkLegacyInvoiceLead, LinkResolved, ClassCancelled),
		link(LinkFollowInvoice, LinkResolved, ClassPaidOrProtected), link(LinkDiscoverySimilarity, LinkResolved, ClassCommitted)}, true)
	if len(v.Outcomes) != 2 || v.Considered != 2 || v.Discovery != 1 {
		t.Fatalf("union bookkeeping: %+v", v)
	}
	// A validated compatibility id counts like any resolved link.
	ok := link(LinkCompatibilityID, LinkResolved, ClassCommitted)
	ok.Validated = true
	if v := ClassifyDependencies([]DependencyLink{ok}, true); v.Outcome != BlockedFinancialCommitment {
		t.Fatalf("validated compatibility id: %+v", v)
	}
}
