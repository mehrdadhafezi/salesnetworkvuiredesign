# Sales Deputy Redesign — Design V1 (SN-203)

Status: DESIGN PROTOTYPE · no plugin / backend / database / API / permission / capability / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/SALES-DEPUTY-PRODUCT-SPEC.md` → `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 + Senior V1 (SN-201) + Manager V1 (SN-202) shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `deputy/index.html`. Deep links: `?view=ov|perf|exc|rep|alloc|hr|mon|inv|diag|wallet&theme=&density=&focus=1&sim=…&open=kind:id` (+ `flow=`, `sel=`, `pm=`, `pb=`, `tb=`, `scope=`, `aq=`, `rf=`, `dq=`, `mq=`, `monAuth=`, `ec=`, `em=`, `hq=`, `iq=`, `report=`, `rscope=`).
Result label: **SALES DEPUTY DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision).

## 1. Current → Target mapping (Product Spec §3, §5)
| Current destination | Decision | Target (labels only; routes/IDs/statuses unchanged) |
|---|---|---|
| `sd-overview` | KEEP + ADAPT | **نمای رهبری** (branch table + exception classes + metric trust; no vanity KPI) |
| User ID | identity only | context line, "not a KPI" |
| invoice-status free-text filter | SIMPLIFY | namespaced queues in **زمینه فاکتور** |
| `sd-hierarchy`, `sd-managers`, `sd-seniors`, `sd-supervisors`, `sd-sellers` | MERGE presentation | one **سلسله‌مراتب و عملکرد** explorer (Manager → Senior → Supervisor → Seller → Case/Invoice); entities and parent data kept |
| `sd-distribution` | KEEP + CONDITIONAL | **تخصیص مستقیم (مشروط)** (Manager = ordinary path; every lower level = exceptional bypass) |
| `sd-invoices` | MOVE contextual, read-only | **زمینه فاکتور** |
| `sd-wallet` | MOVE Own Account | **کیف پول من** (pure read) |
| Shared report center | KEEP | **گزارش‌ها** |
| Exception coordination / coverage / metric trust | ADD (target, no new command) | **استثناها**, metric-trust table, **عیب‌یابی شاخص و منشأ** |
| HR entry | CONDITIONAL | **زمینه HR (مشروط)** (assigned current-reviewer step only) |
| Archive / extra-number aggregates | CONDITIONAL READ | **پایش آرشیو و شماره اضافه (مشروط)** (never a Manager queue) |

## 2. Deputy IA
Groups in the shared RoleNavigation: **رهبری** (ov, perf, exc, rep) · **مشروط** (alloc, hr, mon) · **زمینه** (inv) · **پیشرفته** (diag) · **شخصی** (wallet). Mobile bottom nav: رهبری، عملکرد، استثناها، گزارش‌ها، بیشتر. Old level-specific tabs are not primary destinations.

