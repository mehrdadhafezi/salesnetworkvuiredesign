# RM-P07 — Finance operational workspace (read only)

Status: implemented on `claude/rm-p07-finance-operational-workspace`. Base `main` = `ce4a15a`.
Authority: AGENTS.md, `FINANCE-PRODUCT-SPEC.md` (§5–§8, §14, §15, §17), `GATE-0-PRODUCT-INVARIANTS.md`, `CRM-CROSS-ROLE-ARCHITECTURE-V1.md`,
`prototype/finance/FINANCE-DESIGN-SPEC.md` (**READY_FOR_REVIEW — not frozen; unchanged**).
No permission, capability, role, grant, flag, status, workflow transition, financial rule, migration or write endpoint is added.
D-41, OPD-08, OPD-09, D-43, OPD-04, OPD-06 stay open. The Finance role is **NOT LIVE VERIFIED**. Group B closed — **PRODUCTION NO-GO**.

## Found before implementation (main `ce4a15a`)

| Area | Existing contract | Capability / flag |
|---|---|---|
| Invoices | `GET /invoices` (presence, customer/case link state, has_unknown_status, customer_id, case_id), `/invoices/{id}` + items, stages, totals, payments, receipts (RM-P02 workspace) | `billing.invoice.read` / `payments.attempt.read` · `billing_payments.read_shadow` |
| Legacy attempts | `GET /payments/attempts` (channel, state, state_mapping, approval_provenance, presence, invoice_link_state), `/{id}`, `/{id}/history` (append-only, `regression` flag) | `payments.attempt.read` |
| Receipt references | per invoice only (`/invoices/{id}/receipts`), `/payments/receipts/{id}`; no global list | `payments.attempt.read` |
| Coverage | `GET /billing/reconciliation-report` (present-only counts; F02 NOT_EVALUATED) | `billing.reconciliation.read` |
| Target submissions | `GET /finance/review/attempts` (invoice_id; only `PENDING_REVIEW` / `NOT_REVIEWED` exist) and `/{id}` with D-43 + maker/checker gates; seller batch reads are submitter-scoped | `finance.review.read` · `finance.review_read` |
| Target receipt diagnostics | `GET /payments/evidence` (receipt/upload counts, reconciliation records) | `payments.evidence.read` · `payments.manual_write` |
| Shadow ledger | `/finance/shadow-runs`, ledger, balances, reconciliation (GET only) | `finance.*.read` · `finance.shadow_ledger_read` |
| Decisions | **none**: no approve/reject/post/settle/reverse/refund/retry/reconcile/bank route; evaluator never opens a gate; unknown reviewer identity = maker (fail closed) | — |

## Workspace

| Section | Route | Reads |
|---|---|---|
| میز کار مالی (landing) | `/finance` (existing URL; shadow dashboard kept unchanged below) | billing report + payment evidence (one request each) |
| فاکتورها / تلاش‌های پرداخت | existing `/billing/invoices`, `/billing/attempts`, RM-P02 `/billing/invoices/:id` | unchanged |
| صف بررسی و پرداخت‌های ثبت‌شده | `/finance/review` (existing, upgraded), `/finance/review/:id` | review queue with `invoice_id` + **new** `evidence_state`, keyset paging |
| شواهد رسید | `/finance/receipts` (new) | report (legacy refs) and evidence diagnostics (target receipts), kept apart |
| استثناها و شواهد حل‌نشده | `/finance/exceptions?type=` (new) | 27 queues, each a stored state + server filter, or "no list contract" stated |

Section navigation is presentation only, gated through existing destinations (`billing.invoices`, `billing.attempts`, `finance.review`,
`billing.report`, `payments.evidence`, `finance.dashboard`); no destination was added. Invoice detail deep links (`#inv-attempts`,
`#inv-receipts`, `#inv-review`, `#inv-stages`, `#inv-provenance`) scroll once the detail has loaded.

Counters: each names its aggregate field, scope and **unit** (invoice, legacy attempt, submission, target receipt, legacy receipt
reference, history row, amount evidence). A queue whose counter and list count different units says so ("دو مجموعه متفاوت"); a receipt
count is never paired with a payment list (`target-receipt-nr` counts receipts and has no list contract; submissions with
NOT_RECOVERABLE evidence are the separate `review-evidence-nr` queue, no one-to-one implied). Each counter the review-queue total, absent rows and invoice-level unknown-status counts have no
contract and show «در دسترس نیست». A denied / disabled source never becomes 0. Regression count is history rows, not attempts (stated).
Amounts are shown per row in their own unit; no sum, balance, conversion or "paid" inference exists. Nine financial operations are listed
as unavailable with their blocking decisions; no control is rendered.

## Backend (narrow, read only)

1. `GET /api/v1/finance/review/attempts?evidence_state=PENDING_FINALISATION|READY|NOT_RECOVERABLE` — stored evidence state, conjunctive
   with `invoice_id`; invalid value 400. `payments.ReviewSource.ReviewQueue` takes a `ReviewFilter`.
2. `GET /api/v1/payments/attempts?has_state_regression=true|false` — EXISTS over the attempt's own append-only history.

OpenAPI + generated Go/TS clients, query allowlist, module-owned READ ONLY transactions, same capabilities.

## Verification

- Go: `TestRMP07FinanceWorkspaceReads` (filters vs stored rows, partition, paging, 400s, review ⟂ invoice ⟂ attempt authorization,
  maker/checker, no command route for POST/PUT/PATCH/DELETE, read-only md5 fingerprint), full `go test ./...` on disposable PostgreSQL.
- Web: typecheck, ESLint, Vitest (16 new in `web/src/modules/finance/workspace.test.tsx`, incl. count/list unit semantics), OpenAPI lint/regenerate, build.
- Browser: `TestBrowserFinanceWorkspaceReadOverlay` (5/5; financial snapshot unchanged, evidence store empty) plus every orchestrator
  of `e2e-ci.yml`. Screenshots 1366 / 820 / 390 px with no horizontal overflow in `docs/evidence/rm-p07/` — only `finance-*.png`, written
  through the dedicated `E2E_FINANCE_SCREENSHOT_DIR` (other suites use `E2E_SCREENSHOT_DIR`); `docs/governance/rm_p07_evidence_test.go`
  pins the directory to exactly the 15 Finance files.

## Not done (approval-dependent, not invented)

Any Finance decision, posting, refund, retry or reconciliation write; a queue total; a global receipt-reference list; a list filter for
per-invoice amount classes; a real Finance capability bundle and live account verification.
