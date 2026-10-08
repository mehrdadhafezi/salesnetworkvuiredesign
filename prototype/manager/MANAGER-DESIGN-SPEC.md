# Sales Manager Redesign — Design V1 (SN-202)

Status: DESIGN PROTOTYPE · no plugin / backend / database / API / permission / capability / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/SALES-MANAGER-PRODUCT-SPEC.md` → `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 + accepted Senior V1 (SN-201) shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `manager/index.html`. Deep links: `?view=ov|dist|perf|inv|arch|xnum|hr|rep|cust|wallet&theme=&density=&focus=1&sim=…&open=kind:id` (+ `flow=`, `sel=`, `pm=`, `tb=`, `cs=`, `ar=`, `xq=`).
Result label: **SALES MANAGER DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision).

## 1. Current → Target mapping (Product Spec §3, §5)
| Current destination | Decision | Target destination (labels only; routes/IDs/statuses unchanged) |
|---|---|---|
| `manager-overview` | KEEP + ADAPT | **عملیات و استثناها** (exception console + unit table; own pool separate) |
| User ID tile | REMOVE from KPI | identity context line only (account + ID, "not a KPI") |
| `manager-distribution` | KEEP | **توزیع** (direct recipients; skip-level conditional; return = preview only) |
| `manager-report` | MERGE facts contract | **عملکرد و تطبیق** (Senior → Supervisor → Seller; F04 reconcile mode; metric definitions) |
| `manager-invoices` | KEEP | **فاکتورها و استثناها** (action-specific policy; Finance locked) |
| `manager-archives` (5 reasons) | MERGE presentation / KEEP semantics | **کاوشگر آرشیو** (reason-aware; read/review first; revival deferred) |
| `manager-extra-number-requests` | KEEP / ADAPT provenance | **شماره اضافه** (decision queue + explicit approval side effects) |
| `manager-hr-requests` | KEEP | **درخواست‌های HR** (Manager step ≠ final HR apply) |
| Shared Reports | KEEP in Reports | **گزارش‌ها** (governed metadata; bounded rows; export conditional) |
| `manager-customer-actions` | MOVE context | **زمینه رفتار مشتری** (contextual view of the Shared Customer Profile; 8 truthful states) |
| `manager-wallet` | MOVE to Own Account | **کیف پول من** (pure read) |

## 2. Manager IA
Reuses the shared RoleNavigation (groups + priority overflow «بیشتر»). Groups: **کار** (ov, dist, perf, inv, arch) · **درخواست‌ها** (xnum, hr, rep) · **زمینه** (cust) · **شخصی** (wallet). Mobile bottom nav: عملیات، توزیع، عملکرد، فاکتورها، بیشتر. Secondary navigation = shared QueueTabs. Empty queues are never removed.

