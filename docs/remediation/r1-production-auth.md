# R1 — Production authentication and sessions: implementation matrix

Status: **technical authentication readiness implemented; business account provisioning BLOCKED.**
Production remains **NO-GO**.

| Item | Value |
|---|---|
| Baseline | `main` = `84c373c3f674116a06eec6068d889467041f7de1` (Slice 15 audit PR #40 merged). |
| Branch | `feat/readiness-r1-production-auth` |
| Slice 15 blocker addressed | P0-1 / E-1 "no production authentication or session". The **technical** part is done. Provisioning (who gets a credential, and how) remains blocked on D-34 / D-42. |
| Legacy | `plugin/current` 2.0.160 is unchanged. Legacy authentication (WordPress passwords, generated passwords, SMS OTP, gateway auto-login `wp_set_auth_cookie`) was read as evidence only. Nothing is reused. |

## 1. Audit before coding (Step 1)

| Finding | Evidence |
|---|---|
| Production requests were all anonymous (`authorization.Anonymous`); the dev stub refuses production | `cmd/crm-api/writes.go` (before R1), `internal/platform/authstub/authstub.go:35,46`, `internal/platform/apiserver/config.go:79` |
| Every anonymous / dev subject use (11 sites in `cmd/crm-api`) | read.go ×5, biawin.go, commission.go, finance.go, writes.go, operations.go, payments.go: now one `subjects(devAuth)` |
| Identity model is shadow-only: `user_account_state = SHADOW_IMPORTED`, `user_account_no_login`, `role_grant_shadow_only` | `internal/identity/migrations/sql/00001_identity.sql`: **kept unchanged** |
| Runtime authority counts ACTIVE grants only; role bundles are code-defined; only the HR role carries capabilities | `internal/authorization/authorization.go`, `internal/identity/api/api.go` `roles` |
| Architecture assigns User / Credential / Session to **identity** | Arch V1 §5 (line 263), §6 (line 323), §7 (line 356) |
| The sensitive-read audit recorded only the dev principal | `internal/reporting/http/http.go`, **fixed**: it now records the session `user_id` and `session_id` |
| Reason-code taxonomy is frozen (E0 §9C) | no new code added; auth failures use `ACTION_DENIED` / `FEATURE_DISABLED` plus two RFC 9457 problem types |
| Open decisions | D-26 (customer auto-login), D-34 (initial password / provisioning), D-42 (who grants capabilities), D-45 (audit-failure behaviour), ADR #18 (PII / staging privacy) |

## 2. What is implemented (decision-independent subset)

| Control | Implementation | Evidence |
|---|---|---|
| Login | `POST /api/v1/auth/login`: password verified against an identity credential (Argon2id, `golang.org/x/crypto/argon2`, PHC encoding, OWASP baseline m=19 MiB t=2 p=1, bounded parameters) | `internal/identity/password`, `internal/identity/session/session.go` `Login` |
| Uniform failure | Wrong password, unknown login, disabled credential and absent account all return the same 401. A dummy Argon2 derivation equalises timing. The reason is recorded internally only. | `TestLoginOutcomes`, `TestHTTPLoginFailures` |
| Server-side sessions | `identity.session`: 256-bit `crypto/rand` token. Only its SHA-256 digest is stored. Opaque base64url. | `00012_identity_authentication.sql`, `websession.NewToken/Digest` |
| Cookie | `__Host-crm_session`; Secure, HttpOnly, SameSite=Strict, Path=/, no Domain; Max-Age = remaining absolute lifetime | `websession.Cookie`, `TestCookieAttributes`, `TestHTTPLoginCookieAndSession` |
| Expiry | Idle 30 min (sliding, write-throttled to 1/min) and absolute 12 h (never moves). Configurable within bounds via `CRM_AUTH_SESSION_IDLE` / `CRM_AUTH_SESSION_ABSOLUTE`. | `session.Config`, `TestSessionLifecycle`, `TestHTTPSessionStates` |
| Revocation | Logout (current session), logout-all (every session of the caller). Conditional UPDATE makes them idempotent and race-safe. A disabled credential or absent account ends sessions at the next request. Flag OFF rejects every session (kill switch). | `TestSessionLifecycle`, `TestHTTPConcurrentLogout` |
| Fixation | The server never adopts a client-chosen token. A session held at login is revoked (`REPLACED_AT_LOGIN`) and a new token is minted. | `TestSessionLifecycle` (6), `TestHTTPSessionStates` |
| CSRF | Session-bound token `HMAC-SHA256(session token, "crm-csrf-v1")`, never stored, required in `X-CSRF-Token` on every unsafe request with a session. `Sec-Fetch-Site: cross-site` is refused. `Origin` must be an allowed origin (`CRM_AUTH_ALLOWED_ORIGINS`, https in production). SameSite=Strict on top. | `internal/platform/apiserver/auth.go` `crossSiteGuard`, `TestHTTPCSRF`, `TestSessionAuthorizationBoundaries` |
| Credential abuse | A sliding per-login throttle (5 failures / 15 min, configurable), with attempts serialised per login key by an advisory lock so bursts cannot exceed it. This is **not** an account lockout. | `TestThrottle` (20 concurrent → exactly 5 evaluated), `TestHTTPLoginFailures` (429) |
| Authentication middleware | Resolves the cookie on every request. An invalid cookie is cleared and the request continues anonymously. The middleware never reads `X-Dev-User`. | `sessionAuth` |
| Current user | `GET /api/v1/session`: `auth_mode=session`, `session_state` (none / active / invalid), `session` (user id, display name, CSRF token, expiries), `login_enabled` | `TestHTTPLoginCookieAndSession` |
| Identity ≠ authority | `authorization.SessionSubjects`: subject = session user; authority = that user's **ACTIVE** role grants only. None exist today, because legacy evidence is SHADOW. Synthetic DEV grants and direct capability sets are dropped. A loader error fails closed (BLOCKED). | `internal/authorization/session.go`, `TestSessionSubjects*`, `TestSessionAuthorizationBoundaries` |
| Dev / production separation | The dev stub is refused in production (unchanged). Dev stub plus sessions is a startup error (`ErrAuthConflict`). The production resolver ignores the stub principal. `X-Dev-User` has no effect in production. | `TestHTTPNoDevBypassInProduction`, `TestSessionSubjectsAnonymousAndNoDevFallback`, `TestAuthSurfaceConfig` |
| Auditable events | `identity.auth_event` is append-only: LOGIN_SUCCEEDED / FAILED / THROTTLED, LOGOUT, LOGOUT_ALL, SESSION_REPLACED, CSRF_REJECTED. It holds no credential, token, raw login, IP address or user agent (an unknown login is stored only as a SHA-256). | `TestLoginOutcomes` (17), `TestHTTPLoginCookieAndSession` (18: logs) |
| Frontend | Persian RTL login page with every route protected in session mode. Session restore on refresh; expired-session notice; logout; CSRF header on unsafe requests (memory only, never storage); re-check after a 401/403 and at idle expiry, with no polling. | `web/src/shared/auth/*`, `web/src/app/Shell.tsx`, `auth.test.tsx` |

## 3. Flows that remain DISABLED (policy-dependent)

| Flow | Why | Owner decision |
|---|---|---|
| Credential enrolment / account provisioning | Who gets an account, and how the first secret is set and delivered. The application role has `SELECT` only on `identity.credential`. Lint rule `r1-credential-write` refuses any production write. | D-34 (+ HR / Security) |
| Initial password distribution | same | D-34 |
| Password change / reset / recovery ("forgot password") | Recovery channel and proof (SMS / e-mail / admin) undecided | D-34 + Security |
| Password strength / length / rotation policy | Policy. Only a 1024-byte denial-of-service bound exists. | Security |
| MFA / OTP | Factor policy undecided. Legacy SMS OTP not reused. | Security (P-SEC) |
| Customer login / portal / auto-login after payment | Customer identity and D-26. `issuance_reason` is fixed to `PASSWORD_LOGIN` by a DB CHECK. | D-26, Product + Security |
| Reuse of legacy WordPress passwords or sessions | Explicitly not done. Only `$argon2id$` verifies (phpass / bcrypt / md5 rejected, tested). | Security |
| Account activation in identity / ACTIVE role grants | `user_account_no_login`, `user_account_state`, `role_grant_shadow_only` all stay (CI asserts it) | D-42 |
| Account lockout | Not invented. Only the self-recovering throttle exists. | Security |
| Session retention / purge of expired rows | Rows are kept, with no delete grant | D-44-like retention decision |
| Per-IP throttling and trusted-proxy client IP | Depends on the deployment topology. No IP is stored. | Security + operations |
| Concurrent-session limit | Not a policy here. Logout-all exists. | Security |

**Net effect today:** even with `identity.session_auth` ON, nobody can log in in production, because no credential exists and none can be created. And even a test account that logs in has no authority, because no ACTIVE grant exists. This is the intended fail-closed state.

## 4. Required tests (Step 5)

| # | Scenario | Test |
|---|---|---|
| 1 | Valid login / logout | `TestLoginOutcomes`, `TestHTTPLoginCookieAndSession`, `TestHTTPCSRF`, `auth.test.tsx` |
| 2 | Invalid credentials | `TestLoginOutcomes`, `TestHTTPLoginFailures` (uniform body) |
| 3 | Expired session | `TestSessionLifecycle` (idle and absolute) |
| 4 | Revoked session | `TestSessionLifecycle`, `TestHTTPSessionStates` (logout-all) |
| 5 | Stolen / malformed token | `TestSessionLifecycle` (tampered, foreign, injection), `TestHTTPSessionStates` |
| 6 | Session fixation | `TestSessionLifecycle`, `TestHTTPSessionStates` (client-chosen cookie never adopted) |
| 7 | CSRF | `TestHTTPCSRF` (missing / wrong / other-session token, foreign Origin, cross-site), `TestSessionAuthorizationBoundaries` |
| 8 | Repeated login attempts | `TestThrottle` (serialised burst), `TestHTTPLoginFailures` (429) |
| 9 | Inactive user | `TestLoginOutcomes` (disabled credential, absent account, no credential), `TestSessionLifecycle` (disabled mid-session) |
| 10 | Unauthorized role | `TestSessionSubjectsIdentityNotAuthority`, `TestSessionSubjectsGrantFiltering` (HR ≠ Sales / Finance), `TestSessionAuthorizationBoundaries` (real session → 403) |
| 11 | Unauthorized row scope | `TestSessionAuthorizationBoundaries` (other seller's case 404; supervisor without per-service scope cannot assign) |
| 12 | Feature flag OFF | `TestLoginOutcomes`, `TestHTTPLoginFailures` (login BLOCKED, sessions rejected), `TestSessionAuthorizationBoundaries` (surface flag stays effective) |
| 13 | Dev bypass rejected in production | `TestHTTPNoDevBypassInProduction`, `TestSessionSubjectsAnonymousAndNoDevFallback`, `TestAuthSurfaceConfig`, `TestRunRefusesDevAuthInProduction` |
| 14 | Refresh with a valid session | `TestSessionLifecycle` (sliding idle), `TestHTTPLoginCookieAndSession`, `auth.test.tsx` |
| 15 | Refresh with an expired session | `TestHTTPSessionStates`, `auth.test.tsx` (expired notice) |
| 16 | Concurrent logout / requests | `TestSessionLifecycle` (exactly one revocation), `TestHTTPConcurrentLogout` |
| 17 | Security event history | `TestLoginOutcomes` (events, append-only), `TestHTTPCSRF` (CSRF_REJECTED) |
| 18 | No secrets in logs | `TestHTTPLoginCookieAndSession` (log buffer), `TestLoginOutcomes` (DB rows), `TestAuditRecordsSessionActor` |
| 19 | PostgreSQL integration | every `internal/identity/session`, `internal/identity/http` and `internal/integration/biawin` test above runs on real PostgreSQL |
| 20 | Slices 0–14 regression | full `go test ./...`, web suite, CI migrate invariants |

Also covered:
- Finance approval and posting are not capabilities anyone can hold (`TestNoFinancialExecutionCapability`).
- Identity changes never rewrite history (`TestSessionAuthorizationBoundaries`).
- The identity owner role holds no grant outside its schema (CI and test).

## 5. Deployment requirements (not executed)

1. Apply migration `00012_identity_authentication.sql` with `crm-migrate`. This is additive.
2. Provide a **dedicated** identity login for `CRM_IDENTITY_AUTH_DATABASE_URL`, plus the flag store login.
3. Set `CRM_AUTH_ALLOWED_ORIGINS` to the exact https origin(s) of the SPA. Serve the SPA and the API from the **same site** behind TLS, so `__Host-` / Secure / SameSite=Strict work.
4. Terminate TLS at a proxy you control. Add HSTS, and per-IP rate limiting / WAF at the edge (not implemented in the app).
5. Keep `identity.session_auth` **OFF** until P-SEC (D-34 provisioning, recovery, MFA) is decided and a provisioning path is approved and implemented.
6. Ship logs without request bodies. The app never logs credentials or tokens, but a proxy might.
7. Optionally override the session parameters (idle / absolute / throttle) within the bounds in `session.Config.Validate`, after Security sign-off.
