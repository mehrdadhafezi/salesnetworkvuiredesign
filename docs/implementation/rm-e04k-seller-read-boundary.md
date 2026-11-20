# RM-E04K — Seller S1 read boundary & interactive data states

Baseline: `main` @ `814554f570c929278a6515820b433e01f2c78605` (PR #83 merged).
Scope: DEV/TEST-only Seller preview (`/dev/seller/leads`), SYNTHETIC fixtures. **Presentation increment only — not an authorized Seller read.**

## What changed

S1 no longer imports fixtures directly. Every S1 record arrives through a small, typed read boundary:

| Read | Method | Synthetic answer |
|---|---|---|
| List | `listCases()` | whole synthetic set as one page; `count` EXACT / ABSENT; `nextPage: null` (no pagination contract exists) |
| Detail | `getCase(id)` | record + invoices from the explicit `invoiceRefs` only (never a phone join); missing ≡ out-of-scope → one generic 404 |
| History | `listCaseHistory(id)` | events + `complete`; PARTIAL truncates (synthetic demonstration) and the UI shows «—» as the count |
| Contact outcomes | `listCaseContactOutcomes(id)` | `UNKNOWN`, `available: false` — fixtures hold no outcome records and status is not turned into invented history; not rendered |

- `web/src/modules/seller/data/s1ReadSource.ts` — `S1ReadSource` interface, **preview presentation types** (not transport DTOs), the synthetic source, `ReadSequencer`.
- `web/src/modules/seller/data/s1Reads.tsx` — `useS1ReadSource` (synthetic source per `?sim=` scenario, or an injected source) and `useSequencedRead`.
- `SellerAnswerScope` (in `SellerPreviewContext.tsx`) re-provides the existing A7 `ui` from the answer S1 actually received, so `ScreenFrame`, `Restricted`, status pills, counts and `NotFoundDrawer` are reused unchanged.
- `LeadsPage.tsx` — list / detail / history are separate sequenced reads; loading panels for list, drawer and history; inline history failure with a history-only retry.

The proposed `listMyCases` / `getMyCase` / `listMyCaseHistory` / `getMyCaseContactOutcomes`, `cases.own.read` and `CUSTODY_CURRENT_HOLDER` are **not** approved and are not called, mirrored or implemented. The org-scoped `/cases` endpoints are not used. No OpenAPI, client, backend or CI file changed.

## Lifecycle and ordering

- **Loading**: no rows, counts or drawer until the list answer; pending detail shows a loading drawer with no record content; `role=status aria-busy`.
- **Ordering**: `ReadSequencer` aborts the previous request and applies only the newest generation; the hook compares against the current key at render, so the first frame after a change is already `loading`.
- **Fail closed**: a 401 / 403 / FEATURE_DISABLED answer on any S1 read latches the screen closed for that source; a late success cannot restore content. A new scenario is a new source (all reads restart).
- **Drafts**: the RM-E04J body is keyed by case and mounted only while its detail answer is current — case change, scenario change or a blocking answer unmounts it (draft and simulated result discarded). `validateOutcome` and the draft reducer are unchanged; `cases.contact_outcome.record` is not wired.
- **Failure**: a rejected read is a controlled failure, never an empty list. Retry repeats only reads. `?sim=offline` on S1 now presents a network failure (previously rows stayed visible under the offline banner).

Synthetic latency: 450 ms in the DEV preview, 0 under Vitest. `?sim=loading` never settles (pending demo).

## Tests

- `data/s1ReadSource.test.ts` (19): list / detail / empty / count-absent / partial / field-denied / generic 404 / transport failure / retry / no network-storage / DEV-only guard / sequencer ordering.
- `seller.read-boundary.test.tsx` (13): controlled-source races — A vs B, delayed success after 403, closed drawer, stale history, unmount, rejected read; session expiry mid-draft; J discards draft; failure → retry; offline.
- Existing seller tests: only waits added for the asynchronous reads (no assertion removed).
- Playwright `E04K` (3 × 4 viewports) with the page clock; full seller suite 81/81.

## Evidence

`docs/evidence/rm-e04k/` — desktop: loading, detail, detail-pending, partial-history, count-absent, field-denied, blocked, session-expired, failure, retry-list, offline; plus mobile / tablet / wide samples.
Regenerate: `cd web && RM_E04K_EVIDENCE_DIR=../docs/evidence/rm-e04k npx playwright test -c playwright.seller.config.ts -g "E04K"`

## Limits

The frontend does not and cannot enforce real row-level access; only a future approved backend can. Search, queues and pagination stay local presentation logic over the synthetic page.
`R5-SEC-01 = PREFERRED_A / NOT_APPROVED · F1 = INTERIM_GUARD_ONLY · Group B / S4 / D-42 / Real grants = BLOCKED · Production = NO-GO` — unchanged.

## Rollback

Revert the PR commit. No data, schema, permission or configuration state to undo.
