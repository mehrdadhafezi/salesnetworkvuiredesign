# DB_SCHEMA

## Migration rules
- Use versioned, idempotent migrations.
- Prefer create-if-missing and add-column-if-missing repairs.
- Do not use `DROP`, `TRUNCATE`, or `RENAME` without explicit approval.
- Do not rename legacy columns in production without a migration/rollback plan.
- Store migration/repair reports in WordPress options when useful.

## Invoice code sequence
- Existing invoice codes are preserved as-is, including legacy `INV-*` formats.
- New invoice creation uses exactly six numeric digits.
- The next code comes from the greater of option `sn_invoice_numeric_sequence` and the maximum existing six-digit `sn_invoices.invoice_code`.
- Integration diagnostics should show next numeric code, six-digit count, legacy-format count, and duplicate code groups.

## Legacy sales tables
Known from available code/package and task reports:

- `sn_leads`
  - Purpose: imported/assigned lead records.
  - Important columns: `phone`, `seller_id`, `supervisor_id`, `status`, `lead_status`, `customer_name`, `sales_prediction`, routing fields.
  - Sensitive: phone/customer data.
- `sn_lead_statuses`
  - Configurable lead statuses.
- `sn_lead_status_history`
  - Lead status change history.
- `sn_invoices`
  - Invoice/pre-invoice records.
  - Important columns: `invoice_code`, `access_token`, `seller_id`, `lead_id`, customer fields, amount/final totals, statuses, payment/receipt fields, financial return fields, WooCommerce order id.
  - Sensitive: customer phone, token, payment metadata.
- `sn_invoice_items`
  - Invoice line items.
- `sn_payments`
  - Gateway/manual receipt/payment records.
- `sn_activity_logs`
  - Operational action logs.
- `sn_invoice_wheel`, coupon/discount fields
  - Customer invoice gamification/discount logic.

## Wallet/commission legacy tables
- `sn_wallets`
  - Current wallet balances and totals.
- `sn_wallet_transactions`
  - Credit/debit ledger for legacy wallet/commission.
- Do not change legacy wallet/commission logic during HR/MIS/panel work.

## Commission dry-run tables
- `sn_commission_rules`
  - Draft/new commission engine rules.
  - Important columns: `rule_code`, `title`, `applies_to_position`, `applies_to_level`, `payment_method`, `invoice_status`, `employment_status`, `rate_percent`, `fixed_amount`, min/max invoice amount, `is_active`, `priority`.
  - Rules are not posted to wallets in Phase 4B.
- `sn_commission_dry_runs`
  - Dry-run headers for preview calculations.
  - Important columns: `run_code`, `title`, date range, `status`, `approval_status`, approval/rejection actor and timestamp fields, `locked_at`, `created_by`, `summary_json`, `created_at`.
  - Approval status is review metadata only; approving a run does not post wallet transactions.
- `sn_commission_dry_run_items`
  - Per-invoice preview rows.
  - Important columns: `invoice_id`, `invoice_code`, `seller_user_id`, seller position/level, invoice amount/status/payment method, `matched_rule_id`, `calculated_commission`, `legacy_commission`, `difference_amount`, `item_status`, `wallet_transaction_id`, `posted_at`, `posting_status`, `posting_error`.
  - HR compensation trace fields may exist: `compensation_history_id`, `compensation_effective_from`, `compensation_effective_to`, `rule_source`, `warning_code`, `calculation_date`.
  - Dry-run items must not create/update `sn_wallet_transactions` until a separate approved wallet-posting action is run with feature flag and confirmation.
- `sn_commission_wallet_posting_logs`
  - Audit log for commission run approval/rejection and wallet posting attempts.
  - Important columns: `run_id`, `item_id`, `wallet_transaction_id`, `actor_user_id`, `action`, `status`, `old_value`, `new_value`, `message`, `created_at`.
