# RM-P05 — MIS operational workspace and case custody (read only)

Status: implemented on `claude/busy-pasteur-ycgtto`. Base `main` = `fa8f0e0`.
Authority: AGENTS.md, `MIS-PRODUCT-SPEC.md` (§7–§10, §14, §16), `GATE-0-PRODUCT-INVARIANTS.md`, `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` §8,
`prototype/mis/MIS-DESIGN-SPEC.md` (**READY_FOR_REVIEW — not frozen; this work does not change that status**).
No permission, capability, role, grant, flag, status, workflow transition, eligibility rule, financial rule or identity semantic is added or changed.
Group B, R5-SEC-01, D-34, D-42 and production readiness are untouched (**PRODUCTION NO-GO**).

## Found before implementation (main `fa8f0e0`)

| Area | Existing contract |
|---|---|
| Case inventory | `GET /cases` (workflow_type, state_mapping, customer_link_state, presence, customer_id; keyset on case_id), `GET /cases/{id}`, `/history`, `/attribution` — `cases.case.read` |
| Presence | `PRESENT` / `ABSENT_IN_LATEST` on cases, sources, aliases, custody projections and findings; absence set only by a COMPLETE import, never deletes |
| Custody | `GET /custody/assignments` (parity, derivation, presence), `GET /custody/cases/{id}` (projection + append-only events + findings) — `custody.custody.read`. Parity: MATCH, HISTORY_GAP, LEGACY_UNKNOWN, TARGET_UNKNOWN, CONFLICT, MULTIPLE_CURRENT, NOT_RECOVERABLE. Eight finding kinds |
| Attribution | Immutable snapshots ORIGINAL_OWNER / ORIGINAL_SELLER / CREDIT_OWNER, frozen at import, never recomputed from today's hierarchy |
| Aggregates | `GET /sales/coverage-report`, `GET /sales/custody-parity` — `sales.report.read`; `GET /mis/overview`, `/mis/source-quality`, `/mis/reconciliation` — `reporting.mis.read` |
| Write commands | Only `POST /cases/{id}/contact-outcomes` (seller, own held lead, flag `cases.contact_outcome_write`). **No** assign / transfer / return / reset / release / reallocate / recycle / finalize route. `internal/custody/release` is a pure gate evaluator without a route that never returns SUCCESS while ADR-01 / F01 (and OPD-03 for reset/release) are open |
| Eligibility / reservation | No eligibility, reservation or reuse state exists in the target model (claimgate is the ADR-01 CANDIDATE invoice-claim half only) |
| Permissions | MIS role carries **no** capability (fail closed for every real account); flags `sales.read_shadow`, `reporting.read_shadow` |
| UI | `/mis` (overview), `/mis/source-quality`, `/mis/reconciliation`; `/sales/cases`, `/sales/cases/:id` (Case 360° incl. RM-P03 timeline), `/sales/custody` |

## Workspace

| Section | Route | Reads |
|---|---|---|
| امروز MIS (landing) | `/mis` (existing route, new page) | coverage + parity aggregates, existing MIS overview unchanged below |
| موجودی پرونده‌ها | `/mis/cases` (new) | `GET /cases` with every server filter + **new** exact `source_key`; custody of the visible page via **new** `case_id` filter |
| نگهداری و انتساب | `/mis/custody` (new) | `GET /custody/assignments` with parity / derivation / presence + **new** `finding_kind`, `holder_link` |
| شواهد نگهداری پرونده | `/mis/custody/:caseId` (new) | case, custody (projection, events, findings), attribution — each authorized on its own |
| استثناها و تطبیق | `/mis/exceptions?type=` (new) | 20 queues, each = an existing stored state + existing server filter |
| کیفیت منبع / تطبیق و پوشش | existing | unchanged, now with the section navigation |

