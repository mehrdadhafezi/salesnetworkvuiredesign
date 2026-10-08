# Release 2.0.55 — Upgrade invoice SMS parity and no callback SMS

## Scope

- Operations callback dates are still saved and displayed, but never schedule or send an SMS.
- Existing callback reminder cron events are unscheduled once after the update.
- Operations upgrade-payment invoices use `SN_SMS::send_invoice_link()`, the same service used by normal Sales invoices.
- Faraz therefore uses the existing `sn_faraz_pattern_invoice` pattern and its normal variables: `customer_name`, `invoice_code`, `invoice_url`, `amount`, and `card_number`.
- `amount` is exactly the upgrade amount stored on the Operations case.
- New upgrade invoices receive a unique `short_code`; existing upgrade invoices receive one lazily when their payment link is resent. The long tokenized URL remains the fallback for legacy schemas.

## Expected customer SMS

The message body is controlled by the same configured Sales invoice pattern. With the currently configured pattern it is expected to render in this form:

```text
جناب آقای {customer_name} گرامی
فاکتور شما با مبلغ {amount} تومان صادر گردید.
لطفا جهت مشاهده فاکتور به لینک زیر مراجعه فرمایید.
در لینک روش پرداخت آنلاین و کارت به کارت موجود است
مشاهده فاکتور : {invoice_url}
بیاوین
```

This release does not hard-code that text. It intentionally delegates rendering to the existing Sales invoice pattern so Sales and Operations stay identical.

## Staging acceptance

1. Select and update a callback date as manager, supervisor, and expert; confirm the date appears in the case and customer profile.
2. Run due cron processing and confirm no callback SMS is sent and no `sn_operations_followup_reminder` event remains.
3. Create a new upgrade invoice and verify the SMS uses `sn_faraz_pattern_invoice`.
4. Verify the SMS amount equals the upgrade amount only.
5. Verify the SMS URL has the `/i/{short_code}/` form and opens the correct token-protected invoice.
6. Retry an existing upgrade invoice and repeat checks 3–5.
7. Confirm normal Sales invoice SMS behavior is unchanged.

## Rollout note

Run the test on Staging with the real Faraz configuration before Production. Local static checks cannot confirm provider delivery or pattern-variable naming on the remote account.
