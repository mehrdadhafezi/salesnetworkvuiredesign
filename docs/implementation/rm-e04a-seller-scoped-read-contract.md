# RM-E04A — Seller-Scoped Read Contract and Security Architecture

| | |
|---|---|
| Mode | **READ-ONLY AUDIT.** No code, schema, OpenAPI, migration, grant, bundle, flag or data change |
| Branch | `docs/rm-e04a-seller-scoped-read-contract` |
| Baseline | `main` @ `d85b770028aaa206b678a47d31db35c3be418500` (PR #68, RM-E03, merged) — verified. Revision 2 re-checked at `e5fabbe` (PR #69 merge; docs only, no code change) |
| Production | **NO-GO** (unchanged) |
| Status of every proposal below | **DRAFT — NOT APPROVED. No option selected.** |

This document closes the open item in RM-E03 §11: "the Seller-scoped reads have no contract". It records the server facts that exist today and the security architecture that any Seller-scoped read must follow. It also lists the business decisions that must be made before any of it can be built.

It does **not** decide which capabilities the Seller holds (**M2 proposed record 1**, eligibility), which rows the Seller sees (**M2 proposed record 2**, read-scope policy), or who may grant, edit and revoke (**D-42**, grant administration authority). Account provisioning (**D-34**), person identity / SoD (**R5-SEC-01**) and real-data authority (**ADR #18**) are further separate decisions. D-42 does not approve Seller data access. All six stay open; see [`../governance/proposals/rm-e04a-seller-decisions.md`](../governance/proposals/rm-e04a-seller-decisions.md). Per AGENTS.md, scope predicates in §4 are presented as **options with consequences**. They are not selected rules.

### Document set (revision 2)

| Document | Content |
|---|---|
| this file | findings F-01 … F-09, security requirements SR-01 … SR-16, contract options |
| [`../security/rm-e04a-seller-row-scope.md`](../security/rm-e04a-seller-row-scope.md) | seven-screen contracts S1 … S7 |
| [`../security/rm-e04a-seller-leak-surface-audit.md`](../security/rm-e04a-seller-leak-surface-audit.md) | 105-operation leak-surface inventory |
| [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) | NT-01 … NT-24 (supersets T1 … T14) |
| [`../governance/proposals/rm-e04a-seller-decisions.md`](../governance/proposals/rm-e04a-seller-decisions.md) | decision separation and precision corrections |
| [`rm-e04a-execution-plan.md`](rm-e04a-execution-plan.md) | future PR groups A … D |
| [`rm-e04a-executive-summary.fa.md`](rm-e04a-executive-summary.fa.md) | Persian executive summary |

Naming note: the R7 audit row `RM-E04` is "shared read components (MetricCoverage, LinkConfidence, …)" and is still **NEEDS DESIGN FREEZE**. RM-E04A is a separate, prerequisite security contract for the Seller reads. It does not unblock RM-E04's design freeze.

---

## 1. Sources used

Frozen / governing: `AGENTS.md`; `docs/remediation/rm-access-01-authorization-contract.md` (RM-ACCESS-01); `docs/remediation/m2-d42-seller-authorization-fa.md` (M2, §4–§6); `docs/governance/proposals/rm-b02-d42-decision-package.md` (RM-B02 §3–§4); `docs/implementation/rm-e03-seller-panel.md` (RM-E03 §11).

Code evidence (baseline commit, read only): `internal/authorization/`, `internal/{cases,custody,billing,payments,customers,commission,reporting,finance,biawin,fulfillment}/http/`, `internal/cases/command/read.go`, `internal/custody/query/query.go`, `internal/payments/command/command.go`, `internal/commission/query/query.go`, the module migration bodies, and `api/openapi/openapi.yaml`.

No full audit was restarted. No runtime was exercised.

---

## 2. Current-state findings

### F-01 — The evaluator has only two scopes: `NONE` and `ORGANIZATION`

`internal/authorization/authorization.go:72-78` defines `Scope` as `NONE | ORGANIZATION`. `Evaluate` returns `ORGANIZATION` for every allowed decision (`:163`) and `NONE` for every denial. `OrgContext` is accepted but "deliberately never a source of authority" (`:39-42`). `Request` carries only a capability, with no resource (`:68-70`).

**Consequence.** Today any holder of `cases.case.read`, `billing.invoice.read`, `custody.custody.read` and the other reads gets the **whole organisation**. That is the RM-B02 statement "every read is organization-wide today". No Seller capability may be granted on these endpoints as they stand (M2 §5; RM-B02 §7).

### F-02 — Scope is asserted by only some read handlers (latent fail-open) — **HIGH, prerequisite**

Some handlers refuse an allowed decision whose scope is not `ORGANIZATION`. They return `UNAUTHORIZED / ROW_SCOPE_DENIED`:

| Asserts `Scope == ORGANIZATION` | Evidence |
|---|---|
| commission (org reads; the own read is filtered separately) | `internal/commission/http/http.go:77` |
| reporting | `internal/reporting/http/http.go:109` (and `query/spec.go:162-165` rejects any non-ORGANIZATION report scope) |
| finance | `internal/finance/http/http.go:71` |
| biawin | `internal/biawin/http/http.go:69` |
| fulfillment | `internal/fulfillment/http/http.go:63` |

Other handlers check only `Gate` (flag plus `Allowed`). They then run the organisation-wide query:

| Does **not** assert scope | Evidence |
|---|---|
| cases (list, detail, history, attribution, coverage report) | `internal/cases/http/http.go:87-101` |
| custody (assignments, case custody, parity) | `internal/custody/http/http.go:69-80` |
| billing (invoices, items, stages, totals, reconciliation) | `internal/billing/http/http.go:125-140` |
| payments (attempts, receipts, history) | `internal/payments/http/http.go:103` |
| customers, organization, identity, provider-ops, write-back reads | `internal/{customers,organization,identity}/http`, `internal/integration/{providerops,writeback}/http` |
| finance review queue, biawin activation decisions, fulfillment work cases, HR requests, payment submission / batch / evidence reads (revision 2) | `finance/http/review.go`, `biawin/http/activation.go`, `fulfillment/http/work.go`, `organization/http/hrrequest.go`, `payments/http/write.go` — guarded by their own `Check` calls, outside the package `guardOp` that asserts scope |

**Today this is safe,** because `Evaluate` can never return `Allowed=true` with any scope other than `ORGANIZATION`. **It becomes a data leak** as soon as the evaluator learns a narrower scope. Suppose a Seller is given `cases.case.read` with an "own" scope. `GET /api/v1/cases` would then return every case in the organisation, because the handler never looks at `Decision.Scope`. These are exactly the endpoints the Seller panel needs (cases, custody, billing, payments).

**Scale (revision 2, reconciled against all 105 OpenAPI operations):** **51 GET handlers** have no scope assertion; 37 of them return organisation-wide Seller-relevant rows or aggregates. See the leak-surface audit §2–§3.

**Required before any non-ORGANIZATION scope exists (engineering, no business decision needed):** the guard must be applied on **every** data-returning operation — not only the endpoints a Seller screen uses — because a capability could later be granted with a narrower scope on any of them. Every data-returning handler must make an exhaustive decision on `Decision.Scope`. It serves ORGANIZATION queries only for `ORGANIZATION`, serves scoped queries only for a scope it explicitly implements, and denies anything else (`ROW_SCOPE_DENIED`). Add a regression test per module that injects an allowed decision with an unknown scope and expects 403 with no rows. This change has no behaviour impact today. It is the first follow-up in §7.

### F-03 — Precedents for own-row scope already exist and agree

| Precedent | Predicate | Unknown / missing handling | Evidence |
|---|---|---|---|
| `commission.own.read` → `GET /commission/me/earnings` | `historical_entitlement.holder_user_id = caller` in SQL | `Subject.UserID == ""` → 403 `ROW_SCOPE_DENIED` ("never all") | `commission/http/http.go:427-450`; `commission/query/query.go:468` |
| Contact outcome eligibility (Slice 11) | custody current holder `= caller` | no user → `ROW_SCOPE_DENIED`; holder unknown → `DEPENDENCY_UNKNOWN`; not present → `STALE_STATE` | `cases/command/read.go:88-123` |
| Custody current holder | `assignment.legacy_holder_user_id` only when PRESENT, not MULTIPLE, parity not CONFLICT / MULTIPLE_CURRENT | otherwise `Known=false` with a named basis | `custody/query/query.go:194-223` |
| Payment submission target | invoice `seller_user_id = caller` **and** `issued_by_user_id = caller` | either NULL → `BLOCKED / DEPENDENCY_UNKNOWN`; mismatch → `ROW_SCOPE_DENIED` (OPD-10) | `payments/command/command.go:144-160` |

These give a vocabulary that is already in use and should be reused, not reinvented: `ROW_SCOPE_DENIED`, `DEPENDENCY_UNKNOWN`, `STALE_STATE`, and "unknown never widens".

### F-04 — Ownership facts available for a predicate (and their gaps)

| Fact | Column | Kind | Gap |
|---|---|---|---|
| Current custody (legacy master) | `custody.assignment.legacy_holder_user_id` + `derivation`, `parity`, `presence` | current, reconcilable projection | refreshed per import batch; can be stale between batches; `NO_RESOLVED_LEGACY_HOLDER` when the user mapping is missing |
| Derived custody | `custody.assignment.derived_user_id` | derived from events | may disagree with legacy (`parity`) |
| Original owner / original seller / credit owner | `cases.attribution_snapshot` (`facet`, `resolution`, `subject_user_id`) | historical snapshot, append-only | `resolution` can be UNKNOWN / CONFLICT / NOT_RECOVERABLE; `subject_user_id` is nullable even when RESOLVED |
| Invoice seller / issuer | `billing.invoice.seller_user_id`, `issued_by_user_id` | source facts | **no link-state column**: a NULL cannot tell "no seller" from "seller key not mapped to a user". It must be read as UNKNOWN, never as "not mine" plus a zero count |
| Invoice ↔ case | `billing.invoice.case_id` + `case_link_state` | link with state | UNKNOWN / CONFLICT links must not be used to place an invoice in scope |
| Commission credit holder | `commission.historical_entitlement.holder_user_id` | historical | already used by F-03 |

The phone is not a key anywhere in these tables (`cases` migration header: "Phone, names and seller usernames are never keys"). No predicate below uses it (AGENTS.md Identity).

### F-05 — Subject identity is per account, not per person

`SessionSubjects.Subject` sets `UserID = PrincipalID = session user` (`authorization/session.go:38`). Own scope is therefore **account-scoped**. Person identity is not modelled (R5-SEC-01 open). This is enough for a read scope. It is **not** enough for any SoD-sensitive action, and nothing here enables one.

### F-06 — Not-found and out-of-scope responses

Detail endpoints answer a missing row with `404` + `authzhttp.NotFound(ctx)` (e.g. `cases/http/http.go:158,197,229`). M2 §5 requires an out-of-scope row to be **indistinguishable** from a missing one. A scoped handler must therefore apply the predicate **inside** the lookup query, so that out-of-scope rows produce the same `404` body. It must not load the row and then refuse with 403, because a 403 here would confirm the row exists.

### F-07 — Data responses carry no `Cache-Control: no-store` — **MEDIUM (hardening; no active leak claimed)**

| Evidence class | Finding |
|---|---|
| **Verified application behaviour** | `no-store` is set only on problem responses (`platform/httpx/problem.go:106`), auth responses (`apiserver/auth.go:142`) and metrics (`apiserver/metrics.go:229`, `worker/metrics.go:142,150`). No middleware in `internal/platform/httpx` or the `apiserver` chain sets a cache header (the only middlewares set the correlation header and record metrics). Successful JSON data responses carry **no** `Cache-Control`, `ETag` or `Last-Modified`. The session cookie is `__Host-`, `Secure`, `HttpOnly`, `SameSite=Strict` (`platform/websession/websession.go:136`). The SPA fetches with `credentials: 'same-origin'` and sets no `cache` mode (`web/src/shared/api/client.ts:57`). |
| **Verified deployment configuration (staging only)** | `deploy/staging/nginx/nginx.conf` proxies `/api/` to the API with **no** `proxy_cache` directive (nginx does not cache unless configured) and adds HSTS / nosniff / Referrer-Policy / CSP but **no** `Cache-Control`. |
| **Deployment behaviour not verified** | No production proxy / CDN configuration exists in the repository. Browser heuristic caching, back/forward cache and any intermediary in production are **NOT VERIFIED**. |

Because responses carry no validators and staging has no proxy cache, **no active cache leak is claimed**. Own-scope responses would make the gap material, so the hardening (execution plan A2: one middleware + NT-17) should land before any Seller-scoped read.

### F-08 — Seller destinations versus server data that exists

| RM-E03 destination | Server data today | Status |
|---|---|---|
| (full per-screen contracts: [`../security/rm-e04a-seller-row-scope.md`](../security/rm-e04a-seller-row-scope.md)) | | |
| S1 My numbers (leads + queues + counts) | `cases.work_case` / `cases.lead` + custody holder; contact outcomes (`cases.contact_outcome_*`) | data exists; **no scoped read**; queue definitions are not a server contract |
| S2 Ready for conversion | case attribution + invoices linked by `case_id` | partial; "ready for conversion" has **no server status** (must not be invented) |
| S3 Repeat action | invoice stages + custody | **no hand-off model**; issuer versus current custody versus next actor not modelled as a read |
| S4 Invoices | `billing.invoice`, stages, totals; `payments` attempts / receipts | data exists; organisation-scoped only; confirmed / submitted / unknown per stage partly available |
| S5 Customer behaviour | none (link opened, visit, paid after visit) | **no source** |
| S6 Extra-number request | none | **BLOCKED on OPD-10** |
| S7 Wallet | `commission.historical_entitlement` via `commission.own.read` | own read **exists** (historical, shadow); confirmed / pending / settleable semantics blocked on D-41 / OPD-09 / ADR #18 |

### F-09 — Payment submission context is an existence oracle — **LOW–MEDIUM** (revision 2)

`GET /api/v1/payments/invoices/{invoice_id}/submission` (`payments/http/write.go:142`) returns 404 only when the invoice does not exist (`payments/command/read.go:67-104`). For an existing invoice owned by another seller it returns **200** with `eligible:false` and `ROW_SCOPE_DENIED`. Amounts and attempts are correctly withheld for a non-owner (`read.go:84-97`). This confirms that a given invoice ID exists, which SR-06 forbids. Reachable only with `payments.submission.read`, which no one holds today. Remediation: execution plan A3; test NT-03.

---

## 3. Security architecture (requirements for any Seller-scoped read)

These rules restate M2 §5, RM-ACCESS-01 and AGENTS.md as engineering requirements. They add no business capability.

| ID | Requirement |
|---|---|
| SR-01 | **Server-side only.** The row predicate is part of the SQL `WHERE` of every query: list, detail, counts, totals, search and export. Client filtering of an organisation-wide response is prohibited (RM-E03 §11). |
| SR-02 | **Scope is part of the decision.** Scope is produced by the evaluator from the grant's role bundle (never from a query parameter, header, role name, `OrgContext`, position or hierarchy). It is carried in `Decision.Scope`. Hierarchy ≠ permission (AGENTS.md). |
| SR-03 | **Exhaustive scope handling.** Every data-returning handler switches on `Decision.Scope`. An unknown or unimplemented scope is denied (`ROW_SCOPE_DENIED`). This closes F-02. |
| SR-04 | **No proven identity, no rows.** `Subject.UserID == ""` → 403 `ROW_SCOPE_DENIED`. It never falls back to organisation scope (commission precedent). |
| SR-05 | **Unknown never widens.** A row whose ownership fact is NULL, UNKNOWN, CONFLICT, MULTIPLE or NOT_RECOVERABLE is **outside** own scope. The Seller sees no such row and the server reports no count of them. Revealing how many unknown rows exist organisation-wide would itself leak. |
| SR-06 | **Non-enumeration.** Out-of-scope detail lookups return the same status code and body as a missing ID (F-06). Sub-resources (`/history`, `/attribution`, `/stages`, `/totals`, `/payments`, `/receipts`, `/contact-outcomes`) re-apply the parent's predicate. They never trust that the caller already saw the parent. |
| SR-07 | **Counts after scope.** Page totals, queue counters, KPI tiles and coverage figures are computed over the scoped set only. A scoped endpoint never returns an organisation aggregate. |
| SR-08 | **Search inside scope.** Phone, name, invoice code and ID search run inside the scoped predicate. Phone is a discovery input only and is never a join or an ownership proof (AGENTS.md Identity). |
| SR-09 | **Facts are not collapsed.** Current custody, original owner, original seller, credit owner, issuer and event actor stay separate fields. Which fact defines a destination's scope is a per-destination decision (§4). One fact must not stand in for another. |
| SR-10 | **History is not rewritten.** Historical attribution is read from `attribution_snapshot` / entitlements as recorded. A change in current custody or hierarchy never re-attributes past rows. |
| SR-11 | **Freshness.** Scope is evaluated on every request against current data, with no cached scope or row set (M2 N8). Custody comes from the latest import, so the response should carry the batch / as-of time it was evaluated against. |
| SR-12 | **No caching.** Scoped responses are sent with `Cache-Control: no-store` and `Vary: Cookie`, or the equivalent is proven at the proxy (F-07). |
| SR-13 | **Multi-role is a union and never widens.** A union is evaluated per capability. Holding an organisation-scope capability through another role does not make the Seller's own endpoints organisation-wide, and an own scope never makes an organisation endpoint answer (RM-ACCESS-01 test J; RM-B02 §4). |
| SR-14 | **Honest outcomes.** Responses use the existing `outcome` / `reasons` envelope: SUCCESS, PARTIAL (e.g. stages incomplete), UNKNOWN, CONFLICT, BLOCKED (flag off), UNAUTHORIZED. Missing amounts are never zero (AGENTS.md Sensitive Outcomes). |
| SR-15 | **Sensitive-read audit.** Whether a Seller read of customer contact data is audited, and what happens when the audit write fails, follows D-45 (safe default: fail closed). |
| SR-16 | **Fail closed on infrastructure errors.** A flag-store or holder-resolver error answers 500 / BLOCKED, never a partial or unscoped list (RM-ACCESS-01 §5). |

---

## 4. Contract shape — options (no option selected)

### 4.1 Where the scope lives

| Option | Description | Consequences |
|---|---|---|
| **A. Dedicated own-scope endpoints** ("me" family, like `GET /commission/me/earnings`), each guarded by its own `*.own.read` capability | e.g. `/api/v1/me/cases`, `/me/invoices`, … with a hard-coded own predicate | Matches the only shipped precedent. Organisation endpoints and their tests stay untouched. Counts are scoped by construction. **Needs new capabilities**, which is a governance change: registry + DB catalog + OpenAPI enum + `capabilityFlags` (RM-ACCESS-01 §11). Some query code is duplicated |
| **B. Scope as a property of the decision on existing endpoints** | `cases.case.read` granted with scope `OWN_*`; the existing handlers branch on scope | No new capability. Needs SR-03 on every handler first (F-02), and a per-grant or per-bundle scope model in the identity schema (migration). Higher regression risk, because one handler serves two populations |
| **C. Defer** | Seller panel stays on synthetic fixtures | No risk; no Seller value |

**Engineering view (not a decision):** option A has the smaller blast radius and an existing precedent. Option B is reasonable only once F-02 is fixed and a scope model is approved. Either way, F-02 should be fixed first, because option B depends on it and option A benefits from it as defence in depth. The choice is a governance decision, together with M2 proposed record 1.

### 4.2 Which fact defines "own" — per destination (M2 proposed record 2; not selected)

| Destination | Candidate predicates (from M2 / RM-B02 option list) | Notes |
|---|---|---|
| S1 leads / cases | 1 current holder (custody, legacy basis) · 2 current legacy assignment (`sn_leads.seller_id`) · 3 current + historical · 4 organisation · 5 later | M2 engineering proposal is 1, which matches the Slice 11 holder guard. Option 2 needs evidence M2 Q10. Choosing 1 differs from 2.0.160 |
| S4 invoices | a `seller_user_id = me` · b `issued_by_user_id = me` · c both (payments precedent) · d via linked case's holder | c is what the write path already uses. d depends on `case_link_state = RESOLVED`. The prototype copy "original issuer keeps sale ownership" points to a/b. **Not decided** |
| S4 payments / receipts | inherit from the invoice predicate | never scoped independently |
| S2 conversions / S3 repeat | requires the S1 and S4 decisions **and** a hand-off contract | No contract exists. BLOCKED |
| S7 wallet | `holder_user_id = me` (exists) | The scope is already shipped. The semantics (confirmed / settleable) are blocked on D-41 / OPD-09 / ADR #18 |
| Customer identity / contact points | case-linked read only, or none | GAP-04 / D-30. Not part of any proposed Seller bundle (M2 §5) |
| `custody.custody.read`, `sales.report.read` | — | Explicitly **not** in the proposed Seller bundle (M2 §5). Needs a separate decision |

### 4.3 Minimal read contract per endpoint (whichever option is chosen)

Each Seller-scoped endpoint must specify the following before implementation: capability; scope basis (fact + `basis` string as in custody); predicate SQL; unknown handling (SR-05); not-found parity (SR-06); counts (SR-07); search fields (SR-08); outcome mapping (SR-14); freshness / as-of field (SR-11); cache headers (SR-12); and its negative tests (§5). The RM-E03 `types.ts` are presentation types and **not** API contracts. The API shape is defined by OpenAPI under the existing envelope (`outcome`, `reasons`, `correlation_id`, `items`, `page`, `allowed_actions`).

---

## 5. Required negative test matrix (extends M2 N1–N8)

Revision 2: T1–T14 are kept below unchanged as the summary. The detailed specifications (invariant, fixture, actor, request, status, outcome, body, SQL assertions, source) and ten additional scenarios are in [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md) as NT-01 … NT-24. **They are future tests; none has run.**

Seller X; row Y owned by Seller Z; row U whose ownership fact is UNKNOWN.

| # | Test | Expected |
|---|---|---|
| T1 | List as X | only X's rows. Y and U are on no page, under any sort, filter or cursor |
| T2 | Totals / queue counts / KPI as X | equal to X's scoped count, never the organisation count. U is not counted |
| T3 | Detail and every sub-resource of Y, and of a random UUID | identical status and body (404 problem); no timing-dependent branch on existence |
| T4 | Search by Y's phone, name, code or ID | empty, count 0 |
| T5 | Cursor tampering (cursor taken from another user / org list) | no row outside scope; invalid cursor → 400 |
| T6 | `UserID` empty (session without a resolved user) | 403 `ROW_SCOPE_DENIED`, no rows |
| T7 | Evaluator returns allowed with an unknown scope (injected) on **every** organisation handler | 403 `ROW_SCOPE_DENIED` (regression for F-02) |
| T8 | X also holds an organisation-scope capability through another role | own endpoints stay own; organisation endpoints follow only their own capability |
| T9 | Custody of Y moves to X, then away | visible only while X is the holder; no stale cache (M2 N8) |
| T10 | Holder resolver / DB error | 500 or BLOCKED; never an unscoped list |
| T11 | Flag off | `BLOCKED / FEATURE_DISABLED`, no rows |
| T12 | Response headers | `Cache-Control: no-store` |
| T13 | Invoice with `seller_user_id` NULL | invisible to every Seller; not counted |
| T14 | Unknown stage outcome | stage shown as UNKNOWN; payable amount not computed (RM-E03 §8 rule, server-side) |

All tests run against synthetic data on the R7 harness (real browser → SPA → crm-api → disposable PostgreSQL). There is no real data (ADR #18) and no real account (D-34).

---

## 6. Blockers and dependencies

| Item | Blocks | State |
|---|---|---|
| D-42 (grant administration authority only) | issuing any ACTIVE Seller grant through a production path | READY_FOR_REVIEW, not closable (RM-B02 §6) |
| M2 proposed record 1 (Seller eligibility) | which reads the Seller holds; option A capabilities | proposed, no ID |
| M2 proposed record 2 (Seller read scope) | every predicate in §4.2 | proposed, no ID |
| D-34 (accounts), R5-SEC-01 (person identity) | real Seller login; person-level guarantees | open |
| ADR #18 (real data) | any test on real data | open |
| OPD-10 | S6; issuer-versus-holder for invoices and assisted issuance | open |
| D-30 / GAP-04 | customer identity fields to the Seller | open |
| D-41, OPD-09, D-36 | wallet semantics | open / evidence missing |
| D-45 | sensitive-read audit failure policy | READY_FOR_REVIEW (safe default: fail closed) |
| No product contract | S2 "ready for conversion", S3 hand-off, S5 behaviour events | needs a Codex-owned workflow / data contract; Claude must not invent statuses |

---

## 7. Follow-ups (bounded; none executed here)

Revision 2: the detailed, ordered plan with dependencies, code areas, tests, rollback boundaries and approval needs is [`rm-e04a-execution-plan.md`](rm-e04a-execution-plan.md) (Groups A–D). The list below is the original summary.

1. **RM-E04B — scope-assertion hardening (F-02); F-07 and F-09 as separate small PRs (plan A2, A3).** Engineering only, no decision needed, no behaviour change today. Add an exhaustive scope check to every data-returning handler, add T7 per module, and add `no-store` on data responses (or prove it at the proxy). Safe to do now.
2. **Scope model ADR** (Codex): option A versus B (§4.1), the scope vocabulary, and where scope is stored. This is input for the M2 record 1 and 2 sessions.
3. After M2 records 1–2 are `APPROVED_VERIFIED` (eligibility and scope; tested with synthetic grants), and later D-42 + D-34 for real grants and accounts and ADR #18 for real data: build Seller reads one destination at a time (S1 first, then S4), each with the §4.3 contract and the §5 tests, then wire RM-E03 screens to them behind the existing fail-closed navigation.
4. Product / workflow contracts for S2, S3, S5 and S6 before any API is designed for them.

Not allowed before the decisions: any grant, bundle change, scope implementation presented as approved, client-side scoping, or use of real data.

---

## 8. Verification of this audit

- Baseline `main` = `d85b770028aaa206b678a47d31db35c3be418500` confirmed with `git rev-parse`.
- Every code claim cites a file and line at that baseline. F-02 was established by listing the `guardOp` / `Check` call sites per module and grepping for `ScopeOrganization`. F-07 was established by grepping `Cache-Control` under `internal/`.
- Revision 1: the only change was this document. Revision 2: documentation only (`docs/`); no file under `internal/`, `cmd/`, `api/`, `web/`, `deploy/` or any migration changed. No test was needed for a docs-only change. `docs/governance` tests are unaffected (no governance file was touched).
- HR and Finance runtime remain **NOT LIVE VERIFIED**.
