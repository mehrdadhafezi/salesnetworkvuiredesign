package domain

import (
	"errors"
	"strings"
	"testing"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	financeapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

func s(v string) *string { return &v }
func n(v int64) *int64   { return &v }

func entry(key string, lines ...Line) Entry {
	return Entry{BusinessKey: key, Basis: CashShadow, Unit: money.IRT, EffectKind: "TEST", Provenance: "TEST",
		SourceRefs: []SourceRef{{Domain: "t", Ref: "1", Role: "subject"}}, TimeEvidence: TimeAbsent, Status: EntryConsistent, Lines: lines}
}

func journal() *Journal {
	j := NewJournal()
	j.Account(Account{Key: "a", Category: CashBankClearing})
	j.Account(Account{Key: "b", Category: RevenueByProductKind})
	return j
}

var k1 = BusinessKey("payments", "attempt", "x1", "cash_receipt", "IRT", CashShadow)
var k2 = BusinessKey("payments", "attempt", "x2", "cash_receipt", "IRT", CashShadow)

// Double entry: balanced accepted, unbalanced / one-sided / non-positive
// rejected; cross-unit rejected; duplicate business key rejected.
func TestDoubleEntryInvariant(t *testing.T) {
	j := journal()
	if err := j.Add(entry(k1, Line{"a", DebitSide, 100, nil}, Line{"b", CreditSide, 100, nil})); err != nil {
		t.Fatalf("balanced entry rejected: %v", err)
	}
	for name, e := range map[string]Entry{
		"unbalanced":      entry(k2, Line{"a", DebitSide, 100, nil}, Line{"b", CreditSide, 99, nil}),
		"one-sided":       entry(k2, Line{"a", DebitSide, 100, nil}),
		"zero":            entry(k2, Line{"a", DebitSide, 0, nil}, Line{"b", CreditSide, 0, nil}),
		"negative":        entry(k2, Line{"a", DebitSide, -5, nil}, Line{"b", CreditSide, -5, nil}),
		"bad side":        entry(k2, Line{"a", "LEFT", 5, nil}, Line{"b", CreditSide, 5, nil}),
		"no key":          entry("random-uuid", Line{"a", DebitSide, 5, nil}, Line{"b", CreditSide, 5, nil}),
		"unknown account": entry(k2, Line{"zz", DebitSide, 5, nil}, Line{"b", CreditSide, 5, nil}),
	} {
		if err := j.Add(e); err == nil {
			t.Fatalf("%s entry accepted", name)
		}
	}
	if err := j.Add(entry(k1, Line{"a", DebitSide, 7, nil}, Line{"b", CreditSide, 7, nil})); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("duplicate business key: %v", err)
	}
	// Cross-unit: even when each unit balances on its own, one entry may not
	// carry two units.
	if _, err := SingleUnit([]UnitLine{{Line{"a", DebitSide, 10, nil}, money.IRT}, {Line{"b", CreditSide, 10, nil}, money.IRT},
		{Line{"a", DebitSide, 100, nil}, money.IRR}, {Line{"b", CreditSide, 100, nil}, money.IRR}}); !errors.Is(err, ErrCrossUnit) {
		t.Fatalf("cross-unit entry: %v", err)
	}
	if u, err := SingleUnit([]UnitLine{{Line{"a", DebitSide, 10, nil}, money.IRR}, {Line{"b", CreditSide, 10, nil}, money.IRR}}); err != nil || u != money.IRR {
		t.Fatal("single-unit entry")
	}
	if len(j.Entries) != 1 {
		t.Fatal("rejected entries must leave the journal unchanged")
	}
}

