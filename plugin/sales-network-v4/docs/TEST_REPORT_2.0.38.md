# Test report — Sales Network 2.0.38

Date: 2026-09-12

## Result

All executable static and parser checks available in the build environment
passed. A WordPress/PHP/MySQL staging smoke test remains required before
production deployment.

## Passed automated checks

- Full Sales Network static audit: PASS (36 check groups).
- Plugin header/constant version parity: PASS (`2.0.38`).
- All 15 JavaScript assets parsed by Node: PASS.
- Product-metabox inline JavaScript parsed after neutralizing PHP template
  expressions: PASS.
- PHP lexical validation: PASS for all 17 PHP files, with zero lexer errors.
- Duplicate PHP methods: 0.
- Literal include and asset references: 100 checked, PASS.
- AJAX action registration/usage wiring: PASS.
- Existing sales, staged-payment, assessment, conversion, archive, commission,
  Jalali/Tehran and role-routing regression contracts: PASS.

## Product* upgrade contract checks

- Additive upgrade-rules table and unique source/target key: PASS.
- Additive nullable membership-item snapshot columns: PASS.
- Product editor presence marker and repeatable UI wiring: PASS.
- Server validation for duplicate, invalid and non-increasing destinations:
  PASS.
- Subscription and direct Product* activation snapshot both credit and upgrade
  options: PASS.
- Current project action reads base credit from the card snapshot: PASS.
- Configured upsell amount/credit overwrite browser input on the server: PASS.
- Empty configured options disable upsell creation for new cards: PASS.
- Search found no runtime statement updating the new snapshot columns and no
  destructive `DROP`/`TRUNCATE` schema statement: PASS.

## Staging acceptance required

PHP CLI, WordPress and MySQL are unavailable in this build environment. Before
production, run these scenarios on staging:

1. Update from 2.0.37 and confirm `wp_sn_project_card_upgrade_rules` is created
   and the two nullable snapshot columns are added without changing old rows.
2. Define Silver Shoe (credit 3,000,000) -> Gold Shoe (credit 5,000,000) with
   an upgrade payment, save and reload the product editor.
3. Verify self-target, duplicate target, zero amount and a target credit less
   than or equal to the source are rejected while the old rule set remains.
4. Fully pay a subscription containing Silver Shoe and confirm its membership
   item stores the 3,000,000 base credit and configured upgrade JSON.
5. Change the live rule/credit, then confirm the existing membership still
   shows the old snapshot and a newly paid membership shows the new values.
6. Put either Product* in draft and confirm old invoice/membership/action and
   commission history remains readable and unchanged.
7. Tamper with target credit and amount in an upsell AJAX request; confirm the
   stored action uses the trusted snapshot values.
