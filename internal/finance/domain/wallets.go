package domain

import (
	"sort"

	financeapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

type walletState struct {
	account              string
	incomplete, conflict bool
	reasons              []string
	routeUnresolved      bool
}

// deriveWallets mirrors legacy wallet transactions (WALLET_MIRROR_SHADOW)
// and reconciles each legacy wallet balance. Routing (OPD-09) and wallet
// meaning are reported, never decided; no engine is activated.
func deriveWallets(d *Derivation, ev *financeapi.WalletEvidence, find func(dom, ref string, basis *string, reason, state, detail string)) error {
	basis := strp(WalletMirrorShadow)
	if ev == nil || ev.WalletStatus == financeapi.SourceMissing && ev.TransactionStatus == financeapi.SourceMissing {
		find("legacy_wallet", "*", basis, ReasonMissingSource, Unknown, "no staged legacy wallet evidence; wallet balances are UNKNOWN, not zero")
		return nil
	}
	j := d.Journal
	wallets := map[string]financeapi.LegacyWallet{}
	for _, w := range ev.Wallets {
		wallets[w.WalletKey] = w
	}
	states := map[string]*walletState{}
	payable := func(walletKey string, tx *financeapi.LegacyWalletTx) *walletState {
		if s, ok := states[walletKey]; ok {
			return s
		}
		a := Account{SubjectDomain: "legacy_wallet", SubjectRef: walletKey, Category: LegacyWalletUnclassed}
		if w, ok := wallets[walletKey]; ok {
			a.HolderKey, a.HolderUserID, a.WalletTypeRaw = strp(w.SubjectKey), w.SubjectUserID, strp(w.WalletTypeRaw)
			if w.Category == financeapi.WalletCommission {
				a.Category = CommissionPayable
			}
		} else if tx != nil {
			a.HolderKey, a.WalletTypeRaw = strp(tx.SubjectKey), strp(tx.WalletTypeRaw)
		}
		a.Key = CategoryKey(a.Category, "legacy_wallet", walletKey)
		s := &walletState{account: j.Account(a)}
		if a.Category == LegacyWalletUnclassed {
			s.reasons = append(s.reasons, ReasonEntitlementRoute)
			find("legacy_wallet", walletKey, basis, ReasonEntitlementRoute, Notice,
				"wallet type is not produced by a verified commission writer; its meaning (entitlement / stored value / other) is not decided")
		}
		states[walletKey] = s
		return s
	}
	for _, w := range ev.Wallets {
		payable(w.WalletKey, nil)
	}

	txs := append([]financeapi.LegacyWalletTx(nil), ev.Transactions...)
	sort.Slice(txs, func(a, b int) bool { return numericLess(txs[a].TxKey, txs[b].TxKey) })

	// Possible duplicates: same wallet, invoice, type, direction and amount,
	// at least one without a legacy idempotency key. Never deduplicated.
	type dupKey struct{ wallet, invoice, typ, dir, amount string }
	groups := map[dupKey][]string{}
	keyed := map[dupKey]int{}
	for _, t := range txs {
		if t.InvoiceKey == nil || t.Amount.Raw == nil {
			continue
		}
		k := dupKey{t.WalletKey, *t.InvoiceKey, t.TypeRaw, t.Direction, *t.Amount.Raw}
		groups[k] = append(groups[k], t.TxKey)
		if t.IdempotencyKeyPresent {
			keyed[k]++
		}
	}
	duplicate := map[string]bool{}
	for k, g := range groups {
		if len(g) > 1 && keyed[k] < len(g) {
			for _, id := range g {
				duplicate[id] = true
			}
		}
	}

	type posted struct {
		key, wallet, dir string
		amount           int64
	}
	byTx := map[string]posted{}
	for _, t := range txs {
		if t.WalletKey == "" {
			find("legacy_wallet_tx", t.TxKey, basis, ReasonOrphanLegacyTransaction, NotRecoverable, "transaction names no wallet")
			continue
		}
		s := payable(t.WalletKey, &t)
		if _, ok := wallets[t.WalletKey]; !ok {
			find("legacy_wallet_tx", t.TxKey, basis, ReasonOrphanLegacyTransaction, Unknown, "transaction names a wallet that is not staged")
		}
		if t.Route == financeapi.RouteUnresolved && !s.routeUnresolved {
			s.routeUnresolved = true
			s.reasons = append(s.reasons, ReasonCommissionRoute)
			find("legacy_wallet", t.WalletKey, basis, ReasonCommissionRoute, Notice, "a transaction type has no verified commission writer (OPD-09)")
		}
		if t.RewrittenInPlace {
			find("legacy_wallet_tx", t.TxKey, basis, ReasonLegacyHistoryRewritten, Notice,
				"legacy rewrote this row's type in place ("+t.TypeRaw+"); the original effect is still mirrored")
		}
		skip := func(reason, detail string) {
			s.incomplete = true
			s.reasons = append(s.reasons, reason)
			find("legacy_wallet_tx", t.TxKey, basis, reason, Incomplete, detail)
		}
		if t.Status != financeapi.StatusApproved {
			skip(ReasonWalletTxStatusUnknown, "legacy status is not 'approved'; not mirrored")
			continue
		}
		if t.Direction != financeapi.Credit && t.Direction != financeapi.Debit {
			skip(ReasonDirectionUnknown, "legacy direction unknown; not mirrored")
			continue
		}
		p := ParseExact(t.Amount.Raw, t.Amount.Unit)
		if p.Unit == nil {
			skip(ReasonUnknownUnit, "unit not evidenced; not mirrored")
			continue
		}
		if p.Repr != "REPRESENTABLE" || *p.Amount < 0 {
			skip(ReasonUnrepresentableAmount, "amount is not an exact integer in "+p.Unit.String()+"; never rounded")
			continue
		}
		if *p.Amount == 0 {
			continue // a real zero has no ledger effect
		}
		contra := j.Account(Account{Key: CategoryKey(LegacyWalletContra, "route", t.Route), Category: LegacyWalletContra,
			SubjectDomain: "commission_route", SubjectRef: t.Route, Route: strp(t.Route)})
		dr, cr := contra, s.account
		if t.Direction == financeapi.Debit {
			dr, cr = s.account, contra
		}
		refs := []SourceRef{{Domain: "legacy.wallet_transaction", Ref: t.TxKey, Role: "subject"}, {Domain: "legacy.wallet", Ref: t.WalletKey, Role: "wallet"}}
		if t.InvoiceID != nil {
			refs = append(refs, SourceRef{Domain: "billing.invoice", Ref: *t.InvoiceID, Role: "invoice"})
		} else if t.InvoiceKey != nil {
			refs = append(refs, SourceRef{Domain: "legacy.invoice", Ref: *t.InvoiceKey, Role: "invoice_unresolved"})
		}
		e := Entry{BusinessKey: BusinessKey("legacy_wallet", "transaction", t.TxKey, "wallet_effect", p.Unit.String(), WalletMirrorShadow),
			Basis: WalletMirrorShadow, Unit: *p.Unit, EffectKind: "LEGACY_WALLET_" + t.Direction, Provenance: "LEGACY_WALLET_" + t.Route,
			SourceRefs: refs, SourceTimeRaw: t.SourceTimeRaw, TimeEvidence: TimeZoneUnasserted, Status: EntryConsistent,
			Lines: []Line{{AccountKey: dr, Side: DebitSide, Amount: *p.Amount, SourceRef: strp(t.TxKey)}, {AccountKey: cr, Side: CreditSide, Amount: *p.Amount, SourceRef: strp(t.TxKey)}}}
		if t.SourceTimeRaw == nil || *t.SourceTimeRaw == "" {
			e.TimeEvidence = TimeAbsent
		}
		if duplicate[t.TxKey] {
			e.Status = EntryConflict
			s.conflict = true
			s.reasons = append(s.reasons, ReasonDuplicateSource)
			find("legacy_wallet_tx", t.TxKey, basis, ReasonDuplicateSource, Conflict,
				"another transaction has the same wallet, invoice, type, direction and amount without proof of distinct effects; not deduplicated")
		}
		if t.ReversesTxKey != nil {
			o, ok := byTx[*t.ReversesTxKey]
			opposite := ok && o.dir != t.Direction
			if ok && o.wallet == t.WalletKey && opposite && o.amount == *p.Amount && !j.Reversed(o.key) {
				e.ReversesKey = strp(o.key)
			} else {
				e.Status = EntryUnresolvedCorrection
				find("legacy_wallet_tx", t.TxKey, basis, ReasonUnresolvedCorrection, Conflict,
					"claims to reverse transaction "+*t.ReversesTxKey+" but the relationship is not provable (missing, later, other wallet, same direction, other amount or already reversed); no link is invented")
			}
		}
		if err := j.Add(e); err != nil {
			return err
		}
		byTx[t.TxKey] = posted{key: e.BusinessKey, wallet: t.WalletKey, dir: t.Direction, amount: *p.Amount}
	}

	// Reconcile every wallet (staged, or named by a transaction).
	keys := make([]string, 0, len(states))
	for k := range states {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return numericLess(keys[a], keys[b]) })
	snap := strp(ev.SnapshotID)
	for _, k := range keys {
		s := states[k]
		w, staged := wallets[k]
		lb := LegacyBalance{WalletKey: k, SnapshotID: ev.SnapshotID, Category: financeapi.WalletUnclassified}
		r := Reconciliation{SubjectDomain: "legacy_wallet", SubjectRef: k, AccountKey: strp(s.account), Basis: WalletMirrorShadow,
			Unit: "UNKNOWN", Figure: FigureWalletBalance, Coverage: worst(ev.WalletStatus, ev.TransactionStatus), Snapshot: snap}
		if staged {
			lb.HolderKey, lb.HolderUserID, lb.WalletTypeRaw, lb.Category, lb.BalanceRaw = w.SubjectKey, w.SubjectUserID, w.WalletTypeRaw, w.Category, w.Balance.Raw
			lb.Balance = ParseExact(w.Balance.Raw, w.Balance.Unit)
			lb.TotalCredit = ParseExact(w.TotalCredit.Raw, w.TotalCredit.Unit)
			lb.TotalDebit = ParseExact(w.TotalDebit.Raw, w.TotalDebit.Unit)
			lb.Unit = lb.Balance.Unit
			d.LegacyBalances = append(d.LegacyBalances, lb)
			r.Legacy, r.LegacyRaw = lb.Balance, w.Balance.Raw
		} else {
			r.Legacy = Parsed{Repr: "ABSENT"}
		}
		blockers := append([]string(nil), s.reasons...)
		state := ""
		if s.incomplete {
			state = Incomplete
		}
		unit := r.Legacy.Unit
		if unit == nil {
			if staged {
				blockers, state = append(blockers, ReasonUnknownUnit), worstState(state, Unknown)
			}
			irt := money.IRT
			unit = &irt // the mirrored transactions carry the evidenced unit
		}
		r.Unit = unit.String()
		bal := d.Journal.AccountBalance(s.account, WalletMirrorShadow, *unit)
		if net, ok := bal.Net(CreditSide); ok {
			r.Shadow = &net
		}
		if staged && lb.Balance.Amount != nil && lb.TotalCredit.Amount != nil && lb.TotalDebit.Amount != nil {
			if net, ok := sub(*lb.TotalCredit.Amount, *lb.TotalDebit.Amount); !ok || net != *lb.Balance.Amount {
				blockers = append(blockers, ReasonLegacyTotalsInconsistent)
			}
		}
		if !staged {
			blockers = append(blockers, ReasonOrphanLegacyTransaction)
		}
		compare(&r, blockers, state, s.conflict, worst(ev.WalletStatus, ev.TransactionStatus), ReasonMissingLegacyTransaction)
		d.Reconciliations = append(d.Reconciliations, r)
	}
	return nil
}
