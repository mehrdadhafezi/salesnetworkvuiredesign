# CRM Design System V1 (SN-207)

Status: DESIGN SYSTEM / CROSS-ROLE UX INTEGRATION · **no** plugin / backend / database / API / permission / capability / workflow / KPI / status change.
Business authority: `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md` → `GATE-0-PRODUCT-INVARIANTS.md` → frozen role Product Specs. Design authority: Seller V1 baseline + the seven role designs SN-201…SN-206 (each `prototype/<role>/*-DESIGN-SPEC.md`).
Scope rule (AGENTS.md): *a shared component may introduce a reusable presentation pattern; it must not introduce a business capability.* Mock data is not business authority.
Runtime: **HR and Finance remain NOT LIVE VERIFIED** (no dedicated real role accounts tested). Nothing here changes that.

How the system is built: Seller keeps its frozen standalone runtime (`../app.js`, `../help.js`). Supervisor, Senior, Manager, Deputy, MIS, HR and Finance run on `shared/crm-core.js` + `shared/crm-ext.css` + `shared/help-engine.js` + `../styles.css` (tokens + base components). Role files supply **data, dictionaries and domain views only**; role CSS files add no tokens, no raw colours.

---

## 1. Design principles
1. **One product, many scopes.** Same shell, same words, same states; roles differ in scope and domain, not in visual language.
2. **Presentation never grants authority.** Visibility ≠ write; hierarchy ≠ permission; position ≠ role ≠ capability ≠ credential. Bulk UI, buttons and menus never imply a permission (disabled/explained, never hidden-and-implied).
3. **Truthful states.** Unknown / Incomplete / Conflict / Stale are first-class and never rendered as zero, success, or "none".
4. **Status is never colour-only** — every status = label + icon + tone.
5. **Four distinctions are always visible where relevant:** requested ≠ applied · current ≠ historical · event time ≠ posting time ≠ gather time · Custody ≠ Original Owner ≠ Next Actor ≠ Event Actor ≠ Credit Owner.
6. **Reuse before invent.** New shared pattern only with proven multi-role reuse.

## 2. Navigation
`RoleNavigation` (GLOBAL SHARED): one `<nav aria-label>` in the header, groups separated by `nav-sep`, priority+ overflow into «بیشتر» (current tab always visible), bottom nav ≤760px with «بیشتر» sheet. Depth may differ by role (Seller flat tabs; Supervisor/Senior grouped; Manager/Deputy broader groups; MIS/HR/Finance domain groups with a `cond` secondary group) — interaction language (tab = link with `aria-current="page"`, count badges, `alert` tone for needs-action) is identical. Command palette (Ctrl/⌘+K), user menu, theme/density/focus switches, Help button are in every shell. `AppShell` also exposes a **skip link** (SN-207) to the `#ws` main landmark.

## 3. Tokens
Single token source: `../styles.css` `:root` / `[data-theme]` (Light / Dim / Dark) and `[data-density]` (Comfortable / Compact), `[data-focus="1"]` (Focus Mode). Families in use and **frozen**: colour (surface/text/border/primary + 8 semantic tones `teal green amber orange red blue violet slate`, each `-bg/-fg/-dot`), type scale (`--t-*`, `--fs*`), radius (`--r-sm --r --r-lg`, pills 99px), spacing (4/8/12/16 grid), elevation (`--shadow*`), motion (`--dur`, `0ms` under `prefers-reduced-motion`), focus ring (`--ring`). **No new token families in SN-207.** Role CSS: 0 token declarations, 0 raw colour values (verified).

## 4. Typography
Vazirmatn (+Tahoma fallback), RTL, Persian digits for display numerals (`fa()`/`num()`), tabular `.num` for amounts. Amounts always carry a unit (`money()`); unit unknown → «واحد نامشخص», never guessed.

## 5. Layout
`header` (brand · role crumb · search/palette · pref/user) → `nav` → `main#ws` (PageHead → banners → KPI/queues → toolbar → table/cards → tfoot) → `aside#drawer` → `#bottom-nav` (mobile). One `<h1>` per view. No page-level horizontal scroll at any width (verified 1920–390).

## 6. Identity / Ownership
- **Identity:** `CaseRef` / `SourceRef` / native ID shown as primary; phone is *discovery data only* (masked, copy-only, never a join/identity chip). Linkage is a 4-state `LineageConfidence`: **Verified · Incomplete · Conflict · Unknown** (label + icon; Unknown/Conflict never fall back to phone matching). Alias shown as alias, never as canonical.
- **OwnershipContext** (`OwnershipGrid` + `X.who`/`nextActor` captions): fixed vocabulary — مسئول فعلی (Current Custody) · مالک اولیه (Original Owner) · اقدام بعدی با (Next Actor) · عامل رویداد (Event Actor) · مالک اعتبار (Credit Owner) · مالک منبع (Source Owner, MIS/Deputy only). Roles *show only the cells relevant to them* but never merge two cells into «مالک». Current hierarchy never relabels historical attribution.

