# Test Report — 2.0.64

Scope: Product* zero-base-credit upgrade persistence regression.

## Static checks
- PHP lint: PASS for plugin PHP files.
- JavaScript syntax checks: PASS for relevant operations/project scripts.
- `node tools/static-audit.js`: PASS on version 2.0.64.
- Legacy error guard `sn_upgrade_source_credit` / message requiring source credit > 0: removed from runtime code.

## Validations preserved
- Source product must still be Product*.
- Target must be a different valid Product*.
- Duplicate upgrade targets remain rejected.
- Upgrade payment amount must remain > 0.
- Target credit must remain greater than source credit; therefore source `0` accepts any valid positive-credit target.
- Existing membership/card snapshots are unchanged.

## Staging smoke test
1. Edit a Product* and set base credit to `0`.
2. Add at least one upgrade target with positive target credit and positive upgrade amount.
3. Save/update the product; confirm no validation error is shown and base credit remains `0` after reload.
4. Create a new card from the Product*; confirm `0` is not shown as customer credit and normal activation is not offered.
5. Confirm the configured upgrade option is available and can enter the upgrade-payment flow.
