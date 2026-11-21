# RM-P03 — CRM unified activity timeline (read only)

Status: implemented on `claude/rm-p03-unified-activity-timeline-l8qxmc`. Base `main` = `2617c02`.
Authority: AGENTS.md, `GATE-0-PRODUCT-INVARIANTS.md` (INV-004 owners never collapsed), RM-P01 / RM-P02 relationship and workspace rules.
No permission, capability, role, flag, status, workflow, financial rule or identity semantic is added or changed.

## Shared component

`web/src/shared/timeline/` — `model.ts` (timestamp evidence, ordering, coverage) and `ActivityTimeline.tsx` (presentation).
The component never reads; each page passes the read states it already owns. A source whose read is denied, blocked,
failed, gated (not requested) or awaiting carries no data, so it can contribute nothing to any view.

Each event shows: machine type, Persian label, timestamp (Jalali + raw source text), clock and precision, source
provenance, actor only when the source records one ("ثبت نشده" when it records none), raw transition, related record
(link only when verified), and expandable details (rendered — and for attempt history, requested — only when opened).

## Clocks (never compared with each other)

| Clock | Meaning | Sources |
|---|---|---|
| `LEGACY_SOURCE` | raw legacy wall-clock text; zone never recorded; never converted to an instant | case history `occurred_raw`, custody `occurred_raw`, attribution `source_time_raw`, invoice `issued_at_raw` / `updated_at_raw`, attempt `created_at_raw` / `updated_at_raw`, customer `observed_raw` |
| `TARGET_UTC` | RFC 3339 instant written by the target system when the event happened | contact outcome `occurred_at`, merge case `created_at`, payment submission `submitted_at` |
| `IMPORT_OBSERVED` | when the shadow import observed something — not when it happened | `recorded_at`, `captured_at`, `first_seen_at`, attempt history `recorded_at` (details only) |

Legacy text is recognised only as `YYYY-MM-DD[ HH:MM[:SS[.f]]]`; anything else (Jalali text, epoch numbers,
`0000-00-00`, impossible dates) stays **unparsed** and is shown raw. Null / empty is **missing**.

## Views and ordering

* **Per source (default):** server order, server cursor pager, coverage line (all loaded / page N only / capped — unknown).
* **Combined:** one group per clock, oldest first, ordered by interval start, interval end (precision), fixed source rank,
  then source index. Equal instants are flagged `TIE`; overlapping intervals of different precision are flagged
  `PRECISION_OVERLAP`; an insertion-ordered source whose time order disagrees with its own ordinal is flagged
  `SOURCE_ORDER_DIFFERS`. Events without a comparable timestamp are listed apart in source order. A banner states the
  view is incomplete whenever any source is not fully loaded or has no data. No global cross-clock order is claimed.

## Pages and sources

| Page | Source | Read (own authorization) | Order / coverage |
|---|---|---|---|
| Case | State history | `GET /cases/{id}/history` (cases.case.read) — same read as before | source ordinal, paged (100) |
| Case | Custody events + findings | `GET /custody/cases/{id}` only when `custody.custody.read` hint allows; otherwise not requested | source ordinal, cap 1000 |
| Case | Contact outcomes | `GET /cases/{id}/contact-outcomes` only when the session feature `contact_outcome_write` is on (Slice 11 rule); one request shared with the panel | newest first, cap 200 (no cursor ⇒ "completeness unknown" at the cap) |
| Case | Attribution snapshots | `GET /cases/{id}/attribution` — same read as before | list order |
| Invoice | Invoice record time fields | `GET /invoices/{id}` | record fields (not a change history) |
| Invoice | Legacy payment attempts (+ RM-P02 per-attempt history in details) | the workspace's own `/payments` page read, only with `payments.attempt.read`; history only when details opened | list order, paged (25) |
| Invoice | Target-submitted payments | the RM-P02 Finance review read (`invoice_id` filter), one request shared with the review section, only after invoice detail SUCCESS | list order, paged (10) |
| Customer | Customer record observation | `GET /customers/{id}` | record field |
| Customer | Identity merge cases | from the detail, only with `customers.merge_case.read` hint | list order |
| Customer | Invoice issuance | `GET /invoices?customer_id=` (exact id); rows whose own link is not RESOLVED to this id are dropped and counted | list order, paged (10) |

The customer timeline is explicitly a **safe subset**: case-level and payment-level events stay on their own pages
(aggregating them would require one read per case / attempt and is not claimed).

## Backend change (narrow, read only)

`GET /payments/attempts/{id}/history` paged `ORDER BY history_id`, a random UUIDv4 — an arbitrary order, while the
code comment promised "keyset on recorded_at, id". It now pages in append order `(recorded_at, history_id)` with an
opaque compound cursor (`pagination.Cursor{Sort: "recorded_at"}`); a cursor issued for another keyset is
`CONTRACT_INVALID`. Still a READ ONLY transaction; response shape unchanged; the OpenAPI description states the order.
Proven by `internal/payments/query/history_test.go` (fails on the previous code).

## Known gaps (not invented, recorded)

* Legacy wall-clock time zone is unknown → legacy and target clocks are never merged.
* Contact outcomes have no cursor (server cap 200).
* Attempt history time is import observation, not occurrence.
* No item / stage / receipt change history exists in the sources.
* No Finance decision events exist (D-41 / OPD-08 / OPD-09 open).
* HR / Finance runtime remain NOT LIVE VERIFIED (dev overlay only).
