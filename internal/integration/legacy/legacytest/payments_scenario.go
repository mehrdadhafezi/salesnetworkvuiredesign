package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

// payInvoice is a SYNTHETIC invoice for the Slice 12 scenario: Toman,
// integral amounts, current stage 1.
func payInvoice(id, seller, issuer, status, total, remaining string) Rec {
	r := Invoice(id, "", status, total, total)
	r["seller_id"], r["issued_by_user_id"], r["remaining_amount"], r["current_due_amount"] = seller, issuer, remaining, remaining
	r["discount_total"], r["paid_total_amount"] = "0", "0"
	return r
}

// PaymentsScenario is the Slice 12 SYNTHETIC source: the Slice 11 users
// (seller X = 6, seller Y = 7, supervisor A = 4) plus complete billing and
// payments entities. Every staged attempt names a staged invoice.
//   - 3001: seller 6 / issuer 6, pending, stage 1 due 1000 → payable;
//   - 3002: seller 7 / issuer 7 → not seller 6's invoice;
//   - 3003: seller 6, main status paid → not payable;
//   - 3004: seller 6 but issued by supervisor 4 → fail closed;
//   - 3005: seller 6, due 800 with a legacy pending card payment of 300 →
//     available 500;
//   - 3006: seller 6, stage with an unmapped status → UNKNOWN;
//   - 3007, 3008: seller 6, further payable invoices (gateway flow).
func PaymentsScenario() []snapshot.EntityRows {
	ents := WritesScenario()
	pay := Payment("71", "3005", "300", "pending", "card", "", "seller_manual")
	pay["uploaded_by_user_id"] = "6"
	billing := []snapshot.EntityRows{
		rows("legacy.invoice.v2", []Rec{
			payInvoice("3001", "6", "6", "pending", "1000", "1000"),
			payInvoice("3002", "7", "7", "pending", "1000", "1000"),
			payInvoice("3003", "6", "6", "paid", "1000", "0"),
			payInvoice("3004", "6", "4", "pending", "1000", "1000"),
			payInvoice("3005", "6", "6", "pending_financial_approval", "800", "800"),
			payInvoice("3006", "6", "6", "pending", "600", "600"),
			payInvoice("3007", "6", "6", "pending", "1200", "1200"),
			payInvoice("3008", "6", "6", "pending", "900", "900"),
		}),
		rows("legacy.invoice_item.v1", nil),
		rows("legacy.invoice_payment_stage.v1", []Rec{
			Stage("51", "3001", "1", "1000", "pending", ""),
			Stage("52", "3002", "1", "1000", "pending", ""),
			Stage("53", "3003", "1", "1000", "paid", ""),
			Stage("54", "3004", "1", "1000", "pending", ""),
			Stage("55", "3005", "1", "800", "pending_financial_approval", ""),
			Stage("56", "3006", "1", "600", "synthetic_stage_status", ""),
			Stage("57", "3007", "1", "1200", "pending", ""),
			Stage("58", "3008", "1", "900", "pending", ""),
		}),
		rows("legacy.payment.v1", []Rec{pay}),
	}
	for i := range billing {
		billing[i].Completeness = snapshot.Complete
	}
	return append(ents, billing...)
}
