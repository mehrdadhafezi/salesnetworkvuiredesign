# R6 — Human approval workflow (documentation and offline validation only)

This document defines how a decision moves from a draft to a verified outcome. It creates **no live system and
no new approval authority**. Every authority named here is a *role*. Who holds each role, and what proves it,
is itself a management question (M1 §C, `r6-m1-governance-data-session-fa.md`).

**Approvals made by R6: 0. Production remains NO-GO.**

## 1. What never changes a status

The following events change no lifecycle status, alone or together:
- generating or editing a document;
- scheduling or holding a meeting;
- passing tests or green CI;
- a GitHub PR approval;
- merging a PR;
- code behaviour or a safe default;
- an engineering recommendation;
- the repository owner's account acting alone (rule 8: owning the repository does not prove organizational
  authority for every domain).

## 2. States

The machine-readable status of each decision is in `docs/governance/r6-decision-tracker.json`.
`r6_test.go` enforces the rules in §4.

| State | Meaning | Today |
|---|---|---|
| `EVIDENCE_MISSING` | Facts needed to decide do not exist (R5 "needs further evidence", D-03…D-22) | 30 |
| `DRAFT` | Record exists; a prerequisite (conflict resolution or prior decision) prevents review (D-35a–e, blocked by C-03) | 5 |
| `READY_FOR_REVIEW` | Question, options, evidence and approver roles are documented (R5 "ready for human review") | 26 |
| `UNDER_HUMAN_REVIEW` | The authorized approvers have formally opened the review | 0 |
| `DECISION_RECORDED_PENDING_VERIFICATION` | An approval record exists and passes the structural check; human verification not done | 0 |
| `APPROVED_VERIFIED` | Approval record + independent human verification of authority and evidence | 0 |
| `REJECTED_VERIFIED` | All options rejected by the authorized approvers, verified | 0 |
| `DEFERRED` | Authorized approvers postponed the decision, with a recorded reason and revisit condition | 0 |
| `SUPERSEDED_VERIFIED` | Replaced by an explicitly identified, verified decision | 0 |
| `VOID_VERIFIED` | Declared void by a verified decision (P-GOV option 3) | 0 |

`DEFERRED` is a decision of the authorized approvers. "Nobody decided yet" is not `DEFERRED`.

## 3. Transitions

**Common audit record.** Every transition is a reviewed commit to `main` that changes:
- the decision's entry in `r6-decision-tracker.json`;
- for every state from `DECISION_RECORDED_PENDING_VERIFICATION` on, the approval record in
  `docs/governance/approvals/`.

The history is append-only. A correction is a new commit, never a rewrite.

| # | From → To | Required evidence | Authorized actor (role) | Independent approvers | Validation conditions | Possible conflict | Recovery / correction |
|---|---|---|---|---|---|---|---|
| T1 | `EVIDENCE_MISSING` → `READY_FOR_REVIEW` | The missing evidence listed in the record, committed with path / commit (evidence index entry) | Engineering or evidence owner prepares; business owner confirms sufficiency | — | Evidence item passes the R5 provenance test; record updated; no approval field touched | Evidence contradicts a frozen source → new conflict entry | Revert the status; record the gap |
| T2 | `DRAFT` → `READY_FOR_REVIEW` | Resolution of the blocking prerequisite (for D-35a–e: Product + Finance confirmation of the C-03 mapping as a verified record) | Owners of the prerequisite | Product **and** Finance for C-03 | Prerequisite record is `APPROVED_VERIFIED` | Prerequisite decision changes the question | Re-draft the record (new version) |
| T3 | `READY_FOR_REVIEW` → `UNDER_HUMAN_REVIEW` | A `REVIEW_NOTICE` record: a written notice from an authorized approver that the review is open (minutes or a dated message), with the opener's authority evidence; no decision content | Any required approver | — | Reference is resolvable; approver role matches the record | Approver's authority disputed | Return to `READY_FOR_REVIEW` with a note |
| T4 | `UNDER_HUMAN_REVIEW` → `DECISION_RECORDED_PENDING_VERIFICATION` | A `DECISION` record (all fields of `r6-approval-evidence-standard.md` §2) | Recorder (any role); the record names the approvers | Every role listed as required, each signing separately | Offline structural check passes (`validateApprovalRecord`); selected option is one of the record's options or explicit new text | Signatures from roles other than the required ones; missing independent sign-off | Record stays pending; recorder supplies the missing evidence |
| T5 | `DECISION_RECORDED_PENDING_VERIFICATION` → `APPROVED_VERIFIED` / `REJECTED_VERIFIED` / `SUPERSEDED_VERIFIED` / `VOID_VERIFIED` | Human verification entry: verifier, role, method (e.g. confirmation with the approvers through an independent channel, check of the authority document), date, result | Verifier who is **not** one of the approvers and not the recorder | — | `human_verification.result = VERIFIED`; verifier ∉ approvers; verifier ≠ automation | Verifier finds authority or evidence insufficient | Status stays pending; result `REJECTED` or `INCONCLUSIVE` is recorded |
| T6 | `READY_FOR_REVIEW` or `UNDER_HUMAN_REVIEW` → `DEFERRED` | A `DEFERRAL` record by an authorized approver (`outcome: DEFERRED`, reason, revisit condition; no selected option) | Required approver | — | Reason and revisit condition non-empty | — | Back to `READY_FOR_REVIEW` when the condition is met |
| T7 | Any verified state → new version | A new decision record (v2.0…) referencing the earlier version | As T4–T5 | As T4–T5 | The earlier record stays; tracker points to the newest version | Conflicting concurrent versions | Both recorded; a conflict entry; neither treated as current until resolved |
| T8 | Any state → previous state (correction) | Evidence that a recorded transition was wrong (forged reference, missing authority) | Verifier or Security | Security for suspected forgery | Correction commit explains the reason | — | The wrong record is kept and marked invalid; never deleted |

## 4. Offline validation (what the repository can and cannot check)

`docs/governance/r6_test.go` checks:
- every status is one of the ten states;
- a status from `UNDER_HUMAN_REVIEW` on requires an approval record path that exists and passes the structural
  check;
- the record type matches the status (`REVIEW_NOTICE` → `UNDER_HUMAN_REVIEW`, `DEFERRAL` → `DEFERRED`, `DECISION`
  → recorded / verified);
- a `*_VERIFIED` status requires a human verification entry by a verifier who is not an approver, and must equal
  the recorded outcome (a `REJECTED` decision can never back `APPROVED_VERIFIED`);
- outcome fields are empty when no approval record exists;
- tracker statuses match R5 readiness for decisions without an approval record.

The repository **cannot** check:
- that a named person exists, holds the stated role, or signed;
- that an authority document is genuine.

Those are human verification steps (T5). Automation never promotes a decision to `APPROVED_VERIFIED`.

## 5. After approval

An approved policy creates follow-up engineering work (`r6-engineering-followup-map.md`). It does not
implement, activate or close a P0 blocker by itself.
