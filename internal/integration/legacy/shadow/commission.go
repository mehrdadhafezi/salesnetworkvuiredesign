package shadow

// Slice 13 commission evidence connector. It maps staged legacy commission
// sources into the neutral commission evidence (commission/api). All legacy
// knowledge (transaction type vocabulary, meta keys, engine routing, unit
// evidence) lives here; the commission module never sees a legacy table.
// It reads STAGING only, never the live legacy system, and writes nothing.
//
// Engine routing (OPD-09: classified by provable writer, never chosen):
//   - purpose_commission / _reversed credits with meta.engine=purpose_matrix:
//     E1 PURPOSE_MATRIX (class-sn-purpose-commission.php:614-618); any other
//     meta is a provenance CONFLICT -> ENGINE_ROUTING_UNKNOWN.
//   - commission_engine (+ _reversed / _reopened) with meta.commission_run_id:
//     E3 DRY_RUN_POSTING (class-sn-plugin.php:33637-33642); with meta.source
//     invoice_auto_credit_finance_rules / wallet_recalculate_finance_rules:
//     E2 RULES_AUTO (:35884-35896); otherwise ENGINE_ROUTING_UNKNOWN.
//   - supervisor_commission: E2 supervisor share (:35913-35935).
//   - seller_commission: no 2.0.160 writer exists -> ENGINE_ROUTING_UNKNOWN.
//   - biavin_commission*: E4 PROJECT_CARD (class-sn-projects.php:1181-1255).
//   - reversal debits inherit the engine of the credit they name.
//
// Only these meta keys are ever read: engine, payment_purpose, role_key,
// rate_percent, base_amount, stage_no, recipient_user_id, snapshot_version,
// commission_run_id, commission_dry_run_item_id, matched_rule_id,
// matched_rule_ids, source and the numeric reversal-link keys. The free
// text reason / note keys are never read or copied.

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	commissionapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
)

// Spec ids consumed by Slice 13 (the wallet transactions are Slice 9's).
const (
	SpecCommissionMatrix  = "legacy.purpose_commission_rate.v1"
	SpecCommissionRule    = "legacy.commission_rule.v1"
	SpecCommissionRun     = "legacy.commission_dry_run.v1"
	SpecCommissionRunItem = "legacy.commission_dry_run_item.v1"
	SpecCommissionLog     = "legacy.commission_posting_log.v1"
)

// CommissionSpecs lists every staged spec Slice 13 reads.
func CommissionSpecs() []string {
	return []string{SpecWalletTx, SpecCommissionMatrix, SpecCommissionRule, SpecCommissionRun, SpecCommissionRunItem, SpecCommissionLog}
}

// commissionTypes are the commission wallet transaction types of a writer.
var commissionTypes = func() map[string]bool {
	m := map[string]bool{}
	for _, t := range []string{"commission_engine", "seller_commission", "supervisor_commission"} {
		for _, sfx := range []string{"", "_reversal", "_reversed", "_reopened"} {
			m[t+sfx] = true
		}
	}
	for _, t := range []string{"purpose_commission", "purpose_commission_reversal", "purpose_commission_reversed",
		"biavin_commission", "biavin_commission_recredit", "biavin_commission_reversal", "finance_reopen_reversal"} {
		m[t] = true
	}
	return m
}()

// metaView reads ONLY the allow-listed scalar keys of a meta JSON.
type metaView map[string]json.RawMessage

var metaAllow = set("engine", "payment_purpose", "role_key", "rate_percent", "base_amount", "stage_no", "recipient_user_id", "snapshot_version",
	"commission_run_id", "commission_dry_run_item_id", "matched_rule_id", "matched_rule_ids", "source")

func readMeta(raw *string) metaView {
	if raw == nil || *raw == "" {
		return metaView{}
	}
	var all map[string]json.RawMessage
	if json.Unmarshal([]byte(*raw), &all) != nil {
		return metaView{}
	}
	m := metaView{}
	for k, v := range all {
		if metaAllow[k] {
			m[k] = v
		}
	}
	return m
}

// scalar renders a JSON string / number as its exact text (numbers keep their
// literal text; nothing is parsed into a float).
func (m metaView) scalar(k string) *string {
	raw, ok := m[k]
	if !ok {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return nil
	}
	switch x := v.(type) {
	case json.Number:
		s := string(x)
		return &s
	case string:
		if x == "" {
			return nil
		}
		return &x
	}
	return nil
}

