# Slice 13 — Commission engine: command matrix, dual-engine report, decision readiness and parity

Status: **controlled implementation, historical parity, zero financial activation.**
Baseline: `origin/main` = `b4c93983a0ab9d0550e92a1fed8b821df8ff3f24` (verified; no delta).
Legacy evidence: `plugin/current` (2.0.160), **read only — not modified**.

## 1. Recorded conditions

- No merge, deploy or Production activation. The legacy plugin is unchanged.
- No unresolved business decision is treated as approved. No formula or eligibility rule is invented: the
  only registered calculation functions reproduce source-evidenced legacy formulas, and they are used for
  **parity only**.
- No real posting command exists. `CommissionPostingRequested` is a designed, validated, **never dispatched**
  contract (lint `s13-posting-dispatch`; DB: commission outbox stays empty in the e2e test).
- Commission writes only the `commission` schema. It never writes finance ledger, wallet, billing, payments,
  organization or any other module (grants, lint `foreign-schema-write` / `s13-foreign-sql`, CI `go list -deps`).
- GET endpoints never import, backfill or recalculate (lint `read-path-import`, `s13-read-write`; HTTP e2e
  asserts every commission table count is unchanged after all GETs).
- No floating-point money: amounts are exact `int64` hundredths + explicit unit; rates are exact `int64`
  ten-thousandths of a percent (lint `s13-float`; DB: no `real` / `double` / `numeric` / `money` column).
- Missing rule, base, unit or rounding **blocks** the calculation (no assumption, no zero).
- D-36 remains blocked (component allocation not registered). Historical parity is reported only from
  synthetic fixtures in this PR — **no production parity percentage is claimed.**

## 2. Legacy forensic audit — four engines (file:line matrix)

| # | Engine | Trigger / entry | Rule source | Base | Formula | Recipients / attribution | Identity / duplicate guard | Reversal / correction | States |
|---|---|---|---|---|---|---|---|---|---|
| E1 | Purpose matrix | `sn_invoice_payment_stage_approved` prio 18 (`class-sn-purpose-commission.php:39`, `:584-591`); enabled by option `sn_purpose_commission_engine_enabled` default `'1'` (`:26-28`, set `'1'` at install `:103`) | `sn_payment_commission_matrix` (`:75-87`), **mutable in place** (`persist_rates` ON DUPLICATE KEY UPDATE `:245-270`), no history; rate clamped 0..100 (`:326-331`) | approved stage amount `context['stage_amount']` (`class-sn-plugin.php:6582-6591`) | `round(stage_amount * rate / 100, 2)` PHP float (`:606`); skip if rate ≤ 0 or amount ≤ 0 (`:603`, `:606`) | purpose via `resolve_purpose` (`:333-370`, **writes back** `payment_purpose`); recipients via `resolve_recipients` (`:420-540`): seller = original seller / seller; supervisor+ from the **current** HR hierarchy (`ancestor_for_position` `:390-418`, `is_current=1`) | key `purpose-commission:{inv}:stage:{n}:{purpose}:{role}:user:{uid}` + role-prefix LIKE guard (`:607-612`); meta snapshot engine/purpose/role/rate/base/stage/recipient/snapshot_version (`:614-618`) | debit `purpose_commission_reversal` key `purpose-commission-reversal:{creditId}`, then original **rewritten** to `purpose_commission_reversed` with key NULL (`:675-689`); financial cancel reverses all stages, reopen the latest stage (`:692-705`); backfill/reconcile recomputes with **current** rates & recipients (`:272-300`, `:624-641`) | credit inserted `approved`; no generated/approved/posted split |
| E2 | Rules auto-credit | `sn_credit_wallet_for_invoice` (`class-sn-plugin.php:35854`); returns if E1 enabled (`:35856-35858`); option `sn_wallet_auto_credit` default `'1'` | `sn_commission_rules` (`class-sn-migration-service.php:1240-1271`); matcher `:32744-32801` (position, level, employment, payment method, invoice status, min/max, effective interval; equal from/to treated open-ended `:32730`); HR override `:32394` | invoice total (`:35830`) | per rule `round(max(0, fixed + amount*pct/100), 2)` (`:32905-32912`), stacked cumulatively (`:32803`) | seller from invoice context; supervisor from lead / usermeta / **current** HR assignment (`:35902-35921`) | existence check invoice+user+type (`:35844-35851`); supervisor check (`:35923`) | `on_invoice_financial_rejected_commission` debit `commission-reversal:{id}:{type}` + original rewritten (`:29297-29331`); finance reopen (`:30795-30815`) | credit `approved` |
| E3 | Dry-run engine | actions `sn_commission_run_dry_run` / `review_run` / `post_wallet` (`class-sn-plugin.php:134-138`, `:33301`, `:33555`, `:33602`) | same rules table; context at `calculation_date` (`:11357`, `:32578`) | invoice amount | same rule function | item `seller_user_id`, position/level/compensation snapshot (`class-sn-migration-service.php:1368-1407`) | item ↔ wallet tx (`wallet_transaction_id`, meta `commission_dry_run_item_id`, `:33011-33024`) | none of its own (E2 reversal handler covers `commission_engine`) | run `approval_status` generated / approved / rejected; **rejected → approved allowed** (`:33568`); approve and reject both set `locked_at`; posting needs flag `sn_enable_commission_wallet_posting` (default `'0'`, false when E1 on `:32956-32962`) + `APPLY` + approved (`:33612-33621`); per-item `posted` / `error` (`:33641-33652`); no maker/checker |
| E4 | Biawin project card | stage hook prio 20 (`class-sn-projects.php:43`); `credit_biavin_commission` returns if E1 on (`:1182`) | `sn_project_commission_rules` per content product (`:370-390`) | membership item `line_total` (component) | fixed or `round(base*value/100, 2)` (`:1181-1218`) | membership item | per project commission row | reversal + recredit (`:1245-1255`) | pending → credited → reversed |

