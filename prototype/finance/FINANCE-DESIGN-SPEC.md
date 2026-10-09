# Finance Redesign — Design V1 (SN-206)

Status: DESIGN PROTOTYPE · no plugin / backend / database / API / permission / capability / workflow change.
Product authority (WHAT/WHY): `docs/source-of-truth/FINANCE-PRODUCT-SPEC.md` → `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md`.
Design authority (HOW): Seller V1 + Supervisor V1 + Senior V1 (SN-201) + Manager V1 (SN-202) + Deputy V1 (SN-203) + MIS V1 (SN-204) + HR V1 (SN-205) shared system (`../shared/*`, `../styles.css`). Mock data in `data.js` is not business authority; every name, ID, amount and run is fictional.
**FINANCE ROLE RUNTIME IS NOT LIVE VERIFIED (FIN-RUNTIME).** No real non-Admin Finance account was tested; Admin behaviour is not Finance evidence. The header tag, welcome text and every drawer footer say so; nothing implies a live-confirmed Finance workflow.
Open: serve `prototype/` statically (`python3 -m http.server 8777`) → `finance/index.html`. Deep links: `?view=rev|rec|led|run|rep|aud|cfg|mnt&theme=&density=&focus=1&sim=…&open=kind:id&perm=view` (+ `flow=`, `rq=`, `recq=`, `lq=`, `ldom=`, `runq=`). Flows: `item gateway noevidence mismatch unit dup link stale stalecommit approve reject rejected approved bulkapp bulkrev refund refundunknown issue ledger ledgerunk orphan runpreview runapproved runpartial rununknown runposted runfailed runrecon rundraft report view`.
Result label: **FINANCE DESIGN V1 — READY FOR REVIEW** (freeze is a reviewer decision; this PR is not merged by the author).

Role boundary: **Finance = Financial Review + Reconciliation + Ledger Control + Ruled Posting.** Not Sales leadership, not HR, not MIS, not unrestricted Admin, not a customer-service workspace. Reads never create financial state; the prototype saves, posts and sends nothing.

## 1. Current → Target mapping (Product Spec §3, §5)
| Current destination | Decision | Target (labels only; routes/IDs/statuses unchanged) |
|---|---|---|
| `finance-payments` (needs_review, receipt_uploaded, online_paid, approved, rejected) | KEEP + MERGE | **صف بررسی** — one queue, grain = reviewable payment stage; the five tabs become overlapping facets |
| `finance-overview`, flow guide | KEEP / SIMPLIFY | page KPIs with grain + meaning; no engine-wide flow guide |
| approve / reject handlers | KEEP + ADAPT | **Sensitive Financial Action Confirmation** (new shared pattern) |
| `refund_confirmed`, cancellation | KEEP CONDITIONAL | **تطبیق و استرداد › استرداد** — 7 separate concepts; cancel ≠ refund (OPD-04) |
| `finance-wallet` / ledger, logs | KEEP | **دفتر کل** — audit explorer, two domains, linked corrections, read-only (F07) |
| rules create/toggle, generate dry-run, run review, APPLY | KEEP CONDITIONAL | **قوانین، اجراها و ثبت** — 9-state lifecycle, coverage, Outcome Unknown context |
| gateway CSV, readiness, matrix reports | KEEP | **گزارش‌ها** — 7 grains with trust context |
| recalculate, backfill, manual adjustment, posting flag, readiness diagnostics | MOVE | **نگهداری و بازیابی (پیشرفته)** — explained, not executable here |
| (missing) reconciliation workspace | ADD (target, conditional) | **تطبیق** — 10 issue categories, owner + proof, no Fix All |
| (missing) permission contract | ADD (display only) | **پیکربندی و اختیار** — conceptual, action-aware; no capability editor |

## 2. Finance IA
Primary: صف بررسی · تطبیق و استرداد · دفتر کل · قوانین، اجراها و ثبت · گزارش‌ها. Secondary (`cond` group): ممیزی و تاریخچه · پیکربندی و اختیار. Advanced: نگهداری و بازیابی. Daily review never contains diagnostic/maintenance controls.

