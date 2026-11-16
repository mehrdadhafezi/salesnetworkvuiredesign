# RM-B02 — Scope, hierarchy and organizational ownership

**PROPOSAL — NOT APPROVED. Documentation only.** Baseline `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634`. Production **NO-GO**.

Companion to the [role-capability matrix](rm-b02-role-capability-matrix.md). Scope labels below are **analytical labels**, not implemented permission claims. Where a predicate is not implemented, this document says so.

## 1. Scope labels: predicate and implementation state

| Label | Proposed predicate (target) | Data source that would decide it | Implemented today? | Unknown / conflict behaviour |
|---|---|---|---|---|
| OWN | The row's owner facet equals the caller's account (e.g. invoice seller, uploader, batch submitter, commission recipient) | the facet named per object | **Yes, only** in `payments.upload.stage`, `payments.batch.submit`, `payments.submission.read`, `commission.own.read` | No owner → no row (fail closed) |
| ASSIGNED_CASES | Cases whose **current holder** is the caller (M2 option 1, engineering proposal, not selected); alternatives: current legacy assignment, current + historical, organization | `custody.assignment` (`derivation = 'DERIVED'`, `derived_user_id`) or legacy holder evidence | **Command only** (`cases.contact_outcome.record` checks the current legacy holder). **No read** is filtered this way | `UNKNOWN`, `MULTIPLE`, `RELEASED`, `NONE` → no seller sees the case (M2 §4-2) |
| ASSIGNED_QUEUE | Items where the caller is the requester or the current reviewer / assignee | the request's requester and current-reviewer snapshots | **Yes** in `org.hr_request.read` | Snapshot never re-derived from a later hierarchy |
| SPECIFIC_APPROVAL_SCOPE | One subject on which the caller holds a named step (current reviewer of this request; reviewer of this financial item) | request or review record | **Yes** for HR step review (current reviewer). Not for Finance (no decision capability) | Reviewer changed → stale → CONFLICT / recheck |
| DIRECT_TEAM | Subjects whose single current REPORTS_TO parent is the caller | organization current assignments | **No** (reads are ORGANIZATION) | Zero or several current parents → BLOCKED (S11 rule) |
| AUTHORIZED_SUBTREE | Descendants of the caller through current REPORTS_TO links, each step unambiguous | organization current assignments | **Command only**: `org.hr_request.create` (seller descendants). Not for reads | Ambiguous step → BLOCKED |
| GOVERNED_TERRITORY | The branches a Deputy (or Manager) governs. **Predicate UNRESOLVED**: DP §15 "authorized branches; no entire-org assumption"; no source defines "territory" beyond the subtree | unresolved | **No** | Until defined, treat as AUTHORIZED_SUBTREE or deny |
| ORGANIZATION | Every row in the module | — | **Yes**: the evaluator's only read scope (`authorization.go:163`) | — |

Additional proposed modifier used in the matrix: **linked item** (`LNK`): a Finance or MIS reader sees a case, customer or person only through the invoice / discrepancy it is working on. Not implemented.

**Implementation requirement for every restricted scope** (from M2 §5, unchanged): enforced on the server (evaluator and query), on every data-returning endpoint (lists, detail, history, attribution, outcomes, search, phone search, totals and counters, exports and reports); out-of-scope rows indistinguishable from missing rows; counts computed after scope; phone search limited to the same scope.

## 2. Implemented scope per capability (system facts)

| Capability group | Implemented scope | Where |
|---|---|---|
| All 6 HR reads, 3 customer reads, `cases.case.read`, `custody.custody.read`, `sales.report.read`, 3 billing reads, 3 reporting reads, 3 finance shadow reads, 3 Biawin/fulfillment reads, `integration.writeback.read`, `payments.evidence.read`, `finance.review.read`, `commission.shadow.read`, `commission.policy.read`, `biawin.activation.read`, `integration.provider_operation.read` | ORGANIZATION (no row filter) | evaluator; each module's `http.go` |
| `cases.contact_outcome.record` | actor == current legacy holder; unknown holder BLOCKED; any other actor UNAUTHORIZED (no `manage_options` override) | `internal/cases/http/write.go`; S11 |
| `org.hr_request.create` | subject is a seller descendant via current REPORTS_TO; actor has one LINKED ACTIVE employment with an eligible position slug; HR creator BLOCKED; SD-04 target inside the actor's scope | `internal/organization/http/hrrequest.go`; S11 |
| `org.hr_request.review` | actor == current reviewer snapshot; `pending_review` only; HR / `pending_hr` BLOCKED | same |
| `org.hr_request.read` | requested by or currently reviewed by the caller | same |
| `payments.upload.stage` | caller's own uploads | `internal/payments/http/write.go`; S12 |
| `payments.batch.submit` | `invoice.seller_user_id = actor` AND `issued_by_user_id = actor` | same |
| `payments.submission.read` | caller's own batches | same |
| `commission.own.read` | authenticated user only; never broadened by hierarchy | `internal/commission/http/http.go:430`; S13 |
| `fulfillment.work.read` / `assign` / `execute` | per-service SUPERVISOR / EXECUTOR scope events (catalog); no HR hierarchy | `internal/fulfillment/http/work.go`; S14 §5 |
| `integration.provider_operation.reconcile` | any proven user id; UNKNOWN / RECONCILIATION_REQUIRED operations only | `internal/integration/providerops/http/http.go:78`; S14 §7 |

