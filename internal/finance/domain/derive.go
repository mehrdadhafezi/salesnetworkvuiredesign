package domain

import (
	"sort"
	"strconv"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	financeapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// Difference reasons. Each refines one frozen kernel reason code (the
// envelope taxonomy); they are not a parallel envelope reason system.
const (
	ReasonMissingSource            = "MISSING_SOURCE"
	ReasonUnknownUnit              = "UNKNOWN_UNIT"
	ReasonUnrepresentableAmount    = "UNREPRESENTABLE_AMOUNT"
	ReasonUnresolvedInvoiceTotal   = "UNRESOLVED_INVOICE_TOTAL"
	ReasonPaymentStatusUnknown     = "PAYMENT_STATUS_UNKNOWN"
	ReasonMissingLegacyTransaction = "MISSING_LEGACY_TRANSACTION"
	ReasonDuplicateSource          = "DUPLICATE_SOURCE"
	ReasonCrossUnitForbidden       = "CROSS_UNIT_COMPARISON_FORBIDDEN"
	ReasonEntitlementRoute         = "ENTITLEMENT_ROUTE_UNRESOLVED"
	ReasonCommissionRoute          = "COMMISSION_ROUTE_UNRESOLVED"
	ReasonUnresolvedInvoiceLink    = "UNRESOLVED_INVOICE_LINK"
	ReasonUnresolvedCorrection     = "UNRESOLVED_CORRECTION"
	ReasonWalletTxStatusUnknown    = "WALLET_TX_STATUS_UNKNOWN"
	ReasonDirectionUnknown         = "DIRECTION_UNKNOWN"
	ReasonLegacyTotalsInconsistent = "LEGACY_TOTALS_INCONSISTENT"
	ReasonLegacyHistoryRewritten   = "LEGACY_HISTORY_REWRITTEN"
	ReasonLifecycleNotEvaluated    = "INVOICE_LIFECYCLE_NOT_EVALUATED"
	ReasonOrphanLegacyTransaction  = "ORPHAN_LEGACY_TRANSACTION"
)

// Reasons maps every difference reason to its kernel reason code.
var Reasons = map[string]reasoncodes.Code{
	ReasonMissingSource:            reasoncodes.SourceUnavailable,
	ReasonUnknownUnit:              reasoncodes.CoverageIncomplete,
	ReasonUnrepresentableAmount:    reasoncodes.CoverageIncomplete,
	ReasonUnresolvedInvoiceTotal:   reasoncodes.PolicyUnresolved,
	ReasonPaymentStatusUnknown:     reasoncodes.OutcomeUnknown,
	ReasonMissingLegacyTransaction: reasoncodes.CoverageIncomplete,
	ReasonDuplicateSource:          reasoncodes.LinkageConflict,
	ReasonCrossUnitForbidden:       reasoncodes.PolicyUnresolved,
	ReasonEntitlementRoute:         reasoncodes.PolicyUnresolved,
	ReasonCommissionRoute:          reasoncodes.PolicyUnresolved,
	ReasonUnresolvedInvoiceLink:    reasoncodes.LinkageConflict,
	ReasonUnresolvedCorrection:     reasoncodes.LinkageConflict,
	ReasonWalletTxStatusUnknown:    reasoncodes.OutcomeUnknown,
	ReasonDirectionUnknown:         reasoncodes.CoverageIncomplete,
	ReasonLegacyTotalsInconsistent: reasoncodes.CoverageIncomplete,
	ReasonLegacyHistoryRewritten:   reasoncodes.CoverageIncomplete,
	ReasonLifecycleNotEvaluated:    reasoncodes.PolicyUnresolved,
	ReasonOrphanLegacyTransaction:  reasoncodes.LinkageConflict,
}

// Finding / reconciliation states. MATCH and MISMATCH are reconciliation
// outcomes only; NOTICE is an informational finding.
const (
	Match          = "MATCH"
	Mismatch       = "MISMATCH"
	Unknown        = "UNKNOWN"
	Incomplete     = "INCOMPLETE"
	Conflict       = "CONFLICT"
	NotRecoverable = "NOT_RECOVERABLE"
	Notice         = "NOTICE"
)

// Legacy figures reconciled.
const (
	FigureWalletBalance    = "WALLET_BALANCE"
	FigureInvoiceRemaining = "INVOICE_REMAINING"
	FigureInvoicePaid      = "INVOICE_PAID"
)

// Finding is one explained derivation gap (no entry was fabricated for it).
type Finding struct {
	SubjectDomain, SubjectRef string
	Basis                     *string
	Reason, State, Detail     string
}

// LegacyBalance is one legacy wallet balance as evidence (never truth).
type LegacyBalance struct {
	WalletKey, HolderKey, WalletTypeRaw, Category string
	HolderUserID                                  *string
	Unit                                          *money.Unit
	BalanceRaw                                    *string
	Balance, TotalCredit, TotalDebit              Parsed
	SnapshotID                                    string
}

// Reconciliation is one comparison of a legacy figure with a shadow amount
// for one subject, basis and unit, in one run.
type Reconciliation struct {
	SubjectDomain, SubjectRef string
	AccountKey                *string
	Basis                     string
	Unit                      string // IRR / IRT / UNKNOWN
	Figure                    string
	Legacy                    Parsed
	LegacyRaw                 *string
	Shadow, Difference        *int64
	State                     string
	Reasons                   []string
	Coverage                  string
	Snapshot                  *string
}

// Sources are the normalized inputs of one derivation.
type Sources struct {
	Invoices       []billingapi.InvoiceFact
	Evidence       map[string][]billingapi.AmountEvidence
	Attempts       []paymentsapi.AttemptFact
	Wallets        *financeapi.WalletEvidence
	BillingStatus  string
	PaymentsStatus string
}

// Derivation is the full output of one derivation.
type Derivation struct {
	Journal         *Journal
	Findings        []Finding
	LegacyBalances  []LegacyBalance
	Reconciliations []Reconciliation
}

func strp(s string) *string { return &s }

// unknownProductKind: revenue is never split by a guessed product kind.
const unknownProductKind = "UNRESOLVED"

// Derive builds the shadow journal, findings and reconciliation from the
// sources. It is pure and deterministic: the same sources and derivation
// version produce the same facts. It never fabricates an entry for
// ambiguous evidence; the gap becomes a finding.
func Derive(src Sources) (*Derivation, error) {
	d := &Derivation{Journal: NewJournal()}
	j := d.Journal
	find := func(dom, ref string, basis *string, reason, state, detail string) {
		d.Findings = append(d.Findings, Finding{SubjectDomain: dom, SubjectRef: ref, Basis: basis, Reason: reason, State: state, Detail: detail})
	}
	revenue := j.Account(Account{Key: CategoryKey(RevenueByProductKind, "product_kind", unknownProductKind), Category: RevenueByProductKind,
		SubjectDomain: "product_kind", SubjectRef: unknownProductKind})

	invoices := map[string]billingapi.InvoiceFact{}
	ids := make([]string, 0, len(src.Invoices))
	for _, f := range src.Invoices {
		if f.Presence != "PRESENT" {
			continue
		}
		invoices[f.InvoiceID] = f
		ids = append(ids, f.InvoiceID)
	}
	sort.Strings(ids)
	receivable := func(id string) string {
		a := Account{Key: CategoryKey(CustomerReceivable, "invoice", id), Category: CustomerReceivable, SubjectDomain: "invoice", SubjectRef: id}
		if f, ok := invoices[id]; ok {
			a.CustomerLink = strp(f.CustomerLink)
			a.CustomerID = f.CustomerID
		} else {
			a.CustomerLink = strp("UNKNOWN")
		}
		return j.Account(a)
	}

	// 1. Accrual: receivable recognition only where the invoice amount
	// evidence MATCHes (declared totals agree with item and stage sums).
	// TAD #19: no total is chosen among disagreeing values.
	acc := strp(AccrualShadow)
	recognized := map[string]map[string]bool{} // invoice → unit → posted
	for _, id := range ids {
		f := invoices[id]
		evs := src.Evidence[id]
		if len(evs) == 0 {
			find("invoice", id, acc, ReasonUnresolvedInvoiceTotal, Unknown, "no amount evidence for the invoice")
			continue
		}
		for _, e := range evs {
			u, err := money.ParseUnit(e.Unit)
			if err != nil {
				find("invoice", id, acc, ReasonUnrepresentableAmount, Incomplete, "amounts not attributable to any unit (unrepresentable or absent); never rounded")
				continue
			}
			if e.Class != Match {
				state, reason := Unknown, ReasonUnresolvedInvoiceTotal
				switch e.Class {
				case "MISMATCH", "CONFLICT":
					state = Conflict
				case "INCOMPLETE":
					state, reason = Incomplete, ReasonUnrepresentableAmount
				}
				find("invoice", id, acc, reason, state, "invoice total evidence "+e.Class+" in "+e.Unit+"; no receivable is posted (TAD #19)")
				continue
			}
			amount := agreed(e)
			if amount == nil {
				find("invoice", id, acc, ReasonUnresolvedInvoiceTotal, Unknown, "no agreed amount")
				continue
			}
			if *amount == 0 {
				// A real zero: no ledger effect, but the total IS resolved, so
				// the zero receivable reconciles.
				if recognized[id] == nil {
					recognized[id] = map[string]bool{}
				}
				recognized[id][e.Unit] = true
				continue
			}
			r := receivable(id)
			ent := Entry{BusinessKey: BusinessKey("billing", "invoice", id, "receivable_recognition", e.Unit, AccrualShadow), Basis: AccrualShadow, Unit: u,
				EffectKind: "RECEIVABLE_RECOGNITION", Provenance: "BILLING_AMOUNT_EVIDENCE_MATCH",
				SourceRefs:   []SourceRef{{Domain: "billing.invoice", Ref: id, Role: "subject"}, {Domain: "legacy." + f.SourceSpec, Ref: f.SourceKey, Role: "source_row"}},
				TimeEvidence: TimeAbsent, Status: EntryConsistent,
				Lines: []Line{{AccountKey: r, Side: DebitSide, Amount: *amount, SourceRef: strp(id)}, {AccountKey: revenue, Side: CreditSide, Amount: *amount}}}
			if err := j.Add(ent); err != nil {
				return nil, err
			}
			if recognized[id] == nil {
				recognized[id] = map[string]bool{}
			}
			recognized[id][e.Unit] = true
		}
	}

	// 2. Payment attempts: only ACCEPTED attempts with a representable amount
	// in a known unit are cash evidence. Their provenance is kept (a gateway
	// auto-approval stays GATEWAY_AUTO; nothing is Finance-reviewed).
	unknownPay := map[string]bool{}  // invoice with an unknown-state attempt
	unpostedPay := map[string]bool{} // invoice with an accepted attempt not posted
	cashByInvoice := map[string]map[string]int64{}
	attempts := append([]paymentsapi.AttemptFact(nil), src.Attempts...)
	sort.Slice(attempts, func(a, b int) bool { return attempts[a].AttemptID < attempts[b].AttemptID })
	for _, a := range attempts {
		if a.Presence != "PRESENT" {
			continue
		}
		inv := ""
		if a.InvoiceID != nil {
			inv = *a.InvoiceID
		}
		if a.StateTarget == nil {
			if inv != "" {
				unknownPay[inv] = true
			}
			find("payment_attempt", a.AttemptID, nil, ReasonPaymentStatusUnknown, Unknown, "attempt state not mapped; never counted as cash")
			continue
		}
		if *a.StateTarget != "ACCEPTED" {
			continue
		}
		var u money.Unit
		var err error
		if a.Unit == nil {
			err = money.ErrInvalidUnit
		} else {
			u, err = money.ParseUnit(*a.Unit)
		}
		if err != nil {
			unpostedPay[inv] = true
			find("payment_attempt", a.AttemptID, nil, ReasonUnknownUnit, Incomplete, "accepted attempt without a known unit")
			continue
		}
		if a.AmountRepr != "REPRESENTABLE" || a.Amount == nil || *a.Amount < 0 {
			unpostedPay[inv] = true
			find("payment_attempt", a.AttemptID, nil, ReasonUnrepresentableAmount, Incomplete, "accepted attempt amount is "+a.AmountRepr)
			continue
		}
		if *a.Amount == 0 {
			continue
		}
		prov := a.Provenance
		if prov == "" {
			prov = "UNKNOWN"
		}
		clearing := j.Account(Account{Key: CategoryKey(CashBankClearing, "provenance", prov), Category: CashBankClearing,
			SubjectDomain: "approval_provenance", SubjectRef: prov, Provenance: strp(prov)})
		refs := []SourceRef{{Domain: "payments.attempt", Ref: a.AttemptID, Role: "subject"}}
		if inv != "" {
			refs = append(refs, SourceRef{Domain: "billing.invoice", Ref: inv, Role: "invoice"})
		}
		if inv != "" {
			r := receivable(inv)
			if err := j.Add(Entry{BusinessKey: BusinessKey("payments", "attempt", a.AttemptID, "cash_receipt", u.String(), AccrualShadow),
				Basis: AccrualShadow, Unit: u, EffectKind: "CASH_RECEIPT_AGAINST_RECEIVABLE", Provenance: prov, SourceRefs: refs,
				TimeEvidence: TimeAbsent, Status: EntryConsistent,
				Lines: []Line{{AccountKey: clearing, Side: DebitSide, Amount: *a.Amount, SourceRef: strp(a.AttemptID)},
					{AccountKey: r, Side: CreditSide, Amount: *a.Amount, SourceRef: strp(inv)}}}); err != nil {
				return nil, err
			}
		} else {
			find("payment_attempt", a.AttemptID, acc, ReasonUnresolvedInvoiceLink, Unknown,
				"accepted attempt is not linked to an invoice; no receivable is credited (link "+a.InvoiceLink+")")
		}
		if err := j.Add(Entry{BusinessKey: BusinessKey("payments", "attempt", a.AttemptID, "cash_receipt", u.String(), CashShadow),
			Basis: CashShadow, Unit: u, EffectKind: "CASH_RECEIPT", Provenance: prov, SourceRefs: refs, TimeEvidence: TimeAbsent,
			Status: EntryConsistent, Lines: []Line{{AccountKey: clearing, Side: DebitSide, Amount: *a.Amount, SourceRef: strp(a.AttemptID)},
				{AccountKey: revenue, Side: CreditSide, Amount: *a.Amount}}}); err != nil {
			return nil, err
		}
		if inv != "" {
			if cashByInvoice[inv] == nil {
				cashByInvoice[inv] = map[string]int64{}
			}
			cashByInvoice[inv][u.String()] += *a.Amount
		}
	}

	// 3. Legacy wallets: mirror every approved, directed, representable
	// transaction; reconcile against the legacy (mutable) balance.
	if err := deriveWallets(d, src.Wallets, find); err != nil {
		return nil, err
	}

	// 4. Invoice reconciliation (ACCRUAL receivable vs legacy remaining;
	// CASH receipts vs legacy paid), per unit, never across units.
	for _, id := range ids {
		for _, e := range src.Evidence[id] {
			u, err := money.ParseUnit(e.Unit)
			if err != nil {
				// Amounts attributable to no unit: both comparisons stay
				// explicit and INCOMPLETE (never dropped, never zero).
				for _, b := range []struct{ basis, figure string }{{AccrualShadow, FigureInvoiceRemaining}, {CashShadow, FigureInvoicePaid}} {
					d.Reconciliations = append(d.Reconciliations, Reconciliation{SubjectDomain: "invoice", SubjectRef: id, Basis: b.basis, Unit: "UNKNOWN",
						Figure: b.figure, Legacy: Parsed{Repr: "UNREPRESENTABLE"}, State: Incomplete, Reasons: []string{ReasonUnrepresentableAmount},
						Coverage: worst(src.BillingStatus, src.PaymentsStatus)})
				}
				continue
			}
			rk := CategoryKey(CustomerReceivable, "invoice", id)
			// accrual
			r := Reconciliation{SubjectDomain: "invoice", SubjectRef: id, AccountKey: &rk, Basis: AccrualShadow, Unit: e.Unit,
				Figure: FigureInvoiceRemaining, Legacy: Parsed{Amount: e.LegacyRemaining, Repr: e.LegacyRemainingRepr},
				Coverage: worst(src.BillingStatus, src.PaymentsStatus)}
			var blockers []string
			state := ""
			if !recognized[id][e.Unit] {
				blockers = append(blockers, ReasonUnresolvedInvoiceTotal)
				switch e.Class {
				case "MISMATCH", "CONFLICT":
					state = Conflict
				case "INCOMPLETE":
					state = Incomplete
				default:
					state = Unknown
				}
			}
			if unknownPay[id] {
				blockers, state = append(blockers, ReasonPaymentStatusUnknown), worstState(state, Incomplete)
			}
			if unpostedPay[id] {
				blockers, state = append(blockers, ReasonUnrepresentableAmount), worstState(state, Incomplete)
			}
			// Without a recognized invoice amount the receivable has no
			// comparable shadow amount: none is shown.
			if recognized[id][e.Unit] {
				if net, ok := j.AccountBalance(rk, AccrualShadow, u).Net(DebitSide); ok {
					r.Shadow = &net
				}
			}
			compare(&r, blockers, state, false, src.PaymentsStatus, ReasonLifecycleNotEvaluated)
			d.Reconciliations = append(d.Reconciliations, r)

			// cash
			c := Reconciliation{SubjectDomain: "invoice", SubjectRef: id, Basis: CashShadow, Unit: e.Unit, Figure: FigureInvoicePaid,
				Legacy: Parsed{Amount: e.LegacyPaid, Repr: e.LegacyPaidRepr}, Coverage: src.PaymentsStatus}
			blockers, state = nil, ""
			if unknownPay[id] {
				blockers, state = append(blockers, ReasonPaymentStatusUnknown), Incomplete
			}
			if unpostedPay[id] {
				blockers, state = append(blockers, ReasonUnrepresentableAmount), Incomplete
			}
			v := cashByInvoice[id][e.Unit]
			c.Shadow = &v
			compare(&c, blockers, state, false, src.PaymentsStatus, ReasonMissingLegacyTransaction)
			d.Reconciliations = append(d.Reconciliations, c)
		}
	}
	return d, nil
}

// compare classifies one reconciliation. state is the BLOCKING state of
// the comparison ("" when nothing blocks; UNKNOWN / INCOMPLETE / CONFLICT
// when no comparable amount exists): a blocked comparison never carries a
// difference. conflict marks a comparison whose amounts are complete but
// whose sources possibly duplicate each other: it keeps its exact
// difference and is CONFLICT — but only when nothing else blocks it.
// A legacy figure in another unit is never compared.
func compare(r *Reconciliation, blockers []string, state string, conflict bool, sourceStatus, mismatchReason string) {
	switch r.Legacy.Repr {
	case "OTHER_UNIT":
		blockers, state = append(blockers, ReasonCrossUnitForbidden), worstState(state, Unknown)
	case "ABSENT", "":
		blockers, state = append(blockers, ReasonMissingSource), worstState(state, Unknown)
	case "UNREPRESENTABLE":
		blockers, state = append(blockers, ReasonUnrepresentableAmount), worstState(state, Incomplete)
	}
	if sourceStatus != financeapi.SourceAvailable {
		blockers, state = append(blockers, ReasonMissingSource), worstState(state, Incomplete)
	}
	if r.Shadow == nil && state == "" {
		blockers, state = append(blockers, ReasonUnrepresentableAmount), Incomplete
	}
	if state == "" && r.Legacy.Amount == nil {
		state = Unknown
	}
	if state != "" {
		if conflict {
			state = worstState(state, Conflict)
		}
		r.State, r.Reasons = state, dedupe(blockers)
		return
	}
	diff, ok := sub(*r.Shadow, *r.Legacy.Amount)
	if !ok {
		r.State, r.Reasons = Incomplete, dedupe(append(blockers, ReasonUnrepresentableAmount))
		return
	}
	r.Difference = &diff
	switch {
	case conflict:
		r.State, r.Reasons = Conflict, dedupe(blockers)
	case diff == 0:
		r.State, r.Reasons = Match, dedupe(blockers)
	default:
		r.State, r.Reasons = Mismatch, dedupe(append(blockers, mismatchReason))
	}
}

func sub(a, b int64) (int64, bool) {
	d := a - b
	if (b > 0 && d > a) || (b < 0 && d < a) {
		return 0, false
	}
	return d, true
}

var stateRank = map[string]int{"": 0, Match: 1, Mismatch: 2, Incomplete: 3, Unknown: 4, Conflict: 5, NotRecoverable: 6}

func worstState(a, b string) string {
	if stateRank[b] > stateRank[a] {
		return b
	}
	return a
}

var sourceRank = map[string]int{financeapi.SourceAvailable: 0, financeapi.SourceStale: 1, financeapi.SourcePartial: 2,
	financeapi.SourceUnknown: 3, financeapi.SourceMissing: 4, financeapi.SourceNotRecoverable: 5}

func worst(a, b string) string {
	if sourceRank[b] > sourceRank[a] {
		return b
	}
	return a
}

func dedupe(rs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rs {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

// agreed returns the value every comparable evidence agrees on (MATCH).
func agreed(e billingapi.AmountEvidence) *int64 {
	for _, s := range []billingapi.PaymentUnitSum{e.ItemSum, e.StageSum} {
		if s.State == "COMPLETE" && s.Amount != nil {
			return s.Amount
		}
	}
	return nil
}

// CategoryKey builds an account key: category|kind:ref.
func CategoryKey(category, kind, ref string) string { return category + "|" + kind + ":" + ref }

// numericLess orders legacy decimal keys numerically.
func numericLess(a, b string) bool {
	x, e1 := strconv.ParseUint(a, 10, 64)
	y, e2 := strconv.ParseUint(b, 10, 64)
	if e1 != nil || e2 != nil {
		return a < b
	}
	return x < y
}
