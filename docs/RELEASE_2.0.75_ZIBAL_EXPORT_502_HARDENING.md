# Sales Network v4 — 2.0.75 Zibal Export 502 Hardening

## Scope

This release hardens the read-only Finance → Zibal transaction CSV export after a live LiteSpeed `Abort request processing / req processed: 0` failure.

## Root cause addressed

The export used the shared admin hierarchy mapper. For a small list of sellers, that helper loaded every HR profile and every current assignment, and also resolved seller labels one user at a time. On a production HR dataset this work happened before CSV headers were emitted and could cause the web server to abort the synchronous request.

## Changes

- Added an export-only HR resolver that queries only sellers present in the export and walks only their parent branches.
- Bounded hierarchy traversal to 12 levels and database batches of 400 IDs.
- Preserved seller, supervisor and senior-supervisor labels and the existing fallback when HR tables/profiles are absent.
- Added a short non-sensitive request ID and stage metrics for `remote_report`, `merge`, `hr_chain`, `build_rows`, `complete` and exceptions.
- No API token, customer data, transaction ID, invoice code, phone or card number is written to diagnostics.
- Added guarded error handling so a PHP failure before download headers returns a trackable message rather than an opaque gateway error.
- If hierarchy enrichment alone fails, transaction rows are still exported with safe seller fallback labels and the failure is marked as `hr_chain_fallback`.
- Kept the export read-only: no schema repair, migration, invoice update, payment update, wallet update or commission write is executed.

## Compatibility

The shared HR mapper used by other panels was intentionally not changed. Existing invoice, payment, callback, financial approval, wallet, commission, MIS, HR and operations write paths are untouched.

## Runtime verification required

Static verification cannot reproduce the hosting proxy/LiteSpeed boundary. After deployment, download the export once and confirm a `complete` line for the same request ID in the PHP error log. If the host still interrupts the request, the last recorded stage identifies the remaining boundary.