## 3. Review Queue
Grain: **one reviewable payment stage linked to an invoice.** Columns (priority at 1366): invoice + stage ID + Case + link confidence · stage · evidence + issue chips · claimed / validated amount + unit · finance review + prior decisions; at ≥1500: customer minimum + seller/team, reviewer + next actor; at ≥1100: total / paid / remaining. KPIs state their own grain (stage ≠ invoice); claimed amounts are per unit, unknown unit is counted apart and never summed. Facets overlap (receipt / gateway-recorded / approved…) and are labelled not-a-partition. Selection → Bulk Review (read-only), Bulk Approve (conditional), Bulk Reject (needs validation, disabled), Export (conditional, disabled).

## 4. Financial State Model — FinancialStateLayers (new shared pattern)
Nine independent cells: invoice status · payment stage · finance review · payment evidence · validated paid · remaining · refund state · ledger posting · entitlement. **Each cell names what it must NOT be inferred from** (e.g. entitlement ← not from stage approval or full payment). receipt uploaded ≠ validated ≠ stage approved ≠ invoice fully paid ≠ entitlement ≠ wallet credit ≠ settlement. Approve preview never says “fully paid”: tolerance is undefined (OPD-06). Unknown/unit-ambiguous amounts render as `—` / “واحد نامشخص”, never 0 or a default unit.

## 5. Approve / Reject
Drawer: state layers → evidence panel (type, ref+version, submitter/time/source, amount in evidence vs claimed vs validated, unit; protected evidence viewing) → Case/link → separate roles (original seller / current owner / reviewer / next actor / commission recipient) → prior decisions → eligibility checks → impact of approve **and** of reject. Fail-closed blockers: missing evidence, amount mismatch, unit ambiguity (OPD-06), unknown/conflicting link, duplicate-payment suspicion, stale stage. Reject requires a reason, and the confirmation states it is not delete / cancel / refund / history reset. Stale: Conflict state, both decisions disabled until reload; also a commit-time conflict result (“no change recorded”).

## 6. Refund (conditional, OPD-04)
Ladder of 7 concepts (requested · reported · proof pending · confirmed · partial · full · unknown) with the current one marked; original invoice/payment, requested, already refunded (reported vs recorded), eligible remainder (unknown stays unknown), execution source, proof, reviewer, history. Cancelled invoice ≠ refunded money is stated. Execution is unavailable with the reason; no bank workflow or proof policy is invented; bulk refund does not exist.

## 7. Ledger
Audit explorer: transaction ID, business key, domain/account, debit/credit, amount+unit, source/relation, actor, effective vs posting time, posting state (committed / preview / unknown). Corrections and reversals are new linked rows (chain shown in the drill). No edit/delete, no editable balance, no single combined balance across domains/units. Orphan-looking transactions are preserved (no phone relink). The page states F07: rendering never triggers legacy wallet autopost.

## 8. Rules / Runs / Posting
Lifecycle ribbon (draft · preview · approved · posting · posted · partial · failed · outcome unknown · reconciled) with per-run state. Console: snapshot/engine (purpose engine may auto-credit at approval — OPD-09; lock caveat FIN-LOCK), 7-cell coverage (intended / processed / posted / existing / failed / unprocessed / unknown), approval chain (maker/checker not assumed — OPD-08), bounded item sample labelled as such. Generate Preview is declared a metadata write. Approved ≠ Posted. A 2000-item bounded call never reads as full-run success; the unprocessed tail is counted and can only be continued on the same snapshot with the same business keys. Rules tab is a read-only snapshot.

## 9. Outcome Unknown / Idempotency — OutcomeUnknownContext (new shared pattern)
Original intent · transaction/business key · possible committed state · lookup/reconciliation requirement · existing-transaction result · retry eligibility · next safe action. No Retry button exists in this state (a disabled one explains why). Read-only lookup classifies each item; only proven-uncommitted items may be continued. Existing-transaction hits are counted separately from new credits.

