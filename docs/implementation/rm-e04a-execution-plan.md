# RM-E04A — Future Engineering PR Plan

**PLAN ONLY. None of these PRs has been started.** DRAFT — NOT APPROVED. Production **NO-GO**.

Cross-references: findings in [`rm-e04a-seller-scoped-read-contract.md`](rm-e04a-seller-scoped-read-contract.md); inventory in [`../security/rm-e04a-seller-leak-surface-audit.md`](../security/rm-e04a-seller-leak-surface-audit.md); contracts in [`../security/rm-e04a-seller-row-scope.md`](../security/rm-e04a-seller-row-scope.md); tests in [`../security/rm-e04a-seller-negative-test-plan.md`](../security/rm-e04a-seller-negative-test-plan.md); decisions in [`../governance/proposals/rm-e04a-seller-decisions.md`](../governance/proposals/rm-e04a-seller-decisions.md).

Groups:
- **A** — safe security hardening; needs **no** business-policy selection; changes no authority.
- **B** — implementation of an **approved** scope policy, in an isolated test environment with synthetic grants and data.
- **C** — integration of real users and real data.
- **D** — production activation; requires its own authorization.

Every PR follows AGENTS.md: own branch, bounded change, QA, PR into `main`, no self-merge.

---

## Group A — hardening (can start when the owner asks; no decision needed)

### A1 · RM-E04B — Exhaustive scope assertion (F-02)

| | |
|---|---|
| Purpose | Make every data-returning handler refuse an allowed decision whose `Decision.Scope` it does not explicitly implement, **before** any narrower scope exists |
| Dependencies | none |
| Code areas | A shared helper in `internal/authorization/http` (e.g. "require ORGANIZATION scope or deny `ROW_SCOPE_DENIED`"), applied in `internal/{cases,custody,billing,payments,customers,organization,identity}/http`, `internal/integration/{providerops,writeback}/http`, `internal/finance/http/review.go`, `internal/biawin/http/activation.go`, `internal/fulfillment/http/work.go`. The 51 GET handlers marked MISSING in the leak audit. Commission `own=true` path unchanged |
| Tests | NT-07 per module (table-driven over all 92 GET operations; guard stub returning an unknown scope; recording DB runner proves no module query ran); existing package tests; `go test ./...` with PostgreSQL; `-race` on touched packages; architecture lint |
| Rollback boundary | Revert the PR; no schema, data, flag, grant or contract change |
| Acceptance | 0 GET operations with "Scope assertion: MISSING"; every existing test unchanged and green; effective-access output unchanged; no OpenAPI diff |
| Human approval | Code review only (security-sensitive) |

### A2 · Cache headers (F-07)

| | |
|---|---|
| Purpose | `Cache-Control: no-store` on every `/api/*` response; `Vary: Cookie` on authenticated responses |
| Dependencies | none (may land with or after A1) |
| Code areas | One middleware in `internal/platform/httpx` or the `apiserver` handler chain (same place as correlation / metrics middleware); no per-handler edits. Optionally mirror in `deploy/staging/nginx/nginx.conf` with `add_header … always` |
| Tests | NT-17: handler test over every route (200, 4xx, 5xx); staging compose test (`deploy/staging/stage_test.go` pattern) asserting the header survives nginx |
| Rollback boundary | Revert; headers only |
| Acceptance | Header present on all 105 operations incl. existing `no-store` paths (no duplicate / conflicting values) |
| Human approval | Code review |

### A3 · Submission-context existence parity (F-09)

| | |
|---|---|
| Purpose | `GET /api/v1/payments/invoices/{invoice_id}/submission` answers a non-owner exactly like a missing invoice |
| Dependencies | none; coordinate with Codex because it changes an observable response for a non-owner (the 404 is already in the contract) |
| Code areas | `internal/payments/command/read.go` (`Context`), `internal/payments/http/write.go` (`GetPaymentSubmission`) |
| Tests | NT-03 rows for the submission context; existing payments tests (owner path unchanged) |
| Rollback boundary | Revert; no data |
| Acceptance | non-owner / NULL owner / missing → identical 404 body; owner response unchanged |
| Human approval | Code review + Codex sign-off on the contract behaviour |

### A4 · Operation inventory guard (NT-22)

| | |
|---|---|
| Purpose | A static test that fails when an OpenAPI operation has no declared scope class or no NT-07 case |
| Dependencies | A1 |
| Code areas | A test next to `cmd/crm-api/access_test.go` or `internal/authorization/http`, plus a small committed registry (operationId → `ORGANIZATION` / `OWN:<basis>` / `NOT_APPLICABLE`) |
| Tests | the guard itself; negative fixture proving it fails on an unclassified operation |
| Rollback boundary | Revert |
| Acceptance | 105 / 105 operations classified; adding an operation without a class fails CI |
| Human approval | Code review |

---

## Group B — approved policy, isolated test environment

**Entry condition for every B PR:** M2 proposed record 1 and record 2 are registered by the inventory owner and reach `APPROVED_VERIFIED` (r6 approval evidence standard). Synthetic data and synthetic test grants only; no production grant command (that is Group C).

### B1 · Scope-model ADR (documentation)

| | |
|---|---|
| Purpose | Record Option A vs B (RM-E04A-Q1), the scope vocabulary and where scope is stored (bundle vs grant) |
| Dependencies | may be drafted now; accepted only with M2 rec. 1 / rec. 2 |
| Code areas | `docs/` only |
| Tests | governance checks |
| Rollback boundary | docs revert |
| Acceptance | Codex-owned ADR referencing this plan; no status change outside the approval workflow |
| Human approval | **Yes** (architecture review) |