## 3. Screens
- **Leadership Overview** — 4 KPIs (branches needing attention · open exception classes · exceptions with unknown owner · trustworthy metrics), each with scope/basis tooltip; no ranking/score/target/rate. Branch table (alphabetical, "not a ranking"): active sellers, cases at sellers (workload bar), distribution constraint (inactive actors), largest exception + responsible owner, next actor, **coverage chip**. Exception-class list (AttentionList). **Metric trust table** for the six approved KPIs: value shown only when definition and coverage allow; Open Pre-invoices = «نیازمند تطبیق (F02)», Total Cases = «پوشش ناقص (F03)», Sales Completed = «تعریف نهایی نشده (OPD-06/07)», Legacy Leads = «فقط منبع قدیمی». Scope is stated as "not the entire organisation".
- **Hierarchy & Performance** — DimensionSwitcher (Manager · Senior · Supervisor · Seller · Invoice · Case workload), ScopeBreadcrumb, TimeBasis (Current Snapshot / Event Range; Historical As-Of disabled without proof), AttributionBasis (current / historical). A "four concepts" legend separates Current Hierarchy, Historical Attribution, Event Actor, Credit Owner. Unique total is not built from overlapping sub-totals; ancestors and descendants are never summed. Supervisors directly under a Manager are shown with their own badge.
- **Exceptions** — all 11 Product-Spec categories as owner-aware rows: subject · branch · responsible owner (or **UNKNOWN**) · state · why attention is needed · allowed Deputy action · next actor · resolution signal. Drawer adds prohibited actions. No escalation / ACK / SLA / notification (SD-G07); bulk exception action = «نیازمند اعتبارسنجی».
- **Conditional Direct Allocation** — banner "CONDITIONAL / NEEDS POLICY (SD-G05)". Primary path = Manager. «عبور از مدیر (استثنایی)» is a collapsed, dashed group with Senior/Supervisor/Seller sub-groups (collapsed). Out-of-scope Manager and inactive/unverified recipients are disabled with reasons. Review drawer: custody, recipient relationship, scope, SourceRef, recipient eligibility, financial dependency, impact, **Apply-recheck expectations**, explicit acknowledgement for bypass, optional (not mandatory) reason. Results use the shared BulkOutcomeStrip (7 states reconcile; retry only failed; unknown → reconcile). States: conflict (custody changed), recipient changed before apply, out-of-scope, partial, retryable, unknown, empty pool.
- **Return** — preview only (SD-G01 / F01). Eligible · Blocked · Conflict · Unknown/Reconciliation-needed; rank ≠ recall; cancelled/rejected ≠ release (OPD-03); fail-closed on unresolved links; commit disabled.
- **Scoped Invoice Context** — read-only. Columns: invoice · Manager›Senior›Supervisor›Seller chain · Case/source + link confidence · lifecycle · payment stage · Finance review · total/paid/remaining (تومان) · next actor + reason. Drawer action policy lists every non-read action as «نیازمند اعتبارسنجی / محدود / در این نقش نیست»; no assist/write/finance buttons. "Recent sample ≠ total" (SD-G12).
- **HR Context** — queues: waiting for you (current reviewer) / in chain (observe) / closed. Drawer separates «تأیید این مرحله» from «اعمال نهایی توسط منابع انسانی»; shows current / prior / next reviewer, HR final state, failed apply; reject needs a reason; termination shows OPD-05; no bulk, no submit form, no HR admin.
- **Archive / Extra-number Monitor** — conditional read aggregates per Manager; `null` renders as «مجاز نیست» (never 0); archive record drill and request-outcome context = «نیازمند اعتبارسنجی»; demo toggle for partial / unauthorized / unavailable. No review, revive, delete or decision controls.
- **Diagnostics (advanced, read-only)** — F02 (tile 0 vs pre-invoice rows, predicate evidence), F03 (source-coverage matrix: counted / not counted / unproven; zero legacy ≠ zero cases), F04 (Manager-observed 0 vs 1; **not reproduced in Deputy today**, cause UNVERIFIED), lineage (unresolved links, unknown owner, no phone joins). No repair or merge control.
- **Reports** — scope picker lists only permitted Managers (out-of-scope option disabled); Manager grouping → Senior/Supervisor drill; time basis Event Range / Current / Historical (disabled); metadata chips (grain, basis, scope, definition version, coverage, bounded 60–120 rows, export permission, **sensitive fields restricted**); export disabled.
- **Wallet** — personal, pure read.

