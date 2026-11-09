# R6 — Approval evidence standard (offline validation contract)

This standard defines what an approval record must contain before a decision can be treated as recorded,
and what a human must verify before it can be treated as approved. It approves nothing. **No approval record
exists today**, and the directory `docs/governance/approvals/` is intentionally absent.

## 1. Principles

- An automated check can confirm only that a record is **structurally complete**. It cannot:
  - authenticate a human signature;
  - prove that a person holds a role;
  - prove that a role carries the authority claimed.
- A signature image or a typed name alone does not prove authority. Each approver needs separate authority
  evidence, such as an appointment letter, a delegation-of-authority document, or a board / director
  resolution, each with a reference.
- A record that passes the structural check is `DECISION_RECORDED_PENDING_VERIFICATION`, never more. Only a
  human verification entry (§3) can lead to a `*_VERIFIED` state.
- The repository owner's account is not proof of authority for organizational decisions.

## 2. Record types (one JSON file per decision, version and type)

Path: `docs/governance/approvals/<decision-id>-v<version>-<record_type>.json`. In the decision ID, characters
other than letters, digits and `-` become `_`. Separate files per type and version mean that no record
overwrites another. A committed file may later gain only a human verification entry. It is never deleted or
renamed (§4).

| `record_type` | Workflow step | Template | Can justify tracker status | Required | Forbidden |
|---|---|---|---|---|---|
| `REVIEW_NOTICE` | T3 | `REVIEW-NOTICE-TEMPLATE.json` | `UNDER_HUMAN_REVIEW` only | decision ID + version, `notice_date`, `initiator` (name, role, authority evidence), notice reference (`evidence_source`), audit reference (`immutable_ref`), recorder | outcome, option, decision text, decision / effective date, approvers, conditions, supersedes, human verification, deferral fields |
| `DEFERRAL` | T6 | `DEFERRAL-RECORD-TEMPLATE.json` | `DEFERRED` only (never closed, never actionable) | decision ID + version, `outcome: DEFERRED`, `initiator` (name, role, authority evidence), `deferral_date`, `deferral_reason`, `revisit_condition`, evidence reference, audit reference, recorder | option, decision text, decision / effective date, approvers, conditions, supersedes, human verification, notice date |
| `DECISION` | T4 / T5 | `APPROVAL-RECORD-TEMPLATE.json` | `DECISION_RECORDED_PENDING_VERIFICATION`; with a consistent human verification `VERIFIED`, exactly the `*_VERIFIED` state equal to its outcome | all fields below | initiator, notice and deferral fields |

`revisit_date` is optional in a deferral and must be an ISO date when present. It is filled only when the
revisit condition itself gives a date. It is never invented.

**Outcome-to-state mapping for decisions** (all other pairs are rejected):

| Outcome | Human verification `VERIFIED` | Human verification `REJECTED` / `INCONCLUSIVE` / none |
|---|---|---|
| `APPROVED` | `APPROVED_VERIFIED` | `DECISION_RECORDED_PENDING_VERIFICATION` |
| `REJECTED` | `REJECTED_VERIFIED` | `DECISION_RECORDED_PENDING_VERIFICATION` |
| `SUPERSEDED` | `SUPERSEDED_VERIFIED` | `DECISION_RECORDED_PENDING_VERIFICATION` |
| `VOID` | `VOID_VERIFIED` | `DECISION_RECORDED_PENDING_VERIFICATION` |

Fields of a `DECISION` record:

| Field | Requirement |
|---|---|
| `record_type` | `DECISION` |
| `decision_id` | Exact ID from the tracker (unchanged from R5) |
| `decision_version` | `1.0` for the first approval; later versions increment |
| `outcome` | One of `APPROVED`, `REJECTED`, `SUPERSEDED`, `VOID` |
| `selected_option` | The option number from the R5 record, or `NEW` with full text |
| `decision_text` | The full selected decision text, not a summary |
| `approvers[]` | For each: `name`, `role`, `authority_evidence` (type + reference). Every role in the record's "required approver roles" appears |
| `independent_approvals` | Roles that had to sign separately (from the record), each matched by its own approver entry |
| `decision_date`, `effective_date` | ISO dates; the effective date is not earlier than the decision date |
| `evidence_source` | `type` (signed minutes, signed document, PR review comment by the approver, other) and `reference` |
| `immutable_ref` | Commit SHA, document hash or versioned archive id of the evidence |
| `supersedes[]` | References of superseded decisions (required for `SUPERSEDED`; for P-GOV option 2 the replacement ID) |
| `conditions` | Conditions and constraints, or the explicit text `none` |
| `recorded_by` | Name and role of the person who wrote the record |
| `human_verification` | Empty until verified (§3) |

## 3. Human verification entry

| Field | Requirement |
|---|---|
| `verifier_name`, `verifier_role` | A person who is not an approver and not the recorder |
| `method` | How authority and evidence were checked (for example: confirmed with each approver through an independent channel; checked the delegation-of-authority document) |
| `date` | ISO date |
| `result` | `VERIFIED`, `REJECTED` or `INCONCLUSIVE` |

The verifier's judgement is the human step. The repository records it but cannot check it.

## 4. Offline structural check (`validateApprovalRecord` in `docs/governance/r6_test.go`)

| Check | Automated? |
|---|---|
| All required fields present and non-empty | Yes |
| `record_type` known; each type has its required fields and none of its forbidden fields (§2) | Yes |
| File name equals `<decision-id>-v<version>-<record_type>.json`; history append-only (no delete / rename; only a verification entry may be added) | Yes (`TestR6ZeroRecordedApprovals`, `TestApprovalRecordsAppendOnly`; history check needs a full clone) |
| Blank templates never validate | Yes |
| Decision outcome allowed (`APPROVED`, `REJECTED`, `SUPERSEDED`, `VOID`; never `DEFERRED`); `SUPERSEDED` has `supersedes` | Yes |
| Tracker status matches the record type and, for a `*_VERIFIED` state, the recorded outcome (`statusMatchesRecord`) | Yes |
| Dates are valid ISO dates; effective ≥ decision | Yes |
| At least one approver; each has name, role and authority evidence reference | Yes |
| Every role in `independent_approvals` is matched by its own approver | Yes (string match of role) |
| `human_verification`, if present: verifier is not an approver and not the recorder, verifier is not automation, result allowed | Yes |
| No synthetic / placeholder / fixture marker in a record under `approvals/` | Yes |
| The person exists, holds the role, signed; the authority document is genuine | **No — human verification only** |
| The decision text is what the approvers actually decided | **No — human verification only** |

Regression tests feed synthetic fixtures, built in test code and never written to `approvals/`, to show that
each structural defect is rejected and that a complete record without verification cannot reach a verified
state.

## 5. Limits carried over from R5

The R5 provenance test (`provenance_test.go`) proves only that cited text existed at a commit. The same limit
applies to approval evidence committed to the repository: committing it proves its presence, not its
authenticity.
