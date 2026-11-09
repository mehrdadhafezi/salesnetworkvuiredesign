package shadow_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/review"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// R4: the Finance decision contract over the REAL payments published
// review read on PostgreSQL (synthetic data). The read now carries the
// attempt version and the immutable receipt digests; the contract never
// answers SUCCESS: a maker is UNAUTHORIZED, a wrong digest or a stale
// version is a CONFLICT, and an independent, eligible reviewer is BLOCKED by
// the open decisions. Nothing is decided or recorded.
func TestR4FinanceDecisionContractOverReviewRead(t *testing.T) {
	e := setupR2(t)
	ctx := context.Background()
	seller := e.actor("6")
	up := e.upload(t, "6", 71)
	e.submit(t, batch("r4-review-0001", e.invoices["3001"], 0, seller, item("a", 600, up)))
	e.pass(t, "receipt_finalise")
	queue, err := e.svc.ReviewQueue(ctx, nil, 10)
	if err != nil || len(queue) == 0 {
		t.Fatalf("review queue: %v %v", queue, err)
	}
	a := queue[0]
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if a.Version < 1 || len(a.Receipts) != 1 || !hex64.MatchString(a.Receipts[0].ContentSHA256) || a.Receipts[0].ContentSHA256 != up.ContentSHA256 {
		t.Fatalf("review read: version %d receipts %+v (uploaded %s)", a.Version, a.Receipts, up.ContentSHA256)
	}
	if a.Receipts[0].State != paymentsapi.ReceiptReady || a.Receipts[0].EvidenceRef == nil ||
		!strings.HasSuffix(*a.Receipts[0].EvidenceRef, a.Receipts[0].ContentSHA256) {
		t.Fatalf("READY receipt must carry its content-addressed reference: %+v", a.Receipts[0])
	}
	cite := []review.EvidenceRef{{ReceiptID: a.Receipts[0].ReceiptID, ContentSHA256: a.Receipts[0].ContentSHA256}}
	reviewer := review.Reviewer{UserID: e.users["2"], Verified: true, Capabilities: []string{review.ActionApprove}, InRecordScope: true,
		PersonLink: review.PersonLinkDistinct}
	req := review.DecisionRequest{Action: review.ActionApprove, Attempt: a, Reviewer: reviewer, ExpectedVersion: a.Version, Evidence: cite}
	if d := review.EvaluateDecision(req); d.Status != result.Blocked || !d.PreconditionsMet {
		t.Fatalf("independent eligible reviewer: %+v", d)
	}
	maker := req
	maker.Reviewer.UserID = seller.UserID
	if d := review.EvaluateDecision(maker); d.Status != result.Unauthorized {
		t.Fatalf("maker as checker: %+v", d)
	}
	stale := req
	stale.ExpectedVersion = a.Version - 1
	if d := review.EvaluateDecision(stale); d.Status != result.Conflict {
		t.Fatalf("stale version: %+v", d)
	}
	tampered := req
	tampered.Evidence = []review.EvidenceRef{{ReceiptID: cite[0].ReceiptID, ContentSHA256: strings.Repeat("0", 64)}}
	if d := review.EvaluateDecision(tampered); d.Status != result.Conflict {
		t.Fatalf("wrong digest: %+v", d)
	}
	// The read path still records nothing: the attempt is unchanged.
	after, ok, err := e.svc.ReviewAttempt(ctx, a.AttemptID)
	if err != nil || !ok || after.Version != a.Version || after.State != paymentsapi.AttemptPendingReview {
		t.Fatalf("attempt changed by an evaluation: %+v %v", after, err)
	}
}
