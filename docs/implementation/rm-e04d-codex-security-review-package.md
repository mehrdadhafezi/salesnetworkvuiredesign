# RM-E04D — Codex and Security Review Package (Seller authorization)

| | |
|---|---|
| Status | **REVIEW REQUESTED — PENDING.** No Codex or Security sign-off has been obtained. None is claimed. |
| Baseline | `main` @ `a053a9580c961c2d05b696f60c93f11f1b44ccea` (PR #76 merged) |
| Reviewers asked | Codex (engineering architecture, AGENTS.md *Agent Ownership*); Security owner |
| Runtime change | none (documentation, governance tests and a read-only synthetic evidence test) |
| Production | **NO-GO** |

There is no authorized review integration with Codex in this environment. This package is therefore the request; the sign-off has to be recorded by the reviewers themselves on the PR, or as an approval record where the R6 workflow applies. Until then every row below is **PENDING**.

Companion: Persian governance package [`../governance/proposals/rm-e04d-seller-governance-readiness.fa.md`](../governance/proposals/rm-e04d-seller-governance-readiness.fa.md).

---

## 1. Verified current implementation (HEAD `a053a95`)

| # | Fact | Evidence |
|---|---|---|
| V1 | `seller` bundle is empty | `internal/identity/api/api.go` (`{RoleSeller, "Seller", nil}`); guarded by `TestRME04CProposalsStayOutOfRuntime` |
| V2 | DB refuses non-SHADOW grants | `internal/identity/migrations/sql/00001_identity.sql:195`; now also guarded by `TestRME04DNoActiveGrantPathWithoutD42` |
| V3 | Evaluator scopes: `NONE`, `ORGANIZATION` only; every allow is `ORGANIZATION` | `internal/authorization/authorization.go:72-78`, `:163` |
| V4 | Every capability-guarded read asserts a supported scope (A1) | `internal/authorization/guard.go:182-221` |
| V5 | `no-store` + `Vary: Cookie` on every `/api` response (A2) | `httpx.CachePolicy` |
| V6 | Submission context: non-owner ≡ missing (A3) | `internal/payments/command/read.go` `ownsTarget` |
| V7 | 105 operations classified; 7 `OWN_READ` (A4) | `internal/authorization/http/testdata/operation_scope_registry.yaml` |
| V8 | Commission own read: SQL `holder_user_id = caller`; empty user → 403 `ROW_SCOPE_DENIED`; guard asks `ScopeOrganization` | `internal/commission/http/http.go:63-86,425-450`; `query/query.go:468,491` |
| V9 | `CurrentHolder` (used by the Slice 11 contact-outcome guard) admits a case only when PRESENT, **`legacy_holder_user_id` NOT NULL**, derivation ≠ `MULTIPLE`, parity ∉ {`CONFLICT`, `MULTIPLE_CURRENT`}; basis `LEGACY_HOLDER_COLUMNS` | `internal/custody/query/query.go:194-223` |
| V10 | Legacy holder source: lead cases ← `sn_leads.seller_id`; distribution-item cases ← `current_owner_user_id` | `internal/integration/legacy/shadow/sales.go:343-348` |
| V11 | Invoice seller / issuer imported from `sn_invoices.seller_id` / `issued_by_user_id`; **`original_seller_id` and the DOT conversion link (`sn_dot_invoice_links.flow_kind`) are NOT imported** | `shadow/billing.go:266-267`; no match for `dot_invoice_link` / `flow_kind` / `original_seller` under `internal/` (billing) |
| V12 | Legacy 2.0.160 Seller lead list: `WHERE l.seller_id = me` | `plugin/current/includes/class-sn-plugin.php:5282` |
| V13 | Legacy 2.0.160 Seller invoice list: `seller_id = me AND (NOT conversion_payment OR issued_by_user_id = me)` — "preventing another converter's payment links from leaking to the seller" | `class-sn-plugin.php:27443-27459` |
| V14 | Legacy conversion issuance sets `issued_by_user_id = actor` and keeps `original_seller_id = case seller` | `class-sn-dot-flow.php:6611`; converter list `:5952` |
| V15 | `AccessDecision.scope` OpenAPI enum is `[NONE, ORGANIZATION]` | `api/openapi/components/hr.yaml` |

### Correction of record (RM-E04A / RM-E04C terminology)

RM-E04A and RM-E04C used the label "current holder (M2 option 1)" for the predicate in V9. The M2 intake (`m2-authorization-decision-intake-fa.md` §4-2) defines option 1 as the **derived** holder (`derivation = 'DERIVED'`, `derived_user_id`), and the legacy column as option 2, "current assignment". V9 is therefore **M2 option 2 with a contradiction guard**, not option 1. The decision sheets now name predicates by column (P1 / P2 / P3 below), never by option number alone. The merged documents are not edited; this table is the correction of record.

### Correction of record (RM-E04C S4 recommendation)

RM-E04C recommended `INVOICE_SELLER_AND_ISSUER` as the initial S4 basis. V13–V14 show that legacy shows ordinary invoices by **seller** alone and requires the issuer only for conversion-payment invoices. Synthetic evidence (§2) has seller ≠ issuer on 4 of 4 invoices. The AND predicate would hide every ordinary invoice that was issued by someone else, so it is withdrawn as the recommended **final** basis. It remains only as the fail-closed interim (§3).

## 2. Synthetic evidence (R7-5 integrated dataset, read only)

Test: `internal/integration/legacy/shadow/rm_e04d_ownership_evidence_test.go`. Command:

```
CRM_TEST_DATABASE_URL=postgres://…/postgres CRM_RME04D_EVIDENCE=1 go test -v -run TestRME04DOwnershipPredicateEvidence ./internal/integration/legacy/shadow/
```

Measured at `a053a95` on a disposable PostgreSQL 16:

| Measure | Value | Reading |
|---|---|---|
| case assignments (all PRESENT) | 6 | |
| P1 derived holder known | 3 | |
| P2 legacy holder, guarded (= `CurrentHolder`) | 4 | |
| P2 legacy holder, unguarded | 5 | the guard removes 1 contradicting case |
| P3 P1 and P2 agree (`parity = MATCH`) | 2 | |
| P1 and P2 both known but different | 1 | a real reassignment / disagreement case |
| legacy holder unresolved | 1 | never "mine" for anyone |
| cases reassigned (≥ 2 distinct historical holders) | 1 | |
| custody events of kind `UNKNOWN` | 1 | |
| invoices | 4 | |
| seller NULL / issuer NULL | 0 / 0 | |
| seller = issuer | **0** | |
| seller ≠ issuer | **4** | the AND predicate admits no invoice |
| invoice → case link RESOLVED / not | 1 / 3 | "via linked case" admits at most 1 |
| via case: holder = seller / ≠ seller | 1 / 0 | |
| attribution facets | CREDIT_OWNER UNKNOWN 4; ORIGINAL_OWNER RESOLVED 4, NOT_RECOVERABLE 2; ORIGINAL_SELLER RESOLVED 3, NOT_RECOVERABLE 2, UNKNOWN 1 | facets must stay separate |
| commission entitlements / holder NULL | 19 / 0 | |
| entitlement holder ≠ invoice seller | 4 | an entitlement must not admit its invoice into S4 |

These are **fixture** properties: they show which categories exist and how each predicate treats them. They are **not** production rates. Production rates need ADR #18 (real-data authority). No real data was read.

## 3. Predicates proposed for review (none selected)

| Code | Predicate (SQL inside the scoped query) | Screens | Engineering view |
|---|---|---|---|
| P1 | `derivation = 'DERIVED' AND derived_user_id = :me AND presence = 'PRESENT'` | S1 | derived from events; differs from legacy (V12) |
| **P2** | `legacy_holder_user_id = :me AND presence = 'PRESENT' AND derivation <> 'MULTIPLE' AND parity NOT IN ('CONFLICT','MULTIPLE_CURRENT')` | S1 (+ case sub-resources, S2 attribution) | **recommended**: identical to `CurrentHolder` (V9), so read scope = contact-outcome write scope; matches legacy (V12) |
| P3 | P1 AND P2 AND `parity = 'MATCH'` | S1 | strictest; hides every case the two sources have not reconciled |
| I-a | `seller_user_id = :me` | S4 | matches legacy for ordinary invoices; **leaks** conversion-payment invoices issued by another converter (V13 boundary) |
| I-b | `issued_by_user_id = :me` | S4 | not the legacy rule |
| I-c | `seller_user_id = :me AND issued_by_user_id = :me` | S4 | fail-closed interim only; synthetic 0/4 |
| I-d | linked case passes P2 AND `case_link_state = 'RESOLVED'` | S4 | depends on link quality (synthetic 1/4 resolved) |
| **I-L** | `seller_user_id = :me AND (conversion_payment IS FALSE OR issued_by_user_id = :me)`; a NULL / unknown flow is treated as conversion (fail closed) | S4 | **recommended**: legacy-equivalent (V13). Needs the conversion-payment fact imported (V11 gap); until then it degrades to I-c by construction |
| C | `historical_entitlement.holder_user_id = :me` | S7 | exists (V8) |

Every predicate excludes NULL, UNKNOWN, CONFLICT, MULTIPLE and NOT_RECOVERABLE ownership: such rows are not shown and not counted (SR-05).

## 4. Architecture review checklist (`/me/...` proposal, RM-E04C ADR Option C)

| Item | What to review | Proposed answer | Status |
|---|---|---|---|
| Dedicated own-read endpoints | `/api/v1/me/cases*`, `/me/invoices*` instead of narrower scopes on organisation endpoints | yes; organisation handlers keep `ScopeOrganization` (V4) | PENDING |
| Capability-derived OWN scope | scope class lives in the code vocabulary, never in grant data | yes (ADR §4.1) | PENDING |
| Effective-access compatibility | `AccessDecision.scope` enum gains `OWN` (V15) → contract change, regenerated Go + TS | B2a, after B1 + M2 approval | PENDING |
| NT-07 | `TestNT07OwnReadSemantics` pins own reads at `ORGANIZATION` | rewritten in B2a: org handlers only `ORGANIZATION`, own handlers only `OWN` (NT-25) | PENDING |
| NT-22 | each new op gets an `OWN_READ` registry entry with its basis in the same PR | required in B4/B5 | PENDING |
| 404 parity | predicate inside the lookup query; identical problem body | as A3 (V6) | PENDING |
| SQL-enforced predicates | P2 / I-L as `WHERE` fragments owned by the module query package; same predicate for list, count, search, sub-resources | yes | PENDING |
| Multi-role | different capabilities → different endpoints; union per capability never mixes scopes | yes | PENDING |
| Caller identity | `Subject.UserID` proven, else 403 `ROW_SCOPE_DENIED` (V8 precedent) | yes | PENDING |
| Revocation | grants re-read per request; no scope cache | yes | PENDING |
| Cache | global `no-store`, `Vary: Cookie` (V5) | done | — |
| Field minimisation | Seller DTOs without `UserRef` / legacy keys; `FIELD_DENIED` omission | API proposal + A7 UI | PENDING |
| Data contract (new) | import `sn_dot_invoice_links.flow_kind` (conversion payment) and `sn_invoices.original_seller_id` into billing as facts with a link state | prerequisite of B5 (I-L) | PENDING (Codex data-contract owner) |

## 5. A3 contract sign-off evidence (outstanding)

| | |
|---|---|
| Change | `GET /api/v1/payments/invoices/{invoice_id}/submission` answers a caller who does not own the invoice (seller AND issuer = caller, NULL never owns) with the same generic 404 as a missing invoice. Before, it returned 200 with `eligible:false` + `ROW_SCOPE_DENIED` / `DEPENDENCY_UNKNOWN` / `STALE_STATE` |
| Merged | PR #73 (commit `c6dcb85`), in `main` |
| Required sign-off | Codex contract sign-off "required before merge" (`docs/remediation/rm-e04b-payment-submission-non-enumeration.md` §9). **Not recorded. The PR was merged without it** — a governance gap, recorded here and not hidden |
| Evidence for the reviewer | owner response byte-identical to base; 403 and `FEATURE_DISABLED` paths unchanged; the 404 was already in the contract; NT-03 T1–T14 HTTP test on disposable PostgreSQL; predicate-equivalence unit test (`internal/payments/command/ownership_test.go`) |
| Question for Codex | does any downstream workflow contract rely on a non-owner receiving `ROW_SCOPE_DENIED` from this read? |
| Remedy if rejected | revert of PR #73 is self-contained (no schema / data / flag) |
| Status | **PENDING — post-merge sign-off owed** |

## 6. Governance-test readiness (changed in this PR, tests only)

| Test | Before | After |
|---|---|---|
| `TestM2TrackerUnchanged` | any status change of D-42 / D-34 / R5-SEC-01 failed | baseline with nothing recorded, or a valid repository-mode R6 record with every R5 role |
| `TestR6BTrackerUnchanged` | ADR #18 / P-GOV frozen, and global counts 26/30/5 pinned (this also blocked registering a record) | same rule per record; counts replaced by the existing per-record R5-baseline test |
| `TestR6ZeroRecordedApprovals` | any file under `approvals/` failed | every file must be a valid repository-mode record, correctly named, for a tracked decision |
| new `TestRME04DNoActiveGrantPathWithoutD42` | — | `00001_identity.sql` must keep `role_grant_shadow_only`; no other migration may touch `identity.role_grant` in code (comments ignored, quoted identifiers normalised) until D-42 is `APPROVED_VERIFIED` |
| `validateApprovalRecord` (final review) | one person could sign under several roles; verifier / recorder compared without normalisation | one person = one approver (case and spacing normalised); verifier must differ from every approver and the recorder after normalisation |
| RM-E04C Group B gate (final review) | opened by a Security-only approval; any approved id could be named | approver floor Product + Sales + Security; each gate id must be an R5 inventory row titled as Seller record 1 (capability) / 2 (scope) |
| new `TestRME04DS4NeedsConversionPaymentContract` (+ `…ContractRules`, `…StaticRegistrationRules`, `…ExecutionEvidencePending`) | — | **fail-closed, four separate proofs.** They are recorded in `docs/governance/rm-e04d-s4-readiness.json`, which is empty today. (1) **contract**: an approved and verified conversion-payment contract decision (Finance + Security, R6 rules). (2) **schema**: the declared column is `NOT NULL`, with a `CHECK` listing an explicit unknown state. (3a) **static registration**, which is *necessary, not sufficient*: one named test per I-L case; its comment-stripped body is non-empty, uses the declared column in code and calls `t.Error`/`t.Fatal`. Empty bodies, comment-only mentions, failure calls only in comments and tests defined only in comments are refused. Source text **cannot** show that a test asserts ownership, ran or passed; a non-empty test with an unrelated assertion still passes 3a, and the fixtures document this limit. (3b) **execution evidence**: **PENDING by construction**, with no manifest field that can satisfy it (§7). S4 therefore cannot enter the gate today, even if (1), (2) and (3a) hold |
| new `TestRME04DNoClaimedSignOff` | — | no Codex / Security sign-off claimed in the RM-E04B/C/D documents; this package stays PENDING |
| `TestRME04DOwnershipPredicateEvidence` (final review) | invariants only (passed on an empty seed) | every measure pinned to the §2 value; an empty seed fails 18 pins |
| unchanged | — | `TestR6NoStatusOrOutcomeWithoutApprovalRecord`, `TestApprovalRecordsAppendOnly`, `TestR6TrackerAccountsForEveryR5Record` (41-record tripwire, updated deliberately by the registering PR), RM-E04C Group B gate |

Negative controls (synthetic fixtures in code, plus temporary real-file mutations, restored): a forged status without a record fails; a record without human verification, with `INCONCLUSIVE`, or missing a required role fails; a fixture record fails in repository mode; a wrongly named or untracked approval file fails; dropping the SHADOW-only constraint while D-42 is open fails; a legitimate, verified, role-complete D-42 record **passes** the whole suite.

## 7. S4 enforcement: fail-closed approval process (design; not implemented)

**What the repository can verify today.** The offline governance tests read committed files: tracker entries, approval records, migrations and test source. They **cannot** verify that a test asserts the I-L policy, that it ran, or that it passed. That is why proof 3b is PENDING and no data can change it. Opening it needs a reviewed change to `s4ExecutionEvidenceProblems` together with the mechanism below.

| Step | Requirement | Verified by | Exists today? |
|---|---|---|---|
| E1 Executable tests | The six I-L tests (§6, proof 3a) run against a disposable PostgreSQL with synthetic fixtures. The fixtures cover: seller = me, issuer ≠ me, conversion payment → hidden and not counted; own conversion → visible; non-conversion mismatch → visible; conversion state `UNKNOWN` → hidden; NULL seller or fact → hidden; hidden ≡ missing (404 body). Each asserts rows **and** counts | Code review (Codex) | No (S4 not implemented) |
| E2 Mutation evidence | Each I-L test is shown to fail when the predicate is broken: drop the conversion clause, treat `UNKNOWN` as non-conversion, or remove the seller clause | Recorded mutation run in the implementing PR | No |
| E3 CI execution evidence | A required CI job runs exactly the six tests (`go test -run '^(…)$' -json`) on the implementing commit, fails if any is skipped or missing, and publishes the `test2json` output as an artefact bound to the commit SHA and run ID | CI configuration + artefact | **No such job or artefact** |
| E4 Independent Security review | A Security owner reviews E1–E3 and records an R6 `DECISION` (decision ID assigned by the inventory owner). `evidence_source.reference` = the CI run URL; `immutable_ref` = the commit SHA. Signed by Security and Finance, then T5-verified by a person who is neither approver nor recorder, and who confirms the run independently | R6 record + human T5 | No |
| E5 Governance-test change | Only after E1–E4 exist: replace the constant in `s4ExecutionEvidenceProblems` with a check that the manifest names that E4 record, `APPROVED_VERIFIED` under the R6 rules, and that its `immutable_ref` is the commit implementing the tests. Reviewed by Codex and Security | PR review | No |

Until E1–E5 exist, S4 readiness stays **BLOCKED**. Any change that lets S4 into the gate without them is a defect. `TestRME04DS4ExecutionEvidencePending` keeps that visible.

