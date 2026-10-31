package spec

// Slice 5 extension of the published v1 specs (customer identity shadow).
//
// Slice 3 deliberately deferred billing extract specs. Slice 5 needs only the
// invoice ISSUED-TO identity evidence, so this spec extracts exactly those
// columns of the 2.0.160 invoice table (raw, verbatim, integration-owned) —
// not the invoice: no amounts, statuses, payment, address or card columns.
// The full billing spec belongs to Slice 7 and will be a separate spec id.
//
// Every column below is verified in plugin/current (2.0.160). Notably the
// legacy customer account id is NOT independent identity evidence: 2.0.160
// creates/looks up the customer WordPress account by the normalized phone
// (user_login = phone) and writes that id onto the invoice.

const activatorFile = "plugin/current/includes/class-sn-activator.php"

// V1Slice3 returns the specs published by Slice 3 (frozen; golden-pinned).
func V1Slice3() []Spec {
	return []Spec{
		wpUserV1(), wpUsermetaV1(),
		hrPositionV1(), hrLevelV1(), hrProfileV1(), hrPositionRoleMappingV1(),
		hrDepartmentV1(), hrUnitV1(), hrTeamV1(),
		hrAssignmentV1(), hrAssignmentHistoryV1(),
	}
}

// V1Slice5 returns the specs published by Slice 5 (golden-pinned separately).
func V1Slice5() []Spec { return []Spec{invoiceIssuedToV1()} }

func invoiceIssuedToV1() Spec {
	c := func(line string) string { return activatorFile + ":" + line + " (table definition)" }
	return v1("invoice_issued_to", "sn_invoices", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("276")),
		colNote("invoice_code", "VARCHAR(20)", NotNull, ClassNone, c("277")+"; :359 (UNIQUE KEY)",
			"legacy invoice reference; evidence reference only, never a customer identifier"),
		colNote("customer_wp_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("283")+"; :543 (ADD COLUMN statement)",
			"legacy customer account id; PHONE-DERIVED (class-sn-plugin.php:5023-5048 get_or_create_customer_account looks the account up by normalized phone; :4479-4481, :4949-4951 write it); never independent identity evidence"),
		colNote("customer_name", "VARCHAR(120)", NotNull, ClassPersonal, c("284"), "issued-to name as typed; raw"),
		colNote("customer_phone", "VARCHAR(20)", NotNull, ClassPersonal, c("285"), "raw; discovery evidence only, never identity"),
		colNote("customer_phone_secondary", "VARCHAR(20)", Nullable, ClassPersonal, c("286"), "raw; discovery evidence only, never identity"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("356"), "raw legacy DATETIME; zone not asserted"),
	}, []Evidence{
		{File: activatorFile, Lines: "275-362", Note: "table definition of sn_invoices (issued-to columns only)"},
		{File: activatorFile, Lines: "542-544", Note: "customer_wp_id added on upgraded installs"},
		{File: pluginFile, Lines: "4479-4481, 4949-4951, 5023-5048", Note: "customer_wp_id is the account found/created by normalized phone"},
	},
		"issued-to identity evidence only; amounts, statuses, payment, card, address and postal columns are deliberately NOT extracted (billing is Slice 7)",
	)
}
