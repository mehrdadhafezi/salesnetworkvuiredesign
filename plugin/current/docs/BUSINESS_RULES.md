# BUSINESS_RULES

## Role model
- WordPress Role / legacy role is compatibility data.
- HR Position is the real organizational position for routing and hierarchy.
- HR Level is metadata for future salary/commission/training; it is not the commission engine yet.

## Target hierarchy
Final target:
- sales_deputy → sales_manager → senior_supervisor → supervisor → seller

Current transitional state in test/staging reports:
- sales_deputy → sales_manager → senior_supervisor → seller
- sales_manager may still have direct sellers.
- `supervisor` position exists but may have 0 users until new hires are defined.

## Current verified counts from staging reports
- seller: 123
- senior_supervisor: 8
- sales_manager: 1
- sales_deputy: 1
- hr: 1
- MIS may be 1 after test user setup.

## Panel routing rules
- If HR profile exists, route by HR position.
- If HR profile is missing, use legacy role/WP role fallback.
- `senior_supervisor` has its own panel and no longer reuses supervisor panel.
- `sales_deputy`, `senior_supervisor`, `hr`, and `mis` panels are currently base/read-only or staging-focused.
- A panel resolved by `[sn_my_panel]` should accept the matching HR position in its own access check, without changing WordPress roles.

## Scope rules
- Seller sees self only.
- Supervisor/senior_supervisor sees descendant sellers in HR hierarchy.
- Sales manager sees descendant sellers under that sales manager.
- Sales deputy sees descendant sellers under all sales managers below them.
- Admins with `manage_options` are not restricted by scope.
- Scope enforcement must remain feature-flagged.

## Invoice/payment flow constraints
- Seller creates pre-invoice.
- New invoice codes are six numeric digits. Existing invoice codes, including legacy `INV-*` codes, remain valid and must not be rewritten.
- Customer receives tokenized public invoice link through SMS.
- Public/customer invoice actions require `access_token`.
- Internal/admin invoice actions must use login + nonce + capability, not public token.
- ZarinPal callback must not require public access token.
- Seller panel must not expose public invoice URL or access token.

## Finance rules
- Finance permissions are granular user-level caps:
  - `sn_view_finance`
  - `sn_approve_payment`
  - `sn_reject_payment`
- View, approve, and reject must be independently enforced.
- Seller/supervisor/sales-manager must not receive finance caps automatically.

## Commission rules
- Legacy wallet/commission behavior remains active until an explicit apply/posting phase replaces it.
- Phase 4B commission engine is dry-run only:
  - Rules can be created/activated/deactivated.
  - Dry-runs can calculate preview values for eligible invoices.
  - Dry-runs may compare with legacy wallet transaction amounts if available.
  - Dry-runs must not create wallet transactions or change wallet balances.
- HR defines compensation readiness:
  - HR may set base salary, salary currency, commission eligibility, default commission rule, effective dates, and notes.
  - HR may maintain effective-dated compensation history rows. A new period can close the previous open period to the day before the new `effective_from`.
  - HR assignment of a default commission rule is policy/readiness metadata only.
  - Finance remains responsible for calculation, dry-run review, wallet posting, and settlement.
- Finance commission dry-runs should use the historical HR compensation row for the invoice calculation date when one exists. If no historical row exists, dry-run may fall back to current HR profile/readiness data and show a warning.
- Finance commission approval:
  - Generated dry-runs can be approved or rejected by Finance/admin.
  - Approval locks the review state and logs the actor/note.
  - Approval does not create wallet transactions.
- Finance commission wallet posting:
  - Disabled by default behind `sn_enable_commission_wallet_posting`.
  - Requires an approved run and explicit `APPLY`.
  - Posts only eligible unposted dry-run items to seller wallets with the new `commission_engine` transaction type.
  - Must be idempotent and must not call legacy invoice commission auto-credit/recalculation helpers.
- Settlement readiness:
  - Finance may view positive wallet balances and posted commission readiness.
  - Phase 4E readiness does not debit wallets, create bank transfers, or mark payouts complete.
- Finance portal UX:
  - Finance workflow should clearly separate read-only readiness, dry-run calculation, approval, wallet posting, and settlement readiness.
  - Posting must remain manual, feature-flagged, approved-run-only, and confirmation-gated.
  - Regression readiness notes are informational and must not run heavy queries or change data.
- When multiple active rules match, the highest priority rule wins.
- Rejected/returned invoices are excluded unless a rule or dry-run filter explicitly includes those statuses.

