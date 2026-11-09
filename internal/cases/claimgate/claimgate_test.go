package claimgate

import (
	"slices"
	"testing"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func s(v string) *string { return &v }

func eligible() Input {
	return Input{CaseID: "c1", CasePresent: true, ActorUserID: "seller-a", ActorVerified: true, HolderUserID: s("seller-a"),
		ExpectedVersion: 3, CurrentVersion: 3, Dependencies: casesapi.ClassifyDependencies(nil, true),
		Customer: casesapi.IdentityResolved, Lineage: casesapi.LineageKnown}
}

// R4: even with every precondition met the claim is BLOCKED by ADR-01;
// each precondition failure has its precise outcome.
func TestClaimGate(t *testing.T) {
	g := Evaluate(eligible())
	if g.Status != result.Blocked || !g.PreconditionsMet || !slices.Contains(g.BlockedBy, casesapi.DecisionADR01) {
		t.Fatalf("eligible claim must stay BLOCKED by ADR-01: %+v", g)
	}
	for _, c := range []struct {
		name   string
		mut    func(*Input)
		status result.Status
	}{
		{"unverified actor", func(i *Input) { i.ActorVerified = false }, result.Unauthorized},
		{"another seller (no rank shortcut)", func(i *Input) { i.ActorUserID = "seller-b" }, result.Unauthorized},
		{"holder unknown", func(i *Input) { i.HolderUserID = nil }, result.Unknown},
		{"stale version", func(i *Input) { i.ExpectedVersion = 2 }, result.Conflict},
		{"second claim", func(i *Input) { i.ActiveClaim = &ActiveClaim{ClaimID: "k", SellerUserID: "seller-b"} }, result.Conflict},
		{"return in progress", func(i *Input) { i.ReturnInProgress = true }, result.Conflict},
		{"existing committed invoice", func(i *Input) {
			i.Dependencies = casesapi.ClassifyDependencies([]casesapi.DependencyLink{{Source: casesapi.LinkLegacyInvoiceLead,
				Resolution: casesapi.LinkResolved, Class: casesapi.ClassCommitted}}, true)
		}, result.Blocked},
		{"ambiguous links", func(i *Input) {
			i.Dependencies = casesapi.ClassifyDependencies([]casesapi.DependencyLink{{Source: casesapi.LinkLegacyInvoiceLead,
				Resolution: casesapi.LinkUnresolved}}, true)
		}, result.Unknown},
		{"customer only from similarity", func(i *Input) { i.Customer = casesapi.IdentityDiscoveryOnly }, result.Unknown},
		{"lineage ambiguous", func(i *Input) { i.Lineage = casesapi.LineageAmbiguous }, result.Unknown},
		{"case absent", func(i *Input) { i.CasePresent = false }, result.Unknown},
	} {
		in := eligible()
		c.mut(&in)
		g := Evaluate(in)
		if g.Status != c.status || g.PreconditionsMet || !slices.Contains(g.BlockedBy, casesapi.DecisionADR01) {
			t.Fatalf("%s: %+v", c.name, g)
		}
	}
}
