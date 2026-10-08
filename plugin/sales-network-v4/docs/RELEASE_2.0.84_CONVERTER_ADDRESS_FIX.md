# Release 2.0.84 — Converter address fix

## Fixed

- The converter manual-invoice form now exposes the full customer address and optional postal-code fields for ordinary/physical products.
- Shipping fields appear only when at least one selected invoice item is an ordinary product.
- The converter request now sends `customer_address` and `customer_postal_code`, matching the shared server-side invoice contract.
- Client-side validation reports missing province, city, or address before submission and validates an entered postal code as exactly 10 digits.
- The server accepts the older `address` and `postal_code` aliases during upgrades while continuing to store the canonical invoice columns.

## Data safety

- No table, column, historical invoice, payment, conversion case, or commission data is migrated or rewritten.
- The change is additive to the converter form and keeps the existing shared server-side address rules intact.