// Reversal: must point to an existing earlier entry, not itself, at most
// once, same unit; the original is never changed and both remain.
func TestReversal(t *testing.T) {
	j := journal()
	orig := entry(k1, Line{"a", DebitSide, 100, nil}, Line{"b", CreditSide, 100, nil})
	if err := j.Add(orig); err != nil {
		t.Fatal(err)
	}
	rev := entry(k2, Line{"b", DebitSide, 100, nil}, Line{"a", CreditSide, 100, nil})
	rev.ReversesKey = s(k1)
	if err := j.Add(rev); err != nil {
		t.Fatalf("valid reversal: %v", err)
	}
	k3 := BusinessKey("payments", "attempt", "x3", "cash_receipt", "IRT", CashShadow)
	again := entry(k3, Line{"b", DebitSide, 100, nil}, Line{"a", CreditSide, 100, nil})
	again.ReversesKey = s(k1)
	if err := j.Add(again); !errors.Is(err, ErrInvalidReverse) {
		t.Fatalf("double reversal: %v", err)
	}
	again.ReversesKey = s(k3)
	if err := j.Add(again); !errors.Is(err, ErrInvalidReverse) {
		t.Fatalf("self reversal: %v", err)
	}
	again.ReversesKey = s(BusinessKey("payments", "attempt", "x9", "cash_receipt", "IRT", CashShadow))
	if err := j.Add(again); !errors.Is(err, ErrInvalidReverse) {
		t.Fatalf("reversal of a missing / later entry: %v", err)
	}
	if got, _ := j.Entry(k1); got.Lines[0].Amount != 100 || got.ReversesKey != nil || len(j.Entries) != 2 {
		t.Fatal("original must stay unchanged and both entries remain")
	}
	if net, _ := j.AccountBalance("a", CashShadow, money.IRT).Net(DebitSide); net != 0 {
		t.Fatalf("net after reversal = %d", net)
	}
}

// Money: exact parse only; absent is not zero; no rounding; unit required.
func TestParseExact(t *testing.T) {
	for raw, want := range map[string]int64{"100": 100, "100.00": 100, "-120.00": -120, "0": 0} {
		if p := ParseExact(s(raw), s("IRT")); p.Repr != "REPRESENTABLE" || *p.Amount != want || p.Unit.String() != "IRT" {
			t.Fatalf("%s: %+v", raw, p)
		}
	}
	for _, raw := range []string{"10.5", "10.50", "0.5", "1,000", "abc", "99999999999999999999"} {
		if p := ParseExact(s(raw), s("IRT")); p.Repr != "UNREPRESENTABLE" || p.Amount != nil {
			t.Fatalf("%s must be UNREPRESENTABLE (never rounded): %+v", raw, p)
		}
	}
	if p := ParseExact(nil, s("IRT")); p.Repr != "ABSENT" || p.Amount != nil {
		t.Fatal("absent must not be zero")
	}
	if p := ParseExact(s("100"), nil); p.Repr != "UNREPRESENTABLE" || p.Unit != nil {
		t.Fatal("unknown unit")
	}
	if a, b := ParseExact(s("100"), s("IRR")), ParseExact(s("100"), s("IRT")); *a.Amount != *b.Amount || *a.Unit == *b.Unit {
		t.Fatal("the same digits stay distinct units (no conversion)")
	}
}

func inv(id string, link string, cust *string) billingapi.InvoiceFact {
	return billingapi.InvoiceFact{InvoiceID: id, SourceSpec: "invoice.v2", SourceKey: id[len(id)-2:], Presence: "PRESENT", CustomerLink: link, CustomerID: cust}
}

func evidence(unit, class string, v int64, paid, remaining *int64) billingapi.AmountEvidence {
	sum := billingapi.PaymentUnitSum{Amount: n(v), State: "COMPLETE", Rows: 1}
	pr, rr := "ABSENT", "ABSENT"
	if paid != nil {
		pr = "REPRESENTABLE"
	}
	if remaining != nil {
		rr = "REPRESENTABLE"
	}
	return billingapi.AmountEvidence{Unit: unit, Declared: map[string]*int64{"FINAL_TOTAL": n(v)}, ItemSum: sum, StageSum: sum, Class: class,
		LegacyPaid: paid, LegacyPaidRepr: pr, LegacyRemaining: remaining, LegacyRemainingRepr: rr}
}

func attempt(id, invID, target, prov string, amount int64, unit string) paymentsapi.AttemptFact {
	a := paymentsapi.AttemptFact{AttemptID: id, InvoiceID: s(invID), InvoiceLink: "RESOLVED", Presence: "PRESENT", Provenance: prov,
		Amount: n(amount), Unit: s(unit), AmountRepr: "REPRESENTABLE"}
	if target != "" {
		a.StateTarget = s(target)
	}
	return a
}

