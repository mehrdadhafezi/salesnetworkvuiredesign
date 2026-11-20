# RM-E04L — Seller own-case owner preference: engineering handoff

**DRAFT — NOT APPROVED.** Owner-preference handoff only. Not an authorization decision, not an R6 record, not a gate
change. Production **NO-GO**.

| | |
|---|---|
| Baseline | `main` @ `a8f3c4df9215f4496f2547e89d950403f485ead3` (PR #84, RM-E04K, merged) |
| Owner preference 1 | `M2_REC_1 = OWNER_PREFERRED_READ_ONLY_OWN_CASES` → candidate `cases.own.read` only (Record 1 option B) |
| Owner preference 2 | `M2_REC_2 = OWNER_PREFERRED_CUSTODY_CURRENT_HOLDER` → candidate S1 basis `CUSTODY_CURRENT_HOLDER` (engineering mapping P2) |
| Governance state | `M2_REC_1 = NOT_APPROVED_VERIFIED` · `M2_REC_2 = NOT_APPROVED_VERIFIED` (neither registered) · `R5-SEC-01 = PREFERRED_A / NOT_APPROVED` · D-42, D-34 `NOT_APPROVED` · Group B, S4, real grants **BLOCKED** |

The preferences and their normalization live in the existing forms, not here:
Record 1 §13 ([`rm-e04e-m2-record-1-seller-capability-form.md`](../governance/proposals/rm-e04e-m2-record-1-seller-capability-form.md)),
Record 2 §12 ([`rm-e04e-m2-record-2-seller-row-scope-form.md`](../governance/proposals/rm-e04e-m2-record-2-seller-row-scope-form.md)).
Registration and approval procedure: [`rm-e04e-authority-and-registration-packet.md`](../governance/proposals/rm-e04e-authority-and-registration-packet.md) §3–§4;
R6 workflow: [`../remediation/r6-human-approval-workflow.md`](../remediation/r6-human-approval-workflow.md).
Build order: [`rm-e04c-seller-implementation-roadmap.md`](rm-e04c-seller-implementation-roadmap.md) (B1, B2a, B2, B4).

## 1. Blocker: the preferences cannot be registered today

The repository has a valid registration workflow (packet §4, Step R1–R10), but it may be executed only by the
**authorized inventory owner**, whose holder is **UNASSIGNED** (packet §1.2 Q-A/Q-B). The Product, Sales and Security
signer roles, the recorder and the T5 verifier are also UNASSIGNED. Therefore no decision ID, tracker entry,
approval record, approver, date or verification is created in this PR. The owner's instruction is not authority
evidence for any of these roles (packet §1.2, last paragraph).

## 2. Dependency chain to the first authorized Seller read

| # | Step | Governing dependency | Permitted now? | Blocking gate | Evidence to move forward |
|---|---|---|---|---|---|
| 0 | Owner preference documented | this PR | **Yes — done here** | — | Record 1 §13, Record 2 §12 |
| 1 | M1 authority: name the inventory owner, the Product / Sales / Security signers, the recorder and a T5 verifier; decide whether Legal signs Record 2 (PQ-4) | packet §1.3–§1.4 | No (human) | packet §1.2 "registration cannot legally proceed" | committed authority evidence per person (packet §1.3) |
| 2 | Register Records 1 and 2 (R1–R10) | packet §3.6, §4 Step R | No — needs step 1 | `TestEveryDecisionIDIsInventoried`, `TestR6TrackerAccountsForEveryR5Record`, `TestR6PersianWorksheets` | inventory, evidence index, tracker (`READY_FOR_REVIEW`, empty outcome), worksheets; suite green |
| 3 | Codex review: `cases.own.read` naming, `OWN` scope model, P1/P2/P3 choice | AGENTS.md *Agent Ownership*; ADR | Review may start; sign-off **PENDING** | ADR status (`TestRME04CDocumentsAreMarkedNotApproved`) | recorded Codex review |
| 4 | M2 session decides: Record 1 option, Record 2 S1 predicate, answers to PQ-1/PQ-4/PQ-6, and conditions for pending transfer, cross-organization isolation and staleness window (Record 2 §12) | R6 workflow T3–T4 | No — needs step 2 | R6 lifecycle | `approvals/<ID>-v1.0-DECISION.json` per record, named approvers with authority evidence |
| 5 | T5 independent verification → both `APPROVED_VERIFIED` | R6 T5; evidence standard §3 | No — needs step 4 | `TestR6NoStatusOrOutcomeWithoutApprovalRecord` | `human_verification` by a non-approver, non-recorder human |
| 6 | **B-PR1 / B1**: open the Group B gate (both IDs, exactly `cases.own.read`, the approved `/me/cases*` operation ids, no S4) and accept the ADR | packet §4 Step G; roadmap B1 | No — needs step 5 + step 3 | `rm-e04c-group-b-gate.json` + `TestRME04CProposalsStayOutOfRuntime` | reviewed gate commit citing the approval records; ADR → ACCEPTED |
| 7 | **B2a**: `ScopeOwn`, scope class on the capability vocabulary, `AccessDecision.scope` enum contract change, `/me` guard helper | roadmap B2a | No — needs step 6 | Group B gate | B2a compatibility suite, NT-25; security review + Codex contract sign-off |
| 8 | **B2**: register `cases.own.read` (registry, catalog migration, OpenAPI enum, `capabilityFlags`); `seller` bundle in code = exactly the approved set | roadmap B2 | No — needs step 7 | Group B gate; `TestL_ContractEnumEqualsRegistry` | no ACTIVE grant possible in any non-test DB; NT-18, NT-27 |
| 9 | **B4**: S1 own reads `listMyCases`, `getMyCase`, `listMyCaseHistory`, `getMyCaseContactOutcomes` with the approved predicate; `listMyCaseAttribution` only if S2 scope was approved | roadmap B4; API proposal; SR-01…SR-16 | No — needs step 8 | Group B gate | S1 rows of the traceability matrix on the R7 harness with synthetic grants; org endpoints unchanged |
| 10 | Frontend transport: an `S1ReadSource` implementation backed by `/me/cases*`, replacing the synthetic source in the DEV preview only | RM-E04K boundary (`web/src/modules/seller/data/s1ReadSource.ts`) | No — needs step 9 | same | existing E04K lifecycle tests against the real transport; fail-closed latch unchanged |
| 11 | Real grants, real accounts, real data, production | Group C (D-42 + D-34, + R5-SEC-01 if person binding is required), Group D (ADR #18), Group E | No | roadmap groups C–E | separate approvals; production authorization |

B-PR2 in the RM-E04D grouping combines steps 7–8; B-PR3 is step 9 (B3 wallet is **not** in scope while
`commission.own.read` stays deferred under the preference).

**First executable code increment after valid approval:** B2a (step 7). It changes no Seller data access by itself
and is the prerequisite that makes an `OWN` decision representable. The first Seller read PR is B4 (step 9).

## 3. Kept separate

- **M2 Record 1** — which capabilities the role is eligible for.
- **M2 Record 2** — which rows "own" means.
- **D-42** — who may grant, edit and revoke. D-42 approval alone authorizes no Seller read.
- **D-34** — how accounts are provisioned.
- **R5-SEC-01** — person identity / SoD; own-read scope is per account (F-05) and does not need it.

## 4. What this PR does not change

Group B gate file, R6 tracker, S4 manifest, authorization evaluator, scope enums, capability registry, Seller grants,
`seller` bundle, OpenAPI, case SQL, identity / custody data, authentication, production navigation. A feature flag
set to off is not authorization for any blocked step above.
