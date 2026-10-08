# PROJECT_CONTEXT

## What this project is
- WordPress plugin for a sales network / CRM (`Sales Network`, `شبکه فروش`).
- Originally a fixed role sales plugin: seller → supervisor → sales manager → finance.
- Current target is a configurable CRM with HR structure, data intake, scope visibility, financial workflows, wallet/commission, after-sales, and MIS.

## Product goal
- Preserve existing sales, invoice, payment, wallet, and commission data.
- Add a dynamic organization model:
  - WordPress Role / legacy role: compatibility only.
  - HR Position: real organizational position.
  - HR Level: employment/commission/salary metadata.
- Route panels and data visibility by HR Position and hierarchy, not only WordPress role.

## Current main users / positions
- `seller` — فروشنده
- `supervisor` — سرپرست, planned for new hires; currently 0 in test data.
- `senior_supervisor` — سرپرست ارشد; legacy supervisors were converted here.
- `sales_manager` — مدیر فروش
- `sales_deputy` — معاون فروش
- `finance` — مالی
- `hr` — منابع انسانی
- `mis` — ورود و پخش دیتا
- `after_sales` — خدمات پس از فروش
- `operations_sales_manager` — مدیر فروش عملیات
- `operations_sales_supervisor` — سرپرست فروش عملیات
- `operations_sales_expert` — کارشناس فروش عملیات

## Current panel state
- Existing and active panels: seller, supervisor, senior supervisor, sales manager, sales deputy, finance, HR, MIS and after-sales.
- Added/updated panels:
  - `[sn_my_panel]` unified router.
  - `[sn_operations_sales_manager_panel]`, `[sn_operations_sales_supervisor_panel]` and `[sn_operations_sales_expert_panel]` operations dashboards.
  - Legacy project shortcodes remain compatibility aliases and open the corresponding operations dashboard.
  - Per-card routing from the originating Sales Manager branch directly to a configured Operations Sales Supervisor under the selected Operations Sales Manager.
  - Subscription/content operations tracking, file chat and unread notifications.
  - Biavin ledger rendered inside «کیف پول من».
  - Shared HR user transfer module in WordPress admin and the HR panel.
- Current release target: 2.0.58. It adds titled profile accordions, 3:2 WooCommerce product images and main product descriptions inside card modals while preserving the 2.0.57 payment and repurchase-awareness behavior.

## Stack
- WordPress plugin, PHP 8+, MySQL/MariaDB via `$wpdb` and `dbDelta`.
- WooCommerce product/order integration exists in invoice flow.
- SMS provider integration exists; Faraz SMS guide exists in older package.
- ZarinPal payment request/verify/callback exists.
- Frontend assets in `assets/js` and `assets/css`.

## Production-sensitive notes
- SMS provider logic must not be changed unless explicitly requested.
- ZarinPal/payment request, verify, callback must not be changed unless explicitly requested.
- Wallet and legacy commission logic must not be changed unless explicitly requested.
- Public invoice access is tokenized; do not weaken `access_token` validation.
- Admin/internal invoice actions must use logged-in capability/nonce checks, not public token checks.
- MIS imports must stay in MIS staging tables until a later approved conversion/distribution phase.
- Do not run destructive DB changes on production.

## Development handoff
- Treat `class-sn-plugin.php` as a legacy compatibility core; add isolated modules for new bounded features when possible.
- Use `docs/CURRENT_TASKS.md` for the active task and `docs/RELEASE_2.0.58_CUSTOMER_PROFILE_ACCORDIONS.md` for the current release contract. Earlier release contracts remain applicable.
- Exact runtime behavior must be verified on Staging before Production deployment.
