# API_CONTRACTS

## Response conventions
- AJAX handlers should return JSON with `wp_send_json_success()` / `wp_send_json_error()`.
- Do not return raw PHP errors to frontend.
- Public/customer errors should be safe and not reveal internal state.
- Internal/admin endpoints must use nonce + capability/role checks.

## Public/customer invoice actions
Known actions from available code and reports:
- `sn_invoice_info`
- `sn_pay_online`
- `sn_upload_receipt`
- `sn_submit_manual_payment`
- `sn_invoice_recontact`
- `sn_spin_invoice_wheel`
- `sn_apply_invoice_wheel_reward`
- `sn_apply_invoice_coupon`
- `sn_remove_invoice_coupon`
- `sn_invoice_customer_action`
- `sn_invoice_customer_actions_batch`

Rules:
- Public/customer actions require invoice code + valid `access_token`.
- Link without token must reject safely.
- Token must not be required for ZarinPal callback.
- Internal/admin invoice actions must not use public token validation.
- New pre-invoices receive exactly six numeric invoice codes; existing legacy codes remain accepted by read/search/payment paths.

## Seller actions
Known actions include:
- `sn_seller_leads`
- `sn_seller_invoices`
- `sn_create_invoice`
- `sn_update_lead_status`
- `sn_seller_resend_financial`
- `sn_seller_customer_actions`

Rules:
- Seller read/list scope: seller sees own records.
- Invoice code is plain text; no public URL/token in seller panel.
- Pre-invoice success message should show invoice code and SMS status, then optionally switch to invoice tab.

## Supervisor / senior supervisor actions
Known actions include:
- `sn_supervisor_data`
- `sn_supervisor_invoices`
- `sn_get_unassigned`
- `sn_assign_supervisor_leads`
- `sn_supervisor_unassign_leads`
- `sn_lead_profile`
- `sn_seller_profile`

Rules:
- Senior supervisor panel should use HR descendant seller scope.
- Current Phase panels are read-only unless explicitly using existing legacy actions.
- No public invoice links/tokens in supervisor/senior supervisor panels.

## Sales manager actions
Known actions include:
- `sn_sales_manager_leads`
- `sn_sales_manager_export`
- shared invoice listing via supervisor invoice code paths in reports.

Rules:
- Sales manager scope is descendants under HR hierarchy when `enforce_sales_management` is active.
- Exports made scope-aware only for read-only reports.

## Finance actions
Known actions include:
- `sn_financial_invoices`
- `sn_financial_approve_payment`
- `sn_financial_reject_payment`
- `sn_commission_save_rule`
- `sn_commission_run_dry_run`
- `sn_commission_review_run`
- `sn_commission_save_posting_flag`
- `sn_commission_post_wallet`

Rules:
- List/view requires `sn_view_finance` or `manage_options`.
- Approve requires `sn_approve_payment` or `manage_options`.
- Reject requires `sn_reject_payment` or `manage_options`.
- Commission rule/dry-run actions require finance view access and nonce.
- Commission dry-run writes only to `sn_commission_dry_runs` and `sn_commission_dry_run_items`.
- Commission dry-run must not credit/debit wallets, call legacy wallet write functions, update invoices, or send SMS.
- Commission run review requires finance view access, nonce, and confirmation text:
  - approve requires `APPROVE`
  - reject requires `REJECT`
  - review only changes commission run approval metadata and logs the action.
- Commission wallet posting is guarded by `sn_enable_commission_wallet_posting`.
  - Default is off.
  - Posting requires an approved run, nonce, finance view access, and confirmation text `APPLY`.
  - Posting may create wallet credit transactions with type `commission_engine` only for eligible, unposted dry-run items.
  - Posting must be idempotent by dry-run item and must not call legacy invoice auto-credit/recalculate paths.

