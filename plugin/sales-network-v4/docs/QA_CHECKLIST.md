# QA_CHECKLIST

## Static checks
Run after every code change:

```bash
node tools/static-audit.js --json
php -l sales-network.php
find includes -name '*.php' -print0 | xargs -0 -n1 php -l
find assets/js -name '*.js' -print0 | xargs -0 -n1 node --check
```

Expected PHP CLI warning seen in local reports:
- missing `xml`, `xmlreader`, `xmlwriter` extensions; not a syntax error.

## 2.0.16 Campaign/UTM merge and atomic pre-invoice

- Campaign query-string and legacy fragment UTM values reach the Marketing Dot submission snapshot.
- A valid Marketing Dot submit is stored before payment and repeated submits remain independent rows.
- Campaign report Jalali dates resolve to Gregorian SQL boundaries.
- Page exit without a lead is reported separately from funnel abandonment.
- Campaign report help buttons expose escaped Persian tooltips for every metric column.
- Seller stage-one, HR transfer and Campaign/UTM modules are all loaded by the selective bootstrap.
- Legacy and V4 source rows are locked before seller-flow state during source-linked invoice creation.
- Invoice, payment stage, items, source status, Dot link and seller-flow event roll back together on any write failure.
- A concurrent source reassignment returns a conflict and creates no invoice.

## 2.0.13 seller stage-one

- Seller status endpoint returns exactly: `جواب نداده`, `تماس مجدد`, `عدم خرید`, `پیش‌فاکتور`.
- Legacy and MIS/V4 rows both use the shared seller-flow state/event layer.
- First no-answer selection records attempt 1; the explicit action records attempts 2 and 3; attempt 4 fails.
- Retrying the same no-answer request token does not increment the counter twice.
- Retrying the same token does not update the source activity timestamp or postpone the archive deadline.
- Three attempts alone do not archive before three idle days.
- Three attempts plus three idle days hide the row from the seller and show it only in the resolved manager's `بایگانی‌ها ← جواب نداده` tab.
- Reassigning the source to a different seller before the deadline resets the shared counter and never archives it for the previous seller/manager.
- A simultaneous real seller activity wins over the archive job; the row remains active and its deadline moves forward.
- Missing HR manager does not route the row to an unrelated manager and shows a configuration warning.
- Opening/closing an editor without changing data does not reset the idle timer.
- Not-purchased without reason fails in the browser and through a direct AJAX request.
- Selecting pre-invoice opens the product form but does not set the final status until invoice creation succeeds.
- Seller product select groups assessment, subscription and subscription-star products.
- New invoice items snapshot `product_type`; legacy items remain valid with a nullable value.
- Seller cannot edit/archive another seller's source or issue an invoice from an archived source.
- Run the full 2.0.12 panel regression after these checks.

## Grep checks
```bash
rg "DROP|TRUNCATE|RENAME" .
rg "sn_validate_public_invoice_access" includes
rg "handle_zarinpal_callback|ZarinPal|zarinpal" includes
rg "sn_leads" includes/class-sn-plugin.php includes/class-sn-migration-service.php
rg "access_token|public_invoice_url|invoice_url" assets/js includes
```

For MIS tasks:
- Ensure no `INSERT`, `UPDATE`, or `DELETE` to `sn_leads` is introduced in MIS import/assign paths.

## Package checks
- Verify ZIP contains:
  - `sales-network.php`
  - all `includes/*.php`
  - all `assets/js/*.js`
  - all `assets/css/*.css`
  - no removed required services.
- Check file count against previous ZIP if packaging changed.

## Manual regression checks
Core:
- Seller panel opens.
- Senior supervisor panel opens through `[sn_my_panel]`.
- Sales manager panel opens.
- Sales manager panel shows direct HR children separately from all descendant sellers; for `sales_manager #15 → senior_supervisor #20 → supervisor #21 → seller #22`, #20 appears as direct senior supervisor and #22 appears in «همه فروشنده‌های زیرمجموعه» and report seller scope.
- Sales deputy panel opens.
- HR panel opens for HR user.
- MIS panel opens for MIS user.
- Old shortcodes still work if they were not targeted.

Invoice:
- Seller can create pre-invoice.
- Success message appears and invoice code is shown.
- New pre-invoice invoice code is exactly six numeric digits.
- Existing legacy invoice codes still search/open internally and are not rewritten.
- Seller invoice code is not a public link.
- Internal invoice code search works.
- Tokenized public invoice link opens.
- Public invoice without token rejects.
- Admin/internal upload/payment action does not require public token.

