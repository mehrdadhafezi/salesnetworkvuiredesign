# Release 2.0.57 — Customer Profile and Purchase Awareness

## Scope

- Removes the customer-facing timeline navigation and section.
- Reworks the profile into a compact, responsive dashboard inspired by the supplied HTML: identity, credit wallet, account summary, compact membership/cards, purchases and payments.
- Keeps card actions in the existing modal so details no longer expand beneath a card.
- Adds invoice-item and payment-stage based reasons to every row in «سوابق پرداخت».
- Adds an informational warning to the public payment page when the same customer has a confirmed purchase of the exact subscription product.
- Adds a small «خریداری‌شده» marker to matching subscription options in seller and converter invoice forms after a valid customer mobile is entered.

## Duplicate definition

The match is intentionally exact by WooCommerce product ID and only applies to products whose effective sales type is `subscription`. Project memberships are the authoritative history; confirmed legacy invoices are the fallback. The currently viewed invoice is excluded from its own comparison.

The warning does not disable online payment, card-to-card payment or invoice creation.

## Security and compatibility

- Public invoice access-token and nonce validation are unchanged.
- The internal phone lookup requires authentication, an accepted nonce and seller/converter permission, and returns product IDs only.
- No payment callback, verification, SMS, wallet, commission or destructive reset behavior changed.

## Staging acceptance

1. Open an existing customer profile and confirm there is no timeline.
2. Confirm purchases and payment reasons match actual invoice items and stages.
3. Enter a mobile with a confirmed subscription in seller and converter forms; the exact subscription must show «خریداری‌شده».
4. Select another subscription; it must remain unmarked.
5. Open a new invoice for the exact previously purchased subscription; the requested Persian warning must appear without blocking either payment method.
6. Repeat with a first-time subscription buyer; no warning or tag must appear.
7. Check desktop and mobile layouts, card modal actions, OTP login and province/city display.
