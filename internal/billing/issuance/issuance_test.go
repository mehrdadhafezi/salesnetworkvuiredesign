package issuance

import (
	"slices"
	"testing"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func s(v string) *string { return &v }

func eligible() Input {
	return Input{CaseID: "c1", ActorUserID: "seller-a", ActorVerified: true, ExpectedCaseVersion: 4,
		Claim:        &Claim{ClaimID: "k1", CaseID: "c1", SellerUserID: "seller-a", State: ClaimActive, CaseVersion: 4},
		Idempotency:  Idempotency{Key: "idem-1", RequestHash: "h1"},
		Attribution:  Attribution{SnapshotOriginalSeller: s("seller-o"), ProposedOriginalSeller: s("seller-o")},
		Total:        money.Money{Amount: 1_250_000, Unit: money.IRR},
		Dependencies: casesapi.ClassifyDependencies(nil, true), Customer: casesapi.IdentityResolved, Lineage: casesapi.LineageKnown}
}

// R4: issuance never succeeds while ADR-01 is open; every precondition has
// a precise outcome; attribution is immutable; money is exact.
func TestIssuanceGate(t *testing.T) {
	g := Evaluate(eligible())
	if g.Status != result.Blocked || !g.PreconditionsMet || !slices.Contains(g.BlockedBy, casesapi.DecisionADR01) {
		t.Fatalf("eligible issuance must stay BLOCKED by ADR-01: %+v", g)
	}
	for _, c := range []struct {
		name   string
		mut    func(*Input)
		status result.Status
	}{
		{"no claim", func(i *Input) { i.Claim = nil }, result.Blocked},
		{"another seller", func(i *Input) { i.ActorUserID = "seller-b" }, result.Unauthorized},
		{"unverified actor", func(i *Input) { i.ActorVerified = false }, result.Unauthorized},
		{"claim of another case", func(i *Input) { i.Claim.CaseID = "c2" }, result.Conflict},
		{"claim confirmed already", func(i *Input) { i.Claim.State = ClaimConfirmed }, result.Conflict},
		{"claim expired", func(i *Input) { i.Claim.Expired = true }, result.Conflict},
		{"stale version", func(i *Input) { i.ExpectedCaseVersion = 3 }, result.Conflict},
		{"duplicate issuance", func(i *Input) { i.IssuedForClaim = s("inv-1") }, result.Conflict},
		{"key reused, different payload", func(i *Input) { i.Idempotency.PriorRequestHash = s("other") }, result.Conflict},
		{"no idempotency key", func(i *Input) { i.Idempotency.Key = "" }, result.Failed},
		{"attribution rewrite", func(i *Input) { i.Attribution.ProposedOriginalSeller = s("seller-a") }, result.Conflict},
		{"zero total", func(i *Input) { i.Total.Amount = 0 }, result.Failed},
		{"no unit", func(i *Input) { i.Total.Unit = 0 }, result.Failed},
		{"partial assessment", func(i *Input) { i.AssessmentPartialPayment = true }, result.Blocked},
		{"customer from similarity", func(i *Input) { i.Customer = casesapi.IdentityDiscoveryOnly }, result.Unknown},
	} {
		in := eligible()
		c.mut(&in)
		g := Evaluate(in)
		if g.Status != c.status || g.PreconditionsMet || !slices.Contains(g.BlockedBy, casesapi.DecisionADR01) {
			t.Fatalf("%s: %+v", c.name, g)
		}
	}
	in := eligible()
	in.AssessmentPartialPayment = true
	if g := Evaluate(in); !slices.Contains(g.BlockedBy, casesapi.DecisionD28) {
		t.Fatalf("D-28 not named: %+v", g)
	}
	// Same key, same payload is a replay: not a conflict.
	in = eligible()
	in.Idempotency.PriorRequestHash = s("h1")
	if g := Evaluate(in); g.Status != result.Blocked {
		t.Fatalf("replay: %+v", g)
	}
}
