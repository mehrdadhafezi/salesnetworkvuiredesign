# Release 2.0.60 — Typed Operations Execution

## Scope

- Adds the HR role and panel «کارشناس اجرایی عملیات» under «مدیر اجرایی عملیات».
- The executive manager can search/filter cards and assign one or up to 500 filtered cards to a direct execution expert.
- Every product* can define one immutable fulfillment type:
  - wallet charge;
  - physical-product invoice;
  - dynamic form.
- The product type, instructions and form schema are snapshotted when the card enters the executive queue.
- Pending, unassigned records created before configuration may adopt their first valid configuration; configured or assigned histories are never rewritten.
- Adds an admin Forms page with active/inactive form definitions and typed, required fields.
- Adds unified execution outcomes: no answer with counter, date-only Jalali callback, and cancellation with a mandatory reason.
- Physical cards enter the existing Shipping panel as independent card records with address, carrier, tracking and terminal statuses.
- The customer card modal displays current execution type/status, credit, callback date, activation/tracking codes and a per-card event trail.

## Wallet integration boundary

- No wallet credit is faked while the destination plugin is unavailable.
- The destination may integrate through the `sn_operations_wallet_request` filter or the configured HTTPS endpoint.
- Charge requests contain a stable `request_id` and CRM-side processing lock. The destination must also treat `request_id` idempotently.
- A successful charge response must include `activation_code`; `external_id`, `customer_url` and `usage` are supported.
- Assisted customer access is accepted only as a destination-issued HTTPS one-time URL. CRM never reads the customer password and never bypasses OTP locally.

## Data safety

- All new tables and columns are additive; no destructive migration is used.
- Historic invoices, memberships, project actions, commissions, payments and product snapshots are not rewritten.
- Form definitions are deactivated instead of deleted; submitted cases retain their schema snapshot.
- Shipping card records do not reuse the legacy one-row-per-invoice shipping table, preventing collisions when a subscription contains multiple physical cards.

## Staging acceptance

1. Define one product* for each fulfillment type and create a form with required/select/checkbox fields.
2. Route normal and upsell cards to the executive manager and verify type/form snapshots.
3. Assign single and filtered bulk selections to a direct execution expert.
4. Test no-answer count, Jalali callback date and mandatory cancellation reason.
5. Verify a wallet endpoint receives a stable request ID, duplicate charge submit is blocked, and activation code/SMS/report are stored only after a successful response.
6. Confirm assisted access opens only the destination-issued HTTPS one-time URL.
7. Confirm a physical card enters Shipping, requires address before preparation, and requires carrier/tracking before shipped/delivered.
8. Fill a dynamic form, then test final execution and non-execution with mandatory reason.
9. Log in as the customer and verify each card modal shows the latest status and event history.

