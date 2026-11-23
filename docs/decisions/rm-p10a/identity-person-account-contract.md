# RM-P10A / ۱ — قرارداد هویت شخص و حساب (Option A)

**DRAFT — NOT APPROVED.** قرارداد فنی آمادهٔ پیاده‌سازی برای *زمانی* که R5-SEC-01 با گزینهٔ A تصویب و راستی‌آزمایی شود. هیچ مهاجرت، کد اجرایی، API یا دادهٔ واقعی در این فایل تولید نمی‌شود. **GROUP B CLOSED — PRODUCTION NO-GO.**

بستهٔ اصلی: `docs/decisions/rm-p10a-identity-access-closure-package.md`. این فایل `docs/security/rm-e04i-option-a-implementation-contract.md` (E1 تا E7، resolver، W1 تا W11، RC-1 تا RC-9) را **بازاستفاده** می‌کند و فقط شکاف‌هایی را می‌بندد که RM-P09 نشان داد: F01، F02، F03، F04، F05 و F08.

---

## ۱. وضعیت امروز (شاهد)

| حلقه | امروز | شاهد | وضعیت |
|---|---|---|---|
| شخص | `organization.person` یک ردیف به ازای هر پروفایل HR قدیمی است، نه انسان تأییدشده | `internal/organization/migrations/sql/00001_organization.sql:194-212` | `EXISTING_VERIFIED` |
| پیوند حساب | ستون `employment.user_id` بدون FK و `user_link_state` با ۴ مقدار؛ بدون تاریخچه، بازبینی و لغو | همان، `:215-231,263-264`؛ `internal/organization/imports/imports.go:172-186` | `EXISTING_VERIFIED` |
| همکاری | `employment_status` با ۸ مقدار؛ یک ردیف برای هر شخص؛ با هر واردسازی بازنویسی می‌شود | همان، `:233-234,265` | `EXISTING_VERIFIED` |
| حساب | `SHADOW_IMPORTED`، `login_enabled = false` با CHECK؛ نوع انسانی/خدمت ندارد | `internal/identity/migrations/sql/00001_identity.sql:146-170` | `EXISTING_VERIFIED` |
| احراز نشست | فقط digest، انقضا، ابطال، `c.status = 'ACTIVE'` و `u.presence = 'PRESENT'` | `internal/identity/session/session.go:284-292` | `EXISTING_VERIFIED` |
| دلایل ابطال نشست | `LOGOUT`، `LOGOUT_ALL`، `REPLACED_AT_LOGIN`، `CREDENTIAL_DISABLED`، `ACCOUNT_NOT_PRESENT`؛ ابطال اداری ندارد | `00012_identity_authentication.sql:60`؛ نوع رویداد `SESSION_REVOKED` نویسنده ندارد (`:80`) | `EXISTING_VERIFIED` |
| Grant در هر درخواست | دوباره خوانده می‌شود | `internal/identity/query/query.go:155-164` | `EXISTING_VERIFIED` |
| اعمال نهایی HR | `HR_APPLY_IMPLEMENTATION_BLOCKED` | `internal/organization/hrrequest/hrrequest.go:62` | `EXISTING_VERIFIED` |

---

## ۲. Person Registry معتبر

| الزام | قرارداد | مرجع | وضعیت |
|---|---|---|---|
| مالک ماژول | organization (زیرحوزهٔ «رجیستری شخص»). Security مالک *سیاست* پیوند است، نه ماژول | `rm-e04i…` §4.1؛ IQ-1 | `REQUIRES_APPROVAL` |
| ساختارها | E1 `VerifiedPerson`، E2 `PersonIdentityEvidence`، E3 `IdentityVerification`، E4 `VerifiedPersonAccountLink`، E5 `PersonLinkReview`، E6 `IdentityConflictFinding`، E7 `IdentityLinkRevocation` — بدون تغییر | `rm-e04i…` §4.2 | `PROPOSED` |
| افزوده‌های این بسته | E8 `IdentityLinkSuspension` (§۴)؛ پروجکشن `identity.account_admission` (§۶)؛ ستون `account_kind` روی پیوند و رجیستر حساب‌های غیرشخصی (§۷) | این فایل | `PROPOSED` |
| حریم خصوصی | هیچ نام، شمارهٔ سند، تلفن یا ایمیل در ستون‌ها یا کلیدها. مدرک فقط ارجاع + SHA-256 | `rm-e04i…` §4.2 E2؛ HQ-5 | `REQUIRES_APPROVAL` |
| KEEP + ADAPT | `organization.person`، `employment` و `user_link_state` بدون تغییر می‌مانند و فقط «نامزد» و «شاهد» هستند | `rm-e04i…` §3.1 | `PROPOSED` |