Finance:
- User with only `sn_view_finance` can view finance list.
- User with HR position `finance` can view the Finance panel even if their WordPress role is basic/subscriber.
- Same user cannot approve/reject.
- `sn_approve_payment` can approve.
- `sn_reject_payment` can reject.
- Seller/supervisor cannot hit finance endpoints.
- Commission dry-run creates preview rows only and does not change wallets.
- Commission dry-run shows HR compensation history snapshot or fallback warning for rows where history is missing.
- Commission run approval/rejection changes only approval metadata and logs.
- Commission wallet posting is blocked while `sn_enable_commission_wallet_posting` is off.
- Commission wallet posting requires approved run + `APPLY`.
- Re-running commission wallet posting does not duplicate item wallet transactions.
- Settlement readiness is read-only and does not debit wallets or create bank/batch payouts.
- Finance workflow guide is visible and explains which steps write data and which steps affect wallets.
- Regression readiness block is visible and lightweight.

Portal routing:
- `[sn_my_panel]` routes HR position `seller` to seller panel and seller panel access accepts HR position seller.
- `[sn_my_panel]` routes HR position `finance` to finance panel and finance panel access accepts HR position finance.
- `[sn_my_panel]` routes HR/MIS/After Sales positions to their matching panels.
- `crm-dashboard` / `[sn_my_panel]` shows a clean entry card for users with dedicated portal pages and does not embed the full resolved panel.
- Normal users do not see route debug chips; `manage_options` can see route details only inside collapsed technical diagnostics.
- Dedicated portal pages such as `crm-seller`, `crm-finance`, `crm-hr`, `crm-mis`, and `crm-after-sales` still render their panels normally.
- Unauthorized users see a safe diagnostic without sensitive data.
- Integration → «راه‌اندازی صفحات پورتال CRM» dry-run reports page status without writing.
- Portal setup apply requires `APPLY` and creates only missing pages.
- Created portal pages contain the expected shortcode and no nested duplicate plugin directory is introduced by packaging.
- `[sn_portal_nav]` shows only links the current user can access; `manage_options` sees all internal links.
- Portal links do not contain `access_token`, `token`, `public_invoice_url`, or `invoice_url`.
- `crm-dashboard` normal users see one clean entry button and no route debug chips or unrelated generic links.
- Seller main lead/customer editor autosaves name, province, city, sales prediction, contact status, and note; invoice creation/product forms are not autosaved.
- Public invoice receipt/manual transfer messages appear next to the payment section, with global notice only as fallback.
- Public guest recontact without token rejects; authorized internal recontact with invoice code/id and valid nonce works without public token.
- Finance panel shows payment rejection reason, rejecting user, and reject time where stored.
- Seller can resend a rejected invoice for finance review with note and optional corrected transfer fields; no new invoice or payment approval is created.
- HR position `hr` can add/edit/activate/deactivate HR levels without needing `manage_options`.
- HR level activate/deactivate persists after reload and failed persistence does not show a false success message.
- HR level activate/deactivate submits through a toggle-only payload and does not include the edit form `is_active=1` checkbox when deactivating.
- Integration page shows compact top summary cards and the Portal Page Setup section.
- Integration long technical sections such as migration JSON, sync JSON, scope decisions, HR profile list, hierarchy tree, invoice token sample list, skipped users, and raw reports are collapsed by default.
- Integration page keeps dangerous APPLY tools grouped under «ابزارهای اصلاح داده» and still requires explicit confirmation where applicable.
- HR compensation readiness appears on Integration page with total profiles, sellers without compensation profile, sellers without level, and sellers without commission rule.
- Integration invoice diagnostics show next six-digit invoice code, six-digit invoice count, legacy-format invoice count, and duplicate invoice code groups.

Scope:
- `off`: legacy behavior.
- `audit_only`: no visible behavior change.
- `enforce_seller_supervisor`: senior supervisor `12209` sees 28 sellers, 24422 leads, 2028 invoices.
- `enforce_sales_management`: sales manager `17780` sees 123 sellers and matching audit counts.
- Sales deputy `18730` sees all sales hierarchy in diagnostics/panel.