## HR admin/post actions
Current HR panel/actions from phase reports:
- create/update position metadata.
- create/update level metadata.
  - Level activate/deactivate uses the same `sn_hr_save_level` action, nonce, and HR/admin permission; it only updates `is_active` and never deletes levels.
  - `sn_hr_save_level` accepts `level_status_action=activate|deactivate|save`; after writing, it re-reads the level row and reports success only if the stored active value matches the expected value.
  - Explicit `activate`/`deactivate` actions ignore `is_active` checkbox values and update only the active column plus timestamp. The edit/save form is the only path that uses posted label/description/checkbox fields.
- create HR profile.
- change HR position/level/active state.
- assign/end direct manager assignment.
- `sn_hr_profile_action`
  - Dry-run/apply profile action from `[sn_hr_panel]`.
  - Can update employee metadata (`employee_code`, employment status/type, dates, salary, department/unit/team, masked-sensitive HR fields), position, level, active state, and direct manager.
  - Can update HR compensation readiness metadata: salary currency, commission eligibility, default commission rule, effective dates, and notes.
  - Accepts readable selector fields `user_lookup_select` and `manager_user_id`; raw lookup fields remain fallback/debug.
  - Apply requires confirmation text `APPLY`.
- `sn_hr_compensation_timeline_action`
  - Adds a new effective-dated compensation/commission history row for an existing HR profile.
  - Requires HR/admin access, nonce `sn_hr_compensation_timeline_action`, reason text, and confirmation text `APPLY`.
  - May close one open previous compensation history row to the day before the new start date.
  - Writes HR compensation history and HR logs only; it does not calculate commission, post wallet transactions, change invoices/payments, or send SMS.
- `sn_hr_save_structure`
  - Add/edit department, unit, or team metadata.
  - No delete; existing slug is not changed on edit.
- `sn_hr_csv_dry_run`
  - Parses HR CSV for validation only.
  - Writes only `sn_hr_csv_dry_run_report`; it does not create/update HR profiles.
- `sn_restore_seller_hierarchy_from_leads`
  - Admin-post tool for Integration / HR repair tools.
  - Access: `manage_options` only.
  - Nonce: `sn_restore_seller_hierarchy_from_leads`.
  - Fields: `restore_mode=dry_run|apply`, `confirm_apply`, `only_unassigned_sellers`, optional `supervisor_filter`, optional `seller_filter`, `conflict_strategy=keep_existing|report_conflict|overwrite`, optional `confirm_override=OVERRIDE`, `reason`.
  - Dry-run reads `sn_leads.seller_id` and `sn_leads.supervisor_id` to infer seller → historical supervisor.
  - Apply requires `confirm_apply=APPLY` and uses `SN_Hierarchy_Service::assign_parent()` to create HR assignments.
  - Existing seller manager assignments are preserved by default; overwrite is allowed only with `conflict_strategy=overwrite` and `confirm_override=OVERRIDE`.
  - Writes latest report to `sn_hr_legacy_lead_hierarchy_restore_report`.
  - Must not change WordPress roles, `legacy_role`, legacy user_meta, HR position/level, payments, invoices, SMS, wallet/commission, MIS, or After Sales data.

Rules:
- Access: `manage_options` or HR position `hr`, except explicitly admin-only repair tools such as `sn_restore_seller_hierarchy_from_leads`.
- Sensitive changes require dry-run and apply confirmation `APPLY`.
- Must log reason and actor.
- WordPress roles, `legacy_role`, and legacy user_meta must remain display/compatibility data unless a future task explicitly changes them.

## MIS actions
Current MIS panel/actions from phase reports:
- create draft batch.
- import CSV to MIS staging.
- preview batch.
- assign valid rows to sales manager.
- prepare Lead Pool from manager-assigned MIS staging rows.
- create draft distribution rules.
- create dry-run/review distribution plans from Lead Pool rows.
- archive batch.
- repair MIS schema.

