# CURRENT_TASKS

Use this file as the lightweight handoff for the active Codex task.

## Active task
- Name: 2.0.58 Customer profile accordions and WooCommerce card content
- Phase: Local verification complete; Staging acceptance pending
- Owner: Codex / project maintainer
- Date: 2026-09-15

## Goal
- Read each card image directly from its linked WooCommerce product and render it at a fixed 3:2 ratio.
- Display the linked product's main WooCommerce description inside the card modal.
- Convert account, cards, purchases and payments into titled, single-open profile accordions.

## Current implementation
- OTP bootstrap: `sales-network.php`
- Customer authentication: `includes/class-sn-customer-portal.php`
- Static audit tool: `tools/static-audit.js`
- Release notes and acceptance plan: `docs/RELEASE_2.0.58_CUSTOMER_PROFILE_ACCORDIONS.md`

## Acceptance criteria
- A card uses the featured image of its own `content_product_id`, not the subscription image.
- Card and modal images use a landscape 3:2 crop without distortion.
- Modal description comes from the main product description and is safely rendered.
- Only one top-level profile accordion remains open; the Cards accordion is open initially.
- Header links open and scroll to their target accordion.

## Verification steps
- Run `node tools/static-audit.js --json` and require `ok: true`.
- Run `docs/QA_CHECKLIST.md`, including the 2.0.13 seller stage-one section.
- Run PHP lint in an environment with PHP 8+, and syntax-check every JavaScript file with Node.
- Test products with image, without image, with rich description and without description.
- Install the standard WordPress ZIP on Staging and run the 2.0.58 acceptance plan before Production.

## Notes / risks
- Local static checks cannot replace a real WordPress/WooCommerce/PHP runtime test.
- The local build environment does not provide PHP CLI, WordPress or MySQL; runtime acceptance is therefore still open.
- Import is an upsert. Blank manager/position cells do not delete existing relationships automatically.
- Historical notes below remain as an audit trail; they are not the active task.

## 2026-06-09 — Supervisor/Senior Supervisor distribution UI consolidation

- Unified the v4 `sn_distribution_items` transfer workflow into the old-style Supervisor allocation tab.
- Removed the separate supervisor v4 distribution widget from above the legacy tabs.
- The Supervisor `تخصیص شماره` tab now uses `current_owner_user_id = current user` and submits `distribution_item_ids[]` to `sn_distribution_transfer_items`.
- Senior Supervisor distribution tab label changed to `تخصیص شماره` and uses the same embedded v4 allocation component.
- Existing live lead creation remains controlled and separate; distribution transfer does not create `sn_leads`.

## 2026-06-09 — Supervisor HR Seller Visibility Runtime Fix
- پنل سرپرست در تب فروشندگان دیگر فقط به WP role `sn_seller` یا user_meta قدیمی `sn_supervisor_id` وابسته نیست.
- فروشنده‌های مستقیم HR assignment با سمت `seller` برای سرپرست جاری به عنوان منبع اصلی V4 خوانده می‌شوند.
- کاربران HR که با نقش WordPress عادی مثل Subscriber ساخته شده‌اند، اگر HR position آن‌ها `seller` باشد در پنل سرپرست نمایش داده می‌شوند.
- debug پاسخ `sn_supervisor_data` شامل `hr_direct_seller_ids`, `legacy_seller_ids`, `final_seller_ids` شد.

## 2026-06-09 - Supervisor Allocation UX restored on V4 backend

- Unified the Supervisor allocation tab so the old, cleaner allocation UX is backed by V4 `sn_distribution_items`.
- Recipient dropdown now uses the same HR direct-seller source as the KPI/list logic and does not depend on owned items.
- Added allocation modes:
  - selected item transfer via `distribution_item_ids[]`
  - assign N numbers to one seller
  - round-robin distribution across active HR sellers
- No live `sn_leads` are created during distribution transfer; seller conversion remains behind the controlled conversion action.

## 2026-06-09 - Seller Panel V4 Delivered Data Visibility

- Seller panel now reads V4 delivered data from `sn_distribution_items` using `current_owner_user_id = current user` or `final_seller_user_id = current user`.
- Supported visible statuses include `assigned_to_role`, `delivered_to_role`, `distributed_forward`, `delivered_to_seller`, and `ready_for_lead_conversion` without requiring a live `sn_lead`.
- Added seller-facing section «دیتای تحویل‌شده به شما» with masked phone, batch code, customer/province/city, status, follow-up, inline edit, and controlled conversion button.
- Added safe seller operational fields on distribution items for contact status, sale probability, notes, next follow-up, and customer/province/city overrides.
- Controlled conversion to `sn_leads` remains explicit and idempotent; viewing/editing delivered data does not create a live lead.

## 2026-06-09 - Seller Inline Save Stabilization
- Fixed seller old-panel inline save flow for V4 distribution items.
- Save no longer aborts in-flight requests; later field changes are queued and persisted after the current request.
- Contact-status changes are saved explicitly on change/focusout and do not require closing/reopening the editor.
- V4 inline save now ensures distribution schema and writes seller_updated_by/seller_updated_at.
- No automatic sn_leads creation was added.

## 2026-06-09 - Seller customer actions, recontact payment reopen, paid-product referral
- Fixed seller customer behavior AJAX access to accept HR position `seller`, not only legacy WP role `sn_seller`.
- Recontact-requested invoices remain payable: online payment, receipt upload, and manual card payment now accept `recontact_requested` status.
- Public invoice UI now shows a recontact notice while keeping payment/upload controls visible.
- Added WooCommerce product-level setting for post-payment referral after finance approval:
  - enabled flag
  - target: self / supervisor / senior supervisor
  - optional suggested follow-up product ID
  - note
