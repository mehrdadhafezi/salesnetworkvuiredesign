# RM-E04E — Authority audit, approver matrix and registration packet (M2 Records 1 and 2)

**DRAFT — NOT APPROVED.** This packet registers nothing, appoints nobody and approves nothing. It prepares the
material an **authorized** inventory owner would need.

| | |
|---|---|
| Baseline | `main` @ `238645206040b2e4a715df3855480890b01d2dd6` (PR #77 merged) |
| Approval records on main | **0.** `docs/governance/approvals/` does not exist |
| R6 tracker | 61 records: 30 `EVIDENCE_MISSING`, 26 `READY_FOR_REVIEW`, 5 `DRAFT`. None `APPROVED_VERIFIED`. Unchanged by this PR |
| Group B gate / S4 manifest | Closed / empty. Unchanged by this PR |
| Production | **NO-GO** |

---

## 1. Governance authority audit

### 1.1 Classification key

| Class | Meaning |
|---|---|
| VERIFIED | A committed approval record with independent T5 verification names the holder and their authority evidence |
| DOCUMENTED_BUT_UNVERIFIED | A committed document names the role or rule, but no holder is named with verified authority evidence |
| UNASSIGNED | No committed document names a holder |
| CONFLICTING | Committed sources disagree |

### 1.2 Findings

| # | Question | What the repository says | Source | Role rule | Holder |
|---|---|---|---|---|---|
| Q-A | Who maintains the decision inventory? | Not stated. R5 was produced as engineering documentation. M2 Q2 asks "who is the authorized inventory owner?" and is unanswered | `m2-authorization-decision-intake-fa.md` §7 Q2; `r5-decision-inventory.md` header | DOCUMENTED_BUT_UNVERIFIED (the role exists as a requirement) | **UNASSIGNED** |
| Q-B | Who may register new decision IDs? | Same role as Q-A ("only the authorized inventory owner assigns an ID") | M2 §4, §6; RM-E04D §5 | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-C | Product owner signer | Required role for both records (proposed); gate floor in `rm_e04c_test.go` | M2 §6; `rmE04CGateFloorRoles` | DOCUMENTED_BUT_UNVERIFIED (proposed, not registered) | **UNASSIGNED** |
| Q-D | Sales owner signer | same | same | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-E | Security owner signer | same | same | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-F | Legal approval for Record 2? | Open question (M2 Q8; RM-E04D ت-۴). Precedent: Legal (privacy) is a required approver of ADR #18 and R5-SEC-01, both about access to personal data | M2 §4-2, §7; R5 rows ADR #18, R5-SEC-01 | **Undetermined** (not CONFLICTING: no source says Legal is excluded) | **UNASSIGNED** |
| Q-G | Who records decisions? | "Recorder (any role)"; the record names the approvers | `r6-human-approval-workflow.md` T4; evidence standard §2 `recorded_by` | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-H | Who may perform T5 verification? | A person who is not an approver, not the recorder, and not automation | workflow T5; evidence standard §3 | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-I | Authority hierarchy for signers per domain | M1 §ج question "who has signing authority per domain and what is the evidence?" is blank | `r6-m1-governance-data-session-fa.md` §ج | DOCUMENTED_BUT_UNVERIFIED | **UNASSIGNED** |
| Q-J | Owner lists for cross-unit decisions | Conflict C-02 (Cross-Role §19 vs Slice 15 register) | `r5-decision-conflict-register.md` | **CONFLICTING** (affects OPD owners; not the Product / Sales / Security roles of Records 1–2) | — |
| Q-K | "Human Owner" (E0) | Final merge authority (E0 §5 rule 2); approver of D-01, D-02, MONEY (E0). R6 rule: the repository owner's account alone does not prove organizational authority | E0 §5; `r6-human-approval-workflow.md` §1 | DOCUMENTED_BUT_UNVERIFIED **for merge only** | Not evidence for Q-A…Q-H |

**Result:** no role needed for Records 1 and 2 is VERIFIED. All holders are **UNASSIGNED**. M2 registration
**cannot legally proceed** until M1 names the inventory owner and the signers with authority evidence.

Not counted as authority: merging a PR, a GitHub approval, CI results, this document, an agent's recommendation, or
the user's instruction to run RM-E04E.

### 1.3 Mandatory authority evidence per person

| Item | Requirement | Source |
|---|---|---|
| Authority document | Appointment letter, delegation-of-authority document, or board / director resolution | evidence standard §1 |
| Reference | Number or archive location of that document | §2 `authority_evidence.reference` |
| Scope match | The document covers the domain signed for (Product / Sales / Security / Legal / inventory) | M1 §ج |
| Independence | One person = one approver, whatever the number of roles or accounts | M2 §6; `rmE04CGateFloorRoles` comment |
| Immutable evidence | `immutable_ref`: commit SHA, document hash or versioned archive id of the decision evidence | §2 |
| Not sufficient alone | Typed name, signature image, repository-owner account | §1 |

### 1.4 Authority-assignment worksheet (for M1; blank by design)

Filled only by the people with authority to appoint. Never by engineering or an agent.

| Role | Person (name) | Authority document type | Reference | Appointed by (name, role) | Date | Verified by (T5-style, independent) |
|---|---|---|---|---|---|---|
| Decision inventory owner (assigns IDs) |  |  |  |  |  |  |
| Product owner |  |  |  |  |  |  |
| Sales owner |  |  |  |  |  |  |
| Security owner |  |  |  |  |  |  |
| Legal owner (privacy), if required for Record 2 |  |  |  |  |  |  |
| Finance owner (conversion-payment contract) |  |  |  |  |  |  |
| Recorder |  |  |  |  |  |  |
| T5 verifier (not an approver, not the recorder) |  |  |  |  |  |  |

## 2. Approver matrix

| Item | Product | Sales | Security | Legal | Finance | Codex | Recorder | T5 verifier |
|---|---|---|---|---|---|---|---|---|
| Assign IDs (inventory owner) | — | — | — | — | — | — | — | — |
| Record 1 (capability) | sign | sign | sign | — | — | architecture review | record | verify |
| Record 2 (row scope) | sign | sign | sign | **open (PQ-4)** | — | architecture review | record | verify |
| Conversion-payment contract (S4) | per registration | — | sign (floor) | — | sign (floor) | data contract | record | verify |
| S4 execution evidence (E4) | — | — | sign | — | sign | code review | record | verify |
| ADR acceptance (B1) | — | — | review | — | — | owner | — | — |
| A3 post-merge sign-off | — | — | — | — | — | owner (owed) | — | — |

"sign" = an independent approver entry in the R6 `DECISION` record. A verifier who signs is invalid.

## 3. Registration packet

### 3.1 Candidates

| Field | Record 1 | Record 2 |
|---|---|---|
| Candidate title (R5 Title column) | `Seller first-release capability eligibility` | `Seller row-scope (ownership predicate) per destination` |
| Title check | Must match `(?i)\bseller\b.*\bcapabilit` (`rmE04CGateTitle[1]`) | Must match `(?i)\bseller\b.*\b(row[- ]?scope\|read[- ]?scope\|ownership)` (`rmE04CGateTitle[2]`) |
| Decision question | Which target capabilities may the `seller` bundle hold in the first release? | For S1, S2 attribution, S4 and S7, which ownership fact defines "my rows"? |
| Options | A / B / C / D (form §3) | P1 / P2 / P3 for S1; inherit for S2; I-a / I-b / I-c / I-d / I-L for S4; C for S7; defer (form §4) |
| Proposed wording | Form §12 (blanks) | Form §11 (blanks) |
| Evidence references | Review package V1, V2, V7, V8; `TestRME04CProposalsStayOutOfRuntime` | Review package V3, V6, V8–V14, §2 synthetic evidence; `rm_e04d_ownership_evidence_test.go` |
| Dependencies | Record 2; ADR (B1); D-42 / D-34 for real grants | Record 1; GAP-04; OPD-10; conversion-payment contract; ADR #18 |
| Required approver roles | Product owner, Sales owner, Security owner | Product owner, Sales owner, Security owner; Legal owner if PQ-4 = yes |
| Business owner (proposed) | Product + Sales | Product + Sales |
| R5 package (proposed) | P-SEC (same as D-42) — inventory owner decides | P-SEC — inventory owner decides |
| Readiness after registration | READY FOR HUMAN REVIEW | READY FOR HUMAN REVIEW if PQ-1…PQ-6 are attached as open questions; otherwise NEEDS FURTHER EVIDENCE |

A third candidate is needed before S4 can ever open: **conversion-payment data contract** (title must match
`(?i)conversion[- ]payment`; floor signers Finance owner + Security owner; `s4ContractTitle`,
`s4ContractFloorRoles`). It is prepared only as a review request in the handoff (§4 there), not as a form.

### 3.2 Candidate draft records (outside `approvals/`)

`docs/governance/proposals/rm-e04e-candidate-records/` holds one `DECISION` draft per record. Each:
- carries a placeholder `decision_id`, so the validator rejects it in repository mode;
- has every outcome, approver and verification field empty;
- lists the options in a `_candidate_options` field that the validator ignores.

`TestRME04EDraftsNeverValidate` keeps them invalid. They must never be copied into `approvals/` unfilled, and
never be filled on someone's behalf.

### 3.3 Approval-record schema and file naming

- Schema: `docs/remediation/r6-approval-evidence-standard.md` §2 (`DECISION`, `REVIEW_NOTICE`, `DEFERRAL`) and §3
  (human verification). Templates: `docs/governance/templates/*.json`.
- Path: `docs/governance/approvals/<decision-id>-v<version>-<record_type>.json`. Characters other than letters,
  digits and `-` become `_`.
- `decision_id` = the exact tracker ID. `decision_version` = `1.0` first.
- `independent_approvals` lists every required role, each matched by its own approver entry.
- No `SYNTHETIC`, `PLACEHOLDER`, `TODO`, `EXAMPLE` or `FIXTURE` text anywhere in the record.
- Append-only: a later change may only add `human_verification`. Corrections are new version files.

### 3.4 Recorder requirements

- Writes the record from the signed evidence; never decides.
- Is named in `recorded_by` (name + role).
- Is not the T5 verifier. May be any role, including an approver.
- Fills `decision_text` with the full text actually decided, not a summary.

### 3.5 T5 independent verification checklist

- [ ] Verifier is not an approver, not the recorder, not automation (an agent is automation).
- [ ] Each approver confirmed through an independent channel (not the channel that delivered the record).
- [ ] Each authority document inspected; scope matches the signed domain; reference matches the record.
- [ ] `immutable_ref` resolves to the evidence that the approvers actually signed.
- [ ] `decision_text` and `selected_option` match the signed evidence word for word.
- [ ] Every required role signed separately; no person counted twice.
- [ ] `effective_date` ≥ `decision_date`; conditions recorded or `none`.
- [ ] Result recorded as `VERIFIED`, `REJECTED` or `INCONCLUSIVE`, with method and date.

### 3.6 Acceptance criteria for registration (per record)

1. The inventory owner's own authority evidence is committed (or referenced) with the registration commit.
2. The R5 row, R5 package record, evidence-index entry, R6 tracker entry, M2 worksheet, management-tracker row
   and executive-summary entry all exist and agree (§4).
3. Tracker status is `READY_FOR_REVIEW` or `EVIDENCE_MISSING`; every outcome field empty; `approval_record` empty;
   `verification` = `NOT_STARTED`.
4. No gate, manifest, capability, grant, schema or OpenAPI change in the same commit.
5. Full governance suite green, including provenance with full history.

## 4. Registration transaction plan (for the authorized inventory owner)

Derived from a **throwaway dry-run** in a scratch copy (placeholder IDs, never committed): with only the R5 row,
evidence-index entry and tracker entry added, four tests failed; after the remaining files below were added, the
suite passed (provenance excluded because the scratch copy had no history).

### Step R — Registration (one reviewed commit per record, or one for both)

| # | File | Change | Test that requires it |
|---|---|---|---|
| R1 | `docs/remediation/r5-decision-inventory.md` | New 17-column row; title per §3.1; blocker column consistent with the tracker | `TestEveryDecisionIDIsInventoried`, `TestR6BlockersMatchInventory`, `r5Titles` (gate) |
| R2 | `docs/remediation/r5-person-identity-sod-DRAFT.md` (or the package chosen) | Full R5 record block, `**DRAFT — NOT APPROVED**`, all six approval fields empty | `TestDraftRecordsCarryNoApproval` (package records = index records) |
| R3 | `docs/governance/r5-evidence-index.json` | `decision_records` entry (id, package, status, readiness, evidence ids) | same; `TestR6TrackerAccountsForEveryR5Record` |
| R4 | `docs/governance/r6_test.go` | R5 record count `41` → `43` (two records) | `TestR6TrackerAccountsForEveryR5Record` |
| R5 | `docs/governance/r6-decision-tracker.json` | Record with status per readiness, meeting `M2`, empty outcome, `NOT_STARTED` | `TestR6NoStatusOrOutcomeWithoutApprovalRecord` |
| R6 | `docs/remediation/r6-m2-security-session-fa.md` | `### <ID> — <title>` worksheet with `**DRAFT — NOT APPROVED / پیش‌نویس — تأییدنشده**` and empty outcome table | `TestR6PersianWorksheets` |
| R7 | `docs/remediation/r6-management-decision-tracker.md` | Table row `\| <ID> \| …` | `TestR6PersianWorksheets` |
| R8 | `docs/remediation/r6-management-executive-summary-fa.md` | Mention of the ID (review-ready) | `TestR6PersianWorksheets` |
| R9 (recommended) | `docs/governance/m2_test.go` → `m2Approvers` | Add both IDs with their registered approver roles, so the M2 freeze and role check cover them | `TestM2TrackerUnchanged` |
| R10 (recommended) | `docs/governance/intake/m2/<ID>-v1.0-{DECISION,REVIEW_NOTICE,DEFERRAL}.BLANK.json` | Blank intake forms | `TestM2IntakeBlanksNeverValidate` |

The ID format chosen by the inventory owner must be recognised by `decisionID` in `governance_test.go`, or that
regex must be extended in the same commit; otherwise documents citing the new ID are not checked.

### Step T — Review and decision (R6, by the named people)

| # | Transition | File |
|---|---|---|
| T3 | `READY_FOR_REVIEW` → `UNDER_HUMAN_REVIEW` (optional) | `approvals/<ID>-v1.0-REVIEW_NOTICE.json` + tracker |
| T4 | → `DECISION_RECORDED_PENDING_VERIFICATION` | `approvals/<ID>-v1.0-DECISION.json` + tracker outcome fields |
| T5 | → `APPROVED_VERIFIED` | `human_verification` added to the same file + tracker `verification` |
| T6 | Option D / defer | `approvals/<ID>-v1.0-DEFERRAL.json`; status `DEFERRED` (never opens the gate) |

### Step G — Only after both records are `APPROVED_VERIFIED` (B-PR1)

| # | File | Change |
|---|---|---|
| G1 | `docs/governance/rm-e04c-group-b-gate.json` | Both IDs; exactly the approved capabilities and `/me` operation ids. **No S4 entries** unless `rm-e04d-s4-readiness.json` passes every proof |
| G2 | `docs/implementation/rm-e04c-seller-authorization-adr-PROPOSED.md` | Status may become ACCEPTED only with an open gate (`TestRME04CDocumentsAreMarkedNotApproved`) and Codex architecture review |
| G3 | Capability / operation lists | Code registration happens later in B-PR2 / B-PR3, never in the gate commit |

### Things that must **not** change during registration

Gate, S4 manifest, capability catalogue, `seller` bundle, grants, OpenAPI, migrations, runtime authorization.

### Negative controls run in this PR (scratch copy; not committed)

| Control | Result |
|---|---|
| Gate names two registered IDs that are still `READY_FOR_REVIEW` | Rejected (`TestRME04CProposalsStayOutOfRuntime`: "is READY_FOR_REVIEW, not APPROVED_VERIFIED") |
| Tracker set to `APPROVED_VERIFIED` without an approval record | Rejected (`TestR6NoStatusOrOutcomeWithoutApprovalRecord`) |
| Registration with only inventory + index + tracker | Rejected by four tests (shows the full file set is needed) |
| Candidate drafts validated as repository records | Rejected (`TestRME04EDraftsNeverValidate`, committed) |

## 5. Evidence inventory (what exists, what is missing)

| Evidence | Status | Where |
|---|---|---|
| Seller bundle empty; no active grant path | PRESENT (code + tests) | V1, V2; `TestRME04DNoActiveGrantPathWithoutD42` |
| Evaluator scopes `NONE` / `ORGANIZATION` only | PRESENT | V3, V15 |
| `CurrentHolder` = P2 | PRESENT | V9 |
| Legacy holder sources | PRESENT (code) | V10 |
| Legacy Seller invoice rule (I-L) | PRESENT (code) | V13, V14 |
| Conversion-payment fact in target | **MISSING** | V11 |
| Synthetic predicate rates | PRESENT (synthetic only) | review package §2 |
| Real-data rates (seller ≠ issuer, unknown holders) | **MISSING** (needs ADR #18) | — |
| Answers PQ-1…PQ-6 | **MISSING** | Record 2 form §8 |
| Authority evidence of any signer or of the inventory owner | **MISSING** | §1 |
| Codex review (architecture, A3, data contract) | **PENDING** | handoff |
| S4 execution-evidence mechanism (E1–E5) | **MISSING** | review package §7 |
| Approval records | **NONE** | — |

## 6. Dependency summary

```
M1 authority (inventory owner + signers)            ── blocks ─▶ registration of Records 1, 2
Registration ─▶ M2 session ─▶ DECISION records ─▶ T5 ─▶ both APPROVED_VERIFIED
Codex architecture review (ADR, scope enum)          ── blocks ─▶ B-PR1 (ADR accepted) / B-PR2
Both APPROVED_VERIFIED + B-PR1                         ─▶ Group B gate open (no S4)
Conversion-payment contract (Finance+Security, T5) + schema + 3a + 3b(E1–E5) ─▶ S4 may enter the gate
D-42 + D-34 + Group C authorization                    ─▶ any real grant to a person
D-41 / OPD-09 / D-36                                   ─▶ any settlement meaning in S7
ADR #18                                                ─▶ any real-data evidence
```