- `sn_wallet_settlement_batches`
  - Readiness shell for future settlement batches.
  - Important columns: `batch_code`, `status`, `created_by`, `total_items`, `total_amount`, `notes`, timestamps.
  - Phase 4E readiness must not debit wallets or create bank payouts.
- `sn_wallet_settlement_items`
  - Readiness shell for future settlement items.
  - Important columns: `batch_id`, `user_id`, `wallet_id`, `wallet_type`, `suggested_amount`, `item_status`, timestamps.

## HR tables
- `sn_hr_positions`
  - HR positions such as seller, supervisor, senior_supervisor, sales_manager, sales_deputy, finance, hr, mis, after_sales.
- `sn_hr_levels`
  - HR levels such as trainee/probation/standard/senior/manager.
- `sn_hr_profiles`
  - User-to-HR profile, position, level, active state, legacy_role.
  - Phase 3F-1 employee metadata fields: `employee_code`, `employment_status`, `employment_type`, `hire_date`, `termination_date`, `base_salary`, `salary_currency`, `national_id`, `work_phone`, `emergency_phone`, `department_id`, `unit_id`, `team_id`, `notes`.
  - Sensitive display fields: national ID and phone fields must be masked in list/log UI.
- `sn_hr_compensation_profiles`
  - HR-side compensation and commission readiness metadata.
  - Important columns: `profile_id`, `user_id`, `base_salary`, `salary_currency`, `commission_enabled`, `default_commission_rule_id`, `effective_from`, `effective_to`, `notes`, `is_active`.
  - HR may assign a default commission rule for readiness; Finance still owns dry-run, wallet posting, and settlement.
- `sn_hr_compensation_history`
  - Effective-dated HR compensation and commission policy timeline.
  - Multiple rows per user/profile are allowed.
  - Important columns: `profile_id`, `user_id`, `base_salary`, `salary_currency`, `commission_enabled`, `commission_rule_id`, `effective_from`, `effective_to`, `employment_type`, `employment_status`, `position_slug_snapshot`, `level_slug_snapshot`, `notes`, `is_active`, `locked_after_payroll`, `created_by`, `updated_by`, `change_reason`.
  - Open previous rows may be closed to the day before a newly applied period.
  - Rows are not deleted; they are audit/readiness history for payroll and commission review.
- `sn_hr_position_role_mappings`
  - Legacy role → HR position/default level mapping.
- `sn_hr_profile_logs`
  - HR profile changes and manual corrections.
- `sn_hr_assignments`
  - Current direct manager/parent assignments.
- `sn_hr_assignment_history`
  - Assignment changes over time.
- `sn_hr_departments`
  - HR organization departments. Important columns: `slug`, `label`, `description`, `is_active`, `sort_order`, timestamps.
- `sn_hr_units`
  - HR organization units under departments. Important columns: `department_id`, `slug`, `label`, `description`, `is_active`, `sort_order`, timestamps.
- `sn_hr_teams`
  - HR organization teams under units. Important columns: `unit_id`, `slug`, `label`, `description`, `is_active`, `sort_order`, timestamps.

## MIS tables
- `sn_mis_import_batches`
  - Batch metadata: code, title, source, campaign, category, counts, assigned manager, status.
- `sn_mis_data_rows`
  - Staged imported CSV rows.
  - Important columns: `batch_id`, `row_number`/quoted column, customer fields, normalized phone, status, duplicate reason, assigned manager.
  - Must exist before CSV import.
- `sn_mis_distribution_logs`
  - Batch create/import/assign/archive logs.
- `sn_mis_lead_pool`
  - Prepared, manager-assigned MIS rows for future controlled distribution.
  - Important columns: `source_row_id`, `batch_id`, `assigned_manager_user_id`, `assigned_seller_user_id`, `pool_status`, customer fields, normalized phone.
  - Must not create or update live `sn_leads` in Phase 3E-4.1.
  - `source_row_id` is unique to keep Lead Pool preparation idempotent.