Rules:
- Access: `manage_options` or HR position `mis`.
- CSV import must not write to `sn_leads`.
- Assign batch to sales manager only updates MIS staging rows.
- Lead Pool preparation only inserts into `sn_mis_lead_pool`; it must not create live `sn_leads` or assign rows to sellers.
- Distribution planning only inserts into `sn_mis_distribution_rules`, `sn_mis_distribution_plans`, and `sn_mis_distribution_plan_items`; it must not create live `sn_leads`, update `sn_mis_lead_pool.assigned_seller_user_id`, or send SMS.
- Distribution plan submit uses `seller_ids[]` for selected sellers. Server-side handling must sanitize IDs, remove duplicates, and keep only sellers inside the selected sales manager scope.
- Controlled lead creation from MIS is allowed only from an approved distribution plan, requires dry-run/apply with confirmation, may only `INSERT` new rows into `sn_leads`, and must never update/delete existing leads or send SMS.
- If `sn_mis_data_rows` is missing, import must be blocked.

## Hierarchical distribution engine actions

### `admin_post_sn_distribution_save_chain`
- Access: `manage_options` or HR position `hr`.
- Nonce: `sn_distribution_save_chain`.
- Modes:
  - `distribution_save_mode=chain`: add/edit chain metadata.
  - `distribution_save_mode=step`: add/edit chain step metadata.
- Writes only distribution chain tables.
- Does not change HR profiles, WordPress roles, MIS rows, live leads, invoices, payments, wallets, or commissions.
- No delete action; deactivate with `is_active=0`.

### `admin_post_sn_distribution_deliver_pool`
- Access: `manage_options` or HR position `mis`.
- Nonce: `sn_distribution_deliver_pool`.
- Fields: `batch_id`, optional `manager_user_id`, `run_mode=dry_run|apply`, `reason`, `confirm`.
- Apply requires `confirm=APPLY`.
- Reads `sn_mis_lead_pool` rows with `pool_status=ready` and `assigned_manager_user_id > 0`.
- Inserts idempotent rows into `sn_distribution_items` with `source_type=mis_pool`.
- Logs `item_created_from_mis_pool`.
- Must not insert/update/delete `sn_leads` and must not send SMS.

### `admin_post_sn_distribution_transfer_items`
- Access: logged-in user.
- Nonce: `sn_distribution_transfer_items`.
- Fields: `item_ids[]`, `to_user_id`, `note`, `confirm`.
- Apply requires `confirm=APPLY`.
- Actor must own the item unless `manage_options`.
- Recipient must be allowed by the active distribution chain and be inside the actor HR subtree.
- Updates only distribution item ownership/status and logs the transfer.
- If the recipient is final seller, status becomes `delivered_to_seller`; live lead creation remains separate/controlled unless a later approved phase enables it safely.

### `admin_post_sn_distribution_convert_items`
- Access: logged-in final seller for the delivered item, or `manage_options` / super admin.
- Nonce: `sn_distribution_convert_items`.
- Fields:
  - Single item: `item_id`, optional `reason`.
  - Bulk: `item_ids[]`, optional `reason`, `confirm=APPLY`.
- Eligible items must be in `delivered_to_seller` or `ready_for_lead_conversion` status and must have a final seller.
- Inserts at most one new row into `sn_leads` for each eligible item.
- Does not update/delete existing `sn_leads`.
- Detects duplicate phones before insert; duplicate items are marked on `sn_distribution_items` and logged.
- Stores conversion tracking on `sn_distribution_items`: `live_lead_id`, `converted_by`, `converted_at`, `conversion_status`, `conversion_error`, `conversion_attempts`.
- Must not send SMS, create invoices, touch payments, wallets, commissions, public invoice tokens, HR, MIS staging rows, or After Sales cases.

## After-sales actions
Current after-sales panel/actions:
- `sn_after_sales_create_case`
- `sn_after_sales_update_case`

