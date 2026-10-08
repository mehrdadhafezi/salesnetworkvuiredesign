# Sales Network 2.0.89 — Marketing request neutral copy

## Scope
Customer-facing Marketing Dot Flow form only. Internal database keys, flow identifiers, accounting logic, product configuration, and Dot assessment mechanics are unchanged.

## Changes
- Replaced credit-specific headings and CTA with neutral request wording.
- Primary CTA is now «ادامه درخواست».
- Success and validation messages use «درخواست» only.
- Removed the WooCommerce product name from the customer confirmation modal so product titles containing «اعتبار» or «اشتراک» cannot leak into the form.
- Reworded customer-visible gifted-request and expired-code messages to avoid credit/assessment terminology.
- Reworded OTP SMS validity text without using the word «اعتبار».
- Added a static audit guard for neutral customer copy.

## Safety
No database migration. No schema change. No payment-routing change. No permissions change.
