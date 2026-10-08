# Supervisor Redesign — Shared Design System Extension (Design only)

Status: DESIGN PROTOTYPE · no plugin / live CRM / permission / data change.
Sources: Seller Design V1 Frozen (`../index.html`, `../styles.css`, `../help.css`) · Codex `CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md` §2 · `GATE-0-PRODUCT-INVARIANTS.md` §8 §9 §14 §18 §19 (OPD-01…10) · Live read-only audit 2026-10-04 of `crm.maximumclub.ir/supervisor-panel/` (Supervisor #5).

---

## 1. Live Supervisor Audit (2026-10-04, read-only)

Entered as Supervisor #5 (view-panel session prepared by the user). No mutation executed: no assign, return, unassign, receipt, link, edit, HR request, settings.
Wallet tab **not opened** on purpose: Gate 0 F07 says the wallet renderer can call commission autopost (`force`), so opening it is not guaranteed pure-read. Wallet content taken from code evidence + Codex 09-30 capture.

| Current tab | Observed content (live) | Notes |
|---|---|---|
| فروشندگان و آمار (default) | 6 counters show «…» until manual «بارگذاری آمار فروشندگان» (504 guard). After load: فروشندگان فعال ۱ از ۱ · شماره‌های زیرمجموعه ۱ · تحویل/تخصیص ۱ (مانده آزاد ۰) · فاکتورهای تایید/پرداخت ۰ از ۰ · فروش تاییدشده ۰ تومان; second strip سرنخ زنده/قدیمی 1، دارای فروشنده 1، آزاد 0، تعداد در بازه 0، دارای فاکتور 0، فروش موفق 0. Filters: date/time range, seller, contact status (11 legacy values), import code, assigned/unassigned. Table: نام، شماره، تخصیص‌یافته، فاکتور، پرداخت‌شده، وضعیت، عملیات(پروفایل) | Two KPI strips with overlapping meanings; «فاکتور 0» while Invoices tab shows 1 invoice (F04 still visible live). No freshness. |
| تبدیل‌کنندگان و آمار | 0 active converters, 0 assigned, 0 completed, **نرخ تبدیل کل 0٪**, «0 تومان پرداخت تأییدشده», empty message | 0٪ on empty denominator = fake zero (Gate 0 §9). |
| تبدیل‌های من | KPIs کل/نیازمند پیگیری/تماس سررسیده/دارای مانده/تکمیل‌شده = 0; «فقط پرونده‌هایی که با تخصیص به خودم برداشته‌اید» | Personal work; permanent top tab while empty. |
| تخصیص شماره | 6 counters (مانده فعلی 0، آماده تخصیص سریع 0، گیرنده‌های مجاز 1، پرونده فعال 0، آخرین تخصیص —، آخرین برگشت —); method «تعداد مشخص به هر گیرنده» (only option); count, note, case/data-code filter, target level (فروشنده), recipient checkboxes (#6), «اعمال تخصیص»; second form «برگشت تخصیص‌ها به پنل من»: case filter, return count, note, «برگرداندن به پنل من» — text says eligibility is checked only *at submit* | No preview, no eligibility per item, no result detail. |
| آماده‌های تبدیل | KPIs انتخاب ثبت‌شده / انتخاب نکرده / تبدیل‌کننده فعال; note «فاکتورهای اعتبارسنجی پس از تکمیل پرداخت و تأیید مالی…»; table — پرونده، مشتری، اعتبار، انتخاب مشتری، وضعیت، پرداخت، تبدیل‌کننده، اقدام; empty | Self-assign path explicitly offered when no converter exists. |
| اشتراک‌های فروشنده | table فاکتور، مشتری، فروشنده، اشتراک، پرداخت، تغییر اشتراک; rule text: editable until first payment, then locked | Belongs to invoice context. |
| لاگ و خطا | «فقط خلاصه آخرین عملیات…» empty | Operation history. |
| جدا کردن سرنخ زنده/قدیمی | seller, last-N, assignment date/time range, contact status, import code, «جدا کردن و برگشت به لیست قابل تخصیص» | Legacy unassign; destructive bulk without preview. |
| فاکتورها | KPIs کل ۱، تأیید/پرداخت ۰، جمع مبلغ ۱۵٬۰۰۰، فروش تأییدشده ۰; quick filters همه/پیش‌فاکتور/مرحله‌ای تکمیل‌نشده/مرحله‌ای تکمیل‌شده/پرداخت آنلاین/فیش آپلود شده/رد شده/نیاز به اقدام مجدد; status select adds «نیاز به بررسی مالی»; row 89663612 پیش‌فاکتور with 7 inline actions (ثبت واریز و فیش، بارگذاری فیش، ارسال لینک انتخاب، ارسال مجدد لینک پرداخت، کپی لینک، ویرایش پیش از پرداخت، وضعیت و تاریخچه مالی) | One «وضعیت» column mixes invoice/stage/review. Seven equal buttons per row. |
| رفتار مشتریان | search; list item «بسی · 89663612 · پیش‌فاکتور صادر شده · ۰ رویداد»; **«در حال بارگذاری…» still visible above loaded data** | Loading/data shown together (F05-type). |
| درخواست HR | form: seller (#6), type جابجایی/قطع همکاری, destination supervisor (**list contains self #5**), reason, «ثبت درخواست»; text «اعمال واقعی فقط بعد از تایید منابع انسانی»; table #، نوع، فروشنده، درخواست‌دهنده، مقصد، وضعیت، مرحله فعلی، دلیل، عملیات — empty | Self as destination = noop (Gate 0 validation). |
| کیف پول | not opened (F07) | Personal. |
| گزارش جامع (#sn-reports-panel) | 10 report cards; type «کل دوره یا بازه» / «وضعیت در لحظه»; date/time; **سمت: معاون فروش، مدیر فروش، سرپرست ارشد، سرپرست، فروشنده**; «فرد و زیرمجموعه» | Upper roles listed in a Supervisor scope picker (visibility ≠ scope; confusing). |
| Shell | sidebar with 12 items + header (گزارش جامع، رمز من، خروج، حالت شب، اعلان ۰) | 12 flat destinations, no grouping. |

---

## 2. Current → Target mapping

| Current section | Product purpose | Decision | Target destination | Component (Shared / New) |
|---|---|---|---|---|
| فروشندگان و آمار | Direct team capacity, daily | KEEP + SIMPLIFY (freshness instead of «…», one KPI strip, no fake 0) | **نمای تیم** › ظرفیت تیم مستقیم | StatStrip+MetricMeta (Shared ext), FreshnessIndicator (Shared new), DataTable (Shared), **TeamCapacityPanel** (Sales-mgmt family) |
| (missing) exception queue | Daily responsibility | ADD as presentation of existing exceptions only | **نمای تیم** › نیازمند توجه | **AttentionList** (Shared new) |
| تبدیل‌کنندگان و آمار | Converter workload | MERGE; rate hidden (OPD-07) | **تبدیل** › تبدیل‌کنندگان | DataTable + MetricMeta «تعریف نشده» |
| تبدیل‌های من | Personal conversions | CONDITIONAL (only when assigned-to-self > 0) | **تبدیل** › تبدیل‌های من (secondary queue, hidden when empty) | Seller conversions pattern (Shared) |
| تخصیص شماره (assign) | Feed direct sellers | KEEP + preview/eligibility/item results | **تخصیص و برگشت** › تخصیص | **SelectableDataTable, BulkActionBar, BulkReview, BulkResult, StepIndicator** (Shared new), **RecipientList** (Sales-ops shared candidate) |
| تخصیص شماره (return to my panel) | Pull eligible unused cases back | KEEP, high-risk → eligibility with reason/proof | **تخصیص و برگشت** › برگشت | **EligibilityPanel, OwnershipGrid, ConfirmStep** (Shared new) |
| جدا کردن سرنخ زنده/قدیمی | Legacy unassign | MERGE entry; semantics separate (source = «سرنخ قدیمی») | **تخصیص و برگشت** › برگشت (source filter) | same Return workflow + Source tag |
| لاگ و خطا | Distribution troubleshooting | MOVE | **تخصیص و برگشت** › سوابق عملیات | **OperationLog** (Shared new) |
| آماده‌های تبدیل | Ready queue management | KEEP + RENAME «آماده تخصیص تبدیل» | **تبدیل** › صف تیم | SelectableDataTable + Bulk (Shared) |
| فاکتورها | Team invoice exceptions | KEEP; split status into facets; one primary action | **فاکتورها و استثناها** | **FinancialFacets** (Shared new), Drawer (Shared) |
| اشتراک‌های فروشنده | Pre-payment subscription correction | MOVE into invoice drawer | **فاکتورها و استثناها** › drawer › اشتراک | Drawer section + lock note |
| رفتار مشتریان | Customer activity | MOVE to shared customer context | **سوابق مشتری** | **CustomerContext** drawer (Shared new, reuses Timeline) |
| درخواست HR | Transfer/termination request | KEEP; chain visible; HR final | **درخواست‌های HR** | **ApprovalChain** (Shared new), Field (Shared) |
| گزارش جامع | Scoped reports | MOVE to Reports destination; scope limited to own subtree | **گزارش‌ها** | **ReportCatalog / ReportView** (Shared new) |
| کیف پول | Own earnings | KEEP personal, not team revenue | **کیف پول من** (personal, last nav group + user menu) | Wallet pattern (Shared, from Seller) |
| Header (حالت شب، رمز، خروج) | Account | KEEP in shared header/user menu | Header + user menu | Header, Menu, Theme/Density/Focus (Shared) |

## 3. Information Architecture

Top tabs (Shared RoleNavigation) — 3 groups, 8 destinations, priority-overflow into «بیشتر» when width is short:

1. **کار تیم**: نمای تیم (home) · تخصیص و برگشت · تبدیل · فاکتورها و استثناها
2. **زمینه و درخواست**: سوابق مشتری · درخواست‌های HR · گزارش‌ها
3. **شخصی**: کیف پول من

Secondary navigation = Shared QueueTabs inside each destination:
- تخصیص و برگشت: تخصیص به فروشنده · برگشت به پنل من · سوابق عملیات
- تبدیل: صف تیم · تبدیل‌کنندگان · تبدیل‌های من (conditional)
- فاکتورها و استثناها: نیازمند اقدام · در بررسی مالی · رد شده · مرحله‌ای ناتمام · پیش‌فاکتور · تکمیل‌شده · همه

Mobile bottom nav (5): نمای تیم · تخصیص · تبدیل · فاکتورها · بیشتر (rest + display settings).

## 4. Product Gaps (recorded — DO NOT IMPLEMENT)

| ID | Gap | Safe baseline in prototype |
|---|---|---|
| PG-1 | No defined escalation channel/owner for unresolved exceptions (job 12; Gate 0 «escalation→failure owner» without mechanism) | «ارجاع» shown disabled + explanation |
| PG-2 | SLA / overdue definition beyond callback due date not defined (Audit «نیاز SLA مصاحبه شود») | Only «تماس سررسیده» (existing callback date) counted |
| PG-3 | Conversion rate / performance cohort (OPD-07) | Rate column shows «تعریف نشده», never 0٪ |
| PG-4 | Release of cancelled-invoice cases (OPD-03) | Return blocked, reason «سیاست آزادسازی تعیین نشده» |
| PG-5 | Supervisor as HR reviewer: no request type below Supervisor creates reviewable requests | No «review» action rendered; chain read-only |
| PG-6 | Indirect-user write (OPD-10) | Indirect section view-only; never a recipient |
| PG-7 | Report scope picker lists roles above Supervisor (live) | Target scope picker lists only own subtree |
| PG-8 | Manual stats load (504) root cause | Freshness + manual refresh; no auto-zero |
| PG-9 | Recipient ACK / handoff receipt (OPD-02) | Result shows «ثبت شد» (commit), never «دریافت شد» |
| PG-10 | Wallet render may autopost (F07) | Prototype wallet is read-only view; flagged |

## 5. Proposed shortcuts (documented, NOT implemented)

| Key | Proposal | Conflict check |
|---|---|---|
| X | toggle selection of cursor row in selectable tables | free in Seller |
| Shift+A | open review for current selection | free |

Implemented shortcuts are only the shared Seller set: Ctrl+K, /, ?, Esc, F, J/K, Enter, digits for queues.

## 6. Structure (non-destructive)

```
seller-redesign-prototype/
  index.html, app.js, help.js, styles.css, help.css, data.js   ← Seller V1 FROZEN (untouched)
  shared/crm-core.js      ← shared runtime (shell, nav overflow, drawer, toast, tooltip, palette, keyboard, states)
  shared/help-engine.js   ← shared Help Center / Tour / Welcome / Demo engine (content via registry)
  shared/crm-ext.css      ← shared design-system extensions (tokens unchanged)
  supervisor/index.html, app.js, data.js, help-content.js, supervisor.css   ← role layer
  supervisor/qa/          ← QA screenshots + gallery (index.html)
  supervisor/device.html, frame.html   ← capture helpers (phone frame / settled tablet frame)
```

Seller V1 still runs its own frozen runtime copy; migrating Seller onto `shared/` is a post-freeze engineering step with regression QA.

## 7. Component classification

| Component | Class | Notes |
|---|---|---|
| AppShell, Header, Theme/Density/Focus, Menu, Tooltip, Toast, CommandPalette, Drawer/Sheet, Timeline, StatusPill, Button, Field, DataTable (stackable), Pagination, Empty/NoResult/Error/Loading/Offline | Shared (reused from Seller V1) | no fork |
| RoleNavigation + groups + priority overflow («بیشتر») | Shared — new | config-driven per role |
| MetricMeta (meaning + time basis tooltip; stale / unavailable / undefined) | Shared — extends StatStrip | never renders fake 0 |
| FreshnessIndicator + ScopeBadge + PageBanner (stale / incomplete) | Shared — new | |
| StateBlock kinds: unauthorized, stale, incomplete, conflict, partial, retryable, unknown | Shared — extends states | |
| SelectableDataTable, BulkActionBar | Shared — new | MIS / HR / Finance reuse |
| BulkReview, BulkResult (item-level, retry failed only, reconcile unknown) | Shared — new | |
| StepIndicator | Shared — new | |
| EligibilityPanel (reason/proof), OwnershipGrid (custody/original/next actor/commission), ConfirmStep | Shared — Sales-operations family | |
| FinancialFacets (invoice / stage / evidence / finance review / completed sale / next actor) | Shared — new | Seller, Manager, Finance |
| ApprovalChain | Shared — new | HR, Finance |
| AttentionList (exception inbox) | Shared — new | |
| ReportCatalog / ReportView (grain, basis, scope, freshness, export) | Shared — new | |
| OperationLog | Shared — new | |
| RecipientList (direct-only, capacity hints) | Shared candidate (Sales ops / MIS distribution) | |
| Glossary view in Help Center | Shared — new | |
| TeamCapacityPanel | Supervisor / Sales-management family | not a leaderboard |
| Team attention categories, Supervisor KPI definitions, assignment/return/conversion copy, Supervisor tours | Supervisor-specific | |

## 8. Shared Design System delta

- No new color, radius, type size, spacing or shadow token. All new pairs checked ≥ 4.5:1 in light/dim/dark.
- New CSS: `shared/crm-ext.css` (components above) + header rules (`.brand` no-shrink, demo-tag hidden ≤1200, crumb hidden ≤900), mobile `.sum-grid` 2-col, `.drawer.wide` (660px).
- New icons in shared set: inbox, flag, chart, shield, question, eye, briefcase, split, unlink, download, user, compass.
- New status namespaces in role layer: eligibility (قابل تخصیص/قابل برگشت/مسدود/تغییر کرده/نیاز به تطبیق), bulk outcome (ثبت شد/ناموفق·قابل تکرار/تعارض/نامعلوم), finance_review, payment_stage, hr_request — all icon + text, tones by existing semantics (orange = waiting on others/conflict, amber = attention/unknown, red = blocked/failed, teal = informational/eligible, green = completed).
- Separator rule: no «·» adjacent to Persian digits (reads as «۰»); use «،».
