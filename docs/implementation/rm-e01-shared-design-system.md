# RM-E01 — Shared CRM Design System (React)

Status: **IMPLEMENTED FOR REVIEW** (PR, not merged). Production remains **NO-GO**.
Baseline: `main` @ `3e1671955fde32abdb860db43479ce6ae78417ae` (PR #63 merged, R7 consolidated audit).
Branch: `feat/rm-e01-shared-design-system`.

Scope: presentation only. No backend, API contract, migration, database, permission, capability hint,
authentication, feature flag, grant, workflow or role-routing change. Existing module pages and the module
`Shell` are untouched; the new components are not mounted on any production route.

---

## 1. Design authority used

| Source | Status | Used for |
|---|---|---|
| `prototype/shared/CRM-DESIGN-SYSTEM-V1.md` (SN-207) | Frozen shared system (audit RM-E01 dependency: "FROZEN Seller design + SN-207") | Component contracts, states, a11y, responsive rules |
| `prototype/styles.css`, `index.html`, `app.js`, `help.css`, `help.js` (Seller V1) | **FROZEN** | Tokens, base components, shell, nav, queues, drawer, bottom nav, help |
| `prototype/shared/crm-core.js`, `crm-ext.css`, `help-engine.js`, `a11y.js` | Frozen shared runtime (built on Seller V1 + Supervisor V1) | Nav overflow, drawer anatomy, menus, stateBlock / banner, metric states, glossary core terms |
| `prototype/supervisor/*` | Frozen (Supervisor V1) | Referenced only through the shared runtime |
| `prototype/{senior,manager,deputy,mis,hr,finance}/*-DESIGN-SPEC.md` | **READY_FOR_REVIEW** (no freeze record — audit §7, NOT VERIFIED) | **Not used** as authority. Components that originate there are deferred (§8) |
| `prototype/data.js`, role mock data | Not business authority | Not used, except Seller V1 nav/queue *labels* reused as synthetic showcase copy for visual comparison |

---

## 2. Source-to-component traceability matrix

All components live in `web/src/shared/ui/ds/` (exported from `index.ts`); CSS in `ds.css`, every selector namespaced `ds-`.

| # | Component (React) | File | Frozen source | SN-207 § | Contract status | Implemented |
|---|---|---|---|---|---|---|
| 1 | `RoleNavigation` (+ priority+ overflow «بیشتر», `computeNavOverflow`) | `navigation.tsx` | `styles.css` `.nav*`; `crm-core.js` `renderNav`/`fitNav`/`openNavMore`; `crm-ext.css` overflow | §2 | Frozen (G) | ✔ |
| 2 | `QueueTabs` | `navigation.tsx` | `styles.css` `.queues/.q`; `crm-core.js` `queues()` | §8 | Frozen (G) | ✔ |
| 3 | `AppHeader`, `HeaderIconButton` (responsive header) | `Layout.tsx` | `styles.css` `.hdr/.brand/.crumb/.search/.icon-btn/.user`; `crm-ext.css` header density | §2, §5, §18 | Frozen (G) | ✔ |
| 4 | `MobileBottomNavigation` | `navigation.tsx` | `styles.css` `.bottom-nav` (≤760); `crm-core.js` mobile «بیشتر» | §2, §18 | Frozen (G) | ✔ |
| 5 | `Drawer`, `DrawerSection` (sheet ≤760, wide 660) | `Drawer.tsx` | `styles.css` `.drawer/.dr-*/.sheet-grip/.sec`; `crm-core.js` `openDrawer`/`closeDrawer`/`dtop`/`dfoot`; `crm-ext.css` `.drawer.wide` | §9, §20 | Frozen (G, M) | ✔ |
| 6 | `Menu`, `MenuButton`, `Popover` | `Menu.tsx` | `styles.css` `.menu`; `crm-core.js` `openMenu`/`placeMenu`/`menuItem` (disabled + reason) | §2, §17 | Frozen (G) | ✔ |
| 7 | `StatusIndicator`, `TonePill`, `StatePanel`, `Banner`, `CountBadge`, `Metric`, `MetricGroup` | `Status.tsx` | `styles.css` `.pill/.t-*`, `.kpis/.kpi`; `help.css` `.state*`, `.sk*`; `crm-ext.css` extra state kinds, `.ws-banner`, metric states; `crm-core.js` `stateBlock`/`banner`/`kpiHtml` | §1.3, §1.4, §7, §15 | Frozen (G) | ✔ |
| 8 | `DsAppShell`, `PageHead`, `Panel`, `PanelHeader`, `Toolbar`, `Freshness` | `Layout.tsx` | `styles.css` `.ws/.page-head/.panel/.toolbar`, `.skip-link`; `crm-ext.css` `.head-meta/.fresh/.sec-h.ph`; `crm-core.js` `pageHead`/`freshness` | §5 | Frozen (G) | ✔ |
| 9 | `ContextHint`, `Glossary`, `ShortcutList`, `HelpCenterButton` | `Help.tsx` | `help.css` `.qhint/.hc-*`; `crm-ext.css` `.terms`; `help-engine.js` | §16 | Frozen (G); content is role-owned | ✔ (tour engine deferred) |
| — | `DsIcon` (prototype stroke icon set, JSX, no HTML injection) | `icons.tsx` | `crm-core.js` `P`, `app.js` | §1.4 | Frozen | ✔ |
| — | `toFaDigits`, `countText`, `countLabel`, `isKnownCount` | `format.ts` | `crm-core.js` `fa()`/`num()` | §4, §15 | Frozen | ✔ |

Data-state vocabulary (`dataStatePresentation`): `unknown` (amber, dashed, «نامعلوم») · `conflict` (orange, «تعارض») ·
`incomplete` (amber, «ناقص») · `not_evaluated` (slate, «ارزیابی نشده») · `stale` (amber, «قدیمی») · `partial` (orange, «بخشی»).
Tones follow SN-207 §7 colour meaning; each is label + icon + tone. These are **presentation states, not business statuses**;
domain dictionaries stay in modules and the seven outcomes keep using the existing `shared/ui/outcome.ts`.

---

## 3. Design token migration report

- `web/src/shared/ui/tokens.css` already was a verbatim copy of `prototype/styles.css` lines 5–79 (R-series). RM-E01 verified it
  programmatically instead of rewriting it: **198 / 198** token declarations (80 token names × light / dim / dark / density /
  reduced-motion scopes) present with **identical values**; 0 missing, 0 different. No new token family was introduced.
- The audit's "58 vs 231" figure counted differently (lines vs declarations); the declaration-level comparison above is the reproducible one.
- `ds.css` references only these tokens (0 undefined `var()`), plus component-local custom properties (`--qc/--qbg/--qfg` queue tone,
  `--k` metric accent, `--cols` bottom-nav columns) exactly as the prototype does. One raw value remains, copied verbatim from the
  prototype's mobile sheet shadow (`rgba(0,0,0,.2)`, `styles.css` l.471).
- **Families covered:** colour (surface/text/border/primary + 9 tones × fg/bg/dot), type scale (`--t-*`, `--fs*`, `--lh`),
  radius (`--r-sm/--r/--r-lg/--r-xl`, pills 99px), spacing/sizing (`--row --cell-x --btn --btn-sm --gap --ws-pad --h-*`),
  elevation (`--shadow-1/-pop/-drawer`, scrims), motion (`--dur`, `0ms` under reduced motion), focus (`--ring`).
- **Themes / density:** `data-theme` light/dim/dark and `data-density` comfortable/compact work on any subtree (the showcase scopes them to its root).
- **Breakpoints:** ≤1200 (demo tag hidden), ≤1100 (tablet), ≤900 (crumb hidden), ≤760 (mobile: bottom nav, sheet drawer, 40px controls).
- **RTL:** drawer at the inline-end (left) edge, separators / badges positioned as in the prototype, arrow keys follow reading direction.
- **No global overwrite:** existing global classes (`.nav`, `.state`, `.who`, `.page-head` in `base.css`) are not redefined. Only two
  global additions, both absent from `base.css` and unused elsewhere: `.btn-soft` and `.btn-lg` (verbatim prototype rules).
- **Contrast** (computed from the tokens, alpha composited over `--surface`; `docs/evidence/rm-e01/contrast.json`): every text pair
  (`--text/--text-2/--muted` on `--bg/--surface/--surface-2/--raised`, each tone fg on its bg, on-primary on primary, primary on
  surface and primary-soft) is ≥ 4.5:1 in all three themes — minimum **light 4.76 · dim 4.84 · dark 4.70**; focus ring ≥ 3:1.
- **Font:** `--font: 'Vazirmatn', Tahoma, sans-serif` is kept. **No web font is bundled or loaded** in this PR: the repository has no
  font asset or licensing decision, and the prototype loads Vazirmatn from Google Fonts (an external dependency the task forbids
  without that check). The React app renders with Vazirmatn if installed locally, otherwise Tahoma. → recorded as a follow-up (§8).

---

## 4. Development-only showcase

- Route **`/dev/design-system`** (`web/src/routes/dev/DesignSystemPage.tsx`, `showcase.css`). Mounted only under the existing guard
  `import.meta.env.DEV && config.devTools` (same as `/dev/primitives`), rendered outside the module `Shell` so the full shared shell
  (header, nav, main, drawer, bottom nav) can be inspected. Linked from the DEV nav as «سامانه طراحی».
- Shows: header (search trigger, notifications, help center, user menu with a disabled-with-reason item), RoleNavigation with
  overflow, MobileBottomNavigation, QueueTabs (known 0 vs unknown), metrics (ok / stale / unavailable / undefined / unknown),
  freshness, banners, all six data states, the seven outcomes, all 13 StatePanel kinds (incl. loading), Menu, Popover, disabled
  buttons, drawer (normal, wide, loading, error, scrolling body), help hint, SN-207 core glossary, keyboard help, tone swatches,
  theme and density switches.
- Deep links for review/screenshots: `?theme=light|dim|dark&density=comfortable|compact&drawer=open|wide&drawerState=content|loading|error`.
- Synthetic content only. Every control only updates a local status line; no request is sent (unit-tested, §5).
- **Production exclusion:** verified locally — `npm run build` output contains no `DesignSystemPage`, `PrimitivesPage`, `ds-drawer`
  or showcase strings, and no such chunk. `web-ci.yml` now greps for `DesignSystemPage` and fails on any dev-only chunk name.

---

## 5. Tests and results (local, Windows, Node 22, Chromium 141 headless)

| Check | Command | Result |
|---|---|---|
| Typecheck | `npm run typecheck` | ✔ |
| E2E typecheck | `npm run typecheck:e2e` | ✔ |
| Lint | `npm run lint`; `npx eslint e2e playwright.config.ts playwright.ds.config.ts` | ✔ 0 problems |
| Unit tests | `npm test` | ✔ **152 / 152** (15 files; 119 pre-existing unchanged + 30 `ds.test.tsx` + 3 new in `App.test.tsx`) |
| Production build + leak check | `npm run build`; grep of `dist/assets` | ✔ no dev code / chunk |
| Browser checks | `npx playwright test -c playwright.ds.config.ts` | ✔ **11 / 11** (8 desktop 1366×768 + 3 mobile 390×844) |
| Visual evidence | `DS_EVIDENCE_DIR=../docs/evidence/rm-e01 npx playwright test -c playwright.ds.config.ts` | ✔ 4 / 4 (light, dark × desktop, mobile) |

Unit / interaction coverage (`web/src/shared/ui/ds/ds.test.tsx`, `web/src/app/App.test.tsx`):
- **Truthful states:** unknown / NaN / negative counts never render as 0; a known 0 is shown in queues; metric unknown, unavailable,
  undefined and "no value" render «—» + reason, never 0 (a value passed with `state="unknown"` is suppressed); six data states are
  distinct (label + icon + tone), none reads as success; StatePanel roles (alert for error/conflict, status otherwise, loading busy).
- **RTL / keyboard:** QueueTabs ← = next in RTL, → = next in LTR, Home/End, focus moves without changing selection; Menu ↑/↓/Home/End,
  wrap, ↑ on trigger opens on last item, Esc/Tab close; Popover Esc; ContextHint Esc dismiss.
- **Focus:** Drawer takes focus on open and restores it to the opener on Esc; Menu/Popover return focus to the trigger.
- **Responsive navigation:** priority+ overflow algorithm (never hides the current item), MobileBottomNavigation ≤4 primary + «بیشتر»
  with `aria-current` when the current item is in the overflow.
- **Accessibility states:** `aria-current`, `aria-pressed`, `aria-expanded`, `aria-controls`, `aria-haspopup`, `aria-disabled` +
  `aria-describedby` reason, `aria-modal="false"`, `inert` + `aria-hidden` while closed, one h1, landmarks, skip link.
- **No unintended business action:** a disabled menu item never fires; the showcase, after clicking drawer, queue, nav, menu and CTA
  controls, issued only `GET /api/v1/session` (the App-level session read) — no other request and no non-GET.
- **Routing:** showcase mounted in development, 404 in production config.

Browser checks (`web/e2e/design-system.spec.ts`, real Chromium): skip link → `main#ws`; nav `aria-current` and «—» unknown badge;
overflow at 820px keeps the current item, no nav or page side-scroll, «بیشتر» menu keyboard + Esc focus return; QueueTabs RTL arrows and
geometry (first queue is to the right); drawer at x=0 (inline-end), 560px / wide 660px, focus in, Esc out, focus restored; disabled menu
item with accessible description does nothing; help center and popover Esc / outside click; visible 2px solid focus outline; mobile:
tab row hidden, bottom nav pinned to the bottom edge, «بیشتر» menu navigates and marks «بیشتر» current, drawer is a full-width bottom
sheet with grip, touch targets ≥ 40px, no page-level horizontal scroll.

CI: `web-ci.yml` gains a `design-system` job (Chromium install, e2e typecheck + lint, `playwright.ds.config.ts`). It starts only a
local Vite dev server. The Go full-stack orchestrators and `playwright.config.ts` projects are unchanged.

---

## 6. Visual comparison evidence (`docs/evidence/rm-e01/`)

Same viewport, same theme, prototype opened from `prototype/index.html` (`view=leads&queue=none`, welcome overlay off, demo
comparison bar removed, external font request blocked so both sides use the same local fallback font).

| File | What |
|---|---|
| `prototype-{1366,390}-{light,dark}.png` / `react-…png` | Shell + nav + metrics + queues |
| `prototype-…-drawer.png` / `react-…-drawer.png` | Drawer (desktop) / bottom sheet (mobile) |
| `react-…-full.png` | Full showcase page |
| `parity-{1366,390}-{light,dark}.json` | Computed-style comparison of matched elements |
| `contrast.json` | Token contrast ratios per theme |

Computed-style parity (height, background, colour, font size/weight, radius, inline padding, border colour; plus width for drawer parts)
of matched prototype ↔ React elements: **1366 light 15/15 · 1366 dark 15/15 · 390 light 13/13 · 390 dark 13/13 identical**
(header, role nav, current/normal nav item, queue bar, pressed/normal queue, metric strip, metric, page title, brand mark, search
trigger, drawer, drawer top bar, drawer footer; mobile: bottom nav, current bottom item, sheet, grip). Elements not rendered at a
width (e.g. tab row on mobile) are reported as n/a. Two differences found during comparison were fixed: brand-mark weight (600 → 700)
and a 6px drawer radius leaking from the global `:focus-visible` rule.

**Not claimed:** pixel-perfect parity. Remaining known visual differences:
- Content differs by design (showcase content is synthetic; tables / lead cards are RM-E03).
- Web font: both sides were compared on the fallback font; with Vazirmatn loaded the prototype's glyph metrics differ from React's fallback until the font decision (§8).
- No screenshot pair is committed for the dim theme or for tablet widths (1024/768); tablet behaviour is covered only by the 820px nav-overflow browser test and the token/contrast check covers dim.

---

## 7. Accessibility findings

Implemented / verified: landmarks (`header`, labelled `nav` ×2, single `main#ws`, dialog drawer), skip link, one h1 per view (on
phones the h1 is visually hidden but stays in the accessibility tree — the prototype used `display:none`), status = label + icon + tone,
live/alert roles, `aria-current` / `aria-pressed` / `aria-expanded` / `aria-controls`, disabled items explained (not hidden), visible
2px `--ring` focus, Esc closes every layer with focus return, reduced motion (`--dur:0ms`, static skeleton), forced-colours borders,
touch targets ≥ 40px on mobile, contrast ≥ 4.5:1.

Deviations from the prototype made for accessibility (presentation only):
1. Context-hint icon colour `--faint` → `--muted` (prototype `--faint` is < 4.5:1 on surfaces).
2. Drawer always takes focus on open (prototype only for keyboard openers) so screen-reader users land in it; Esc still restores focus.
3. Mobile «بیشتر» is a `button` with `aria-haspopup="menu"` (prototype used an `<a href="#more">`).
4. Mobile page h1 visually hidden instead of `display:none` (keeps one h1 for assistive tech).

Open findings (not blockers for a design-system PR): no automated axe scan in CI (RM-E09); the drawer is non-modal by contract, so
focus is not trapped (by design, SN-207 §9); tooltip is hover/focus only (no touch long-press).

---

## 8. Deferred / not implemented

| Component | Reason | Next |
|---|---|---|
| `DimensionSwitcher`, `AttributionBasis`, `MetricCoverage` (+ BranchCoverage matrix), `LinkConfidence` / `SourceBadge`, `ScopeBreadcrumb` | Originate in Senior / Manager / Deputy / MIS designs — **READY_FOR_REVIEW**, no freeze record; audit RM-E04 = NEEDS DESIGN FREEZE. CSS exists in `crm-ext.css` but the data contracts (grain, basis, coverage states, lineage ladder) are not frozen | RM-E04 after design freeze |
| `FinancialStateLayers` | Finance design READY_FOR_REVIEW; Finance runtime NOT LIVE VERIFIED | RM-E07 after freeze |
| `OutcomeUnknownContext` (domain `oux` block) | Finance / MIS / HR variants, all READY_FOR_REVIEW. The generic *unknown* state (dashed amber, "reconcile before retry") **is** implemented in `StatusIndicator` / `StatePanel` / `Banner` | RM-E04 / RM-E07 |
| Command palette (Ctrl/⌘+K) | Frozen G pattern, but needs the destination registry produced by role routing; the header exposes the trigger (`search.onOpen`) | RM-E02 |
| Help tour engine / per-role tours, welcome dialog | Frozen behaviour, but tours are role content and need role routes; primitives (hint, glossary, help center, shortcuts) are in place | RM-E02 / RM-E03 |
| DataTable / SelectableDataTable / tfoot / stacked cards, BulkActionBar, BulkResult, StepIndicator, Timeline, OwnershipGrid, SensitiveActionConfirmation, ApprovalChain | Frozen G/D, but not in the RM-E01 list; they carry ownership / bulk / approval vocabularies that need the RM-E03 page contracts | RM-E03 (Seller tables) and later |
| Toasts, offline chip / bar | Frozen G; no consumer yet | With RM-E02 shell integration |
| Web font (Vazirmatn) | No font asset or licence decision in the repository; no external font loaded | Product / design decision (self-host OFL font or keep Tahoma) |
| Replacing the module `Shell` / role routing | Out of scope by contract | RM-E02 |

---

## 9. Changed files

- `web/src/shared/ui/ds/` — **new**: `index.ts`, `ds.css`, `icons.tsx`, `format.ts`, `Status.tsx`, `Menu.tsx`, `navigation.tsx`, `Drawer.tsx`, `Layout.tsx`, `Help.tsx`, `ds.test.tsx`.
- `web/src/routes/dev/DesignSystemPage.tsx`, `showcase.css` — **new** (DEV-only).
- `web/src/app/routes.tsx` — DEV-only nav link + top-level DEV route (same build-time guard as `/dev/primitives`).
- `web/src/app/App.test.tsx` — 3 new tests (existing tests unchanged).
- `web/e2e/design-system.spec.ts`, `web/e2e/design-system-evidence.spec.ts`, `web/playwright.ds.config.ts` — **new**.
- `web/e2e/tsconfig.json` (include the new config), `web/e2e/env.d.ts` (one `node:fs` declaration for the evidence report).
- `.github/workflows/web-ci.yml` — bundle check also covers `DesignSystemPage` and dev chunk names; new `design-system` job.
- `.gitignore` — `web/test-results-ds/`.
- `docs/implementation/rm-e01-shared-design-system.md` (this file), `docs/evidence/rm-e01/*` (screenshots, parity, contrast).

Not changed: `tokens.css`, `base.css`, `primitives.tsx`, `icons.tsx`, `outcome.ts`, module pages, `Shell.tsx`, API client, generated
schema, backend, migrations, workflows other than `web-ci.yml`.

---

## 10. RM-E02 readiness

RM-E02 (role-entry routing) can start on top of this PR once it is merged: `RoleNavigation` / `MobileBottomNavigation` accept any
destination list plus a `renderLink` (router links) and `onNavigate`, and present only what they are given — RM-E02 must derive that
list from the server's capability hints (`allowed_actions` / features), never from a client-side role guess. `DsAppShell` /
`AppHeader` are ready to replace the module shell chrome behind that work. RM-E02 and RM-E03 are **not** started here.
