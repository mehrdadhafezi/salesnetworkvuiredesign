# Merge release 2.0.16

## Scope

- Preserves the complete seller stage-one workflow from 2.0.13.
- Preserves the HR transfer module and the later panel/bootstrap fixes.
- Integrates the Campaign/UTM, Marketing Dot funnel, page-exit metric and report-tooltip changes supplied in 2.0.15.
- Does not change SMS provider logic, payment gateway request/verify/callback logic, wallet posting or public invoice token validation.

## Atomic pre-invoice correction

For source-linked invoices, the live source row is locked before the shared seller-flow state. The following writes now complete in one database transaction:

1. invoice row and access token;
2. payment stage;
3. invoice items and product-type snapshots;
4. Dot source-invoice link, when applicable;
5. legacy lead or V4 distribution source status;
6. shared seller-flow state and `pre_invoice_created` event.

Failure of the source-status write or seller-flow event rolls the whole transaction back. A concurrent reassignment is detected after the source lock and returns a retryable conflict instead of issuing an invoice for a stale owner.

## Campaign/UTM merge

- Supports query-string and legacy fragment UTM links.
- Persists the five UTM fields and landing URL with each Marketing Dot submission.
- Keeps repeated form submissions as independent marketing leads.
- Records Marketing Dot funnel events, gateway/cancellation outcomes and page exit.
- Adds Jalali-compatible campaign report filters and report-column tooltips.
- Preserves partner allowlists and hashed read-only API tokens.

## Staging acceptance

1. Run the seller stage-one checklist in `SELLER_STAGE_ONE_2.0.13.md`.
2. Force a source-status database failure during pre-invoice creation and confirm no invoice, item, payment stage or seller-flow event remains.
3. Reassign a source during invoice creation and confirm the request returns a conflict without creating an invoice.
4. Run the UTM and funnel matrices in `UTM_ATTRIBUTION_2.0.13.md` and `MARKETING_FUNNEL_AUDIT_2.0.12.md`.
5. Confirm the HR transfer panel and export/search endpoints still load.
6. Run PHP lint and WordPress/WooCommerce smoke tests on Staging.
