package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

func ex(t *testing.T, s string) *Exact {
	t.Helper()
	e, err := ParseExact(s, money.IRT)
	if err != nil {
		t.Fatal(err)
	}
	return &e
}

func rt(t *testing.T, s string) *Rate {
	t.Helper()
	r, err := ParseRate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &r
}

func key(s string) *string { return &s }

func matrixInput(t *testing.T, invoice string, stage int, purpose, role, recipient string) Input {
	t.Helper()
	return Input{
		Identity:   EntitlementIdentity{Kind: KindInvoiceStage, InvoiceKey: invoice, Stage: stage, Purpose: purpose, ShareFacet: role, Context: string(EnginePurposeMatrix)},
		FunctionID: FnPurposeMatrixStagePercentV1, PolicyKey: "purpose_matrix:" + purpose + ":" + role, PolicyVer: 1,
		Base: ex(t, "1000000"), BaseKind: BaseApprovedStageAmount, Rate: rt(t, "2.5"), Unit: money.IRT,
		Rounding:       RoundingLegacyHalfAwayTwoDP,
		Attribution:    Attribution{FacetCreditOwner: {State: AttrSnapshot, UserKey: key(recipient), Basis: "meta.recipient_user_id"}},
		RecipientFacet: FacetCreditOwner, EventRef: "invoice:" + invoice + ":stage", Engine: EnginePurposeMatrix,
	}
}

// Golden fixture: the source-evidenced purpose matrix path
// (class-sn-purpose-commission.php:606): 1,000,000 IRT x 2.5% = 25,000.00.
func TestGoldenPurposeMatrixStagePercent(t *testing.T) {
	out := Calculate(matrixInput(t, "501", 1, "product_sale", "seller", "u7"))
	if out.Eligibility != Eligible || out.Amount == nil || out.Amount.String() != "25000.00 IRT" || out.Coverage != "FULL" {
		t.Fatalf("out = %+v", out)
	}
	if *out.EntitlementKey != "inv:501|stage:1|purpose:product_sale|facet:seller|ctx:PURPOSE_MATRIX" {
		t.Fatalf("key = %s", *out.EntitlementKey)
	}
	if *out.Recipient.UserKey != "u7" || out.PolicyVer != 1 || out.InputSHA == "" || out.EventRef == "" || out.Rate.String() != "2.5000" {
		t.Fatalf("snapshot = %+v", out)
	}
	// Deterministic: identical input -> identical output and input hash.
	again := Calculate(matrixInput(t, "501", 1, "product_sale", "seller", "u7"))
	if again.InputSHA != out.InputSHA || again.Amount.Hundredths != out.Amount.Hundredths {
		t.Fatal("calculation is not deterministic")
	}
}

// Golden fixtures for the half-away rounding on the exact product.
func TestGoldenRounding(t *testing.T) {
	for _, c := range []struct{ base, rate, want string }{
		{"333.33", "1.5", "5.00"},   // 4.99995 -> 5.00
		{"100", "0.0050", "0.01"},   // 0.005 -> 0.01 (half away)
		{"100", "0.0049", "0.00"},   // 0.0049 -> 0.00
		{"-100", "0.0050", "-0.01"}, // half away from zero
		{"123456789.12", "12.3456", "15241481.36"},
	} {
		a, err := PercentOf(*ex(t, c.base), *rt(t, c.rate), RoundingLegacyHalfAwayTwoDP)
		if err != nil || a.Decimal() != c.want {
			t.Fatalf("%s x %s%% = %v %v, want %s", c.base, c.rate, a.Decimal(), err, c.want)
		}
	}
}

// Golden fixture: the E2 / E3 rule path round(max(0, fixed + amount*p/100), 2).
func TestGoldenRuleInvoicePercentFixed(t *testing.T) {
	in := Input{Identity: EntitlementIdentity{Kind: KindInvoice, InvoiceKey: "9", Purpose: "invoice_rule", ShareFacet: "seller", Context: string(EngineDryRunPosting)},
		FunctionID: FnRuleInvoicePercentFixedV1, PolicyKey: "rule:R1", PolicyVer: 1, Base: ex(t, "2000000"), BaseKind: BaseInvoiceTotal,
		Rate: rt(t, "3"), Fixed: ex(t, "-70000"), Unit: money.IRT, Rounding: RoundingLegacyHalfAwayTwoDP,
		Attribution: Attribution{FacetOriginalSeller: {State: AttrSnapshot, UserKey: key("u1"), Basis: "dry run item seller_user_id"}}, RecipientFacet: FacetOriginalSeller,
		Engine: EngineDryRunPosting}
	out := Calculate(in)
	if out.Amount == nil || out.Amount.Decimal() != "0.00" || out.Eligibility != Ineligible {
		t.Fatalf("floor at zero: %+v", out)
	}
	in.Fixed = ex(t, "5000")
	if out = Calculate(in); out.Amount.Decimal() != "65000.00" || out.Eligibility != Eligible {
		t.Fatalf("percent + fixed: %+v", out)
	}
}

