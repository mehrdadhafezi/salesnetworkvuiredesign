# Sales Network v4 — 2.0.74 Zibal Finance Export (Audited)

## Scope
Adds a read-only combined Zibal/CRM export to Financial Panel → Payment Review → Online Gateway Payments.

## Output
The export contains 21 columns in the requested order and joins Zibal `orderId` to `sn_invoices.invoice_code`.

## Safety design
- Existing invoice issuance, payment request, gateway callback/verify, financial approval/rejection, wallet, commission, HR, MIS and after-sales write paths are not modified.
- Export action requires an authenticated user with existing finance-view access and a WordPress nonce.
- The export action itself performs no database writes and does not trigger schema repair/migrations.
- Zibal Report API token is server-side only and is never rendered back into HTML after storage.
- CSV formula injection protection is applied to text cells.
- Sensitive download response is marked `no-store` and `nosniff`.

## Audit fixes after 2.0.73
1. Prevented Persian `تاییدنشده` text from being misclassified as a successful transaction.
2. Replaced per-row product-item lookups with a batched invoice-item query.
3. Local fallback payment timestamp now prefers the payment row timestamp over invoice-level final paid time.
4. Bearer token handling preserves opaque token punctuation and strips only control characters.
5. Removed ambiguous generic nested aliases for gateway/terminal/IP/mobile extraction.
6. If a successful Report API call returns no transactions for an optional body, a second safe request body is tried before accepting empty data.
7. Removed schema-repair/migration invocation from the export request; the download action is read-only.
8. Added `Cache-Control: no-store` and `X-Content-Type-Options: nosniff` to the export response.

## Known integration gate
The supplied Zibal Platform documentation documents the transaction report endpoint and the standard/verbose response fields, but does not expose the complete request pagination/filter contract in the provided document. Terminal ID and payer IP are also not present in the documented transaction schema. The implementation therefore does not invent undocumented pagination/filter parameters. These fields are populated only if the live API returns specifically named fields.
