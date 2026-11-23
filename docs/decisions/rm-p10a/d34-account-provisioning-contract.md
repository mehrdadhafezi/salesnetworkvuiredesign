# RM-P10A / ۳ — قرارداد تأمین حساب، اعتبارنامه و ابطال (D-34)

**DRAFT — NOT APPROVED.** فقط قرارداد؛ هیچ حساب، اعتبارنامه، راز، مهاجرت یا API ایجاد نمی‌شود. D-34 در ردیاب `READY_FOR_REVIEW` است و `approval_record` آن خالی است. **GROUP B CLOSED — PRODUCTION NO-GO.**

بستهٔ اصلی: `docs/decisions/rm-p10a-identity-access-closure-package.md`. این قرارداد S1 در `docs/remediation/m2-engineering-readiness.md` را دقیق می‌کند و یافته‌های F03 و F04 را پوشش می‌دهد.

---

## ۱. وضعیت امروز (شاهد)

| موضوع | امروز | شاهد | وضعیت |
|---|---|---|---|
| نوشتن اعتبارنامه | نقش برنامه فقط SELECT دارد؛ صفر ردیف | `internal/identity/migrations/sql/00012_identity_authentication.sql:87`؛ PRB-S2 | `EXISTING_VERIFIED` |
| یکتایی | `credential.user_id UNIQUE`؛ Argon2id؛ `status ∈ {ACTIVE, DISABLED}`؛ نام ورود شبیه تلفن ممنوع | همان، `:22-40` | `EXISTING_VERIFIED` |
| lint | `r1-credential-write` نوشتن اعتبارنامه را ممنوع می‌کند | `internal/architecture/lint.go:325` | `EXISTING_VERIFIED` |
| ورود و `login_enabled` | ورود به `login_enabled` و `account_state` نگاه نمی‌کند (F04) | PRB-S3 | `EXISTING_VERIFIED` |
| اثر `DISABLED` | درخواست بعدی رد می‌شود | `session.go:289-292`؛ `session_test.go` | `EXISTING_VERIFIED` |
| ابطال اداری نشست | وجود ندارد | `00012:60,80` | `EXISTING_VERIFIED` |
| تعارض C-04 | پیش‌فرض مستند («رمز تصادفی + جریان بازنشانی») پیاده‌سازی نشده؛ باز | `docs/remediation/r5-decision-conflict-register.md` ردیف C-04 | `EXISTING_VERIFIED` |

## ۲. آنچه D-34 باید تصمیم بگیرد

| # | پرسش | گزینه‌ها / پیش‌فرض امن | وضعیت |
|---|---|---|---|
| Q34-1 | روش ایجاد اعتبارنامه | (۱) رمز یک‌بارمصرف صادرشده توسط مدیر + تغییر اجباری؛ (۲) پیوند دعوت با انقضا؛ (۳) IdP خارجی (= R5-SEC-01 B). با R5 = A، فقط ۱ یا ۲ سازگار است | `REQUIRES_APPROVAL` |
| Q34-2 | مرجع ایجاد حساب (چه جایگاهی فرمان را اجرا می‌کند) | پیشنهاد: اپراتور Security، با درخواست HR؛ دو نفر متمایز | `REQUIRES_APPROVAL` |
| Q34-3 | کانال تحویل راز | پیش‌فرض امن: تحویل حضوری یا کانال سازمانی جدا از CRM. اگر تلفن کانال باشد، تلفن فقط مقصد تحویل است و هرگز شناسه نمی‌شود | `REQUIRES_APPROVAL` |
| Q34-4 | مدت انقضای راز اولیه یا دعوت | پیش‌فرض امن: کوتاه (مثلاً ۲۴ ساعت) و یک‌بارمصرف | `REQUIRES_APPROVAL` |
| Q34-5 | چند حساب فعال برای یک شخص | پیش‌فرض امن: حداکثر یک اعتبارنامهٔ `ACTIVE` برای هر شخص تأییدشده؛ بیشتر = `CONFLICT` | `REQUIRES_APPROVAL` |
| Q34-6 | مرجع غیرفعال‌سازی و فعال‌سازی مجدد | پیشنهاد: غیرفعال‌سازی توسط هر اپراتور HR یا Security (یک نفر کافی، چون محدودکننده است)؛ فعال‌سازی مجدد دو نفر | `REQUIRES_APPROVAL` |
| Q34-7 | مرجع ابطال اداری نشست | پیشنهاد: اپراتور Security؛ همیشه مجاز (محدودکننده) | `REQUIRES_APPROVAL` |
| Q34-8 | آیا فرمان دستی خاتمه برای دورهٔ پایلوت جای OPD-05 را می‌گیرد؟ | فقط برای پایلوت؛ OPD-05 / SD-04 برای تولید | `REQUIRES_APPROVAL` |
| Q34-9 | سرنوشت `login_enabled` و `SHADOW_IMPORTED` | پیشنهاد: ردیف‌های سایه دست نمی‌خورند (KEEP)؛ ورود به پذیرش (فایل ۱ §۶) وابسته می‌شود، نه به این ستون‌ها | `REQUIRES_APPROVAL` |