## 3. Proposed read scope vs write scope by role (never equal by assumption)

| Role | Proposed read scope (where a read is proposed) | Proposed write scope (existing capabilities only) | Must not be inferred |
|---|---|---|---|
| Seller | ASSIGNED_CASES for cases/customers; OWN for invoices, attempts, submissions, commission | ASSIGNED_CASES (current holder) for contact outcome; OWN for receipt upload and batch submit | Seeing a case does not prove the seller may record on it (option 2 / 3 of M2 record 2 would let a seller see cases they cannot act on) |
| Supervisor | DIRECT_TEAM (structure, cases, custody, invoices, metrics); OWN commission | HR request create in AUTHORIZED_SUBTREE (implemented as seller descendants); step review as SPECIFIC_APPROVAL_SCOPE | Team read does not grant contact outcome, invoice assist or allocation (OPD-10; no capability) |
| Senior Supervisor | AUTHORIZED_SUBTREE; OWN commission | Same HR request writes | Ready oversight is read only (XR §12 `R baseline`); no direct ready assignment; no personal receiving from rank |
| Sales Manager | AUTHORIZED_SUBTREE ("current territory" in DV); OWN commission | Same HR request writes | No Finance decision; extra-number review has no capability yet |
| Sales Deputy | GOVERNED_TERRITORY (predicate unresolved); OWN commission | HR request create / step / read CONDITIONAL (entry NEEDS VALIDATION) | Never inherits Manager extra-number review or all descendant writes (XR §2, §12) |
| MIS | ORGANIZATION for source / reporting data (scope predicate to confirm, `MISS`); field-minimized workforce data for recipient eligibility | none proposed (no MIS write capability exists) | Read of financial linkage is not a financial decision |
| HR | ORGANIZATION for workforce reads | none proposed now (HR request writes DEFERRED; final apply has no capability) | Workforce read does not include customer, sales-operations or Finance receipt data |
| Finance | ORGANIZATION for financial domain (Finance scope policy undecided, `FRS`); linked item for cases, customers and persons | none proposed (no Finance write capability exists) | Review-queue read does not grant approve / reject / post |

## 4. Ownership and actor roles per action

**Existing capabilities** (actions that can run in the target behind flags, granted to nobody today):

| Action | Record owner (module) | Who may view | Who performs | Who reviews | Who approves | Final execution | Reverse / correct | Audited decisions |
|---|---|---|---|---|---|---|---|---|
| Record contact outcome | cases | case readers in scope | current holder (proposed: Seller) | — | — | the command itself (target-native SHADOW case; write-back by integration) | a later outcome; history append-only; legacy archive refusal → REJECTED + reconciliation | actor, outcome, reason, callback, version, idempotency key, write-back result |
| Create HR request | organization | requester, current reviewer (HR queue not implemented) | Supervisor / Senior / Manager (proposed A), Deputy (C) | current reviewer = actor's current parent | chain of step reviewers; then HR | **none**: HR final apply has no capability (OPD-05, SD-04) | reject by reviewer; no apply to reverse | requester, subject, target, reviewer snapshots, each event |
| Step-review HR request | organization | same | current reviewer snapshot | — | — | none (step or reject only) | — | step decision, reason, next reviewer |
| Stage a receipt upload | payments | uploader | Seller (proposed C) | — | — | binding happens in batch submit | unbound uploads expire (+7 days) | uploader, SHA-256 |
| Submit payment batch | payments | submitter; Finance queue (read) | owning Seller (proposed C) | Finance reviewer (**no capability**) | Finance (**no capability**) | none (legacy decides) | — | batch, items, attempts, receipts, idempotency |
| Assign / execute fulfillment work | fulfillment | service supervisors / executors | service SUPERVISOR / EXECUTOR scope holders (no business role mapped) | `fulfillment.submit_review` BLOCKED (OPD-08) | — | completion only for financial behaviour NONE | cancel / reopen BLOCKED (OPD-04) | events, evidence hashes, outcome |
| Reconcile provider operation | integration | provider-operation readers | any proven user (no role mapped) | — | — | records CONFIRMED / CONFIRMED_ABSENT | append-only events | evidence note, expected version |

**Missing actions** (no capability; listed so ownership is decided before design):

