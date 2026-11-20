# RM-E04O — Seller identity chain characterization

**EVIDENCE ONLY — NOT APPROVED.** This records how the code on `main` behaves today. It approves no policy, registers
no capability, creates no grant and authorizes no Seller read. M2 records 1 and 2 stay `NOT_APPROVED_VERIFIED`;
Group B stays **closed**; production **NO-GO**.

| | |
|---|---|
| Baseline | `main` @ `3dc5136aa0aef8736ff1af447fce3e57eb760b7e` (PR #86, RM-E04N, merged) |
| Tests | [`internal/integration/legacy/shadow/rm_e04o_identity_chain_test.go`](../../internal/integration/legacy/shadow/rm_e04o_identity_chain_test.go): 7 tests (A–G), disposable PostgreSQL via `pgtest`, SYNTHETIC data only |
| Harness | the legacy shadow harness (`setupSales`: staging loader + identity, organization, cases, custody migrations), the same one the R7 suites build on, with `legacytest.HRScenario` accounts plus synthetic leads 701–705 and items 601–602 |
| Runtime changes | none (no Go runtime, SQL migration, OpenAPI, grant, capability, gate, tracker or approval change) |

## 1. Identifier namespaces

| # | Namespace | Where | Produced by | Notes |
|---|---|---|---|---|
| N1 | Legacy WordPress user id (`wp_users.ID`, e.g. `"3"`) | staging; `identity.user_account.source_key` with `source_system = 'legacy_wp'` | legacy snapshot | provenance only, never target identity |
| N2 | Legacy `user_login` | `identity.user_account.login_evidence` | identity import | evidence only; **not** a login name (test D2) |
| N3 | Account id `identity.user_account.user_id` (uuid) | identity | minted once per `(source_system, source_key)` by `identity/store.UpsertUser` (`store.go:73`) | unique per system; refreshes keep it (E1) |
| N4 | Credential `login_name` | `identity.credential` | **no production path** (D-34) | FK + UNIQUE on `user_id`; one credential per account |
| N5 | Session token (256-bit) / its SHA-256 / `session_id` | cookie / `identity.session` | `session.Login` (`session.go:160`) | only the digest is stored; `session.user_id` FK → N3 |
| N6 | `websession.Principal.UserID` | request context | `session.Authenticate` (`session.go:274`) = `session.user_id` | resolves only if not revoked, not expired, credential `ACTIVE`, `c.user_id = s.user_id`, account `PRESENT` (`session.go:292`) |
| N7 | `authorization.Subject.PrincipalID` = `.UserID` | per request | `SessionSubjects.Subject` (`authorization/session.go:38`) | grants = ACTIVE only (none can exist) |
| N8 | `custody.assignment.legacy_holder_key` | custody | `shadow.CustodyCommand` (`sales.go:344`) | N1 of the holder |
| N9 | `custody.assignment.legacy_holder_user_id` | custody | `Users.custodyRef` (`sales.go:126`) from `identityquery.ResolveUsers(legacy_wp, keys)` (`cmd/crm-legacy-import/shadow.go:193`) → `imports.go:132` | resolved **at import time**; no FK, no existence check |
| N10 | `CurrentHolder.UserID` | custody read | `custody/query.CurrentHolder` (`query.go:219`) = N9 | |
| — | Person / employment (`organization.employment.user_id`) | organization | organization import | **not on the chain**; nothing on the chain reads it (D3) |

Chain: **N1 →(identity import)→ N3 ←FK— N5 →(Authenticate)→ N6 →(SessionSubjects)→ N7**, and
**N1 →(ResolveUsers at custody import)→ N9 → N10**. The two branches meet only through N3: equality
`N7 == N10` holds because both are N3 values reached from the same N1 key in the same `source_system`.

## 2. Evidence matrix

Real = production code executed against PostgreSQL. Seams are named in the test file header.

| Stage | Input identity | Output identity | Verification | Result |
|---|---|---|---|---|
| Staging → identity import | N1 `"3"` (`legacy_wp`) | N3 uuid, `SHADOW_IMPORTED`, `login_enabled=false` | real `IdentityCommand` + `ImportIdentity` (A) | VERIFIED_END_TO_END |
| Identity → custody import | N1 seller key | N8 = N1, N9 = N3 | real `ResolveUsers` + `CasesCommand`/`CustodyCommand` + imports (A) | VERIFIED_END_TO_END |
| Credential provisioning | N3 | N4 bound to N3 | **test SQL** (no enrolment command exists) | BLOCKED_BY_EXISTING_ARCHITECTURE |
| HTTP login → session | N4 + password | N5, `session.user_id` = N3, `credential_id` | real `POST /api/v1/auth/login` (production `apiserver` config) + SQL check (A) | VERIFIED_END_TO_END |
| Session → HTTP principal | N5 cookie | `GET /api/v1/session` `session.user_id` = N3 | real HTTP stack incl. `sessionAuth` (A, F) | VERIFIED_END_TO_END |
| Session → Subject | N5 | N7 = N3, no grants | real `Authenticate` + real `SessionSubjects{Grants: identity ActiveGrants}`; principal attached by the middleware's own two calls (unexported `sessionAuth`) (A) | VERIFIED_PARTIAL_CHAIN |
| Holder read | case id | N10 = N3, `LEGACY_HOLDER_COLUMNS` (lead TARGET_UNKNOWN, item MATCH) | real `CurrentHolder` (A) | VERIFIED_END_TO_END |
| Subject vs holder | N7, N10 | equal; existing Slice 11 comparator `Eligible` (`read.go:120`) true on own lead | real comparator, actor built as `cases/http/write.go:101` (A) | VERIFIED_PARTIAL_CHAIN (comparator is a write guard, not an own-read) |
| Authoritative join | token digest | `session.user_id = user_account.user_id = assignment.legacy_holder_user_id` and `source_key = legacy_holder_key` | one SQL join (A) | VERIFIED_END_TO_END |
| Cross-Seller | A, B | distinct N3; each N7 equals only its own N10; swapped pairs never equal; comparator `ROW_SCOPE_DENIED` | real chain for both (B) | VERIFIED_END_TO_END (identity separation only) |
| Unmapped holder key | N8 `"77"` | N9 NULL, `Known=false` `NO_RESOLVED_LEGACY_HOLDER`; no account invented; resolves after a later batch imports the account | real imports (C) | VERIFIED_END_TO_END |
| Unverified holder uuid | custody command with uuid ∉ identity, and key `"11"` paired with a uuid that is not key 11's account | stored; `Known=true` with that uuid | real `ImportCustody` (command contract, not the connector) (D1) | SECURITY_GAP |
| Session for nonexistent account | uuid ∉ identity | credential insert refused `23503 credential_user_id_fkey` | DB (D1) | VERIFIED_END_TO_END |
| Account without credential | N3 of key 11; tried names `dev-hr-user-011`, other | `INVALID_CREDENTIALS`, no session | real `Login` (D2) | VERIFIED_END_TO_END |
| HR employment INACTIVE | key 10 | login OK, N7 = N3, holder Known | real chain + organization import (D3) | VERIFIED_END_TO_END (behaviour; policy undecided) |
| Account ABSENT_IN_LATEST | COMPLETE identity batch without key 9 | existing session → no subject; login refused `ACCOUNT_NOT_PRESENT`; **holder stays Known = B, also after a fresh sales import** | real imports + `Authenticate` (D4) | VERIFIED_END_TO_END; holder part SECURITY_GAP |
| Key repetition / refresh | same N1, changed content | same N3, `version=2`, one row | real import (E1) | VERIFIED_END_TO_END |
| Same key, other system | `syn_other`/`"3"` | different N3; resolution per system; custody keeps `legacy_wp` account | real import + `ResolveUsers` (E2) | VERIFIED_END_TO_END |
| Duplicate key in a batch | two rows key `"3"` | `ErrInvalidImport`, nothing written | real command (E3) | VERIFIED_END_TO_END |
| Second account for a key | DB insert | `23505 user_account_source_unique` | DB (E4) | VERIFIED_END_TO_END |
| Re-key by identity owner role | `UPDATE source_key` | onto B's key: `23505`; onto a fresh key: **permitted** | owner role, rolled back (E5) | SECURITY_GAP |
| Change `user_id` by owner role | `UPDATE user_id` | with credential/grant: `23503`; account without dependants: **permitted** (custody would not notice) | owner role, rolled back (E5) | SECURITY_GAP |
| Session lifecycle | valid, missing, malformed, unknown, logout, logout-all, replaced at login, idle expiry, absolute expiry, flag OFF, credential disabled after issue | only *valid* yields a subject; every other case yields `Subject{}`, comparator `ROW_SCOPE_DENIED`, evaluator `UNAUTHORIZED`; HTTP `authenticated=false` for unknown and revoked | real `Login`/`Authenticate`/`Logout`/`LogoutAll` (F) | VERIFIED_END_TO_END (credential disable via test SQL) |
| Privilege inference | imported `sn_seller` label, matching holder | SHADOW grant only; subject has 0 grants; no runtime capability allowed; Seller bundle empty even in Shadow mode; ACTIVE grant insert `23514 role_grant_shadow_only`; `cases/billing/payments.own.read` unregistered; gate file closed | (G) | VERIFIED_END_TO_END |
| Seller own-read | — | no endpoint exists | — | NOT_TESTED (does not exist; not created) |

## 3. Answer to the primary question

Yes, **for identity only**: the real import → login → session → `SessionSubjects` path produces a `Subject.UserID`
equal to `CurrentHolder.UserID` for the same synthetic Seller, both being the identity account uuid minted for the
legacy key, and never equal across Sellers. Two links are not production-executed: credential provisioning (no
command exists) and the attachment of the principal to the context (unexported middleware; replicated with its own
two calls and checked separately over HTTP). Equality is **not** a guarantee of validity: custody does not verify
that the uuid exists, is `PRESENT`, belongs to the recorded key, or can authenticate. What makes a matching subject
trustworthy today is the session side (`Authenticate` requires an active credential and a `PRESENT` account), not the
custody side.

## 4. Findings, ranked

| # | Severity | Finding | Evidence |
|---|---|---|---|
| F1 | High | Custody accepts any well-formed holder uuid and any (key, uuid) pair: no FK (cross-schema), no existence or key↔uuid consistency check. Only connector discipline keeps it right. Extends RM-E04N test 850. | D1 |
| F2 | High | A holder fact outlives its identity: an `ABSENT_IN_LATEST` account stays `Known` holder, and a fresh sales import re-resolves it because `UserIDsByKey` (`store.go:130`) ignores presence. Not reachable by that user (no session), but any holder-keyed list/count would include it, and a later `Reappeared` account regains everything silently. | D4 |
| F3 | Medium | Identity owner role may re-key `source_key` and change `user_id` of an account without dependants; no immutability guard. Either would silently redirect or orphan custody resolution. No code path does this. | E5 |
| F4 | Medium (policy) | Authentication ignores HR employment status; an INACTIVE seller with a credential gets a subject that matches their holder facts. Whether that must block is a human decision. | D3 |
| F5 | Low | Disabling a credential leaves the session row unrevoked (`revoke_reason CREDENTIAL_DISABLED` exists in the schema; no code writes it). Resolution still refuses it, so no access results; audit trail is incomplete. | F |
| F6 | Info | No credential provisioning command (D-34); a true E2E chain cannot be produced without a test seam. | A |

## 5. Comparison with RM-E04N

| RM-E04N item | Now |
|---|---|
| §3.7 / §5 "identity end-to-end UNKNOWN" | **Resolved to VERIFIED_PARTIAL_CHAIN**: intended space confirmed by execution; the only non-production links are provisioning and principal attachment |
| Holder user id not present in identity (test 850) | Confirmed and extended (F1: contradicting key/uuid pair also admitted) |
| No organization key / single deployment | Unchanged |
| Deny-list guard, freshness, set-based query, cases presence, key-level parity (711), PR-1…PR-6 | Unchanged; not in scope |
| New | F2 (absent account stays holder), F3, F4, F5 |

## 6. Guarantees

Established (synthetic, current code): one account per `(system, key)`; same key in another system never resolves to
it; session principal = the account bound to the credential; inactive/expired/revoked/unknown sessions and flag OFF
yield no subject; distinct Sellers never share a uuid; identity equality creates no grant or capability.

Not established: holder uuid validity or presence (F1, F2); immutability of account keys/ids (F3); any policy on
employment status (F4); Seller row-scope enforcement (no own-read exists); real-data distribution (ADR #18).

## 7. R5-SEC-01 person identity

Not needed for any step of the account-level chain (N1→N3→N5→N7 and N1→N9→N10): nothing on it reads
`organization.person` or `employment`. It would be needed only if a decision ties eligibility to employment (F4) or
must treat several legacy accounts or profiles as one person (e.g. HRScenario account 7 with two profiles). This
document does not claim R5-SEC-01 is approved.

## 8. Engineering changes that would need later authorization

Holder uuid existence/presence check or an identity-verified resolver for any own-read (F1, F2); a presence-aware
resolution in the custody connector (F2); DB immutability guard on `user_account.source_key` / `user_id` (F3);
revoking sessions on credential disable (F5); an approved provisioning command (D-34). All need their own approval;
the Seller read itself remains Group B (B2a → B2 → B4).

## 9. Reproduce

```bash
CRM_TEST_DATABASE_URL=postgres://…/postgres CRM_TEST_REQUIRE_POSTGRES=1 go test -count=1 -v -run RME04O ./internal/integration/legacy/shadow/
```

Rollback: delete the test file and this document; nothing else depends on them.
