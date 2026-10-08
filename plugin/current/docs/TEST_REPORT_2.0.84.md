# Test report — 2.0.84

## Automated checks

- `node --check assets/js/public-converter.js`: passed.
- `node --check tools/static-audit.js`: passed.
- `node tools/static-audit.js --json`: passed with no errors.
- Converter physical-address invariant: passed (form fields, conditional UI, request payload, and postal validation).
- Plugin header and `SN_VERSION`: both 2.0.84.

## Scenarios covered statically

1. Ordinary/physical product: address fields become visible and province, city, and full address are required.
2. Subscription, assessment, or product-star without an ordinary item: shipping fields remain hidden and do not block submission.
3. Mixed invoice containing an ordinary product: shipping rules apply to the whole invoice.
4. Optional postal code: empty is allowed; a supplied value must contain exactly 10 digits, including Persian/Arabic digit normalization.
5. Successful invoice creation clears the address fields together with the other customer fields.
6. Older clients sending `address`/`postal_code` remain compatible with the server endpoint.

## Environment limitation

PHP CLI and a running WordPress/WooCommerce database were not available in this workspace. Run `php -l` and one end-to-end converter invoice smoke test on staging before production deployment.