Rules:
- Access: `manage_options`, HR position `after_sales`, or legacy role/capability `sn_after_sales` / `sn_view_customer_profiles`.
- Actions must use nonce + permission checks.
- Case creation writes only to `sn_after_sales_cases` and logs `case_created`.
- Status/priority/assignment/note updates write only to after-sales case tables and logs.
- Invoice/customer context is read-only; no invoice, payment, wallet, commission, SMS, public invoice token, or MIS rows are modified.
- Public invoice URLs and `access_token` must not be shown.

## Operations execution and wallet adapter

### `admin_post_sn_operations_execution_action`

- Access:
  - assignment: administrator or «مدیر اجرایی عملیات»;
  - card work: the assigned «کارشناس اجرایی عملیات» or administrator.
- Assignment nonce: `sn_operations_execution_assign`.
- Card nonce: `sn_operations_execution_case_{case_id}`.
- Supported card actions: `no_answer`, `follow_up`, `cancel`, `wallet_charge`, `wallet_refresh`, `wallet_assisted_login`, `physical_confirm`, `form_save`, `form_execute`, `form_skip`.
- Terminal records cannot be changed through execution-expert actions.

### `admin_post_sn_operations_execution_shipping_update`

- Access: `manage_options` or `sn_manage_shipping`.
- Nonce: `sn_operations_execution_shipping_{case_id}`.
- Updates only the physical execution case and its operations-stage mirror.
- Address is mandatory for preparation/shipping/delivery; carrier and tracking code are mandatory for shipping/delivery; failure reason is mandatory for not-sent/returned/cancelled.

### Wallet adapter

- Integration mechanisms:
  1. return a response array from `sn_operations_wallet_request`;
  2. configure an HTTPS JSON endpoint and optional Bearer token.
- Request payload:
  - `action`: `charge`, `status`, or `assisted_login`;
  - `request_id`: stable `sn-execution-{case_id}` idempotency key;
  - `operation_id`;
  - `customer`: WordPress ID, name and normalized phone;
  - `card`: product ID, name, credit and normal/upsell mode;
  - `service_key`;
  - `external_id` when available.
- Response for `charge`:
  - `success=true`;
  - required `activation_code`;
  - optional `external_id`, HTTPS `customer_url`, and `usage` object/array.
- Response for `status`: `success=true`, optional `status`, `activation_code`, HTTPS `customer_url`, and `usage`.
- Response for `assisted_login`: `success=true` and a destination-issued HTTPS one-time `url`.
- The destination must make charge idempotent by `request_id`; CRM also blocks concurrent/repeated charge submits.
- The CRM must never request, store or set the customer password and must not bypass customer OTP locally.

## Customer profile and reports
Shortcodes/actions:
- `[sn_customer_profile]`
  - Read-only Customer 360 view.
  - Filters: `sn_customer_q`, `sn_invoice_code`, `sn_lead_id`.
  - Shows identity, leads, invoices, payments, activity, and after-sales cases according to viewer scope.
- `[sn_reports_panel]`
  - Read-only cross-module report panel.
  - Filters: `date_from`, `date_to`, `seller_id`, `status`.
- `admin_post_sn_export_report_csv`
  - CSV export for supported report types.
  - Requires logged-in authorized viewer and nonce `sn_export_report_csv`.

Rules:
- Exports must be scope-aware, limited, formula-safe, and must not include `access_token`, public invoice URLs, passwords, hashes, or full sensitive payloads.
- Customer/invoice context links are internal chips only and must not be public invoice/customer payment links.
- These views are read-only and must not create or update leads, invoices, payments, wallets, commissions, MIS rows, HR profiles, or after-sales cases.

## Needs confirmation
- Exact current action names for HR/MIS handlers must be read from active v4 source before editing.
- Nonce names differ between legacy and new paths; verify before changing frontend code.
## Portal Page Setup + Navigation