Section navigation is gated through the **existing** destinations (`mis.today`, `sales.cases`, `sales.custody`, `mis.quality`,
`mis.reconciliation`): server feature switch AND ALLOWED decision. No global destination was added, so a principal that only reads cases
is not offered an MIS link. Case 360° gained a `#activity` anchor on its timeline section (deep link from MIS).

### Counters (every one has a named source)

| Counter | Source | Scope |
|---|---|---|
| همه / حاضر / وضعیت نگاشت‌نشده | `coverage-report · cases_total / cases_present / cases_with_unknown_state_mapping` | all / present / all |
| غایب در آخرین واردسازی | `cases_total − cases_present` | all |
| پیوند مشتری متعارض/نامعلوم/غیرقابل بازیابی | `coverage-report · customer_link_states.*` | present |
| پرونده بدون منبع، پیوند متعارض | `coverage-report · sourceless_cases`, `aliases_by_resolution.CONFLICT` | count only — no list contract (stated) |
| پروجکشن‌ها، MATCH، رویدادها، پرونده با رویداد | `custody-parity · assignments_present, current_holder_parity.MATCH, custody_events, cases_with_custody_events` | present / all events |
| صف‌های تطابق نگهدار | `custody-parity · current_holder_parity.<P>` (list adds `presence=PRESENT` to match) | present |
| صف‌های یافته | `custody-parity · findings_by_kind.<K>` — **number of findings, not cases** (stated) | present findings |
| هویت نگهدار حل‌نشده | **none** — shown "در دسترس نیست", never 0 | — |

A denied / disabled / failed source shows "—" with the reason; counters are not a partition and are never summed.

## Backend (narrow, read only)

1. `GET /api/v1/custody/assignments?case_id=…` — repeatable, 1..200 UUIDs; projections of exactly those Cases (unknown id = empty, never a holder).
2. `GET /api/v1/custody/assignments?finding_kind=` — projections with ≥ 1 PRESENT finding of that kind.
3. `GET /api/v1/custody/assignments?holder_link=UNRESOLVED` — a derived or legacy holder key exists with no resolved identity user.
4. `GET /api/v1/cases?source_key=` — exact LegacyKey match of the Case's own source row (provenance, never identity; no prefix/phone/name).

All: OpenAPI + generated Go/TS clients, query allowlist in `apiserver`, module-owned READ ONLY transactions, no cross-module SQL,
same capabilities as before. Proven by `internal/integration/legacy/shadow/rm_p05_mis_workspace_test.go`.

## Operations

Assign, transfer, return, reset, release, reallocate, reuse/recycle and finalize are listed as **unavailable** with the grounded reason
and blocking decisions (F01, ADR-01, OPD-03/04, MIS-G01/G07, GATE-0 INV-006…010, STATUS-TRANSITIONS). No button, link or disabled
placeholder exists for any of them. `ABSENT_IN_LATEST` never becomes eligible, free, released or reusable (tested in Go and UI).

## Verification

- Go: `go vet`, full `go test ./...` with disposable PostgreSQL (unit + integration), `-race` on cases/custody and RM-P05, architecture lint, OpenAPI contract test, governance.
- Web: typecheck, ESLint, Vitest 380/380 (21 new in `web/src/modules/mis/mis.test.tsx`), OpenAPI lint/bundle/regenerate (no drift), e2e typecheck/lint, build.
- Browser: new `TestBrowserMISWorkspaceReadOverlay` (5/5) plus every existing orchestrator of `e2e-ci.yml` (all pass). Screenshots at 1366 / 820 / 390 px in `docs/evidence/rm-p05/` with a no-horizontal-overflow assertion.

## Not done (approval-dependent, not invented)

Any inventory write (above); a logical-Case resolver (MIS-G02 / OPD-01); Next Actor; an unresolved-holder aggregate; a cases-by-id list
filter (custody rows show a short case id + links instead of the source key); import runs / files / batches (no target contract);
a real MIS capability bundle. MIS runtime remains **NOT LIVE VERIFIED** (overlay evidence is dev-stub only).
