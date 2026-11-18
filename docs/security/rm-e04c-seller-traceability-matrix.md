# RM-E04C — Seller Security Traceability Matrix

**DRAFT — NOT APPROVED. Every test below is a future test; none has run.** Baseline `main` @ `c0d7f4b`. Production **NO-GO**.

Chain: **screen → operation → capability → ownership policy (basis) → security requirement → test → decision gate → PR.** IDs: SR-01…SR-16 from [`../implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md) §3; NT-01…NT-24 from [`rm-e04a-seller-negative-test-plan.md`](rm-e04a-seller-negative-test-plan.md); NT-25…NT-32 defined in §3 below; PR ids from [`../implementation/rm-e04c-seller-implementation-roadmap.md`](../implementation/rm-e04c-seller-implementation-roadmap.md). The consistency of this file is checked by `docs/governance/rm_e04c_test.go`.

## 1. Operation traceability

| Screen | Operation (proposed unless marked) | Capability (candidate) | Basis | SR | Tests | Decision gates | PR |
|---|---|---|---|---|---|---|---|
| S1 | `listMyCases` | `cases.own.read` | `CUSTODY_CURRENT_HOLDER` | SR-01, 02, 03, 04, 05, 07, 08, 11, 12, 13, 14, 16 | NT-01, NT-04, NT-05, NT-06, NT-08, NT-09, NT-10, NT-16, NT-17, NT-18, NT-21, NT-24, NT-25, NT-26, NT-27, NT-28 | M2 rec. 1, M2 rec. 2 | B4 |
| S1 | `getMyCase` | `cases.own.read` | same | SR-01, 05, 06 | NT-03, NT-09, NT-28 | M2 rec. 1, M2 rec. 2 | B4 |
| S1 | `listMyCaseHistory` | `cases.own.read` | same | SR-06, 09, 10, 14 | NT-03, NT-28 | M2 rec. 1, M2 rec. 2, RM-E04A-Q4 | B4 |
| S1 | `getMyCaseContactOutcomes` | `cases.own.read` | same | SR-06, 14 | NT-03, NT-19 | M2 rec. 1, M2 rec. 2 | B4 |
| S2 | `listMyCaseAttribution` | `cases.own.read` | parent case | SR-06, 09, 10 | NT-03, NT-29 | M2 rec. 2, RM-E04A-Q3, RM-E04A-Q4 | B4 |
| S4 | `listMyInvoices` | `billing.own.read` | `INVOICE_SELLER_AND_ISSUER` | SR-01, 02, 03, 04, 05, 07, 08, 12, 13, 14 | NT-01, NT-04, NT-05, NT-06, NT-08, NT-11, NT-13, NT-17, NT-18, NT-21, NT-24, NT-25, NT-26, NT-27, NT-28, NT-30 | M2 rec. 1, M2 rec. 2 (RM-E04A-Q2), OPD-10 | B5 |
| S4 | `getMyInvoice`, `listMyInvoiceStages`, `getMyInvoiceTotals` | `billing.own.read` | same | SR-05, 06, 14 | NT-03, NT-11, NT-15, NT-28 | M2 rec. 1, M2 rec. 2, GAP-03 | B5 |
| S4 | `listMyInvoicePayments` | `payments.own.read` | inherits invoice | SR-06, 14 | NT-03, NT-11 | M2 rec. 1, M2 rec. 2 | B5 |
| S4 | `getPaymentSubmission` (**exists**, owner-only) | `payments.submission.read` | `INVOICE_SELLER_AND_ISSUER` (exists) | SR-06 | NT-03 (done in A3) | M2 rec. 1 | — |
| S7 | `listMyCommissionEarnings` (**exists**) | `commission.own.read` (exists) | `COMMISSION_HOLDER` | SR-04, 07, 12, 14 | NT-04, NT-12, NT-17, NT-18, NT-21, NT-23, NT-25, NT-31 | M2 rec. 1; D-41, OPD-09, D-36 (semantics) | B3 |
| S2 list, S3, S5, S6 | none (no server source) | — | — | — | NT-22 | workflow contracts; OPD-10; GAP-04 | — |

## 2. Decision → artefact traceability

| Decision | Status (tracker, quoted) | Blocks | Artefacts |
|---|---|---|---|
| M2 proposed record 1 (no ID) | not registered | B2–B5 | ADR §4.2; API §2 |
| M2 proposed record 2 (no ID) | not registered | B4, B5 | ADR §5; screen designs S1 / S4 |
| D-42 | `READY_FOR_REVIEW` | C1 | roadmap C1 |
| D-34 | `READY_FOR_REVIEW` | C1, C3 | roadmap C1 |
| R5-SEC-01 (proposed ADR) | `READY_FOR_REVIEW` | C1 (person binding), every SoD-sensitive action | ADR §12 |
| ADR #18 | `READY_FOR_REVIEW` | D1 | roadmap D1 |
| D-41 / OPD-09 / D-36 | `EVIDENCE_MISSING` | S7 semantics | ADR §13 |
| OPD-10 | `READY_FOR_REVIEW` | S3, S6, S4 basis revisit | ADR §14 |
| D-45 | `READY_FOR_REVIEW` | any customer-contact read | ADR §7 |
| GAP-04 (proposed record, no ID) / D-30 | not registered / `EVIDENCE_MISSING` | customer name / phone; phone search | ADR §7–§8 |

## 3. New tests NT-25 … NT-32 (future)

Fixture: the NT §0 fixture (Seller X, Seller Z, row Y owned by Z, row U with UNKNOWN ownership, synthetic data on the R7 harness).

| ID | Invariant | Setup | Expected |
|---|---|---|---|
| **NT-25** | Scope class separation (ADR §4.1) | Inject an allowed decision with scope `ORGANIZATION` into every `/me` handler, and `OWN` into every organisation handler | 403 `ROW_SCOPE_DENIED`, no module query ran (recording runner) |
| **NT-26** | Cursor binding | (a) cursor from `/cases` replayed on `/me/cases`; (b) cursor from `/me/invoices` on `/me/cases`; (c) X replays a cursor issued to Z | (a)(b) 400 `CONTRACT_INVALID`; (c) only X's rows |
| **NT-27** | Multi-role cannot widen own reads | X holds `cases.own.read` (Seller) and `cases.case.read` (another role); then X holds only `cases.own.read` | `/me/cases` returns only X's rows in both cases; `/cases` answers only with `cases.case.read`, else 403 |
| **NT-28** | Field minimisation | Any `/me` response | no `legacy_key`, no other user's `user_id`, no customer name / phone; withheld fields absent with `FIELD_DENIED` |
| **NT-29** | Facets not collapsed | Case where original seller = Z, holder = X, credit owner UNKNOWN | facets returned separately; Z's facet `is_caller=false` with no identity; UNKNOWN facet `is_caller=null` |
| **NT-30** | Assisted-issuance limitation is explicit | Invoice seller = X, issuer = Z (basis `INVOICE_SELLER_AND_ISSUER`) | invisible to X and Z; not counted; not reported as "excluded" |
| **NT-31** | Wallet semantics not invented | `/commission/me/earnings` and the wired page | no confirmed / settleable / payable field; totals labelled historical; units never summed |
| **NT-32** | Revocation mid-pagination | X pages `/me/cases`; grant revoked; X sends `next_cursor` | 403 `ACTION_DENIED` before the cursor is decoded; no row |

## 4. Requirement coverage

| SR | Covered by |
|---|---|
| SR-01 server-side predicate | NT-01, NT-08 |
| SR-02 scope from the decision | NT-10, NT-25 |
| SR-03 exhaustive scope handling | NT-07 (done, A1), NT-25 |
| SR-04 no proven identity, no rows | NT-04 |
| SR-05 unknown never widens | NT-09, NT-11, NT-30 |
| SR-06 non-enumeration | NT-03 |
| SR-07 counts after scope | NT-08 |
| SR-08 search inside scope | NT-05, NT-14 |
| SR-09 facets not collapsed | NT-29 |
| SR-10 history not rewritten | NT-29, NT-16 |
| SR-11 freshness | NT-16, NT-32 |
| SR-12 no caching | NT-17 (done, A2) |
| SR-13 multi-role union never widens | NT-18, NT-27 |
| SR-14 honest outcomes | NT-15, NT-31 |
| SR-15 sensitive-read audit | NT-14 (with D-45) |
| SR-16 fail closed on infrastructure errors | NT-24 |
