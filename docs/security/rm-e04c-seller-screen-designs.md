# RM-E04C — Seller Screens S1–S7: Implementation-Ready Designs

**DRAFT — NOT APPROVED. Nothing here is implemented or wired.** Baseline `main` @ `c0d7f4b`. Production **NO-GO**.

This document extends the RM-E04A screen contracts ([`rm-e04a-seller-row-scope.md`](rm-e04a-seller-row-scope.md), which keeps the full field inventories) with what implementation needs next: the capability and predicate **recommended** by the proposed ADR, the UI behaviour for every state, and readiness. Where no approved source exists the design says so; it does not invent statuses, money facts or workflows (AGENTS.md Claude MUST NOT invent).

References: ADR [`../implementation/rm-e04c-seller-authorization-adr-PROPOSED.md`](../implementation/rm-e04c-seller-authorization-adr-PROPOSED.md) · API [`../implementation/rm-e04c-seller-api-contract-PROPOSAL.md`](../implementation/rm-e04c-seller-api-contract-PROPOSAL.md) · tests [`rm-e04a-seller-negative-test-plan.md`](rm-e04a-seller-negative-test-plan.md) (NT-01…NT-24) and [`rm-e04c-seller-traceability-matrix.md`](rm-e04c-seller-traceability-matrix.md) (NT-25…NT-32) · decisions [`../governance/proposals/rm-e04c-seller-decision-package.fa.md`](../governance/proposals/rm-e04c-seller-decision-package.fa.md).

## 0. Shared UI state model (all screens)

The UI never decides scope; it renders what the server returned. States map to the existing envelope and the RM-E01 / RM-E03 design system (no new business state).

| Server answer | UI state | Copy rule |
|---|---|---|
| 200 `SUCCESS`, items > 0 | data | — |
| 200 `SUCCESS`, items = 0, `count.state = EXACT` | **empty** | "موردی برای شما ثبت نشده است" + `evaluated_as_of`; never "سازمان خالی است" |
| 200 `PARTIAL` (`COVERAGE_INCOMPLETE`, truncated history, incomplete stages) | data + amber banner | name the missing fact; amounts depending on it shown «—» |
| 200 with `count.state = ABSENT` | counters show «—» with tooltip reason | never `0` |
| field absent + `FIELD_DENIED` | field row shows «محدود» (restricted) | no partial value |
| `state_mapping = UNKNOWN`, stage `OUTCOME_UNKNOWN` | dashed / amber "نامعلوم" badge | never coerced to a known state |
| 200 `BLOCKED` + `FEATURE_DISABLED` | module unavailable panel | no rows, no retry loop |
| 403 `ACTION_DENIED` | **unauthorized** panel; destination hidden by navigation (RM-E02) when the capability is absent | no hint about what exists |
| 403 `ROW_SCOPE_DENIED` (no proven user) | unauthorized panel "هویت کاربر تأیید نشده" | — |
| 404 on a detail | "یافت نشد" — identical for missing and out-of-scope | never "متعلق به فروشندهٔ دیگر" |
| 401 | session expired → login | — |
| 500 / network | error panel with `correlation_id`; retry only for reads | never show cached data (SR-12) |

Readiness legend: **R0** blocked — no server source or contract · **R1** contract complete, governance-blocked · **R2** buildable on synthetic grants once M2 rec. 1 + 2 are approved · **R3** server scope exists.

---

## S1 — شماره‌های من (Leads)

| Item | Design |
|---|---|
| Required data | own cases / leads: id, workflow type, raw + mapped state, lead facts (`lead_status_raw`, `assigned_at_raw`, `updated_at_raw`), channel, customer link (id + state), current contact outcome, scope basis + as-of |
| Existing API coverage | `GET /cases`, `/cases/{id}`, `/history`, `/contact-outcomes`, `/custody/cases/{id}` — **organisation-scoped**; not usable for a Seller |
| Missing contracts | `listMyCases`, `getMyCase`, `listMyCaseHistory`, `getMyCaseContactOutcomes` (API proposal §2); server queue definitions (RM-E03 chips have none) |
| Capability | `cases.own.read` (candidate; M2 rec. 1) + flag `sales.read_shadow` |
| Row predicate | `CUSTODY_CURRENT_HOLDER` (recommended; M2 rec. 2 option 1) |
| Sensitive fields | `customer_ref` (S-PII-adjacent: id only); history `actor_user_id` (S-STAFF → "شما" / withheld); no phone or name until GAP-04 |
| Backend dependencies | custody legacy holder columns (batch-refreshed); `cases.contact_outcome_*` (200-event read cap → PARTIAL) |
| Expected UI | list + search by code; drawer with case facts and own outcome history; outcome grid stays a **demo** (write not wired); queue chips hidden or «—» until a server queue contract exists |
| Empty state | §0 empty, with as-of batch time |
| Unauthorized | §0; destination hidden without `cases.own.read` |
| Unknown / partial | holder unknown → row absent (not counted, not mentioned); `state_mapping` UNKNOWN → badge; history truncated → PARTIAL banner |
| Security tests | NT-01, 03, 04, 05, 06, 07, 08, 09, 10, 14, 16, 17, 18, 21, 24, 25, 26, 27, 28 |
| Readiness | **R1** → R2 after M2 rec. 1 + 2 |