const (
	i1 = "00000000-0000-4000-8000-000000000001"
	i2 = "00000000-0000-4000-8000-000000000002"
	i3 = "00000000-0000-4000-8000-000000000003"
	i4 = "00000000-0000-4000-8000-000000000004"
	c1 = "00000000-0000-4000-8000-0000000000c1"
)

func irt(v string) financeapi.RawAmount { u := "IRT"; return financeapi.RawAmount{Raw: &v, Unit: &u} }

func scenario() Sources {
	return Sources{
		Invoices: []billingapi.InvoiceFact{inv(i1, "RESOLVED", s(c1)), inv(i2, "CONFLICT", nil), inv(i3, "UNKNOWN", nil), inv(i4, "UNKNOWN", nil)},
		Evidence: map[string][]billingapi.AmountEvidence{
			i1: {evidence("IRT", Match, 1000, n(1000), n(0))},
			i2: {evidence("IRT", "MISMATCH", 2000, n(0), n(2000))},
			i3: {evidence("IRR", Match, 5000, n(5000), n(0)), evidence("IRT", Match, 70, n(0), n(70))},
			i4: {evidence("IRT", Match, 300, n(0), n(300))},
		},
		Attempts: []paymentsapi.AttemptFact{
			attempt("a1", i1, "ACCEPTED", paymentsapi.ProvGatewayAuto, 1000, "IRT"),
			attempt("a2", i2, "ACCEPTED", paymentsapi.ProvLegacyManualApproval, 500, "IRT"),
			attempt("a3", i3, "ACCEPTED", paymentsapi.ProvGatewayAuto, 5000, "IRR"),
			attempt("a4", i4, "", "UNKNOWN", 300, "IRT"),
			attempt("a5", i4, "PENDING", "NOT_APPROVED", 300, "IRT"),
		},
		Wallets: &financeapi.WalletEvidence{SnapshotID: "syn", WalletStatus: financeapi.SourceAvailable, TransactionStatus: financeapi.SourceAvailable,
			Wallets: []financeapi.LegacyWallet{
				{WalletKey: "1", SubjectKey: "5", WalletTypeRaw: "seller", Category: financeapi.WalletCommission, Balance: irt("50.00"), TotalCredit: irt("150"), TotalDebit: irt("100")},
				{WalletKey: "2", SubjectKey: "6", WalletTypeRaw: "supervisor", Category: financeapi.WalletCommission, Balance: irt("80"), TotalCredit: irt("80"), TotalDebit: irt("0")},
				{WalletKey: "3", SubjectKey: "7", WalletTypeRaw: "mystery", Category: financeapi.WalletUnclassified, Balance: irt("10"), TotalCredit: irt("10"), TotalDebit: irt("0")},
			},
			Transactions: []financeapi.LegacyWalletTx{
				wtx("10", "1", financeapi.Credit, "100", "seller_commission_reversed", financeapi.RouteLegacyRules, nil),
				wtx("11", "1", financeapi.Debit, "100", "seller_commission_reversal", financeapi.RouteLegacyRules, s("10")),
				wtx("12", "1", financeapi.Credit, "50", "purpose_commission", financeapi.RoutePurposeMatrix, nil),
				wtx("20", "2", financeapi.Credit, "40", "supervisor_commission", financeapi.RouteLegacyRules, nil),
				wtx("21", "2", financeapi.Credit, "12.50", "supervisor_commission", financeapi.RouteLegacyRules, nil),
				wtx("30", "3", financeapi.Credit, "10", "manual_thing", financeapi.RouteUnresolved, nil),
			}},
		BillingStatus: financeapi.SourceAvailable, PaymentsStatus: financeapi.SourceAvailable,
	}
}