Competing-engine facts: E1 enabled disables E2 auto-credit, E3 posting and E4. The engine option has **no
history**, so which engine was live for any past window is unknowable from the database alone
(→ `ENGINE_ROUTING_UNKNOWN`). `commission_engine` credits are written by both E2 and E3; only meta
(`commission_run_id` vs `source=invoice_auto_credit_finance_rules`) distinguishes them. `seller_commission`
has no 2.0.160 writer (pre-engine history).

## 3. Decision readiness (Phase 3)

| Decision | Classification | Effect in Slice 13 |
|---|---|---|
| OPD-09 (engine routing, non-invoice keys, correction) | **REQUIRES HUMAN OWNER DECISION** | approval and posting commands blocked; non-invoice (daily attendance) entitlements BLOCKED `NON_INVOICE_KEY_UNAPPROVED`; unproven engines BLOCKED |
| D-36 (component allocation) | **IMPLEMENTATION BLOCKED** | `component_pro_rata` not registered; E4 rows `NOT_COMPARABLE` |
| D-41 (shadow computes both, publishes neither) | **SAFE DEFAULT AVAILABLE** | implemented: shadow run only, nothing published |
| OPD-06 (unit / rounding) | **REQUIRES HUMAN OWNER DECISION** | no default rounding; the legacy half-away-2dp is evidenced **parity-only**; `--rounding NONE` blocks every calculation |
| OPD-08 (maker/checker, posting idempotency / unknown recovery) | **IMPLEMENTATION BLOCKED** | no posting dispatch; deterministic posting request key + duplicate refusal designed and tested only |
| HR attribution (supervisor+ recipients) | **EVIDENCE INCOMPLETE** | legacy uses the current hierarchy; only writer meta snapshots are event-time evidence; missing snapshot → `ATTRIBUTION_UNKNOWN`, never filled from today's hierarchy |

## 4. Command matrix

| Command | Status | Where |
|---|---|---|
| Import legacy evidence + shadow recalculation + parity | **IMPLEMENTED (offline job only)** | `cmd/crm-commission-shadow run --snapshot-id ID --rounding …` |
| Read overview / runs / calculations / findings / policies / legacy runs | **IMPLEMENTED (GET, flag `commission.read_shadow`, default off)** | `/api/v1/commission/*` |
| Own earnings | **IMPLEMENTED (GET, own scope only)** | `/api/v1/commission/me/earnings` (row filter = authenticated user; no user → 403) |
| Approve / reject run | **BLOCKED (OPD-09)** | `domain.Approve` validates the transition then returns `ErrCommandBlocked`; no route, no capability |
| Request posting / post / credit wallet | **BLOCKED (OPD-08/09, D-41)** | `domain.RequestPosting` → `ErrCommandBlocked`; `PostingRequested` never appended to an outbox |
| Settle / reverse / correct | **BLOCKED** | not implemented; reversal linkage is preserved and validated on historical evidence only |
| Recalculate / backfill from current hierarchy or rates | **FORBIDDEN** | not implemented; lint `s13-forbidden-command` |

