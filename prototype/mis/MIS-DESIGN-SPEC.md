# MIS Redesign — Design V1 (SN-204)

Status: DESIGN PROTOTYPE · no plugin / backend / database / API / permission / capability / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/MIS-PRODUCT-SPEC.md` → `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 + Senior V1 (SN-201) + Manager V1 (SN-202) + Deputy V1 (SN-203) shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority; every number, ID and name is fictional.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `mis/index.html`. Deep links: `?view=today|import|cases|custody|rec|plan|quick|maint|diag&theme=&density=&focus=1&sim=…&open=kind:id` (+ `flow=`, `sel=`, `iq=`, `sq=`, `cq=`, `rf=`, `rq=`, `rep=`, `tb=`, `dq=`, `icls=`, `maintAuth=verified`, `palette=`). Flows: `partial|retry|unknown|conflict|importnew|importpartial|importunknown|blocked|retconflict|retunknown|lineage|issue|recon|planapprove|planmat|quick|maintblocked|maintauth`.
Result label: **MIS DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision; this PR is not merged by the author).

Role boundary: **MIS = Data Operations + Source Quality + Custody Traceability + Governed Reporting.** Not Sales leadership, not Finance, not HR, not unrestricted admin, not a repair console. Broad read ≠ repair authority. Read/report pages never mutate business state.

## 1. Current → Target mapping (Product Spec §3, §5)
| Current destination | Decision | Target (labels only; routes/IDs/statuses unchanged) |
|---|---|---|
| `mis-overview` | KEEP + SIMPLIFY | **امروز** — work queue with next responsible actor; overlapping counters in a separate non-partition table |
| `mis-import` | KEEP + ADAPT | **ورود و کیفیت داده** (runs/batches · files · source quality); readiness separated from cached health |
| latest «import result» (really a quick-assignment summary) | RENAME / SIMPLIFY | **Latest import vs latest operation** shown as two different things; operation counters are N/A, never 0 |
| `mis-batches-list` | KEEP | **مسئولیت و تحویل › تخصیص منبع** (source assign) + **برگشت (مشروط)** |
| `mis-pool` | KEEP | **مسئولیت و تحویل › آماده‌سازی استخر** (pool ≠ live lead) |
| `mis-preview` | MERGE | **ورود › کیفیت منبع** (breakdown) + report entry |
| `mis-batches` | MOVE | batch/file lifecycle under **ردیف‌های منبع** and **ورود › فایل‌ها**; destructive controls → Maintenance |
| delete / cleanup controls | MOVE | **نگهداری** (separate visual zone; capability untouched) |
| `mis-quick` | KEEP + CONDITIONAL | **تحویل سریع (مشروط)** (advanced, no default recipient) |
| `mis-distribution` + `mis-live` | MOVE | **برنامه‌ریزی (مشروط)** (rule → preview → save → approval → materialization) |
| `mis-logs` | KEEP | **عیب‌یابی › لاگ عملیات** |
| `mis-report` (+XLSX) | KEEP | **گزارش‌ها و تطبیق › گزارش‌ها و اعتماد** (explicit export, disabled in sample) |
| Reconciliation queue / trust context | ADD (target, no new command) | **گزارش‌ها و تطبیق › مسائل تطبیق** (diagnostic, read-only) |
The full 12-row old→new table is also visible in the prototype (عیب‌یابی › نقشه مقصدهای قدیمی).

## 2. MIS IA
Shared RoleNavigation groups: **PRIMARY** امروز · ورود و کیفیت داده · ردیف‌های منبع · مسئولیت و تحویل · گزارش‌ها و تطبیق → **SECONDARY / CONDITIONAL** برنامه‌ریزی · تحویل سریع → **ADVANCED** نگهداری · عیب‌یابی (visual separators; overflow into «بیشتر»). Mobile bottom nav: امروز، ورود، ردیف‌ها، تحویل، گزارش + «بیشتر». The 11 old tabs are not kept as equal top-level destinations. ActionZone badges (**عادی / پیشرفته · مشروط / نگهداری**, icon + text) appear in page headers.

## 3. Today / Work Queue
No KPI wall (3 compact indicators with scope/basis tooltips). Work queue = 9 items, each with **next responsible actor** (import partial, unknown write outcome, rows waiting for source assignment, blocked return F01, missing transfer proof, custody changed after preview, report-trust issues, non-verified lineage, health contradiction). Right column: **Latest import vs latest operation** (two different cards), **Readiness vs live read** (HealthContradiction; cause UNKNOWN; no repair control, no «import 0»), **Overlapping counters** table (each counter lists its own predicate and trust; legacy «returnable» is flagged F01-untrusted; per-batch «waiting» flagged «different predicate»). Explicit note: waiting + delivered + returnable ≠ partition of total. Missing/unavailable → «—», never 0.

