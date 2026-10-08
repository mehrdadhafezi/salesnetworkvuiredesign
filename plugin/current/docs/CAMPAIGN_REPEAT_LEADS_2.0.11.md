# Campaign repeat leads and attribution — 2.0.11

## Expected flow

1. Landing UTM is captured on the page or, as a guaranteed fallback, from the form snapshot POST.
2. The first valid submit inserts one independent `sn_dot_marketing_leads` row.
3. The confirmation/OTP/final gateway POST reuses only that owned submission ID.
4. Closing the popup, cancelling at the gateway, gateway failure and successful payment all preserve the lead row and update its status.
5. A later attempt with the same phone creates another marketing submission and another campaign Lead/Registration conversion.

## Report definitions

- Lead/Registration: form submission count; repeat phones are included.
- Unique Visitor: distinct campaign visitor cookie.
- Paid Customer: distinct paid identity.
- Successful Payment: distinct paid invoice.
- Sales: sum of successful paid invoice amounts.

## QA

- Open a UTM landing in a private browser and submit the form once: Lead and Registration become 1 before payment.
- Cancel at the gateway and allow the callback to return: lead remains present with `gateway_cancelled` (or `gateway_started` when the gateway never returns a callback).
- Submit again with the same phone from a fresh landing attempt: Lead and Registration become 2.
- Complete one payment: Paid Customer becomes 1, Successful Payment becomes 1 and Sales contains the paid amount.
- Confirm the existing payment request/verify/callback implementation was not modified.
