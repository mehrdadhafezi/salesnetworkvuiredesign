# RM-E04B / A4 — API operation scope registry and CI enforcement (NT-22)

| | |
|---|---|
| Mode | Security architecture regression hardening (tests + test data only) |
| Base `main` | `c8a490c3a40c5a08266573c41b68b156a1d1a49e` (PR #73 merged; verified with `git rev-parse origin/main`) |
| Branch | `test/rm-e04b-operation-scope-inventory` |
| Plan row | [`../implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md) A4 |
| Test | [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) NT-22 (with NT-07) |
| Inventory source | [`../security/rm-e04a-seller-leak-surface-audit.md`](../security/rm-e04a-seller-leak-surface-audit.md) §3, re-derived against HEAD |
| Runtime change | **None.** No endpoint, OpenAPI contract, evaluator, row scope, grant, role, capability, flag, migration or command is changed. |
| Human approval | Code review (plan A4). Not obtained; not claimed. |
| Production | **NO-GO** |

Group A is complete with this PR (A1 scope assertion, A2 no-store, A3 submission non-enumeration, A4 inventory guard). Group B is **not** started.

---

## 1. What was added

| File | Kind | Purpose |
|---|---|---|
| `internal/authorization/http/testdata/operation_scope_registry.yaml` | test data | The registry: one entry per OpenAPI operation (105). Read by tests only. |
| `internal/authorization/http/operation_registry_test.go` | test | NT-22 validator, evidence collection, command fail-closed check, negative controls. |
| `internal/authorization/http/scope_nt07_test.go` | test (refactor) | NT-07 now reads the shared spec inventory (`specOperations`) and derives its exemptions and own-read list from the registry instead of two hand-kept maps. All NT-07 assertions are unchanged. |
| `cmd/crm-api/operation_registry_flags_test.go` | test | Each registry `feature_flag` equals the flag the composed guard uses (`capabilityFlags`, `SessionAuthFlag`). |
| this document | docs | |

No file outside `_test.go`, `testdata/` and `docs/` changed.

## 2. Inventory at HEAD (recalculated)

Enumerated from `api/openapi/openapi.bundled.yaml` (`paths` × methods) by the test itself, not by hand:

| | Count |
|---|---|
| Operations | **105** |
| `GET` | **92** |
| `POST` | **13** |
| `PUT` / `PATCH` / `DELETE` | 0 |
| Registry entries | **105** (0 unregistered, 0 orphan, 0 duplicate) |

Unchanged from the RM-E04A audit. `TestNT22Inventory` pins these numbers and the category totals below, so a change is always an explicit, reviewed edit.

### Classification totals

| Category | Scope class | Capability | NT-07 | Count |
|---|---|---|---|---|
| `ORG_READ` — capability-guarded organization read | `ORGANIZATION` | required | REQUIRED | **79** |
| `OWN_READ` — caller-owned read | `OWN:<basis>` | required | REQUIRED | **7** |
| `REVIEW_READ` — reviewer (Finance review) read | `ORGANIZATION` | required | REQUIRED | **2** |
| `PUBLIC_HEALTH` — public probe | `NOT_APPLICABLE` | forbidden | EXEMPT | **2** |
| `SESSION_READ` — caller's own session / decisions | `SESSION_SELF` | forbidden | EXEMPT | **2** |
| `SESSION_COMMAND` — login / logout | `SESSION_SELF` | forbidden | NOT_A_READ | **3** |
| `BUSINESS_COMMAND` — command with own predicate | `COMMAND:<basis>` | required | NOT_A_READ | **10** |
| **Total** | | | | **105** |

Reads: 92 = 79 + 7 + 2 + 2 + 2. NT-07-covered (capability-guarded) GETs: **88** = 79 + 7 + 2 (same as A1). Commands: 13 = 3 session + 10 business. No `POST` is classified as a read, and no command is classified as scope-free except the three session commands on the approved list.

## 3. Registry format

YAML, decoded strictly (`KnownFields`: a misspelt key is an error). One entry per operation:

| Field | Meaning | How it is checked |
|---|---|---|
| `operation_id`, `method`, `path` | identity of the operation | both directions against the contract |
| `category` | closed list (§2) | category ⇒ method, scope kind, capability presence, subject, NT-07 value |
| `capability` | the **gate** capability | known to `identityapi.KnownCapability` **and** equal to the capability the handler actually asks (observed) |
| `also_requires` | further gate capabilities, in order | equal to the observed gate chain |
| `field_capabilities` | capabilities that only mask fields | known capabilities; not a gate; reviewer-verified |
| `feature_flag` | flag of the composed guard | equal to `capabilityFlags[capability]` (or `SessionAuthFlag`) in `cmd/crm-api` |
| `scope` | `ORGANIZATION` · `OWN:<basis>` · `COMMAND:<basis>` · `SESSION_SELF` · `NOT_APPLICABLE` | syntax, category match, basis in a closed list |
| `subject` | `NONE` · `SESSION_PRINCIPAL` · `ANONYMOUS_CREDENTIAL_EXCHANGE` · `AUTHENTICATED_PRINCIPAL` · `AUTHENTICATED_USER_ID` | allowed per category |
| `enforcement` | repository file holding the strict handler | file must define `func (…) <OperationId>(` |
| `nt07` | `REQUIRED` · `EXEMPT` · `NOT_A_READ` | per category; a guarded GET can never be EXEMPT; must be in NT-07's guarded set |
| `exemption_reason` | required for every capability-free entry | must equal the approved list in the test |
| `unproven_user` | `OWN_READ` only | equal to the **observed** answer (§5) |
| `row_authorization`, `separation_of_duties` | `BUSINESS_COMMAND` only | required, non-empty |
| `tests` | `file_test.go:TestName` | file exists and defines `func TestName(t *testing.T)`; required where NT-07 does not apply |
| `governance` | decision references from the audit | informational |

### Scope classes

| Class | Meaning |
|---|---|
| `ORGANIZATION` | Allowed only for an `ORGANIZATION` decision (A1); rows are organization-wide. An unproven user still reaches the query (the handler does not filter by caller). |
| `OWN:<basis>` | Allowed only for an `ORGANIZATION` decision (A1, NT-07 applies), **and** rows are filtered to the caller by an existing server-side predicate named by `<basis>`. |
| `COMMAND:<basis>` | A command: capability + feature flag at the HTTP layer, then the row / actor predicate named by `<basis>` inside the command service. GET row-scope (NT-07) does not apply; command row authorization is **not** treated as GET row scope. |
| `SESSION_SELF` | Acts on, or returns, only the caller's own session or decisions. No module rows. |
| `NOT_APPLICABLE` | No business data at all (public health probes). |

Ownership bases (existing predicates only; no Seller policy is introduced):

| Basis | Existing predicate | Operations |
|---|---|---|
| `commission_holder_user_id` | `holder_user_id = caller` (commission query) | `listMyCommissionEarnings` |
| `fulfillment_service_scope` | catalog service scope of the caller (not Seller) | `listFulfillmentWorkCases`, `getFulfillmentWorkCase` |
| `hr_request_participant` | requester / reviewer / actor = caller (HR request reads) | `listHRRequests`, `getHRRequest` |
| `payment_batch_actor_user_id` | `actor_user_id = caller` (payments read) | `getPaymentBatch` |
| `invoice_seller_and_issuer_user_id` | `ownsTarget`: seller = issuer = caller (A3) | `getPaymentSubmission` |

`getCaseContactOutcomes` stays `ORGANIZATION`: only its `allowed_actions` use a holder check; its data is not caller-scoped (audit: PARTIAL). The test confirms an unproven user reaches its query.

## 4. Command-side classification (13 POST)

| Operation | Capability | Feature flag | Scope / row authorization | SoD | Command tests (referenced, existence-checked) | Governance |
|---|---|---|---|---|---|---|
| `loginWithPassword` | — | `identity.session_auth` | `SESSION_SELF` (credential exchange) | — | `TestHTTPLoginCookieAndSession`, `TestHTTPLoginFailures` | D-34 |
| `logout` | — | `identity.session_auth` | `SESSION_SELF` | — | `TestHTTPCSRF`, `TestHTTPConcurrentLogout` | D-34 |
| `logoutAll` | — | `identity.session_auth` | `SESSION_SELF` | — | `TestHTTPCSRF`, `TestHTTPConcurrentLogout` | D-34 |
| `recordContactOutcome` | `cases.contact_outcome.record` | `cases.nonfinancial_write` | `COMMAND:case_current_holder` — current custody holder only; unknown holder fails closed | not enforced in command | `TestContactOutcomeCommand`, `TestWritesHTTP` | M2 rec. 1 |
| `createHRRequest` | `org.hr_request.create` | `organization.hr_request_write` | `COMMAND:hr_subject_descendant_of_actor` — creator employment; subject a current REPORTS_TO descendant; HR creator and apply BLOCKED | **applies**: next reviewer never the requester (DEF-HR-01) | `TestHRRequestCommands`, `TestHRRequestR74Creation`, `TestHRRequestR74HTTPCapability` | OPD-05 |
| `decideHRRequest` | `org.hr_request.review` | `organization.hr_request_write` | `COMMAND:hr_request_next_reviewer` — actor = `next_reviewer_user_id`; HR finalisation BLOCKED | **applies**: snapshotted reviewer, never the requester | `TestHRRequestR74Decisions`, `TestHRRequestR74NoApplyEffects`, `TestR74DefectHRSelfParentReviewer`, `TestDEFHR01HierarchyChangesBeforeDecision` | OPD-05 |
| `stagePaymentUpload` | `payments.upload.stage` | `payments.manual_write` | `COMMAND:upload_actor_user_id` — bound to the actor | not enforced in command | `TestPaymentsHTTP`, `TestUploadRevivalRace` | M2 rec. 1 |
| `submitPaymentBatch` | `payments.batch.submit` | `payments.manual_write` | `COMMAND:invoice_seller_and_issuer_user_id` — seller = issuer = actor; NULL never owns | not enforced in command (Finance review is a separate read surface; no review command exists) | `TestPaymentBatchCommand`, `TestOwnsTargetMatchesCommandPredicate` | OPD-10 |
| `assignFulfillmentWorkCase` | `fulfillment.work.assign` | `fulfillment.work_write` | `COMMAND:fulfillment_supervisor_service_scope` — actor supervisor, assignee executor; in-progress reassignment BLOCKED | not enforced in command | `TestR74FulfillmentAssignmentScope`, `TestOperationsHTTP` | GAP-06 |
| `recordFulfillmentWorkAction` | `fulfillment.work.execute` | `fulfillment.work_write` | `COMMAND:fulfillment_assignee_executor_scope` | not enforced in command | `TestR74FulfillmentOperationalActions` | GAP-06 |
| `attachFulfillmentWorkEvidence` | `fulfillment.work.execute` | `fulfillment.work_write` | `COMMAND:fulfillment_assignee_executor_scope` | not enforced in command | `TestR74FulfillmentEvidence` | GAP-06 |
| `completeFulfillmentWorkCase` | `fulfillment.work.execute` | `fulfillment.work_write` | `COMMAND:fulfillment_assignee_executor_scope` | not enforced in command | `TestR74FulfillmentCompletion`, `TestR74FulfillmentOutcomeUnknownIsNotRetriedBlindly` | GAP-06 |
| `resolveProviderOperation` | `integration.provider_operation.reconcile` | `integration.provider_reconcile` | `COMMAND:no_row_predicate` — **UNRESOLVED**: capability + proven user id only (refused `ROW_SCOPE_DENIED` without one); evidence-only transition | not enforced in command | `TestR74FulfillmentProviderBoundary`, `TestOperationsHTTP` | GAP-06 |

Subject: every business command requires a proven user id (`AUTHENTICATED_USER_ID`); the services refuse an empty actor (`cases/command/contact_outcome.go`, `payments/command/upload.go`, `fulfillment/work/work.go`, `organization/hrrequest`, `providerops/http`). This column is reviewer-verified (§7).

Usage: every write family runs through `rollout.FromFlags`; all write flags are off by default, so every command answers `BLOCKED` / `TARGET_WRITE_DISABLED` without a data path when off (observed by `TestNT22CommandsFailClosed`). Actual use is blocked by the governance references in the table. Nothing here enables a write.

## 5. Exemptions and their justification

The approved list lives in Go (`approvedCapabilityFree`), not in the registry. A registry entry may be capability-free only if it is on this list with the **same category and the same reason**; adding to the list is a security change that needs review.

| Operation | Category | Reason (exact) |
|---|---|---|
| `getHealthLive` | `PUBLIC_HEALTH` | public liveness probe; no data |
| `getHealthReady` | `PUBLIC_HEALTH` | public readiness probe; no data |
| `getSession` | `SESSION_READ` | session-only: the caller's own principal and feature switches |
| `getEffectiveAccess` | `SESSION_READ` | session-only: the caller's own decisions, from the same evaluator |
| `loginWithPassword` | `SESSION_COMMAND` | credential exchange; creates only the caller's own session |
| `logout` | `SESSION_COMMAND` | ends only the caller's own session |
| `logoutAll` | `SESSION_COMMAND` | ends only the caller's own sessions |

NT-07 exemptions = the four `GET` entries above (unchanged from A1). Each one is independently testable: the evidence harness proves none of them asks the guard for a capability (`getEffectiveAccess` is composed and asks none; the other six are served by the API server, not a business handler). The referenced tests (`TestHealth`, `TestSessionWithDevStub`, `TestHTTPCSRF`, …) cover their own behaviour, and NT-17 covers their `no-store` headers.

## 6. How coverage is enforced (mechanically verified)

`TestNT22RegistryMatchesContract` runs `validate(spec, registry, evidence)`. The **evidence** is collected from the same composed NIL-dependency server as NT-07 (`compose` in `scope_nt07_test.go`), never from the registry:

1. **Capability actually checked.** A recording deny-all decider is served every one of the 105 operations (with synthetic path values, `{}` bodies, `If-Match` and `Idempotency-Key`). Each of the 101 business-handler operations asks for exactly one capability; the registry `capability` must equal it.
2. **Gate chain.** For every guarded operation, the gates found so far are allowed and everything else is denied; while the answer is `403 ACTION_DENIED`, the newly denied capability is the next gate. Observed: only `getInvoiceTotals` has a second gate (`payments.attempt.read`), declared as `also_requires`. Field-masking capabilities never refuse, so they are never mistaken for gates.
3. **Capability-free is real.** An entry without a capability fails (`GUARDED_AS_FREE`) if its handler asks for one; an entry with one fails (`UNGUARDED_HANDLER`) if its handler asks for none.
4. **Caller dependence.** Every guarded GET is served an allowed `ORGANIZATION` decision for a subject **without** a proven user id. Own reads must match their declared `unproven_user`; organization reads must reach their query (`CALLER_FILTERED_AS_ORG` otherwise, meaning "classify it OWN"). Observed:
   - `REFUSED_ROW_SCOPE_DENIED`: `listMyCommissionEarnings`, `listFulfillmentWorkCases` (these drive `TestNT07MissingIdentity`);
   - `NOT_FOUND_WITHOUT_QUERY`: `getHRRequest`, `getPaymentBatch`, `getPaymentSubmission`;
   - `EMPTY_WITHOUT_QUERY`: `listHRRequests`;
   - `QUERY_BEFORE_IDENTITY`: `getFulfillmentWorkCase` (existing residual risk R-2, recorded honestly; a label claiming otherwise fails);
   - all 81 `ORGANIZATION` reads: `QUERY_BEFORE_IDENTITY` (no caller filter).
5. **NT-07 coverage.** Every guarded GET must be `nt07: REQUIRED` and must be in NT-07's `guardedOps` set. NT-07 itself derives its exemptions from the registry through `exemptionsOf`, which **refuses** any exemption that is not on the approved list. A label alone therefore cannot remove an operation from NT-07: both NT-07 and NT-22 fail.
6. **References exist.** `enforcement` must hold the handler method; every `tests` reference must name an existing `func TestX(t *testing.T)` in an existing `_test.go` file.
7. **Feature flags.** `TestOperationRegistryFeatureFlags` (in `cmd/crm-api`, the only package that can see `capabilityFlags`) checks every flag.
8. **Commands fail closed.** `TestNT22CommandsFailClosed`: for all 10 business commands, deny ⇒ `403 ACTION_DENIED`; the real evaluator with the anonymous subject ⇒ `403 ACTION_DENIED`; feature off ⇒ `200` without entering the data path and never `SUCCESS`. Any `POST` that is not a business command must be an approved session command.

### NT-07 integration

* One inventory: `specOperations` (bundled contract) feeds NT-07's `getOperations` and NT-22. The 105 / 92 / 88 reconciliation checks in NT-07 are unchanged.
* The two hand-kept NT-07 maps (`notScopeGuarded`, `ownRowFiltered`) are gone; both are derived from the registry. `TestNT07MissingIdentity` fails if the derived own list is empty (no vacuous pass).
* All NT-07 tests (`Inventory`, `PositiveControl`, `UnsupportedScopeRefused`, `DeniedStaysDenied`, `Anonymous`, `FeatureDisabled`, `GuardError`, `MissingIdentity`, `OwnReadSemantics`) run unchanged against the same 88 guarded GETs.

## 7. Limits of automated verification

| Claim | Status |
|---|---|
| OpenAPI ⇄ registry identity (id, method, path), uniqueness | **Mechanically verified** |
| Gate capability and gate chain per operation | **Mechanically verified** (observed) |
| Capability is a known registry capability | **Mechanically verified** |
| Feature flag per operation | **Mechanically verified** (`capabilityFlags`, `SessionAuthFlag`) |
| GET refuses unsupported scope before any query (NT-07) | **Mechanically verified** |
| Own-read behaviour for an unproven user; organization reads do not filter by caller | **Mechanically verified** (observed) |
| Commands deny / anonymous / feature-off fail closed | **Mechanically verified** |
| Enforcement file holds the handler; referenced tests exist | **Mechanically verified** (existence, not content) |
| Exemptions only from the approved list | **Mechanically verified** |
| The **ownership basis** of an OWN read is the predicate named (e.g. `holder_user_id = caller`) | **Reviewer-verified** (code references in §3); the harness proves caller-dependence, not the exact SQL predicate |
| Command row / actor predicates, subject requirement, separation of duties | **Reviewer-verified** here; exercised by the referenced PostgreSQL integration tests, which this guard only checks for existence |
| `field_capabilities` mask fields only | **Reviewer-verified** (denial does not refuse, which the gate-chain search confirms) |
| A referenced test actually asserts what the registry says | **Reviewer-verified** |

## 8. NT-22 negative controls

`TestNT22NegativeControls` first requires the real registry to validate clean, then mutates **in-memory copies** of the contract inventory or the registry (never the committed files) and asserts the named failure codes. **44** controls, all detected:

| # | Drift | Detected as |
|---|---|---|
| 1 | synthetic `GET` added to the contract, not the registry | `UNREGISTERED` |
| 2 | synthetic `POST` added to the contract, not the registry | `UNREGISTERED` |
| 3 | registry entry deleted | `UNREGISTERED` |
| 4 | registry operationId changed | `UNREGISTERED` + `ORPHAN` |
| 5 | contract operationId changed | `UNREGISTERED` + `ORPHAN` |
| 6 | registry path changed | `PATH_MISMATCH` |
| 7 | contract path changed | `PATH_MISMATCH` |
| 8 | registry method changed | `METHOD_MISMATCH` + `CATEGORY_METHOD` |
| 9 | duplicate operationId | `DUPLICATE` |
| 10 | bogus capability | `UNKNOWN_CAPABILITY` + `CAPABILITY_MISMATCH` |
| 11 | known but wrong capability (label vs handler) | `CAPABILITY_MISMATCH` |
| 12 | second gate dropped | `GATE_MISMATCH` |
| 13 | second gate invented | `GATE_MISMATCH` |
| 14 | capability missing | `MISSING_CAPABILITY` |
| 15 | bogus field capability | `UNKNOWN_CAPABILITY` |
| 16 | invalid scope label (`SELLER_OWN`) | `INVALID_SCOPE` |
| 17 | scope absent | `INVALID_SCOPE` |
| 18 | `OWN:` without basis | `OWN_WITHOUT_BASIS` |
| 19 | bare `OWN` | `OWN_WITHOUT_BASIS` |
| 20 | `OWN` with an invented basis | `INVALID_SCOPE` |
| 21 | unknown category | `UNKNOWN_CATEGORY` |
| 22 | protected operation classified `NOT_APPLICABLE` | `GUARDED_AS_FREE` + `UNAPPROVED_EXEMPTION` + `NT07_MISMATCH` |
| 23 | operation excluded from NT-07 | `NT07_MISMATCH` |
| 24 | exemption without reason | `EXEMPTION_WITHOUT_REASON` |
| 25 | exemption reason changed without approval | `UNAPPROVED_EXEMPTION` |
| 26 | command treated as scope-free session endpoint | `GUARDED_AS_FREE` + `UNAPPROVED_EXEMPTION` |
| 27 | command treated as public health endpoint | `CATEGORY_METHOD` + `GUARDED_AS_FREE` + `UNAPPROVED_EXEMPTION` |
| 28 | command treated as organization read | `CATEGORY_METHOD` |
| 29 | command with unknown authorization basis | `INVALID_SCOPE` |
| 30 | command without row authorization | `MISSING_ROW_AUTHORIZATION` |
| 31 | command without SoD statement | `MISSING_SOD` |
| 32 | command without test references | `MISSING_TESTS` |
| 33 | command without feature flag | `MISSING_FEATURE_FLAG` |
| 34 | reference to a non-existent test | `TEST_NOT_FOUND` |
| 35 | enforcement file not holding the handler | `ENFORCEMENT_NOT_FOUND` |
| 36 | own read relabelled organization read | `CALLER_FILTERED_AS_ORG` |
| 37 | organization read relabelled own read | `UNPROVEN_USER_MISMATCH` |
| 38 | residual risk R-2 hidden by a label | `UNPROVEN_USER_MISMATCH` |
| 39 | invalid subject | `INVALID_SUBJECT` |
| 40 | unapproved NT-07 exemption fed to NT-07's derivation | `exemptionsOf` error (NT-07 would otherwise skip the operation) |
| 41 | NT-07 coverage gap (operation missing from NT-07's set) | `NT07_UNCOVERED` |
| 42 | entry deleted from the committed file **bytes** (parsed copy) | `UNREGISTERED` |
| 43 | misspelt registry key | strict-decode error |
| 44 | schema version drift | `SCHEMA` |

### CI failure evidence (disposable copies, deleted afterwards)

On a scratch copy of the tree (never the real contract), a synthetic `GET /api/v1/nt22/synthetic` (`listNt22Synthetic`) was added to the copy's bundled contract:

```
--- FAIL: TestNT22RegistryMatchesContract
    UNREGISTERED listNt22Synthetic: GET /api/v1/nt22/synthetic has no registry entry
--- FAIL: TestNT22Inventory
    contract inventory 106 operations map[GET:93 POST:13], want 105 (92 GET, 13 POST)
--- FAIL: TestNT07Inventory / TestNT07PositiveControl / TestNT07UnsupportedScopeRefused …
    ListNt22Synthetic: got 404  [], want 403 UNAUTHORIZED [ROW_SCOPE_DENIED]
```

Adding a self-declared `ORG_READ` registry entry for it (no handler) still fails:

```
ENFORCEMENT_NOT_FOUND listNt22Synthetic: internal/cases/http/http.go has no match for …ListNt22Synthetic\(
UNGUARDED_HANDLER listNt22Synthetic: declared ORG_READ but the handler asks the guard for no capability
```

Deleting the `listInvoices` entry from a copy of the registry: `UNREGISTERED listInvoices` (NT-22) and `registry has 104 entries, want 105` (`cmd/crm-api`). Changing one `feature_flag` in a copy: `stagePaymentUpload: registry feature_flag "payments.manual_read", guard flag of payments.upload.stage is "payments.manual_write"`.

## 9. How CI detects an unclassified operation

No new workflow. `go-ci.yml` already triggers on `api/**`, `internal/**` and `cmd/**` and runs `go test -count=1 ./...` and `go test -count=1 -race ./...`; both include `./internal/authorization/http/` and `./cmd/crm-api/`. Its Slice 4 and Slice 7 verbose steps also run these packages. A contract change that adds, removes, renames or moves an operation without a matching registry entry fails `TestNT22RegistryMatchesContract` (and the NT-07 reconciliation counts).

Exact local command:

```bash
go test -count=1 -run 'NT22|NT07' ./internal/authorization/http/
go test -count=1 -run 'OperationRegistry' ./cmd/crm-api/
```

## 10. Registering a future API operation

1. Add the operation to the contract and generated code as today (separate, reviewed work; not part of A4).
2. Add **one** entry to `internal/authorization/http/testdata/operation_scope_registry.yaml` with the real gate capability, flag, scope class, subject, handler file and NT-07 value.
3. A capability-guarded `GET` is covered by NT-07 automatically and must refuse unsupported scopes before any query. A caller-owned read needs `OWN:<basis>` with a basis from `ownBases` and the observed `unproven_user`; a new basis is a reviewed change to `ownBases` with its code reference.
4. A command needs `COMMAND:<basis>` (from `commandBases`), `row_authorization`, `separation_of_duties` and at least one existing command test reference.
5. A capability-free operation additionally needs an entry in `approvedCapabilityFree` with an exact reason — a security-review decision.
6. Update the pinned totals in `TestNT22Inventory` and NT-07's reconciliation counts, and this document.
7. Run the commands in §9.

## 11. Local validation (this PR)

Environment: Go per `go.mod`; disposable local PostgreSQL 16 cluster on `localhost:55432`, created for this run and deleted afterwards (CI uses `postgres:18`). Synthetic fixtures only; no staging or production system contacted.

| Check | Command | Result |
|---|---|---|
| NT-22 registry, inventory, commands, negative controls | `go test -count=1 -run NT22 -v ./internal/authorization/http/` | PASS (4 tests; 44 negative-control subtests) |
| NT-07 regression | `go test -count=1 -run NT07 -v ./internal/authorization/http/` | PASS (9 tests, unchanged assertions) |
| Feature-flag cross-check | `go test -count=1 -run OperationRegistry ./cmd/crm-api/` | PASS |
| NT-17 cache headers | `-run 'TestNT17\|TestSessionSensitiveResponsesAreNoStore\|TestHTTPSessionResponsesNotCacheable'` (apiserver, identity/http) | PASS |
| NT-03 submission non-enumeration | `-run 'TestPaymentSubmissionNonEnumeration\|TestOwnsTargetMatchesCommandPredicate'` (PostgreSQL) | PASS |
| All 30 tests referenced by the registry | `-run '^(…30 names…)$'` over 6 packages (PostgreSQL) | 30 PASS, 0 SKIP |
| Full suite | `CRM_TEST_REQUIRE_POSTGRES=1 go test -count=1 ./...` | PASS (exit 0; 94 packages ok) |
| Race | `go test -count=1 -race` on `internal/authorization/...`, `cmd/crm-api`, `internal/platform/apiserver`, `internal/identity/http`, `internal/payments/...` | PASS |
| Vet | `go vet ./...` | PASS |
| Architecture lint | `go test -count=1 ./internal/architecture/` (and the CI `-run` list) | PASS |
| Governance consistency | `go test -count=1 ./docs/governance/` | PASS |
| gofmt (changed files) | `gofmt -l` | clean |
| Boundary | `git diff --name-only c8a490c` | only `_test.go`, `testdata/`, `docs/` |

Pre-existing, unrelated observations (identical on the unmodified baseline `c8a490c`): without a database, `TestPgtestTemplateConcurrentClones` fails (it passes with PostgreSQL, as CI runs it); the local `gofmt -l .` lists `internal/customers/domain/domain_test.go`, which this PR does not touch.

A full `-race ./...` and the browser / Playwright suites were **not** run locally; CI runs `-race ./...`.

## 12. Residual risks

| # | Risk | Status |
|---|---|---|
| R-2 | `getFulfillmentWorkCase` runs its detail query before identity is proven (404 to an unproven user, nothing returned) | Unchanged; now pinned as `QUERY_BEFORE_IDENTITY` so a fix or regression must update the registry |
| R-A4-1 | Ownership bases, command predicates and SoD are reviewer-verified, not proven by this guard | Covered by referenced PostgreSQL integration tests; existence only checked here |
| R-A4-2 | `resolveProviderOperation` has no row predicate beyond capability + proven user (`COMMAND:no_row_predicate`) | Recorded as UNRESOLVED; no policy invented |
| R-A4-3 | Evidence relies on handlers touching a NIL dependency on their data path (same sentinel as NT-07); a handler with a non-NIL in-memory data source would not trip it | Same limit as NT-07; positive controls show the sentinel is on every guarded path today |
| R-A4-4 | Registry `governance` references are informational and not cross-checked against the decision tracker | Reviewer-verified |

## 13. Governance dependencies (unchanged, not decided here)

M2 rec. 1 / rec. 2 (Seller capability bundle and row predicates), OPD-05 (HR request authority), OPD-08 and GAP-03 (Finance review), OPD-09 and D-41 (commission), OPD-10 (payment batch ownership), D-34 (session), D-30 and GAP-04 (customer identity), GAP-01, GAP-06, GAP-07, D-23, D-24, D-44, D-45. Group B needs A4 plus the M2 decisions; it is **not** started.

**Production remains NO-GO.**
