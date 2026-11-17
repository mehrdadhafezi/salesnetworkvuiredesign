# RM-E04A — Seller Row Scope: Seven-Screen Read Contracts

**DRAFT — NOT APPROVED. Every proposed endpoint and DTO below is NOT IMPLEMENTED.** Baseline `main` @ `d85b770` (unchanged at `e5fabbe`). Production **NO-GO**.

This document turns the RM-E03 Seller screens (S1–S7) into server read contracts. It records what the server returns today, what is missing, and what a scoped contract would need. It selects **no** capability, scope, predicate or field policy. Those choices belong to the records in [`docs/governance/proposals/rm-e04a-seller-decisions.md`](../governance/proposals/rm-e04a-seller-decisions.md). No business state is invented: where the server has no state, the contract says **NO SERVER SOURCE**.

Cross-references:
- Findings F-01 … F-09 and security requirements SR-01 … SR-16: [`docs/implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md)
- Per-operation leak inventory (105 operations): [`rm-e04a-seller-leak-surface-audit.md`](rm-e04a-seller-leak-surface-audit.md)
- Negative tests NT-01 … NT-24: [`rm-e04a-seller-negative-test-plan.md`](rm-e04a-seller-negative-test-plan.md)
- PR sequence: [`docs/implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md)

## 0. Conventions used by every screen

| Topic | Rule (applies to S1–S7 unless a screen says otherwise) |
|---|---|
| Endpoint shape | Two alternatives, **not selected** (parent §4.1): **Option A** = new own-scope operations under `/api/v1/me/…` with their own `*.own.read` capability (precedent: `GET /api/v1/commission/me/earnings`). **Option B** = the existing operation, with a narrower `Decision.Scope` applied by the handler. The tables name Option A paths for readability only. |
| Method | `GET` only. No Seller write is in scope; every write stays a demo toast in RM-E03 (§4 of the RM-E03 doc). |
| Envelope | Existing envelope: `outcome`, `reasons`, `correlation_id`, `notice`, `items`, `page {next_cursor, has_more}`, `allowed_actions`. No new envelope. |
| Authorization guard | (1) feature flag of the module (`GateDisabled` → 200 `BLOCKED / FEATURE_DISABLED`, no rows); (2) capability through an ACTIVE grant (`GateDenied` → 403 `ACTION_DENIED`); (3) **exhaustive scope switch** (SR-03, closes F-02); (4) `Subject.UserID` proven, else 403 `ROW_SCOPE_DENIED` (SR-04); (5) row predicate inside SQL (SR-01). |
| Pagination | Cursor + `limit` as today (`PageInfo {next_cursor, has_more}`). The cursor must encode only a position inside the scoped ordering. A cursor from another user or from an organisation list must not reveal rows (NT-06). |
| Sorting | Today lists sort by a fixed key (e.g. zero-padded `source_key`). Any new sort key must be computed after the predicate. Sort by an ownership field of another user is not offered. |
| Counts | Lists have **no total today**. Any Seller count (queue chip, KPI tile) must be computed over the scoped set (SR-07), returned by the same scoped operation, and never taken from `/reports/*`, `/sales/coverage-report` or other organisation aggregates. A count that cannot be computed is returned as absent with a reason, never `0` (AGENTS.md). |
| Cache | `Cache-Control: no-store` and `Vary: Cookie` on every response (SR-12, F-07). |
| UNKNOWN / CONFLICT | Rows whose ownership fact is NULL, UNKNOWN, CONFLICT, MULTIPLE or NOT_RECOVERABLE are **outside** own scope (SR-05). Inside a row, unknown business facts are rendered as UNKNOWN (dashed/amber) and never as zero. |
| PARTIAL | When a row is in scope but some facts are incomplete (`items_source_complete=false`, `stages_source_complete=false`, `payment_evidence_complete=false`, `unknown_state_count>0`), the response `outcome` is `PARTIAL` with a reason. Amounts that depend on missing facts are absent. |
| Not found | Out-of-scope `/{id}` = missing `/{id}` (same 404 problem body; SR-06). |
| Phone | Discovery data only; never a join, never ownership proof. Phone search only after the customer field policy (GAP-04). |
| Frontend | RM-E03 module `web/src/modules/seller/`. Replace fixtures with calls through `web/src/shared/api/client.ts` (generated types from `web/src/generated/api/schema.ts`). Destinations appear only through `web/src/app/navigation/destinations.ts` / `roles.ts` with a CONTRACT capability (RM-ACCESS-01 §11). `types.ts` stays presentation-only. The `/dev/seller` preview stays DEV-only. |
| Provenance | Every read model is a shadow projection of legacy 2.0.160 (WordPress plugin). Legacy remains the **authoritative writer**; the target reads only. |

