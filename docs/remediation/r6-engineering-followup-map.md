# R6 — Engineering follow-up map (decision → engineering)

**No decision is approved, so no follow-up work item exists today.** This document defines:
- how a work item is created once a verified approval exists;
- the decision-to-engineering dependency matrix;
- the separate acceptance criteria that close each P0 blocker.

## 1. Creating a follow-up work item

A work item is created only for a decision in `APPROVED_VERIFIED` (or `SUPERSEDED_VERIFIED` / `VOID_VERIFIED`
where the replacement needs work). The approval record is the input. The work item is a GitHub issue or a
`docs/remediation` entry with:

| Field | Content |
|---|---|
| Decision ID and version | From the approval record (exact) |
| Approval record | Path under `docs/governance/approvals/` + `immutable_ref` |
| Impacted modules | From the matrix below |
| Engineering owner | A named engineer, assigned by the engineering lead (role) at creation; not pre-assigned here |
| Implementation dependency | Prior decisions and engineering items (matrix "Prerequisites") |
| Required tests | Unit and integration tests of the decided rule; regression of every invariant the record lists; fail-closed tests for the remaining undecided paths |
| Runtime verification | Evidence in a target environment with real role accounts (HR / Finance are NOT LIVE VERIFIED today) |
| Operational owner | Named by the business owner (role) at creation |
| Rollback requirement | How to return to the previous safe default without losing history (flag off, command removal, data kept) |
| Activation approval | A separate verified approval to switch the feature on; never implied by the policy approval |

**An approved policy does not close a P0 blocker.** A P0 blocker closes only when all four criteria in §3 are
met.

## 2. Decision-to-engineering dependency matrix

Generated from the R5 records. "Implementation follow-up" and "Activation gate" are the R5 texts. Status is from
`docs/governance/r6-decision-tracker.json`.

