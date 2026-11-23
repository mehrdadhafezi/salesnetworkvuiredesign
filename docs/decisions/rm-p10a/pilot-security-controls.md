# RM-P10A / ۶ — کنترل‌های امنیتی پیش از پایلوت (اصلاح RM-P09-F01 تا F12)

**DRAFT — NOT APPROVED.** جدول اصلاح برای هر یافتهٔ RM-P09. هیچ تغییری اجرا نمی‌شود. «مالک فنی» به معنای مالک مهندسی است (طبق AGENTS.md، Claude)؛ مالک تصمیم کسب‌وکار در ستون «تصمیم لازم» آمده است. **GROUP B CLOSED — PRODUCTION NO-GO.**

منبع یافته‌ها: `docs/audits/rm-p09-identity-access-pilot-readiness.md` §۵.۲. ترتیب پیاده‌سازی به مراحل `docs/decisions/rm-p10a/rm-p10b-delivery-plan.md` اشاره دارد.

---

## ۱. جدول اصلاح

| یافته | مالک فنی | تصمیم لازم | قرارداد لازم | تغییر احتمالی Backend | تغییر احتمالی دیتابیس | تغییر احتمالی Frontend | تست مثبت | تست منفی | شرط پذیرش | ترتیب | وضعیت |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **F01** حساب به انسان تأییدشده وصل نیست | مهندسی (organization) | R5-SEC-01 (A)، BOOT، IQ-1، IQ-2 | فایل ۱ §۲ تا §۵ | فرمان‌های رجیستری W1 تا W11 + L-7/L-8؛ resolver | جدول‌های E1 تا E8 (append-only، exclusion) | بعداً PR-F (صفحهٔ مدیریت؛ نیازمند تصمیم Capability) | پیوند دو نفره `EFFECTIVE` می‌شود | خوداتصالی؛ پیوند هم‌پوشان؛ تطبیق نام؛ `UPDATE` | IT-1 تا IT-8 سبز؛ هیچ پیوندی از واردسازی | P10B-1، P10B-2 | `REQUIRES_APPROVAL` |
| **F02** نشست با شخص و همکاری تطبیق ندارد | مهندسی (identity) | R5-SEC-01؛ AP-4 (نگاشت همکاری، TTL) | فایل ۱ §۶ | `Login` و `Authenticate` به پذیرش وابسته؛ مصرف رویداد | جدول `identity.account_admission` | پیام عمومی شکست ورود (بدون تغییر رابط) | حساب `ADMITTED` وارد می‌شود | بدون پیوند، `SHARED_USER`، همکاری `UNMAPPED`، ردیف منقضی | T-ID-01، T-ID-02 سبز؛ نبود ردیف = رد | P10B-3 | `REQUIRES_APPROVAL` |
| **F03** خاتمه دسترسی را قطع نمی‌کند | مهندسی (identity + organization) | D-34 (P-3، P-5، P-6)؛ OPD-05/SD-04 برای خودکار، یا پذیرش فرمان دستی برای پایلوت | فایل ۱ §۸؛ فایل ۳ P-5، P-6 | فرمان ابطال اداری؛ فرمان ترکیبی خاتمه | افزودن `ADMIN_REVOKED` و `ADMISSION_REVOKED` به CHECK `session_revoke_reason` | — | فرمان، همهٔ نشست‌ها را باطل می‌کند | کنشگر بدون Capability؛ خاتمه بدون ابطال | T-ID-03 و T-SE-02: درخواست بعدی ۴۰۱؛ `SESSION_REVOKED` ثبت | P10B-4 | `REQUIRES_APPROVAL` |
| **F04** `login_enabled` در ورود بررسی نمی‌شود | مهندسی (identity) | D-34 (Q34-9) | فایل ۳ | ورود به پذیرش وابسته (نه به `login_enabled`)؛ فرمان تأمین تنها نویسنده | مهاجرت مجوز INSERT/UPDATE فقط برای نقش فرمان | — | حساب تأمین‌شده از مسیر مجاز وارد می‌شود | درج دستی DBA بدون پذیرش ← رد؛ پایش DP-5 هشدار می‌دهد | PT-1 تا PT-8 سبز | P10B-3، P10B-5 | `REQUIRES_APPROVAL` |
| **F05** حساب خدمت مدل نشده | مهندسی (identity) | HQ-4 (تعیین غیرشخصی)؛ قرارداد principal خدمت | فایل ۱ §۷ | `account_kind`؛ رد ورود انسانی برای `NON_PERSONAL` | ستون یا جدول `account_kind` (پیش‌فرض `UNKNOWN`) | — | حساب `PERSONAL` وارد می‌شود | حساب `NON_PERSONAL` یا `UNKNOWN` رد می‌شود | هیچ حساب بدون نوع وارد نمی‌شود | P10B-3 | `REQUIRES_APPROVAL` |
| **F06** فقط ALL؛ TEAM ناموجود | مهندسی (authorization + هر ماژول) | پیشنهاد ثبت ۱ و ۲ M2، D-32؛ برای HR: پذیرش ALL | فایل ۲ | مقادیر `SELF` و `TEAM` در ارزیاب؛ گزاره در کوئری هر خواندن | ایندکس برای گزاره‌های مالکیت (در صورت نیاز) | فقط حالت خالی / ۴۰۴؛ بدون فیلتر امنیتی سمت کاربر | فروشنده پروندهٔ خود را می‌بیند | N1 تا N8؛ SC-1 تا SC-8 | همهٔ endpointهای دادهٔ پرونده پوشش داده شوند؛ آزمون جهش | P10B-0 (فهرست و قفل)؛ P10B-8 | `REQUIRES_APPROVAL` |
| **F07** گزارش و شمارنده فقط سازمانی | مهندسی (reporting) | تصمیم دامنهٔ گزارش؛ یا محدودکردن Capability گزارش به ALL | فایل ۲ §۳.۷ | قاعدهٔ CR-2 و CR-3؛ بعداً گزارش محدود | — | حالت «در دسترس نیست» برای نقش محدود | دارندهٔ ALL گزارش را می‌بیند | Grant گزارش با SELF رد؛ endpoint گزارش با دامنهٔ SELF ← `ROW_SCOPE_DENIED` | T-AC-03 سبز | P10B-0 (آزمون قفل)؛ P10B-9 | `REQUIRES_APPROVAL` |
| **F08** OWN و SoD بر پایهٔ `user_id` | مهندسی (authorization + finance) | R5-SEC-01 | فایل ۱ §۴.۴؛ فایل ۲ §۳.۱ | رد SELF و SoD برای حساب `CONFLICT`؛ SoD با شخص | — | — | SoD با دو شخص متمایز عبور می‌کند | دو حساب یک شخص به‌عنوان سازنده و بازبین رد | T-ID-02 سبز | P10B-2، P10B-3 | `REQUIRES_APPROVAL` |
| **F09** کوئری پیش از بررسی دامنه در `getFulfillmentWorkCase` | مهندسی (fulfillment) | هیچ تصمیم کسب‌وکار؛ فقط بازبینی امنیتی مستقل | فایل ۲ DI-5 | جابه‌جایی بررسی دامنه پیش از کوئری جزئیات | — | — | کاربر درون دامنه جزئیات را می‌بیند | خارج از دامنه ۴۰۴ یکسان، **بدون** کوئری جزئیات (ثبت `QUERY_BEFORE_IDENTITY` در رجیستری برطرف شود) | آزمون رجیستری `TestNT22*` به‌روز و سبز | P10B-0b | `PROPOSED` |
| **F10** دید درخواست HR برای کنشگر سابق منقضی نمی‌شود | مهندسی (organization) | تصمیم HR دربارهٔ ماندگاری دید | فایل ۲ §۳.۴ | در صورت تصمیم: محدودکردن `visible` به دورهٔ مشارکت | احتمالاً ستون زمان مشارکت | — | مشارکت‌کنندهٔ فعلی می‌بیند | کنشگر سابق پس از جابه‌جایی (اگر تصمیم این باشد) نمی‌بیند | طبق تصمیم | پس از پایلوت | `REQUIRES_APPROVAL` |
| **F11** Staging ورود واقعی را پیکربندی نکرده | مهندسی (deploy) | BN-16؛ ADR #18 فقط برای دادهٔ واقعی | این فایل §۲ (ایزوله بودن) | — | DSNهای جدا برای نقش احراز و خواندن | — | ورود حساب مصنوعی روی Staging | ورود بدون پذیرش؛ stub در `CRM_ENV=production` | drill روی Staging + Runbook امضاشده | P10B-11 | `REQUIRES_APPROVAL` |
| **F12** ۴۰۳ و خواندن‌ها ممیزی پایدار ندارند | مهندسی (platform/audit) | D-45، D-23؛ طرح‌وارهٔ و نگه‌داری لاگ (Legal) | این فایل §۳ | ثبت رد دسترسی و خواندن حساس در `audit.audit_event` با `correlation_id` | احتمالاً بدون جدول جدید (بازاستفادهٔ `audit.audit_event`) | — | خواندن حساس ثبت می‌شود | شکست ثبت ← رفتار طبق D-45 (پیش‌فرض امن: fail closed)؛ هیچ PII در لاگ | T-LOG-01 سبز | P10B-10 | `REQUIRES_APPROVAL` |

