# Release 2.0.102 — Optional destination-card last four for sales payment entry

## Goal
Allow sales-team users to submit customer manual payment information without entering the destination card's last four digits.

## Behavior
- The source card last four digits remain mandatory and must be exactly 4 digits.
- Jalali payment date and Tehran time remain mandatory and use the existing validation path.
- The destination card last four digits are optional for internal sales-team manual payment entry.
- If a destination value is entered, it must still be exactly 4 digits.
- Converter/supervisor receipt upload also accepts an empty destination last-four; if entered, it must be valid.
- Public/customer card-to-card payment remains strict and still requires a valid configured destination card.
- Existing full-card snapshots and Finance last-four display compatibility are unchanged.

## Updated surfaces
- Seller manual-payment modal.
- Converter manual-payment modal.
- Supervisor manual-payment modal.
- Internal manager/supervisor invoice payment handlers.
- Converter/supervisor receipt upload.
- Seller/converter financial resubmission compatibility path.

## Safety
The server determines whether the request is internal before allowing an empty destination value. A public customer request cannot use the internal empty-destination exception.