func wtx(id, wallet, dir, amount, typ, route string, reverses *string) financeapi.LegacyWalletTx {
	return financeapi.LegacyWalletTx{TxKey: id, WalletKey: wallet, SubjectKey: "5", Direction: dir, TypeRaw: typ, Route: route,
		Status: financeapi.StatusApproved, Amount: irt(amount), InvoiceKey: s("2001"), ReversesTxKey: reverses,
		RewrittenInPlace: strings.HasSuffix(typ, "_reversed"), IdempotencyKeyPresent: true, SourceTimeRaw: s("2025-03-03 10:00:00")}
}

func recon(d *Derivation, ref, basis string) Reconciliation {
	for _, r := range d.Reconciliations {
		if r.SubjectRef == ref && r.Basis == basis {
			return r
		}
	}
	return Reconciliation{}
}

func reconUnit(d *Derivation, ref, basis, unit string) Reconciliation {
	for _, r := range d.Reconciliations {
		if r.SubjectRef == ref && r.Basis == basis && r.Unit == unit {
			return r
		}
	}
	return Reconciliation{}
}

func TestDerive(t *testing.T) {
	d, err := Derive(scenario())
	if err != nil {
		t.Fatal(err)
	}
	j := d.Journal
	for _, e := range j.Entries {
		if err := e.Validate(); err != nil {
			t.Fatalf("%s: %v", e.BusinessKey, err)
		}
		if e.Provenance == "FINANCE_REVIEWED" {
			t.Fatal("nothing is Finance-reviewed in Slice 9")
		}
	}
	// D-41: the same sources give separate ACCRUAL and CASH bases.
	if _, ok := j.Entry(BusinessKey("billing", "invoice", i1, "receivable_recognition", "IRT", AccrualShadow)); !ok {
		t.Fatal("accrual recognition for a MATCH invoice")
	}
	if _, ok := j.Entry(BusinessKey("billing", "invoice", i2, "receivable_recognition", "IRT", AccrualShadow)); ok {
		t.Fatal("an ambiguous invoice total must not be posted")
	}
	ac, ok1 := j.Entry(BusinessKey("payments", "attempt", "a1", "cash_receipt", "IRT", AccrualShadow))
	ca, ok2 := j.Entry(BusinessKey("payments", "attempt", "a1", "cash_receipt", "IRT", CashShadow))
	if !ok1 || !ok2 || ac.Basis == ca.Basis {
		t.Fatal("cash receipt in both bases, separately")
	}
	// Gateway auto keeps its provenance and its own clearing account.
	if ca.Provenance != paymentsapi.ProvGatewayAuto || ca.Lines[0].AccountKey != CategoryKey(CashBankClearing, "provenance", paymentsapi.ProvGatewayAuto) {
		t.Fatalf("gateway-auto provenance: %+v", ca)
	}
	// Unknown / pending attempts are never cash.
	for _, id := range []string{"a4", "a5"} {
		if _, ok := j.Entry(BusinessKey("payments", "attempt", id, "cash_receipt", "IRT", CashShadow)); ok {
			t.Fatalf("%s counted as cash", id)
		}
	}
	// IRR and IRT: separate entries, separate balances, no conversion.
	irr, okr := j.Entry(BusinessKey("payments", "attempt", "a3", "cash_receipt", "IRR", CashShadow))
	if !okr || irr.Unit != money.IRR || irr.Lines[0].Amount != 5000 {
		t.Fatal("IRR receipt kept in IRR")
	}
	gw := CategoryKey(CashBankClearing, "provenance", paymentsapi.ProvGatewayAuto)
	if net, _ := j.AccountBalance(gw, CashShadow, money.IRT).Net(DebitSide); net != 1000 {
		t.Fatalf("IRT gateway cash = %d", net)
	}
	if net, _ := j.AccountBalance(gw, CashShadow, money.IRR).Net(DebitSide); net != 5000 {
		t.Fatalf("IRR gateway cash = %d", net)
	}

	// Reconciliation.
	if r := recon(d, i1, AccrualShadow); r.State != Match || *r.Difference != 0 {
		t.Fatalf("accrual i1: %+v", r)
	}
	if r := recon(d, i1, CashShadow); r.State != Match {
		t.Fatalf("cash i1: %+v", r)
	}
	if r := recon(d, i2, AccrualShadow); r.State != Conflict || r.Difference != nil {
		t.Fatalf("ambiguous total must be CONFLICT without a difference: %+v", r)
	}
	if r := recon(d, i2, CashShadow); r.State != Mismatch || *r.Difference != 500 || !has(r.Reasons, ReasonMissingLegacyTransaction) {
		t.Fatalf("cash i2 (legacy paid 0, accepted 500): %+v", r)
	}
	if r := recon(d, i4, AccrualShadow); r.State != Incomplete || r.Difference != nil || !has(r.Reasons, ReasonPaymentStatusUnknown) {
		t.Fatalf("unknown payment state must make the comparison INCOMPLETE: %+v", r)
	}
	if r := reconUnit(d, i3, CashShadow, "IRR"); r.State != Match {
		t.Fatalf("IRR reconciled in IRR: %+v", r)
	}
	if r := reconUnit(d, i3, CashShadow, "IRT"); r.State != Match || *r.Shadow != 0 {
		t.Fatalf("IRT paid 0 is a real zero: %+v", r)
	}
	// Wallet 1: credit 100, its proven reversal, credit 50 = legacy 50.
	w1 := recon(d, "1", WalletMirrorShadow)
	if w1.State != Match || *w1.Shadow != 50 {
		t.Fatalf("wallet 1: %+v", w1)
	}
	rev, _ := j.Entry(BusinessKey("legacy_wallet", "transaction", "11", "wallet_effect", "IRT", WalletMirrorShadow))
	if rev.ReversesKey == nil || *rev.ReversesKey != BusinessKey("legacy_wallet", "transaction", "10", "wallet_effect", "IRT", WalletMirrorShadow) {
		t.Fatal("a provable legacy reversal is linked")
	}
	// Wallet 2: a fractional amount is never rounded → INCOMPLETE.
	if w2 := recon(d, "2", WalletMirrorShadow); w2.State != Incomplete || w2.Difference != nil || !has(w2.Reasons, ReasonUnrepresentableAmount) {
		t.Fatalf("wallet 2: %+v", w2)
	}
	// Wallet 3: unclassified wallet + unresolved route stay unresolved (OPD-09).
	w3 := recon(d, "3", WalletMirrorShadow)
	if w3.State != Match || !has(w3.Reasons, ReasonEntitlementRoute) || !has(w3.Reasons, ReasonCommissionRoute) {
		t.Fatalf("wallet 3: %+v", w3)
	}
	if _, ok := j.Accounts[CategoryKey(LegacyWalletUnclassed, "legacy_wallet", "3")]; !ok {
		t.Fatal("unclassified wallet is not collapsed into commission payable")
	}
	// Determinism.
	d2, _ := Derive(scenario())
	if len(d2.Journal.Entries) != len(j.Entries) {
		t.Fatal("deterministic")
	}
	for i := range j.Entries {
		if j.Entries[i].BusinessKey != d2.Journal.Entries[i].BusinessKey {
			t.Fatal("deterministic order")
		}
	}
}

