package release

import (
	"slices"
	"testing"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func s(v string) *string { return &v }

func eligible(k Kind) Input {
	return Input{Kind: k, ActorUserID: "seller-a", ActorVerified: true, HolderUserID: s("seller-a"), ExpectedVersion: 2, CurrentVersion: 2,
		Dependencies: casesapi.ClassifyDependencies(nil, true), Lineage: casesapi.LineageKnown}
}

func dep(class casesapi.InvoiceClass) casesapi.DependencyVerdict {
	return casesapi.ClassifyDependencies([]casesapi.DependencyLink{{Source: casesapi.LinkLegacyInvoiceLead, Resolution: casesapi.LinkResolved,
		Class: class}}, true)
}

// R4 (F01): no return / reset / release executes; paid and cancelled
// invoices never release automatically; an active claim blocks the return
// (no issuance race); reset / release also need OPD-03.
func TestReleaseGate(t *testing.T) {
	for _, k := range []Kind{KindReturn, KindReset, KindRelease} {
		g := Evaluate(eligible(k))
		if g.Status != result.Blocked || !g.PreconditionsMet || !slices.Contains(g.BlockedBy, casesapi.DecisionADR01) ||
			!slices.Contains(g.BlockedBy, casesapi.DecisionF01) {
			t.Fatalf("%s must stay BLOCKED: %+v", k, g)
		}
		if (k != KindReturn) != slices.Contains(g.BlockedBy, casesapi.DecisionOPD03) {
			t.Fatalf("%s OPD-03: %+v", k, g)
		}
	}
	for _, c := range []struct {
		name   string
		mut    func(*Input)
		status result.Status
		named  string
	}{
		{"paid invoice", func(i *Input) { i.Dependencies = dep(casesapi.ClassPaidOrProtected) }, result.Blocked, ""},
		{"cancelled invoice", func(i *Input) { i.Dependencies = dep(casesapi.ClassCancelled) }, result.Blocked, casesapi.DecisionOPD03},
		{"committed invoice", func(i *Input) { i.Dependencies = dep(casesapi.ClassCommitted) }, result.Blocked, ""},
		{"missing invoice target", func(i *Input) {
			i.Dependencies = casesapi.ClassifyDependencies([]casesapi.DependencyLink{{Source: casesapi.LinkFollowInvoice,
				Resolution: casesapi.LinkMissingTarget}}, true)
		}, result.Unknown, casesapi.DecisionF01},
		{"active claim (issuance race)", func(i *Input) { i.ActiveClaim = true }, result.Conflict, ""},
		{"stale version", func(i *Input) { i.ExpectedVersion = 1 }, result.Conflict, ""},
		{"other seller", func(i *Input) { i.ActorUserID = "seller-b" }, result.Unauthorized, ""},
		{"unknown kind", func(i *Input) { i.Kind = "ARCHIVE" }, result.Failed, ""},
	} {
		in := eligible(KindReturn)
		c.mut(&in)
		g := Evaluate(in)
		if g.Status != c.status || g.PreconditionsMet || (c.named != "" && !slices.Contains(g.BlockedBy, c.named)) {
			t.Fatalf("%s: %+v", c.name, g)
		}
	}
}