func TestParseDecimalIsExact(t *testing.T) {
	if _, err := ParseDecimal("1.005", 2); !errors.Is(err, ErrNotExact) {
		t.Fatal("a third decimal must not be rounded away")
	}
	if v, err := ParseDecimal("1.5000", 2); err != nil || v != 150 {
		t.Fatalf("trailing zeros: %d %v", v, err)
	}
	for _, bad := range []string{"", "1e3", "abc", "1,5", "."} {
		if _, err := ParseDecimal(bad, 2); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if _, err := ParseDecimal("99999999999999999999", 2); !errors.Is(err, ErrOverflow) {
		t.Fatal("overflow must fail")
	}
}

// 1: the same entitlement processed twice has one key.
// 2 / 3 / 4: different stages, purposes and share facets are distinct keys.
// 6: a recipient change never changes the key.
func TestEntitlementIdentity(t *testing.T) {
	k := func(in Input) string { return *Calculate(in).EntitlementKey }
	base := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	if k(base) != k(matrixInput(t, "501", 1, "product_sale", "seller", "u7")) {
		t.Fatal("1: same entitlement must have the same key")
	}
	seen := map[string]bool{k(base): true}
	for _, in := range []Input{matrixInput(t, "501", 2, "product_sale", "seller", "u7"), matrixInput(t, "501", 1, "assessment_fee", "seller", "u7"),
		matrixInput(t, "501", 1, "product_sale", "supervisor", "u7")} {
		if seen[k(in)] {
			t.Fatalf("2/3/4: key collision %s", k(in))
		}
		seen[k(in)] = true
	}
	if k(matrixInput(t, "501", 1, "product_sale", "seller", "u99")) != k(base) {
		t.Fatal("6: a recipient change must not mint a new entitlement")
	}
}

// 5: a hierarchy change after the event changes nothing: the recipient is
// the event-time snapshot, never a current hierarchy lookup (there is none).
func TestHierarchyChangeAfterEvent(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "supervisor", "sup-at-event")
	in.Attribution[FacetCurrentOrgAssignment] = FacetValue{State: AttrSnapshot, UserKey: key("sup-today"), Basis: "current HR assignment"}
	out := Calculate(in)
	if *out.Recipient.UserKey != "sup-at-event" {
		t.Fatal("5: recipient must stay the event-time credit owner")
	}
}

// 7: a missing attribution snapshot blocks; it is never filled.
func TestMissingAttributionBlocks(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	in.Attribution = Attribution{}
	out := Calculate(in)
	if out.Eligibility != Blocked || out.Amount != nil || !hasBlock(out, BlockAttributionUnknown) || out.Recipient.UserKey != nil {
		t.Fatalf("7: %+v", out)
	}
	in.Attribution = Attribution{FacetCreditOwner: {State: AttrConflict, Basis: "meta recipient != wallet holder"}}
	if out := Calculate(in); !hasBlock(out, BlockAttributionConflict) {
		t.Fatalf("conflict: %+v", out)
	}
}

// 8: conflicting / unknown engine provenance blocks.
func TestEngineRoutingUnknownBlocks(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	in.Engine = EngineUnknown
	if out := Calculate(in); !hasBlock(out, BlockEngineRoutingUnknown) || out.Amount != nil {
		t.Fatalf("8: %+v", out)
	}
}

// 9: a non-invoice entitlement has no approved key (OPD-09).
func TestNonInvoiceEntitlementBlocked(t *testing.T) {
	in := matrixInput(t, "501", 1, "daily_attendance", "seller", "u7")
	in.Identity = EntitlementIdentity{Kind: KindNonInvoice, Purpose: "daily_attendance", ShareFacet: "seller", Context: "DRY_RUN"}
	out := Calculate(in)
	if out.EntitlementKey != nil || !hasBlock(out, BlockNonInvoiceKey) || out.Eligibility != Blocked {
		t.Fatalf("9: %+v", out)
	}
	if _, err := in.Identity.Key(); !errors.Is(err, ErrNoApprovedKey) {
		t.Fatal("9: Key must refuse")
	}
}

