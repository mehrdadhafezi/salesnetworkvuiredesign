# 2.0.135 — Gateway export source correction (partial resolution)

- Finance export explicitly selects the received Zibal report or local CRM payments.
- Zibal rows are authoritative: local history is no longer unioned into the provider report. Matching local payments only supply the CRM invoice for enrichment.
- Provider amount, payment time, reference and status are preserved, including empty values.
- Paid-but-unverified transactions remain in the report with their distinct status; this does not approve any payment or invoice.
- Provider failures are displayed before download headers; no silent fallback to local history.
- Source is included in the download filename; rial/toman units and report scope are explained in the panel.
- Logs contain raw row count and top-level response key names, without logging credentials or full response bodies.

## Still unresolved

The current Report API client sends one request. Pagination was not implemented because its contract could not be verified from the accessible official documentation. The UI explicitly warns that the received report may represent only one page. No undocumented request parameters or arbitrary 100-row rejection were introduced. This release must not be described as a verified full reconciliation fix. Live merchant/token routing and API response metadata must be checked to explain the absent shared transaction IDs and missing latest dates.

## Validation

- PHP 8.0 syntax parsed for both changed PHP files.
- 15 executable assertions passed using PHP 8.4 WebAssembly: source isolation, deduplication, invoice enrichment, immutable provider fields, empty remote report, paid/failed/refunded statuses and unverified labeling.
- Uploaded CSV (4340 rows) and XLSX (870 rows) used as reconciliation fixtures. Merger returns exactly the 870 provider rows and preserves 28,885,000,000 rial total.
- No live WordPress, database or authenticated Zibal API test was available.
