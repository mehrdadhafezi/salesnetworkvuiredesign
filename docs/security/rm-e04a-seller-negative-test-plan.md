# RM-E04A — Seller Negative Security Test Plan

**FUTURE TESTS — NONE OF THESE HAS BEEN IMPLEMENTED OR RUN.** No result below is a passing runtime test. DRAFT — NOT APPROVED. Production **NO-GO**.

Cross-references: findings / SR rules in [`../implementation/rm-e04a-seller-scoped-read-contract.md`](../implementation/rm-e04a-seller-scoped-read-contract.md); per-operation inventory in [`rm-e04a-seller-leak-surface-audit.md`](rm-e04a-seller-leak-surface-audit.md); screen contracts in [`rm-e04a-seller-row-scope.md`](rm-e04a-seller-row-scope.md); delivery in [`../implementation/rm-e04a-execution-plan.md`](../implementation/rm-e04a-execution-plan.md).

## 0. Shared fixture and conventions

**Harness.** R7 full-stack harness (real browser → SPA → `crm-api` → disposable PostgreSQL), plus Go handler / PostgreSQL tests as in `cmd/crm-api/access_pg_test.go`. **Synthetic data only** (ADR #18). Grants are synthetic test grants (dev/test stub or test-only ACTIVE grant rows), **never** a production grant command, because D-42 / D-34 are open. Tests in Group A of the execution plan need no Seller scope at all; tests that exercise a Seller scope run only in Group B, after the scope policy exists.

**Fixture F0** (all scenarios unless stated):

| Object | Facts |
|---|---|
| Users | `X` (Seller A), `Z` (Seller B), `M` (holds an organisation-scope read through another synthetic role), `N` (no grant) |
| Cases | `C_X1, C_X2` custody holder X (PRESENT, parity MATCH); `C_Z1` holder Z; `C_U` holder unknown (`legacy_holder_user_id` NULL → `NO_RESOLVED_LEGACY_HOLDER`); `C_K` CONFLICT parity; `C_M` derivation MULTIPLE; `C_A` presence ABSENT_IN_LATEST |
| Invoices | `I_X` seller = issuer = X; `I_Z` seller = issuer = Z; `I_XZ` seller X, issuer Z; `I_N` seller NULL; `I_UK` seller X with a stage `state_mapping` UNKNOWN |
| Customers | `K_X` linked to `C_X1` (has a contact point with phone `0900000xxxx`); `K_Z` linked to `C_Z1` |
| Commission | entitlements for X and Z in run `R1` |
| Flags | module flags on unless stated; all `_writeback` flags off; command rollout modes `Disabled` unless stated |

**Notation.** `own-read` = the Seller-scoped read under whichever option (A/B) is approved. Expected bodies refer to the existing envelope (`outcome`, `reasons`, `items`, `page`) and the existing problem body (`authzhttp.NotFound` / `authzhttp.Denied`). "SQL assertion" means a query the test runs against the database after the request.

Original T-numbers from the parent document are kept as aliases.

---

### NT-01 — List isolation (alias T1)
| | |
|---|---|
| Invariant | SR-01, SR-05: a list returns only rows inside the caller's scope |
| Fixture | F0 |
| Actor / role | X with own-read on cases and invoices |
| Request | `GET` own-read cases list; own-read invoices list; iterate every page with `limit=1` and with the maximum limit; every enum filter value |
| Status | 200 |
| Outcome / reason | `SUCCESS` (or `PARTIAL` with reason for incomplete facts) |
| Body | Cases = `{C_X1, C_X2}` exactly; invoices = the set admitted by the approved S4 predicate (e.g. `{I_X, I_UK}` under predicate (c)); never `C_Z1, C_U, C_K, C_M, I_Z, I_N` |
| SQL | Response IDs ⊆ `SELECT case_id FROM custody.assignment WHERE legacy_holder_user_id = X AND presence='PRESENT' AND derivation <> 'MULTIPLE' AND parity NOT IN ('CONFLICT','MULTIPLE_CURRENT')` (if predicate 1 is approved) |
| Source / decision | M2 §5, N1; M2 rec. 2 |

### NT-02 — Revoked grant and revoked session
| | |
|---|---|
| Invariant | Authority is evaluated per request; revocation takes effect on the next request |
| Fixture | F0; X holds an ACTIVE synthetic grant |
| Actor / role | X |
| Request | (a) own-read list → 200; set grant status `REVOKED` (test DB) → repeat. (b) `POST /api/v1/auth/logout-all` → repeat with the old cookie |
| Status | (a) 403; (b) 403 (anonymous) |
| Outcome / reason | (a) `UNAUTHORIZED / ACTION_DENIED`; (b) anonymous: `UNAUTHORIZED`; effective-access `authenticated:false` |
| Body | No `items`; problem body identical to the never-granted case |
| SQL | Grant row is append-only history (revocation recorded, not deleted) — only once the E-7 design exists; until then assert no `identity.role_grant` row was deleted |
| Source / decision | M2 §6 test 7; RM-ACCESS-01 test H; `authorization/session.go:42` (grants loaded per request); D-42 (who revokes), E-7 |

### NT-03 — Non-enumeration on detail and sub-resources (alias T3; covers F-09)
| | |
|---|---|
| Invariant | SR-06: out-of-scope = missing |
| Fixture | F0 plus a random UUID `R` |
| Actor / role | X |
| Request | `GET` detail, `/history`, `/attribution`, `/contact-outcomes` for `C_Z1` and `R`; invoice detail, `/items`, `/stages`, `/totals`, `/payments`, `/receipts` for `I_Z` and `R`; `GET /api/v1/payments/invoices/{I_Z}/submission` and `{R}` |
| Status | 404 for every pair |
| Outcome / reason | Problem body from `authzhttp.NotFound` |
| Body | Byte-identical except `correlation_id`; no ID echo, no reason code that implies existence (today the submission context returns 200 + `ROW_SCOPE_DENIED` for `I_Z` — **expected to fail until fixed**, F-09) |
| SQL | — |
| Source / decision | M2 N3, N4; leak audit row 100 |

### NT-04 — No proven user identity (alias T6)
| | |
|---|---|
| Invariant | SR-04: no `Subject.UserID` → no rows, never organisation fallback |
| Fixture | F0; a session whose principal has no resolved identity user (unit level: `Subject{PrincipalID:"p", UserID:""}`) |
| Actor / role | principal with an own-read grant |
| Request | own-read list; `GET /api/v1/commission/me/earnings`; `GET /api/v1/payments/batches/{id}` |
| Status | 403 |
| Outcome / reason | `UNAUTHORIZED / ROW_SCOPE_DENIED` |
| Body | No `items` |
| SQL | — |
| Source / decision | `commission/http/http.go:441-447`; `payments/command/read.go:170` |

### NT-05 — Search stays inside scope (alias T4)
| | |
|---|---|
| Invariant | SR-08: search, including phone, never crosses the predicate |
| Fixture | F0 |
| Actor / role | X |
| Request | Every search parameter that the approved contract introduces (case code, invoice code, customer name, phone) with values belonging to `C_Z1`, `I_Z`, `K_Z` |
| Status | 200 |
| Outcome / reason | `SUCCESS` |
| Body | `items: []`, `has_more:false`, any count = 0 **for X's scope** (a count of 0 here is correct, because X's scoped result is empty, not unknown) |
| SQL | — |
| Source / decision | M2 N5; GAP-04 (phone / name search needs the field policy). **N/A today** (no search parameter exists) |