// Wallet reconciliation states: known different balance → MISMATCH with the
// exact difference; missing source → UNKNOWN, not zero; unprovable reversal
// is not linked; duplicates are CONFLICT and never deduplicated.
func TestWalletReconciliation(t *testing.T) {
	src := scenario()
	src.Wallets.Wallets[0].Balance = irt("70")
	src.Wallets.Transactions = append(src.Wallets.Transactions,
		wtx("40", "3", financeapi.Debit, "5", "manual_thing", financeapi.RouteUnresolved, s("999")))
	d, err := Derive(src)
	if err != nil {
		t.Fatal(err)
	}
	if w := recon(d, "1", WalletMirrorShadow); w.State != Mismatch || *w.Difference != -20 || *w.Legacy.Amount != 70 || !has(w.Reasons, ReasonMissingLegacyTransaction) {
		t.Fatalf("mismatch with exact difference: %+v", w)
	}
	e, _ := d.Journal.Entry(BusinessKey("legacy_wallet", "transaction", "40", "wallet_effect", "IRT", WalletMirrorShadow))
	if e.ReversesKey != nil || e.Status != EntryUnresolvedCorrection {
		t.Fatalf("unprovable reversal must not be linked: %+v", e)
	}

	dup := scenario()
	dup.Wallets.Transactions = append(dup.Wallets.Transactions, wtx("13", "1", financeapi.Credit, "50", "purpose_commission", financeapi.RoutePurposeMatrix, nil))
	dup.Wallets.Transactions[2].IdempotencyKeyPresent = false
	dup.Wallets.Transactions[6].IdempotencyKeyPresent = false
	dd, _ := Derive(dup)
	if w := recon(dd, "1", WalletMirrorShadow); w.State != Conflict || *w.Shadow != 100 || !has(w.Reasons, ReasonDuplicateSource) {
		t.Fatalf("duplicates are CONFLICT, both kept: %+v", w)
	}

	none := scenario()
	none.Wallets = nil
	dn, _ := Derive(none)
	for _, r := range dn.Reconciliations {
		if r.SubjectDomain == "legacy_wallet" {
			t.Fatal("no wallet evidence: no wallet reconciliation is invented")
		}
	}
	if !hasFinding(dn, ReasonMissingSource, Unknown) {
		t.Fatal("missing wallet source is reported UNKNOWN")
	}
	partial := scenario()
	partial.Wallets.TransactionStatus = financeapi.SourcePartial
	dp, _ := Derive(partial)
	if w := recon(dp, "1", WalletMirrorShadow); w.State != Incomplete || w.Difference != nil {
		t.Fatalf("partial source must be INCOMPLETE: %+v", w)
	}
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func hasFinding(d *Derivation, reason, state string) bool {
	for _, f := range d.Findings {
		if f.Reason == reason && f.State == state {
			return true
		}
	}
	return false
}

func TestEveryReasonMapsToAKernelCode(t *testing.T) {
	for r, c := range Reasons {
		if !c.Valid() {
			t.Fatalf("%s → %s", r, c)
		}
	}
}

// Review fixes: a CONFLICT built on incomplete evidence carries no
// difference; an agreed zero invoice total is resolved (it reconciles) but
// posts nothing.
func TestConflictOnIncompleteEvidenceHasNoDifference(t *testing.T) {
	for name, mutate := range map[string]func(*Sources){
		"partial transactions": func(s *Sources) { s.Wallets.TransactionStatus = financeapi.SourcePartial },
		"skipped transaction": func(s *Sources) {
			s.Wallets.Transactions = append(s.Wallets.Transactions, wtx("14", "1", financeapi.Credit, "7.50", "purpose_commission", financeapi.RoutePurposeMatrix, nil))
		},
	} {
		src := scenario()
		src.Wallets.Transactions = append(src.Wallets.Transactions, wtx("13", "1", financeapi.Credit, "50", "purpose_commission", financeapi.RoutePurposeMatrix, nil))
		src.Wallets.Transactions[2].IdempotencyKeyPresent = false
		src.Wallets.Transactions[6].IdempotencyKeyPresent = false
		mutate(&src)
		d, err := Derive(src)
		if err != nil {
			t.Fatal(err)
		}
		if w := recon(d, "1", WalletMirrorShadow); w.State != Conflict || w.Difference != nil || !has(w.Reasons, ReasonDuplicateSource) {
			t.Fatalf("%s: conflict on incomplete evidence must carry no difference: %+v", name, w)
		}
	}
}

func TestAgreedZeroTotalReconciles(t *testing.T) {
	src := scenario()
	src.Evidence[i4] = []billingapi.AmountEvidence{evidence("IRT", Match, 0, n(0), n(0))}
	src.Attempts = src.Attempts[:3] // no unknown attempt on i4
	d, err := Derive(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Journal.Entry(BusinessKey("billing", "invoice", i4, "receivable_recognition", "IRT", AccrualShadow)); ok {
		t.Fatal("a zero total posts nothing")
	}
	if r := recon(d, i4, AccrualShadow); r.State != Match || r.Shadow == nil || *r.Shadow != 0 || *r.Difference != 0 {
		t.Fatalf("an agreed zero total reconciles as a real zero: %+v", r)
	}
}