### Shortcode: `[sn_portal_nav]`
- Renders permission-aware internal CRM navigation for logged-in users.
- Guest output is empty.
- Navigation targets are resolved from `sn_portal_page_ids` and safe page registry fallbacks.
- Links must never include `access_token`, `token`, `public_invoice_url`, or `invoice_url`.

### Admin action: `admin_post_sn_portal_page_setup`
- Capability: `manage_options`.
- Nonce: `sn_portal_page_setup`.
- Fields:
  - `mode`: `dry_run`, `refresh`, or `apply`.
  - `confirm`: required value `APPLY` for `apply`.
- Behavior:
  - `dry_run`: reports missing/existing/conflicting portal pages without writing.
  - `refresh`: refreshes stored page status/report without creating pages.
  - `apply`: creates missing pages only; does not overwrite existing pages with conflicting content.
- Options:
  - `sn_portal_page_ids`: associative page registry IDs.
  - `sn_portal_page_setup_report`: latest setup/refresh report.

### Portal Registry
- Internal CRM pages:
  - `crm_dashboard` → `[sn_my_panel]`
  - `seller` → `[sn_seller_panel]`
  - `supervisor` → `[sn_supervisor_panel]`
  - `senior_supervisor` → `[sn_senior_supervisor_panel]`
  - `sales_manager` → `[sn_sales_manager_panel]`
  - `sales_deputy` → `[sn_sales_deputy_panel]`
  - `finance` → `[sn_financial_panel]`
  - `hr` → `[sn_hr_panel]`
  - `mis` → `[sn_mis_panel]`
  - `after_sales` → `[sn_after_sales_panel]`
  - `reports` → `[sn_reports_panel]`
  - `customer` → `[sn_customer_profile]`
  - `public_invoice` → `[sn_invoice_page]`

## Operational UX hotfix contracts

### `sn_save_customer_info`
- Used by the seller main lead/customer editor for field-level autosave.
- Autosaved fields: customer name, province, city, sales prediction, contact status, and note.
- Requires logged-in seller/admin access, `sn_public` nonce, and lead ownership/scope.
- Must not create leads, invoices, SMS, payments, wallet rows, or commission rows.

### `sn_invoice_recontact`
- Public route: requires `sn_public` nonce, `invoice_code`, `access_token`, and public invoice validator.
- Internal route: logged-in authorized users may submit `invoice_code` or `invoice_id` with a valid `sn_public` or `sn_admin` nonce and no `access_token`.
- Internal access uses read/scope permission and must not expose public invoice URLs or tokens.

### `sn_seller_resend_financial`
- Allows seller owner/admin to resend a rejected invoice for finance review.
- Inputs: `invoice_id`, `note`, optional corrected transfer fields `card_from`, `card_to`, `amount`, `paid_at`, and optional file field `receipt`.
- Receipt re-upload uses the existing receipt upload validation for image/PDF type and size.
- Does not create a new invoice, auto-approve payment, trigger gateway callbacks, require public token, or expose public invoice URLs.

### MIS operator logs
- MIS operator-facing log rows should show Persian summaries for batch creation, manager assignment, Lead Pool preparation, distribution plan actions, and controlled lead creation.
- Raw JSON belongs inside collapsed technical details and must mask phone/token/password/url values.

## HR Hierarchy Reporting + Assignment Hotfix

### Central HR hierarchy helpers
- `sn_get_direct_children($manager_user_id, $position_filter = null)` reads current HR assignment children for one manager.
- `sn_get_descendant_users($manager_user_id, $position_filter = null)` reads nested HR descendants.
- `sn_get_seller_ids_in_hr_scope($actor_user_id)` returns seller IDs visible to the actor from HR assignments.
- `sn_get_hierarchy_scope_summary($actor_user_id)` returns counts used for diagnostics and report consistency.
- Current reporting must use HR assignment/current direct manager data, not legacy `sn_leads.supervisor_id`.

