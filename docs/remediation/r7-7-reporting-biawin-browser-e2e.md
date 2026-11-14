# R7-7 — Reporting / MIS and Biawin read browser E2E (OP-26, OP-21)

Status: TEST-ONLY. No production code, migration, schema, flag default,
grant, capability, role bundle, KPI definition or business rule is changed.
No projection is rebuilt, no subscription is activated, no provider is
called. Production remains **NO-GO**.

Baseline: `main` @ `19b9ed70ed143fae35d0efb94df0ae2851e4a8f2`.
Stack: real Chromium (Playwright 1.56.1) → built SPA (`web/dist`) → real
`cmd/crm-api` binary → disposable PostgreSQL cloned from the CI-PERF-01
template. It runs on the R7-1 harness (`startCRMAPI`, `serveSPA`,
`runPlaywright`, the non-local network guard) in the existing
`Full-stack E2E` workflow.

**Data:** both orchestrators seed the **R7-5 integrated synthetic dataset** through the existing import path:
- staging → billing, payments, Biawin and fulfillment imports → finance and commission shadow runs → reporting projections;
- one reporting projection run, one Biawin import run, one Biawin parity check.

The seed fails the test if any fixture the suites rely on is missing (`seedR77`). No real customer information is used.

## Two evidence classes (never mixed)

| | **A. Production-equivalent** — `TestBrowserReportingBiawinFullStack` | **B. Test-only overlay** — `TestBrowserReportingBiawinReadOverlay` |
|---|---|---|
| Authentication | Real R1 sessions: login form, cookie, CSRF, idle and absolute expiry of 1m | Existing development auth stub: `CRM_DEV_AUTH=1` under `CRM_ENV=test`. It is refused in production and never combined with sessions. |
| Subject resolver | **Production** resolver: ACTIVE role grants only, and none exist (D-42). No role bundle carries a reporting or Biawin read capability. | `authorization.DevSubjects`, in memory only. `dev-user-001` holds the existing synthetic reporting and Biawin read capabilities; `dev-user-002` holds nothing. No database grant exists. |
| Identity in the browser | Two synthetic credentials for imported synthetic users | The Playwright context adds `X-Dev-User`. The production SPA bundle contains no dev identity. |
| What it proves | What the shipped product does today: every read is refused although the data exists, the UI shows no data and no action, nothing changes | What the pages render, checked as **UI == API answer == database**. This is **backend-only** authorization evidence, never production authorization. |

No new authentication bypass, capability, grant or overlay was added. Class B uses only the existing `DevSubjects` entries.

## Coverage matrix (page → API → database)

Legend: **E2E-A** browser on the production resolver · **E2E-B** browser on the test-only overlay · **BE** backend Go suite.