## ۳. پیش‌شرط‌های مشترک همهٔ فرمان‌ها

| # | پیش‌شرط | نقض ← خطا | وضعیت |
|---|---|---|---|
| PP-1 | D-34 و R5-SEC-01 (A یا B) هر دو `APPROVED_VERIFIED` | `BLOCKED(DECISION_NOT_APPROVED)` | `REQUIRES_APPROVAL` |
| PP-2 | پرچم فرمان روشن است (پیش‌فرض خاموش) | `BLOCKED(FEATURE_DISABLED)` | `PROPOSED` |
| PP-3 | کنشگر نشست معتبر، `ADMITTED`، با Capability فرمان، و `resolvePerson(actor, now, GRANT_ACTOR)` = `VERIFIED` | `UNAUTHORIZED` | `PROPOSED` |
| PP-4 | کنشگر ≠ شخص موضوع (در سطح شخص) | `UNAUTHORIZED(SELF_ACTION)` | `PROPOSED` |
| PP-5 | `Idempotency-Key` الزامی؛ تکرار با همان بدنه همان نتیجه؛ با بدنهٔ متفاوت `CONFLICT` | `CONFLICT(IDEMPOTENCY_MISMATCH)` | `PROPOSED` (الگوی `internal/platform/idempotency`) |
| PP-6 | هر نتیجه یک `auth_event` و یک `audit.audit_event` بدون راز و بدون PII | — | `PROPOSED` |
| PP-7 | هیچ ورودی تلفن، ایمیل یا نام به‌عنوان شناسه؛ ورودی فقط `user_id` | ۴۰۰ | `PROPOSED` (lint `phoneIdent`) |
| PP-8 | نتیجهٔ نامعلوم (مثلاً قطع اتصال پس از commit) پیش از تکرار کور با کلید idempotency تطبیق داده می‌شود | `UNKNOWN` | `PROPOSED` (AGENTS.md §Sensitive Outcomes) |

## ۴. فرمان‌های پیشنهادی

### P-1 `ProvisionAccountCredential`

| مورد | قرارداد |
|---|---|
| ورودی | `user_id`، `method` (طبق گزینهٔ D-34)، `request_ref` (درخواست HR)، `Idempotency-Key` |
| پیش‌شرط خاص | `resolvePerson(user_id, now, ACCOUNT_PROVISIONING) = VERIFIED`؛ پذیرش `ADMITTED`؛ `account_kind = PERSONAL`؛ اعتبارنامه‌ای برای این حساب وجود ندارد؛ سیاست Q34-5 |
| خروجی | `SUCCESS { credential_id, status: PENDING_FIRST_USE, expires_at }`؛ راز فقط یک بار و فقط از کانال Q34-3 |
| خطا | `BLOCKED(PERSON_LINK_UNVERIFIED / ACCOUNT_NOT_ADMITTED / NON_PERSONAL_ACCOUNT)`، `CONFLICT(CREDENTIAL_EXISTS / PERSON_HAS_ACTIVE_CREDENTIAL)`، `UNAUTHORIZED`، `UNKNOWN` |
| اثر امنیتی | اولین مسیر مجاز نوشتن اعتبارنامه. نیاز به مهاجرت جدا که INSERT/UPDATE را **فقط** به نقش این فرمان بدهد و استثنای lint `r1-credential-write` فقط برای همین بسته |
| رویداد | `CREDENTIAL_PROVISIONED` (نوع جدید در CHECK `auth_event`) |
| وضعیت | `REQUIRES_APPROVAL` |

