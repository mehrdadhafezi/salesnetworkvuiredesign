# HR Redesign — Design V1 (SN-205)

Status: DESIGN PROTOTYPE · no plugin / backend / database / API / permission / capability / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/HR-PRODUCT-SPEC.md` → `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 + Senior V1 (SN-201) + Manager V1 (SN-202) + Deputy V1 (SN-203) + MIS V1 (SN-204) shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority; every name, ID and number is fictional.
**HR ROLE RUNTIME IS NOT LIVE VERIFIED (HR-G01).** No real non-Admin `sn_hr` account was tested; Admin behaviour is not HR evidence. The prototype says so in the header tag, the welcome text and every drawer footer; nothing in it implies a live-confirmed HR behaviour or a successful write.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `hr/index.html`. Deep links: `?view=work|onb|req|acc|comp|exc|cred|bulk|diag&theme=&density=&focus=1&sim=…&open=kind:id` (+ `flow=`, `wq=`, `oq=`, `rq=`, `aq=`, `xq=`, `dq=`, `compAuth=denied`, `imp=1`, `palette=`). Flows: `onb|xfer|targetinactive|effconflict|xpartial|xunknown|term|termclear|reqstep|reqapply|applyfail|reqfailed|reqtermblock|reqchanged|reqoos|reqapplied|access|accpartial|comp|compunknown|compdenied|cred|credlead|credpartial|imp|implead|impactive|bulkwf|bulkcred|exc|staff|dup`.
Result label: **HR DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision; this PR is not merged by the author).

Role boundary: **HR = Workforce + Structure + Access Lifecycle + Request Application + Temporal History.** Not Sales leadership, not Finance, not MIS, not unrestricted WP Admin, not an identity-merging tool. Current structure never rewrites historical attribution.

## 1. Current → Target mapping (Product Spec §3, §5)
| Current destination | Decision | Target (labels only; routes/IDs/statuses unchanged) |
|---|---|---|
| `hr-workforce`, `hr-overview`, `hr-extra` | KEEP + MERGE | **نیروها** (Workforce Explorer; one profile context, readiness chips are not scores) |
| inline autosave | SIMPLIFY | sensitive changes only through **Sensitive Action Confirmation** |
| `hr-manual-add` | KEEP | **ورود نیرو و انتقال › ورود نیرو** (6 stages, credentials not a stage) |
| `hr-hierarchy` | KEEP + ADAPT | **تغییر مدیر** (Hierarchy Change Preview, temporal contract) |
| `hr-change-requests` | KEEP | **درخواست‌ها** (step approval ≠ applied; failed distinct) |
| `hr-positions`, `hr-structure`, manual invoice / extra-number overrides | MOVE | **ساختار و دسترسی** (Access State Matrix, positions/levels, unit ≠ reports_to) |
| `hr-compensation` | KEEP, separate | **جبران خدمات** (authorized only; separate from wallet/Finance) |
| `hr-logs`, missing-profile users, termination impact (new) | MERGE / ADD | **استثناها و ممیزی** (11 exception classes + audit timeline) |
| password / reset / SMS, staff-panel view + return | MOVE | **اعتبارنامه (محدود)** (restricted; impersonation ≠ read-only) |
| `hr-csv`, `hr-bulk` | KEEP, secondary | **ورود/گروهی/خروجی** (conditional baseline) |
| legacy mappings / raw log | MOVE | **نگاشت قدیمی** (advanced, read-only) |
The old→new table is also visible in the prototype (نگاشت قدیمی › نقشه ابزارهای قدیمی).

## 2. HR IA
Groups in the shared RoleNavigation: **PRIMARY** نیروها · ورود نیرو و انتقال · درخواست‌ها · ساختار و دسترسی · جبران خدمات · استثناها و ممیزی → **ADVANCED / RESTRICTED** اعتبارنامه (محدود) → **SECONDARY / CONDITIONAL** ورود/گروهی/خروجی · نگاشت قدیمی (priority-overflow into «بیشتر»). Mobile bottom nav: نیروها، ورود، درخواست، دسترسی، استثنا + «بیشتر». ActionZone badges in page headers: **عادی / مشروط / محدود · حساس** (icon + text).

## 3. Workforce Explorer
Table of profiles with **separate columns** for identity linkage (person↔profile and profile↔account, each with its own state), position·level·mapped role, current manager («اعتبار تاریخی نیست»), employment state, access state, and conflict chips (inactive manager, duplicate profile, missing user, role/position mismatch, no parent, partial access, history gap, termination with open work). Queues: all · open mismatches · inactive manager · incomplete history. 3 compact indicators with scope/basis tooltips; no score or percentage. Profile drawer: **IdentityLinkage** (Person ↔ Workforce Profile ↔ WordPress User; global person ID explicitly «تعریف نشده»), 7 separate entity cells (position, level, mapped role, unit/team, current manager, employment state, access state), conflicts, **Employment Timeline**, entry points to transfer preview, termination impact, access inspector, compensation. There is no «disable user» or delete shortcut.

