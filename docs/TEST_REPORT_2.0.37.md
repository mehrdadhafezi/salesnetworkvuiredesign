# Test report — Sales Network 2.0.37

Date: 2026-09-12

## Result

All executable checks available in the build environment passed. The package is
ready for a staging WordPress/MySQL smoke test.

## Passed checks

- Full project static audit: PASS.
- Plugin version/header parity: PASS (`2.0.37`).
- All 15 JavaScript files parsed by Node: PASS.
- Inline JavaScript added to project settings parsed by Node after PHP-template
  placeholders were neutralized: PASS.
- PHP lexical delimiter validation with the PHP lexer: PASS for all 17 PHP
  files.
- Duplicate PHP method declarations: 0.
- Undefined local method/hook callback checks: PASS.
- Literal include/asset references: 100 checked, PASS.
- Registered/used AJAX action wiring: PASS.
- Cross-panel role routing and existing sales/conversion regressions: PASS.
- Source archive comparison: exactly three original files changed and two
  release/test documents added; no original files removed.

## Project-routing contract checks

- Unique key exists for `(content_product_id, sales_manager_user_id)`: PASS.
- Exact source-manager route lookup: PASS.
- Default project-manager fallback when no route exists: PASS.
- Original seller is evaluated before invoice issuer/converter: PASS.
- Dot case seller compatibility fallback: PASS.
- Subscription and Product* activation paths use routing: PASS.
- New settings presence marker protects routes from old/cached forms: PASS.
- New membership stores the source sales manager and route snapshot: PASS.
- No backfill/update of historical source-manager assignments: PASS.
- Incomplete, duplicate and invalid-user route validation exists server-side:
  PASS.

## Regression checks retained by the full audit

The existing automated audit also passed staged payments, assessment SMS choice,
conversion ownership, callback reminders, supervisor queues, draft-product
history safety, commissions, Tehran/Jalali date helpers, sales-manager archives,
WordPress init callback safety, shortcodes and portal page wiring.

## Staging acceptance still required

PHP CLI, WordPress and MySQL are not installed in this build environment, so the
following must be run on staging before production deployment:

1. Activate/update the plugin and confirm `dbDelta` creates
   `wp_sn_project_manager_sales_routes` and adds
   `source_sales_manager_user_id` to project memberships.
2. Configure shoe content with default A, X -> A and Z -> B.
3. Fully pay one Gold subscription from a seller under X and one under Z;
   confirm the shoe project appears only for A and B respectively.
4. Change Z -> A and confirm only a newly paid membership routes to A; the old
   Z membership must remain assigned to B.
5. Remove an exact route and confirm a new sale uses the default manager.
6. Test a Dot/conversion payment and confirm the original seller's sales-manager
   branch is used, not the converter's branch.
7. Save once from the current form, then repeat with a captured old form that
   lacks `sn_project_manager_routes_present`; existing routes must remain.

