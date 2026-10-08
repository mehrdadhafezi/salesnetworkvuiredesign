# گزارش تست 2.0.77

- `node tools/static-audit.js --json`: **PASS** — نسخه `2.0.77`، بدون invariant ناموفق.
- `node --check assets/js/public-supervisor.js`: **PASS**.
- تطابق Header افزونه و `SN_VERSION`: **PASS**.
- مسیر AJAX بارگذاری فیش، nonce، نقش مجاز و scope فاکتور: **PASS** در ممیزی ایستا.
- نوع فایل‌های مجاز: JPG, JPEG, PNG, WebP, PDF با سقف پیش‌فرض 5MB.
- رگرسی پرداخت مرحله‌ای: وضعیت `partial_paid` در UI و server guard پوشش داده شد — **PASS**.
- مقایسه با بسته `2.0.76`: فقط فایل‌های مرتبط با UI سرپرست، endpoint پرداخت، ممیزی ایستا، نسخه و مستندات تغییر کرده‌اند.
- از آنجا که محیط WordPress/PHP/MySQL در این workspace فعال نیست، smoke test واقعی آپلود باید روی Staging انجام شود.