| Action | Record owner | Performer (source) | Independent reviewer / approver (source) | Final executor | Reverse / correct | Key open decisions |
|---|---|---|---|---|---|---|
| Invoice issuance | billing | Seller (XR §12) | — | billing service | Finance cancellation / refund path (separate) | ADR-01, F01 |
| Custody allocation / return | custody | owner of current custody (XR §12) | — | custody service with fresh recheck | append-only correction | OPD-10, F01, OPD-03 |
| Finance stage decision | finance | — | Finance reviewer (FS §8) independent of submitter | finalizer under policy | correction linked to original | OPD-08, D-43, R5-SEC-01 |
| Ledger posting | finance | Posting Actor (G0 §10) | Run Reviewer (G0 §10) | approved engine | reversal linked to original (INV-033) | D-41, OPD-09, OPD-08 |
| Refund | finance (+ after-sales request) | requester (Sales / after-sales) | Finance (OPD-04) | Finance under proof | — | OPD-04 |
| HR final apply | organization | HR (XR §12) | per policy | HR | append-linked correction; no history rewrite | OPD-05, SD-04 |
| Grant change | identity | per D-42 option | per D-42 option (option 2 separates) | identity command | revocation; append-only history (M2 §6) | D-42, E-7, R5-SEC-01 |
| Customer merge | customers | per D-30 option | independent reviewer (D-30 option 3) | customers | reversible merge records | D-30 |

## 5. Hierarchy policy (what the sources already fix)

1. **Hierarchy is an input, never sufficient.** "Approver eligibility always requires a capability; an org relation is only a scope input" (ARCH §17); "Row scope … Hierarchy is an input, never sufficient alone" (ARCH §18); G0 §10; XR §12.
2. **No blanket inheritance.** A higher tier inherits neither Finance, HR or MIS authority nor lower-tier writes (XR §1, §2). Ready oversight (Senior) and Manager-only extra-number review do not extend by Deputy rank (XR §12). The target has no rank shortcut (lint `s11-rank-shortcut`, OPD-10 row in R5).
3. **Direct vs indirect.** Assignment writes target direct eligible recipients; indirect users are aggregate / drill-down only within an authorized scope (G0 §19 "Direct vs indirect users"). Indirect write expansion is OPD-10.
4. **Ambiguity fails closed.** Exactly one current parent per step; zero or several → BLOCKED (S11).
5. **Snapshots, not re-derivation.** Requester, subject parent, target and current reviewer are captured at creation and never re-derived (S11). A hierarchy change does not silently rewrite a request (XR §11).
6. **Current hierarchy never rewrites history.** HR hierarchy edits may change current access / scope; case custody and credit do not move automatically (XR §5). Historical missing ownership = UNKNOWN / NOT RECOVERABLE.
7. **Organizational transfer (proposal).** After a transfer, row scopes computed from current hierarchy follow the new position from the effective time; records already attributed (event actor, credit owner, original owner) are unchanged. Access timing at termination / transfer is OPD-05 and is **not decided here**.

## 6. Ownership facets that must never collapse

| Facet | Meaning (XR §5) | Data fact in target | Who reads it (proposal) | Who may change it |
|---|---|---|---|---|
| Current Custody | who holds the case now | `custody.assignment` derived holder | holder; scoped sales tiers; MIS governed; Finance / HR linked or impact only | custody commands (none exist) |
| Original Owner / Original Seller | first source owner / first seller (distinct fields) | case lineage / attribution snapshot | all with case read in scope | nobody (immutable history) |
| Next Actor | who must act next in a workflow | workflow state per domain | actors of that workflow | the domain command |
| Event Actor | who did a recorded action | event actor column | auditors and readers in scope | nobody after recording |
| Credit Owner | who earns credit | commission credits | the owner (`commission.own.read`); Finance org-wide | Finance correction / reversal only, under OPD-09 |
| Request Owner / Reviewer | requester vs current reviewer vs final applier | HR request snapshots | requester, reviewer, HR | reviewer step; HR apply (no capability) |

Using any facet other than current custody for **visibility** (for example, letting the original seller keep reading a transferred case) needs its own decision (M2 §4-2).

## 7. Open scope questions (no decision ID exists; see the decision package)

| Gap | Question | Why it blocks |
|---|---|---|
| GAP-02 | Read scope for Supervisor, Senior, Manager, Deputy (direct team, subtree, territory) — M2 record 2 covers the seller only | 46 matrix cells carry `RS` |
| GAP-03 | Finance row scope for billing / payment reads | registry cites decisions that do not answer it (CF-04) |
| GAP-04 | Who may read customer identity, and with which fields | registry ties it to D-30 (CF-05) |
| GAP-07 | MIS row / source scope predicate | MS §14 names a scope but no predicate |
| GAP-08 | Definition of GOVERNED_TERRITORY | DP §15 forbids an entire-org assumption but gives no predicate |
