package review

import (
	"slices"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

const sumA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func readyAttempt() paymentsapi.ReviewAttempt {
	return paymentsapi.ReviewAttempt{AttemptID: "att", State: paymentsapi.AttemptPendingReview, EvidenceState: paymentsapi.EvidenceReady,
		SubmittedByUserID: "seller", Version: 3,
		Receipts: []paymentsapi.ReviewReceipt{{ReceiptID: "r1", State: paymentsapi.ReceiptReady, ContentSHA256: sumA}}}
}

func fullyEligible(action string) DecisionRequest {
	req := DecisionRequest{Action: action, Attempt: readyAttempt(), ExpectedVersion: 3,
		Reviewer: Reviewer{UserID: "finance-1", Verified: true, Capabilities: []string{action}, InRecordScope: true, PersonLink: PersonLinkDistinct},
		Evidence: []EvidenceRef{{ReceiptID: "r1", ContentSHA256: sumA}}}
	if action == ActionReject {
		req.Rejection = &Rejection{Code: "ANY", Detail: "synthetic detail"}
	}
	return req
}

// R4: even a hypothetical, fully eligible, independent, capable, in-scope
// reviewer citing exact evidence is BLOCKED: OPD-08 / D-41 / OPD-09 and
// (for reject) the missing rejection contract and vocabulary.
func TestDecisionContractNeverSucceeds(t *testing.T) {
	for _, a := range []string{ActionApprove, ActionReject} {
		d := EvaluateDecision(fullyEligible(a))
		if d.Status != result.Blocked || !slices.Contains(d.BlockedBy, DecisionOPD08) || !slices.Contains(d.BlockedBy, NoDecisionCapability) {
			t.Fatalf("%s: %+v", a, d)
		}
		if a == ActionApprove && (!d.PreconditionsMet || !slices.Contains(d.BlockedBy, DecisionD41) || !slices.Contains(d.BlockedBy, DecisionOPD09)) {
			t.Fatalf("approve: %+v", d)
		}
		if a == ActionReject && (!slices.Contains(d.BlockedBy, RejectionContractUndefined) || !slices.Contains(d.BlockedBy, RejectionVocabularyUndefined)) {
			t.Fatalf("reject: %+v", d)
		}
	}
}

func TestDecisionContractGuards(t *testing.T) {
	cases := []struct {
		name   string
		mut    func(*DecisionRequest)
		status result.Status
		named  string
	}{
		{"maker acts as checker", func(r *DecisionRequest) { r.Reviewer.UserID = "seller" }, result.Unauthorized, DecisionOPD08},
		{"unverified reviewer", func(r *DecisionRequest) { r.Reviewer.Verified = false }, result.Unauthorized, ""},
		{"same human, another account", func(r *DecisionRequest) { r.Reviewer.PersonLink = PersonLinkSameHuman }, result.Unauthorized, ""},
		{"person link not modelled", func(r *DecisionRequest) { r.Reviewer.PersonLink = PersonLinkNotModelled }, result.Blocked, PersonLinkUnmodelled},
		{"out of record scope", func(r *DecisionRequest) { r.Reviewer.InRecordScope = false }, result.Unauthorized, ""},
		{"no capability", func(r *DecisionRequest) { r.Reviewer.Capabilities = nil }, result.Blocked, NoDecisionCapability},
		{"stale version", func(r *DecisionRequest) { r.ExpectedVersion = 2 }, result.Conflict, ""},
		{"duplicate review", func(r *DecisionRequest) { p := "APPROVED"; r.PriorDecision = &p }, result.Conflict, ""},
		{"evidence not READY", func(r *DecisionRequest) {
			r.Attempt.EvidenceState = paymentsapi.EvidencePendingFinalisation
			r.Attempt.Receipts[0].State = paymentsapi.ReceiptQuarantined
		}, result.Blocked, DecisionD43},
		{"wrong digest cited", func(r *DecisionRequest) { r.Evidence[0].ContentSHA256 = strings.Repeat("b", 64) }, result.Conflict, ""},
		{"foreign receipt cited", func(r *DecisionRequest) {
			r.Evidence = append(r.Evidence, EvidenceRef{ReceiptID: "r9", ContentSHA256: sumA})
		}, result.Conflict, ""},
		{"receipt not cited", func(r *DecisionRequest) { r.Evidence = nil }, result.Failed, ""},
		{"not pending review", func(r *DecisionRequest) { r.Attempt.State = "ACCEPTED" }, result.Conflict, ""},
	}
	for _, c := range cases {
		r := fullyEligible(ActionApprove)
		c.mut(&r)
		d := EvaluateDecision(r)
		if d.Status != c.status || d.PreconditionsMet || (c.named != "" && !slices.Contains(d.BlockedBy, c.named)) {
			t.Fatalf("%s: %+v", c.name, d)
		}
	}
	// Rejection structure.
	r := fullyEligible(ActionReject)
	r.Rejection = nil
	if d := EvaluateDecision(r); d.Status != result.Failed {
		t.Fatalf("rejection without reason: %+v", d)
	}
	r = fullyEligible(ActionReject)
	r.Rejection.Detail = strings.Repeat("x", MaxRejectionDetail+1)
	if d := EvaluateDecision(r); d.Status != result.Failed {
		t.Fatalf("unbounded detail: %+v", d)
	}
	r = fullyEligible(ActionApprove)
	r.Rejection = &Rejection{Code: "X", Detail: "y"}
	if d := EvaluateDecision(r); d.Status != result.Failed {
		t.Fatalf("approval with a rejection reason: %+v", d)
	}
}

// The evaluation digest is deterministic and changes with any input.
func TestDecisionEvaluationDigest(t *testing.T) {
	a, b := EvaluateDecision(fullyEligible(ActionApprove)), EvaluateDecision(fullyEligible(ActionApprove))
	if a.EvaluationSHA256 == "" || a.EvaluationSHA256 != b.EvaluationSHA256 {
		t.Fatal("digest must be deterministic")
	}
	r := fullyEligible(ActionApprove)
	r.Reviewer.Capabilities = []string{"z", ActionApprove}
	r2 := fullyEligible(ActionApprove)
	r2.Reviewer.Capabilities = []string{ActionApprove, "z"}
	if EvaluateDecision(r).EvaluationSHA256 != EvaluateDecision(r2).EvaluationSHA256 {
		t.Fatal("digest must not depend on capability order")
	}
	r.ExpectedVersion = 2
	if EvaluateDecision(r).EvaluationSHA256 == a.EvaluationSHA256 {
		t.Fatal("digest must change with the input")
	}
}
