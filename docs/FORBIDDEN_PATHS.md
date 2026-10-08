# FORBIDDEN_PATHS

## Never change without explicit request
- SMS provider selection, credentials, payload templates, provider protocol.
- ZarinPal/payment request, verify, callback flow.
- Public invoice token validation and access-token security.
- Invoice creation core and invoice code format.
- WooCommerce order creation side effects.
- Legacy wallet/commission calculation and crediting.
- Finance approve/reject side effects.
- SN Data Porter plugin unless the task explicitly targets it.

## Protected files/modules
- `includes/class-sn-sms.php`
- Payment/ZarinPal methods in `includes/class-sn-plugin.php`
- Wallet/commission methods in `includes/class-sn-plugin.php`
- Invoice creation methods/core in `includes/class-sn-plugin.php` and `includes/class-sn-invoice.php`
- Public invoice page and validator methods
- Data Porter plugin files outside the main Sales Network plugin

## Protected tables
- `sn_leads`
- `sn_invoices`
- `sn_invoice_items`
- `sn_payments`
- `sn_wallets`
- `sn_wallet_transactions`
- `sn_activity_logs`

## Forbidden SQL / destructive changes
Do not add or run:
- `DROP TABLE`
- `TRUNCATE`
- `RENAME TABLE`
- mass `DELETE`
- destructive schema rewrites
- direct production data migrations without dry-run/report

Allowed only when explicitly requested:
- `CREATE TABLE IF NOT EXISTS`
- add missing columns/indexes idempotently
- safe backfill with batch limits and reports

## Forbidden data exposure
- Do not expose `access_token` in seller/supervisor/manager/deputy panels.
- Do not print full row payloads in errors.
- Do not show passwords/hashes/tokens.
- Public/customer endpoints must not reveal internal diagnostics.

## Forbidden behavior regressions
- Do not redirect `[sn_my_panel]` guests to `wp-login.php`; show plugin login form.
- Do not require public invoice token for admin/internal invoice actions.
- Do not auto-change WordPress roles from HR panel.
- Do not auto-change `legacy_role` or legacy user_meta.
- Do not create live seller leads from MIS import in current phase.

## Needs confirmation
- Some older packages include `fix-jwt-ajax.php`; current v4 reports say it was removed. Verify active repository before mentioning or reintroducing it.
