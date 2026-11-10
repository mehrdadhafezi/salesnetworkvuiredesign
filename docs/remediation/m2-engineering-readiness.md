# M2 — Engineering readiness for seller authorization (post-approval slices)

**DRAFT — NOT APPROVED.** Baseline `a1c2ac34f08fd32ba9c33d6c36c1dfca27ea5e1e`.

This document defines the smallest implementation slices that become eligible **only after** the relevant
decisions reach `APPROVED_VERIFIED` in `docs/governance/r6-decision-tracker.json`. It approves nothing, changes
no tracker status and authorizes no implementation. **Production remains NO-GO.**

Companion: `m2-authorization-decision-intake-fa.md` (the M2 management package). Earlier analysis:
`m2-d42-seller-authorization-fa.md` (PR #50).

## 0. Rules for every slice

- A slice starts only when **every** decision in its "Requires" column is `APPROVED_VERIFIED`. Meetings,
  merged documents, green CI and `DECISION_RECORDED_PENDING_VERIFICATION` do not count.
- The two proposed decisions (seller capability eligibility, seller read scope) have **no ID yet**. Below they
  are written as *Proposal 1* and *Proposal 2*. These are labels, not IDs. A slice that depends on them waits
  for ID assignment, inventory registration and verified approval.
- Implementation follows the approved option text exactly. If the approved text is ambiguous, engineering
  records a blocker. It does not pick an interpretation.
- No test-only resolver, test-only grant or direct capability grant from data may stand in for the production
  path in acceptance tests.
- Everything runs on synthetic data in disposable databases. There is no deployment, no real customer data, no
  write-back and no financial execution.
- Activation (feature flags in a real environment) is a separate authorization after implementation
  (`r6-engineering-followup-map.md`).

## 1. Current state (evidence)

| Area | Today | Evidence |
|---|---|---|
| Accounts | Imported accounts are `SHADOW_IMPORTED` with `login_enabled = false`. A row in `identity.credential` is the only target login enablement. The application role has SELECT only, so no code writes a credential | `internal/identity/migrations/sql/00001_identity.sql:167-168`; `00012_identity_authentication.sql` |
| Credential lifecycle | `credential.status IN ('ACTIVE','DISABLED')`; login names may not look like phone numbers | `00012_identity_authentication.sql` |
| Sessions | R1 server-side sessions, logout, logout-all, idle expiry | `internal/identity/session`; R7-1 E2E |
| Grants | `identity.role_grant` has `CHECK (status = 'SHADOW')` (`role_grant_shadow_only`). No grant / revoke / audit command | `00001_identity.sql:195` |
| Seller role | Capability bundle is empty | `internal/identity/api/api.go` |
| Read scope | Every capability read is `ScopeOrganization` | `internal/authorization/authorization.go:163` |
| Existing own-row pattern | Commission's "own" read is capability-gated and then row-filtered by the caller (`guardOp(..., own bool, ...)`). This is a presentation of an existing pattern, not a new capability | `internal/commission/http/http.go:61-83` |
| Contact-outcome row check | Command allows only the current holder; unknown holder → `BLOCKED`; anyone else → `UNAUTHORIZED` | `internal/cases/command/contact_outcome.go`; Slice 11 matrix |
| Browser E2E | A seller logs in, then gets 403 on case read and outcome record (asserted, not skipped) | `web/e2e/seller-contact-outcome.spec.ts` (R7-1) |
| Person identity | Not modelled. SoD fails closed with `IDENTITY_PERSON_LINK_UNMODELLED` | `r5-person-identity-sod-DRAFT.md` |

## 2. Slices

### S1 — Account provisioning and lifecycle

| Item | Content |
|---|---|
| Requires (verified) | **D-34**. Also **R5-SEC-01** if D-34 option 3 (external IdP) is chosen |
| Smallest scope | One approved migration that grants the provisioning command, and only it, INSERT / UPDATE on `identity.credential`. A provisioning command implementing the D-34 option. Credential disable (`DISABLED`) and re-enable. Idempotency, outbox event, audit record. Imported shadow accounts and their history stay untouched |
| Must not | Copy legacy WordPress hashes, phone numbers or sessions; use a phone as login name; grant any capability; activate `user_account` rows by rewriting history |
| Acceptance | Account created through the approved path logs in (R7-1 harness). The first secret follows the D-34 option. A disabled credential cannot log in and its sessions are revoked. A replayed idempotency key gives the same result. The audit shows actor, time and outcome |
| Safe before approval | R1 behaviour is already covered (wrong password, logout, session invalidation in R7-1; identity unit tests). Design notes may be written. **No credential-writing code is merged** |

### S2 — Active role grants and revocation

| Item | Content |
|---|---|
| Requires (verified) | **D-42**, **Proposal 1**, and engineering prerequisite **E-7** completed. R5-SEC-01 only if the approved D-42 option requires person-level maker ≠ checker (D-42 option 2 relies on SoD) |
| Smallest scope | A migration that redefines `role_grant_shadow_only` so `ACTIVE` / `REVOKED` grants are allowed only with a new approved `basis`, distinct from `LEGACY_PROFILE_ROLE`. Existing SHADOW grants are unchanged. Append-only grant history. `GrantRole` / `RevokeRole` commands that check grant authority exactly as approved in D-42. Seller role bundle = exactly the Proposal 1 capabilities |
| Must not | Infer grants from legacy role names or HR position (2.0.160 grants the seller panel by position, `class-sn-plugin.php:13764`); grant capabilities directly from data; bundle `custody.custody.read` or `sales.report.read` into the seller role without their own decision |
| Acceptance | A grant made by an authorized role through the production command is effective on the next request. A grant attempt by an unauthorized role → `UNAUTHORIZED`, nothing written. Revocation → next request 403 `ACTION_DENIED`. History shows grant and revoke with actor and basis. Replay is idempotent |
| Safe before approval | E-7 design (audit fields, history shape) as a document. Tests that pin today's rule: an `ACTIVE` grant row is rejected by the database and a session subject resolves to zero capabilities |

### S3 — Server-enforced seller read scope

| Item | Content |
|---|---|
| Requires (verified) | **Proposal 2** (and Proposal 1 for the capability itself) |
| Smallest scope | A scope value in the evaluator for the approved basis (for example current holder). Query-level filtering in **every** case-data read: list (search, filter, sort, paging, totals), detail, history, attribution, contact-outcome reads, custody reads and the sales coverage report if those capabilities are ever granted. Out-of-scope = not-found with the same status and body shape. Counts computed after scope. Phone search limited to scope. Reuse the existing own-row pattern (commission) rather than a new mechanism |
| Must not | Hide only in the UI; derive scope from Original Owner, Original Seller, Credit Owner or event actor unless the approved text says so; treat `UNKNOWN` / `MULTIPLE` / `CONFLICT` custody as visible |
| Acceptance | Negative tests N1–N8 from `m2-d42-seller-authorization-fa.md` §6, run on the R7-1 harness with synthetic data |
| Safe before approval | A complete inventory of endpoints that return case data (including exports and reports), as a document or a test listing. Tests that pin today's organization-wide behaviour, so the later change is explicit. Evidence check that `custody.assignment.legacy_holder_*` corresponds to legacy `sn_leads.seller_id` (management question Q10) |

### S4 — Contact-outcome positive E2E through real authorization

| Item | Content |
|---|---|
| Requires (verified) | S1, S2 and S3 merged; Proposal 1 includes `cases.contact_outcome.record` |
| Smallest scope | Replace the R7-1 refusal assertions for the seller-with-grant case with acceptance tests 1–8 from `m2-d42-seller-authorization-fa.md` §6. The account comes from the S1 path. The grant comes from the S2 command, issued by an authorized role. Keep a no-grant seller in the suite to preserve the refusal tests |
| Must not | Use a test resolver, dev auth or direct SQL grant to reach the positive path. Remove the write-flag-off project |
| Acceptance | Seller records an outcome on a held case → `SUCCESS`, `applied = true`, visible in history. Another seller's case → `UNAUTHORIZED` from the holder check (not from a missing capability). Same Idempotency-Key → same result, one row. Revoked grant → 403. Flag off → `BLOCKED` / `FEATURE_DISABLED`, `applied = false` |
| Safe before approval | R7-1 refusal tests stay green. Backend holder-check tests (`writes_http_e2e_test.go`) already exercise the row layer with test subjects. These are backend-only and never count as real-authorization evidence |

### S5 — Audit and person-level separation-of-duties controls

| Item | Content |
|---|---|
| Requires (verified) | **R5-SEC-01** option A or B for person-level SoD. **D-42** for grant-change audit (grant audit itself ships in S2 via E-7) |
| Smallest scope | Person entity and append-only account-to-person link records with independent link review (option A) or IdP federation (option B). An SoD evaluation that compares persons as of decision time. Maker ≠ checker by person for grant approval if D-42 requires it |
| Must not | Merge identities automatically or by phone, name or e-mail; rewrite historical event actors; enable any finance, refund or HR apply action (they have their own decisions) |
| Acceptance | Two accounts linked to one person cannot act as maker and checker. An unlinked account → SoD-dependent action `BLOCKED`. Unlink is recorded; decisions made under a wrong link are flagged, not rewritten |
| Safe before approval | Existing fail-closed tests for `IDENTITY_PERSON_LINK_UNMODELLED` stay green. If R5-SEC-01 is C or D, S5 does not start, and every SoD-dependent action stays blocked |

## 3. What needs a decision vs. what is safe now

| Work | Needs verified decision | Safe now (existing rules, synthetic data, no activation) |
|---|---|---|
| Credential write / provisioning command | D-34 (+ R5-SEC-01 for IdP) | — |
| R1 login, logout, session expiry tests | — | Yes (exists, R7-1) |
| ACTIVE grant, grant / revoke commands | D-42, Proposal 1, E-7 | — |
| Tests pinning "no ACTIVE grant possible" | — | Yes |
| E-7 audit procedure design (document) | — | Yes (engineering prerequisite, not a business decision) |
| Seller capability bundle | Proposal 1 | — |
| Read-scope enforcement | Proposal 2 | — |
| Endpoint inventory for case data; legacy-holder evidence check | — | Yes |
| Positive contact-outcome browser E2E | S1–S3 + Proposal 1 | — |
| Refusal E2E and backend holder-check tests | — | Yes (exist) |
| Person model and person-level SoD | R5-SEC-01 (A / B) | — |

## 4. Dependency sequence

```
M1: signing authority per domain + decision-inventory owner (Q1, Q2)
  │
  ├─ ID assignment + inventory/tracker registration of Proposal 1 and Proposal 2
  │
M2 decisions → independent human verification (T5) → APPROVED_VERIFIED
  │   R5-SEC-01 ⇄ D-34 / D-42  (circular prerequisite; managers resolve, Q3)
  │
  ├─ E-7 grant-change audit procedure (engineering prerequisite; can be designed before M2)
  │
  ├─ S1 provisioning            needs D-34
  ├─ S2 grants + revocation     needs D-42 + Proposal 1 + E-7          (independent of S1 in code)
  ├─ S3 read scope              needs Proposal 2                        (independent of S1/S2 in code)
  │
  ├─ S4 positive E2E            needs S1 + S2 + S3 merged
  │
  └─ S5 person-level SoD        needs R5-SEC-01 (A/B); parallel to S1–S4; required before any SoD-dependent action
```

S1, S2 and S3 touch different modules and can be separate PRs in parallel once their decisions are verified.
S4 is the first point where a real seller completes the flow. S5 never blocks the non-sensitive seller flow, but
nothing SoD-dependent activates before it.
