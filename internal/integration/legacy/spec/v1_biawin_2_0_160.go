package spec

// Slice 10 extension of the published v1 specs (Biawin / fulfillment READ).
//
// Every column verified in plugin/current (2.0.160) at its table
// definition line. Each source is classified for its target owner:
//
//	sn_project_memberships            SUBSCRIPTION        (biawin)
//	sn_project_membership_items       SUBSCRIPTION_ITEM   (biawin) and its
//	                                  follow-up workflow FULFILLMENT_CASE /
//	                                  FULFILLMENT_ASSIGNMENT (fulfillment)
//	sn_project_events                 history (biawin / fulfillment);
//	                                  chat / note keys MESSAGING_ONLY (cases)
//	sn_project_invoice_links          billing link of an item (billing ref)
//	sn_project_actions                FULFILLMENT_CASE (upgrade request)
//	sn_operations_execution_cases     FULFILLMENT_CASE (execution) + ASSIGNMENT
//	sn_operations_execution_events    history (fulfillment)
//	sn_operations_form_submissions    SERVICE_EVIDENCE metadata + SERVICE_OUTCOME
//	sn_shipping_orders                FULFILLMENT_CASE (shipping)
//	sn_shipping_order_logs            history (fulfillment)
//
// Inspected and NOT staged: sn_project_subscriptions / _subscription_contents
// / _card_upgrade_rules / _manager_* routes (NOT_APPLICABLE: catalog and
// routing configuration), sn_project_notes / _messages / _message_reads
// (MESSAGING_ONLY: cases owns messaging), sn_project_commission_rules /
// _commissions (NOT_APPLICABLE: commission, Slice 13), sn_project_api_configs
// (NOT_APPLICABLE: provider credentials, secrets), sn_operations_wallet_codes
// (NOT_APPLICABLE: activation codes, secrets), sn_operations_form_definitions
// (NOT_APPLICABLE: workflow configuration) and sn_product_flow_events (UNKNOWN:
// invoice-level product-flow log with no proven fulfillment meaning). There
// is no sn_project_action_events table in 2.0.160; action history is the
// sn_project_events project_action_* keys.

const (
	operationsExecutionFile = "plugin/current/includes/class-sn-operations-execution.php"
	productFlowFile         = "plugin/current/includes/class-sn-product-flow.php"
)

// V1Slice10 returns the specs published by Slice 10 (golden-pinned separately).
func V1Slice10() []Spec {
	return []Spec{projectMembershipV1(), projectMembershipItemV1(), projectEventV1(), projectInvoiceLinkV1(), projectActionV1(),
		operationsExecutionCaseV1(), operationsExecutionEventV1(), operationsFormSubmissionV1(), shippingOrderV1(), shippingOrderLogV1()}
}

