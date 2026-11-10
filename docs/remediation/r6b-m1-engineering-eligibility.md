# R6-B — Engineering eligibility after a verified ADR #18 decision

**Nothing listed here is authorized today.**
- ADR #18 is `READY_FOR_REVIEW`.
- Approvals: 0.
- This document lists only the work that would become *eligible for planning* after a real, verified decision.
- Each later stage needs its own authorization.
- No command runs because a policy was approved.

## 1. Four separate gates

| Gate | Condition | Engineering may | Engineering may not |
|---|---|---|---|
| **G1 Policy approved** | ADR #18 `APPROVED_VERIFIED` (approval record + human verification; `r6-approval-evidence-standard.md`) | Write the plan and work items (§2), estimate, design | Write or run anything that touches real data; change the loader; provision staging with real data |
| **G2 Implementation permitted** | A follow-up work item created from the approval record (`r6-engineering-followup-map.md` §1) with a named engineering owner, plus Security review of the design | Build and test the tooling with **synthetic data only** | Load, extract, transfer or retain real data |
| **G3 Staging evidence verified** | Rehearsal checklist (`r6b-m1-staging-rehearsal-checklist-fa.md`) complete and independently verified, **and** a per-run written permission from the data owner and Security for one bounded extraction / transfer / import / retention | One bounded rehearsal in the isolated staging environment; record results in the reconciliation register | Any other run; any business use; any connection to Production or legacy write paths |
| **G4 Authority to activate** | A separate verified decision naming what may use the results (for example, evidence for F01, D-41, OPD-06) | Only what that decision names | Production (remains NO-GO); write-back; financial or business operations |

If the verified outcome is option 1 (synthetic only) or `REJECTED`, nothing in §2 becomes eligible. P0-3 then
stays open, and the decisions that wait on real data stay `EVIDENCE_MISSING`.

## 2. Work that becomes eligible for planning at G1

| # | Work item | Today | Needs before it runs |
|---|---|---|---|
| W1 | Extraction runbook and tooling for an authorised snapshot (E-4): manifest, checksums, cutoff, stored outside the repository | `crm-legacy-import specs / validate / inspect` exist; no extraction from a real system | G2 (build, synthetic); G3 (any real run) |
| W2 | Extract specs for the 55 uncovered legacy tables (E-5), limited to the fields the decision allows | 34 of 89 tables covered (go/no-go) | G2 |
| W3 | Anonymisation / pseudonymisation transform, if the decided option requires it; re-identification test | Absent | G2; Security review |
| W4 | Allow `crm-legacy-import load` to accept a non-synthetic package, behind an explicit, audited switch | Non-synthetic packages are **BLOCKED** (`cmd/crm-legacy-import/main.go`, `load`) | G2 (code change with tests); G3 (switch used for one run) |
| W5 | Isolated staging environment for real data: network isolation, encryption, secret management, access list with expiry | R3 staging engineering for synthetic data only | G2 (provisioning plan); G3 (verified controls) |
| W6 | Destruction tooling and destruction evidence (data, backups, keys) | Absent | G2 |
| W7 | Access and key-use audit logging for the staging environment | Absent for real data | G2 |
| W8 | Reconciliation runs per cohort, results in `docs/cutover/data-reconciliation-register.md` with snapshot id and cutoff | Synthetic only; register `NOT RUN` | G3 |

Each item needs, as its own follow-up work item:
- tests;
- runtime verification;
- an operational owner;
- a rollback plan;
- its own activation approval (`r6-engineering-followup-map.md` §1).

## 3. P0-3 closure (unchanged criteria)

P0-3 closes only when the snapshot-based reconciliation register is filled with zero unexplained deltas
(go/no-go). That needs:
- G1 (ADR #18);
- D-23;
- W1–W8;
- G3 for every cohort run.

Approving ADR #18 alone does not change P0-3.

## 4. What stays prohibited at every gate

- Production deployment.
- Legacy write-back and changes to the operational WordPress CRM or `plugin/current`.
- Financial or business operations.
- Real data in the repository, CI, personal machines or external tools.
- Phone-based identity merges.
- Slice 16.
