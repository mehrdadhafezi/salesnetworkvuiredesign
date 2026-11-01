package spec

// Slice 7 extension of the published v1 specs (billing + payments read).
//
// Only the sources Slice 7 needs, each column verified in plugin/current
// (2.0.160). sn_invoices, sn_invoice_payment_stages, sn_payments and
// sn_invoice_items are created by class-sn-activator.php (dbDelta, which
// also adds missing columns on upgraded installs) and re-ensured by
// class-sn-plugin.php:690-712 / :894-908.
//
// Amounts are DECIMAL(18,2) in the legacy unit Toman: class-sn-plugin.php
// :31200 ("CRM invoice totals are kept in toman") and class-sn-invoice.php
// :130/:183/:212 (Toman → Rial *10 only at the gateway boundary). Staging
// keeps the raw decimal text; no conversion happens here.
//
// Deliberately NOT extracted (PII, secrets, free text, not needed):
// customer name/phones/address/postal code, access_token, short_code, card
// numbers/owners and manual card/deposit fields, catalog discount snapshot,
// reason/note texts, sn_payments.proof_data (manual proof JSON with account
// owner and card fragments) and submission_key, and the stage note. Gateway
// references (authority, ref_id, payment_ref_id) are staged raw for
// provenance and are only ever exposed hashed. Finance ledger and wallet
// tables are not staged (later slice).

// V1Slice7 returns the specs published by Slice 7 (golden-pinned separately).
func V1Slice7() []Spec {
	return []Spec{invoiceV2(), invoiceItemV1(), invoicePaymentStageV1(), paymentV1()}
}

// invoiceV2 is the second published extraction of sn_invoices: v1 is the
// frozen Slice 5 issued-to identity extract (legacy.invoice_issued_to.v1);
// the registry allows one extraction per table and version, so the broader
// Slice 7 billing extract is published as version 2 of the table.
func invoiceV2() Spec {
	s := invoiceV1Body()
	s.ID, s.Version, s.StagingTable = "legacy.invoice.v2", 2, "stg_invoice_v2"
	return s
}

func invoiceV1Body() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	money := func(name, line string, n Nullability) Column {
		return colNote(name, "DECIMAL(18,2)", n, ClassNone, c(line), "raw decimal text in Toman (class-sn-plugin.php:31200); never converted in staging")
	}
	raw := func(name, typ, line string, n Nullability, note string) Column {
		return colNote(name, typ, n, ClassNone, c(line), note)
	}
	return v1("invoice", "sn_invoices", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("276")),
		raw("invoice_code", "VARCHAR(20)", "277", NotNull, "legacy invoice reference (UNIQUE :359); provenance only"),
		raw("seller_id", "BIGINT UNSIGNED", "280", NotNull, "legacy seller user reference"),
		raw("issued_by_user_id", "BIGINT UNSIGNED", "281", Nullable, "legacy issuer user reference"),
		raw("lead_id", "BIGINT UNSIGNED", "282", Nullable, "explicit native link to sn_leads.id"),
		raw("product_id", "BIGINT UNSIGNED", "291", NotNull, "legacy product reference (live catalog is not historical truth)"),
		money("product_price", "292", NotNull),
		raw("pay_method", "VARCHAR(20)", "295", Nullable, "raw legacy pay method"),
		raw("status", "VARCHAR(60)", "300", NotNull, "legacy MAIN status (default 'pending'); raw"),
		raw("invoice_status", "VARCHAR(60)", "301", Nullable, "legacy invoice status; raw (:549 ADD COLUMN)"),
		raw("payment_status", "VARCHAR(60)", "302", Nullable, "legacy payment status; raw (:552 ADD COLUMN)"),
		raw("receipt_url", "VARCHAR(500)", "303", Nullable, "receipt reference (compatibility projection); raw"),
		raw("receipt_file", "VARCHAR(500)", "304", Nullable, "receipt reference; raw (:554 ADD COLUMN)"),
		raw("receipt_urls", "LONGTEXT", "305", Nullable, "JSON array of receipt URLs (class-sn-helpers.php:527-539); raw"),
		raw("receipt_source", "VARCHAR(50)", "306", Nullable, "raw"),
		raw("payment_source", "VARCHAR(50)", "307", Nullable, "raw"),
		raw("paid_at", "DATETIME", "318", Nullable, "raw legacy DATETIME; zone not asserted"),
		raw("approved_at", "DATETIME", "320", Nullable, "raw legacy DATETIME; zone not asserted"),
		raw("financial_reviewed_at", "DATETIME", "325", Nullable, "raw legacy DATETIME; zone not asserted"),
		money("original_total", "336", Nullable),
		money("discount_total", "337", Nullable),
		money("final_total", "338", Nullable),
		raw("payment_plan", "VARCHAR(20)", "339", NotNull, "raw (default 'full')"),
		money("payment_total_amount", "340", Nullable),
		money("current_due_amount", "341", Nullable),
		money("paid_total_amount", "342", NotNull),
		money("remaining_amount", "343", Nullable),
		raw("current_payment_stage", "INT UNSIGNED", "344", NotNull, "raw"),
		raw("payment_workflow_status", "VARCHAR(60)", "345", NotNull, "legacy payment workflow status (default 'awaiting_payment'); raw"),
		raw("payment_completed_at", "DATETIME", "347", Nullable, "raw legacy DATETIME; zone not asserted"),
		raw("financial_return_state", "VARCHAR(40)", "353", Nullable, "raw"),
		raw("created_at", "DATETIME", "356", NotNull, "raw legacy DATETIME; zone not asserted"),
		raw("updated_at", "DATETIME", "357", NotNull, "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: activatorFile, Lines: "275-365", Note: "table definition of sn_invoices"},
		{File: activatorFile, Lines: "543-604", Note: "columns ensured on upgraded installs"},
		{File: pluginFile, Lines: "31200", Note: "amounts are kept in Toman"},
		{File: "plugin/current/includes/class-sn-invoice-review.php", Lines: "8-12", Note: "legacy TOTAL and STATUS display expressions (evidence only; TAD #19/#20 deferred)"},
	},
		"invoice header, status, amount and receipt-reference columns only; customer PII, tokens, card data and free text are deliberately NOT extracted",
	)
}