## 7. Status / System states
One `stateBlock(kind)` + `banner(kind)` family (`role=status|alert`, icon + title + explanation + recovery action):

| Kind | Tone | Meaning (presentation) |
|---|---|---|
| loading (skeleton) · empty · noresult | neutral | no data yet / nothing exists / filter excluded everything |
| unauthorized (`locked`) | slate | not permitted — explained, no hidden controls implied |
| stale | amber | data older than freshness window; actions re-check at commit |
| incomplete | amber | partial coverage — never a final total, never 0 |
| conflict | orange | sources disagree — reconcile before acting |
| partial | orange | some items applied, some not — see BulkResult |
| retry (known failed, retryable) | red | safe to retry |
| unknown (Outcome Unknown) | amber, dashed/`oux` | may have been applied — **reconcile before any retry** |
| offline | slate | actions paused |
| blocked · conditional · restricted · needs-validation | red / violet / slate / amber pills | status pills from role dictionaries; same tone ⇒ same meaning |

Colour meaning is global: green = completed/verified · teal = ready/eligible · amber = needs attention/unknown · orange = conflict/partial · red = failed/blocked · violet = draft/conditional · slate = restricted/inert · blue = informational.

## 8. Tables / Lists
`.tbl` (sticky head, row cursor, arrow-key navigation, `data-row` opens drawer), `SelectableDataTable` (checkbox column, `cbx`), `toolbar` (search + filter chips + right slot), `tfoot` (shown/total + pager), `queues` (pressed-chip facets with counts). Collapse rule: priority columns at ≤1100, stacked cards at ≤760 (`.tbl.stackable`). **SN-207 semantics:** every `#ws` table gets `th[scope=col]` and an accessible name automatically (`shared/a11y.js`), checkbox targets ≥24px and non-shrinking.

## 9. Drawers / Sheets
`aside#drawer` (`role=dialog`, non-modal, labelled): 560px (wide 660px for review/result/finance context) ≥1024; **bottom sheet** with grip ≤760. Esc closes and restores focus to the opener; keyboard-opened drawers take focus. Anatomy via shared `dtop(title, extra)` / body / `dfoot(primary, secondary, hint)`.

## 10. Forms
Shared `.field`/`.input` (label above, helper text, inline error with icon + text, 3px focus ring), `StepIndicator` (`steps()`, `aria-current="step"`), `confirm` checkbox row, reason textarea. Placeholder text uses `--muted` (≥4.5:1). Required reason fields are labelled, not asterisk-only.

## 11. Request / Review
Shared grammar: **Requester · Subject · Current Reviewer · State · Reason · History · Result · Next Actor**, rendered with `ApprovalChain`, state pill, `timeline`. Domains keep their own flows and vocabulary: HR change request, Extra-number request, Finance Review (stage-level), Reconciliation issue. Approved ≠ Applied is always two visible facts.

## 12. Bulk actions
`BulkActionBar` (`bulkbar()`; region, count + scope sentence + actions) → eligibility/preview → `BulkResult` (`outcomeStrip(counts, opt)`): **Requested · Eligible · Applied · Skipped · Rejected · Failed · Unknown**; the sum of the post-eligible cells must equal Requested or the note flips to a warning. Domain relabelling is allowed only through `opt.cells` (Finance: applied = «ثبت‌شده», extra cell «تراکنش/تصمیم موجود», no bulk «ردشده»). Unknown ⇒ reconcile, never blind retry.

## 13. Sensitive actions
`SensitiveActionConfirmation` = `ConfirmStep` panel with fixed rows: **Target · Current state · Proposed change · Impact · Dependencies · Reason (required) · Expected next state · Audit context**, plus unchecked acknowledgement. Used for financial action, HR change, credential action, maintenance, destructive source action, bulk high-impact. Never a bare «مطمئنید؟». Prototype confirmations record nothing.

## 14. Temporal UI
Time labels are fixed: **زمان رویداد (Event Time) · زمان اعمال (Applied At) · تاریخ اثر از/تا (Effective From/To) · زمان ثبت/Posting Time · زمان جمع‌آوری (Gather Time)**. Chips: *Current* (ساختار فعلی) vs *Historical* (انتساب تاریخی); *Requested* vs *Applied*. A historical as-of view is offered only where the data supports it; otherwise «تاریخچه ناقص / نامعلوم».

## 15. KPI / Report trust
`MetricMeta` row on every KPI/report: **grain · scope · source · cohort · time basis · unit · freshness · coverage · version · reconciliation state** (chips `basis`, `grain`, `cov`, `lc`, `fresh`). Unavailable / not-counted / undefined renders as «—» with reason, **never 0**. `MetricCoverage`/`BranchCoverage` matrix shows which sources are counted.