## 4. Import & Data Quality
Four distinct concepts always labelled: **File · Batch · Import Run · Source Row** (the old «پرونده» = Batch). Queues: **Runs** (parsed · imported · skipped = duplicate + invalid · failed · unknown · reconcile check · File › Batch · time), **Files**, **Source Quality** (counters, equation, duplicate classes in-file/same-batch/other-MIS/legacy, parser warnings, limited-sample note). Persistent banner: import preview is not guaranteed side-effect free. **Import flow (3 steps)**: select → parse/review with acknowledgement gate → result with 7-state strip; an interrupted file ends as **Outcome Unknown**, never all-success. Run drawer: reconcile-first for Unknown (no blind re-import), retry only known-failed. Duplicates are stored and classified, never merged.

## 5. Source Cases / Case Lineage Inspector
Rows keyed by native MIS row ID (batch, file row_number); validity (valid/duplicate/invalid) and lineage (**تأییدشده · جزئی · متعارض · نامشخص**) are separate columns. **LineageLadder** drawer: native MIS row → legacy lead → V4 item → Dot case/invoice → alias relation (type + evidence) → logical Case («تعریف نشده» — resolver not implemented, MIS-G02), each node with its own confidence; **phone = discovery box** («هویت نیست», count of same-phone rows, no merge); **4-domain financial dependency union**; **7-concept ownership grid**; condensed timeline. No merge/relink/repair/reset control exists anywhere.

## 6. Custody / Delivery
Queues: **تخصیص منبع به مدیر** (source-field only; managers selectable only if active; ineligible shown disabled with reason) · **آماده‌سازی استخر** (no recipient; pool ≠ live lead) · **سابقه تحویل** (custody timeline) · **برگشت (مشروط)**. **OwnershipGrid-7**: Source Owner · Current Custody · Original Owner · Next Actor · Credit Owner · Event Actor · Recipient — never merged; «assigned_manager» is not current custody. Direct delivery to a Seller is not rewritten as a Manager receipt. Quick Delivery is a separate conditional view (no default recipient, bypass levels shown as skipped, reason recorded but no approval chain invented, APPLY gate).

## 7. Return Safety
Preview-only until F01 / MIS-G01. States: **Eligible (preview) · Blocked · Conflict · Unknown/Reconciliation Required** (+ N/A for undelivered). Drawer shows current custody + ownership grid, transfer proof, source linkage confidence, **financial dependency union across four domains**, blocked reason (including cancelled ≠ release, OPD-03), unknown dependency (fail-closed), next responsible actor, «preview ≠ permission», apply-recheck warning. Only Eligible rows are selectable; the commit control is permanently disabled («مشروط به F01»). No reset/clear-link affordance for ambiguous cases. Conflict offers refresh only; refresh lands on current truth (Unknown), never auto-eligible.

## 8. Reconciliation Workspace
14 issue rows covering all spec classes (identity, coverage, duplicate, alias, missing custody, hierarchy, invoice/report, stale report, partial import, unknown write, orphan-looking, health contradiction, + F01 financial dependency, + F09 time semantics). Each issue drawer: **evidence · affected entity · owner/domain · confidence · allowed MIS action · not allowed in this panel · resolution proof · next actor**. Read-only **bulk reconcile (conditional)** with the 7-state result («applied» = compared; nothing changed). No «Fix All».

## 9. Reports & Report Trust
Catalog (4) → report detail. **ReportTrustContext**: grain, cohort, scope, metric/definition version, gather window (+03:30), freshness, bounded/partial banner, **source-coverage matrix** (counted / not counted / unproven). **Time basis** segmented: Current Snapshot · Event Range + Current Gathered State · Historical As-Of (disabled, no proof); metrics are shown only under their own basis. Metric rows carry coverage state: F02 (Open Pre-invoices → «—», needs reconciliation), F03 (Total Cases → «—», partial), F04 and F09 as reconciliation issues; undefined metrics («فروش تکمیل‌شده») show «—», never 0. Export is conditional and disabled in the sample. Metric-definition drawer (Invoice Created, Open Pre-invoices, Pre-invoices Issued, Sales Completed, Total Cases, Legacy Leads Only).

## 10. Planning
Five separate stages (**Rule Draft · Plan Preview · Plan Save · Plan Approval · Materialization**) shown as a mini stepper per plan. Explicit consequence pills: «تأییدشده — هنوز لیدی ساخته نشده» (zero leads ≠ failure). Approval drawer: reason required; consequences list says no lead is created. Materialization drawer: APPLY gate, recheck note, special-path note (no SMS, no update/delete of existing leads), per-item partial result. No forecasting, quota or capacity score; candidate completeness «نیازمند اعتبارسنجی».

## 11. Maintenance Boundary
Zone legend (Normal / Advanced-Conditional / Maintenance) + a visually separated, hatched **MaintenanceBoundary** box. Authority line: «اختیار نگهداری مستقل: تأیید نشده (MIS-G07)». Each action opens an **impact review**: impact summary, affected objects, protected/unknown dependencies, reason, typed confirmation + acknowledgement, audit context, rollback warning. Delete file (file ≠ records), delete batch (blocked), cleanup (partial → only unlinked traces), schema rebuild (blocked: cause unknown) are reviewable; relink/repair, exceptional reassignment, derived-data rebuild are visibly unavailable. A demo toggle (`maintAuth=verified`) shows the enabled-only-after-reason+id+ack state; blocked is shown as a correct protective outcome.

