# Sales Network 2.0.70 — Shipping + Product* Wallet Execution Audit

## Scope

Audit and hardening of the Shipping panel and Product* execution type `wallet_charge`.

## Fixes

- Shipping logout now returns staff to the canonical `/crm-login` page.
- Shipping updates validate customer name and a valid normalized mobile on the server, not only through HTML `required`.
- Product* wallet execution now snapshots an explicit destination wallet type (`cash` / `installment`).
- Wallet `service_key` is validated before manager assignment, manual CSV export, and remote wallet charge.
- Manual CSV exports now include `wallet_type` while retaining the same execution/card/customer snapshot fields.
- Legacy Product* wallet configurations without a stored wallet type default safely to `cash`; historic execution-case snapshots remain unchanged unless still pending configuration repair.

## Verification

- PHP lint over plugin PHP files.
- JavaScript syntax check via project static audit.
- Plugin static audit.
- ZIP integrity check.

Runtime WordPress/MySQL smoke tests remain required on staging for external wallet endpoint behavior.