## ۳. شناسهٔ پایدار شخص

| قاعده | قرارداد | وضعیت |
|---|---|---|
| SP-1 | `verified_person_id uuid` یک بار و فقط توسط یک E3 با نتیجهٔ `VERIFIED` ساخته می‌شود | `PROPOSED` |
| SP-2 | هرگز حذف، دوباره استفاده یا به شخص دیگری منتقل نمی‌شود. ادغام = tombstone (`MERGED_INTO`) | `PROPOSED` |
| SP-3 | با `organization.person.person_id` و `identity.user_account.user_id` برابر نیست و از آن‌ها مشتق نمی‌شود | `PROPOSED` |
| SP-4 | انتقال، تغییر سمت، تغییر نام یا تغییر حساب `verified_person_id` را تغییر نمی‌دهد | `PROPOSED` |
| SP-5 | چند حساب یک شخص به یک `verified_person_id` می‌رسند. هر قاعدهٔ تفکیک وظایف «شخص» می‌شمارد | `PROPOSED` |

### ماشین وضعیت `VerifiedPerson`

| از | به | مجاز؟ | شرط / دلیل | وضعیت |
|---|---|---|---|---|
| — | `ACTIVE` | مجاز | E3 با `VERIFIED`؛ بازبین ≠ موضوع | `PROPOSED` |
| `ACTIVE` | `MERGED_INTO` | مجاز | E6 `DUPLICATE_PERSON` حل‌شده با دو نفر (W11)؛ مقصد `ACTIVE` | `PROPOSED` |
| `ACTIVE` | `SPLIT_SOURCE` | مجاز | E6 `SPLIT_PERSON` حل‌شده؛ اشخاص جدید با E3 جدا | `PROPOSED` |
| `ACTIVE` | `RETIRED` | مجاز | فقط وقتی هیچ پیوند `EFFECTIVE` یا `SUSPENDED` ندارد؛ دلیل ثبت‌شده | `REQUIRES_APPROVAL` (قاعدهٔ بازنشستگی، HQ-7) |
| `MERGED_INTO` / `SPLIT_SOURCE` / `RETIRED` | هر حالت دیگر | **ممنوع** | حالت‌های پایانی؛ اصلاح فقط با رویداد جبرانی جدید (مثلاً split پس از merge) | `PROPOSED` |
| هر حالت | حذف ردیف | **ممنوع** | append-only | `PROPOSED` |

---

## ۴. اتصال حساب به شخص و چرخهٔ عمر پیوند

### ۴.۱ حالات

| حالت | معنی | مجوز ورود؟ | وضعیت |
|---|---|---|---|
| `PROPOSED` | سازندهٔ HR نامزد (شخص + حساب) را ثبت کرد | خیر | `PROPOSED` |
| `UNDER_REVIEW` | مدرک کامل است و منتظر بازبین مستقل | خیر | `PROPOSED` |
| `VERIFIED` | بازبین تأیید کرد؛ هنوز `valid_from` نرسیده | خیر | `PROPOSED` |
| `EFFECTIVE` | مؤثر است | بله (همراه سایر شرط‌های پذیرش §۶) | `PROPOSED` |
| `SUSPENDED` | **افزودهٔ این بسته.** موقتاً بی‌اثر (مثلاً تعلیق همکاری، ظن تقلب، بررسی تعارض)؛ قابل بازگشت | خیر | `REQUIRES_APPROVAL` (IQ-2) |
| `REVOKED` | لغو دائم؛ پایانی | خیر | `PROPOSED` |
| `EXPIRED` | پایان `valid_to` یا انقضای مدرک؛ پایانی | خیر | `PROPOSED` |
| `REJECTED` | بازبین رد کرد؛ پایانی | خیر | `PROPOSED` |
| `SUPERSEDED` | پیوند جدید جایگزین شد (مثلاً اصلاح مدرک پیش از مؤثرشدن)؛ پایانی | خیر | `PROPOSED` |
| نگه‌داشت تعارض (لایه) | E6 باز روی حساب یا شخص؛ روی هر حالت اعمال می‌شود | خیر؛ resolver `CONFLICT` | `PROPOSED` |