// projectMembershipV1 stages sn_project_memberships (SUBSCRIPTION).
func projectMembershipV1() Spec {
	c := func(line string) string { return projectsFile + ":" + line + " (table definition)" }
	return v1("project_membership", "sn_project_memberships", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("242")),
		colNote("source_key", "VARCHAR(191)", NotNull, ClassNone, c("243"), "raw legacy idempotency key of the membership (source type + invoice / Dot refs); never a customer key"),
		colNote("source_type", "VARCHAR(30)", NotNull, ClassNone, c("244"), "raw source path (e.g. dot, product_star)"),
		colNote("source_invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("245"), "legacy invoice reference (billing link)"),
		colNote("dot_case_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("246"), "legacy Dot case reference (raw; Dot cases are not a published Case source)"),
		colNote("subscription_product_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("247"), "legacy product reference of the bundle (historical; catalog read only)"),
		colNote("subscription_name_snapshot", "VARCHAR(255)", NotNull, ClassNone, c("248"), "title SNAPSHOT at activation; historical evidence, never replaced by the live catalog"),
		colNote("customer_wp_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("249"), "legacy customer account reference (raw; not an identity key)"),
		colNote("origin_user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("252"), "legacy staff user reference (historical origin)"),
		colNote("first_payment_invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("257"), "legacy invoice reference of the activating payment"),
		colNote("status", "VARCHAR(30)", NotNull, ClassNone, c("261"), "raw membership status ('active' on insert / reactivation, 'suspended' on financial reversal)"),
		colNote("activated_at", "DATETIME", NotNull, ClassNone, c("266"), "raw legacy DATETIME of the LAST activation (overwritten on reactivation; zone not asserted)"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("267"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("268"), "raw legacy DATETIME"),
	}, []Evidence{{File: projectsFile, Lines: "241-279", Note: "table definition of sn_project_memberships"}, {File: projectsFile, Lines: "952", Note: "active insert (product_star); activated_at = now"}, {File: projectsFile, Lines: "1059-1069", Note: "suspended -> active reactivation; activated_at overwritten"}, {File: projectsFile, Lines: "1093", Note: "active insert (subscription); activated_at = now"}, {File: projectsFile, Lines: "1241", Note: "active -> suspended on financial reversal"}},
		"customer_name and customer_phone (PII), the portal invite fields and contents_snapshot_json are deliberately NOT extracted",
	)
}

// projectMembershipItemV1 stages sn_project_membership_items (SUBSCRIPTION_ITEM).
func projectMembershipItemV1() Spec {
	c := func(line string) string { return projectsFile + ":" + line + " (table definition)" }
	return v1("project_membership_item", "sn_project_membership_items", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("283")),
		colNote("membership_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("284"), "legacy membership reference"),
		colNote("content_product_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("285"), "legacy service / content product reference (historical)"),
		colNote("content_name_snapshot", "VARCHAR(255)", NotNull, ClassNone, c("286"), "title SNAPSHOT at activation; historical evidence"),
		colNote("manager_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("289"), "legacy staff user reference (current manager)"),
		colNote("expert_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("290"), "legacy staff user reference (current expert; mutable)"),
		colNote("workflow_status", "VARCHAR(30)", NotNull, ClassNone, c("291"), "raw follow-up workflow status (mutable)"),
		colNote("assigned_at", "DATETIME", Nullable, ClassNone, c("295"), "raw legacy DATETIME of the current expert assignment"),
		colNote("status_updated_at", "DATETIME", Nullable, ClassNone, c("296"), "raw legacy DATETIME"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("297"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("298"), "raw legacy DATETIME"),
	}, []Evidence{{File: projectsFile, Lines: "282-305", Note: "table definition of sn_project_membership_items"}, {File: projectsFile, Lines: "2350-2371", Note: "workflow_status writer (waiting / no_answer / follow_up / cancelled / normal / upsell / purchased / regretted)"}},
		"cancel_reason (free text), base_credit_snapshot and upgrade_options_snapshot_json (D-35 credit semantics) and the contact counters are deliberately NOT extracted",
	)
}

// projectEventV1 stages sn_project_events (HISTORY).
func projectEventV1() Spec {
	c := func(line string) string { return projectsFile + ":" + line + " (table definition)" }
	return v1("project_event", "sn_project_events", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("401")),
		colNote("membership_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("402"), "legacy membership reference"),
		colNote("membership_item_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("403"), "legacy membership item reference"),
		colNote("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("404"), "legacy staff user reference (event actor)"),
		colNote("event_key", "VARCHAR(100)", NotNull, ClassNone, c("405"), "raw event key (messaging keys are classified MESSAGING_ONLY by the connector)"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("407"), "raw legacy DATETIME"),
	}, []Evidence{{File: projectsFile, Lines: "400-413", Note: "table definition of sn_project_events"}},
		"details (JSON with free text and reasons) is deliberately NOT extracted",
	)
}

// projectInvoiceLinkV1 stages sn_project_invoice_links (BILLING_LINK).
func projectInvoiceLinkV1() Spec {
	c := func(line string) string { return projectsFile + ":" + line + " (table definition)" }
	return v1("project_invoice_link", "sn_project_invoice_links", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("350")),
		colNote("membership_item_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("351"), "legacy membership item reference"),
		colNote("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("352"), "legacy invoice reference"),
		colNote("content_product_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("353"), "legacy content product reference"),
		colNote("status", "VARCHAR(30)", NotNull, ClassNone, c("355"), "raw link status"),
		colNote("verified_at", "DATETIME", NotNull, ClassNone, c("357"), "raw legacy DATETIME"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("358"), "raw legacy DATETIME"),
	}, []Evidence{{File: projectsFile, Lines: "349-365", Note: "table definition of sn_project_invoice_links"}},
		"line_total and linked_by_user_id are not extracted (billing owns amounts)",
	)
}

// projectActionV1 stages sn_project_actions (FULFILLMENT_CASE).
func projectActionV1() Spec {
	c := func(line string) string { return projectsFile + ":" + line + " (table definition)" }
	return v1("project_action", "sn_project_actions", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("416")),
		colNote("membership_item_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("417"), "legacy membership item reference"),
		colNote("actor_user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("418"), "legacy staff user reference"),
		colNote("action_type", "VARCHAR(20)", NotNull, ClassNone, c("419"), "raw action type (upgrade / upsell request kind)"),
		colNote("fulfillment_mode", "VARCHAR(20)", NotNull, ClassNone, c("420"), "raw fulfillment mode"),
		colNote("target_product_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("421"), "legacy target product reference"),
		colNote("status", "VARCHAR(30)", NotNull, ClassNone, c("426"), "raw action status (payment / remote-provider centric; mapped only where evidenced)"),
		colNote("invoice_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("427"), "legacy invoice reference"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("431"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("432"), "raw legacy DATETIME"),
	}, []Evidence{{File: projectsFile, Lines: "415-438", Note: "table definition of sn_project_actions"}},
		"target_product_name, credits and amounts, payment counters and note (free text) are deliberately NOT extracted",
	)
}

// operationsExecutionCaseV1 stages sn_operations_execution_cases (FULFILLMENT_CASE).
func operationsExecutionCaseV1() Spec {
	c := func(line string) string { return operationsExecutionFile + ":" + line + " (table definition)" }
	return v1("operations_execution_case", "sn_operations_execution_cases", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("102")),
		colNote("operation_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("103"), "legacy operation reference"),
		colNote("membership_item_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("104"), "legacy membership item reference"),
		colNote("content_product_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("105"), "legacy content product reference"),
		colNote("fulfillment_type", "VARCHAR(30)", NotNull, ClassNone, c("106"), "raw execution type (unconfigured / form / wallet_charge / activation_code / ...)"),
		colNote("manager_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("110"), "legacy staff user reference"),
		colNote("expert_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("111"), "legacy staff user reference (current assignee; mutable)"),
		colNote("assigned_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("112"), "legacy staff user reference"),
		colNote("assigned_at", "DATETIME", Nullable, ClassNone, c("113"), "raw legacy DATETIME"),
		colNote("status", "VARCHAR(40)", NotNull, ClassNone, c("114"), "raw execution status"),
		colNote("wallet_status", "VARCHAR(40)", Nullable, ClassNone, c("123"), "raw provider wallet status (evidence only; D-38)"),
		colNote("wallet_external_id", "VARCHAR(191)", Nullable, ClassSensitive, c("121"), "external provider reference; exposed only namespaced and hashed"),
		colNote("shipping_status", "VARCHAR(40)", Nullable, ClassNone, c("127"), "raw shipping status"),
		colNote("shipping_carrier", "VARCHAR(120)", Nullable, ClassNone, c("132"), "raw carrier name"),
		colNote("shipping_tracking_code", "VARCHAR(191)", Nullable, ClassSensitive, c("133"), "tracking reference; exposed only hashed"),
		colNote("form_submission_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("134"), "legacy form submission reference"),
		colNote("final_decision", "VARCHAR(30)", Nullable, ClassNone, c("135"), "raw final decision"),
		colNote("completed_at", "DATETIME", Nullable, ClassNone, c("136"), "raw legacy DATETIME"),
		colNote("completed_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("137"), "legacy staff user reference"),
		colNote("cancelled_at", "DATETIME", Nullable, ClassNone, c("120"), "raw legacy DATETIME"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("138"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("139"), "raw legacy DATETIME"),
	}, []Evidence{{File: operationsExecutionFile, Lines: "101-142", Note: "table definition of sn_operations_execution_cases"}},
		"shipping address / province / city / postal code (PII), wallet_activation_code and wallet_customer_url (secrets), wallet_usage_json, snapshots, last_note and cancel_reason (free text) are deliberately NOT extracted",
	)
}

// operationsExecutionEventV1 stages sn_operations_execution_events (HISTORY).
func operationsExecutionEventV1() Spec {
	c := func(line string) string { return operationsExecutionFile + ":" + line + " (table definition)" }
	return v1("operations_execution_event", "sn_operations_execution_events", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("160")),
		colNote("execution_case_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("161"), "legacy execution case reference"),
		colNote("operation_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("162"), "legacy operation reference"),
		colNote("membership_item_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("163"), "legacy membership item reference"),
		colNote("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("164"), "legacy staff user reference (event actor)"),
		colNote("event_key", "VARCHAR(90)", NotNull, ClassNone, c("165"), "raw event key"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("167"), "raw legacy DATETIME"),
	}, []Evidence{{File: operationsExecutionFile, Lines: "159-169", Note: "table definition of sn_operations_execution_events"}},
		"details_json is deliberately NOT extracted",
	)
}

// operationsFormSubmissionV1 stages sn_operations_form_submissions (SERVICE_EVIDENCE).
func operationsFormSubmissionV1() Spec {
	c := func(line string) string { return operationsExecutionFile + ":" + line + " (table definition)" }
	return v1("operations_form_submission", "sn_operations_form_submissions", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("144")),
		colNote("execution_case_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("145"), "legacy execution case reference"),
		colNote("form_definition_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("146"), "legacy form definition reference"),
		colNote("submitted_by_user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("149"), "legacy staff user reference"),
		colNote("submitted_at", "DATETIME", NotNull, ClassNone, c("150"), "raw legacy DATETIME"),
		colNote("final_decision", "VARCHAR(30)", Nullable, ClassNone, c("151"), "raw final decision (service outcome evidence)"),
		colNote("decided_by_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("153"), "legacy staff user reference"),
		colNote("decided_at", "DATETIME", Nullable, ClassNone, c("154"), "raw legacy DATETIME"),
	}, []Evidence{{File: operationsExecutionFile, Lines: "143-158", Note: "table definition of sn_operations_form_submissions"}},
		"values_json (customer-entered answers), schema_snapshot_json and final_note are deliberately NOT extracted: evidence METADATA only",
	)
}

// shippingOrderV1 stages sn_shipping_orders (FULFILLMENT_CASE).
func shippingOrderV1() Spec {
	c := func(line string) string { return productFlowFile + ":" + line + " (table definition)" }
	return v1("shipping_order", "sn_shipping_orders", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("70")),
		colNote("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("71"), "legacy invoice reference"),
		colNote("seller_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("72"), "legacy staff user reference"),
		colNote("customer_wp_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("73"), "legacy customer account reference (raw)"),
		colNote("contact_status", "VARCHAR(30)", NotNull, ClassNone, c("82"), "raw contact status"),
		colNote("fulfillment_status", "VARCHAR(40)", NotNull, ClassNone, c("83"), "raw shipping fulfillment status"),
		colNote("shipping_method", "VARCHAR(80)", Nullable, ClassNone, c("84"), "raw shipping method"),
		colNote("carrier", "VARCHAR(120)", Nullable, ClassNone, c("85"), "raw carrier name"),
		colNote("tracking_code", "VARCHAR(120)", Nullable, ClassSensitive, c("86"), "tracking reference; exposed only hashed"),
		colNote("handled_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("89"), "legacy staff user reference (current handler)"),
		colNote("shipped_at", "DATETIME", Nullable, ClassNone, c("91"), "raw legacy DATETIME"),
		colNote("delivered_at", "DATETIME", Nullable, ClassNone, c("92"), "raw legacy DATETIME"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("93"), "raw legacy DATETIME"),
		colNote("updated_at", "DATETIME", NotNull, ClassNone, c("94"), "raw legacy DATETIME"),
	}, []Evidence{{File: productFlowFile, Lines: "69-100", Note: "table definition of sn_shipping_orders"}},
		"customer_name, customer_phone, province, city, address, postal_code (PII), items_snapshot_json, total_amount, expert_note and failure_reason (free text) are deliberately NOT extracted",
	)
}

// shippingOrderLogV1 stages sn_shipping_order_logs (HISTORY).
func shippingOrderLogV1() Spec {
	c := func(line string) string { return productFlowFile + ":" + line + " (table definition)" }
	return v1("shipping_order_log", "sn_shipping_order_logs", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("102")),
		colNote("shipping_order_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("103"), "legacy shipping order reference"),
		colNote("invoice_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("104"), "legacy invoice reference"),
		colNote("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("105"), "legacy staff user reference (event actor)"),
		colNote("event_key", "VARCHAR(80)", NotNull, ClassNone, c("106"), "raw event key"),
		colNote("old_status", "VARCHAR(40)", Nullable, ClassNone, c("107"), "raw previous status"),
		colNote("new_status", "VARCHAR(40)", Nullable, ClassNone, c("108"), "raw new status"),
		colNote("created_at", "DATETIME", NotNull, ClassNone, c("110"), "raw legacy DATETIME"),
	}, []Evidence{{File: productFlowFile, Lines: "101-115", Note: "table definition of sn_shipping_order_logs"}},
		"details_json is deliberately NOT extracted",
	)
}
