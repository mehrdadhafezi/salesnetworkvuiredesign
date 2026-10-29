# CRM-STANDALONE-TARGET-ARCHITECTURE-V1

Date: 2026-10-29 · Type: architecture freeze record · NO CODE · NO SCHEMA · NO MIGRATION · NO RUNTIME CHANGE

## 0. Status and evidence labels

| Item | Value |
|---|---|
| Repository | mehrdadhafezi/salesnetworkvuiredesign |
| Base main | add2a36e9e122fac0abc8e9b7302287ca7430eec |
| Governance baseline | E0 FROZEN — SLICE 0 APPROVED (`docs/implementation/E0-governance-baseline-freeze.md`) |
| Official legacy code baseline | Sales Network 2.0.160 (`plugin/current/`) |
| Development behavioural overlay | 2.0.182-inline-batch-incomplete (SN-209), behavioural evidence only |
| SN-209 verdict | KEEP 2.0.160 AS OFFICIAL EXECUTABLE LEGACY BASELINE + CREATE A SELECTIVE 2.0.182 BEHAVIOURAL OVERLAY |
| This record | Freezes the standalone target architecture V1, including the "final corrections before Owner freeze" |
| Implementation authority | Unchanged: only Slice 0 is approved (E0 §11–12). This record authorizes no other slice. |
| Owner decisions | None resolved by this record (§32) |
| Production / staging / DB touched | NO |
| HR / Finance runtime | NOT LIVE VERIFIED (unchanged) |

Evidence labels used throughout:

| Label | Meaning |
|---|---|
| **VLF** | VERIFIED LEGACY FACT — read from 2.0.160 code or from the SN-209 audit of 2.0.182 |
| **FPR** | FROZEN PRODUCT RULE — Gate-0 (`GATE-0-PRODUCT-INVARIANTS.md`), `CRM-CROSS-ROLE-ARCHITECTURE-V1.md`, AGENTS.md global contract, frozen role specs, SN-207 design |
| **ABO** | APPROVED BEHAVIOURAL OVERLAY — the SN-209 "adopt as target evidence" list |
| **TAD** | TARGET ARCHITECTURE DECISION. **TAD (E0)** = frozen earlier by E0/D-01/D-02. **TAD** without suffix = frozen by this record (former V1 proposals, approved for freeze). |
| **ODR** | OWNER DECISION REQUIRED — open; never resolved implicitly. Safe defaults below are defaults, not decisions. |

Legacy behaviour is never silently transformed into a target requirement. Where a legacy
behaviour informs the target, the label says which way it was used.

---

## 1. Executive architecture summary

- **TAD (E0)** Standalone CRM; Go modular monolith; PostgreSQL; React + TypeScript + Vite SPA;
  OpenAPI contract with a generated typed client; WordPress/WooCommerce only through a connector
  boundary; transactional outbox/inbox; append-only audit; strangler migration; no big-bang (D-02).
- **TAD (E0)** Slice 0 kernel contracts: seven-outcome Result, NativeRef/SourceRef/CaseRef/LegacyRef,
  reason codes `foundation-v1`, Money `{int64, IRR|IRT}`, UTC + SourceTime, import lint (E0 §9).
- **VLF** 2.0.160 declares zero foreign keys across ~90 `sn_*` tables, contains 312
  `dbDelta`/`CREATE TABLE`/`ALTER TABLE` call sites (largely runtime-reachable), ~414 `DATETIME`
  declarations populated via `current_time('mysql')`, ~20 WordPress roles (three overlapping finance
  roles: `sn_finance`, `sn_financial`, `sn_financial_approval`), and a Biavin `charge_wallet` path.
- **TAD** 21 bounded business modules plus two persisted platform capabilities (workflow, approvals) (§5).
- **TAD** Final architecture rules (normative):
  1. **One aggregate → one write owner.**
  2. **One command → one primary write owner.** A command's DB transaction writes only its owning
     module's aggregates, its own outbox/inbox/idempotency rows and its own state-history rows.
  3. **No cross-module writes by default.** Synchronous cross-module calls are limited to reads,
     invariant checks, authorization and immutable-reference validation.
  4. **Cross-module effects go through the transactional outbox**: owner commits → event → consumer
     runs its own command in its own transaction. A multi-module transaction requires a specific ADR.
  5. **PostgreSQL module ownership**: one schema per module; write grants on own schema only.
  6. **Append-only** finance ledger, custody events, audit events, attribution snapshots,
     approval decisions and state history; corrections are compensating entries.
  7. **Phone is never canonical identity**, never a financial join and never merge proof.
  8. **Read paths never write business state.**
  9. **React never owns authoritative business logic.**
  10. **Moving legacy is absorbed through controlled Legacy Change Intake**, with **immediate
      high-impact Delta Audit triggers** and **selective behavioural overlays**.
  11. **Legacy never automatically overrides the frozen target architecture.**
- **TAD** Code-defined state machines for invoice, payment stage, payment attempt, financial review,
  ledger, custody, commission run, customer merge, HR request base lifecycle and session.
  Versioned configuration (data, not a language) only where variation is evidenced: financing-product
  workflows and evidence, Biawin service definitions, approval policies, commission policies.
- **TAD** No generic BPM engine, no rules language, no microservices, no event sourcing beyond
  custody, ledger, audit and state history.

---

## 2. Baseline and moving-legacy policy

| Tier | Item | Authority |
|---|---|---|
| A | **Official legacy code baseline**: 2.0.160 in `plugin/current` | FPR (E0 §4B). Canonical executable *evidence*; never the target design. |
| B | **Approved development overlay**: 2.0.182-inline-batch-incomplete | ABO. Behavioural evidence only; never code to port; scope = SN-209 adopt list (§2.1). |
| C | **Future legacy versions** (> 2.0.182) | No authority until classified by a Legacy Delta Audit (§3). Never automatically baseline. |

**TAD**
1. `plugin/current` changes only through a dedicated `sync/*` PR following a Delta Audit (E0 rule 8).
2. The target architecture is stable and independent of legacy release velocity; it never follows the
   plugin line-by-line.
3. Undocumented legacy builds (2.0.177–2.0.182) are EXPERIMENTAL / INCOMPLETE.

### 2.1 SN-209 overlay register

**ABO — adopt as target evidence:** transactional batch-payment pattern; validate the complete batch
before file movement; one transaction for multi-row payment submission; row-level idempotency,
improved with persisted batch identity; locking before financial mutation; next-stage gating;
server-authoritative `allowed_actions`; SMS timeout = UNKNOWN, not FAILED; no blind retry after an
unknown external outcome; SMS failure after DB commit never rolls back committed financial state;
optimistic-concurrency / revision-hash pattern; server-side HR filtering and sorting; allowlisted
SQL/filter expressions; InnoDB/atomic-write intent; customer self-service payment capability; inline
next-stage action concept; upgrade-import dry-run + guarded rollback concept; hierarchy-aware action
visibility as legacy evidence only.

**ODR (from SN-209):** see §32 D-28…D-42.

**DO NOT PORT (from SN-209):** see §26.

---

## 3. Legacy change intake process

**TAD** Flow:

```
Legacy Version N → Capture → Structural Diff → Behavioural Delta Audit
  → Classification → Architecture Impact Review → (Owner Decision if required)
  → Overlay Register / Backlog / ADR / Migration-Mapping update → (Optional Rebaseline)
```

Classification set: NO ARCHITECTURE IMPACT · TARGET BEHAVIOURAL OVERLAY · TARGET DOMAIN IMPACT ·
TARGET DATA-MODEL IMPACT · TARGET PERMISSION IMPACT · TARGET WORKFLOW IMPACT · LEGACY-ONLY FIX ·
EXPERIMENTAL / INCOMPLETE · DO NOT PORT · OWNER DECISION REQUIRED. One item may carry several classes.

| # | Step | Mechanism |
|---|---|---|
| 1 | Detection | Owner supplies a release ZIP or names a release. Intake record captures ZIP SHA-256, header version, `SN_VERSION`, `SN_BUILD`, changelog presence. No polling of production. |
| 2 | Comparison | Extract outside the repo; diff against tier A with line-ending normalisation (SN-209: 8 of 40 "modified" files were CRLF/LF only). Output added/modified/deleted with substantive line counts. |
| 3 | Behaviour extraction | Per substantive hunk: AJAX/admin-post hooks, status literals, permission predicates, product types, financial formulas, external calls. Stable IDs `LD-<version>-<n>`. |
| 4 | DB/schema | Grep `dbDelta`, `CREATE`, `ALTER`, `DROP`, new `sn_*` names, option keys. Classify: versioned migration / runtime DDL / read-path DDL / external-migration dependency. Runtime or read-path DDL is DO NOT PORT; its data shape feeds migration mapping (§25). |
| 5 | Permissions | Changed `current_user_can`, roles, caps, `nopriv` hooks, hierarchy predicates mapped to Route → Module → Action → Row/Field. New `nopriv` write or hierarchy-derived grant → TARGET PERMISSION IMPACT + ODR. |
| 6 | Workflow/state | New/removed status literals and guards compared with `STATUS-TRANSITIONS.fa.md` and Gate-0 §8. A removed business rule → ODR. |
| 7 | Target impact | Classification above. |
| 8 | Architecture change | Only when an item is DOMAIN / DATA-MODEL / PERMISSION / WORKFLOW impact, owner-approved where required, and not absorbable by existing configuration points (§27). Then an ADR. |
| 9 | Backlog only | Absorbable by configuration (product, service definition, approval policy, commission policy version) or UI/UX reference. |
| 10 | Rejection | DO NOT PORT, LEGACY-ONLY FIX, or conflict with an FPR/TAD without owner override. Rejection recorded so it is not re-audited. |

Artifacts per version: `docs/legacy-intake/<version>/` (intake record, delta audit, classification
table, owner-decision requests) plus Overlay Register and Migration Mapping updates.

### 3.1 Immediate high-impact Delta Audit triggers

**TAD** Release intake remains event-driven and rebaseline remains periodic (§29). In addition,
regardless of version count or elapsed time, an **immediate Delta Audit** is required when a legacy
release changes any of:

- database / schema or migration behaviour
- customer identity
- a financial formula
- invoice / payment / ledger semantics
- a permission / role / capability
- a workflow status or transition
- a commission formula
- the Biawin / service lifecycle
- an external integration contract
- historical attribution / ownership
- data export / PII behaviour

Detection: the step 3–6 scans run on every captured release; any hit marks it HIGH-IMPACT.
A HIGH-IMPACT release **never becomes baseline automatically**; it only triggers immediate intake,
classification and, where required, owner-decision requests. If it touches a module under
construction, §29 "feature during construction" applies immediately.