### ۴.۲ جدول انتقال‌های مجاز

| # | از | به | کنشگر | پیش‌شرط | تفکیک وظایف (در سطح شخص) | رویداد ممیزی | وضعیت |
|---|---|---|---|---|---|---|---|
| L-1 | — | `PROPOSED` | سازندهٔ HR | سازنده `VERIFIED` و `ADMITTED`؛ حساب `PERSONAL`؛ بدون E6 باز | سازنده ≠ موضوع | `organization.person_registry.link_proposed` | `PROPOSED` |
| L-2 | `PROPOSED` | `UNDER_REVIEW` | سیستم پس از ثبت مدرک | E2 با ارجاع و hash | — | `…evidence_submitted` | `PROPOSED` |
| L-3 | `UNDER_REVIEW` | `VERIFIED` | بازبین مستقل | hash مجدد مدرک برابر؛ روش طبق HQ-2 | بازبین ∉ {سازنده، موضوع}؛ بازبین `VERIFIED`؛ هرگز اتوماسیون | `…link_review_approved` | `PROPOSED` |
| L-4 | `UNDER_REVIEW` | `REJECTED` | بازبین مستقل | دلیل | مانند L-3 | `…link_review_rejected` | `PROPOSED` |
| L-5 | `UNDER_REVIEW` | `PROPOSED` | بازبین ← سازنده | درخواست اصلاح | مانند L-3 | `…link_correction_requested` | `PROPOSED` |
| L-6 | `VERIFIED` | `EFFECTIVE` | سیستم در `valid_from` | قید exclusion: پیوند `EFFECTIVE` هم‌پوشان برای همین حساب وجود ندارد | — | `…link_effective` | `PROPOSED` |
| L-7 | `EFFECTIVE` | `SUSPENDED` | تعلیق‌کنندهٔ HR یا Security | دلیل از فهرست بسته (`EMPLOYMENT_SUSPENDED`، `FRAUD_SUSPECTED`، `CONFLICT_REVIEW`، `SECURITY_HOLD`) | تعلیق‌کننده ≠ موضوع | `…link_suspended` | `REQUIRES_APPROVAL` |
| L-8 | `SUSPENDED` | `EFFECTIVE` | بازگرداننده + بازبین مستقل | علت تعلیق رفع شده؛ مدرک هنوز معتبر؛ بدون E6 باز | بازگرداننده ≠ تعلیق‌کننده ≠ موضوع | `…link_reinstated` | `REQUIRES_APPROVAL` |
| L-9 | `EFFECTIVE` یا `SUSPENDED` | `REVOKED` | لغوکنندهٔ HR یا Security | E7 با دلیل (`TERMINATION`، `REASSIGNMENT`، `EVIDENCE_REVOKED`، `FRAUD_SUSPECTED`، `CONFLICT_RESOLUTION`، `CORRECTION`) | لغوکننده ≠ موضوع | `…link_revoked` | `PROPOSED` |
| L-10 | `EFFECTIVE` یا `SUSPENDED` | `EXPIRED` | سیستم | رسیدن `valid_to` یا انقضای مدرک | — | `…link_expired` | `PROPOSED` |
| L-11 | `PROPOSED` / `UNDER_REVIEW` / `VERIFIED` | `SUPERSEDED` | سازنده | پیوند جایگزین ثبت شد | — | `…link_superseded` | `PROPOSED` |
| L-12 | هر حالت غیرپایانی | + نگه‌داشت تعارض | هر شرکت‌کننده یا آشکارساز | E6 | بازکننده نمی‌تواند حل کند | `…conflict_opened` | `PROPOSED` |

### ۴.۳ جدول انتقال‌های ممنوع