| Page | API | Database source | Before R7-7 | Added | Outcome asserted |
|---|---|---|---|---|---|
| `/mis` MIS overview | `GET /api/v1/mis/overview` | `reporting.projection_run`, `*_fact` | BE (`reporting_e2e_test.go`), Vitest | E2E-A, E2E-B | **B:** run id = DB; freshness pill = API; one card per FINAL metric, each value = API = DB; `payments.by_state` and accepted amount in IRT = DB sums; deferred NOT FINAL contracts listed with no number. **A:** 403 `ACTION_DENIED`, "no access", no card |
| MIS drill-down | card link → `/reports/cases?…` | `reporting.case_fact` | Vitest | E2E-B | the query contract is carried; report total = card value = DB unknown-state count |
| `/mis/source-quality` | `GET /api/v1/mis/source-quality` | projections + customer resolution | BE, Vitest | E2E-A, E2E-B | **B:** every cell = API; a null count renders "not applicable", **never 0**; cases unknown and custody conflict = DB |
| `/mis/reconciliation` | `GET /api/v1/mis/reconciliation` | `projection_run`, `metric_contract` | BE, Vitest | E2E-A, E2E-B | **B:** F02 is `NOT_EVALUATED` (no number, no drill link); F03 values = DB; runs table = API with run id = DB; contracts listed; pager disabled |
| `/reports/cases`, `/custody`, `/invoices`, `/payments` | `GET /api/v1/reports/{report}` (+ `/{id}`) | `reporting.case_fact`, `custody_fact`, `invoice_fact`, `payment_fact` | BE, Vitest | E2E-A, E2E-B | **B:** summary total = API = DB; table rows = API; run id shown; limit-2 cursor walk visits every row exactly once |
| Payments report | `?state=…` | `payment_fact` | BE | E2E-B | IRT sum and row count = DB; `UNMAPPED` rows = DB; `state=ACCEPTED` changes card, table and API together; `state=CANCELLED` is a real evaluated zero ("zero rows, real result") |
| Invoices / custody reports | — | `invoice_fact`, `invoice_unit_fact`, `custody_fact` | BE | E2E-B | unknown main state shown as `raw (UNKNOWN)`; total classes `UNKNOWN` and `INCOMPLETE` shown; custody `CONFLICT` rows = DB |
| Row detail | `GET /api/v1/reports/cases/{id}?run_id` | `case_fact` | Vitest | E2E-B | detail pinned to the answering run (`run_id` in URL); membership pill |
| Unknown pinned run | `?run_id=<none>` on a report and on the subscription list | — | BE | E2E-B | no value and no table rendered (never 0). Reports: the API answers **500** today (DEF-RP-01). Biawin: `UNKNOWN` with no meta, counts or rows. An unknown row id answers 404 |
| `/biawin/subscriptions` | `GET /api/v1/biawin/subscriptions` | `biawin.subscription`, `import_run` | BE (`biawin_e2e_test.go`), Vitest | E2E-A, E2E-B | **B:** run id = DB; SHADOW banner; envelope `PARTIAL / STALE_STATE` and the "stale (older than 24h)" pill, never shown as fresh; state counts = DB (EMPTY is a real 0); rows = DB; unresolved invoice (`#2999`, a known-unresolved R7-5 reference) shown UNKNOWN with its key; unmapped state shown UNKNOWN with the raw value; limit-2 cursor walk |
| Subscription filters | `?invoice_link_state=UNKNOWN`, `?state=EMPTY` | `biawin.subscription` | BE | E2E-B | unresolved filter = DB count; EMPTY shows the empty state, API `[]` |
| `/biawin/subscriptions/:id` | `GET …/{id}`, `GET …/{id}/items` | `subscription`, `subscription_item` | BE, Vitest | E2E-A, E2E-B | **B:** link pins the run; item count and rows = DB; limit-5 item cursor walk; unresolved invoice link stays UNKNOWN with its key; unknown id 404 |
| `/biawin/parity` | `GET /api/v1/biawin/parity` | `biawin.parity_check`, `parity_row` | BE, Vitest | E2E-A, E2E-B | **B:** outcome `UNKNOWN` rendered "unknown (not zero)", never MATCH; reasons shown; all eight counts = DB; rows = DB; `result=UNKNOWN` filter |
| API-only reads | `GET /api/v1/reporting/metrics`, `/metrics/{id}`, `/api/v1/invoice-review` (+ `/{id}`) | — | BE | E2E-A | 403 `ACTION_DENIED` (A), `BLOCKED / FEATURE_DISABLED` (flags off) |
| No grant | every read above | — | BE | E2E-B (`dev-user-002`) | pages "no access", no table; API 403 `ACTION_DENIED` |
| Filters / pinned run / row / cursor on a refused read | — | — | — | E2E-A | refusal unchanged |
| Unsafe methods | `POST`/`PUT`/`PATCH`/`DELETE` on the read paths | — | — | E2E-A | without CSRF: 403 problem+json `ACTION_DENIED`; with CSRF: no such operation (404/405) |
| Anonymous, logout, expiry | — | `identity.session` | R7-1/R7-6 pattern | E2E-A | anonymous and replayed cookie refused everywhere; absolute expiry after ≥ 55 s returns the page to login |
| Flags OFF | — | `platform_featureflags` (disposable DB only) | — | E2E-A (`@flags-off`) | navigation hidden; "not enabled"; every read 200 `BLOCKED / FEATURE_DISABLED` with no data |
| Read-only UI | — | — | — | E2E-A, E2E-B | no button other than filter / pager / detail; no dialog; no non-GET request to a read surface (`writes` fixture) |
| No non-local request | — | — | R7 guard | E2E-A, E2E-B | `blocked` list empty |

## Synthetic data and fixture mapping

