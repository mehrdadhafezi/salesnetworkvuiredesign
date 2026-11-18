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
| A5 | RM-E04C architecture / decision / screen / API / traceability documents + governance consistency test | A1–A4 | `docs/**/rm-e04c-*`, `docs/governance/rm_e04c_test.go`, `docs/governance/rm-e04c-group-b-gate.json` (closed) | keeps proposals out of runtime while the Group B gate is closed; a decision status may move only through a valid R6 approval record; no status change in this PR | `go test ./docs/governance/`; YAML + `$ref` check | revert | **IN THIS PR** | docs review; Codex architecture review of the ADR |
| ~~A6~~ | **Withdrawn from Group A** (PR #75 review remediation). The scope-class change alters observable authorization behaviour and the public contract; it is now **B2a** in Group B. See §"B2a — why it is not Group A" | — | — | — | — | — | — | — |
| A7 | Seller UI state model on synthetic fixtures (RM-E03 module): empty / PARTIAL / ABSENT count / FIELD_DENIED / 404 parity copy per screen-design §0 | — | `web/src/modules/seller/**` | no client-side scoping; DEV preview isolation unchanged | component tests; NT-23 | revert | fixtures only; no API call added | design review |

## Group B — needs approved M2 rec. 1 + M2 rec. 2 (synthetic only)

| PR | Objective / scope | Deps | Files | Security requirements | Tests | Rollback | Acceptance | Approvals |
|---|---|---|---|---|---|---|---|---|
| B1 | Accept the ADR (status change through the approval workflow) | M2 rec. 1 + 2 | `docs/` | — | governance tests | docs revert | ADR references the approval records | architecture review + the records' approvers |
| B2a | Scope-class plumbing (formerly A6): `ScopeOwn`; capability vocabulary carries a scope class; `commission.own.read` and the other `OWN_READ` capabilities move from `ORGANIZATION` to `OWN`; `/me` guard helper. **Observable change** — see the B2a section below | A1, A4 (done); **B1 (ADR accepted)**; M2 rec. 1 + 2 `APPROVED_VERIFIED` (RM-E04A plan B2 entry condition) | `internal/authorization`, `internal/identity/api`, `internal/{commission,fulfillment,payments,organization}/http` own reads, `api/openapi/components/hr.yaml` (`AccessDecision.scope` enum), regenerated `internal/platform/httpx/oapi` and `web/src/generated/api/schema.ts`, registry YAML, NT-07 | SR-02, SR-03, SR-13 | compatibility suite in the B2a section; NT-25 | revert code + contract together; no data | every listed compatibility test green; enum change reviewed as a contract change | **security review + Codex contract sign-off** |
| B2 | Register the **approved** own capabilities (e.g. `cases.own.read`, `billing.own.read`, `payments.own.read`); populate the `seller` bundle **in code** with exactly the approved set | B2a, B1 | `internal/identity/api`, identity catalog migration, `api/openapi` enum, `cmd/crm-api/access.go` `capabilityFlags`, `web/src/app/navigation/destinations.ts` (RM-ACCESS-01 §11) | SR-02; nothing beyond the approved set | `TestL_ContractEnumEqualsRegistry`, `TestCapabilityFlagsCoverRegistry`, evaluator matrix, NT-18, NT-27 | revert + catalog down-migration (no grant data can exist: C3 constraint) | no ACTIVE grant possible in any non-test DB | security review |
| B3 | Wire S7 to `/commission/me/earnings` | B2 (bundle contains `commission.own.read`) | `web/src/modules/seller/pages/WalletPage.tsx` | SR-14; no settlement wording | NT-04, 12, 17, 18, 21, 23, 31; R7 browser test with synthetic grant | revert frontend | settlement «— تعریف نشده»; units not summed | code review |
| B4 | S1 own case reads (`listMyCases`, `getMyCase`, history, contact outcomes, attribution) with the approved basis | B2 | `internal/cases/{query,http}`, `internal/custody/query`, `api/openapi`, registry, `LeadsPage.tsx` | SR-01…SR-16 | S1 + S2 rows of the traceability matrix | revert; read-only, no migration | all listed NT pass on the R7 harness; org endpoints unchanged | security review |
| B5 | S4 own invoice reads + payments sub-resource with the approved basis | B4 | `internal/billing/{query,http}`, `internal/payments/http`, `api/openapi`, registry, `InvoicesPage.tsx` | SR-01…SR-16; no confirmed amount | S4 rows of the matrix | revert | NULL-owner invoices invisible; PARTIAL semantics | security + Finance review |

S2 list, S3, S5, S6: **no PR** until their workflow / data contracts exist (Codex / Product).

**RM-E04D update (baseline `a053a95`):** B5 gains a prerequisite data-contract PR (import of the legacy conversion-payment link `sn_dot_invoice_links.flow_kind` and `sn_invoices.original_seller_id` into billing as facts with a link state; Codex + Finance), because the recommended S4 basis is now the legacy-equivalent I-L. The grouping into the fewest reviewable PRs (B-PR1 … B-PR5) is in [`../governance/proposals/rm-e04d-seller-governance-readiness.fa.md`](../governance/proposals/rm-e04d-seller-governance-readiness.fa.md) §12.

### B2a — why it is not Group A

The earlier draft of this roadmap called the scope-class change "behaviour-neutral" with "no OpenAPI diff". That was wrong. Verified at `c0d7f4b`:

**Who observes `Decision.Scope` today**

| Consumer | Where | What changes if `commission.own.read` returns `OWN` |
|---|---|---|
| Effective-access API | `internal/authorization/http/http.go:117` serialises `Scope` into `AccessDecision.scope` for `GET /api/v1/authorization/effective-access` | the response value changes for every holder of the capability |
| Public contract | `api/openapi/components/hr.yaml` `AccessDecision.scope` is `enum: [NONE, ORGANIZATION]` | `OWN` is outside the enum → a contract change (OpenAPI, generated `oapi.AccessDecisionScope`, `web/src/generated/api/schema.ts` type `"NONE" \| "ORGANIZATION"`) |
| Read guard | `guard.go:182-221` (`CheckRead` / `RequireScope`) | a handler still calling `CheckRead(…, ScopeOrganization)` would **deny** with `ROW_SCOPE_DENIED` — `commission/http/http.go:67` uses one `guardOp` for all commission reads, so the own read must switch in the same change or it breaks |
| Other own reads | registry `OWN_READ` entries for `fulfillment.work.read`, `org.hr_request.read`, `payments.submission.read` all check `ScopeOrganization` | moving only commission makes the vocabulary inconsistent; moving all of them changes four modules |
| Tests that pin the value | `scope_nt07_test.go` (`TestNT07OwnReadSemantics`: own read allowed with `ORGANIZATION`; `OWN_TEST` / `SELLER_OWN` denied), `authorization_test.go`, `scope_test.go`, `http_test.go`, `operation_registry_test.go`, `cmd/crm-api/access_pg_test.go`, `web/src/app/navigation/navigation.test.tsx` (decision fixtures) | each must be updated deliberately, not incidentally |
| Web navigation | `web/src/app/navigation` decides on `allowed`, not `scope` | no behavioural change expected; type check affected by the enum |

**Compatibility tests B2a must add or update**

1. Contract: `AccessDecision.scope` enum includes `OWN`; `TestL_ContractEnumEqualsRegistry`-style check that every capability's scope class appears in the enum; oapi regeneration has no other diff.
2. Evaluator matrix: every capability returns its declared class on allow and `NONE` on deny; no grant data can change the class (`DevGrantBasis` and stored grants).
3. NT-07 rewritten: organisation handlers accept only `ORGANIZATION`; own handlers accept only `OWN`; every other value denied (NT-25).
4. `TestNT07OwnReadSemantics`: commission own read still answers the same rows for the same caller, now with `OWN`.
5. Effective-access golden test: only the `scope` field of the moved capabilities changes; `allowed`, `outcome`, `reason_code`, `matched_roles` identical.
6. Web: `navigation.test.tsx` and generated-type compile with the new enum; no destination visibility change.
7. Registry (NT-22): `OWN_READ` entries declare the `OWN` class; flags unchanged.

**Gate decision (governed by the RM-E04A execution plan and AGENTS.md)**

| Question | Answer |
|---|---|
| Allowed before M2 rec. 1 + 2 are `APPROVED_VERIFIED`? | **No.** RM-E04A plan B2 ("Teach `authorization.Evaluate` the approved own-scope(s)") lists M2 rec. 1 + rec. 2 approved as a dependency, and the Group B entry condition applies to every B PR. |
| Must the proposed ADR be accepted first? | **Yes.** RM-E04A plan B2 depends on B1 (scope-model ADR), and AGENTS.md makes engineering architecture Codex-owned; this ADR is only PROPOSED. |
| Group A or Group B? | **Group B (B2a).** Group A is defined as "changes no authority" and needs "no business-policy selection"; B2a changes a served authorization value and a public contract, and implements the scope vocabulary that M2 rec. 2 governs. AGENTS.md also requires current permissions to be preserved until explicitly approved. |
| Does deferring weaken anything? | No. A1 (exhaustive scope assertion) already guarantees an unknown scope is denied; nothing depends on `OWN` existing before Group B. |

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

A5 → A7 → *(M2 rec. 1 + 2 `APPROVED_VERIFIED`)* → B1 → B2a → B2 → B3 → B4 → B5 → *(D-42, D-34, R5-SEC-01 as required)* → C1 → C2 → C3 → *(ADR #18)* → D1 → *(production authorization)* → E1.

**Next implementable step:** A7 (UI state model on synthetic fixtures; design review only). Codex architecture review of the proposed ADR may proceed in parallel, but its acceptance (B1) is recorded only with the M2 records. Nothing in Group B starts before then; `docs/governance/rm_e04c_test.go` keeps Group B runtime artefacts refused until the Group B gate (`docs/governance/rm-e04c-group-b-gate.json`) is opened by verified approvals.
