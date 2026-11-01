package shadow

// Slice 7 billing + payments read connector. It maps staged legacy invoice,
// invoice item, payment stage and payment rows into the neutral billing and
// payments import commands, and reconciles the result. All legacy knowledge
// (column names, status vocabularies, unit evidence, the legacy display
// status expression) lives here; billing and payments never see it.
//
// Evidence (plugin/current 2.0.160):
//   - Amounts are DECIMAL(18,2) in TOMAN (IRT): class-sn-plugin.php:31200
//     ("CRM invoice totals are kept in toman"); the *10 conversion to rial
//     happens only at the gateway boundary (class-sn-invoice.php:130/183/212)
//     and is never applied here.
//   - Invoice status fields are kept separate. Their vocabularies are the
//     values written by the plugin and labelled by SN_Helpers::status_label.
//     Membership is KNOWN_VOCABULARY / UNKNOWN / EMPTY only: the canonical
//     invoice lifecycle resolver is deferred (TAD #20), so nothing maps to a
//     target invoice state.
//   - The legacy display status COALESCE(NULLIF(status,''), NULLIF(
//     invoice_status,''), NULLIF(payment_status,''), '')
//     (class-sn-invoice-review.php:12) is carried as LEGACY_DERIVED evidence
//     only; it never decides anything.
//   - Stage and attempt states are mapped only where the writer is verified;
//     every other raw value is UNKNOWN with the raw value kept.
//   - Gateway approval is legacy auto-approval (class-sn-plugin.php:7164,
//     D-25 open): GATEWAY_AUTO provenance, never Finance-reviewed.

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// Spec ids consumed by Slice 7.
const (
	SpecInvoice      = "legacy.invoice.v2"
	SpecInvoiceItem  = "legacy.invoice_item.v1"
	SpecPaymentStage = "legacy.invoice_payment_stage.v1"
	SpecPayment      = "legacy.payment.v1"

	// legacyUnit is the evidenced unit of every staged legacy amount.
	legacyUnit = "IRT"

	// Link bases are shown to readers: no legacy table name, no contact data.
	caseLinkBasis     = "invoice records the legacy lead it was issued from; Case resolved via the cases published API"
	customerLinkBasis = "customer minted for this invoice's issued-to subject (Slice 5, customers published API)"
)

// BillingSpecs lists every staged spec Slice 7 reads.
func BillingSpecs() []string {
	return []string{SpecInvoice, SpecInvoiceItem, SpecPaymentStage, SpecPayment}
}

// Invoice status vocabularies (2.0.160 writers + SN_Helpers::status_label).
var (
	invoiceMainVocabulary = set("pre_invoice", "pending", "pending_payment", "receipt_uploaded", "pending_financial_approval", "paid",
		"approved", "rejected", "cancelled", "recontact_requested", "partial_paid", "awaiting_next_payment", "payment_stage_issued",
		"payment_archived")
	invoiceStatusVocabulary = set("pre_invoice", "pending_financial_approval", "partial_paid", "payment_archived", "cancelled",
		"recontact_requested", "rejected")
	paymentStatusVocabulary = set("awaiting_payment", "cancelled", "partial_paid", "payment_archived", "pending_financial_approval",
		"pending_payment", "rejected")
	workflowVocabulary = set("archived", "awaiting_assignment", "awaiting_financial_approval", "awaiting_payment", "completed")
	returnVocabulary   = set("returned_to_seller", "resent_after_return")
)

// stageTargets: pending (class-sn-plugin.php:6371), pending_financial_approval
// (:30286), approved / paid (sn_finalize_current_payment_stage :6454), rejected
// (:30676). Anything else is UNKNOWN.
var stageTargets = map[string]string{"pending": "PENDING", "pending_financial_approval": "AWAITING_REVIEW", "approved": "ACCEPTED",
	"paid": "ACCEPTED", "rejected": "REJECTED"}