Capabilities (`identity` migration 00010, granted to **no role**): `commission.shadow.read`,
`commission.policy.read`, `commission.own.read`. No commission command capability exists (CI checks).

## 5. Domain model

- `domain.Policy` — key, purpose, share facet, eligibility predicate ref, function id, rate (E4), fixed,
  base kind, unit, effective interval (nullable = UNKNOWN, never "always"), attribution facet, scope,
  provenance, evidence state (`OBSERVED_UNVERSIONED` / `EVENT_SNAPSHOT` / `INVALID`), approval state (only
  `NOT_APPROVED`). Versions are content-addressed (`policy_version`, UNIQUE per key + content).
- `domain.Registry()` — closed typed set, no DSL: `purpose_matrix_stage_percent_v1`,
  `rule_invoice_percent_fixed_v1`. Not registered: `component_pro_rata` (D-36), `daily_attendance_plus_percent` (OPD-09).
- `domain.Calculate` — pure, deterministic; output has input snapshot SHA, policy version, event ref,
  entitlement key, recipient snapshot, share facet, base, rate, exact amount + unit, eligibility, blocks,
  reasons, coverage.
- `domain.EntitlementIdentity` — `inv|stage|purpose|facet|ctx`; recipient is an attribute, never part of the key.
- Attribution facets kept distinct: SOURCE_OWNER, ORIGINAL_SELLER, CREDIT_OWNER, CURRENT_CUSTODY,
  CURRENT_ORG_ASSIGNMENT, EVENT_ACTOR (SNAPSHOT / UNKNOWN / CONFLICT).
- Lifecycle (only evidenced transitions): generated → approved | rejected; rejected → approved;
  approved → posting requested → posted | posting failed; posted → reversed | settled.
  approved → rejected is **not** evidenced. A CRM shadow run is only ever `CALCULATED` (DB CHECK).
- `PostingRequested` / `PostingResult` (COMPLETED / REJECTED / OUTCOME_UNKNOWN) + deterministic request key,
  reversal references the original request key; `PostingRequests` refuses duplicates.

## 6. Importer and storage (all INSERT-only, append-only triggers, 11 tables)

`shadow_run`, `run_source`, `policy_version`, `entitlement_identity`, `historical_entitlement`
(original tx ids, engine + basis, raw type, rewritten-in-place flag, reversal link, raw + exact amount),
`historical_run` (raw approval, approved/rejected provenance, rejected-then-approved), `historical_run_item`,
`historical_posting_log`, `calculation`, `parity_summary`, `reconciliation_finding`.
Staging specs (integration migration 00010): `sn_payment_commission_matrix`, `sn_commission_rules`,
`sn_commission_dry_runs`, `sn_commission_dry_run_items`, `sn_commission_wallet_posting_logs`; wallet
transactions reuse the Slice 9 spec. Free-text columns and meta `reason` / `note` are never read.
E4 tables are inspected and not staged (D-36).

## 7. Parity harness and report

Classes: MATCH, AMOUNT_MISMATCH, RECIPIENT_MISMATCH, POLICY_MISMATCH, ATTRIBUTION_UNKNOWN, DUPLICATE_RISK,
ENGINE_ROUTING_UNKNOWN, SOURCE_INCOMPLETE, NOT_COMPARABLE. Compared fields: eligibility, base, rate, amount,
unit, recipient, recipient snapshot, entitlement identity, observed policy rate, engine. Report: cohort,
raw window, legacy rows, compared rows, coverage, class counts, engine counts, totals per unit (exact),
missing-amount counts, reasons. Findings: DUPLICATE_RISK (same entitlement or cross-engine on one invoice),
REWRITTEN_IN_PLACE, REVERSAL_LINK_MISSING, ENGINE_ROUTING_UNKNOWN, POSTING_LINK_MISMATCH,
RECIPIENT_MISMATCH, REJECTED_THEN_APPROVED, NON_INVOICE_BLOCKED, COMPONENT_ALLOCATION_BLOCKED.