Sensitivity labels: **S-PII** (customer contact data), **S-FIN** (money facts), **S-STAFF** (other staff identities), **S-OP** (operational, low).

---

## S1 — شماره‌های من (Leads)

| Item | Contract |
|---|---|
| Purpose | Seller's own leads / lead cases, with queue chips and a contact-outcome drawer |
| Existing endpoints | `GET /api/v1/cases` (`ListCases`, `cases/http/http.go:105`); `GET /api/v1/cases/{case_id}` (`GetCase`, `:144`); `GET /api/v1/cases/{case_id}/history` (`:180`); `GET /api/v1/cases/{case_id}/contact-outcomes` (`GetCaseContactOutcomes`, `cases/http/write.go:54`); `GET /api/v1/custody/cases/{case_id}` (`custody/http/http.go:114`). **All organisation-scoped** (leak audit rows 24–27, 29, 90). |
| Proposed endpoints (NOT IMPLEMENTED) | `GET /api/v1/me/cases` · `GET /api/v1/me/cases/{case_id}` · `GET /api/v1/me/cases/{case_id}/contact-outcomes` |
| Capability / flag | Existing: `cases.case.read` / `sales.read_shadow` (session feature `sales_read`). Option A would need a new capability (e.g. a `cases.own.read`-style key). **Name not chosen; any new key follows RM-ACCESS-01 §11.** |
| Guard | §0 steps 1–5. Predicate source: custody current holder (`custody/query/query.go:197-223` returns `Known` only when PRESENT, single holder and parity not CONFLICT / MULTIPLE_CURRENT). |
| Existing response fields | `CaseSummary`: `case_id, workflow_type, state_namespace, state_raw, state_mapping, customer_id, customer_link_state, customer_link_reason, presence, source{channel_raw, origin_ref_raw…}, alias_count, history_count, imported_at, refreshed_at`. `CaseDetail.lead` (`LeadFacts`): `status_raw, lead_status_raw, destination_panel_raw, imported_at_raw, assigned_at_raw, updated_at_raw, presence`. Contact outcome: `current{outcome, reason, callback_at…}`, `history[]` (with `actor_user_id`). |
| Missing fields | Customer display name and phone (only `customer_id`; contact points are a separate, more sensitive read, GAP-04); queue membership (RM-E03 queues have **no server definition**); "probability"; per-queue counts; holder basis / as-of in the case row. |
| Proposed DTO (NOT IMPLEMENTED) | `SellerCaseRow { case_id, workflow_type, state_raw, state_mapping, presence, lead{lead_status_raw, assigned_at_raw, updated_at_raw}, source{channel_raw}, customer_ref{customer_id?, link_state}, contact_outcome_current?{outcome, callback_at}, scope_basis: "CUSTODY_CURRENT_HOLDER"|…, evaluated_as_of }` — every field maps to an existing schema field; `scope_basis` restates the custody `Basis` string. |
| Provenance / sensitivity | `cases.work_case`, `cases.lead`, `cases.contact_outcome_*`, `custody.assignment` (legacy import, batch-refreshed). S-OP; `customer_ref` S-PII-adjacent; history actors S-STAFF. |
| Ownership predicate alternatives | M2 rec. 2 options: (1) current holder via custody legacy columns; (2) current legacy assignment `sn_leads.seller_id` (needs M2 Q10 evidence); (3) current + historical holder; (4) organisation; (5) decide later. **Not selected.** |
| Field-level visibility | Show: case facts, own contact outcomes. **Decision needed:** whether `actor_user_id` of other staff in history is shown (S-STAFF) and whether any customer contact data is shown (GAP-04). Default until decided: withhold both. |
| Search / sort / pagination | No search parameter exists. Proposed search (case code, customer name, phone) only inside the predicate (SR-08) and only after GAP-04 for name/phone. Sort: newest assignment (`assigned_at_raw`, raw text — parsing rules needed) or fixed key. Cursor per §0. |
| Counts / aggregation | Queue counts only if the queue definition becomes a server contract; otherwise **no counts** (UI shows «—» with a reason, as RM-E03 does). Never organisation counts. Unknown-holder rows are not counted. |
| Cache | `no-store`, `Vary: Cookie`. |
| UNKNOWN / CONFLICT / PARTIAL | Holder unknown / conflicting → row excluded (SR-05). `state_mapping` UNKNOWN → shown as UNKNOWN. Contact-outcome history truncated at 200 events (`cases/command/read.go:66`) → `PARTIAL` if truncated. |
| Frontend integration | `pages/LeadsPage.tsx`, `state/selectors.ts` (queue/search logic moves server-side or is removed), `components/overlays.tsx` (`OutcomeGrid` stays a demo; the write is not wired), `RoleNavigation` destination with a CONTRACT capability. |
| Governance blockers | M2 rec. 1 (does the Seller hold any case read), M2 rec. 2 (predicate), D-42 (who may grant), D-34 (account), GAP-04 / D-30 (customer fields), M2 Q10. |
| Future implementation | (1) F-02 hardening; (2) scoped query with predicate in SQL + 404 parity; (3) contract tests; (4) wire `LeadsPage`. |
| Future tests | NT-01, NT-03, NT-05, NT-06, NT-07, NT-09, NT-10, NT-14, NT-16, NT-17, NT-21 |
| **Completion status** | **CONTRACT COMPLETE — implementation BLOCKED** (M2 rec. 1, rec. 2) |

