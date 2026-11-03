package writeback

import (
	"context"
	"strconv"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	wb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
)

// Slice 12 legacy mapping of a target manual payment attempt:
//
//	attempt → sn_save_invoice_manual_payment (class-sn-plugin.php:30189), receipt
//	mode, source "seller_manual", submission_key = attempt id without dashes
//	(legacy UNIQUE (invoice_id, submission_key) makes a repeated apply
//	idempotent and is the reconciliation key). Amounts are legacy Toman
//	(IRT); any other unit is refused before anything is sent (no conversion,
//	OPD-06). Receipt files travel by their content-addressed evidence key.
//
// Legacy 2.0.160 re-checks ownership, invoice / stage state and the stage
// availability itself; its refusals are reported as proven non-commit.

// ManualPayment is the legacy manual payment insert.
type ManualPayment struct {
	InvoiceKey, SubmissionKey, ActorLegacyUserID, Source string
	StageNo                                              int
	AmountToman                                          int64
	EvidenceRefs                                         []string
}

// PaymentObserved is the legacy state found for a submission key.
type PaymentObserved struct {
	// Matches is the number of legacy payment rows carrying the key.
	Matches      int
	PaymentRef   string
	StageNo      int
	AmountToman  int64
	StatusRaw    string
	ReceiptCount int
}

// PaymentsLegacy is the legacy store surface for payments write-back.
type PaymentsLegacy interface {
	SubmitManualPayment(ctx context.Context, op ManualPayment) (Result, error)
	ObserveManualPayment(ctx context.Context, invoiceKey, submissionKey string) (PaymentObserved, error)
}

// WithPayments enables the payments mapping over pl. Without it a payment
// intent is BLOCKED (nothing sent).
func (c *Connector) WithPayments(pl PaymentsLegacy) *Connector {
	c.payments = pl
	return c
}

// Legacy payment statuses mapped to the neutral attempt state.
var paymentStatus = map[string]string{"pending": "PENDING_REVIEW"}

func (c *Connector) paymentOp(ctx context.Context, in wb.Intent) (ManualPayment, string, error) {
	f := in.Fields
	if c.payments == nil {
		return ManualPayment{}, "PAYMENTS_LEGACY_NOT_CONFIGURED", nil
	}
	if f["invoice_source_system"] != shadow.SourceSystem || f["invoice_source_spec"] != shadow.SpecInvoice || f["invoice_source_key"] == "" {
		return ManualPayment{}, "SUBJECT_NOT_LEGACY_INVOICE", nil
	}
	if f["unit"] != "IRT" {
		return ManualPayment{}, "UNIT_NOT_LEGACY_TOMAN_OPD-06", nil
	}
	amount, err := strconv.ParseInt(f["amount"], 10, 64)
	if err != nil || amount <= 0 {
		return ManualPayment{}, "AMOUNT_INVALID", nil
	}
	stage, err := strconv.Atoi(f["stage_no"])
	if err != nil || stage < 1 {
		return ManualPayment{}, "STAGE_INVALID", nil
	}
	refs := strings.Split(f["evidence_refs"], ",")
	if f["evidence_refs"] == "" || len(refs) == 0 {
		return ManualPayment{}, "EVIDENCE_NOT_READY", nil
	}
	keys, err := c.keys(ctx, f["actor_user_id"])
	if err != nil {
		return ManualPayment{}, "", err
	}
	actor, ok := keys[f["actor_user_id"]]
	if !ok {
		return ManualPayment{}, "ACTOR_NOT_MAPPED", nil
	}
	return ManualPayment{InvoiceKey: f["invoice_source_key"], SubmissionKey: RequestToken(in.CommandID), ActorLegacyUserID: actor,
		Source: "seller_manual", StageNo: stage, AmountToman: amount, EvidenceRefs: refs}, "", nil
}

func (c *Connector) applyPayment(ctx context.Context, in wb.Intent) (wb.Applied, error) {
	op, why, err := c.paymentOp(ctx, in)
	if err != nil || why != "" {
		return blocked(why), err
	}
	r, err := c.payments.SubmitManualPayment(ctx, op)
	if err != nil {
		return wb.Applied{}, err
	}
	return applied(r), nil
}

func (c *Connector) observePayment(ctx context.Context, in wb.Intent) (wb.Observed, error) {
	f := in.Fields
	if c.payments == nil || f["invoice_source_key"] == "" {
		return wb.Observed{Observation: wb.Unseen, Detail: "payments legacy not configured or subject unknown"}, nil
	}
	o, err := c.payments.ObserveManualPayment(ctx, f["invoice_source_key"], RequestToken(in.CommandID))
	if err != nil {
		return wb.Observed{}, err
	}
	obs := wb.Observed{Observation: count(o.Matches), LegacyRef: o.PaymentRef, State: map[string]string{}}
	if obs.Observation != wb.Committed {
		return obs, nil
	}
	status, ok := paymentStatus[o.StatusRaw]
	if !ok {
		status = "UNMAPPED:" + o.StatusRaw
	}
	obs.State = map[string]string{"amount": strconv.FormatInt(o.AmountToman, 10), "unit": "IRT", "stage_no": strconv.Itoa(o.StageNo),
		"status": status, "receipt_count": strconv.Itoa(o.ReceiptCount)}
	return obs, nil
}
