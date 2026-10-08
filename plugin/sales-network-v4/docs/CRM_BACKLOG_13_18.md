# CRM backlog 13–18 — release 2.0.111

| Item | Implementation |
| --- | --- |
| CRM-13 | A one-click "Today Tehran" button sets the current Jalali date and Tehran time in manual payment forms. |
| CRM-14 | Staff manual payment forms use separate 24-hour and minute selects. The customer form also uses separate selects and Tehran time. |
| CRM-15 | Seller, supervisor, and converter forms accept card details, the current stage amount, date/time and an optional receipt file in one submission. Seller receipt upload is authorized only for the seller's own invoice through the existing internal manual-payment access check. The customer manual submission can also include a receipt. Server verifies the submitted amount against the current stage. |
| CRM-16 | Finance review dialog presents current status, customer, invoice, amounts, receipt, inline review history, and approval/rejection actions; rejection reason is entered in the dialog. |
| CRM-17 | Invoice-scoped finance history is accessible from seller, supervisor, manager and converter invoice rows. The endpoint checks each user's invoice scope. |
| CRM-18 | On resubmission, invoice and stage enter pending finance approval; the current rejection fields are cleared while every rejection reason and later review remains in activity history. The seller may submit a corrected receipt in the same correction form. The finance list labels a resubmitted pending invoice "در انتظار بررسی مجدد". |

## Staging checks

1. With the browser configured outside Tehran, use Today in seller, supervisor, converter and customer payment forms; verify the current Jalali date and 24-hour Tehran time on submission. Try exact minutes and an invalid future timestamp.
2. Submit the current stage amount, card digits and a small valid receipt together as a seller. Confirm Finance sees one pending submission and the receipt. Try another seller's invoice and a different amount; both must fail.
3. Reject a receipt with a reason, resubmit corrected details and file, then verify current status becomes pending review again and the previous rejection appears only in the timeline. Reject it again and verify both reasons are preserved.
4. Open the finance review dialog and check customer, invoice, amounts, file, history and approve/reject controls. Verify scoped history is available to each authorized role, and denied for an unrelated invoice.

Local JavaScript syntax and static audit pass; PHP CLI and configured WordPress/WooCommerce staging are unavailable in this workspace.