## HR rules
- Changing HR position/level/manager requires permission, nonce, reason, and should log.
- Manual correction must not change WordPress role, `legacy_role`, or legacy user_meta automatically.
- Assignment changes must prevent cycles and write history.
- Sync must not silently overwrite manual HR corrections.
- Legacy seller hierarchy may be restored from existing lead ownership data:
  - If `sn_leads.seller_id = seller user id` and historical `sn_leads.supervisor_id = X`, user `X` may become that seller's HR direct manager.
  - Legacy supervisors that were converted to HR position `senior_supervisor` may still directly own sellers in the transitional structure.
  - This restore must use dry-run/apply, preserve existing managers by default, and never change WordPress roles, `legacy_role`, legacy user_meta, HR position, or HR level.
  - The restore tool only assigns seller → inferred manager; it must not decide or change which sales_manager profile is valid.
- Employee metadata such as employee code, employment status/type, dates, salary, department/unit/team, national ID, and HR phone fields belongs to HR profiles, not WordPress roles.
- Department, unit, and team rows are additive/configuration data. They can be activated/deactivated but should not be deleted in the base HR panel.
- Fresh launch HR import must support a simple people file with only name, mobile, and HR position.
- Simple HR import must not require HR level, direct manager, salary, commission, or effective dates.
- Simple HR import apply may create/update HR profiles after validation and may create a basic WordPress user only when explicitly enabled.
- Invalid simple-import rows must not create WordPress users.
- Sensitive HR fields in logs and panel summaries must be masked for display; stored DB values are preserved unless a future privacy task changes storage policy.
- HR workforce inline edits may update HR profile metadata, HR position/level, active status, direct manager assignment, and HR compensation policy, but must not touch WordPress roles, `legacy_role`, or legacy user_meta.
- HR manager assignment candidates follow the sales hierarchy:
  - seller → supervisor, senior_supervisor, sales_manager
  - supervisor → senior_supervisor, sales_manager
  - senior_supervisor → sales_manager, sales_deputy
  - sales_manager → sales_deputy
  - finance/hr/mis/after_sales are not sales hierarchy manager candidates.
- HR manager assignment must prevent self-manager relationships, inactive managers, invalid parent positions, and cycles.
- HR compensation changes from workforce UI require an effective date. Same-date unlocked history rows may be updated; later effective dates close the previous open row to the day before and create a new row. Overlapping rows and locked rows are rejected.
- HR bulk workforce changes are dry-run first and require explicit `APPLY` before writing.

## MIS rules
- MIS file import is temporary MIS staging only.
- MIS import must not insert/update/delete `sn_leads`.
- Assigning a MIS batch to sales manager must not create seller leads yet.
- Conversion reports may read `sn_leads`/`sn_invoices` but must be read-only.
- MIS operator workflow is hierarchical:
  1. MIS creates an input data case/batch.
  2. MIS imports and validates the data file into temporary staging rows.
  3. MIS assigns valid rows to a sales manager.
  4. MIS prepares a ready distribution queue.
  5. Sales manager distributes assigned rows to senior supervisors.
  6. Senior supervisor distributes to supervisors.
  7. Supervisor distributes to sellers.
  8. Only after a row reaches a seller should it become a seller lead.
- Current transitional hierarchy may skip missing layers to the next valid lower layer, for example senior supervisor directly to sellers when no supervisor users exist, or direct sellers under a sales manager.
- Existing controlled direct plan/live-lead tooling is a special audited path and must be labeled as controlled/technical until the full hierarchy distribution UX is completed.
- `assigned_rows=0` usually means no valid unassigned staging rows exist in the selected data case.
- `eligible=0` for the ready distribution queue usually means there are no valid rows assigned to a sales manager.
- Raw MIS technical JSON should be collapsed and masked; operators should see Persian summaries first.
- The standard MIS distribution path is a configurable hierarchy engine, not direct MIS-to-seller assignment:
  1. MIS prepares the Lead Pool from valid manager-assigned staging rows.
  2. MIS delivers ready pool rows to the active distribution chain starting owner, usually sales manager.
  3. Current owners distribute only owned rows to allowed next-role users under their HR subtree.
  4. Each chain step defines from position, to position, skip-if-missing, final-delivery, and live-lead eligibility.
  5. If a layer is missing, skip is allowed only when the active step allows it and the recipient remains inside the actor subtree.
  6. Final seller selection happens progressively through the sales hierarchy.
  7. Live `sn_leads` creation must not happen before final seller delivery. If mapping is risky, keep the item as `delivered_to_seller` and run live lead creation only through a separate controlled action.
