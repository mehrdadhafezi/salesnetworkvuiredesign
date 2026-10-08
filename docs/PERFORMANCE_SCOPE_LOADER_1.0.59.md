# v1.0.59 Performance Scope Loader

این نسخه برای کاهش اثر افزونه روی صفحات غیر CRM ساخته شده است.

## تغییرات اصلی

- Bootstrap سبک در فایل اصلی افزونه:
  - کلاس‌های سنگین CRM فقط در این حالت‌ها load می‌شوند:
    - wp-admin عادی
    - admin-ajax/admin-post با actionهای `sn_*`
    - callback پرداخت با `sn_callback`
    - REST routeهای `sn-crm`
    - صفحه‌ای که shortcodeهای CRM دارد
  - صفحات عمومی سایت مثل home، landing page و نوشته‌ها دیگر کلاس اصلی ۲۳هزار خطی را load نمی‌کنند مگر shortcode CRM داشته باشند.

- Assetها:
  - CSS/JS فرانت فقط روی صفحات shortcode CRM یا My Account enqueue می‌شوند.
  - حذف فونت‌های خارجی فقط روی صفحات CRM انجام می‌شود، نه کل سایت.

- JWT:
  - دستکاری JWT فقط برای AJAXهای خود افزونه با action prefix `sn_` انجام می‌شود.
  - filterهای whitelist عمومی JWT حذف شدند.

- Database / Migration:
  - `maybe_create_tables` با option نسخه‌دار `sn_runtime_tables_version` محدود شد.
  - `maybe_run_migration_service` فقط وقتی `sn_db_version` یا `sn_perf_index_version` عقب باشد اجرا می‌شود.
  - indexهای عملکردی برای لید، فاکتور، distribution و wallet اضافه می‌شود.

- Role/Table maintenance:
  - عملیات role hardening و ایجاد roleهای مالی/مدیر فروش/after-sales نسخه‌دار شد.
  - ساخت wallet tables در admin با version gate محدود شد.

## نکات تست

- صفحه اصلی سایت بدون shortcode CRM نباید `class-sn-plugin.php` را load کند.
- صفحات پنل CRM باید مثل قبل کار کنند.
- admin-ajax با actionهای غیر `sn_*` نباید JWT/CRM را درگیر کند.
- روی اولین ورود wp-admin بعد از نصب، migration/index یک بار اجرا می‌شود.
