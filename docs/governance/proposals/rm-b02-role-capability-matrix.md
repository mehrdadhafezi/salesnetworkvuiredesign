# RM-B02 — Proposed role-to-capability matrix (8 roles × 43 capabilities)

**PROPOSAL — NOT APPROVED. Documentation only. No grant, bundle, flag, migration, account or configuration is created or changed.**

| | |
|---|---|
| Workstream | RM-B02 (R7 roadmap §7 B), associated with D-42 |
| Baseline | `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634` (PR #66, RM-ACCESS-01, merged) |
| Branch | `docs/rm-b02-role-capability-matrix` |
| D-42 tracker status | `READY_FOR_REVIEW` (unchanged by this document) |
| Approvals recorded by this document | **0** |
| Production | **NO-GO** (unchanged) |

Companion documents: [scope and ownership](rm-b02-scope-and-ownership.md) · [separation of duties](rm-b02-separation-of-duties.md) · [D-42 decision package](rm-b02-d42-decision-package.md) · [executive summary (fa)](rm-b02-executive-summary.fa.md) · [evidence traceability](rm-b02-evidence-traceability.md).

## 1. What this document is and is not

It **is** an evidence-based proposal: for each of the eight business roles and each of the 43 existing backend capabilities, a documentation classification, a proposed analytical scope, and the conditions that would have to be satisfied first.

It **is not**:
- an authorization approval, a grant, or a runtime state. No cell here is `ACTIVE`. The classifications `PROPOSED_ALLOW`, `PROPOSED_CONDITIONAL`, `PROPOSED_DENY`, `DEFERRED_DECISION` and `NOT_APPLICABLE` exist only in this document;
- a decision on D-42. In R5 and M2, D-42 decides only **who may grant, edit and revoke** capabilities. Which role is eligible for which capability is a separate question with no decision ID yet (M2 proposed record 1). This document informs both, approves neither (see CF-01 in §10 and the [decision package](rm-b02-d42-decision-package.md));
- a design of new capabilities. Product actions with no backend capability are listed in §9 as **requirements**, never as implemented keys.

## 2. Existing system facts (CODE VERIFIED at the baseline)

| Fact | Evidence |
|---|---|
| The registry holds **43** capabilities; the database catalog and the OpenAPI enum hold the same 43 | `internal/identity/api/api.go` `vocabulary`; `identity.capability` (migrations 00001–00010); `api/openapi/components/hr.yaml#/Capability`; test `TestL_ContractEnumEqualsRegistry` (RM-ACCESS-01) |
| There are **9** target roles: the 8 business roles plus the technical `invoice_reviewer` | `api.go` `roles`; legacy `sn_invoice_reviewer` maps to it (`internal/integration/legacy/shadow/mapping.go:37`) |
| Only `hr` has a bundle: the 6 HR reads. The other 8 roles, including `invoice_reviewer`, have empty bundles | `api.go` `roles`; `00001_identity.sql:137-143`; `TestVocabularyShape` |
| **No ACTIVE grant exists or can exist**: the grant table only accepts `SHADOW` grants with basis `LEGACY_PROFILE_ROLE`; no grant, revoke or audit command exists | `00001_identity.sql:195` (`role_grant_shadow_only`); M2 D-42 analysis §1 |
| The evaluator is a **union**: every counting ACTIVE grant whose bundle holds the capability matches; all matched roles are reported, sorted, with no selection | `internal/authorization/authorization.go`; RM-ACCESS-01 test J |
| Every allowed decision is scope **ORGANIZATION**. There is no own, team or subtree scope in the evaluator | `authorization.go:163` (`ScopeOrganization`) |
| Row guards exist **inside** some commands and own-scope reads only: contact outcome (current holder), HR request create (descendant seller) and review (current reviewer), HR request read (requested or reviewing), payments upload/batch/submission (own), commission own (own user), fulfillment work (per-service scope) | Slice 11, 12, 13, 14 command matrices |
| Feature flags give **availability only**, never authority; `allowed` = ACTIVE grant **and** that capability's own flag | RM-ACCESS-01 §4–5 |
| No capability exists for: Finance approve/reject, posting, refund, settlement, commission approval, HR final apply, credential management, grant administration, invoice issuance, next stage, allocation/return, MIS import or maintenance, Biawin activation | `TestNoFinancialExecutionCapability`; Slice 11–14 blocked lists; registry comments |
| Real users can log in (R1) but every business read and write answers 403 for them | R7 §3; `web/e2e/seller-contact-outcome.spec.ts` |
| HR and Finance role runtime: **NOT LIVE VERIFIED** | AGENTS.md; Cross-Role V1 §1 |

## 3. Classification rules and global preconditions

### 3.1 Meaning of each classification

| Classification | Meaning in this document | Evidence rule |
|---|---|---|
| `PROPOSED_ALLOW` | The capability's **implemented** semantics and scope match a need stated in a frozen or accepted source for this role, with no role-specific condition beyond the global preconditions | Must cite the source (§6 justification) |
| `PROPOSED_CONDITIONAL` | A stated need exists, but a named condition must be met first (usually a row scope that does not exist yet, a field policy or an open decision) | Must cite the source **and** name every condition key (§7) |
| `PROPOSED_DENY` | No source states a need for this role, or a source places it outside the role (`MUST NOT OWN`, `Should Hide`, an SoD conflict). Not proposed; a later decision may revisit it | Cites the excluding source where one exists |
| `DEFERRED_DECISION` | A need may exist, but the deciding question is open (or no product contract exists). The cell is left undecided on purpose | Names the open decision or gap |
| `NOT_APPLICABLE` | The capability is own-scoped to an activity the role never performs (an own-scope read that would return nothing) | Explained per cell |

Rules applied to every cell:
- **No rank inheritance.** A higher tier gets nothing because a lower tier has it. Each cell for Supervisor, Senior, Manager and Deputy was decided from that role's own source (Cross-Role V1 §1: higher position does not inherit Finance/HR/MIS authority or lower-tier writes).
- **Navigation is not authorization.** RM-E02 destination mappings are cited only as context; one of them conflicts with a role contract (CF-06).
- **Read scope is not write scope.** Writes are classified separately even when the role reads the same object.
- **Unknown stays unknown.** Where no source answers, the cell is `DEFERRED_DECISION`, not `ALLOW` or `DENY` by seniority.
- **Cross-domain writes are never proposed**: no Sales role receives a Finance, HR-apply or MIS write; HR receives no Finance or Sales capability; Finance receives no HR or Sales write; MIS receives no financial or HR capability.

### 3.2 Global preconditions (apply to every cell, not repeated)

| Key | Precondition | Source |
|---|---|---|
| G1 | D-42 decided (who grants, edits, revokes) **and** E-7 grant-change procedure with audit implemented (RM-B03) | R5 inventory; M2 §3 |
| G2 | A per-role capability-eligibility decision exists. For the seller this is M2 proposed record 1; for the other seven roles **no record is proposed yet** (gap GAP-01 in the decision package) | M2 §2, §4 |
| G3 | D-34 account provisioning decided and implemented (RM-B01) | R5; R7 §7 B |
| G4 | For any SoD-sensitive action: R5-SEC-01 decided with option A or B and implemented (RM-B04). Account-level grants never satisfy SoD | R5 P-SEC §2; M2 §2-2 |
| G5 | The capability's own feature flag and rollout mode stay separate availability gates; activation is separately authorized (R5 readiness ladder steps 4–6); Production remains NO-GO | RM-ACCESS-01 §4; R5 inventory |

### 3.3 Sources and their status (summary; full list in the traceability document)

| Source | Status at HEAD | Used as |
|---|---|---|
| Cross-Role Architecture V1 (XR) | FROZEN TARGET — IMPLEMENTATION DEFERRED | Product authority for role map (§2), ownership (§5), permission boundary (§12) |
| Gate 0 (G0) | FROZEN TARGET / IMPLEMENTATION DEFERRED | Invariants INV-023…025; Supervisor input §19 |
| Product-role Audit (A) and Decisions/Visibility appendix (DV) | ACCEPTED (cited as `Audit پذیرفته‌شده` by XR) | Seller/Supervisor contract basis (XR §1); visibility per role |
| Senior (SS), Manager (MG), Deputy (DP), MIS (MS), HR (HRS), Finance (FS) specs | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED (header of each) | Per-role permission concepts (§22 / §15 / §14 / §16) |
| Standalone Target Architecture V1 (ARCH) §4A, §17, §18 | Architecture freeze record; authorizes no slice | Module-level role → capability family intent |
| M2 intake and D-42 seller analysis | DRAFT — NOT APPROVED | Seller options and scope definitions |
| R5 inventory, R6 tracker | Inventory / tracker (statuses unchanged) | Decision IDs and statuses |
| RM-E02, RM-ACCESS-01, R7 audit | Merged engineering / audit records | Code facts and navigation context only |
| Seller Design V1, Supervisor design | Visual only | **Not** business or authorization authority |


## 8. Technical role `invoice_reviewer` (documented separately; not a ninth business role)

| Item | Fact or proposal |
|---|---|
| Runtime fact | Target role key `invoice_reviewer` exists with an **empty** bundle; legacy label `sn_invoice_reviewer` maps to it as SHADOW evidence only (`mapping.go:37`) |
| Architecture intent | ARCH §4A: "Invoice Reviewer → reporting (Invoice Review query surface)"; ARCH §18: `sn_invoice_reviewer` → invoice review read |
| Proposed cell for `reporting.invoice_review.read` | **DEFERRED_DECISION**: D-23 (visibility, PII, export) and D-24 (product/design owner) are open; D-45 governs audit-write failure (safe default fail closed) |
| Proposed cells for the other 42 capabilities | **PROPOSED_DENY**: no source states another need |
| Not proposed | Treating `invoice_reviewer` as a business role, or folding it into Finance. Whether a Finance person also holds it is a multi-role question (decision package §4) |

## 9. Product actions with no backend capability (requirements, not implemented keys)

The registry has no capability for the actions below. **Nothing here is implemented, and no key here is registered.** Where ARCH §18 already gives an *example* name, it is quoted as an example only; otherwise no name is proposed. Adding any capability needs, in one change: registry entry, catalog migration, OpenAPI enum value, `capabilityFlags` entry, and a navigation CONTRACT destination if navigable (RM-ACCESS-01 §11).

| Req | Product action | Roles needing it (source) | Nearest existing capability and gap | Blocking decisions / gaps |
|---|---|---|---|---|
| MC-01 | Grant, revoke and periodically review role grants | HR `manage_business_access` (HRS §16); Security (D-42 options) | none | D-42, E-7, R5-SEC-01 |
| MC-02 | Account provisioning; credential reset / send | HR `manage_credentials` RESTRICTED (HRS §13, §16) | `identity.user.read` is read only | D-34, R5-SEC-01 |
| MC-03 | Impersonation with purpose, expiry and trail | HR `impersonate_user` RESTRICTED (HRS §13) | none | D-34, R5-SEC-01 |
| MC-04 | HR final apply of transfer / termination with handover | HR (XR §12 `A specific authority`; HRS §9–11); ARCH example `org.assignment.apply` | `org.hr_request.review` is step only; HR actor BLOCKED | OPD-05, SD-04, R5-SEC-01 |
| MC-05 | Workforce profile create / edit; current hierarchy edit | HR (HRS §16) | org reads only | F08, OPD-05 |
| MC-06 | Compensation view / edit | HR (HRS §14; G0 §10 `view_compensation`, `edit_compensation`) | none (no compensation read either) | OPD-09 (input boundary), F08 |
| MC-07 | Workforce import / export; HR audit view | HR (HRS §16) | none | ADR #18 (data handling) |
| MC-08 | Finance stage approve / reject, including bulk | Finance (FS §8, §15); ARCH example `finance.stage.approve` | `finance.review.read` (read only) | OPD-08, D-43, REJECT-VOCAB (R4 gap), NO_DECISION_CAPABILITY (R4 gap), R5-SEC-01 |
| MC-09 | Refund request, approval, execution, confirmation | Finance (FS §9); Sales/after-sales request (XR §12 `request/context`) | none | OPD-04, OPD-08, R5-SEC-01, D-44 |
| MC-10 | Ledger posting, reversal, manual adjustment | Finance (FS §10) | `finance.*.read` (shadow read) | D-41, OPD-06, OPD-09, OPD-08 |
| MC-11 | Commission rule authoring, run review / approval, posting execution, settlement | Finance (FS §11; G0 §10 Rule Author / Run Reviewer / Posting Actor) | `commission.shadow.read`, `commission.policy.read` | OPD-09, D-36, D-41, OPD-08 |
| MC-12 | Rollout / posting-activation flag management | Restricted rollout owner (DV Finance error prevention; FS §11) | none; flags change out of band | E-7 |
| MC-13 | Sensitive export (finance, workforce, reports) as an audited command | All roles `export_*` CONDITIONAL (SS §22, MG §15, DP §15, MS §14, HRS §16, FS §15); ARCH §18 `CreateExport` | none | D-23 (invoice review), ADR #18, D-45 |
| MC-14 | Invoice issuance (incl. pre-invoice) and case claim for invoice | Seller A; Sup/Senior/Manager C (XR §12); ARCH example `billing.invoice.issue` | none | ADR-01, F01, OPD-10 |
| MC-15 | Next payment stage issuance | Seller / converter (legacy) | none | D-32 |
| MC-16 | Allocation, distribution and transfer of case custody | Sup A, Senior/Manager A/C, Deputy C, MIS A governed (XR §12) | `custody.custody.read` (read only) | OPD-10 (alias SD-01), F01, ADR-01 |
| MC-17 | Ordinary return, release, reset | Same tiers (XR §12) | none | ADR-01, F01, OPD-03 |
| MC-18 | Ready conversion: mark ready, ready assignment, own conversion actions | Seller (ready event), Supervisor `A scoped` (XR §12); Senior read only | none (`MarkReady` BLOCKED, S11) | OPD-10 |
| MC-19 | Extra-number request (Seller) and decision (assigned Manager only) | XR §12; MG §12 | none | OPD-10 |
| MC-20 | Archive explorer read and archive revival | Manager (MG §11) | none | OPD-03 (revival NEEDS VALIDATION) |
| MC-21 | Invoice assist: resend link, receipt / prepayment assist, allowed sales correction | Sup/Senior/Manager (MG §15; SS §22 `assist_invoice_action`) | none | OPD-10 |
| MC-22 | Contact outcome value DUPLICATE | Seller | `cases.contact_outcome.record` blocks the value | D-29 / D-31 |
| MC-23 | Customer merge propose / resolve | MIS / named role (D-30 options) | `customers.merge_case.read` (read only) | D-30, ADR #18 |
| MC-24 | Own wallet balances (posted / earned / settled) | Seller and every sales tier (`view_own_wallet`) | `commission.own.read` covers historical commission credits only | D-41, OPD-09 |
| MC-25 | MIS import and batch / row views | MIS (MS §6, §14 `import_source_data`, `view_import_batches`) | none | OPD-01 |
| MC-26 | MIS source assignment, delivery, quick delivery | MIS (MS §14 `assign_source_cases`) | none | OPD-10, F01 |
| MC-27 | MIS return of source cases | MIS (MS §14 CONDITIONAL) | none | F01, OPD-03 |
| MC-28 | MIS data correction and maintenance actions | MIS (MS §13 MAINTENANCE ONLY) | `reporting.mis.read` (diagnosis only) | OPD-01, F01 |
| MC-29 | MIS planning | MIS (MS §12) | none | OPD-07 |
| MC-30 | Row-scoped variants of existing reads (own, direct team, subtree, territory, linked item) | All sales tiers, MIS, Finance (condition RS / LNK) | the 43 reads evaluate ORGANIZATION | M2 proposed record 2; GAP-02 (decision package) |
| MC-31 | Delegation / acting on behalf | Sales tiers (XR §6 skip-level, assist) | none; delegation not modelled (R5 P-SEC §1) | OPD-10, R5-SEC-01 |
| MC-32 | HR request queue for HR (`pending_hr`) | HR (HRS §11) | `org.hr_request.read` returns requested / reviewing only | OPD-05 |
| MC-33 | Deputy exceptions coordination view | Deputy (DP §10 `view_exceptions` ADD) | none | OPD-02 |
| MC-34 | Biawin production activation, wallet charge, membership lifecycle | not assigned to any of the eight roles | `biawin.activation.read` | D-35a … D-35e, D-38 |
| MC-35 | Fulfillment cancel / reopen / submit review | not assigned | none | OPD-04, OPD-08 |
| MC-36 | Operations upgrade payment | not assigned | none | D-33 |

## 10. Major access conflicts and tensions (recorded, not resolved)

| ID | Conflict | Sources in tension | Effect on this proposal |
|---|---|---|---|
| CF-01 | **Scope of D-42.** The RM-B02 brief treats the role-capability matrix and multi-role policy as D-42 work. R5 and M2 restrict D-42 to *who may grant, edit and revoke*; capability eligibility and seller read scope are separate (unnumbered) proposed records, and M2 states D-42 must not be widened | RM-B02 brief vs R5 inventory row D-42; M2 §2, §4 | The decision package keeps D-42's R5 question and options verbatim and lists the other questions as separate proposed records without IDs |
| CF-02 | **Role spec status labels.** Each role spec header says FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED. R7 §8 says each role spec labels itself READY FOR REVIEW; no such label exists in `docs/source-of-truth/` at HEAD. RM-E02 uses READY_FOR_REVIEW for the **design** freeze | spec headers vs R7 §8 vs RM-E02 §7 | Product content of the specs is used as product authority; their design status is not relied on |
| CF-03 | **Supervisor freeze.** SS header calls Supervisor Design V1 frozen; RM-E02 marks it FREEZE_UNVERIFIED; G0 §19 says "READY FOR PRODUCT SPEC" and no standalone Supervisor product spec exists. XR §1 adopts A/G/D as the Supervisor contract | SS vs RM-E02 vs G0 §19 vs XR §1 | Supervisor cells rely on XR §2/§5/§12, G0 §19 and DV only |
| CF-04 | **Finance / billing row scope.** The registry comment says billing and payment reads stay unassigned because D-25, D-26 and D-28 are open. Those decisions cover gateway auto-approval, auto-login and partial payment, not read scope. No record decides Finance row scope | `api.go` Slice 7 comment vs R5 inventory rows D-25/D-26/D-28 | Finance billing/payment cells are CONDITIONAL on `FRS`; a separate scope record is proposed (GAP-03) |
| CF-05 | **Customer read assignment.** The registry ties customer reads to D-30, which decides merge authority, not read eligibility | `api.go` Slice 5 comment vs R5 row D-30 | Customer read cells carry `CUS`; GAP-04 proposes a record |
| CF-06 | **Deputy and MIS diagnostics.** RM-E02 maps Deputy diagnostics to `/mis/reconciliation` (`reporting.mis.read`). DV says Deputy should hide daily technical logs; XR §2 lists MIS repair among Deputy's MUST NOT OWN (repair, not read) | RM-E02 §5 vs DV vs XR §2 | Cell left DEFERRED_DECISION; navigation does not prove authority |
| CF-07 | **Position as an eligibility input.** `CreateHRRequest` requires the actor's HR position slug in {supervisor, senior_supervisor, sales_manager, sales_deputy} in addition to the capability (legacy guard port). The target rule is Position ≠ Role ≠ Capability; ARCH §18 allows hierarchy only as an input | S11 vs XR §12 / ARCH §18 | Not a violation (the capability is still required) but a role holder without that position is refused; must be part of the eligibility decision |
| CF-08 | **HR finalizing `pending_review`.** XR §11 says current HR may also finalize `pending_review`; Slice 11 blocks every HR action (final apply has no capability) | XR §11 vs S11 | HR cells for HR request capabilities DEFERRED (OPD-05, SD-04) |
| CF-09 | **Scope gap (fact vs target).** Every role spec requires own / team / subtree / territory scope; every implemented read is organization-wide. Granting any read to a sales tier today would expose the whole organization | `authorization.go:163` vs SS §22, MG §15, DP §15, M2 §5 | 46 CONDITIONAL cells carry `RS`; none can become a grant before server-side scope exists |
| CF-10 | **Architecture capability families with no capability.** ARCH §4A lists Seller billing issue, Supervisor custody allocation and ready assignment, Finance posting and run approval, HR organization (all) and credential policy; none of these writes exists | ARCH §4A vs registry | Listed in §9; ARCH §4A "organization (all)" is read as module ownership, not as one undivided grant (G0 §10 requires HR concepts to be independent) |
| CF-11 | **Own wallet vs commission credits.** Sales specs ask for own wallet balances; the only own-scope capability reads historical commission credits | specs vs registry | `commission.own.read` ALLOW covers part of the need; MC-24 records the rest |

## 11. What this document does not change

- No file under `internal/`, `api/`, `web/`, `cmd/` or any migration.
- No role bundle, grant, flag default, rollout mode, account, credential or configuration.
- No status in `docs/governance/r6-decision-tracker.json` or `docs/remediation/r5-decision-inventory.md`.
- No source-of-truth document.
- No decision ID is created; proposed records carry descriptive labels only.