## ۲. کنترل‌های ویژه

| # | کنترل | پیاده‌سازی پیشنهادی | آزمون پذیرش | یافته | وضعیت |
|---|---|---|---|---|---|
| SC-A | لغو نشست پس از خاتمهٔ همکاری | فایل ۱ §۸؛ فایل ۳ P-5، P-6 | خاتمه در میانهٔ نشست ← درخواست بعدی ۴۰۱؛ همهٔ نشست‌های همهٔ حساب‌های شخص باطل | F03 | `REQUIRES_APPROVAL` |
| SC-B | رد حساب با پیوند نامعتبر | پذیرش ورود AD-1 تا AD-7 | بدون پیوند، معلق، لغوشده، منقضی، مشترک ← رد در ورود و در هر درخواست | F01، F02 | `REQUIRES_APPROVAL` |
| SC-C | ممیزی خواندن حساس و پاسخ ۴۰۳ | رویداد `access.denied` و `access.sensitive_read` در `audit.audit_event`؛ فقط شناسه، Capability، عملیات، `correlation_id`؛ بدون مقدار فیلد | ۴۰۳ ← یک رویداد؛ خواندن حساس ← یک رویداد؛ شکست ثبت ← fail closed (اگر D-45 همین باشد) | F12 | `REQUIRES_APPROVAL` |
| SC-D | جداسازی SELF و TEAM | فایل ۲ | SC-1 تا SC-3 | F06 | `REQUIRES_APPROVAL` |
| SC-E | محدودسازی گزارش‌ها و شمارنده‌ها | CR-2، CR-3 (موقت)؛ CR-4 (بعداً) | SC-4؛ T-AC-03 | F07 | `PROPOSED` (CR-2/CR-3)؛ CR-4 `REQUIRES_APPROVAL` |
| SC-F | جلوگیری از نشت با Direct URL | DI-1 تا DI-5 | شناسهٔ پروندهٔ دیگری ← ۴۰۴ با بدنهٔ برابر بایت‌به‌بایت با رکورد ناموجود | F06، F09 | `PROPOSED` |
| SC-G | ایزوله بودن Staging | §۳ | بند ۳ | F11 | `REQUIRES_APPROVAL` |
| SC-H | توقف اضطراری پایلوت | §۴ | بند ۴ | PC-12 | `REQUIRES_APPROVAL` |

