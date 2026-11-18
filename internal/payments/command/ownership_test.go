package command

import (
	"context"
	"testing"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ownsTarget (the read gate of the submission context, RM-E04B / A3) must be
// exactly the ownership part of evaluateTarget (the command's gate): every
// combination of seller / issuer / actor either passes both or is refused by
// evaluateTarget's ownership checks. StagesComplete=false makes an owner stop
// at the first post-ownership check (COVERAGE_INCOMPLETE) without a query.
func TestOwnsTargetMatchesCommandPredicate(t *testing.T) {
	a, b := "00000000-0000-4000-8000-00000000000a", "00000000-0000-4000-8000-00000000000b"
	ids := []*string{nil, &a, &b}
	s := &Service{}
	for _, actor := range []string{"", a, b} {
		for _, seller := range ids {
			for _, issuer := range ids {
				tg := billingapi.PaymentTarget{Present: true, SellerUserID: seller, IssuedByUserID: issuer, StagesComplete: false}
				ac := Actor{Principal: "synthetic", UserID: actor}
				e, err := s.evaluateTarget(context.Background(), nil, tg, ac)
				if err != nil {
					t.Fatal(err)
				}
				ownershipRefusal := (e.Status == result.Unauthorized && len(e.Reasons) == 1 && e.Reasons[0] == reasoncodes.RowScopeDenied) ||
					(e.Status == result.Blocked && len(e.Reasons) == 1 && e.Reasons[0] == reasoncodes.DependencyUnknown)
				owner := ownsTarget(tg, ac)
				if owner == ownershipRefusal {
					t.Fatalf("actor=%q seller=%v issuer=%v: ownsTarget=%v but evaluateTarget=%+v", actor, deref(seller), deref(issuer), owner, e)
				}
				if owner && (e.Status != result.Blocked || e.Reasons[0] != reasoncodes.CoverageIncomplete) {
					t.Fatalf("owner must reach the post-ownership checks: %+v", e)
				}
				if owner != (actor != "" && seller != nil && issuer != nil && *seller == actor && *issuer == actor) {
					t.Fatalf("predicate drift: actor=%q seller=%v issuer=%v", actor, deref(seller), deref(issuer))
				}
			}
		}
	}
}

func deref(p *string) string {
	if p == nil {
		return "<NULL>"
	}
	return *p
}