## S2 — آماده‌های تبدیل (Conversions)

| Item | Design |
|---|---|
| Required data | list of "ready for conversion" cases; per case: current custody, original seller, credit owner, linked invoices |
| Existing API coverage | `/cases/{id}/attribution`, `/invoices?case_link_state=` — organisation-scoped |
| Missing contracts | **"ready for conversion" has no server status** → list cannot be specified. `listMyCaseAttribution` is specifiable as an S1 sub-resource |
| Capability | `cases.own.read` for attribution only |
| Row predicate | parent case predicate (S1). Which facet admits a case to "my conversions" = RM-E04A-Q3 (open) |
| Sensitive fields | facet subjects → `CallerFacet.is_caller`; linked invoices S-FIN |
| Backend dependencies | Codex workflow contract for readiness; XR §6 "Seller → Supervisor / ready conversion Case" hand-off is product WHAT, not a server state |
| Expected UI | keep synthetic list (DEV preview); in a case drawer opened from S1, show attribution facets with "شما / نامعلوم / محدود" |
| Empty state | list: "این بخش هنوز به دادهٔ سرور متصل نیست" (not "no conversions") |
| Unauthorized | §0 |
| Unknown / partial | facet UNKNOWN / CONFLICT / NOT_RECOVERABLE shown as such; credit owner UNKNOWN stays UNKNOWN |
| Security tests | NT-03, 12, 14, 22, 29 |
| Readiness | attribution **R1**; list **R0** |

## S3 — نیاز به اقدام مجدد (Repeat Actions)

| Item | Design |
|---|---|
| Required data | items handed back for a repeat action: invoice / stage, original issuer, current custody, next actor, reason |
| Existing API coverage | `/invoices/{id}/stages`, `/custody/cases/{id}` — organisation-scoped |
| Missing contracts | hand-off model, next actor, repeat-action state — **no server source** |
| Capability | none proposed |
| Row predicate | original issuer vs current custody vs both — open (OPD-10, M2 rec. 2) |
| Sensitive fields | S-FIN, S-STAFF |
| Backend dependencies | workflow contract (Codex/Product); OPD-10; D-32; OPD-02 |
| Expected UI | synthetic rows in DEV preview only; production destination hidden |
| Empty state | "این بخش هنوز به دادهٔ سرور متصل نیست" |
| Unauthorized | §0 |
| Unknown / partial | all three facts UNKNOWN; never inferred from hierarchy |
| Security tests | NT-22 (no endpoint without registry + tests) |
| Readiness | **R0** |

## S4 — فاکتورها (Invoices)

| Item | Design |
|---|---|
| Required data | own invoices: code, lifecycle, case / customer links, completeness flags, unknown-state count, per-unit declared and paid sums, stages, payments |
| Existing API coverage | `/invoices*`, `/invoices/{id}/payments`, `/receipts` — organisation-scoped; `/payments/invoices/{id}/submission` — owner-only, 404 parity (A3) |
| Missing contracts | `listMyInvoices`, `getMyInvoice`, `listMyInvoiceStages`, `getMyInvoiceTotals`, `listMyInvoicePayments` |
| Capability | `billing.own.read`; `payments.own.read` for payments (candidates) |
| Row predicate | `INVOICE_SELLER_AND_ISSUER` recommended initial (M2 rec. 2 option c); payments inherit |
| Sensitive fields | money (S-FIN, per unit); `approved_by`, stage `issued_by` of others (S-STAFF → withheld); customer (S-PII → id only) |
| Backend dependencies | `billing.invoice` seller / issuer columns (NULL = unknown, no link-state column); GAP-03 for any "confirmed" amount; D-25 / D-43 |
| Expected UI | list with KPI «مانده قابل وصول» only as a scoped per-unit sum with PARTIAL flag; drawer with stages; "financially confirmed" column shows «— تعریف نشده»; "current payable" only from the owner submission context |
| Empty state | §0 empty |
| Unauthorized | §0 |
| Unknown / partial | `unknown_state_count > 0` or incomplete stages → PARTIAL, dependent amounts «—»; cancelled → remaining absent; NULL seller / issuer invoices are absent for everyone |
| Security tests | NT-01, 03, 05, 06, 07, 08, 11, 12, 13, 15, 17, 18, 21, 24, 25, 26, 27, 28, 30 |
| Readiness | **R1** → R2 only after M2 rec. 1 + 2 **and** the three S4 readiness proofs in `docs/governance/rm-e04d-s4-readiness.json`: an approved and verified conversion-payment contract (Finance + Security), a NOT NULL fact with an explicit unknown state, and six I-L enforcement tests on synthetic seller ≠ issuer and unknown-conversion fixtures. Enforced by `TestRME04DS4NeedsConversionPaymentContract`. Money labels remain limited by GAP-03 |

