# RM-E04B / A2 — HTTP Sensitive Response Cache Hardening (F-07)

**BOUNDED BACKEND SECURITY HARDENING.** Production **NO-GO**. Only HTTP response headers change. Response bodies, HTTP statuses, API paths and methods, OpenAPI DTOs, authentication, cookies, CSRF, the authorization evaluator, role grants, feature flags, row scopes, the database schema, the legacy CRM, the Seller React module and financial behaviour are unchanged. No migration, no deployment.

| | |
|---|---|
| Baseline | `main` @ `3906831fd8df6853530334fde51e249f324654a1` (PR #71 merge, RM-E04B / A1) |
| Branch | `fix/rm-e04b-cache-control-no-store` |
| Parent finding | F-07 in [`../implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md); plan item A2 in [`../implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md) |
| Contract closed | SR-12 (no caching); NT-17 / T12 in [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) |

## 1. Root cause

At the baseline, `Cache-Control: no-store` was set in three places only:

1. `httpx.WriteProblem` (`internal/platform/httpx/problem.go:106`) — every RFC 9457 problem response;
2. the `noStore` middleware in `internal/platform/apiserver/auth.go` — only for the paths `sessionSensitive` listed (`/api/v1/session`, `/api/v1/authorization/effective-access`, `/api/v1/auth/*`, `/api/v1/identity*`), plus `sessionAuth` when it clears an invalid cookie;
3. the metrics endpoints (separate listeners).

Every other successful `/api/*` JSON response (customers, cases, invoices, payments, commission, reporting, …) carried **no** `Cache-Control`, and **no** response carried `Vary: Cookie`. The policy was a path allow-list, so any new data endpoint was cacheable by default — the gap the Seller own-scope reads would have made material.

## 2. Change

### 2.1 Middleware location

One reusable middleware, `httpx.CachePolicy`, in `internal/platform/httpx/cache.go` (the HTTP foundation package that already holds correlation, problem writing and panic recovery). It is installed once, in `apiserver.NewHandlerWithSurfaces` (`internal/platform/apiserver/server.go`), **outside** every middleware that can answer early:

```
Metrics → CachePolicy → Correlation → Recover → LimitBody → sessionAuth → crossSiteGuard
        → (dev auth stub) → bodyAllowlist → queryAllowlist → OpenAPI router / strict handler → fallback 404/405
```

It therefore sees every response the API produces: handler responses, strict-handler request and response errors, central 404 / 405, body-limit 413, body / query allowlist 400, CSRF / Origin / `Sec-Fetch-Site` 403, dev-stub 400, recovered panics (500) and the correlation middleware's own failure path.

The path-specific `noStore` / `sessionSensitive` / `setNoStore` in `apiserver/auth.go` were removed (their paths are a strict subset of `/api/*`). `sessionAuth` now calls `httpx.MarkPrivate` when it clears a cookie (same headers as before, plus `Vary: Cookie`). No per-handler logic was added.

### 2.2 Classification

| Response | Policy | Reason |
|---|---|---|
| Any request path `/api` or `/api/…` (`httpx.APIPath`; `/apix` is not) | `Cache-Control: no-store`, `Pragma: no-cache`, `Vary` ⊇ `Cookie` | Every `/api` operation is evaluated against the caller's session cookie (`sessionAuth` runs on every request): the same URL answers data, `401` or `403` depending on the caller. This includes `GET /api/v1/session` when anonymous. |
| Any response carrying `Set-Cookie`, on any path | same | A response that sets or clears a session cookie is private (e.g. `/health/live` clearing a revoked cookie). |
| `/health/live`, `/health/ready` (contract operations outside `/api`) | unchanged (no cache header) | Liveness / readiness documents carry no caller, session or business data and do not depend on the cookie. Classified by actual route and content, not by being "API-adjacent". |
| `/metrics` (separate listener, not this handler) | unchanged (`no-store` already, `apiserver/metrics.go`) | Not reachable through the public handler. |
| Static SPA assets (`/`, `/assets/*`) | unchanged | Served by nginx (`location /`), never reach the Go handler; verified in §6. |

### 2.3 Header precedence and deduplication rules

1. The policy is applied **before** the inner handler runs and **again when the final status is written** (`WriteHeader` ≥ 200, first `Write`, or `Flush`). A handler, an inner middleware or a panic recovery that sets a conflicting value (e.g. `public, max-age=600`) or a duplicate line cannot win.
2. `Cache-Control` and `Pragma` are **replaced** (`Header.Set`): exactly one line each, value `no-store` / `no-cache`. `no-store` alone is sufficient (RFC 9111 §5.2.2.5); no `private`, `max-age` or `no-cache` is added, so there is nothing to conflict.
3. `Vary` is **merged** into one line: existing tokens are kept in their order, empty tokens and case-insensitive duplicates are dropped, `Cookie` appears once, last (an existing spelling of it is kept). `Vary: *` is kept as the only value (it already forbids reuse). Unrelated tokens (`Origin`, `Accept-Encoding`, …) are preserved.
4. Informational `1xx` responses do not finalise the headers.
5. Applying the middleware more than once yields identical headers (idempotent; tested three-deep).
6. Correlation (`X-Correlation-ID`), `Content-Type`, `Location`, `Allow`, `Set-Cookie`, security headers and the response body / status are never touched (tested).
7. The writer wrapper forwards `Flush` and implements `Unwrap` for `http.ResponseController`.

## 3. Operation coverage

`api/openapi/openapi.bundled.yaml` was counted at the baseline and in the test: **105 operations** (103 under `/api`, 2 health). `TestNT17EveryOperationIsNoStore` loads the bundled contract (it fails if the count changes, forcing re-classification of new operations) and sends, for **every** operation, through the **full handler chain** in three configurations (production anonymous; production with a synthetic session surface, metrics and a failing read surface; development with the dev identity stub):

| Variant | Purpose |
|---|---|
| anonymous | unauthenticated / BLOCKED / 401 / 403 paths |
| authenticated (session cookie + CSRF, or dev identity) | authenticated read / command responses |
| revoked cookie | `sessionAuth` clears the cookie (`Set-Cookie` + no-store) |
| unknown query parameter | `queryAllowlist` middleware 400 |
| wrong method | central 405 |
| CSRF mismatch | `crossSiteGuard` middleware 403 (unsafe methods) |
| unknown child path | central 404 |

103 `/api` operations × 7 variants × 3 configurations = **2 163** responses, every one asserted: exactly one `Cache-Control: no-store`, one `Pragma: no-cache`, one `Vary` line containing `Cookie` exactly once, a valid correlation id. Observed status distribution (local run): `200:1118 400:468 401:11 403:12 404:240 405:306 500:8`.

The contract defines no `201`, `204` or redirect responses (its success responses are all `200`; error statuses `400 401 403 404 409 413 429 503`). `201`, `202`, `204`, `301`, `302`, `304`, `409`, `429` and `503` are covered at the middleware level (`TestCachePolicyAllStatusesAndBodiesUnchanged`).

## 4. NT-17 test results

All tests are new unless marked. Local run, `go test -count=1 -v`:

| NT-17 requirement | Test | Result |
|---|---|---|
| 200 with no-store | `TestNT17EveryOperationIsNoStore`, `TestNT17SpecificPaths` (session, dev authenticated read) | PASS |
| 201 / other success | `TestCachePolicyAllStatusesAndBodiesUnchanged` (201, 202, 204; contract has no 201) | PASS |
| 400 validation | invalid JSON, unknown body field (middleware), invalid path UUID, unknown query (middleware), unknown dev identity (middleware) | PASS |
| 401 unauthenticated | `TestNT17EveryOperationIsNoStore` (11 responses) | PASS |
| 403 unauthorized | CSRF mismatch, cross-site, foreign Origin (all middleware) + sweep | PASS |
| 404 not found | unknown route, `/api` root, unknown child path (sweep: 240) | PASS |
| 405 method not allowed | sweep (306) + `PUT /api/v1/session` | PASS |
| 413 body too large (middleware) | `TestNT17SpecificPaths` | PASS |
| 500 internal error | handler error (`ResponseErrorHandlerFunc`) and recovered panic | PASS |
| Session response | `GET /api/v1/session` anonymous, authenticated, revoked cookie | PASS |
| Authenticated read | sweep (session cookie) + dev identity read; NT-17 named reads `GET /api/v1/commission/me/earnings`, `GET /api/v1/payments/batches/{batch_id}` | PASS |
| Set-Cookie response | login sets the session cookie | PASS |
| Logout / clear cookie | logout clears the cookie; revoked cookie cleared on `/api` and on `/health/*` | PASS |
| Pre-existing Cache-Control | `TestCachePolicyOverridesConflictingCacheControl` (public max-age, duplicate lines, private, no-store twice) | PASS |
| Pre-existing Vary | `TestCachePolicyMergesVary` (none, Accept-Encoding, cookie, multi-line + duplicates, empty tokens, `*`) | PASS |
| Repeated middleware application | `TestCachePolicyIdempotent` | PASS |
| No duplicate / conflicting directives | asserted on every response (exactly one line each) | PASS |
| Static assets unaffected | `TestCachePolicyLeavesNonAPIAlone`, `TestAPIPath`; nginx check §6 | PASS |
| Correlation and CSRF unchanged | correlation asserted on every response; inbound id preserved through recovery; CSRF refusals still 403; existing `TestHTTPCSRF`, `TestHTTPSessionResponsesNotCacheable` (real sessions, PostgreSQL) unchanged and passing | PASS |
| Health classification | `TestNT17HealthClassification` | PASS |
| Existing session no-store test | `TestSessionSensitiveResponsesAreNoStore` (existing, unchanged assertions) | PASS |

Mutation check: with the `CachePolicy` line removed from the chain, `TestNT17EveryOperationIsNoStore`, `TestNT17SpecificPaths` and `TestSessionSensitiveResponsesAreNoStore` fail.

## 5. Validation

| Check | Where | Result |
|---|---|---|
| `gofmt -l` on changed files, `go vet ./...` | local | clean |
| `go test -count=1 ./...` with `CRM_TEST_REQUIRE_POSTGRES=1` (local PostgreSQL 16; CI uses 18) | local | PASS (93 packages) |
| `go test -count=1 -race ./...` with PostgreSQL | local | PASS (93 packages) |
| Architecture lint (`./internal/architecture/`) and governance (`./docs/governance/`) | local (part of `./...`) | PASS |
| nginx proxy harness with the staging `nginx.conf` (§6) | local | PASS |
| Full-stack browser E2E (`-tags e2e`, the 8 tests `e2e-ci.yml` runs: Browser → SPA → crm-api → PostgreSQL) | local | PASS (8/8 Go tests, 80 Playwright specs). Local deviation: the pinned Playwright 1.56.1 was pointed at the container's preinstalled Chromium build via `PLAYWRIGHT_BROWSERS_PATH`; CI uses its own pinned browser |
| `go-ci.yml`, `e2e-ci.yml`, `release-ci.yml` (staging `stage.sh verify` incl. the new cache probe), `governance-ci.yml` | CI | reported on the PR; not claimed here |

## 6. Proxy verification

**Configuration inspected.** `deploy/staging/nginx/nginx.conf` proxies `location /api/` and `location /health/` with plain `proxy_pass`, no `proxy_cache`, no `proxy_hide_header`, no `proxy_ignore_headers`, and `add_header` only for HSTS, `nosniff`, `Referrer-Policy` and CSP (none of which is `Cache-Control` or `Vary`); `gzip` is not enabled, so nginx does not rewrite `Vary`. nginx forwards upstream `Cache-Control`, `Pragma`, `Vary` and `Set-Cookie` unchanged by default. **No nginx change was needed or made.**

**Local harness (executed).** nginx 1.24 was run with the staging `nginx.conf` verbatim except for four local substitutions (pid path, self-signed certificate paths, `api:8080` → `127.0.0.1:18080`, static root). Upstream: `apiserver.NewHandler` with the default production configuration.

| Request through nginx (TLS) | Status | Cache-Control | Pragma | Vary |
|---|---|---|---|---|
| `GET /api/v1/session` | 200 | `no-store` | `no-cache` | `Cookie` |
| `GET /api/v1/identity/users` | 200 | `no-store` | `no-cache` | `Cookie` |
| `GET /api/v1/zz-unknown` | 404 | `no-store` | `no-cache` | `Cookie` |
| `PUT /api/v1/session` | 405 | `no-store` | `no-cache` | `Cookie` |
| `POST /api/v1/auth/login` | 200 | `no-store` | `no-cache` | `Cookie` |
| `GET /api/v1/identity/users/not-a-uuid` | 400 | `no-store` | `no-cache` | `Cookie` |
| `GET /health/live`, `/health/ready` | 200 | — | — | — |
| `GET /`, `GET /assets/app.js` (nginx static) | 200 | — (ETag / Last-Modified kept) | — | — |

Each header appeared exactly once; HSTS and `X-Correlation-ID` were preserved. The same probe against the **baseline** binary (`3906831`) reported `missing` for `/api/v1/session` and the 404 (both had `no-store` from the old path list / problem writer but no `Vary: Cookie`); a baseline `200` data read (`/api/v1/identity/users`) had no `Vary` either, and data reads outside the old list had no `Cache-Control` at all, and `missing` with the API stopped (nginx 502) — so the probe can fail.

**Staging harness (CI gate added).** `deploy/staging/stage.sh verify` gained `cache_probe`, which requests `/api/v1/session`, an unknown `/api` route and `/` through the real staging stack (compose: nginx `web` + `api` + PostgreSQL) and fails the verification unless both API responses have exactly one `Cache-Control: no-store` line and a `Vary` with `Cookie`, and `/` answers 200 without `no-store`. The results are written to the verification event (`cache_api_session`, `cache_api_problem`, `cache_static`). This runs in `release-ci.yml` on this PR (Docker is not available in the authoring container, so it was not run locally; the probe function itself was exercised locally against the nginx harness above).

**Not inferred.** No production proxy, CDN or load balancer configuration exists in the repository. Production proxy behaviour is **NOT VERIFIED** and is not inferred from staging.

## 7. Browser and proxy limitations

- `no-store` governs HTTP caches (browser HTTP cache, shared caches that honour RFC 9111). It does not by itself guarantee exclusion from the browser back/forward cache, session history, devtools, memory dumps, crash reports or extensions; browsers increasingly exclude `no-store` pages from bfcache but this is browser behaviour, not a guarantee.
- Intermediaries that ignore `Cache-Control` / `Vary` (misconfigured CDNs, forced caching) are outside the application's control.
- Responses that `net/http` writes before any handler runs (malformed request line, oversized headers → `400` / `431`) carry no body data and are not covered by the middleware.
- The development identity stub selects the caller by the `X-Dev-User` header, not a cookie; `Vary: Cookie` does not describe that dimension, but `no-store` already prevents reuse. The stub is refused in production.
- The SPA's `fetch` sets no `cache` mode; with `no-store` on every response the browser HTTP cache keeps nothing to reuse. No frontend change was made.

## 8. Before / after

| Response | Before | After |
|---|---|---|
| `/api` data 200 (e.g. customers, cases, invoices, commission) | no `Cache-Control`, no `Vary` | `no-store`, `Pragma: no-cache`, `Vary: Cookie` |
| `/api` session / auth / identity / effective-access | `no-store`, `Pragma: no-cache` | same + `Vary: Cookie` |
| `/api` problem responses (4xx / 5xx) | `no-store` (problem writer) | same + `Pragma: no-cache`, `Vary: Cookie` |
| Any response clearing a cookie | `no-store`, `Pragma` | same + `Vary: Cookie` |
| `/health/*` without cookie change | no cache header | unchanged |
| Static SPA (nginx) | nginx defaults (ETag / Last-Modified) | unchanged |
| Bodies, statuses, other headers | — | unchanged |

## 9. Compatibility and authorization

- **No authorization change.** No file under `internal/authorization`, any module's `guardOp` / scope assertion, role grants, capabilities, feature flags or row scopes was touched. `crossSiteGuard` and `sessionAuth` logic is unchanged except that the cookie-clearing branch calls `httpx.MarkPrivate` instead of the removed local `setNoStore` (adds `Vary: Cookie`).
- No change to API paths, methods, OpenAPI files, generated code, DTOs, cookies (`__Host-`, `Secure`, `HttpOnly`, `SameSite=Strict`), CSRF, database schema, migrations, legacy CRM, `web/`, or financial behaviour.
- `deploy/staging/stage.sh` gained a read-only verification probe; `nginx.conf` and `compose.yaml` are unchanged.
- Rollback boundary: revert the PR (headers only, no data).

## 10. Status

A2 (F-07 / NT-17 / SR-12 header part) is implemented and tested in code. **Production remains NO-GO.** A3 and A4 were not started. HR and Finance runtime remain **NOT LIVE VERIFIED**.
