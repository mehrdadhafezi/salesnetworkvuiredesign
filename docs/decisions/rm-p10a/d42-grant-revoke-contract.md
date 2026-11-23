# RM-P10A / ۴ — قرارداد درخواست، تأیید، اجرا و لغو Grant (D-42 و E-7)

**DRAFT — NOT APPROVED.** فقط قرارداد؛ هیچ Grant یا Revoke اجرا نمی‌شود و قید `role_grant_shadow_only` دست نمی‌خورد. D-42 در ردیاب `READY_FOR_REVIEW` است و `approval_record` آن خالی است. E-7 پیش‌نیاز مهندسی باز است. **GROUP B CLOSED — PRODUCTION NO-GO.**

بستهٔ اصلی: `docs/decisions/rm-p10a-identity-access-closure-package.md`. این قرارداد S2 در `docs/remediation/m2-engineering-readiness.md` و H-1 تا H-8 در `docs/security/rm-e04i-option-a-implementation-contract.md` §8 را دقیق می‌کند.

---

## ۱. وضعیت امروز (شاهد)

| موضوع | امروز | شاهد | وضعیت |
|---|---|---|---|
| Grant فعال در DB | غیرممکن: `CHECK (status = 'SHADOW')` و `CHECK (basis = 'LEGACY_PROFILE_ROLE')` | `internal/identity/migrations/sql/00001_identity.sql:194-195`؛ PRB-S1 | `EXISTING_VERIFIED` |
| خوانندهٔ Runtime | `ActiveGrants` فقط `ACTIVE` و `PRESENT` را می‌خواند؛ در هر درخواست | `internal/identity/store/store.go:392`؛ `internal/identity/query/query.go:155-164` | `EXISTING_VERIFIED` |
| بستهٔ نقش‌ها | ۸ نقش تهی؛ HR شش Capability خواندنی | `internal/identity/api/api.go:211-221` | `EXISTING_VERIFIED` |
| فرمان Grant / Revoke | وجود ندارد | RM-P09 §۳.۳ | `EXISTING_VERIFIED` |
| قفل حاکمیتی | `grantConstraintProblems` فقط وضعیت D-42 را می‌سنجد (G1 تا G5) | `docs/governance/rm_e04d_test.go`؛ `rm-e04h…` §7.2 | `EXISTING_VERIFIED` |
| سلسله‌مراتب | هرگز مجوز نمی‌دهد | `TestHierarchyAndPositionNeverGrant` | `EXISTING_VERIFIED` |

## ۲. آنچه D-42 باید تصمیم بگیرد

گزینه‌های ثبت‌شده (`r5-decision-inventory.md:105`): (۱) فقط Identity / Security؛ (۲) HR درخواست، Security تأیید؛ (۳) HR در مجموعهٔ محدود با ممیزی. هیچ‌کدام انتخاب نشده است.

| # | پرسش | گزینهٔ ۱ | گزینهٔ ۲ | گزینهٔ ۳ | وضعیت |
|---|---|---|---|---|---|
| Q42-1 | مرجع درخواست | Security | HR (یا مدیر صف با تأیید HR) | HR | `REQUIRES_APPROVAL` |
| Q42-2 | مرجع تأیید | Security (نفر دوم) | Security | HR (نفر دوم) در مجموعهٔ محدود؛ بیرون از آن Security | `REQUIRES_APPROVAL` |
| Q42-3 | مرجع اجرا | سیستم پس از تأیید (بدون کنشگر انسانی سوم) | همان | همان | `PROPOSED` |
| Q42-4 | مرجع لغو عادی | Security | HR درخواست، Security اجرا؛ یا هر دو مستقیم | HR یا Security | `REQUIRES_APPROVAL` |
| Q42-5 | مرجع لغو فوری | هر اپراتور Security یا HR، یک نفر (محدودکننده) | همان | همان | `REQUIRES_APPROVAL` |
| Q42-6 | نقش‌های قابل اعطا و حداکثر دامنهٔ هر نقش | فقط نقش‌هایی که بسته‌شان تصویب شده (پیشنهاد ثبت ۱ برای فروشنده؛ HR موجود) | همان | «مجموعهٔ محدود» باید فهرست دقیق داشته باشد | `REQUIRES_APPROVAL` |
| Q42-7 | سقف مدت Grant | پیش‌فرض امن: `valid_to` الزامی؛ مقدار با Security | همان | همان | `REQUIRES_APPROVAL` |
| Q42-8 | آیا تفکیک وظایف در سطح شخص الزامی است؟ | بله (دو نفر Security) | بله (HR ≠ Security) | بله | `REQUIRES_APPROVAL` (وابسته به R5-SEC-01) |

## ۳. مدل داده (پیشنهادی؛ بدون مهاجرت)

