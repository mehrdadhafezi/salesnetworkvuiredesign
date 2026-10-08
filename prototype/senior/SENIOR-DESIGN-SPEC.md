# Senior Supervisor Redesign — Design V1 (SN-201)

Status: DESIGN PROTOTYPE · no plugin / backend / database / permission / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/SENIOR-SUPERVISOR-PRODUCT-SPEC.md`, `CRM-CROSS-ROLE-ARCHITECTURE-V1.md`, `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 frozen shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `senior/index.html`. Deep links: `?view=ov|perf|dist|ready|inv|hr|rep|cases|wallet|mine&theme=&density=&focus=1&sim=…&open=kind:id`.

## 1. Current → Target mapping
| Current (live) | Decision | Target destination |
|---|---|---|
| ss-overview | KEEP + ADAPT | **نمای چندتیمی** (team rows, exceptions, own pool separated) |
| ss-team + ss-sellers + ss-converter-stats | MERGE (presentation only) | **ساختار و عملکرد** (6 grain modes) |
| ss-distribution | KEEP | **توزیع و برگشت** (assign → Supervisor; exception path; eligible return; operation log) |
| ss-repeat-actions | RENAME label only | **آماده‌های تبدیل تیم‌ها** (readonly) |
| ss-invoices | KEEP, facets split | **فاکتورها و استثناها** |
| ss-hr-requests | KEEP | **درخواست‌های HR** (step review ≠ final apply) |
| report center | KEEP shared | **گزارش‌ها** (multi-team scope) |
| ss-leads | KEEP + source-explicit | **پرونده و سوابق منبع** (legacy facet baseline) |
| ss-wallet | KEEP personal | **کیف پول من** (pure read) |
| ss-my-conversions | CONDITIONAL | **تبدیل‌های من** (hidden unless eligible) |
| gaps F01–04/F09 | ADVANCED | diagnostics drawer «تطبیق شاخص‌ها» (not in daily nav) |
Internal IDs/routes/statuses are unchanged; labels are presentation only.

## 2. Senior IA
Reuses Supervisor RoleNavigation (groups + priority overflow «بیشتر»). Groups: **کار تیم‌ها** (ov, perf, dist, ready, inv) · **زمینه و درخواست** (hr, rep, cases) · **شخصی** (wallet, mine-conditional). Mobile bottom nav: نما، عملکرد، توزیع، فاکتورها، بیشتر. Secondary navigation = shared QueueTabs.