- On invoice paid/finance-approved, enabled products create one idempotent V4 distribution item with `source_type=paid_invoice_referral` and do not create `sn_leads` automatically.


## 2026-06-09 - Paid invoice manual repeat action referral
- Added product setting for manual repeat action after finance approval.
- Manual mode creates a `paid_invoice_referral` item with `needs_repeat_action` for seller decision.
- Seller/Supervisor/Senior Supervisor panels show `نیاز به اقدام مجدد` and can issue follow-up invoices while preserving the original seller for commission.
- Supervisor-issued follow-up invoices log `issued_by_user_id` in activity metadata and keep `seller_id` as the original seller.

## 2026-06-09 - Commission rule matching resolver
- Added commission context resolver for invoice/seller/payment/HR history.
- Commission matching now uses seller_id as commission owner and treats issued_by_user_id as audit only.
- Added employment_type and normalized payment method matching for rules: training/contractor × online/card_to_card.
- Finance dry-run and Seller panel commission preview share the same resolver and matcher.
- No wallet posting or payment callback behavior changed.


## Commission cumulative matching update
- Seller panel standalone `پورسانت من` tab removed; seller wallet remains the user-facing commission area.
- Commission dry-run now sums all active matching commission rules for an invoice instead of selecting only one best rule.
- Dry-run notes include `matched_rule_ids`, `commission_components`, and `stack_mode=cumulative_all_matching_rules`.
- Wallet posting remains controlled and uses the aggregated dry-run amount; no automatic posting was added.
Wallet finance-rule integration QA:
- Wallet recalculation now uses commission engine finance rules and cumulative matched rules.
- Seller wallet transaction rows display finance rule ids from transaction meta.
- Auto credit still respects sn_wallet_auto_credit; admin recalculate forces controlled recalculation.

## Wallet Finance Commission Visibility Fix - 2026-06-09
- Wallet finance preview no longer relies only on `seller_id` + a narrow SQL paid-status condition.
- Added robust invoice owner resolution for seller/original/commission/seller_user columns.
- Added broader approved/paid detection across status/date columns and PHP-level eligibility filtering.
- Wallet preview and recalculation now use the same eligible invoice row helper before resolving Finance commission rules.
- No auto wallet posting was added; recalculation remains controlled by Finance/Wallet action.

- 2026-06-10: Sales deputy panel audit: removed operational admin guide text, enforced sidebar tabs with stable panel id, fixed summary helper to sales-deputy scope, embedded distribution in clean mode, verified syntax/assets.

## 2026-06-10 - Sales Deputy panel scope cleanup
- Removed operational data distribution from the Sales Deputy panel. MIS hands data to Sales Managers; Sales Managers distribute within their branch.
- Added Sales Deputy visibility tabs for supervisors and sellers so the deputy can monitor the full HR sales structure without becoming part of the distribution workflow.
- Sales Manager distribution recipient options now include all active descendant senior supervisors, supervisors, and sellers in the manager HR scope.


## Supervisor panel functional audit - 2026-06-10
- Cleaned supervisor operational copy and legacy labels.
- Supervisor KPI cards now separate V4 delivered items from legacy live leads.
- Direct HR sellers are the source for seller count and dropdowns; WP role is not required.
- Fixed supervisor invoices tab target id mismatch.
- V4 distribution reports/debug are hidden in embedded operational panels unless explicitly enabled.

## 2026-06-10 — Remove APPLY text gate from operational distribution panels
- Removed manual `APPLY` text inputs from operational distribution/allocation forms used by sales manager, senior supervisor, supervisor, and related front-office data transfer sections.
- Distribution transfer remains protected by nonce, login, ownership validation, and HR recipient scope validation.
- Dangerous admin/finance/HR tools and controlled live-lead creation keep explicit confirmation gates.


## 2026-06-12 — HR advanced people import + Excel templates
- HR people import now defaults to advanced mode from the UI while preserving simple mode as an explicit option.
- Manager headers were expanded for Persian and English aliases; `موبایل مدیر` / `manager_mobile` take precedence over manager names.
- Advanced dry-run reports manager lookup/resolution status and would-assign counts.
- Apply now saves all valid users/profiles first, then applies direct-manager hierarchy assignments in a second pass.
- Added packaged XLSX templates under `docs/samples/` for HR advanced import, HR simple import, and MIS data import.
- CSS/JS panel assets were intentionally left unchanged.


## 1.0.26 XLSX sparse-cell fix
- HR/MIS XLSX parser now preserves empty cells based on Excel cell references, preventing mid-row blank cells from shifting columns during import dry-run/apply.
- No CSS or JS assets changed.


## 2026-06-12 - HR bulk password tools
- Added a no-SMS password reset utility for a single HR user and bulk password reset for selected/all HR profiles.
- Bulk password reset requires APPLY confirmation and never stores the raw password in reports or logs.
- Current actor is skipped for safety if included in the target scope.


## v1.0.33 - Zibal Gateway
- Added selectable online payment gateway: ZarinPal or Zibal.
- Added Zibal merchant and test-mode settings.
- Zibal uses `/v1/request`, `/start/{trackId}`, and `/v1/verify`; invoice amounts remain stored in Toman and are sent to gateways as Rial.
- Callback route remains the existing `sn_callback=1&invoice_id=...` path and now also accepts `sn_gateway=zibal`.
- Normal online payment completion still marks invoices paid and triggers `sn_invoice_paid`.