---

## 4. Complete capability map

**FPR** Legacy roles are actors, not modules (Position ≠ Role ≠ Capability).

### 4A. Primary operational roles → capabilities used

| Role | Capabilities (write / read) | Notes |
|---|---|---|
| Seller | cases (contact/outcome), sales (pre-invoice intent), billing (issue/evidence under guard), payments (submit), commission (own earnings, read) | FPR Cross-Role §2; own scope |
| Supervisor | custody (owned allocation, ready assignment), cases, HR requests (create/step), billing assist (conditional) | |
| Senior Supervisor | custody (owned distribution), HR requests (step), reporting (scoped) | FPR: no direct ready-assignment baseline |
| Sales Manager | custody, extra-number review, HR requests, archive explorer (read) | |
| Sales Deputy | custody (conditional), HR requests (current reviewer), reporting (territory) | FPR: never inherits Manager review |
| MIS | mis (ingest/lineage), custody (governed source assignment), reporting | FPR: no financial truth |
| HR | organization (all), identity credential policy (restricted) | No ledger; no sales capabilities by default (D-42) |
| Finance | payments (review), finance (ledger/posting), commission (run approval), billing (financial metadata) | VLF: three legacy finance roles collapse into one role with distinct capabilities |
| Invoice Reviewer | reporting (Invoice Review query surface) | FPR D-23/D-24 |

### 4B. Capabilities

Format: Responsibility · Objects / aggregates · Commands · States · Invariants · Legacy sources (VLF) ·
Defects · Future change · Classification.

1. **Customer Profile** — canonical customer + contact identity · Customer ⊃ ContactPoint,
   CustomerIdentifier, IdentityEvidence, MergeCase · RegisterCustomer, AddContactPoint, RecordEvidence,
   ProposeMerge, ResolveMerge · RESOLVED/RELATED/UNKNOWN/CONFLICT/NOT_RECOVERABLE · FPR phone never identity;
   merge evidence-based and reversible · invoice `customer_phone`, Woo customers, 2.0.182 ID registries ·
   phone grouping · new identifiers · **SPLIT + ADAPT**.
2. **Leads / Cases** — sales work unit + lineage · Lead, Case ⊃ CaseSource, CaseAlias, AttributionSnapshot,
   CaseFinancialClaim · CreateCase, RecordContactOutcome, MarkReady, LinkAlias, ClaimCaseForInvoice ·
   case state, sales-workflow state · FPR INV-001…004 · `sn_leads`, `sn_lead_statuses`,
   `sn_lead_status_history`, `sn_seller_flow_states/_events`, `sn_dot_cases` · lead/V4 duality · new
   sources · **ADAPT**.
3. **Custody / Distribution** — append-only holding record · Assignment, CustodyEvent, Transfer, Release,
   DistributionPlan · Allocate, Return, Transfer, Release · FPR INV-006…010, four owners never collapse ·
   `sn_distribution_items/_logs/_chains/_chain_steps`, `sn_mis_distribution_*`, `sn_hr_assignments` (lead side) ·
   mutable history, F01 · **ADAPT**.
4. **Sales** — pre-invoice intent, extra-number, campaign · SalesIntent, ExtraNumberRequest, Campaign ·
   `sn_extra_number_requests`, seller-flow, `sn_campaign*`, `sn_dot_marketing_*` · SN-209 W7 phone guard
   DO NOT PORT · **KEEP / ADAPT**.
5. **Converter / Dot** — conversion case + conversion payments · `sn_dot_cases`, `_payments`,
   `_invoice_links`, `_case_logs`, `_operational_archives` · Subscription+ direct Dot completion (SN-209 W4) ·
   **MERGE into cases (workflow type) + billing**; partial-payment policy D-28.
6. **Catalog** — Product, ProductVersion, Service, ServiceDefinition, Provider, CatalogPolicy ·
   invoices snapshot ProductVersion; live title for display only (E0 §10) · Woo products + `_sn_*` meta,
   `class-sn-sales-catalog`, `sn_project_card_upgrade_rules` · very high change · **SPLIT**; `all_except` D-39.
7. **Billing** — Invoice ⊃ InvoiceItem, PaymentStage · IssueInvoice, IssueNextStage, CancelInvoice ·
   separate invoice and stage machines · FPR INV-011…013 · `sn_invoices`, `sn_invoice_items`,
   `sn_invoice_payment_stages` · overloaded status · **ADAPT + SPLIT state**.
8. **Public Invoice / Payment** — customer self-service · token-scoped, rate-limited (ABO) ·
   `nopriv_sn_submit_manual_payment`, `nopriv_sn_submit_payment_batch` · auto-login D-26 · **ADAPT**.
9. **Payments** — PaymentBatch, PaymentAttempt, UploadObject, Receipt, GatewayAttempt, CollectionAccount,
   ReconciliationCase · legacy values `approved`, `reconcile_error` preserved (TAD (E0)) · `sn_payments`,
   payment-transactions, gateway callbacks · D-25 · **ADAPT**.
10. **Finance** — FinancialReview, LedgerAccount, LedgerEntry, Posting · FPR INV-014/015/031 · `sn_wallets`,
    `sn_wallet_transactions`, `sn_commission_wallet_posting_logs`, `sn_operations_wallet_codes` · wallet
    mutation, F06, F07 · **ADAPT (ledger replaces wallets)**.
11. **Commission** — CommissionPolicy, Eligibility, Calculation, Run, PostingRequest · `sn_commission_rules`,
    `sn_commission_dry_runs/_items`, `sn_payment_commission_matrix`, `sn_project_commissions`,
    `sn_project_commission_rules`, `class-sn-purpose-commission` · dual engines (OPD-09), formula in payment
    code · **SPLIT policy from engine**; D-36.
12. **HR / Organization** — Person, Employment, Position, Level, OrgUnit, OrgAssignment, Compensation,
    Delegation, HRRequest · `sn_hr_profiles`, `_positions`, `_levels`, `_assignments`, `_assignment_history`,
    `_units/_teams/_departments`, `_compensation_*`, `_change_requests`, `_profile_logs`,
    `_position_role_mappings` · user/person/employment mixed; position↔role dual mapping; hierarchy
    duplicated · **MERGE organization+hr; SPLIT lifecycle entities**.
13. **Authentication / Login** — User, Credential, Session, Role, Capability, RoleGrant · `wp_users/usermeta`,
    ~20 `add_role`, unified login, OTP · role/position collapse · **ADAPT**; D-34.
14. **MIS** — ImportBatch, SourceRow, LeadPoolEntry, DistributionPlan · `sn_mis_import_batches`,
    `sn_mis_data_rows`, `sn_mis_lead_pool`, `sn_mis_distribution_*` · MIS report module missing (E0 §10);
    read-path DDL · **SPLIT** (ingest = write module; reports = reporting).
15. **Reports** — read models only · `includes/reports`, `sn_report_invoice_state_events` · **ADAPT to projections**.
16. **Invoice Reviewer** — query surface under reporting · D-23/D-24 · **KEEP as query surface**.
17. **After Sales** — AfterSalesCase with refund/barter/other outcomes (FPR §7) · `sn_after_sales_cases/_logs` ·
    refund execution in finance (OPD-04) · **KEEP**.
18. **Operations Sales** — upgrade opportunity on memberships · `sn_project_operations*`,
    `class-sn-operations-flow` · **MERGE into cases + fulfillment**; D-33.
19. **Execution** — FulfillmentCase, FulfillmentAssignment, ServiceEvidence, ServiceOutcome ·
    `sn_operations_execution_*`, `sn_operations_form_*` · **MERGE into fulfillment**.
20. **Projects / Memberships** — Subscription, SubscriptionItem · `sn_project_memberships`,
    `_membership_items`, `sn_project_subscriptions/_contents`, `sn_project_actions/_events/_notes/_messages/
    _message_reads/_invoice_links/_api_configs` · **SPLIT** (subscriptions → biawin/fulfillment; messaging → cases).
21. **Shipping** — FulfillmentCase of service kind `shipping` · `sn_shipping_orders/_logs`, product-flow,
    bulk order export · **MERGE into fulfillment**.
22. **Biawin** — external fulfilment/subscription provider · `class-sn-biavin-api` (`products`,
    `create_order`, `charge_wallet`) · **ADAPT behind provider port**; D-38.
23. **Customer Portal** — BFF over customers, billing, payments, fulfillment; owns nothing ·
    `class-sn-customer-portal` (`pending_payment`/`expired`/`failed`) · **ADAPT**.
24. **Campaign** — sales `marketing` sub-package · **KEEP** (low priority).
25. **Payment reward / wheel** — `sn_payment_reward_claims`, `sn_invoice_wheel` · **ODR D-40**.
26. **Administration** — flags, configuration, diagnostics · restricted technical boundary (FPR §18) · **ADAPT**.
27. **Activity logs** — `sn_activity_logs`, `*_logs` · **MERGE into audit**.

---

## 5. Target bounded modules

**TAD**

