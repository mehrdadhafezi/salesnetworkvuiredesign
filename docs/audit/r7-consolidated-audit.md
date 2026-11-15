# R7 Consolidated Audit — CRM completion and readiness assessment

**Mode:** strict read-only audit. No application code, schema, migration, permission, flag, grant or integration was changed, and nothing was deployed or merged.
**Audited baseline:** `main` @ `e2c4b76208daf4bd454651435f8240069771863c` (merge of PR #61).
**Previous audit:** R7-A (`docs/remediation/r7a-operational-implementation-audit.md`) at `f0c497b`.
**Persian executive summary:** [`r7-consolidated-audit-summary.fa.md`](r7-consolidated-audit-summary.fa.md).
**Production remains NO-GO.**

Labels used throughout:
- **VERIFIED**: established from code, tests or CI on this baseline.
- **NOT VERIFIED**: claimed somewhere, but not establishable from the repository.
- **UNKNOWN**: no evidence either way.

---

## 0. Headline findings

1. **R7 produced evidence, not features.** Between R7-A (`f0c497b`) and this baseline (49 commits):
   - The API contract still has 103 paths and 105 operations (92 GET, 13 POST). The only change is the 404 responses added by DEF-RP-01.
   - No SQL migration changed, and `web/src` changed only in the generated client.
   - Production Go code changed in exactly three files, all defect fixes: `internal/fulfillment/work/work.go`, `internal/organization/hrrequest/hrrequest.go` and `internal/reporting/http/http.go`.
   - Everything else was tests, test infrastructure, CI and documentation. **VERIFIED** (`git diff --stat f0c497b e2c4b76`).
2. **No user of the shipped product can perform any business action.**
   - Of 43 capabilities (`internal/identity/api/api.go`), only the HR role bundle carries any: 6 HR-read capabilities. The other 8 roles carry none.
   - A subject holds only ACTIVE grants, and none exist (D-42).
   - Every production-equivalent browser suite (R7-1, R7-2, R7-3, R7-6, R7-7 class A) proves **refusal**.
   - Every positive browser test runs on the test-only dev stub (`authorization.DevSubjects`, `CRM_DEV_AUTH=1`, refused in production).
3. **The CRM is a read-shadow of the legacy CRM plus four controlled write families**: contact outcome, HR request steps, payment submission, and fulfillment work with provider resolution.
   - It has no native creation of customers, leads, cases, invoices, complaints or tickets.
   - It has no financial execution (approve or reject, posting, refunds, commission credit).
   - It has no user provisioning.
   - It has no legacy write-back connector.
   - Legacy WordPress remains the only system of record and writer (P1-8).
4. **The frontend is a module-shaped technical shell, not the role-based product designed in `prototype/`.**
   - None of the 9 shared design-system components named in the role design specs exists in `web/src`. That set is RoleNavigation, QueueTabs, DimensionSwitcher, AttributionBasis, MetricCoverage, LinkConfidence/SourceBadge, ScopeBreadcrumb, FinancialStateLayers and OutcomeUnknownContext.
   - The home page is a 14-line placeholder (`web/src/routes/HomePage.tsx`).
   - Navigation is one flat list of feature-flagged module links, not the per-role entry required by the target architecture (`CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md` §app layout, "route tree per role entry").
5. **Governance:** the tracker (`docs/governance/r6-decision-tracker.json`) holds 61 decisions:
   - **0 approved**: 26 `READY_FOR_REVIEW`, 30 `EVIDENCE_MISSING`, 5 `DRAFT`;
   - `docs/governance/approvals/` does not exist, so no approval record exists.

---

## 1. Baseline verification

| Item | Evidence | Status |
|---|---|---|
| `main` SHA | `e2c4b76208daf4bd454651435f8240069771863c` | VERIFIED |
| CI on `e2c4b76` | Full-stack E2E success; Release artifact and staging rehearsal success; Governance docs success; Go CI in progress at audit time (green on the previous merge `f3291dd`) | VERIFIED; Go CI on `e2c4b76` NOT VERIFIED at audit time |
| API contract | `api/openapi/openapi.yaml`: 103 paths (101 under `/api/v1` + 2 health), 92 GET + 13 POST, 0 PUT, PATCH or DELETE | VERIFIED |
| Frontend | 132 files under `web/src`; 48 route entries in `web/src/app/routes.tsx` (incl. `dev/primitives`, `*`, `/`); 14 unit-test files with 119 tests (all against `web/src/test/fakeApi.ts`) | VERIFIED |
| Go tests | 593 `Test*` functions; the largest areas are integration 167, platform 113, kernel 53 and cmd 37 (full list in §6) | VERIFIED |
| Browser E2E | 7 Go orchestrators (build tag `e2e`); 7 Playwright spec files; 80 Playwright tests (reconciliation in §6.2) | VERIFIED |
| Database | 17 module migration sets; tables per schema: integration 50, payments 20, organization 16, biawin 15, fulfillment 15, commission 14, customers 14, cases 13, identity 12, reporting 12, finance 11, billing 9, custody 7, catalog 6, audit 1, platform_featureflags 1 | VERIFIED |
| Commands | `cmd/`: crm-api, crm-worker, crm-migrate, crm-legacy-import, crm-reporting, crm-finance-shadow, crm-commission-shadow, crm-biawin-shadow, crm-backup, crm-healthcheck | VERIFIED |
| Feature flags | About 20 runtime keys, all default OFF; `biawin.activate_production` is CONTROLLED and BLOCKED by a database CHECK | VERIFIED (key list from code) |

### 1.1 PR history #49–#62 (all merged)

| PR | Merge | Purpose | Production code? |
|---|---|---|---|
| #49 | `33470f3` | R7-1 browser E2E harness: login, case, contact outcome | No |
| #50 | `a1c2ac3` | M2 D-42 seller authorization analysis (docs) | No |
| #51 | `9b1a80b` | M2 authorization decision intake package (docs) | No |
| #52 | `62f30fc` | R7-2 invoice and payment submission E2E | No |
| #53 | `038699d` | R7-3 fulfillment and provider-resolution E2E | No |
| #54 | `3406843` | R7-4 fulfillment and HR request domain hardening tests | No |
| #55 | `3cd3b7e` | CI: wait for the first Prometheus scrape (staging rehearsal flake) | CI only |
| #56 | `7a793f4` | **DEF-HR-01**: reviewer chain must reach a root without the requester | **Yes** (`hrrequest.go`) |
| #57 | `c048617` | R7-5 deterministic integrated synthetic dataset | Test-only package (`legacytest`) |
| #58 | `fa8ad7a` | R7-6 HR read and request browser E2E | No |
| #59 | `72c3524` | **Work-intake race**: concurrent intake answers EXISTING_CASE and fails closed on an identity conflict | **Yes** (`work.go`) |
| #60 | `19b9ed7` | CI-PERF-01: pgtest templates, Go CI 19m41s → 11m39s | Test-only package (`pgtest`) |
| #62 | `f3291dd` | **DEF-RP-01**: unknown report `run_id` answers 404, not 500 | **Yes** (`reporting/http/http.go`, contract) |
| #61 | `e2c4b76` | R7-7 reporting, MIS and Biawin browser E2E | No |

---

## 2. Operational inventory OP-01 … OP-31 (re-audit)

**Class legend (as in R7-A):**
- A: implemented and tested.
- B: implemented, tested insufficiently.
- C: partial.
- D: contract, evaluator or mock only.
- E: missing.
- F: blocked by an unapproved decision.
- G: needs real-data or runtime evidence.

**Layer legend:**
- BE: backend.
- FE: frontend.
- E2E-A: browser on the production resolver.
- E2E-B: browser on the test-only overlay.
- RT: runtime-verified.
- PA: production-authorized.

**Business purpose and expected workflow:** as stated in R7-A §1a. They are not repeated here and are unchanged.

**Two columns are the same in every row, so they are stated once here:**
- **RT is ✘ for every OP.** There is no deployment and no real-data run.
- **PA is ✘ for every OP.** There are no ACTIVE grants (D-42) and 0 approved decisions.

| ID | Area | FE (route) | BE / API | DB | Capability · flag | Unit / integration | Browser E2E | Class (R7-A → now) | Missing | Blocking decisions | Next engineering task |
|---|---|---|---|---|---|---|---|---|---|---|---|
| OP-01 | Login and session | `shared/auth/LoginPage.tsx` (`/` when unauthenticated) | `POST /api/v1/auth/login`, `/logout`, `/logout-all` (`internal/identity/session`) | `identity.credential`, `identity.session` | flag `identity.session_auth` | identity 16 tests; `auth.test.tsx` | **E2E-A ✔** (R7-1, R7-6 and R7-7: wrong password, logout, cookie replay, absolute expiry) | B → **A** (behaviour) | Credentials exist only as test inserts; no provisioning (OP-03) | D-34, R5-SEC-01 | — (blocked; see RM-B01) |
| OP-02 | Capability enforcement | `shared/permissions/allowedActions.ts` (hints only) | `internal/authorization/guard.go` | `identity.role_grant` | 43 capabilities; only HR bundle has 6 | authorization 13; architecture 26 | **E2E-A ✔** (every suite: 403 `ACTION_DENIED`, no hidden action shown) | A (enforcement) | 37 of 43 capabilities in no role; no grant administration | D-42, E-7 | RM-B02 (role → capability design proposal, docs) |
| OP-03 | User and grant administration | — | — | — | — | — | — | E + F | Everything | D-34, D-42, E-7, R5-SEC-01 | Blocked |
| OP-04 | Organisation / workforce read | `modules/hr/pages/{Workforce,Person,Structure,Exceptions}Page.tsx` | `GET /api/v1/organization/*` | `organization.*` | `org.*.read` · `organization.hr_read_shadow` | organization 5 test fns (+ shadow suites) | **E2E-A refusal ✔; E2E-B render ✔** (R7-6) | B → **B+** (render proven test-only) | Production read authority; HR NOT LIVE VERIFIED | ADR #18, D-42 | RM-E05 (HR workforce UX against the HR spec) |
| OP-05 | HR transfer / termination requests | `modules/writes/pages/HRRequest(s)Page.tsx` | `POST /api/v1/hr/requests`, `/{id}/decisions` | `organization.hr_request*` | `org.hr_request.*` · `organization.hr_request_write(back)` | R7-4 hardening + DEF-HR-01 (default suite) | **E2E-A refusal ✔**; positive browser path **BLOCKED** (no overlay maps a browser identity to an org user) | C (unchanged) | Final apply; positive browser workflow | OPD-05, SD-04, R5-SEC-01, D-42 | Blocked (apply); RM-G03 test-only org-user overlay **needs approval** |
| OP-06 | Customer identity read | `modules/customers/pages/*` | `GET /api/v1/customers/*` | `customers.*` | `customers.*.read` · `customers.identity_shadow_read` | customers 14; `customers_e2e_test.go` | **none** | A (read) | Browser E2E; merge command | D-30, D-29/D-31, ADR #18 | **RM-G01** (customers browser E2E) |
| OP-07 | Native customer create / edit | — | — | — | — | — | — | E + F | Everything | P1-8, D-29/D-31 | Blocked |
| OP-08 | Lead intake and case creation | — | — | — | — | — | — | E + F | Everything | ADR-01, OPD-10, P1-8 | Blocked |
| OP-09 | Sales cases and custody read | `modules/sales/pages/*` | `GET /api/v1/cases*`, custody reads | `cases.*`, `custody.*` | `cases.case.read`, `custody.custody.read` · `sales.read_shadow` | cases 9, custody 12; `sales_e2e_test.go` | E2E-A refusal only (R7-1) | A (read) | Browser render of list, detail and custody (test-only overlay) | ADR #18 | **RM-G02** |
| OP-10 | Seller contact outcome | `modules/writes/ContactOutcomePanel.tsx` | `POST /api/v1/cases/{id}/contact-outcomes` | `cases.contact_outcome*`, command, outbox | `cases.contact_outcome.record` · `cases.nonfinancial_write` | `writes_http_e2e_test.go` | **E2E-A ✔** (R7-1): refusal, flag OFF, holder check, no write | B → **B+** | Positive browser write (no seller grant); legacy write-back | D-42, D-29/D-31, E-3 | Blocked (positive) |
| OP-11 | Claim / transfer / return / reset / release | — | Evaluators only (`internal/cases/claimgate`, `internal/custody/release`) | — | — | R4 race harness | — | D + F | Commands, routes, UI | ADR-01, F01, OPD-03, OPD-10, P1-8 | Blocked |
| OP-12 | Invoices, attempts, billing report | `modules/billing/pages/*` | `GET /api/v1/billing/*` | `billing.*`, `payments.*` | `billing.*.read`, `payments.attempt.read` · `billing_payments.read_shadow` | billing 14; `billing_e2e_test.go` | E2E-A refusal (R7-2) | A (read) | Browser render (overlay) | ADR #18 | **RM-G02** |
| OP-13 | Invoice issuance / next stage | — | Evaluator only (`internal/billing/issuance`) | — | — | issuance tests | — | D + F | Commands | ADR-01, F01, D-32, D-28 | Blocked |
| OP-14 | Receipt upload and payment batch | `modules/payments/PaymentSubmissionPanel.tsx`, `PaymentBatchPage`, `PaymentEvidencePage` | `POST /api/v1/payments/uploads`, `/invoices/{id}/batches` | `payments.upload_*`, `payment_batch`, `submitted_attempt*` | `payments.upload.stage`, `payments.batch.submit` · `payments.manual_write` | payments 10 | **E2E-A ✔** (R7-2): refusal, flags off, no Finance decision possible | B → **B+** | Positive browser submission; object-storage runtime; purge | D-42, D-44, P0-4 | Blocked (positive); RM-H02 (object-store runtime drill) |
| OP-15 | Finance review queue and decision | `modules/payments/pages/FinanceReview(Attempt)Page.tsx` (read) | read `GET /api/v1/finance/review*`; decision evaluator `internal/finance/review` | `payments.*` | `finance.review.read` · `finance.review_read` | finance 27 | none | A read · D + F decision | Approve / reject command and UI | OPD-08, D-43, REJECT-VOCAB, R5-SEC-01 | Blocked; RM-G02 read E2E |
| OP-16 | Shadow ledger and posting | `modules/finance/pages/*` | `GET /api/v1/finance/*`; `crm-finance-shadow`; posting validator | `finance.*` | `finance.shadow.read`, `finance.ledger.read`, `finance.reconciliation.read` · `finance.shadow_ledger_read` | finance 27 | none | A shadow · D + F posting | Posting; real-data parity | D-41, OPD-06, D-37, ADR #18 | RM-G02 read E2E |
| OP-17 | Refunds | — | State machine only (`internal/finance/refund`, every transition BLOCKED) | — | — | refund tests | — | D + F | Everything executable | OPD-04, R5-SEC-01, OPD-08, D-41, D-44 | Blocked |
| OP-18 | Commission shadow / my earnings / policies | `modules/commission/pages/*` | `GET /api/v1/commission/*`; `crm-commission-shadow` | `commission.*` | `commission.*.read` · `commission.read_shadow` | commission 23 | none | A shadow · E + F credit / payout | Credit and payout; browser E2E | OPD-09, D-36, D-41 | RM-G02 read E2E |
| OP-19 | Complaints / after-sales (pathway 1) | — | — (route enum `RouteComplaint` only) | — | — | — | — | E (+F) | Everything | OPD-02 (owner / ACK / SLA), OPD-04 (refund outcome); **no product contract in repo** | Product contract first (RM-F01) |
| OP-20 | Archive with reason (pathway 2) | — | — (route enum `RouteArchive` only) | — | — | — | — | E | Everything | OPD-01, OPD-02 | Product contract first (RM-F02) |
| OP-21 | Biawin subscriptions and parity | `modules/biawin/pages/*` | `GET /api/v1/biawin/*`; `crm-biawin-shadow` | `biawin.*` | `biawin.subscription.read`, `biawin.parity.read` · `biawin_fulfillment.read_shadow` | biawin 4; `biawin_e2e_test.go` | **E2E-A refusal ✔; E2E-B UI = API = DB ✔** (R7-7) | B → **A (read, test-only authority)** | Real data; production authority | ADR #18, D-42 | — |
| OP-22 | Plus subscription activation | `operations/pages/ActivationDecisionsPage.tsx` | `internal/biawin/activation` (SHADOW_COMPARE only; production BLOCKED by CHECK) | `biawin.activation*` | `biawin.activation.read` · `biawin.activate_production` (BLOCKED) | biawin tests | E2E-A (R7-3 decisions read / refusal) | C + F | Production activation | D-35a–e (C-03), D-38 | Blocked |
| OP-23 | Fulfillment work cases | `operations/pages/WorkQueue/WorkCasePage.tsx` | `POST /api/v1/fulfillment/work-cases/{id}/{assignment,operational-actions,evidence-records,outcome}` | `fulfillment.work_*` | `fulfillment.work.*` · `fulfillment.work` flags | R7-4 hardening; intake race fix (#59) | **E2E-A ✔** (R7-3: refusal, flags on, mixed and off; financial components refused) | B → **B+** | Positive browser lifecycle; real Biawin API contract | D-42, D-35, D-38 | Blocked (positive) |
| OP-24 | Provider operations and UNKNOWN resolution | `operations/pages/ProviderOperationsPage.tsx` | `POST /api/v1/integration/provider-operations/{id}/resolution` | `integration.provider_operation*` | `integration.provider_operation.*` | integration suites | **E2E-A ✔** (R7-3) | B → **B+** | Real provider contract | D-42 | — |
| OP-25 | Inter-organisational tickets | — | — | — | — | — | — | E | Everything; **no product contract** | Product contract | Product contract first (RM-F03) |
| OP-26 | Reporting, MIS, dashboards | `modules/reporting/pages/*` | `GET /api/v1/{mis,reports,reporting}/*`; `crm-reporting` | `reporting.*` | `reporting.*.read` · `reporting.read_shadow` | reporting 11; `reporting_e2e_test.go`; DEF-RP-01 test | **E2E-A refusal ✔; E2E-B UI = API = DB ✔** (R7-7) | A (read) | Trusted KPIs; role dashboards (see §4); export | OPD-07, OPD-01, D-23, D-24 | RM-E06 (role dashboards UX) |
| OP-27 | Background processing (worker) | — | `cmd/crm-worker` | `platform.outbox/inbox` | — | worker tests; R2 fault injection | — | B | Runtime on staging | P0-4, D-44 | RM-H01 |
| OP-28 | Legacy snapshot import and shadow model | — | `cmd/crm-legacy-import` (real `load` BLOCKED) | `integration.*` | — | integration 167; **R7-5 integrated dataset** | used by R7-6 and R7-7 | A (synthetic) · G (real) | Real snapshot path | ADR #18, E-4, E-5, D-23 | Blocked (real); RM-D01 spec-only |
| OP-29 | Legacy write-back | `modules/writes/pages/WritebackPage.tsx` (diagnostics) | `internal/integration/writeback` (synthetic adapter) | `integration.writeback*` | `integration.writeback.read` | integration tests | none | D + F | Production connector | E-3, P1-8 | Blocked |
| OP-30 | Deployment, backup, restore, monitoring | — | `deploy/staging`, `cmd/crm-backup`, `deploy/observability` | — | — | staging rehearsal (CI green; PR #55 fixed a scrape race) | — | B | Real environment | P0-4 | RM-H03 |
| OP-31 | UI shell, routes, design system | `app/*`, `shared/ui/*` | — | — | — | `App.test.tsx`, `contract.test.ts` | Shell exercised by every E2E suite | A (unit) → **C (product)** | Role-based IA and the shared design system of the frozen specs (§4) | — | **RM-E01…E04** |

**Read this matrix with one rule:** a passing class-A browser test for OP-05, OP-10, OP-14 or OP-23 proves the product **refuses** the action safely. It does not prove the action is operational.

---

## 3. Role-by-role functional coverage

**Columns:**
- **Can act today (prod resolver):** what a real logged-in user of that role can do on this baseline. Each answer is VERIFIED from the role bundles in `internal/identity/api/api.go` and the E2E-A suites.
- **Workflows only in legacy:** inferred from the role product specs and the absence of target commands.

| Role | Product spec | Target role bundle | Can act today (prod resolver) | Implemented target surfaces (read, if authorized) | Workflows only in legacy CRM | Key blockers |
|---|---|---|---|---|---|---|
| Seller | Seller Design V1 (FROZEN, `prototype/index.html`) | **empty** | Log in and log out only | Contact outcome on a held case (write family exists, flag-gated) | Lead queues, conversions, invoice creation and next stage, wallet, behaviour, extra-number request, repeat customer | D-42, ADR-01, OPD-10, P1-8 |
| Supervisor | `prototype/supervisor/SUPERVISOR-DESIGN-SPEC.md` | **empty** | Log in only | HR request create (refused today) | Team view, assign / return, conversions, invoices and exceptions, customer history, reports, wallet | D-42, F01, OPD-03, SD-01 |
| Senior supervisor | `SENIOR-SUPERVISOR-PRODUCT-SPEC.md`; senior design (READY FOR REVIEW) | **empty** | Log in only | HR request step decision (refused today) | Overview, performance explorer, distribution and return, ready view, invoices, case explorer | D-42, F01, SD-01, SD-04 |
| Sales manager | `SALES-MANAGER-PRODUCT-SPEC.md`; manager design (READY FOR REVIEW) | **empty** | Log in only | — | Operations and exceptions, distribution, return, performance, archive explorer, extra-number requests, HR requests, reports | D-42, M-G01/F01, M-G08, OPD-10 |
| Sales deputy | `SALES-DEPUTY-PRODUCT-SPEC.md`; deputy design (READY FOR REVIEW) | **empty** | Log in only | — | Leadership overview, exceptions (11 categories), conditional allocation, diagnostics | D-42, SD-G01/G05/G07 |
| MIS | `MIS-PRODUCT-SPEC.md`; MIS design (READY FOR REVIEW) | **empty** | Log in only | MIS overview, source quality, reconciliation, 4 reports (read) | Import and data quality workspace, lineage inspector, delivery, return safety, planning, maintenance | D-42, OPD-07, OPD-01, D-23 |
| HR | `HR-PRODUCT-SPEC.md`; HR design (READY FOR REVIEW) | **6 read capabilities** | Log in; reads still refused (no ACTIVE grant, D-42) | Workforce, person, structure, exceptions, requests | Onboarding and transfer wizard, termination and handover, final apply, structure and access, compensation, credentials | D-42, OPD-05, SD-04, R5-SEC-01; **HR runtime NOT LIVE VERIFIED** |
| Finance | `FINANCE-PRODUCT-SPEC.md`; finance design (READY FOR REVIEW) | **empty** | Log in only | Review queue (read), shadow ledger, balances, reconciliation | Approve / reject, refunds, posting, rules and runs, reports; **Finance runtime NOT LIVE VERIFIED** | OPD-08, D-43, REJECT-VOCAB, D-41, OPD-04, R5-SEC-01 |
| Invoice reviewer | `ROLE-ACCESS-MAP.fa.md` | **empty** | Log in only | Invoice Review API (no page, D-24) | — | D-24, D-42 |
| Customer success / after-sales | none found in repo | — | — | — | Complaint handling, after-sales intake | **No product contract** (OP-19) |

### 3.1 Cross-domain workflows

| Workflow | Executable in target today? | Evidence |
|---|---|---|
| Lead → case → contact → conversion → invoice → payment → finance review → commission | **No**: every stage after "contact outcome" is legacy-only or decision-gated | OP-08, OP-11, OP-13, OP-15, OP-18 |
| Distribution / return across hierarchy | **No**: evaluators only | OP-11 (`claimgate`, `release`) |
| HR transfer / termination to final apply | **No**: steps exist; final apply BLOCKED | OP-05 (OPD-05, SD-04) |
| Payment receipt → batch → finance decision → posting | **Partial**: submission exists; decision and posting are evaluator or contract only | OP-14, OP-15, OP-16 |
| Biawin activation → fulfillment work → provider operation | **Partial**: SHADOW activation, synthetic provider; financial components refused | OP-22, OP-23, OP-24 |
| Dissatisfied customer → after-sales → complaint → refund / other | **No**: no module or contract | OP-19, OP-17 |
| No-service → archive with reason | **No** | OP-20 |
| Interdepartmental tickets, escalation, SLA, notifications | **No**: no contract; the role specs mark escalation / ACK / SLA as product gaps (e.g. Deputy SD-G07) | OP-25 |

Handoffs, owners, SLAs and approval steps for the missing workflows are **not invented here**. The specs record them as open product gaps (OPD-02, SD-G07, M-G08, SD-01).

---

## 4. Frontend, UX and visual design audit

### 4.1 Measurable facts

| Check | Result | Evidence |
|---|---|---|
| Role-based entry / navigation | **Absent.** One flat list of about 40 links, each shown only when a feature flag is on | `web/src/app/routes.tsx` (nav array); target requires "route tree per role entry" (`CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md`, app layout) |
| Home / role dashboard | **Placeholder**: "no business data, no KPIs exist yet" | `web/src/routes/HomePage.tsx` (14 lines) |
| Shared design-system components named in the role specs (9) | **0 of 9** in `web/src` | grep over `web/src` |
| Design tokens | `web/src/shared/ui/tokens.css` defines 58 custom properties, against 231 in `prototype/styles.css` | count of `--*:` declarations |
| Shared primitives | 5 (`Button`, `StatusPill`, `Card`, `PageHeader`, `StateBlock`) | `web/src/shared/ui/primitives.tsx` |
| RTL and Persian | `lang="fa" dir="rtl"`; Persian labels throughout | `web/index.html`; module `labels.ts` files |
| Responsive rules | 13 `@media` rules across `web/src` CSS; no mobile bottom navigation (every role spec defines one) | grep |
| Empty, loading, error, denied, disabled states | **Present and consistent** via `shared/read/ReadStates.tsx`, with UNKNOWN never shown as zero | proven in E2E R7-6 and R7-7 |
| Drawers / dialogs | No drawer component; dialogs only inside the 4 write surfaces | grep `Drawer` = 0 |
| Data integration | Real API through the generated client; no mock data in pages (tests use `fakeApi.ts`) | `web/src/shared/api` |
| Help / tour engine (all role specs) | Absent | — |

### 4.2 Role screen matrix (spec destination → web)

Status values:
- **MISSING**: no page.
- **TECHNICAL READ**: a module page exists that shows related shadow data, but not the specified screen.
- **PARTIAL**: some of the specified behaviour exists.
- **BLOCKED**: the specified screen needs an unapproved decision.

| Role | Spec destinations | Status in `web/src` |
|---|---|---|
| Seller (FROZEN) | Leads queues (none / no-answer / callback / not-purchased / duplicate / invoiced / all) with the outcome drawer · invoices · conversions · wallet · behaviour · extra-number request · repeat customer (7) | Leads **PARTIAL**: no queues; only the outcome action exists (`ContactOutcomePanel` inside `sales/cases/:id`). Invoices **TECHNICAL READ** (`billing/invoices`). The other 5 destinations **MISSING** |
| Supervisor | Team view · assign and return · conversion · invoices and exceptions · customer history · HR requests · reports · wallet | HR requests **PARTIAL** (`hr/requests`). Customer history **TECHNICAL READ** (`customers`). Every other destination **MISSING** |
| Senior | Overview · performance explorer · distribution and return · ready team view · invoices and exceptions · HR · reports · case explorer · wallet | HR **PARTIAL**. Case explorer **TECHNICAL READ** (`sales/cases`). Reports **TECHNICAL READ** (`reports/*`, no scope picker, no DimensionSwitcher). Every other destination **MISSING** |
| Manager | Operations and exceptions · distribution · return · performance and reconciliation · invoices · archive explorer · extra-number requests · HR · reports · customer behaviour · wallet | HR **PARTIAL**. Reports **TECHNICAL READ**. Every other destination **MISSING** (archive and extra-number are also BLOCKED by OP-20 and OPD-10) |
| Deputy | Leadership overview · hierarchy and performance · exceptions (11) · conditional allocation · return · scoped invoices · HR context · archive / extra-number monitor · diagnostics · reports · wallet | Diagnostics **PARTIAL** (`mis/reconciliation` shows F02 and F03). Every other destination **MISSING** |
| MIS | Today · import and data quality · source rows / lineage · custody and delivery · reports and reconciliation · planning · quick delivery · maintenance · diagnostics | Today **TECHNICAL READ** (`/mis`). Reports and reconciliation **PARTIAL** (`/reports/*`, `/mis/reconciliation`, `/mis/source-quality`). Custody **TECHNICAL READ** (`sales/custody`). Import, planning, delivery and maintenance **MISSING** |
| HR | Workforce explorer · employment / structure timeline · onboarding and transfer · termination and handover · requests · structure and access · compensation · credentials (restricted) · exceptions and audit | Workforce **PARTIAL** (`/hr`, person page). Structure **TECHNICAL READ**. Exceptions **PARTIAL**. Requests **PARTIAL**. Onboarding, termination, access, compensation and credentials **MISSING** (several BLOCKED: OPD-05, D-34) |
| Finance | Review queue · reconciliation and refund · ledger · rules, runs and posting · reports · audit · configuration · maintenance | Review queue **TECHNICAL READ** (read-only, no decision). Ledger (with balances) and reconciliation **TECHNICAL READ** (shadow; refund part BLOCKED). Rules, runs and posting **BLOCKED**. Reports, audit, configuration and maintenance **MISSING** |

**Denominator:**
- The role specs define **72 primary destinations**: Seller 7, Supervisor 8, Senior 9, Manager 11, Deputy 11, MIS 9, HR 9, Finance 8 (counted from the IA lists above).
- Of these, **0 are implemented as specified**.
- 9 are PARTIAL (counting each role-destination once).
- 11 have only a TECHNICAL READ module page.
- The other 52 are MISSING or BLOCKED.

No visual-completion percentage is claimed beyond these counts.

### 4.3 What the existing pages do well (keep)

- **Honest uncertainty rendering.** UNKNOWN, NOT_EVALUATED, INCOMPLETE, CONFLICT, STALE and not-applicable states are never shown as zero (R7-7).
- **Read-only banners and state handling.** Denied, disabled and loading states are consistent.
- **Query contract in the URL.** Pinned run, filters and cursor live in the URL.
- **Correctness.** Server-side pagination and generated-client typing.

These are correct foundations for the role screens. They are not themselves the role screens.

---

## 5. Security and governance register

| Decision / item | Tracker status | Blocks | Notes |
|---|---|---|---|
| ADR #18 (real-data access) | READY_FOR_REVIEW | OP-28 real path; all real-data parity; HR / Finance live verification | intake files in `docs/governance/intake/m1` are BLANK templates |
| R5-SEC-01 (person identity, SoD) | READY_FOR_REVIEW | OP-03, OP-05 apply, OP-15, OP-17 | intake BLANK (`intake/m2`) |
| D-34 (provisioning) | READY_FOR_REVIEW | OP-01 real accounts, OP-03 | intake BLANK |
| D-42 (seller / role authorization) | READY_FOR_REVIEW | **every positive user action in production** | intake BLANK; analysis in `m2-d42-seller-authorization-fa.md` |
| D-35a–e | DRAFT | OP-22, financial parts of OP-23 | conflict C-03 OPEN — UNRESOLVED |
| D-38 | READY_FOR_REVIEW | OP-22, OP-23 financial | — |
| OPD-05, SD-04 | READY_FOR_REVIEW | OP-05 final apply | — |
| OPD-08, D-43, REJECT-VOCAB, D-25, D-44 | READY_FOR_REVIEW | OP-15 decisions, OP-14 purge | — |
| D-41, OPD-06 | EVIDENCE_MISSING | OP-16 posting, OP-18 credit | — |
| D-37 | READY_FOR_REVIEW | OP-16 | — |
| OPD-09, D-36 | EVIDENCE_MISSING | OP-18 credit and payout | — |
| OPD-04 | READY_FOR_REVIEW | OP-17, OP-19 refund outcome | — |
| ADR-01, OPD-03, OPD-10 | READY_FOR_REVIEW | OP-08, OP-11, OP-13 | — |
| F01, D-32, D-28, D-30 | EVIDENCE_MISSING | OP-11, OP-13, OP-06 merge | F01 needs ADR #18 |
| OPD-01, OPD-07 | EVIDENCE_MISSING | OP-20, trusted OP-26 | — |
| OPD-02, D-23, D-24 | READY_FOR_REVIEW | OP-19 owner / ACK, export, Invoice Review page | — |
| D-03 … D-22 | EVIDENCE_MISSING | authority of the earlier decision set | conflict C-01 OPEN |
| E-3, P1-8 (engineering + owner) | not tracker records | OP-29; any authoritative target write | single-writer constraint: legacy remains the writer |
| P0-4 (environment sign-off) | not a tracker record | OP-30 real environment, runtime | column B unsigned (R7-A) |

**Rule restated from the tracker:** status changes only through an approval record under `docs/governance/approvals/` plus human verification. That directory does not exist, so **0 decisions are approved**. Code, tests, documents and merged PRs change no status.

---

## 6. Testing and quality evidence

### 6.1 Evidence classes

| Class | What exists | Proves |
|---|---|---|
| Unit (Go) | 593 `Test*` functions across modules | Domain rules, evaluators, contracts |
| Unit (web) | 119 tests, 14 files, against `fakeApi.ts` | Rendering logic; **not** the real API |
| Backend integration (PostgreSQL) | `internal/integration/legacy/shadow/*_e2e_test.go`, `internal/integration/biawin/*`; CI-PERF-01 templates | HTTP → DB behaviour per module |
| Synthetic dataset | R7-5 integrated dataset (`legacytest.IntegratedScenario`), determinism across two independent databases | A coherent synthetic world; **not** real-data parity |
| Browser E2E, production resolver (E2E-A) | R7-1, R7-2, R7-3, R7-6 class A, R7-7 class A | **Refusal**, flags, CSRF, session lifecycle, no mutation |
| Browser E2E, test-only overlay (E2E-B) | R7-6 class B, R7-7 class B | What pages render (UI = API = DB). **Not** production authorization |
| Runtime validation | Disposable staging rehearsal in CI (`release-ci.yml`) | Packaging and boot; **not** an operated environment |
| Real-data parity | none | — (ADR #18) |

### 6.2 Playwright inventory reconciliation (80 tests)

| Spec | Playwright project(s) | Tests | Class |
|---|---|---|---|
| `seller-contact-outcome.spec.ts` | write-flag-on 6 · write-flag-off 1 | 7 | A |
| `payments.spec.ts` | payments-flags-on 8 · off 3 | 11 | A |
| `fulfillment.spec.ts` | flags-on 16 · mixed 3 · off 4 | 23 | A |
| `hr.spec.ts` | hr-flags-on 8 · off 1 | 9 | A |
| `hr-read-overlay.spec.ts` | hr-read-overlay | 7 | B |
| `reporting-biawin.spec.ts` | flags-on 5 · off 1 | 6 | A |
| `reporting-biawin-overlay.spec.ts` | read-overlay (13 declared + 1 loop declaring 4 report tests) | 17 | B |
| **Total** | 13 projects, 7 orchestrators | **80** (56 class A, 24 class B) | |

The count matches the local run on `241ee02`: 6+1+16+3+4+8+1+7+8+3+5+1+17 = 80, all passed, and the same suite was green in CI on `e2c4b76`.

**Coverage gaps (VERIFIED absent):**
- No browser E2E for customers (OP-06), sales list, detail and custody render (OP-09), billing render (OP-12), finance shadow / review render (OP-15, OP-16), commission (OP-18), or write-back diagnostics (OP-29).
- No positive browser write for any write family under production authorization. This is impossible today (D-42).
- No positive browser HR workflow: BLOCKED, because no overlay maps a browser identity to an org user.
- No visual regression, accessibility (axe) or responsive-viewport browser tests.

### 6.3 Defect register (risk-ranked)

| ID | Severity | Status | Description | Evidence |
|---|---|---|---|---|
| DEF-HR-01 | High (correctness of HR authority) | **FIXED** #56 | Reviewer chain could cycle or include the requester | `hrrequest.go` `nextReviewer`; `r7_4_def_hr_01_test.go` |
| Work-intake race | High (500 under concurrent redelivery) | **FIXED** #59 | Concurrent intake raised 23505; identity conflict now fails closed | `work.go`; `work_intake_race_test.go` |
| DEF-RP-01 | Medium (500 on unknown run, fails closed) | **FIXED** #62 | Report list unknown `run_id` → 404 | `reporting/http/http.go`; `reporting_unknown_run_test.go`; R7-7 overlay |
| CI-STG-01 | Low (CI flake) | **FIXED** #55 | Prometheus first-scrape race in staging rehearsal | `release-ci.yml` |
| UX-01 | Low | OPEN | Biawin pages say "no import run recorded yet" for an unknown pinned `run_id` (wording, not data) | `biawin/components.tsx` `NoRun` |
| UX-02 | Low | OPEN | A report 404 renders the generic problem title ("Resource not found") with no recovery link | `shared/read/ReadStates.tsx` |
| RISK-01 | Medium | OPEN (risk) | pgtest template key does not cover the migration-runner version; a runner change needs a `templateFormat` bump | `internal/platform/pgtest/template.go` |
| RISK-02 | Low | OPEN (risk) | R7-5 subtests are ordered; mitigated by pristine preconditions | `r7_5_dataset_checks_test.go` |
| RISK-03 | Medium | OPEN (risk) | E2E session-expiry tests add about 1 min each (R7-6, R7-7); the E2E job grows with each suite | `e2e-ci.yml` |
| RISK-04 | High (product) | OPEN | Frontend IA diverges from the frozen role designs; continuing to add module pages widens the gap | §4 |

**Unresolved functional defects:** none known in the implemented surfaces beyond UX-01 and UX-02.

---

## 7. Completion roadmap

Notes for the whole roadmap:
- Priority: **P0** blocks product usefulness; **P1** needed for a complete CRM; **P2** quality and hardening.
- Complexity: S / M / L / XL.
- **Start:** NOW = no management decision needed (synthetic data, no new capability, no activation); BLOCKED = needs the listed approved decision.
- Every NOW task keeps flags OFF outside disposable test databases, adds no capability or grant, and deploys nothing.
- **Acceptance criteria** for each task are in §7.1.

### A. Critical missing business functionality

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-A01 | Native lead intake and case creation | Target can start a sale | P0 | RM-D02 | ADR-01, OPD-10, P1-8 | command, API, UI, tests | XL | BLOCKED |
| RM-A02 | Claim / return / reset / release commands | Distribution in target | P0 | RM-A01 | ADR-01, F01, OPD-03, OPD-10 | wire existing evaluators into commands | L | BLOCKED |
| RM-A03 | Invoice issue / next stage | Billing in target | P0 | RM-A02 | ADR-01, F01, D-32, D-28 | wire `billing/issuance` | L | BLOCKED |
| RM-A04 | Finance approve / reject | Finance can decide | P0 | RM-B03 | OPD-08, D-43, REJECT-VOCAB, R5-SEC-01 | command + UI (FinancialStateLayers) | L | BLOCKED |
| RM-A05 | Posting | Ledger of record | P1 | RM-A04 | D-41, OPD-06, D-37, ADR #18 parity | wire `finance/posting` | L | BLOCKED |
| RM-A06 | Refunds | After-sales money | P1 | RM-A04 | OPD-04, R5-SEC-01, D-41, D-44 | wire `finance/refund` | L | BLOCKED |
| RM-A07 | Commission credit / payout | Seller earnings of record | P1 | RM-A05 | OPD-09, D-36, D-41 | command + reconciliation | L | BLOCKED |
| RM-A08 | HR final apply | HR changes take effect | P1 | RM-B03 | OPD-05, SD-04, R5-SEC-01 | apply with named successor | M | BLOCKED |
| RM-A09 | Customer create / edit; merge | Customer master in target | P1 | RM-D02 | P1-8, D-29/D-31, D-30 | write model | L | BLOCKED |
| RM-A10 | Production subscription activation | Biawin plus | P2 | — | D-35a–e (C-03), D-38 | activation command | L | BLOCKED |

### B. Security and role authorization

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-B01 | Account provisioning and credential lifecycle | Real users can log in | P0 | — | D-34, R5-SEC-01 | provisioning command + audit | L | BLOCKED |
| RM-B02 | Role → capability bundle **proposal** (doc only; mapping each frozen role spec destination to existing capabilities) | Unblocks D-42 review with evidence | P0 | — | none (proposal only; changes no bundle) | `docs/` proposal + gap list | M | **NOW** |
| RM-B03 | Apply approved role bundles and grant administration | Users can act | P0 | RM-B01, RM-B02 | D-42, E-7 | bundles + grant admin + audit | L | BLOCKED |
| RM-B04 | Person-level separation of duties | Maker ≠ checker | P0 | RM-B01 | R5-SEC-01 | identity model | L | BLOCKED |

### C. Governance-dependent engineering

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-C01 | Decision-to-engineering readiness pack per M1–M6: for each READY_FOR_REVIEW decision, the exact code and tests that would change | Speeds approval → delivery | P1 | — | none (docs) | `docs/remediation/` | M | **NOW** |
| RM-C02 | Implement each approved decision, one PR per command family | — | P0/P1 | approvals | per decision | per R6 follow-up map | — | BLOCKED |

### D. Database and integration readiness

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-D01 | E-5 extract specifications (spec only, no extraction) | Prepares real import | P1 | — | none for specs | `cmd/crm-legacy-import` specs + docs | M | **NOW** |
| RM-D02 | Single-writer cutover design (P1-8) and write-back connector (E-3) | Target writes become authoritative | P0 | — | E-3, P1-8, owner | connector + reconciliation | XL | BLOCKED |
| RM-D03 | Real snapshot path and parity register | Real-data evidence | P0 | — | ADR #18, E-4, D-23 | load + reconcile | L | BLOCKED |
| RM-D04 | Real Biawin / provider API contract | Non-synthetic fulfillment | P1 | — | product + provider | contract + adapter | L | BLOCKED (external) |

### E. Frontend and UX completion

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-E01 | Port the frozen shared design system (tokens, RoleNavigation, QueueTabs, drawer, mobile bottom nav, Help engine) from `prototype/` into `web/src/shared/ui` | Foundation for every role screen | P0 | — | none (FROZEN Seller design + SN-207) | components + Storybook-like dev page + unit tests | L | **NOW** |
| RM-E02 | Role-entry routing: one route tree per role. Navigation is chosen from the session's capability hints, never from a client-side role guess | Matches the target architecture | P0 | RM-E01 | none (presentation only) | `web/src/app` routes and shell | M | **NOW** |
| RM-E03 | Seller panel read screens on existing APIs (leads queues as read views of cases, invoices, case drawer with the existing contact-outcome panel) | First role screen | P0 | RM-E01, RM-E02 | Seller design FROZEN | pages + E2E-B | L | **NOW** (read); actions stay refused |
| RM-E04 | Shared read components from the role specs: MetricCoverage, LinkConfidence / SourceBadge, ScopeBreadcrumb, DimensionSwitcher, AttributionBasis | Correct presentation of uncertainty | P1 | RM-E01 | role designs are READY FOR REVIEW (freeze NOT VERIFIED) | components | M | **NEEDS DESIGN FREEZE** |
| RM-E05 | HR role screens on existing reads (workforce explorer, structure timeline, exceptions, requests) | HR usability | P1 | RM-E04 | HR design freeze | pages + E2E-B | L | **NEEDS DESIGN FREEZE** |
| RM-E06 | MIS / Senior / Manager / Deputy read dashboards on existing reporting APIs | Management visibility | P1 | RM-E04 | role design freeze; KPI trust (OPD-07) for any new KPI | pages | L | **NEEDS DESIGN FREEZE** (no new KPI) |
| RM-E07 | Finance read screens on existing reads (review queue, ledger, reconciliation) with FinancialStateLayers | Finance usability (read) | P1 | RM-E04 | finance design freeze | pages | M | **NEEDS DESIGN FREEZE** |
| RM-E08 | UX-01 / UX-02 copy and recovery fixes | Clarity | P2 | — | none | copy + link | S | **NOW** |
| RM-E09 | Responsive and accessibility browser checks (5 widths, axe) | Quality | P2 | RM-E01 | none | Playwright projects | M | **NOW** |

### F. Cross-role workflows and interdepartmental operations

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-F01 | Complaint / after-sales **product contract** | Enables OP-19 | P0 | — | OPD-02, OPD-04 | product spec (Product / Codex) | M | BLOCKED (owner) |
| RM-F02 | Archive-with-reason product contract | Enables OP-20 | P1 | — | OPD-01, OPD-02 | spec | S | BLOCKED (owner) |
| RM-F03 | Interdepartmental ticket product contract | Enables OP-25 | P1 | — | product owner | spec | M | BLOCKED (owner) |
| RM-F04 | Escalation / ACK / SLA / notification model | Cross-role handoffs | P1 | RM-F01 | OPD-02, SD-G07 | spec then engineering | L | BLOCKED |

### G. Testing and operational quality

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-G01 | Customers browser E2E (A + B) | Closes the OP-06 gap | P1 | — | none | `web/e2e` | S | **NOW** |
| RM-G02 | Sales, billing, finance-shadow and commission read browser E2E (A + B) | Closes the OP-09/12/15/16/18 gaps | P1 | — | none | `web/e2e` | M | **NOW** |
| RM-G03 | Test-only overlay mapping a dev identity to a synthetic org user (positive HR browser flow) | Positive HR evidence | P2 | — | **explicit approval** (a new test-only overlay) | test harness | M | NEEDS APPROVAL |
| RM-G04 | Frontend contract tests against the real OpenAPI responses (replace part of `fakeApi.ts`) | Catches drift | P2 | — | none | vitest | M | **NOW** |
| RM-G05 | E2E runtime budget (parallel orchestrators or a shared expiry test) | Keeps CI fast | P2 | — | none | CI | S | **NOW** |

### H. Staging, release readiness and cutover

| ID | Task | Impact | P | Depends on | Governance prerequisite | Scope / deliverable | Cx | Start |
|---|---|---|---|---|---|---|---|---|
| RM-H01 | Worker runtime drill on disposable staging | OP-27 runtime evidence | P1 | — | none (disposable) | drill script + report | S | **NOW** |
| RM-H02 | Object-storage upload runtime drill on disposable staging | OP-14 runtime evidence | P1 | — | none (disposable) | drill | S | **NOW** |
| RM-H03 | Real environment and P0-4 sign-off | Production prerequisite | P0 | RM-B01…B04, RM-D02, RM-D03 | P0-4 owner | operations | L | BLOCKED |
| RM-H04 | HR and Finance live role verification | Clears NOT LIVE VERIFIED | P0 | RM-B03, RM-H03 | HR / Finance accounts | runtime test | M | BLOCKED |

### 7.1 Acceptance criteria (selected)

- **RM-E01:**
  - every token in `prototype/styles.css` used by the frozen Seller design exists in `web/src/shared/ui/tokens.css`;
  - each component has a unit test and a dev page;
  - contrast is ≥ 4.5:1 in all themes;
  - no business logic in components.
- **RM-E02:**
  - navigation per role entry;
  - a destination appears only when the server's `allowed_actions` or capability hints allow it;
  - with no grants, the user sees an honest "no access" home, never an empty dashboard;
  - all existing E2E suites stay green.
- **RM-E03:**
  - the Seller destinations of the frozen design render from existing read APIs on the R7-5 dataset;
  - E2E-B proves UI = API = DB;
  - E2E-A proves refusal;
  - no new endpoint or capability.
- **RM-G01 / RM-G02:** the same two-class pattern as R7-6 and R7-7: UI = API = DB on the overlay, exact refusals on the production resolver, and no mutation.
- **RM-B02:**
  - each role destination is mapped to existing capability keys, with the spec reference;
  - gaps are listed;
  - no bundle is changed;
  - the document is labelled a proposal for D-42.

### 7.2 Dependency-aware sequence and critical path

1. **NOW (no decision needed):**
   - RM-E01 → RM-E02 → RM-E03;
   - in parallel: RM-B02, RM-C01, RM-G01, RM-G02, RM-D01, RM-H01, RM-H02, RM-E08, RM-G04, RM-G05;
   - after the role design freeze: RM-E04 → RM-E05, RM-E06, RM-E07.
2. **After M2 (D-34, D-42, R5-SEC-01):** RM-B01 → RM-B04 → RM-B03. Users can then act within the four existing write families: RM-H04 (partial) and RM-A08 (with OPD-05 / SD-04).
3. **After M1 (ADR #18):** RM-D03, then real-data parity for every read module.
4. **After M3 (ADR-01, F01, OPD-03, OPD-10, D-32, D-28) and P1-8 / E-3:** RM-D02 → RM-A01 → RM-A02 → RM-A03.
5. **After M4-A, M4-B, M4-C and M4-D:** RM-A04 → RM-A05 → RM-A07; then RM-A06.
6. **After product contracts (OPD-01, OPD-02, OPD-04):** RM-F01 → RM-F04; RM-F02; RM-F03.
7. **After M6:** RM-A10.
8. **Release:** RM-H03, then cutover.

**Critical path to a usable CRM:**
- M2 decisions → provisioning and role bundles (RM-B01, B03, B04).
- M3 decisions plus the single-writer cutover (RM-D02) → native intake, distribution and invoicing (RM-A01…A03).
- M4 decisions → finance execution (RM-A04, A05).
- The role-based frontend (RM-E01…E07) runs in parallel and is **not** on the decision path. It is the largest engineering stream that can start today.

---

## 8. Recommended next PR sequence (no decision needed)

| # | PR | Objective | Size |
|---|---|---|---|
| 1 | RM-E01 | Shared design system port from the frozen prototype (tokens, RoleNavigation, QueueTabs, drawer, bottom nav) | L |
| 2 | RM-E02 | Role-entry routing and capability-driven navigation (presentation only) | M |
| 3 | RM-B02 | Role → capability bundle proposal for D-42 (docs) | M |
| 4 | RM-G01 | Customers browser E2E (A + B) | S |
| 5 | RM-E03 | Seller panel read screens (frozen design) + E2E | L |
| 6 | RM-G02 | Sales, billing, finance-shadow and commission read E2E | M |
| 7 | RM-H01 / RM-H02 | Worker and object-storage drills on disposable staging | S each |
| 8 | RM-E08 | UX-01 / UX-02 copy fixes | S |

Role screens beyond the Seller (RM-E04…E07) wait for an explicit design freeze of each role spec. Each role spec currently labels itself **READY FOR REVIEW**, and a freeze record was not found (NOT VERIFIED).

---

## 9. Final status

- **Audited SHA:** `e2c4b76208daf4bd454651435f8240069771863c`.
- **Confirmed state:** a tested, synthetic-data, read-shadow CRM with four controlled write families. Every action is refused under the production resolver. The frontend is a technical module shell.
- **Not changed by this audit:** no code, schema, flag, permission, grant, integration or deployment.
- **Production remains NO-GO.** The roadmap is a proposal. It is not to be started until it has been independently reviewed and explicitly authorized.
