# Test Report — 2.0.74

## Regression comparison against 2.0.72
Only these runtime files differ:
- `sales-network.php` (version/build only)
- `includes/class-sn-plugin.php` (Zibal report setting/export)
- `assets/js/public-manager.js` (show export control on online-paid subtab)
- `assets/css/public.css` (export control styling)

No runtime file was added or removed.

## Static gates
- All PHP files: syntax PASS
- All JS files: syntax PASS
- `tools/static-audit.js`: PASS
- Duplicate PHP methods: 0
- Baseline behavioral audit booleans that were true in 2.0.72 remain true in 2.0.74.

## Focused export tests
- Numeric status `1`: successful
- Numeric status `2`: rejected from successful-only export
- Persian `موفق`: successful
- Persian `پرداخت شده - تاییدنشده`: not successful
- English `verified`: successful
- Blank status with payment evidence: successful fallback
- Specific gateway/terminal/IP/mobile aliases normalize correctly
- Generic nested provider/terminal/ip/phone aliases do not leak into fields
- Report data enriches local transaction data without dropping local rows
- Spreadsheet formula-injection prefixes are neutralized

## User sample reconciliation
The supplied Zibal sample contained 591 successful transactions. All 591 matched a CRM invoice by `Zibal orderId = CRM invoice_code`; for all 591 matched records, Zibal amount in rial equaled CRM amount in toman × 10.

## Not executable without production/staging credentials
A live call to the user's Zibal Report API was intentionally not performed because the API token must not be sent in chat or bundled in the plugin. Live pagination/filter behavior and undocumented terminal/IP fields therefore require one server-side smoke test after deployment.
