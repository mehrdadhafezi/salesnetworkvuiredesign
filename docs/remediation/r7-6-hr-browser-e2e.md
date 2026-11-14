# R7-6 — HR read and HR request browser E2E (OP-04, OP-05)

Status: TEST-ONLY. No production code, migration, schema, flag default,
grant, capability, role bundle or business rule is changed. HR final apply
stays BLOCKED (OPD-05 / SD-04). Production remains **NO-GO**.

Baseline: `main` @ `c04861775e4a6c8729c671f44589b0bcb9b2168a`.
Stack: real Chromium (Playwright 1.56.1) → built SPA (`web/dist`) → real
`cmd/crm-api` binary → disposable PostgreSQL. It runs on the R7-1 harness
(`startCRMAPI`, `serveSPA`, `runPlaywright`, the non-local network guard)
in the existing `Full-stack E2E` workflow.

## Two evidence classes (never mixed)

| | **A. Production-equivalent** — `TestBrowserHRFullStack` | **B. Test-only overlay** — `TestBrowserHRReadOverlay` |
|---|---|---|
| Authentication | Real R1 sessions (login form, cookie, CSRF, idle and absolute expiry of 1m) | Existing development auth stub: `CRM_DEV_AUTH=1` under `CRM_ENV=test`. It is refused in production (`apiserver.Config.Validate`, `TestRunRefusesDevAuthInProduction`) and never combined with sessions. |
| Subject resolver | **Production** resolver: a subject holds only ACTIVE role grants, and none exist (D-42) | `authorization.DevSubjects`: in memory only. `dev-user-001` holds a synthetic HR role; `dev-user-002` holds nothing. No database grant exists. |
| Identity in the browser | Synthetic credentials for supervisor A (legacy user 4) and the senior (legacy user 3) | The Playwright context adds `X-Dev-User`. The production SPA bundle contains no dev identity. |
| Data | `setupWrites` (clean Slice 11 hierarchy), plus two `pending_review` requests created **before** the browser runs, through the domain service (not the API) | R7-5 integrated dataset: imported SHADOW evidence, including cycles, self-parent lines and multiple parents |
| What it proves | What the shipped product does today: every HR read and command is refused, the UI offers no action, and nothing changes | What the HR read pages render, as **backend-only** authorization evidence. Never production authorization, and never live HR data. |

**Positive create / approve / reject in the browser is BLOCKED.**
- No existing, isolated overlay maps a browser identity to a synthetic organisation user. `DevSubjects` carries no user id, so the service answers `UNAUTHORIZED / ROW_SCOPE_DENIED`; class B asserts exactly that.
- The `testSubjects` overlay of the Go HTTP tests is header-based and in-process. It has no session login, so the SPA cannot use it.
- A browser path would need a new authorization bypass, which the brief rules out.

The positive workflow and every domain refusal therefore stay covered by the **backend** suites, which run in Go CI:
- `r7_4_hrrequest_hardening_test.go`: creation, decisions, authority, no apply, concurrency, HTTP capability.
- `r7_4_def_hr_01_test.go`: DEF-HR-01 self-parent, cycle, depth, requester above chain.

## Coverage matrix

Legend: **E2E-A** browser on the production resolver · **E2E-B** browser on the test-only overlay · **BE** backend Go suite (CI) · **BLOCKED** not reachable without a new bypass.

