package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

// Invoice builds one SYNTHETIC legacy invoice row (amounts in Toman text).
// No customer name, phone or address column is staged at all.
func Invoice(id, lead, status, final, payTotal string) Rec {
	return nullable(Rec{"id": id, "invoice_code": "SYN-INV-" + id, "seller_id": "5", "issued_by_user_id": "4", "lead_id": lead,
		"product_id": "301", "product_price": final, "pay_method": "", "status": status, "invoice_status": "", "payment_status": "awaiting_payment",
		"receipt_url": "", "receipt_file": "", "receipt_urls": "", "receipt_source": "", "payment_source": "", "paid_at": "", "approved_at": "",
		"financial_reviewed_at": "", "original_total": final, "discount_total": "0.00", "final_total": final, "payment_plan": "full",
		"payment_total_amount": payTotal, "current_due_amount": "", "paid_total_amount": "0.00", "remaining_amount": "",
		"current_payment_stage": "1", "payment_workflow_status": "awaiting_payment", "payment_completed_at": "", "financial_return_state": "",
		"created_at": "2025-03-01 10:00:00", "updated_at": "2025-03-02 10:00:00"})
}

// InvoiceItem builds one SYNTHETIC invoice line.
func InvoiceItem(id, invoice, total string) Rec {
	return nullable(Rec{"id": id, "invoice_id": invoice, "product_id": "301", "product_name": "Synthetic product " + id, "product_type": "main",
		"qty": "1", "unit_price": total, "total_price": total, "is_free": "0", "created_at": "2025-03-01 10:00:00"})
}

// Stage builds one SYNTHETIC payment stage.
func Stage(id, invoice, no, amount, status, receipts string) Rec {
	return nullable(Rec{"id": id, "invoice_id": invoice, "stage_no": no, "stage_type": "full", "requested_amount": amount, "status": status,
		"pay_method": "", "payment_source": "", "receipt_urls": receipts, "issued_by_user_id": "4", "approved_by_user_id": "",
		"approved_at": "", "paid_at": "", "created_at": "2025-03-01 10:00:00", "updated_at": "2025-03-02 10:00:00"})
}

// Payment builds one SYNTHETIC payment attempt. A non-empty authority is a
// synthetic gateway reference (never a real one).
func Payment(id, invoice, amount, status, method, authority, source string) Rec {
	return nullable(Rec{"id": id, "invoice_id": invoice, "payment_stage_no": "1", "authority": authority, "ref_id": "", "amount": amount,
		"status": status, "pay_method": method, "payment_source": source, "uploaded_by_type": "seller", "uploaded_by_user_id": "5",
		"created_at": "2025-03-03 10:00:00", "updated_at": "2025-03-03 11:00:00"})
}

// BillingScenario is a SYNTHETIC billing + payments source:
//   - 2001: 1000 Toman; item, stage and an approved gateway payment all 1000
//     → MATCH; lead 101 (Case link); two stage receipt references;
//   - 2002: unknown main status; item 1800 vs declared 2000 → MISMATCH;
//     unknown stage status; card approved 500, a reconcile_error and a
//     'verified' attempt (ambiguous → UNKNOWN);
//   - 2003: fractional total 300.50 → UNREPRESENTABLE → INCOMPLETE; an
//     invoice-level receipt reference and an unparseable one;
//   - 2004: pre-invoice with no lines, stages or payments;
//   - payment 9 names invoice 2999, which is not staged.
func BillingScenario(c snapshot.Completeness) []snapshot.EntityRows {
	inv3 := Invoice("2003", "", "pending_payment", "300.50", "300.50")
	inv3["receipt_url"] = "https://files.example.test/uploads/receipt-2003.pdf"
	inv3["receipt_urls"] = "not-json"
	ents := []snapshot.EntityRows{
		rows("legacy.invoice.v2", []Rec{
			Invoice("2001", "101", "pending", "1000.00", "1000.00"),
			Invoice("2002", "", "synthetic_unfamiliar_status", "2000", "2000"),
			inv3,
			Invoice("2004", "", "pre_invoice", "50", "50"),
		}),
		rows("legacy.invoice_item.v1", []Rec{
			InvoiceItem("31", "2001", "1000.00"), InvoiceItem("32", "2002", "1800"), InvoiceItem("33", "2003", "300.50"),
		}),
		rows("legacy.invoice_payment_stage.v1", []Rec{
			Stage("41", "2001", "1", "1000", "approved", `["https://files.example.test/r/41-a.jpg","https://files.example.test/r/41-b.png"]`),
			Stage("42", "2002", "1", "2000", "synthetic_stage_status", ""),
			Stage("43", "2003", "1", "300.50", "pending", ""),
		}),
		rows("legacy.payment.v1", []Rec{
			Payment("1", "2001", "1000.00", "approved", "gateway", "SYN-AUTH-0001", "zibal"),
			Payment("2", "2002", "500", "approved", "card", "", "seller_upload"),
			Payment("3", "2002", "700", "reconcile_error", "gateway", "SYN-AUTH-0003", "zarinpal"),
			Payment("4", "2002", "800", "verified", "gateway", "SYN-AUTH-0004", "zibal"),
			Payment("9", "2999", "10", "pending", "card", "", ""),
		}),
	}
	for i := range ents {
		ents[i].Completeness = c
	}
	return ents
}