func invoiceItemV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("invoice_item", "sn_invoice_items", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("413")),
		col("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("414")),
		colNote("product_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("415"), "legacy product reference"),
		colNote("product_name", "VARCHAR(255)", NotNull, ClassNone, c("416"), "title SNAPSHOT at issuance; historical truth, not the live title"),
		colNote("product_type", "VARCHAR(30)", Nullable, ClassNone, c("417"), "raw line kind"),
		col("qty", "INT UNSIGNED", NotNull, ClassNone, c("418")),
		colNote("unit_price", "DECIMAL(18,2)", NotNull, ClassNone, c("419"), "raw decimal text in Toman"),
		colNote("total_price", "DECIMAL(18,2)", NotNull, ClassNone, c("420"), "raw decimal text in Toman"),
		colNote("is_free", "TINYINT(1)", NotNull, ClassNone, c("421"), "raw"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("422"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: activatorFile, Lines: "412-426", Note: "table definition of sn_invoice_items"},
		{File: pluginFile, Lines: "894-908", Note: "re-ensured definition"},
	})
}

func invoicePaymentStageV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("invoice_payment_stage", "sn_invoice_payment_stages", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("368")),
		col("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("369")),
		colNote("stage_no", "INT UNSIGNED", NotNull, ClassNone, c("370"), "stage order; (invoice_id, stage_no) UNIQUE at :386"),
		colNote("stage_type", "VARCHAR(30)", NotNull, ClassNone, c("371"), "raw (default 'full')"),
		colNote("requested_amount", "DECIMAL(18,2)", NotNull, ClassNone, c("372"), "raw decimal text in Toman"),
		colNote("status", "VARCHAR(60)", NotNull, ClassNone, c("373"), "legacy STAGE status (default 'pending'); raw"),
		colNote("pay_method", "VARCHAR(30)", Nullable, ClassNone, c("374"), "raw"),
		colNote("payment_source", "VARCHAR(60)", Nullable, ClassNone, c("375"), "raw"),
		colNote("receipt_urls", "LONGTEXT", Nullable, ClassNone, c("377"), "JSON array of receipt URLs; raw"),
		colNote("issued_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("378"), "legacy user reference"),
		colNote("approved_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("379"), "legacy user reference recorded on approval"),
		colNote("approved_at", "DATETIME", Nullable, ClassNone, c("380"), "raw legacy DATETIME"),
		colNote("paid_at", "DATETIME", Nullable, ClassNone, c("381"), "raw legacy DATETIME"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("383"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("384"), "raw legacy DATETIME"),
	}, []Evidence{
		{File: activatorFile, Lines: "367-389", Note: "table definition of sn_invoice_payment_stages"},
		{File: pluginFile, Lines: "690-712", Note: "re-ensured definition"},
	},
		"payment_ref_id (gateway reference duplicate) and note are deliberately NOT extracted",
	)
}

func paymentV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("payment", "sn_payments", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("392")),
		col("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("393")),
		colNote("payment_stage_no", "INT UNSIGNED", NotNull, ClassNone, c("394")+"; :636 (ADD COLUMN statement)", "raw stage number"),
		colNote("authority", "VARCHAR(100)", Nullable, ClassSensitive, c("395"), "gateway transaction reference; present only on gateway attempts (class-sn-payment-transactions.php:16); exposed hashed only"),
		colNote("ref_id", "VARCHAR(191)", Nullable, ClassSensitive, c("396"), "gateway reference id; exposed hashed only"),
		colNote("amount", "DECIMAL(18,2)", NotNull, ClassNone, c("397"), "raw decimal text in Toman"),
		colNote("status", "VARCHAR(20)", NotNull, ClassNone, c("398"), "legacy ATTEMPT status (default 'pending'); 'approved' and 'reconcile_error' preserved raw"),
		colNote("pay_method", "VARCHAR(30)", Nullable, ClassNone, c("399")+"; :639 (ADD COLUMN statement)", "raw"),
		colNote("payment_source", "VARCHAR(60)", Nullable, ClassNone, c("400")+"; :642 (ADD COLUMN statement)", "raw (gateway provider or upload source)"),
		colNote("uploaded_by_type", "VARCHAR(20)", Nullable, ClassNone, c("403")+"; :645 (ADD COLUMN statement)", "raw"),
		colNote("uploaded_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("404")+"; :648 (ADD COLUMN statement)", "legacy user reference"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("405"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("406")+"; :651 (ADD COLUMN statement)", "raw legacy DATETIME"),
	}, []Evidence{
		{File: activatorFile, Lines: "391-410", Note: "table definition of sn_payments"},
		{File: activatorFile, Lines: "634-657", Note: "columns ensured on upgraded installs"},
		{File: pluginFile, Lines: "7155-7198", Note: "gateway callback writes approved / reconcile_error / cancelled (D-25 gateway_auto)"},
	},
		"proof_data (manual proof JSON with account owner and card fragments) and submission_key are deliberately NOT extracted",
	)
}
