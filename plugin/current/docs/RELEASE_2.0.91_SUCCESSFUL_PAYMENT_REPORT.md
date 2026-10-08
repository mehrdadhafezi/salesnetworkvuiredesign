# Sales Network 2.0.91 — Successful Payment Report

## Goal
Add a dedicated **گزارش** tab that lists successful CRM payments across every product type without changing payment or invoice data.

## Coverage
- Real rows from `sn_payments` with successful statuses (`paid`, `approved`, `verified`, `success`, `completed`, `financial_approved`, `gateway_paid`, `online_paid`).
- Legacy/manual successful invoices that have no successful `sn_payments` row are included once as a fallback.
- Multi-stage payments remain separate rows.
- Product names and product types are read from `sn_invoice_items`, with the invoice product as a name fallback.

## Filters / columns
Filters: Jalali date range, product type, payment method, seller ID, and free-text invoice/customer/phone/product search.
Columns: payment date, invoice, customer, product(s), product type(s), stage, paid amount, method, seller, status.

## Access control
The report reuses the existing Reports-panel access and seller scope. Admin/Finance retain full access; scoped sales roles only see seller IDs already visible to them.

## Export
A scoped CSV export named `successful_payments` was added (maximum 5000 rows). Phone numbers remain masked and public invoice tokens/URLs are not exported.

## Data safety
No schema migration. No destructive query. No payment/invoice mutation.
