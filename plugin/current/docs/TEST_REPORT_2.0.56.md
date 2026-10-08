# Test report 2.0.56

## Automated checks

- Plugin header and `SN_VERSION`: `2.0.56`.
- JavaScript syntax: passed with Node for all 17 packaged JavaScript files.
- Static audit: passed with `ok: true`, zero errors; Operations route, Select All, invoice-source policy and reset invariants are present.
- ZIP integrity and single `sales-network-v4/` package root: passed.
- PHP lint: unavailable in the local build environment.

## Code invariants

- Supervisor route table has one route per content/source Sales Manager and stores both Operations manager and supervisor IDs.
- Settings reject supervisors outside the selected manager's direct HR reports.
- Future cards snapshot the direct supervisor; historic operation rows are not rewritten.
- `mis_only` is checked before a no-lead manual invoice is accepted.
- Reset preserves WordPress options, pages/products, administrators and the explicit structural configuration allowlist.

## Runtime checks still required

- WordPress `dbDelta` creation of the new routing table.
- Real HR parent validation with production-like role assignments.
- MIS-only issuance from both legacy and V4-distributed numbers.
- Full reset only on an expendable Staging database with a verified backup.