- `sn_mis_distribution_rules`
  - Draft MIS distribution rules for later review.
  - Important columns: `rule_name`, `rule_type`, `manager_user_id`, `status`, `config`, actor/timestamps.
  - Supported initial rule types: `equal_split`, `manual_selection`.
- `sn_mis_distribution_plans`
  - Reviewable MIS distribution plans created from ready Lead Pool rows.
  - Important columns: `plan_code`, `rule_id`, `batch_id`, `manager_user_id`, `plan_status`, `total_pool_rows`, `planned_rows`, `skipped_rows`.
  - Plans do not create live leads and do not update `sn_mis_lead_pool.assigned_seller_user_id` in Phase 3E-4.2.
- `sn_mis_distribution_plan_items`
  - Planned seller assignment rows for a distribution plan.
  - Important columns: `plan_id`, `pool_id`, `source_row_id`, `manager_user_id`, `planned_seller_user_id`, `item_status`, `skip_reason`.
  - Lead creation tracking columns: `created_lead_id`, `lead_created_at`, `lead_creation_status`, `lead_creation_error`.
  - Unique `plan_id + pool_id` protects one plan from duplicate items.

## Configurable distribution engine tables
- `sn_distribution_chains`
  - Configurable chain template for post-MIS data distribution.
  - Important columns: `chain_code`, `title`, `description`, `is_active`, `is_default`, timestamps.
  - No delete workflow; deactivate instead.
- `sn_distribution_chain_steps`
  - Ordered HR-position transitions inside a chain.
  - Important columns: `chain_id`, `step_order`, `from_position_slug`, `to_position_slug`, `allow_skip_if_missing`, `is_final_delivery_step`, `creates_live_lead`, `is_active`.
  - Default current chain: `sales_manager → senior_supervisor → supervisor → seller`.
  - `creates_live_lead` is configuration for future/final-stage controlled lead creation; the standard MIS handoff must not create live `sn_leads` before final delivery.
- `sn_distribution_items`
  - Ownership state for each MIS Lead Pool row after it is delivered to the hierarchy engine.
  - Important columns: `source_type`, `source_id`, `batch_id`, `pool_item_id`, `current_owner_user_id`, `current_owner_position`, `current_step_order`, `final_seller_user_id`, `live_lead_id`, `converted_by`, `converted_at`, `conversion_status`, `conversion_error`, `conversion_attempts`, `status`.
  - Unique `source_type + source_id` prevents duplicate delivery of the same pool row.
  - Status examples: `assigned_to_role`, `distributed_forward`, `delivered_to_seller`, `ready_for_lead_conversion`, `live_lead_created`, `duplicate_lead_detected`, `conversion_failed`, `cancelled`.
  - Live `sn_leads` creation is allowed only from a final seller-delivered item through the controlled conversion action. Existing `sn_leads` rows are never updated or deleted by this path.
- `sn_distribution_item_logs`
  - Transfer audit log for distribution items.
  - Important columns: `item_id`, `action`, `actor_user_id`, `from_user_id`, `to_user_id`, `from_position`, `to_position`, `step_order`, `note`, `payload_json`, `created_at`.
  - Payloads must be masked for display and must not expose tokens or public URLs.

## After-sales tables
- `sn_after_sales_cases`
  - Purpose: base after-sales follow-up cases.
  - Important columns: `case_code`, `invoice_id`, `invoice_code`, `lead_id`, customer fields, `seller_user_id`, `assigned_after_sales_user_id`, `case_type`, `case_status`, `priority`, `source`, `opened_at`, `due_at`, `resolved_at`, actor/timestamps, `notes`.
  - Must not modify invoice/payment/wallet rows; invoice/customer context is read-only.
