package spec

// Slice 13 extension of the published v1 specs (commission evidence, READ /
// SHADOW only).
//
// Every column verified in plugin/current (2.0.160) at its table definition
// line. Four legacy commission engines exist (see
// docs/implementation/slice-13-commission-command-matrix.md):
//
//	E1 purpose matrix   sn_payment_commission_matrix (rates; MUTABLE in place,
//	                    no version history) + wallet transactions
//	                    purpose_commission* (already staged by Slice 9)
//	E2 rules auto-credit / E3 dry-run engine
//	                    sn_commission_rules (rules), sn_commission_dry_runs
//	                    (runs + approval), sn_commission_dry_run_items (items
//	                    + posting), sn_commission_wallet_posting_logs (audit)
//	E4 Biawin project   sn_project_commission_rules / sn_project_commissions:
//	                    inspected and NOT staged (D-36 component allocation is
//	                    unsupported; its wallet credits stay NOT_COMPARABLE)
//
// Deliberately NOT extracted (free text): rule title / description, run
// title / summary_json / approval_note, item notes / posting_error /
// invoice_code, posting log old_value / new_value / message.

// V1Slice13 returns the specs published by Slice 13 (golden-pinned separately).
func V1Slice13() []Spec {
	return []Spec{purposeCommissionRateV1(), commissionRuleV1(), commissionDryRunV1(), commissionDryRunItemV1(), commissionPostingLogV1()}
}

// purposeCommissionRateV1 stages sn_payment_commission_matrix (E1 rates).
func purposeCommissionRateV1() Spec {
	c := func(line string) string { return purposeCommissionFile + ":" + line + " (table definition)" }
	return v1("purpose_commission_rate", "sn_payment_commission_matrix", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("76")),
		colNote("payment_purpose", "VARCHAR(60)", NotNull, ClassNone, c("77"), "raw purpose key (assessment_fee / subscription_direct / conversion_purchase / product_sale / product_star_sale / product_star_upgrade)"),
		colNote("role_key", "VARCHAR(80)", NotNull, ClassNone, c("78"), "raw recipient role key (share facet)"),
		colNote("rate_percent", "DECIMAL(7,4)", NotNull, ClassNone, c("79"), "raw decimal text; MUTABLE in place (ON DUPLICATE KEY UPDATE); a snapshot observation, never a historical version"),
		colNote("updated_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("80"), "legacy staff user reference (last editor only)"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("81"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("82"), "raw legacy DATETIME of the LAST edit (overwritten)"),
	}, []Evidence{
		{File: purposeCommissionFile, Lines: "75-87", Note: "table definition of sn_payment_commission_matrix"},
		{File: purposeCommissionFile, Lines: "245-270", Note: "rates overwritten in place; no history; a change queues a backfill"},
		{File: purposeCommissionFile, Lines: "326-331", Note: "rate read at credit time, clamped to 0..100"},
	})
}