| Decision ID | Meeting | Modules | Prerequisites | Blocker | Implementation follow-up | Activation gate | Status |
|---|---|---|---|---|---|---|---|
| ADR #18 | M1 | integration, all | — | P0-3 | E-4 real snapshot. | Separate per environment. | READY_FOR_REVIEW |
| R5-SEC-01 (proposed ADR) | M2 | identity, organization, authorization, finance, HR | D-34, D-42, ADR #18 | P0-7 (SoD), P0-6 (handover) | Person entity + link records + SoD evaluation; no grants. | Security activation + real account tests. | READY_FOR_REVIEW |
| D-34 | M2 | identity | R5-SEC-01 | P0-1 (provisioning part) | Provisioning command (credential write). | Security approval + real accounts. | READY_FOR_REVIEW |
| D-42 | M2 | organization, authorization, identity | R5-SEC-01, E-7 | P1-4 | Grant-change procedure. | Separate. | READY_FOR_REVIEW |
| D-45 | M2 | reporting, audit | — | P1 | — | — | READY_FOR_REVIEW |
| D-26 | M2 | identity, payments, portal | D-25 | P1-3 | Portal auth. | Separate. | READY_FOR_REVIEW |
| ADR-01 | M3 | cases, custody, billing | F01 (union-link contract), OPD-03 (release), OPD-10 (who may claim), D-32 (next-stage authority), P-SEC person identity (actor verification) | P0-5 | Claim / issue / return commands in owner transactions; owner-schema migrations; outbox / inbox wiring; legacy write-back; real-data parity. | Separate activation authority per command + legacy writer disabled for the same command (P1-8) + runtime evidence on real-shaped data. | READY_FOR_REVIEW |
| F01 | M3 | cases, custody, billing, payments, MIS | ADR #18 (real data), E-4 (real snapshot), OPD-01 (lineage) | P0-5 | Link adapters per proof source; coverage report; reconciliation-owner routing. | Per inventory writer, after ADR-01 and OPD-03. | EVIDENCE_MISSING |
| OPD-03 | M3 | cases, custody, billing, MIS | F01, OPD-04 (refund-reported), ADR-01 | P0-5 | Release / reset / revival commands using the release gate; reason taxonomy. | After ADR-01 + F01 + OPD-03; per command. | READY_FOR_REVIEW |
| OPD-10 | M3 | sales tiers, MIS delivery, billing, authorization | P-SEC person identity, D-42 (capability grants), ADR-01 | P0-5 | Delegation record + guards; attribution preservation tests. | Per path. | READY_FOR_REVIEW |
| D-28 | M3 | billing, cases, commission | OPD-09, D-41 | P0-5 / P0-7 | Plan rule at issuance. | With issuance. | EVIDENCE_MISSING |
| D-32 | M3 | billing, authorization | Billing lifecycle contract (engineering), D-42, P-SEC | P0-5 (billing authority) | Stage state machine + command. | After lifecycle contract + decision. | EVIDENCE_MISSING |
| D-29 / D-31 | M3 | cases, customers, MIS | D-30 | P1 (sales workflow) | Duplicate outcome command. | Per command. | READY_FOR_REVIEW |
| D-30 | M3 | customers, MIS, sales | ADR #18, E-4 | P1 | Reversible merge records. | After real-data report. | EVIDENCE_MISSING |
| OPD-08 | M4-A | finance, payments, authorization | P-SEC (person identity, capability model), D-43, rejection vocabulary | P0-7 | Decision capability; approve / reject commands over the R4 contract; rejection vocabulary; bulk with per-item results. | Finance approval of activation + real role-account tests. | READY_FOR_REVIEW |
| D-43 | M4-A | payments, finance | OPD-08 | P0-7 | — | With OPD-08. | READY_FOR_REVIEW |
| D-25 | M4-A | payments, finance | D-26 (no auto-login), OPD-08 | P0-7 (gateway path) | Real gateway adapter + callback route. | Separate. | READY_FOR_REVIEW |
| D-44 | M4-A | payments, object store, backup | ADR #18 (privacy) | P1-5 | Purge mode enablement; retention planner deletion (R3 dry-run only). | Separate operator authority. | READY_FOR_REVIEW |
| REJECT-VOCAB (R4 gap) | M4-A | finance, billing | OPD-08 | P0-7 | Vocabulary registry; transition. | With OPD-08. | READY_FOR_REVIEW |
| D-41 | M4-B | finance | OPD-06, ADR #18, E-4 | P0-7 | Posting authority over the R4 contract; reconciliation. | Finance activation authority + parity. | EVIDENCE_MISSING |
| OPD-06 | M4-B | billing, payments, finance, reporting | ADR #18, E-4 | P0-7 | Money contract extension. | With D-41. | EVIDENCE_MISSING |
| D-37 | M4-B | finance, catalog, biawin | D-35b (§14 labelling; C-03), D-41 | P1 | Account model. | Separate. | READY_FOR_REVIEW |
| OPD-09 | M4-C | commission, finance, HR, sales | D-36, D-41, ADR #18 | P0-7 | Posting engine; legacy engines disabled first. | Separate Finance activation. | EVIDENCE_MISSING |
| D-36 | M4-C | commission | OPD-09 | P0-7 | Allocation function. | With OPD-09. | EVIDENCE_MISSING |
| OPD-04 | M4-D | finance, payments, billing, complaint (after-sales) | P-SEC, OPD-08, D-41 (posting), D-44 (evidence) | P0-7 | Refund commands over the R4 state machine; proof storage; reconciliation. | Separate refund activation authority. | READY_FOR_REVIEW |
| OPD-05 | M5 | organization, identity, custody, billing, finance | SD-04, R5-SEC-01 (person identity), OPD-08 (pending approvals) | P0-6 | Final-apply command with handover; access revocation. | HR + Security activation. | READY_FOR_REVIEW |
| SD-04 | M5 | organization, sales | OPD-05 | P0-6 | Transfer command. | With OPD-05. | READY_FOR_REVIEW |
| D-35a | M6 | catalog, biawin, finance | C-03 mapping confirmation, D-41, D-37 | P1-2 | Entitlement model. | Separate. | DRAFT |
| D-35b | M6 | finance, biawin | C-03 mapping confirmation, D-37 | P1-2 | — | — | DRAFT |
| D-35c | M6 | catalog, biawin | C-03 mapping confirmation, D-41 | P1-2 | Lifecycle commands. | Separate. | DRAFT |
| D-35d | M6 | biawin, integration | C-03 mapping confirmation, D-38 | P1-2 | Provider adapter. | Separate. | DRAFT |
| D-35e | M6 | biawin, sales | C-03 mapping confirmation | P1-2 | — | — | DRAFT |
| D-38 | M6 | biawin, finance | D-41, D-35d (§14 labelling; C-03) | P1-2 | Provider operation. | Separate. | READY_FOR_REVIEW |
| D-33 | M6 | payments, fulfillment | OPD-08 | P1 | — | — | READY_FOR_REVIEW |
| D-39 | M6 | catalog, authorization | — | P1 | — | — | READY_FOR_REVIEW |
| OPD-07 | M6 | reporting, sales, MIS | OPD-01 | P1-7 | Metric contracts. | — | EVIDENCE_MISSING |
| D-23 | M6 | reporting | ADR #18, D-45 | P1-7 | — | — | READY_FOR_REVIEW |
| D-24 | M6 | reporting UI | — | P1-7 | — | — | READY_FOR_REVIEW |
| OPD-01 | M6 | all | ADR #18 | P0-3 / P1-7 | — | — | EVIDENCE_MISSING |
| OPD-02 | M6 | cross-domain | — | P2 | — | — | READY_FOR_REVIEW |
| D-40 | M6 | sales, finance | — | P1 | — | — | READY_FOR_REVIEW |

D-03 … D-22 (P-GOV): no engineering follow-up until a verified decision records, supersedes or voids each ID.

## 3. P0 blocker closure criteria (all four required)

| Blocker | Policy (verified decisions) | Technical | Operational | Runtime | Status after R6 |
|---|---|---|---|---|---|
| P0-5 inventory safety | ADR-01, F01, OPD-03, OPD-10, D-32, D-28 | Claim / issue / return commands per the decisions; P1-8 legacy-writer disable | Named operational owner; rollback rehearsed | Real-data parity (ADR #18); race evidence in the target environment | **PARTIAL**: R4 evaluators and synthetic race evidence; 0 decisions |
| P0-6 HR handover | OPD-05, SD-04, R5-SEC-01 | Final-apply command with successor rules | HR operating procedure | HR real-role-account tests | **OPEN** |
| P0-7 financial execution | OPD-08, D-43, REJECT-VOCAB, D-41, OPD-06, OPD-09, D-36, OPD-04, R5-SEC-01 | Approve / reject / post / refund commands; person-level SoD | Finance operating procedure; activation authority | Real-data ledger parity; Finance real-role-account tests | **PARTIAL / BLOCKED**: contracts only; 0 decisions |
| P0-3 real-data reconciliation | ADR #18 | Extraction and reconciliation tooling (E-4) | Data owner, retention and destruction | Reconciliation on real data | **OPEN** |

Production remains **NO-GO**.
