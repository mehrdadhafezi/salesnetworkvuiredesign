# Sales Network 1.0.149 — HR manual invoice access, product limit, finance payment detail

## 1. HR-controlled manual pre-invoice access

A per-user tri-state access flag is stored in WordPress user meta under `sn_manual_invoice_access`:

- `inherit` (meta removed): preserve legacy access based on HR position / WordPress role.
- `allow`: explicitly allow manual pre-invoice issuance for an otherwise eligible sales-role user.
- `deny`: explicitly block manual pre-invoice issuance.

The control is available in the HR workforce table and profile edit modal. The seller panel hides the manual invoice tab when denied. The create-invoice server endpoint enforces the same permission, so hiding the UI is not the security boundary.

Administrators retain access.

## 2. Maximum products per invoice

Option: `sn_invoice_max_products`

- Default: `2`
- `0`: unlimited
- Positive integer: maximum number of selected product rows in one pre-invoice.

Enforcement exists in both seller UI and `sn_create_invoice_core()`. The server-side check is authoritative.

## 3. Finance approval context and payment history

The financial review list now exposes:

- invoice total amount,
- amount of the current/latest payment under review,
- cumulative approved amount,
- remaining invoice amount,
- current payment stage.

A `جزئیات فاکتور` action loads a finance-only AJAX endpoint and displays:

- invoice/customer/seller/products summary,
- total / approved paid / remaining amounts,
- all staged-payment rows,
- all payment records including amount, stage, method, source, status, reference, user and Jalali date.

Fully settled invoices are explicitly marked as settled and retain the full deposit/payment history.

## Compatibility / data safety

This release adds no destructive schema operation. HR access is stored in user meta and the product-row limit is a WordPress option. Existing staged-payment tables from 1.0.148 are reused.