- MIS can inspect delivery/report status and hand off ready rows to the chain, but MIS does not pick final sellers unless that user also has an authorized role in the active sales chain.
- A delivered distribution item may become a live seller lead only through the controlled delivered-item conversion action.
- That conversion may insert a new `sn_leads` row, but must not update/delete existing leads, send SMS, create invoices, touch payments, wallets, commissions, HR, After Sales, or public invoice token logic.

## Customer profile / reporting / exports
- Customer 360 is an internal read-only context view. It may aggregate lead, invoice, payment, activity, and after-sales case data, but it must not create or update operational records.
- Internal invoice/customer context chips must use invoice codes or internal query context only; they must not expose public invoice URLs or `access_token`.
- Seller-level users can only view/export their own customer/sales rows.
- Sales hierarchy users can only view/export rows within their visible seller scope.
- Finance/admin may view financial report context according to existing finance permissions.
- CSV exports must be formula-safe and must mask or omit sensitive fields unless an existing role policy explicitly permits them.
- Tokens, passwords, hashes, gateway raw payloads, and public customer payment links must never be exported.

## UI state
- Current panels are functional/base panels, not final UX.
- English field keys, raw IDs, and admin/debug-style tables are acceptable during test only.
- A later UI/UX polish phase must Persianize labels and clean layout.

## Needs confirmation
- Exact definitions of active/inactive sellers and financial status groups should be confirmed before new reporting/commission work.
## Portal Page And Navigation Rules

- CRM portal pages are internal routing helpers and do not change business ownership, HR hierarchy, scope enforcement, payment, invoice, MIS, or After Sales behavior.
- Admins may create missing CRM portal pages from Integration → «راه‌اندازی صفحات پورتال CRM».
- The page setup tool must not delete pages and must not overwrite an existing page when the slug exists but does not contain the expected shortcode.
- `[sn_portal_nav]` shows links according to current access:
  - `manage_options` sees all internal portal links.
  - sellers see seller/dashboard links.
  - supervisors, senior supervisors, sales managers, and sales deputies see their matching dedicated portal links through HR position access.
  - finance users see finance links through existing finance access rules.
  - HR, MIS, and After Sales users see their matching panels through HR position access.
  - reports and customer profile links use their existing read permissions.
- Portal navigation and internal context links must not expose public invoice URLs or `access_token`.
- `[sn_my_panel]` remains the preferred unified entry point and must continue showing the plugin login form for guests.
- Normal CRM dashboard users should see a single clear route to their own panel. Technical route details and generic extra links are admin-only.
- Seller lead/customer autosave is limited to existing lead profile fields and must not create leads or invoices.
- Finance rejection reasons must be visible to finance and seller users in their internal panels without exposing public invoice links.
- A rejected invoice may be resent by the seller for finance review with a note, optional corrected transfer fields, and optional new receipt upload through the internal seller flow; this does not approve payment, create a new invoice, require public token, or trigger payment callbacks.
- Payment review labels should distinguish receipt review from manual transfer-information review when display context is available.

## Launch Follow-up: Credentials And Commission Readiness

- HR/Admin credential SMS is a reset-and-send flow only; existing passwords are never read.
- Plain generated passwords may exist only in memory for the immediate SMS body and must not be stored in options, HR logs, debug reports, or audit rows.
- Credential SMS audit may store user ID, masked phone, send status, actor, and time.
- CRM portal access for launch users is based on HR Position, not privileged WordPress roles. WordPress Subscriber is acceptable for imported CRM users when the HR profile position is valid.
- CRM login may use mobile number, username, or email. Duplicate mobile numbers are a data-quality blocker and must be corrected by HR/Admin instead of guessing the user.
- Finance commission rule setup is HR-aware: position, level, payment method, invoice status, employment status, and active state should be selected from operator-friendly dropdowns.
- HR compensation rule selection must show Finance-defined commission rules with non-empty readable labels. If a rule has no title, display its rule code; if that is also missing, display `قانون #ID`.
- HR bulk workforce actions must treat compensation fields as optional unless the requested operation is compensation. Manager, position, level, and employment bulk changes must not create compensation history.
- Commission diagnostics are readiness-only and must not post wallets, approve runs, alter invoices/payments, or send SMS.
- MIS launch self-check is read-only and guides operators through staging, manager assignment, Lead Pool, distribution, seller delivery, and finance readiness.
- MIS quick assignment for launch testing may create distribution items directly from valid MIS staging rows. It must not create live `sn_leads`; seller live-lead creation remains a separate controlled action after delivery.
- MIS import must accept common Iranian mobile formats and Persian phone headers. Invalid rows must remain stored with explicit reasons so operators can fix the source file before assignment.
- Integration «بررسی آماده‌سازی لانچ» is read-only and only summarizes HR/Finance/MIS readiness; it must not write operational data.