| # | انتقال | دلیل | کنترل پیشنهادی | وضعیت |
|---|---|---|---|---|
| X-1 | `PROPOSED` یا `UNDER_REVIEW` → `EFFECTIVE` (بدون بازبینی) | دور زدن قاعدهٔ دو نفر | تریگر DB + فرمان | `PROPOSED` |
| X-2 | `REVOKED` / `EXPIRED` / `REJECTED` / `SUPERSEDED` → هر حالت | پایانی؛ بازگشت فقط با پیوند جدید L-1 تا L-6 | تریگر append-only | `PROPOSED` |
| X-3 | `SUSPENDED` → `EFFECTIVE` توسط همان تعلیق‌کننده یا موضوع | تفکیک وظایف | فرمان + آزمون منفی | `REQUIRES_APPROVAL` |
| X-4 | هر انتقال با کنشگر `UNKNOWN`، `CONFLICT` یا `BLOCKED` | کنشگر باید شخص تأییدشده باشد | resolver با purpose ثبت پیوند | `PROPOSED` |
| X-5 | ساخت پیوند از واردسازی، تطبیق نام، تلفن، ایمیل یا نام کاربری | AGENTS.md §Identity؛ P16 | lint `phoneIdent`؛ نبود ورودی نام در فرمان | `PROPOSED` |
| X-6 | پیوند حساب `NON_PERSONAL` یا `SHARED_USER` | حساب مشترک = `CONFLICT` | CHECK روی `account_kind`؛ resolver گام ۴ و ۵ | `PROPOSED` |
| X-7 | دو پیوند `EFFECTIVE` هم‌پوشان برای یک حساب | اتصال متعارض | exclusion constraint روی `tstzrange` | `PROPOSED` |
| X-8 | `UPDATE` ستون‌های کسب‌وکار، `DELETE` یا `TRUNCATE` | append-only | تریگر، مانند `audit.audit_event` | `PROPOSED` |
| X-9 | پیوند با `valid_from` پیش از `recorded_at` برای اعتبار دادن به عمل گذشته | back-dating | resolver گام ۶ (`recorded_at < effective_at`)؛ RC-5 | `PROPOSED` |

### ۴.۴ جلوگیری از اتصال متعارض و حساب مشترک نامعتبر

| حالت | نتیجهٔ resolver | اثر | وضعیت |
|---|---|---|---|
| `user_link_state = SHARED_USER` در زمان عمل | `CONFLICT(SHARED_ACCOUNT)` | ورود رد؛ خواندن SELF رد؛ SoD رد (F08) | `PROPOSED` |
| دو پیوند هم‌پوشان | `CONFLICT(OVERLAPPING_LINKS)` | همان | `PROPOSED` |
| E6 باز روی حساب یا شخص | `CONFLICT(OPEN_FINDING)` | همان | `PROPOSED` |
| حساب `NON_PERSONAL` | `BLOCKED(NON_PERSONAL_ACCOUNT)` | ورود انسانی رد؛ فقط مسیر principal خدمت (§۷) | `PROPOSED` |
| پیوند به شخص `MERGED_INTO` | `VERIFIED` با شناسهٔ بازمانده + پرچم E6 | پذیرش ورود: `CONFLICT` تا بازبینی | `PROPOSED` |
| `USER_NOT_IMPORTED` / `USER_NOT_RECORDED` / بدون پیوند | `UNKNOWN(NO_VERIFIED_LINK)` | ورود رد | `PROPOSED` |

---

## ۵. تاریخچه، تصمیم‌گیرنده و بازبین

| الزام | قرارداد | وضعیت |
|---|---|---|
| H-A1 | هر ساختار دوزمانه است: `recorded_at` (زمان سیستم) و `[valid_from, valid_to)` (زمان مؤثر) | `PROPOSED` |
| H-A2 | هر ستون کنشگر هم `actor_account_id` و هم `actor_verified_person_id` در زمان عمل را نگه می‌دارد. اگر شخص `UNKNOWN` باشد، عمل رد می‌شود | `PROPOSED` |
| H-A3 | هر انتقال یک ردیف در `audit.audit_event` با `subject_domain = 'organization'` و فقط شناسه و hash در `metadata` | `PROPOSED` (`audit.audit_event` موجود است: `internal/platform/migrations/sql/00001_platform_audit.sql`) |
| H-A4 | تصمیم‌گیرنده (سازنده، تعلیق‌کننده، لغوکننده) و بازبین مستقل در ستون‌های جدا ثبت می‌شوند. هیچ ستونی هر دو نقش را نمی‌پذیرد | `PROPOSED` |
| H-A5 | لغو با اثر گذشته (`retroactive`) نتایج گذشته را بازنویسی نمی‌کند. فقط رویداد پرچم ثبت می‌شود (RC-3) | `PROPOSED` |
| H-A6 | نگه‌داری: هرگز حذف نمی‌شود. حداقل مدت قانونی با Legal (HQ-5) | `REQUIRES_APPROVAL` |

