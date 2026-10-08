# Merge release 2.0.17

## Scope

- Builds on the integrated 2.0.16 package containing seller stage one, HR transfer, Campaign/UTM and the atomic pre-invoice correction.
- Adds the supplied Dot-flow ability for a supervisor to assign eligible cases to themselves.
- Keeps subordinate converter discovery and team statistics separate from the self-assignment target.

## Access and ownership rules

- The self target is the case's own supervisor, not an arbitrary user.
- A supervisor can act as converter only when both `converter_id` and `supervisor_id` equal their own user ID.
- The converter workspace query applies both ownership predicates for a supervisor-only account.
- A supervisor can see the converter navigation entry, but does not inherit unrestricted manual-invoice issuance unless they independently have the converter role/position.
- Single and bulk assignment retain nonce, live case eligibility, row locking and subordinate-target validation.

## Staging acceptance

1. Assign one ready Dot case to the logged-in supervisor and confirm it appears under “پنل تبدیل‌کننده (تخصیص‌های خودم)”.
2. Bulk-assign multiple ready cases to the same supervisor and confirm all-or-nothing persistence.
3. Attempt to access or mutate a case assigned to another supervisor/converter and confirm rejection.
4. Confirm a supervisor-only account cannot issue an unrelated manual invoice from the converter workspace.
5. Run `node tools/static-audit.js` and the `2.0.17` section of `docs/QA_CHECKLIST.md`.
6. Run PHP lint plus WordPress/WooCommerce payment smoke tests on Staging.