## 12. Bulk result patterns
One result drawer for import, source assign, pool, quick delivery, read-only reconcile, plan materialization and maintenance: **requested · eligible · applied · skipped · rejected · failed · unknown** (counts reconcile, shown), grouped detail lists with per-item reasons, retry only known-failed, reconcile Unknown first (read), recheck at apply. Baseline: Import supported · Assignment supported · Return conditional (disabled) · Reconcile conditional read · Repair/Delete/Cleanup maintenance only · Export conditional.

## 13. Shared Component Delta
Reused unchanged: AppShell, RoleNavigation, PageHeader, FreshnessIndicator, ScopeBadge, MetricCoverage (`.cov`), LinkConfidence (`.lc`), TimeBasis (`.basis`), Grain, DataTable (+stackable), BulkActionBar, BulkOutcomeStrip (SN-202), BranchCoverage / source-coverage matrix (SN-203), Drawer/Sheet, Timeline, EligibilityPanel, OwnershipGrid (extended to 7 cells via role CSS `.own7`), ReportCatalog cards, Result/Empty/Stale/Conflict/Partial/Unknown/Offline states, Help/Tour/Demo panel, Command palette, Theme/Density/Focus Mode.
**Added to `shared/crm-ext.css` (additive, SN-204 block, no new tokens):** `SourceRunContext` (`.sc`, `.src-ctx`), `LineageLadder` (`.lin-ladder`, `.ln-*`), `ActionZone` (`.zone`, `.z-normal/.z-adv/.z-maint`), `MaintenanceBoundary` header (`.zone-box`, `.zone-box-h`).
**Role-layer patterns that are candidates for promotion in SN-207:** HealthContradiction (`.hc-*`), metric/report trust grid (`.trust-grid`), PhoneDiscovery box (`.phone-box`, `.phone-tag`), team-grid / attention-panel layout (shared with Manager/Deputy).
Frozen Seller/Supervisor/Senior/Manager/Deputy HTML, JS and role CSS are not modified; their QA scripts are re-run (see QA-REPORT).

## 14. Help / Tour
Reuses the shared engine. **MIS tour (9 steps + help step):** Today → Import → Source Quality → Case Lineage → Custody → Reconciliation → Reports → Planning → Maintenance Boundary. Short tours: lineage, return safety, maintenance boundary. Glossary (16 terms), shortcuts, demo-state panel (15 page states + 18 flow states). Tours only navigate and open read-only panels; QA asserts the mock data is byte-identical before/after the full tour.

## 15. Final QA evidence
See `qa/QA-REPORT.md` (runner `qa/qa-automated.js`, screenshots in `qa/`).

## 16. Known limitations
- Mock data only; no real MIS account, no real import/assign/return/delete executed anywhere (spec: writes NOT LIVE VERIFIED).
- Real assistive technology, real data volume (large batches, 5 MiB files), real XLSX/CSV parsing and a real live-lead creation were not exercised.
- «Maintenance authority verified» is a demo toggle only; the real authority model is undefined (MIS-G07).
- Role-layer CSS repeats a few layout rules also present in Manager/Deputy (promotion deferred to SN-207).
- Phone search in the palette uses masked mock numbers.

## 17. Product gaps / deferred (not solved, not hidden)
MIS-G01 F01 return/reset protection · MIS-G02 resolver / logical Case / OPD-01 · MIS-G03 health/readiness cause · MIS-G04 F02/F03/F04 definitions · MIS-G05 Historical As-Of (F09) · MIS-G06 bounded/stepped reports · MIS-G07 repair/maintenance authority · MIS-G08 planning candidates / maker-checker (OPD-08) · MIS-G09 large-batch idempotency/outcomes · MIS-G10 export parity/security · MIS-G11 latest-operation/import history · MIS-G12 reconciliation ownership/escalation. Open decisions OPD-03 (cancelled release), OPD-06/07 (currency/cohort/attribution), OPD-09, OPD-10 remain open; no UI invents them.

## 18. Freeze review
| Item | Result |
|---|---|
| Fixed during QA | contrast ≥ 4.5:1 for HealthContradiction cells in dim/dark; run-drawer audit section opened by default; QA assertions that matched explanatory text were changed to assert controls |
| Verified | structure (9 views), critical flows (import result, duplicate/invalid, lineage, custody timeline, blocked return, reconciliation issue, report trust, planning approval vs materialization, maintenance impact), 15 page-state sims, matrix 5 widths × 3 theme/density/focus combos |
| Shared components added | SourceRunContext · LineageLadder · ActionZone · MaintenanceBoundary header |
| MIS-specific components | HealthContradiction · ReportTrustContext grid · PhoneDiscovery · ReturnSafety drawer · ImpactReview · work-queue |
| Cross-role consistency | same tokens, nav, drawers, tables, states, tours, theme/density/focus; Manager/Deputy bulk and coverage patterns reused unchanged |
| Ready for next role? | **YES — HR design may start after this PR is reviewed and merged by a human.** No HR work was started here. |
| Result | **MIS DESIGN V1 — READY FOR REVIEW** |
