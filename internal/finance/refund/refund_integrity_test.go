package refund

import (
	"math"
	"slices"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// P2-A (PR #44 review): the refund evaluator must establish the monetary
// invariants of OriginalPaid / PreviouslyRefunded itself; invalid, negative,
// mismatched or inconsistent facts fail closed and are never BLOCKED-as-
// eligible.
func TestRefundMonetaryInvariants(t *testing.T) {
	irt := func(n int64) money.Money { return money.Money{Amount: n, Unit: money.IRT} }
	for _, c := range []struct {
		name               string
		paid, prev, amount money.Money
		status             result.Status
	}{
		{"negative previously refunded inflates the refundable amount", irr(1000), irr(-500), irr(1200), result.Failed},
		{"negative original paid", irr(-100), irr(0), irr(50), result.Failed},
		{"original paid without a unit", money.Money{Amount: 1000}, irr(0), irr(10), result.Failed},
		{"previously refunded without a unit", irr(1000), money.Money{Amount: 0}, irr(10), result.Failed},
		{"facts in another unit than the refund", irt(1000), irt(0), irr(10), result.Failed},
		{"previously refunded in another unit", irr(1000), irt(0), irr(10), result.Failed},
		{"previously refunded exceeds original paid", irr(500), irr(600), irr(1), result.Conflict},
		{"amount above the remaining refundable", irr(1000), irr(200), irr(801), result.Conflict},
		{"overflowing facts", irr(math.MaxInt64), irr(-1), irr(1), result.Failed},
		{"zero amount", irr(1000), irr(0), irr(0), result.Failed},
		{"negative amount", irr(1000), irr(0), irr(-5), result.Failed},
	} {
		r := base(Requested)
		r.OriginalPaid, r.PreviouslyRefunded, r.Amount = c.paid, c.prev, c.amount
		e := EvaluateTransition(r, Reviewed, Evidence{}, auditor)
		if e.Status != c.status || !slices.Contains(e.BlockedBy, DecisionOPD04) {
			t.Errorf("%s: got %s, want %s", c.name, e.Status, c.status)
		}
	}
	// Boundaries that stay structurally sound (still BLOCKED by policy).
	for _, c := range []struct{ paid, prev, amount int64 }{{1000, 200, 800}, {1000, 0, 1000}, {math.MaxInt64, 0, math.MaxInt64}, {1000, 999, 1}} {
		r := base(Requested)
		r.OriginalPaid, r.PreviouslyRefunded, r.Amount = irr(c.paid), irr(c.prev), irr(c.amount)
		if e := EvaluateTransition(r, Reviewed, Evidence{}, auditor); e.Status != result.Blocked {
			t.Fatalf("%+v: %+v", c, e)
		}
	}
}

// P2-B (PR #44 review): the projection validates the append-only sequence
// contract and never skips, repairs or manufactures history.
func TestProjectionHistoryIntegrity(t *testing.T) {
	ok := []Event{{1, "rf", Requested}, {2, "rf", Reviewed}, {3, "rf", Authorized}}
	for _, c := range []struct {
		name   string
		events []Event
	}{
		{"empty history", nil},
		{"duplicate sequence", []Event{{1, "rf", Requested}, {1, "rf", Reviewed}, {2, "rf", Authorized}}},
		{"missing sequence (gap)", []Event{{1, "rf", Requested}, {3, "rf", Reviewed}}},
		{"zero sequence", []Event{{0, "rf", Requested}, {1, "rf", Reviewed}}},
		{"negative sequence", []Event{{-1, "rf", Requested}, {0, "rf", Reviewed}}},
		{"sequence not starting at 1", []Event{{2, "rf", Requested}, {3, "rf", Reviewed}}},
		{"event of another refund", []Event{{1, "rf", Requested}, {2, "other", Reviewed}, {3, "rf", Reviewed}}},
		{"invalid transition is not skipped", []Event{{1, "rf", Requested}, {2, "rf", Completed}, {3, "rf", Reviewed}}},
		{"first event not REQUESTED", []Event{{1, "rf", Reviewed}}},
	} {
		in := slices.Clone(c.events)
		p := Project("rf", c.events)
		if p.Valid || p.State != "" || len(p.Findings) == 0 {
			t.Errorf("%s: an invalid history must not yield a current state: valid=%v state=%q", c.name, p.Valid, p.State)
		}
		if !slices.Equal(in, c.events) {
			t.Fatalf("%s: projection mutated its input", c.name)
		}
	}
	// Out-of-order delivery of a contiguous history is ordered, not repaired.
	if p := Project("rf", []Event{ok[2], ok[0], ok[1]}); !p.Valid || p.State != Authorized || len(p.Findings) != 0 {
		t.Fatalf("contiguous history: %+v", p)
	}
}

// Separation of duties: distinct account ids are necessary but never
// sufficient; without authoritative person linkage the transition carries
// IDENTITY_PERSON_LINK_UNMODELLED, and a same-human link is UNAUTHORIZED.
func TestRefundPersonLevelIndependence(t *testing.T) {
	for _, to := range []State{Reviewed, Authorized} {
		from := Requested
		if to == Authorized {
			from = Reviewed
		}
		e := EvaluateTransition(base(from), to, Evidence{}, Actor{UserID: "finance-2", Verified: true})
		if e.Status != result.Blocked || !slices.Contains(e.BlockedBy, PersonLinkUnmodelled) {
			t.Fatalf("%s without person linkage: %+v", to, e)
		}
		e = EvaluateTransition(base(from), to, Evidence{}, Actor{UserID: "finance-2", Verified: true, Independence: PersonSameHuman})
		if e.Status != result.Unauthorized {
			t.Fatalf("%s by the same human on another account: %+v", to, e)
		}
		e = EvaluateTransition(base(from), to, Evidence{}, Actor{UserID: "finance-2", Verified: true, Independence: PersonDistinct})
		if e.Status != result.Blocked || slices.Contains(e.BlockedBy, PersonLinkUnmodelled) || !slices.Contains(e.BlockedBy, DecisionOPD04) {
			t.Fatalf("%s with proven independence stays BLOCKED by OPD-04: %+v", to, e)
		}
	}
	// Transitions without a separation-of-duties step do not claim one.
	e := EvaluateTransition(base(EvidenceReceived), Reconciled, Evidence{Kind: EvidenceReconciliation, DocumentSHA256: doc}, auditor)
	if slices.Contains(e.BlockedBy, PersonLinkUnmodelled) {
		t.Fatalf("reconcile: %+v", e)
	}
}

// The verified prefix is diagnosis only and stops at the first problem.
func TestProjectionVerifiedPrefix(t *testing.T) {
	p := Project("rf", []Event{{1, "rf", Requested}, {2, "rf", Reviewed}, {4, "rf", Authorized}, {5, "rf", Unknown}})
	if p.Valid || p.State != "" || p.VerifiedState != Reviewed || p.VerifiedThrough != 2 {
		t.Fatalf("gap: %+v", p)
	}
}