### `admin_post_sn_hr_bulk_assign_hierarchy`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_bulk_assign_hierarchy`.
- Inputs: `assignment_type`, `selected_user_ids[]`, `target_manager_user_id`, `overwrite_existing`, `reason`, `confirm_apply`, submit mode `dry_run`/`apply`.
- Apply requires `confirm_apply=APPLY`.
- Writes only HR assignment relationships through `SN_Hierarchy_Service::assign_parent()`.
- Must not change WordPress roles, legacy roles, user_meta, payments, invoices, wallets, commissions, MIS rows, or After Sales cases.

### `admin_post_sn_hierarchy_consistency_check`
- Permission: `manage_options`.
- Nonce: `sn_hierarchy_consistency_check`.
- Read-only diagnostic for report/hierarchy consistency.
- Stores latest report in `sn_hierarchy_consistency_report`.

### `admin_post_sn_distribution_current_structure_preset`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_distribution_current_structure_preset`.
- Supports dry-run and APPLY-protected apply.
- Updates distribution chain configuration for the current transitional structure only.
- Does not create live leads, assign sellers live, send SMS, or modify `sn_leads`.

## HR Workforce Inline/Bulk Contracts

### `admin_post_sn_hr_inline_profile_update`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_inline_profile_update`.
- Inputs: `user_id`, `position_id`, `manager_user_id`, `level_id`, `employment_status`, `employment_type`, `is_active`, `fixed_salary_enabled`, `base_salary`, `commission_enabled`, `default_commission_rule_id`, `effective_from`, `reason`.
- Updates HR profile fields and HR assignment through safe HR/hierarchy services.
- Compensation changes require `effective_from` and write through effective-dated HR compensation history.
- Must not change WordPress roles, `legacy_role`, legacy user_meta, payments, invoices, wallets, SMS, MIS, or After Sales.

### `admin_post_sn_hr_bulk_workforce_action`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_bulk_workforce_action`.
- Inputs: `user_ids[]`, `bulk_action`, `run_mode`, `confirm_apply`, `reason`, plus action-specific target fields.
- Supported actions: `manager`, `position`, `level`, `employment`, `compensation`.
- `run_mode=dry_run` reports candidates without writing.
- `run_mode=apply` requires `confirm_apply=APPLY`.
- Manager changes require an explicit target manager and preserve existing managers unless overwrite is selected.
- Compensation fields such as `base_salary`, `fixed_salary_enabled`, `commission_enabled`, `default_commission_rule_id`, and `effective_from` are optional for non-compensation bulk actions.
- Bulk `manager`, `position`, `level`, and `employment` actions must not create compensation history and must not require missing compensation fields.
- HR commission-rule dropdowns read `sn_commission_rules` and must render a readable label using title, then rule code, then `قانون #ID`; inactive saved rules remain visible for review.
- Must not change WordPress roles, `legacy_role`, legacy user_meta, payments, invoices, wallets, SMS, MIS, or After Sales.

## Launch readiness diagnostics
- Integration page section «بررسی آماده‌سازی لانچ» is admin-only and read-only.
- It summarizes HR profile/readiness counts, Finance commission rule readiness, MIS staging/distribution counts, and PASS/نیازمند اقدام states.
- It must not write data, post wallets, approve payments, create leads, send SMS, or expose tokens/public invoice URLs.

## Launch Stabilization Admin Contracts

