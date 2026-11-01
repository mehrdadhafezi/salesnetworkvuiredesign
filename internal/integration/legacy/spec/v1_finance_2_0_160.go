package spec

// Slice 9 extension of the published v1 specs (finance shadow ledger).
//
// Only the legacy wallet balance and wallet transaction tables, each column
// verified in plugin/current (2.0.160). sn_wallets / sn_wallet_transactions
// are created by three writers with the same columns
// (class-sn-plugin.php:35534-35572, class-sn-purpose-commission.php:111-139,
// class-sn-projects.php:588-622; the projects variant has no
// idempotency_key column, which dbDelta adds back on the plugin path).
//
// Wallets are STAFF wallets keyed (user_id, wallet_type): the writers are
// seller / supervisor commission (class-sn-plugin.php:35588-35635), purpose
// matrix commission (class-sn-purpose-commission.php:562-582) and Biawin
// project commission (class-sn-projects.php:1159-1180). Amounts are
// DECIMAL(18,2) computed from Toman invoice / stage amounts (rate x amount,
// rounded to 2 decimals by the legacy writer): raw text is staged, never
// rounded or converted. The legacy balance is a MUTABLE running total
// (UPDATE ... SET balance = balance +/- amount) and legacy reversals rewrite
// the original credit's type and idempotency key in place
// (class-sn-plugin.php:29327-29330, :30811-30813;
// class-sn-purpose-commission.php:688): both are evidence, not truth.
//
// Deliberately NOT extracted: the transaction description (free text) and
// created_by. meta (JSON) is staged SENSITIVE because it can carry a free
// text reason; the connector reads only the numeric reversal-link and stage
// keys from it. Not staged at all (inspected): the commission wallet
// posting logs (a commission-run audit trail, Slice 13 scope; no balance
// effect of its own) and the operations wallet codes (service activation
// codes: secrets, not money). Customer cash / installment wallets live in
// the external Biawin service (class-sn-biavin-api.php:55-66, D-38): no
// legacy table holds them.

const (
	purposeCommissionFile = "plugin/current/includes/class-sn-purpose-commission.php"
	projectsFile          = "plugin/current/includes/class-sn-projects.php"
)

// V1Slice9 returns the specs published by Slice 9 (golden-pinned separately).
func V1Slice9() []Spec {
	return []Spec{walletV1(), walletTransactionV1()}
}

func walletV1() Spec {
	c := func(line string) string { return pluginFile + ":" + line + " (table definition)" }
	money := func(name, line string) Column {
		return colNote(name, "DECIMAL(18,2)", NotNull, ClassNone, c(line),
			"raw decimal text; legacy MUTABLE running total (UPDATE balance = balance +/- amount); evidence, not truth")
	}
	return v1("wallet", "sn_wallets", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("35535")),
		colNote("user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("35536"), "legacy staff user reference (wallet holder)"),
		colNote("wallet_type", "VARCHAR(30)", NotNull, ClassNone, c("35537"), "raw wallet type (seller / supervisor / biavin written; others kept raw)"),
		money("balance", "35538"),
		money("total_credit", "35539"),
		money("total_debit", "35540"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("35541"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("35542"), "raw legacy DATETIME"),
	}, []Evidence{
		{File: pluginFile, Lines: "35534-35546", Note: "table definition of sn_wallets"},
		{File: purposeCommissionFile, Lines: "111-121", Note: "same definition (purpose matrix writer)"},
		{File: projectsFile, Lines: "588-600", Note: "same definition (Biawin project writer)"},
		{File: pluginFile, Lines: "35626-35631", Note: "balance mutated in place on every transaction insert"},
	})
}

func walletTransactionV1() Spec {
	c := func(line string) string { return pluginFile + ":" + line + " (table definition)" }
	return v1("wallet_transaction", "sn_wallet_transactions", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("35548")),
		colNote("wallet_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("35549"), "legacy wallet reference"),
		colNote("user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("35550"), "legacy staff user reference (historical recipient)"),
		colNote("wallet_type", "VARCHAR(30)", NotNull, ClassNone, c("35551"), "raw wallet type"),
		colNote("invoice_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("35552"), "legacy invoice reference"),
		colNote("lead_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("35553"), "legacy lead reference"),
		colNote("amount", "DECIMAL(18,2)", NotNull, ClassNone, c("35554"), "raw decimal text; positive, direction carries the sign"),
		colNote("direction", "VARCHAR(10)", NotNull, ClassNone, c("35555"), "raw: credit / debit"),
		colNote("type", "VARCHAR(60)", NotNull, ClassNone, c("35556"), "raw transaction type; may be rewritten in place by legacy reversal (*_reversed / *_reopened)"),
		colNote("status", "VARCHAR(30)", NotNull, ClassNone, c("35557"), "raw; every 2.0.160 writer inserts 'approved'"),
		colNote("meta", "LONGTEXT", Nullable, ClassSensitive, c("35559"),
			"JSON; may hold a free text reason; only the numeric reversal-link and stage keys are ever read"),
		colNote("idempotency_key", "VARCHAR(191)", Nullable, ClassNone, c("35560"), "raw; nulled in place on legacy reversal"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("35562"), "raw legacy DATETIME (site-local; zone not asserted)"),
	}, []Evidence{
		{File: pluginFile, Lines: "35547-35572", Note: "table definition of sn_wallet_transactions"},
		{File: pluginFile, Lines: "35588-35635", Note: "seller / supervisor / commission engine writer"},
		{File: pluginFile, Lines: "29297-29331", Note: "legacy commission reversal: debit with original_credit_transaction_id; original type rewritten"},
		{File: pluginFile, Lines: "30795-30815", Note: "finance reopen reversal: debit with reopened_credit_transaction_id; original type rewritten"},
		{File: purposeCommissionFile, Lines: "562-640", Note: "purpose matrix writer (idempotency key purpose-commission:...)"},
		{File: purposeCommissionFile, Lines: "675-689", Note: "purpose matrix reversal: debit with original_credit_transaction_id"},
		{File: projectsFile, Lines: "1159-1260", Note: "Biawin project commission, recredit and reversal (original_transaction_id)"},
	},
		"description (free text) and created_by are deliberately NOT extracted",
	)
}