## ۶. وضعیت UNKNOWN و CONFLICT، رفتار fail-closed و کنترل همکاری فعال

### ۶.۱ purposeهای جدید resolver

فهرست بستهٔ purpose در `rm-e04i…` §5.2 شش مقدار دارد. این بسته چهار مقدار پیشنهاد می‌دهد. افزودن آن‌ها تصمیم IQ-3 است.

| purpose | مصرف‌کننده | وضعیت |
|---|---|---|
| `SESSION_ADMISSION` | پذیرش ورود و ادامهٔ نشست | `REQUIRES_APPROVAL` |
| `ACCOUNT_PROVISIONING` | فرمان‌های D-34 | `REQUIRES_APPROVAL` |
| `GRANT_SUBJECT` | گیرندهٔ Grant در D-42 | `REQUIRES_APPROVAL` |
| `GRANT_ACTOR` | درخواست‌کننده، تأییدکننده و لغوکنندهٔ Grant | `REQUIRES_APPROVAL` |

### ۶.۲ پذیرش ورود (Account Admission)

**مسئله:** نشست امروز به شخص یا همکاری نگاه نمی‌کند (F02، PRB-S4). identity نباید جدول organization را بخواند یا بنویسد (lint `foreign-schema-write`).

**پیشنهاد:** پروجکشن `identity.account_admission` در schema هویت که **فقط identity** آن را می‌نویسد. ورودی آن رویدادهای منتشرشدهٔ organization (outbox) و نتیجهٔ resolver با purpose `SESSION_ADMISSION` است.

| فیلد | قرارداد | وضعیت |
|---|---|---|
| `user_id` | حساب | `PROPOSED` |
| `state` | `ADMITTED`، `NOT_ADMITTED` | `PROPOSED` |
| `reason` | `LINK_NOT_VERIFIED`، `LINK_SUSPENDED`، `LINK_REVOKED`، `EMPLOYMENT_NOT_ACTIVE`، `EMPLOYMENT_UNKNOWN`، `IDENTITY_CONFLICT`، `NON_PERSONAL_ACCOUNT`، `ADMIN_HOLD` | `PROPOSED` |
| `verified_person_id`، `link_id`، `registry_revision` | snapshot | `PROPOSED` |
| `valid_until` | TTL کوتاه؛ پس از آن بدون تجدید، رد | `REQUIRES_APPROVAL` (مقدار TTL) |
| `recorded_at`، `source_event_id` | ردیابی | `PROPOSED` |

**گزینهٔ جایگزین:** فراخوانی هم‌زمان API کوئری organization در هر احراز. ساده‌تر است، اما هر درخواست را به در دسترس بودن organization وابسته می‌کند و lint مرز ماژول را سخت‌تر می‌کند. توصیهٔ مهندسی: پروجکشن. وضعیت: `REQUIRES_APPROVAL` (بازبینی معماری و DBA).

### ۶.۳ قاعدهٔ پذیرش (fail-closed)

`ADMITTED` فقط وقتی همهٔ شرط‌های زیر برقرار باشند. ترتیب ارزیابی: BLOCKED ← CONFLICT ← UNKNOWN ← ADMITTED. **هر خطا، نبود ردیف، ردیف منقضی یا مقدار ناشناخته = رد.**

| # | شرط | نقض ← نتیجه | وضعیت |
|---|---|---|---|
| AD-1 | R5-SEC-01 `APPROVED_VERIFIED` با A (یا B) | `BLOCKED(RESOLVER_NOT_AUTHORISED)` | `PROPOSED` |
| AD-2 | حساب `PERSONAL` | `BLOCKED(NON_PERSONAL_ACCOUNT)` | `PROPOSED` |
| AD-3 | بدون `SHARED_USER`، پیوند هم‌پوشان یا E6 باز | `CONFLICT` | `PROPOSED` |
| AD-4 | دقیقاً یک پیوند `EFFECTIVE` با purpose `SESSION_ADMISSION` | `UNKNOWN(NO_VERIFIED_LINK)` یا `LINK_SUSPENDED` | `PROPOSED` |
| AD-5 | همکاری شخص در زمان بررسی «فعال» است (جدول §۶.۴) | `EMPLOYMENT_NOT_ACTIVE` / `EMPLOYMENT_UNKNOWN` | `REQUIRES_APPROVAL` |
| AD-6 | `credential.status = 'ACTIVE'` و `presence = 'PRESENT'` | رد (رفتار موجود) | `EXISTING_VERIFIED` |
| AD-7 | ردیف پذیرش وجود دارد و `valid_until > now` | رد | `PROPOSED` |