### NT-06 — Pagination and cursor tampering (alias T5)
| | |
|---|---|
| Invariant | A cursor cannot move a caller outside scope |
| Fixture | F0 with ≥ 25 rows per seller |
| Actor / role | X; cursors captured as Z and as M (organisation list) |
| Request | own-read list with Z's cursor, M's cursor, a hand-crafted cursor pointing between Z's rows, a malformed cursor |
| Status | 200 (foreign or crafted but well-formed) / 400 (malformed) |
| Outcome / reason | `SUCCESS` / problem `400` |
| Body | Only X's rows; no row of Z appears on any page |
| SQL | — |
| Source / decision | SR-01; RM-E03 §11 |

### NT-07 — Scope assertion on every organisation handler (alias T7; F-02 regression)
| | |
|---|---|
| Invariant | SR-03: an allowed decision with an unknown or narrower scope is refused by every data-returning handler |
| Fixture | Handler-level: a guard stub returning `Decision{Allowed:true, Scope:"OWN_TEST"}` (test-only value) |
| Actor / role | any |
| Request | **Each of the 51 GET operations marked "Scope assertion: MISSING" in the leak audit**, plus the 37 already PRESENT (regression) |
| Status | 403 |
| Outcome / reason | `UNAUTHORIZED / ROW_SCOPE_DENIED` |
| Body | No `items`, no aggregate fields |
| SQL | No query against module tables executed (use a recording `db.Runner`) |
| Source / decision | F-02; no business decision needed (Group A) |

