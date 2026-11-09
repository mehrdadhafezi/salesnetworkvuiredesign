package refund

import (
	"slices"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func irr(n int64) money.Money { return money.Money{Amount: n, Unit: money.IRR} }

var doc = strings.Repeat("c", 64)

func base(state State) Refund {
	return Refund{RefundID: "rf", OriginalInvoiceID: "inv", OriginalPaymentID: "pay", State: state, Amount: irr(300), OriginalPaid: irr(1000),
		PreviouslyRefunded: irr(200), RequestedBy: "seller", ReviewedBy: "finance-1"}
}

func proof(n int64) Evidence {
	m := irr(n)
	return Evidence{Kind: EvidenceBankStatement, ExternalRef: "bank-ref-1", DocumentSHA256: doc, Amount: &m}
}

var auditor = Actor{UserID: "finance-2", Verified: true}

// Every structurally sound transition is BLOCKED by OPD-04; nothing executes.
func TestEveryLegalTransitionIsBlocked(t *testing.T) {
	ev := map[State]Evidence{ExternallyExecuted: proof(300), EvidenceReceived: proof(300),
		Reconciled: {Kind: EvidenceReconciliation, DocumentSHA256: doc}}
	n := 0
	for from, tos := range transitions {
		for _, to := range tos {
			n++
			e := EvaluateTransition(base(from), to, ev[to], auditor)
			if !e.Legal || e.Status != result.Blocked || !slices.Contains(e.BlockedBy, DecisionOPD04) {
				t.Fatalf("%s -> %s: %+v", from, to, e)
			}
		}
	}
	if n != 11 {
		t.Fatalf("transition count %d", n)
	}
}

func TestRefundGuards(t *testing.T) {
	for _, c := range []struct {
		name   string
		r      Refund
		to     State
		ev     Evidence
		actor  Actor
		status result.Status
	}{
		{"cancellation is not money returned", base(Authorized), ExternallyExecuted, Evidence{Kind: EvidenceInvoiceCancelled}, auditor, result.Conflict}, // conflict: no proven amount; also refused as evidence,
		{"HTTP success is not a refund", base(Authorized), ExternallyExecuted, Evidence{Kind: EvidenceHTTPSuccess, ExternalRef: "x", DocumentSHA256: doc,
			Amount: func() *money.Money { m := irr(300); return &m }()}, auditor, result.Failed},
		{"proof amount differs", base(ExternallyExecuted), EvidenceReceived, proof(299), auditor, result.Conflict},
		{"proof in another unit", base(ExternallyExecuted), EvidenceReceived, func() Evidence {
			e := proof(300)
			e.Amount.Unit = money.IRT
			return e
		}(), auditor, result.Conflict},
		{"skip straight to completed", base(Authorized), Completed, Evidence{}, auditor, result.Conflict},
		{"no way back from UNKNOWN to AUTHORIZED", base(Unknown), Authorized, Evidence{}, auditor, result.Conflict},
		{"requester reviews own refund", base(Requested), Reviewed, Evidence{}, Actor{UserID: "seller", Verified: true}, result.Unauthorized},
		{"reviewer authorizes", base(Reviewed), Authorized, Evidence{}, Actor{UserID: "finance-1", Verified: true}, result.Unauthorized},
		{"unverified actor", base(Requested), Reviewed, Evidence{}, Actor{UserID: "x"}, result.Unauthorized},
		{"exceeds refundable", func() Refund { r := base(Requested); r.Amount = irr(801); return r }(), Reviewed, Evidence{}, auditor, result.Conflict},
		{"zero amount", func() Refund { r := base(Requested); r.Amount = irr(0); return r }(), Reviewed, Evidence{}, auditor, result.Failed},
		{"no original invoice", func() Refund { r := base(Requested); r.OriginalInvoiceID = ""; return r }(), Reviewed, Evidence{}, auditor, result.Failed},
		{"reconcile without record", base(EvidenceReceived), Reconciled, Evidence{}, auditor, result.Failed},
		{"terminal completed", base(Completed), Rejected, Evidence{}, auditor, result.Conflict},
	} {
		e := EvaluateTransition(c.r, c.to, c.ev, c.actor)
		if e.Status != c.status || !slices.Contains(e.BlockedBy, DecisionOPD04) {
			t.Fatalf("%s: %+v", c.name, e)
		}
	}
	// Exactly refundable is allowed structurally (still blocked by policy).
	r := base(Requested)
	r.Amount = irr(800)
	if e := EvaluateTransition(r, Reviewed, Evidence{}, auditor); e.Status != result.Blocked {
		t.Fatalf("exact refundable: %+v", e)
	}
}

func TestProjection(t *testing.T) {
	full := []Event{{1, "rf", Requested}, {2, "rf", Reviewed}, {3, "rf", Authorized}, {4, "rf", Unknown}, {5, "rf", EvidenceReceived},
		{6, "rf", Reconciled}, {7, "rf", Completed}}
	// Out-of-order delivery is ordered by sequence.
	shuffled := []Event{full[3], full[0], full[6], full[2], full[1], full[5], full[4]}
	if p := Project("rf", shuffled); !p.Valid || p.State != Completed {
		t.Fatalf("full history: %+v", p)
	}
	if p := Project("rf", []Event{{1, "rf", Requested}, {2, "rf", Completed}}); p.Valid || p.State != "" || p.VerifiedState != Requested ||
		p.VerifiedThrough != 1 {
		t.Fatalf("an illegal step invalidates the history (verified prefix kept for diagnosis only): %+v", p)
	}
	if p := Project("rf", []Event{{1, "rf", Authorized}}); p.Valid {
		t.Fatalf("history must start with REQUESTED: %+v", p)
	}
	if p := Project("rf", []Event{{1, "rf", Requested}, {2, "other", Reviewed}}); p.Valid || p.State != "" {
		t.Fatalf("foreign event: %+v", p)
	}
	for _, s := range States() {
		if Terminal(s) && len(transitions[s]) != 0 {
			t.Fatalf("terminal %s has transitions", s)
		}
	}
}