| Scenario | Existing coverage before R7-6 | Added here | Outcome asserted |
|---|---|---|---|
| Workforce / person / structure / exceptions pages, production resolver | none (browser) | E2E-A | page heading + "no access" state, no table, no person link; every read API 403 `UNAUTHORIZED / ACTION_DENIED` |
| Filters, sort, cursor on a refused read | none | E2E-A | refusal unchanged by query parameters (403, or 400 for a malformed cursor) |
| Workforce renders imported evidence | Go HTTP read tests | E2E-B | every PRESENT person row; read-only marker; no edit control |
| Workforce filters (search, empty result) | Go HTTP | E2E-B | `q` narrows to the 2 cycle persons; "no result for this filter" |
| Cursor pagination | Go HTTP | E2E-B | limit 5 walks every person exactly once; next disabled when ≤ 25 |
| Unresolved evidence (multiple parents, self-parent) | Go read tests | E2E-B | "conflict: multiple current managers" pill; self-parent person page shows the SELF_PARENT finding |
| Hierarchy exceptions (CYCLE, SELF_PARENT), kind filter | Go read tests | E2E-B | API counts equal the database; the UI filter lists only SELF_PARENT |
| Subject without a capability | Go HTTP | E2E-B (`dev-user-002`) | pages show "no access"; API 403 `UNAUTHORIZED` |
| HR request list / detail (seeded, other reviewer's, unknown id) | BE | E2E-A | the same refusal for all (no existence oracle); no create form, no approve or reject button, no dialog |
| Create: in scope, out of scope, eligible / ineligible / no-op transfer, about oneself | BE (R7-4) | E2E-A | every case 403 `ACTION_DENIED` before any rule; retry with the same key gets the same refusal, nothing stored |
| Decide: recorded reviewer, wrong reviewer, stale `If-Match`, replay, unknown request, requester | BE (R7-4) | E2E-A | 403 `ACTION_DENIED`; no transition, no event |
| CSRF missing (with session) | R7-1/R7-3 pattern | E2E-A | 403 problem+json `ACTION_DENIED` before the contract check |
| Contract invalid (no key, array body, extra field, `decision: apply`, malformed id) | BE | E2E-A | 400 problem+json `CONTRACT_INVALID` |
| Anonymous | — | E2E-A | every read and command 403 `ACTION_DENIED` |
| Logout | R7-1/R7-3 | E2E-A | old cookie unauthenticated; every read and decision refused |
| Real session expiry (absolute 1m) | R7-3 | E2E-A | server reports unauthenticated after ≥ 55s; page returns to login |
| Flags OFF | — | E2E-A (`@flags-off`) | navigation hidden; "not enabled" state; reads and commands 200 `BLOCKED / FEATURE_DISABLED`, command `applied=false`, `TARGET_WRITE_DISABLED`, no command id |
| Positive create → step approve → pending_hr; rejection; history | BE (R7-4 Decisions, DEF-HR-01 ValidChains) | **BLOCKED** in browser | — |
| Wrong reviewer / stale version / duplicate / concurrent decisions (domain answer) | BE | **BLOCKED** in browser (capability first) | — |
| DEF-HR-01 self-parent and cycle refusal | BE (`r7_4_def_hr_01_test.go`, default suite) | **BLOCKED** in browser (capability first) | BLOCKED `LINKAGE_CONFLICT` in BE |
| HR final apply | BE (pending_hr BLOCKED OPD-05 / SD-04) | E2E-A (`decision: apply` is no contract value; flags-off shows no applied command) | never applied |
| Overlay identity tries to create | — | E2E-B | `UNAUTHORIZED / ROW_SCOPE_DENIED`, no command id, no row |
| No non-local request | R7 guard | E2E-A, E2E-B | `blocked` list empty |

## Database no-mutation evidence (orchestrator, after every project)

`schemaSnapshot` takes the row count plus a SHA-256 fingerprint of every row for every base table in `organization`, `identity`, `cases`, `custody`, `customers`, `billing`, `payments`, `finance`, `commission`, `integration`, `fulfillment`, `biawin` and `catalog`.
- **After versus before:** the snapshot must be identical, except the auth runtime tables a real login, logout or expiry writes (`identity.session`, `identity.auth_event`, `identity.credential`).
- **Class A, in addition:**
  - both seeded requests are still `pending_review`, version 1;
  - the only events are their `created` events;
  - there are no other requests;
  - every grant is `SHADOW`;
  - there are no write-back commands.
- **Class B, in addition:** `organization.hr_request` is empty.

## Remaining governance blockers (unchanged)

- **D-34, D-42, R5-SEC-01:** no role bundle carries HR read or HR request capabilities, there are no ACTIVE grants, and person-level separation of duties is not provable.
- **OPD-05, SD-04:** HR final apply is implementation-blocked.
- **G-H1, G-H2, G-H3** (R7-4 governance questions): the subject's employment status, snapshot versus live reviewer, and refusal order.
- **HR runtime** is NOT LIVE VERIFIED (no real HR role account).

## R7-7 readiness

R7-7 (reporting / MIS and Biawin read E2E, OP-21 / OP-26) can reuse both patterns:
- **Production-resolver refusal:** class A.
- **Read rendering:** class B, on `setupIntegrated` + `seedIntegrated`.

`DevSubjects` already gives `dev-user-001` the reporting, MIS, biawin and fulfillment read capabilities. The R7-5 dataset populates every projection, including UNKNOWN, CONFLICT and INCOMPLETE values.
