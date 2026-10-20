# E0 — Governance & Baseline Freeze

Date: 2026-10-20 · Type: governance record · NO CODE · NO SCHEMA · NO RUNTIME CHANGE

## 1. Status

| Item | Value |
|---|---|
| Phase | E0 — Governance & Baseline Freeze |
| Verdict | **E0 FROZEN — SLICE 0 APPROVED** (§14) |
| Slice 0 | Scope frozen (§11–12); may begin only after the Human Owner merges this documentation PR |
| Owner approvals applied | D-01 APPROVED / FROZEN · D-02 APPROVED / FROZEN · Money contract APPROVED / FROZEN FOR SLICE 0 |
| Implementation started | NO |
| Production / staging touched | NO |
| HR / Finance runtime | NOT LIVE VERIFIED (unchanged) |

Evidence labels: VERIFIED-GIT (read from the repo at 86859d7), VERIFIED-CODE (legacy
source file:line), OWNER-BRIEF (stated by the owner; artifact not in repo), OWNER-APPROVED
(explicit owner approval recorded in this E0 freeze).

## 2. Verified baseline

Re-verified immediately before this record was committed; no drift detected.

| Item | Value | Label |
|---|---|---|
| Repository | mehrdadhafezi/salesnetworkvuiredesign | VERIFIED-GIT |
| main | 86859d7404c39de333ec5a83863d370281bd56ef | VERIFIED-GIT |
| SN-207 redesign end | b528264 (PR #17) → 056a51d (PR #18 epic/SN-redesign-v1) | VERIFIED-GIT |
| SN-208 plugin 2.0.160 | f1f3ded, merged 86859d7 (PR #20) | VERIFIED-GIT |
| SN-208A minimal CI | 9b8d901, merged 7b65390 (PR #22) | VERIFIED-GIT |
| SN_VERSION / SN_BUILD | 2.0.160 / 2026-10-10-finance-dialog-layout (`plugin/current/sales-network.php:18-19`) | VERIFIED-CODE |
| Newer snapshot | none (single `plugin/current`; latest CHANGELOG-2.0.160) | VERIFIED-GIT |
| Working tree at verification | clean | VERIFIED-GIT |
| CI workflow | `.github/workflows/plugin-snapshot-ci.yml` (PR on `plugin/current/**` + `workflow_dispatch`) | VERIFIED-GIT |
| CI manual run on 86859d7 | success | OWNER-BRIEF |
| Prior baseline tags | crm-redesign-baseline-v1, crm-redesign-clean-baseline-v2 (historical; superseded by 86859d7) | VERIFIED-GIT |

Prior audit corpus (OWNER-BRIEF, not committed to this repository): Implementation-Readiness
Inventory; DB & Data Model Audit; Go Backend Architecture; Frontend Technology Decision;
Panel-by-Panel Mapping; Biawin Deep Audit; Standalone Implementation Blueprint;
2.0.156→2.0.160 Reconciliation. In-repo predecessors: `CRM-IMPLEMENTATION-FOUNDATION-PLAN.md`,
`CRM-PHASE-0-IMPLEMENTATION-SPEC.md`, OPD-01…10 (`CRM-CROSS-ROLE-ARCHITECTURE-V1` §19).

## 3. Frozen product/design boundary (A)

- SN-207 is the end of product/UI redesign. Frozen: the shared CRM design system
  (SN-207), role UI/UX contracts SN-201…206, the `docs/source-of-truth` role product specs,
  `GATE-0-PRODUCT-INVARIANTS`, `CRM-CROSS-ROLE-ARCHITECTURE-V1`.
- Reopen only on concrete implementation evidence of a contradiction or impossibility.
  Record it as a blocker; never change it silently.
- The prototype and mock data are not business authority.
- Invoice Review (2.0.158–2.0.160) has **no frozen SN-207 design**. Its UI is legacy-only
  (see D-24).

## 4. Target architecture baseline

**B. Legacy implementation baseline.** `plugin/current` 2.0.160 at 86859d7 (lineage f1f3ded ← PR #20).
Legacy behaviour is *evidence*, not the target design. Legacy facts are preserved verbatim (§10).

**C. Target implementation architecture** — **D-02 APPROVED / FROZEN** (OWNER-APPROVED):
standalone CRM; Go modular monolith; PostgreSQL; React + TypeScript + Vite SPA;
OpenAPI contract with a generated typed client; WordPress/WooCommerce reached only
through a connector / legacy-integration boundary; transactional outbox/inbox; append-only
audit; reconciliation jobs; feature flags; gradual strangler migration, no big-bang rewrite.
Migration order follows the approved standalone Blueprint / ADR N-22.

**D-01 APPROVED / FROZEN** (OWNER-APPROVED): the legacy PHP Phase-0 implementation plan
(`CRM-PHASE-0-IMPLEMENTATION-SPEC.md`, placement under `plugin/current` `includes/foundation/`,
§4) is SUPERSEDED for target implementation. The PHP Phase-0 target harness is not implemented.
Its semantics (result, reason codes, refs — §5–7) and its diagnostic/legacy evidence are
preserved and carried forward into Go. Slice 0 and all new target implementation begin in Go.

## 5. Governance rules (D)

1. GitHub (main) is the single source of truth. Anything not on main is not authority.
   Decisions cited by ID must be committed or recorded on main (as in this record).
2. The Human Owner has final merge authority. No agent merges its own PR. No auto-merge.
3. No production or staging access or mutation without explicit, per-action owner approval.
4. No big-bang migration. Strangler path only, behind flags, with rollback.
5. No force-push to shared branches or main unless explicitly authorized.
6. Never work on main. Use `claude/*`, `codex/*`, `fix/*`, `docs/*` branches. One bounded PR per task.
7. AGENTS.md global contract applies (KEEP + ADAPT, phone not identity, ownership
   facets never collapsed, view ≠ write, seven-outcome honesty).
8. `plugin/current` is a read-only evidence snapshot for target work. It changes only through
   a dedicated `sync/*` PR.
9. Claude owns HOW/UX/UI. Codex owns WHAT/WHY, architecture, implementation and tests.
   Neither invents permissions, statuses, financial rules or KPI formulas.

## 6. ADR status table

Rows are by topic. Blueprint ADR identifiers are bound by the Blueprint itself (e.g. N-22 for
migration order); this table records E0 status only.

| # | Decision | Status | Note |
|---|---|---|---|
| 1 | Standalone CRM vs WP-hosted runtime | FROZEN (D-02) | Standalone. WP stays a legacy/integration source. |
| 2 | Go modular monolith | FROZEN (D-02) | No microservices |
| 3 | PostgreSQL | FROZEN (D-02) / APPROVED FOR IMPLEMENTATION | Not used before Slice 1+ |
| 4 | React + TS + Vite SPA | FROZEN (D-02) / APPROVED FOR IMPLEMENTATION | Implements frozen SN-207 design; not in Slice 0 |
| 5 | WP/Woo connector boundary | FROZEN (D-02, boundary) / DEFERRED (impl) | Only the connector knows `wp_*`/`sn_*` tables |
| 6 | Migration order (strangler) | FROZEN (D-02, Blueprint ADR N-22) | No big-bang rewrite |
| 7 | Stable refs; phone not canonical | FROZEN | Slice 0 codes the refs |
| 8 | Single-writer aggregate ownership | FROZEN | One owning domain per aggregate |
| 9 | Append-only audit/events/ledger | FROZEN (principle) / DEFERRED (storage) | Corrections are compensating entries |
| 10 | Outbox/inbox | APPROVED FOR IMPLEMENTATION / DEFERRED | Not Slice 0 |
| 11 | Idempotency | FROZEN (principle) / DEFERRED (store) | Key + reconcile-before-retry on UNKNOWN |
| 12 | Result model | FROZEN | Slice 0 (§9A) |
| 13 | Money representation | FROZEN FOR SLICE 0 (OWNER-APPROVED) / OWNER DECISION REQUIRED for business conversion/rounding (OPD-06) | §9D |
| 14 | UTC / source TZ / Jalali | FROZEN | Slice 0 (§9E) |
| 15 | Authorization model (route→module→action→row/field) | FROZEN (model) / DEFERRED (runtime) | Not Slice 0 |
| 16 | Audit of sensitive reads/exports | APPROVED FOR IMPLEMENTATION / DEFERRED | Policy for Invoice Review export is D-23 |
| 17 | Feature flags | APPROVED FOR IMPLEMENTATION / DEFERRED | Default-off; storage chosen later |
| 18 | Staging / anonymization | DEFERRED | Must exist before any real-data import |
| 19 | Canonical invoice-total resolver | DEFERRED | Legacy TOTAL expression is evidence (`class-sn-invoice-review.php`) |
| 20 | Canonical main-status resolver | DEFERRED | Legacy: main status first, then invoice/payment fallback (2.0.159) |
| 21 | Gateway auto-approval modeling | OWNER DECISION REQUIRED (D-25) | Model as distinct provenance, not Finance review |
| 22 | PHP in-plugin Phase-0 placement (Phase-0 spec §4) | SUPERSEDED (D-01) | Replaced by Go kernel; semantics kept |

## 7. Owner decision gate

### A. Blocked Slice 0 — now closed

| ID | Question | Resolution |
|---|---|---|
| D-01 | Is the legacy PHP Phase-0 implementation plan the target path? | **APPROVED / FROZEN.** PHP Phase-0 placement SUPERSEDED for target implementation; semantics and diagnostic/legacy evidence preserved; Go Slice 0 is the target implementation path. |
| D-02 | Target architecture and migration order | **APPROVED / FROZEN.** Standalone CRM, Go modular monolith, PostgreSQL, React + TS + Vite, WP/Woo via connector only, strangler migration per Blueprint ADR N-22, no big-bang. |
| Money | Slice 0 money representation | **APPROVED / FROZEN FOR SLICE 0.** `int64` + explicit IRR/IRT; no implicit conversion (§9D). |
| P-1 | Blueprint / prior decisions sufficient for Slice 0 | **CLOSED** by owner approval of D-01, D-02 and the money contract. |
| P-2 | Owner acceptance of this E0 record | **CLOSED** by this owner approval. |

### B. Does not block Slice 0; blocks a later phase (UNRESOLVED)

D-03…D-26 are **not resolved** by this record. They remain gated at the later phases defined
by the Blueprint. The decisions below are those whose content is recorded in this repository.

| ID | Question | Why | Earliest phase blocked | Safe default until decided |
|---|---|---|---|---|
| D-03…D-22 | Not recorded on main | Under rule 1 (§5) they carry no authority in this repository until committed | Any later phase that relies on them (not Slice 0) | No target behaviour depending on them is implemented; each must be committed to main before the phase that relies on it starts |
| D-23 | Invoice Review visibility / PII / export policy | Legacy is organisation-wide, uncapped CSV with name, phones, address, postcode | Any Invoice Review read model or export in the target | Target ships no export. Read access only behind a flag, with read/export audit. Legacy unchanged. |
| D-24 | Invoice Review product/design owner | No frozen SN-207 design exists | Target Invoice Review UI | Legacy UI only; no target UI work |
| D-25 | Gateway auto-approval policy | Payment marked `approved` without Finance review | Payments/finance slice | Preserve as distinct provenance (`gateway_auto`); never present as Finance-reviewed |
| D-26 | Customer auto-login after gateway callback | Session issued off a payment callback | Portal/auth/payments slice | Not replicated in target; legacy unchanged |
| OPD-06 | Money business conversion / rounding / normalization | Toman vs Rial, tolerances | Billing/payments | Explicit unit, no conversion |
| OPD-01, -03, -04, -05, -08, -09, -10 | Per `CRM-CROSS-ROLE-ARCHITECTURE-V1` §19 | Already registered | Their respective domains | As in Phase-0 spec §21 |

### C. Informational / closed

| ID | Status |
|---|---|
| D-27 | **CLOSED (reconciled).** SN-208 merged to main as 86859d7 (PR #20); SN-208A as 7b65390 (PR #22). No owner action. |

## 8. Domain boundary map

Rule: one writer per aggregate. Cross-domain access goes through the owning domain's Go
interface or events, never its tables. Only `integration` knows legacy tables.

| Domain | Owns (aggregates) | Mode | May depend on | Forbidden | Legacy sources |
|---|---|---|---|---|---|
| kernel | result, refs, reasoncodes, money, timeutil (values only) | n/a (pure) | stdlib | every domain, DB, net, OS clock | Phase-0 spec §5–7 |
| identity | User, Credential, Session | write | kernel, audit | org internals; phone as key | wp_users/usermeta, roles/caps, crm-login |
| org/HR | Profile, Position, Assignment (hierarchy), Unit/Team, Compensation | write | kernel, identity (ref), audit | billing, finance tables | `sn_hr_*`, class-sn-hr-service / -hr-transfer / -hierarchy-service |
| customers/contacts | Customer, ContactPoint | write | kernel, audit | phone as identity/join proof | Woo customers, invoice customer fields |
| leads/cases | Lead, Case, CaseAlias (evidence) | write | kernel, customers (ref), audit | financial status writes | sn_leads, sn_lead_statuses, sn_lead_status_history, sn_mis_data_rows, sn_mis_lead_pool, sn_dot_cases |
| assignments/custody | CustodyRecord, DistributionPlan/Chain | write (append-only history) | kernel, org (ref), leads (ref), audit | rewriting historical custody | `sn_distribution_*`, `sn_mis_distribution_*`, sn_hr_assignments (lead side) |
| sales | SalesActivity, pre-invoice intent, catalog view | write | kernel, leads, customers, org (ref) | invoice/payment state writes | seller-flow, sales-catalog, product-flow, projects/operations |
| billing/invoices | Invoice, InvoiceItem, PaymentStage | write | kernel, customers (ref), sales (ref), audit | payment verification, ledger posting | sn_invoices, sn_invoice_items, sn_invoice_payment_stages, class-sn-invoice |
| payments | Payment, GatewayAttempt, Receipt, Reconciliation | write | kernel, billing (ref/events), integration (gateway port), audit | ledger writes; session issuance | sn_payments, payment-transactions, gateway callbacks |
| finance/ledger | Wallet, LedgerEntry (append-only), FinancialReview | write | kernel, billing/payments (events), audit | mutating posted entries | sn_wallets, sn_wallet_transactions, finance review fields |
| commission | CommissionRule, CommissionRun, posting request | write (rules/runs) | kernel, org (ref), billing (events), finance (port) | direct ledger writes | `sn_commission_*`, purpose-commission |
| reporting/MIS | none (read models) | read-only / query | published events, read models of owners | any write to owner aggregates | report-service / -executive, MIS panels, sn_report_invoice_state_events |
| ↳ Invoice Review | **query surface under billing/reporting, not a domain** | read-only | billing, payments read models | new schema, writes | class-sn-invoice-review.php |
| after-sales | AfterSalesCase | write | kernel, customers, billing (ref) | invoice/payment writes | sn_after_sales_cases, sn_after_sales_case_logs |
| Biawin / operational services | ServiceOrder, ExternalOpRef | write | kernel, integration (Biawin port), finance (port) | direct wallet mutation | class-sn-biavin-api, shipping orders |
| integration/connector | LegacyImportBatch, SyncCursor, Inbox/Outbox records | write (own tables only) | kernel; WP/Woo/gateway/SMS/Biawin externals | exposing legacy schema to domains | all `wp_*`/`sn_*` tables, Woo, gateways, SMS |
| audit | AuditEvent (append-only) | write (append only) | kernel | update/delete of events | sn_activity_logs, `*_logs` tables |

## 9. Cross-cutting contracts (only what Slice 0 may code)

**A. Result.** Status ∈ {SUCCESS, PARTIAL, FAILED, UNKNOWN, CONFLICT, BLOCKED, UNAUTHORIZED}
(closed set). `ItemOutcome{ItemRef?, Status, ReasonCodes[], Message?, CorrelationID?,
BeforeRef?, AfterRef?, Metadata (allowlisted)}`. A nil ItemRef requires a missing-ref reason.
Aggregation (deterministic, order-independent):
- empty set → contract error (not SUCCESS)
- all SUCCESS → SUCCESS
- all the same non-SUCCESS class → that class
- any mix that includes ≥1 SUCCESS → PARTIAL (per-item classes preserved)
- mix without SUCCESS: UNKNOWN wins only if every item is UNKNOWN; otherwise PARTIAL
- UNKNOWN is never coerced to FAILED or SUCCESS. Transport/HTTP success ≠ business SUCCESS.

Semantics per Phase-0 spec §5 (preserved under D-01).

**B. Refs.** `NativeRef{Domain, ID string}` (decimal string IDs, domain-qualified; equal IDs
in different domains are unequal). `SourceRef{Native, OriginBatch?, OriginActor?, OriginAt?,
Provenance ∈ VERIFIED|INCOMPLETE|UNKNOWN|NOT_APPLICABLE}`. `CaseRef{Natives (non-empty),
Sources, CanonicalID (never minted), Resolution ∈ RESOLVED|RELATED|UNKNOWN|CONFLICT}`.
RESOLVED requires a supplied CanonicalID (`GATE-0-PRODUCT-INVARIANTS.md` §3: `canonical_id` required for resolved CaseRef);
a proven grouping without an existing canonical ID is RELATED, never RESOLVED. CanonicalID is nil
for RELATED/UNKNOWN/CONFLICT unless an existing one is supplied.
`LegacyRef{System (wp|woo|gateway|biawin), Table/Entity, ID}` is opaque and only constructible
by the integration package. A phone, name or token is never a valid ID in any ref (validation
rejects it). Nil/UNKNOWN/N/A are distinct; zero is never a sentinel.

**C. Reason codes.** Versioned taxonomy `foundation-v1` (Phase-0 spec §6 codes). Two kinds:
(1) CRM reason codes; (2) `LegacyValue{Source, Field, Raw}` preserved verbatim. Unknown raw
values map to UNKNOWN, never to false or a default. The known-legacy registry is
namespace-qualified (e.g. `invoice/approved`, `payment_stage/approved`) and must contain the full
frozen status catalog: every name/namespace row of `docs/source-of-truth/STATUS-TRANSITIONS.fa.md`
and every value in `GATE-0-PRODUCT-INVARIANTS.md` §8 Status Namespaces (invoice, invoice_workflow,
payment_stage, transaction, Dot transaction, finance_review, conversion, case/custody, contact,
hr_request, commission_run/item, wallet_transaction, report_job). It additionally includes the
legacy payment values `approved` and `reconcile_error` (`class-sn-plugin.php:7160-7164`) and the
customer-portal values `pending_payment`, `expired`, `failed` (`class-sn-customer-portal.php:806-814`).

**D. Money — APPROVED / FROZEN FOR SLICE 0** (OWNER-APPROVED).

```text
Money {
  Amount int64
  Unit   IRR | IRT
}
```

- no `float32` / `float64`
- unit is mandatory
- IRR and IRT are distinct units
- arithmetic only between equal units; cross-unit operations are rejected
- no implicit conversion
- unknown amount is absent, never zero
- overflow fails explicitly (never wraps)
- business conversion / rounding rules remain deferred to OPD-06 / later Finance decisions

**E. Time.** Canonical storage is a UTC instant. `SourceTime{UTC, SourceZone (IANA, e.g.
Asia/Tehran), RawText?}` preserves source evidence. Legacy MySQL `current_time('mysql')`
values are local-time text with an unknown/declared zone; parsing requires an explicit zone,
never the server default. Jalali is a formatter/parser for presentation and evidence only, never
a storage type. The clock is injected (no `time.Now` in kernel).

**F. Architecture import rules.** `internal/kernel/**` imports stdlib only.
`internal/<domain>/**` may import kernel and other domains' published `api` packages only;
no `internal/<other>/store` imports; no shared DB handles across domains. Only
`internal/integration/legacy/**` may contain WordPress/Woo table or column names (`wp_`/`sn_`
literals elsewhere fail lint). Enforced by an automated import/lint check in CI.

## 10. Legacy parity vs target-improvement classification

Classes: PARITY REQUIRED · PARITY TEMPORARY · TARGET IMPROVEMENT · RETIRE · OWNER DECISION REQUIRED.
These classify migration intent only. Nothing is fixed in legacy.

| Legacy behaviour | Evidence | Class |
|---|---|---|
| Phone used in identity joins | AGENTS.md; search fields in invoice-review | TARGET IMPROVEMENT (phone = search only) |
| Read-path writes (get_state→ensure_schema, wallet autopost on render, boot role/page creation incl. Invoice Review ensure_role/ensure_page) | Phase-0 spec §2; `class-sn-invoice-review.php:40-89` | TARGET IMPROVEMENT (reads pure) |
| Gateway path writes `approved` without Finance review | `class-sn-plugin.php:7158-7164` | OWNER DECISION REQUIRED (D-25); provenance preserved meanwhile |
| `reconcile_error` payment status | `class-sn-plugin.php:7160`; `class-sn-customer-portal.php:806,814` | PARITY REQUIRED (value preserved verbatim; target adds reconciliation workflow) |
| Customer auto-login after gateway callback | `class-sn-plugin.php:7168` | OWNER DECISION REQUIRED (D-26) |
| Invoice Review organisation-wide view | CHANGELOG-2.0.158; `class-sn-invoice-review.php` | PARITY TEMPORARY pending D-23 |
| Invoice Review uncapped CSV with PII | `class-sn-invoice-review.php:413-432` | OWNER DECISION REQUIRED (D-23) |
| HR position ↔ WP role dual mapping (invoice_reviewer ↔ sn_invoice_reviewer) | `class-sn-hr-service.php:24,526,554` | PARITY TEMPORARY (target: Position ≠ Role ≠ Capability) |
| Wallet reversal by mutation | Finance audit (OWNER-BRIEF) | TARGET IMPROVEMENT (compensating ledger entries) |
| Mutable custody history | DB audit (OWNER-BRIEF); hierarchy end/assign | TARGET IMPROVEMENT (append-only); history imported as-is |
| Live Woo product title with snapshot fallback | CHANGELOG-2.0.157 | PARITY REQUIRED for display; snapshot stays authoritative for history |
| MIS report module missing from plugin/current | `includes/reports` contains only report-service and report-executive | PARITY TEMPORARY — gap recorded; target MIS built from MIS-PRODUCT-SPEC |
| Commission on assignment | Commission audit (OWNER-BRIEF) | OWNER DECISION REQUIRED (existing OPD-08/09) |
| Biawin wallet charge | Biawin Deep Audit (OWNER-BRIEF) | PARITY TEMPORARY behind connector; OWNER DECISION before target writes |
| Finance/wallet/commission/Biawin legacy risks | Prior audits | Unchanged by 2.0.160; carried as deferred risks (§13) |

Blueprint delta patch (2.0.160), applied only to the changed facts:
- Invoice Review = role-specific read/query workspace under billing/reporting.
  Role `sn_invoice_reviewer`, capability `sn_review_invoices`, HR position `invoice_reviewer`.
- No new domain, no new schema (legacy page/option/role created on init/admin_init only).
- Organisation-wide parity behaviour exists in legacy; export is uncapped and contains PII.
- Gateway path may write `approved` without Finance review; `reconcile_error` is a known legacy
  payment status; gateway callback may auto-login the customer.
- Portal product title is a live-display value with snapshot fallback.
- MIS report module remains missing from plugin/current; read-path write concerns remain.
- Status filter: main status first, fallback to invoice/payment status (2.0.159).
- Finance / wallet / commission / Biawin legacy risks unchanged.
- All other Blueprint sections unchanged.

## 11. Slice 0 frozen scope — "Go Contracts Spine"

Allowed: `go.mod` and minimal repo skeleton; `internal/kernel/{result,refs,reasoncodes,money,timeutil}`;
architecture import-lint rules; unit + property tests; a separate GitHub Actions workflow for Go
(path-filtered; does not alter `plugin-snapshot-ci.yml`).

Excluded: PostgreSQL, schema/migrations, HTTP server, REST/OpenAPI endpoints, auth/authz runtime,
WP/Woo connector, legacy importer, queues/jobs, outbox/inbox, billing/payments/finance/commission/
Biawin implementation, React frontend, production/staging access, any change under `plugin/current`,
the PHP Phase-0 harness (D-01), any network or file I/O in kernel packages.

## 12. Slice 0 acceptance criteria (objective)

1. `go build ./...`, `go vet ./...`, `gofmt -l` (empty) and `go test -race ./...` pass in CI on the PR head.
2. Only `go.mod`, `go.sum`, `internal/kernel/**`, the lint config/tool, tests and the new workflow
   file change. `git diff --stat main -- plugin/ docs/source-of-truth/` is empty.
3. `go list -deps ./internal/kernel/...` contains only stdlib (no third-party, no `database/sql`, `net/*`, `os/exec`).
4. Result: tests cover all 7 statuses; empty aggregate → error; every aggregation rule in §9A;
   property test: aggregation is permutation-invariant; UNKNOWN never becomes SUCCESS/FAILED.
5. Refs: tests reject phone-shaped, empty, zero, negative and non-decimal IDs; cross-domain equal IDs
   compare unequal; CaseRef rejects empty Natives; CaseRef with Resolution RESOLVED and nil
   CanonicalID is rejected; CanonicalID is never generated; serialization round-trip property test.
6. Reason codes: registry versioned `foundation-v1`; a table-driven test asserts every
   namespace/value of the frozen status catalog (§9C) is registered as known (none maps to UNKNOWN),
   including payment `approved` and `reconcile_error`; unknown raw → UNKNOWN with raw preserved
   byte-for-byte (property test over arbitrary strings).
7. Money: no `float32/float64` in kernel (lint-enforced); Unit required (zero-value Unit invalid);
   cross-unit ops rejected; no conversion function exists; overflow detected (property test at
   int64 bounds); absent ≠ zero.
8. Time: UTC normalization; parsing without an explicit zone fails; Asia/Tehran evidence round-trips;
   Jalali↔Gregorian conversion tests over a fixed date table including leap years; no `time.Now` in kernel.
9. Import lint: a deliberately violating fixture (kernel→domain import; `sn_`/`wp_` literal outside
   integration) fails the check in CI.
10. Coverage ≥ 90 % statements for `internal/kernel/**`.
11. Fixtures are synthetic only (no real names/phones/IDs).
12. One PR into main, owner merge, no force-push.

## 13. Deferred risks / decisions

- D-03…D-26 remain unresolved and gated at later phases (§7B); this record resolves none of them.
  D-03…D-22 are not recorded on main and must be committed before any phase that relies on them.
- D-23/D-25/D-26 legacy behaviours remain live in production (PII export, auto-approval, auto-login).
- OPD-06 business conversion / rounding / normalization of money remains open.
- Legacy read-path/boot writes (incl. Invoice Review role/page creation).
- Canonical invoice-total and main-status resolvers deferred; legacy formulas are evidence only.
- MIS report module absent from plugin/current.
- Finance/wallet/commission/Biawin risks unchanged from prior audits.
- Staging/anonymization not yet available; it gates any real-data import.
- HR and Finance runtime NOT LIVE VERIFIED.
- The prior audit corpus / Blueprint is cited by owner approval, not committed to this repository.
- Local workspace (`D:\…`) not inspected; verification was against the GitHub clone at 86859d7.
- Manual Plugin Snapshot CI run on 86859d7 taken from the owner brief, not re-queried from Actions.

## 14. Final E0 verdict

**E0 FROZEN — SLICE 0 APPROVED**

Satisfied: baseline re-verified at 86859d7 with no drift; D-01 and D-02 APPROVED / FROZEN;
money contract APPROVED / FROZEN FOR SLICE 0; contracts required by Slice 0 resolved (§9);
Slice 0 bounded (§11–12); no production dependency; P-1 and P-2 CLOSED; D-27 CLOSED;
no unresolved owner decision blocks Slice 0.

Slice 0 implementation may begin only after the Human Owner merges this documentation PR.