// commissionRuleV1 stages sn_commission_rules (E2 / E3 rules).
func commissionRuleV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("commission_rule", "sn_commission_rules", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1241")),
		colNote("rule_code", "VARCHAR(100)", NotNull, ClassNone, c("1242"), "raw unique rule code"),
		colNote("applies_to_position", "VARCHAR(80)", NotNull, ClassNone, c("1245"), "raw position slug filter (empty = any)"),
		colNote("applies_to_level", "VARCHAR(80)", NotNull, ClassNone, c("1246"), "raw level slug filter (empty = any)"),
		colNote("payment_method", "VARCHAR(80)", NotNull, ClassNone, c("1247"), "raw payment method filter"),
		colNote("invoice_status", "VARCHAR(120)", NotNull, ClassNone, c("1248"), "raw invoice status filter"),
		colNote("employment_status", "VARCHAR(80)", NotNull, ClassNone, c("1249"), "raw employment status filter"),
		colNote("employment_type", "VARCHAR(80)", NotNull, ClassNone, c("1250"), "raw employment type filter"),
		colNote("calculation_type", "VARCHAR(60)", NotNull, ClassNone, c("1251"), "raw: invoice_percent_fixed / daily_attendance_plus_percent"),
		colNote("rate_percent", "DECIMAL(10,4)", NotNull, ClassNone, c("1252"), "raw decimal text"),
		colNote("fixed_amount", "DECIMAL(18,2)", NotNull, ClassNone, c("1253"), "raw decimal text (unit not asserted)"),
		colNote("daily_base_amount", "DECIMAL(18,6)", Nullable, ClassNone, c("1254"), "raw decimal text (non-invoice entitlement; OPD-09)"),
		colNote("attendance_first_before", "TIME", Nullable, ClassNone, c("1255"), "raw TIME"),
		colNote("attendance_last_after", "TIME", Nullable, ClassNone, c("1256"), "raw TIME"),
		colNote("min_invoice_amount", "DECIMAL(18,2)", Nullable, ClassNone, c("1257"), "raw decimal text"),
		colNote("max_invoice_amount", "DECIMAL(18,2)", Nullable, ClassNone, c("1258"), "raw decimal text"),
		colNote("effective_from", "DATE", Nullable, ClassNone, c("1259"), "raw DATE (inclusive)"),
		colNote("effective_to", "DATE", Nullable, ClassNone, c("1260"), "raw DATE; equal from / to is treated as open-ended by the legacy matcher"),
		colNote("is_active", "TINYINT(1)", NotNull, ClassNone, c("1261"), "raw flag (mutable)"),
		colNote("priority", "INT", NotNull, ClassNone, c("1262"), "raw priority"),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("1263"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", Nullable, ClassNone, c("1264"), "raw legacy DATETIME of the LAST edit"),
	}, []Evidence{
		{File: migFile, Lines: "1240-1271", Note: "table definition of sn_commission_rules"},
		{File: pluginFile, Lines: "32730-32801", Note: "rule matching, effective interval and specificity / priority ordering"},
		{File: pluginFile, Lines: "32905-32912", Note: "round(max(0, fixed + amount * percent / 100), 2)"},
	},
		"title and description (free text) are deliberately NOT extracted",
	)
}

// commissionDryRunV1 stages sn_commission_dry_runs (E3 runs + approval).
func commissionDryRunV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("commission_dry_run", "sn_commission_dry_runs", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1313")),
		colNote("run_code", "VARCHAR(100)", NotNull, ClassNone, c("1314"), "raw unique run code"),
		colNote("date_from", "DATE", Nullable, ClassNone, c("1316"), "raw DATE (window start)"),
		colNote("date_to", "DATE", Nullable, ClassNone, c("1317"), "raw DATE (window end)"),
		colNote("status", "VARCHAR(40)", NotNull, ClassNone, c("1318"), "raw run status"),
		colNote("created_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("1319"), "legacy staff user reference (generator)"),
		colNote("approval_status", "VARCHAR(40)", NotNull, ClassNone, c("1321"), "raw: generated / approved / rejected (rejected -> approved is possible)"),
		colNote("approved_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("1322"), "legacy staff user reference"),
		colNote("approved_at", "DATETIME", Nullable, ClassNone, c("1323"), "raw legacy DATETIME"),
		colNote("rejected_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("1324"), "legacy staff user reference (kept after a later approval)"),
		colNote("rejected_at", "DATETIME", Nullable, ClassNone, c("1325"), "raw legacy DATETIME (kept after a later approval)"),
		colNote("locked_at", "DATETIME", Nullable, ClassNone, c("1327"), "raw legacy DATETIME (set by approve AND reject)"),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("1328"), "raw legacy DATETIME"),
	}, []Evidence{
		{File: migFile, Lines: "1312-1335", Note: "table definition of sn_commission_dry_runs"},
		{File: pluginFile, Lines: "33555-33590", Note: "approve allowed unless already approved; reject allowed unless approved"},
	},
		"title, summary_json and approval_note (free text) are deliberately NOT extracted",
	)
}

