# RM-ACCESS-01: Authorization Evaluation and Capability Contract Remediation

| | |
|---|---|
| Branch | `fix/rm-access-01-authorization-contract` |
| Baseline | `main` @ `25e294e618999f09adf063f6423d9e224c932f6d` (PR #65, RM-E02, merged) |
| Mode | Security-sensitive controlled remediation |
| Production | **NO-GO** (unchanged) |
| Defects | DEF-ACCESS-01 (HR flag applied to every capability), DEF-ACCESS-02 (OpenAPI `Capability` enum incomplete) |

No grant, role bundle, flag default, identity or session policy, migration or data was changed. No production surface was touched.

## 1. Root cause

### DEF-ACCESS-01: one HR switch decided every module

`internal/authorization/http/http.go` `GetEffectiveAccess` (baseline) did this:

```go
enabled, authenticated, ds, err := h.guard.Effective(ctx)   // enabled = organization.hr_read_shadow
...
ad := oapi.AccessDecision{..., Allowed: d.Allowed && enabled, ...}
if !enabled { ad.Outcome, ad.ReasonCode, ad.Scope = BLOCKED, FEATURE_DISABLED, NONE }
```

The handler is built in `cmd/crm-api/read.go` `NewReadAPI` from the Slice 4 HR read guard, so `enabled` is the HR read flag `organization.hr_read_shadow`.

**History (audit question 7).** The gate came with Slice 4 (`833660a`, 2026-10-31). At that point the vocabulary held only the six HR read capabilities, so gating every capability by the HR read flag was correct. From Slice 5 on (`2604146`, customers), each slice added its capabilities to the same registry and the same `identityapi.Capabilities()` loop, and each built its own guard with its own flag (`cmd/crm-api/*.go`). The effective-access handler was never updated. **The gate was not a deliberate cross-module rule; it became an accidental global dependency.** No spec or ADR made non-HR access depend on HR availability. The OpenAPI description only guaranteed "same evaluator as allowed_actions" and "anonymous callers receive an all-denied evaluation" (audit question 6).

Consequences:
- HR flag off: every one of the 43 decisions was `BLOCKED / FEATURE_DISABLED`, even when a module's own flag was on and its endpoint answered `SUCCESS`. RM-E02 navigation then showed BLOCKED for every module.
- HR flag on: every held capability was reported `allowed`, even when the capability's own flag was off and its endpoint answered `BLOCKED`. This is the more serious half: the view over-reported availability. The endpoints themselves stayed safe (each guard checks its own flag), and the SPA also ANDs the session feature switch, so no link became executable. But the authoritative evaluation was wrong in both directions.

### DEF-ACCESS-02: contract drift

`api/openapi/components/hr.yaml#/Capability` listed 29 values. The backend registry `internal/identity/api` (`vocabulary`) holds **43**. The database catalog `identity.capability` (migrations 00001–00008) also holds 43. The server already evaluated and returned all 43 at runtime through an unchecked `oapi.Capability(c.Key)` cast. 14 values the server returned were therefore outside its own published contract, and the SPA had to treat 9 destinations as UNKNOWN. Nothing checked contract against registry.

## 2. Before / after behaviour

| Situation | Before | After |
|---|---|---|
| HR flag OFF; customers flag ON; caller holds `customers.customer.read` | `BLOCKED / FEATURE_DISABLED` (endpoint answered SUCCESS) | `allowed, SUCCESS, ORGANIZATION` (same as the endpoint) |
| HR flag OFF; caller holds HR reads | `BLOCKED / FEATURE_DISABLED` | unchanged: `BLOCKED / FEATURE_DISABLED`; endpoint BLOCKED |
| HR flag ON; customers flag OFF; caller holds `customers.customer.read` | **`allowed`** (endpoint answered BLOCKED) | `BLOCKED / FEATURE_DISABLED` (same as the endpoint) |
| HR flag ON; caller holds HR reads | allowed | unchanged: allowed |
| No ACTIVE grant, module flag ON | HR off: BLOCKED; HR on: UNAUTHORIZED | `UNAUTHORIZED / ACTION_DENIED` |
| Anonymous | all denied | unchanged: all denied, `authenticated:false` |
| `hr_read_enabled` | HR read flag | unchanged: HR read flag |
| Decisions returned | 43 (14 outside the contract) | 43 (all in the contract) |
| Unconfigured read surface (`apiserver.unconfigured`) | 29 decisions, all BLOCKED | 43 decisions, all BLOCKED |

The old behaviour was reproduced on this branch: with only `internal/authorization/http/http.go` and `cmd/crm-api/read.go` reverted to baseline, the new PostgreSQL regression fails:

```
--- FAIL: TestEffectiveAccessPerCapabilityAvailabilityPostgres
    access_pg_test.go:195: B customers.customer.read: allowed=false outcome=BLOCKED reason=FEATURE_DISABLED; want true SUCCESS
```

## 3. Backend capability registry vs OpenAPI (exact difference)

Registry (`internal/identity/api.vocabulary`): **43**. OpenAPI enum at baseline: **29**. Missing from the contract: **14**. Present in the contract but not in the registry: **0**. Renamed: **0**.

| # | Capability (canonical, unchanged) | Enforced by | Used at |
|---|---|---|---|
| 1 | `payments.upload.stage` | payments write guard | `internal/payments/http/write.go` |
| 2 | `payments.batch.submit` | payments write guard | `internal/payments/http/write.go` |
| 3 | `payments.submission.read` | payments write guard | `internal/payments/http/write.go` |
| 4 | `payments.evidence.read` | payments write guard | `internal/payments/http/write.go` |
| 5 | `finance.review.read` | finance review guard | `internal/finance/http/review.go` |
| 6 | `commission.shadow.read` | commission guard | `internal/commission/http/http.go` |
| 7 | `commission.policy.read` | commission guard | `internal/commission/http/http.go` |
| 8 | `commission.own.read` | commission guard | `internal/commission/http/http.go` |
| 9 | `fulfillment.work.read` | operations read guard | `internal/fulfillment/http/work.go` |
| 10 | `fulfillment.work.assign` | operations write guard | `internal/fulfillment/http/work.go` |
| 11 | `fulfillment.work.execute` | operations write guard | `internal/fulfillment/http/work.go` |
| 12 | `biawin.activation.read` | operations read guard | `internal/biawin/http/activation.go` |
| 13 | `integration.provider_operation.read` | operations read guard | `internal/integration/providerops/http/http.go` |
| 14 | `integration.provider_operation.reconcile` | provider reconcile guard | `internal/integration/providerops/http/http.go` |

All 14 are existing vocabulary entries with descriptions, already present in the `identity.capability` database catalog and already enforced by the handlers above. **No exclusion was needed.** They were added to the enum in registry order. No capability was invented, renamed, bundled or granted.

## 4. Feature → capability mapping (availability source)

Each capability is gated by the flag of the guard that calls `Check(ctx, capability)` for it. The table is `cmd/crm-api/access.go` `capabilityFlags`. The session feature (`GET /api/v1/session` → `features`) used by navigation reports that same guard's flag.

| Flag (default OFF; unchanged) | Session feature | Capabilities | Kind |
|---|---|---|---|
| `organization.hr_read_shadow` | `hr_read` | `identity.user.read`, `org.person.read`, `org.employment.read`, `org.position.read`, `org.unit.read`, `org.assignment.read` | availability (read) |
| `customers.identity_shadow_read` | `customers_identity` | `customers.customer.read`, `customers.merge_case.read`, `customers.identity_report.read` | availability |
| `sales.read_shadow` | `sales_read` | `cases.case.read`, `custody.custody.read`, `sales.report.read` | availability |
| `billing_payments.read_shadow` | `billing_read` | `billing.invoice.read`, `payments.attempt.read`, `billing.reconciliation.read` | availability |
| `reporting.read_shadow` | `reporting_read` | `reporting.metrics.read`, `reporting.mis.read`, `reporting.invoice_review.read` | availability |
| `finance.shadow_ledger_read` | `finance_shadow_read` | `finance.shadow.read`, `finance.ledger.read`, `finance.reconciliation.read` | availability |
| `biawin_fulfillment.read_shadow` | `biawin_fulfillment_read` | `biawin.subscription.read`, `fulfillment.case.read`, `biawin.parity.read` | availability |
| `cases.nonfinancial_write` | `contact_outcome_write` | `cases.contact_outcome.record` | availability (command; the rollout mode is a separate gate) |
| `organization.hr_request_write` | `hr_request_write` | `org.hr_request.create`, `org.hr_request.review`, `org.hr_request.read` | availability (no final apply exists) |
| `integration.writeback_read` | `writeback_read` | `integration.writeback.read` | availability (diagnostics) |
| `payments.manual_write` | `payment_submission` | `payments.upload.stage`, `payments.batch.submit`, `payments.submission.read`, `payments.evidence.read` | availability (no review / posting) |
| `finance.review_read` | `finance_review_read` | `finance.review.read` | availability (no approve / reject) |
| `commission.read_shadow` | `commission_read` | `commission.shadow.read`, `commission.policy.read`, `commission.own.read` | availability |
| `fulfillment.work_read` | `fulfillment_work` | `fulfillment.work.read`, `biawin.activation.read`, `integration.provider_operation.read` | availability |
| `fulfillment.work_write` | — | `fulfillment.work.assign`, `fulfillment.work.execute` | availability (command) |
| `integration.provider_reconcile` | — | `integration.provider_operation.reconcile` | availability (evidence only; never dispatches) |

**Availability vs authority (audit question 5).** Every flag above controls availability only. Authority comes only from a counting ACTIVE grant whose role bundle contains the capability (or, in dev/test only, the synthetic stub grant). No flag grants anything, and the table never feeds `Evaluate`. The `_writeback` flags (real legacy write-back) and any posting, approval, refund, payout or activation switch are deliberately absent: they are not availability inputs, and the test `TestAvailabilityNeverFromWritebackOrApply` enforces this.

## 5. Effective-access semantics

Field meanings are **unchanged**. RM-ACCESS-01 corrects only the availability input:

- `allowed`: the caller holds the capability through a counting ACTIVE grant **and** the feature that guards that capability is on. This is the meaning `allowed` already had for the HR capabilities, now applied to every capability with its own flag.
- `outcome` / `reason_code`: the evaluator's result (`SUCCESS`; `UNAUTHORIZED / ACTION_DENIED`; `BLOCKED / LINKAGE_CONFLICT`; `BLOCKED / POLICY_UNRESOLVED`), replaced by `BLOCKED / FEATURE_DISABLED` when that capability's own feature is off.
- `scope`: `ORGANIZATION` when allowed, `NONE` otherwise (forced `NONE` when unavailable).
- `matched_roles`: the grants that hold the capability, sorted, with no selection. As before, they are reported even when the capability is unavailable, and only allowed decisions feed the SPA's role context.
- `hr_read_enabled`: still the HR read flag only.

Fail-closed rules (all tested):
- A capability with no availability entry is unavailable.
- A nil availability source makes every capability unavailable.
- An unreadable flag store fails the whole request (500), never a partial view.
- An availability entry for a non-registry capability is ignored.
- Unknown capabilities are still `BLOCKED / POLICY_UNRESOLVED` in `Evaluate`.

No new endpoint. No schema shape change: the only OpenAPI changes are enum values and descriptions.

## 6. Code changes

| File | Change |
|---|---|
| `internal/authorization/guard.go` | New `Availability` type (per-request capability → enabled map; availability only, never authority) |
| `internal/authorization/http/http.go` | `New(guard, available)`; each decision uses its own capability's availability instead of the HR flag; `hr_read_enabled` unchanged |
| `cmd/crm-api/access.go` (new) | `capabilityFlags` (43 entries = the guard flags above) and `capabilityAvailability` (one read-only transaction over the 16 distinct flags) |
| `cmd/crm-api/read.go` | passes `capabilityAvailability(flags)` to the effective-access handler |
| `internal/platform/apiserver/unconfigured.go` | unconfigured fallback lists all 43 contract capabilities (all BLOCKED) |
| `api/openapi/components/hr.yaml`, `api/openapi/openapi.yaml` | +14 enum values (registry order); descriptions state the per-capability semantics |
| `api/openapi/openapi.bundled.yaml`, `internal/platform/httpx/oapi/oapi.gen.go`, `web/src/generated/api/schema.ts` | regenerated (`npm --prefix web run openapi:gen`) |
| `web/src/app/navigation/destinations.ts` | the 9 UNKNOWN destinations are CONTRACT; the UNKNOWN variant is kept as the fail-closed rule for any future drift |

Tests: `internal/authorization/http/http_test.go` (new), `cmd/crm-api/access_test.go` (new), `cmd/crm-api/access_pg_test.go` (new), `internal/integration/legacy/shadow/api_e2e_test.go`, `internal/platform/apiserver/slice4_test.go`, `web/src/app/navigation/navigation.test.tsx`, and R7 specs `web/e2e/{fulfillment,reporting-biawin,reporting-biawin-overlay}.spec.ts` and `web/e2e/fixtures.ts`.

### Changed expectations (old behaviour, why it was wrong, corrected contract)

| Test | Old expectation | Why it was wrong | New expectation (safety assertions kept) |
|---|---|---|---|
| `shadow/api_e2e_test.go` `TestReadAPIEndToEnd` | dev-user-001 with only the HR flag on: **43** allowed | 37 of those capabilities belonged to surfaces this server does not even compose, and their flags were off | exactly the 6 HR reads allowed; `customers.customer.read` matched but `BLOCKED / FEATURE_DISABLED`; viewer still has no `allowed:true`; no-write fingerprint unchanged |
| `apiserver/slice4_test.go` | unconfigured view has 29 decisions | the contract omitted 14 registry capabilities | 43 decisions, none allowed |
| `navigation.test.tsx` "outside the OpenAPI enum is UNKNOWN" | 9 destinations hidden even when allowed | the capability was missing from the contract, not unauthorised | the 9 are CONTRACT; new test **M**: hidden with the feature alone, with the capability alone, or when the server says BLOCKED; shown only when allowed AND feature on. A capability string outside the contract still shows nothing |
| `fulfillment.spec.ts` (×2), `reporting-biawin.spec.ts` | home page BLOCKED | BLOCKED came from the HR flag, not from these modules | home page NO_GRANT (modules on, user has no grant). Every link-absent, refusal, no-data and no-write assertion kept |
| `reporting-biawin-overlay.spec.ts` | links withheld (BLOCKED) | same HR-flag artefact; the synthetic reads were served by every deep link | the effective-access answer is asserted (reporting and Biawin allowed; HR, customers, finance, commission and operations `BLOCKED / FEATURE_DISABLED`); the reporting and Biawin links are listed; HR, customers, finance and operations links stay absent. All render = API = DB checks still run |

No test was removed.

## 7. Test evidence (local, Windows host, disposable PostgreSQL 18)

| Suite | Result |
|---|---|
| `go build ./...`, `go vet ./...`, `gofmt -l .` | clean |
| `go test ./internal/authorization/...` (incl. new matrix A–J, L) | pass |
| `go test ./cmd/crm-api/` with PostgreSQL (new real-API matrix A–F, H, K, N, O) | pass (the test ran; it was not skipped) |
| `go test ./internal/integration/legacy/shadow/ -run TestReadAPIEndToEnd` with PostgreSQL | pass |
| `go test ./...` with PostgreSQL (Linux container, LF export of this branch) | **all packages pass** |
| `go test -race` (authorization, authorization/http, cmd/crm-api, apiserver, httpx/...) with PostgreSQL, Linux container | **pass** |
| OpenAPI lint + bundle + Go/TS generation; stale check | regenerated; no diff after a second run |
| `npm run typecheck`, `typecheck:e2e`, `lint`, `build` | clean |
| `vitest` | **179/179** (was 176; +3 net: M, contract-coverage, outside-contract, B-navigation; the old UNKNOWN test replaced) |
| `playwright.nav.config.ts` (RM-E02 browser navigation) | **9/9** |
| R7 full-stack E2E (`e2e-ci.yml`) | **CI only**: the orchestrator needs Linux process groups |
| Governance (`docs/governance`), LF export, Linux container | **pass** |

### 7a. Host notes

- On the Windows host, a CRLF checkout makes byte-exact suites fail with or without this change: gofmt, rendered migrations, governance, staging (Linux-only) and payments object-store paths. Every Go result above therefore comes from an LF export of this branch run in the official `golang:1.27.2` Linux image against the same disposable PostgreSQL 18, matching CI.
- `gofmt -l .` is empty and `go vet ./...` is clean there.
- See the PR for the GitHub CI status.

## 8. Security regression matrix

| ID | Scenario | Expected | Evidence |
|---|---|---|---|
| A | HR OFF, non-HR not granted | DENIED, never allowed | `TestA_HROffNonHRNotGranted`; PG: `customers.customer.read` UNAUTHORIZED, `/api/v1/customers` 403 |
| B | HR OFF, non-HR granted (synthetic ACTIVE, test only) | not blocked by HR; endpoint governed by its own flag | `TestB_…`; PG: allowed + `/api/v1/customers` SUCCESS; baseline reproduction fails (§2) |
| C | HR OFF, HR granted | HR unavailable | `TestC_…`; PG: 6 HR decisions BLOCKED; `/api/v1/organization/persons` BLOCKED, no items |
| D | HR ON, HR not granted | UNAUTHORIZED | `TestD_…`; PG: dev-user-002 UNAUTHORIZED, persons endpoint 403 |
| E | HR ON, HR granted | HR reads only | `TestE_…`; PG: 6 HR allowed, persons SUCCESS; HR ON never makes a non-HR capability available |
| F | non-HR feature OFF, granted | endpoint disabled; no executable link | `TestF_…`; PG: customers BLOCKED + endpoint BLOCKED; navigation tests 3 and M |
| G | no ACTIVE grant (none, SHADOW, PENDING, REVOKED, unmapped/unknown role, empty bundles, non-synthetic direct capabilities) | nothing allowed | `TestG_NoActiveGrant` |
| H | anonymous / expired / revoked session (resolves anonymous) | no permission | `TestH_…`; PG: anonymous `authenticated:false`, customers 403; RM-E02 session-expiry tests unchanged |
| I | conflicting identity; invalid availability evidence | fail closed | `TestI_…`: LINKAGE_CONFLICT for all; flag-store error → 500; nil / partial / foreign availability → unavailable |
| J | multiple role grants | all matched roles, sorted; no selection, no escalation | `TestJ_…`; navigation MULTIPLE (RM-E02 test 5) |
| K | dev synthetic authorization in production | rejected | `TestDevAuthRefusedInProductionForAccess`; existing `TestRunRefusesDevAuthInProduction` |
| L | 43 registry capabilities = contract | fails on drift | `TestL_ContractEnumEqualsRegistry` (both directions, parsed from the generated contract); `TestCapabilityFlagsCoverRegistry` |
| M | 9 formerly hidden destinations | no visibility from contract membership or flags alone | `navigation.test.tsx` M |
| N | existing role and grant data | no mutation | PG: `identity.role_grant` / `role_capability` / `capability` fingerprint unchanged; `TestVocabularyShape` (only HR has a bundle) unchanged and passing |
| O | financial, HR apply, write-back, Biawin activation restrictions | still enforced | PG: payment batch, work assign/execute, provider reconcile, HR review, contact outcome and finance review stay BLOCKED with their own flags off while other modules are on; `TestAvailabilityNeverFromWritebackOrApply`; `TestNoFinancialExecutionCapability` (no approve / post / refund / activate capability exists) unchanged; per-surface command tests unchanged |

Synthetic grants exist only in test code and in `authorization.DevSubjects` (DEV/TEST stub, refused in production). Nothing in this change persists a grant.

## 9. Governance boundaries (unchanged, unresolved)

- **D-42** role/capability authorization: open. Every target role except HR still has an empty bundle.
- **D-34** account provisioning: open.
- **R5-SEC-01** person-level separation of duties: open.
- **ADR #18** real-data authority: open.
- Other business approvals (D-25/26/28, D-30, D-37/38, OPD-01/04/05/10, SD-04): open.

This change does not approve, bypass or modify any of them. It does not add production grants, populate bundles, change flag defaults, add test bypasses, change identity or session policy, or enable HR final apply, financial approval/posting/refund, commission payout, Biawin production activation, real legacy write-back, or real-data import.

## 10. Remaining runtime limitations and residual risks

1. **Availability is flag-based, not configuration-based.** If a flag is on but its surface's database is not configured, effective-access reports the capability as available while the endpoint answers `BLOCKED / FEATURE_DISABLED` from the `apiserver` fallback. The session feature for that surface is false, so navigation still hides it. This only affects misconfigured deployments and fails closed at the endpoint.
2. **Table/guard coupling.** `capabilityFlags` restates the flags passed to each `NewGuard`. Tests fail if a capability is unmapped or mapped to a flag no guard uses. They cannot detect a guard silently switched to a *different* existing flag. A future refactor could derive both from one declaration.
3. **Command rollout mode** (`rollout.FromFlags`) is a second gate for commands and is not reflected in effective-access. An allowed command capability can still answer `TARGET_WRITE_DISABLED` per its mode. This is unchanged and fails closed.
4. **HR and Finance runtime remain NOT LIVE VERIFIED** (AGENTS.md). Every allowed decision in the tests comes from synthetic dev/test grants; no real role account exists.
5. **R7 full-stack browser specs** run only in CI (Linux). Their updated expectations are verified there, not locally.

## 11. Follow-up requirements

### RM-B02 (role/capability authorization, D-42)
- RM-E02 §10 items 1 and 2 are **resolved** here. Items 3 (server-decided role context and an approved multi-role selection policy) and 4 (role bundles and ACTIVE grants: D-42, D-34, R5-SEC-01) remain.
- Any bundle change must keep `TestVocabularyShape`, `TestCapabilityFlagsCoverRegistry` and `TestL_ContractEnumEqualsRegistry` green. A new capability needs, in the same change: a registry entry, a DB catalog migration, an OpenAPI enum value, a `capabilityFlags` entry for its guard flag, and (if navigable) a CONTRACT destination.

### RM-E03 (seller read screens)
- No longer blocked by the HR flag. A seller destination appears only when the seller holds the capability through an ACTIVE grant (none exists until D-42) **and** the destination's own module flag is on.
- New destinations must use a CONTRACT capability (all 43 are available) and map in `roles.ts`. The resolver and shell need no change.
- Until D-42, RM-E03 can be exercised only with synthetic fixtures or the dev stub.

**Can RM-B02 and RM-E03 proceed?** Yes, safely, from the authorization-contract side. Both remain bound by the open governance decisions above. Production remains **NO-GO**.