| موجودیت | فیلدهای کلیدی | قاعده | وضعیت |
|---|---|---|---|
| `GrantRequest` | `request_id`، `subject_user_id`، `subject_person_id` (snapshot)، `role_key`، `scope` (NONE/SELF/TEAM/ALL)، `valid_from`، `valid_to`، `justification`، `requested_by_account_id`، `requested_by_person_id`، `state`، `recorded_at` | append-only؛ تغییر حالت = ردیف رویداد جدید | `PROPOSED` |
| `GrantApproval` | `approval_id`، `request_id`، `decision` (`APPROVE` / `REJECT`)، `approver_account_id`، `approver_person_id`، `reason`، `recorded_at` | تأییدکننده ≠ درخواست‌کننده ≠ موضوع (شخص) | `PROPOSED` |
| `identity.role_grant` (ردیف جدید) | `basis` جدید (نام پیشنهادی: `APPROVED_GRANT_REQUEST`)، `status` (`ACTIVE` / `REVOKED` / `EXPIRED`)، `request_id`، `scope`، `valid_from`، `valid_to` | ردیف‌های `LEGACY_PROFILE_ROLE` و `SHADOW` قدیمی هرگز تغییر نمی‌کنند | `REQUIRES_APPROVAL` (مهاجرت پس از H-1 تا H-8) |
| `GrantEvent` (E-7) | `event_id`، `grant_id`، `kind` (`GRANTED`، `REVOKED`، `EXPIRED`، `EMERGENCY_REVOKED`)، کنشگر و شخص کنشگر، دلیل، `correlation_id` | append-only با تریگر | `PROPOSED` |

### ماشین وضعیت `GrantRequest`

| از | به | کنشگر | شرط | وضعیت |
|---|---|---|---|---|
| — | `SUBMITTED` | درخواست‌کننده | PP-G1 تا PP-G6 | `PROPOSED` |
| `SUBMITTED` | `REQUEST_APPROVED` | تأییدکننده | SoD؛ نقش و دامنه در مجموعهٔ مجاز | `PROPOSED` |
| `SUBMITTED` | `REJECTED` | تأییدکننده | دلیل | `PROPOSED` |
| `SUBMITTED` | `WITHDRAWN` | درخواست‌کننده | — | `PROPOSED` |
| `SUBMITTED` | `EXPIRED` | سیستم | بدون تصمیم پس از مهلت | `REQUIRES_APPROVAL` (مهلت) |
| `REQUEST_APPROVED` | `EXECUTED` | سیستم | همهٔ پیش‌شرط‌ها **دوباره** در لحظهٔ اجرا بررسی می‌شوند (TOCTOU) | `PROPOSED` |
| `REQUEST_APPROVED` | `EXECUTION_REFUSED` | سیستم | پیش‌شرطی در لحظهٔ اجرا نقض شد | `PROPOSED` |
| حالت پایانی | هر حالت | — | **ممنوع** | `PROPOSED` |
| `SUBMITTED` | `EXECUTED` | هر کس | **ممنوع** (بدون تأیید) | `PROPOSED` |

### ماشین وضعیت Grant

| از | به | کنشگر | شرط | وضعیت |
|---|---|---|---|---|
| — | `ACTIVE` | سیستم (اجرای درخواست) | `GrantRequest = REQUEST_APPROVED` | `PROPOSED` |
| `ACTIVE` | `REVOKED` | لغوکنندهٔ مجاز | دلیل | `PROPOSED` |
| `ACTIVE` | `EXPIRED` | سیستم | `valid_to` رسید | `PROPOSED` |
| `REVOKED` / `EXPIRED` | `ACTIVE` | — | **ممنوع**؛ فقط درخواست جدید | `PROPOSED` |
| `ACTIVE` | `ACTIVE` با دامنه یا نقش دیگر | — | **ممنوع**؛ تغییر = لغو + درخواست جدید | `PROPOSED` |

## ۴. پیش‌شرط‌های مشترک

