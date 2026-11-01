package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

// Wallet builds one SYNTHETIC legacy wallet row (amounts in Toman text).
func Wallet(id, user, typ, balance, credit, debit string) Rec {
	return nullable(Rec{"id": id, "user_id": user, "wallet_type": typ, "balance": balance, "total_credit": credit, "total_debit": debit,
		"created_at": "2025-03-01 10:00:00", "updated_at": "2025-03-05 10:00:00"})
}

// WalletTx builds one SYNTHETIC legacy wallet transaction. meta is a
// synthetic JSON document (never real data).
func WalletTx(id, wallet, user, typ, invoice, amount, direction, txType, meta, key string) Rec {
	return nullable(Rec{"id": id, "wallet_id": wallet, "user_id": user, "wallet_type": typ, "invoice_id": invoice, "lead_id": "",
		"amount": amount, "direction": direction, "type": txType, "status": "approved", "meta": meta, "idempotency_key": key,
		"created_at": "2025-03-03 10:00:00"})
}

// WalletScenario is a SYNTHETIC legacy wallet source:
//   - wallet 1 (user 5, seller): credit 100 on invoice 2001 later reversed
//     in place (type *_reversed, key nulled) by debit 100 naming it, plus a
//     purpose credit 50 → legacy 50 = shadow 50 → MATCH;
//   - wallet 2 (user 4, supervisor): credits 40 and 12.50 (fractional,
//     never rounded) → INCOMPLETE;
//   - wallet 3 (user 6, a type no verified writer produces) with an
//     unverified transaction type → MATCH, routes reported unresolved;
//   - wallet 4 (user 5, biavin): one credit 25 but legacy balance 999 →
//     MISMATCH (balance not explained by recorded transactions).
func WalletScenario(c snapshot.Completeness) []snapshot.EntityRows {
	ents := []snapshot.EntityRows{
		rows("legacy.wallet.v1", []Rec{
			Wallet("1", "5", "seller", "50.00", "150.00", "100.00"),
			Wallet("2", "4", "supervisor", "52.50", "52.50", "0.00"),
			Wallet("3", "6", "synthetic_bonus", "10.00", "10.00", "0.00"),
			Wallet("4", "5", "biavin", "999.00", "999.00", "0.00"),
		}),
		rows("legacy.wallet_transaction.v1", []Rec{
			WalletTx("10", "1", "5", "seller", "2001", "100.00", "credit", "seller_commission_reversed", `{"stage_no":1}`, ""),
			WalletTx("11", "1", "5", "seller", "2001", "100.00", "debit", "seller_commission_reversal",
				`{"original_credit_transaction_id":10,"idempotency_key":"commission-reversal:10:seller_commission_reversal","reason":"synthetic reason text"}`,
				"commission-reversal:10:seller_commission_reversal"),
			WalletTx("12", "1", "5", "seller", "2002", "50.00", "credit", "purpose_commission", `{"engine":"purpose_matrix","stage_no":1}`,
				"purpose-commission:2002:stage:1:assessment_fee:seller:user:5"),
			WalletTx("20", "2", "4", "supervisor", "2001", "40.00", "credit", "supervisor_commission", "", "invoice:2001:user:4:wallet:supervisor:type:supervisor_commission"),
			WalletTx("21", "2", "4", "supervisor", "2003", "12.50", "credit", "supervisor_commission", "", "invoice:2003:user:4:wallet:supervisor:type:supervisor_commission"),
			WalletTx("30", "3", "6", "synthetic_bonus", "", "10.00", "credit", "synthetic_attendance_bonus", "", ""),
			WalletTx("40", "4", "5", "biavin", "2001", "25.00", "credit", "biavin_commission", `{"project_commission_id":1}`, ""),
		}),
	}
	for i := range ents {
		ents[i].Completeness = c
	}
	return ents
}

// WalletEntity builds the legacy.wallet.v1 entity from rows.
func WalletEntity(recs []Rec) snapshot.EntityRows { return rows("legacy.wallet.v1", recs) }