HR:
- HR «مدیریت نیروها» tab loads with compact audit cards and workforce rows.
- Search/filter HR workforce by user/name/phone/position/level/manager/employment state.
- Inline save can update one test employee's level, employment status/type, manager, fixed salary, commission eligibility, default rule, and effective date.
- Manager selector blocks invalid parent role, inactive manager, self-manager, and cycle.
- HR «تغییرات گروهی» dry-run reports candidates without writing.
- HR bulk apply requires `APPLY` and logs per-user results.
- HR bulk manager/position/level/employment actions do not require `base_salary`, `commission_enabled`, or other compensation fields and emit no PHP warnings.
- HR bulk compensation is the only bulk action that writes effective-dated compensation history.
- HR commission rule dropdowns show Finance rules with readable non-empty labels; inactive saved rules remain visible with an inactive marker.
- Compensation inline/bulk changes write `sn_hr_compensation_history` without overlapping ranges.
- WordPress roles, `legacy_role`, and legacy user_meta remain unchanged after HR workforce actions.
- Search `18241`: WP role/legacy role remain sales manager, HR position is senior supervisor, manager is `17780`.
- Search `17780`: HR position sales manager, manager `18730`.
- Search `18730`: HR position sales deputy, top-level.
- HR changes require dry-run and `APPLY`.
- Integration → «بازسازی زیرمجموعه فروشنده‌ها از داده قدیمی سرپرست» dry-run reads `sn_leads.supervisor_id` and reports seller → inferred manager without writing.
- Restore dry-run shows scanned sellers, legacy-supervisor matches, eligible assignments, skipped reasons, conflicts, cycle rejects, manager summary, and affected samples.
- Restore apply requires `APPLY`, uses HR assignment history, and preserves WordPress roles, `legacy_role`, HR position/level, and legacy user_meta.
- Restore default conflict behavior keeps existing managers; overwrite requires explicit `OVERRIDE`.
- Restore tool does not change sales_manager counts and only assigns seller → inferred manager.
- After restore apply, `senior_supervisor → seller direct` may be greater than zero in the transitional structure.
- HR compensation timeline action requires `APPLY`, creates a history row, closes the previous open row where applicable, and does not touch wallet/payment/invoice/SMS.

MIS:
- Fresh-site Integration Launch readiness cards are visible and show whether the site is ready for install, HR import, hierarchy setup, MIS testing, seller testing, finance testing, and internal testing.
- HR «ورودی/خروجی کاربران» accepts CSV/XLSX in simple mode with only `نام`, `نام خانوادگی`, `موبایل`, `سمت`, and optional `توضیحات`.
- Simple HR import does not require level, direct manager, salary, commission, or effective date.
- Simple HR import dry-run with 5 valid people reports no `level_not_found` or `manager_not_found`.
- Simple HR import apply requires `APPLY`, creates/updates HR profiles only after row validation, and does not create users for invalid rows.
- HR «افزودن دستی نیرو» creates/updates a basic HR profile with position and active status; manager/level/compensation are configured later.
- Simple-imported and manually-added people appear immediately in HR «مدیریت نیروها» even with no manager, no level, no employee code, and no compensation history.
- HR «مدیریت نیروها» default view shows all HR profiles; incomplete rows display «بدون مدیر»، «بدون سطح»، «بدون تاریخچه حقوق/پورسانت»، and «نیازمند تکمیل» badges.
- HR «بررسی نمایش نیروهای واردشده» shows latest profiles and whether each row is visible in the workforce table.
- HR people import does not change existing WordPress roles, `legacy_role`, or legacy user_meta.
- Imported Subscriber users with HR Position can access their matching CRM panel without custom WordPress roles.
- CRM login accepts mobile number, username, and email; duplicate mobile is rejected with a clear message.
- HR/Admin single-user password reset works and stores no plain password in options/logs/reports.
- HR credential SMS reports masked phone and send status only; provider failure reports that the password changed but SMS was not sent.
- Integration launch readiness shows users with mobile, duplicate mobile count, and mobile-login-ready count.
- MIS «ورود فایل اطلاعات» accepts Persian CRM data headers: شماره اول، شماره دوم، نام مشتری، استان، شهر، آدرس، توضیحات.
- MIS import accepts phone headers: شماره اول، شماره اصلی، موبایل، شماره موبایل، تلفن همراه، شماره تماس، شماره، phone، mobile، mobile_1، phone_1.
- MIS import reads a one-column UTF-8 CSV whose first non-empty header is `شماره اول`; BOM, semicolon-delimited, and tab-delimited variants are parsed without producing `detected_headers=[""]`.
- MIS import stops with a clear Persian XLSX warning if XLSX reading is unavailable and does not create blank invalid staging rows from an unreadable XLSX.
- MIS import normalizes `09120000001`, `9120000001`, `+989120000001`, `989120000001`, `۰۹۱۲۰۰۰۰۰۰۱`, and `0912-000-0001` to `09120000001`.
- MIS import marks empty, too-short, too-long, non-mobile-prefix, missing-column, and non-numeric rows with explicit invalid reasons.
- Minimal launch template with only `شماره اول` and valid mobile rows imports as valid rows.
- MIS import stores rows only in `sn_mis_data_rows`; `sn_leads` count remains unchanged.
- MIS «تخصیص سریع دیتا» dry-run reads valid `sn_mis_data_rows` and shows eligible rows without requiring Lead Pool or writing.
- MIS «تخصیص سریع دیتا» apply with `APPLY` creates `sn_distribution_items` with `source_type=mis_data_row` and logs direct MIS assignment when target is seller.
- Re-running MIS quick assignment reports already assigned rows and creates no duplicate distribution items.
- Direct MIS-to-seller launch assignment does not create live `sn_leads`; seller live lead creation remains a separate controlled action.
- Tables exist: `sn_mis_import_batches`, `sn_mis_data_rows`, `sn_mis_distribution_logs`.
- MIS guide is visible and explains the hierarchical flow: input data case, temporary import/validation, manager assignment, ready distribution queue, sales manager, senior supervisor, supervisor, seller, then seller lead.
- MIS UI uses Persian operator labels such as «پرونده ورود اطلاعات»، «ورود فایل»، «صف آماده توزیع»، and «برنامه پخش» instead of raw English labels in the main UI.
- MIS direct/controlled live-lead tooling is labeled as a controlled/technical path, not the standard operator flow.
- MIS create/import forms are compact and aligned on desktop and usable on mobile.
- MIS logs show Persian summaries for `batch_created`, `assigned_rows=0`, and `eligible=0`; raw JSON is collapsed and masked.
- Create batch.
- Import small CSV.
- Dashboard/preview/conversion counts match row table.
- Assign to sales manager updates MIS rows only.
- `sn_leads` count unchanged.
- No SMS sent.

