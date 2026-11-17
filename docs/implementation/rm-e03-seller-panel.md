# RM-E03 — Seller V1 React Panel (DEV/TEST preview)

Status: **CONTROLLED FRONTEND IMPLEMENTATION — PRESENTATION AND SYNTHETIC DATA ONLY.** Production: **NO-GO.**
Branch: `feat/rm-e03-seller-panel` · Baseline: `main @ 3e9e3b012e0589864dfaaafa5d3932b3a9aa7f8b` (PR #67 merged; `origin/main` had not advanced).

Nothing in this change grants a permission, role, capability, credential or flag. It adds no backend, OpenAPI, schema, migration, session, identity or authorization change, and the preview reads no real customer record. D-42, D-34, R5-SEC-01, ADR #18, OPD-10 and the RM-B02 proposed records remain **unresolved**.

---

## 1. What was built

Seller Design V1 (`prototype/index.html`, `app.js`, `styles.css`, `data.js`, `mockups/`) is now a React + TypeScript module. All seven destinations are there, on the RM-E01 shared design system and **synthetic fixtures**, at an isolated **DEV/TEST-only** route.

| # | Section | Route (development preview, **not** an approved business URL) | Page |
|---|---|---|---|
| S1 | شماره‌های من | `/dev/seller/leads` | `pages/LeadsPage.tsx` |
| S2 | آماده‌های تبدیل | `/dev/seller/conversions` | `pages/ConversionsPage.tsx` |
| S3 | نیاز به اقدام مجدد | `/dev/seller/repeat` | `pages/RepeatPage.tsx` |
| S4 | فاکتورها | `/dev/seller/invoices` | `pages/InvoicesPage.tsx` |
| S5 | رفتار مشتریان | `/dev/seller/behavior` | `pages/BehaviorPage.tsx` |
| S6 | درخواست اضافه کردن شماره | `/dev/seller/request` | `pages/RequestPage.tsx` |
| S7 | کیف پول من | `/dev/seller/wallet` | `pages/WalletPage.tsx` |

`/dev/seller` and any unknown section redirect to `/dev/seller/leads`.

## 2. Architecture

```
web/src/modules/seller/
  SellerPreviewPage.tsx        preview shell: DsAppShell + AppHeader + RoleNavigation + MobileBottomNavigation, palette, toasts
  types.ts                     presentation types (not API contracts)
  labels.ts                    prototype dictionaries (LEAD, INV, outcomes, NAV) — display vocabulary only
  fixtures/sellerFixtures.ts   SYNTHETIC data + SELLER_PREVIEW_MARKER
  state/selectors.ts           pure logic: search, queues, filters, sort, pagination, invoice/wallet figures, validation
  state/SellerPreviewContext.tsx  in-memory theme / density / demo-state / toasts (nothing persisted)
  state/hooks.ts               drawer ↔ URL query, page keyboard shortcuts
  components/bits.tsx          Money, pills, NextStep, Who, Timeline, Note, OwnershipFacts, TableFoot, FilterChip, SearchField, useListState
  components/overlays.tsx      OutcomeGrid, Toasts, CommandPalette
  pages/*.tsx                  the seven sections (+ their drawers)
  seller.css                   Seller-domain styles ported from prototype/styles.css, `sl-` namespace, tokens only
  seller.test.tsx              unit + component tests
web/e2e/seller-preview.spec.ts browser tests · web/playwright.seller.config.ts
```

The prototype's imperative DOM rendering is replaced by typed React state. Page-local `useState` holds the queue, search, filters, sort, page and form values. The drawer target lives in the URL query (`?lead=`, `?inv=`, `?case=`, `?item=`, `?b=`), which keeps the prototype's deep links. Preferences and demo states live in a small context. Every derived figure comes from pure functions in `state/selectors.ts`.

## 3. Route isolation proof

- **Build-time guard.** `app/routes.tsx` mounts `/dev/seller` and `/dev/seller/:section` only inside `if (import.meta.env.DEV && config.devTools)`. That is the same guard as the RM-E01 `/dev/design-system` showcase. Production builds drop the branch, and `configFromEnv` forces `devTools=false` in production.
- **Outside the RM-E02 shell.** The routes sit at the top level, as the DS showcase does. They are never added to `navigation/destinations.ts` or `roles.ts`, so the fail-closed role navigation is untouched. A test asserts that no destination contains `/dev`.
- **Production bundle.** `npm run build` followed by the `web-ci` grep comes back clean. The grep now also fails on `SellerPreviewPage`, `RM-E03-SYNTHETIC-SELLER-FIXTURE` and any synthetic `0900000xxxx` phone, and on any `SellerPreview`/`seller` chunk name.
- **Production browser test.** `e2e/role-navigation.spec.ts` (production bundle via `vite preview`) now opens `/dev/seller`, `/dev/seller/leads` and `/dev/seller/wallet`. It expects «صفحه پیدا نشد», no `[data-preview]` element, no non-GET request and no dev header.
- **No authorization substitute.** No query parameter, `localStorage` value or role name is read to decide access. Query parameters only choose presentation (theme, demo state, open drawer). Production routes are unchanged, and a test asserts that the `/sales`, `/billing`, `/customers` and `/commission` paths are still present.
- **No real Seller account can reach it.** It exists only in dev/test builds, and it uses no session or identity at all.

## 4. No-write / no-read evidence

- The module never calls the API client or `fetch`. The component test spies on `fetch` while rendering all seven sections and records **0 calls**.
- The browser suite intercepts every `/api/*` and `/health*` request and records it. Only the app-level `GET /api/v1/session` probe made by `SessionProvider` appears (it is answered as unauthenticated). There are **no other paths and no POST/PUT/PATCH/DELETE** in any viewport.
- Every simulated business action ends in a toast that ends with «(نمایشی — چیزی ثبت یا ارسال نشد)» or says the action is not available. No toast claims that a server action succeeded.

## 5. Synthetic fixture policy

- Values are new. `window.MOCK` is not imported; `prototype/data.js` is used for structure and wording only.
- Names read «مشتری نمونه …», the seller is «فروشنده نمونه · SYN-0001», phones are in the unallocated `0900000xxxx` range, and invoice codes are `TST-xxxx`. A unit test enforces all of these.
- Amounts, commission lines and rules («قاعده نمونه #…») are invented for layout. They are not financial facts or entitlements.
- **Identity (AGENTS.md).** The prototype matched a lead's invoices by phone; the React version uses explicit synthetic `invoiceRefs`. Phone is shown as discovery data only, and the copy and call buttons are demo toasts.

## 6. Shared design-system reuse (RM-E01)

`DsAppShell`, `AppHeader`, `HeaderIconButton`, `HelpCenterButton`, `MenuButton`/`Menu`, `RoleNavigation` (priority+ overflow), `MobileBottomNavigation` (with «بیشتر»), `PageHead`, `Panel`, `Toolbar`, `QueueTabs`, `Metric`/`MetricGroup`, `Drawer`/bottom sheet, `StatePanel`, `Banner`, `StatusIndicator`, `TonePill`, `ContextHint`, `DsIcon`, `toFaDigits`/`formatFaNumber`, and the tokens in `tokens.css`/`base.css`.

The one shared change is the prototype's `phone` stroke icon, added to `ds/icons.tsx`. It is the same path as in `prototype/app.js` and adds no capability. Everything else that is Seller-specific is in `seller.css`: table, cards, outcomes, stages, wallet, palette and toasts, all `sl-` namespaced, tokens only and with no raw colours.

## 7. Prototype → React mapping and interaction inventory

| Prototype | React | Behaviour now |
|---|---|---|
| `NAV`, `renderNav`, bottom nav | `sellerNavItems` → `RoleNavigation` / `MobileBottomNavigation` | Real routes. `aria-current`, overflow into «بیشتر», and a mobile «بیشتر» menu |
| `pageHead` KPIs | `Metric` | Computed from fixtures; undefined or unavailable metrics show «—» with a reason |
| `queues()` | `QueueTabs` | Filter the list for real; ←/→/Home/End move between tabs; keys ۰–۶ switch lead queues |
| `toolbar()` search | `SearchField` | Local search over the fixtures; Persian digits are normalised; Esc clears; **F** focuses it |
| Filter chips (static in prototype) | `FilterChip` (menu) | Working local filters: source, probability, assignment date, product and wallet state |
| Sort «جدیدترین» | toggle button | Newest or oldest |
| `tfoot()` pager (static) | `TableFoot` | Working pagination, 10 per page; previous/next disabled at the ends |
| Row click / Enter, `J`/`K`, ↑/↓ | rows + `usePageKeys` | Open the drawer; step through the current list; roving focus |
| `leadDrawer` / `caseDrawer` | `Drawer` + `OutcomeGrid` | Outcome choice (keys ۱–۵); conditional fields; required validation with `aria-invalid` and an inline error; Esc closes and focus returns to the row |
| `invoiceDrawer` | `InvoiceDrawer` (wide) | Four money facts, stages with their state, history |
| `behaviorDrawer` → «مشاهده فاکتور» | navigate to `/invoices?inv=` | Opens that invoice's drawer |
| Command palette (Ctrl/⌘+K, `/`) | `CommandPalette` | Dialog over the synthetic fixtures; ↑↓, Enter, Esc; focus is restored |
| User menu: theme, density | `MenuButton` | Light, dim and dark themes and density are applied to `<html>` and restored when you leave |
| Help demo states (`?sim=`) | user menu «حالت‌های نمایشی» + `?sim=` | loading, empty, error (with retry) and offline |
| `toast()` | `Toasts` | Polite status, at most 3, closable, auto-dismiss after 4.5 s |

### Disabled or non-executable actions (visible, never silent)

| Control | Behaviour |
|---|---|
| «ثبت …» / «صدور پیش‌فاکتور» / «ذخیره نتیجه تماس» | Disabled until an outcome is chosen, then validated, then a demo toast. Nothing is recorded |
| Invoice primary action (send link / register payment / resend to finance), «ثبت پیگیری مرحله» | Demo toast only |
| Invoice menu: «ویرایش پیش از پرداخت», «ارسال مجدد لینک پرداخت» | Shown **disabled with a stated reason** |
| «کپی لینک پرداخت», copy phone, call | Demo toast; the clipboard and dialer are not touched |
| Extra-number «ثبت درخواست (نمایشی)» | Validates, then shows an OPD-10 warning toast. The list never grows |
| User menu «رمز من», «خروج» | Disabled with a stated reason (no account or session) |
| Wallet «پروژه‌ها» | Locked state: not defined for Seller V1 |
| Lead customer info | Read-only. The note is a local draft labelled «ذخیره نمی‌شود» |

## 8. Truthful-state decisions (no invented certainty)

- **Invoices.** The four facts are total · **financially confirmed** · remaining (= total − confirmed) · current payable. A submitted receipt (`finance_review`) is never counted as paid; it gets a note. A stage whose submission result never arrived is **Outcome Unknown**: dashed stage, amber pill, and a banner saying «پیش از هر ارسال مجدد باید تطبیق شود». In that case current payable is «نامعلوم», not an amount. A cancelled invoice shows «—» for remaining. When an unknown exists, the «مانده قابل وصول» KPI says it is not final.
- **Wallet.** The prototype's «موجودی قابل تسویه» (settleable balance) is replaced by **confirmed commission (sample)**, **pending confirmation** and **withdrawal/settlement: «— تعریف نشده»**. Unknown lines are excluded from both sums, and a banner says the totals are incomplete. The amount column is labelled as a commission amount, not the sale amount.
- **Conversions.** The attribution facts use the fixed SN-207 vocabulary (مسئول فعلی · اقدام بعدی با · عامل رویداد · مالک اعتبار). The credit owner is **UNKNOWN** because no approved rule exists in the preview. A remaining amount that is not known shows «نامعلوم».
- **Repeat.** The original seller (issuer), the current custody of the stage and the next actor are three separate facts. The prototype's own copy is kept: «مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود».
- **Behavior.** An unknown event count shows «نامعلوم» (incomplete), not 0. «لینک باز شده» and «پرداخت پس از بازدید» are shown as unavailable or undefined instead of invented percentages.

## 9. Visual parity

**Method.** Playwright renders every screen at the prototype's render sizes (1366×768, 1920×1080, 768×1024, 390×844; the prototype breakpoints are ≤1100 tablet and ≤760 mobile). The renders were compared by eye with `prototype/mockups/01…11`.

**Matches.** Header (brand, crumb, demo tag, search with Ctrl K, notifications, help, user), the separated tab groups with counts, page head with the KPI strip, queue chips with tone and count, toolbar with search and chips, sticky table head, row density, alternating rows, the selected-row inset bar, status pills, next-step tones, row actions, tfoot and pager, the 560/660 px drawer on the left in RTL and a bottom sheet on phones, the outcome grid with key hints, stage list, sum grid, the wallet three-card layout with segmented filter, the two-column request form, mobile cards with call and record actions, labelled stacked rows for tables without cards, and the bottom bar with «بیشتر». At tablet width the optional columns collapse and the table scrolls inside its panel with a sticky action column. No viewport has a page-level horizontal scroll (tested for all 7 screens × 4 viewports).

**Documented deviations.**

1. Status pills carry an **icon** (frozen `styles.css` / SN-207 §1.4, RM-E01 `TonePill`). Some older mockup PNGs show dots.
2. The wallet hero card uses `primary-soft`, as the current frozen `styles.css` `.wcard.hero` does; the older `06-desktop-wallet.png` shows a solid fill. Its wording changed for truthfulness (§8).
3. The invoice table has an extra **«قابل پرداخت فعلی»** column and the drawer has four sum cards instead of three. The task requires total, paid, remaining and current payable to be distinguished.
4. KPI tiles are not clickable filters (RM-E01 `Metric` is presentational); the queue tabs give the same filtering.
5. «حالت تمرکز» (focus mode) is not ported in this slice.
6. The `Drawer` frame title is the RM-E01 `h2`. The prototype's head line (pill, meta, facts, quick actions) sits directly under it in the drawer body. The lead and case footers are sticky inside the body because their state lives there.
7. The repeat-action screen has synthetic rows. The frozen prototype shows only its empty state, which is still reachable through the «انجام‌شده»/`?sim=empty` states with the prototype's wording.
8. The wallet «بازه تاریخ» chip is replaced by a working «وضعیت» filter (confirmed, pending, unknown), because the fixture has no date-range semantics.

**Screenshots** are in `docs/evidence/rm-e03/` (39 PNGs, synthetic only): `{desktop,wide,tablet,mobile}-{leads,conversions,repeat,invoices,behavior,request,wallet}.png`, `{desktop,wide}-leads-drawer-validation.png`, `{desktop,wide}-invoices-drawer-unknown.png`, `{desktop,wide}-conversions-drawer.png`, `*-request-validated.png` and `mobile-leads-sheet.png`. To regenerate them:

```
cd web && RM_E03_EVIDENCE_DIR=../docs/evidence/rm-e03 npx playwright test -c playwright.seller.config.ts
```

## 10. Test results (local, Windows, Node 24)

| Check | Result |
|---|---|
| `npm run typecheck`, `npm run typecheck:e2e` | pass |
| `npm run lint`; eslint on e2e and Playwright configs | pass |
| `npm test` (vitest) | **193 / 193** pass in 17 files, including **14 new** in `seller.test.tsx` |
| `npm run build` + bundle grep and chunk-name check | pass, no dev/preview code in the production bundle |
| `playwright.seller.config.ts` (new) | **33 pass** (23 skipped by viewport gating) across seller-desktop, wide, tablet and mobile |
| `playwright.ds.config.ts` (RM-E01) | 11 pass |
| `playwright.nav.config.ts` (RM-E02, production bundle, +RM-E03 404 checks) | 9 pass |
| `go test ./docs/governance/` | pass on an LF checkout. Locally it fails under `core.autocrlf=true` (CRLF Markdown); this predates the change and also happens at the baseline, and CI on `main` is green |

### Coverage against brief §10

1 all seven render · 2 desktop navigation · 3 mobile navigation and «بیشتر» · 4 tablet layout · 5 RTL with no horizontal overflow · 6 queue filtering · 7 search · 8 pagination · 9 drawer open/close · 10 keyboard and focus (Enter, Esc with focus return, J, ۰–۶, ۱–۵, F, Ctrl+K) · 11 form validation · 12 loading/empty/error/offline/unknown states · 13 no business requests · 14 no writes · 15 grants and role bundles untouched (no file under `internal/`, `api/` or `navigation/roles.ts` changed) · 16 no DEV Seller route in the production build · 17 no synthetic identity in production (bundle grep) · 18 existing production URLs unchanged · 19 RM-E02 fail-closed navigation unchanged (its suite passes unmodified apart from the added 404 checks) · 20 R7 full-stack suites (`playwright.config.ts`, Go) untouched.

## 11. Remaining API gaps, row-scope blockers and governance dependencies

- **Read APIs.** The Seller-scoped reads have no contract: own leads and queues with counts, own conversion cases, repeat-action hand-offs, own invoices with stage-level confirmed/submitted/unknown, behaviour events, own extra-number requests, and wallet lines with a posting state. The existing `/sales`, `/billing` and `/commission` read shadows are organisation-scoped. Client-side filtering of them to fake Seller scope is **prohibited** and was not done.
- **Row scope.** «Only invoices issued by me», «leads assigned to me» and the handling of current custody versus original owner for repeat items all need server-side row scope (Route → Module → Action → Row/Field). That is **BLOCKED** on D-42 (Seller authorization) and the RM-B02 role-capability proposal.
- **Writes.** The contact outcome on a lead (`cases.contact_outcome.record` exists only as a guarded Slice 11 command), pre-invoice issuance, payment-link sending, receipt submission and conversion outcomes are **not wired**. Each needs an approved workflow contract and its idempotency/Outcome-Unknown reconciliation semantics.
- **OPD-10.** The extra-number request workflow is unresolved.
- **Wallet.** Commission entitlement, settlement and «قابل تسویه» semantics depend on finance rules (D-34 / ADR #18). Nothing is shown as settleable.
- HR and Finance runtime remain **NOT LIVE VERIFIED**.

## 12. Remaining implementation work

Wire the screens to approved Seller-scoped read contracts once D-42 and RM-B02 land, behind the existing fail-closed navigation (a real `/seller/*` route family is a product decision, not this slice). Then add the approved writes one at a time with server-decided actions. Port focus mode, and make the KPI tiles act as queue shortcuts if RM-E01 gains that pattern.