## 3. Screens
- **Operations & Exceptions** — 5 KPIs, each with scope/time-basis tooltip (exception classes · my allocatable pool · open pre-invoices · extra-number waiting · HR step waiting); User ID is not a KPI. Unit table (3 Seniors incl. one inactive; Manager-direct Supervisor/Seller group) with workload bar (no rank/score), top exception + responsible actor and **MetricCoverage** chip. Exception console (AttentionList) = the 10 spec exception classes; each opens a drawer: subject · owner · state · Next Actor · resolution signal · «اقدام مجاز شما» · «ممنوع در این پنل». No escalation command, SLA, or notification (M-G11). «حل‌شده» is explicitly not «released/allocatable».
- **Distribution** — primary: direct recipients (Senior; Manager-direct Supervisor/Seller). **Skip-level** is a collapsed, dashed, CONDITIONAL group: recipient eligibility (active + provable write eligibility) is separate from visibility; review shows bypassed level(s), «سیاست مصوب نیست (M-G08)», optional traceability reason, and an explicit acknowledgement before commit — the current technical path is kept, no approval chain invented. Count-per-recipient mode is direct-only. **Review ≠ apply** is stated. **Result** uses the shared **BulkOutcomeStrip**: requested · eligible · applied · skipped · rejected · failed · unknown (counts reconcile); retry only failed; unknown → reconcile, never blind retry.
- **Return** — preview only. F01 banner (M-G01): commit is disabled («مشروط به F01»); rank ≠ authority; fail-closed on unknown links; cancelled/rejected ≠ release (OPD-03); the five ownership concepts are shown.
- **Performance & Reconciliation** — **DimensionSwitcher**: Senior · Supervisor · Seller · Invoice · Case workload · Reconcile; **TimeBasis** switch (Current Snapshot / Event Range; Historical As-Of disabled without proof). Columns carry a snapshot/event tag and never mix. Attribution basis (current / historical; unknown row stays «ناشناخته»). **Reconcile** mode shows F04 (0 vs 1) as an analytic Conflict/Incomplete state: five unverified comparability checks, cause UNVERIFIED (hypotheses only), no fix control. **«تعریف شاخص‌ها»** drawer renders the 6 KPI definitions with `name/grain/source/cohort/time_basis/scope/formula/freshness` and the still-open cohort decisions.
- **Invoices & Exceptions** — separate columns for invoice status, payment stage, Finance review, paid/remaining, next actor+reason, LinkConfidence. Drawer action policy is per action: Read ✓ · copy link (assist) ✓ · resend (assist + send effect, outcome unverified; request ≠ delivered) · receipt/edit → NEEDS VALIDATION (OPD-10, not copied from Senior) · Finance approve/reject/refund/post/commission locked · cancel/relink restricted. Link to customer-behaviour context.
- **Archive Explorer** — 5 reasons as QueueTabs with their own rule text (no-answer 3 attempts + 3 days; assessment 3 days; subscription/product/product* 5 days). Columns: case/source/LinkConfidence · reason · age since archive · Seller/team · financial dependency · **displayed owner** (renderer priority, labelled «not custody / not credit owner») · resolution eligibility (late payment can resolve per existing rule, runtime unverified; partial payment ≠ revival). 200-row bound banner; authorized-empty is bounded. Revive/reassign/bulk resolve → «نیازمند اعتبارسنجی / معوق» (M-G09/M-G10); delete → not allowed.
- **Extra-number Requests** — queues (waiting on me / decided). Row shows requester (inactive flag → NEEDS VALIDATION), phone as discovery only, duplicate check **with its real scope** (same Seller, pending/approved only), status/decider, result lead (legacy) + CaseRef mapping «اتصال نامشخص». Drawer states the approval side effect (legacy lead created, supervisor=0, source tag; not V4; rollback leaves it pending). Reject reason is **recommended, not forced** (current handler). Failure → request stays pending; unknown → reconcile. 100-row bound; no bulk (M-G10); Admin-routing note.
- **HR Requests** — queues waiting-on-you / in-chain (observe) / closed; four distinct states (step approved · pending HR · applied · failed apply); reject needs a reason; termination shows OPD-05 handover undefined; cross-scope destination flagged; no bulk review.
- **Reports** — scope picker lists only the authorised subtree; basis event/now (historical disabled); metadata chips (grain, basis, scope, definition version, coverage, **bounded 60–120 rows**, export permission); Supervisor/Seller label row flagged for validation; export disabled (unverified).
- **Customer behaviour (contextual)** — 8 demo states: loading, partial, residual-loading (data arrived but loader still on — not success), loaded, error (≠ «no activity»), no-result, authorized-empty (scoped), stale. System issuance and customer actions are distinct types; the 0 vs 2 population note is shown as «may be different populations, being validated», not a defect claim. No event fabrication, scoring, or phone identity.
- **Wallet** — Own Account, pure read, «reading posts nothing»; not team revenue/commission.

