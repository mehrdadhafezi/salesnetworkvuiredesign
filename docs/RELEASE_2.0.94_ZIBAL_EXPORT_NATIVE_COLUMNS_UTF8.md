# Release 2.0.94 — Native Zibal finance export on 2.0.93 branch

This historical release introduced the native Zibal Finance export that is now merged into 2.0.96.

- Sends configured `sn_zibal_merchant` as `merchantId` directly from the main plugin.
- Supports Zibal `terminalNumber`, `cardHolderIP` and `psp` fields.
- Finance users can choose export columns; all 28 columns are selected by default.
- Adds seller/supervisor/senior-supervisor IDs plus sales manager and sales deputy names/IDs.
- Emits UTF-8 CSV with BOM and preserves identifier-like values in Excel.
- Falls back to successful local CRM gateway rows when the provider report is unavailable.
- No database schema changes.
