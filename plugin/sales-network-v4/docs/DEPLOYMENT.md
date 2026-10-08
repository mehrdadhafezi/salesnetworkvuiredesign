# DEPLOYMENT

## Environments
- Develop on local/staging first.
- Do not enable new enforcement modes on production until staging tests pass.
- Production-like staging should contain imported copy of real legacy data before major migrations.

## Pre-deploy backup
Before installing any new ZIP on staging/production:
- Full database backup.
- Plugin directory backup.
- Export current plugin ZIP if possible.
- Record current plugin version and active options:
  - DB version option.
  - scope enforcement mode.
  - latest migration reports.

## Install/update steps
1. Upload plugin ZIP.
2. Activate/update plugin.
3. Open WordPress admin as `manage_options` user.
4. Visit Integration/diagnostics page.
5. Check migration reports and table statuses.
6. Run explicit repair buttons only if needed:
   - HR defaults repair.
   - invoice token backfill.
   - MIS table repair.
7. Keep scope mode `off` unless testing a specific mode.

## CRM portal page setup
After installing a build that includes the portal registry:
1. Open شبکه فروش → یکپارچه‌سازی.
2. In «راه‌اندازی صفحات پورتال CRM», run Dry-run first.
3. Review missing/existing/conflict statuses.
4. Run Apply only with confirmation text `APPLY`.
5. Verify `sn_portal_page_ids` points to the created or existing pages.
6. If a slug conflict is reported, inspect that page manually; the setup tool does not overwrite user-created content.

Expected portal shortcodes:
- `[sn_my_panel]` unified dashboard entry.
- `[sn_portal_nav]` internal navigation.
- `[sn_seller_panel]`, `[sn_supervisor_panel]`, `[sn_senior_supervisor_panel]`, `[sn_sales_manager_panel]`, `[sn_sales_deputy_panel]`, `[sn_financial_panel]`, `[sn_hr_panel]`, `[sn_mis_panel]`, `[sn_after_sales_panel]`, `[sn_reports_panel]`, `[sn_customer_profile]`, `[sn_invoice_page]`.

Portal navigation must not expose public invoice URLs or access tokens.

## Fresh-site CRM launch checklist
For a clean CRM setup, use this order before operator testing:
1. Open Integration and confirm Launch readiness cards are visible.
2. Run portal page setup dry-run, then apply with `APPLY` only if missing pages are expected.
3. In HR panel, use «ورود ساده نیروها» dry-run with a simple people file: نام، نام خانوادگی، موبایل، سمت، توضیحات.
4. Apply HR people import only with `APPLY`; level, direct manager, salary, commission, and effective date are not required during simple import.
5. Use «مدیریت نیروها» and «تغییرات گروهی» to set direct managers, levels, employment status, salary, and commission readiness after people exist.
6. Confirm WordPress roles, `legacy_role`, and legacy user_meta were not auto-changed.
7. Confirm HR hierarchy assignments exist for sales deputy, sales manager, senior supervisor, supervisor if available, and sellers.
8. In MIS panel, create a data batch and import Excel/CSV into staging.
9. Assign valid MIS rows to a sales manager or use «تخصیص سریع دیتا» only for internal launch testing.
10. Confirm quick assignment creates distribution items only and does not create live `sn_leads`.
11. Move distribution items down the hierarchy until seller delivery.
12. Seller opens «داده‌های تحویل‌شده» and only then uses the controlled action to create a live sales lead if approved for the test.
13. Seller verifies «مشاهده / ویرایش» on assigned leads and creates a pre-invoice from the existing seller flow.
14. Finance reviews payment queue and commission dry-run, but wallet/commission posting flags remain off unless separately approved.
15. In Finance, create or review commission rules with the Persian dropdown form; if dry-run has zero items, check the diagnostics card for missing HR level/history or unmatched rule values.
16. In MIS, review «بررسی آمادگی پخش دیتا» before pushing launch data down the hierarchy.
17. In HR, use «ارسال اطلاعات ورود کاربران» only after selecting users and confirming `APPLY`; this resets temporary passwords and sends SMS without storing the plain password.
18. For one-off testing, HR/Admin may use «تنظیم رمز یک نیرو» to reset one user and optionally send the login SMS. Plain passwords must not appear in saved reports.
19. Test CRM login with mobile number, username, and email. If duplicate mobile is reported, correct the mobile in HR management before launch.
20. In Integration, review «بررسی آماده‌سازی لانچ» for HR/Finance/MIS/mobile-login PASS/نیازمند اقدام summaries before runtime testing.

## Integration admin page
The Integration page is the main diagnostics/setup hub. After deploy:
- Check the top summary cards first: DB version, HR profiles, MIS, After Sales, Commission, HR compensation, portal pages, scope mode, and invoice token coverage.
- Keep scope mode `off` or `audit_only` on production until scope enforcement is separately approved.
- Use collapsed technical sections only when debugging: migration JSON, sync JSON, scope decisions, HR profile list, hierarchy tree, invoice token samples, skipped users, and raw reports.
- Treat «ابزارهای اصلاح داده» as sensitive: run dry-run first and use `APPLY` only after reviewing the report.

## Post-deploy health checks
- PHP fatal-free admin load.
- Public frontend pages load.
- `[sn_my_panel]` guest shows plugin login form.
- Seller/senior supervisor/manager/deputy/HR/MIS panels route correctly.
- Public tokenized invoice link works.
- Public invoice without token rejects.
- Admin/internal invoice upload/payment works.
- Finance approve/reject still works with granular caps.

## Migration safety
- Migrations must be safe, idempotent, and non-destructive.
- Do not rely only on DB version; verify table/column existence for new modules.
- Large backfills must be batch-limited and report progress.
- If a repair fails, show exact DB error in admin report.

## Rollback process
- Set `sn_scope_enforcement_mode` to `off` first.
- Reinstall previous known-good plugin ZIP.
- Restore database backup only if data/schema changes must be undone.
- Do not manually delete HR/MIS tables unless explicitly approved.

## Server-specific notes
- PHP CLI may warn about missing XML extensions in local lint; verify web runtime separately.
- MySQL/MariaDB may treat some identifiers as reserved; quote risky column names such as `row_number` or use safer future names.
- `dbDelta` can fail silently; verify table existence after calling it.

## Needs confirmation
- Exact production deployment process, hosting constraints, and backup tooling are not available in this workspace.
- No server environment variables were identified in available files.