## 4. Shared Component Delta
| Component | Class | Notes |
|---|---|---|
| AppShell, Header, RoleNavigation, PageHeader, KPI/MetricMeta, FreshnessIndicator, ScopeBadge, DataTable, SelectableDataTable, Bulk bar/review, Drawer, Timeline, status pills, EligibilityPanel, OwnershipGrid, FinancialFacets, ApprovalChain, AttentionList, Reports, Help/Tour, Toast, Tooltip, CommandPalette, system states, Theme/Density/Focus; **ScopeBreadcrumb, DimensionSwitcher, AttributionBasis, MetricCoverage, LinkConfidence/SourceBadge, `.btn-danger`** (all from SN-201) | **REUSED unchanged** | Seller/Supervisor/Senior files untouched |
| **BulkOutcomeStrip** (`.outcome-strip`) | SHARED CRM — **added** (`crm-ext.css`, additive) | presentation of the Cross-Role bulk outcome language; no new tokens |
| `AttributionBasis` variants `snap` / `range` / `asof` (TimeBasis) | reuse of `.basis` classes, role-layer config only | no CSS added |
| Operations Exception Console | composed from AttentionList + drawer (role layer) | no new shared CSS |
| Archive Explorer, Extra-number Review Queue, Metric Reconciliation indicator | MANAGER-SPECIFIC compositions of QueueTabs + DataTable + OwnershipGrid + EligibilityPanel; candidates to promote in SN-207 | `manager.css` (`.rc-*`, `.tb-tag`, `.owner-shown`, `.chk-ok`, `.bulk-note`, `.resid`) |
No new colour/radius/type/spacing/shadow tokens. Shared additions introduce presentation only, never a business capability.

## 5. System states
Shared states reused: loading, empty, no-result, unauthorized, stale, incomplete, conflict, partial success, retryable failure, outcome unknown, offline, table/page/drawer error. Manager additions (Help → demo): stale unit (`sim=stale`), partial coverage (`sim=incomplete`), one unit failed / others loaded (`sim=unitfail`), unknown historical attribution (`sim=unknownhist`), extra-number failure/unknown (`flow=xfail|xunknown`), skip-level review (`flow=skip|skipseller`), customer-behaviour states (`cs=`).

## 6. Help / Tour
Shared engine; Manager content only: welcome, tours `manager` (10 steps), `perf`, `dist`; Manager glossary (5 ownership concepts, current vs historical, snapshot/event/as-of, direct/indirect, skip-level, 7 outcome states, coverage, link confidence, archive, step vs final). Tours only navigate/open read-only panels.

## 7. QA
See `qa/QA-REPORT.md`.

## 8. Known limitations
- Mock data; one dataset; counts illustrative (never a workload benchmark). Prototype represents the target UX only.
- Multi-level behaviour not validated on real multi-team data; HR and Finance runtime remain NOT LIVE VERIFIED; no real Manager account was tested.
- Drawer/Help engine and Seller/Supervisor/Senior runtimes reused as-is; no screen-reader pass with real AT (structure/ARIA/focus verified by automation only).
- The Reports/Customer states are demo-switchable; real data loading behaviour (F05 root cause) is unknown and intentionally not simulated as "fixed".
- The `سرپرست ارشد` label is used for Senior to stay consistent with the accepted SN-201 design.

## 9. Product Gaps / Deferred (surfaced as conditional / disabled / informational — never as capabilities)
M-G01 F01 return protection — return is preview-only, commit disabled · M-G02 F04 parity — reconcile mode, cause UNVERIFIED, no count fixed · M-G03 F05 residual loading / event population — states + populations note · M-G04 Case/source + extra-number edge cases — flags + NEEDS VALIDATION list · M-G05 shell vs action authority — visible ≠ writable recipients · M-G06 pure-read wallet (F07 dependency) · M-G07 HR handover (OPD-05) · M-G08 skip-level policy — conditional, no approval chain · M-G09 manual archive revival — disabled · M-G10 bulk review (archive/extra-number/HR) — absent · M-G11 escalation ownership — no ارجاع/ACK/SLA · M-G12 KPI cohorts/attribution/units (OPD-06/07) — «تعریف نشده»/open cohorts · M-G13 field/context completeness — row-label validation notes. OPD-03/08/09/10 remain binding/deferred.

## 10. Freeze Review
**Fixed during QA:** invoice action-policy collapsed by default (now open so «Finance-only / OPD-10» is visible); partial-result demo showed «complete» with 3 items (now 6, all 7 states exercised); archive «next actor» no longer reuses the displayed owner (custody/next actor/credit stay separate); Senior label unified with SN-201 («سرپرست ارشد»).
**Verified:** see QA report (structural, interaction, matrix: 5 widths × 3 themes × 2 densities × focus, contrast ≥ 4.5:1 sampled, no JS errors; Senior/Supervisor/Seller regression smoke).
**Cross-role consistency:** same tokens, shell, drawer, states, Help engine as Seller/Supervisor/Senior.
**SALES MANAGER DESIGN V1 — READY FOR REVIEW.**
