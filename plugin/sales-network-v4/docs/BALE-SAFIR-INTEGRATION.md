# Bale / Safir Integration — Sales Network CRM 2.0.119

این نسخه، کانال بله را به‌صورت **additive** کنار SMS فعلی اضافه می‌کند. هیچ Provider پیامکی حذف یا با بله جایگزین نشده است. نتیجه SMS همچنان رفتار Business Flow را تعیین می‌کند و خطای بله باعث Rollback فاکتور/پرداخت/اعتبارسنجی نمی‌شود.

## معماری

```text
Sales Network CRM 2.0.119
  ├─ Existing SMS -> Faraz / Kavenegar / MeliPayamak
  └─ SN_Notification_Service
       └─ SN_Bale_Gateway (HMAC)
            └─ https://bale.andishesazanco.ir/api/v1/notify
                 └─ Safir API -> @biawincrmbot

@biawincrmbot -> Gateway Webhook -> CRM REST
  ├─ POST /wp-json/sn/v1/bale/link
  └─ POST /wp-json/sn/v1/bale/status
```

Bot ID: `197632729`

> Bot Token و `BALE_SAFIR_API_KEY` نباید داخل WordPress/CRM ذخیره شوند. این Secretها فقط روی Gateway نگهداری می‌شوند.

## تنظیمات پیشنهادی Production

بهتر است URL و Shared Secret در `wp-config.php` قرار گیرند:

```php
define('SN_BALE_GATEWAY_URL', 'https://bale.andishesazanco.ir');
define('SN_BALE_GATEWAY_SHARED_SECRET', 'PUT_THE_SAME_VALUE_AS_GATEWAY_CRM_SHARED_SECRET_HERE');
```

مقدار `SN_BALE_GATEWAY_SHARED_SECRET` باید دقیقاً با `CRM_SHARED_SECRET` در `.env` Gateway یکسان باشد.

اگر Constant تعریف نشود، Shared Secret از تب «تنظیمات شبکه فروش → بله / Safir» به‌صورت write-only قابل ذخیره است و هیچ‌وقت در HTML بازگردانده نمی‌شود.

## تنظیمات پنل CRM

در تب «بله / Safir»:

1. `فعال‌سازی کانال بله` را روشن کنید.
2. `Mirror پیامک‌ها` را روشن کنید.
3. برای OTP نیز `Mirror کدهای OTP` را روشن کنید.
4. Gateway URL باید HTTPS باشد.
5. Shared Secret باید حداقل ۳۲ کاراکتر باشد.
6. با «ارسال تست بله» یک شماره واقعی را آزمایش کنید.

## Eventهای پوشش‌داده‌شده

- `invoice.created`
- `invoice.payment_stage_issued`
- `invoice.payment_link_resent`
- `invoice.payment_completed`
- `customer.portal_invite`
- `customer.portal_otp`
- `dot.access`
- `dot.gift_access`
- `dot.payment`
- `dot.reject`
- `dot.followup`
- `dot.marketing_otp`
- `operations.upgrade_invoice`
- `operations.wallet_charged` (secure)
- `hr.login_credentials` (secure)
- `hr.password_reset` (secure)
- `payment.reward`
- `system.admin_test`

## دیتابیس

### `wp_sn_message_deliveries`

Delivery Log مستقل بله با `request_id` یکتا، `message_id`، status، attempt_count، error code و metadata غیرحساس. متن پیام، Password، OTP و Token در این جدول ذخیره نمی‌شوند.

### `wp_sn_bale_accounts`

اتصال حساب تأییدشده بله به شماره موبایل/مشتری CRM:

- `bale_user_id`
- `chat_id`
- `phone`
- `wp_user_id`
- `verified_at`
- `last_interaction_at`

## REST Inbound

Gateway با HMAC به این Endpointها درخواست می‌زند:

```text
POST /wp-json/sn/v1/bale/link
POST /wp-json/sn/v1/bale/status
```

Headerها:

```text
X-SN-Timestamp
X-SN-Nonce
X-SN-Signature
```

Canonical string:

```text
<timestamp>\n<nonce>\nPOST\n<path>\n<sha256(raw-body)>
```

Signature = HMAC-SHA256 با Shared Secret. اختلاف زمان بیش از ۵ دقیقه یا Nonce تکراری رد می‌شود.

در `/link`، Gateway فقط Contact خود کاربر بله را می‌پذیرد (`contact.user_id === from.id`) و بعد CRM شماره را با مشتری/فاکتور تطبیق می‌دهد.

## Idempotency

`SN_Notification_Service` برای هر ارسال Bale یک `request_id` قطعی تولید می‌کند. Gateway همان `request_id` را برای Safir می‌فرستد. ارسال موفق با همان request id دوباره از CRM ارسال نمی‌شود. ارسال مجدد دستی فاکتور، dedupe key مستقل دارد.

## رفتار خطا

- خطای بله Business Flow اصلی را Fail نمی‌کند.
- `NotBaleUser` به status `skipped` تبدیل می‌شود.
- خطاهای موقت / Rate Limit در Delivery Log با `retryable=1` مشخص می‌شوند.
- SMS همچنان مستقل از Bale کار می‌کند.

## Upgrade

نسخه پلاگین: `2.0.119`

در اولین درخواست ادمین بعد از Upgrade، `maybe_create_tables()` به‌دلیل تغییر `SN_VERSION` جدول‌های جدید را با `dbDelta()` ایجاد می‌کند. Activation تازه نیز همین جداول را می‌سازد.

## Acceptance Test

1. افزونه بدون Fatal/Syntax Error فعال شود.
2. تب بله / Safir باز شود و Secret قبلی را نمایش ندهد.
3. Test Bale از CRM، پیام را در بله تحویل دهد.
4. صدور فاکتور: SMS رفتار قبلی + Bale delivery log.
5. پرداخت مرحله‌ای و resend: Bale بدون duplicate ناخواسته.
6. OTP پرتال: SMS + Safir OTP.
7. `/start` در `@biawincrmbot`، ارسال Contact و Link واقعی CRM.
8. `/status` آخرین فاکتور متصل را نمایش دهد.
9. اطلاعات ورود/رمز در Delivery Log ذخیره نشده باشد.
10. با خاموش کردن Bale، تمام SMSها دقیقاً مانند قبل کار کنند.