// 10: an invalid effective interval is rejected; an open interval is UNKNOWN.
func TestEffectiveInterval(t *testing.T) {
	d := func(s string) *time.Time { v, _ := time.Parse("2006-01-02", s); return &v }
	p := Policy{PolicyKey: "rule:R1", Purpose: "invoice_rule", ShareFacet: "seller", EligibilityRef: "legacy.e3.rule_match", FunctionID: FnRuleInvoicePercentFixedV1,
		Rate: &Rate{E4: 10000}, Base: BaseInvoiceTotal, Unit: money.IRT, AttributionFacet: FacetOriginalSeller, Scope: "seller", Provenance: "legacy.commission_rule.v1",
		Evidence: EvidenceObservedUnversioned, ApprovalState: PolicyNotApproved, SourceRef: "1", EffectiveFrom: d("2025-02-01"), EffectiveTo: d("2025-01-01")}
	if err := p.Validate(); !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("10: %v", err)
	}
	p.EffectiveTo = d("2025-03-01")
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if in, ok := p.Covers(*d("2025-02-15")); !in || !ok {
		t.Fatal("covered date")
	}
	p.EffectiveTo = nil
	if _, ok := p.Covers(*d("2025-02-15")); ok {
		t.Fatal("an open bound is not provable coverage")
	}
	p.ApprovalState = "APPROVED"
	if p.Validate() == nil {
		t.Fatal("Slice 13 approves no policy")
	}
}

// 11: a policy version change after a calculation leaves it pinned.
func TestPolicyVersionPinned(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	first := Calculate(in)
	in.PolicyVer, in.Rate = 2, rt(t, "3")
	second := Calculate(in)
	if first.PolicyVer != 1 || first.Amount.Decimal() != "25000.00" || second.PolicyVer != 2 || first.InputSHA == second.InputSHA {
		t.Fatalf("11: %+v / %+v", first, second)
	}
	if *first.EntitlementKey != *second.EntitlementKey {
		t.Fatal("11: a policy change does not mint a new entitlement")
	}
}

// 12: unsupported pro-rata component allocation (D-36) blocks.
func TestProRataBlocked(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_star_upgrade", "seller", "u7")
	in.FunctionID, in.BaseKind = FnComponentProRata, BaseComponentLine
	if out := Calculate(in); !hasBlock(out, BlockComponentAllocation) || out.Amount != nil {
		t.Fatalf("12: %+v", out)
	}
	if _, ok := Registry()[FnComponentProRata]; ok {
		t.Fatal("12: pro-rata must not be registered")
	}
	if _, ok := Registry()[FnDailyAttendance]; ok {
		t.Fatal("daily attendance must not be registered")
	}
}

// 13: different units never mix.
func TestUnitMismatchBlocks(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	b, _ := ParseExact("1000", money.IRR)
	in.Base = &b
	if out := Calculate(in); !hasBlock(out, BlockUnitMismatch) {
		t.Fatalf("13: %+v", out)
	}
	if _, err := AddExact(*ex(t, "1"), b); !errors.Is(err, ErrUnitMismatch) {
		t.Fatal("13: add across units")
	}
	var tot Totals
	_ = tot.Add(ex(t, "1.50"))
	_ = tot.Add(&b)
	_ = tot.Add(nil)
	if tot.ByUnit["IRT"] != 150 || tot.ByUnit["IRR"] != 100000 || tot.Missing != 1 {
		t.Fatalf("totals by unit: %+v", tot)
	}
}

// 14: a missing rounding policy blocks (no silent default).
func TestMissingRoundingBlocks(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	in.Rounding = RoundingNone
	if out := Calculate(in); !hasBlock(out, BlockRoundingMissing) || out.Amount != nil {
		t.Fatalf("14: %+v", out)
	}
	if _, err := PercentOf(*ex(t, "1"), Rate{E4: 1}, "BANKERS"); err == nil {
		t.Fatal("14: unknown rounding accepted")
	}
}

