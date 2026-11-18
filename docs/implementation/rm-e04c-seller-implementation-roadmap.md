# RM-E04C — Seller Access Implementation Roadmap

**PLAN ONLY.** Only the PRs marked DONE or IN THIS PR exist. DRAFT — NOT APPROVED. Production **NO-GO**. Baseline `main` @ `c0d7f4b`.

This roadmap supersedes the *ordering* of [`rm-e04a-execution-plan.md`](rm-e04a-execution-plan.md) (which stays as the record of Group A) and regroups future work into the five gates the owner asked for:

| Group | Gate | May start when |
|---|---|---|
| **A** | no new governance approval | now (code review only) |
| **B** | approved Seller capability (M2 rec. 1) **and** row-scope (M2 rec. 2) decisions | both records registered and `APPROVED_VERIFIED`; synthetic grants and data only |
| **C** | real account and grant infrastructure | D-42 and D-34 `APPROVED_VERIFIED` (+ R5-SEC-01 if person binding is required) |
| **D** | real data | ADR #18 `APPROVED_VERIFIED` + privacy review |
| **E** | production activation | separate written production authorization + go / no-go |

A design being complete is **not** a reason to start a blocked PR. Every PR: own branch, bounded diff, QA, PR into `main`, no self-merge (AGENTS.md).

Mapping to the RM-E04A plan: A1–A4 = A1–A4 · B1 = B1 · B2 = B2 · B3 = B3 · B4 = B4 · B5 = B5 · old C1 = C1 + C2 · old C2 = D1 · old D1 = E1.

---

## Group A — permitted now

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| A1 | Exhaustive scope assertion (F-02) | — | `internal/*/http` | SR-03 | NT-07 | revert | **DONE — PR #71** | code review |
| A2 | `no-store` + `Vary: Cookie` (F-07) | — | `internal/platform/httpx` | SR-12 | NT-17 | revert | **DONE — PR #72** | code review |
| A3 | Submission-context 404 parity (F-09) | — | `internal/payments/{command,http}` | SR-06 | NT-03 | revert | **DONE — PR #73**; Codex contract sign-off recorded as PENDING in its doc | code review + Codex |
| A4 | Operation scope registry + CI guard | A1 | registry YAML, tests | NT-22 | NT-22 | revert | **DONE — PR #74** | code review |
| A5 | RM-E04C architecture / decision / screen / API / traceability documents + governance consistency test | A1–A4 | `docs/**/rm-e04c-*`, `docs/governance/rm_e04c_test.go` | keeps proposals out of runtime; no status change | `go test ./docs/governance/`; YAML + `$ref` check | revert | **IN THIS PR** | docs review; Codex architecture review of the ADR |
| A6 | Scope-class plumbing: `ScopeOwn`; capability vocabulary carries a scope class; `commission.own.read` → `OWN`; `/me` guard helper. **No new capability, no bundle change** | A5 ADR **accepted by Codex architecture review** (engineering, not business) | `internal/authorization`, `internal/identity/api`, `internal/commission/http`, registry YAML | SR-02, SR-03, SR-13 | NT-25 (commission + all org handlers), existing NT-07 / NT-22, evaluator matrix | revert; no data | effective-access output identical except the scope label of `commission.own.read`; no OpenAPI diff | code + security review |
| A7 | Seller UI state model on synthetic fixtures (RM-E03 module): empty / PARTIAL / ABSENT count / FIELD_DENIED / 404 parity copy per screen-design §0 | — | `web/src/modules/seller/**` | no client-side scoping; DEV preview isolation unchanged | component tests; NT-23 | revert | fixtures only; no API call added | design review |

