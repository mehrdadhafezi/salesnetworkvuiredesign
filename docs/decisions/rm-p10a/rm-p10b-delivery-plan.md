# RM-P10A / ۷ — برنامهٔ توسعهٔ مرحله‌بندی‌شدهٔ RM-P10B

**DRAFT — NOT APPROVED.** این برنامه هیچ مرحله‌ای را مجاز نمی‌کند. هر مرحله فقط وقتی شروع می‌شود که **همهٔ** پیش‌نیازهای تصویب آن با رکورد R6 `APPROVED_VERIFIED` ثبت شده باشند. جلسه، سند ادغام‌شده، CI سبز یا ترجیح مالک پروژه کافی نیست (`m2-engineering-readiness.md` §۰). **GROUP B CLOSED — PRODUCTION NO-GO.**

بستهٔ اصلی: `docs/decisions/rm-p10a-identity-access-closure-package.md`. نگاشت به برنامه‌های قبلی: PR-A = F-PR1، PR-B = رجیستری سایه، PR-C = F-PR2، PR-D = سخت‌سازی D-42 (`rm-e04i…` §10)؛ S1 تا S5 (`m2-engineering-readiness.md` §۲).

## ۰. قواعد همهٔ مراحل

| # | قاعده | وضعیت |
|---|---|---|
| DR-1 | هر مرحله یک PR کوچک و مستقل است؛ PR بزرگ برای کل زیرساخت هویت و دسترسی ممنوع | `PROPOSED` |
| DR-2 | همهٔ آزمون‌ها روی PostgreSQL موقت و دادهٔ مصنوعی؛ بدون دسترسی Production یا Staging واقعی (جز P10B-11) | `PROPOSED` |
| DR-3 | هر قابلیت جدید پشت پرچم پیش‌فرض خاموش؛ روشن‌کردن پرچم در محیط واقعی مجوز جدا دارد | `PROPOSED` |
| DR-4 | هر مرحله فقط شرط رد اضافه می‌کند تا P10B-5 و P10B-7؛ این دو تنها مراحلی‌اند که نوشتن جدید (اعتبارنامه، Grant) را *در کد* ممکن می‌کنند و هیچ‌کدام در محیط واقعی اجرا نمی‌شوند | `PROPOSED` |
| DR-5 | هر PR بازبینی مستقل دارد (انسان یا عاملی غیر از نویسنده)؛ نویسنده ادغام نمی‌کند | `EXISTING_VERIFIED` (AGENTS.md §Review) |
| DR-6 | آزمون‌های قفل P10B-0 در هر مرحله سبز می‌مانند؛ تغییر هر کدام فقط در همان PRی که قفل را با تصویب برمی‌دارد | `PROPOSED` |

## ۱. مراحل