| Module | Purpose | Write authority | Allowed deps (published `api` only) | Forbidden | External ports |
|---|---|---|---|---|---|
| kernel | values only (TAD (E0)) | none | stdlib | everything else | none |
| identity | users, credentials, sessions, roles, capabilities | User, Credential, Session, RoleGrant | kernel | org internals; phone as key | OTP/SMS |
| organization | people, employment, positions, org units, assignments, compensation, delegation, HR requests | Person, Employment, Position, OrgUnit, OrgAssignment, Compensation, Delegation, HRRequest | kernel, identity (ref) | sales/billing tables; acting as permission engine | none |
| authorization | policy evaluation route→module→action→row→field | PolicyBinding (config) | kernel, identity, organization (read) | owning business objects | none |
| customers | identity and contact | Customer, IdentityEvidence, MergeCase | kernel | phone-as-identity joins | none |
| cases | leads, cases, aliases, attribution snapshots, claims, workflow-typed cases | Lead, Case, AttributionSnapshot, CaseFinancialClaim | kernel, customers, organization, catalog (read) | financial status writes | none |
| custody | assignment / transfer / release | Assignment, CustodyEvent | kernel, cases, organization (read/check) | rewriting events | none |
| sales | pre-invoice intent, extra-number, campaign | SalesIntent, ExtraNumberRequest, Campaign | kernel, cases, custody, catalog (read) | invoice state writes | none |
| catalog | product, version, provider, service definition, price, sales policy | Product, ProductVersion, Provider, ServiceDefinition, CatalogPolicy | kernel | invoice tables | Woo product connector |
| billing | invoices and stages | Invoice | kernel, customers, cases, catalog, payments (read/check) | payment verification, ledger | none |
| payments | batches, attempts, evidence, gateway, reconciliation | PaymentBatch, PaymentAttempt, UploadObject, Receipt, GatewayAttempt, CollectionAccount, ReconciliationCase | kernel, billing (read/check) | ledger writes, session issuance, billing writes | gateway, object storage |
| finance | financial review, ledger, postings | FinancialReview, LedgerEntry, LedgerAccount, Posting | kernel, billing, payments (read/check) | mutating posted entries; billing writes | bank import (future) |
| commission | policy, eligibility, calculation, run, posting request | CommissionPolicy, CommissionRun, CommissionCalculation | kernel, organization, billing, payments (read) | ledger writes | none |
| products-finance | financing products, applications, eligibility, evidence | FinancialProduct, Application | kernel, customers, catalog (read) | bank-specific columns | provider ports |
| fulfillment | operational execution (execution, shipping, upgrade execution) | FulfillmentCase, FulfillmentAssignment, ServiceEvidence | kernel, catalog, cases (read) | invoice / wallet writes | shipping provider |
| biawin | subscriptions and Biawin service delivery | Subscription, SubscriptionItem | kernel, catalog, fulfillment (read) | direct wallet / ledger mutation | Biawin API |
| aftersales | complaints and outcomes | AfterSalesCase | kernel, customers, billing (read) | invoice / payment writes | none |
| mis | ingest, lineage, source quality, distribution plans | ImportBatch, SourceRow, DistributionPlan | kernel, cases, custody (read) | financial truth | file ingest |
| reporting | projections; Invoice Review query | projections only | events; owner query APIs | any owner write; querying module schemas | none |
| integration | connector, staging, legacy refs, sync cursors | LegacyReference, staging, SyncCursor | kernel | exposing `wp_`/`sn_` to domains | WP DB snapshot / Woo REST, SMS, gateways |
| audit | append-only audit | AuditEvent | kernel | update / delete | none |
| platform/workflow | workflow execution metadata | WorkflowDefinition, WorkflowVersion, WorkflowStep, Transition, WorkflowInstance | kernel | domain tables; importing business modules | none |
| platform/approvals | approval lifecycle | ApprovalPolicy, ApprovalRequest, ApprovalDecision | kernel | domain tables; importing business modules | none |

**TAD Platform capabilities with persisted state.** `internal/platform/workflow` and
`internal/platform/approvals` are platform capabilities, not business modules, with explicit ownership:
- workflow owns only WorkflowDefinition / Version / Step / Transition and WorkflowInstance execution
  metadata (current step key, step history, timers);
- approvals owns only ApprovalPolicy / ApprovalRequest / ApprovalDecision;
- the business domain owns the business subject and its final state;
- workflow and approvals never write domain tables; business modules never query workflow/approvals
  tables; interaction is through published `api` packages and events only.

**TAD** Outbox, inbox and idempotency are per-module tables inside each module's own schema (structure
from a platform library); writing them is never a cross-module write. `notifications` is an outbox
consumer owning only its delivery records.

---

## 6. Aggregate ownership matrix

**TAD Rules**

