# Test report — Sales Network 2.0.52

- Version: `2.0.52`
- Build: `2026-09-15-customer-profile-date-only-v1`
- Test date: 2026-09-15

## Passed locally

- JavaScript syntax: `assets/js/operations-flow.js`
- JavaScript syntax: `assets/js/customer-portal.js`
- JavaScript syntax: `tools/static-audit.js`
- Full static audit: PASS, including customer-profile refresh and executable Jalali conversion/status-form tests.
- Date-only invariant: active customer, Manager, Supervisor and Expert forms contain no `followup_time` input.
- Calendar invariant: picker mounts under `document.body`, is fixed above theme containers, and past days are disabled.
- Location invariant: authenticated invoice query includes `province` and `city`; profile metadata remains the first source.
- Data safety: no `DROP`, `TRUNCATE`, destructive migration or historical rewrite was added.

## Runtime status

PHP CLI, WordPress, WooCommerce and MySQL are not available in this workspace. PHP lint and end-to-end runtime acceptance therefore remain required on Staging before Production. Follow `docs/RELEASE_2.0.52_CUSTOMER_PROFILE_DATE_ONLY.md`.
