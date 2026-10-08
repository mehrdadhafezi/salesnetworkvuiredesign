# Sales Network 2.0.30 — Assessment SMS choice and seller queue

## Behavior

- The seller invoice form shows `ارسال اعتبارسنجی` only for assessment products.
- Checked (default): after full settlement and final Finance approval, the established customer subscription-selection SMS is sent.
- Unchecked: no customer access SMS is queued, retried, or sent; the case enters conversion directly as `ready_for_conversion`.
- The SMS choice is independent from `ارسال برای سرپرست و تبدیل‌کننده` versus `تبدیل توسط خودم`.
- A `seller_self` assessment is materialized in the seller's `آماده‌های تبدیل من` queue after full payment and Finance approval.

## Data safety

- `send_assessment_sms` is stored on the invoice link as an invoice-time snapshot.
- `access_sms_enabled` is copied to the case and enforced by the due worker, dispatcher, and sender.
- Both new columns default to `1`, preserving the behavior of every existing link/case and cached legacy form.
- No invoice, payment, product, commission, or history row is rewritten.
- Missed paid callbacks are repaired in bounded, actor-scoped, idempotent batches.
