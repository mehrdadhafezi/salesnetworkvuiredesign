package shadow

// Slice 9 finance shadow connector. It maps staged legacy wallets and wallet
// transactions into the neutral finance wallet evidence (finance/api). All
// legacy knowledge (wallet type and transaction type vocabularies, the meta
// link keys, the unit evidence) lives here; finance never sees it.
//
// Evidence (plugin/current 2.0.160):
//   - Wallets are staff wallets keyed (user_id, wallet_type). Verified
//     writers produce 'seller' and 'supervisor' (class-sn-plugin.php
//     :35588-35635) and 'biavin' (class-sn-projects.php:1159-1180), all
//     commission credits / their reversals. Any other wallet type is kept
//     raw as UNCLASSIFIED (meaning not decided, OPD-09).
//   - Amounts are rate x Toman stage / invoice amounts rounded by the legacy
//     writer to 2 decimals: the unit is IRT (same evidence as Slice 7,
//     class-sn-plugin.php:31200). A fractional amount stays raw and is
//     UNREPRESENTABLE in finance; nothing is rounded or converted here.
//   - Reversal links are the writers' own meta keys:
//     original_credit_transaction_id (class-sn-plugin.php:29325,
//     class-sn-purpose-commission.php:687), original_transaction_id
//     (class-sn-projects.php:1252) and reopened_credit_transaction_id
//     (class-sn-plugin.php:30808). Only these numeric keys are read; the
//     free text reason in meta is never read or copied.
//   - Legacy reversal rewrites the original credit's type in place
//     (*_reversed / *_reopened): reported as RewrittenInPlace.
//   - Commission engine routes (OPD-09) are classified by the transaction
//     type's writer; no engine is chosen or activated.

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	financeapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/api"
)

// Spec ids consumed by Slice 9.
const (
	SpecWallet   = "legacy.wallet.v1"
	SpecWalletTx = "legacy.wallet_transaction.v1"
)

// FinanceSpecs lists every staged spec Slice 9 reads.
func FinanceSpecs() []string { return []string{SpecWallet, SpecWalletTx} }

// commissionWallets are the wallet types verified writers produce.
var commissionWallets = set("seller", "supervisor", "biavin")

// routes classifies transaction types by their verified writer.
var routes = func() map[string]string {
	m := map[string]string{}
	for _, t := range []string{"commission_engine", "seller_commission", "supervisor_commission"} {
		for _, sfx := range []string{"", "_reversal", "_reversed", "_reopened"} {
			m[t+sfx] = financeapi.RouteLegacyRules
		}
	}
	for _, t := range []string{"purpose_commission", "purpose_commission_reversal", "purpose_commission_reversed"} {
		m[t] = financeapi.RoutePurposeMatrix
	}
	for _, t := range []string{"biavin_commission", "biavin_commission_recredit", "biavin_commission_reversal"} {
		m[t] = financeapi.RouteBiawinProject
	}
	m["finance_reopen_reversal"] = financeapi.RouteFinanceReopen
	return m
}()

// reversalLinkKeys are the meta keys a verified writer uses to name the
// transaction a debit reverses.
var reversalLinkKeys = []string{"original_credit_transaction_id", "original_transaction_id", "reopened_credit_transaction_id"}

func entityStatus(e Entity) string {
	switch {
	case e.Complete():
		return financeapi.SourceAvailable
	case e.Usable():
		return financeapi.SourcePartial
	}
	return financeapi.SourceMissing
}

// FinanceUserKeys lists every legacy user key the Slice 9 rows reference.
func FinanceUserKeys(b *Batch) []string {
	s := map[string]bool{}
	for _, r := range b.Entity(SpecWallet).Rows {
		if k, _ := key(r.Get("user_id")); k != nil {
			s[*k] = true
		}
	}
	return sorted(s)
}

// FinanceInvoiceKeys lists every legacy invoice key the transactions name.
func FinanceInvoiceKeys(b *Batch) []string {
	s := map[string]bool{}
	for _, r := range b.Entity(SpecWalletTx).Rows {
		if k, _ := key(r.Get("invoice_id")); k != nil {
			s[*k] = true
		}
	}
	return sorted(s)
}

