# Sales Network 2.0.64 — Zero Base Credit Upgrade Fix

## Fix
- Product* can now persist a base credit of `0` while having one or more active upgrade rules.
- The legacy validation that required source/base credit to be greater than zero before saving upgrade rules has been removed.
- Target Product* credit must still be greater than the source credit. Therefore, for a zero-credit source card, upgrade targets must still have credit greater than zero.
- Upgrade amount must still be greater than zero and all previous target/type/duplicate validations remain intact.
- Existing card snapshots and historical records are not modified.

## Regression expectation
- Saving Product* with base credit `0` + a valid upgrade target succeeds.
- Product* with base credit `0` does not expose normal activation and remains upgrade-only until an upgrade is selected/paid.
- Existing Product* with positive credit continues to behave exactly as before.