1. **One aggregate → one write owner** (TAD (E0) #8).
2. **One command → one primary write owner.** A command's DB transaction writes only that module's
   aggregates, its own outbox/inbox/idempotency rows and its own state-history rows.
3. **Allowed synchronous cross-module calls** (read-only, before or inside the transaction):
   (a) reads via a published query API; (b) invariant checks; (c) authorization evaluation;
   (d) immutable-reference validation. None may lock or write another module's rows; `SELECT … FOR
   UPDATE` is valid only on the caller's own rows.
4. **Default cross-module effect:** owner commits → outbox event (same transaction) → consumer runs its
   own idempotent command in its own transaction, re-checking its own invariants. If it refuses, it emits
   a rejection event and opens a reconciliation item; it never silently drops the event.
5. **Exception:** a transaction writing aggregates of more than one module requires a specific ADR naming
   the aggregates, the business invariant that cannot tolerate an intermediate state, why a reservation
   or saga is insufficient, the lock order and the test plan. **No such ADR is approved.**
6. Asynchronous chains expose honest intermediate states (e.g. `review_approved / stage_transition_pending`),
   never presented as complete.

| Aggregate | Owner | Readers (sync read API) | Cross-module change path (async unless stated) |
|---|---|---|---|
| User / Credential / Session / RoleGrant | identity | authorization, organization | identity commands only |
| Person / Employment / OrgAssignment / Compensation / Delegation / HRRequest | organization | authorization, custody, commission, reporting | `ApprovalDecided` → organization `ApplyHRRequest` |
| Customer / IdentityEvidence / MergeCase | customers | cases, billing, portal | customers commands; billing validates customer ref (rule 3d) |
| Case / Lead / AttributionSnapshot / CaseFinancialClaim | cases | custody, sales, billing | `InvoiceIssued` → cases links invoice and confirms claim (ADR-01) |
| Assignment / CustodyEvent | custody | cases, reporting | custody commands; return checks claims (rule 3b) |
| Invoice (+ items, stages) | billing | payments, finance, commission, portal | `StageReviewApproved/Rejected` → billing stage transition → `StageApproved`, `InvoiceFullyPaid` |
| PaymentBatch / PaymentAttempt / UploadObject / Receipt / GatewayAttempt / CollectionAccount / ReconciliationCase | payments | finance, billing | `PaymentAttemptSubmitted` → finance opens review |
| FinancialReview / LedgerEntry / LedgerAccount / Posting | finance | commission, biawin, reporting | review + posting in one finance transaction (same module); posting requests arrive as events |
| CommissionRun / Calculation / Policy | commission | finance, reporting | `CommissionPostingRequested` → finance → `PostingCompleted/Rejected` → commission |
| Subscription / SubscriptionItem | biawin | fulfillment, portal | `InvoiceFullyPaid` → biawin activates; `FulfillmentOutcomeRecorded` → biawin updates item |
| FulfillmentCase | fulfillment | biawin, reporting | `SubscriptionItemActivated` → fulfillment opens case |
| AfterSalesCase | aftersales | finance, reporting | `RefundRequested` → finance |
| ImportBatch / SourceRow / DistributionPlan | mis | custody, reporting | `SourceDelivered` → custody allocates |
| WorkflowInstance | platform/workflow | owning domain (API) | domain events → workflow advances (§11.1) |
| ApprovalRequest / ApprovalDecision | platform/approvals | owning domain (API) | `ApprovalRequested` → approvals; `ApprovalDecided` → domain applies |
| LegacyReference / SyncCursor / staging | integration | — | module `Imported` event → integration records LegacyReference (§24) |
| AuditEvent | audit | authorized readers | every module outbox → audit append |

**ADR-01 candidate — Invoice issuance vs case return mutual safety (FPR Gate-0 INV-009).**
Candidate protocol (reservation, no cross-module transaction): billing `IssueInvoice` requires a
`claim_id` from `cases.ClaimCaseForInvoice` (own cases transaction, serialised against custody
`ReturnCase` via active-claim check (rule 3b) and case version); billing validates the claim
synchronously and commits; `InvoiceIssued` confirms the claim; an unconfirmed claim expires after a
short TTL into reconciliation, never a silent release (FPR INV-007). **Status: ADR-01 CANDIDATE —
not accepted by this record.** Safe default until accepted: no target invoice-issue or return writes
(gates Slice 11).

---

## 7. Canonical target domain model

**TAD** (entities → key attributes; *value objects in italics*)

- **Identity:** User (`user_id`, status); Credential (kind password/otp_phone, secret hash, rotated_at);
  Session (issued_at, expires_at, revoked_at, issuance_reason — never a payment callback, D-26).
- **Organization / HR:** Person (legal name, *NationalID?*); Employment (type, status, interval, `user_id?`);
  Position; OrgUnit (tree); OrgAssignment (employment, position, unit, reports_to, valid interval,
  recorded_at); Compensation (policy ref, effective interval); Delegation (delegator, delegate,
  capability set, scope, interval, reason); HRRequest.
- **Customer:** Customer (minted `customer_id`, resolution state); ContactPoint (kind, *NormalizedValue*,
  valid interval, source); CustomerIdentifier (scheme crm_code / national_id / biawin / woo /
  legacy_aggregate_v1 / legacy_map_v2, value, status); IdentityEvidence (claim, source, strength,
  observed_at); MergeCase.
- **Sales:** Lead; Case (`case_id`, workflow_type, *CaseRef*); CaseSource (*SourceRef*);
  AttributionSnapshot (facet original_owner / original_seller / credit_owner, employment ref, org path
  at time, captured_at; immutable); CaseFinancialClaim.
- **Custody:** Assignment; CustodyEvent (append-only).
- **Catalog:** Product (kind goods / service / subscription_bundle / financing); ProductVersion
  (immutable price, components, credit terms, effective interval); Service / ServiceDefinition +
  ServiceVersion; Provider.
- **Billing:** Invoice (customer, case refs, *Money* total, lifecycle state, payment_plan); InvoiceItem
  (product_version, qty, *Money*, line_kind product / credit_option / fee); PaymentStage (stage_no,
  *Money* due, stage state).
- **Payments:** PaymentBatch (*IdempotencyKey*, invoice, submitter, channel, outcome); PaymentBatchItem
  (client-stable `item_key`, *Money*, evidence refs); PaymentAttempt; UploadObject; Receipt;
  GatewayAttempt (*ExternalReference*, outcome); CollectionAccount; ReconciliationCase.
- **Finance:** LedgerAccount; LedgerEntry (balanced lines, *Money*, business_key, reverses_entry_id?);
  FinancialReview; Posting.
- **Commission:** CommissionPolicy (versioned); CommissionCalculation; CommissionRun;
  CommissionPostingRequest.
- **Biawin / fulfillment:** Subscription; SubscriptionItem; FulfillmentCase; FulfillmentAssignment;
  ServiceEvidence; ServiceOutcome.
- **Integration:** LegacyReference; ImportBatch; SyncCursor; InboxEvent; OutboxEvent (per module).
- **Audit:** AuditEvent (actor, subject refs, action, before/after refs, reason, correlation, occurred_at).
- **Platform:** WorkflowDefinition/Version/Step/Transition/Instance; ApprovalPolicy/Request/Decision.

---

## 8. Canonical PostgreSQL model

**TAD** Principles (no DDL in this record):

1. One schema per module plus `platform_workflow`, `platform_approvals`, `integration`, `audit`,
   `reporting`. Each module's DB role has write grants on its own schema only; cross-schema grants are
   limited to `REFERENCES`.
2. Surrogate keys (`bigint` identity or UUIDv7). Legacy native IDs only in `integration.legacy_reference`.
3. Real FKs within a module. Cross-module FKs only to immutable identity rows never deleted
   (customer, invoice, employment primary keys), `ON DELETE RESTRICT`, never cascade. An FK is a
   reference, never a write path.
4. Money: `amount bigint NOT NULL` + `unit` CHECK IN ('IRR','IRT'). No float/numeric money. Absent money
   is NULL, never 0 (TAD (E0)).
5. Time: `timestamptz` UTC + optional `source_zone` + `source_raw` for imported evidence. No Jalali columns (TAD (E0)).
6. State: one column per state machine, CHECK-constrained, with transition-history tables. Legacy raw
   values in provenance (`LegacyValue`), never as the state (TAD (E0) reason codes).
7. Append-only tables (custody events, ledger lines/entries, audit, attribution snapshots, state history,
   approval decisions, receipt history): INSERT only, enforced by grants and a trigger rejecting
   UPDATE/DELETE.
8. Idempotency and outbox/inbox are per-module tables in the module schema; `UNIQUE (scope, key)`.
9. Concurrency: `version bigint` per aggregate root (ABO revision pattern); `FOR UPDATE` only on own rows.
10. Configuration data (workflow, approval, commission policies, service definitions) is versioned and
    immutable once published.
11. Migrations: versioned files run by `crm-migrate` only; zero DDL at runtime.
12. The reporting role reads only `reporting.*`; projections are built from events and owner query APIs,
    never by querying module schemas.

---

## 9. Database defect-remediation matrix

| Legacy defect | Evidence | Target correction | Label |
|---|---|---|---|
| Phone as identity | invoice grouping; 2.0.182 registries | Customer + ContactPoint + IdentityEvidence + CustomerIdentifier (§10) | VLF → TAD |
| No FK constraints | 0 FK declarations | Real FKs (§8.3) | VLF → TAD |
| Runtime CREATE/ALTER | 312 DDL call sites; `SN_Subscription_Plus::maybe_upgrade` on `init` | Versioned migrations only | VLF → TAD |
| Read-path writes | report-view backfill; Invoice Review role/page creation; HR page on `admin_init`; F07 autopost | Strict CQS; read-only query transactions | VLF → FPR INV-026 |
| Mutable custody history | E0 §10 | Append-only CustodyEvent; Assignment as projection | FPR / TAD |
| Wallet mutation | `sn_wallets`, `sn_wallet_transactions`; Biavin `charge_wallet` | Append-only ledger; derived balances | VLF → TAD (E0) #9 |
| Reversal mutates original | E0 §10; SN-209 `on_reversed` | Compensating entries + state history | TAD (E0) #9 |
| Hierarchy duplicated | `sn_hr_assignments` + distribution chains + scope-service | Single `org_assignment` | VLF → TAD |
| User/Person/Employment mixed | `sn_hr_profiles` keyed on WP user | Separate lifecycles | TAD |
| Local DATETIME | ~414 DATETIME; `current_time('mysql')` | `timestamptz` + SourceTime | TAD (E0) #14 |
| Jalali persistence | `manual_paid_at_jalali`, `deposit_jalali_datetime` | Presentation/evidence only | TAD (E0) |
| Overloaded status | `status`/`invoice_status`/`payment_status`; main-status fallback | Separate state machines | VLF → FPR INV-016 |
| Implicit joins | `invoice.lead_id`, `follow_invoice_id`, phone matches | Explicit refs/FKs; ambiguous → UNKNOWN | FPR §8 |
| Weak idempotency | position-based batch dedupe | First-class idempotency + persisted batch identity | ABO → TAD |
| Historical ownership ambiguity | current hierarchy used for credit | Immutable AttributionSnapshot | FPR INV-029 |
| Product-specific schema | `subscription_plus*`, `product_star`, `_sn_*` meta | ProductVersion components + line_kind | TAD |
| Hard-coded markers | `15475:26204` | Import manifests with checksums | VLF → TAD |
| Evidence referenced before durable storage | receipt URLs + post-commit upload moves (SN-209) | §15 evidence model | VLF → TAD |

---

## 10. Customer identity architecture

**FPR** Phone is never canonical identity, financial join or merge proof. **TAD**:

- **Customer** — `customer_id` minted only by `customers` on registration with evidence or a migration
  RESOLVED outcome.
- **ContactPoint** — many per customer, time-bounded; one phone may belong to different customers at
  different times; unexplained overlap is CONFLICT.
- **CustomerIdentifier** — external/legacy codes (Woo ID, legacy aggregate `a00001`, legacy map v2,
  Biawin ID) stored with scheme and status; never primary keys.
- **IdentityEvidence** — claims with strength strong/medium/weak. Phone equality is weak and can never
  reach RESOLVED alone. An OTP credential proves control of a ContactPoint at a time (medium evidence);
  it never establishes customer equality.
- **MergeCase** — candidates, evidence, decision, actor; reversible (unmerge appends evidence, never deletes).

| State | Meaning | Allowed |
|---|---|---|
| RESOLVED | Strong evidence → one canonical customer | financial joins, portal access |
| RELATED | Proven relationship without equality | linked display; no financial aggregation |
| UNKNOWN | Insufficient evidence | discovery only; separate report counts |
| CONFLICT | Contradictory evidence | fail closed; work queue |
| NOT_RECOVERABLE | Source permanently missing | preserved as such |

Migration rule: each legacy invoice attaches to a Customer only through its own issued-to record;
phone grouping produces RELATED-candidate MergeCases, never automatic merges. Merge authority: D-30.

---

## 11. Workflow architecture

**TAD A. Code-defined state machines:** invoice lifecycle; payment stage; payment attempt / gateway attempt /
reconciliation; financial review; ledger posting; custody; commission run (draft → calculated → approved →
posting_requested → posted / failed / unknown); customer MergeCase; HR request base lifecycle (submitted →
in_review → approved / rejected → applied / failed / unknown); session.

**TAD B. Configurable workflow definitions:** Biawin service fulfilment per ServiceDefinition;
financing-product applications per Provider/FinancialProduct; operations upgrade/execution steps and forms
(VLF `sn_operations_form_definitions`); HR approval routing (not apply semantics); after-sales outcome routing.

| Element | Shape | Not allowed |
|---|---|---|
| WorkflowDefinition / WorkflowVersion | key, owning module, immutable published version | cross-module ownership |
| WorkflowStep | key, kind (task / decision / wait_external / approval), assignee rule (position / capability / org-scope ref), SLA | arbitrary code |
| Transition | from, to, guard = named Go-registered predicate with parameters | expression language |
| Decision | named outcomes per step | scripts |
| EvidenceRequirement | type, required/optional, validator name | — |
| ApprovalPolicy | §17 | — |

TAD (E0) status mapping: legacy raw → target via `foundation-v1`; unknown raw → UNKNOWN with raw preserved.

### 11.1 Ownership and execution

**TAD**
- The domain state machine is authoritative. A workflow definition only routes, orders and parameterises
  transitions the domain already permits; it can never make a transition valid that a domain invariant
  forbids. Code-defined machines (§11 A) cannot be bound to configurable workflows except approval routing.
- Execution: user → domain command (e.g. `fulfillment.CompleteTask`) → domain reads `workflow.api`
  synchronously (transition allowed from step S at version V? required evidence?) → domain commits its own
  state + outbox `TaskCompleted{instance, expected_step S, version V}` → workflow consumes and advances.
  If the instance moved, workflow records CONFLICT and emits `WorkflowConflict` → reconciliation; it never
  rewrites domain state.
- Guards and effects are Go functions registered by the owning domain; workflow executes no domain writes.

---

## 12. Financial-product flexibility

**TAD** Provider (bank/financier); FinancialProduct (terms in ProductVersion); ProductWorkflow (WorkflowVersion
bound to a product version); Application (customer, product version, requested *Money*, state);
EligibilityDecision (inputs snapshot, outcome, reason codes, external ref); Evidence (per EvidenceRequirement);
Approval (via approvals).

Variation lives in workflow versions, evidence requirements, approval policies and provider adapters
(`products-finance/providers/<name>` implementing `ProviderPort`: submit / status / callback). Bank names
appear only in adapters. No bank-specific columns or statuses: provider statuses are stored as
`external_status` + raw value and mapped to generic application states. Money effects of financing go
through finance postings only.

ODR D-37: whether "installment credit" (VLF `_sn_product_credit_amount`) is a financing product or an entitlement.

---

## 13. Biawin / service architecture

**TAD**
- ServiceDefinition / ServiceVersion (catalog): owning team (position/capability), workflow version,
  evidence requirements, SLA, completion outcomes, financial behaviour flag (none / consumes_credit /
  charges_ledger), integration provider.
- Subscription (biawin): customer, source invoice ref, product version, status, interval.
- SubscriptionItem: one per included service instance (1 … 30+ per subscription).
- FulfillmentCase (fulfillment) per SubscriptionItem execution, running that service's workflow version;
  FulfillmentAssignment, ServiceEvidence, ServiceOutcome.
- Invoices never carry service-specific columns.
- Activation, entitlement posting and fulfilment-case creation are separate owner commands triggered by
  events (`InvoiceFullyPaid`, `SubscriptionActivated`, `SubscriptionItemActivated`); never one fan-out transaction.
- Biawin API calls only in `integration` via `BiawinPort`, outbox delivery, inbox for callbacks; unknown
  outcome → reconciliation, no blind retry (ABO).
- Biawin "charge wallet" → finance posting request + external call (ODR D-38).

---

## 14. Subscription+ domain analysis

**VLF (SN-209)** `subscription_plus` product type with base price and installment credit; optional cash-credit
options adding a `subscription_plus_cash` line; activation after full settlement creating a project
membership/item and storing a `cash_wallet` credit in JSON; a no-cash path via a Dot case that may be
completed as `plus_none` without payment; in-place suspension on reversal; pro-rata commission split.

| Candidate meaning | Fit | Reasoning |
|---|---|---|
| Product | Partial | sold via catalog with a price |
| Subscription / Membership | Strong | produces an active entitlement card |
| Product bundle | Medium | base + optional cash option sold together |
| Service bundle | Possible | depends on attached services (unknown) |
| Financial product | Partial | installment credit resembles a credit line |
| Wallet / credit product | Strong (cash option) | pay X, receive spendable credit Y — stored-value liability |

**TAD placement** (business meaning still ODR):
- catalog: `subscription_bundle` Product whose ProductVersion components are `subscription`,
  `installment_credit_grant`, optional `cash_credit_purchase`;
- billing: generic invoice lines (`line_kind` product / credit_option); no Subscription+ columns;
- biawin: Subscription lifecycle (pending_activation → active → suspended → ended), activated from `InvoiceFullyPaid`;
- finance: cash credit as stored-value ledger liability; installment credit as entitlement account; balances
  derived; suspension/reversal by compensating entries;
- commission: component allocation as CommissionPolicy (D-36);
- cases: any conversion step is an ordinary case workflow, never direct Dot completion;
- each effect is a separate event-triggered owner command (§6).

**DO NOT PORT:** JSON cash wallet; runtime-created Subscription+ table; direct Dot case completion; direct
project-membership writes; recursive cross-module commission calls.

**ODR D-35 a–e:** (a) cash credit = liability vs voucher vs merchant credit; (b) installment credit =
financing product?; (c) expiry/spend rules; (d) Biawin as redemption system of record; (e) is conversion
mandatory for the no-cash path.

---

## 15. Payment architecture

**TAD**, incorporating ABO.

### 15.1 Evidence objects (payments-owned)

1. **Upload first.** `POST /uploads` (payments `StageUpload`) streams the file into **durable quarantine
   object storage** (same durability class as evidence storage). The server computes SHA-256, size and
   media type (content sniffing) and persists an **UploadObject** (object_ref, content_hash, size,
   media_type, uploader context, `expires_at`).
2. Upload is idempotent on `(uploader_context, content_hash)`: re-sending the same file returns the
   existing UploadObject and never stores bytes twice.
3. Unbound UploadObjects are not evidence. A payments GC job purges them after `expires_at` only after a
   DB check that they were never bound. Bucket TTL never applies to bound objects.

### 15.2 Batch submission (single owner: payments)

1. The client keeps one **batch idempotency key** per submission intent until a terminal outcome is known;
   each item has a stable client `item_key`. Position is never the dedupe key.
2. `payments.SubmitBatch` validates the whole envelope first (counts, totals, units; referenced
   UploadObjects exist, are unbound or bound to the same intent, hashes match).
3. Invariant checks (synchronous reads, §6 rule 3b): read the target stage from `billing.api` (stage id,
   open state, immutable due amount, stage version); serialise on payments' own per-stage
   **CollectionAccount** row (`FOR UPDATE`, payments schema); available = due − (pending + approved
   attempts) from payments' own data.
4. One payments transaction inserts PaymentBatch + items, PaymentAttempts, one **Receipt** per bound
   UploadObject (state `QUARANTINED`, object_ref, content_hash, size, media_type), the idempotency record,
   and outbox `PaymentAttemptSubmitted` + `ReceiptBound`. Commit.
5. Because bytes are durable before commit, a committed PaymentAttempt always references an existing object.
6. Idempotency: same key + same request hash → stored outcome replayed; same key + different hash → CONFLICT.
7. Outcomes per batch and per item use the kernel Result set: SUCCESS / PARTIAL / FAILED / UNKNOWN /
   CONFLICT / BLOCKED / UNAUTHORIZED.

### 15.3 Receipt states

| State | Meaning |
|---|---|
| `QUARANTINED` | bound; bytes durable in quarantine; not finalised |
| `READY` | finalised into evidence storage; hash verified |
| `STORAGE_ERROR` | finalisation failed after bounded retries; quarantine object retained; reconciliation pending |
| `NOT_RECOVERABLE` | bytes missing or hash mismatch on every copy; ReconciliationCase opened |

Receipt state changes are recorded append-only; content hash is immutable.

### 15.4 Post-commit finalisation

The worker consumes `ReceiptBound`, performs a conditional server-side copy to the content-addressed key
`evidence/<sha256>` (if absent), verifies the hash and sets `READY` with evidence_ref. Idempotent: a re-run
finds the object, verifies the hash and only updates state — no duplicate file on retry. Repeated failure
sets `STORAGE_ERROR`. **The financial transaction is never rolled back because finalisation failed.**

### 15.5 Evidence reconciliation

For `STORAGE_ERROR`: if the evidence key exists with matching hash → `READY`; else retry copy from
quarantine; if the quarantine object is missing or mismatched → `NOT_RECOVERABLE` + ReconciliationCase
(Finance owner). Imported legacy receipts whose files are missing are `NOT_RECOVERABLE` from import.

### 15.6 Review gating

Finance approval of an attempt whose required receipts are not `READY` is **ODR D-43**; safe default: blocked
unless `READY`.

### 15.7 Gateway

`GatewayAttempt` with external reference. Timeout/transport failure = **UNKNOWN**; reconciliation queries the
provider before any retry (ABO, FPR Cross-Role §14). Callbacks via inbox, idempotent on
`(provider, external_ref)`. Callbacks never issue a session (D-26) and never mark Finance-reviewed
(D-25: `gateway_auto` provenance).

### 15.8 Notifications

SMS via outbox after commit. SMS failure or UNKNOWN never rolls back financial state. "Resend link" is a
new message, not a retry (ABO).

### 15.9 Next stage

`billing.IssueNextStage` (billing-only command) requires plan = partial, paid > 0, remaining > 0, active
due = 0, previous stage approved (billing state), no pending attempts (`payments.api`, rule 3b), not
cancelled/archived (ABO). Authority: ODR D-32.

### 15.10 Customer self-service

Token-scoped `public` route group, rate-limited, same commands with `channel=customer` (ABO).

---

## 16. Finance / ledger architecture

**TAD**, on the frozen Money type (TAD (E0)):

- Double-entry journal: LedgerEntry (business_key UNIQUE, source refs, posted_at, effective_at) with
  LedgerLine (account, *Money*, debit/credit); per-unit sum = 0; cross-unit lines rejected.
- Accounts (examples): customer receivable, cash/bank clearing, customer stored-value, customer entitlement
  credit, commission payable, refund payable, revenue per product kind.

| Concept | Representation |
|---|---|
| invoice amount | billing fact; receivable posting basis = ODR D-41 |
| payment amount | PaymentAttempt fact → on approval, cash ↔ receivable posting |
| credit (entitlement) | entitlement account postings |
| wallet-like value | stored-value account postings; balance = sum of lines |
| commission | posting request → payable posting; settlement separate |
| refund | refund request → refund-payable posting → settlement posting |
| settlement | cash-out posting with external proof |

- Balances are derived projections; caches are rebuilt by workers, never by reads.
- Reversal = new entry with `reverses_entry_id`; originals never mutated (TAD (E0) #9).
- `finance.ApproveAttempt` commits the FinancialReview decision and the LedgerEntry in one finance
  transaction (same module), then emits `StageReviewApproved`. Billing consumes and transitions
  stage/invoice in its own transaction (`StageApproved`, `InvoiceFullyPaid`). If billing refuses, it emits
  `StageTransitionRejected`; finance opens reconciliation; the ledger is corrected only by compensating entry.
- No rounding/conversion until OPD-06 (TAD (E0)).

---

## 17. Approval architecture

**TAD**
- ApprovalPolicy (versioned config): subject type (e.g. `finance.stage_review`, `hr.transfer`,
  `commission.run`, `products_finance.application`), ordered ApprovalSteps (eligible approver rule =
  capability + org-scope relation, quorum, amount band, maker≠checker, escalation, delegation allowed).
- ApprovalRequest per subject instance; ApprovalDecision append-only (actor, step, decision, reason, delegation ref).
- Lifecycle: domain commits subject in its own `pending_approval` state and emits `ApprovalRequested`;
  approvals creates the request and records decisions; `ApprovalDecided` → domain re-checks its invariants
  and applies or refuses the final state. Approvals never writes domain tables.
- Domain invariants remain in domain code. No `*_approved_by` columns on business tables.
- Approver eligibility always requires a capability; an org relation is only a scope input (FPR Hierarchy ≠ Permission).
- Delegation uses organization Delegation (time-bound, capability-limited, audited).
- Maker/checker thresholds: OPD-08.

---

## 18. Permission architecture

**FPR** Route → Module → Action → Row → Field; View ≠ Write; Hierarchy ≠ Permission;
Position ≠ Role ≠ Capability ≠ Credential. **TAD** runtime:

- Capability: module-qualified action (e.g. `billing.invoice.issue`, `payments.batch.submit`,
  `finance.stage.approve`, `org.assignment.apply`).
- Role: named capability bundle; RoleGrant (interval, granted_by, reason) owned by identity.
- Position: HR fact. PositionRoleDefault may suggest grants; applying them is an explicit audited identity
  command. Position never grants by itself. HR granting sales capabilities is ODR D-42.
- Row scope: per-module resolver from ownership facets, org-scope relation (organization read API),
  delegation and object state. Hierarchy is an input, never sufficient alone.
- Field scope: per-capability masks in query projection and DTO mapping (e.g. PII, D-23).
- Evaluation in the application layer for every command and query via `authorization.Evaluate`.
- `allowed_actions` hints are computed by the same evaluator (ABO); the UI never decides.
- Legacy mapping: ~20 WP roles → target roles; three finance roles → one finance role with distinct
  capabilities (VLF); `sn_invoice_reviewer` → `invoice_review.read`.
- Sensitive reads: query transactions are read-only (FPR INV-026). **Sensitive exports are commands**
  (`CreateExport`). Ordinary sensitive views emit a read-audit record through the audit API outside the
  query transaction — a bounded technical effect (FPR INV-027). Audit-write failure behaviour: ODR D-45
  (safe default: fail closed for PII-bearing views).

---

## 19. Commission architecture

**TAD**
- CommissionPolicy (versioned): purpose, eligibility rule ref (named predicate + params), rate table, base
  definition, effective interval, attribution facet.
- CommissionEligibility computed per business event using AttributionSnapshot, never current hierarchy
  (FPR INV-029).
- CommissionCalculation: immutable input snapshot + result; business key `(invoice, stage, purpose,
  share_facet, policy_version)` (FPR INV-032).
- CommissionRun: state machine + approvals (FPR INV-031).
- Approved run emits `CommissionPostingRequested` → finance posts in its own transaction →
  `PostingCompleted/Rejected` → commission records the result. Commission never writes ledger tables;
  finance never writes commission tables.
- Component allocation (SN-209 pro-rata) is a policy option, not code; correctness = ODR D-36.
- One engine with migrated policies; historic engine routing = OPD-09.

---

## 20. Reporting / MIS architecture

**TAD**
- Write models stay normalised. Projections (`reporting.*`) built from outbox events and owner query APIs:
  facts (invoice_issued, stage_approved, payment_verified, custody_event, case_state_change,
  commission_calc) and dimensions (employment-at-time, org path-at-time, customer, product version,
  provider, service, region, source, attribution facet).
- Temporal modes (FPR Cross-Role §13, INV-022): Current Snapshot, Event Range, Historical As-Of (only with
  proven history; coverage metadata required).
- Metric contract (FPR): name, grain, source, cohort, time_basis, scope, formula, unit, freshness,
  coverage/version. Missing metadata → NOT FINAL; denied/incomplete → never zero.
- MIS writes only ingest/lineage aggregates; MIS reports are projections.
- Invoice Review: projection + query surface; no export until D-23.
- Exports are commands with audit (§18); views follow the read-audit rule.

---

## 21. Go modular monolith layout

**TAD**

```
cmd/
  crm-api/            HTTP server (OpenAPI-generated server interfaces)
  crm-worker/         outbox dispatch, inbox consumers, reconciliation, projections, evidence finalisation
  crm-migrate/        versioned migrations runner (only DDL path)
  crm-legacy-import/  connector-driven staging import + reconciliation reports
internal/
  kernel/             (Slice 0, TAD (E0)) result, refs, reasoncodes, money, timeutil
  platform/           db (tx manager, per-module schema handles), outbox/inbox/idempotency library,
                      workflow, approvals, notifications, featureflags, clock, httpx, authz glue
  identity/ organization/ authorization/ customers/ cases/ custody/ sales/ catalog/
  billing/ payments/ finance/ commission/ products-finance/ fulfillment/ biawin/
  aftersales/ mis/ reporting/ audit/
  integration/
    legacy/           ONLY place with wp_/sn_ literals (lint-enforced, TAD (E0))
    gateways/ sms/ biawin/ woo/ storage/
api/openapi/          OpenAPI source (per-module files, bundled)
```

| Path | Contents |
|---|---|
| `<mod>/api/` | published query, check and command interfaces (separate), DTOs, event types, ports |
| `<mod>/domain/` | entities, value objects, state machines, invariants (pure; no SQL) |
| `<mod>/app/` | command and query handlers, authorization calls, transaction boundaries, outbox emission |
| `<mod>/app/consumers/` | outbox/inbox consumers; call only their own module's commands |
| `<mod>/store/` | repositories + SQL; private, never imported by other modules (TAD (E0)) |
| `<mod>/http/` | OpenAPI handler adapters → app |
| `<mod>/events/` | versioned event payload schemas |
| `integration/<x>/` | adapters implementing ports declared in owning modules' `api` |

Rules: handlers → app → domain. Other modules' `api` packages may be imported for §6 rule-3 calls only;
lint forbids invoking another module's command interface inside a transaction scope.
`internal/platform/workflow` and `internal/platform/approvals` import kernel only. Interfaces only where a
second implementation exists (tests count); no generic repository framework.

---

## 22. React architecture

**TAD**

```
web/src/
  app/        bootstrap, providers, router, error boundaries, theme (SN-207 tokens)
  routes/     route tree per role entry → module pages (lazy)
  modules/<capability>/  pages/ features/ components/ queries/ commands/ schemas/ permissions/
  shared/     ui/ (SN-207 design system) api/ auth/ forms/ tables/ permissions/
  generated/api/   OpenAPI-generated typed client and types (never edited)
```

- Frozen SN-207 visual and product contracts implemented as-is; legacy UI never copied (FPR).
- Flow: React → generated OpenAPI client → Go API → application layer → domain.
- **React never owns authoritative business logic.** It renders server `allowed_actions` hints and server
  validation; client validation mirrors server schemas for UX only.
- Inline next-stage and inline payment-batch are acceptable concepts (ABO); final UX designed against SN-207.
  The 2.0.182 unified modal is not frozen UX.

---

## 23. API contract model

**TAD**

| Concern | Convention |
|---|---|
| Result envelope | `{ outcome, data?, items?: ItemOutcome[], reasons: ReasonCode[], correlation_id, audit_id? }` (TAD (E0) Result) |
| HTTP mapping | 200 for business outcomes incl. PARTIAL/UNKNOWN; 4xx contract/auth; 409 CONFLICT; 403 UNAUTHORIZED; 5xx server faults; HTTP 2xx is never business SUCCESS by itself |
| Errors | RFC 9457 problem+json with `reason_code` (`foundation-v1`) |
| Pagination | cursor-based; total only when part of the metric contract |
| Filters / sorting | allowlisted keys per endpoint in OpenAPI (ABO); unknown key → 400 |
| Validation | field-level `errors[]` with path + reason code |
| Permission hints | `allowed_actions: { action: { allowed, reason_code? } }` (ABO) |
| Optimistic concurrency | `ETag` = aggregate version; mutations require `If-Match`; mismatch → CONFLICT (ABO) |
| Idempotency | `Idempotency-Key` required on every mutating command; replay returns stored result |
| Correlation / audit | `X-Correlation-ID` propagated; `audit_id` returned for sensitive commands |
| Money | `{ amount: "string-int64", unit: "IRR" \| "IRT" }`; never float |
| Dates / timezone | RFC 3339 UTC; `display_zone` from profile (Asia/Tehran default); Jalali formatting client-side |
| Upload evidence | `POST /uploads` → durable quarantined UploadObject (id, sha256, size, type), idempotent on hash; commands reference UploadObject ids; binding creates `QUARANTINED` Receipts; finalisation async; receipt state exposed on read |
| Bulk | preview (eligible set + version) → apply with preview token; Requested / Applied / Skipped / Rejected / Failed / Unknown (FPR Cross-Role §15) |
| Partial results | per-item outcomes; never collapsed into a single SUCCESS |

---

## 24. Legacy connector boundary

**TAD (E0)** Only `internal/integration/legacy/**` knows `wp_*`, `sn_*` and WooCommerce internals. **TAD**:

1. Connector contract: versioned extract specs per legacy entity (e.g. `legacy.invoice.v1`) with source
   columns, semantics, raw-value preservation and checksums.
2. Extraction from a replica or snapshot export only; never production DB writes; the target runtime
   never queries WordPress live for domain logic.
3. Staging: `integration.stg_*` raw verbatim tables + batch manifests (counts, hashes, extract time,
   source version).
4. Transform: import calls each module's import command; the module stores SourceRef provenance in its
   own tables and emits `Imported{native_ref, legacy_key}`; integration records LegacyReference in its own
   transaction. The legacy key is the module command's idempotency key, so a crash between steps is
   safely re-runnable. Import never writes two modules in one transaction.
5. Reconciliation: per-entity row/amount parity reports with reasons (FPR INV-036).
6. Strangler write-back (later slices): target → legacy via WP/Woo REST adapter or a narrow plugin
   endpoint through the outbox; never direct SQL writes into WordPress.

---

## 25. Migration mapping model

**TAD** Register format: Legacy source → target aggregate → target table → class → transformation →
reconciliation. Classes: DIRECT · TRANSFORM · SPLIT · MERGE · DERIVED · AMBIGUOUS · UNRECOVERABLE.
Same phone is never treated as same customer without evidence.

| Legacy source (VLF) | Target aggregate | Target table (logical) | Class | Transformation | Reconciliation |
|---|---|---|---|---|---|
| `wp_users`/usermeta | User, Credential | identity.user / credential | TRANSFORM | hashes kept or rehashed on login; roles → RoleGrant | user counts; grant diff |
| WP roles/caps + `sn_hr_position_role_mappings` | Role, RoleGrant, PositionRoleDefault | identity.role_grant | TRANSFORM | finance roles → capabilities | effective-capability diff |
| `sn_hr_profiles` | Person + Employment | org.person / employment | SPLIT | profile → person; employment fields → interval | 1:1; orphan users flagged |
| `sn_hr_positions` / `_levels` | Position, Level | org.position | DIRECT | — | counts |
| `sn_hr_departments` / `_units` / `_teams` | OrgUnit | org.org_unit | MERGE | three tables → one tree | tree parity |
| `sn_hr_assignments` + `_assignment_history` | OrgAssignment | org.org_assignment | TRANSFORM | intervals; missing history → UNKNOWN | overlap/gap report |
| `sn_distribution_chains` / `_chain_steps` | — | none (hierarchy from org) | DERIVED | validated against org | chain vs org conflicts |
| `sn_hr_compensation_*` | Compensation | org.compensation | TRANSFORM | effective periods | sums per period |
| `sn_hr_change_requests`, `_profile_logs` | HRRequest, AuditEvent | org.hr_request; audit | TRANSFORM | — | counts by state |
| invoice customer fields, Woo customers | Customer, ContactPoint, IdentityEvidence | customers.* | SPLIT + AMBIGUOUS | issued-to evidence per invoice; phone grouping → MergeCase candidates only | resolution distribution |
| 2.0.182 `sn_aggregate_customer_ids`, `sn_customer_id_map_v2` | CustomerIdentifier | customers.customer_identifier | TRANSFORM | legacy identifiers; never keys | code↔customer conflicts |
| `sn_leads`, `sn_lead_statuses`, `_status_history` | Lead, Case, state history | cases.* | TRANSFORM | namespaced states + LegacyValue | status counts |
| `sn_seller_flow_states/_events` | Case workflow state + events | cases.* | MERGE | duality → CaseAlias | alias report |
| `sn_mis_import_batches`, `_data_rows`, `_lead_pool` | ImportBatch, SourceRow | mis.* | DIRECT/TRANSFORM | — | rows by quality |
| `sn_distribution_items/_logs`, `sn_mis_distribution_*` | Assignment + CustodyEvent | custody.* | TRANSFORM | logs → events; gaps → UNKNOWN | current-holder parity |
| `sn_extra_number_requests` | ExtraNumberRequest | sales.* | DIRECT | — | counts |
| Woo products + `_sn_*` meta, `sn_project_card_upgrade_rules` | Product, ProductVersion, UpgradeRule | catalog.* | TRANSFORM | meta → components | product parity |
| `sn_invoices` | Invoice | billing.invoice | TRANSFORM + SPLIT | three status columns → separate states + LegacyValue; total via deferred resolver (TAD (E0) #19) | counts; total per unit |
| `sn_invoice_items` | InvoiceItem | billing.invoice_item | TRANSFORM | snapshot; `subscription_plus_cash` → credit_option | line sums = total |
| `sn_invoice_payment_stages` | PaymentStage | billing.payment_stage | DIRECT/TRANSFORM | — | stage sums |
| `sn_payments` | PaymentAttempt, GatewayAttempt | payments.* | SPLIT | gateway vs manual; `approved`, `reconcile_error` preserved | paid totals |
| receipt URLs / uploads | Receipt | payments.receipt | TRANSFORM | copy + hash; missing → NOT_RECOVERABLE | file existence |
| `sn_wallets`, `sn_wallet_transactions`, `sn_commission_wallet_posting_logs`, `sn_operations_wallet_codes` | LedgerEntry | finance.ledger_* | DERIVED + AMBIGUOUS | transactions → entries; mutated → UNKNOWN/CONFLICT; opening-balance entries with notes | derived vs legacy balance |
| `sn_commission_rules`, `sn_payment_commission_matrix`, `sn_project_commission_rules` | CommissionPolicy | commission.policy | TRANSFORM | — | counts |
| `sn_commission_dry_runs/_items`, `sn_project_commissions` | CommissionRun/Calculation | commission.* | TRANSFORM | — | amount parity |
| `sn_dot_cases`, `_payments`, `_invoice_links`, `_case_logs`, `_operational_archives` | Case (conversion), invoice links | cases.* + billing refs | MERGE | — | case/link parity |
| `sn_project_memberships`, `_membership_items`, `sn_project_subscriptions/_contents` | Subscription, SubscriptionItem | biawin.* | TRANSFORM | — | active parity |
| `sn_project_operations*`, `sn_operations_execution_*`, `sn_operations_form_*` | FulfillmentCase, workflow definitions | fulfillment.* | TRANSFORM | — | open-case parity |
| `sn_shipping_orders/_logs`, `sn_product_flow_events` | FulfillmentCase (shipping) | fulfillment.* | TRANSFORM | — | counts |
| `sn_after_sales_cases/_logs` | AfterSalesCase | aftersales.* | DIRECT | — | counts |
| `sn_campaign*`, `sn_dot_marketing_*` | Campaign, Attribution | sales.marketing | DIRECT | — | counts |
| `sn_payment_reward_claims`, `sn_invoice_wheel` | pending D-40 | — | AMBIGUOUS | — | — |
| `sn_project_notes/_messages/_message_reads` | CaseNote/Message | cases.* | DIRECT | — | counts |
| `sn_activity_logs`, `*_logs` | AuditEvent (imported) | audit.audit_event | TRANSFORM | actor/time as recorded; missing → UNKNOWN | counts |
| `sn_report_invoice_state_events` | reporting seed | reporting.* | DERIVED | — | — |
| missing early history | — | — | UNRECOVERABLE | NOT_RECOVERABLE | — |

---

## 26. DO-NOT-PORT register

| # | Item | Evidence | Target replacement |
|---|---|---|---|
| 1 | Phone identity joins | invoice grouping; SN-209 | §10 |
| 2 | Phone-keyed / phone-derived customer IDs | `sn_aggregate_customer_ids`, `map_v2` | CustomerIdentifier |
| 3 | Phone as dedupe authority | SN-209 W7 | MergeCase evidence (D-29/D-31) |
| 4 | Runtime schema mutation; CREATE/ALTER during requests | 312 DDL sites | crm-migrate |
| 5 | Read-path writes; schema creation from report/read paths | report backfill; F07 | read-only query transactions |
| 6 | Backfills during reads | SN-209 D2/D3 | import/jobs only |
| 7 | Admin-page / role creation as request side effect | HR page on `admin_init`; Invoice Review | SPA routes |
| 8 | Mutable custody history | E0 §10 | CustodyEvent |
| 9 | Mutable financial history | wallet reversal | compensating entries |
| 10 | JSON wallet balance as canonical finance model | `cash_wallet` | ledger |
| 11 | Hierarchy as sole/direct permission engine | SN-209 P2 | capability + row scope |
| 12 | Dual hierarchy truth | HR assignments + chains + scope-service | org_assignment |
| 13 | Role/position collapsing | position↔role mapping | Position ≠ Role ≠ Capability |
| 14 | Overloaded / product-specific invoice state | three status columns | separate machines |
| 15 | Implicit status fallback | 2.0.159 main-status fallback | explicit resolver + UNKNOWN |
| 16 | Hidden cross-module writes | Subscription+ reconcile → Dot / membership / commission | one primary write owner per command + outbox (§6 rules 2–5); multi-module transaction only by ADR |
| 17 | Direct WordPress DB runtime dependency | whole plugin | connector only |
| 18 | Product-specific schema hacks | `subscription_plus*`, `product_star` meta | ProductVersion components |
| 19 | Hard-coded migration markers | `15475:26204` | import manifests |
| 20 | Blind retry after unknown provider outcome | legacy Faraz second POST (removed in 2.0.161) | reconcile-first |
| 21 | Callback side effects unrelated to payment | D-26 auto-login; activation fan-out | inbox → events |
| 22 | UI-driven authorization | client gating; nonce as authority | server evaluator |
| 23 | Silent historical rewriting | — | append-only + audit |
| 24 | Undocumented 2.0.177–2.0.182 behaviour | SN-209 | intake only |
| 25 | Incomplete unified payment modal as frozen UX | SN-209 | SN-207-based design |
| 26 | Direct Dot completion; direct membership writes; recursive commission calls | SN-209 | §14 |
| 27 | Evidence referenced before durable storage | receipt URLs + post-commit moves (SN-209) | §15.1–15.5 |
| 28 | Mutable membership/case history without event evidence | SN-209 `on_reversed`, W4 | state history + events |

---

## 27. Three-year flexibility analysis

| Element | Likely change | Classification (TAD) |
|---|---|---|
| Organization structure | re-orgs, new units | STABLE DOMAIN CORE (tree + intervals); structure is data |
| Roles | new roles/bundles | CONFIGURATION CANDIDATE |
| Hierarchy | depth, skip levels | STABLE CORE; skip-level rules VARIABLE POLICY (OPD-10) |
| Delegation | acting roles | STABLE CORE entity; scope configuration |
| Sales flows | new case workflow types | core sales state MUST REMAIN CODE-DEFINED; sub-steps WORKFLOW CONFIGURATION CANDIDATE |
| Product types | many | CONFIGURATION CANDIDATE; new component kinds are code |
| Financial products | new loans/credit | WORKFLOW CONFIGURATION + PROVIDER ADAPTER |
| Banks / providers | added | PROVIDER ADAPTER CANDIDATE |
| Payment gateways | added/switched | PROVIDER ADAPTER CANDIDATE |
| Approval rules | thresholds, steps | CONFIGURATION CANDIDATE |
| Commission formulas | frequent | VARIABLE POLICY; new formula types are code |
| Customer identifiers | national ID, Biawin | CONFIGURATION CANDIDATE |
| Biawin services | 1 → 30+ | WORKFLOW CONFIGURATION CANDIDATE |
| Workflow steps | frequent | WORKFLOW CONFIGURATION CANDIDATE |
| Evidence requirements | frequent | CONFIGURATION CANDIDATE |
| Reporting dimensions | grow | STABLE fact grain; additive dimensions |
| Integrations | new partners | PROVIDER ADAPTER CANDIDATE |
| Notifications | templates/channels | CONFIGURATION + ADAPTER |
| Invoice / payment / ledger machines | rare | MUST REMAIN CODE-DEFINED |
| Money units / rounding | OPD-06 | OWNER DECISION REQUIRED |
| Subscription+ semantics | unknown | OWNER DECISION REQUIRED |

---

## 28. Overengineering risk review

| Risk | Mitigation (TAD) |
|---|---|
| Workflow grows into BPM | definitions select only Go-registered steps/guards/effects; no expressions, loops or scripts; used only by fulfillment, products-finance, HR routing |
| Rules language for commission | parameter sets over named Go functions; new formula type = code + tests |
| Event sourcing everywhere | only custody, ledger, audit, state history append-only |
| Too many modules | 21 business modules is the ceiling; merge a module with no aggregate after Slice 8 |
| Hexagonal ceremony | ports only for external systems and cross-module APIs |
| Generic approvals for everything | only real multi-actor decisions use ApprovalPolicy |
| Microservice drift | one binary set on one database (TAD (E0)) |
| Configurable everything | every configuration point must cite evidenced variation (§27) |

---

## 29. Continuous legacy-sync strategy

**TAD** Definitions: **LEGACY BASELINE VERSION** = tier A (2.0.160) · **LEGACY DELTA VERSION** = captured
version under audit · **DELTA AUDIT** = §3 output · **APPROVED OVERLAY** = register of adopted behaviours
(currently §2.1) · **ARCHITECTURE IMPACT** = ADRs required · **BACKLOG IMPACT** = slice items added/changed ·
**CUTOVER IMPACT** = migration mapping, reconciliation, parity tests, flag plan changes.

Process: capture release → automated structural diff → behavioural delta audit → classify impact → update
overlay register → update target backlog → change architecture only if required → record migration
implications → continue target implementation. Plugin changes are never continuously merged.

Cadence: per-release intake (event-driven); **immediate intake for HIGH-IMPACT releases (§3.1)**; rebaseline
(move tier A) only when (a) ~3 months or ~10 versions have accumulated in the overlay and (b) the version is
documented, stable in production and passes plugin snapshot CI, or (c) immediately before a migration slice
that imports from that version's schema. A high-impact trigger never shortens the path to rebaseline.
Importer extract specs are versioned per baseline.

Feature added in legacy while the equivalent target module is under construction: intake classifies it; if
DOMAIN/WORKFLOW impact and owner-approved, it becomes a scoped backlog item in that module; the current slice
continues unless the item contradicts its acceptance criteria, in which case a blocker is recorded and the
owner chooses absorb now / defer / reject; the target never pauses to catch up automatically; a schema-shape
change in an already-imported module bumps its extract-spec version and re-runs reconciliation.

---

## 30. Source-of-truth precedence

**TAD**

1. Explicit Human Owner decision (recorded on main)
2. Frozen target architecture / ADR (E0, D-01, D-02, this record, future ADRs)
3. Frozen product invariants (Gate-0, Cross-Role V1, AGENTS.md global contract, frozen role specs, SN-207 design)
4. Approved behavioural overlay (§2.1 and future approved overlays)
5. Official legacy baseline behaviour (2.0.160)
6. Development candidate evidence (2.0.182 and later, unclassified)
7. Prototype / UI mock
8. Assumptions (labelled; never sufficient for writes)

**Legacy (5–6) never automatically overrides levels 1–4.** Any contradiction between levels is recorded as a
blocker and surfaced, never resolved silently. Level 1 may override level 2 only through a new ADR.

---

## 31. Implementation sequence (Slice 0 – Slice 16)

**TAD** Vertical, flag-gated, reversible slices. Only Slice 0 is approved for implementation (E0). Each later
slice requires its own owner approval and the decisions listed.

| # | Slice | Scope | Modules | DB | API | Frontend | Legacy dependency | Acceptance | Rollback | Owner decisions | Migration effect |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | Go Contracts Spine (APPROVED) | kernel | kernel | none | none | none | none | E0 §12 | revert PR | — | none |
| 1 | Platform foundation | migrate runner, per-module schemas and grant model, tx manager, per-module outbox/inbox/idempotency library, audit, clock, flags, CI lint | platform, audit | first migrations | health only | none | none | migrations up/down on empty DB; lint blocks cross-schema SQL and cross-module command calls in tx | drop dev DB | staging/anonymization env (TAD (E0) #18) | none |
| 2 | API + SPA shell | OpenAPI pipeline, generated client, envelope, errors, dev-only auth stub, SN-207 shell | platform, web | none | §23 conventions | shell + design system | none | contract tests; client compiles | revert | — | none |
| 3 | Legacy connector + staging | extract specs v1 (2.0.160), snapshot extraction, staging, manifests | integration | stg schema | none | none | snapshot export, no prod writes | checksums and counts match source | drop staging | D-23 (PII in staging), anonymization | none |
| 4 | Identity + Org read | users, roles, persons, employment, org units, assignments; read evaluator | identity, organization, authorization | yes | read | HR read (SN-201…206) | staging | capability diff ≈ legacy; hierarchy conflicts reported | flag off | — (read) | read-only |
| 5 | Customer identity (shadow) | customers, contact points, evidence, MergeCase candidates | customers | yes | read + merge queue read | customer read | invoice staging | resolution report; zero phone-only merges | flag off | D-30 for later writes | shadow |
| 6 | Sales read | cases, custody events, attribution snapshots | cases, custody | yes | read | seller/supervisor/manager read | staging | custody parity; honest F02/F03 labels | flag off | OPD-01 | shadow |
| 7 | Billing + payments read | invoices, stages, attempts, receipts; separated states | billing, payments | yes | read | invoice views | staging | per-invoice totals per unit; unknown statuses preserved | flag off | TAD (E0) #19/#20 resolvers | shadow |
| 8 | MIS / reporting read | projections, metric contracts, Invoice Review query (no export) | reporting, mis | projections | read | MIS/report pages | events from 4–7 | card = table = detail parity (INV-021) | flag off | D-23, D-24, D-45, OPD-07 | none |
| 9 | Finance shadow ledger | ledger derived from legacy wallets/payments; balance reconciliation | finance, commission (read) | yes | read | finance read | staging | derived vs legacy balance diff with reasons | flag off | D-41, OPD-06, OPD-09 | shadow |
| 10 | Biawin / fulfillment read | subscriptions, items, fulfillment cases | biawin, fulfillment, catalog | yes | read | portal/operations read | staging | active subscription parity | flag off | D-35, D-38 | shadow |
| 11 | Controlled non-financial writes | case outcomes, custody moves, HR requests; write-back via connector | cases, custody, organization | yes | commands | seller/supervisor actions | write-back adapter | dual-run parity; per-item outcomes; UNKNOWN reconciled | per-flag revert; legacy master | ADR-01, OPD-02/03/05/10, SD-04, D-32, D-33, D-34, D-42 | strangler starts |
| 12 | Financial writes | payment batch with durable evidence (quarantine + evidence storage, finalisation worker, evidence reconciliation), gateway, review, ledger posting, next stage | payments, finance, billing | yes | commands | finance + seller payment UI | gateway adapters; write-back | idempotency/replay; reconcile-first; fault injection proves no committed attempt without a durable object; finalisation idempotent; storage failure leaves financial state intact; maker/checker per policy | flag + compensating entries | D-25, D-26, D-28, D-32, D-43, D-44, OPD-04/06/08 | finance cutover per channel |
| 13 | Commission engine | policies, runs, posting requests | commission | yes | commands | finance run UI | — | parity vs legacy runs on a historical window | flag off | OPD-09, D-36 | engine switch |
| 14 | Biawin writes | activation, fulfillment workflows, Biawin port | biawin, fulfillment | yes | commands | operations UI | Biawin API | outbox delivery; unknown reconciled | flag off | D-35, D-38 | service cutover |
| 15 | Final cutover | target master per module; legacy read-only | all | — | — | — | — | INV-036 old/new metrics within tolerance | per-module rollback window | owner go/no-go | cutover |
| 16 | Legacy retirement | archive legacy snapshot; keep LegacyReference | integration | — | — | — | — | evidence archive verified | archive retained | owner | complete |

---

## 32. Open owner decisions (none resolved by this record)

| ID | Question | Domain | Phase blocked | Safe default | Owner | Target may continue? |
|---|---|---|---|---|---|---|
| D-03…D-22 | Content not recorded on main; **not authoritative until separately recorded** (E0 rule 1) | various | any phase relying on them | no dependent behaviour implemented | Human Owner | Yes, until the dependent phase |
| D-23 | Invoice Review visibility / PII / export | reporting | Slice 8 export | no export; audited read behind flag | Product + Finance | Yes |
| D-24 | Invoice Review product/design owner | reporting UI | target Invoice Review UI | no UI | Product | Yes |
| D-25 | Gateway auto-approval policy | payments/finance | Slice 12 | `gateway_auto` provenance; not Finance-reviewed | Finance | Yes until 12 |
| D-26 | Customer auto-login after gateway callback | identity/payments | Slice 12 | not replicated | Product + Security | Yes |
| OPD-01…10 | Per `CRM-CROSS-ROLE-ARCHITECTURE-V1` §19 | per row | per row | per Phase-0 spec §21 | per row | Yes (read slices) |
| D-28 | Assessment invoice partial-payment policy (2.0.176 removed the full-payment rule) | billing/cases | Slice 12 | keep 2.0.160 rule (full payment) | Product + Finance | Yes |
| D-29 | Seller duplicate-phone business intent | sales | Slice 11 | no target guard; duplicates as MergeCase hints | Product + Sales | Yes |
| D-30 | Customer merge authority and evidence threshold | customers | Slice 5 writes / 11 | no merges; RELATED/CONFLICT only | Product + MIS + Sales | Yes (read) |
| D-31 | Phone as dedupe of any kind (alias of D-29) | customers/sales | Slice 11 | discovery only | Product | Yes |
| D-32 | Hierarchy-based next-stage authority | billing/authorization | Slices 11/12 | 2.0.160 rule: owning seller + admin | Product + Sales + Finance | Yes |
| D-33 | Operations upgrade-payment authority | payments/fulfillment | Slice 12 | not granted | Product + Finance | Yes |
| D-34 | HR initial-password policy | identity | Slice 4 writes / 11 | random password + reset flow | Security + HR | Yes |
| D-35a–e | Subscription+ business model, lifecycle, cash-credit semantics, Biawin fulfilment, mandatory conversion | catalog/biawin/finance | Slices 10 writes / 14 | read-only representation; no activation logic | Product + Finance | Yes (read) |
| D-36 | Pro-rata commission split across bundle components | commission | Slice 13 | no component allocation; legacy runs historical | Finance + Product | Yes |
| D-37 | Installment credit: financing product or entitlement | products-finance/finance | Slices 9/12 | entitlement account, flagged provisional | Finance | Yes |
| D-38 | Biawin wallet charge semantics and system of record | biawin/finance | Slice 14 | no target charge; legacy master | Finance + Product | Yes |
| D-39 | Catalog `all_except` business semantics | catalog/authorization | Slice 11 | `selected`/`all` only; `all_except` imported read-only | Sales + Product | Yes |
| D-40 | Payment reward / wheel: keep or retire | sales/finance | Slice 12 | read-only import | Product | Yes |
| D-41 | Ledger basis: accrual vs cash | finance | Slice 9 | shadow ledger computes both, publishes neither | Finance | Yes |
| D-42 | HR editing sales capabilities (portal_access bulk/inline) | organization/authorization | Slice 4 writes / 11 | HR cannot grant sales capabilities; identity owns grants | Product + HR + Sales | Yes |
| D-43 | Finance approval while receipts are not READY | payments/finance | Slice 12 | approval blocked unless READY | Finance | Yes |
| D-44 | Evidence retention; unbound-upload expiry; legal retention for NOT_RECOVERABLE records | payments | Slice 12 | retain bound evidence indefinitely; unbound 7 days | Finance + Legal | Yes |
| D-45 | Behaviour when a sensitive-read audit write fails | reporting/audit | Slice 8 | fail closed for PII-bearing views | Product + Security | Yes |
| ADR-01 (candidate) | Accept the reservation protocol for INV-009, or approve a specific multi-module transaction ADR | cases/custody/billing | Slice 11 | no target invoice-issue/return writes | Human Owner + Engineering | Yes, until Slice 11 |

D-27 remains CLOSED (E0 §7C).

---

## 33. Risks and failure modes

| Risk | Impact | Mitigation |
|---|---|---|
| D-03…D-22 not on main | later phases stall or rely on undocumented decisions | commit before Slice 4 planning |
| No staging/anonymization | Slice 3+ cannot import real data | environment owner blocker (TAD (E0) #18) |
| Identity migration mostly UNKNOWN/CONFLICT | incomplete customer views | honest states; work queues; no forced merges |
| Wallet history unreconstructable | disputed opening balances | opening-balance entries + reconciliation; Finance sign-off |
| Legacy changes faster than intake | overlay growth, rebaseline churn | §3.1 + §29 cadence; reject undocumented builds |
| Over-generic workflow/approvals | slow, opaque | §28 guardrails; ADR per new configuration point |
| Cross-module coupling returns | hidden writes | §6 rules, schema grants, lint, architecture tests |
| Eventual-consistency windows | users see intermediate states | honest intermediate states (§6 rule 6); reconciliation on rejection |
| HR/Finance runtime NOT LIVE VERIFIED | wrong parity baselines | dedicated role accounts before Slices 4/9 acceptance |
| Write-back divergence | dual-master corruption | single master per module per flag; outbox write-back + reconciliation |
| PII in staging | privacy breach | D-23; anonymised non-production copies |
| Gateway/SMS UNKNOWN | double charge/notification | reconcile-first; no blind retry |
| Evidence storage failure | unverifiable payments | §15 durable quarantine, idempotent finalisation, reconciliation |
| Subscription+ undecided | Biawin slices blocked | read-only representation until D-35 |

---

## 34. Final implementation-readiness statement

Consistency check performed at freeze across: one aggregate → one write owner; one command → one primary
write owner; append-only finance/custody/audit; read paths never writing; phone never canonical identity;
target precedence over moving legacy; workflow configurability versus domain invariants; permissions versus
hierarchy; DB ownership; React holding no authoritative business rules. No open contradiction remains in
this record. INV-009 cross-module mutual safety is addressed by the ADR-01 candidate and gates Slice 11.

Conditions: this record resolves no owner decision; D-03…D-22 must be recorded on main before any phase that
relies on them; ADR-01 must be accepted before Slice 11; each slice after Slice 0 requires separate owner
approval; HR and Finance runtime remain NOT LIVE VERIFIED.

**ARCHITECTURE V1 FROZEN — SLICE 0 IMPLEMENTATION MAY BEGIN**
