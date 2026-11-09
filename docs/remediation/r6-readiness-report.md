# R6 — Readiness report (management decision workflow)

**Governance is NOT closed.** R6 turns the R5 DRAFT records into management sessions, worksheets, an approval
workflow and an offline evidence standard. It records no decision.

- **Approvals obtained: 0.**
- **Production remains NO-GO.**
- Merging this PR does not close governance and changes no decision status.

## 1. Baseline

`6fce8a709e6878dda550a53f7b568917c1d2a2b5` (main after PR #45, R5).

## 2. Deliverables

| # | File | Language | Content |
|---|---|---|---|
| 1 | `r6-management-executive-summary-fa.md` | Persian | Executive decision summary |
| 2 | `r6-management-decision-tracker.md` | Persian + IDs | Prioritized decision table (18 columns, 61 rows) |
| 3 | `r6-m1-governance-data-session-fa.md` | Persian | M1 agenda, ADR #18 option comparison, P-GOV worksheet, hierarchy review, ownership table |
| 4 | `r6-m2-security-session-fa.md` | Persian | M2 security decisions |
| 5 | `r6-m3-sales-session-fa.md` | Persian | M3 sales decisions (now vs evidence-first) |
| 6 | `r6-m4-finance-sessions-fa.md` | Persian | M4-A … M4-D finance worksheets (independent) |
| 7 | `r6-m5-m6-operations-biawin-reporting-fa.md` | Persian | HR, Biawin (C-03 first), reporting |
| 8 | `r6-approval-form-fa.md` | Persian | Blank decision approval form |
| 9 | `r6-followup-action-register-fa.md` | Persian | Follow-up register (empty) + preparatory actions |
| 10 | `r6-unresolved-evidence-conflicts-fa.md` | Persian | Unresolved evidence and conflict register |
| — | `r6-human-approval-workflow.md` | English | Lifecycle states and transitions |
| — | `r6-approval-evidence-standard.md` | English | Offline validation contract |
| — | `r6-engineering-followup-map.md` | English | Decision-to-engineering matrix; P0 closure criteria |
| — | `docs/governance/r6-decision-tracker.json` | JSON | Lifecycle status per decision + conflict statuses (single machine-readable status source) |
| — | `docs/governance/templates/APPROVAL-RECORD-TEMPLATE.json` | JSON | Blank approval record |
| — | `docs/governance/r6_test.go` | Go | Offline governance checks |

**No duplicated source of truth:**
- R5 stays the source of each record's content (question, options, evidence).
- The R6 tracker holds only lifecycle status.
- The tests tie the two together.

## 3. Results

| Item | Result |
|---|---|
| R5 decision records accounted for | 41 of 41, plus 20 P-GOV identifiers (61 tracked) |
| Management-ready (`READY_FOR_REVIEW`) | 26 |
| Evidence-dependent (`EVIDENCE_MISSING`) | 10 R5 records + 20 P-GOV |
| Blocked by a conflict (`DRAFT`) | 5 (D-35a–e, C-03) |
| Conflicts | 12, statuses unchanged from R5. C-03 OPEN — UNRESOLVED |
| Approvals obtained | **0**. No approval record exists. `docs/governance/approvals/` is absent |
| Options pre-selected | 0 (tested) |
| Approver names recorded | 0 (outcome fields empty, tested) |

## 4. Approval workflow

There are ten states: `EVIDENCE_MISSING`, `DRAFT`, `READY_FOR_REVIEW`, `UNDER_HUMAN_REVIEW`,
`DECISION_RECORDED_PENDING_VERIFICATION`, `APPROVED_VERIFIED`, `REJECTED_VERIFIED`, `DEFERRED`,
`SUPERSEDED_VERIFIED` and `VOID_VERIFIED`. For each transition, the workflow document defines:
- required evidence and the authorized actor (role);
- independent approvers and validation conditions;
- the audit record, possible conflicts and recovery.

None of these events changes a status: document generation, a meeting, tests, a PR approval or a merge.

## 5. Evidence validation limits

| Check | Automated | Human only |
|---|---|---|
| Approval record structure (fields, dates, outcome, approvers with authority evidence, independent roles, recorder) | Yes | — |
| Placeholder / synthetic markers in committed records | Yes | — |
| Verifier ≠ approver, ≠ recorder, ≠ automation; allowed result | Yes | — |
| Person exists, holds the role, signed; authority document genuine; decision text is what was decided | **No** | Yes (T5 verification) |
| Evidence text existed at a commit (R5 provenance test, preserved) | Yes | Authorship / authenticity: human |

The best automated level is "human verification entry consistent". Automation never sets `APPROVED_VERIFIED` on
its own judgement.

## 6. Validation

| Check | Result |
|---|---|
| All 41 R5 records accounted for; none disappears | `TestR6TrackerAccountsForEveryR5Record` |
| D-03 … D-22 non-authoritative (`EVIDENCE_MISSING`; three unticked options each) | `TestR6NoStatusOrOutcomeWithoutApprovalRecord`, `TestR6PersianWorksheets` |
| C-03 unresolved; all conflict statuses equal R5 | `TestR6ConflictsPreserved` |
| No draft marked approved; no outcome or signatory filled | `TestR6NoStatusOrOutcomeWithoutApprovalRecord`, `TestR6PersianWorksheets` |
| No option pre-selected | `TestR6PersianWorksheets` (no ☑ ☒ ✓ [x] in any R6 document) |
| References consistent with R5 (IDs, readiness, blockers) | `TestR6TrackerAccountsForEveryR5Record`, `TestR6BlockersMatchInventory` |
| P0 / P1 classification honest (from the R5 inventory) | `TestR6BlockersMatchInventory` |
| Approval validator rules | `TestApprovalValidatorRules` (15 defect fixtures plus positive, inconclusive and placeholder cases; all synthetic, built in test code, never written to `approvals/`) |
| R5 tests, including the provenance test and its limits | Unchanged, passing |
| Mutation checks (each makes the tests fail) | (1) a status set without a record; (2) an outcome filled; (3) a record dropped; (4) C-03 changed; (5) a blocker changed; (6) a P-GOV box ticked; (7) a worksheet outcome filled; (8) a placeholder approval file |
| Production code / flags | Diff is `docs/` and the Go CI step name only. No change to `internal/`, `cmd/`, `plugin/current`, migrations, flags or permissions |

## 7. Blockers

| Blocker | Status after R6 |
|---|---|
| P0-5 inventory safety | **PARTIAL**: R4 evaluators and synthetic race evidence. 0 decisions (ADR-01, OPD-03, OPD-10 review-ready; F01, D-28, D-32 evidence-first). No command |
| P0-6 HR handover | **OPEN**: OPD-05, SD-04 review-ready; person identity (R5-SEC-01) undecided; HR NOT LIVE VERIFIED |
| P0-7 financial execution | **PARTIAL / BLOCKED**: contracts only. OPD-08 and OPD-04 review-ready, but execution is blocked by person-level SoD. D-41, OPD-06, OPD-09, D-36 evidence-first |
| P0-3 real-data reconciliation | **OPEN**: ADR #18 review-ready (M1) |

## 8. Stop

Ready for independent review. **No merge by the agent, no deployment, no financial execution, no legacy
write, no authority switch, no automatic approval, no Slice 16.**