- `sn_after_sales_case_logs`
  - Purpose: audit log for after-sales case creation, status/priority/assignment updates, and notes.
  - Important columns: `case_id`, `actor_user_id`, `action`, `old_value`, `new_value`, `note`, `created_at`.
  - No sensitive tokens or raw payment gateway payloads.

## Campaign attribution tables (2.0.7)

- `sn_campaigns`: canonical UTM tuple, title, budget, actual cost, currency and status.
- `sn_campaign_sessions`: visitor/session identifiers plus immutable first touch and mutable last touch.
- `sn_campaign_events`: deduplicated landing/page-view events with URL, UTM, user agent and optional IP.
- `sn_campaign_attributions`: first/last touch linked to a typed CRM subject. `identity_hash` is a salted HMAC, not raw customer data.
- `sn_campaign_conversions`: registration, lead, approved lead, active customer and sale milestones/revenue.
- `sn_campaign_partner_campaigns`: strict partner-to-campaign allowlist.
- `sn_campaign_partner_tokens`: token hash/prefix/status/usage metadata. Raw API tokens must never be stored.

## Dot validation gift fields (2.0.8)

- `sn_dot_cases.validation_fee_gifted`: audit flag; never means that customer money was received.
- `sn_dot_cases.validation_gifted_by`: supervisor/admin actor that skipped the assessment fee.
- `sn_dot_cases.validation_gifted_at`: time the selection case was opened as a company gift.
- `sn_dot_cases.validation_gift_sms_sent_at`: real dedicated SMS delivery time.
- `sn_dot_cases.validation_gift_sms_error`: last dedicated SMS error marker.
- The related source invoice remains `pre_invoice`; its payment workflow becomes `validation_gifted`, current due becomes zero and its first stage becomes `gifted`.

## Assessment lifecycle and operational archives (2.0.18)

- `sn_dot_cases.converter_decline_reason` and `converter_declined_at`: mandatory-reason snapshot for a final customer withdrawal.
- `sn_dot_cases.operational_archived_at` and `operational_archive_reason`: reversible queue archive metadata; no invoice or payment row is deleted.
- `sn_dot_operational_archives`: additive manager-facing snapshots for `assessment_unpaid` and `subscription_unpaid`.
- The archive table has a unique `(archive_type, entity_kind, entity_id)` key. A later approved payment resolves and may safely reopen the same archive record.
- Existing Dot cases receive nullable defaults and keep their status, owner, selected option and payment totals unchanged.

## Seller stage-one tables (2.0.13)

- `sn_seller_flow_states`
  - Shared current state for both legacy `sn_leads` and MIS/V4 `sn_distribution_items`.
  - Unique source identity: `source_kind + source_id`.
  - Important columns: `seller_user_id`, `sales_manager_user_id`, `flow_status`, `no_answer_attempts`, `third_no_answer_at`, `not_purchase_reason`, `last_activity_at`, `archived_at`, `archive_reason`, `archive_error`, `last_invoice_id`.
  - Archiving is additive and does not delete or rewrite the referenced source.
- `sn_seller_flow_events`
  - Immutable audit history for status transitions, no-answer attempts, reason edits, reassignment, invoice creation and automatic archive.
  - `request_token` is unique when present and protects retryable attempt requests from duplicate increments.
- `sn_invoice_items.product_type`
  - Nullable snapshot with one of `assessment`, `subscription`, `subscription_star` for invoices created by the updated seller flow.
  - Existing invoice items remain valid with `NULL`.

## Protected tables
- `sn_leads`, `sn_invoices`, `sn_payments`, `sn_wallets`, `sn_wallet_transactions` must not be changed by HR/MIS panel work unless explicitly requested.
- MIS import may read `sn_leads` for duplicate/conversion checks only.
- Public invoice token data must not be exposed in panel outputs.

## Needs confirmation
- Exact current v4 columns should be regenerated from the active DB/schema once the current plugin ZIP is available.
- Some tables mentioned in earlier planning reports may exist only in future phases, not current code.
