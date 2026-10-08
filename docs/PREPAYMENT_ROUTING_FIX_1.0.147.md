# Prepayment routing fix 1.0.147

- Fixed the dedicated seller controller omitting `payment_plan` and `prepayment_amount`.
- Removed conflicting invoice-create click handlers at runtime and installed one authoritative handler.
- Added reliable preset/custom prepayment field toggling and live total/due/remaining preview.
- Public invoice now labels the current payment stage and displays its exact payable amount on payment buttons.
- Server-side validation remains authoritative: partial amount must be > 0 and < invoice total.