---

## S2 — آماده‌های تبدیل (Conversions)

| Item | Contract |
|---|---|
| Purpose | Cases ready for conversion, with attribution facts (current custody · next actor · event actor · credit owner) |
| Existing endpoints | `GET /api/v1/cases/{case_id}/attribution` (`ListCaseAttribution`, `cases/http/http.go:212`); `GET /api/v1/invoices?…` with `case_link_state` filter (`billing/http/http.go:153`). Organisation-scoped. |
| Proposed endpoints (NOT IMPLEMENTED) | `GET /api/v1/me/conversions` — **cannot be specified**: "ready for conversion" has **NO SERVER SOURCE** (no status, no workflow contract). `GET /api/v1/me/cases/{case_id}/attribution` is specifiable. |
| Capability / flag | `cases.case.read` / `sales.read_shadow` for attribution; `billing.invoice.read` / `billing_payments.read_shadow` for linked invoices. New own capability under Option A: not chosen. |
| Guard | §0. The attribution sub-resource re-applies the parent case predicate (SR-06). |
| Existing response fields | `AttributionSnapshot`: `snapshot_id, facet (ORIGINAL_OWNER / ORIGINAL_SELLER / CREDIT_OWNER), resolution, subject, position_raw, basis, source_time_raw, evidence_spec, evidence_key, captured_batch_id, captured_at`. |
| Missing fields | Conversion readiness; **next actor** (not modelled in any read); current custody inside the attribution response (separate custody read); conversion remaining amount. |
| Proposed DTO (NOT IMPLEMENTED) | `SellerAttributionView { case_id, current_custody{basis, is_caller}, facets[]{facet, resolution, is_caller?, basis, captured_at}, next_actor: UNKNOWN (no source), linked_invoices[]{invoice_id, case_link_state} }`. `is_caller` replaces another seller's identity when the field policy withholds S-STAFF. |
| Provenance / sensitivity | `cases.attribution_snapshot` (append-only, historical), `custody.assignment`, `billing.invoice.case_id`. S-STAFF (other owners), S-FIN (linked invoices). |
| Ownership predicate alternatives | Same as S1 for the case. Which facet admits a case to "my conversions" (current custody vs original seller vs credit owner) is **a separate decision**; facets are never collapsed (SR-09). |
| Field-level visibility | Facet subjects that are not the caller: **decision needed** (withhold by default; show resolution only). |
| Search / sort / pagination | Not specifiable until the list exists. |
| Counts / aggregation | None until the list exists. |
| Cache | `no-store`. |
| UNKNOWN / CONFLICT / PARTIAL | Facet `resolution` UNKNOWN / CONFLICT / NOT_RECOVERABLE shown as such; credit owner UNKNOWN stays UNKNOWN (RM-E03 §8). Invoice links with `case_link_state ≠ RESOLVED` are not used to pull an invoice into scope. |
| Frontend integration | `pages/ConversionsPage.tsx` (ownership facts already use the fixed vocabulary). |
| Governance blockers | M2 rec. 1, M2 rec. 2, a product/workflow contract for conversion readiness (owner: Codex/Product), OPD-10 (assisted issuance), ADR-01 (issuance vs return). |
| Future implementation | Attribution sub-resource only, after S1; list after a readiness contract exists. |
| Future tests | NT-03, NT-12, NT-14, NT-22 |
| **Completion status** | **PARTIAL — attribution read specifiable; list BLOCKED (NO SERVER SOURCE for readiness)** |

