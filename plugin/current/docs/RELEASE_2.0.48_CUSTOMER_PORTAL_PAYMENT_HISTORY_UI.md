# Release 2.0.48 — Customer portal payment history and UI cleanup

## Scope

- Locks customer-portal section, membership, card and timeline heading colors against theme-level typography overrides.
- Compacts activity timeline rows and removes inherited article/container minimum heights.
- Uses an auto-fit membership-card grid so a single card fills the available row.
- Adds a responsive payment-history section with confirmed totals, count and last confirmed payment date.
- Shows invoice code, payment stage, amount, channel/provider, status, date and reference id.

## Privacy boundary

Payment records are never searched globally by phone. The portal first resolves invoices through the authenticated customer id, with the existing normalized-phone fallback only for legacy invoices whose `customer_wp_id` is zero. The payment query then accepts only those invoice ids.

## Compatibility

- Successful payment records: `approved`, `paid`, `verified`.
- Pending and failed attempts remain visible with separate status styling.
- Paid or approved legacy invoices with no successful `sn_payments` row receive one invoice-level fallback record.
- No gateway request, verification, callback, finance approval, invoice total or schema behavior changed.

## Staging checks

1. Open the customer portal under the production theme and verify all headings are readable.
2. Verify one-card and multi-card subscriptions on desktop and mobile.
3. Compare payment rows with `sn_payments` for the same customer's invoices.
4. Verify another customer's payment reference cannot appear in the profile.
5. Check approved gateway, approved card-to-card, pending, cancelled and rejected states.
6. Confirm OTP login still uses the configured Faraz customer-login pattern.
