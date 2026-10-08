# Test report — 2.0.86 card payment access hotfix

## Root cause

- Internal card-to-card actions use the `sn_submit_manual_payment` / receipt endpoints without a public `access_token`, which is correct for authenticated CRM panels.
- In 2.0.85, when internal authorization returned false, the handler fell through to public invoice-token validation and returned the misleading expired/invalid-link message.
- Manager invoice visibility and payment write authorization were also using different scope resolvers, so a visible invoice could fail only at payment submission in some HR hierarchy states.

## Fix verification

- Plugin header and `SN_VERSION`: `2.0.86`.
- PHP syntax: PASS — 22 PHP files.
- JavaScript syntax: PASS — 18 JS files.
- `node tools/static-audit.js --json`: PASS (`ok: true`).
- Internal denied requests without a public token no longer fall through to public-token validation: PASS.
- Valid public-token fallback is preserved for customer links, including when a staff user is logged in: PASS.
- Supervisor / senior supervisor / sales manager payment scope uses canonical `SN_Scope_Service::can_view_invoice()` with the previous actor-aware scope check retained as fallback: PASS.
- Seller restriction preserved: manual deposit info only; receipt upload remains blocked: PASS.
- No DB migration and no data mutation required for upgrade: PASS.

## Live QA recommended after deployment

1. Seller: own normal invoice → manual deposit info succeeds; receipt upload is not available.
2. Seller: self-conversion invoice → manual deposit info succeeds.
3. Supervisor: invoice visible in own scope → manual deposit info succeeds and receipt upload succeeds in allowed statuses.
4. Converter: owned conversion invoice → manual deposit info / receipt upload succeeds in allowed statuses.
5. Out-of-scope internal invoice → explicit access error, never “expired invoice link”.
6. Customer public invoice link → invoice loads with token; manual card info and receipt upload continue to validate the public token.