### ۶.۴ نگاشت وضعیت همکاری به پذیرش

مقادیر از CHECK `employment_status` (`00001_organization.sql:265`). **معنای کسب‌وکاری هر وضعیت تصمیم HR است.** این ستون فقط پیش‌فرض امن را پیشنهاد می‌دهد.

| `employment_status` | پیش‌فرض امن پیشنهادی | وضعیت |
|---|---|---|
| `ACTIVE` | `ADMITTED` (اگر AD-1 تا AD-4 برقرار باشند) | `REQUIRES_APPROVAL` |
| `PROBATION` | `NOT_ADMITTED` تا تصمیم HR | `REQUIRES_APPROVAL` |
| `SUSPENDED` | `NOT_ADMITTED` + تعلیق پیوند (L-7) | `REQUIRES_APPROVAL` |
| `INACTIVE` | `NOT_ADMITTED` | `REQUIRES_APPROVAL` |
| `RESIGNED` | `NOT_ADMITTED` + لغو پیوند (L-9، `TERMINATION`) | `REQUIRES_APPROVAL` |
| `TERMINATED` | `NOT_ADMITTED` + لغو پیوند (L-9، `TERMINATION`) | `REQUIRES_APPROVAL` |
| `UNMAPPED` | `NOT_ADMITTED(EMPLOYMENT_UNKNOWN)` | `PROPOSED` |
| `NOT_RECORDED` | `NOT_ADMITTED(EMPLOYMENT_UNKNOWN)` | `PROPOSED` |

**محدودیت شاهد:** `employment` تاریخچهٔ مؤثر-تاریخ ندارد و با هر واردسازی بازنویسی می‌شود (`rm-e04h…` §3.2). پس این بررسی فقط «اکنون» را می‌سنجد و برای پرسش تاریخی پاسخ `UNKNOWN` است. مدل همکاری مؤثر-تاریخ تصمیم و کار جداگانه است. وضعیت: `UNKNOWN`.

### ۶.۵ نقاط بررسی در ورود و نشست

| نقطه | رفتار پیشنهادی | محل | وضعیت |
|---|---|---|---|
| `POST /auth/login` | پس از بررسی رمز و پیش از صدور نشست، پذیرش `ADMITTED` لازم است. در غیر این صورت پاسخ عمومی شکست ورود (بدون افشای دلیل) و `auth_event` با دلیل داخلی | `session.go:160` (`Login`) | `PROPOSED` |
| هر درخواست (`Authenticate`) | JOIN به `identity.account_admission` با `state = 'ADMITTED' AND valid_until > now`؛ نبود ردیف = ناشناس | `session.go:284-292` | `PROPOSED` |
| تغییر پذیرش به `NOT_ADMITTED` | فرمان ابطال همهٔ نشست‌های حساب با دلیل `ADMISSION_REVOKED` | §۸ | `PROPOSED` |
| پاسخ ناشناس | همان رفتار موجود: کوکی پاک، ۴۰۱ | `apiserver/auth.go:94-124` | `EXISTING_VERIFIED` |

این تغییر **فقط شرط رد اضافه می‌کند** (یکنوا) و هیچ مسیر پذیرش جدیدی نمی‌سازد.

---

## ۷. حساب خدمت و حساب غیرشخصی (F05)

| قاعده | قرارداد | وضعیت |
|---|---|---|
| NP-1 | هر حساب یک `account_kind` دارد: `PERSONAL` یا `NON_PERSONAL`. پیش‌فرض برای حساب‌های واردشده: `UNKNOWN`، که برای ورود معادل رد است | `PROPOSED` |
| NP-2 | تعیین `NON_PERSONAL` توسط Security + HR (HQ-4)، با رکورد append-only | `REQUIRES_APPROVAL` |
| NP-3 | حساب غیرشخصی هرگز وارد رابط کاربری نمی‌شود، هرگز امضا یا بازبینی نمی‌کند و هرگز Grant انسانی نمی‌گیرد | `PROPOSED` |
| NP-4 | principal خدمت برنامه‌ای (یکپارچگی) قرارداد جدا دارد. worker امروز با login دیتابیس کار می‌کند، نه حساب کاربری | `UNKNOWN` (مدل نشده؛ RM-P09 §۳.۲) |