## 4. Employment / Structure Timeline
**EffectiveIntervalTimeline**: per interval — effective from/to, manager in that interval, position, **applied time**, **event actor**, source (direct edit / request / system). Missing interval → **نامعلوم (UNKNOWN)**; partly known → **ناقص (INCOMPLETE)**; both are never filled from today's manager. Footer note: current structure does not rewrite Credit Owner, Event Actor or past hierarchy attribution.

## 5. Onboarding / Transfer
**Onboarding** (6 stages: identity discovery · profile/linkage · position/level · hierarchy placement · access activation · completion). Ambiguous identity (one user + one profile match) is a conflict with **no default choice**; inactive managers are not selectable; the access stage is a Sensitive Action Confirmation; **no credential field exists in onboarding**; the result lists each effect separately (profile ✓, link ✓, position ✓, hierarchy ✓, access **unknown** — «active flag ≠ usable login», credentials **skipped**). No training/document/legal workflow is invented.
**Transfer — Hierarchy Change Preview**: subject, current manager, target manager (inactive / cyclic / same / effective-date-conflict blocked), position context, requested effective date («پشتیبانی‌شده تأیید نشده — فقط زمان اعمال») vs actual apply time, review path (direct edit ≠ request finalization), effect on access (inherited overrides), effect on operational responsibilities («دیده‌شدن زیر مدیر جدید ≠ انتقال پرونده»; OPD-05 conditional), historical preservation. Confirmation → per-effect result (close previous interval, insert new, access inheritance, history preserved); partial and Outcome-Unknown variants.

## 6. Termination / Handover impact
Not «Disable user». Impact review shows employment state and **access state as two separate changes**, current manager, **open work table** (leads, invoices, customers, tasks), current custody, **next responsible actor = UNKNOWN until OPD-05**, invoice/customer responsibility retained, compensation boundary (earned credit preserved, future entitlement separate), and unresolved handover items. With open work the apply control is disabled («تحویل ناتمام · OPD-05»). Even with no open work, the action only changes employment state; the result lists role/session revocation as **skipped** and states that full offboarding is not claimed. No automatic reassignment anywhere.

## 7. Requests
Lifecycle stepper (requested → review → step approval/rejection → HR → apply → result) with internal status IDs shown as tooltips. **RequestedVsApplied** two-column context (requested change | actually applied change). Queues: waiting for HR apply · in review chain · closed. Flags: changed before review, target manager inactive, out of scope, handover incomplete. Step approval ≠ applied (the request moves to «در انتظار منابع انسانی»; applied stays «اعمال نشده»). Reject needs a reason (`aria-invalid`). Final apply is a separate Sensitive Action Confirmation with fresh recheck; failed apply is its own state with unknown actual effect, reconcile-read only, **no blind re-apply** (recovery policy undefined, HR-G08). HR chain-bypass (code allows finalizing pending review) is explained, not offered (HR-G11).

## 8. Structure & Access
**AccessStateMatrix**: per product permission — source (position mapping / role / direct override / inherit), effective result, override reason, sensitive flag; comparison column for a proposed position. Four things shown separately: business position · WP role/capability · product permission · credential. Unrelated roles and direct overrides are explicitly preserved; there is no capability editor. Positions/levels table with dependency counts (deactivation blocked by staff); unit/team ≠ reports_to note.

## 9. Compensation
Separate sensitive workspace (authorized only; unauthorized state shows a locked block, never zeros). Current base/unit/commission **reference** (not the engine), interval timeline with before→after, overlap and incomplete-period warnings, explicit «جدا از کیف پول / عملکرد فروش / مالی». New-period form → confirmation: earned credit **not recalculated**, previous period closed then new inserted (no transaction proof; payroll lock unverified); failure after closing the previous period → Outcome Unknown.

## 10. Credentials / Impersonation (RESTRICTED)
Separate advanced zone, never part of workforce CRUD or onboarding. Banner: current code accepts the broad HR gate (not «admin-only»); target policy conditional (HR-G04/G05). Leadership targets blocked («سیاست هدف مجاز تأیید نشده»). Password reset form: password fields never echoed, stored or logged (validated in the DOM only; `aria-invalid` + message); reset and SMS are **two separate outcomes** (reset ✓ + SMS ✗ is its own state). Impersonation: actor, target, start state, **explicitly not read-only**, active-session warning, return action, audit requirement (durable trail unverified), purpose required, no nesting; an active-session banner stays at the top until return.

