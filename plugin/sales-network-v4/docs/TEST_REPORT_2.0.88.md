# Test report — 2.0.88 Sales Deputy allocation

## Static gates

- Plugin header and `SN_VERSION`: `2.0.88`.
- PHP lint: **22/22 PASS**.
- JavaScript syntax (`node --check`): **17/17 PASS**.
- Sales Network static audit: **PASS**.
- Static audit deputy-allocation invariant: `salesDeputyDistribution: 1`.
- Duplicate PHP methods: `0`.
- Database migration: **none**.
- Destructive rewrite: **none**.

## Contract checks

- `sn_sales_deputy_panel` contains the `sd-distribution` / `تخصیص شماره` tab.
- The tab is wired to `sn_render_supervisor_progressive_assign_tab($viewer_id)` rather than a new duplicate implementation.
- Backend recipient scope for `sales_deputy` is limited to active HR descendants at `sales_manager`, `senior_supervisor`, `supervisor`, and `seller` levels.
- Transfer authorization still requires item ownership (`current_owner_user_id === actor_id`) for non-admin users.
- Existing duplicate guard, case filter, count-per-recipient distribution and safe-return logic remain shared and unchanged.

## Runtime acceptance still required on staging

Use a real Sales Deputy account that owns at least one V4 distribution item. Confirm the new tab opens, only HR descendants appear, allocation transfers only owned rows, the receiving panel gets the rows, and safe return restores only eligible/unconverted rows.