## 16. Help
One engine (`shared/help-engine.js`; Seller's `help.js` is the same behaviour): Help Center · welcome · tour registry (per-role tours; tours never execute actions) · shortcut help · glossary · context hints (`hint()`). Role content stays in `help-content.js`. **SN-207:** a core glossary (Current Custody, Original Owner, Next Actor, Event Actor, Credit Owner, Outcome Unknown — wording reused verbatim from frozen role glossaries) is appended to any role glossary that lacks it; role entries always win.

## 17. Accessibility
Keyboard: Ctrl/⌘+K palette, `?` shortcuts, arrow row cursor, Esc closes drawer/menu/palette and restores focus; focus ring `--ring` 2px; skip link; one `h1`; landmarks `header / nav[aria-label] / main / aside`; tables scoped + named; status icon+label; live regions (`role=status|alert`, toasts); reduced motion honoured; dialogs labelled; contrast ≥4.5:1 text (disabled controls exempt). Hit targets ≥24px (WCAG 2.2 AA) desktop, 40px controls on touch (`--btn:40px` ≤760).

## 18. Responsive rules
1920 / 1366: full table, wide drawer. 1024: priority columns, nav overflow. 768: stacked cards, header trimmed (demo tag → crumb), sheet drawer. 390: bottom nav, one-column, drawer footer (`dr-foot`) pinned under a scrolling `dr-body`, financial/analytic tables become labelled cards (no side-scroll at page level; wide matrices scroll inside their own container only).

## 19. Component registry
Classes: **G** GLOBAL SHARED · **D** DOMAIN SHARED (shared markup/CSS, role/domain supplies data) · **R** ROLE-SPECIFIC · **C** CONDITIONAL (shown only when the role/domain applies) · **M** merged in SN-207 · **X** deprecated.
Implementation lives in `crm-core.js` (JS `C.h.*`), `crm-ext.css` (CSS), `styles.css` (base), `help-engine.js`, `a11y.js`.

| Component | Class | Where | Notes |
|---|---|---|---|
| AppShell (header, nav, ws, drawer, palette, toasts, bottom-nav) | G | crm-core / styles | Seller has the frozen standalone twin |
| SkipLink + table semantics | G (new) | a11y.js | all 8 roles |
| RoleNavigation (+ overflow) | G | crm-core | depth varies by role |
| PageHead, queues, toolbar, tfoot, DataTable, SelectableDataTable | G | crm-core / crm-ext | |
| Drawer/Sheet (+ `dtop`, `dfoot`) | G, **M** | crm-core | 6 identical copies → 1 |
| StepIndicator (`steps`) | G, **M** | crm-core | 6 copies → 1 |
| BulkActionBar (`bulkbar`) | G, **M** | crm-core | 6 copies → 1 |
| BulkResult (`outcomeStrip`) | G, **M** | crm-core | 5 copies → 1 (+ Finance vocabulary via `opt.cells`) |
| Timeline (`timeline`) | G, **M** | crm-core | 2 variants → 1 superset (optional 3rd field) |
| stateBlock / banner / Freshness / Metric states | G | crm-core / crm-ext | |
| Pill / status dictionaries (`X.*` label+tone+icon) | G pattern, R data | lib.js | dictionaries stay per role |
| MetricMeta / MetricCoverage / FreshnessIndicator (`basis grain cov lc fresh`) | D | crm-ext + lib.js | content per role |
| OwnershipContext / OwnershipGrid | D | crm-ext + lib.js | cells chosen per role |
| IdentityContext / SourceContext / LineageConfidence | D | crm-ext (+MIS lineage ladder) | |
| ScopeBreadcrumb (`crumb`) | D | senior/manager/deputy lib | tree-scoped roles only |
| ApprovalChain / RequestReview | D | crm-ext | HR/Extra-number/Finance/Reconciliation variants |
| SensitiveActionConfirmation (ConfirmStep, EligibilityPanel) | D | crm-ext | |
| OutcomeUnknownContext (`oux`) | D | crm-ext (Finance, MIS, HR) | |
| AttentionList, ReportCatalog/ReportView, FinancialFacets, CoverageMatrix | D | crm-ext | |
| Help engine + glossary + tours | G | help-engine | content R |
| PerformanceExplorer | C | senior/manager/deputy | leadership roles only |
| Reconciliation workspace, Ledger explorer, Lineage inspector, Access comparison, Compensation history | R | MIS / Finance / HR | domain-specific, not promoted |
| `.linklike` (Deputy) | **X** | — | merged into `.linkish` |
| Demo switcher / sim controls | R (DEMO-ONLY) | per index.html | not product UI |

## 20. Role-specific exceptions (justified)
- **Seller:** keeps standalone runtime and flat tabs (frozen V1); shares tokens, CSS base, help behaviour; receives only the shared skip link, table semantics and contrast fixes.
- **Finance bulk vocabulary:** «ثبت‌شده» instead of «اعمال‌شده», extra «موجود» cell, no «ردشده» — the act is Ruled Posting (Finance spec); implemented through `outcomeStrip` options, not a fork.
- **HR/Finance:** carry the permanent **NOT LIVE VERIFIED** tag; Finance additionally shows the OPD-06 unit-unknown block.
- **MIS:** 7-cell ownership grid (adds Source Owner, Recipient), lineage ladder.
- **Wide drawers** (660px) for review/result/finance-context kinds only.
- Per-role `crumb`, `banners`, `freshPart`, `basis`, `grain`, `cov`, `who`, `checks`, `chain`, `sec`, `details` helpers stay in role `lib.js` (same markup contract, different data) — see §21.

## 21. Deferred design items
- Extract the remaining *similar-not-identical* helpers (`banners`, `freshPart`, `basis`, `grain`, `cov`, `who`, `checks`, `chain`, `sec`, `details`, `crumb`) into one parametrised component each — needs real data contracts → **implementation foundation**.
- Machine-readable component registry / Storybook-style catalogue.
- Seller runtime migration onto `crm-core.js` (Seller stays frozen until implementation).
- Shared glossary content beyond the six core terms (domain terms remain role-owned).
- Live-account verification of HR and Finance (NOT LIVE VERIFIED).

## 22. Cross-role QA
See PR description and `Appendix B` below. Passes: A (shell/components/tokens), B (families), C (final regression matrix).

---

## Appendix A — Phase 1 inventory & contradictions

| # | Area | Finding | Classification |
|---|---|---|---|
| 1 | Shell | 7 roles on one core; Seller standalone twin; identical landmarks/ids | GLOBAL SHARED / KEEP |
| 2 | Navigation | Same markup & overflow; depth differs by role | GLOBAL SHARED / KEEP (depth ROLE-SPECIFIC) |
| 3 | Page headers, search, filters | `pageHead`, `toolbar`, `queues` shared | GLOBAL SHARED |
| 4 | Tables | shared `.tbl`; **semantics inconsistent** (captions/scope only in 4 roles) | STANDARDIZE → `a11y.js` |
| 5 | Drawers | identical behaviour; `top/foot` copied 6× | DUPLICATE → MERGE (`dtop/dfoot`) |
| 6 | Forms / steps | `steps` copied 6× | DUPLICATE → MERGE |
| 7 | Status / states | one `stateBlock` family; tone meaning consistent | GLOBAL SHARED / KEEP |
| 8 | Identity / lineage | shared 4-state lineage; MIS ladder is richer | DOMAIN SHARED / KEEP |
| 9 | Ownership | same fixed vocabulary; each role shows relevant cells | DOMAIN SHARED / KEEP |
| 10 | Request/Review | shared grammar, 4 domain variants | DOMAIN SHARED / ROLE-SPECIFIC flows |
| 11 | Bulk | `bulkbar` 6×, `outcomeStrip` 5× identical; Finance vocabulary differs | MERGE + ROLE-SPECIFIC (options) |
| 12 | Sensitive action | ConfirmStep rows consistent | DOMAIN SHARED / KEEP |
| 13 | Timeline/Audit | `tl` two variants | MERGE (superset) |
| 14 | KPI trust | chips shared; content per role | DOMAIN SHARED / KEEP |
| 15 | Help/Tour | one engine; **core glossary terms missing in some roles** | STANDARDIZE (core glossary) |
| 16 | Theme/Density/Focus | all 8 honour `theme/density/focus` | GLOBAL SHARED / KEEP |
| 17 | Responsive | no page-level h-scroll in any role/view/width (8 roles × 5 widths × all nav views) | KEEP |
| 18 | A11y | no skip link; placeholder & pressed-count contrast 3.6–3.7:1; dark `btn-soft` 4.49:1; checkboxes 13–20px and shrinking in flex; `.linklike` ≈ `.linkish` | STANDARDIZE / MERGE (fixed once, shared) |
| 19 | Terminology | «مسئول فعلی / مالک اولیه / اقدام بعدی با / عامل رویداد / مالک اعتبار» consistent; Finance «ثبت‌شده» | KEEP / ROLE-SPECIFIC |
| 20 | Similar-not-identical helpers (`banners`, `freshPart`, `basis`…) | per-role variants | DEFERRED |

## Appendix B — QA evidence
Recorded in the PR (staged Pass A/B/C; automated structural/contrast/overflow/target/focus sweeps + representative screenshots).