## ۳. ایزوله بودن Staging (SC-G)

| # | الزام | شاهد موجود | وضعیت |
|---|---|---|---|
| ST-1 | `CRM_ENV=production` در Staging (بدون dev stub) | `deploy/staging/compose.yaml:105` | `EXISTING_VERIFIED` |
| ST-2 | شبکهٔ داخلی compose؛ DSN جدا برای هر نقش ماژول | `compose.yaml` (`networks: [internal]`)؛ `TestModuleLoginsBoundToAttestedDatabase` | `EXISTING_VERIFIED` |
| ST-3 | بستهٔ غیرمصنوعی رد می‌شود | `internal/integration/legacy/staging/staging.go:119` (`BlockedNonSyntheticMessage`) | `EXISTING_VERIFIED` |
| ST-4 | DSN احراز و خواندن فقط به DB Staging؛ attestation محیط | پیکربندی F11 | `REQUIRES_APPROVAL` |
| ST-5 | هیچ DSN، راز یا اعتبارنامه‌ای از Production در Staging | Runbook + بررسی پیکربندی | `PROPOSED` |
| ST-6 | هیچ writeback یا اتصال به CRM قدیمی از Staging | پرچم‌های نوشتن OFF (PC-10) | `EXISTING_VERIFIED` (پیش‌فرض‌ها) |

## ۴. توقف اضطراری پایلوت (SC-H)

| سطح | اقدام | اثر | زمان مورد انتظار | وضعیت |
|---|---|---|---|---|
| K-1 | خاموش‌کردن `identity.session_auth` | هیچ نشستی پذیرفته نمی‌شود؛ ورود `BLOCKED` | درخواست بعدی | `EXISTING_VERIFIED` (`session.go:279-281`) |
| K-2 | خاموش‌کردن پرچم‌های خواندن ماژول | خواندن‌ها `BLOCKED / FEATURE_DISABLED` | درخواست بعدی | `EXISTING_VERIFIED` |
| K-3 | لغو اضطراری Grantها (G-6) | ۴۰۳ برای همه | درخواست بعدی | `REQUIRES_APPROVAL` |
| K-4 | ابطال همهٔ نشست‌ها (P-5) و غیرفعال‌سازی اعتبارنامه‌ها (P-3) | ۴۰۱ | درخواست بعدی | `REQUIRES_APPROVAL` |
| K-5 | Runbook امضاشده: چه کسی، با کدام کانال، با چه ثبت | E-7 (یک نفر برای خاموش‌کردن) | — | `REQUIRES_APPROVAL` |
| K-6 | drill روی Staging پیش از پایلوت | گزارش drill | — | `REQUIRES_APPROVAL` |

**GROUP B CLOSED — PRODUCTION NO-GO.**