**Historical parity coverage: NOT MEASURED ON REAL DATA.** Only synthetic fixtures were run
(`legacytest.CommissionScenario`: 11 rows → MATCH 4, AMOUNT_MISMATCH 1, POLICY_MISMATCH 1,
ATTRIBUTION_UNKNOWN 1, DUPLICATE_RISK 1, ENGINE_ROUTING_UNKNOWN 1, NOT_COMPARABLE 2; coverage PARTIAL).
These numbers prove the harness, not legacy parity. Real coverage requires a staged production snapshot and
the offline job; known structural limits: E2 multi-rule stacking and supervisor settings share are
NOT_COMPARABLE, rule values are observed at snapshot time (not event time), PHP float tie rounding can differ
by 0.01 (surfaces as AMOUNT_MISMATCH).

## 8. Test map (Phase 13)

| # | Case | Test |
|---|---|---|
| 1 | same entitlement twice | `domain TestEntitlementIdentity`; e2e reversed+recredited = one calculation |
| 2/3/4 | stages / purposes / share facets distinct | `TestEntitlementIdentity` |
| 5 | hierarchy change after event | `TestHierarchyChangeAfterEvent` |
| 6 | recipient change, no duplicate | `TestEntitlementIdentity`; e2e RECIPIENT_MISMATCH finding, one identity |
| 7 | missing attribution | `TestMissingAttributionBlocks`; e2e tx 106 |
| 8 | engine provenance conflict | `TestEngineRoutingUnknownBlocks`; e2e tx 111 |
| 9 | non-invoice without key | `TestNonInvoiceEntitlementBlocked`; e2e item 2 |
| 10 | invalid effective interval | `TestEffectiveInterval` (+ DB CHECK) |
| 11 | policy version change after calculation | `TestPolicyVersionPinned`; e2e versions reused |
| 12 | pro-rata unsupported | `TestProRataBlocked`; e2e tx 113 |
| 13 | different units | `TestUnitMismatchBlocks` |
| 14 | missing rounding | `TestMissingRoundingBlocks`; e2e `RoundingNone` run all BLOCKED |
| 15/16/17 | generated≠approved≠posted≠settled | `TestLifecycle`; e2e item posting states |
| 18 | replay historical approved run | e2e replay (no table changes); append-only UPDATE refused |
| 19 | reversal references original | `TestPostingContract`; e2e tx 104 → 103 |
| 20 | no finance ledger writes | e2e: no grant outside `commission`; lint; CI deps |
| 21 | no wallet credit on GET | HTTP e2e: counts unchanged after all GETs |
| 22 | duplicate posting request | `TestPostingContract` |
| 23 | no real posting route | HTTP e2e POST 404/405; contract test; CI grep |
| 24 | missing legacy data explicit | `TestCommissionShadowMissingSource`; parity SOURCE_INCOMPLETE |
| 25 | Slices 0–12 unchanged | full Go + web suites, golden specs byte-identical |

Golden fixtures: `TestGoldenPurposeMatrixStagePercent`, `TestGoldenRounding`, `TestGoldenRuleInvoicePercentFixed`.

## 9. Rollback

Flag `commission.read_shadow` off (default) hides every read; the offline job is never scheduled by the API.
Schema rollback: goose Down of commission 00001/00002, identity 00010, integration 00010 on a disposable
database. No legacy or other-module data is touched.

## 10. Readiness (never conflated)

| Item | State |
|---|---|
| Code complete for the safe subset | YES |
| Tests passed | see PR (Go unit + PostgreSQL + race, web, lint, gates) |
| Historical parity coverage | NOT MEASURED on real data (synthetic only) |
| Decisions blocked | OPD-09, D-36, OPD-06, OPD-08, HR attribution |
| Posting readiness | NOT READY — blocked by OPD-08/09 |
| Merge readiness | after human review + green CI; do not merge without approval |
| Production readiness | NO |
