package legacytest

import (
	"context"
	"strconv"

	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// SyntheticInvoice is one synthetic legacy invoice for payments write-back
// (SYNTHETIC data only).
type SyntheticInvoice struct {
	SellerLegacyUserID string
	StatusRaw          string
	StageNo            int
	StageDueToman      int64
	Payments           []SyntheticPayment
}

// SyntheticPayment is one synthetic legacy sn_payments row.
type SyntheticPayment struct {
	ID            int
	StageNo       int
	AmountToman   int64
	StatusRaw     string
	SubmissionKey string
	Evidence      []string
}

var _ lwb.PaymentsLegacy = (*SyntheticLegacy)(nil)

// AddInvoice registers a synthetic legacy invoice.
func (s *SyntheticLegacy) AddInvoice(key string, inv SyntheticInvoice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Invoices == nil {
		s.Invoices = map[string]*SyntheticInvoice{}
	}
	cp := inv
	s.Invoices[key] = &cp
}

// Invoice returns a copy of a synthetic legacy invoice.
func (s *SyntheticLegacy) Invoice(key string) SyntheticInvoice {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inv := s.Invoices[key]; inv != nil {
		cp := *inv
		cp.Payments = append([]SyntheticPayment(nil), inv.Payments...)
		return cp
	}
	return SyntheticInvoice{}
}

// DuplicatePayment copies the payment carrying submissionKey (an ambiguous
// legacy result: two rows for one intent).
func (s *SyntheticLegacy) DuplicatePayment(invoiceKey, submissionKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv := s.Invoices[invoiceKey]
	for _, p := range inv.Payments {
		if p.SubmissionKey == submissionKey {
			s.nextPay++
			p.ID = s.nextPay
			inv.Payments = append(inv.Payments, p)
			return
		}
	}
}

// SubmitManualPayment implements lwb.PaymentsLegacy with the 2.0.160
// sn_save_invoice_manual_payment rules that matter for write-back.
func (s *SyntheticLegacy) SubmitManualPayment(_ context.Context, op lwb.ManualPayment) (lwb.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls["payment"]++
	fault := s.take("payment")
	if fault == TimeoutBeforeCommit {
		return lwb.Result{}, ErrTransport
	}
	inv := s.Invoices[op.InvoiceKey]
	if inv == nil {
		return lwb.Result{Outcome: result.Failed, Reason: "invoice_not_found"}, nil
	}
	for _, p := range inv.Payments { // UNIQUE (invoice_id, submission_key): replay
		if p.SubmissionKey == op.SubmissionKey {
			return lwb.Result{Outcome: result.Success, Ref: "payment:" + strconv.Itoa(p.ID)}, nil
		}
	}
	var pending int64
	for _, p := range inv.Payments {
		if p.StageNo == inv.StageNo && p.StatusRaw == "pending" {
			pending += p.AmountToman
		}
	}
	switch {
	case inv.SellerLegacyUserID != op.ActorLegacyUserID:
		return lwb.Result{Outcome: result.Unauthorized, Reason: "not_invoice_seller"}, nil
	case inv.StatusRaw == "paid" || inv.StatusRaw == "approved" || inv.StatusRaw == "cancelled" || inv.StatusRaw == "payment_archived":
		return lwb.Result{Outcome: result.Conflict, Reason: "invoice_not_payable"}, nil
	case op.StageNo != inv.StageNo:
		return lwb.Result{Outcome: result.Conflict, Reason: "stage_changed"}, nil
	case len(op.EvidenceRefs) == 0:
		return lwb.Result{Outcome: result.Failed, Reason: "receipt_required"}, nil
	case op.AmountToman > inv.StageDueToman-pending:
		return lwb.Result{Outcome: result.Failed, Reason: "amount_exceeds_available"}, nil
	}
	s.nextPay++
	inv.Payments = append(inv.Payments, SyntheticPayment{ID: s.nextPay, StageNo: op.StageNo, AmountToman: op.AmountToman, StatusRaw: "pending",
		SubmissionKey: op.SubmissionKey, Evidence: append([]string(nil), op.EvidenceRefs...)})
	if fault == TimeoutAfterCommit {
		return lwb.Result{}, ErrTransport
	}
	return lwb.Result{Outcome: result.Success, Ref: "payment:" + strconv.Itoa(s.nextPay)}, nil
}

// ObserveManualPayment implements lwb.PaymentsLegacy.
func (s *SyntheticLegacy) ObserveManualPayment(_ context.Context, invoiceKey, submissionKey string) (lwb.PaymentObserved, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.take("payment_observe") == ObserveDown {
		return lwb.PaymentObserved{}, ErrTransport
	}
	inv := s.Invoices[invoiceKey]
	if inv == nil {
		return lwb.PaymentObserved{}, nil
	}
	var out lwb.PaymentObserved
	for _, p := range inv.Payments {
		if p.SubmissionKey == submissionKey {
			out.Matches++
			out.PaymentRef, out.StageNo, out.AmountToman, out.StatusRaw, out.ReceiptCount = "payment:"+strconv.Itoa(p.ID), p.StageNo, p.AmountToman,
				p.StatusRaw, len(p.Evidence)
		}
	}
	return out, nil
}