### NT-08 — Aggregates and totals (alias T2)
| | |
|---|---|
| Invariant | SR-07: counts and sums are computed after the predicate; organisation aggregates never reach a Seller |
| Fixture | F0 |
| Actor / role | X |
| Request | Every count / KPI / totals field of the own-read contract; `GET /api/v1/sales/coverage-report`, `/sales/custody-parity`, `/billing/reconciliation-report`, `/customer-identity/resolution-report`, `/payments/evidence`, `/reports/*` |
| Status | own-read: 200; organisation aggregates: 403 |
| Outcome / reason | own-read `SUCCESS`/`PARTIAL`; aggregates `UNAUTHORIZED / ACTION_DENIED` (not in the Seller bundle) |
| Body | Counts equal `|NT-01 set|`; `C_U`, `C_K`, `C_M` not counted; sums are per unit and marked `PARTIAL` when `I_UK` is included |
| SQL | Count equals the SQL count under the approved predicate |
| Source / decision | M2 N2, N6; M2 §5 (`sales.report.read` not in the bundle) |

### NT-09 — Unknown and conflicting custody
| | |
|---|---|
| Invariant | SR-05: unknown, conflicting, multiple or absent ownership never admits a row |
| Fixture | F0 (`C_U`, `C_K`, `C_M`, `C_A`); additionally a case whose legacy holder is X but derived holder is Z (parity `CONFLICT`) |
| Actor / role | X and Z |
| Request | own-read list and detail for each |
| Status | list 200; detail 404 |
| Outcome / reason | `SUCCESS` (list); not found (detail) |
| Body | None of the five cases appears for X **or** Z; no count includes them |
| SQL | `custody.assignment` rows unchanged (read-only) |
| Source / decision | `custody/query/query.go:212-219`; AGENTS.md (unknown stays UNKNOWN / CONFLICT); M2 rec. 2 |

### NT-10 — Scope injection
| | |
|---|---|
| Invariant | SR-02: scope comes only from the evaluator |
| Fixture | F0 |
| Actor / role | X |
| Request | own-read and organisation operations with `?scope=ORGANIZATION`, `?scope=ALL`, `?seller_id=Z`, `?user_id=Z`, header `X-Role: sales_manager`, header `X-Scope: ORGANIZATION`, a forged `role` in a cookie or body; `GET /api/v1/reports/cases?scope=OWN` |
| Status | own-read: 200 (parameter ignored or 400 if unknown params are rejected); organisation: 403; reports with non-ORG scope: 400 / refused (`reporting/query/spec.go:165`) |
| Outcome / reason | unchanged from the uninjected request |
| Body | Byte-identical items to the uninjected request |
| SQL | — |
| Source / decision | RM-E02 §6 (client role is never authority); RM-ACCESS-01 |

### NT-11 — Invoice isolation and NULL ownership (aliases T13)
| | |
|---|---|
| Invariant | F-04: NULL seller / issuer is UNKNOWN; payments inherit the invoice predicate |
| Fixture | F0 (`I_N`, `I_XZ`) |
| Actor / role | X, Z |
| Request | own-read invoice list / detail / payments / receipts for `I_N`, `I_XZ`; `GET /api/v1/payments/attempts` |
| Status | list 200; details per predicate (200 if admitted, else 404); `/payments/attempts` 403 for a Seller |
| Outcome / reason | — |
| Body | `I_N` invisible to every Seller; `I_XZ` visible to X, Z, both or neither exactly as the approved S4 predicate says (a/b/c/d) |
| SQL | — |
| Source / decision | `payments/command/command.go:156-160`; M2 rec. 2 (S4 predicate); OPD-10 |

### NT-12 — Commission isolation and wallet → invoice link
| | |
|---|---|
| Invariant | Own entitlements only; an entitlement does not widen invoice scope |
| Fixture | F0; an entitlement of X on `I_Z` |
| Actor / role | X |
| Request | `GET /api/v1/commission/me/earnings?run_id=R1`; organisation commission reads; then `GET` own-read invoice `I_Z` |
| Status | 200; organisation commission 403; invoice per S4 predicate (404 unless the approved S4 rule admits entitlement-linked invoices) |
| Outcome / reason | `SUCCESS` |
| Body | Only X's entitlements; `totals` per unit equal X's sum; no Z entitlement |
| SQL | `SELECT count(*) FROM commission.historical_entitlement WHERE run_id=R1 AND holder_user_id=X` = number of items over all pages |
| Source / decision | `commission/query/query.go:468`; OPD-09; M2 rec. 2 |

