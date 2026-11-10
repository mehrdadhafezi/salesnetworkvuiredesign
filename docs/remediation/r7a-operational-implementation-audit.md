# R7-A — Operational CRM implementation gap audit

**Baseline:** `f0c497b82638283901172079822e634567bd70c8` (main, PR #46 merged).
- PR #47 (R6-B) is **open and not merged**, so nothing in it counts as implemented.
- This audit reads code, routes, handlers, migrations and tests. It infers nothing from design documents alone.
- **Production remains NO-GO.**

## 0. How the evidence was gathered

| Evidence | Result on the baseline |
|---|---|
| API contract `api/openapi/openapi.yaml` | 103 unique paths carrying **105 operations: 92 GET + 13 POST** (two paths have both a GET and a POST), 0 PUT / PATCH / DELETE |
| Frontend `web/src` | 132 files, 48 routed pages (incl. dev primitives) (`web/src/app/routes.tsx`), 14 test files |
| Frontend tests `npx vitest run` | **119 / 119 passed** (14 files); `tsc --noEmit` clean. All against `web/src/test/fakeApi.ts`: no test drives the real API |
| Go tests | 92 packages `ok`, 73 without tests. 521 `Test*` functions; per module: platform 113, integration 128, kernel 53, finance 27, architecture 26, commission 23, identity 16, billing 14, customers 14, authorization 13, custody 12, reporting 11, payments 10, cases 9, organization 5, biawin 4, catalog 3, fulfillment 3 |
| Database-backed tests | Skipped locally (no PostgreSQL in this container). CI runs them with `CRM_TEST_REQUIRE_POSTGRES=1` (`internal/platform/pgtest`). Go CI "Go kernel and platform gates" was **green** on `a0c2abf`, whose tree is identical to the baseline |
| Feature flags in code | About 30 keys. All default OFF; reads are `*.read_shadow` / `*_read`; writes are `cases.nonfinancial_write`, `payments.manual_write`, `organization.hr_request_write`, fulfillment / provider operations; `biawin.activate_production` is CONTROLLED → BLOCKED |
| Deployment | `deploy/docker` (app, web, backup images), `deploy/staging` (compose, nginx serving the SPA, bootstrap, drill, fault scripts) |

**Structural finding.** The target is a **read-shadow of the legacy CRM, plus a small set of controlled writes.**
- All business data (customers, cases, custody, invoices, payments, subscriptions, commission) enters only through
  `crm-legacy-import` (synthetic packages only; real packages are BLOCKED in `cmd/crm-legacy-import/main.go`
  `load`).
- There is **no native creation** of a customer, lead / case, invoice, payment stage, complaint or ticket.
- The legacy WordPress CRM remains the only system of record and the only writer (P1-8).

**Classes:**
- A — implemented and tested.
- B — implemented, tested insufficiently.
- C — partial.
- D — contract / evaluator / mock only.
- E — missing.
- F — blocked by an unapproved decision.
- G — needs real-data or runtime evidence.

**Layers:** BE backend · FE frontend · E2E integrated end-to-end · RT runtime-verified · PA production-authorized.

## 1. Feature inventory and implementation matrix

### 1a. Implementation and evidence

| ID | عنوان | Workflow | Frontend | Backend route / command | Model (schema.table) | Tests (result) | Class | BE / FE / E2E / RT / PA | Authorization and evidence blockers |
|---|---|---|---|---|---|---|---|---|---|
| OP-01 | ورود و نشست کاربر | Login, session, logout, expiry | `shared/auth/LoginPage.tsx`, `currentSession.tsx` | `POST /api/v1/auth/login`, `/logout`, `/logout-all`; `internal/identity/session`, `password` | `identity.credential`, `identity.session` (identity migrations) | identity 16 + `auth.test.tsx` 6 (pass; DB tests via CI) | **B** | BE ✔ · FE ✔ · E2E ✘ (no browser↔API test) · RT ✘ · PA ✘ | D-34 (provisioning) F; R5-SEC-01 F |
| OP-02 | کنترل دسترسی (قابلیت‌ها) | Capability check per read / command; `allowed_actions` hints | `shared/permissions/allowedActions.ts` | `internal/authorization` (`guard.go`, `Evaluate`); grants from imported legacy roles | `identity.user_account`, grant tables | authorization 13, architecture lint 26 (pass) | **A** (enforcement) | BE ✔ · FE ✔ (hint only) · E2E ✘ · RT ✘ · PA ✘ | Grants exist only from legacy import; D-42, E-7 F |
| OP-03 | ایجاد کاربر و مدیریت دسترسی | Provision accounts, grant / revoke capabilities | — | — (no command) | — | — | **E + F** | none | D-34, D-42, E-7, R5-SEC-01 |
| OP-04 | ساختار سازمانی و نیروی کار (خواندنی) | Units, positions, employments, hierarchy, exceptions | `modules/hr/pages/{Workforce,Person,Structure,Exceptions}Page.tsx` | `GET /api/v1/hr/...` (`internal/organization/query`) | `organization.*` | organization 5, `hr.test.tsx` 9 (pass) | **B** | BE ✔ · FE ✔ · E2E ✘ · RT ✘ (HR NOT LIVE VERIFIED) · PA ✘ | G (real HR data, ADR #18) |
| OP-05 | درخواست‌های منابع انسانی (ایجاد و تأیید مرحله‌ای) | Supervisor creates transfer / termination request; step decision | `modules/writes/pages/HRRequest(s)Page.tsx` | `POST /api/v1/hr/requests`, `POST /api/v1/hr/requests/{id}/decisions` (`internal/organization/hrrequest`) | `organization.hr_request*` | `writes.test.tsx` 7; HTTP e2e in `internal/integration/legacy/shadow/writes_http_e2e_test.go` | **C** | BE ✔ (create + step) · FE ✔ · E2E backend only · RT ✘ · PA ✘ | Final apply BLOCKED: OPD-05, SD-04, R5-SEC-01 |
| OP-06 | هویت مشتری (خواندنی، گزارش، پروندهٔ ادغام) | Resolve customer identity; list merge candidates | `modules/customers/pages/*` | `GET /api/v1/customers/...` (`internal/customers/query`, `domain`) | `customers.*` | customers 14, `customers_e2e_test.go`, `customers.test.tsx` 12 (pass) | **A** (read) | BE ✔ · FE ✔ · E2E backend ✔ · RT ✘ · PA ✘ | Merge: D-30, D-29/D-31 F; G real data |
| OP-07 | ایجاد و ویرایش مشتری | Create / edit customer natively | — | — | — | — | **E + F** | none | Legacy is the only writer (P1-8); D-29/D-31; phone is never identity |
| OP-08 | ورود سرنخ و ایجاد پرونده | Create lead / case; initial assignment | — | — (`cases.legacy_leads_only`) | — | — | **E + F** | none | ADR-01, OPD-10, P1-8 (single writer) |
| OP-09 | پرونده‌های فروش و متولی (خواندنی) | List / detail of cases, custody history | `modules/sales/pages/{Cases,Case,Custody,SalesReport}Page.tsx` | `GET /api/v1/cases...`, custody reads | `cases.case`, `custody.*` | cases 9, custody 12, `sales_e2e_test.go`, `sales.test.tsx` 7 (pass) | **A** (read) | BE ✔ · FE ✔ · E2E backend ✔ · RT ✘ · PA ✘ | G real data |
| OP-10 | ثبت نتیجهٔ تماس فروشنده | Seller records contact outcome on a held lead | `modules/writes/ContactOutcomePanel.tsx` | `POST /api/v1/cases/{case_id}/contact-outcomes` (`internal/cases/command`) | `cases.contact_outcome*`, command record, outbox | `writes_http_e2e_test.go`, `writes.test.tsx` (pass) | **B** | BE ✔ · FE ✔ · E2E backend ✔, browser ✘ · RT ✘ · PA ✘ (flag OFF) | Duplicate outcome D-29/D-31 F; legacy write-back E-3 |
| OP-11 | ادعا، انتقال، برگشت، بازنشانی و آزادسازی پرونده (کسری و برگشت) | Claim case for invoice; return; reset; release | — | Evaluators only: `internal/cases/claimgate`, `internal/custody/release` (imported by tests only; lint `r4-evaluator-wiring`) | — | claimgate / release tests, R4 race harness (synthetic) | **D + F** | BE evaluator · FE ✘ · E2E ✘ | ADR-01, F01, OPD-03, OPD-10, P1-8 |
| OP-12 | فاکتورها، تلاش‌های پرداخت و گزارش صورت‌حساب (خواندنی) | Invoice list / detail; attempts; billing report | `modules/billing/pages/*` | `GET /api/v1/billing/...` | `billing.*`, `payments.attempt` | billing 14, `billing_e2e_test.go`, `billing.test.tsx` 7 (pass) | **A** (read) | BE ✔ · FE ✔ · E2E backend ✔ · RT ✘ | G real data |
| OP-13 | صدور فاکتور و مرحلهٔ بعد | Issue invoice; next payment stage | — | Evaluator only: `internal/billing/issuance` | — | issuance tests (pass) | **D + F** | BE evaluator only | ADR-01, F01, D-32 (billing lifecycle), D-28 |
| OP-14 | بارگذاری مدرک و ارسال دستهٔ پرداخت | Upload receipt; submit payment batch for an invoice | `modules/payments/PaymentSubmissionPanel.tsx`, `PaymentBatchPage`, `PaymentEvidencePage` | `POST /api/v1/payments/uploads`, `POST /api/v1/payments/invoices/{id}/batches` (`internal/payments/command`) | `payments.upload`, `payments.batch`, evidence store | payments 10, `payments.test.tsx` 8 (pass) | **B** | BE ✔ · FE ✔ · E2E ✘ browser · RT ✘ · PA ✘ (flag OFF) | D-44 retention F; object storage runtime G |
| OP-15 | صف بررسی مالی و تصمیم تأیید یا رد | Finance reviews attempt; approve / reject | `modules/payments/pages/FinanceReview(Attempt)Page.tsx` | Read: `GET /api/v1/finance/review...`. Decision: evaluator only (`internal/finance/review`) | `payments.*` (read) | finance 27 (pass) | **A** (read) · **D + F** (decision) | BE read ✔ / decision evaluator · FE read ✔ | OPD-08, D-43, REJECT-VOCAB, R5-SEC-01 (person-level SoD), NO_DECISION_CAPABILITY |
| OP-16 | دفتر کل (سایه) و ثبت سند | Shadow ledger derivation; posting | `modules/finance/pages/{Dashboard,Ledger,Balances,Reconciliation}` | `GET /api/v1/finance/...`; `crm-finance-shadow`; posting validator `internal/finance/posting` | `finance.*` | finance 27 (pass) | **A** (shadow read) · **D + F** (posting) | BE shadow ✔ · posting contract only | D-41, OPD-06, D-37; G real-data parity |
| OP-17 | استرداد وجه | Refund request → review → authorise → execute | — | State machine only: `internal/finance/refund` (all transitions BLOCKED) | — | refund tests (pass) | **D + F** | none executable | OPD-04, R5-SEC-01, OPD-08, D-41, D-44 |
| OP-18 | پورسانت (سایه، درآمد من، سیاست‌ها) | Shadow commission runs; reconciliation; my earnings | `modules/commission/pages/*` | `GET /api/v1/commission/...`; `crm-commission-shadow` | `commission.*` | commission 23, `commission.test.tsx` 6 (pass) | **A** (shadow read) · credit / payout **E + F** | BE shadow ✔ · FE ✔ | OPD-09, D-36, D-41 |
| OP-19 | مسیر مشتری ناراضی: خدمات پس از فروش → شکایت | After-sales intake, complaint case, outcomes (refund / barter / other) | — | — (only the route enum `RouteComplaint` in `internal/biawin/activation`) | — | — | **E** (+ F for refund outcome) | none | Owner / ACK / SLA: OPD-02; runtime owner NOT VERIFIED (Cross-Role §7); refund outcome OPD-04 |
| OP-20 | مسیر بدون خدمات → بایگانی | Archive with business reason | — | — (route enum `RouteArchive` only) | — | — | **E** | none | Reason vocabulary and owner (OPD-01/OPD-02); never deletes or voids history |
| OP-21 | اشتراک‌ها و برابری بیاوین (خواندنی) | Subscriptions, items, parity | `modules/biawin/pages/{Subscriptions,Subscription,Parity}Page.tsx` | `GET /api/v1/biawin/...`; `crm-biawin-shadow` | `biawin.*` | biawin 4, `biawin.test.tsx` 12 (pass) | **B** | BE ✔ · FE ✔ · E2E ✘ | G real data |
| OP-22 | فعال‌سازی اشتراک پلاس | Activation decisions (SHADOW_COMPARE only) | `operations/pages/ActivationDecisionsPage.tsx` | `internal/biawin/activation`; production activation CONTROLLED → BLOCKED (DB CHECK) | `biawin.activation*` | biawin tests (pass) | **C + F** | SHADOW only | D-35a–e (C-03 mapping), D-38 |
| OP-23 | پرونده‌های کاری انجام خدمت (مسیر خدمات مازاد) | Assign, record action, attach evidence, complete | `operations/pages/WorkQueue/WorkCasePage.tsx`, `CommandResult.tsx` | `POST /api/v1/fulfillment/work-cases/{id}/{assignment,operational-actions,evidence-records,outcome}` (`internal/fulfillment/work`) | `fulfillment.*` | fulfillment 3, `operations.test.tsx` 8 (pass) | **B** | BE ✔ · FE ✔ · E2E ✘ · RT ✘ (synthetic provider) | Financial components BLOCKED (D-35, D-38); real Biawin API contract G |
| OP-24 | عملیات ارائه‌دهنده و حل نتیجهٔ نامعلوم | Provider operations list; resolve UNKNOWN | `operations/pages/ProviderOperationsPage.tsx` | `POST /api/v1/integration/provider-operations/{id}/resolution` (`internal/integration/providerops`) | `integration.provider_operation*` | integration 128 (pass) | **B** | BE ✔ · FE ✔ · RT ✘ | Real provider contract G |
| OP-25 | تیکت بین‌سازمانی | Inter-organisational tickets | — | — | — | — | **E** | none | No contract or spec found in `docs/source-of-truth` (0 code references) |
| OP-26 | گزارش‌ها، MIS و داشبوردها | Projections, MIS overview, source quality, report pages | `modules/reporting/pages/*` | `GET /api/v1/reporting/...`; `crm-reporting` rebuild | `reporting.*` | reporting 11, `reporting_e2e_test.go`, `reporting.test.tsx` 9 (pass) | **A** (read) | BE ✔ · FE ✔ · E2E backend ✔ · RT ✘ | KPI trust: OPD-07, OPD-01; D-23 (PII / export); G |
| OP-27 | پردازش پس‌زمینه (outbox، رسیدها، انقضا، تطبیق ارائه‌دهنده) | Relay, receipt finalise, upload-expiry report, provider reconcile | — | `cmd/crm-worker` (`compose.go`) | `platform.outbox`, `inbox` | crm-worker 5, platform 113 (pass); R2 fault injection | **B** | BE ✔ · RT ✘ (flags OFF, no deployment) | D-44 (purge); write-back delivery BLOCKED |
| OP-28 | ورود snapshot قدیمی و مدل سایه | Stage synthetic snapshot; shadow import per module; reconcile | — | `cmd/crm-legacy-import` (specs, validate, load, import-*, reconcile-*) | `integration.*` staging, module shadows | integration 128, crm-legacy-import 9, shadow e2e tests (pass) | **A** (synthetic) · **G** (real) | Real load BLOCKED | ADR #18, E-4, E-5, D-23 |
| OP-29 | بازنویسی در سیستم قدیمی (write-back) | Deliver target writes to legacy | `modules/writes/pages/WritebackPage.tsx` (diagnostics) | `internal/integration/writeback` (synthetic adapter, development only) | `integration.writeback*` | integration tests (pass) | **D + F** | no production connector | E-3, P1-8 |
| OP-30 | استقرار، پشتیبان، بازیابی، پایش | Staging compose, backup / restore, DR drill, metrics / alerts | — | `deploy/staging/*`, `cmd/crm-backup`, `cmd/crm-healthcheck`, `deploy/observability` | — | crm-backup 2, `deploy/staging/stage_test.go`, R3 fault-injection (pass) | **B** | Disposable staging only · RT ✘ | P0-4 column B unsigned |
| OP-31 | پوستهٔ رابط کاربری، مسیرها، طراحی | Shell, nav, design primitives, error boundary | `app/*`, `shared/ui/*` | — | — | `App.test.tsx` 10, `contract.test.ts` 9 (pass) | **A** (unit) | Browser E2E ✘ | — |

### 1b. Missing work, dependencies, order and acceptance

| ID | Missing work | Depends on | Order | Complexity | Acceptance criteria | Source references |
|---|---|---|---|---|---|---|
| OP-01 | Browser E2E through the real API; provisioning (blocked) | D-34 for provisioning | 1 (E2E) | S | A browser test logs in against `crm-api` + PostgreSQL with a synthetic credential; session expiry and logout verified | `internal/identity/session`, `web/src/shared/auth` |
| OP-02 | Full-stack check that the UI hides actions the server denies; grant administration (blocked) | D-42, E-7 | 1 | S | E2E: a user without the capability sees no action and the server returns 403 | `internal/authorization/guard.go` |
| OP-03 | Everything | D-34, D-42, R5-SEC-01, E-7 | after M2 | L | Approved provisioning flow with audit; no grant without a recorded authorization | — |
| OP-04 | Browser E2E; real HR verification | ADR #18, HR role accounts | 2 | S | E2E with synthetic org data | `internal/organization/query` |
| OP-05 | Final apply | OPD-05, SD-04, R5-SEC-01 | after M2/M5 | M | Apply only with a named successor; tests for every open-work type | `internal/organization/hrrequest/hrrequest.go` |
| OP-06 | Merge command | D-30, ADR #18 | after M3 | M | Merge only on approved evidence threshold; never phone alone | `internal/customers/domain` |
| OP-07 | Native customer write model | Authority switch (P1-8), D-29/D-31 | late | L | Single-writer guarantee proven before any target write | — |
| OP-08 | Native intake and assignment | ADR-01, OPD-10, P1-8 | late | L | Intake never races legacy writers | — |
| OP-09 | Browser E2E | — | 1 | S | E2E: list → detail → custody history on synthetic data | `internal/cases/query`, `internal/custody/query` |
| OP-10 | Browser E2E of the full write path; write-back delivery | E-3 for delivery | **1 (first task)** | S | §6 | `internal/cases/command/contact_outcome.go`, `web/src/modules/writes/ContactOutcomePanel.tsx` |
| OP-11 | Commands, routes, UI | ADR-01, F01, OPD-03, OPD-10 | after M3 | L | R4 race harness scenarios pass against the real command | `internal/cases/claimgate`, `internal/custody/release` |
| OP-12 | Browser E2E | — | 2 | S | E2E invoice list / detail / attempts | `internal/billing/query` |
| OP-13 | Issue / next-stage commands | ADR-01, F01, D-32, D-28 | after M3 | L | Issue never races a return (INV-009) | `internal/billing/issuance` |
| OP-14 | Browser E2E with object storage stub; runtime | D-44 (purge) | 2 | M | E2E upload → batch → evidence visible; idempotent retry | `internal/payments/command` |
| OP-15 | Approve / reject commands | OPD-08, D-43, REJECT-VOCAB, R5-SEC-01 | after M2 + M4-A | L | Maker ≠ checker by person; never SUCCESS without READY evidence | `internal/finance/review` |
| OP-16 | Posting command | D-41, OPD-06, ADR #18 parity | after M4-B | L | Posting idempotent; basis named; parity on real data | `internal/finance/posting` |
| OP-17 | Refund commands | OPD-04 (+ M2) | after M4-D | L | Independent humans; proof stored; reconciliation | `internal/finance/refund` |
| OP-18 | Credit / payout | OPD-09, D-36, D-41 | after M4-C | L | One engine per entitlement; no duplicate credit | `internal/commission` |
| OP-19 | Complaint module (intake, case, outcomes, routing from contact outcome) | OPD-02 owner / ACK; product contract for complaint workflow; OPD-04 for the refund outcome | after product contract | L | Pathway 1 end to end; refund outcome only through P-FIN-D | `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md` §7 |
| OP-20 | Archive command with reason | Reason vocabulary / owner (OPD-01/OPD-02) | after product contract | M | Archive never deletes, releases or voids history | — |
| OP-21 | Browser E2E | — | 2 | S | E2E subscription list / detail / parity | `internal/biawin/query` |
| OP-22 | Production activation | D-35a–e (C-03), D-38 | after M6 | L | — | `internal/biawin/activation` |
| OP-23 | Browser E2E; real provider contract | Biawin API contract | 2 | M | E2E assign → action → evidence → outcome with the synthetic provider | `internal/fulfillment/work` |
| OP-24 | Browser E2E | — | 3 | S | E2E resolve an UNKNOWN operation | `internal/integration/providerops` |
| OP-25 | Contract, then everything | Product contract (none found) | — | ? | — | — |
| OP-26 | Browser E2E; trusted KPIs | OPD-07, OPD-01, D-23 | 3 | S | E2E report pages render projections from synthetic data | `internal/reporting` |
| OP-27 | Runtime on staging | Deployment decision | 3 | S | Worker drill on disposable staging | `cmd/crm-worker/compose.go` |
| OP-28 | Real snapshot path | ADR #18 (G1–G3), E-4, E-5, D-23 | after M1 | L | Reconciliation register filled; zero unexplained deltas | `cmd/crm-legacy-import` |
| OP-29 | Production connector | E-3, P1-8 | late | L | Outcome-unknown reconciled before retry | `internal/integration/writeback` |
| OP-30 | Real environment | P0-4 sign-off | late | M | Column B signed | `deploy/` |
| OP-31 | Browser E2E harness | — | 1 | S | Harness runs in CI | `web/src/app` |

## 2. Critical missing functionality

1. **No native write model for core entities.** There is no creation of customers, leads / cases or invoices, and no
   claim / return / issue command. The CRM cannot run a sale without the legacy system.
2. **Customer pathways 1 and 2 are absent.** No after-sales, complaint or archive module exists (OP-19, OP-20).
   Pathway 3 exists only as Biawin read plus fulfillment work cases (OP-21–OP-23).
3. **No financial execution.** Approve / reject, posting, refunds and commission credit exist as evaluators or
   contracts only (OP-15–OP-18), by design, pending decisions.
4. **No user provisioning or grant administration** (OP-03).
5. **No full-stack test.** No test drives the real UI against the real API and database. The frontend is tested
   only against a fake API (OP-31).
6. **No legacy write-back connector** (OP-29). So even the implemented writes (OP-10, OP-14, OP-05) cannot become
   the record of truth while legacy remains the writer.
7. **Inter-organisational tickets** have no contract or code (OP-25).

## 3. Completeness assessments

**Frontend.** 48 routed pages cover read views for:
- HR, customers, sales, billing, finance, commission, Biawin, fulfillment, reporting and operations;
- six write surfaces: contact outcome, HR request create / decide, payment upload / batch, fulfillment work
  actions, provider resolution, and login.

Typecheck is clean and 119 unit tests pass. Missing: creation forms, complaint / archive flows, financial decision
UIs (deliberately absent) and any browser-level test against the real API.

**Backend.**
- 14 business modules plus kernel, platform and integration, each owning its schema (architecture lint).
- Read APIs for every imported domain.
- 13 POST commands.
- Evaluators for the decision-gated commands.
- Worker runtime.
- Backup / DR tooling.

Missing: native create / lifecycle commands, financial execution, write-back connector and provisioning.

**End-to-end coverage.**
- Backend HTTP → DB end-to-end tests exist for sales, billing, customers, reporting, generic API and writes, in
  `internal/integration/legacy/shadow/*_e2e_test.go`.
- There is no UI → API → DB test.
- No complete business workflow, from lead to paid invoice to commission, can run in the target. Each stage
  depends on legacy or on a blocked decision.

**Automated tests.**
- Go: 521 test functions; database tests enforced in CI.
- Web: 119 unit tests.
- Synthetic race harness (R4) and fault injection (R2, R3).
- Gaps:
  - browser E2E;
  - frontend tests against the real OpenAPI responses (only `contract.test.ts` checks the generated schema);
  - fulfillment (3 tests) and organization (5 tests) are thin for their surface.

## 4. Integration gaps and operational blockers

| Gap / blocker | Effect | Owner |
|---|---|---|
| Legacy write-back connector (E-3) and legacy writer disable (P1-8) | Target writes cannot become authoritative | Engineering + Human Owner |
| Real-data access (ADR #18) | No parity evidence; P0-3 open | M1 |
| Person identity (R5-SEC-01) | No person-level separation of duties; finance / refund / HR apply blocked | M2 |
| Real Biawin API contract | Fulfillment and activation stay synthetic | Product + provider |
| Object storage and deployment (P0-4) | Uploads and workers not runtime-verified | Operations |
| HR and Finance real role accounts | NOT LIVE VERIFIED | HR, Finance |

## 5. Features that are safe to develop now (synthetic data, existing approved rules, no activation)

These need no new business authority. Every flag stays OFF outside disposable test databases, and nothing is
deployed.

1. **Full-stack browser E2E harness** (OP-31) and E2E tests for the implemented flows: OP-01, OP-02, OP-09, OP-10,
   OP-12, OP-14, OP-21, OP-23, OP-24, OP-26.
2. **Test depth** for thin modules: fulfillment work commands, organization HR request edge cases.
3. **Read-surface completeness** inside existing capabilities, where the data already exists in the shadow model
   (for example, missing filters or pagination parity). Only where the frozen design specifies it.
4. **Synthetic seed / demo dataset command** that composes existing synthetic fixtures (`legacytest`) into one
   coherent dataset for local runs and E2E, without real data.
5. **Operational readiness on disposable staging** (worker drill, backup / restore drill), as R3 already allows.
6. **Engineering prerequisites that need no decision:** E-5 extract specs as specifications only (no extraction),
   OpenAPI-driven contract tests.

## 6. Features blocked pending decisions

| Wait for | Features |
|---|---|
| ADR #18 (M1) | OP-28 real path, real-data parity for OP-04 / OP-06 / OP-09 / OP-12 / OP-16 / OP-18 / OP-26, F01 evidence, D-30 merge evidence |
| R5-SEC-01 (M2) (+ D-34, D-42) | OP-03, OP-05 apply, OP-15 decisions, OP-17 refunds |
| ADR-01 (M3) | OP-08, OP-11, OP-13 |
| F01 (M3, needs ADR #18) | OP-11, OP-13 |
| OPD-03, OPD-10, D-32, D-28 (M3) | OP-11, OP-13 |
| OPD-08, D-43, D-25, D-44, REJECT-VOCAB (M4-A) | OP-15, OP-14 purge |
| D-41, OPD-06, D-37 (M4-B) | OP-16 |
| OPD-09, D-36 (M4-C) | OP-18 credit / payout |
| OPD-04 (M4-D) | OP-17, OP-19 refund outcome |
| OPD-05, SD-04 (M5) | OP-05 apply |
| D-35a–e (after C-03), D-38, D-33 (M6) | OP-22, financial parts of OP-23 |
| OPD-02, OPD-01, OPD-07, D-23 (M6) | OP-19 owner / ACK, OP-20, trusted OP-26 |
| Product contract (not in repo) | OP-19 workflow detail, OP-25 tickets |
| E-3, P1-8 (engineering + owner) | OP-29; authoritative use of OP-07 / OP-08 / OP-10 / OP-14 |

## 7. Proposed engineering PR sequence

Every PR follows the same rules:
- one objective;
- synthetic data only;
- flags set only inside disposable test databases;
- no Production activation;
- no new capability;
- no change to `plugin/current`.

| PR | Objective | Expected files / modules | Tests | Acceptance criteria |
|---|---|---|---|---|
| **R7-1** | Full-stack browser E2E harness + first flow: login → case → contact outcome (OP-01, OP-02, OP-09, OP-10) | `web/e2e/` (Playwright), a Go test helper that boots `crm-api` against a disposable PostgreSQL with a synthetic snapshot, CI job | The E2E test itself | §8 |
| R7-2 | E2E: invoices, attempts, payment upload → batch (OP-12, OP-14) | `web/e2e/` | E2E | Upload and batch visible; idempotent retry; no Finance decision possible |
| R7-3 | E2E: fulfillment work case and provider resolution (OP-23, OP-24) | `web/e2e/` | E2E | Full synthetic work-case lifecycle; financial components refused with their decision ids |
| R7-4 | Fulfillment and organization test depth (backend) | `internal/fulfillment/work`, `internal/organization/hrrequest` tests | Unit + DB | Every refusal reason covered |
| R7-5 | Synthetic demo dataset command (one coherent synthetic world) | `cmd/crm-legacy-import` or `internal/integration/legacy/legacytest` | Unit + e2e | Deterministic; marked synthetic; real packages still BLOCKED |
| R7-6 | E2E: HR request create / step decision; HR read (OP-04, OP-05) | `web/e2e/` | E2E | Final apply refused (OPD-05 / SD-04) |
| R7-7 | E2E: reporting / MIS and Biawin read (OP-21, OP-26) | `web/e2e/` | E2E | Projections render; unknown never shown as zero |
| Later (decision-gated) | Commands for OP-11 / OP-13 / OP-15–OP-18 / OP-19 / OP-20, one per verified decision | per module | per R6 follow-up map | per decision record |

## 8. First executable task: R7-1

**Objective:** prove one integrated slice end to end, real browser → real `crm-api` → real PostgreSQL, on synthetic
data. The slice is: seller logs in, opens a held case, records a contact outcome, and sees it in the case history.

**Scope:**
- A Playwright E2E suite under `web/e2e/`. Chromium is pre-installed in this environment; CI installs it.
- A Go-side fixture (test-only command or `TestMain` helper) that:
  - creates a disposable database;
  - runs migrations;
  - stages the existing synthetic snapshot through the existing import path;
  - creates one synthetic seller credential through the existing test helpers;
  - sets `cases.nonfinancial_write` and the read flags **only in that disposable database**;
  - starts `crm-api` and the Vite preview.
- A CI job, similar to Go CI's PostgreSQL service, that runs the suite.

**Acceptance tests:**
1. Login with the synthetic seller succeeds. A wrong password fails with the generic error.
2. Case reads are capability-scoped (not limited to held cases). On a case held by another seller, the
   contact-outcome action hint is absent and a direct command is refused by the holder check.
3. Recording a contact outcome on a held case succeeds and appears in the history. Replaying the same idempotency
   key does not duplicate it.
4. A synthetic user without the write capability sees no action, and a direct API call is refused (403).
5. With the write flag OFF, the UI shows the command as unavailable and the API refuses it.
6. Logout ends the session; a protected page redirects to login.
7. The suite fails if any request reaches a non-local host, and it uses no real data.

**Restrictions:**
- No new endpoint, capability, migration or business rule.
- No flag is set in any config, staging or production file.
- No deployment.

## 9. Completion estimate (evidence-based, with assumptions)

| Track | Scope | Size (PRs) | Assumptions |
|---|---|---|---|
| Safe now (R7-1…R7-7) | E2E coverage of every implemented flow, test depth, synthetic dataset | ≈ 7 PRs | Velocity similar to Slices 0–15 / R1–R6 (36 merged PRs on main since 2026-10-08); CI stays green; no decision needed |
| Decision-gated execution | OP-11, OP-13, OP-15–OP-18, OP-05 apply, OP-22 | ≈ 10–14 PRs **after** the verified decisions | One PR per command family; each needs its verified decision (M2–M6) first. The calendar time is set by management decisions, not engineering |
| Missing pathways and native writes | OP-19, OP-20, OP-07, OP-08, OP-25 | Not estimable until product contracts exist (complaint workflow, archive reasons, tickets) | Contracts are not in the repository |
| Cutover prerequisites | OP-28 real path, OP-29 connector, OP-30 real environment, P1-8 | ≈ 6–10 PRs + operational work | ADR #18, E-3, P0-4 sign-off |

A **usable standalone CRM** needs all four tracks. It cannot be dated from the repository alone, because most of
the remaining work waits on decisions that are 0 / 61 approved today.

**What can be completed without decisions:** a fully tested, integrated, synthetic-data CRM covering:
- read-only operation for every domain;
- the four implemented write families (contact outcome, HR request steps, payment submission, fulfillment work).

## 10. Status

- Approvals: 0.
- Tracker unchanged (26 / 30 / 5).
- P0-3 OPEN, P0-5 PARTIAL, P0-6 OPEN, P0-7 PARTIAL / BLOCKED.
- No code, flag, permission or `plugin/current` change in this audit.
- **Production remains NO-GO.**