- Distribution engine tables exist: `sn_distribution_chains`, `sn_distribution_chain_steps`, `sn_distribution_items`, `sn_distribution_item_logs`.
- Default configurable chain is seeded and active.
- HR panel shows «زنجیره پخش داده» and can add/edit/deactivate chains/steps without deleting rows.
- MIS «تحویل به زنجیره پخش» dry-run reports eligible/already delivered rows without writing.
- MIS delivery apply with `APPLY` creates distribution items and does not create `sn_leads`.
- Running MIS delivery twice does not duplicate distribution items.
- Sales manager sees owned distribution items and can transfer only to allowed HR-subtree recipients.
- Senior supervisor sees owned distribution items and can transfer to the next configured layer or skip missing layer only when allowed.
- Supervisor can deliver owned items to allowed sellers.
- Unauthorized users cannot transfer someone else’s distribution item.
- Invalid/out-of-subtree recipient is rejected.
- Final seller delivery sets distribution item status to `delivered_to_seller`; live `sn_leads` creation remains separate/controlled in this phase.
- Distribution logs show every transfer with masked payloads and no tokens/public URLs.
- Seller panel shows delivered distribution items in «داده‌های تحویل‌شده».
- Single delivered item conversion creates at most one `sn_leads` row.
- Bulk delivered item conversion requires `APPLY`.
- Conversion writes `live_lead_id`, `converted_by`, `converted_at`, and `conversion_status` on `sn_distribution_items`.
- Duplicate phone conversion does not update/delete an existing lead and marks the item as duplicate/error.
- Re-running conversion for the same item creates 0 additional leads.
- Non-final/non-delivered distribution items cannot be converted to live leads.
- MIS launch self-check shows PASS/next-action rows for HR hierarchy, MIS tables, valid rows, manager assignment, Lead Pool, distribution items, seller delivery, controlled lead conversion, and finance readiness.
- MIS self-check is read-only and does not alter `sn_mis_data_rows`, `sn_mis_lead_pool`, `sn_distribution_items`, or `sn_leads`.