## Group B — needs approved M2 rec. 1 + M2 rec. 2 (synthetic only)

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| B1 | Accept the ADR (status change through the approval workflow) | M2 rec. 1 + 2 | `docs/` | — | governance tests | docs revert | ADR references the approval records | architecture review + the records' approvers |
| B2 | Register the **approved** own capabilities (e.g. `cases.own.read`, `billing.own.read`, `payments.own.read`); populate the `seller` bundle **in code** with exactly the approved set | A6, B1 | `internal/identity/api`, identity catalog migration, `api/openapi` enum, `cmd/crm-api/access.go` `capabilityFlags`, `web/src/app/navigation/destinations.ts` (RM-ACCESS-01 §11) | SR-02; nothing beyond the approved set | `TestL_ContractEnumEqualsRegistry`, `TestCapabilityFlagsCoverRegistry`, evaluator matrix, NT-18, NT-27 | revert + catalog down-migration (no grant data can exist: C3 constraint) | no ACTIVE grant possible in any non-test DB | security review |
| B3 | Wire S7 to `/commission/me/earnings` | B2 (bundle contains `commission.own.read`) | `web/src/modules/seller/pages/WalletPage.tsx` | SR-14; no settlement wording | NT-04, 12, 17, 18, 21, 23, 31; R7 browser test with synthetic grant | revert frontend | settlement «— تعریف نشده»; units not summed | code review |
| B4 | S1 own case reads (`listMyCases`, `getMyCase`, history, contact outcomes, attribution) with the approved basis | B2 | `internal/cases/{query,http}`, `internal/custody/query`, `api/openapi`, registry, `LeadsPage.tsx` | SR-01…SR-16 | S1 + S2 rows of the traceability matrix | revert; read-only, no migration | all listed NT pass on the R7 harness; org endpoints unchanged | security review |
| B5 | S4 own invoice reads + payments sub-resource with the approved basis | B4 | `internal/billing/{query,http}`, `internal/payments/http`, `api/openapi`, registry, `InvoicesPage.tsx` | SR-01…SR-16; no confirmed amount | S4 rows of the matrix | revert | NULL-owner invoices invisible; PARTIAL semantics | security + Finance review |

S2 list, S3, S5, S6: **no PR** until their workflow / data contracts exist (Codex / Product).

## Group C — needs real account and grant infrastructure

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| C1 | Grant administration: redefine `role_grant_shadow_only` to allow ACTIVE grants with a new approved basis; append-only grant / revoke history; grant and revoke commands with the authority chosen in D-42; audit (E-7) | D-42 `APPROVED_VERIFIED`; E-7 design; R5-SEC-01 if requester ≠ approver must hold per person | identity migrations, `internal/identity` commands, audit | SoD at grant time; idempotency; no grant from data | M2 §6 acceptance 1–8; NT-02, NT-32 | revoke via command; documented down path; history never deleted | per M2 §6 | **D-42 approvers + Security** |
| C2 | Account provisioning with the D-34 option | D-34 `APPROVED_VERIFIED` | identity credential write path | secret never logged; forced change / expiry per option | provisioning tests | disable command | real Seller can log in through the approved path only | **D-34 approvers** |
| C3 | Person registry / links (only if R5-SEC-01 selects A or B) | R5-SEC-01 `APPROVED_VERIFIED` | identity / organization | reversible reviewed links; no automatic merge | link tests | revert | no grant implied by a link | **R5-SEC-01 approvers** |

## Group D — needs real data

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| D1 | Staging rehearsal of B3–B5 on the ADR #18 data option; reconcile scoped counts against legacy 2.0.160; report UNKNOWN-holder and NULL-owner rates; measure seller ≠ issuer invoices (input to OPD-10 / M2 rec. 2 revisit) | ADR #18 `APPROVED_VERIFIED`; privacy review; C1 + C2 for real logins | staging configuration only | data handling per ADR #18 | full NT suite on staging | destroy data per ADR #18 | zero cross-Seller rows; rates reported, not hidden | **ADR #18 approvers** |

## Group E — production activation

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| E1 | Turn on Seller read flags for named accounts | A–D complete; written production authorization; go / no-go | none (flags and grants through approved paths) | legacy WordPress stays the authoritative writer; no write flag touched | production read smoke; NT-17, NT-20, NT-23 | flags off; revoke via C1 | — | **separate production authorization** |

## Order

A5 → A6 → A7 (parallel) → *(M2 rec. 1 + 2)* → B1 → B2 → B3 → B4 → B5 → *(D-42, D-34, R5-SEC-01 as required)* → C1 → C2 → C3 → *(ADR #18)* → D1 → *(production authorization)* → E1.

**Next implementable step:** A6 after Codex architecture review of the ADR (no business decision needed); A7 at any time.