## 3. Screens
- **Overview** — scope identity + evaluated_at; 5 spec metrics only (direct Supervisors, active/all Sellers, team workload, open pre-invoices, own pool — pool is separate from team workload). Team table: responsible Supervisor, workload (count + proportional bar, no score/rank), ready count, top exception + responsible actor, **MetricCoverage** chip. Direct Seller in a separate group row. Exception list (AttentionList): subject / responsible actor / state / reason / allowed next action; opens a detail drawer; **no escalation command** (SG-02).
- **Performance Explorer** — **DimensionSwitcher**: Team · Supervisor · Seller · Conversion · Invoice · Case workload; each shows its own *grain* label and never mixes grains. **AttributionBasis**: Current Hierarchy vs Historical Attribution (Team/Invoice modes) with Event Actor shown separately; unknown history stays «ناشناخته» (own row, never today's parent). **ScopeBreadcrumb** drilldown narrows, never widens. Direct/indirect shown by relationship badges («غیرمستقیم · فقط مشاهده»). Totals are distinct-fact totals; `—` + reason for unavailable; «تعریف نشده» / «داده ناکافی» for conversion rate and empty cohorts (never 0%).
- **Distribution & Return** — primary path Senior → Supervisor (single or count-per-Supervisor). **Exception path** (direct-to-Seller) is a collapsed, dashed, conditional group: relationship, bypassed level, mandatory reason, current custody, eligibility, dependency conflict; **commit disabled** («نیازمند سیاست SD-01») — no approval policy invented. Return reuses safe-return pattern but scoped to *own* transfers: Eligible / Blocked / Conflict / Unknown, with proof («انتقال توسط»), reasons (financial dependency, consumed V4, transfer by Supervisor, MIS source, cancelled OPD-03, unresolved link) and responsible next actor. Preview ≠ Apply stated; bulk return flagged conditional. Results: per-item, partial, retry-failed-only, outcome-unknown → reconcile.
- **Ready — Team View** — readonly, grouped by team/responsible Supervisor; owner, next actor, status namespace, source badge, allowed financial context; waiting time «تعریف نشده». No assign / escalate button; unresolved owner (inactive Supervisor) shown as exception, not auto-assigned.
- **Invoices & Exceptions** — Invoice status, Payment stage, Finance review, Paid/Remaining, Next actor, reason, LinkConfidence as separate columns. Drawer: facets, ownership (5 concepts), current-vs-historical team, action policy (Read allowed · Assist needs explicit permission, unverified · Finance-only locked). No approve/reject/refund/post.
- **HR** — queues: waiting for you / in chain (observe) / closed. «تأیید این مرحله و ارجاع به مرحله بعد» is visibly not «اعمال نهایی توسط منابع انسانی»; reject needs a reason; no bulk review; cross-scope destination flagged (SD-04).
- **Reports** — shared catalog; scope picker lists only the authorised subtree; basis: event-in-range / current; historical-as-of disabled (no proof); metadata chips: grain, basis, scope, definition version, coverage, freshness, export permission (button disabled — unverified).
- **Case Explorer** — baseline legacy facet with «پوشش ناقص» banner, SourceBadge, raw+mapped status, LinkConfidence, source-history drawer; phone labelled as discovery only; same-phone cases stay distinct. Unified view option disabled.
- **Wallet** — personal, read-only, «reading posts nothing». **Personal conversions** — conditional (toggle in Help → demo, `?mine=1`).

## 4. Shared Component Delta
| Component | Class | Notes |
|---|---|---|
| AppShell, Header, RoleNavigation, PageHeader, KPI/MetricMeta, FreshnessIndicator, ScopeBadge, DataTable, SelectableDataTable, Bulk bar/review/result, Drawer, Timeline, status pills, EligibilityPanel (`checks`), OwnershipGrid, FinancialFacets, ApprovalChain, AttentionList, Reports, OperationLog, Help/Tour, Toast, Tooltip, CommandPalette, system states, Theme/Density/Focus | **REUSED unchanged** | OwnershipGrid now renders 5 concepts incl. Event Actor and Credit Owner in the role layer (Supervisor untouched) |
| **ScopeBreadcrumb** | SHARED CRM — added | `crm-ext.css` |
| **DimensionSwitcher** | SHARED CRM — added | one grain at a time |
| **AttributionBasis** (+ grain label) | SHARED CRM — added | current / historical / event actor |
| **MetricCoverage** | SHARED CRM — added | complete · partial · stale · reconcile · undefined · failed |
| **LinkConfidence / SourceBadge** | SHARED CRM — added | |
| `.btn-danger` variant | SHARED CRM — added | presentation of «رد با دلیل» |
| PerformanceExplorer | **not created** | composed from DimensionSwitcher + DataTable (no speculative component) |
| Workload bar, exception-path panel, ready team grouping, action-policy list | SENIOR-SPECIFIC | `senior.css` |
No new colour/radius/type/spacing/shadow tokens. Additions are additive to `crm-ext.css`; Seller and Supervisor files are untouched.

## 5. System states
Shared states reused: loading, empty, no-result, unauthorized, stale, incomplete, conflict, partial success, retryable failure, outcome unknown, offline, table/page/drawer error. Multi-team additions (Help → demo): **stale team metric** (`sim=stale`), **partial source coverage** (`sim=incomplete`), **one team failed / others loaded** (`sim=teamfail`), **unknown historical attribution** (`sim=unknownhist`).

## 6. Help / Tour
Shared engine; Senior content only: welcome, tours `senior` (nav, exceptions, teams, explorer, distribution, ready, invoices, HR, reports, help), `perf`, `dist`; Senior glossary (5 ownership concepts, current vs historical, direct/indirect, exception path, coverage, reconcile, undefined, link confidence, step vs final). Tours only navigate/open read-only panels.

## 7. QA
See `qa/QA-REPORT.md`.

## 8. Known limitations
- Mock data; one dataset; numbers are illustrative. Prototype only represents target UX.
- Multi-team behaviour not validated on real multi-team data; real roles not runtime-tested (HR/Finance remain NOT LIVE VERIFIED).
- Drawer/Help engine and Seller/Supervisor frozen runtimes are reused as-is; Seller still runs its own copy of the runtime.
- No screen-reader pass with real AT; ARIA/focus verified by structure and automated checks only.

## 9. Product Gaps / Deferred (not implemented; surfaced as conditional/disabled/informational)
SG-01 metric dictionary/coverage reconcile (F02/F03) · SG-02 escalation command & failure owner — no «ارجاع» action anywhere · SG-03/SD-02 personal conversion receiving · SG-04/SD-01 skip-level policy — exception commit disabled · SG-05/OPD-07/SD-05 ready/blocked/age/rate definitions — «تعریف نشده» · SG-06 unified case coverage — legacy facet only · SG-07/SD-03 invoice assist fields/permissions — shown unverified · SG-08 all-link return protection (Gate 0 F01) — design shows blocks + preview≠apply · SD-04 HR cross-scope destination · OPD-03 cancelled-case release · OPD-06 sale-completion boundary.

## 10. Freeze Review
**Fixed during QA:** table overflow at 1366 (column priority); mobile overflow from demo switch and HR queue grid; team grid breakpoint order; attention text truncation; contrast of muted dash, selected-queue count, soft buttons in dim/dark; drawer footer wrap on phones; fake zero for reconcile-pending collected amount.
**Verified:** see QA report (structural, interaction, matrix: 5 widths × 3 themes × 2 densities × focus, contrast ≥4.5:1 sampled, no JS errors).
**Shared components added:** ScopeBreadcrumb, DimensionSwitcher, AttributionBasis, MetricCoverage, LinkConfidence/SourceBadge, btn-danger variant. **Senior-specific:** workload bar, exception-path panel, ready team grouping, action-policy list.
**Cross-role consistency:** same tokens, shell, drawer, states, Help engine as Seller/Supervisor.
**Ready for next role:** yes (SN-202 may reuse the shared delta).
**SENIOR SUPERVISOR DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision).
