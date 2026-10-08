# Sales Network 2.0.67 — Purpose Commission Matrix + Sales/Biavin Wallets

## Scope
- Adds a new **پورسانت** settings tab and admin submenu.
- Matrix: 6 payment purposes × 11 business roles.
- Percentage input: `0` = disabled, otherwise `0.01` through `100.00`.
- Commission accounting is stage-based: each approved payment stage is an accounting event.
- Idempotent wallet credits per invoice + stage + purpose + role + recipient.

## Payment purposes
1. `assessment_fee` — اعتبارسنجی
2. `subscription_direct` — خرید مستقیم اشتراک
3. `conversion_purchase` — خرید/تبدیل پس از اعتبارسنجی
4. `product_sale` — فروش محصول عادی
5. `product_star_sale` — فروش اولیه محصول*
6. `product_star_upgrade` — افزایشی/ارتقای اعتبار محصول*

## Roles
- seller
- supervisor
- converter
- senior_supervisor
- sales_manager
- sales_deputy
- operations_sales_expert
- operations_sales_supervisor
- operations_sales_manager
- operations_executive_manager
- operations_execution_expert

## Wallet routing
- Sales-cycle purpose + sales-family role => wallet `seller` (UI label: **کیف پول فروش**).
- Biavin-cycle purpose + any role => wallet `biavin` (UI label: **کیف پول بیاوین**).
- Any operations-family role => wallet `biavin`, even when a sales-cycle payment is configured to reward that role.

## History / audit
Each purpose commission transaction snapshots:
- payment purpose + label + cycle
- role + label
- configured percentage
- approved stage amount
- stage number
- invoice id/code
- recipient user id
- target wallet label

Transaction description is human-readable and visible in wallet history.

## Deferred recipient reconciliation
If a configured recipient is not known at payment approval time, no fake/fallback user is credited.
The invoice is reconciled later when Dot/Operations assignment becomes known. Idempotency prevents duplicate credits.
This is especially relevant to initial Product* purchases where operations roles may be assigned after payment.

## Financial reversals
- Rejecting a new/pending payment stage does **not** reverse older approved-stage commissions.
- Reopening the latest approved stage reverses only that stage's purpose commissions.
- Full finance cancellation reverses all purpose commissions for the invoice.

## Legacy compatibility
When the purpose matrix engine is enabled:
- legacy automatic seller/supervisor invoice commission posting is bypassed;
- legacy Biavin project commission auto-credit is bypassed;
- legacy dry-run wallet posting is disabled;
- old wallet rows and transaction history are retained and not migrated/deleted.
