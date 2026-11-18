# RM-E04B / A3 — Payment submission existence non-enumeration (F-09)

| | |
|---|---|
| Mode | Bounded security remediation |
| Base `main` | `895f61ad1a81dd24c35aad46c4584ef2eb4f3b03` (PR #72 merged; verified with `git rev-parse origin/main`) |
| Branch | `fix/rm-e04b-payment-submission-non-enumeration` |
| Plan row | [`../implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md) A3 |
| Finding | [`../implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md) F-09 |
| Test | [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) NT-03 (submission-context rows) |
| Affected operation | `GET /api/v1/payments/invoices/{invoice_id}/submission` (`getPaymentSubmission`) only |
| Codex contract sign-off | **PENDING** — required before merge (plan A3 "Human approval"). Not obtained; not claimed. |
| Production | **NO-GO** |

---

## 1. Root cause (F-09)

`command.Service.Context` (`internal/payments/command/read.go`) called `evaluate`, which returned `found=false` only when billing had no invoice row. For an existing invoice, the evaluator's ownership refusal was serialised into a 200 envelope (`eligible:false`, `evaluation_status: UNAUTHORIZED`, `reasons: [ROW_SCOPE_DENIED]`, `allowed_actions … reason_code: ROW_SCOPE_DENIED`, `blocked_by: [OPD-10]`). NULL seller / issuer produced 200 `BLOCKED` / `DEPENDENCY_UNKNOWN`, and an absent invoice produced 200 `BLOCKED` / `STALE_STATE` before ownership was even checked. Each of these confirms that the invoice ID exists (SR-06 violation). Amounts and attempts were already withheld.

Re-verified at HEAD before the change: the endpoint, the `payments.submission.read` capability (flag `payments.manual_write`), `ScopeOrganization` guard and the ownership predicate are exactly as documented in F-09. A negative control (§6.3) reproduces the defect on the base code.

## 2. Ownership predicate (unchanged)

Source: `internal/payments/command/command.go`, `evaluateTarget` (2.0.160 row scope):

```
owner ⇔ actor.UserID ≠ ""
       ∧ invoice.seller_user_id    IS NOT NULL ∧ invoice.seller_user_id    = actor.UserID
       ∧ invoice.issued_by_user_id IS NOT NULL ∧ invoice.issued_by_user_id = actor.UserID
```

* `actor.UserID` is the guard subject's server-resolved user id, never a client-supplied id.
* Seller AND issuer must both match (a conversion invoice cannot be told apart, so issuer ≠ seller fails closed).
* NULL / unknown on either side never owns. Hierarchy, Finance, administrators, phone, names, role labels and historical attribution never substitute (OPD-10, OPD-08).

The read path now applies this predicate through a new helper, `ownsTarget`. `evaluateTarget` and the payment command (`SubmitBatch`) are **not changed**. `TestOwnsTargetMatchesCommandPredicate` checks every actor × seller × issuer combination (including NULL and empty) and confirms that `ownsTarget` is true exactly when `evaluateTarget` does not stop at an ownership refusal. If the two ever drift, the test fails.

## 3. Change

| File | Change |
|---|---|
| `internal/payments/command/read.go` | `Context` returns `found=false` when there is no proven user id (no DB access), when the invoice is missing, **or when `ownsTarget` is false**. The ownership gate runs before `evaluateTarget`, so a non-owner's eligibility, stage, amounts, collection version, attempts and receipts are never evaluated or loaded from payments. The owner path is the same code as before. Same `READ ONLY, REPEATABLE READ` payments transaction. |
| `internal/payments/http/write.go` | Comment only; the existing `!found → authzhttp.NotFound` 404 path is reused unchanged. |
| `internal/payments/command/ownership_test.go` | New: predicate-equivalence test (unit, no DB). |
| `internal/integration/legacy/shadow/payments_submission_nonenum_e2e_test.go` | New: NT-03 T1–T14 over the real HTTP stack and disposable PostgreSQL. |
| `internal/integration/legacy/shadow/payments_http_e2e_test.go` | The old assertion that pinned the leak (non-owner 200 + `ROW_SCOPE_DENIED`) now expects 404 with no data. This is the intended contract change. |

No change to OpenAPI (404 `Problem` is already declared for `getPaymentSubmission`), generated code, schema, migrations, capabilities, grants, the authorization evaluator, feature flags, rollout defaults, payment commands, idempotency, receipt storage, other endpoints, legacy CRM or the web app.

## 4. Before / after HTTP matrix

Caller has `payments.submission.read` and the flag on, unless stated otherwise.

| Case | Before | After |
|---|---|---|
| Owner (seller = issuer = actor), eligible | 200 `SUCCESS`, full context | **unchanged** |
| Owner, business-blocked (paid invoice, unmapped / UNKNOWN stage, coverage, rollout) | 200, `evaluation_status` `BLOCKED` + reason; unknown amounts absent | **unchanged** |
| Owner, absent invoice (`presence ≠ PRESENT`) | 200 `BLOCKED` / `STALE_STATE` | **unchanged** |
| Another seller's invoice | 200, `UNAUTHORIZED` / `ROW_SCOPE_DENIED`, `OPD-10` | **404** generic problem |
| Manager / supervisor / HR above the seller | 200 `ROW_SCOPE_DENIED` | **404** |
| Seller matches, issuer differs | 200 `ROW_SCOPE_DENIED` | **404** |
| Issuer matches, seller differs | 200 `ROW_SCOPE_DENIED` | **404** |
| NULL seller and/or NULL issuer | 200 `BLOCKED` / `DEPENDENCY_UNKNOWN` | **404** |
| Non-owner, absent invoice | 200 `BLOCKED` / `STALE_STATE` | **404** |
| Capability but no resolved user id | 200 `UNAUTHORIZED` / `ROW_SCOPE_DENIED` (existing invoice) / 404 (missing) | **404** for both |
| Missing invoice (valid UUID) | 404 | 404 (unchanged) |
| Malformed id | 400 (router) | 400 (unchanged) |
| No capability | 403 `UNAUTHORIZED` (`Denied`) | **unchanged**; before any lookup |
| Flag off | 200 `BLOCKED` / `FEATURE_DISABLED`, no data | **unchanged** |

The 404 body is `authzhttp.NotFound` = `httpx.ProblemNoRecord`: `application/problem+json`, `status 404`, `title "Resource not found"`, `reason_code CONTRACT_INVALID`, `detail "No record with this id."`, `correlation_id`. It carries no invoice id, amount, stage, attempt, receipt, ownership, eligibility or decision id. The A2 headers are unchanged: `Cache-Control: no-store`, `Pragma: no-cache`, `Vary: Cookie`.

### Why 404 and not 403

SR-06 / NT-03: out of scope must look the same as missing. A 403 for a non-owner would still separate "exists, not yours" from "does not exist". 403 stays reserved for the capability decision, which does not depend on the invoice. That is why it does not leak existence, and that distinction is kept.

## 5. Tests and synthetic fixtures

Fixture: `legacytest.PaymentsScenario()`, synthetic legacy users `1`–`10` and invoices `3001`–`3008`, imported into a disposable PostgreSQL database per test. NULL-ownership variants are made inside the test's own throwaway database with `UPDATE billing.invoice` on `3007`, `3008` and `3005`. No real data.

| ID | Case | Fixture / actor | Assertion |
|---|---|---|---|
| T1 | Owner | 3001 (6/6), actor 6 | 200 `SUCCESS`; due/remaining/available 1000, used 0, version 0, `CONTROLLED_WRITEBACK`, allowed |
| T2 | Other seller / hierarchy | 3002 (7/7)↔6; 3001↔7, 2 (manager), 4 (supervisor) | 404, byte-identical to T3 |
| T3 | Random UUID | `5b0c6f0e-…` | 404 reference |
| T4 | Seller ✓ issuer ✗ | 3004 (6/4), actor 6 | 404 identical |
| T5 | Issuer ✓ seller ✗ | 3004 (6/4), actor 4 | 404 identical |
| T6 | NULL seller | 3007 (NULL/6), actor 6 | 404 identical |
| T7 | NULL issuer | 3008 (6/NULL), actor 6 | 404 identical |
| T8 | Both NULL | 3005 (NULL/NULL), actor 6 | 404 identical |
| T9 | Capability, no user id | `ghost` on 3001 and on a random UUID | 404 identical (fails closed) |
| T10 | No capability | actor 9 on 3001 and on a random UUID | 403 `UNAUTHORIZED` (unchanged) |
| T11 | Flag off | actors 6 and 7 | 200 `BLOCKED` / `FEATURE_DISABLED`, no data (unchanged) |
| T12 | Owner, ineligible stage | 3003 (paid), actor 6 | 200 `BLOCKED` / `ACTION_DENIED` (unchanged) |
| T13 | Owner, UNKNOWN stage state | 3006 (unmapped), actor 6 | 200 `BLOCKED` / `DEPENDENCY_UNKNOWN`; amounts and `stage_no` **absent** (not 0, not 404) |
| T14 | No leakage | every T2–T9 response | body contains none of: invoice id, user ids, `1000`, `ROW_SCOPE_DENIED`, `DEPENDENCY_UNKNOWN`, `eligible`, `evaluation_status`, `attempts`, `amount`, `stage`, `collection_version`, `allowed_actions`, `blocked_by`, `OPD-10`, `receipt`, `"data"` |
| T15 | Regressions | `TestPaymentsHTTP`, `TestPaymentBatchCommand` (authority matrix of the **command** still `UNAUTHORIZED` / `ROW_SCOPE_DENIED`), all payments, NT-07, NT-17 suites | pass |

T2 vs T3 comparison:

* With the same inbound `X-Correlation-ID`, status, **every header** (`Content-Type`, `Cache-Control`, `Pragma`, `Vary`, `X-Correlation-ID`) and the **exact body bytes** must be identical (`reflect.DeepEqual` on headers, `==` on bodies).
* Without an inbound id, header names and values must match except `X-Correlation-ID`, and the decoded body must match except `correlation_id`. These are the only per-request differences, and both are documented.
* The comparison cannot pass by accident on an invoice-specific body. The reference is a random UUID with no row, and T14 separately rejects any invoice-identifying token.

Read-only evidence: the test fingerprints `payments.payment_batch`, `payments.submitted_attempt`, `payments.receipt_evidence`, `payments.collection_account`, `payments.upload_object`, `payments.payment_attempt`, `billing.invoice`, `billing.payment_stage`, `billing.amount_evidence`, `finance.ledger_entry`, `finance.ledger_line`, `biawin.subscription`, `fulfillment.fulfillment_case`, `payments.command_record`, `payments.submitted_attempt_event` and `payments.receipt_state_history` before and after all probes (md5 of all rows). They are identical. The path runs only inside `READ ONLY` transactions (payments `RepeatableRead`; billing's own read-only reader).

## 6. Results (executed locally, this branch)

Environment: Go toolchain from the container, local disposable PostgreSQL 16.15 (CI uses 18), `CRM_TEST_DATABASE_URL` pointed at it.

| Check | Result |
|---|---|
| `go vet ./...` | pass |
| `gofmt -l` (changed files) | clean. One pre-existing finding in `internal/customers/domain/domain_test.go`, which is unchanged from `main` |
| `go test -count=1 ./...` | 94 packages `ok`, 0 failures |
| `TestPaymentSubmissionNonEnumeration` (T1–T14) | pass |
| `TestOwnsTargetMatchesCommandPredicate` | pass (27 combinations) |
| `TestPaymentsHTTP`, `TestPaymentBatchCommand` | pass |
| shadow package `-v` | 74 PASS, 6 SKIP (pre-existing R3 backup / R7.5 export skips, unrelated), 0 FAIL |
| A1 NT-07 (`TestNT07*`), A2 NT-17 (`TestNT17*`), `TestCachePolicy*` | 23 PASS |
| Architecture (`./internal/architecture/`) | pass |
| OpenAPI contract (`./internal/platform/httpx/oapi/`) | pass |
| `go test -race` on payments, billing, authorization, apiserver, httpx, oapi and the payments shadow tests | pass |
| Full-stack browser E2E (CI step's 8 tests, `-tags e2e`) | 8/8 pass. **Caveat:** the container's Playwright Chromium build differs from the one the web lockfile pins, so a local, uncommitted path alias to the installed headless shell was used. CI runs with its own installed browser. |

### 6.1 Owner compatibility

The owner's response for invoices 3001, 3003, 3005, 3006, 3007 and 3008 (fixed correlation id; UUIDs normalised because each run creates a new database) was captured on the base code and on this branch. The two captures are **byte-identical**, including `Content-Type` and `Cache-Control`. The capture harness was temporary and is not committed.

### 6.2 Non-owner vs missing

Asserted byte-for-byte as described in §5. Example response, the same for every T2–T9 case and for T3:

```json
{"correlation_id":"a3-nonenum-corr-0001","detail":"No record with this id.","reason_code":"CONTRACT_INVALID","status":404,"title":"Resource not found",...}
```

### 6.3 Negative control

With `read.go` reverted to the base version, `TestPaymentSubmissionNonEnumeration` fails at T2 with `200 … "reason_code":"ROW_SCOPE_DENIED"` versus the 404 reference. The test detects F-09.

## 7. Residual risks / side channels

| ID | Risk | Assessment |
|---|---|---|
| R-1 | **Timing / DB work.** For a missing invoice, billing runs one `billing.invoice` lookup with no row. For an existing non-owned invoice, billing's published `PaymentTarget` reader runs its normal 3–4 read-only queries (invoice, main-state evidence, remaining amount, stage), and the result is then discarded. No payments query runs for a non-owner. The latency difference is small but may be measurable. **Constant-time equivalence is NOT claimed**, and no timing measurement was made. | Accepted residual for A3. Mitigation option (needs a billing API decision and is out of A3 scope): an ownership-filtered billing reader that puts the predicate in the SQL `WHERE` (SR-01). Reachability today: no subject holds `payments.submission.read`. |
| R-2 | Billing facts of a non-owned invoice are loaded into server memory (never serialised) before the gate. | Same mitigation as R-1. |
| R-3 | The capability check (403) and the "no user id" case are decided before any lookup and do not depend on the invoice. They are not an existence oracle. | — |
| R-4 | Other invoice sub-resources (`/api/v1/invoices/{id}…`, `/payments`, `/receipts`) are organisation-scoped and not part of A3 (NT-03 other rows, plan B-series). | Unchanged; tracked by the plan. |
| R-5 | Web `PaymentSubmissionPanel`: a non-owner now gets the generic read-failure state (404) instead of the "not available to you" message. No web change was made in this PR. No one holds the capability today. Wording for this state is a UX follow-up. | Design follow-up, not a security issue. |
| R-6 | HR / Finance runtime stays NOT LIVE VERIFIED. Admin behaviour is not proof. | Unchanged (AGENTS.md). |

## 8. Authority and safety

No new capability, grant, role, status, workflow transition, financial rule, write path, migration or schema change. Nothing was deployed. No production or staging access. No real data. The command-side authority matrix (`SubmitBatch` → 403 `UNAUTHORIZED` / `ROW_SCOPE_DENIED` for non-owners) is unchanged: it is a write on an ID the caller already supplied, and changing it is not part of A3.

## 9. Contract review note for Codex (sign-off required before merge)

**Requested decision:** accept the observable contract change for `getPaymentSubmission`. Every caller who has the read capability but does not satisfy the payment-submission ownership predicate (§2) now gets the generic 404 `Problem` (§4) instead of a 200 `PaymentSubmissionEnvelope` with `ROW_SCOPE_DENIED` / `DEPENDENCY_UNKNOWN` / `STALE_STATE`.

* This is already allowed by the OpenAPI document: 404 `Problem` is declared, so there is no spec diff.
* Owner responses are byte-identical (§6.1). 403 and `FEATURE_DISABLED` are unchanged.
* The web client has no consumer that depended on the 200 non-owner shape for a business decision. The only test fixture using it is a mocked unit test (`payments.test.tsx`), which still passes.
* Codex: check that no downstream workflow contract relies on a non-owner receiving `ROW_SCOPE_DENIED` from this read.

Status: **PENDING.** No Codex approval has been obtained or implied. Do not merge until it is recorded on the PR.

Production: **NO-GO.**