| # | پیش‌شرط | نقض ← خطا | وضعیت |
|---|---|---|---|
| PP-G1 | D-42 `APPROVED_VERIFIED`؛ R5-SEC-01 `APPROVED_VERIFIED` (A یا B)؛ E-7 پذیرفته؛ قفل حاکمیتی سخت‌شده (H-1 تا H-6) | `BLOCKED(DECISION_NOT_APPROVED)` | `REQUIRES_APPROVAL` |
| PP-G2 | پرچم فرمان روشن (پیش‌فرض خاموش؛ تغییر پرچم خودش تابع E-7) | `BLOCKED(FEATURE_DISABLED)` | `PROPOSED` |
| PP-G3 | کنشگر `ADMITTED`، با Capability فرمان، و `resolvePerson(actor, now, GRANT_ACTOR) = VERIFIED` | `UNAUTHORIZED` | `PROPOSED` |
| PP-G4 | موضوع: `resolvePerson(subject, now, GRANT_SUBJECT) = VERIFIED`، `account_kind = PERSONAL`، پذیرش `ADMITTED` | `BLOCKED(GRANTEE_NOT_VERIFIED)` | `PROPOSED` |
| PP-G5 | نقش و دامنه در «مجموعهٔ قابل اعطا»ی تصویب‌شده؛ Capability گزارش فقط با ALL (فایل ۲، CR-3) | `UNAUTHORIZED(ROLE_NOT_GRANTABLE)` | `REQUIRES_APPROVAL` |
| PP-G6 | ورودی نقش فقط `role_key` صریح. هیچ استنتاجی از سمت، واحد، سلسله‌مراتب، نام نقش قدیمی یا پروفایل | ۴۰۰ | `PROPOSED` |
| PP-G7 | `Idempotency-Key` الزامی | `CONFLICT(IDEMPOTENCY_MISMATCH)` | `PROPOSED` |

## ۵. فرمان‌ها

| فرمان | ورودی | خروجی | خطا | پیش‌شرط خاص | اثر امنیتی | وضعیت |
|---|---|---|---|---|---|---|
| G-1 `RequestGrant` | `subject_user_id`، `role_key`، `scope`، `valid_from`، `valid_to`، `justification` | `SUCCESS { request_id, state: SUBMITTED }` | `BLOCKED`، `UNAUTHORIZED`، `CONFLICT(DUPLICATE_OPEN_REQUEST)` | درخواست‌کننده ≠ موضوع | هیچ؛ فقط درخواست | `REQUIRES_APPROVAL` |
| G-2 `ApproveGrantRequest` | `request_id`، `reason` | `SUCCESS { state: REQUEST_APPROVED }` سپس اجرای خودکار G-4 | `UNAUTHORIZED(SAME_PERSON)`، `CONFLICT(STATE)` | تأییدکننده ∉ {درخواست‌کننده، موضوع} در سطح شخص | راه‌اندازی اجرا | `REQUIRES_APPROVAL` |
| G-3 `RejectGrantRequest` | `request_id`، `reason` | `SUCCESS { state: REJECTED }` | `UNAUTHORIZED`، `CONFLICT(STATE)` | مانند G-2 | هیچ | `REQUIRES_APPROVAL` |
| G-4 `ExecuteGrant` (سیستمی) | `request_id` | `SUCCESS { grant_id }` یا `EXECUTION_REFUSED(reason)` | `UNKNOWN` (تطبیق پیش از تکرار) | بررسی مجدد PP-G1 تا PP-G5 | ردیف `ACTIVE` جدید؛ مؤثر از درخواست بعدی | `REQUIRES_APPROVAL` |
| G-5 `RevokeGrant` | `grant_id`، `reason` | `SUCCESS`؛ اگر قبلاً لغو شده `SUCCESS` بدون تغییر | `UNAUTHORIZED` | لغوکنندهٔ مجاز (Q42-4)؛ نیازی به `VERIFIED` بودن موضوع نیست | درخواست بعدی ۴۰۳ `ACTION_DENIED` | `REQUIRES_APPROVAL` |
| G-6 `EmergencyRevokeAllGrants` | `user_id`، `reason` | `SUCCESS { revoked_count }` | `UNAUTHORIZED` | یک اپراتور مجاز (Q42-5) | همهٔ Grantهای `ACTIVE` حساب (و اختیاری: همهٔ حساب‌های شخص) لغو؛ اختیاری: P-5 فایل ۳ | `REQUIRES_APPROVAL` |
| G-7 `ExpireGrants` (سیستمی) | — | تعداد | — | `valid_to ≤ now` | `EXPIRED` | `PROPOSED` |
| G-8 `ChangeGrantScope` | — | — | — | **فرمان جدا ندارد**: G-5 + G-1 | جلوگیری از ارتقای درجا | `PROPOSED` |

## ۶. E-7: رویهٔ تغییر Grant و Flag با ممیزی

| الزام | قرارداد | وضعیت |
|---|---|---|
| E7-1 | تاریخچهٔ append-only برای Grant (`GrantEvent`) و پرچم (`flag_change_event`)؛ بدون UPDATE و DELETE | `PROPOSED` |
| E7-2 | هر رویداد: کنشگر، شخص کنشگر، دلیل، مقدار قبل و بعد، `correlation_id`، ارجاع به رکورد تصمیم | `PROPOSED` |
| E7-3 | تغییر پرچم امنیتی (`identity.session_auth` و پرچم‌های نوشتن) دو نفر دارد، جز خاموش‌کردن اضطراری که یک نفر کافی است | `REQUIRES_APPROVAL` |
| E7-4 | گزارش روزانهٔ «Grant فعال بدون رویداد `GRANTED`» و «پرچم روشن بدون رویداد» = هشدار | `PROPOSED` |
| E7-5 | شواهد Runtime روی Staging با دادهٔ مصنوعی پیش از پذیرش (H-5) | `REQUIRES_APPROVAL` |
| E7-6 | طراحی این سند پیش از تصویب مجاز است (`m2-engineering-readiness.md` §۳). **پذیرش** آن تصمیم Engineering + Security/Ops است | `PROPOSED` |

