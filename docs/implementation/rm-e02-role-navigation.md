# RM-E02 — Role-oriented CRM navigation and entry (fail-closed)

| | |
|---|---|
| Baseline | `main` @ `95c3f74b466d46f64330b58db233d208e213e4d8` (merge of PR #64, RM-E01) |
| Branch | `feat/rm-e02-role-navigation` |
| Mode | Controlled frontend implementation, presentation only |
| Backend, OpenAPI, database, migrations, role bundles, grants, feature flags | **Unchanged** |
| Production | **NO-GO** (unchanged). Role-based navigation does not make the CRM production-ready |

## 1. Outcome in one paragraph

The flat list of about 40 links, each shown when a feature flag was on, is gone. It is replaced by a role-oriented shell built on the RM-E01 components (`DsAppShell`, `AppHeader`, `RoleNavigation` with priority+ overflow, `MobileBottomNavigation`, `PageHead`, `Panel`, `StatePanel`, `Banner`, `TonePill`, `Menu`).

A business destination now appears only when **both** of these hold:

1. the server session reports the destination's feature switch on;
2. the server's effective-access evaluation (`GET /api/v1/authorization/effective-access`) returns an **ALLOWED / SUCCESS** decision for its capability.

A feature flag alone never shows a link. Today no role bundle except HR carries a capability, and no ACTIVE grant exists (D-42). So every real user lands on an honest Persian no-access state, with only the two safe destinations: «خانه» and «وضعیت سامانه».

All 46 existing routes, their URLs and their deep links work unchanged. Every page still renders the server's own answer. The client never guards a route, and a page the navigation hides still answers with the server's refusal.

## 2. Server contract analysis (what the SPA may read)

| Need | Server field | Authority? | Used by RM-E02 |
|---|---|---|---|
| **A. Authentication state** | `GET /api/v1/session` → `authenticated`, `auth_mode`, `session_state`, `session` (identity only) | Authoritative for *who*, never for *what* | Yes: anonymous visitors get no business link and trigger no evaluation call |
| **B. Feature availability** | `session.features.*` (14 booleans; "navigation only, never authority") | Not authority | Yes, but only as a **second** condition, never alone |
| **C. Role / organizational context** | **None in the session.** `effective-access.decisions[].matched_roles` names the role (or synthetic basis) that matched an ALLOWED decision | Evidence of the grant that matched, not a role context | Presentation only: header crumb and ordering when exactly one known role matched; never a permission |
| **D. Authorised actions / capabilities** | `GET /api/v1/authorization/effective-access` → `decisions[]` (`capability`, `allowed`, `outcome`, `scope`, `matched_roles`): the same evaluator as `allowed_actions` | **Server-authoritative** | Yes: the sole source of "may see this destination" |
| (rejected) | `session.capability_hints` (`dev.*`, synthetic, "never authoritative permissions") | No | **Never read**. Test 13 sets hints and asserts that nothing appears |

Facts verified in code (not assumed):

- **Resolved by RM-ACCESS-01** (`docs/remediation/rm-access-01-authorization-contract.md`): each capability is now gated by its own guard flag. Original finding: `internal/authorization/http/http.go` `GetEffectiveAccess` sets `Allowed: d.Allowed && enabled`, where `enabled` is the **`organization.hr_read_shadow`** flag. When that flag is off, every decision comes back `BLOCKED / FEATURE_DISABLED`, including non-HR capabilities. RM-E02 renders this as the explicit "evaluation unavailable" state. It does not fall back to flags.
- **Resolved by RM-ACCESS-01**: the enum now lists all 43. Original finding: `internal/identity/api` holds 43 capabilities, but the OpenAPI `Capability` enum lists only **29**. The server returns all 43 at runtime. The 14 missing from the contract (commission, fulfillment work, activation, provider operations, payments upload/batch/submission/evidence, finance review) are **UNKNOWN** to the SPA: their 9 destinations never appear in navigation, though they still work by deep link.
- Role bundles: only `hr` carries capabilities (6 HR reads). The other eight roles are empty. A subject holds only ACTIVE grants, and none exist (D-42).
- `DevSubjects` (dev stub, refused in production) grants `dev-user-001` the HR role plus a direct synthetic capability set, which matches as `synthetic_dev_stub`. RM-E02 treats that as an **UNKNOWN** role context and names no role.

## 3. Existing versus proposed structure

| | Before (main @ 95c3f74) | After (this PR) |
|---|---|---|
| Source of links | `routes.tsx` `nav[]`: 39 items, each `feature?` | `navigation/destinations.ts`: 37 business destinations plus 2 safe ones, typed |
| Rule | `!item.feature \|\| features[item.feature] === true` | `features[d.feature] === true` **and** effective-access ALLOWED **and** capability in the OpenAPI enum |
| Presentation | Plain `<nav>` list, a «منو» toggle on mobile | `RoleNavigation` (priority+ overflow into «بیشتر», current item never hidden) plus `MobileBottomNavigation` (4 primary + «بیشتر») |
| Home | Placeholder «هنوز بخشی فعال نیست» | Role entry: an explicit state for each fail-closed case, or the exact allowed set grouped by role or module; no metrics |
| Header | Brand, identity, logout | `AppHeader`: brand, role crumb («نقش: …» only for a single verified role), identity, logout, page title on phones |
| Routes / URLs | 46 child routes | **Identical** (only the nav array was removed from `routes.tsx`) |

Order: «خانه» → role-mapped destinations in spec order (single role) → other allowed destinations in registry order → «وضعیت سامانه» → DEV tools (DEV builds only, suffixed «· DEV»).

## 4. Destination → feature → capability matrix

Generated from `web/src/app/navigation/destinations.ts`. Capabilities come from each list endpoint's OpenAPI `Requires …` text.

| # | Route | Page label | Kind | Feature (`session.features`) | Server capability | Contract status | Shown in navigation when |
|---|---|---|---|---|---|---|---|
| 1 | `/hr` | نیروها | READ | `hr_read` | `org.person.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 2 | `/hr/structure` | ساختار سازمانی | READ | `hr_read` | `org.unit.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 3 | `/hr/exceptions` | استثناهای سلسله‌مراتب | READ | `hr_read` | `org.assignment.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 4 | `/hr/requests` | درخواست‌های منابع انسانی | CONTROLLED_WRITE | `hr_request_write` | `org.hr_request.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 5 | `/customers` | مشتریان | READ | `customers_identity` | `customers.customer.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 6 | `/customers/merge-cases` | پرونده‌های تطبیق | READ | `customers_identity` | `customers.merge_case.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 7 | `/customers/identity-report` | گزارش تطبیق هویت | READ | `customers_identity` | `customers.identity_report.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 8 | `/sales/cases` | پرونده‌های فروش | READ | `sales_read` | `cases.case.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 9 | `/sales/custody` | نگهداری پرونده‌ها | READ | `sales_read` | `custody.custody.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 10 | `/sales/report` | گزارش پوشش فروش | READ | `sales_read` | `sales.report.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 11 | `/billing/invoices` | فاکتورها | READ | `billing_read` | `billing.invoice.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 12 | `/billing/attempts` | تلاش‌های پرداخت | READ | `billing_read` | `payments.attempt.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 13 | `/billing/report` | گزارش تطبیق صورت‌حساب | READ | `billing_read` | `billing.reconciliation.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 14 | `/mis` | امروز MIS | READ | `reporting_read` | `reporting.mis.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 15 | `/mis/source-quality` | کیفیت منبع | READ | `reporting_read` | `reporting.mis.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 16 | `/mis/reconciliation` | تطبیق و پوشش | READ | `reporting_read` | `reporting.mis.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 17 | `/reports/cases` | گزارش پرونده‌ها | READ | `reporting_read` | `reporting.metrics.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 18 | `/reports/custody` | گزارش نگهداری | READ | `reporting_read` | `reporting.metrics.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 19 | `/reports/invoices` | گزارش فاکتورها | READ | `reporting_read` | `reporting.metrics.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 20 | `/reports/payments` | گزارش پرداخت‌ها | READ | `reporting_read` | `reporting.metrics.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 21 | `/finance` | دفتر سایه مالی | READ | `finance_shadow_read` | `finance.shadow.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 22 | `/finance/ledger` | دفتر روزنامه سایه | READ | `finance_shadow_read` | `finance.ledger.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 23 | `/finance/balances` | مانده‌های سایه | READ | `finance_shadow_read` | `finance.shadow.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 24 | `/finance/reconciliation` | تطبیق سایه | READ | `finance_shadow_read` | `finance.reconciliation.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 25 | `/biawin/subscriptions` | اشتراک‌های بیاوین | READ | `biawin_fulfillment_read` | `biawin.subscription.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 26 | `/biawin/parity` | تطبیق اشتراک‌های فعال | READ | `biawin_fulfillment_read` | `biawin.parity.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 27 | `/fulfillment/cases` | پرونده‌های اجرای خدمت | READ | `biawin_fulfillment_read` | `fulfillment.case.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 28 | `/commission` | پورسانت · نمای سایه | READ | `commission_read` | `commission.shadow.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 29 | `/commission/calculations` | محاسبات سایه پورسانت | READ | `commission_read` | `commission.shadow.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 30 | `/commission/policies` | نسخه‌های سیاست پورسانت | READ | `commission_read` | `commission.policy.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 31 | `/commission/me` | پورسانت‌های من | READ | `commission_read` | `commission.own.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 32 | `/operations/work` | صف کار اجرای خدمت | CONTROLLED_WRITE | `fulfillment_work` | `fulfillment.work.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 33 | `/operations/activations` | تصمیم‌های فعال‌سازی اشتراک | READ | `fulfillment_work` | `biawin.activation.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 34 | `/operations/provider-operations` | عملیات ارائه‌دهنده خارجی | CONTROLLED_WRITE | `fulfillment_work` | `integration.provider_operation.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 35 | `/writeback` | بازنویسی به سامانه قدیمی | DIAGNOSTIC | `writeback_read` | `integration.writeback.read` | CONTRACT | feature ON **and** capability ALLOWED |
| 36 | `/payments/evidence` | شواهد پرداخت | DIAGNOSTIC | `payment_submission` | `payments.evidence.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |
| 37 | `/finance/review` | صف بررسی مالی (فقط خواندنی) | READ | `finance_review_read` | `finance.review.read` | CONTRACT (was UNKNOWN; resolved by RM-ACCESS-01) | feature ON **and** capability ALLOWED |

28 of 37 destinations had a contract capability; 9 were UNKNOWN. **RM-ACCESS-01:** all 37 are CONTRACT now (the enum equals the backend registry).

Safe destinations, always shown: `/` «خانه» (role entry, no business data) and `/system` «وضعیت سامانه» (technical health only). Logout stays in the header, unchanged.

**Not functional yet.** Every destination above is an existing *technical* read or controlled-write page (R7 §4: TECHNICAL READ / PARTIAL). None of them is the role screen its spec describes. Under today's grants none is reachable with data in production: each answers `UNAUTHORIZED / ACTION_DENIED`.

## 5. Role → destination matrix

Sources: `web/src/app/navigation/roles.ts` and R7 §4 (`docs/audit/r7-consolidated-audit.md`). Section names are English contract identifiers. Persian labels for screens that do not exist yet belong to each role's design freeze, so they are deferred and never rendered. Only sections mapped to an existing route can ever become a link.

| Role (server key) | Design status | Existing destinations mapped (spec order) | Spec sections with no screen |
|---|---|---|---|
| Seller (`seller`) | **FROZEN** (Seller Design V1) | leads → `/sales/cases` (PARTIAL) · invoices → `/billing/invoices` (TECHNICAL READ) | conversions, repeat action, customer behaviour, wallet: MISSING. Extra-number request: MISSING (OPD-10) |
| Supervisor (`supervisor`) | **FREEZE_UNVERIFIED**: its own spec says "DESIGN PROTOTYPE"; only `SENIOR-SUPERVISOR-PRODUCT-SPEC.md` calls it frozen | customer history → `/customers` · HR requests → `/hr/requests` | team view, assign/return, conversion, invoices & exceptions, reports, wallet: MISSING |
| Senior supervisor (`senior_supervisor`) | READY_FOR_REVIEW | HR → `/hr/requests` · reports → `/reports/{cases,custody,invoices,payments}` · case explorer → `/sales/cases` | overview, performance explorer, distribution & return, ready team view, invoices & exceptions, wallet: MISSING |
| Sales manager (`sales_manager`) | READY_FOR_REVIEW | HR → `/hr/requests` · reports → `/reports/*` | operations & exceptions, distribution, return, performance, invoices, customer behaviour, wallet: MISSING. Archive explorer (OP-20) and extra-number requests (OPD-10): BLOCKED |
| Sales deputy (`sales_deputy`) | READY_FOR_REVIEW | diagnostics → `/mis/reconciliation` (PARTIAL) | ten sections: MISSING |
| MIS (`mis`) | READY_FOR_REVIEW | today → `/mis` · custody → `/sales/custody` · reports & reconciliation → `/reports/*`, `/mis/reconciliation`, `/mis/source-quality` | import & data quality, lineage, planning, quick delivery, maintenance, diagnostics: MISSING |
| HR (`hr`) | READY_FOR_REVIEW. **HR runtime NOT LIVE VERIFIED** | workforce → `/hr` · structure → `/hr/structure` · requests → `/hr/requests` · exceptions → `/hr/exceptions` | compensation: MISSING. Onboarding/transfer and termination: BLOCKED (OPD-05, SD-04). Structure & access, credentials: BLOCKED (D-34, R5-SEC-01) |
| Finance (`finance`) | READY_FOR_REVIEW. **Finance runtime NOT LIVE VERIFIED** | review queue → `/finance/review` (capability UNKNOWN in RM-E02; CONTRACT since RM-ACCESS-01) · reconciliation → `/finance/reconciliation` · ledger → `/finance`, `/finance/ledger`, `/finance/balances` | rules, runs & posting: BLOCKED (D-37, D-38). Refund: BLOCKED (OPD-04). Reports, audit, configuration, maintenance: MISSING |
| Invoice reviewer (`invoice_reviewer`) | NO_DESIGN | — | invoice review: BLOCKED (D-24, API only, no page) |

What the role context does and does not do:

- **SINGLE.** Every ALLOWED decision matched exactly one known role key. The header shows «نقش: …». Role-mapped destinations come first; other allowed destinations follow under «سایر بخش‌های مجاز». The set is exactly the server's.
- **MULTIPLE.** More than one role or grant basis matched. No role is chosen, and no role switcher is invented: there is no approved context-selection policy (D-42 / RM-B02). The home page says so, and the navigation lists exactly the server-allowed set, grouped by module. The server already evaluates the union of a subject's grants; the SPA neither adds nor removes a destination.
- **UNKNOWN.** No known role matched, for example the synthetic dev grant. No role is named.

## 6. Explicit fail-closed cases

| Case | Server answer | Navigation | Home (role entry) |
|---|---|---|---|
| Session loading | — | safe only | loading status (`aria-busy`), no data, no business call |
| Session failed / unknown | network or 5xx | safe only | Gate: «وضعیت نشست مشخص نشد» with retry (unchanged) |
| Not authenticated (session mode) | `authenticated:false` | safe only | login page (unchanged); «نشست شما پایان یافته است» when `session_state: invalid` |
| Not authenticated (no auth / dev stub without identity) | `authenticated:false` | safe only | «هویتی برای این نشست ثبت نشده است». No evaluation call is made |
| Evaluation pending | — | safe only | «در حال بررسی دسترسی‌ها…» |
| Evaluation failed | non-2xx, transport error or non-SUCCESS envelope | safe only | «ارزیابی دسترسی انجام نشد» with «بررسی دوباره» |
| Evaluation contradicts the session | `authenticated:false` while the session is authenticated | safe only | CONFLICT state with retry |
| Evaluation BLOCKED | every decision `BLOCKED` (RM-E02: `organization.hr_read_shadow` off; since RM-ACCESS-01 only when every capability's own feature is off) | safe only | «ارزیابی دسترسی در سرور در دسترس نیست» with the server reason code |
| No ACTIVE grant | every decision `UNAUTHORIZED` | safe only | «دسترسی فعالی برای شما ثبت نشده است» |
| Allowed, but feature off | ALLOWED, feature `false` | safe only | «بخش‌های مجاز شما هنوز فعال نشده‌اند» |
| Feature on, not allowed | — | that link hidden | — |
| Capability outside the OpenAPI enum | ALLOWED at runtime | hidden (UNKNOWN; none today since RM-ACCESS-01) | — |
| `allowed:true` without `SUCCESS`, or `SUCCESS` without `allowed` | — | hidden | — |
| Direct URL to a hidden page | the page's own read answers 403 | no link is marked current | the page renders the server refusal; the route is not guarded client-side |
| Session expires or is revoked mid-visit | 401/403 → session re-check → `invalid` | safe only | login page with the expiry notice; the previous evaluation is never reused for a new session answer |

Never inputs: `localStorage`, query parameters, a hard-coded or selected role, the dev identity, `capability_hints`, feature flags alone, UI state. Test 13 sets the first three and the hints, and asserts that nothing appears.

## 7. Design references

- Components: RM-E01 shared design system (`web/src/shared/ui/ds`, frozen SN-207 tokens). No new token or business pattern was added. `routes/home.css` is a small destination-list layout on existing tokens.
- Seller Design V1 is **FROZEN**. Navigation chrome follows it: tab row, «بیشتر» overflow, bottom bar, `aria-current`.
- Supervisor: freeze **not verified** (see §5). Senior, Manager, Deputy, MIS, HR and Finance are **READY_FOR_REVIEW**, so they are not frozen (AGENTS.md). For these roles RM-E02 delivers the navigation *contract* only (`roles.ts`). Their labels, grouping, iconography and dashboards are deferred to each role's freeze.
- RM-E01 Menu, fixed while integrating it. A long overflow menu (about 25 items) could extend past the viewport, leaving its last items unreachable. The bug was found by the RM-E02 browser test. The menu is now bounded by the viewport and scrolls internally (`Menu.tsx` placement, `.ds-menu{overflow-y:auto}`; on phones `max-height: calc(100dvh - 88px)`). The RM-E01 browser suite still passes (11/11).

## 8. Screenshots (synthetic fixtures, production bundle, Chromium)

| | Desktop 1366 | Mobile 390 |
|---|---|---|
| No ACTIVE grant | `docs/evidence/rm-e02/no-access-1366.png` | `docs/evidence/rm-e02/no-access-390.png` |
| Single role (seller fixture) | `single-role-1366.png` | — |
| Deep link to a refused page | `deep-link-refused-1366.png` | — |
| Several roles, priority+ overflow | `multi-role-overflow-1366.png` | `multi-role-390.png` |

Regenerate them with `RM_E02_EVIDENCE_DIR=../docs/evidence/rm-e02 npx playwright test -c playwright.nav.config.ts` (from `web/`). The role fixtures are synthetic answers injected by the test. They are presentation evidence, never proof of authorization.

## 9. Test evidence

| Suite | Result | Covers (task §8 numbering) |
|---|---|---|
| `vitest` (all) | **176/176** (was 152; +24) | `navigation/navigation.test.tsx`: resolver (feature AND capability; UNKNOWN capability; BLOCKED; LOADING/FAILED/CONFLICT/ANONYMOUS; SINGLE/MULTIPLE/UNKNOWN role; longest-prefix current), routes and deep links (8), direct URL to a refused page (7), cases 1–6, 12, 13, 13b, 14, 14b |
| `playwright.nav.config.ts` (new; production bundle via `vite preview`) | **9/9** (desktop 7, mobile 2) | 1, 7, 8, 9 (overflow keeps current; menu keyboard; client-side routing without reload), 10, 11 (skip link, focus order, focus ring), 12, 13 (`/dev/design-system` → 404; no DEV text; no `X-Dev-User`), 14, expiry |
| `playwright.ds.config.ts` (RM-E01) | **11/11** | regression after the Menu fix |
| typecheck, e2e typecheck, lint, build | clean | production bundle grep: no `X-Dev-User`, `dev-user-`, `PrimitivesPage`, `DesignSystemPage`, DEV labels or chunks |
| Full-stack R7 E2E (`e2e-ci.yml`) | **CI only**: the Go orchestrator uses `Setpgid`/`syscall.Kill` (Linux), so it cannot run on this Windows host | see below |

R7 browser specs updated. Each asserted the old flag-only navigation, which this task forbids. Every refusal, no-data and no-write assertion is kept:

- `fulfillment.spec.ts`, `reporting-biawin.spec.ts`: links are absent and the home page shows BLOCKED (`organization.hr_read_shadow` is off on those stacks). *RM-ACCESS-01: now NO_GRANT (the modules' own flags are on; the users hold no grant).* Pages are reached by deep link and still show the server refusal.
- `hr.spec.ts`: HR flags on and no grant, so home shows NO_GRANT and the links are absent; `/hr` and `/hr/requests` are reached by deep link.
- `reporting-biawin-overlay.spec.ts`: the overlay grants are synthetic, but effective-access is BLOCKED (HR flag off), so links are withheld. *RM-ACCESS-01: the reporting and Biawin links now appear (their own flags are on); HR and every other module stay absent.* All render-equals-API-equals-DB checks still run through deep links.
- `hr-read-overlay.spec.ts`: unchanged. HR flags are on and `dev-user-001` is ALLOWED HR reads, so the links appear (role context UNKNOWN or MULTIPLE).
- `e2e/fixtures.ts`: a new `expectFailClosedNavigation()` helper.

Unit tests changed: `hr.test.tsx` and `customers.test.tsx` asserted a link driven by the flag alone; both now assert it is absent. `App.test.tsx` replaced the «منو» toggle test with a bottom-navigation test. `test/setup.ts` raised RTL's async timeout from 1 s to 3 s, because lazy routes plus the extra evaluation answer exceeded 1 s on cold starts under a parallel run. No assertion was weakened.

A passing navigation test is **not** evidence of business authorization. No grant, bundle, flag or account was created anywhere.

## 10. Missing server contracts and follow-ups

### RM-B02 / D-42: narrowly scoped proposal (documentation only, no change here)

1. **RESOLVED by RM-ACCESS-01.** **Decouple effective-access from `organization.hr_read_shadow`.** Today one HR flag turns every decision into BLOCKED, so navigation fails closed for every module whenever that flag is off. Proposal: evaluate each capability on its own (the per-module flags are already separate). This needs an authorization-semantics review, so it is not changed here.
2. **RESOLVED by RM-ACCESS-01.** **Complete the OpenAPI `Capability` enum** with the 14 capabilities the server already evaluates and returns: `payments.{upload.stage,batch.submit,submission.read,evidence.read}`, `finance.review.read`, `commission.{shadow,policy,own}.read`, `fulfillment.work.{read,assign,execute}`, `biawin.activation.read`, `integration.provider_operation.{read,reconcile}`. Until then 9 destinations stay UNKNOWN to the SPA.
3. **Role context.** Expose an explicit, server-decided context: the active memberships and, if multiple, an approved selection policy. Do not derive one from `matched_roles`. Until then MULTIPLE stays neutral, with no switcher.
4. **Role bundles and ACTIVE grants** (D-42, D-34, R5-SEC-01). With those decisions approved, users would see navigation; nothing in the SPA needs to change.

### Deferred

- Every MISSING or BLOCKED section in §5. No fake screens were built.
- Role-specific labels, dashboards, queue tabs, counts and metrics (RM-E03+; they need role design freezes and KPI contracts).
- Role switcher: not invented (see above).
- Search («جستجو») and the Help centre in the production header: no search API contract exists, and the help content belongs to each role design.

## 11. Dependencies for RM-E03 (Seller read screens)

- Seller design is FROZEN, so presentation is not blocked.
- **Blocking for any real user:** the seller bundle is empty and no ACTIVE grant exists (D-42). RM-E03 can only be tested with synthetic fixtures or the dev stub. ~~With `organization.hr_read_shadow` off, effective-access is BLOCKED and the seller would see no navigation (RM-B02 item 1).~~ Resolved by RM-ACCESS-01: a seller capability is gated only by its own module flag.
- RM-E03 must add its new destinations to `destinations.ts` with a **contract** capability, and map them in `roles.ts` (seller). The resolver and shell need no change.
