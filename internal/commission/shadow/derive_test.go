package shadow

import (
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
)

func s(v string) *string { return &v }

func e1(tx, holder, recipient, typ, dir string) api.LegacyCredit {
	return api.LegacyCredit{TxKey: tx, Engine: "PURPOSE_MATRIX", TypeRaw: typ, Direction: dir, HolderKey: s(holder), InvoiceKey: s("2002"),
		AmountRaw: s("40.00"), Unit: "IRT", StageRaw: s("1"), PurposeRaw: s("assessment_fee"), RoleRaw: s("seller"), RateRaw: s("2"),
		BaseRaw: s("2000"), RecipientSnapKey: s(recipient), RewrittenInPlace: typ == "purpose_commission_reversed"}
}

// The original recipient of an entitlement is the holder of its EARLIEST
// credit, not the live re-credited one (staging order must not matter).
func TestFirstRecipientIsTheOriginalHolder(t *testing.T) {
	recredit := e1("105", "6", "6", "purpose_commission", api.DirCredit)
	original := e1("103", "5", "5", "purpose_commission_reversed", api.DirCredit)
	rev := api.LegacyCredit{TxKey: "104", Engine: "PURPOSE_MATRIX", TypeRaw: "purpose_commission_reversal", Direction: api.DirDebit,
		HolderKey: s("5"), ReversesTxKey: s("103"), Unit: "IRT"}
	for _, order := range [][]api.LegacyCredit{{original, rev, recredit}, {recredit, rev, original}} {
		d := Derive(api.Evidence{SnapshotID: "x", Cutoff: time.Now(), Credits: order}, domain.RoundingLegacyHalfAwayTwoDP)
		d.Finish(map[string]int{})
		if len(d.Calcs) != 1 {
			t.Fatalf("one entitlement expected, got %d", len(d.Calcs))
		}
		c := d.Calcs[0]
		if c.SourceKey != "tx:105" || c.FirstRecipient == nil || *c.FirstRecipient != "5" || *firstRecipient(c) != "5" {
			t.Fatalf("calc %s first recipient %v", c.SourceKey, c.FirstRecipient)
		}
	}
}

// A posted dry-run item credited to someone other than its seller is a
// RECIPIENT_MISMATCH, never a MATCH; seller and credit owner stay distinct.
func TestDryRunPostedToAnotherHolder(t *testing.T) {
	ev := api.Evidence{SnapshotID: "x", Cutoff: time.Now(),
		Rules: []api.Rule{{SourceKey: "1", RuleCode: s("R1"), CalculationTypeRaw: s("invoice_percent_fixed"), RateRaw: s("3"), FixedRaw: s("5")}},
		DryRunItems: []api.DryRunItem{{SourceKey: "1", RunKey: s("1"), InvoiceKey: s("2001"), SellerKey: s("5"), InvoiceAmountRaw: s("1000.00"),
			MatchedRuleKey: s("1"), CalculatedRaw: s("35.00"), WalletTxKey: s("112"), PostingStatusRaw: s("posted"), Unit: "IRT"}},
		Credits: []api.LegacyCredit{{TxKey: "112", Engine: "DRY_RUN_POSTING", TypeRaw: "commission_engine", Direction: api.DirCredit, HolderKey: s("9"),
			InvoiceKey: s("2001"), AmountRaw: s("35.00"), Unit: "IRT", DryRunKey: s("1"), DryRunItemKey: s("1")}}}
	d := Derive(ev, domain.RoundingLegacyHalfAwayTwoDP)
	d.Finish(map[string]int{})
	var item *CalcRow
	for i := range d.Calcs {
		if d.Calcs[i].SourceKey == "item:1" {
			item = &d.Calcs[i]
		}
	}
	if item == nil || item.Parity.Class != domain.ParityRecipientMismatch {
		t.Fatalf("item parity: %+v", item)
	}
	if *item.Output.Recipient.UserKey != "5" {
		t.Fatal("the seller stays the attribution snapshot")
	}
	found := false
	for _, f := range d.Findings {
		found = found || (f.Kind == "RECIPIENT_MISMATCH" && f.SourceKey == "112")
	}
	if !found {
		t.Fatalf("missing RECIPIENT_MISMATCH finding: %+v", d.Findings)
	}
	// Same holder as seller: MATCH and no finding.
	ev.Credits[0].HolderKey = s("5")
	d = Derive(ev, domain.RoundingLegacyHalfAwayTwoDP)
	d.Finish(map[string]int{})
	for _, c := range d.Calcs {
		if c.SourceKey == "item:1" && c.Parity.Class != domain.ParityMatch {
			t.Fatalf("matching holder: %s", c.Parity.Class)
		}
	}
	for _, f := range d.Findings {
		if f.Kind == "RECIPIENT_MISMATCH" {
			t.Fatal("no finding when holder equals seller")
		}
	}
}
