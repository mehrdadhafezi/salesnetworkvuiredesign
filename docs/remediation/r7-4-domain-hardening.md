# R7-4 — Fulfillment and HR domain hardening (test evidence)

Status: TEST-ONLY. No production code, migration, flag default, grant or
business rule is changed. Production remains **NO-GO**.

> **Follow-up (separate PR, branch `fix/def-hr-01-reviewer-cycle`):** the one
> defect found here, DEF-HR-01, has a code fix in review. Its status is
> tracked in the [Defect](#defect) section below; it is **not** resolved
> until that PR is independently reviewed and merged.

Baseline: `main` @ `038699df517c918bcf5bb2e7fc3fd862105421a1`.

Scope (R7-A §R7-4): backend + disposable PostgreSQL depth for
`internal/fulfillment/work` (OP-23) and `internal/organization/hrrequest`
(OP-05). The R7-1 / R7-2 / R7-3 browser suites are not duplicated: R7-3
already proves that the production resolver refuses every fulfillment
command; this layer proves the domain rules behind that gate.

**Evidence class.** Every test here calls the domain service with explicit
synthetic actors, or (HTTP test only) the test-only `testSubjects` overlay.
This is **backend-only coverage, not evidence of production authorization**:
no role bundle carries `fulfillment.work.*` or `organization.hr_request.*`
(D-42, D-34 and R5-SEC-01 remain open and unapproved). Provider behaviour
uses the test-only `integrationbiawin.NewSynthetic()` port; nothing reaches a
real provider.

## Files

| File | Layer | Tests |
|---|---|---|
| `internal/integration/biawin/r7_4_fulfillment_hardening_test.go` | integration (PostgreSQL) + concurrent transactions | 7 |
| `internal/integration/legacy/shadow/r7_4_hrrequest_hardening_test.go` | integration (PostgreSQL) + concurrent transactions + HTTP | 6 |
| `internal/integration/legacy/shadow/r7_4_def_hr_01_test.go` | DEF-HR-01 regressions, **default suite** since the fix (previously `r7_4_defects_test.go`, build tag `r74defects`, failing by design) | 2 original + 5 targeted |

All default-suite tests run in the existing CI jobs (`go test ./...`,
`go test -race ./...`, the Slice 14 job for `./internal/integration/biawin/`
and the Slice 11 `HRRequest` job, including its `-race` step). No workflow
changed.

## Assertion standard

- Every refusal: exact `status`, exact `reasons`, exact `blocked_by`,
  `applied=false`, no command id, and an **unchanged database** — an md5
  per table of the `fulfillment`, `integration`, `biawin` and `catalog`
  schemas (fulfillment), or of every HR request table (HR). A refusal stores
  nothing, not even its idempotency key.
- Every success: the exact rows written (case / request row, history event
  with before / after versions, assignment / evidence row, command record,
  outbox event) and nothing else.
- HR: `identity`, `custody`, `cases`, `billing`, `payments`, `finance`,
  `commission`, `integration`, `fulfillment`, `biawin` and the organization
  evidence (`person`, `employment`, `org_assignment`, `position`, `level`,
  `org_unit`) are snapshotted with the R7-2 SHA-256 helper and compared after
  every test (the `commission` schema is not migrated in this environment:
  no table, nothing to change).

## Coverage matrix — fulfillment (`internal/fulfillment/work`)

Existing = before R7-4 (`flow_e2e_test.go`: `TestFulfillmentCommands`,
`TestCompletionGuards`, `TestProviderOperations`). "status only" means the
existing test checked the status but not the reason code or the database.

| # | Scenario | Existing | Added assertion | Expected (current code) | Governance |
|---|---|---|---|---|---|
| A1 | Current supervisor assigns current executor | yes (status) | rows: case, ASSIGNED event (versions, supervisors of the time), assignment, command record; no completion event | SUCCESS, ASSIGNED v2 | — |
| A2 | Supervisor of another service / no scope / executor as assigner | partial | exact reason + unchanged DB | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| A3 | Assignee outside service executor scope (other service, supervisor-only, no scope) | partial | exact reason + unchanged DB | BLOCKED `RECIPIENT_INELIGIBLE` | — |
| A4 | Revoked supervisor / revoked executor (as assignee and as actor) | supervisor only | executor revocation both ways; history of the revoked actor kept | UNAUTHORIZED `ROW_SCOPE_DENIED` / BLOCKED `RECIPIENT_INELIGIBLE` | — |
| A5 | Unknown case / malformed ids / version 0 / short key / no principal / disabled mode | no | `ErrNotFound`, `ErrInvalid` (400), FEATURE_DISABLED | as stated, DB unchanged | — |
| A6 | Reassignment while ASSIGNED | no | previous assignee recorded | SUCCESS | — |
| A7 | Reassignment after work started | no | exact `blocked_by` `OPD-05`; precedes the version check | BLOCKED `POLICY_UNRESOLVED` [OPD-05] | OPD-05 (open) |
| A8 | No-op assignment (same assignee) | no | **observed**: accepted, new event, version bump | SUCCESS | **G-F1** |
| A9 | Missing / ambiguous scope evidence | n/a | catalog `InScope` is boolean; no ambiguous state exists | — | — |
| A10 | HR position used as authority | n/a | fulfillment reads no HR data; authority is catalog scope only (doc + A2) | — | — |
| B1 | NO_ANSWER / FOLLOW_UP / NOTE | NO_ANSWER only | version chain 2→3→4→5, `follow_up_at` kept | SUCCESS IN_PROGRESS | — |
| B2 | Wrong assignee, supervisor substitute, unassigned case | partial | exact reason + unchanged DB | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| B3 | Invalid payloads (unknown action, missing date / note, empty / long note) | no | 400 + unchanged DB | `ErrInvalid` | — |
| B4 | Stale version | status only | reason + current version | CONFLICT `STALE_STATE` | — |
| B5 | Replay same key | **no** (computed, never replayed) | same command id, stored version, DB unchanged | SUCCESS replayed | — |
| B6 | Same key + other payload / other actor | no | never another actor's result | CONFLICT `CONTRACT_INVALID` | — |
| B7 | A refused command does not consume its key | no | no idempotency row; key later succeeds | — | — |
| B8 | Same content, new key | no | a new fact (dedupe is by key) | SUCCESS | — |
| B9 | Action on a closed case | status only | exact | CONFLICT `STALE_STATE` | — |
| B10 | SHADOW mode | partial | command record `applied=false`, case unchanged | SUCCESS not applied | — |
| C1 | Required evidence; row content | yes | row with sha, submitter, command | SUCCESS | — |
| C2 | Type not defined by the service | status only | exact | BLOCKED `CONTRACT_INVALID` | — |
| C3 | Invalid type / sha (short, upper-case, non-hex) / label | no | 400 + unchanged DB | `ErrInvalid` | — |
| C4 | Duplicate (case, type, sha) | no | exact; per-type and per-case independence | CONFLICT `CONTRACT_INVALID` | — |
| C5 | Wrong assignee / supervisor | partial | exact | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| C6 | After completion | no | exact | CONFLICT `STALE_STATE` | — |
| C7 | Replay | no | one row | SUCCESS replayed | — |
| D1 | EXECUTED with evidence | yes | exactly one completion outbox event with the command id | SUCCESS COMPLETED | — |
| D2 | NOT_EXECUTED with reason; from ASSIGNED; reason in history | partial | exact | SUCCESS NOT_EXECUTED | — |
| D3 | Missing / empty / long reason; reason on EXECUTED; unknown outcome | missing reason only | 400 + unchanged DB | `ErrInvalid` | — |
| D4 | Missing evidence (also with stale version) | status only | exact `blocked_by`; precedes version | BLOCKED `COVERAGE_INCOMPLETE` [REQUIRED_EVIDENCE_MISSING] | — |
| D5 | Outcome not approved by the service | no | exact | BLOCKED `CONTRACT_INVALID` | — |
| D6 | Repeated completion / stale If-Match | no | no second event | CONFLICT `STALE_STATE` | — |
| D7 | Financial service (CONSUMES_CREDIT, CHARGES_LEDGER) | no | activation refuses it (no case, D-35); a case delivered at the inbox boundary still cannot complete, any outcome | BLOCKED `POLICY_UNRESOLVED` [D-35, D-38] | D-35, D-38 (open) |
| D8 | Provider service without confirmation: none / PREPARED / disabled port / ACKNOWLEDGED / UNKNOWN / RECONCILIATION_REQUIRED / REJECTED / other case confirmed / no reader | none + PREPARED-less | each state | BLOCKED `DEPENDENCY_UNKNOWN` [PROVIDER_CONFIRMATION_REQUIRED] | Biawin contract |
| D9 | Provider read unavailable | no | error, never SUCCESS, DB unchanged | error | — |
| D10 | CONFIRMED via synthetic signed callback | yes | — | SUCCESS | — |
| D11 | NOT_EXECUTED of a provider service | no | not provider-gated | SUCCESS | — |
| E1 | n concurrent assignments, two supervisors, two assignees | 2 same-assignee | one transition, one event / assignment / record, version 2; losers exact | 1 SUCCESS, rest CONFLICT `STALE_STATE` | — |
| E2 | n concurrent actions, same version | no | one event, version 3 | same | — |
| E3 | n concurrent completions (EXECUTED vs NOT_EXECUTED) | no | one completion event, version 5 | same | — |
| E4 | n × same key + same command | no | executed once, n-1 replays, one record | SUCCESS (replayed) | — |
| E5 | n × same key + different payloads | no | a different payload is never answered with the winner's result | 1 execution, rest CONFLICT `CONTRACT_INVALID` | — |
| E6 | Concurrent intake: one message redelivered + a second message for the same item | partial (sequential) | one case, one CREATED event, one created outbox event, two intake rows | — | — |
| E7 | Idempotency record left in progress | no | never executed blindly | UNKNOWN `OUTCOME_UNKNOWN` | — |

Provider boundary (domain layer, synthetic port): `Resolve` is refused on an
ACKNOWLEDGED operation and on a stale version (`ErrState`), for present
without reference / absent with reference (`ErrInvalid`) and for an unknown
operation (`ErrNotFound`); callbacks: duplicate → `DUPLICATE`, REJECTED after
CONFIRMED → `ALREADY_FINAL`, same callback id with other content →
`ID_CONFLICT`, unknown reference → `UNMATCHED`; the final state never moves;
the synthetic port receives exactly the scripted calls.

## Coverage matrix — HR requests (`internal/organization/hrrequest`)

Existing = `TestHRRequestCommands`, `TestWritesInvariants`, `TestWritesHTTP`
(contact outcome only for capability).

| # | Scenario | Existing | Added assertion | Expected (current code) | Governance |
|---|---|---|---|---|---|
| H1 | Eligible creator; exact rows (reviewer, subject parent, requester snapshots, created event, record, one outbox event) | partial | rows | SUCCESS `pending_review` v1, `LEGACY_WRITEBACK_PENDING` | — |
| H2 | Requester without a parent (manager) | no | no reviewer | SUCCESS `pending_hr` | — |
| H3 | Seller outside hierarchy / sibling supervisor's seller | yes (one) | exact + no write | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| H4 | Requester = subject; subject not a seller | no | exact | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| H5 | Seller / HR as requester | yes | exact `blocked_by` [HR_APPLY_IMPLEMENTATION_BLOCKED, OPD-05] | UNAUTHORIZED `ACTION_DENIED` / BLOCKED | OPD-05 |
| H6 | Unknown subject / requester / destination; unmapped status | partial | exact | BLOCKED `LINKAGE_CONFLICT` | — |
| H7 | Inactive (TERMINATED, `is_active=false`) requester | no | exact | UNAUTHORIZED `ACTION_DENIED` | — |
| H8 | Ambiguous identity (SHARED_USER) of requester / subject | no | exact | BLOCKED `LINKAGE_CONFLICT` | — |
| H9 | Undecidable current flag (subject line, reviewer line) | no | exact | BLOCKED `LINKAGE_CONFLICT` | — |
| H10 | No current reporting line | no | exact (no scope evidence) | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| H11 | Hierarchy cycle (depth > 12) | no | exact | BLOCKED `LINKAGE_CONFLICT` | — |
| H12 | Destination: seller, senior, suspended supervisor | partial | exact | BLOCKED `RECIPIENT_INELIGIBLE` | — |
| H13 | No-op transfer / cross-scope destination | yes (status) | exact `blocked_by` [SD-04] | BLOCKED `POLICY_UNRESOLVED` | SD-04 |
| H14 | Contract errors (upper-case enum, pairing, ids, reason, key, correlation) | no | 400 + no write | `ErrInvalid` | — |
| H15 | Replay / same key other body / duplicate open request (two creators) / other type same seller | duplicate only | exact | replay / CONFLICT `CONTRACT_INVALID` / CONFLICT `STALE_STATE` / SUCCESS | — |
| H16 | Request about a TERMINATED seller | no | **observed**: accepted | SUCCESS | **G-H1** |
| H17 | SHADOW create / decide; disabled | no | recorded not applied; FEATURE_DISABLED | — | — |
| H18 | Correct reviewer step; next reviewer = reviewer's parent; step event | yes (status) | rows | SUCCESS `pending_review` v2 | — |
| H19 | Wrong reviewer, requester, subject | partial | exact | UNAUTHORIZED `ROW_SCOPE_DENIED` | — |
| H20 | Reviewer resigned / ambiguous / undecidable next reviewer | no | exact | UNAUTHORIZED `ACTION_DENIED` / BLOCKED `LINKAGE_CONFLICT` | — |
| H21 | Stale version | status only | exact + current version | CONFLICT `STALE_STATE` | — |
| H22 | Replay / same key other decision / duplicate approval by previous reviewer | no | exact | replay / CONFLICT `CONTRACT_INVALID` / UNAUTHORIZED | — |
| H23 | pending_hr: approve, reject, HR decision | approve only | exact `blocked_by` [HR_APPLY_IMPLEMENTATION_BLOCKED, OPD-05, SD-04] | BLOCKED `POLICY_UNRESOLVED` | OPD-05, SD-04 |
| H24 | Rejection: terminal, notes, reviewer cleared; a new request is then possible | status only | exact | SUCCESS `rejected`; then CONFLICT `STALE_STATE` | — |
| H25 | Hierarchy changed between creation and decision | no | **observed**: recorded reviewer still decides, the new parent cannot; next reviewer from the live hierarchy | SUCCESS / UNAUTHORIZED | **G-H2** |
| H26 | No apply effects after approval to pending_hr and after rejection | md5 of 4 tables | all schemas above; seller still ACTIVE under the same parent; no ACTIVE grant; only `organization.hr_request_*` events; no applied status | — | OPD-05, SD-04 |
| H27 | Capability at HTTP (test-only overlay): none / unrelated / anonymous → 403; seller holding it → 403 ACTION_DENIED; create ≠ review; review without row scope → 403 ROW_SCOPE_DENIED; HR final → BLOCKED; duplicate → 409 | contact outcome only | exact | as stated | D-42 (overlay only) |
| H28 | n concurrent creates (two creators) | no | one request | 1 SUCCESS, rest CONFLICT `STALE_STATE` | — |
| H29 | n × same key same body / different bodies | no | executed once; different body never answered with winner | replay / CONFLICT `CONTRACT_INVALID` | — |
| H30 | n concurrent approve / reject on v1 | no | one event, version 2; loser answer depends on the winner | 1 SUCCESS; losers `STALE_STATE`, `ROW_SCOPE_DENIED` or HR-apply BLOCKED | **G-H3** |
| H31 | Person-level separation of duties (one person, two accounts) | n/a | — | **BLOCKED — not testable** (no person identity model) | R5-SEC-01 |
| H32 | Self-parent / cycle evidence makes the requester their own reviewer | no | defect regression, default suite after the fix | BLOCKED `LINKAGE_CONFLICT` (was SUCCESS on the R7-4 baseline) | **DEF-HR-01** (fix in review) |

## Defect

**DEF-HR-01 — HR reviewer resolution accepts SELF_PARENT / CYCLE evidence.**
Severity: Medium. Domain: organization / HR requests (OP-05).

| Status | State |
|---|---|
| FIXED IN CODE | **Yes**, on branch `fix/def-hr-01-reviewer-cycle` (separate PR). Not merged until independently reviewed. |
| VERIFIED BY TESTS | **Yes**: 2 original regressions + 5 targeted tests in the default suite, disposable PostgreSQL; 8 of 8 mutations killed (below). |
| NOT DEPLOYED | **Not deployed anywhere.** Production remains NO-GO. |
| REMAINING GOVERNANCE BLOCKERS | R5-SEC-01 (person-level SoD), OPD-05 / SD-04 (HR apply), G-H2 (snapshot vs. live reviewer). See below. |

**Defect (as found on the R7-4 baseline `038699d`).**
- Reproduction then: `go test -tags r74defects -count=1 -run TestR74Defect ./internal/integration/legacy/shadow/`.
  Both tests failed on `main` @ `038699d` and again @ `3406843` before the fix.
- With an imported reporting line that points at its own employment
  (SELF_PARENT), the requester was recorded as `current_reviewer_user_id`
  and could record `approved_step` decisions on their own request
  indefinitely. With a two-person cycle, the request returned to the
  requester after one step.
- Impact: no apply exists (OPD-05 / SD-04 block every final step), so no
  employment, hierarchy or permission changed. The defect corrupted the
  decision audit trail and defeated the requester ≠ reviewer property that
  the normal hierarchy provides.
- Existing contract: `hrrequest` states "ambiguous or missing hierarchy
  evidence is BLOCKED, never guessed", and the organization import already
  classifies SELF_PARENT and CYCLE as `CONFLICT` findings
  (`internal/organization/domain/hierarchy.go`). The fix follows that
  contract and adds no business rule.

**Root cause.** `nextReviewer` returned the actor's direct current parent
without looking at the chain above it. The direct parent of a self-parent
line is the requester. In a cycle, each step moves to the next parent until
the request reaches the requester again.

**Fix (`internal/organization/hrrequest/hrrequest.go`, `nextReviewer`).**
The actor's current REPORTS_TO chain is walked with the existing
`maxHierDepth` (12) bound. The direct parent counts as evidence only if:
- the chain reaches a root within that bound; and
- no ancestor is the request's requester, compared by user.

Otherwise the result is the existing `BLOCKED / LINKAGE_CONFLICT`.
- A self-parent line or a cycle never reaches a root, so it ends at the
  bound, the same rule `descendant` already uses.
- `DecideRequest` reads `requested_by_user_id` for the check.
- No new reason code, status, policy, permission or apply was added.

This differs from the fix first proposed above, which changed `parentOf`
and tracked visited employments. Both of those guards proved redundant:
weakening either changed no outcome, because the bound and the requester
check already cover them. They were dropped to keep the change minimal.

| Situation | Before | After |
|---|---|---|
| Requester's own line is self-parent | SUCCESS, reviewer = requester | BLOCKED `LINKAGE_CONFLICT` |
| Self-parent above the requester | SUCCESS; the request stalls at that reviewer | BLOCKED `LINKAGE_CONFLICT` |
| Cycle through the requester | SUCCESS; returns to the requester after one step | BLOCKED `LINKAGE_CONFLICT` at creation |
| Cycle above the requester, not involving them | SUCCESS; reviewers alternate forever | BLOCKED `LINKAGE_CONFLICT` |
| Hierarchy changed after creation, so the next reviewer would be the requester | SUCCESS, routed to the requester | Approve: BLOCKED, nothing written. Reject: unchanged (terminal) |
| Another (non-PRESENT) employment of the requester above the chain | SUCCESS | BLOCKED `LINKAGE_CONFLICT` |
| Chain of 13 or more ancestors | SUCCESS | BLOCKED `LINKAGE_CONFLICT` (same bound as `descendant`) |
| Valid chain, step progression, root-level requester → `pending_hr`, rejection | unchanged | unchanged |

**Tests (default suite, `r7_4_def_hr_01_test.go`).**
- `TestR74DefectHRSelfParentReviewer` and `TestR74DefectHRCycleReturnsToRequester`: the original assertions, unchanged.
- `TestDEFHR01CreationEvidence`, `TestDEFHR01HierarchyChangesBeforeDecision`, `TestDEFHR01ValidChainsUnchanged`, `TestDEFHR01DepthBoundary` (12 accepted / 13 refused) and `TestDEFHR01RequesterAboveOwnChain`.
- Every refusal goes through `refusedCreate` / `refusedDecide`. These assert that the request, event, command-record, outbox and idempotency tables are byte-identical afterwards.
- Each test ends with `noSideEffects`, which checks that identity, custody, cases, billing, payments, finance, commission, integration, fulfillment, biawin and the organization evidence are unchanged.
- Fixtures are restored, and checked restored, by md5.

**DEF-HR-01 mutation evidence.** Each change was made locally, one at a
time, and never committed.

| Mutation of the fix | Result | Failing tests |
|---|---|---|
| F1 requester check removed | killed | HierarchyChangesBeforeDecision |
| F2 depth bound removed (1000) | killed | DepthBoundary |
| F3 depth bound 11 | killed | DepthBoundary |
| F4 depth bound 13 | killed | DepthBoundary |
| F5 chain not walked (direct parent only) | killed | CreationEvidence, HierarchyChangesBeforeDecision, DepthBoundary, RequesterAboveOwnChain |
| F6 decision ignores the requester | killed | HierarchyChangesBeforeDecision |
| F7 creation ignores the requester | killed | RequesterAboveOwnChain |
| F8 the original code (`main`) | killed | both originals + 4 targeted |

**Not resolved by this fix.**
- Person-level separation of duties (one person holding two accounts) remains **BLOCKED** under R5-SEC-01.
- The check compares user IDs and does not invent a person model.
- The reviewer snapshot (G-H2) is unchanged.
- HR apply stays implementation-blocked (OPD-05 / SD-04).

## Governance questions (behaviour pinned as observed, not claimed correct)

| Id | Observation | Decision needed |
|---|---|---|
| G-F1 | Assigning a case to its current assignee is accepted (new event, version bump) | Should a no-op assignment be refused? (no contract today) |
| G-H1 | A request about an already TERMINATED / RESIGNED seller is accepted (the subject's employment status is not checked) | Must the subject be active? |
| G-H2 | The reviewer is the snapshot of the previous transition; after a hierarchy change the recorded reviewer still decides, while the next reviewer is computed from the live hierarchy | Snapshot vs. live reviewer (related to OPD-05 / SD-04) |
| G-H3 | A concurrent losing decision may be answered `ROW_SCOPE_DENIED` (reviewer moved) instead of `STALE_STATE` | Is the refusal order (reviewer before version) the intended contract? |

## Mutation evidence

Each guard below was weakened locally (one at a time, never committed), the
targeted suite was run, and the file was restored from git.

| Mutation (production guard disabled) | Result | Failing tests |
|---|---|---|
| M1 supervisor row scope (assign) | killed | AssignmentScope |
| M2 assignee executor scope | killed | AssignmentScope |
| M3 executor must be the assignee | killed | OperationalActions, Evidence, Completion |
| M4 expected-version check | killed | AssignmentScope, OperationalActions, Completion, Concurrency |
| M5 required evidence | killed | Completion |
| M6 duplicate evidence | killed | Evidence |
| M7 provider confirmation | killed | ProviderBoundary |
| M8 financial behaviour block (D-35 / D-38) | killed | Completion |
| M9 OPD-05 reassignment block | killed | AssignmentScope |
| M10 idempotency request-hash comparison | killed | fulfillment OperationalActions, Concurrency; HR Creation, Decisions, Concurrency |
| M11 HR hierarchy (descendant) | killed | HRRequestR74Creation, HierarchyEvidence |
| M12 HR depth / cycle → not evidence | killed | HierarchyEvidence |
| M13 HR recorded-reviewer check | killed | Decisions, HTTPCapability |
| M14 HR expected-version check | killed | Decisions |
| M15 HR requester ≠ subject | **survived — equivalent** | the requester must hold a creator position and the subject must be a seller, so one employment cannot be both; the `hr_request_not_self` CHECK is a third layer |
| M16 HR pending_hr block | killed | Decisions, HTTPCapability |
| M17 HR active requester | killed | HierarchyEvidence |
| M18 HR no-op transfer (SD-04) | killed | Creation |
| M19 HR duplicate open-request guard | **survived — equivalent** | the unique index `hr_request_one_open` returns the same CONFLICT `STALE_STATE` (defence in depth, both layers asserted by behaviour) |

17 of 19 killed; the 2 survivors cannot change observable behaviour.

## Remaining blockers

- D-42, D-34, R5-SEC-01: no positive work-case or HR command is reachable
  with the production resolver; person-level SoD not provable.
- OPD-05, SD-04: HR apply implementation-blocked; reassignment after start
  blocked.
- D-35, D-38: financial services never complete.
- Real Biawin provider contract absent: confirmation exists only through the
  synthetic port.
- HR and Finance runtime remain NOT LIVE VERIFIED.
