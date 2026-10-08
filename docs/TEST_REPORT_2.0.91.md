# Test Report — 2.0.91

Static release checks cover:
- PHP syntax for all plugin PHP files.
- JavaScript syntax for all plugin JS files.
- Existing Sales Network static audit.
- Presence of the dedicated `reports-successful-payments` tab.
- Successful payment source statuses and legacy invoice fallback.
- Product type/name extraction from `sn_invoice_items`.
- Scoped seller access through `sn_report_seller_sql`.
- CSV export registration for `successful_payments`.

Runtime staging verification is still required against the real WordPress database to confirm live row counts and historical data shapes.