**دربارهٔ وضعیت `PENDING_FIRST_USE`:** امروز `status` فقط `ACTIVE` یا `DISABLED` است. افزودن مقدار جدید یا نگه‌داشتن آن در جدول جدا تصمیم Security است. وضعیت: `REQUIRES_APPROVAL`

### P-2 `CompleteFirstUse` (تغییر اجباری رمز یا پذیرش دعوت)

| مورد | قرارداد |
|---|---|
| ورودی | راز یک‌بارمصرف یا توکن دعوت، رمز جدید |
| پیش‌شرط | راز منقضی نشده و استفاده نشده؛ پذیرش `ADMITTED` در همان لحظه |
| خروجی | `SUCCESS`؛ اعتبارنامه `ACTIVE`؛ **بدون** صدور خودکار نشست (کاربر باید وارد شود) |
| خطا | `UNAUTHORIZED` (عمومی، بدون افشای دلیل)؛ throttle موجود اعمال می‌شود |
| اثر امنیتی | راز اولیه باطل؛ `auth_event` |
| وضعیت | `REQUIRES_APPROVAL` |

### P-3 `DisableCredential`

| مورد | قرارداد |
|---|---|
| ورودی | `user_id`، `reason` (`TERMINATION`، `SECURITY_HOLD`، `LINK_REVOKED`، `REQUESTED_BY_HR`)، `Idempotency-Key` |
| پیش‌شرط | PP-3، PP-4. **بدون** نیاز به پیوند `VERIFIED` موضوع (محدودکننده همیشه مجاز است) |
| خروجی | `SUCCESS`؛ اگر قبلاً `DISABLED`: `SUCCESS` بدون تغییر (idempotent) |
| خطا | `UNAUTHORIZED`، `UNKNOWN` |
| اثر امنیتی | `status = DISABLED`؛ در همان تراکنش P-5 برای همهٔ نشست‌ها |
| رویداد | `CREDENTIAL_DISABLED`، `SESSION_REVOKED` |
| وضعیت | `REQUIRES_APPROVAL` |

### P-4 `ReenableCredential`

| مورد | قرارداد |
|---|---|
| ورودی | `user_id`، `reason`، `approver_ref` |
| پیش‌شرط | دو نفر متمایز (درخواست‌کننده ≠ تأییدکننده ≠ موضوع)؛ پیوند `VERIFIED`؛ پذیرش `ADMITTED`؛ دلیل غیرفعال‌سازی `TERMINATION` نباشد (بعد از خاتمه فقط حساب جدید) |
| خروجی | `SUCCESS`؛ نشست قبلی زنده نمی‌شود |
| خطا | `BLOCKED(TERMINATED)`، `UNAUTHORIZED`، `CONFLICT` |
| وضعیت | `REQUIRES_APPROVAL` |

### P-5 `RevokeAllSessionsForAccount` (ابطال اداری)

| مورد | قرارداد |
|---|---|
| ورودی | `user_id`، `reason` (`ADMIN_REVOKED`، `ADMISSION_REVOKED`، `CREDENTIAL_DISABLED`) |
| پیش‌شرط | PP-3. سیستم هم از رویداد پذیرش می‌تواند فراخوانی کند |
| خروجی | `SUCCESS { revoked_count }` |
| خطا | `UNAUTHORIZED`، `UNKNOWN` |
| اثر امنیتی | `revoked_at` و `revoke_reason` برای همهٔ نشست‌های باز؛ درخواست بعدی ۴۰۱. نیاز به افزودن دو دلیل به CHECK `session_revoke_reason` (`00012:60`) |
| رویداد | `SESSION_REVOKED` (نوع موجود بدون نویسنده) |
| وضعیت | `REQUIRES_APPROVAL` |

