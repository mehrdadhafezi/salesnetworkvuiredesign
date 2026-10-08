# Payment integrity and public invoice review — 1.0.148

This release is based on 1.0.147 and keeps the staged-payment workflow.

## Integrity fixes

- Reuses the same pending gateway request for the same invoice/stage/amount for 30 minutes.
- Uses a MySQL named lock while creating a gateway request to prevent double-click races.
- Marks older pending gateway requests expired when a fresh request is created.
- Finalizes the invoice stage before marking the gateway payment row paid.
- Rejects a stage amount that exceeds the current invoice balance.
- Locks invoice and stage rows during manual receipt submission and financial rejection.
- Replaces an existing pending receipt record instead of creating duplicates.
- Updates manual payment rows to approved/rejected with the financial decision.
- Validates Jalali payment date/time on the server and rejects future timestamps.
- Adds missing payment audit columns and an invoice/stage/status index through additive migrations.

## Customer invoice review

- Distinguishes completed invoices from staged pre-invoices.
- Shows total, approved paid amount, remaining balance, current stage and current due amount.
- Shows a payment completion progress bar.
- Shows stage history with Persian labels and Jalali dates.
- Shows a dedicated success message for partial-stage payments.
- Masks the middle digits of the customer mobile number in the UI.
- Disables card-to-card payment when the destination card is missing or invalid.
- Displays the exact amount due next to card-to-card instructions and payment buttons.
- Prevents zero-due pages from falling back to a stale previous amount.
- Adds client and server receipt file validation.

## Database safety

All schema changes are additive. No table or historical payment record is removed.