| Fixture (R7-5) | Value checked | Where |
|---|---|---|
| Reporting projection run | `run_id` (one run), freshness `STALE` (fixed cutoff `2026-11-01T09:00Z`) | MIS, reports |
| Case facts | 6 total, 2 legacy leads, 2 unknown state | MIS cards, F03, cases report, drill-down |
| Custody facts | 6 total, 1 `CONFLICT` | custody report, source quality |
| Invoice facts | 4 total; unknown main state `synthetic_unfamiliar_status`; total classes MATCH / INCOMPLETE / UNKNOWN | invoices report |
| Payment facts | 5 attempts, 3010 IRT; 2 accepted, 1500 IRT; 1 `UNMAPPED` (`verified`) | payments report, MIS cards |
| Metric contracts | FINAL metrics with values; NOT FINAL deferred (F02 / OPD-07 / TAD #20) | MIS, reconciliation |
| Biawin import run | 5 subscriptions: 3 ACTIVE, 1 SUSPENDED, 1 UNKNOWN (`synthetic_paused`), 0 EMPTY | subscriptions |
| Subscription 504 | invoice `2999` UNKNOWN (listed in `legacytest.IntegratedKnownUnresolved`), case `77` UNKNOWN | list, detail, filter |
| Subscription 501 | 12 distinct items | detail, items cursor |
| Parity check | legacy 3, target 3, matched 3, unknown 1, outcome `UNKNOWN` / `OUTCOME_UNKNOWN` | parity |

Every expected number is computed by the orchestrator from the disposable database (SQL), passed as `E2E_RP_*` / `E2E_BW_*`, and compared with both the API answer and the rendered page.

## Integrity and no-mutation evidence

Before the browser runs, the orchestrator snapshots every row of `integration`, `identity`, `organization`, `cases`, `custody`, `customers`, `billing`, `payments`, `biawin`, `fulfillment`, `finance`, `commission`, `reporting`, `catalog` and `audit`. After all projects, it requires:
- no difference except the session runtime rows of a real login (`identity.session`, `identity.auth_event`, `identity.credential`; class A only);
- still exactly one reporting projection run, one Biawin import run and one parity check (no rebuild, import or parity run);
- zero non-SHADOW role grants, write-back commands, provider operations, Biawin activations, fulfillment work cases, submitted payment attempts and audit events.

The finance ledger, commission runs, payments and invoices are part of the snapshot, so no financial posting, payment mutation or commission credit can pass unnoticed. Flags are switched only inside the disposable database.

## Defects found (recorded, not fixed here)

R7-7 is test-only, and the fix needs an API contract decision, so it is recorded rather than changed:

| Id | Finding | Evidence | Effect | Proposed resolution (needs a contract decision) |
|---|---|---|---|---|
| DEF-RP-01 | `GET /api/v1/reports/{cases,custody,invoices,payments}?run_id=<unknown>` answers **500** | `internal/reporting/http/http.go`: the list handlers return `query.ErrNotFound` as an internal error; the row and metric handlers map it to 404; the list contracts define no 404 | Fails closed: no value, no table, no zero. But an operator sees a server error, and the error is logged as one | Map `ErrNotFound` on the list reads to 404, or to the existing `meta.available = false` answer that the SPA already renders as "unknown, not zero" |

The Biawin subscription list does not have this defect: it answers an unknown run as `UNKNOWN` (no run), which the suite asserts.

The overlay test pins today's status (500) with a comment naming the defect, so a fix makes it fail and forces the expectation to be updated in the same change.

## Remaining gaps and governance blockers

- **Production authorization of these reads is not proven and cannot be today.** No role bundle carries `reporting.*` or `biawin.*` read capabilities and no ACTIVE grant exists (D-42; D-34, R5-SEC-01 open). Class B is backend-only evidence.
- **KPI trust** is not established: OPD-07, OPD-01 (metric definitions), D-23 (PII / export) remain open. The suites check that values equal the projections, not that the KPI definitions are approved.
- **Invoice Review** has no SPA page (D-24); its API is covered for refusal only. With `CRM_AUDIT_DATABASE_URL` unset it fails closed, and the suite asserts no audit row was written.
- **UI pagination** cannot be exercised with the dataset: the pages request 25 or 50 rows and the dataset is smaller, so "next" is disabled. Cursor pagination is proven on the same API with small limits.
- **Fulfillment work-case pages** (OP-22 / OP-23) are covered by R7-3, not here.
- **Live / real data parity** (ADR #18, G) remains BLOCKED. All data is synthetic.
