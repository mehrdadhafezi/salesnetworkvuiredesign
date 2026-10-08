# ممیزی گزارش‌های CRM (نسخهٔ ورودی 2.0.103)

منبع: بستهٔ `sales-network-v4_4.zip`؛ SHA-256 فایل در `source-hashes.txt` تحویل ثبت شده است. دسترسی به پایگاه دادهٔ زنده یا صفحهٔ اجراشدهٔ `/crm-dashboard/` در این بررسی وجود نداشت؛ این ممیزی بر مبنای کد است.

## مسیر صفحه و نقاط ورود

`/crm-dashboard/` در `SN_Plugin::sn_portal_page_setup_report()` صفحه‌ای با `[sn_my_panel]` است و با سمت کاربر به پنل عملیاتی او هدایت می‌شود. صفحهٔ گزارش جداگانه `/crm-reports/` با `[sn_reports_panel]` است. قالب صفحهٔ عمومی در `class-sn-plugin.php` و اسکریپت‌های پنل در `assets/js/public-*.js` قرار دارند. شورت‌کدهای `[sn_admin_front_dashboard]`, `[sn_seller_panel]`, `[sn_supervisor_panel]`, `[sn_sales_manager_panel]`, `[sn_financial_panel]`, `[sn_mis_panel]` نیز کارت‌ها و گزارش‌های مستقلی تولید می‌کنند. در بخش مدیریت `sn-dashboard` و داشبورد کمپین نیز گزارش‌های جدا هستند. اکشن خروجی عمومی گزارش قدیمی `admin_post_sn_export_report_csv` است؛ مسیرهای مالی و مدیر فروش خروجی جدا دارند. اکشن‌های AJAX در `class-sn-plugin.php` و کلاس‌های عملیاتی و پروژه ثبت می‌شوند؛ گزارش REST اختصاصی واحد در این نسخه شناسایی نشد.

## شاخص‌های پنل گزارش قدیمی

| شاخص | نقش/مسیر | تابع و محاسبه | جدول و فیلتر | مسئله |
| --- | --- | --- | --- | --- |
| لیدها | مدیر/مالی و برخی سمت‌های HR، MIS | `sn_cross_module_report_summary`: `COUNT(*)` | `sn_leads`؛ تاریخ `imported_at`، محدودهٔ فروشنده | لید ورودی MIS که هنوز به فروشنده نرسیده در این عدد نیست. |
| فاکتورها | همان | `sn_cross_module_report_summary`: `COUNT(*)` | `sn_invoices`؛ تاریخ ایجاد، فروشنده، وضعیت | تعداد فاکتور به معنی فروش قطعی نیست. |
| مبلغ فاکتورها | همان | `SUM(final_total)` با fallback مجموع `product_price` تنها در صورت NULL شدن کل مجموع | `sn_invoices`؛ همان فیلترها | این fallback ردیف به ردیف نیست؛ بعضی مبلغ‌های NULL از مجموع حذف می‌شوند. |
| پرداخت/تأیید شده | همان | `SUM(CASE WHEN status IN ('paid','approved') THEN COALESCE(final_total,product_price) END)` | `sn_invoices`؛ تاریخ ایجاد | وضعیت فاکتور جایگزین مبلغ تراکنش دریافتی یا تأیید مالی نیست؛ پرداخت اقساطی می‌تواند دو بار یا نادرست تفسیر شود. |
| خدمات پس از فروش | همان | `COUNT(*)` | `sn_after_sales_cases`؛ بدون فیلتر تاریخ/محدوده | ممکن است داده‌های خارج از محدوده را نمایش دهد. |
| پرونده MIS | همان | `COUNT(*)` | `sn_mis_import_batches`؛ بدون فیلتر | دادهٔ ورودی، لید زنده و تخصیص را تفکیک نمی‌کند. |
| پروفایل HR | همان | `COUNT(*)` | `sn_hr_profiles`؛ بدون فیلتر | برای بعضی نقش‌ها شمارش سراسری است. |
| عملکرد فروشنده | همان | `sn_cross_module_report_sections`: `COUNT(*)`, `SUM(COALESCE(final_total,product_price))` | `sn_invoices`, `wp_users`؛ تاریخ ایجاد، فروشنده، وضعیت؛ ۲۰ ردیف | مبلغ پرداخت‌شده با وضعیت فاکتور یکی دانسته می‌شود. |
| پرداخت‌های موفق | تب اول | `sn_successful_payment_report_rows/stats/sql` | تراکنش/فاکتور، فیلترهای مخصوص تب | از محاسبات نمای کلی جداست. |
| خروجی CSV قدیمی | `handle_export_report_csv` | `sn_export_report_rows` و توابع هر نوع | لید/فاکتور/پرداخت/کیف پول/HR/MIS | چند نوع خروجی قدیمی فیلتر و محدودهٔ فروشنده را اعمال نمی‌کردند؛ در این بسته دسترسی آن به مدیر محدود شد. |

## جداول کلیدی

تعریف پایه در `SN_Activator::create_tables`: `sn_leads`, `sn_lead_statuses`, `sn_invoices`, `sn_invoice_payment_stages`, `sn_payments`, `sn_invoice_items`, `sn_invoice_wheel`, `sn_activity_logs`, `sn_lead_status_history`. مهاجرت‌های افزوده در `SN_Migration_Service` برای `sn_hr_*`, `sn_mis_*`, `sn_distribution_*`, `sn_after_sales_*`, `sn_commission_*` و سایر ماژول‌ها هستند. جدول‌های افزودهٔ کمپین، پروژه، محصول، فروشنده و عملیات در کلاس‌های مربوط تعریف می‌شوند. فهرست کامل جدول‌های موجود و حجم آنها بدون اتصال به پایگاه دادهٔ نصب‌شده قابل تأیید نیست.