---

## S3 — نیاز به اقدام مجدد (Repeat Actions)

| Item | Contract |
|---|---|
| Purpose | Items handed back to the original seller for a repeat action (issuer ≠ current custody ≠ next actor) |
| Existing endpoints | `GET /api/v1/invoices/{invoice_id}/stages` (`ListInvoiceStages`, `billing/http/http.go:252`); `GET /api/v1/custody/cases/{case_id}`. Organisation-scoped. |
| Proposed endpoints (NOT IMPLEMENTED) | `GET /api/v1/me/repeat-actions` — **cannot be specified**: no hand-off model and no "needs repeat action" state exist (**NO SERVER SOURCE**). |
| Capability / flag | Would combine billing and custody reads; capability not chosen. |
| Guard | §0, plus the predicate decision for "original seller" vs "current custody". |
| Existing response fields | `PaymentStage`: `stage_id, stage_no_raw, stage_type_raw, requested, state_namespace, state_raw, state_mapping, state, pay_method_raw, payment_source_raw, issued_by, approved_by, approved_at_raw, paid_at_raw, created_at_raw, source`. |
| Missing fields | Hand-off reason, next actor, repeat-action due state, case↔stage hand-off link. |
| Proposed DTO (NOT IMPLEMENTED) | Placeholder only: `SellerRepeatItem { invoice_id, stage_id, original_issuer{is_caller}, current_custody{basis, is_caller}, next_actor: UNKNOWN, reason: NO SERVER SOURCE }`. Not to be built until a contract exists. |
| Provenance / sensitivity | `billing.payment_stage`, `custody.assignment`. S-FIN, S-STAFF. |
| Ownership predicate alternatives | (a) original issuer = caller (RM-E03 copy: "مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود"); (b) current custody = caller; (c) both. **Not selected; needs OPD-10.** |
| Field-level visibility | Identities of the current custodian / next actor: decision needed. |
| Search / sort / pagination | Not specifiable. |
| Counts / aggregation | Not specifiable. |
| Cache | `no-store`. |
| UNKNOWN / CONFLICT / PARTIAL | Any of the three facts unknown → shown UNKNOWN; never inferred from hierarchy. |
| Frontend integration | `pages/RepeatPage.tsx` (keeps synthetic rows and the prototype empty state). |
| Governance blockers | Product/workflow contract (hand-off), OPD-10, M2 rec. 2, D-32 (next-stage authority), OPD-02 (escalation owner). |
| Future implementation | None until the contract exists. |
| Future tests | NT-12, NT-22 (reserved) |
| **Completion status** | **BLOCKED — NO SERVER SOURCE (contract documented to the extent facts exist)** |

