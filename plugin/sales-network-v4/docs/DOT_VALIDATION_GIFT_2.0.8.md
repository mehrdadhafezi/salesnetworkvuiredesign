# Dot Validation Gift — 2.0.8

## Business behavior

- Marketing Dot landing data creates its source pre-invoice before the confirmation popup opens.
- Popup abandonment, OTP abandonment and gateway abandonment therefore remain visible as pre-invoices in the exact supervisor scope.
- The supervisor invoice tab exposes `ارسال لینک انتخاب` only for an unpaid Dot assessment pre-invoice assigned to that supervisor.
- The action does not record revenue, a customer payment or finance approval.
- It closes only the assessment payment stage as `gifted`, keeps the source invoice in `pre_invoice`, opens the Dot selection case and sends the dedicated gift-access SMS.
- Supervisor and converter case views display `اعتبارسنجی هدیه`.

## SMS settings

Dot Settings now includes:

- `gift_access_pattern`
- `gift_access_template`

Available pattern variables:

- `customer_name`
- `access_code` / `code`
- `access_note`
- `gift_message`
- `access_link` / `link`
- `nominal_credit`
- `invoice_code`

The SMS link is the existing protected Dot customer-selection link and retains the normal access-code security check.

## Safety and audit

- Exact supervisor ownership and the shared public nonce are checked server-side.
- Paid, partially paid, receipt-uploaded and finance-review invoices cannot be gifted.
- Pending gateway requests are marked `cancelled_gifted`; later payment starts and callbacks are blocked.
- Repeated clicks never create a second case. A failed SMS can be retried without reapplying the gift.
- Actor, timestamp, SMS delivery/error and a case log are retained.
- Older abandoned Marketing Dot rows are recovered in batches only when their saved form, product and supervisor still exactly match the active route.

## Deployment

1. Install/replace the plugin ZIP.
2. Open a CRM/admin page once so the 2.0.8 database upgrade runs.
3. In `دات ستینگ`, enter the dedicated gift SMS pattern and save settings.
4. Submit a staging landing form, close the popup, and verify its pre-invoice in the assigned supervisor panel.
5. Click `ارسال لینک انتخاب`, verify the SMS, select an option, assign the case to a converter, and confirm the gift tag in both panels.

