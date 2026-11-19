# RM-E04G — F1 person-identity independence hardening

**DRAFT — NOT APPROVED.** Security analysis, reproduction evidence, identity-independence contract proposal and an
interim fail-closed governance guard. No approval record, person id, account link, decision status, gate, grant,
schema or runtime change. **Production: NO-GO. Group B: BLOCKED. S4: BLOCKED.**

| | |
|---|---|
| Finding | F1 (RM-E04F §2.3): R6 independence is keyed by a normalised display name (`personKey`) |
| Baseline | `main` @ `162ff8981c3331598616cdc38cb122dc9ab17eb9` (PR #79 merged) |
| Hard dependency | R5-SEC-01 (authoritative person identity) — tracker `READY_FOR_REVIEW`, not `APPROVED_VERIFIED`. Not chosen here |
| Tests | `docs/governance/rm_e04g_test.go` |
| Review | `docs/implementation/rm-e04g-codex-security-review-handoff.md` — REVIEW PENDING |

---

## 1. Summary

A display name is presentation data. The current R6 validator uses it as identity, so it both accepts the same
human twice (Persian / Latin spellings, second account) and rejects two different humans who share a name. No
authoritative account→person source exists in the repository (R5-SEC-01 open), and R6 v1 records carry no account
or person reference, so **the repository cannot today prove person independence for any decision**.

This change therefore does not "fix" independence. It:

1. pins the F1 behaviour of the unchanged validator in executable tests (§3);
2. specifies a person-keyed independence contract (§5) and implements it as a pure evaluator that takes the
   resolver as a parameter (it does not pick the R5-SEC-01 source of truth);
3. binds the repository to a resolver that resolves nothing, and adds an **interim Group B identity-independence
   readiness rule**: Group B cannot be ready on names alone (§7);
4. closes a related gap, **F1-I**: committed approval records may carry only R6 v1 fields, because unknown JSON
   keys (e.g. a smuggled `person_ref`) are silently dropped by `loadApproval` and are invisible to
   `appendOnlyChange` (§7.3).

## 2. Preflight evidence

| Check | Result |
|---|---|
| `origin/main` | `162ff89` = expected SHA; PR #79 is its merge commit |
| Worktree | isolated worktree from `origin/main`, branch `fix/rm-e04f-person-identity-independence` |
| Baseline `go test ./docs/governance/` | PASS (also with `R5_REQUIRE_PROVENANCE=1` on a full clone) |
| Baseline `go test ./...` (this container) | one pre-existing failure unrelated to governance: `internal/integration/legacy/shadow` `TestPgtestTemplateConcurrentClones` (no PostgreSQL in the container; DSNs empty). Not touched |
| `docs/governance/approvals/` | does not exist — zero R6 records (R6 Zero Approvals baseline) |
| Group B gate / S4 manifest | both empty (closed) |
| R5-SEC-01 tracker | `READY_FOR_REVIEW`, no record, verification `NOT_STARTED` |

## 3. F1 reproduction (current validator, unchanged)

`TestRME04GReproduceF1WithCurrentValidator` pins the **current** level returned by `validateApprovalRecord`.
Fixture names are synthetic (`نمونه آزمایشی` / `Nemooneh Azmayeshi`, one fictional person).

| Case | Scenario | Representable in R6 v1? | Current validator outcome | Correct outcome | Status |
|---|---|---|---|---|---|
| F1-A | Same person signs Product (Persian) and Sales (Latin) | yes (names) | `DECISION_RECORDED_PENDING_VERIFICATION`; with T5 → `HUMAN_VERIFICATION_ENTRY_CONSISTENT` | not independent | **REPRODUCED (false acceptance)** |
| F1-B | Same person, second account, different spelling | names only; the account is not recorded | `HUMAN_VERIFICATION_ENTRY_CONSISTENT` | not independent | **REPRODUCED** at name level; the account dimension is **NOT REPRESENTABLE** |
| F1-C | Recorder verifies own decision under another spelling | yes | `HUMAN_VERIFICATION_ENTRY_CONSISTENT` | T5 invalid | **REPRODUCED (false acceptance)** |
| F1-D | Approver verifies through another account / spelling | names only | `HUMAN_VERIFICATION_ENTRY_CONSISTENT` | T5 invalid | **REPRODUCED** at name level |
| F1-E | Two different people share a display name | yes | `STRUCTURALLY_INCOMPLETE` ("already signed under another role") | independent | **REPRODUCED (false rejection)** — fail-safe direction, but wrong |
| F1-F | Account→person link UNKNOWN | **no** field | not visible to the validator | fail closed | NOT REPRESENTABLE — needs resolver (REQUIRES_R5_SEC_01) |
| F1-G | CONFLICT / `SHARED_USER` | **no** field | not visible | fail closed | NOT REPRESENTABLE — needs resolver |
| F1-H | Delegated signers with overlapping humans | **no** delegation field | not visible | not independent | NOT REPRESENTABLE — needs resolver + delegation evidence |
| F1-I | Identity-shaped field added to a committed record | silently dropped by `loadApproval` | accepted; `appendOnlyChange` blind to it | refused | **REPRODUCED** (scratch mutation MU-R1: only the new guard rejects it) |

Every case above, evaluated through the R6 v1 adapter and the repository resolver, is
`PERSON_INDEPENDENCE_UNKNOWN`. No display-name comparison (exact, normalised, transliterated or fuzzy) is claimed
to prove or disprove identity.

F1-B, F1-D and F1-F…H are additionally exercised against the person-keyed evaluator with a **synthetic** directory
(`TestRME04GPersonIndependenceRules`), which shows the rule that a future reviewed resolver must satisfy.

## 4. Audit of current contracts and dependency map

### 4.1 Affected functions (all in `docs/governance/`, test-only package)

| Function | File | Role in F1 |
|---|---|---|
| `personKey` | `r6_test.go` | Name normaliser used as identity key (root cause) |
| `validateApprovalRecord` | `r6_test.go` | Approver de-duplication and verifier ≠ approver / recorder by `personKey`; automation by name list |
| `statusMatchesRecord` | `r6_test.go` | Maps validator level → `*_VERIFIED` tracker states |
| `checkRecordedDecision` / `TestR6NoStatusOrOutcomeWithoutApprovalRecord` | `r6_test.go` | Tracker → record consistency |
| `appendOnlyChange` | `r6_test.go` | Compares re-marshalled structs: unknown JSON keys invisible (F1-I) |
| `loadApproval` | `r6_test.go` | `json.Unmarshal` drops unknown keys |
| `closedPolicy`, `followUpEligible` | `r6_test.go` | Treat `APPROVED_VERIFIED` as actionable for engineering follow-up |
| `rmE04CDecisionProblems` | `rm_e04c_test.go` | Quoted-decision transitions; required roles by role string |
| `rmE04CGateState` | `rm_e04c_test.go` | **Group B gate**: opens on both M2 ids `APPROVED_VERIFIED` with floor roles Product + Sales + Security |
| `rmE04CRuntimeProblems` | `rm_e04c_test.go` | Group B runtime artefacts allowed iff gate open |
| `s4ContractApprovalProblems`, `s4ReadinessProblems` | `rm_e04d_test.go` | S4 (inside the Group B gate) needs a Finance + Security approved contract |
| `grantConstraintProblems` | `rm_e04d_test.go` | **D-42 `APPROVED_VERIFIED` unlocks migrations touching `identity.role_grant`** |
| `baselineOrVerifiedProblems`, `approvalFilesProblems` | `rm_e04d_test.go` | Repository record acceptance |
| `TestRME04FOnePersonIsOneApprover`, `…RecorderOrApproverCannotVerify`, `…NomineeIsNotAnApprover` | `rm_e04f_test.go` | Pin name-level behaviour; nominee check also by `personKey` |

### 4.2 Routes that may interpret a decision as verified

| # | Route | Consumer | Covered by interim rule? |
|---|---|---|---|
| RT-1 | M2 record 1 / 2 `APPROVED_VERIFIED` → Group B gate open | `rmE04CGateState` → `rmE04CRuntimeProblems` | **Yes** — `rmE04GGroupBReadiness` + `TestRME04GRepositoryGroupBNeedsPersonIndependence` |
| RT-2 | S4 conversion-payment contract `APPROVED_VERIFIED` → S4 in gate | `s4ReadinessProblems` (requires the gate to list S4) | **Yes, transitively** (S4 needs an open, ready gate) |
| RT-3 | D-42 `APPROVED_VERIFIED` → `role_grant` migrations allowed | `grantConstraintProblems` | **No** — outside the authorised interim scope; decision requested (review item 9b) |
| RT-4 | Any `APPROVED_VERIFIED` → engineering follow-up eligible | `followUpEligible` (documentation rule) | No — documentation only; reviewers to decide (item 2) |
| RT-5 | Tracker `*_VERIFIED` accepted | `statusMatchesRecord` | No — R6 semantics unchanged by design (§6) |
| RT-6 | Nominee → approver | `TestRME04FNomineeIsNotAnApprover` (by `personKey`) | Unchanged; name-keyed, so an alias of the nominee is not caught (listed residual risk) |

### 4.3 Identity data available today

| Source | Content | Usable as verified identity? |
|---|---|---|
| `organization.person` | uuid `person_id`, legacy display name | Candidate anchor only; not verified (R5-SEC-01) |
| `identity.user_account` | uuid `user_id` | Account, not a human |
| `organization.employment.user_link_state` | `LINKED`, `USER_NOT_IMPORTED`, `USER_NOT_RECORDED`, `SHARED_USER` | Evidence only; `SHARED_USER` → CONFLICT, `USER_NOT_*` → UNKNOWN |
| R6 v1 approval record | `name`, `role`, authority evidence reference | **No account / person field** |
| RM-E04F nomination register | `person_ref` empty, `identity_resolution = NOT_STARTED` | No |
| GitHub accounts / PR reviewers | logins | **Never** a person identity |

## 5. Identity-independence contract (proposal)

Evaluated per decision version. Only `PERSON_INDEPENDENCE_VERIFIED` may satisfy a gate.

| # | Rule | Implemented in evaluator (`rmE04GIndependence`) | Proven by |
|---|---|---|---|
| S1 | Each approver resolves to exactly one verified immutable `person_id` | yes | F1-F, F1-F/inc |
| S2 | Every required independent approval uses a distinct `person_id` | yes | F1-A, F1-B, MISS-R |
| S3 | Recorder resolves to a verified `person_id` | yes | F1-F/rec |
| S4r | T5 verifier resolves to a verified `person_id`; exactly one verifier | yes | F1-F/ver, MISS-V |
| S5 | T5 verifier ∉ {approver signers, delegating principals, recorder} | yes | F1-C, F1-D, F1-H/3 |
| S6 | Several accounts of one person count as one human | yes (keyed by `person_id`) | F1-B, F1-C, F1-D |
| S7 | UNKNOWN, CONFLICT, `SHARED_USER`, incomplete proof fail closed; CONFLICT never hidden by another finding | yes | F1-F*, F1-G* |
| S8 | Delegation reserves delegate and delegator; never creates independence | yes | F1-H, F1-H/2, F1-H/4, F1-H/5 |
| S9 | Display names are never evidence | yes (never read) | `TestRME04GDisplayNameIsNotEvidence` |
| S10 | Historical evidence immutable | existing `appendOnlyChange` + F1-I guard | `TestRME04FHistoricalApprovalNotRewritten`, MU-R1 |
| S11 | No CRM access grant created or widened | nothing in this change touches identity / grants | `TestRME04FAuthorityIsNotAGrant` (existing) |
| S12 | Incompatible existing records need migration review | zero records exist; no rewrite path added | `TestR6ZeroRecordedApprovals` (existing) |

Outcome precedence: `CONFLICT` > `UNKNOWN` > `VIOLATED` > `VERIFIED`. Rule I7 (recorder may also be an approver) is
kept unchanged from R6.

Explicitly excluded: biometric, phone-based, e-mail-based, transliteration-based or fuzzy matching; GitHub account
or PR reviewer as a person.

## 6. Implementation boundary

| Change | Class | In this PR? | Why safe / why not |
|---|---|---|---|
| F1 reproduction tests | SAFE_NOW | yes | Read-only use of the existing validator |
| Person-keyed evaluator (pure, resolver as parameter) | SAFE_NOW | yes | Does not choose a source of truth; with no resolver every result is UNKNOWN |
| Repository resolver = resolve nothing | SAFE_NOW | yes | Fail-closed; identical behaviour under every R5-SEC-01 option (A, B, C, D): under C / D independence stays UNKNOWN, as RM-E04F §11.1 already states |
| Interim Group B readiness rule | SAFE_NOW | yes | Scope = Group B gate only; Group B is already closed; does not block recording, verifying or tracking any decision (§7.2) |
| F1-I unknown-field guard | SAFE_NOW | yes | Zero records exist; every template and intake blank passes (calibrated); forbids identity fields until a reviewed schema exists rather than defining them |
| Security analysis, contract, review handoff | SAFE_NOW | yes | Documentation |
| Authoritative `person_id` resolution | REQUIRES_R5_SEC_01 | no | Source of truth undecided |
| Mapping accounts to verified people | REQUIRES_R5_SEC_01 | no | idem |
| Person-based independence in the R6 validator | REQUIRES_R5_SEC_01 | no | Roadmap F-PR2 |
| Final R6 approval schema enforcement | REQUIRES_R5_SEC_01 | no | idem |
| R6 compatibility and migration policy | REQUIRES_ADDITIONAL_APPROVAL | no | Review item 5 |
| New approval-record identity fields (`person_ref`, `verifier_person_ref`, …) | REQUIRES_ADDITIONAL_APPROVAL | no | RQ-13; F-PR1 |
| Historical evidence migration | REQUIRES_ADDITIONAL_APPROVAL | no | None needed today (zero records) |
| Extending the interim rule to RT-3 (D-42 grant path) | REQUIRES_ADDITIONAL_APPROVAL | no | Changes an existing reviewed unlock rule |
| Any production / runtime enforcement | REQUIRES_ADDITIONAL_APPROVAL | no | **BLOCKED**; no governance runtime exists |

The R6 validator, `personKey`, tracker, templates, intake blanks, gate file, S4 manifest, nomination register,
seller bundle and grant constraints are **unchanged**.

## 7. Interim fail-closed rule

### 7.1 Rule

> **RM-E04G-I1.** Group B is ready only when the existing RM-E04C gate is open **and** each M2 record evaluates
> to `PERSON_INDEPENDENCE_VERIFIED` under the repository claims adapter and the repository resolver. Until a
> reviewed change replaces `rmE04GRepositoryResolver` (after R5-SEC-01) and `rmE04GClaimsFromRecord` (after an
> approved R6 identity extension), every M2 record is `PERSON_INDEPENDENCE_UNKNOWN`, so Group B cannot be ready.

Enforcement: `TestRME04GRepositoryGroupBNeedsPersonIndependence` fails CI if the gate file is ever opened while
readiness fails. Runtime (application / database) enforcement: **BLOCKED** — no governance runtime exists.

### 7.2 Why this is not a blanket ban

- R6 decisions can still be recorded, T5-verified and tracked exactly as before; no existing test expectation
  changed.
- `TestRME04GGroupBReadinessRule` proves the rule **opens** for a person-independent claim set under a
  (synthetic) reviewed resolver, and stays closed for name-only, alias, second-account, recorder-self-verification
  and `SHARED_USER` sets.
- It is relaxed only by replacing two named functions in a reviewed PR; it cannot be relaxed by data (tracker,
  gate, register or approval files).

### 7.3 F1-I guard

`rmE04GUnknownFieldProblems` accepts exactly the R6 v1 field set (plus the `_comment` key present in every template
and intake blank) and is applied to every file under `docs/governance/approvals/` and to every committed revision of
those files. Residual: `_comment` itself is outside `appendOnlyChange` (presentation only; not evidence).

### 7.4 Limitations

- Does not prove that two names are the same or different humans; nothing in the repository can.
- RT-3 (D-42 → `role_grant` migrations) and RT-4 (follow-up eligibility) still rely on name-keyed independence.
- Name-level checks of the existing validator stay as the floor; F1-E false rejections remain until F-PR2.
- The T5 human checklist item "no person counted twice" (RM-E04E §3.5) remains the only control for non-Group-B
  decisions.

## 8. Historical and compatibility analysis

| Item | Impact |
|---|---|
| Existing approval-record formats | Unchanged. Unknown keys are now refused in committed records; templates / intake blanks calibrated as accepted |
| R6 tracker states | Unchanged; no status moved |
| R6 Zero Approvals baseline | Holds (no `approvals/` directory) |
| Append-only validation | Unchanged; strengthened by F1-I (history scan of every revision) |
| Group B readiness | Stricter (interim rule); gate file unchanged and closed |
| S4 readiness | Transitively stricter; manifest unchanged and empty |
| A3 post-merge sign-off | Not claimed; requires the human reviewers |
| Codex / Security review requirements | New handoff, PENDING |
| Future governance module (RM-E04F §8, §12) | Evaluator mirrors rules I1–I6; `resolve(user_id, at)` contract unchanged; F-PR2 replaces the repository resolver |
| Historical records | None exist; no invalidation, no rewrite. Any future v1 record stays valid for R6 and is UNKNOWN for Group B identity until a reviewed migration policy says otherwise |

## 9. Threat model

| # | Threat | Today (current code) | After this PR | Remaining dependency |
|---|---|---|---|---|
| T1 | Persian / Latin alias of one approver | Accepted (F1-A) | Group B: refused (UNKNOWN). Other decisions: still accepted at R6 level | R5-SEC-01 + F-PR2 |
| T2 | Two accounts, one human | Not visible | Group B refused; evaluator counts one person | Resolver |
| T3 | Shared user account | Not visible | Evaluator → CONFLICT | Resolver must surface `SHARED_USER` |
| T4 | Homonyms | False rejection (F1-E) | Unchanged at R6 level; evaluator accepts distinct `person_id`s | F-PR2 |
| T5 | Conflicting HR identity records | Not visible | Resolver CONFLICT → fail closed | R5-SEC-01 conflict policy |
| T6 | Delegation chains | Not representable | Evaluator reserves delegate and principal | Delegation evidence model (RM-E04F §5.3, AD5) |
| T7 | Impersonation (signing as someone else) | Not detectable offline | Not detectable offline | Authentication + independent T5 channel |
| T8 | Approval evidence forgery (forged `APPROVED_VERIFIED`, smuggled identity field) | Forged status refused by existing tests; smuggled field accepted | Smuggled field refused (F1-I); forged status refused (MU-R2) | Signed evidence / hashes (RQ-13) |
| T9 | Recorder / T5 collusion or self-verification | Name-level only (F1-C) | Group B refused; evaluator S5 | Resolver; independent verifier pool (AD6) |
| T10 | Revocation between signing and verification | Not modelled | Not modelled (interim rule has no time dimension) | RM-E04F I5, §5.5 |
| T11 | Account reassignment / stale identity link | Not modelled | Not modelled; resolver must be evaluated as-of signing time | `resolve(user_id, at)` (RM-E04F §6.1) |
| T12 | TOCTOU between check and gate change | CI evaluates committed state at one revision | Same; gate changes need a reviewed PR | Runtime locking (RM-E04F §8.4) |
| T13 | GitHub login / PR reviewer used as identity | No rule | Repository resolver refuses all inputs incl. logins | — |

## 10. Verification evidence

### 10.1 Committed tests (`docs/governance/rm_e04g_test.go`)

| Test | Asserts |
|---|---|
| `TestRME04GReproduceF1WithCurrentValidator` | Pins current validator levels for F1-A…E; all are UNKNOWN under the interim rule; the adapter never fabricates an account; R6 v1 has no identity field |
| `TestRME04GPersonIndependenceRules` | 24 cases incl. controls, F1-A…H variants, missing roles / verifier, precedence |
| `TestRME04GDisplayNameIsNotEvidence` | Renaming every participant never changes an outcome |
| `TestRME04GRepositoryResolverIsFailClosed` | Names, aliases, phone, e-mail, GitHub login, bots, reviewers, nominee keys → UNKNOWN |
| `TestRME04GGroupBReadinessRule` | Interim rule closed for name-only / alias / second account / self-verify / `SHARED_USER`; open only with person-independent claims and a (synthetic) resolver; forged status and closed gate refused; synthetic records refused in repository mode |
| `TestRME04GRepositoryGroupBNeedsPersonIndependence` | Repository: gate cannot be open without readiness; readiness false today; gate, S4 manifest and R5-SEC-01 tracker entry unchanged |
| `TestRME04GApprovalRecordsCarryOnlyR6V1Fields` | Templates / intake blanks accepted; five smuggled identity fields refused; committed files checked |
| `TestRME04GApprovalHistoryCarriesOnlyR6V1Fields` | Every committed revision of every approval file checked (needs full history) |
| `TestRME04GDocumentsMarkedNotApproved` | This document and the handoff: DRAFT marker, NO-GO, REVIEW PENDING, no invented decision id, no claimed sign-off |

Full suite: `go test -count=1 ./docs/governance/` PASS; with `R5_REQUIRE_PROVENANCE=1` on a full clone PASS.

### 10.2 Mutation checks (throwaway copies; nothing committed; worktree verified clean afterwards)

| # | Mutation | Result |
|---|---|---|
| MU-G1 | Repository resolver returns VERIFIED | Rejected (`…RepositoryResolverIsFailClosed`, `…GroupBReadinessRule`) |
| MU-G2 | `SHARED_USER` treated as verified | Rejected (`…PersonIndependenceRules`, `…GroupBReadinessRule`) |
| MU-G3 | VERIFIED without `person_id` accepted | Rejected (`…PersonIndependenceRules`) |
| MU-G4 | One person may fill two approver roles | Rejected (same) |
| MU-G5 | Verifier may be the recorder | Rejected (`…PersonIndependenceRules`, `…GroupBReadinessRule`) |
| MU-G6 | Verifier may be an approver | Rejected (`…PersonIndependenceRules`) |
| MU-G7 | Delegating principal not reserved | Rejected (same) |
| MU-G8 | Group B readiness skips identity | Rejected (`…GroupBReadinessRule`) |
| MU-G9 | Display name used as identity | Rejected (`…DisplayNameIsNotEvidence`, `…PersonIndependenceRules`, `…GroupBReadinessRule`) |
| MU-G10 | Adapter derives account from display name | Rejected (`…ReproduceF1WithCurrentValidator`) |
| MU-G11 | Unknown-field guard disabled | Rejected (`…ApprovalRecordsCarryOnlyR6V1Fields`) |
| MU-G12 | CONFLICT hidden behind a violation | Rejected (`…PersonIndependenceRules`) |
| MU-G13 | Missing T5 verifier accepted | Rejected (same) |
| MU-G14 | R6 validator name check removed | Rejected (`TestApprovalValidatorRules`, `TestRME04FOnePersonIsOneApprover`, `…ReproduceF1WithCurrentValidator`) |
| MU-R1 | Approval file smuggles `recorded_by.person_ref` | Rejected **only** by `…ApprovalRecordsCarryOnlyR6V1Fields` (F1-I evidence) |
| MU-R2 | R5-SEC-01 forged `APPROVED_VERIFIED` without a record | Rejected (existing R6 / M2 / RM-E04C tests + `…RepositoryGroupBNeedsPersonIndependence`) |
| MU-R3 | Group B gate names M2 ids | Rejected (`TestRME04CProposalsStayOutOfRuntime`, `…RepositoryGroupBNeedsPersonIndependence`) |
| MU-R4 | Nominee `person_ref` / identity resolution claimed | Rejected (`TestRME04FNominationsStayProvisional`) |
| MU-D1 | Handoff status changed from REVIEW PENDING | Rejected (`…DocumentsMarkedNotApproved`) |
| MU-D2 | A review item marked answered | Rejected (same) |
| MU-D3 | DRAFT marker removed from this document | Rejected (same) |
| MU-D4 | Unregistered decision id cited | Rejected (same, and `TestEveryDecisionIDIsInventoried`) |

An earlier MU-G9 variant was a no-op (did not actually introduce name use) and an earlier MU-G10 survived; the
mutation was corrected and the adapter assertion added. Both now reject.

## 11. Roadmap after R5-SEC-01

| Step | Content | Precondition |
|---|---|---|
| 1 | Security / Codex decisions on the handoff items | this PR reviewed |
| 2 | R5-SEC-01 recorded `APPROVED_VERIFIED` through a valid R6 record (human process) | humans |
| 3 | F-PR1: approved R6 identity extension (`person_ref`, `verifier_person_ref`, delegation evidence); extend `rmE04GR6V1Fields` and `rmE04GClaimsFromRecord` | RQ-13 + review items 4, 5 |
| 4 | F-PR2: reviewed resolver replacing `rmE04GRepositoryResolver`, as-of signing time; validator consults it when identity fields are present | step 2, 3 |
| 5 | Decide RT-3 (D-42 grant path) and RT-4 coverage | review item 9b |
| 6 | Shadow governance module (RM-E04F F-PR3+) | separate authorisation |
| — | Production activation | **separate authorisation; NO-GO** |