### NT-13 — Cross-module leaks through reports
| | |
|---|---|
| Invariant | Report rows keyed by case / invoice / attempt never become a Seller side door |
| Fixture | F0 |
| Actor / role | X (own-read only) |
| Request | `GET /api/v1/reports/cases/{C_Z1}`, `/reports/invoices/{I_Z}`, `/reports/payments/{id}`, `/reports/custody/{C_Z1}`, `/mis/*`, `/invoice-review/{I_Z}` |
| Status | 403 |
| Outcome / reason | `UNAUTHORIZED / ACTION_DENIED` (or `ROW_SCOPE_DENIED` if a reporting capability were held with a narrower scope) |
| Body | No row data |
| SQL | — |
| Source / decision | `reporting/http/http.go:109`; OPD-07 |

### NT-14 — Customer identity and phone cross-module
| | |
|---|---|
| Invariant | Phone is discovery only; customer data reachable only through the approved field policy |
| Fixture | F0 (`K_X`, `K_Z`) |
| Actor / role | X |
| Request | `GET /api/v1/customers`, `/customers/{K_Z}`, `/customers/{K_X}/contact-points`, `/customers/{K_X}/evidence`; own-read case `C_X1` (check `customer_ref`) |
| Status | `/customers*` 403 (not in any proposed Seller bundle) |
| Outcome / reason | `UNAUTHORIZED / ACTION_DENIED` |
| Body | own-read case shows `customer_ref` only as the field policy allows; never Z's customer; no join by phone |
| SQL | No query joins on a phone column (code review assertion + architecture lint) |
| Source / decision | AGENTS.md Identity; GAP-04; D-30; D-45 |

### NT-15 — Unknown stage outcome (alias T14)
| | |
|---|---|
| Invariant | Missing money facts are not zero; Outcome Unknown is reconciled before retry |
| Fixture | `I_UK` |
| Actor / role | X |
| Request | own-read invoice detail / stages / totals for `I_UK` |
| Status | 200 |
| Outcome / reason | `PARTIAL` with a coverage / unknown reason |
| Body | Stage `state_mapping` UNKNOWN; "current payable" absent; remaining marked not final |
| SQL | — |
| Source / decision | RM-E03 §8; AGENTS.md Sensitive Outcomes; D-43 |

### NT-16 — Custody transfer freshness (alias T9)
| | |
|---|---|
| Invariant | SR-11: no cached scope or rows |
| Fixture | `C_Z1` |
| Actor / role | X |
| Request | own-read detail `C_Z1` (404) → import batch moves holder to X → repeat (200) → batch moves holder back → repeat (404) |
| Status | 404 → 200 → 404 |
| Outcome / reason | — |
| Body | `evaluated_as_of` / batch reference changes with each import |
| SQL | `custody.assignment.refreshed_batch_id` matches the batch reported |
| Source / decision | M2 N8; OPD-05 / SD-04 (transfer timing) |

### NT-17 — Cache headers (alias T12; F-07)
| | |
|---|---|
| Invariant | SR-12: per-user responses are not cacheable |
| Fixture | any |
| Actor / role | X; anonymous |
| Request | every own-read operation; `GET /api/v1/session`; `GET /api/v1/commission/me/earnings`; `GET /api/v1/payments/batches/{id}`; problem responses (403 / 404) |
| Status | as per operation |
| Outcome / reason | — |
| Body | Header `Cache-Control: no-store` on every response (including 200 data responses); `Vary: Cookie` on own-read responses. Through the staging nginx (`deploy/staging/nginx/nginx.conf`) the header is preserved |
| SQL | — |
| Source / decision | F-07; no business decision needed (Group A) |

### NT-18 — Multi-role context (alias T8)
| | |
|---|---|
| Invariant | SR-13: union per capability; scopes never merge or widen |
| Fixture | F0; X additionally holds a synthetic role with organisation `cases.case.read` |
| Actor / role | X (two grants) |
| Request | own-read cases list; `GET /api/v1/cases`; effective-access |
| Status | 200 / 200 / 200 |
| Outcome / reason | `SUCCESS` |
| Body | own-read list still `{C_X1, C_X2}`; organisation list governed only by the organisation capability; effective-access lists both matched roles sorted, with no selected role |
| SQL | — |
| Source / decision | RM-ACCESS-01 test J; RM-B02 §4 (GAP-05: multi-role policy open) |