func sorted(s map[string]bool) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// reversalLink reads ONLY the numeric reversal-link keys of the meta JSON.
func reversalLink(meta *string) *string {
	if meta == nil || *meta == "" {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(*meta), &m) != nil {
		return nil
	}
	for _, k := range reversalLinkKeys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var n json.Number
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.UseNumber()
		var v any
		if dec.Decode(&v) != nil {
			continue
		}
		switch x := v.(type) {
		case json.Number:
			n = x
		case string:
			n = json.Number(x)
		default:
			continue
		}
		if id, err := strconv.ParseUint(string(n), 10, 63); err == nil && id > 0 {
			s := strconv.FormatUint(id, 10)
			return &s
		}
	}
	return nil
}

func irtAmount(raw *string) financeapi.RawAmount {
	u := legacyUnit
	return financeapi.RawAmount{Raw: raw, Unit: &u}
}

// StagedWallets is the finance WalletSource over one staged snapshot: the
// batch plus the identity and billing resolutions of its keys.
type StagedWallets struct {
	Batch    *Batch
	Users    Users
	Invoices map[string]string
}

var _ financeapi.WalletSource = StagedWallets{}

// WalletEvidence implements financeapi.WalletSource (staging only).
func (s StagedWallets) WalletEvidence(context.Context) (financeapi.WalletEvidence, error) {
	return WalletEvidence(s.Batch, s.Users, s.Invoices, s.Batch.ExtractedAt), nil
}

// WalletEvidence maps the staged wallets and transactions. users maps legacy
// user keys to identity users and invoices maps legacy invoice keys to
// billing invoices (both from the owners' published APIs).
func WalletEvidence(b *Batch, users Users, invoices map[string]string, cutoff time.Time) financeapi.WalletEvidence {
	ws, ts := b.Entity(SpecWallet), b.Entity(SpecWalletTx)
	ev := financeapi.WalletEvidence{SnapshotID: b.SnapshotID, Cutoff: cutoff.UTC(), WalletStatus: entityStatus(ws),
		TransactionStatus: entityStatus(ts),
		Reason:            "legacy wallets and transactions read from staging only (never the live legacy system); unit IRT by writer evidence"}
	for _, r := range ws.Rows {
		w := financeapi.LegacyWallet{WalletKey: r.Key, Category: financeapi.WalletUnclassified, Balance: irtAmount(r.Get("balance")),
			TotalCredit: irtAmount(r.Get("total_credit")), TotalDebit: irtAmount(r.Get("total_debit")), UpdatedAtRaw: r.Get("updated_at")}
		if k, _ := key(r.Get("user_id")); k != nil {
			w.SubjectKey = *k
			if id, ok := users[*k]; ok {
				w.SubjectUserID = &id
			}
		}
		if t := r.Get("wallet_type"); t != nil {
			w.WalletTypeRaw = *t
			if commissionWallets[*t] {
				w.Category = financeapi.WalletCommission
			}
		}
		ev.Wallets = append(ev.Wallets, w)
	}
	for _, r := range ts.Rows {
		t := financeapi.LegacyWalletTx{TxKey: r.Key, Direction: financeapi.DirUnknown, DirectionRaw: r.Get("direction"),
			Route: financeapi.RouteUnresolved, Status: financeapi.StatusUnknown, StatusRaw: r.Get("status"), Amount: irtAmount(r.Get("amount")),
			ReversesTxKey: reversalLink(r.Get("meta")), SourceTimeRaw: r.Get("created_at")}
		if k, _ := key(r.Get("wallet_id")); k != nil {
			t.WalletKey = *k
		}
		if k, _ := key(r.Get("user_id")); k != nil {
			t.SubjectKey = *k
		}
		if v := r.Get("wallet_type"); v != nil {
			t.WalletTypeRaw = *v
		}
		if v := r.Get("direction"); v != nil {
			switch *v {
			case "credit":
				t.Direction = financeapi.Credit
			case "debit":
				t.Direction = financeapi.Debit
			}
		}
		if v := r.Get("status"); v != nil && *v == "approved" {
			t.Status = financeapi.StatusApproved
		}
		if v := r.Get("type"); v != nil {
			t.TypeRaw = *v
			if rt, ok := routes[*v]; ok {
				t.Route = rt
			}
			t.RewrittenInPlace = strings.HasSuffix(*v, "_reversed") || strings.HasSuffix(*v, "_reopened")
		}
		if v := r.Get("idempotency_key"); v != nil && *v != "" {
			t.IdempotencyKeyPresent = true
		}
		if k, _ := key(r.Get("invoice_id")); k != nil {
			t.InvoiceKey = k
			if id, ok := invoices[*k]; ok {
				t.InvoiceID = &id
			}
		}
		ev.Transactions = append(ev.Transactions, t)
	}
	return ev
}
