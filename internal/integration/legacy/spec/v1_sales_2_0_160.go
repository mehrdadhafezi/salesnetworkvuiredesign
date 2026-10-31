package spec

// Slice 6 extension of the published v1 specs (sales read shadow).
//
// Only the sources Slice 6 needs, each verified in plugin/current (2.0.160):
//
//   - sn_leads                    → legacy lead work units (current holder columns)
//   - sn_lead_status_history      → append-only lead status history
//   - sn_distribution_items       → distribution work items (current holder columns)
//   - sn_distribution_item_logs   → append-only distribution item logs (custody evidence)
//
// Deliberately NOT extracted:
//   - lead phone, customer name, national id, note, province, city and sales
//     prediction (PII / not needed; phone is never a key — sn_leads carries
//     UNIQUE KEY phone in 2.0.160, which the target never adopts);
//   - distribution seller_* customer/contact columns, notes, conversion error
//     text and log note/payload_json (free text / PII);
//   - sn_lead_statuses (labels only; raw status values are preserved instead);
//   - sn_seller_flow_states/_events, sn_dot_cases, sn_mis_distribution_*,
//     sn_distribution_chains/_chain_steps (not required for Slice 6 read;
//     their coverage is reported UNKNOWN). Note: 2.0.160 has no table named
//     sn_distribution_logs; the item log table is sn_distribution_item_logs.

// V1Slice6 returns the specs published by Slice 6 (golden-pinned separately).
func V1Slice6() []Spec {
	return []Spec{leadV1(), leadStatusHistoryV1(), distributionItemV1(), distributionItemLogV1()}
}

func leadV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("lead", "sn_leads", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("236")),
		colNote("marketing_form_id", "VARCHAR(80)", Nullable, ClassNone, c("240")+"; :530 (ADD COLUMN statement)", "raw source/channel evidence"),
		colNote("import_code", "VARCHAR(80)", Nullable, ClassNone, c("241")+"; :520 (ADD COLUMN statement)", "raw import provenance code"),
		colNote("status", "VARCHAR(20)", NotNull, ClassNone, c("244"), "raw legacy lead status; never defaulted or relabelled in staging"),
		colNote("lead_status", "VARCHAR(60)", Nullable, ClassNone, c("245")+"; :508 (ADD COLUMN statement)", "raw legacy contact/lead status value"),
		colNote("destination_panel", "VARCHAR(50)", Nullable, ClassNone, c("247")+"; :536 (ADD COLUMN statement)", "raw routing panel value"),
		colNote("seller_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("250"), "legacy current seller holder (current state only; 2.0.160 keeps no lead assignment log)"),
		colNote("supervisor_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("251")+"; :517 (ADD COLUMN statement)", "legacy current supervisor reference (current state only)"),
		colNote("imported_at", "DATETIME", NotNull, ClassNone, c("252"), "raw legacy DATETIME; zone not asserted"),
		colNote("assigned_at", "DATETIME", Nullable, ClassNone, c("253"), "raw legacy DATETIME; zone not asserted"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("254"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: activatorFile, Lines: "235-262", Note: "table definition of sn_leads (UNIQUE KEY phone at :256 is legacy behaviour; never a target key)"},
		{File: activatorFile, Lines: "505-540", Note: "columns added on upgraded installs"},
	},
		"lead work-unit and current-holder columns only; phone, customer_name, national_id, note, province, city and sales_prediction are deliberately NOT extracted",
	)
}

func leadStatusHistoryV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("lead_status_history", "sn_lead_status_history", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("492")),
		col("lead_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("493")),
		colNote("user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("494"), "legacy event actor reference"),
		colNote("old_status", "VARCHAR(100)", Nullable, ClassNone, c("495"), "raw"),
		colNote("new_status", "VARCHAR(100)", Nullable, ClassNone, c("496"), "raw"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("498"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: activatorFile, Lines: "491-503", Note: "table definition of sn_lead_status_history"},
	},
		"append-only legacy status history; the free-text note column is deliberately NOT extracted",
	)
}

func distributionItemV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("distribution_item", "sn_distribution_items", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("964")),
		colNote("source_type", "VARCHAR(60)", NotNull, ClassNone, c("965"), "raw source kind (default 'mis_pool')"),
		colNote("source_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("966"), "raw source row reference; (source_type, source_id) UNIQUE at :992"),
		col("batch_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("967")),
		colNote("current_owner_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("969"), "legacy current holder of record; also used for payment-workflow routing (class-sn-plugin.php:28687)"),
		colNote("current_owner_position", "VARCHAR(100)", Nullable, ClassNone, c("970"), "raw position slug of the holder of record"),
		col("current_step_order", "INT UNSIGNED", NotNull, ClassNone, c("971")),
		colNote("final_seller_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("972"), "legacy seller reference; not proven to be the original seller"),
		colNote("live_lead_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("973"), "sn_leads id written on conversion (class-sn-plugin.php:21209); explicit native link"),
		colNote("conversion_status", "VARCHAR(60)", Nullable, ClassNone, c("976"), "raw"),
		colNote("status", "VARCHAR(60)", NotNull, ClassNone, c("988"), "raw legacy item status"),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("989"), "raw legacy DATETIME; zone not asserted"),
		colNote("updated_at", "DATETIME", Nullable, ClassNone, c("990"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: migFile, Lines: "962-1052", Note: "table definition and ensured columns of sn_distribution_items"},
		{File: pluginFile, Lines: "20828-20853, 21195-21218, 22462-22500, 28682-28694", Note: "holder updates on delivery, conversion, return to MIS and payment routing"},
	},
		"work-item and holder columns only; seller_customer_name, seller_province/city, seller_contact_status, seller_notes, conversion_error and other free-text columns are deliberately NOT extracted",
	)
}

func distributionItemLogV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("distribution_item_log", "sn_distribution_item_logs", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1056")),
		col("item_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("1057")),
		colNote("action", "VARCHAR(100)", NotNull, ClassNone, c("1058"), "raw legacy action; mapped only by an explicit verified table, unknown values stay UNKNOWN"),
		colNote("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1059"), "legacy event actor"),
		colNote("from_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1060"), "0 is written as NULL (class-sn-plugin.php:24017)"),
		colNote("to_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("1061"), "not always the new holder (e.g. conversion logs carry the final seller)"),
		col("from_position", "VARCHAR(100)", Nullable, ClassNone, c("1062")),
		col("to_position", "VARCHAR(100)", Nullable, ClassNone, c("1063")),
		col("step_order", "INT UNSIGNED", Nullable, ClassNone, c("1064")),
		colNote("created_at", "DATETIME", Nullable, ClassNone, c("1067"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: migFile, Lines: "1054-1100", Note: "table definition of sn_distribution_item_logs"},
		{File: pluginFile, Lines: "24008-24020", Note: "sn_distribution_log writer (append-only insert)"},
	},
		"append-only item log; note and payload_json are deliberately NOT extracted",
	)
}