### NT-19 — No business write activation
| | |
|---|---|
| Invariant | Read contracts enable no command; every command stays behind its own flag and rollout mode |
| Fixture | F0; all command flags off, rollout `Disabled` |
| Actor / role | X with every proposed Seller read |
| Request | `POST /cases/{C_X1}/contact-outcomes`, `POST /payments/uploads`, `POST /payments/invoices/{I_X}/batches`, `POST /hr/requests`, fulfillment POSTs, provider resolution |
| Status | per existing contract: 403 (no capability) or 200 `BLOCKED / FEATURE_DISABLED` / `TARGET_WRITE_DISABLED`, `applied=false` |
| Outcome / reason | never `SUCCESS` with `applied=true` |
| Body | `allowed_actions` false for every command on every own-read response |
| SQL | Row counts of `cases.contact_outcome_event`, `payments.payment_batch`, `payments.submitted_attempt`, `organization.hr_request*` unchanged |
| Source / decision | RM-ACCESS-01 test O; `TestNoFinancialExecutionCapability`; M2 rec. 1 |

### NT-20 — Legacy WordPress remains the authoritative writer
| | |
|---|---|
| Invariant | The target never writes to legacy 2.0.160 as part of any read |
| Fixture | F0; legacy MySQL fixture or the write-back outbox |
| Actor / role | X |
| Request | full own-read browse of S1, S4, S7 |
| Status | as per reads |
| Outcome / reason | — |
| Body | — |
| SQL | Write-back command / outbox tables unchanged; no outbound connection to the legacy writer (network recorder); all `_writeback` flags remain off (`TestAvailabilityNeverFromWritebackOrApply`) |
| Source / decision | RM-ACCESS-01 §4; AGENTS.md (KEEP + ADAPT) |

### NT-21 — Disabled feature flags (alias T11)
| | |
|---|---|
| Invariant | Flag off → no data, no executable link |
| Fixture | F0; module flag off (`sales.read_shadow`, `billing_payments.read_shadow`, `commission.read_shadow` in turn) |
| Actor / role | X with own-read grant |
| Request | own-read operations of that module; SPA navigation |
| Status | 200 |
| Outcome / reason | `BLOCKED / FEATURE_DISABLED` |
| Body | No `items`; effective-access `BLOCKED / FEATURE_DISABLED` for the capability; the SPA destination is hidden |
| SQL | — |
| Source / decision | RM-ACCESS-01 test F, navigation test M |

### NT-22 — Inventory guard for new operations
| | |
|---|---|
| Invariant | No new data-returning operation ships without a declared scope class and SR coverage |
| Fixture | Parsed `api/openapi/openapi.bundled.yaml` + a committed scope-class registry |
| Actor / role | — (static test) |
| Request | Enumerate all operations |
| Status | — |
| Outcome / reason | Test fails if an operation has no scope class, or a `GET` has no NT-07 case |
| Body | — |
| SQL | — |
| Source / decision | Leak audit §1 (105 operations); pattern of `TestL_ContractEnumEqualsRegistry` |

### NT-23 — Production preview isolation
| | |
|---|---|
| Invariant | The RM-E03 DEV preview and synthetic identities never reach production, and the preview never substitutes for authorization |
| Fixture | Production bundle (`vite build` + `vite preview`) |
| Actor / role | anonymous and an authenticated test account |
| Request | `/dev/seller`, `/dev/seller/leads`, `/dev/seller/wallet`; bundle grep |
| Status | SPA «صفحه پیدا نشد» |
| Outcome / reason | — |
| Body | No `[data-preview]`; no `SellerPreviewPage`, `RM-E03-SYNTHETIC-SELLER-FIXTURE` or `0900000xxxx` in the bundle; no non-GET request |
| SQL | — |
| Source / decision | RM-E03 §3 (existing `web-ci` grep and `role-navigation.spec.ts` — keep green after wiring) |

### NT-24 — Fail closed on infrastructure errors (alias T10)
| | |
|---|---|
| Invariant | SR-16 |
| Fixture | Fault injection: flag store unavailable; custody read error; DB timeout inside the scoped query |
| Actor / role | X |
| Request | own-read list and detail |
| Status | 500 (flag store, RM-ACCESS-01 §5) or problem 5xx |
| Outcome / reason | — |
| Body | Never a partial or unscoped list |
| SQL | — |
| Source / decision | RM-ACCESS-01 §5; R3 fault-injection pattern |

## Mapping from T1–T14

T1→NT-01 · T2→NT-08 · T3→NT-03 · T4→NT-05 · T5→NT-06 · T6→NT-04 · T7→NT-07 · T8→NT-18 · T9→NT-16 · T10→NT-24 · T11→NT-21 · T12→NT-17 · T13→NT-11 · T14→NT-15. New: NT-02, NT-09, NT-10, NT-12, NT-13, NT-14, NT-19, NT-20, NT-22, NT-23 (24 scenarios in total).