## 2.0.11 HR user transfer
- WordPress admin shows `شبکه فروش ← کاربران HR`; the HR frontend shows `ورودی/خروجی کاربران` and both use the same module.
- AJAX search returns only HR profiles and rejects logged-out/non-HR users.
- Selected export rejects an empty selection and ignores forged IDs that do not have an HR profile.
- All export streams every HR profile without a fixed 300/500-row display cap.
- CSV opens as UTF-8 and includes `export_schema=sn_hr_users_v1` without password, hash, token or session columns.
- A name/note beginning with `=`, `+`, `-` or `@` cannot execute as a spreadsheet formula and returns to its original text after import.
- Exported manager username resolves on import; numeric source IDs are informational and are not the primary cross-site identity.
- Commission rule code resolves before numeric rule ID.
- Dry-run reports duplicate identities, employee codes already owned by another profile, missing positions/levels/structure, invalid dates, invalid commission rules, self-manager and invalid manager position.
- Apply without `confirm_apply=APPLY` changes nothing and reports `apply_confirmation_missing`.
- A legacy `sn_hr_csv_dry_run` nonce always forces Dry-run, even if a forged request submits `run_mode=apply`.
- CSV quoted multiline notes remain one row.
- Files larger than 5 MB, more than 10,000 rows, XLSX sheets wider than 512 columns, binary `.xls`, oversized XLSX XML and unsupported extensions are rejected.
- Omitted optional columns preserve current profile values; WordPress roles and `legacy_role` are unchanged by people import.
- Project manager/expert Persian labels resolve to `project_manager` / `project_expert`, and compatibility role mapping uses `sn_project_manager` / `sn_project_expert` when the manual-add path applies access.
- Logged-out `sn_get_lead_statuses` is unavailable; logged-in requests require the `sn_public` nonce.
- HR transfer CSS/JavaScript is not enqueued on unrelated frontend pages and is present on the HR page.
- Failed seller/customer/status/extra-number writes return a generic message; raw database/exception text is available only in the server log or restricted diagnostic report.

## 2.0.12 full file and panel audit

- `node tools/static-audit.js --json` returns `ok: true` with no errors.
- Plugin header version equals `SN_VERSION` and both report the current release version (`2.0.13` for this build).
- All literal include/asset references exist in the installed ZIP.
- All JavaScript files pass `node --check`; all PHP files pass `php -l` on Staging.
- Light bootstrap and core register the same 26 shortcodes, including `[sn_admin_front_dashboard]`.
- Every HR-resolved panel and every portal page registry shortcode is registered.
- A normal WordPress page containing each dedicated panel shortcode loads its expected CSS/JavaScript.
- An Elementor page containing `[sn_my_panel]` loads and routes exactly like a normal shortcode page.
- An Elementor page containing a dedicated seller/manager/finance/HR/MIS panel loads its panel asset without a blank or unstyled interface.
- Fresh activation on an empty Staging database creates HR/MIS migration tables before Projects/Dot seed their HR positions.
- Upgrade from 2.0.11 preserves invoices, payments, HR profiles, projects, chat messages, files, notifications, wallet and Biavin ledger rows.
- Logged-out/internal unauthorized requests cannot invoke protected panel actions or download project files.
- Error logs are inspected after opening every role panel and after one full invoice/payment/project/chat/wallet scenario.
- No runtime PHP contains `DROP TABLE`, `TRUNCATE TABLE` or `RENAME TABLE`.

Finance commission launch readiness:
- Finance commission rule form uses dropdowns/Persian labels for payment method, invoice status, employment status, and active state.
- Existing commission rule values display as readable Persian labels with internal slugs kept only as stored values.
- Commission diagnostics show active rules, unknown HR/payment/status values, zero-match rules, and HR compensation-history readiness.
- Commission dry-run still writes only dry-run tables and does not post wallet, approve payments, create invoices, or send SMS.
- Integration «بررسی آماده‌سازی لانچ» shows HR, Finance, and MIS readiness counts and does not write data.

HR credential SMS:
- HR/Admin can select one or more users and send login credentials only after password reset and `APPLY`.
- The SMS text includes username, temporary password, and login URL.
- Latest credential report/audit shows masked phone and status only.
- No plain password is present in stored option reports or HR log display after sending.
- Existing WordPress roles, `legacy_role`, and legacy user_meta are not changed by credential SMS.

Seller resubmit:
- Rejected invoice resend modal accepts note and corrected transfer fields.
- Optional receipt re-upload uses logged-in seller/admin permission and nonce.
- Resubmission returns invoice to finance review without creating a new invoice, auto-approving payment, triggering ZarinPal, exposing public token, or sending SMS.

Customer profile / reports / exports:
- `[sn_customer_profile]` opens for an authorized internal user.
- Search by invoice code, phone/name query, and lead ID does not expose `access_token` or public invoice URL.
- Seller/sales hierarchy users see only scoped customer rows.
- `[sn_reports_panel]` opens for authorized report users.
- CSV exports download with expected headers, limited rows, formula-safe cells, and no token/password/hash/public invoice URL fields.
- Exporting reports does not write to `sn_leads`, invoices, payments, wallets, commissions, MIS, HR, or After Sales tables.

## Final report checklist
- Files changed/created.
- Exact scope of changes.
- What was intentionally not changed.
- Commands run.
- Manual test results.
- Remaining risks.
- Next recommended phase.

## Supervisor/Senior Supervisor unified allocation QA

