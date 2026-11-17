# RM-E04A — Seller Read Access: Decisions Kept Separate

**DRAFT — NOT APPROVED. No option is selected. No decision ID is created. No tracker status changes.** Statuses below are quoted from `docs/governance/r6-decision-tracker.json` at `main` @ `e5fabbe`. Recording any outcome follows `docs/remediation/r6-approval-evidence-standard.md` and `r6-human-approval-workflow.md`.

Completing the RM-E04A engineering contract (this series of documents) **approves nothing**. It only records what each decision must settle before the Seller can read real data.

## 1. The six decisions are distinct

| Record | Decides | Does **not** decide | Tracker status (quoted) | What stays blocked for Seller reads |
|---|---|---|---|---|
| **D-42** | **Grant administration authority**: who may grant, edit and revoke capabilities (Identity / Security only · HR requests + Security approves · HR within a capped set) | which capabilities the Seller is eligible for; which rows the Seller sees | `READY_FOR_REVIEW`, verification `NOT_STARTED` | any ACTIVE grant being *issued* through a production path |
| **M2 proposed record 1** (no ID) | **Seller capability eligibility**: which capabilities the `seller` role bundle contains (A: `cases.case.read` + `cases.contact_outcome.record`; B: read only; C: none yet) | who grants; read scope | proposed — not in inventory, not in tracker | which Seller reads may exist at all; any `*.own.read` capability under Option A |
| **M2 proposed record 2** (no ID) | **Seller read-scope policy**: which rows a Seller sees on every data-returning endpoint (1 current holder · 2 current legacy assignment · 3 current + historical · 4 organisation · 5 later) | eligibility; grant authority | proposed — not in inventory, not in tracker | every predicate in `docs/security/rm-e04a-seller-row-scope.md` |
| **D-34** | **Account provisioning** (one-time password + forced change · invitation link · external IdP) | grants; scope | `READY_FOR_REVIEW`, `NOT_STARTED` | a real Seller login |
| **R5-SEC-01** (proposed ADR) | **Person identity and SoD** at person level | grants; scope | `READY_FOR_REVIEW`, `NOT_STARTED` | person-level guarantees; any SoD-sensitive action (none is enabled by reads) |
| **ADR #18** | **Real-data authority** (synthetic only · anonymised · restricted full extract) | grants; scope | `READY_FOR_REVIEW`, `NOT_STARTED` | any test or use on real data; HR / Finance live verification |

Order of dependence (none implies another): a Seller reading real rows in production needs **all six**, plus a separate production activation authorization (execution plan Group D).

## 2. Precision corrections

| Where | Statement | Correction (documentation only; history preserved) |
|---|---|---|
| `docs/implementation/rm-e03-seller-panel.md` §11 | "Row scope … is **BLOCKED** on D-42 (Seller authorization) and the RM-B02 role-capability proposal" | Row scope is blocked on **M2 proposed record 2** (read-scope policy) and, for the capability itself, **M2 proposed record 1**. D-42 decides grant administration authority only. RM-E03 is a merged record and is not edited; this table is the correction of record. |
| `docs/implementation/rm-e03-seller-panel.md` §12 | "once D-42 and RM-B02 land" | Wiring needs M2 rec. 1, M2 rec. 2, D-42 (to issue a grant), D-34 (account) and ADR #18 (real data), each separately. |
| `docs/implementation/rm-e04a-seller-scoped-read-contract.md` (first version, PR #69) | "Those are D-42, M2 proposed record 1 (eligibility) and M2 proposed record 2 (read scope)" listed after three questions in a different order | Rewritten in that document to map each question to its record explicitly. |

## 3. Questions RM-E04A adds (labels only, no IDs; owners to be confirmed under M1)

These fall **inside** M2 rec. 1 / rec. 2 or existing gaps. They are not new decision records.

| Label | Question | Belongs to | Options (none selected) |
|---|---|---|---|
| RM-E04A-Q1 | Option A (own-scope `/me/…` operations with new capabilities) vs Option B (narrower scope on existing capabilities) | M2 rec. 1 (capabilities) + engineering ADR | A · B · defer |
| RM-E04A-Q2 | S4 invoice predicate: seller · issuer · both · linked case holder | M2 rec. 2 | a · b · c · d |
| RM-E04A-Q3 | Which attribution facet admits a case to S2 (current custody · original seller · credit owner) | M2 rec. 2 + conversion workflow contract | — (contract missing) |
| RM-E04A-Q4 | Whether other staff identities (history actors, approvers, other facet owners) are shown to a Seller | GAP-04-style field policy | show · reference only · withhold |
| RM-E04A-Q5 | Whether an own commission entitlement admits its invoice into S4 | M2 rec. 2 + OPD-09 | yes · no |
| RM-E04A-Q6 | Customer fields (name, phone) visible on S1 / S4 and phone search | GAP-04, D-30, D-45 | per GAP-04 options |

## 4. What may proceed without any of these decisions

Execution plan **Group A** only: scope-assertion hardening (F-02), `no-store` (F-07), 404 parity for the submission context (F-09), and the inventory guard test. They change no authority, add no capability and need no approval. Everything else waits.
