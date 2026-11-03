package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

// CommissionTx builds one SYNTHETIC commission wallet transaction.
func CommissionTx(id, user, invoice, amount, direction, txType, meta string) Rec {
	return WalletTx(id, "1", user, "seller", invoice, amount, direction, txType, meta, "")
}

// CommissionScenario is a SYNTHETIC legacy commission source (never real
// data). Expected shadow outcome per row:
//
//	E1 purpose matrix (stage amount x event-time rate, half-away 2dp):
//	  100 inv 2001 st1 product_sale/seller   1000 x 2.5 = 25.00      MATCH
//	  101 inv 2001 st1 product_sale/supervisor 1000 x 1.5 = 15.00   POLICY_MISMATCH (matrix now 1)
//	  102 inv 2001 st2 product_sale/seller   legacy 26.00 vs 25.00   AMOUNT_MISMATCH
//	  103 inv 2002 st1 assessment_fee/seller credited to 5, reversed in place
//	      (104 debit) and re-credited to 6 (105, then-current hierarchy):
//	      ONE entitlement, live credit 105 -> MATCH; RECIPIENT_MISMATCH finding
//	  106 inv 2003 st1 product_sale/seller   no recipient snapshot   ATTRIBUTION_UNKNOWN
//	  107 + 108 inv 2004 st1 product_sale/seller two live credits    DUPLICATE_RISK
//	E2 rules auto-credit:
//	  110 inv 2002 rule 1 (3% + 5) x 2000 = 65.00                    MATCH
//	      (and a cross-engine DUPLICATE_RISK finding: E1 + E2 on 2002)
//	unproven: 111 commission_engine without markers                  ENGINE_ROUTING_UNKNOWN
//	E3 dry run 1 (rejected, then approved): item 1 inv 2001 rule 1 = 35.00 MATCH,
//	   posted as 112; item 2 daily attendance (non-invoice)          NOT_COMPARABLE
//	E4 project card: 113 biavin_commission                           NOT_COMPARABLE (D-36)
func CommissionScenario(c snapshot.Completeness) []snapshot.EntityRows {
	ents := []snapshot.EntityRows{
		rows("legacy.wallet_transaction.v1", []Rec{
			CommissionTx("100", "5", "2001", "25.00", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"seller","rate_percent":2.5,"base_amount":1000,"stage_no":1,"recipient_user_id":5}`),
			CommissionTx("101", "4", "2001", "15.00", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"supervisor","rate_percent":1.5,"base_amount":1000,"stage_no":1,"recipient_user_id":4}`),
			CommissionTx("102", "5", "2001", "26.00", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"seller","rate_percent":2.5,"base_amount":1000,"stage_no":2,"recipient_user_id":5}`),
			CommissionTx("103", "5", "2002", "40.00", "credit", "purpose_commission_reversed",
				`{"engine":"purpose_matrix","payment_purpose":"assessment_fee","role_key":"seller","rate_percent":2,"base_amount":2000,"stage_no":1,"recipient_user_id":5}`),
			CommissionTx("104", "5", "2002", "40.00", "debit", "purpose_commission_reversal",
				`{"original_credit_transaction_id":103,"reason":"synthetic free text"}`),
			CommissionTx("105", "6", "2002", "40.00", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"assessment_fee","role_key":"seller","rate_percent":2,"base_amount":2000,"stage_no":1,"recipient_user_id":6}`),
			CommissionTx("106", "5", "2003", "5.00", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"seller","rate_percent":2.5,"base_amount":200,"stage_no":1}`),
			CommissionTx("107", "5", "2004", "1.25", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"seller","rate_percent":2.5,"base_amount":50,"stage_no":1,"recipient_user_id":5}`),
			CommissionTx("108", "5", "2004", "1.25", "credit", "purpose_commission",
				`{"engine":"purpose_matrix","payment_purpose":"product_sale","role_key":"seller","rate_percent":2.5,"base_amount":50,"stage_no":1,"recipient_user_id":5}`),
			CommissionTx("110", "5", "2002", "65.00", "credit", "commission_engine",
				`{"base_amount":2000,"matched_rule_ids":[1],"source":"invoice_auto_credit_finance_rules","note":"synthetic"}`),
			CommissionTx("111", "5", "2003", "9.00", "credit", "commission_engine", `{"base_amount":300}`),
			CommissionTx("112", "5", "2001", "35.00", "credit", "commission_engine",
				`{"commission_run_id":1,"commission_dry_run_item_id":1,"matched_rule_id":1,"note":"synthetic note"}`),
			CommissionTx("113", "5", "2001", "20.00", "credit", "biavin_commission", `{"project_commission_id":1}`),
		}),
		rows("legacy.purpose_commission_rate.v1", []Rec{
			nullable(Rec{"id": "1", "payment_purpose": "product_sale", "role_key": "seller", "rate_percent": "2.5000", "created_at": "2025-01-01 10:00:00", "updated_at": "2025-01-01 10:00:00"}),
			nullable(Rec{"id": "2", "payment_purpose": "product_sale", "role_key": "supervisor", "rate_percent": "1.0000", "created_at": "2025-01-01 10:00:00", "updated_at": "2025-06-01 10:00:00"}),
			nullable(Rec{"id": "3", "payment_purpose": "assessment_fee", "role_key": "seller", "rate_percent": "2.0000", "created_at": "2025-01-01 10:00:00", "updated_at": "2025-01-01 10:00:00"}),
		}),
		rows("legacy.commission_rule.v1", []Rec{
			nullable(Rec{"id": "1", "rule_code": "R1", "applies_to_position": "seller", "applies_to_level": "", "payment_method": "", "invoice_status": "",
				"employment_status": "", "employment_type": "", "calculation_type": "invoice_percent_fixed", "rate_percent": "3.0000", "fixed_amount": "5.00",
				"effective_from": "2025-01-01", "effective_to": "2025-12-31", "is_active": "1", "priority": "10", "created_at": "2025-01-01 10:00:00",
				"updated_at": "2025-01-01 10:00:00"}),
			nullable(Rec{"id": "2", "rule_code": "DAILY", "applies_to_position": "seller", "applies_to_level": "", "payment_method": "", "invoice_status": "",
				"employment_status": "", "employment_type": "", "calculation_type": "daily_attendance_plus_percent", "rate_percent": "0.0000",
				"fixed_amount": "0.00", "daily_base_amount": "100.000000", "is_active": "1", "priority": "0"}),
		}),
		rows("legacy.commission_dry_run.v1", []Rec{
			nullable(Rec{"id": "1", "run_code": "RUN-SYN-1", "date_from": "2025-03-01", "date_to": "2025-03-31", "status": "completed", "created_by": "1",
				"approval_status": "approved", "approved_by": "1", "approved_at": "2025-04-03 10:00:00", "rejected_by": "1",
				"rejected_at": "2025-04-02 10:00:00", "locked_at": "2025-04-03 10:00:00", "created_at": "2025-04-01 10:00:00"}),
		}),
		rows("legacy.commission_dry_run_item.v1", []Rec{
			nullable(Rec{"id": "1", "run_id": "1", "invoice_id": "2001", "seller_user_id": "5", "seller_position": "seller", "seller_level": "",
				"invoice_amount": "1000.00", "payment_method": "gateway", "invoice_status": "paid", "matched_rule_id": "1", "calculated_commission": "35.00",
				"item_status": "ready", "wallet_transaction_id": "112", "posted_at": "2025-04-04 10:00:00", "posting_status": "posted",
				"calculation_date": "2025-03-03", "created_at": "2025-04-01 10:00:00"}),
			nullable(Rec{"id": "2", "run_id": "1", "invoice_id": "0", "seller_user_id": "5", "seller_position": "seller", "seller_level": "",
				"invoice_amount": "0.00", "payment_method": "", "invoice_status": "", "matched_rule_id": "2", "calculated_commission": "100.00",
				"item_status": "daily_attendance_bonus", "posting_status": "pending", "calculation_date": "2025-03-04", "created_at": "2025-04-01 10:00:00"}),
		}),
		rows("legacy.commission_posting_log.v1", []Rec{
			nullable(Rec{"id": "1", "run_id": "1", "actor_user_id": "1", "action": "commission_run_rejected", "status": "rejected", "created_at": "2025-04-02 10:00:00"}),
			nullable(Rec{"id": "2", "run_id": "1", "actor_user_id": "1", "action": "commission_run_approved", "status": "approved", "created_at": "2025-04-03 10:00:00"}),
			nullable(Rec{"id": "3", "run_id": "1", "item_id": "1", "wallet_transaction_id": "112", "actor_user_id": "1", "action": "commission_wallet_posted",
				"status": "posted", "created_at": "2025-04-04 10:00:00"}),
		}),
	}
	for i := range ents {
		ents[i].Completeness = c
	}
	return ents
}
