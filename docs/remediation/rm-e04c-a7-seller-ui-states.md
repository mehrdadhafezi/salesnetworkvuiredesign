# RM-E04C / A7 — Seller UI State Implementation (synthetic preview)

| | |
|---|---|
| Baseline | `main` @ `940ca3b892f7c337e208f37141332b934961ddf7` (PR #75 merged; verified with `git log origin/main`) |
| Branch | `feat/rm-e04c-a7-seller-synthetic-ui-states` (isolated worktree) |
| Roadmap row | [`../implementation/rm-e04c-seller-implementation-roadmap.md`](../implementation/rm-e04c-seller-implementation-roadmap.md) **A7** |
| Design authority | [`../security/rm-e04c-seller-screen-designs.md`](../security/rm-e04c-seller-screen-designs.md) §0 + S1–S7; RM-E03 Seller V1 preview; RM-E01 design system |
| Scope | Frontend presentation only, inside the DEV/TEST-only `/dev/seller/*` preview |
| Not changed | authorization evaluator, role bundles, capabilities, grants, accounts, Group B gate, database, backend, OpenAPI, decision tracker, deployment |
| Human approval | Design review (roadmap A7). Not obtained; not claimed |
| Production | **NO-GO** |

Group B (B1, B2a, B2 …) is **not** started. No Seller data is wired to any server.

---

## 1. What was built

### 1.1 One state model (`web/src/modules/seller/state/uiState.ts`, pure)

| Piece | Purpose |
|---|---|
| `SellerUiState` | the twelve presentation states of screen-design §0 |
| `classifyAnswer(answer)` | HTTP status + envelope (`outcome`, `reasons`, count state) → state. Authorization and transport are checked before the envelope. **Anything unrecognised fails closed to `failure`**, never to success or empty |
| `STATE_POLICY` | per state: `blocksContent`, `retry`, `countsAbsent`, `dependentAmountsWithheld`, `fieldsRestricted`, `statusUnknown`, `detailNotFound` |
| `SYNTHETIC_ANSWERS` | one synthetic answer per state; correlation ids are `syn-corr-a7-*` |
| `Sim` / `SIMS` / `stateOfSim` | the preview's `?sim=` switch. The RM-E03 values (`none`, `loading`, `empty`, `error`, `offline`) keep their URLs; nine A7 values were added |

Types come from the contract through `src/shared/api/types` (`Outcome`, `ReasonCode`, `Unit`). No reason code, outcome or business status was added.

### 1.2 Shared presentation (`components/states.tsx`, `components/bits.tsx`)

| Component / hook | Behaviour |
|---|---|
| `ScreenFrame` | wraps every screen. For a blocking state it renders **only** the plain title and one state panel: no KPIs, counts, lists, drawers or amounts are mounted. Otherwise it adds the PARTIAL / UNKNOWN / FIELD_DENIED banners and the S2/S3/S5 contract notice |
| `useStatePresentation` | `data()` (an exact-empty answer has no rows), `count()` / `countText()` / `kpi()` (absent → «—», exact 0 stays ۰) |
| `useDetail` + `NotFoundDrawer` | a missing id and an out-of-scope answer give the **same** drawer: «یافت نشد» / «این مورد وجود ندارد یا برای شما در دسترس نیست.» The requested id is not echoed |
| `Restricted` | «محدود» pill with an accessible reason; no partial value or length hint |
| `Withheld` | «—» for an amount withheld by PARTIAL (announced as withheld, distinct from unknown) |
| `Who`, `useCustomerName` | under FIELD_DENIED: name and phone → «محدود»; drawer titles → «مشتری (محدود)»; copy-phone actions removed |
| `LeadStatusPill`, `InvoiceStatusPill`, `StatusPill` | under UNKNOWN mapping: «نامعلوم» (dashed amber) instead of any known status |
| `TableFoot` | absent total → «—» + «تعداد کل نامعلوم» |
| `Money` | carries a `unit` (`IRT` تومان · `IRR` ریال) |
| selectors `filter*(…, identity)` | under FIELD_DENIED, name and phone are **not searchable**, so a search cannot confirm a withheld value |

## 2. State coverage

| # | State | Server answer (synthetic) | Presentation |
|---|---|---|---|
| 1 | SUCCESS | 200 `SUCCESS` | data |
| 2 | Exact empty | 200 `SUCCESS`, 0 items, count `EXACT` | empty panel; counts are a real ۰ |
| 3 | PARTIAL | 200 `PARTIAL` + `COVERAGE_INCOMPLETE` | incomplete banner; remaining / payable / confirmed / collectible / wallet sums withheld («—»); declared totals and stage amounts stay; lead history marked incomplete |
| 4 | ABSENT count | 200, count `ABSENT` | every count / KPI / table total «—» with «تعداد نامعلوم»; rows still render |
| 5 | FIELD_DENIED | 200 + `FIELD_DENIED` | «محدود» for name and phone in lists, cards, drawers, request list and command palette; identity not searchable |
| 6 | UNKNOWN / unmapped | 200 + `OUTCOME_UNKNOWN` | status and stage pills «نامعلوم»; invoice remaining / payable / confirmed unknown; wallet: unknown run → no sum at all |
| 7 | FEATURE_DISABLED | 200 `BLOCKED` + `FEATURE_DISABLED` | locked panel, no content |
| 8 | ACTION_DENIED | 403 | locked panel «دسترسی به این بخش ندارید», no content, nothing about any record |
| 9 | ROW_SCOPE_DENIED | 403 + `ROW_SCOPE_DENIED` | locked panel «هویت کاربری شما … تأیید نشده است», no content |
| 10 | 404 | 404 | generic «یافت نشد» drawer, identical for missing and out-of-scope |
| 11 | 401 | 401 | locked panel «نشست شما به پایان رسیده است»; no content, no drawer, no palette records, no notification badge |
| 12 | Failure | 500 / network | alert «امکان دریافت لیست وجود ندارد» with the correlation id; **only** state with «تلاش مجدد» (re-reads; no write) |

Blocking states (7, 8, 9, 11, 12) also hide header data: nav counts, the open-work counter, the notification badge and every record in the command palette.

## 3. Screen coverage

| Screen | A7 work |
|---|---|
| S1 شماره‌های من | all 12 states; not-found parity; partial history note; FIELD_DENIED in rows, cards, drawer (phone, name, copy action) and search. The outcome grid stays the RM-E03 demo (validates, toasts «چیزی ثبت یا ارسال نشد»); no write is wired |
| S2 آماده‌های تبدیل | contract notice (no server list source; readiness is not a server state); facets kept as separate rows; **fix:** the case drawer showed «پرداخت تایید‌شده» as `0` whenever the remaining amount was known — now «—» (unknown), because no source exists |
| S3 نیاز به اقدام مجدد | contract notice (no server source; production destination unavailable); not-found parity; FIELD_DENIED |
| S4 فاکتورها | per-unit amounts (new IRR invoice `TST-1015`); collectible remaining as one KPI **per unit**; PARTIAL withholds dependent amounts; UNKNOWN maps stages and payable to «نامعلوم»; not-found parity; no confirmed / settleable amount invented |
| S5 رفتار مشتریان | contract notice (no event source); counts absent → «نامعلوم»; FIELD_DENIED |
| S6 درخواست اضافه کردن شماره | «این گردش‌کار فعال نشده است» notice (OPD-10 open); the button is now «بررسی فرم (نمایشی)» because it only validates locally; the form has no `action` and nothing is sent |
| S7 کیف پول من | sums per unit (new IRR line `w9`), never added across units; hero relabelled «اعتبار تاریخی تایید‌شده (نمونه)» with «اعتبار تاریخی پورسانت است، نه وجه نقد قابل برداشت.»; settlement «— تعریف نشده»; unknown run → banner, no sum; PARTIAL → sums withheld. No API wiring (that is B3) |

## 4. Synthetic fixture strategy

- All data stays in `fixtures/sellerFixtures.ts` (`RM-E03-SYNTHETIC-SELLER-FIXTURE`): «مشتری نمونه …» names, `0900000xxxx` phones, `TST-` codes, `SYN-` user ids.
- A7 adds only `TST-1015` (IRR invoice) and wallet line `w9` (IRR), both following the same rules, to exercise the unit split.
- States are produced by `?sim=<state>` over the same fixtures (an exact-empty answer is modelled as zero rows, so counts and sums are a real 0 rather than fixture values).

## 5. Security and preview isolation

| Check | Result |
|---|---|
| Route | `/dev/seller/*` still mounted only under `import.meta.env.DEV && config.devTools` (`app/routes.tsx`); unit test asserts it is the **only** importer of the Seller module |
| Network | static test: no `fetch`, `XMLHttpRequest`, `sendBeacon`, `WebSocket`, `EventSource`, storage or clipboard API and no API client in the Seller module; render tests spy on `fetch` for all 14 sims × 7 screens (98 renders) and expect zero calls; browser tests intercept `/api` and `/health` and allow only the app-level session probe, no non-GET |
| Production bundle | `npm run build` + the web-ci grep (extended locally with `syn-corr-a7`, `SYNTHETIC_ANSWERS` and A7 copy): no preview code or chunk in `dist/` |
| Authorization | no client-side authorization: the simulator only selects which synthetic answer is presented; the UI never decides scope |
| Stale data | blocking states unmount every list, drawer, KPI and palette record (tests assert no fixture name, phone or `TST-` code remains in the DOM) |
| Not-found parity | rendered text of the not-found drawer is byte-identical for missing and out-of-scope (5 screens, unit + browser) |

No security finding against shipped code. One preview defect (S2 invented `0`) was fixed (§3).

**Review follow-up — FIELD_DENIED accessibility leak (fixed).** The S6 request timelines kept the customer phone in their `aria-label` (`روند درخواست ۰۹۰۰…`) while the visible text showed «محدود». Under FIELD_DENIED the label is now `روند درخواست ثبت‌شده در <date>`; it is unchanged when fields are not restricted. A DOM-wide audit of all seven screens (detail drawers open, collapsible timelines expanded, command palette open) scans **every attribute value** (`aria-*`, `title`, `data-*`, `id`, `href`, `placeholder`, …), input values and all text nodes including hidden ones, for every synthetic customer name and phone in ASCII and Persian digits. Before the fix it found exactly these three labels and nothing else; after the fix it finds none. Timeline event text names only synthetic staff roles, never the customer. A control test proves the scanner does see identity when fields are not restricted. Browser test: the same attribute check on all four viewports.

## 6. Tests and actual results

Run in the worktree at the branch head (Node 22, Vitest 5, Playwright 1.56.1 with the container's Chromium):

| Suite | Command | Result |
|---|---|---|
| Unit (all web) | `npx vitest run` | **19 files, 229 tests passed** |
| Seller unit (incl. new) | `npx vitest run src/modules/seller` | **58 passed** (14 existing, 16 `state/uiState.test.ts`, 28 `seller.ui-states.test.tsx`) |
| Typecheck | `npm run typecheck`, `npm run typecheck:e2e` | pass |
| Lint | `npx eslint .`, `npx eslint e2e playwright.seller.config.ts` | pass |
| Production build | `npm run build` | pass |
| Bundle isolation | web-ci grep + A7 markers | pass (no leak, no dev chunk) |
| Seller browser E2E (4 viewports) | `playwright.seller.config.ts` | **61 passed, 23 skipped** (skips are the viewport-specific tests outside their viewport, by design) |
| Role navigation E2E (production build) | `playwright.nav.config.ts` | **9 passed, 9 skipped** (viewport-specific) |
| Governance | `go test ./docs/governance/` | pass |
| Mutation checks | break FIELD_DENIED in `Who`; make `ScreenFrame` never block; make the classifier fail open; restore the S2 `0` | each caught (1, 6, 1, 1 failing tests); restored tree green |

Playwright was run through a local, uncommitted config that only sets `executablePath: /opt/pw-browsers/chromium`, because the container's preinstalled Chromium build differs from the one the pinned Playwright downloads. CI uses its own `playwright install`.

Not run here: the full-stack E2E (`e2e-ci.yml`, needs the Go API + PostgreSQL; unaffected — A7 touches no backend) and the design-system E2E (unchanged components).

## 7. Browser evidence

`docs/evidence/rm-e04c-a7/` — desktop (1366) and mobile (390) only, 11 states each: `action_denied`, `row_scope_denied`, `feature_disabled`, `session_expired`, `failure`, `count-absent`, `partial-invoices`, `field-denied`, `unknown`, `not-found`, `wallet`. Generated by the E2E run with `RM_E04C_A7_EVIDENCE_DIR`. Tablet / wide renders were checked by the same tests but not committed (AGENTS.md: no repetitive screenshots).

## 8. Accessibility

- Every state panel uses the RM-E01 `StatePanel` roles: failure is `role="alert"`; the others are polite `role="status"`.
- «—» values carry screen-reader text: «تعداد نامعلوم», «مبلغ نامعلوم», «مبلغ نمایش داده نمی‌شود: …».
- «محدود» pills announce which field is withheld.
- Every screen in every state has exactly one `h1`. This is tested for 14 × 7 combinations.
- There is no page-level horizontal scroll at any viewport in the blocking states or the wallet (browser test).
- Retry is a real `<button>`. The S6 submit button was renamed so that it does not claim to submit, and it is described as local-only.

## 9. Known limitations

- The state switch is a preview control (`?sim=`). Real answers will be classified by `classifyAnswer` only once Group B wires an endpoint.
- In the preview, one state applies to the whole screen. A real server could return different states per request, for example a list that succeeds while a detail returns 404. The model supports this because classification is per answer, but the preview does not combine states.
- Under the PARTIAL simulation, every dependent amount is withheld, so the preview cannot show which specific facts are missing. The contract has no per-field completeness for this.
- A known `0` remains visible where it is a computed fact, for example the IRR «در انتظار تایید» sum in the wallet.
- S2 readiness, the S3 hand-off, S5 events and the S6 workflow still have no server source, as before.

## 10. Rollback

Revert the PR. It changes nothing outside `web/src/modules/seller/`, `web/e2e/seller-preview.spec.ts` and `docs/`; there is no data, schema, contract, flag or grant to undo.

## 11. Production

**NO-GO.** The preview is DEV/TEST-only and not in the production bundle. HR and Finance runtime remain **NOT LIVE VERIFIED**.