// id renders a positive integer scalar as a legacy key.
func (m metaView) id(k string) *string {
	s := m.scalar(k)
	if s == nil {
		return nil
	}
	n, err := strconv.ParseUint(*s, 10, 63)
	if err != nil || n == 0 {
		return nil
	}
	out := strconv.FormatUint(n, 10)
	return &out
}

func (m metaView) ids(k string) []string {
	raw, ok := m[k]
	if !ok {
		return []string{}
	}
	var arr []json.Number
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if dec.Decode(&arr) != nil {
		return []string{}
	}
	out := []string{}
	for _, n := range arr {
		if v, err := strconv.ParseUint(string(n), 10, 63); err == nil && v > 0 {
			out = append(out, strconv.FormatUint(v, 10))
		}
	}
	return out
}

func entityState(e Entity) string {
	switch {
	case e.Complete():
		return commissionapi.SourceAvailable
	case e.Usable():
		return commissionapi.SourcePartial
	}
	return commissionapi.SourceMissing
}

// CommissionUserKeys lists every legacy user key the Slice 13 rows reference.
func CommissionUserKeys(b *Batch) []string {
	s := map[string]bool{}
	add := func(raw *string) {
		if k, _ := key(raw); k != nil {
			s[*k] = true
		}
	}
	for _, r := range b.Entity(SpecWalletTx).Rows {
		if t := r.Get("type"); t != nil && commissionTypes[*t] {
			add(r.Get("user_id"))
			add(readMeta(r.Get("meta")).id("recipient_user_id"))
		}
	}
	for _, r := range b.Entity(SpecCommissionRunItem).Rows {
		add(r.Get("seller_user_id"))
	}
	return sorted(s)
}

// CommissionInvoiceKeys lists every legacy invoice key the Slice 13 rows name.
func CommissionInvoiceKeys(b *Batch) []string {
	s := map[string]bool{}
	for _, r := range b.Entity(SpecWalletTx).Rows {
		if t := r.Get("type"); t != nil && commissionTypes[*t] {
			if k, _ := key(r.Get("invoice_id")); k != nil {
				s[*k] = true
			}
		}
	}
	for _, r := range b.Entity(SpecCommissionRunItem).Rows {
		if k, _ := key(r.Get("invoice_id")); k != nil && *k != "0" {
			s[*k] = true
		}
	}
	return sorted(s)
}

// StagedCommission is the commission EvidenceSource over one staged snapshot.
type StagedCommission struct {
	Batch    *Batch
	Users    Users
	Invoices map[string]string
}

var _ commissionapi.EvidenceSource = StagedCommission{}

// CommissionEvidence implements commissionapi.EvidenceSource (staging only).
func (s StagedCommission) CommissionEvidence(context.Context) (commissionapi.Evidence, error) {
	return CommissionEvidence(s.Batch, s.Users, s.Invoices), nil
}

func uid(users Users, k *string) *string {
	if k == nil {
		return nil
	}
	if id, ok := users[*k]; ok {
		return &id
	}
	return nil
}

