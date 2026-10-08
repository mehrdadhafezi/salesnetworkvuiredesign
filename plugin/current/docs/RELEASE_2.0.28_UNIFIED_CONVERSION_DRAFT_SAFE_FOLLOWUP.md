# Sales Network 2.0.28 — Unified conversion and safe history

## Outcome

- Supervisor and senior-supervisor repeat-action tabs now use the shared
  `آماده‌های تبدیل` workspace. They no longer expose a separate new-invoice
  form for paid-referral rows.
- `تخصیص به خودم` keeps the case in the supervisor's own `تبدیل‌های من`
  workspace; assignment to a converter keeps the existing team scope checks.
- Existing legacy referral rows are retained as origin records and linked to a
  Dot conversion case idempotently. No invoice, payment, lead, wallet,
  commission, or distribution history is deleted.
- Seller subscriptions remain available in their own supervisor tab so the
  conversion-tab change does not remove that existing feature.

## Draft-product safety

- Draft products are excluded from new-sale selectors and all new invoice
  creation paths still require a published, enabled product.
- Historical invoice items keep their stored product name, quantity, and price
  snapshots when the WooCommerce product is drafted later.
- Existing conversion history remains visible. A new payment invoice is blocked
  for a drafted, removed, or disabled product with an explicit non-destructive
  message.

## Callback fix

- Callback date and time use separate native fields in every conversion
  workspace (seller, converter, supervisor self-assignment).
- Both values are required before `تماس مجدد` can be saved. The server also
  rejects missing, invalid, or past date-times.
- Changing a callback time resets its prior reminder state and schedules a new
  in-panel/SMS reminder for the currently responsible user.
- The lightweight reminder probe is explicitly registered with zero `init`
  arguments, preserving the 2.0.27 fatal-error safeguard.

## Upgrade behavior

- Schema change is additive: one nullable `referral_item_id` column and a unique
  index are added to `sn_dot_cases` through WordPress `dbDelta()`.
- Legacy referral migration is bounded per tab load and idempotent. Original
  rows are marked as linked, not removed.

## Verification

- All PHP files parse with tree-sitter PHP with zero error or missing nodes.
- `assets/js/public-dot.js` passes `node --check`.
- `node tools/static-audit.js` passes the unified conversion, draft-history,
  callback-reminder, manager-archive, staged-payment, and init-hook checks.
- WordPress/PHP/MySQL staging smoke testing is still required before production.