1. MIS import + quick assignment creates `sn_distribution_items` for manager.
2. Sales Manager transfers one item to Senior Supervisor.
3. Login as Senior Supervisor and verify `تخصیص شماره` shows the item in the embedded old-style allocation UI.
4. Transfer the item to Supervisor and verify `requested_items=1`, `posted_field=distribution_item_ids`, `transferred=1`.
5. Login as Supervisor and verify `تخصیص شماره` shows the item in the same unified UI.
6. Transfer to Seller and verify no `sn_leads` are created automatically.
7. Verify public invoice, ZarinPal, SMS, wallet, and commission paths are unchanged.

## Supervisor HR Seller Visibility Fix QA
- با کاربر `#21` / HR position `supervisor` وارد شوید.
- در HR، فروشنده `#22` باید با parent مستقیم `#21` و position `seller` ثبت شده باشد.
- در پنل سرپرست، تب «فروشندگان» باید `#22` را نشان دهد حتی اگر WP role او `sn_seller` نباشد.
- تب «تخصیص شماره» همچنان باید از `sn_distribution_items.current_owner_user_id = 21` بخواند.
- انتقال `#21 → #22` باید با `distribution_item_ids[]` و `recipient_user_id` کار کند.

## Supervisor Allocation UX / V4 Distribution QA

- Login as supervisor #21 with active HR sellers under the supervisor.
- Confirm the allocation tab shows the same active sellers in KPI and recipient dropdown.
- With zero owned distribution items, confirm the dropdown still contains sellers and the page shows the clear no-owned-items message.
- Transfer one owned item from #21 to #22 using selected row mode; expect `requested_items=1`, `posted_field=distribution_item_ids`, `transferred=1`.
- With multiple owned items, test assigning N items to one seller and round-robin distribution across all active HR sellers.
- Verify no `sn_leads` rows are created by transfer.

## Seller V4 Delivered Data QA - 2026-06-09

1. Transfer a V4 distribution item to seller user #22.
2. Log in as #22 with HR position `seller` even if the WP role is Subscriber.
3. Verify seller panel opens and section «دیتای تحویل‌شده به شما» shows the transferred item, masked phone, batch code, status, and received timestamp.
4. Open «مشاهده/ویرایش», save contact status / probability / note / next follow-up. Confirm no `sn_leads` row is created by this save.
5. Click «تبدیل کنترل‌شده به سرنخ فروش» and verify one `sn_leads` row is created and linked; repeated click does not duplicate.
6. Regression: MIS import, quick assignment, sales manager transfer, senior supervisor transfer, supervisor allocation UX, payment/ZarinPal/SMS/public invoice/wallet/commission remain unchanged.

## Seller Inline Save QA - 2026-06-09
- Open seller old panel → شماره‌های من → مشاهده/ویرایش.
- Change customer name, Tab to next field: editor remains open and field saves.
- Change «وضعیت تماس»: message shows status saved and editor remains open.
- Change several fields quickly: latest state persists; no XHR abort loss.
- V4 distribution item edit must not create sn_leads automatically.

## 2026-06-09 QA additions
- Seller #22 with HR position seller can load “رفتار مشتریان” without legacy `sn_seller` WP role.
- After public “ارتباط مجدد با کارشناس”, the invoice still allows online payment, receipt upload, and manual payment info submission.
- Finance-approved invoice for a product with “ارجاع بعد از پرداخت” creates at most one `paid_invoice_referral` distribution item.
- Referral item owner matches configured product target: self / direct supervisor / senior supervisor.
- Referral item does not auto-create `sn_leads`; live lead conversion remains controlled.


## 2026-06-09 - Paid invoice manual repeat action referral
- Added product setting for manual repeat action after finance approval.
- Manual mode creates a `paid_invoice_referral` item with `needs_repeat_action` for seller decision.
- Seller keeps the original decision step. Once routed to a supervisor or senior supervisor, the item appears in the shared `آماده‌های تبدیل` queue instead of exposing a second invoice form.
- Existing `sn_distribution_items` rows are retained and linked idempotently to `sn_dot_cases`; supervisor self-assignment continues in `تبدیل‌های من`.
- Payment-completion workflow rows remain on the staged-payment path and preserve the original seller for commission.

