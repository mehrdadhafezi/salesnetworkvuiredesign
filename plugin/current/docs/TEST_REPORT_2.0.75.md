# Test Report — 2.0.75

## Automated checks — result

- Sales Network static audit: **PASS**.
- JavaScript syntax (`node --check`, all 17 asset files and the audit script): **PASS**.
- Plugin header version and `SN_VERSION` both equal `2.0.75`: **PASS**.
- Zibal export authentication, nonce and read-only guards: **PASS (static)**.
- Export uses the bounded export-only HR resolver: **PASS (static)**.
- HR queries restricted by requested user/profile IDs, batch size 400 and maximum 12 levels: **PASS (static)**.
- Diagnostic logger contains no token or authorization value: **PASS (static)**.
- Duplicate PHP methods: **0**.
- Registered/used action audit: **200 / 168, PASS**.
- PHP syntax lint: **not run** because PHP CLI is unavailable in this build environment; run `php -l` on staging.
- WordPress/Zibal/LiteSpeed runtime: **not available in this build environment**; live acceptance remains required.

## Regression boundary

No database schema, role, invoice/payment callback, commission, wallet, HR write flow, MIS flow or Operations flow was changed.

## Live acceptance test

1. Install on staging with the configured server-side Zibal report token.
2. Open Financial Panel → Payment Review → Online Gateway Payments.
3. Download the comprehensive gateway CSV.
4. Confirm the file has 21 columns and expected transaction count.
5. Spot-check `orderId = invoice_code`, rial/toman unit relationship, customer/product, seller, supervisor and senior supervisor.
6. Confirm the PHP log contains the same request ID through `stage=complete`.
7. Confirm no `Abort request processing`, PHP fatal or HTTP 502 occurs.