// attemptTargets: pending (class-sn-invoice.php:155/249/363, :30291),
// approved (gateway :7164 / card finalize :6526), rejected (:30677),
// cancelled (gateway user cancel :7188/7198), reconcile_error (gateway
// verified but stage finalize failed :7160). 'verified' (:36776, a restored
// gateway marker after a Finance rejection) is ambiguous → UNKNOWN.
var attemptTargets = map[string]string{"pending": "PENDING", "approved": "ACCEPTED", "rejected": "REJECTED", "cancelled": "CANCELLED",
	"reconcile_error": "RECONCILIATION_REQUIRED"}

// gatewayTargets are the gateway-side states implied by the attempt writer.
var gatewayTargets = map[string]string{"pending": "INITIATED", "approved": "VERIFIED", "reconcile_error": "VERIFIED",
	"verified": "VERIFIED", "cancelled": "CANCELLED_AT_GATEWAY"}

func set(vs ...string) map[string]bool {
	m := map[string]bool{}
	for _, v := range vs {
		m[v] = true
	}
	return m
}

func vocab(raw *string, known map[string]bool) billingapi.Vocabulary {
	switch {
	case raw == nil || *raw == "":
		return billingapi.EmptyValue
	case known[*raw]:
		return billingapi.KnownVocabulary
	}
	return billingapi.UnknownValue
}

func target(raw *string, m map[string]string) *string {
	if raw == nil {
		return nil
	}
	if t, ok := m[*raw]; ok {
		return &t
	}
	return nil
}

// legacyDisplayStatus reproduces the legacy review display expression as
// evidence. It is the ONLY place this fallback chain exists.
func legacyDisplayStatus(r Row) *string {
	for _, c := range []string{"status", "invoice_status", "payment_status"} {
		if v := r.Get(c); v != nil && *v != "" {
			s := *v
			return &s
		}
	}
	return nil
}

func (u Users) billingRef(raw *string) *billingapi.UserRef {
	k, _ := key(raw)
	if k == nil {
		return nil
	}
	r := &billingapi.UserRef{LegacyKey: *k}
	if id, ok := u[*k]; ok {
		r.UserID = &id
	}
	return r
}

func (u Users) paymentsRef(raw *string) *paymentsapi.UserRef {
	k, _ := key(raw)
	if k == nil {
		return nil
	}
	r := &paymentsapi.UserRef{LegacyKey: *k}
	if id, ok := u[*k]; ok {
		r.UserID = &id
	}
	return r
}

// BillingUserKeys lists every legacy user key the Slice 7 rows reference.
func BillingUserKeys(b *Batch) []string {
	s := map[string]bool{}
	add := func(raw *string) {
		if k, _ := key(raw); k != nil {
			s[*k] = true
		}
	}
	for _, r := range b.Entity(SpecInvoice).Rows {
		add(r.Get("seller_id"))
		add(r.Get("issued_by_user_id"))
	}
	for _, r := range b.Entity(SpecPaymentStage).Rows {
		add(r.Get("issued_by_user_id"))
		add(r.Get("approved_by_user_id"))
	}
	for _, r := range b.Entity(SpecPayment).Rows {
		add(r.Get("uploaded_by_user_id"))
	}
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// InvoiceLeadKeys lists the legacy lead keys invoices reference (for the
// Case link, resolved through the cases published API).
func InvoiceLeadKeys(b *Batch) []string {
	s := map[string]bool{}
	for _, r := range b.Entity(SpecInvoice).Rows {
		if k, _ := key(r.Get("lead_id")); k != nil {
			s[*k] = true
		}
	}
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// InvoiceKeys lists staged invoice keys (customer link subjects).
func InvoiceKeys(b *Batch) []string {
	rows := b.Entity(SpecInvoice).Rows
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Key)
	}
	return out
}

func billingSrc(b *Batch, spec string, r Row) billingapi.SourceRef {
	return billingapi.SourceRef{System: SourceSystem, Spec: spec, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA}
}

