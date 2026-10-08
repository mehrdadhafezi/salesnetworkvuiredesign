# Test Report — 2.0.63

## Static checks
- All PHP files: `php -l` PASS.
- Operations JavaScript: `node --check assets/js/operations-flow.js` PASS.
- Project static audit: PASS.

## Recommended staging smoke test
1. Assign an upgrade-capable card to an Operations Sales Expert.
2. Create an upgrade invoice with staged payment and a custom/preset first amount below the total.
3. Verify customer SMS/payment page shows only the first-stage due amount.
4. Approve stage 1 in the existing finance/payment flow.
5. Verify the case remains in the same Operations Sales Expert panel and does not appear in the seller repeat-action queue.
6. Verify total, paid, remaining and current stage values.
7. Issue a second custom stage and approve it.
8. Issue a final stage using “settle full remaining”.
9. Approve final payment and verify the case moves to Executive Operations only after remaining balance reaches zero.
10. Retry/resend an open stage and verify the SMS amount equals the current stage due, not the invoice total.
11. Double-submit stage issuance and verify only one stage is created.