// CommissionEvidence maps the staged commission sources.
func CommissionEvidence(b *Batch, users Users, invoices map[string]string) commissionapi.Evidence {
	tx := b.Entity(SpecWalletTx)
	ev := commissionapi.Evidence{SnapshotID: b.SnapshotID, BatchID: b.BatchID, Cutoff: b.ExtractedAt.UTC(), Sources: map[string]string{},
		SourceRows: map[string]int64{}, Credits: []commissionapi.LegacyCredit{}, Matrix: []commissionapi.MatrixRate{}, Rules: []commissionapi.Rule{},
		DryRuns: []commissionapi.DryRun{}, DryRunItems: []commissionapi.DryRunItem{}, PostingLogs: []commissionapi.PostingLog{}}
	for src, spec := range map[string]string{commissionapi.SourceCredits: SpecWalletTx, commissionapi.SourceMatrix: SpecCommissionMatrix,
		commissionapi.SourceRules: SpecCommissionRule, commissionapi.SourceDryRuns: SpecCommissionRun, commissionapi.SourceDryRunItems: SpecCommissionRunItem,
		commissionapi.SourcePostingLogs: SpecCommissionLog} {
		e := b.Entity(spec)
		ev.Sources[src] = entityState(e)
		ev.SourceRows[src] = int64(len(e.Rows))
	}
	engineOf := map[string]string{}
	pending := []int{}
	for _, r := range tx.Rows {
		t := r.Get("type")
		if t == nil || !commissionTypes[*t] {
			continue
		}
		m := readMeta(r.Get("meta"))
		c := commissionapi.LegacyCredit{TxKey: r.Key, TypeRaw: *t, Direction: commissionapi.DirUnknown, StatusRaw: r.Get("status"),
			WalletTypeRaw: r.Get("wallet_type"), AmountRaw: r.Get("amount"), Unit: legacyUnit, CreatedAtRaw: r.Get("created_at"),
			RewrittenInPlace: strings.HasSuffix(*t, "_reversed") || strings.HasSuffix(*t, "_reopened"),
			StageRaw:         m.scalar("stage_no"), PurposeRaw: m.scalar("payment_purpose"), RoleRaw: m.scalar("role_key"), RateRaw: m.scalar("rate_percent"),
			BaseRaw: m.scalar("base_amount"), RecipientSnapKey: m.id("recipient_user_id"), SnapshotVersion: m.scalar("snapshot_version"),
			ReversesTxKey: reversalLink(r.Get("meta")), DryRunKey: m.id("commission_run_id"), DryRunItemKey: m.id("commission_dry_run_item_id"),
			MatchedRuleKeys: m.ids("matched_rule_ids")}
		if one := m.id("matched_rule_id"); one != nil && len(c.MatchedRuleKeys) == 0 {
			c.MatchedRuleKeys = []string{*one}
		}
		switch v := r.Get("direction"); {
		case v != nil && *v == "credit":
			c.Direction = commissionapi.DirCredit
		case v != nil && *v == "debit":
			c.Direction = commissionapi.DirDebit
		}
		if k, _ := key(r.Get("user_id")); k != nil {
			c.HolderKey = k
			c.HolderUserID = uid(users, k)
		}
		if k, _ := key(r.Get("invoice_id")); k != nil {
			c.InvoiceKey = k
			if id, ok := invoices[*k]; ok {
				c.InvoiceID = &id
			}
		}
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(*t, "_reversed"), "_reopened"), "_recredit")
		engine := ""
		eng := m.scalar("engine")
		srcKey := m.scalar("source")
		switch base {
		case "purpose_commission":
			if eng != nil && *eng == "purpose_matrix" {
				engine, c.EngineBasis = "PURPOSE_MATRIX", "type purpose_commission + meta.engine=purpose_matrix"
			} else {
				engine, c.EngineBasis = "ENGINE_ROUTING_UNKNOWN", "type purpose_commission without meta.engine=purpose_matrix (provenance conflict)"
			}
		case "commission_engine":
			switch {
			case c.DryRunKey != nil:
				engine, c.EngineBasis = "DRY_RUN_POSTING", "type commission_engine + meta.commission_run_id"
			case srcKey != nil && (*srcKey == "invoice_auto_credit_finance_rules" || *srcKey == "wallet_recalculate_finance_rules"):
				engine, c.EngineBasis = "RULES_AUTO", "type commission_engine + meta.source="+*srcKey
			default:
				engine, c.EngineBasis = "ENGINE_ROUTING_UNKNOWN", "type commission_engine without a run id or rules source marker (E2 / E3 indistinguishable)"
			}
		case "supervisor_commission":
			engine, c.EngineBasis = "RULES_AUTO", "type supervisor_commission (E2 supervisor settings share)"
		case "seller_commission":
			engine, c.EngineBasis = "ENGINE_ROUTING_UNKNOWN", "type seller_commission has no 2.0.160 writer"
		case "biavin_commission":
			engine, c.EngineBasis = "PROJECT_CARD", "type biavin_commission* (E4 project card)"
		}
		c.Engine = engine
		if engine == "" { // a reversal debit: inherit from the credit it names
			pending = append(pending, len(ev.Credits))
		} else {
			engineOf[r.Key] = engine
		}
		ev.Credits = append(ev.Credits, c)
	}
	for _, i := range pending {
		c := &ev.Credits[i]
		c.Engine, c.EngineBasis = "ENGINE_ROUTING_UNKNOWN", "reversal debit without a resolvable original credit"
		if c.ReversesTxKey != nil {
			if e, ok := engineOf[*c.ReversesTxKey]; ok {
				c.Engine, c.EngineBasis = e, "reversal of transaction "+*c.ReversesTxKey
			}
		}
	}
	for _, r := range b.Entity(SpecCommissionMatrix).Rows {
		mr := commissionapi.MatrixRate{SourceKey: r.Key, RateRaw: r.Get("rate_percent"), UpdatedAtRaw: r.Get("updated_at")}
		if v := r.Get("payment_purpose"); v != nil {
			mr.PurposeRaw = *v
		}
		if v := r.Get("role_key"); v != nil {
			mr.RoleRaw = *v
		}
		ev.Matrix = append(ev.Matrix, mr)
	}
	for _, r := range b.Entity(SpecCommissionRule).Rows {
		ev.Rules = append(ev.Rules, commissionapi.Rule{SourceKey: r.Key, RuleCode: r.Get("rule_code"), CalculationTypeRaw: r.Get("calculation_type"),
			PositionRaw: r.Get("applies_to_position"), LevelRaw: r.Get("applies_to_level"), RateRaw: r.Get("rate_percent"), FixedRaw: r.Get("fixed_amount"),
			EffectiveFromRaw: r.Get("effective_from"), EffectiveToRaw: r.Get("effective_to"), ActiveRaw: r.Get("is_active"), PriorityRaw: r.Get("priority"),
			UpdatedAtRaw: r.Get("updated_at")})
	}
	k := func(raw *string) *string { v, _ := key(raw); return v }
	for _, r := range b.Entity(SpecCommissionRun).Rows {
		ev.DryRuns = append(ev.DryRuns, commissionapi.DryRun{SourceKey: r.Key, RunCode: r.Get("run_code"), DateFromRaw: r.Get("date_from"),
			DateToRaw: r.Get("date_to"), StatusRaw: r.Get("status"), ApprovalRaw: r.Get("approval_status"), CreatedByKey: k(r.Get("created_by")),
			ApprovedByKey: k(r.Get("approved_by")), ApprovedAtRaw: r.Get("approved_at"), RejectedByKey: k(r.Get("rejected_by")),
			RejectedAtRaw: r.Get("rejected_at"), LockedAtRaw: r.Get("locked_at"), CreatedAtRaw: r.Get("created_at")})
	}
	for _, r := range b.Entity(SpecCommissionRunItem).Rows {
		it := commissionapi.DryRunItem{SourceKey: r.Key, RunKey: k(r.Get("run_id")), SellerKey: k(r.Get("seller_user_id")), PositionRaw: r.Get("seller_position"),
			LevelRaw: r.Get("seller_level"), InvoiceAmountRaw: r.Get("invoice_amount"), MatchedRuleKey: k(r.Get("matched_rule_id")),
			CalculatedRaw: r.Get("calculated_commission"), ItemStatusRaw: r.Get("item_status"), PostingStatusRaw: r.Get("posting_status"),
			WalletTxKey: k(r.Get("wallet_transaction_id")), PostedAtRaw: r.Get("posted_at"), CompensationKey: k(r.Get("compensation_history_id")),
			CalculationDate: r.Get("calculation_date"), Unit: legacyUnit}
		it.SellerUserID = uid(users, it.SellerKey)
		if inv := k(r.Get("invoice_id")); inv != nil && *inv != "0" {
			it.InvoiceKey = inv
			if id, ok := invoices[*inv]; ok {
				it.InvoiceID = &id
			}
		}
		// Daily attendance items are non-invoice entitlements (class-sn-plugin.php:32835).
		it.NonInvoice = it.InvoiceKey == nil || (it.ItemStatusRaw != nil && *it.ItemStatusRaw == "daily_attendance_bonus")
		ev.DryRunItems = append(ev.DryRunItems, it)
	}
	for _, r := range b.Entity(SpecCommissionLog).Rows {
		ev.PostingLogs = append(ev.PostingLogs, commissionapi.PostingLog{SourceKey: r.Key, RunKey: k(r.Get("run_id")), ItemKey: k(r.Get("item_id")),
			WalletTxKey: k(r.Get("wallet_transaction_id")), ActorKey: k(r.Get("actor_user_id")), ActionRaw: r.Get("action"), StatusRaw: r.Get("status"),
			CreatedAtRaw: r.Get("created_at")})
	}
	return ev
}
