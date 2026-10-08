# 2.0.122 — Port کامل Bale / Safir روی مبنای 2.0.121

این نسخه قابلیت‌های بله/Safir نسخه Production تست‌شده را روی مبنای 2.0.121 Port می‌کند، بدون حذف تغییرات 2.0.118 تا 2.0.121.

## حفظ تغییرات نسخه مقصد
- مجوز صدور دستی HR و ارث‌بری آن از 2.0.118/2.0.119 حفظ شده است.
- تشخیص دقیق نتیجه SMS و نمایش پیام واقعی Provider از 2.0.120 حفظ شده است.
- فروش مستقل محصول* و تنظیم نمایش مستقل آن از 2.0.121 حفظ شده است.
- `class-sn-sms.php` نسخه 2.0.121 بدون Rollback به نسخه قدیمی حفظ شده است.

## Bale / Safir
- Gateway مستقل با HMAC و Delivery Log اضافه شده است.
- REST endpointهای `/wp-json/sn/v1/bale/link` و `/wp-json/sn/v1/bale/status` اضافه شده‌اند.
- جداول `sn_message_deliveries` و `sn_bale_accounts` به‌صورت additive ساخته می‌شوند.
- SMS Providerهای فعلی جایگزین نمی‌شوند؛ Bale یک Channel مستقل و best-effort است.
- پیام‌های فاکتور، مراحل/ارسال مجدد لینک، Dot Flow، پرتال مشتری، OTP، HR، عملیات/ارتقای کارت، شارژ کیف پول، تکمیل پرداخت و Reward در بله Mirror می‌شوند.
- پنل تنظیمات «بله / Safir»، Admin Test و Delivery Log اضافه شده‌اند.

## پیش‌فاکتور فروشنده
- گزینه مستقل «ارسال در بله» کنار ارسال پیامک قرار دارد و به‌صورت پیش‌فرض روشن است.
- `send_bale=1/0` به Backend ارسال می‌شود.
- خاموش کردن تیک بله فقط Bale را متوقف می‌کند و SMS همچنان ارسال می‌شود.
- نتیجه دقیق SMS نسخه 2.0.120 و نتیجه Bale هر دو در UI نمایش داده می‌شوند.

## QA
- PHP lint تمام فایل‌های PHP.
- JavaScript syntax check.
- Static audit افزونه.
- بررسی نبود conflict marker.
- بررسی HMAC/signature و Idempotency با harness مستقل.
- بررسی حفظ markerهای HR manual invoice، SMS result و standalone product* نسخه 2.0.121.