### P-6 `DeactivateAccountForTermination` (فرمان ترکیبی پایلوت)

| مورد | قرارداد |
|---|---|
| ورودی | `user_id`، `termination_ref`، `effective_at` |
| پیش‌شرط | PP-3؛ کنشگر HR یا Security؛ پایلوت فقط (Q34-8) |
| خروجی | `SUCCESS { link_revoked, credential_disabled, sessions_revoked }` یا `PARTIAL` با فهرست گام‌های ناموفق |
| خطا | `PARTIAL` هرگز بی‌صدا نیست؛ گام باقی‌مانده تکرارپذیر است |
| اثر امنیتی | TM-2 تا TM-5 فایل ۱ به ترتیب: لغو پیوند ← پذیرش `NOT_ADMITTED` ← P-3 ← P-5 |
| وضعیت | `REQUIRES_APPROVAL` |

## ۵. جلوگیری از تکرار

| کنترل | قرارداد | وضعیت |
|---|---|---|
| DP-1 | یک اعتبارنامه برای هر حساب (`UNIQUE` موجود) | `EXISTING_VERIFIED` |
| DP-2 | سیاست یک اعتبارنامهٔ فعال برای هر شخص (Q34-5) | `REQUIRES_APPROVAL` |
| DP-3 | `Idempotency-Key` با رکورد idempotency ماژول | `PROPOSED` |
| DP-4 | نام ورود یکتا، بدون شباهت تلفن (CHECK موجود) | `EXISTING_VERIFIED` |
| DP-5 | پایش: شمارش ردیف‌های `credential` بدون رویداد `CREDENTIAL_PROVISIONED` متناظر = هشدار (درج دستی DBA؛ F04) | `PROPOSED` |

## ۶. ثبت تاریخچه

| رویداد | محل | فیلدها | وضعیت |
|---|---|---|---|
| `CREDENTIAL_PROVISIONED`، `CREDENTIAL_FIRST_USE_COMPLETED`، `CREDENTIAL_DISABLED`، `CREDENTIAL_REENABLED` | `identity.auth_event` (افزودن نوع به CHECK؛ مهاجرت) | `user_id`، کنشگر، شخص کنشگر، دلیل، `correlation_id`؛ بدون راز | `REQUIRES_APPROVAL` |
| `SESSION_REVOKED` | `identity.auth_event` (نوع موجود) | `user_id`، `session_id`، دلیل | `PROPOSED` |
| همهٔ فرمان‌ها | `audit.audit_event` | `subject_domain = 'identity'` | `PROPOSED` |

## ۷. آزمون‌ها

| # | آزمون | انتظار | وضعیت |
|---|---|---|---|
| PT-1 | تأمین برای حساب بدون پیوند `VERIFIED` | `BLOCKED(PERSON_LINK_UNVERIFIED)`؛ هیچ ردیف | `PROPOSED` |
| PT-2 | تأمین در حالی که R5-SEC-01 تصویب نشده | `BLOCKED(DECISION_NOT_APPROVED)` | `PROPOSED` |
| PT-3 | تکرار با همان کلید | همان نتیجه؛ یک ردیف | `PROPOSED` |
| PT-4 | کنشگر = موضوع از حساب دیگرش | `UNAUTHORIZED(SELF_ACTION)` | `PROPOSED` |
| PT-5 | غیرفعال‌سازی در میانهٔ نشست | درخواست بعدی ۴۰۱؛ `SESSION_REVOKED` | `PROPOSED` |
| PT-6 | فعال‌سازی مجدد پس از `TERMINATION` | `BLOCKED(TERMINATED)` | `PROPOSED` |
| PT-7 | درج مستقیم SQL با نقش برنامه | `permission denied` | `EXISTING_VERIFIED` (PRB-S2) |
| PT-8 | ورود با اعتبارنامهٔ `ACTIVE` ولی پذیرش `NOT_ADMITTED` | رد | `PROPOSED` |

**GROUP B CLOSED — PRODUCTION NO-GO.**