### B2 · Evaluator scope and capability registry

| | |
|---|---|
| Purpose | Teach `authorization.Evaluate` the approved own-scope(s); register any new capability |
| Dependencies | A1 (mandatory), A4, B1, M2 rec. 1 + rec. 2 approved |
| Code areas | `internal/authorization/authorization.go`; `internal/identity/api` vocabulary; identity catalog migration; OpenAPI enum; `cmd/crm-api/access.go` `capabilityFlags`; `web/src/app/navigation/destinations.ts` (RM-ACCESS-01 §11 checklist) |
| Tests | evaluator matrix incl. NT-18; `TestL_ContractEnumEqualsRegistry`, `TestCapabilityFlagsCoverRegistry`, `TestVocabularyShape` updated only as the approval says |
| Rollback boundary | Revert code + down-migration of the catalog row (no grant data exists) |
| Acceptance | no bundle populated in production data; synthetic grant only in tests |
| Human approval | **Yes** (security review; matches approved records) |

### B3 · S7 Wallet wiring (server scope already exists)

| | |
|---|---|
| Purpose | Wire `pages/WalletPage.tsx` to `GET /api/v1/commission/me/earnings` |
| Dependencies | A2; M2 rec. 1 includes `commission.own.read` |
| Code areas | `web/src/modules/seller/pages/WalletPage.tsx`, `web/src/shared/api/client.ts` usage, `roles.ts` destination |
| Tests | NT-04, NT-12, NT-17, NT-21, NT-23; R7 browser test with synthetic grant |
| Rollback boundary | Revert frontend; preview fixtures remain |
| Acceptance | settlement stays «— تعریف نشده»; no write; DEV preview isolation unchanged |
| Human approval | Code review |

### B4 · S1 scoped case reads

| | |
|---|---|
| Purpose | Own-scope case list / detail / history / attribution / contact-outcome read per approved predicate |
| Dependencies | A1–A4, B2 |
| Code areas | `internal/cases/{query,http}`, `internal/custody/query` (holder predicate), OpenAPI, generated code, `LeadsPage.tsx` |
| Tests | NT-01, 03, 05, 06, 08, 09, 10, 14, 16, 18, 19, 21, 24 |
| Rollback boundary | Revert; read-only queries, no migration expected |
| Acceptance | all listed NT pass on the R7 harness; organisation endpoints unchanged |
| Human approval | Security review |

### B5 · S4 scoped invoice reads

| | |
|---|---|
| Purpose | Own-scope invoice list / detail / stages / totals / payments per the approved S4 predicate (RM-E04A-Q2) |
| Dependencies | B4; GAP-03 answer for confirmed amounts (else the field stays absent) |
| Code areas | `internal/billing/{query,http}`, `internal/payments/http`, OpenAPI, `InvoicesPage.tsx` |
| Tests | NT-01, 03, 05, 06, 08, 11, 12, 15, 17, 21 |
| Rollback boundary | Revert |
| Acceptance | `I_N` invisible; PARTIAL semantics; no money fact invented |
| Human approval | Security + Finance review |

S2, S3, S5, S6 have **no B PR** until their product / workflow contracts exist (row-scope doc).

---

## Group C — real users and real data

### C1 · Grant administration and accounts

| | |
|---|---|
| Purpose | Production grant / revoke commands with append-only history and audit (M2 §6 "smallest PR"); account provisioning |
| Dependencies | D-42, D-34 `APPROVED_VERIFIED`; E-7 design; R5-SEC-01 if person binding is required |
| Code areas | identity migrations (`role_grant_shadow_only` redefinition), identity commands, audit |
| Tests | M2 §6 acceptance 1–8; NT-02 |
| Rollback boundary | Revoke grants via the command; migration has a documented down path; no deletion of history |
| Acceptance | per M2 §6 |
| Human approval | **Yes** (D-42, D-34, Security) |

### C2 · Real-data staging rehearsal

| | |
|---|---|
| Purpose | Run B3–B5 against a real-data extract under the ADR #18 option chosen |
| Dependencies | ADR #18 `APPROVED_VERIFIED`; privacy review |
| Code areas | none expected (configuration of staging) |
| Tests | full NT suite on staging; reconciliation of scoped counts against legacy |
| Rollback boundary | destroy staging data per ADR #18 |
| Acceptance | zero cross-Seller rows; UNKNOWN rates reported, not hidden |
| Human approval | **Yes** |

---

## Group D — production activation

### D1 · Seller read activation

| | |
|---|---|
| Purpose | Turn on the Seller read flags for named accounts |
| Dependencies | A, B, C complete; separate written production authorization; go / no-go (`docs/cutover/go-no-go-report.md` process) |
| Code areas | none (flags / grants through approved paths) |
| Tests | production smoke limited to reads; NT-17, NT-20, NT-23 |
| Rollback boundary | flags off; grants revoked via C1 command |
| Acceptance | legacy WordPress remains the authoritative writer; no write flag touched |
| Human approval | **Yes — separate production authorization** |

## Order

A1 → A2 → A3 → A4 → (decisions) → B1 → B2 → B3 → B4 → B5 → (decisions) → C1 → C2 → (authorization) → D1.