| مرحله | عنوان | پیش‌نیاز تصویب | فایل‌ها / ماژول‌ها | معیار پذیرش قابل آزمون | جلوگیری از باز شدن زودهنگام Grant | وضعیت |
|---|---|---|---|---|---|---|
| **P10B-0** | آزمون‌های قفل و فهرست endpointها | **هیچ تصمیم کسب‌وکار**؛ فقط بازبینی PR | `internal/authorization/http/testdata/operation_scope_registry.yaml` (فقط خواندن)؛ آزمون جدید در `internal/authorization/http/`؛ آزمون حاکمیتی در `docs/governance/` | آزمون‌ها رفتار امروز را pin می‌کنند: L1 تا L12 (بستهٔ اصلی §۲.۷)؛ فهرست کامل endpointهای دادهٔ پرونده و گزارش؛ هر endpoint گزارش فقط ORGANIZATION می‌پذیرد | خود این مرحله قفل‌ها را با آزمون محکم‌تر می‌کند | `PROPOSED` |
| **P10B-0b** | رفع F09 | بازبینی امنیتی مستقل (بدون تصمیم کسب‌وکار) | `internal/fulfillment/http/work.go`؛ رجیستری عملیات | خارج از دامنه ← ۴۰۴ بدون کوئری جزئیات؛ برچسب `QUERY_BEFORE_IDENTITY` حذف؛ `TestNT22*` سبز | هیچ دسترسی جدیدی ندارد | `PROPOSED` |
| **P10B-1** | رجیستری سایهٔ شخص (PR-B) | AP-1 (R5-SEC-01 = A)، AP-2 (IQ-1، IQ-2، IQ-4)، AP-3 (BOOT)، بازبینی معماری و DBA، بازبینی Legal | `internal/organization/migrations/sql/` (E1 تا E8)؛ `internal/organization/{domain,store,api}`؛ فرمان CLI Bootstrap | مهاجرت up/down روی DB خالی؛ تریگر append-only؛ exclusion برای هم‌پوشانی؛ بدون FK به identity؛ بدون ستون نام/تلفن؛ نقش برنامه فقط SELECT؛ صفر ردیف از واردسازی؛ IT-4، IT-5، IT-8 | هیچ نوشتن در `identity.*` (lint)؛ پرچم خاموش | `REQUIRES_APPROVAL` |
| **P10B-2** | resolver و R6 هویتی (PR-A + PR-C) | P10B-1 ادغام؛ AP-2 (IQ-3 با چهار purpose جدید)؛ H-RV-09/H-RV-10 | `internal/organization/query`؛ `docs/governance/` (جایگزینی `rmE04GRepositoryResolver` طبق `rm-e04h…` §6.5) | `TestRME04IResolverSpec` روی پیاده‌سازی واقعی؛ F1-A تا F1-H رد؛ BLOCKED/UNKNOWN/CONFLICT هرگز gate را باز نمی‌کنند | resolver هیچ Grant یا Subject برنمی‌گرداند | `REQUIRES_APPROVAL` |
| **P10B-3** | پذیرش ورود (F02، F04، F05) | AP-4 (نگاشت همکاری، TTL)؛ HQ-4 | `internal/identity/migrations/sql/` (جدول پذیرش، `account_kind`)؛ `internal/identity/session/session.go` (`Login`، `Authenticate`)؛ مصرف‌کنندهٔ رویداد | IT-1، IT-2، IT-6؛ T-ID-01، T-ID-02؛ نبود ردیف = رد؛ همهٔ آزمون‌های نشست موجود سبز | فقط شرط رد اضافه می‌کند؛ هیچ اعتبارنامه‌ای هنوز وجود ندارد | `REQUIRES_APPROVAL` |
| **P10B-4** | ابطال اداری نشست و فرمان خاتمه (F03) | AP-5 (بخش Q34-6، Q34-7، Q34-8 از D-34) | `internal/identity/migrations/sql/` (دلیل‌های جدید ابطال)؛ `internal/identity` (پکیج فرمان جدید)؛ P-3، P-5، P-6 | T-ID-03، T-SE-02، PT-5، PT-6 | فقط محدودکننده | `REQUIRES_APPROVAL` |
| **P10B-5** | تأمین حساب (S1؛ F04) | AP-5 (D-34 کامل) + AP-1 | مهاجرت مجوز INSERT/UPDATE `identity.credential` فقط برای نقش فرمان؛ استثنای محدود lint `r1-credential-write`؛ P-1، P-2، P-4 | PT-1 تا PT-8؛ DP-5 | پیش‌شرط پیوند `VERIFIED` و پذیرش؛ پرچم خاموش؛ اجرا فقط در DB موقت | `REQUIRES_APPROVAL` |
| **P10B-6** | سخت‌سازی قفل حاکمیتی Grant (PR-D) | AP-6 (H-RV-08 / I-RV-07، IQ-5) | `docs/governance/rm_e04d_test.go` (`grantConstraintProblems`، `identityMigrations`)؛ `internal/architecture/lint.go` (EX-5) | ND-1 تا ND-11؛ G1 تا G5 هر کدام با fixture منفی کشته شوند | **پیش‌شرط اجباری P10B-7** | `REQUIRES_APPROVAL` |
| **P10B-7** | فرمان‌های Grant و Revoke + E-7 (S2) | AP-7 (D-42 + E-7 + پیشنهاد ثبت ۱ برای نقش پایلوت)؛ P10B-6 ادغام | مهاجرت بازتعریف `role_grant_shadow_only` با basis جدید (Grantهای SHADOW دست نمی‌خورند)؛ `internal/identity` (پکیج فرمان جدید) (G-1 تا G-7)؛ تاریخچهٔ E-7 | GT-1 تا GT-10؛ T-SE-01 | PP-G1 تا PP-G7؛ پرچم خاموش؛ هیچ Grant در محیط واقعی | `REQUIRES_APPROVAL` |
| **P10B-8** | دامنهٔ سطری SELF/TEAM (S3؛ F06) — یک PR برای هر ماژول، پرونده اول | AP-8 (پیشنهاد ثبت ۲؛ D-32 برای TEAM) | `internal/authorization/authorization.go` (مقادیر جدید)؛ `guard.go`؛ کوئری‌های `internal/cases`، `internal/custody`، سپس `billing`، `commission`؛ رجیستری عملیات | SC-1 تا SC-8؛ N1 تا N8؛ آزمون جهش حذف گزاره | دامنه فقط از Grant؛ بدون ارتقای خودکار | `REQUIRES_APPROVAL` |
| **P10B-9** | گزارش و شمارندهٔ محدود (F07) | تصمیم دامنهٔ گزارش (بخش AP-8) | `internal/reporting/http/http.go`؛ پارامتر `scope` در OpenAPI | SC-4؛ T-AC-03 | قاعدهٔ موقت CR-2/CR-3 تا آن زمان | `REQUIRES_APPROVAL` |
| **P10B-10** | ممیزی ۴۰۳ و خواندن حساس (F12) | AP-9 (D-45، D-23، نگه‌داری لاگ با Legal) | `internal/platform/audit`؛ middleware مجوز | T-LOG-01؛ شکست ثبت طبق D-45 | — | `REQUIRES_APPROVAL` |
| **P10B-11** | پیکربندی Staging و drill توقف (F11) | AP-10 (BN-16)؛ ADR #18 فقط برای دادهٔ واقعی | `deploy/staging/compose.yaml`؛ Runbook | ST-1 تا ST-6؛ K-1 تا K-6 | فقط حساب و دادهٔ مصنوعی | `REQUIRES_APPROVAL` |
| **P10B-12** | پذیرش پایلوت HR (PC-15) | AP-11 (مجوز فعال‌سازی جدا، Group C) | گزارش پذیرش | چک‌لیست PC-01 تا PC-15 با حساب واقعی HR روی Staging | تا این مرحله HR و Finance «NOT LIVE VERIFIED» می‌مانند | `BLOCKED` |

