# Production Checklist — Bale Gateway + CRM 2.0.119

## 1. Gateway live state

روی سرور Gateway، Mock تستی باید خاموش باشد:

```env
CRM_MOCK_MODE=false
CRM_BASE_URL=https://YOUR-FINAL-CRM-DOMAIN
CRM_SHARED_SECRET=<same 32+ char secret used by CRM>
BALE_BOT_ID=197632729
BALE_BOT_TOKEN=<server secret>
BALE_SAFIR_API_KEY=<server secret>
APP_BASE_URL=https://bale.andishesazanco.ir
```

بعد از تغییر `.env`:

```bash
curl -i https://bale.andishesazanco.ir/health
php /home/andishesazanco/bale-crm-gateway/bin/get-me.php
php /home/andishesazanco/bale-crm-gateway/bin/webhook-info.php
```

## 2. CRM secret

روش ترجیحی در `wp-config.php`:

```php
define('SN_BALE_GATEWAY_URL', 'https://bale.andishesazanco.ir');
define('SN_BALE_GATEWAY_SHARED_SECRET', '<same CRM_SHARED_SECRET>');
```

Secret را در چت، Git، لاگ یا Screenshot قرار ندهید.

## 3. Enable in CRM

تنظیمات شبکه فروش → بله / Safir:

- فعال‌سازی کانال بله = روشن
- Mirror پیامک‌ها = روشن
- Mirror OTP = روشن (بعد از تست)
- تست ارسال بله = PASS

## 4. Inbound bot

در `@biawincrmbot`:

1. `/start`
2. «ارسال شماره موبایل من»
3. اتصال باید به مشتری واقعی CRM انجام شود (نه «حساب آزمایشی CRM»).
4. `/status` باید وضعیت واقعی آخرین فاکتور را برگرداند.

اگر هنوز پاسخ Mock دریافت می‌شود، Gateway هنوز `CRM_MOCK_MODE=true` است یا `CRM_BASE_URL` به CRM نهایی اشاره نمی‌کند.

## 5. Outbound Safir acceptance

- تست ادمین CRM به شماره دارای بله → Sent
- شماره بدون حساب بله → `skipped / NotBaleUser`
- `wp_sn_message_deliveries` باید request_id و status داشته باشد ولی متن پیام/OTP/Password نداشته باشد.

## 6. Regression

با بله خاموش:

- صدور فاکتور
- پرداخت مرحله‌ای
- ارسال مجدد لینک
- Dot access/payment/reject/followup
- Customer Portal OTP/invite
- HR password/login credentials
- Operations wallet/upgrade
- Payment reward

باید دقیقاً مانند 2.0.117 از مسیر SMS فعلی کار کنند.