## 2026-09-08 - Unified conversion, draft-safe history, callback date/time
- Open Supervisor → `آماده‌های تبدیل`; verify legacy paid-referral rows are migrated once and no `صدور فاکتور جدید` action is rendered there.
- Open Senior Supervisor → `آماده‌های تبدیل`; verify the same shared queue and assignment behavior.
- Assign a case to the current supervisor; verify it appears in `تبدیل‌های من` and cannot be managed outside that supervisor scope.
- Set a previously sold WooCommerce product to Draft; verify it disappears from new-sale selectors while old invoice item name, price, totals, payments, and activity remain readable.
- Verify a draft/deleted/disabled product cannot create a new conversion-payment invoice and returns a history-preserving message.
- Select `تماس مجدد`; verify both date and time become visible/required, save stays disabled until both are set, past time is rejected server-side, and the due reminder appears in the responsible user's conversion panel.
- Verify `انصراف` cannot be saved without a reason.

## 2026-09-09 - Paid assessment to supervisor queue repair
- As a seller, issue a single assessment invoice and select `ارسال برای سرپرست و تبدیل‌کننده`.
- Complete the full invoice payment and approve its final stage in Finance.
- Before the customer selects any subscription/product option, open the linked supervisor's `آماده‌های تبدیل` tab.
- Verify the case is present with `انتخاب نکرده` and can be assigned either to that supervisor or an eligible converter.
- Reopen the same tab without a full page reload and verify it refreshes instead of showing stale lazy-loaded HTML.
- Repeat Finance approval and tab loading; verify only one `sn_dot_cases` row exists for the source invoice.
- Temporarily disable Dot after invoice creation, then approve the already-issued assessment invoice; verify its invoice-time link still materializes safely.
- Confirm partial payment approval does not create the conversion case until the invoice remaining amount reaches zero.

## 2026-09-09 - Optional assessment selection SMS and seller queue
- Select an assessment product in the seller invoice form and verify `ارسال اعتبارسنجی` appears checked by default.
- Keep it checked, select each conversion route separately, complete all payment stages, and approve the final stage in Finance; verify the existing subscription-selection SMS is sent exactly once.
- Clear the checkbox, repeat both conversion routes, complete payment, and approve in Finance; verify no access SMS, retry Cron, or Action Scheduler job is created.
- Verify the SMS-disabled case is created as `ready_for_conversion` with no `sms_due_at`, while the source invoice/payment/history rows remain unchanged.
- With `تبدیل توسط خودم`, verify the paid case appears immediately in the seller's `آماده‌های تبدیل من` tab whether the checkbox is checked or cleared.
- With `ارسال برای سرپرست و تبدیل‌کننده`, verify the SMS-disabled case appears in the supervisor's `آماده‌های تبدیل` queue with `انتخاب نکرده` and remains assignable.
- Submit an invoice from an older cached form that omits `dot_send_assessment_sms`; verify it preserves legacy behavior and sends the selection SMS.
- Reopen the seller tab after simulating a missed `sn_invoice_paid` callback; verify the bounded repair creates one case only and never duplicates the source invoice.

## Commission resolver QA
- Verify four rule combinations: training+online, training+card_to_card, contractor+online, contractor+card_to_card.
- Verify supervisor-issued invoice keeps seller_id as commission owner and issued_by_user_id only as audit/log.
- Verify Finance dry-run and Seller panel preview show the same matched rule and amount.
- Verify no automatic wallet posting occurs from preview.


## Commission cumulative matching update
- Seller panel standalone `پورسانت من` tab removed; seller wallet remains the user-facing commission area.
- Commission dry-run now sums all active matching commission rules for an invoice instead of selecting only one best rule.
- Dry-run notes include `matched_rule_ids`, `commission_components`, and `stack_mode=cumulative_all_matching_rules`.
- Wallet posting remains controlled and uses the aggregated dry-run amount; no automatic posting was added.

Commission wallet finance-rule integration: php -l, node --check, and no destructive SQL in executable code.

## Sales Deputy panel scope QA - 2026-06-10
- Sales Deputy panel must not show a data-distribution/allocation tab.
- Sales Deputy panel must show Managers, Senior Supervisors, Supervisors, Sellers and Invoices as read/monitoring tabs.
- MIS quick assignment must still target Sales Manager, not Sales Deputy.
- Sales Manager distribution recipients must include active descendant senior supervisors, supervisors, and sellers in the manager HR branch.


## Supervisor panel functional audit - 2026-06-10
- Cleaned supervisor operational copy and legacy labels.
- Supervisor KPI cards now separate V4 delivered items from legacy live leads.
- Direct HR sellers are the source for seller count and dropdowns; WP role is not required.
- Fixed supervisor invoices tab target id mismatch.
- V4 distribution reports/debug are hidden in embedded operational panels unless explicitly enabled.

