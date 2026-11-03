package review

import (
	"slices"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

func attempt(evidence string, receipts ...string) paymentsapi.ReviewAttempt {
	a := paymentsapi.ReviewAttempt{AttemptID: "a", State: paymentsapi.AttemptPendingReview, EvidenceState: evidence, SubmittedByUserID: "seller"}
	for _, s := range receipts {
		a.Receipts = append(a.Receipts, paymentsapi.ReviewReceipt{State: s})
	}
	return a
}

// Finance approval while evidence is not READY must fail (D-43), and a
// maker can never act as checker (OPD-08 safe default).
func TestGateD43AndMakerChecker(t *testing.T) {
	for _, a := range []paymentsapi.ReviewAttempt{
		attempt(paymentsapi.EvidencePendingFinalisation, paymentsapi.ReceiptQuarantined),
		attempt(paymentsapi.EvidenceReady, paymentsapi.ReceiptReady, paymentsapi.ReceiptStorageError),
		attempt(paymentsapi.EvidenceNotRecoverable, paymentsapi.ReceiptNotRecoverable),
		attempt(paymentsapi.EvidenceReady),
	} {
		g := Evaluate(a, "finance-user")
		if g.EvidenceReady {
			t.Fatalf("evidence wrongly READY: %+v", a)
		}
		for _, b := range g.Actions {
			if b.Status == result.Success || !slices.Contains(b.BlockedBy, DecisionD43) {
				t.Fatalf("%s not blocked by D-43: %+v", b.Action, b)
			}
		}
	}
	ready := attempt(paymentsapi.EvidenceReady, paymentsapi.ReceiptReady)
	g := Evaluate(ready, "seller")
	if !g.ReviewerIsMaker {
		t.Fatal("maker not detected")
	}
	for _, b := range g.Actions {
		if b.Status != result.Unauthorized || !slices.Contains(b.BlockedBy, DecisionOPD08) {
			t.Fatalf("maker acting as checker: %+v", b)
		}
	}
	if g := Evaluate(ready, ""); !g.ReviewerIsMaker {
		t.Fatal("an unidentified reviewer is never independent")
	}
	// Even a fully eligible independent reviewer is blocked: no decision
	// policy is approved (D-41, OPD-08, OPD-09) and no capability exists.
	g = Evaluate(ready, "finance-user")
	if !g.EvidenceReady || g.ReviewerIsMaker {
		t.Fatalf("gate: %+v", g)
	}
	for _, b := range g.Actions {
		if b.Status != result.Blocked || !slices.Contains(b.BlockedBy, NoDecisionCapability) || !slices.Contains(b.BlockedBy, DecisionOPD08) {
			t.Fatalf("decision must stay blocked: %+v", b)
		}
	}
	if !slices.Contains(g.Actions[0].BlockedBy, DecisionD41) || !slices.Contains(g.Actions[1].BlockedBy, RejectionContractUndefined) {
		t.Fatalf("decision ids: %+v", g.Actions)
	}
}