func irt(raw *string) billingapi.RawMoney { return billingapi.RawMoney{Raw: raw, Unit: legacyUnit} }

// declaredColumns are the declared invoice amount fields (all Toman).
var declaredColumns = map[string]billingapi.AmountKind{"product_price": "PRODUCT_PRICE", "original_total": "ORIGINAL_TOTAL",
	"discount_total": "DISCOUNT_TOTAL", "final_total": "FINAL_TOTAL", "payment_total_amount": "PAYMENT_TOTAL_AMOUNT",
	"current_due_amount": "CURRENT_DUE_AMOUNT", "paid_total_amount": "PAID_TOTAL_AMOUNT", "remaining_amount": "REMAINING_AMOUNT"}

// BillingCommand builds the billing import command. customers maps an
// invoice key to the customer id of its Slice 5 issued-to subject; cases
// maps a legacy lead key to its Case id. Missing entries stay UNKNOWN.
func BillingCommand(b *Batch, users Users, customers, cases map[string]string) (billingapi.BillingImport, map[string]int64) {
	counts := map[string]int64{}
	inv, items, stages := b.Entity(SpecInvoice), b.Entity(SpecInvoiceItem), b.Entity(SpecPaymentStage)
	in := billingapi.BillingImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Complete: map[string]bool{},
		ItemsComplete: items.Complete(), StagesComplete: stages.Complete()}
	for sp, e := range map[string]Entity{SpecInvoice: inv, SpecInvoiceItem: items, SpecPaymentStage: stages} {
		if e.Usable() {
			in.Complete[sp] = e.Complete()
		}
	}
	for _, r := range inv.Rows {
		states := []billingapi.StateEvidence{}
		for _, f := range []struct {
			field billingapi.StateField
			col   string
			known map[string]bool
		}{{billingapi.FieldMain, "status", invoiceMainVocabulary}, {billingapi.FieldInvoice, "invoice_status", invoiceStatusVocabulary},
			{billingapi.FieldPayment, "payment_status", paymentStatusVocabulary}, {billingapi.FieldWorkflow, "payment_workflow_status", workflowVocabulary},
			{billingapi.FieldReturn, "financial_return_state", returnVocabulary}} {
			raw := r.Get(f.col)
			m := vocab(raw, f.known)
			if m == billingapi.EmptyValue {
				raw = nil
			}
			if m == billingapi.UnknownValue {
				counts["invoice_state_unknown_"+string(f.field)]++
			}
			states = append(states, billingapi.StateEvidence{Field: f.field, Namespace: "legacy_invoice." + f.col, Raw: raw, Mapping: m})
		}
		if d := legacyDisplayStatus(r); d != nil {
			states = append(states, billingapi.StateEvidence{Field: billingapi.FieldLegacyDisplay, Namespace: "legacy_invoice.review_display_status",
				Raw: d, Mapping: billingapi.LegacyDerived})
		}
		declared := map[billingapi.AmountKind]billingapi.RawMoney{}
		for col, kind := range declaredColumns {
			declared[kind] = irt(r.Get(col))
		}
		rec := billingapi.InvoiceRecord{Source: billingSrc(b, SpecInvoice, r), CodeRaw: r.Get("invoice_code"), ProductRefRaw: r.Get("product_id"),
			PayMethodRaw: r.Get("pay_method"), PaymentPlanRaw: r.Get("payment_plan"), CurrentStageRaw: r.Get("current_payment_stage"),
			IssuedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at"), Seller: users.billingRef(r.Get("seller_id")),
			IssuedBy: users.billingRef(r.Get("issued_by_user_id")), States: states, Declared: declared,
			Customer: billingapi.Link{State: "UNKNOWN", Basis: "no customer for this invoice's issued-to subject (Slice 5 not imported for it)"},
			Case:     billingapi.Link{State: "UNKNOWN", Basis: "invoice records no lead reference"}}
		if id, ok := customers[r.Key]; ok {
			rec.Customer = billingapi.Link{ID: &id, State: "RESOLVED", Basis: customerLinkBasis}
		}
		if lk, bad := key(r.Get("lead_id")); lk != nil {
			if id, ok := cases[*lk]; ok {
				rec.Case = billingapi.Link{ID: &id, State: "RESOLVED", Basis: caseLinkBasis}
			} else {
				rec.Case = billingapi.Link{State: "UNKNOWN", Basis: "invoice names a legacy lead with no Case in the cases module"}
				counts["case_link_unresolved"]++
			}
		} else if bad {
			counts["case_link_unreadable"]++
		}
		in.Invoices = append(in.Invoices, rec)
	}
	for _, r := range items.Rows {
		ik, _ := key(r.Get("invoice_id"))
		if ik == nil {
			counts["items_without_invoice"]++
			continue
		}
		in.Items = append(in.Items, billingapi.ItemRecord{Source: billingSrc(b, SpecInvoiceItem, r), Invoice: billingapi.RowRef{Spec: SpecInvoice, Key: *ik},
			ProductRefRaw: r.Get("product_id"), TitleSnapshot: r.Get("product_name"), KindRaw: r.Get("product_type"), QtyRaw: r.Get("qty"),
			UnitPrice: irt(r.Get("unit_price")), LineTotal: irt(r.Get("total_price")), IsFreeRaw: r.Get("is_free"), CreatedAtRaw: r.Get("created_at")})
	}
	for _, r := range stages.Rows {
		ik, _ := key(r.Get("invoice_id"))
		if ik == nil {
			counts["stages_without_invoice"]++
			continue
		}
		raw := r.Get("status")
		t := target(raw, stageTargets)
		// Same classification as the store: an empty status is EMPTY, not
		// UNKNOWN; only a non-empty unmapped value is UNKNOWN.
		switch {
		case raw == nil || *raw == "":
			counts["stage_state_empty"]++
		case t == nil:
			counts["stage_state_unknown"]++
		}
		in.Stages = append(in.Stages, billingapi.StageRecord{Source: billingSrc(b, SpecPaymentStage, r),
			Invoice: billingapi.RowRef{Spec: SpecInvoice, Key: *ik}, StageNoRaw: r.Get("stage_no"), StageTypeRaw: r.Get("stage_type"),
			Requested: irt(r.Get("requested_amount")), StateNamespace: "legacy_invoice_payment_stage.status", StateRaw: raw, StateTarget: t,
			PayMethodRaw: r.Get("pay_method"), PaymentSourceRaw: r.Get("payment_source"), IssuedBy: users.billingRef(r.Get("issued_by_user_id")),
			ApprovedBy: users.billingRef(r.Get("approved_by_user_id")), ApprovedAtRaw: r.Get("approved_at"), PaidAtRaw: r.Get("paid_at"),
			CreatedAtRaw: r.Get("created_at")})
	}
	return in, counts
}

