# Release 2.0.52 — Customer profile and date-only callbacks

## Changes

- Callback selection is date-only for the customer, Operations Sales Manager, Supervisor and Expert.
- The selected Jalali day is validated in Tehran and stored in the existing `follow_up_at` field at 09:00 Tehran for scheduler compatibility.
- Existing callback rows and WordPress scheduled reminders are retained; no table or historic row is removed.
- The Jalali calendar is mounted as a viewport-level floating dialog so card overflow and theme containers cannot clip it.
- Past calendar days are disabled and server-side validation rejects past dates.
- Customer location reads account metadata first and falls back to the latest invoice already scoped to that authenticated customer.
- The customer profile has refreshed navigation, identity cards, responsive content cards and compact success/error notices.

## Safety

- No destructive schema or data migration is introduced.
- Invoice, payment gateway, commission, wallet, OTP, normal/upsell and Operations routing logic is unchanged.
- The existing `DATETIME` storage contract is preserved so earlier data remains readable.

## Staging acceptance

1. Open the customer profile on desktop and mobile; verify the picker is visible above every card and accepts today/future dates.
2. Verify customer expert activation submits with a date only.
3. Verify Manager and Supervisor callback editors submit with a date only.
4. Verify Expert and self-assigned Supervisor `تماس مجدد` status submits with a date only.
5. Confirm the chosen date appears without an hour in customer and staff callback labels.
6. Test a customer whose user metadata lacks location but whose invoice has province/city.
7. Re-test normal activation, upsell invoice/payment and automatic execution routing.