## 10. Reconciliation
10 categories (amount mismatch, duplicate payment, missing mirror, stage/review mismatch, invoice/report mismatch, refund mismatch, ledger mismatch, unknown posting outcome, currency/unit ambiguity, orphan-looking relation). Each: evidence, financial impact, owner/domain, allowed Finance action, restricted action, next actor, resolution proof, linkage confidence. Bulk reconcile = diagnosis only. No Fix All.

## 11. Reports
7 catalog cards (review workload, current invoice state, transaction events, ledger activity, runs/posting, historical as-of, settlement readiness). Each drawer shows grain, scope, time basis, unit, source, freshness, coverage, reconciliation state; units never summed; unknown stays unknown; settlement readiness ≠ settlement paid; export disabled (independent authority).

## 12. Shared Component Delta
Reused unchanged: AppShell, RoleNavigation, PageHeader, KPI/MetricMeta, FreshnessIndicator, coverage chips (`.cov`), LinkConfidence (`.lc`), DataTable (+sticky actions), Drawer/Sheet, Timeline, ApprovalChain/OperationLog patterns (`steps`, `ir-list`), BulkReview/BulkResult (`bulkbar`, `outcome-strip`), Attention/Exception patterns, ReportCatalog (`rep-card`), Toast/Tooltip, Help/Tour, CommandPalette, shared system states, Theme/Density/Focus, Sensitive confirmation grid (`sens-*`, from SN-205).
**Added (additive block in `shared/crm-ext.css`, labelled SN-206; presentation only):** `FinancialStateLayers` (`.fsl*`), `OutcomeUnknownContext` (`.oux*`), outcome/coverage cells (`os-posted`, `os-existing`, `os-unprocessed`). Confirmation list tones (`sens-aff`, `sens-eff`) live in `finance/finance.css` only, so frozen HR dialogs are not restyled. JS-side helpers (`amt` unit/unknown rendering, `covRow`, `guardBtn` for action-aware disabled reasons) live in `finance/lib.js` and are candidates for SN-207 promotion.
**Finance-specific (`finance/`):** Review Queue grain, Evidence panel, Refund ladder, Ledger explorer + correction chain, Run console, Reconciliation issue drawer, permission contract table, maintenance cards.

## 13. Help / Tour
Shared engine + `help-content.js`: main tour (9 topics: Review Queue, Financial States, Evidence, Approve/Reject, Reconciliation, Ledger, Runs/Posting, Reports, Advanced) + approve, posting/unknown, reconciliation mini-tours; glossary; keys. Tours only navigate and open read-only panels (verified: data snapshot unchanged). No approve/reject shortcut key exists, deliberately.

## 14. Known limitations
Mock data only; no real Finance account, evidence file, bank, gateway or posting; FIN-RUNTIME not verified. Sensitive confirmations commit to an in-memory overlay (reset on reload) and write nothing. Amount formulas are display-only (paid + claimed); tolerance/rounding is undefined (OPD-06). Prototype locale/numerals are Persian only. No real assistive-technology run. Table internals reflow (priority columns / stacked rows) rather than all columns at every width.

## 15. Product Gaps / Deferred (not resolved by this design)
FIN-RUNTIME, F06 (view gate used for sensitive writes), F07 (legacy wallet autopost on render), FIN-LINK, OPD-04 (refund proof/execution), OPD-06 (unit/tolerance), OPD-08 (maker/checker), OPD-09 (engine/entitlement routing), FIN-IDEMPOTENCY (exact-once across races/engines), FIN-RECOVERY, FIN-RUN-COVERAGE (2000-item bound), FIN-LOCK (all-path snapshot immutability), FIN-PARITY, FIN-TAXONOMY (overlapping facets — presentation handled), FIN-IMPORT (bank statement import). The design assumes none of these is fixed.

## 16. Freeze review
See `qa/QA-REPORT.md`.