---

## S4 — فاکتورها (Invoices)

| Item | Contract |
|---|---|
| Purpose | Seller's invoices with four money facts (total · financially confirmed · remaining · current payable) and stages |
| Existing endpoints | `GET /api/v1/invoices` (`billing/http/http.go:153`), `/{invoice_id}` (`:184`), `/items` (`:218`), `/stages` (`:252`), `/totals` (`:303`); `GET /api/v1/invoices/{invoice_id}/payments`, `/receipts` (`payments/http/http.go:167, 251`); `GET /api/v1/payments/invoices/{invoice_id}/submission` (`payments/http/write.go:142`, owner-only amounts, F-09). Billing / payments reads are **organisation-scoped**. |
| Proposed endpoints (NOT IMPLEMENTED) | `GET /api/v1/me/invoices` · `/me/invoices/{invoice_id}` · `/me/invoices/{invoice_id}/stages` · `/me/invoices/{invoice_id}/totals` · `/me/invoices/{invoice_id}/payments` |
| Capability / flag | Existing: `billing.invoice.read`, `payments.attempt.read` / `billing_payments.read_shadow` (`billing_read`). Submission context: `payments.submission.read` / `payments.manual_write`. RM-B02 matrix lists invoice / attempt reads as **C** (conditional) for the Seller — not granted. |
| Guard | §0. Precedent predicate: `payments/command/command.go:156-160` (seller ∧ issuer; NULL → `DEPENDENCY_UNKNOWN`). |
| Existing response fields | `InvoiceSummary`: `invoice_id, code_raw, lifecycle_state, customer, case, seller, issued_by, items_source_complete, stages_source_complete, item_count, stage_count, unknown_state_count, issued_at_raw, source`. `InvoiceTotals`: `compared_declared, payment_evidence_complete, unclassified_attempts, units[]{unit, declared, item_sum, stage_sum, payment_sum, class}`. Submission context: `due_amount, remaining_amount, pending_or_accepted_amount, available_amount` (owner only). |
| Missing fields | A server field named "financially confirmed" amount — **none exists**; deriving it from attempt `state` / `approval_provenance` is a Finance rule (GAP-03, D-25, D-43) and is not invented here. "Current payable" exists only in the owner-only submission context. |
| Proposed DTO (NOT IMPLEMENTED) | `SellerInvoiceRow { invoice_id, code_raw, issued_at_raw, case_ref{case_id?, link_state}, customer_ref{customer_id?, link_state}, completeness{items, stages, payment_evidence}, unknown_state_count, units[]{unit, declared, payment_sum, class}, confirmed: ABSENT until Finance rule, current_payable?: from submission context, scope_basis }` |
| Provenance / sensitivity | `billing.invoice`, `billing.payment_stage`, `billing.invoice_item`, `payments.*` (legacy import). S-FIN; `customer_ref` S-PII-adjacent; `approved_by` S-STAFF. |
| Ownership predicate alternatives | (a) `seller_user_id = caller`; (b) `issued_by_user_id = caller`; (c) both (write-path precedent); (d) linked case's current holder (only `case_link_state = RESOLVED`). **Not selected.** NULL seller / issuer → excluded (F-04: no link-state column). |
| Field-level visibility | `approved_by` / stage `issued_by` of other staff: decision needed. Customer name / phone: GAP-04. |
| Search / sort / pagination | No search today. Proposed search by invoice code inside the predicate. Sort by `issued_at_raw` (raw text; parsing rules needed) or fixed key. Cursor per §0. |
| Counts / aggregation | Seller KPI «مانده قابل وصول» only as a scoped sum per unit, with `PARTIAL` when any included invoice has unknown or incomplete facts. Never sum across units. Never from `/billing/reconciliation-report` or `/reports/invoices`. |
| Cache | `no-store`, `Vary: Cookie`. |
| UNKNOWN / CONFLICT / PARTIAL | `unknown_state_count > 0` or incomplete stages → `PARTIAL`; stage `state_mapping` UNKNOWN → Outcome Unknown in UI, "current payable" absent (RM-E03 §8). Cancelled → remaining absent. |
| Frontend integration | `pages/InvoicesPage.tsx`, `InvoiceDrawer`, `components/bits.tsx` (`Money`), `state/selectors.ts` invoice figures (move to server values). |
| Governance blockers | M2 rec. 1 (invoice read for Seller: matrix C), M2 rec. 2 (predicate), GAP-03 (Finance row scope), D-25 / D-43 (confirmation semantics), OPD-10. |
| Future implementation | After S1: scoped invoice list + detail + stages + totals, then payments sub-resources. Fix F-09 (404 parity) in the same series. |
| Future tests | NT-03, NT-05, NT-06, NT-08, NT-11, NT-13, NT-15, NT-17, NT-21 |
| **Completion status** | **CONTRACT COMPLETE — implementation BLOCKED** (M2 rec. 1, rec. 2; confirmed amount needs a Finance rule) |

