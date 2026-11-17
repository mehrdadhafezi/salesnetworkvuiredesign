# RM-E04B / A1 — Exhaustive Scope Assertion Hardening (F-02)

**SECURITY-SENSITIVE BACKEND HARDENING.** Production **NO-GO**. No grant, role, capability, feature flag, OpenAPI operation, schema or migration is changed. D-42, D-34, R5-SEC-01 and the unnumbered Seller decisions stay **unresolved**.

| | |
|---|---|
| Baseline | `main` @ `e0b68347963d6ca8d33163c442337b9141103551` (PR #70 merge) |
| Branch | `fix/rm-e04b-scope-assertion-hardening` |
| Parent finding | F-02 in [`../implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md); plan item A1 in [`../implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md) |
| Contract closed | SR-03 (exhaustive scope handling); NT-07 / T7 in [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) |

## 1. Root cause

`authorization.Guard.Check` returns a `Check` whose `Gate` is `GateAllowed` whenever the evaluator allows the capability. The evaluator has two scopes (`NONE`, `ORGANIZATION`, `internal/authorization/authorization.go:72-78`) and returns `ORGANIZATION` for every allowed decision. Only five packages (`commission` org reads, `reporting`, `finance`, `biawin`, `fulfillment` legacy cases) also checked `Decision.Scope == ORGANIZATION` in their own `guardOp`. Every other read checked only the gate and then ran an organisation-wide query.

Today that is harmless, because no narrower scope exists. The moment a narrower scope (e.g. a Seller own scope) is introduced in the evaluator, those handlers would treat it as "allowed" and return organisation-wide rows: a latent **fail-open**.

## 2. Revalidated inventory

`git diff d85b770..e0b6834 -- internal api cmd` is empty: PRs #69 and #70 were documentation-only, so the audit's code evidence still holds at HEAD. The count was then re-derived mechanically, not by hand:

1. `TestNT07Inventory` reads `api/openapi/openapi.bundled.yaml`: **105 operations, 92 GET**; 4 GET operations are not capability-guarded (classified below), **88** are capability-guarded.
2. The NT-07 suite of this PR was run against the **unchanged baseline handlers** (only the test seam added). It reported exactly the operations that entered the data path with an injected unsupported scope.

| Result at baseline | Count |
|---|---|
| Audit F-02 "Scope assertion: MISSING" GET handlers | **51** |
| Baseline NT-07 failures (data path entered with scope `OWN_TEST`) | **52** |
| Difference | `ListMyCommissionEarnings` — the commission `own=true` path skipped the scope assertion; the audit classified it "PRESENT (own path)" because its rows are filtered to the caller |

The 51 audit handlers and the 51 non-own baseline failures are the **same set** (diffed by name). **52 handlers are remediated**: 51 MISSING + the commission own read.

### 2.1 Original affected handlers (51) by module

| Module | # | Operations |
|---|---|---|
| identity | 2 | `ListIdentityUsers`, `GetIdentityUser` |
| organization | 9 | `ListOrganizationPersons`, `GetOrganizationPerson`, `ListOrganizationPositions`, `ListOrganizationLevels`, `ListOrganizationOrgUnits`, `ListOrganizationAssignments`, `ListOrganizationHierarchyFindings`, `ListHRRequests`, `GetHRRequest` |
| customers | 7 | `ListCustomers`, `GetCustomer`, `ListCustomerContactPoints`, `ListCustomerEvidence`, `ListMergeCases`, `GetMergeCase`, `GetCustomerResolutionReport` |
| cases | 6 | `ListCases`, `GetCase`, `ListCaseHistory`, `ListCaseAttribution`, `GetSalesCoverageReport`, `GetCaseContactOutcomes` |
| custody | 3 | `ListCustodyAssignments`, `GetCaseCustody`, `GetCustodyParityReport` |
| billing | 6 | `ListInvoices`, `GetInvoice`, `ListInvoiceItems`, `ListInvoiceStages`, `GetInvoiceTotals`, `GetBillingReconciliationReport` |
| payments | 9 | `ListInvoicePaymentAttempts`, `ListInvoiceReceipts`, `ListPaymentAttempts`, `GetPaymentAttempt`, `ListPaymentAttemptHistory`, `GetReceiptReference`, `GetPaymentSubmission`, `GetPaymentBatch`, `GetPaymentEvidence` |
| finance review | 2 | `ListFinanceReviewQueue`, `GetFinanceReviewAttempt` |
| biawin activation | 1 | `ListBiawinActivationDecisions` |
| fulfillment work | 2 | `ListFulfillmentWorkCases`, `GetFulfillmentWorkCase` |
| provider operations | 1 | `ListProviderOperations` |
| write-back diagnostics | 3 | `GetWritebackSummary`, `ListWritebackCommands`, `GetWritebackCommand` |
| **Total** | **51** | |

Already asserting at baseline (37, behaviour preserved): reporting 15, commission 7 (6 org reads + own read, which is now also asserted), finance ledger 8, biawin subscriptions/parity 4, fulfillment cases 3.

### 2.2 GET operations not capability-guarded (explicit classification)

| Operation | Classification | Why no business scope guard |
|---|---|---|
| `GetHealthLive` | public | liveness only, no data |
| `GetHealthReady` | public | readiness only, no data |
| `GetSession` | session-only | the caller's own principal and feature switches |
| `GetEffectiveAccess` | session-only | the caller's own decisions, from the same evaluator |

This list is a test constant (`notScopeGuarded` in `scope_nt07_test.go`). Adding to it is a security decision and needs review. The 13 POST commands are out of scope for NT-07 (they keep their own command-level row checks, NT-19).

## 3. Shared helper architecture

All in `internal/authorization/guard.go`; the evaluator (`Evaluate`, scope model, grant resolution) is **not** modified.

| Element | Contract |
|---|---|
| `func (g *Guard) CheckRead(ctx, capability, supported ...Scope) (Check, error)` | `Check` followed by `RequireScope`. Every data-returning read calls this and lists the scopes it explicitly implements. Today every reader passes `authorization.ScopeOrganization` only. |
| `func RequireScope(c Check, supported ...Scope) Check` | Pure. Disabled or denied checks are returned unchanged. An allowed check passes only if `Decision.Allowed` is true and `Decision.Scope` is in `supported`. `NONE` and the empty scope never pass, even if listed. Nothing is coerced (e.g. `"organization"` ≠ `ORGANIZATION`). An empty `supported` list refuses everything. |
| Refusal shape | `Gate = GateDenied`, `Decision{Allowed:false, Outcome:UNAUTHORIZED, Reason:ROW_SCOPE_DENIED, Scope:NONE}`. Because the gate becomes `GateDenied`, each handler's **existing** denied branch renders the established 403 `UNAUTHORIZED / ROW_SCOPE_DENIED` envelope via `authzhttp.Denied` — no new response shape, no new reason code. |
| `allowed_actions` hints | `Check.Decisions` is untouched (hints still come from `EvaluateAll`). |
| `Guard.Check` | Unchanged. Still used by the 13 POST commands and by `allowed_actions` hint checks of write capabilities, which carry their own row checks. |
| Test seam `NewGuardWithDecider(flag, subjects, decide)` | Lets a test return a synthetic decision (e.g. allowed with scope `OWN_TEST`) through the real guard. `NewGuard` sets `decide = Evaluate`; a nil decider is rejected. New architecture lint rule **`authz-decision-seam`** fails any call outside `_test.go` files (fixture: `testdata/violating/cmd/crm-seam`; allowed fixture: `testdata/allowed/internal/cases/http/seam_test.go`). |

Why a gate rewrite rather than a new gate value: every handler already switches on `GateDisabled` / `GateDenied` and treats anything else as allowed. A new gate value would **fall through as allowed** in every handler that does not know it. Converting to `GateDenied` makes the refusal ride the existing, already-tested denied path.

## 4. Exact code changes

| File | Change |
|---|---|
| `internal/authorization/guard.go` | `Decider`, `NewGuardWithDecider`, `ErrNoDecider`, `CheckRead`, `RequireScope`, `scopeSupported`; `Check` calls `g.decide` (= `Evaluate` in production) |
| `internal/{identity,customers,cases,custody,billing,payments,organization}/http/*.go`, `internal/integration/{providerops,writeback}/http/http.go`, `internal/finance/http/review.go`, `internal/biawin/http/activation.go`, `internal/fulfillment/http/work.go` | Read guards switched from `Check(ctx, cap)` to `CheckRead(ctx, cap, authorization.ScopeOrganization)` (35 call sites in total, including the package `guardOp` helpers) |
| `internal/{reporting,finance,biawin,fulfillment,commission}/http/http.go` | `guardOp` uses `CheckRead`; the duplicated inline `Scope != ORGANIZATION` block is removed (identical behaviour now comes from the shared helper). Specialised guards preserved: reporting's `query/spec.go` scope-parameter check, the commission own-read user-id requirement and caller filter, the fulfillment work per-service scope, HR request caller filtering, payments batch actor filter |
| `internal/organization/http/http.go` | `lookups` (positions, levels) used to render every denial as `Denied(ctx, Decision{})` = `ACTION_DENIED`. It now passes through **only** the RM-E04B refusal (`ROW_SCOPE_DENIED`); evaluator denials render exactly as before |
| `internal/architecture/lint.go`, `lint_test.go`, `testdata/…` | Rule `authz-decision-seam` + fixtures |
| `internal/authorization/scope_test.go` | Helper unit tests |
| `internal/authorization/http/scope_nt07_test.go` | NT-07 suite (inventory-driven) |

Not changed: `Evaluate`, `EvaluateAll`, `Scope` constants, `identityapi` capability registry and role bundles, grants, `DevSubjects`, feature-flag sources and defaults, `cmd/crm-api` composition, OpenAPI files and generated code, migrations, web, plugin.

## 5. Before / after behaviour

| Decision reaching a read handler | Before | After |
|---|---|---|
| Feature flag off | 200 `BLOCKED / FEATURE_DISABLED`, no rows | **unchanged** (flag checked first, even for an unsupported scope) |
| Flag source error | handler error → 500 | **unchanged** |
| Anonymous / no capability | 403 `UNAUTHORIZED / ACTION_DENIED` | **unchanged** |
| Allowed, `ORGANIZATION` (the only allowed decision the evaluator produces) | 200 with data | **unchanged** |
| Allowed, unknown scope (`OWN_TEST`, `SELLER_OWN`, `organization`) | 51 + 1 handlers: **query ran, org-wide data** | 403 `UNAUTHORIZED / ROW_SCOPE_DENIED`, no query |
| Allowed, `NONE` or empty scope | same fail-open | 403 `UNAUTHORIZED / ROW_SCOPE_DENIED`, no query |
| Commission own read, allowed `ORGANIZATION`, user id proven | caller-filtered rows | **unchanged** |
| Commission own read, user id missing | 403 `ROW_SCOPE_DENIED` | **unchanged** |

Since the evaluator can only produce the first five rows, **runtime behaviour for every real caller is unchanged**. The change only closes the path for scopes that do not exist yet.

## 6. Negative test coverage (NT-07)

`internal/authorization/http/scope_nt07_test.go` composes **every module's strict handlers** behind the **generated router** (`oapi.Handler(oapi.NewStrictHandler(...))`), and drives **every GET operation listed in `openapi.bundled.yaml`** with synthetic path / query values. A new GET operation is therefore covered automatically. It fails unless it refuses an unsupported scope before its data path, or it is explicitly classified as not capability-guarded.

| Test | Requirement | Coverage |
|---|---|---|
| `TestNT07Inventory` | reconcile 105 / 92 / 88 | contract counts and classification |
| `TestNT07PositiveControl` | (1) allowed ORGANIZATION on supported readers | 88 ops reach their data path; none refused |
| `TestNT07UnsupportedScopeRefused` | (2) unknown scope, (3) NONE, (10) no query after refusal | 88 ops × 5 scopes (`OWN_TEST`, `SELLER_OWN`, `organization`, `NONE`, `""`) × user proven / not proven = **880 cases** |
| `TestNT07DeniedStaysDenied` | (4) denied decision | 88 ops keep `ACTION_DENIED` |
| `TestNT07Anonymous` | (5) anonymous caller, real evaluator | 88 ops → `ACTION_DENIED` |
| `TestNT07FeatureDisabled` | (6) flag disabled | 88 ops → 200 `BLOCKED / FEATURE_DISABLED`, no rows, even with an unsupported scope |
| `TestNT07GuardError` | (7) evaluation error (flag-source failure, the only error path of `Guard.Check`) | 88 ops → 5xx, no query |
| `TestNT07MissingIdentity` | (8) missing user identity | `ListMyCommissionEarnings`, `ListFulfillmentWorkCases` → `ROW_SCOPE_DENIED` before query |
| `TestNT07OwnReadSemantics` | (9) existing own-scope read | real evaluator: `commission.own.read` is allowed with `ORGANIZATION`, reaches the caller-filtered query; no user → `ROW_SCOPE_DENIED`; no capability → `ACTION_DENIED` |
| `internal/authorization/scope_test.go` | helper contract | supported / unknown / case variant / NONE / empty / empty list / inconsistent gate / disabled / denied / hints / flag error / nil decider / evaluator scope model unchanged |

Per-module coverage in the unsupported-scope test (all 88 guarded GETs): identity 2, organization 9, customers 7, cases 6, custody 3, billing 6, payments 9, reporting 15, finance 10 (8 ledger + 2 review), biawin 5, fulfillment 5, commission 7, provider operations 1, write-back 3.

### 6.1 Query-not-executed evidence

Every handler in the suite is composed with **nil data dependencies**: query services, command services, the catalog reader, the payments readers, the write-back query and the reporting auditor. Any attempt to read data dereferences nil and panics, or returns an error (5xx). The suite records this as "data path entered".

- **Positive control**: with an allowed `ORGANIZATION` decision, **all 88** guarded operations trip the sentinel. This proves the sentinel sits on each operation's data path. `GetReportingMetric` uses a registered metric id (`cases.staged_total`), because an unknown id is answered 404 from the in-memory contract registry with no query.
- **Negative cases**: with any unsupported scope, **0 of 88** trip the sentinel and all answer 403 `UNAUTHORIZED / ROW_SCOPE_DENIED` with no `items`, `data`, `page`, `totals`, `summary`, `counts` or `allowed_actions` in the body.
- **Regression proof**: the same suite run against the baseline handlers trips the sentinel on **52** operations (§2). It therefore detects F-02, and this PR closes every instance.

This is a deterministic sentinel and needs no database. A wire-level recording `db.Runner` was not built, because `db.Runner` is a concrete type bound to a live pool. The positive control gives the same guarantee for the handler layer.

## 7. Remaining risk

| ID | Risk | Status |
|---|---|---|
| R-1 | The guarantee holds while every new read handler calls `CheckRead`. A handler calling plain `Check` would compile. **Mitigation:** the inventory-driven NT-07 suite fails for any new GET operation that does not refuse an unsupported scope. A future static rule could forbid `Check` in read handlers (plan A4 / NT-22). | mitigated by test |
| R-2 | `GetFulfillmentWorkCase` with an unproven user id runs the case-detail query **before** the caller's service scope resolves to empty, then answers 404. No row is returned, but the query precedes identity proof (`ListFulfillmentWorkCases` refuses first). This is existing behaviour. Changing it to 403 would change an observable response, so it is recorded here, not changed. | open, low |
| R-3 | When a narrower scope is introduced, each handler that should serve it must add it to its `CheckRead` list **and** apply the row predicate in SQL (SR-01). Listing a scope without the predicate would be a new fail-open. This is a review obligation for Group B. | by design |
| R-4 | POST commands still use `Check`; their row authority is the command-level checks (NT-19), not this helper. | unchanged |
| R-5 | F-07 (`no-store`), F-09 (submission-context existence oracle) and A4 (operation inventory guard) are separate plan items A2–A4 and are **not** started. | open |
| R-6 | HR and Finance runtime remain NOT LIVE VERIFIED; all evidence here is synthetic. | unchanged |

## 8. Compatibility findings

- API paths, HTTP methods, OpenAPI schema, generated code: **no diff** (`git diff --stat` shows no file under `api/` or `internal/platform/httpx/oapi/`).
- Capability registry, role bundles, grants, `DevSubjects` synthetic grants: **no diff**. **Zero new grants.**
- Feature flags and defaults, migrations, `cmd/`, `web/`, `plugin/`, `deploy/`: **no diff**.
- Response envelopes: the refusal reuses `authzhttp.Denied` and the existing 403 response of each operation. The one rendering change is limited to positions / levels: only for the new refusal do they now carry `ROW_SCOPE_DENIED` instead of the hard-coded `ACTION_DENIED`. Evaluator denials there are byte-for-byte as before.
- Existing PostgreSQL-backed handler and full-stack HTTP tests (`internal/integration/legacy/shadow`, module tests) pass unchanged, so existing successful ORGANIZATION reads and the BLOCKED / UNAUTHORIZED responses behave as before.

## 9. Test results (local, this branch)

Environment: Go 1.27.0; disposable PostgreSQL **16.15** cluster created for this run (CI uses 18), `CRM_TEST_REQUIRE_POSTGRES=1`, so PostgreSQL tests could not skip. Synthetic data only.

| Check | Result |
|---|---|
| `go build ./...`, `go vet ./...` | PASS |
| `go test -count=1 ./...` (unit + PostgreSQL integration, incl. `docs/governance`, `internal/architecture`) | PASS — 93 packages ok, 0 failures |
| `go test -race` on every touched package + `internal/integration/legacy/shadow` + `internal/architecture` | PASS — 44 packages ok |
| NT-07 suite + helper unit tests | PASS |
| NT-07 against baseline handlers (seam only) | FAILS on 52 operations, as expected (F-02 detected) |
| Architecture lint incl. new `authz-decision-seam` rule and fixtures | PASS |
| All `go-ci.yml` boundary scripts (Kernel stdlib, Slice 8/9/10/3-11/12/13/14, R1–R4) executed locally | PASS (12/12) |
| `gofmt -l` on changed files | clean (`internal/customers/domain/domain_test.go` was already unformatted on `main`; untouched) |
| Not executed locally | `web-ci`, `e2e-ci` (Playwright full-stack browser tests), release / staging rehearsal and "generated outputs are current" (no OpenAPI or web change). These are left to CI and reported in the PR |

## 10. Status

- Zero new grants, roles, capabilities or flags. No database, staging or production action.
- Production: **NO-GO**.
- A2, A3, A4 not started. D-42, D-34, R5-SEC-01 and the unnumbered Seller decisions unresolved.