## Operational distribution APPLY removal QA
- Sales manager allocation: transfer selected item without typing APPLY; nonce/ownership/recipient scope must still validate.
- Senior supervisor allocation: assign N / round-robin / selected transfer without typing APPLY.
- Supervisor allocation: assign N / round-robin / selected transfer without typing APPLY.
- MIS quick assignment and chain delivery forms no longer require typing APPLY for normal operational data handoff.
- Controlled live lead creation, HR destructive tools, finance wallet posting, and integration repair tools must still require explicit confirmation where designed.


## HR advanced import QA — 2026-06-12
- Install package and confirm existing panel styling is unchanged; no `assets/css/*` or `assets/js/*` files were modified.
- In HR CSV tab, download the advanced and simple Excel samples from the UI links.
- Upload `sn-hr-people-advanced-sample.xlsx` with dry-run: `import_mode` should be `advanced_people`; manager rows should show `manager_status=resolved` or `in_file` and `would_assign=true` where applicable.
- Upload the real sales-network Excel with dry-run: manager fields should no longer be blank when `موبایل مدیر` / supported manager headers exist.
- Apply only with `APPLY`: verify profile creation/update counts first, then `assignments_changed` / `assignments_skipped`.
- Simple mode must still ignore hierarchy and act as people/position import only.
- MIS import tab must expose the MIS sample Excel link and continue to stage rows only; no live `sn_leads` rows should be created by MIS import.


## 1.0.26 XLSX sparse-cell fix
- HR/MIS XLSX parser now preserves empty cells based on Excel cell references, preventing mid-row blank cells from shifting columns during import dry-run/apply.
- No CSS or JS assets changed.


### HR password reset QA
- [ ] Set password for one selected HR user; verify login with mobile as username and new password.
- [ ] Bulk set password for selected HR users with APPLY confirmation; verify changed/skipped counts.
- [ ] Bulk set password for all active HR profiles; verify the current actor is skipped if applicable.
- [ ] Confirm raw passwords are not present in option reports, HR logs, or audit output.


## v1.0.33 - Zibal Gateway
- Added selectable online payment gateway: ZarinPal or Zibal.
- Added Zibal merchant and test-mode settings.
- Zibal uses `/v1/request`, `/start/{trackId}`, and `/v1/verify`; invoice amounts remain stored in Toman and are sent to gateways as Rial.
- Callback route remains the existing `sn_callback=1&invoice_id=...` path and now also accepts `sn_gateway=zibal`.
- Normal online payment completion still marks invoices paid and triggers `sn_invoice_paid`.

## 2.0.17 - Supervisor self-converter

- [ ] A supervisor can assign one selected Dot case to “تخصیص به خودم”.
- [ ] Bulk self-assignment updates every selected eligible case atomically.
- [ ] The assigned case appears in the same supervisor's converter workspace.
- [ ] The supervisor can update contact status and create/resend the case payment flow.
- [ ] The supervisor cannot open another supervisor's self-assigned case or a subordinate converter's case.
- [ ] A supervisor without the converter role cannot issue an unrelated manual invoice from this workspace.
- [ ] Subordinate converter counts and team analytics do not count the supervisor as a converter candidate.
- [ ] Seller stage-one, HR transfer, Campaign/UTM and public payment smoke tests remain green.

## 2.0.18 - Assessment lifecycle

- [ ] Mark a WooCommerce product as `اعتبارسنجی`; a seller invoice stores `assessment` in `sn_invoice_items.product_type` even when the product is not an old auto-detected Dot product.
- [ ] The assessment product remains single-quantity/single-item and supports full or partial payment.
- [ ] An unpaid active assessment payment stage is archived after 3 days under `عدم پرداخت اعتبارسنجی` and disappears from the seller queue without deleting its invoice.
- [ ] Approving a late/partial assessment payment resolves that archive and restores the seller source when the assessment is not yet complete.
- [ ] Full payment plus Finance approval creates the Dot case and sends the existing customer-selection SMS.
- [ ] A case with no customer selection appears in `آماده‌های اقدام` with the `انتخاب نکرده` tag and can be assigned to the supervisor or a subordinate converter.
- [ ] Converter/supervisor can record `جواب نداده`, `تماس مجدد`, notes and the final product.
- [ ] `انصراف` is rejected without a reason and becomes terminal when saved with a reason.
- [ ] Full and staged conversion payment still work; a latest unpaid link is archived after 5 days under `عدم پرداخت اشتراک`.
- [ ] A late approved conversion payment resolves the 5-day archive and returns the case to `deposit_paid` or `completed` without duplicating totals.
- [ ] Old invoices/cases retain their IDs, ownership, status, selected option, payment rows and history after migration.