## ۸. قطع دسترسی هنگام خاتمهٔ همکاری

| گام | رفتار | پیش‌نیاز | وضعیت |
|---|---|---|---|
| TM-1 | رویداد خاتمه در organization (اعمال نهایی HR) | OPD-05، SD-04؛ امروز `HR_APPLY_IMPLEMENTATION_BLOCKED` | `BLOCKED` |
| TM-1b | **جایگزین پایلوت:** فرمان دستی `DeactivateAccountForTermination` (فایل ۳، فرمان P-6) توسط اپراتور Security | D-34 | `REQUIRES_APPROVAL` |
| TM-2 | پیوند ← `REVOKED(TERMINATION)` با `effective_at` | R5-SEC-01 | `PROPOSED` |
| TM-3 | پذیرش ← `NOT_ADMITTED(LINK_REVOKED)` | این فایل §۶ | `PROPOSED` |
| TM-4 | ابطال همهٔ نشست‌های همهٔ حساب‌های شخص با دلیل جدید `ADMISSION_REVOKED`؛ رویداد `SESSION_REVOKED` | مهاجرت افزودن دلیل به CHECK `session_revoke_reason` (`00012:60`) | `REQUIRES_APPROVAL` |
| TM-5 | غیرفعال‌سازی اعتبارنامه (`DISABLED`) | D-34 | `REQUIRES_APPROVAL` |
| TM-6 | Grantها لغو نمی‌شوند مگر با فرمان D-42. اما بدون پذیرش، هیچ نشستی برای استفاده از آن‌ها وجود ندارد | D-42 | `PROPOSED` |

**معیار:** درخواست بعدی پس از TM-3 یا TM-4 پاسخ ۴۰۱ می‌گیرد (T-ID-03 در RM-P09 §۷.۴). فاصلهٔ زمانی بین رویداد و اثر باید اندازه‌گیری و در Runbook ثبت شود.

## ۹. ارتباط با منابع انسانی و ماژول Identity

| مرز | جهت | قرارداد | وضعیت |
|---|---|---|---|
| organization → identity | رویداد outbox (`person_registry.link_*`، `employment.*`) | identity فقط پروجکشن خودش را می‌نویسد | `PROPOSED` |
| identity → organization | فراخوانی API کوئری resolver (فقط خواندن) | هرگز نوشتن | `PROPOSED` |
| organization ↛ `identity.*` | نوشتن ممنوع | lint `s11-hr-identity-write`، `foreign-schema-write` | `EXISTING_VERIFIED` |
| resolver ↛ Grant / Capability | resolver فقط هویت برمی‌گرداند، نه `authorization.Subject` | `rm-e04i…` §5.1 | `PROPOSED` |
| Principal نشست | فیلد اختیاری `verified_person_id` از پذیرش؛ هرگز از درخواست | `websession.go:80-88` | `PROPOSED` |

## ۱۰. آزمون‌های پذیرش این قرارداد (برای RM-P10B)

| # | آزمون | انتظار | وضعیت |
|---|---|---|---|
| IT-1 | حساب بدون پیوند `VERIFIED` با اعتبارنامهٔ مصنوعی وارد می‌شود | رد؛ `auth_event` با دلیل `ACCOUNT_NOT_ADMITTED` | `PROPOSED` |
| IT-2 | حساب `SHARED_USER` | رد؛ SELF و SoD رد | `PROPOSED` |
| IT-3 | خاتمه در میانهٔ نشست | درخواست بعدی ۴۰۱؛ همهٔ نشست‌ها با دلیل ثبت‌شده باطل | `PROPOSED` |
| IT-4 | تعلیق و بازگشت پیوند توسط همان فرد | بازگشت رد | `PROPOSED` |
| IT-5 | دو پیوند هم‌پوشان هم‌زمان | یکی شکست می‌خورد؛ وضعیت جزئی وجود ندارد | `PROPOSED` |
| IT-6 | ردیف پذیرش منقضی | رد | `PROPOSED` |
| IT-7 | تغییر نام نمایشی | هیچ نتیجه‌ای تغییر نمی‌کند | `PROPOSED` |
| IT-8 | `UPDATE` یا `DELETE` روی جدول رجیستری | خطای تریگر | `PROPOSED` |

**GROUP B CLOSED — PRODUCTION NO-GO.**