## S5 — رفتار مشتریان (Customer Behaviour)

| Item | Design |
|---|---|
| Required data | link opened, visit, paid-after-visit events |
| Existing API coverage | none — no table or projection holds these events |
| Missing contracts | event source, data contract, privacy review |
| Capability | none |
| Row predicate | would inherit S1 / S4 |
| Sensitive fields | behavioural PII (Legal review) |
| Backend dependencies | product data contract; GAP-04; ADR #18 for any real data |
| Expected UI | synthetic in DEV preview; production hidden; «نامعلوم» never `0` |
| Empty state | "منبع داده‌ای برای این بخش وجود ندارد" |
| Unauthorized | §0 |
| Unknown / partial | everything UNKNOWN |
| Security tests | NT-22 |
| Readiness | **R0** |

## S6 — درخواست اضافه کردن شماره (Extra-Number Request)

| Item | Design |
|---|---|
| Required data | own requests: state, assigned manager, resulting lead (XR §11 names "Seller requester / assigned Manager / state / resulting legacy lead") |
| Existing API coverage | none |
| Missing contracts | target request table, state vocabulary, read and create operations |
| Capability | none exists; the create command is out of scope here |
| Row predicate | `REQUESTER` (reserved) |
| Sensitive fields | reviewer identity (S-STAFF → withheld until decided) |
| Backend dependencies | OPD-10 (as linked by RM-E03 / RM-E04A), M2 rec. 1; workflow contract |
| Expected UI | form validates locally and shows the OPD-10 warning toast; nothing is sent |
| Empty state | "ثبت درخواست هنوز فعال نیست" |
| Unauthorized | §0 |
| Unknown / partial | N/A |
| Security tests | NT-19, NT-22 |
| Readiness | **R0** |

## S7 — کیف پول من (Wallet)

| Item | Design |
|---|---|
| Required data | own commission lines and per-unit totals from a stored run |
| Existing API coverage | **`GET /commission/me/earnings`** — own scope implemented (`holder_user_id = caller`); `no-store` now global (A2) |
| Missing contracts | posting / confirmed / settleable state — no server source |
| Capability | `commission.own.read` (exists; not in any bundle) |
| Row predicate | `COMMISSION_HOLDER` (exists) |
| Sensitive fields | S-FIN; `invoice_id` link must pass S4's predicate |
| Backend dependencies | D-41, OPD-09, D-36 for semantics; ADR #18 for real data |
| Expected UI | lines with run id + as-of; totals labelled "اعتبار تاریخی (shadow) — قابل تسویه نیست"; settlement «— تعریف نشده»; units never summed together |
| Empty state | no run → "اجرای ثبت‌شده‌ای وجود ندارد" (UNKNOWN), not "موجودی صفر" |
| Unauthorized | §0 |
| Unknown / partial | partial run → PARTIAL banner; unknown lines excluded from sums with banner |
| Security tests | NT-04, 12, 17, 18, 21, 23, 25, 31 |
| Readiness | **R3** (server scope exists) — wiring still needs an ACTIVE grant: M2 rec. 1 (+ D-42 / D-34 outside synthetic tests) |

---

## Summary

| Screen | Recommended capability | Basis | Readiness | First unblocker |
|---|---|---|---|---|
| S1 | `cases.own.read` | `CUSTODY_CURRENT_HOLDER` | R1 | M2 rec. 1 + 2 |
| S2 | `cases.own.read` (attribution) | parent case | R1 / R0 | readiness contract; RM-E04A-Q3 |
| S3 | — | — | R0 | hand-off contract; OPD-10 |
| S4 | `billing.own.read`, `payments.own.read` | legacy-equivalent I-L (RM-E04D; `INVOICE_SELLER_AND_ISSUER` withdrawn) | R1 | M2 rec. 1 + 2; conversion-payment data contract; GAP-03 for confirmed money |
| S5 | — | — | R0 | event source; privacy |
| S6 | — | `REQUESTER` | R0 | workflow; OPD-10 |
| S7 | `commission.own.read` | `COMMISSION_HOLDER` | R3 | M2 rec. 1; D-41 / OPD-09 for semantics |