## 11. Exceptions / Audit
11 spec classes (invalid hierarchy, inactive manager with active staff, pending requests, failed apply, duplicate profile, missing user linkage, role/position mismatch, termination with open work, out-of-scope request, access mismatch/apply partial, historical attribution gap). Each item: subject, owner, state, impact, allowed HR action, restricted action, resolution proof; no SLA or escalation command. **Audit timeline**: actor, subject/request, before→after, reason, review chain, effective date, applied time, result, correlation; secrets/salary masked; completeness explicitly unverified.

## 12. Sensitive Action Confirmation
One shared pattern (`D.sens`) for hierarchy change, request apply, termination, access change, compensation, credential reset and impersonation: **what changes · what does not change · affected entities · effective time · audit reason (required) + acknowledgement**, with commit gated; never a bare «Are you sure?». Preview ≠ permission; apply re-checks.

## 13. Shared Component Delta
Reused unchanged: AppShell, RoleNavigation, PageHeader, Freshness, MetricCoverage, LinkConfidence (`.lc`, for linkage states), TimeBasis, DataTable, BulkActionBar, BulkOutcomeStrip (SN-202), Drawer/Sheet, Timeline, ApprovalChain (`.chain`) + `.steps`, EligibilityPanel, ActionZone and MaintenanceBoundary tokens (SN-204), shared states, Help/Tour/Demo panel, palette, Theme/Density/Focus.
**Added to `shared/crm-ext.css` (additive SN-205 block, no new tokens):** IdentityLinkage (`.idl`), EffectiveIntervalTimeline (`.eit`, `.ei-*`), RequestedVsApplied (`.rva`), AccessStateMatrix delta row (`.asm-delta`), SensitiveActionConfirmation grid (`.sens-*`).
**HR role-layer patterns (candidates for SN-207):** ImpersonationBar (`.imp-bar`), OnboardingStageChips (`.onb-st`), HandoverImpact table.
Frozen Seller/Supervisor/Senior/Manager/Deputy/MIS HTML, JS and role CSS are not modified; their QA scripts are re-run (see QA-REPORT).

## 14. Help / Tour
Shared engine. **HR tour (9 steps + help step)**: Workforce → Onboarding → Transfer → Requests → Structure & Access → Compensation → Exceptions → Restricted Credentials. Short tours: transfer preview, termination/requests, restricted path. Glossary (13 terms), shortcuts, demo-state panel (19 page states + 25 flow states). Tours only navigate and open read-only panels; QA asserts the mock data is identical before/after the full tour.

## 15. Final QA evidence
See `qa/QA-REPORT.md` (runner `qa/qa-automated.js`, screenshots in `qa/`).

## 16. Known limitations
- Mock data only; no real HR account, no real hire/transfer/terminate/access/credential/impersonation/compensation write executed (spec: HR runtime NOT LIVE VERIFIED).
- Real assistive technology, real volume, real CSV/XLSX import/export and real SMS were not exercised.
- Effect-level results (e.g. «access partial») are scripted variants of the code-verified failure modes, not observed runtime behaviour.
- A few layout rules repeat Manager/Deputy/MIS role CSS (promotion deferred to SN-207).

## 17. Product gaps / deferred (not solved, not hidden)
HR-G01 real HR runtime · HR-G02 (P0) termination/transfer handover, OPD-05 · HR-G03 role/position/access separation guards · HR-G04 credential authority · HR-G05 impersonation policy/audit · HR-G06 temporal hierarchy integrity · HR-G07 compensation integrity/authority · HR-G08 failed-apply recovery · HR-G09 bulk/export semantics · HR-G10 onboarding identity/completion · HR-G11 HR request entry / chain bypass · HR-G12 documents/training. OPD-02/06/08/09 remain open; no UI invents them.

## 18. Freeze review
| Item | Result |
|---|---|
| Fixed during QA | see QA-REPORT (request `from` snapshot inconsistency; identity-chip cell layout; drawer footer hint overflow; HR-runtime banner made persistent) |
| Verified | structure (9 views), critical flows (workforce profile, onboarding conflict, transfer preview, termination handover, request review, access-state comparison, compensation history, restricted credential/impersonation), page-state sims, matrix 5 widths × 3 theme/density/focus combos |
| Shared components added | IdentityLinkage · EffectiveIntervalTimeline · RequestedVsApplied · AccessStateMatrix delta · SensitiveActionConfirmation |
| HR-specific components | ImpersonationBar · OnboardingStageChips · HandoverImpact · request review drawer |
| Cross-role consistency | same tokens, nav, drawers, tables, states, tours, theme/density/focus; SN-202/204 bulk-outcome and zone patterns reused |
| Ready for next role? | **YES — Finance design may start after this PR is reviewed and merged by a human.** No Finance work was started here. |
| Result | **HR DESIGN V1 — READY FOR REVIEW** |