## ۲. مسیر حیاتی

```
P10B-0 ─ P10B-0b
   │
AP-1 + AP-2 + AP-3 ─▶ P10B-1 ─▶ P10B-2 ─▶ P10B-3 ─▶ P10B-4 ─▶ P10B-5 ──────────────┐
                                    │                                               │
                              AP-6 ─┴▶ P10B-6 ─▶ P10B-7 ─▶ P10B-8 ─▶ P10B-9 ───────┤
                                                                                    ├▶ P10B-12
                                              AP-9 ─▶ P10B-10 ──────────────────────┤
                                              AP-10 ─▶ P10B-11 ─────────────────────┘
```

P10B-10 و P10B-11 از نظر کد مستقل‌اند و می‌توانند پس از تصویب خودشان موازی پیش بروند.

## ۳. اولین PR اجرایی امن

| حالت | PR | دلیل امن بودن | وضعیت |
|---|---|---|---|
| پیش از هر تصویب | P10B-0 (آزمون‌های قفل و فهرست endpointها) | فقط آزمون؛ قفل‌ها را محکم‌تر می‌کند؛ رفتار Runtime تغییر نمی‌کند | `PROPOSED` |
| پیش از هر تصویب (با بازبینی امنیتی) | P10B-0b (F09) | فقط ترتیب بررسی را سخت‌تر می‌کند | `PROPOSED` |
| **اولین PR اجرایی پس از تصویب** | **P10B-1: رجیستری سایهٔ شخص** | جدول‌های خالی؛ append-only؛ پرچم خاموش؛ نقش برنامه SELECT؛ دادهٔ مصنوعی؛ بدون نوشتن در identity؛ بدون اعتبارنامه و Grant؛ rollback = down migration تا وقتی ردیفی وجود ندارد | `REQUIRES_APPROVAL` |

**GROUP B CLOSED — PRODUCTION NO-GO.**
