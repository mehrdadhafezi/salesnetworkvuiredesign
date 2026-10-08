# Sales Network 2.0.29 — Assessment supervisor queue repair

## Fixed behavior

- A seller assessment invoice whose route is `supervisor_queue` is materialized
  as a Dot case immediately after full payment and finance approval.
- The case appears in the linked supervisor's `آماده‌های تبدیل` tab even when
  the customer has not selected a subscription/product option. It is shown as
  `انتخاب نکرده` and remains assignable to the supervisor or a converter.
- Reopening Supervisor `آماده‌های تبدیل`, Supervisor `تبدیل‌های من`, or Senior
  Supervisor `آماده‌های تبدیل` refreshes the workspace instead of serving a
  stale one-time lazy-load result.

## Safe recovery

- Opening the supervisor queue runs a bounded, idempotent repair for paid
  assessment links that missed case creation during the finance request.
- The repair is scoped to the logged-in supervisor and creates at most 20 cases
  per tab load.
- Existing invoices, payment stages, Dot links, customer selections, cases,
  commissions, and activity history are not deleted or overwritten.
- Invoice-time route and supervisor snapshots remain authoritative even if Dot
  settings are changed after the invoice was issued.

## Verification

- All 17 PHP files parse with zero tree-sitter PHP error/missing nodes.
- All JavaScript files pass the Node syntax check through the static audit.
- Static audit passes assessment ownership, staged-payment completion gate,
  unselected-case repair, queue refresh, manager archives, draft history, and
  WordPress init-hook safety checks.
- A WordPress/PHP/MySQL staging smoke test remains required before production.
