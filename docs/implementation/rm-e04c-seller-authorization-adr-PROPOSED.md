# RM-E04C — Seller Authorization Architecture (proposed ADR, no ID)

| | |
|---|---|
| Status | **PROPOSED — NOT ACCEPTED.** No decision ID is created. No tracker status changes. |
| Baseline | `main` @ `c0d7f4bef709148673af6b679b9cbd63cb148b80` (PR #74 merged; PRs #71–#74 verified in history with `git log origin/main`) |
| Supersedes | nothing. Builds on RM-E04A (`rm-e04a-seller-scoped-read-contract.md` §4.1 options) and the merged RM-E04B Group A hardening |
| Runtime change | **None.** No code, OpenAPI, migration, grant, bundle, flag or data change |
| Ownership | Per `AGENTS.md` (Agent Ownership), engineering architecture is **Codex-owned**. This document was drafted by Claude on the owner's explicit instruction and needs **Codex architecture review** and **Security review** before it can be accepted. It does not override that ownership |
| Acceptance gate | Accepted only together with **M2 proposed record 1** (Seller capability eligibility) and **M2 proposed record 2** (Seller read scope) reaching `APPROVED_VERIFIED` (execution plan B1) |
| Production | **NO-GO** |

Companion documents (same series):

| Document | Content |
|---|---|
| [`../governance/proposals/rm-e04c-seller-decision-package.fa.md`](../governance/proposals/rm-e04c-seller-decision-package.fa.md) | Persian executive decision package (every open decision, options, recommendation, approvers, what stays blocked) |
| [`../security/rm-e04c-seller-screen-designs.md`](../security/rm-e04c-seller-screen-designs.md) | S1–S7 implementation-ready designs (data, API, capability, predicate, UI states, tests, readiness) |
| [`rm-e04c-seller-api-contract-PROPOSAL.md`](rm-e04c-seller-api-contract-PROPOSAL.md) | OpenAPI fragment for the recommended `/api/v1/me/...` operations (**not** in `api/openapi`) |
| [`../security/rm-e04c-seller-traceability-matrix.md`](../security/rm-e04c-seller-traceability-matrix.md) | screen → capability → ownership policy → SR → test → decision gate → PR; new tests NT-25 … NT-32 |
| [`rm-e04c-seller-implementation-roadmap.md`](rm-e04c-seller-implementation-roadmap.md) | PR roadmap in groups A–E |

---

## 1. Context — facts at HEAD (verified, not taken from earlier summaries)

| # | Fact | Evidence at `c0d7f4b` |
|---|---|---|
| C1 | The evaluator has two scopes only, `NONE` and `ORGANIZATION`; every allowed decision is `ORGANIZATION` | `internal/authorization/authorization.go:72-78`, `:163` |
| C2 | The `seller` role bundle is empty | `internal/identity/api/api.go:212` (`{RoleSeller, "Seller", nil}`) |
| C3 | The database forbids any non-SHADOW grant | `internal/identity/migrations/sql/00001_identity.sql:195` (`CONSTRAINT role_grant_shadow_only CHECK (status = 'SHADOW')`) |
| C4 | Grants are re-read from the store on **every** request; direct capability sets from data are discarded | `internal/authorization/session.go:33-58` |
| C5 | A store error while loading grants marks the subject `EvidenceConflict` → every decision `BLOCKED / LINKAGE_CONFLICT` | `session.go:42-45`; `authorization.go:117-119` |
| C6 | Every capability-guarded read now asserts a supported scope (`CheckRead(ctx, cap, supported...)`); an unsupported allowed scope is refused `ROW_SCOPE_DENIED` | `internal/authorization/guard.go:182-221` (RM-E04B / A1, PR #71) |
| C7 | Every `/api` response carries `Cache-Control: no-store` and `Vary: Cookie` | `httpx.CachePolicy` (RM-E04B / A2, PR #72) |
| C8 | Submission context answers a non-owner like a missing invoice (404 parity) | `internal/payments/command/read.go` `ownsTarget` (RM-E04B / A3, PR #73) |
| C9 | All 105 OpenAPI operations are classified; 7 are `OWN_READ`; adding an unclassified operation fails CI | `internal/authorization/http/testdata/operation_scope_registry.yaml`; `operation_registry_test.go` (RM-E04B / A4, PR #74) |
| C10 | The only shipped own-scope **read** of Seller-relevant data is `GET /api/v1/commission/me/earnings` (`commission.own.read`, SQL `holder_user_id = caller`, empty user → 403 `ROW_SCOPE_DENIED`) | `internal/commission/http/http.go:425-450`; `internal/commission/query/query.go:468,491` |
| C11 | Keyset cursors are base64 JSON `{s: sort, v: value, i: id}`, bound to the sort name only; not signed, not bound to subject or endpoint | `internal/platform/pagination/pagination.go:27-63` |
| C12 | Product authority: Seller READ DOMAIN = "own Case / source / invoice / stage / result / history / earnings"; MUST NOT OWN = "organisation-wide" | `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md` §2 (source of truth #1) |
| C13 | Product authority: Current Custody, Original Owner / Original Seller, Next Actor, Event Actor and Credit Owner are distinct; Seller sees Credit Owner "V own" only | XR §5 |
| C14 | Reason codes that already exist and fit the design: `ROW_SCOPE_DENIED`, `FIELD_DENIED`, `DEPENDENCY_UNKNOWN`, `STALE_STATE`, `COVERAGE_INCOMPLETE`, `OUTCOME_UNKNOWN`, `FEATURE_DISABLED`, `CONTRACT_INVALID` | `internal/kernel/reasoncodes` |

C12 matters: an organisation-wide Seller read would contradict source of truth #1. The open business question is therefore not *whether* the Seller is own-scoped but **which fact defines "own" per destination** (M2 proposed record 2) and **which reads the Seller holds** (M2 proposed record 1).

## 2. Decision drivers

1. Smallest blast radius if a predicate is wrong (leak of other sellers' rows or customer PII).
2. No change to the 79 organisation reads or to their tests.
3. Counts, search, sub-resources and cursors scoped **by construction**, not by remembering to branch.
4. Multi-role users can never widen a Seller read, and an own capability can never answer an organisation endpoint.
5. Revocation effective on the next request.
6. Reuse the existing vocabulary and precedents (C6, C8, C10, C14); invent no business state.
7. Keep unapproved alternatives out of runtime code.

## 3. Options

| Option | Shape | Pros | Cons / risks |
|---|---|---|---|
| **A. Dedicated own-scope operations** under `/api/v1/me/...`, each guarded by its own `*.own.read` capability, predicate hard-wired in SQL | Precedent C10. Org endpoints untouched. Counts / search / cursors scoped by construction. Registry classifies each op `OWN_READ` with one basis (C9). An own capability cannot match an org handler because the capability differs | New capabilities → governance change (registry, DB catalog, OpenAPI enum, `capabilityFlags`, RM-ACCESS-01 §11 checklist). Some query duplication |
| **B. Narrower `Decision.Scope` on existing endpoints** (`cases.case.read` granted with an own scope) | No new capability | Each org handler serves two populations; a missed branch in a list, count, search or sub-resource leaks the organisation. Needs a per-grant (or per-bundle) scope model and an identity migration. Multi-role users holding the same capability at two scopes force a union (`OR`) predicate — the easiest way to widen silently. Registry needs dual classification per operation |
| **C. Hybrid** — Option A endpoints **plus** an `OWN` scope value in the evaluator that is derived from the capability vocabulary (never from grant data) | Everything in A, plus a second, independent check: `/me` handlers accept only `OWN`; org handlers accept only `ORGANIZATION` (C6 already refuses anything else) | One small evaluator change (B2), still needs the governance change of A |
| **D. Defer** | Seller panel stays on synthetic fixtures | No Seller value |

## 4. Recommendation — Option C (Option A endpoints + capability-derived `OWN` scope)

**Explicit answer to "`/me/...` vs narrower scopes on organisation endpoints": use dedicated `/me/...` endpoints.** Do **not** add narrower scopes to existing organisation endpoints.

Why:

1. **Leak containment.** A defect in a `/me` handler can only expose what that handler's own SQL selects. A defect in a dual-population org handler exposes the organisation (RM-E04A F-02 was exactly that latent class).
2. **No per-grant scope storage.** Scope becomes a property of the capability in the code-defined vocabulary (`internal/identity/api`), not a column on `identity.role_grant`. No identity schema change is needed for scope; grants stay role-based (C4).
3. **Multi-role cannot widen.** Different capabilities → different endpoints. A user who is both Seller and Supervisor reaches the team endpoint only through the Supervisor capability and the own endpoint only through the Seller capability; neither changes the other's predicate (SR-13).
4. **Defence in depth.** The `OWN` scope check makes the own/org separation hold even if a future bundle accidentally contained the wrong capability for a handler.
5. **Precedent and tooling exist.** C9's registry already models `OWN:<basis>`; C10 is the reference implementation.

Accepted cost: duplicated read queries per module. Mitigation: each module exposes one `Own*` query function that takes the proven `userID` and the basis as required parameters; there is no code path that calls it without them.

### 4.1 Evaluator change (future PR B2; **not implemented**)

- Add `ScopeOwn Scope = "OWN"`.
- The vocabulary marks each capability with a scope class (`ORGANIZATION` or `OWN`). `Evaluate` returns the class of the requested capability on an allow. Nothing in grant data can change it (C4 already discards direct capability sets).
- `/me` handlers call `CheckRead(ctx, cap, ScopeOwn)`; org handlers keep `CheckRead(ctx, cap, ScopeOrganization)` (unchanged).
- `commission.own.read` (and the other `OWN_READ` capabilities: `fulfillment.work.read`, `org.hr_request.read`, `payments.submission.read`) move from `ORGANIZATION` to `OWN`. Their rows do not change (the SQL already filters), but this is **not behaviour-neutral**: `Decision.Scope` is served by `GET /api/v1/authorization/effective-access` as `AccessDecision.scope`, whose OpenAPI enum is `[NONE, ORGANIZATION]`, and NT-07 pins the own read at `ORGANIZATION`. It is a contract and authorization change, scheduled as roadmap **B2a** (Group B: after this ADR is accepted and M2 rec. 1 + 2 are `APPROVED_VERIFIED`), with the compatibility tests listed there.
- `EvaluateAll` / effective-access output show the scope class per capability (part of the same B2a contract change).

### 4.2 Capability candidates (engineering proposal only; names **not** registered; eligibility is M2 proposed record 1)

| Candidate key | Serves | Predicate basis (selected by M2 rec. 2) | Feature flag (existing) |
|---|---|---|---|
| `cases.own.read` | S1 list / detail / history / contact-outcome read; S2 attribution sub-resource | `CUSTODY_CURRENT_HOLDER` (recommended) | `sales.read_shadow` |
| `billing.own.read` | S4 invoice list / detail / stages / totals | `INVOICE_SELLER_AND_ISSUER` (recommended initial) | `billing_payments.read_shadow` |
| `payments.own.read` | S4 payments / receipts under an own invoice | inherits the invoice basis (never independent) | `billing_payments.read_shadow` |
| `commission.own.read` | S7 (exists) | `COMMISSION_HOLDER` (exists) | `commission.read_shadow` |

Explicitly **not** in any Seller bundle: `cases.case.read`, `billing.invoice.read`, `payments.attempt.read`, `custody.custody.read`, `sales.report.read`, `customers.customer.read`, `reporting.metrics.read`, any `*.write` or approval capability (M2 §5; RM-B02 matrix).

`cases.contact_outcome.record` (a write; M2 rec. 1 option A) is out of scope for this ADR. Its row guard already exists (Slice 11 holder check); its eligibility is a separate part of M2 rec. 1.

## 5. Scope bases (closed vocabulary)

Each basis is one SQL predicate fragment owned by the module query package. Unknown never widens (SR-05).

| Basis | Predicate (inside the `WHERE`) | Excluded (never "mine", never counted) | Source |
|---|---|---|---|
| `CUSTODY_CURRENT_HOLDER` | `custody.assignment.legacy_holder_user_id = $me` with `presence = PRESENT`, single current holder, `parity NOT IN (CONFLICT, MULTIPLE_CURRENT)` — the same conditions under which `CurrentHolder` returns `Known=true` | holder NULL / `NO_RESOLVED_LEGACY_HOLDER` / MULTIPLE / CONFLICT | `internal/custody/query/query.go:197-223` |
| `INVOICE_SELLER_AND_ISSUER` | `seller_user_id = $me AND issued_by_user_id = $me` | either NULL; either differs | `payments/command` `ownsTarget` (C8) |
| `INVOICE_SELLER` / `INVOICE_ISSUER` | single-column alternatives (M2 rec. 2 options a / b) | NULL | `billing.invoice` |
| `INVOICE_VIA_CASE_HOLDER` | invoice `case_id` with `case_link_state = RESOLVED` whose case passes `CUSTODY_CURRENT_HOLDER` | link UNKNOWN / CONFLICT | `billing.invoice.case_link_state` |
| `COMMISSION_HOLDER` | `historical_entitlement.holder_user_id = $me` | NULL holder | C10 |
| `REQUESTER` | reserved for S6; no table exists | — | XR §11 (extra-number "Seller requester") |

Only bases selected by M2 rec. 2 are ever compiled into runtime code. The others stay in this document (AGENTS.md: keep unapproved alternatives out of runtime).

## 6. Ownership facets — kept apart (SR-09, SR-10, XR §5)

| Facet | Source | Admits a row into a Seller list? | Shown to the Seller as |
|---|---|---|---|
| Current custody (case holder) | `custody.assignment` (legacy batch) | **yes** for S1 if rec. 2 = option 1 | `scope_basis` + `evaluated_as_of` (batch time) |
| Original owner / original seller | `cases.attribution_snapshot` (append-only) | **no** (decision RM-E04A-Q3; default no) | `facet`, `resolution`, `is_caller` boolean only |
| Historical holder | custody history | **no** (rec. 2 option 3 not recommended) | not shown in lists |
| Invoice seller / issuer | `billing.invoice.seller_user_id` / `issued_by_user_id` | **yes** for S4 per rec. 2 (a / b / c / d) | `is_caller` per facet; another user's id withheld (`FIELD_DENIED`) |
| Credit owner (entitlement holder) | `commission.historical_entitlement.holder_user_id` | S7 only; does **not** admit the invoice into S4 (RM-E04A-Q5; default no) | own lines only |
| Event actor | contact-outcome / stage history | no | own events: "شما"; others: withheld until RM-E04A-Q4 |
| Next actor | **not modelled** | no | `UNKNOWN` |

A change in current custody or hierarchy never re-attributes historical rows. Historical missing ownership stays `UNKNOWN` / `NOT_RECOVERABLE` (XR §5).

## 7. Customer data visibility and field masking

| Rule | Design |
|---|---|
| Default | `/me` DTOs carry `customer_ref { customer_id?, link_state }` only. No name, phone, document, address or bank data |
| Masking style | **Omission, not partial masking.** A withheld field is absent from the JSON and the envelope carries reason `FIELD_DENIED` with the field path in `notice`. Partially masked phones (e.g. last 4 digits) are not used: they still support re-identification and phone enumeration |
| Contact points (if GAP-04 permits) | Served by a separate per-case sub-resource (`/me/cases/{case_id}/customer-contact`) that re-applies the case predicate, writes a sensitive-read audit row first, and fails closed when the audit write fails (D-45 safe default) |
| Other staff identities (S-STAFF) | Replaced by `is_caller: true|false`; never another user's id or name until RM-E04A-Q4 is decided |
| Money (S-FIN) | Per unit, never summed across units; missing → absent with `OUTCOME_UNKNOWN` / `COVERAGE_INCOMPLETE`, never `0` |

## 8. Scope-aware search, counts and pagination

| Topic | Design |
|---|---|
| Search | Server-side, appended **inside** the scoped `WHERE` (`AND`). Allowed keys: case / invoice code (exact or prefix ≥ 4 characters). Phone search only if GAP-04 allows it: normalised **exact** match, never prefix or contains (prevents digit-by-digit enumeration), never a join key (AGENTS.md Identity). Unknown query parameters → 400 `CONTRACT_INVALID` (commission precedent) |
| Counts | Computed by the same scoped query (`COUNT(*) … WHERE <basis>`) in the same `READ ONLY` transaction as the page. Returned as `count { value, state: EXACT }` or `count { state: ABSENT, reason }`. Never taken from `/reports/*`, `/sales/coverage-report` or `/billing/reconciliation-report`. Rows excluded by SR-05 are not counted, and the number excluded is **not** reported (it would leak organisation volume) |
| Cursor | Reuse `internal/platform/pagination`. Each `/me` operation uses its **own sort name** (e.g. `me.cases.source_key`), so a cursor from an organisation list or another `/me` list fails `Decode` → 400. The predicate is re-applied on every page; the cursor never carries ownership. Optional hardening (not required for safety, because the predicate is in SQL): HMAC-bind cursors to `(user_id, operation)` |
| Limits | Existing `DefaultLimit 50`, `MaxLimit 200` |

## 9. Non-enumeration

1. Detail lookups put the predicate in the lookup query: out-of-scope and missing produce the **same** 404 problem body (C8 pattern).
2. Every sub-resource re-applies the parent predicate (it never trusts that the parent was seen).
3. One query path for both cases (no "load, then check" branch), so there is no existence-dependent timing branch.
4. IDs are UUIDs; no sequential IDs are exposed.

## 10. Multi-role users

- The evaluator already unions grants per capability (`authorization.go:120-150`). With Option C each capability has exactly one scope class, so the union never mixes scopes.
- Holding `cases.case.read` (org) through another role does not change `/me/cases`; holding `cases.own.read` never makes `/cases` answer (NT-18, NT-27).
- Navigation shows Seller destinations only from Seller capabilities (RM-ACCESS-01 §11; RM-E02 role navigation).

## 11. Permission revocation

- Grant revoked → next request is evaluated against the store again (C4) → 403 `ACTION_DENIED`. No grant, decision or scope cache exists, and none may be added for `/me` (SR-11).
- Session revoked → 401 on the next request (R1).
- A cursor issued before revocation becomes useless (the guard runs before the cursor is decoded).
- Custody moves away → the next request no longer returns the case (NT-16). `evaluated_as_of` tells the UI which import batch the answer reflects.
- Store error while loading grants → `BLOCKED` (C5), never a stale allow.

## 12. Separation of duties

- Own reads are **not** SoD-sensitive and are account-scoped (RM-E04A F-05). They are enough for reads.
- No Seller capability in this ADR approves, posts, reviews or grants anything.
- Person-level SoD (one human holding a Seller account and a Finance-reviewer account) is **not** solvable at account level. It stays blocked on **R5-SEC-01**. Whether such role combinations are refused at grant time is a question for R5-SEC-01 / D-42, not decided here.
- Grant issuance itself needs requester ≠ approver at person level (D-42 options 2 / 3) → also depends on R5-SEC-01.

## 13. Commission own-read semantics (S7)

- Scope: already correct (C10).
- Meaning: lines are **historical shadow entitlements** from a stored legacy run. They are not "confirmed", "settleable" or "withdrawable". Those words need D-41, OPD-09 and D-36, which are `EVIDENCE_MISSING`.
- The UI must label totals as historical credit from run X (as-of), keep settlement «— تعریف نشده», and never present a sum as payable.
- An entitlement's `invoice_id` links to S4 only if S4's own predicate admits the invoice (default: it does not admit it by itself).

## 14. Consequences

- Positive: org endpoints unchanged; each Seller read independently testable; the registry and NT-07 / NT-22 guards extend naturally.
- Negative: 3 new capabilities (governance change); duplicated read SQL; the S4 initial basis (`INVOICE_SELLER_AND_ISSUER`) hides assisted-issuance invoices from both people until OPD-10 is decided — a known, documented limitation, not a defect.
- Neutral: no identity schema change for scope. Grant administration (C3 constraint) still needs its own migration under D-42 / D-34 (Group C).

## 15. What this ADR does not decide

Which capabilities the Seller holds (M2 rec. 1), which basis applies per destination (M2 rec. 2), who grants (D-42), how accounts are created (D-34), person identity (R5-SEC-01), real-data use (ADR #18), wallet semantics (D-41 / OPD-09 / D-36), delegation and assisted issuance (OPD-10), customer field access (GAP-04 / D-30 / D-45). See the decision package.