// 15 / 16 / 17 and the decision-blocked commands.
func TestLifecycle(t *testing.T) {
	if CanTransition(StateGenerated, StatePosted) || CanTransition(StateGenerated, StatePostingRequested) {
		t.Fatal("15: generated is not approved")
	}
	if CanTransition(StateApproved, StatePosted) {
		t.Fatal("16: approved is not posted")
	}
	if CanTransition(StatePostingRequested, StateSettled) || !CanTransition(StatePosted, StateSettled) {
		t.Fatal("17: posted is not settled")
	}
	if err := History(StateGenerated, StateRejected, StateApproved, StatePostingRequested, StatePosted); err != nil {
		t.Fatalf("rejected-then-approved history is evidenced: %v", err)
	}
	if err := History(StateGenerated, StateApproved, StateRejected); !errors.Is(err, ErrTransition) {
		t.Fatal("approved -> rejected is not evidenced")
	}
	if err := Approve(StateGenerated); !errors.Is(err, ErrCommandBlocked) {
		t.Fatal("approval must stay blocked")
	}
	if err := RequestPosting(StateApproved); !errors.Is(err, ErrCommandBlocked) {
		t.Fatal("posting must stay blocked")
	}
	if err := RequestPosting(StateGenerated); !errors.Is(err, ErrTransition) {
		t.Fatal("posting a generated run is not even a transition")
	}
	if LegacyRunState(nil) != StateUnknown || LegacyRunState(key("weird")) != StateUnknown || LegacyItemPosting(key("pending")) == StatePosted {
		t.Fatal("legacy mapping")
	}
}

// 19 / 22: reversal references the original; duplicate posting requests are refused.
func TestPostingContract(t *testing.T) {
	amount := *ex(t, "25000")
	k := PostingRequestKey("ent-1", "pk", 1, nil)
	req := PostingRequested{RequestKey: k, EntitlementKey: "ent-1", PolicyKey: "pk", PolicyVer: 1, RunID: "r", RunState: StateApproved, RecipientKey: "u7", Amount: amount}
	var set PostingRequests
	if err := set.Add(req); err != nil {
		t.Fatal(err)
	}
	if err := set.Add(req); !errors.Is(err, ErrDuplicatePosting) {
		t.Fatal("22: duplicate posting request accepted")
	}
	rev := req
	rev.Reverses = &k
	rev.RequestKey = PostingRequestKey("ent-1", "pk", 1, &k)
	if rev.RequestKey == k || set.Add(rev) != nil {
		t.Fatal("19: a reversal is a distinct request that references the original")
	}
	gen := req
	gen.RunState = StateGenerated
	if gen.Validate() == nil {
		t.Fatal("15/16: a generated run may not request posting")
	}
	bad := req
	bad.RequestKey = "x"
	if bad.Validate() == nil {
		t.Fatal("non-deterministic key accepted")
	}
}

func TestParityClasses(t *testing.T) {
	in := matrixInput(t, "501", 1, "product_sale", "seller", "u7")
	out := Calculate(in)
	l := Legacy{SourceKey: "tx1", Engine: EnginePurposeMatrix, EntitlementKey: out.EntitlementKey, Amount: ex(t, "25000"), RecipientKey: key("u7"),
		SnapshotKey: key("u7"), RateSnapshot: rt(t, "2.5"), RateObserved: rt(t, "2.5"), Base: ex(t, "1000000"), Comparable: true}
	if c := Compare(l, &out, false).Class; c != ParityMatch {
		t.Fatalf("match: %s", c)
	}
	m := l
	m.Amount = ex(t, "25000.01")
	if Compare(m, &out, false).Class != ParityAmountMismatch {
		t.Fatal("amount mismatch")
	}
	m = l
	m.RecipientKey = key("u8")
	if Compare(m, &out, false).Class != ParityRecipientMismatch {
		t.Fatal("recipient mismatch")
	}
	m = l
	m.RateObserved = rt(t, "3")
	if Compare(m, &out, false).Class != ParityPolicyMismatch {
		t.Fatal("policy mismatch")
	}
	if Compare(l, &out, true).Class != ParityDuplicateRisk {
		t.Fatal("duplicate")
	}
	m = l
	m.SnapshotKey = nil
	if Compare(m, &out, false).Class != ParityAttributionUnknown {
		t.Fatal("attribution unknown")
	}
	m = l
	m.Engine = EngineUnknown
	if Compare(m, &out, false).Class != ParityEngineRoutingUnknown {
		t.Fatal("engine")
	}
	m = l
	m.Base = nil
	if Compare(m, &out, false).Class != ParitySourceIncomplete {
		t.Fatal("24: missing legacy data must be explicit uncertainty")
	}
	m = l
	m.Comparable, m.NotComparableBy = false, "D-36"
	if r := Compare(m, &out, false); r.Class != ParityNotComparable || !strings.Contains(strings.Join(r.Reasons, ","), "D-36") {
		t.Fatal("not comparable")
	}
}

func hasBlock(o Output, b Block) bool {
	for _, x := range o.Blocks {
		if x == b {
			return true
		}
	}
	return false
}
