# CRM backlog 06–12 — release 2.0.110

The previous 2.0.109 release covers CRM-01 through CRM-05. This release adds CRM-06 through CRM-12.

| Item | Behavior |
| --- | --- |
| CRM-06 | The last staff member who issued or sent the payment invitation, or submitted it for finance review, may change a full invoice to staged payment before any payment or finance submission is recorded. The seller who owns the invoice can edit other permitted details. |
| CRM-07 | Assessment products require full payment, including in the server validation path. |
| CRM-08 | Before payment begins, the seller can change an existing item's permitted product and quantity, payment plan and first-stage amount. Finance, receipts, active payment requests, discounts and fulfillment lock normal financial editing. Totals and item types are recalculated. |
| CRM-09 | Each payment stage in a conversion case opens its linked invoice, including the receipt entry path. |
| CRM-10 | Staff with invoice access can resend its link to the primary number or an explicitly entered valid mobile number. |
| CRM-11 | Staff with invoice access can copy the short public payment link. |
| CRM-12 | Optional secondary phone is saved on the invoice and shown masked on its public view. Invoice creation sends its automatic SMS only to the primary phone; the secondary number receives an SMS only when explicitly chosen during resend. |

## Validation on a WordPress staging site

1. Create a full invoice and leave it unpaid. Verify its invoice row and payment stage exist before the customer opens the link. Have its last sender change the plan to staged, then verify the first-stage amount on the public invoice.
2. Create an assessment invoice. Verify the staged selection and direct staged submission both fail.
3. Edit the product and quantity on an unpaid ordinary invoice. Verify the item type and total change together. Start a gateway payment or submit a receipt; confirm the edit endpoint rejects changes.
4. From a conversion case, open each linked invoice. Resend a link to another number, copy its short link, and confirm both work only within the actor's invoice scope.
5. Add a secondary phone during issuance. Verify no creation SMS is sent to that number; explicitly resend to it and verify delivery.

JavaScript syntax and the repository static audit pass locally. PHP CLI and a configured WordPress/WooCommerce staging site are unavailable in this workspace, so PHP lint and transaction/SMS integration still need staging verification.
