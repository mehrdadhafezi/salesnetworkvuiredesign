# Sales Network 2.0.63 — Operations Sales Staged Payment

## Scope
Adds staged/partial payment support to the Operations Sales Expert workflow while preserving the existing seller staged-payment accounting and the existing operations routing model.

## Operations Sales Expert flow
- When creating an operations upgrade invoice, the expert can choose full payment or staged payment.
- Staged payment uses the same `sn_partial_payment_presets` configuration as the seller flow and also supports a custom first-stage amount.
- The total invoice amount remains fixed; only `current_due_amount` changes per stage.
- After finance approves a partial stage, the same operations case stays assigned to its Operations Sales Expert.
- The expert sees total, approved paid amount, remaining amount, current stage number, and current due amount.
- The expert can issue the next stage for a custom amount or settle the full remaining balance.
- Every stage uses the same invoice and invoice URL; no duplicate invoice is created.
- Resending the payment link sends the current open-stage amount.
- Only after the total remaining balance reaches zero does the existing `sn_invoice_paid` route move the case to the Executive Operations queue.

## Ownership isolation
Operations upgrade invoices are excluded from the seller `invoice_payment_completion` repeat-action queue after partial approval. Seller/original-seller ownership fields remain unchanged for existing ownership/commission semantics.

## Safety
- Transaction locks prevent duplicate next stages.
- A new stage cannot exceed the actual remaining balance.
- Issuing a new stage is allowed only when the previous stage is approved and the invoice is awaiting assignment of the next amount.
- Operations assignment is rechecked inside the transaction.
- Database failure while updating the related project action rolls back stage issuance.
- Customer self-activation behavior is unchanged and remains full-payment-only in this release.

## Verification
- PHP syntax check: PASS for all plugin PHP files.
- JavaScript syntax check for `assets/js/operations-flow.js`: PASS.
- `node tools/static-audit.js`: PASS on version 2.0.63.
- Runtime WordPress/payment-gateway smoke testing is still required on staging before production deployment.
