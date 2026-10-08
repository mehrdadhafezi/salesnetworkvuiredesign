# Commission Auto Rule Inheritance - v1.0.61

## Business rule
Commission rules are defined in Finance and apply automatically to users based on their HR position and level.

HR no longer needs to assign a commission model per employee by default.

## Default behavior
- Finance defines active rules with position/level filters.
- Commission dry-run and wallet calculation match all applicable finance rules automatically.
- Missing per-employee compensation profile no longer disables commission.
- `commission_enabled=0` in old HR history is not treated as a blocker unless the manual override mode is explicitly `disabled`.

## Manual exceptions
HR may create a historical compensation period with one of these override modes:

- `inherit`: automatic finance rules by position/level
- `manual_rule`: force a selected commission rule for that employee
- `disabled`: exclude that employee from commission

## Database migration
Adds `commission_override_mode` to:

- `sn_hr_compensation_profiles`
- `sn_hr_compensation_history`

Default: `inherit`.

## Calculation impact
Dry-run, seller commission preview, and wallet commission payload now call the same override resolver:

1. Match finance rules by position/level/status/payment/date.
2. Apply HR override only if explicitly set.
3. Calculate all matching rules unless dry-run mode selects best-match or selected-rule.