func digest(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

var (
	hostOK = regexp.MustCompile(`^[a-z0-9.-]{1,253}$`)
	extOK  = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
)

// receiptRef turns one raw receipt reference into metadata. Only host,
// extension and a SHA-256 of the reference leave this function: never the
// URL, its path, query or any filesystem path. A reference is at best
// REFERENCE_ONLY (no file is checked in Slice 7, so never AVAILABLE).
func receiptRef(raw string) (state string, host, ext *string, ref []byte) {
	ref = digest(raw)
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !hostOK.MatchString(strings.ToLower(u.Hostname())) {
		return paymentsapi.ReceiptUnknown, nil, nil, ref
	}
	h := strings.ToLower(u.Hostname())
	if e := strings.ToLower(strings.TrimPrefix(path.Ext(u.Path), ".")); extOK.MatchString(e) {
		ext = &e
	}
	return paymentsapi.ReceiptReferenceOnly, &h, ext, ref
}

// receiptList reads a legacy receipt_urls JSON array. An unparseable value
// is one UNKNOWN reference (kept, hashed), never dropped.
func receiptList(raw *string) []string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(*raw), &list); err != nil {
		return []string{"\x00unparseable:" + *raw}
	}
	out := list[:0]
	for _, v := range list {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// Receipt reference index bands per source row (documented, stable):
// stage receipt_urls[i] → i; invoice receipt_url → 0, receipt_file → 1,
// receipt_urls[i] → 10+i.
const invoiceListBase = 10

func receiptRecords(b *Batch, spec, owner string, r Row, inv string, stage *string, refs map[int]string, counts map[string]int64) []paymentsapi.ReceiptRecord {
	idx := make([]int, 0, len(refs))
	for i := range refs {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var out []paymentsapi.ReceiptRecord
	for _, i := range idx {
		st, host, ext, ref := receiptRef(refs[i])
		if strings.HasPrefix(refs[i], "\x00unparseable:") {
			st, host, ext = paymentsapi.ReceiptUnknown, nil, nil
		}
		counts["receipts_"+st]++
		out = append(out, paymentsapi.ReceiptRecord{
			Source:    paymentsapi.SourceRef{System: SourceSystem, Spec: spec, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA},
			OwnerKind: owner, Invoice: paymentsapi.RowRef{Spec: SpecInvoice, Key: inv}, StageNoRaw: stage, RefIndex: i, RefSHA256: ref,
			Host: host, Extension: ext, State: st})
	}
	return out
}

// PaymentsCommand builds the payments import command.
func PaymentsCommand(b *Batch, users Users) (paymentsapi.PaymentsImport, map[string]int64) {
	counts := map[string]int64{}
	pays, inv, stages := b.Entity(SpecPayment), b.Entity(SpecInvoice), b.Entity(SpecPaymentStage)
	in := paymentsapi.PaymentsImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem,
		AttemptsComplete: pays.Complete(), ReceiptSourcesComplete: inv.Complete() && stages.Complete()}
	for _, r := range pays.Rows {
		ik, _ := key(r.Get("invoice_id"))
		if ik == nil {
			counts["attempts_without_invoice_key"]++
			continue
		}
		raw := r.Get("status")
		t := target(raw, attemptTargets)
		authority := r.Get("authority")
		channel := paymentsapi.ChannelUnknown
		switch {
		case authority != nil && *authority != "":
			channel = paymentsapi.ChannelGateway // class-sn-payment-transactions.php:16
		case r.Get("pay_method") != nil && *r.Get("pay_method") == "card":
			channel = paymentsapi.ChannelManualCard
		}
		prov := paymentsapi.ProvNotApproved
		switch {
		case t == nil:
			prov = paymentsapi.ProvUnknown
			if raw == nil || *raw == "" {
				counts["attempt_state_empty"]++
			} else {
				counts["attempt_state_unknown"]++
			}
		case *t == "ACCEPTED" && channel == paymentsapi.ChannelGateway:
			prov = paymentsapi.ProvGatewayAuto
		case *t == "ACCEPTED" && channel == paymentsapi.ChannelManualCard:
			prov = paymentsapi.ProvLegacyManualApproval
		case *t == "ACCEPTED":
			prov = paymentsapi.ProvUnknown
		}
		a := paymentsapi.AttemptRecord{
			Source:  paymentsapi.SourceRef{System: SourceSystem, Spec: SpecPayment, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA},
			Invoice: paymentsapi.RowRef{Spec: SpecInvoice, Key: *ik}, StageNoRaw: r.Get("payment_stage_no"), Channel: channel,
			PayMethodRaw: r.Get("pay_method"), PaymentSourceRaw: r.Get("payment_source"),
			Amount:         paymentsapi.RawMoney{Raw: r.Get("amount"), Unit: legacyUnit},
			StateNamespace: "legacy_payment.status", StateRaw: raw, StateTarget: t, ApprovalProvenance: prov,
			UploadedByTypeRaw: r.Get("uploaded_by_type"), UploadedBy: users.paymentsRef(r.Get("uploaded_by_user_id")),
			CreatedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at")}
		if channel == paymentsapi.ChannelGateway {
			g := &paymentsapi.GatewayRecord{ProviderRaw: r.Get("payment_source"), AuthoritySHA256: digest(*authority), StateRaw: raw,
				StateTarget: target(raw, gatewayTargets)}
			if ref := r.Get("ref_id"); ref != nil && *ref != "" {
				g.RefSHA256 = digest(*ref)
			}
			a.Gateway = g
		}
		counts["attempts_channel_"+channel]++
		in.Attempts = append(in.Attempts, a)
	}
	for _, r := range stages.Rows {
		ik, _ := key(r.Get("invoice_id"))
		if ik == nil {
			continue
		}
		refs := map[int]string{}
		for i, v := range receiptList(r.Get("receipt_urls")) {
			refs[i] = v
		}
		in.Receipts = append(in.Receipts, receiptRecords(b, SpecPaymentStage, "STAGE", r, *ik, r.Get("stage_no"), refs, counts)...)
	}
	for _, r := range inv.Rows {
		refs := map[int]string{}
		for i, col := range []string{"receipt_url", "receipt_file"} {
			if v := r.Get(col); v != nil && strings.TrimSpace(*v) != "" {
				refs[i] = *v
			}
		}
		for i, v := range receiptList(r.Get("receipt_urls")) {
			refs[invoiceListBase+i] = v
		}
		in.Receipts = append(in.Receipts, receiptRecords(b, SpecInvoice, "INVOICE", r, r.Key, nil, refs, counts)...)
	}
	return in, counts
}

// BillingReport is the Slice 7 reconciliation report (JSON). It never names
// a canonical invoice status or a canonical total.
type BillingReport struct {
	Command       string                      `json:"command"`
	SnapshotID    string                      `json:"snapshot_id"`
	BatchID       string                      `json:"batch_id"`
	Outcome       result.Status               `json:"outcome"`
	Reasons       []reasoncodes.Code          `json:"reasons"`
	ConsumedSpecs []string                    `json:"consumed_specs"`
	Staged        map[string]int64            `json:"staged"`
	Provenance    map[string]map[string]int64 `json:"row_provenance"`
	Billing       billingapi.Coverage         `json:"billing"`
	Payments      paymentsapi.Coverage        `json:"payments"`
	Totals        map[string]map[string]int64 `json:"invoice_totals_by_unit_and_class"`
	Expected      map[string]int64            `json:"expected_from_batch"`
	F02           F02                         `json:"f02_open_preinvoices"`
	Coverage      map[string]string           `json:"source_coverage"`
	Invariants    map[string]bool             `json:"invariants"`
	Notes         []string                    `json:"notes"`
}

// F02 is the frozen Open Pre-invoices metric: NOT_EVALUATED while the
// canonical current invoice state is deferred (TAD #20).
type F02 struct {
	Status string `json:"status"`
	Value  *int64 `json:"value"`
	Reason string `json:"reason"`
}

// ReconcileBilling verifies, row by row, that the billing and payments
// shadow models carry THIS batch (same batch id and checksum) and reports
// state, total and receipt evidence. It reads only.
func ReconcileBilling(ctx context.Context, b *Batch, billing billingapi.Reader, payments paymentsapi.Reader) (BillingReport, error) {
	rep := BillingReport{Command: "reconcile-billing", SnapshotID: b.SnapshotID, BatchID: b.BatchID, Reasons: []reasoncodes.Code{},
		ConsumedSpecs: BillingSpecs(), Staged: map[string]int64{}, Provenance: map[string]map[string]int64{},
		Totals: map[string]map[string]int64{}, Expected: map[string]int64{}, Coverage: map[string]string{},
		F02: F02{Status: "NOT_EVALUATED", Reason: "requires the canonical current invoice state; deferred (TAD #20)"}}
	usable := true
	for _, sp := range BillingSpecs() {
		e := b.Entity(sp)
		rep.Staged[sp] = int64(len(e.Rows))
		rep.Coverage[sp] = coverage(e)
		usable = usable && e.Usable()
	}
	rep.Coverage["legacy.finance_ledger (not staged)"] = "UNKNOWN"
	rep.Coverage["receipt files (never read)"] = "UNKNOWN"
	var err error
	if rep.Billing, err = billing.Coverage(ctx); err != nil {
		return rep, err
	}
	if rep.Payments, err = payments.Coverage(ctx); err != nil {
		return rep, err
	}
	keysOf := func(sp string) []string {
		rows := b.Entity(sp).Rows
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.Key)
		}
		return out
	}
	tally := func(sp string, got map[string]struct {
		batch   string
		sha     []byte
		present bool
	}) {
		m := map[string]int64{"matched": 0, "missing": 0, "other_batch_or_changed": 0}
		for _, r := range b.Entity(sp).Rows {
			g, ok := got[r.Key]
			switch {
			case !ok || !g.present:
				m["missing"]++
			case g.batch != b.BatchID || string(g.sha) != string(r.SHA):
				m["other_batch_or_changed"]++
			default:
				m["matched"]++
			}
		}
		rep.Provenance[sp] = m
	}
	type prov = struct {
		batch   string
		sha     []byte
		present bool
	}
	invs, err := billing.ResolveInvoices(ctx, SourceSystem, SpecInvoice, keysOf(SpecInvoice))
	if err != nil {
		return rep, err
	}
	got := map[string]prov{}
	var invoiceIDs []string
	for k, v := range invs {
		got[k] = prov{v.BatchID, v.RowSHA256, v.Present}
		invoiceIDs = append(invoiceIDs, v.InvoiceID)
	}
	tally(SpecInvoice, got)
	for _, sp := range []string{SpecInvoiceItem, SpecPaymentStage} {
		rows, err := billing.RowProvenance(ctx, SourceSystem, sp, keysOf(sp))
		if err != nil {
			return rep, err
		}
		got := map[string]prov{}
		for k, v := range rows {
			got[k] = prov{v.BatchID, v.RowSHA256, v.Present}
		}
		tally(sp, got)
	}
	att, err := payments.RowProvenance(ctx, SourceSystem, SpecPayment, keysOf(SpecPayment))
	if err != nil {
		return rep, err
	}
	got = map[string]prov{}
	for k, v := range att {
		got[k] = prov{v.BatchID, v.RowSHA256, v.Present}
	}
	tally(SpecPayment, got)
	paymentsRun, err := payments.BatchImported(ctx, b.BatchID)
	if err != nil {
		return rep, err
	}
	// Totals: per invoice and unit, declared vs item/stage/payment sums.
	sort.Strings(invoiceIDs)
	sums, err := payments.AcceptedSums(ctx, invoiceIDs)
	if err != nil {
		return rep, err
	}
	evidence := map[string]billingapi.PaymentEvidence{}
	for id, p := range sums {
		e := billingapi.PaymentEvidence{Units: map[string]billingapi.PaymentUnitSum{}, Complete: p.Complete}
		for u, s := range p.Units {
			e.Units[u] = billingapi.PaymentUnitSum{Amount: s.Amount, State: s.State, Rows: s.Accepted}
		}
		evidence[id] = e
	}
	classes, err := billing.TotalClasses(ctx, invoiceIDs, evidence)
	if err != nil {
		return rep, err
	}
	for _, units := range classes {
		for u, c := range units {
			if rep.Totals[u] == nil {
				rep.Totals[u] = map[string]int64{}
			}
			rep.Totals[u][c]++
		}
	}
	// Expected evidence computed from the batch itself.
	_, bcounts := BillingCommand(b, nil, nil, nil)
	_, pcounts := PaymentsCommand(b, nil)
	for k, v := range bcounts {
		rep.Expected[k] = v
	}
	for k, v := range pcounts {
		rep.Expected[k] = v
	}
	approved, reconcileErr := int64(0), int64(0)
	for _, r := range b.Entity(SpecPayment).Rows {
		switch s := r.Get("status"); {
		case s != nil && *s == "approved":
			approved++
		case s != nil && *s == "reconcile_error":
			reconcileErr++
		}
	}
	rep.Expected["approved_raw"], rep.Expected["reconcile_error_raw"] = approved, reconcileErr
	allMatched := true
	for _, m := range rep.Provenance {
		allMatched = allMatched && m["missing"] == 0 && m["other_batch_or_changed"] == 0
	}
	complete := true
	for _, sp := range BillingSpecs() {
		complete = complete && b.Entity(sp).Complete()
	}
	unknownMain := int64(0)
	for _, n := range rep.Billing.UnknownRawStates["MAIN"] {
		unknownMain += n
	}
	// Exact-count checks are meaningful only when this batch is the whole
	// present model (complete sources, every row matched).
	exact := complete && allMatched
	rep.Invariants = map[string]bool{
		"approved_raw_preserved":         !exact || rep.Payments.ApprovedRaw == approved,
		"reconcile_error_visible":        !exact || rep.Payments.ReconcileErrorRaw == reconcileErr,
		"unknown_main_states_preserved":  !exact || unknownMain == bcounts["invoice_state_unknown_MAIN"],
		"unknown_stage_states_preserved": !exact || sumMap(rep.Billing.UnknownStageRaw) == bcounts["stage_state_unknown"],
		"receipts_never_available":       true, // no file is read in Slice 7; the API never reports AVAILABLE
		"f02_not_evaluated":              rep.F02.Value == nil,
	}
	rep.Notes = []string{
		"invoice lifecycle is UNKNOWN for every invoice: no canonical invoice status is claimed (TAD #20); each legacy status field is reported separately",
		"the legacy display status (COALESCE of status fields) is LEGACY_DERIVED evidence only",
		"amounts are int64 Toman (IRT) from DECIMAL(18,2) text; no conversion; fractional or absent values are never zero",
		"totals: declared FINAL_TOTAL and PAYMENT_TOTAL_AMOUNT, item sum, stage sum and accepted-payment sum are compared per unit; no canonical winner (TAD #19)",
		"payment sums count only attempts explicitly mapped ACCEPTED; 'approved' is preserved raw; gateway approval is GATEWAY_AUTO provenance, never Finance-reviewed (D-25 open)",
		"receipts are reference metadata only (host, extension, reference hash); no URL, path or file is stored or checked",
		"no phone, address, national id, card data, raw payload or gateway reference is stored or exposed",
	}
	if !paymentsRun {
		rep.Notes = append(rep.Notes, "no payments import run exists for this batch")
	}
	violated := false
	for _, ok := range rep.Invariants {
		violated = violated || !ok
	}
	switch {
	case violated:
		rep.Outcome = result.Conflict
		rep.Reasons = append(rep.Reasons, reasoncodes.LinkageConflict)
	case !usable:
		rep.Outcome = result.Unknown
		rep.Reasons = append(rep.Reasons, reasoncodes.SourceUnavailable)
	case !allMatched || !paymentsRun || !complete || rep.Payments.WithoutInvoice > 0:
		rep.Outcome = result.Partial
		rep.Reasons = append(rep.Reasons, reasoncodes.CoverageIncomplete)
	default:
		rep.Outcome = result.Success
	}
	return rep, nil
}

func sumMap(m map[string]int64) int64 {
	n := int64(0)
	for _, v := range m {
		n += v
	}
	return n
}
