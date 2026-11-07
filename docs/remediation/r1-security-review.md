# R1 — Security review report (prepared for independent review)

**Scope.** Commits on `feat/readiness-r1-production-auth` over `84c373c`. This is a self-assessment that
prepares an **independent** security review. It is not a substitute for one.

## 1. Architecture

```
browser ──HTTPS──▶ proxy (TLS, HSTS, edge rate limit: operator) ──▶ crm-api
  crm-api middleware: Correlation → Recover → LimitBody → sessionAuth → crossSiteGuard → allowlists → handlers
  sessionAuth:    __Host-crm_session cookie → identity session (digest lookup) → websession.Principal on ctx
  crossSiteGuard: unsafe methods → Sec-Fetch-Site / Origin / X-CSRF-Token (session-bound HMAC)
  handlers:       authorization.Guard → SessionSubjects → identity ACTIVE role grants → evaluator (unchanged)
```

## 2. Threats and controls

| Threat | Control | Residual risk / note |
|---|---|---|
| Password guessing (online) | Per-login sliding throttle, serialised per key; Argon2id cost | No per-IP or global throttle in the app (edge requirement). Credential stuffing across many logins is limited only by Argon2 cost and the edge. |
| Account enumeration | Uniform 401 body and status; dummy Argon2 for unknown or disabled accounts; reason recorded server-side only | Throttle status (429) is per login key, so it reveals only that the key had 5 recent failures, not whether the account exists. |
| Offline cracking after DB theft | Argon2id with a 16-byte random salt, OWASP baseline cost; parameters stored per hash (upgradable) | Cost parameters need Security sign-off. No pepper / HSM, a key-management decision. |
| Session theft (XSS) | HttpOnly cookie; token never in a body, never in storage; CSRF token kept in memory only | An XSS could still act as the user within the page (it can read the CSRF token from memory). Mitigate with CSP and output encoding; a CSP header is not yet set by the app (edge / follow-up). |
| Session theft (network) | Secure plus the `__Host-` prefix; TLS is an operator requirement | HSTS must be set at the edge. |
| Session fixation | Server-minted tokens only; the held session is revoked at login | — |
| CSRF | SameSite=Strict, an Origin allowlist, `Sec-Fetch-Site`, and a session-bound HMAC token on every unsafe request; login guarded by Origin and Sec-Fetch-Site | Requests without Origin or Sec-Fetch-Site (non-browser clients) skip the Origin check but still need the CSRF token whenever a session cookie is presented. |
| Login CSRF | Origin and Sec-Fetch-Site on login; SameSite=Strict | — |
| Token guessing | 256-bit random tokens, digest lookup, fixed-length format check before any query | — |
| Timing on token lookup | Index lookup on the SHA-256 digest (no secret-dependent compare); constant-time CSRF compare | — |
| Replay after logout | Server-side revocation is checked on every request | — |
| Privilege escalation via identity | Sessions give identity only. ACTIVE grants only; synthetic DEV grants and direct capability sets are dropped. Slice 4 constraints (no login, shadow-only grants) are kept. No grant or activation path exists. | Granting is D-42. |
| Dev bypass in production | Stub refused in production; stub plus sessions is a startup error; the resolver ignores the stub principal; CI grep keeps stub use confined | — |
| Secrets in logs or DB | No password, token, CSRF token, raw login, IP or user agent in logs, events or sessions (tested) | Proxy and access logs are an operator responsibility. |
| Denial of service via hashing | 1024-byte password bound; stored-parameter bounds (≤ 256 MiB, ≤ 10 iterations, ≤ 8 lanes); throttle | Concurrent legitimate logins still cost about 19 MiB each; size the hosts accordingly. |
| Unbounded growth | — | Session and auth_event rows are never deleted (retention decision open). |

## 3. Cryptography

| Use | Implementation |
|---|---|
| Password hashing | `golang.org/x/crypto/argon2` `IDKey` (RFC 9106). No custom primitive. |
| Tokens | `crypto/rand`, 32 bytes. |
| Digests | `crypto/sha256`. |
| CSRF token | `crypto/hmac` with SHA-256, keyed by the session token. |
| Comparisons | `crypto/subtle`. |

## 4. Outstanding security decisions (P-SEC package)

These are existing IDs, not new questions:

1. **D-34:** provisioning, the initial password and its delivery, recovery and reset. Login cannot be used
   until this is decided.
2. **D-26:** customer authentication and the gateway auto-login. Not replicated.
3. **D-42:** who may grant capabilities. ACTIVE grants cannot exist without it.
4. **D-45 and ADR #18:** whether auth events may store an IP address or user agent (privacy); behaviour
   when audit writes fail.
5. **Security parameter sign-off** under P-SEC:
   - session idle and absolute lifetimes
   - throttle budget and window
   - Argon2 cost
   - MFA requirement
   - concurrent-session policy
   - retention of session and auth-event rows

## 5. Reviewer checklist

- [ ] Read `internal/identity/session/session.go`, `internal/identity/password/password.go`,
      `internal/platform/websession/websession.go` and `internal/platform/apiserver/auth.go`.
- [ ] Confirm migration `00012` grants: credential SELECT only; session UPDATE limited to
      expiry and revocation columns; no DELETE anywhere.
- [ ] Run the R1 CI step locally against PostgreSQL.
- [ ] Verify the proxy configuration (TLS, HSTS, CSP, edge rate limits, log redaction) in the target
      environment.
- [ ] Penetration test of login, CSRF and session handling in a staging environment with
      synthetic accounts.
- [ ] Confirm `identity.session_auth` stays OFF until P-SEC is approved.

## 6. Verdict

Technical authentication readiness: **READY FOR INDEPENDENT REVIEW**.

Business provisioning readiness: **BLOCKED** (D-34, D-42).

Production: **NO-GO**.