## 4. Shared Component Delta
| Component | Class | Notes |
|---|---|---|
| AppShell, Header, RoleNavigation, PageHeader, KPI/MetricMeta, Freshness, ScopeBadge, DataTable, SelectableDataTable, Bulk bar, Drawer, Timeline, pills, EligibilityPanel, OwnershipGrid, ApprovalChain, AttentionList, Reports, Help/Tour, Toast, Tooltip, CommandPalette, system states, Theme/Density/Focus, ScopeBreadcrumb, DimensionSwitcher, AttributionBasis, MetricCoverage, LinkConfidence, BulkOutcomeStrip | **REUSED unchanged** | Seller/Supervisor/Senior/Manager files untouched |
| **BranchCoverage / source-coverage matrix** (`.cov-matrix`) | SHARED CRM — **added** (`shared/crm-ext.css`, additive) | icon + text; unknown/not-counted never zero; no new tokens |
| Owner-aware exception row & drawer | composed from AttentionList + Drawer (role layer) | no shared CSS; candidate for SN-207 |
| Metric-trust table, Metric Reconciliation cards (`.rc-*`) | DEPUTY composition mirroring Manager vocabulary | candidate to promote in SN-207 |
| ConditionalAllocationContext (primary vs bypass groups, preview card) | DEPUTY-SPECIFIC | `deputy.css` |
| Step-vs-final HR comparison (`.step-vs`) | DEPUTY-SPECIFIC | candidate to promote with HR role (SN-205) |
`deputy.css` starts from the Manager layout block (duplicated by design, not imported, to avoid cross-role CSS coupling); promotion into shared CSS is deferred to SN-207. No new colour/radius/type/spacing/shadow tokens.

## 5. System states
Shared states reused (loading, empty, no-result, unauthorized, stale, incomplete, conflict, partial success, retryable failure, outcome unknown, offline, table/page/drawer error). Deputy additions (Help → demo): one branch failed (`sim=branchfail`), partial hierarchy (`sim=parthier`), stale branch, unknown historical attribution, empty pool, recipient changed (`flow=recipchg`), out-of-scope allocation (`flow=outscope`), unknown owner (exceptions), metric mismatch (diagnostics), monitor unauthorized / unavailable.

## 6. Help / Tour
Shared engine; Deputy content only: welcome, tours `deputy` (9 steps), `perf`, `alloc`; glossary (ownership concepts, current vs historical, snapshot/event/as-of, primary vs bypass, outcome states, coverage, F02/F03/F04, step vs final). Tours navigate/open read-only panels only.

## 7. QA
See `qa/QA-REPORT.md`.

## 8. Known limitations
- Mock data; one dataset; counts illustrative. Multi-Manager runtime not verified; HR and Finance runtime remain NOT LIVE VERIFIED; no real Deputy account tested.
- No screen-reader pass with real AT (structure/ARIA/focus automated only).
- `deputy.css` duplicates the Manager layout block until SN-207.
- Monitor authorization states are demo-switchable; real read authority is not simulated as granted.
- Allocation/return/HR/exception actions are presentation only; nothing is persisted.

## 9. Product Gaps / Deferred (shown as conditional / disabled / informational — never as capabilities)
SD-G01 F01 return protection — preview only · SD-G02 F02/F03 — diagnostics, not fixed · SD-G03 F04 — same-cohort reconcile, cause UNVERIFIED, not reproduced in Deputy · SD-G04 cohort/attribution/currency (OPD-06/07) — «تعریف نشده» · SD-G05 direct-allocation/skip-level policy — conditional, no approval chain · SD-G06 write authority across hierarchy — disabled recipients, visible ≠ writable · SD-G07 escalation ownership — no ارجاع/ACK/SLA · SD-G08 archive/extra-number visibility — conditional read only · SD-G09 HR entry/reviewer — assigned step only, OPD-05 · SD-G10 bulk exception/review — absent · SD-G11 pure-read wallet · SD-G12 invoice context / hierarchy load — bounded sample, partial-branch states. OPD-01/03/08/09/10 remain binding/deferred.

## 10. Freeze Review
**Fixed during QA:** exception table overflowed (action column off-screen) → fixed-layout columns; bypass recipient list collapsed into per-level groups; monitor null cells now say «مجاز نیست» instead of a bare dash; dim-theme contrast on reconciliation pills (4.44 → ≥4.5).
**Verified:** see QA report. **Cross-role consistency:** same tokens, shell, drawer, states and Help engine as Seller/Supervisor/Senior/Manager.
**SALES DEPUTY DESIGN V1 — READY FOR REVIEW.**