---

## S5 — رفتار مشتریان (Customer Behaviour)

| Item | Contract |
|---|---|
| Purpose | Link opened / visit / paid after visit events per customer |
| Existing endpoints | **None.** No table, projection or operation holds these events (searched `internal/*/migrations/body`). |
| Proposed endpoints (NOT IMPLEMENTED) | None can be specified until an event source exists. |
| Capability / flag | None exists. |
| Guard | Would follow §0. |
| Existing response fields | None. |
| Missing fields | All. |
| Proposed DTO (NOT IMPLEMENTED) | Not proposed (no source). RM-E03 already shows «نامعلوم» / unavailable. |
| Provenance / sensitivity | Unknown source; would be S-PII (behavioural data on customers). Privacy review (Legal) required. |
| Ownership predicate alternatives | Would inherit S1 / S4 (customer reachable only through an in-scope case or invoice). |
| Field-level visibility | Privacy decision required (GAP-04, Legal). |
| Search / sort / pagination | N/A. |
| Counts / aggregation | N/A; never show 0 for an unknown event count. |
| Cache | `no-store`. |
| UNKNOWN / CONFLICT / PARTIAL | Everything UNKNOWN. |
| Frontend integration | `pages/BehaviorPage.tsx` stays synthetic. The «مشاهده فاکتور» link navigates to S4 only. |
| Governance blockers | Product data contract, GAP-04, privacy review (ADR #18 for real data). |
| Future implementation | None. |
| Future tests | NT-22 (no endpoint may be added without SR coverage) |
| **Completion status** | **BLOCKED — NO SERVER SOURCE** |

---

## S6 — درخواست اضافه کردن شماره (Extra-Number Requests)

| Item | Contract |
|---|---|
| Purpose | Seller requests extra numbers and sees own requests |
| Existing endpoints | **None.** |
| Proposed endpoints (NOT IMPLEMENTED) | A read `GET /api/v1/me/number-requests` would be specifiable only after the workflow exists; the create command is out of scope. |
| Capability / flag | None exists. |
| Guard | §0; predicate would be `requester = caller` (pattern: HR requests, `organization/http/hrrequest.go:99`). |
| Existing response fields | None. |
| Missing fields | All (request states are **not defined** and are not invented here). |
| Proposed DTO (NOT IMPLEMENTED) | Not proposed until OPD-10 defines the workflow. |
| Provenance / sensitivity | Unknown. S-OP. |
| Ownership predicate alternatives | Requester only (only plausible option; still requires the workflow decision). |
| Field-level visibility | Reviewer identity: decision needed. |
| Search / sort / pagination | N/A. |
| Counts / aggregation | N/A. |
| Cache | `no-store`. |
| UNKNOWN / CONFLICT / PARTIAL | N/A. |
| Frontend integration | `pages/RequestPage.tsx` keeps the OPD-10 warning toast. |
| Governance blockers | **OPD-10**, M2 rec. 1. |
| Future implementation | None. |
| Future tests | NT-19, NT-22 |
| **Completion status** | **BLOCKED — NO SERVER SOURCE (OPD-10)** |

---

## S7 — کیف پول من (Wallet)

| Item | Contract |
|---|---|
| Purpose | Seller's own commission lines (confirmed / pending / settlement) |
| Existing endpoints | **`GET /api/v1/commission/me/earnings`** (`ListMyCommissionEarnings`, `commission/http/http.go:428`) — **own scope already implemented**: `holder_user_id = caller` in SQL (`commission/query/query.go:468`), no user → 403 `ROW_SCOPE_DENIED` (`:441-447`). |
| Proposed endpoints (NOT IMPLEMENTED) | None new for lines. A posting / settlement state read cannot be specified (NO SERVER SOURCE). |
| Capability / flag | `commission.own.read` / `commission.read_shadow` (`commission_read`). RM-B02 matrix: **A** for the Seller — proposed, **not granted** (every non-HR bundle is empty). |
| Guard | Existing: flag → capability → own path (`guardOp(..., own=true)`, `commission/http/http.go:63-86`) → UserID check. |
| Existing response fields | `CommissionEarning`: `source_key, engine, direction, type_raw, invoice_id, stage_raw, purpose_raw, share_facet_raw, amount, unit, reversed, occurred_at_raw, entitlement_key`; envelope `run`, `totals`, `page`. |
| Missing fields | Posting state, "confirmed vs pending", settleable amount, withdrawal — **NO SERVER SOURCE** (D-41, OPD-09, D-36). |
| Proposed DTO (NOT IMPLEMENTED) | None beyond the existing `CommissionEarning`. UI keeps «— تعریف نشده» for settlement. |
| Provenance / sensitivity | `commission.historical_entitlement` (legacy historical runs, shadow). S-FIN. |
| Ownership predicate alternatives | Already `holder_user_id`. Whether "credit owner" for the wallet equals entitlement holder is a Finance/Sales question (OPD-09), not changed here. |
| Field-level visibility | `invoice_id` links to S4; following the link must still pass the S4 predicate (an entitlement on an invoice does not admit the invoice into S4 scope unless S4's predicate admits it — **decision**). |
| Search / sort / pagination | Ordered by `source_key`; cursor + limit; `run_id` filter. No search. |
| Counts / aggregation | `totals` are own-scoped by construction. Units are never summed together. Unknown lines excluded from sums with a banner (RM-E03 §8). |
| Cache | **Missing today** (F-07) → add `no-store`. |
| UNKNOWN / CONFLICT / PARTIAL | No run → empty with outcome; partial runs follow `outcomeOf(run)`. |
| Frontend integration | `pages/WalletPage.tsx` — first candidate to wire, because the server scope exists. Still requires an ACTIVE grant (none possible until D-42 + M2 rec. 1). |
| Governance blockers | D-42 (grant administration), M2 rec. 1 (eligibility for `commission.own.read`), D-34, D-41 / OPD-09 / D-36 (semantics), ADR #18 (real data). |
| Future implementation | `no-store` (Group A); wire the page in an isolated test environment with a synthetic grant (Group B). |
| Future tests | NT-04, NT-12, NT-17, NT-18 |
| **Completion status** | **CONTRACT COMPLETE — scope PRESENT on server; semantics BLOCKED (D-41 / OPD-09)** |

---

## Summary of completion

| Screen | Status |
|---|---|
| S1 Leads | CONTRACT COMPLETE — implementation BLOCKED |
| S2 Conversions | PARTIAL — attribution specifiable; list NO SERVER SOURCE |
| S3 Repeat Actions | BLOCKED — NO SERVER SOURCE |
| S4 Invoices | CONTRACT COMPLETE — implementation BLOCKED |
| S5 Customer Behaviour | BLOCKED — NO SERVER SOURCE |
| S6 Extra-Number Requests | BLOCKED — NO SERVER SOURCE (OPD-10) |
| S7 Wallet | CONTRACT COMPLETE — server scope PRESENT; semantics BLOCKED |

All seven sections contain every required item. "BLOCKED" sections state explicitly which items cannot be specified and why, rather than inventing a source or state.