### `admin_post_sn_hr_people_import`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_people_import`.
- Inputs: `hr_csv`, `import_mode=advanced_people` by default from the UI, `run_mode`, `confirm_apply`, optional `allow_create_wp_user`, optional `send_user_notification`.
- Simple accepted headers: `نام`, `نام خانوادگی`, `نام و نام خانوادگی`, `موبایل`, `سمت`, optional `توضیحات`. Advanced accepted manager headers include `مدیر`, `مدیر مستقیم`, `نام مدیر`, `مدیر بالادست`, `موبایل مدیر`, `شماره مدیر`, `شماره موبایل مدیر`, `manager`, `manager_lookup`, `manager_mobile`, `manager_phone`, `parent`, `parent_lookup`, and `parent_mobile`.
- Simple mode requires only name, valid mobile, and mapped position.
- Direct manager, level, salary, commission, compensation history, and effective dates are not required in simple mode. In advanced mode, `موبایل مدیر` takes precedence over manager name/label and manager assignments are applied after all valid profiles are created or updated.
- `run_mode=dry_run` parses and validates without writing.
- `run_mode=apply` requires `confirm_apply=APPLY`.
- Writes HR profile basics in simple mode. In advanced mode, dry-run reports `manager_lookup`, `manager_status`, `manager_resolved`, `manager_in_file`, and `would_assign`; apply writes profile basics first and hierarchy assignments second.
- Invalid rows must not create WordPress users.
- Must not auto-change existing WordPress roles, `legacy_role`, legacy user_meta, payments, invoices, wallets, SMS, MIS staging rows, or After Sales cases.

### `admin_post_sn_hr_simple_add_person`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_simple_add_person`.
- Inputs: `first_name`, `last_name`, `mobile`, `position`, optional `notes`, optional `allow_create_wp_user`.
- Creates or updates the HR profile with position and active status only.
- Leaves direct manager, level, salary, commission, and compensation history empty for later setup in «مدیریت نیروها».
- Does not change existing WordPress roles, `legacy_role`, or legacy user_meta.

### `admin_post_sn_mis_quick_assign_data`
- Permission: `manage_options` or HR position `mis`.
- Nonce: `sn_mis_quick_assign_data`.
- Inputs: `batch_id`, `target_user_id`, `limit`, `run_mode`, `confirm`, `reason`.
- `run_mode=dry_run` reports eligible valid MIS staging rows without writing.
- `run_mode=apply` requires `confirm=APPLY`.
- Creates `sn_distribution_items` directly from valid `sn_mis_data_rows` using `source_type=mis_data_row`.
- Does not require Lead Pool for the launch quick-assignment path.
- Does not insert/update/delete `sn_leads`.
- If target is seller, distribution item is marked `delivered_to_seller`; live lead creation remains a separate controlled seller action.

### `admin_post_sn_mis_import_csv`
- Permission: `manage_options` or HR position `mis`.
- Nonce: `sn_mis_import_csv`.
- Upload field: `mis_csv`. The MIS panel ships with `docs/samples/sn-mis-data-import-sample.xlsx` as an operator template.
- CSV parser reads the first non-empty row as the header, removes BOM/zero-width characters, and supports comma, semicolon, tab, and pipe delimiters.
- If the server cannot read XLSX files, import must stop with the Persian warning `خواندن XLSX روی این سرور فعال نیست. فایل را CSV UTF-8 ذخیره و بارگذاری کنید.` and must not create blank invalid rows.
- Accepted primary phone headers include `شماره اول`, `شماره اصلی`, `موبایل`, `شماره موبایل`, `تلفن همراه`, `شماره تماس`, `شماره`, `phone`, `mobile`, `mobile_1`, and `phone_1`.
- Accepted secondary phone headers include `شماره دوم`, `موبایل دوم`, `شماره جایگزین`, `phone_2`, and `mobile_2`.
- Iranian mobile values are normalized to `09xxxxxxxxx` when they are supplied as `09...`, `9...`, `+98...`, `98...`, `0098...`, Persian/Arabic digits, or values with spaces/dashes/parentheses.
- Invalid rows remain in `sn_mis_data_rows` with `row_status=invalid` and a reason such as `empty_phone`, `too_short`, `too_long`, `not_mobile_prefix`, `no_phone_column`, or `non_numeric_after_normalization`.
- Import reports must show detected headers, normalized headers, detected phone columns, invalid reason counts, and up to 10 masked invalid row samples. Admin reports may also include parser diagnostics such as file metadata, selected delimiter, raw/header previews, and normalized header cells.
- This action must not insert/update/delete `sn_leads`.

