# RM-B02 — Evidence traceability

**Documentation only.** Baseline `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634` (PR #66 merged at 2026-11-16T15:18:31Z). Production **NO-GO**.

## 1. Evidence labels used

| Label | Meaning (XR §1 convention) |
|---|---|
| FROZEN / ACCEPTED | product authority as recorded in the document header or by XR §1 |
| CODE VERIFIED | read from code at the baseline (handler, guard, migration, registry) |
| LIVE VERIFIED | none in RM-B02: no runtime observation was made |
| NOT VERIFIED / NOT LIVE VERIFIED | no proof; HR and Finance role runtime are NOT LIVE VERIFIED |
| READY_FOR_REVIEW / EVIDENCE_MISSING / DRAFT | R6 tracker status of a decision (quoted) |
| PROPOSED | this document set; never a grant |

No live audit, browser session, database query or production / staging access was performed for RM-B02.


## 2. Sources reviewed (verified present at the baseline)

| Key | Path | Status at HEAD | Used for |
|---|---|---|---|
| XR | `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md` | FROZEN TARGET — IMPLEMENTATION DEFERRED (header) | §1, §2 role map, §3 domains, §5 ownership, §6 handoffs, §7, §9, §11 requests, §12 permission matrix |
| G0 | `docs/source-of-truth/GATE-0-PRODUCT-INVARIANTS.md` | FROZEN TARGET / IMPLEMENTATION DEFERRED (header) | §10 permission architecture; §16 INV-023…025, INV-030…034; §19 Supervisor input |
| A | `docs/source-of-truth/CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md` | ACCEPTED audit (cited as accepted by XR §1) | basis of Seller / Supervisor contracts via XR §1 |
| DV | `docs/source-of-truth/DECISIONS-VISIBILITY-INTERACTIONS.fa.md` | Audit appendix (INFERENCE proposals; accepted with A) | per-role Must See / Should Hide; navigation counts; Finance error prevention |
| RAM | `docs/source-of-truth/ROLE-ACCESS-MAP.fa.md` | initial access record (2026-09-30); audit not complete at that date | context only; no authorization claims taken |
| S | `docs/source-of-truth/STATUS-TRANSITIONS.fa.md` | CODE VERIFIED enums; HR / Finance NOT LIVE VERIFIED | status namespaces only (no transition invented) |
| C | `docs/source-of-truth/CODE-EVIDENCE-AND-ACTION-REGISTRY.md` | legacy code registry (2.0.123 snapshot) | legacy guard context only |
| SS | `docs/source-of-truth/SENIOR-SUPERVISOR-PRODUCT-SPEC.md` | frozen role spec (states Seller / Supervisor design V1 frozen as visual basis) | §22 permissions; §25 data visibility |
| MG | `docs/source-of-truth/SALES-MANAGER-PRODUCT-SPEC.md` | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED | §11–§15 |
| DP | `docs/source-of-truth/SALES-DEPUTY-PRODUCT-SPEC.md` | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED | §10, §15 |
| MS | `docs/source-of-truth/MIS-PRODUCT-SPEC.md` | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED | §6–§14 |
| HRS | `docs/source-of-truth/HR-PRODUCT-SPEC.md` | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED; HR runtime NOT LIVE VERIFIED | §9–§16 |
| FS | `docs/source-of-truth/FINANCE-PRODUCT-SPEC.md` | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED; Finance runtime NOT LIVE VERIFIED | §6–§16 |
| ARCH | `docs/implementation/CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md` | architecture freeze record; authorizes no slice beyond Slice 0 | §4A role → capability families; §17 approvals; §18 permissions |
| S11 | `docs/implementation/slice-11-command-matrix.md` | implementation record (CODE VERIFIED behaviour) | contact outcome; HR request commands |
| S12 | `docs/implementation/slice-12-command-matrix.md` | implementation record | payments commands; blocked Finance commands |
| S13 | `docs/implementation/slice-13-commission-command-matrix.md` | implementation record | commission reads; blocked commands |
| S14 | `docs/implementation/slice-14-biawin-command-matrix.md` | implementation record | per-service scope; fulfillment and provider commands; blocked list |
| RE2 | `docs/implementation/rm-e02-role-navigation.md` | merged frontend record (PR #65); presentation only | destination → capability map; role → destination map (navigation, not authority) |
| RA1 | `docs/remediation/rm-access-01-authorization-contract.md` | merged remediation record (PR #66) | 43 = contract; availability vs authority; follow-ups |
| M2 | `docs/remediation/m2-authorization-decision-intake-fa.md` | DRAFT — NOT APPROVED | D-42 / D-34 / R5-SEC-01 intake; proposed records 1 and 2 |
| M2 | `docs/remediation/m2-d42-seller-authorization-fa.md` | DRAFT — NOT APPROVED | blocker analysis; scope requirements §5 |
| R5 | `docs/remediation/r5-decision-inventory.md` | inventory (validated by governance tests) | decision rows quoted |
| P-SEC | `docs/remediation/r5-person-identity-sod-DRAFT.md` | DRAFT — NOT APPROVED | §1 concepts; §2 approaches |
| R6 | `docs/governance/r6-decision-tracker.json` | tracker (statuses quoted, unchanged) | statuses in §4 below |
| R7 | `docs/audit/r7-consolidated-audit.md` | audit record | §3 role coverage; §5 register; §7 B roadmap; §8 |
| code | `internal/identity/api/api.go` | code at baseline | vocabulary (43), roles (9), bundles |
| code | `internal/identity/migrations/sql/00001_identity.sql` | code at baseline | role_capability seed (hr only); role_grant_shadow_only |
| code | `internal/authorization/authorization.go` | code at baseline | union evaluation; ScopeOrganization |
| code | `internal/integration/legacy/shadow/mapping.go` | code at baseline | legacy role → target role (SHADOW evidence only) |
| code | `api/openapi/components/hr.yaml` | contract at baseline | Capability enum (43) |

Not used as authority: Seller Design V1 and Supervisor design prototypes (visual only), RM-E02 screenshots and synthetic fixtures (presentation evidence), any product target treated as a runtime grant.

## 3. Capability → evidence

Enforcement points are the handler lines that call the guard for the capability. Matrix basis texts are in the matrix document §6.

| # | Capability | Enforcement (file:line) | Decisions | Cells (SEL/SUP/SNR/MGR/DEP/MIS/HR/FIN) |
|---|---|---|---|---|
| 1 | `identity.user.read` | `internal/identity/http/http.go:40,78` | D-42, D-34, R5-SEC-01 | X X X X X X A X |
| 2 | `org.person.read` | `internal/organization/http/http.go:154,200` | D-42, R5-SEC-01 | X C C C C C A C |
| 3 | `org.employment.read` | `internal/organization/http/http.go:106,177` | D-42, OPD-05 | X C C C C C A C |
| 4 | `org.position.read` | `internal/organization/http/http.go:113,178,241` | D-42 | X C C C C C A C |
| 5 | `org.unit.read` | `internal/organization/http/http.go:117,179,298` | D-42 | X C C C C C A C |
| 6 | `org.assignment.read` | `internal/organization/http/http.go:124,179,220,334` | D-42, OPD-05, SD-04 | X C C C C C A C |
| 7 | `customers.customer.read` | `internal/customers/http/http.go:124,160,191` | D-30, ADR #18, D-29 / D-31 | C C C C D C X C |
| 8 | `customers.merge_case.read` | `internal/customers/http/http.go:148,225,256` | D-30, OPD-01, ADR #18 | X X X X X D X X |
| 9 | `customers.identity_report.read` | `internal/customers/http/http.go:290` | D-30, OPD-01 | X X X X X D X X |
| 10 | `cases.case.read` | `internal/cases/http/http.go:106,145,181,213` | M2 proposed record 1 and 2 (no ID), OPD-01, ADR-01 | C C C C C C X C |
| 11 | `custody.custody.read` | `internal/custody/http/http.go:85,115` | OPD-10, F01, OPD-05 | X C C C C C D C |
| 12 | `sales.report.read` | `internal/cases/http/http.go:273; internal/custody/http/http.go:162` | F02, F03, OPD-07 | X X X X X A X X |
| 13 | `billing.invoice.read` | `internal/billing/http/http.go:154,185,219` | FRS (registry cites D-25, D-26, D-28), OPD-10, D-32 | C C C C C C X C |
| 14 | `payments.attempt.read` | `internal/billing/http/http.go:316; internal/payments/http/http.go:104` | FRS, D-25, D-44, OPD-10 | C C C C X X X C |
| 15 | `billing.reconciliation.read` | `internal/billing/http/http.go:390` | F04, TAD #19 | X X X X X A X A |
| 16 | `reporting.metrics.read` | `internal/reporting/http/http.go:284,321,357` | OPD-07, OPD-01, F02, F03, F04, F09 | D C C C C A X C |
| 17 | `reporting.mis.read` | `internal/reporting/http/http.go:700,769,889` | OPD-01, F09 | X X X X D A X C |
| 18 | `reporting.invoice_review.read` | `internal/reporting/http/http.go:971,1024` | D-23, D-24, D-45 | X X X X X X X D |
| 19 | `finance.shadow.read` | `internal/finance/http/http.go:286,317,349` | D-41, OPD-06, OPD-09 | X X X X X X X A |
| 20 | `finance.ledger.read` | `internal/finance/http/http.go:422,470` | D-41, OPD-06 | X X X X X X X A |
| 21 | `finance.reconciliation.read` | `internal/finance/http/http.go:534` | D-41, OPD-09 | X X X X X X X A |
| 22 | `biawin.subscription.read` | `internal/biawin/http/http.go:200,241,275` | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 | D D D D D D X D |
| 23 | `fulfillment.case.read` | `internal/fulfillment/http/http.go:190,230,264` | D-35a, D-35b, D-35c, D-35d, D-35e, OPD-02 | D D D D D D X D |
| 24 | `biawin.parity.read` | `internal/biawin/http/http.go:314` | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 | X X X X X D X D |
| 25 | `cases.contact_outcome.record` | `internal/cases/http/write.go:91,111,150` | M2 proposed record 1 (no ID), D-29 / D-31, ADR-01, OPD-01, OPD-10 | C D D D X X X X |
| 26 | `org.hr_request.create` | `internal/organization/http/hrrequest.go:115,120,206` | OPD-05, SD-04, R5-SEC-01 | X A A A C X D X |
| 27 | `org.hr_request.review` | `internal/organization/http/hrrequest.go:55,58,248` | OPD-05, SD-04, R5-SEC-01 | X A A A C X D X |
| 28 | `org.hr_request.read` | `internal/organization/http/hrrequest.go:71` | OPD-05, SD-04 | X A A A C X D X |
| 29 | `integration.writeback.read` | `internal/integration/writeback/http/http.go:44` | E-3, P1-8 (engineering gap), E-7 | X X X X X D X D |
| 30 | `payments.upload.stage` | `internal/payments/http/write.go:80` | D-44 | C D D D X X X X |
| 31 | `payments.batch.submit` | `internal/payments/http/write.go:176,196,247` | D-43, D-28, OPD-06, OPD-10, OPD-08 | C D D D X X X X |
| 32 | `payments.submission.read` | `internal/payments/http/write.go:143,303` | D-43, OPD-10 | C D D D N N N N |
| 33 | `payments.evidence.read` | `internal/payments/http/write.go:330` | D-44, OPD-08 | X X X X X X X C |
| 34 | `finance.review.read` | `internal/finance/http/review.go:56` | OPD-08, D-43, REJECT-VOCAB (R4 gap), R5-SEC-01 | X X X X X X X A |
| 35 | `commission.shadow.read` | `internal/commission/http/http.go:180,233,263` | OPD-09, D-36, D-41 | X X X X X X X A |
| 36 | `commission.policy.read` | `internal/commission/http/http.go:352` | OPD-09, D-36 | D X X X X X D A |
| 37 | `commission.own.read` | `internal/commission/http/http.go:430` | OPD-09, D-41 | A A A A A N N N |
| 38 | `fulfillment.work.read` | `internal/fulfillment/http/work.go:108,165` | D-35a, D-35b, D-35c, D-35d, D-35e, OPD-05, OPD-08 | D D D D D D X D |
| 39 | `fulfillment.work.assign` | `internal/fulfillment/http/work.go:323` | OPD-05, D-35a, D-35b, D-35c, D-35d, D-35e | D D D D D D X D |
| 40 | `fulfillment.work.execute` | `internal/fulfillment/http/work.go:348,373,398` | D-35a, D-35b, D-35c, D-35d, D-35e, D-38, OPD-04 | D D D D D D X D |
| 41 | `biawin.activation.read` | `internal/biawin/http/activation.go:31` | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 | D D D D D D X D |
| 42 | `integration.provider_operation.read` | `internal/integration/providerops/http/http.go:43` | D-38, D-35d | D D D D D D X D |
| 43 | `integration.provider_operation.reconcile` | `internal/integration/providerops/http/http.go:78` | D-38, D-35d, R5-SEC-01 | D D D D D D X D |

## 4. Decision references and tracker status (quoted; unchanged)

Every decision identifier cited in the RM-B02 documents, with its status in `docs/governance/r6-decision-tracker.json` at the baseline. Engineering prerequisites and findings that are not tracker records are marked as such.

| Identifier | Tracker record | Status | Verification | Cited in |
|---|---|---|---|---|
| ADR #18 | ADR #18 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| ADR-01 | ADR-01 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| D-23 | D-23 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| D-24 | D-24 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| D-25 | D-25 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix, separation-of-duties |
| D-26 | D-26 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix |
| D-28 | D-28 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix |
| D-29 | D-29 / D-31 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| D-30 | D-30 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix, scope-and-ownership, separation-of-duties |
| D-31 | D-29 / D-31 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| D-32 | D-32 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-33 | D-33 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| D-34 | D-34 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix, separation-of-duties |
| D-35a | D-35a | `DRAFT` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-35e | D-35e | `DRAFT` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-36 | D-36 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-38 | D-38 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-41 | D-41 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| D-42 | D-42 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix, scope-and-ownership, separation-of-duties |
| D-43 | D-43 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| D-44 | D-44 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| D-45 | D-45 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| E-3 | not a tracker record (R5 inventory: engineering prerequisite, finding or deferred item) | — | — | separation-of-duties |
| E-7 | not a tracker record (R5 inventory: engineering prerequisite, finding or deferred item) | — | — | d42-decision-package, executive-summary.fa, role-capability-matrix, scope-and-ownership, separation-of-duties |
| F01 | F01 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| F06 | not a tracker record (R5 inventory: engineering prerequisite, finding or deferred item) | — | — | separation-of-duties |
| F08 | not a tracker record (R5 inventory: engineering prerequisite, finding or deferred item) | — | — | role-capability-matrix |
| OPD-01 | OPD-01 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, separation-of-duties |
| OPD-02 | OPD-02 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| OPD-03 | OPD-03 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| OPD-04 | OPD-04 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| OPD-05 | OPD-05 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| OPD-06 | OPD-06 | `EVIDENCE_MISSING` | `NOT_STARTED` | role-capability-matrix, separation-of-duties |
| OPD-07 | OPD-07 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| OPD-08 | OPD-08 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| OPD-09 | OPD-09 | `EVIDENCE_MISSING` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| OPD-10 | OPD-10 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |
| R5-SEC-01 | R5-SEC-01 (proposed ADR) | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, executive-summary.fa, role-capability-matrix, scope-and-ownership, separation-of-duties |
| REJECT-VOCAB | REJECT-VOCAB (R4 gap) | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix |
| SD-01 | alias of OPD-10 (R5 inventory) | `READY_FOR_REVIEW` | `NOT_STARTED` | role-capability-matrix, separation-of-duties |
| SD-04 | SD-04 | `READY_FOR_REVIEW` | `NOT_STARTED` | d42-decision-package, role-capability-matrix, scope-and-ownership, separation-of-duties |

## 5. Proposed records without decision IDs

| Label | Origin | Status |
|---|---|---|
| M2 proposed record 1 (seller capability eligibility) | `m2-authorization-decision-intake-fa.md` §4-1 | PROPOSED — not in inventory, not in tracker |
| M2 proposed record 2 (seller case read scope) | same, §4-2 | PROPOSED — not in inventory, not in tracker |
| GAP-01 … GAP-08 | `rm-b02-d42-decision-package.md` §3 | PROPOSED labels of this package — **not decision IDs**; must not be used as IDs in code, tracker, approval records or PRs |
| MC-01 … MC-36 | `rm-b02-role-capability-matrix.md` §9 | requirement labels for missing capabilities — **not capability keys** |
| CF-01 … CF-11 | matrix §10 | conflict labels of this package |
| W01 … W11 | `rm-b02-separation-of-duties.md` §2 | workflow labels |

## 6. Verification performed for this document set

| Check | Result |
|---|---|
| Capability count at HEAD | 43 in `api.go` `vocabulary` (generator asserts order and count), 43 distinct keys in the catalog migrations, 43 in the OpenAPI enum |
| Matrix completeness | 8 × 43 = 344 cells, each with exactly one classification (generator assertion) |
| Every ALLOW / CONDITIONAL cell | has a scope and a basis text; every CONDITIONAL cell names at least one defined condition key (generator assertion) |
| Every decision identifier cited | present in `r5-decision-inventory.md` (enforced by `docs/governance` test `TestEveryDecisionIDIsInventoried`) |
| Tracker statuses | quoted from the tracker file by the generator; the tracker is unchanged |
| Source documents | none modified (`git diff --stat` touches only `docs/governance/proposals/`) |
