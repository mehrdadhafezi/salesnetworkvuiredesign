# RM-P06 — MIS case-to-custody evidence continuity (read only)

Status: implemented on `claude/relaxed-fermat-uhk12n`. Base `main` = `08eb2d6`.
Authority: AGENTS.md, `MIS-PRODUCT-SPEC.md`, `GATE-0-PRODUCT-INVARIANTS.md`, `CRM-CROSS-ROLE-ARCHITECTURE-V1.md` §8, RM-P05 (`rm-p05-mis-operational-workspace.md`).
No permission, capability, role, grant, flag, status, workflow transition, eligibility rule or identity semantic is added or changed.
Group B CLOSED; R5-SEC-01, D-34, D-42 and every MIS allocation / reuse / return / identity decision stay open (**PRODUCTION NO-GO**).

## Decision: exact `case_id` on `GET /api/v1/cases`

`GET /cases/{case_id}` already exists, but using it per custody row is an N+1 (25 requests per page). RM-P05 recorded the
missing "cases-by-id list filter" as the reason custody rows showed a short id instead of the source. RM-P06 therefore adds
`case_id` to the existing list, mirroring the RM-P05 `custody/assignments?case_id=` contract:

- repeatable, 1..200 native UUIDs; exact `case_id = ANY(...)` match; duplicates collapse; unknown ids are absent (empty page ≠ denial);
- conjunctive with every existing filter; keyset cursor unchanged; invalid UUID / >200 ids / unknown parameter → 400;
- same capability (`cases.case.read`) and feature gate; cases-owned READ ONLY transaction; no cross-module SQL;
- never phone, name, legacy key or customer id (all rejected as identifiers, proven in tests).

OpenAPI (`CaseIDs` parameter), generated Go/TS clients and the `apiserver` query allowlist are updated. The RM-P01 allowlist
assertion that `/cases` must not accept `case_id` was a scope guard of that slice; it now pins the RM-P06 filter and still
refuses phone/name/search parameters.

## UI

| Page | Change |
|---|---|
| `/mis/custody`, `/mis/exceptions` (custody queues) | Column «پرونده و منبع»: source system · spec · #key, source presence (ABSENT_IN_LATEST never shown as available), full native case UUID, links to Case 360°, custody evidence and the activity timeline (`#activity`). |
| `/mis/custody/:caseId` | New card «خاستگاه و دسته‌های واردسازی»: source-row presence, first / latest observed batch id and time, raw channel / origin ref, imported / refreshed time, import-run status **unavailable**, completeness wording bound to presence. |

The case source comes only from the cases module: one batched `GET /cases?case_id=…` per visible page, sent only when the
custody response's server hint allows `cases.case.read`. States are distinct: not allowed / denied (no case data rendered, custody
evidence stays), loading, disabled, failed (UNKNOWN), and "not returned by the cases module" (UNKNOWN — never a guessed relationship).
Long evidence pills now wrap inside stacked mobile cells.

## Source / import evidence

| Evidence | State |
|---|---|
| First / latest observed batch, first / last seen, source presence | **Supported** — already part of `CaseSummary.source` (cases module). |
| Import-run status, file, batch manifest, batch completeness | **Unavailable** — `integration.import_batch` exists in staging but has no approved read API. Shown as "در دسترس نیست"; not implemented speculatively. |
| Projection freshness | Existing `/mis/reconciliation` (reporting projection runs, `reporting.mis.read`); these are reporting runs, not import runs, and are not presented as such. |

## Verification

- Go: `go vet`, full `go test ./...` with disposable PostgreSQL (incl. governance, architecture lint, OpenAPI contract), `-race` on cases/custody and RM-P05/P06.
  New `internal/integration/legacy/shadow/rm_p06_mis_evidence_test.go`: exact lookup vs stored source row, batch, duplicates, keyset, conjunctive filters,
  no widening via another Case's key, invalid identifiers, permissions, custody independent of cases, ABSENT_IN_LATEST stays exact and labelled, table fingerprints unchanged.
- Web: typecheck, ESLint, Vitest (8 new RM-P06 tests: one batched read, three links, hint-gated request, denied DOM, loading/disabled/failed/missing distinct, exceptions, provenance).
- Browser: new `TestBrowserMISEvidenceReadOverlay` (4/4) and every orchestrator of `e2e-ci.yml` (13/13). Screenshots at 1366 / 820 / 390 px in `docs/evidence/rm-p06/` with a no-horizontal-overflow assertion.

## Not done (approval-dependent)

Import-run / batch read API; a logical-Case resolver (MIS-G02 / OPD-01); Next Actor; any inventory write; a real MIS capability bundle.
MIS runtime remains **NOT LIVE VERIFIED** (overlay evidence is dev-stub only).