## ۷. جلوگیری از گسترش اختیار

| # | کنترل | محل | وضعیت |
|---|---|---|---|
| EX-1 | هیچ Grant از سمت، واحد، سلسله‌مراتب یا نقش قدیمی. 2.0.160 پنل فروشنده را با سمت می‌دهد (`class-sn-plugin.php:13764`)؛ این رفتار منتقل نمی‌شود | PP-G6؛ lint پیشنهادی | `PROPOSED` |
| EX-2 | Capabilityهای مدیریت Grant (`identity.grant.request`، `identity.grant.approve`، `identity.grant.revoke` — نام‌ها پیشنهادی) در هیچ بستهٔ نقش عادی نیستند و فقط با Bootstrap یا تصمیم جدا داده می‌شوند | بستهٔ نقش | `REQUIRES_APPROVAL` |
| EX-3 | هیچ‌کس نمی‌تواند به خودش یا به حساب دیگر خودش Grant بدهد یا تأیید کند (سطح شخص) | G-1، G-2 | `PROPOSED` |
| EX-4 | تأییدکننده نمی‌تواند Capability یا دامنه‌ای بزرگ‌تر از آنچه خودش مجاز به اعطای آن است تأیید کند | PP-G5 | `PROPOSED` |
| EX-5 | lint: نوشتن `status = 'ACTIVE'` در `role_grant` فقط از فرمان G-4 | `internal/architecture/lint.go` (قاعدهٔ پیشنهادی `rm-e04i…` §8.3) | `PROPOSED` |
| EX-6 | قفل حاکمیتی Grant مهاجرت‌های همهٔ ماژول‌ها را پویش کند (G5) | `rm_e04d_test.go` (`identityMigrations`) | `REQUIRES_APPROVAL` |

## ۸. کنترل لغو فوری

| # | رفتار | شاهد / محل | وضعیت |
|---|---|---|---|
| IR-1 | Grant در هر درخواست خوانده می‌شود؛ پس لغو از درخواست بعدی اثر دارد | `authorization/session.go:207-228` | `EXISTING_VERIFIED` |
| IR-2 | لغو اضطراری همه (G-6) با یک نفر | این فایل | `REQUIRES_APPROVAL` |
| IR-3 | توقف کامل: خاموش‌کردن `identity.session_auth` همهٔ نشست‌ها را در درخواست بعدی رد می‌کند | `session.go:279-281` | `EXISTING_VERIFIED` |
| IR-4 | آزمون: لغو در میانهٔ نشست → درخواست بعدی ۴۰۳ (T-SE-01) | RM-P09 §۷.۴ | `PROPOSED` |

## ۹. آزمون‌ها

| # | آزمون | انتظار | وضعیت |
|---|---|---|---|
| GT-1 | Grant پیش از تصویب R5-SEC-01 | `BLOCKED(DECISION_NOT_APPROVED)` | `PROPOSED` |
| GT-2 | درخواست‌کننده و تأییدکننده یک شخص با دو حساب | `UNAUTHORIZED(SAME_PERSON)` | `PROPOSED` |
| GT-3 | Grant به حساب `SHARED_USER` | `BLOCKED(GRANTEE_NOT_VERIFIED)` | `PROPOSED` |
| GT-4 | Grant نقش خارج از مجموعهٔ تصویب‌شده | `UNAUTHORIZED(ROLE_NOT_GRANTABLE)` | `PROPOSED` |
| GT-5 | Capability گزارش با دامنهٔ SELF | `UNAUTHORIZED(ROLE_NOT_GRANTABLE)` | `PROPOSED` |
| GT-6 | لغو موضوع میان تأیید و اجرا | `EXECUTION_REFUSED` | `PROPOSED` |
| GT-7 | لغو در میانهٔ نشست | درخواست بعدی ۴۰۳ | `PROPOSED` |
| GT-8 | درج مستقیم `ACTIVE` در DB پیش از مهاجرت | نقض `role_grant_shadow_only` | `EXISTING_VERIFIED` (PRB-S1) |
| GT-9 | سرپرست با ۴۰ زیرمجموعه بدون Grant | هیچ Capability | `EXISTING_VERIFIED` (PRB-A3) |
| GT-10 | ND-1 تا ND-11 قفل حاکمیتی | طبق `rm-e04i…` §8.4 | `PROPOSED` |

**GROUP B CLOSED — PRODUCTION NO-GO.**