### CRM login handlers
- Handlers: `sn_seller_login`, `sn_sales_manager_login`, `sn_supervisor_login`, `sn_financial_login`, `sn_my_panel_login`.
- Accepted identifier: mobile number, username, or email.
- Mobile lookup normalizes Persian/Arabic digits and common separators, then checks `user_login`, user email, HR profile phone fields, and common user meta phone keys.
- If one user matches a mobile, WordPress password verification is used for that user.
- If multiple users match the same mobile, login is rejected with `duplicate_mobile`.
- CRM panel access is based on HR Position with legacy WordPress role/capability fallback; imported Subscriber users may access their CRM panel when their HR profile position allows it.

### `admin_post_sn_hr_set_user_password`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_set_user_password`.
- Inputs: `user_id`, `new_password`, `new_password_repeat`, optional `generate_random`, optional `send_sms`.
- Minimum password length: 8.
- Sets a new WordPress password for the selected user.
- If `send_sms` is selected, sends a one-time login message through existing `SN_SMS::send()`.
- Plain passwords must not be stored in options, HR logs, debug reports, or audit rows.
- Reports may store user ID, actor ID, masked phone, password reset yes/no, SMS sent yes/no, and safe error/status codes only.

### `admin_post_sn_hr_send_login_credentials_sms`
- Permission: `manage_options` or HR position `hr`.
- Nonce: `sn_hr_send_login_credentials_sms`.
- Inputs: `user_ids[]`, `reset_password`, `sms_template`, `confirm_apply`.
- Apply requires `confirm_apply=APPLY`.
- Existing passwords are never read or sent.
- The action generates a temporary password, sets it for the selected user, sends SMS through existing `SN_SMS::send()`, and must not store/log the plain password.
- Reports/audit may store user ID, masked phone, send status, actor, and time only.
- Must not change WordPress roles, `legacy_role`, legacy user_meta, payments, invoices, wallets, commissions, MIS rows, or After Sales cases.

## Launch Finance Commission UX Contract
- Commission rule forms should use operator-friendly dropdowns for HR position, HR level, payment method, invoice status, employment status, and active state.
- Stored values remain stable internal slugs/keys.
- Commission diagnostics are read-only and may show active rules, unknown HR/payment/status values, zero-match rules, and HR compensation-history readiness.
- Commission dry-run remains calculation/readiness only unless an already-approved finance action explicitly posts to wallet behind its existing guard.

## Seller stage-one AJAX contracts (2.0.13)

### `wp_ajax_sn_save_customer_info`

- Existing login, `sn_public` nonce and source-ownership checks remain mandatory.
- New input: `not_purchase_reason`.
- Only the four stage-one seller statuses are accepted by the shared flow module.
- `عدم خرید` requires a non-empty sanitized reason.
- `پیش‌فاکتور` cannot be set directly by this endpoint; successful invoice creation owns that transition.
- Source update and shared flow state/event update execute in the same database transaction.
- Archived sources reject seller edits.

### `wp_ajax_sn_seller_flow_no_answer_attempt`

- Requires login and `sn_public` nonce.
- Inputs: `source_kind`, `source_id`, optional `distribution_item_id`, required client `attempt_token` for safe retries.
- Server resolves current source ownership; submitted seller or manager ids are not trusted.
- Increments only the shared no-answer attempt state, never beyond three.
- Reusing the same request token returns the prior state without another increment.
- Updates the compatible source status and flow event in one transaction.

### `wp_ajax_sn_sales_manager_lazy_tab` with `tab=manager-archives`

- Uses the existing sales-manager panel gate and `sn_public` nonce.
- A normal sales manager sees only archive rows whose resolved `sales_manager_user_id` equals the current user.
- `manage_options` may view all archive rows for diagnostics.
- This tab is read-only in stage one.