// commissionDryRunItemV1 stages sn_commission_dry_run_items (E3 items).
func commissionDryRunItemV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	money := func(name, line string) Column {
		return colNote(name, "DECIMAL(18,2)", NotNull, ClassNone, c(line), "raw decimal text; never rounded or converted")
	}
	return v1("commission_dry_run_item", "sn_commission_dry_run_items", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1369")),
		colNote("run_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1370"), "legacy dry run reference"),
		colNote("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1371"), "legacy invoice reference (0 for non-invoice daily attendance items)"),
		colNote("seller_user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1373"), "legacy staff user reference (recipient snapshot at run time)"),
		colNote("seller_position", "VARCHAR(80)", NotNull, ClassNone, c("1374"), "raw position snapshot at calculation date"),
		colNote("seller_level", "VARCHAR(80)", NotNull, ClassNone, c("1375"), "raw level snapshot at calculation date"),
		money("invoice_amount", "1376"),
		colNote("payment_method", "VARCHAR(80)", NotNull, ClassNone, c("1377"), "raw"),
		colNote("invoice_status", "VARCHAR(80)", NotNull, ClassNone, c("1378"), "raw"),
		colNote("matched_rule_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1379"), "legacy rule reference"),
		money("calculated_commission", "1380"),
		colNote("legacy_commission", "DECIMAL(18,2)", Nullable, ClassNone, c("1381"), "raw decimal text"),
		colNote("difference_amount", "DECIMAL(18,2)", Nullable, ClassNone, c("1382"), "raw decimal text"),
		colNote("item_status", "VARCHAR(40)", NotNull, ClassNone, c("1383"), "raw item status (no_rule / matched / daily_attendance_bonus / ...)"),
		colNote("wallet_transaction_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1385"), "legacy wallet transaction reference (posting evidence)"),
		colNote("posted_at", "DATETIME", Nullable, ClassNone, c("1386"), "raw legacy DATETIME"),
		colNote("posting_status", "VARCHAR(40)", NotNull, ClassNone, c("1387"), "raw: pending / posted / error"),
		colNote("compensation_history_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1389"), "legacy compensation history reference (attribution snapshot)"),
		colNote("compensation_effective_from", "DATE", Nullable, ClassNone, c("1390"), "raw DATE"),
		colNote("compensation_effective_to", "DATE", Nullable, ClassNone, c("1391"), "raw DATE"),
		colNote("rule_source", "VARCHAR(60)", Nullable, ClassNone, c("1392"), "raw rule source (auto / HR override)"),
		colNote("warning_code", "VARCHAR(120)", Nullable, ClassNone, c("1393"), "raw warning code"),
		colNote("calculation_date", "DATE", Nullable, ClassNone, c("1394"), "raw DATE"),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("1395"), "raw legacy DATETIME"),
	}, []Evidence{
		{File: migFile, Lines: "1368-1407", Note: "table definition of sn_commission_dry_run_items"},
		{File: pluginFile, Lines: "33602-33665", Note: "posting: approved run + flag + APPLY; per-item posted / error; meta commission_dry_run_item_id"},
	},
		"invoice_code, notes and posting_error (free text) are deliberately NOT extracted",
	)
}

// commissionPostingLogV1 stages sn_commission_wallet_posting_logs (E3 audit).
func commissionPostingLogV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("commission_posting_log", "sn_commission_wallet_posting_logs", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1457")),
		colNote("run_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1458"), "legacy dry run reference"),
		colNote("item_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1459"), "legacy dry run item reference"),
		colNote("wallet_transaction_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1460"), "legacy wallet transaction reference"),
		colNote("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1461"), "legacy staff user reference (event actor)"),
		colNote("action", "VARCHAR(80)", NotNull, ClassNone, c("1462"), "raw action key (commission_run_approved / _rejected / commission_wallet_posted / ...)"),
		colNote("status", "VARCHAR(40)", NotNull, ClassNone, c("1463"), "raw"),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("1467"), "raw legacy DATETIME"),
	}, []Evidence{
		{File: migFile, Lines: "1456-1474", Note: "table definition of sn_commission_wallet_posting_logs"},
		{File: pluginFile, Lines: "32964-32984", Note: "log writer"},
	},
		"old_value, new_value and message (free text / JSON) are deliberately NOT extracted",
	)
}
