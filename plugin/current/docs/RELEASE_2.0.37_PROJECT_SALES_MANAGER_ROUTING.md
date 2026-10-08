# Sales Network 2.0.37 — Project routing by sales-manager branch

## Outcome

Each subscription content/project can now keep its existing default project
manager and optionally define one or more branch-specific routes:

`content product + source sales manager -> project manager`

For example, the shoe content inside a Gold subscription can route sales from
sales manager X to project manager A and sales from sales manager Z to project
manager B. X and Z can also both be mapped to A.

## Upgrade and data safety

- The existing `sn_project_manager_products` mapping remains the default and is
  not migrated or deleted by the upgrade.
- A new additive `sn_project_manager_sales_routes` table stores only optional
  branch overrides.
- Existing memberships and membership items are never reassigned. Routing is
  evaluated only while a new fully-paid membership is first activated.
- New memberships snapshot `source_sales_manager_user_id`, the chosen manager,
  the default manager and whether an override was used.
- Old/cached settings forms do not contain the new presence marker, so saving
  one cannot accidentally erase branch routes created by the new form.
- If a source sales manager cannot be resolved, or no exact branch route is
  configured, the existing default project manager is used.

## Source attribution

The source branch is derived from the original seller fields on the invoice.
For Dot/conversion payments, the case seller is also loaded as a compatibility
fallback. The resolver follows the current HR `reports_to` chain up to the
sales-manager position, with the legacy user-meta chain retained as a fallback.
The converter or the person who clicked "issue invoice" therefore does not
replace the original sales branch.

## Admin workflow

In the existing project-content settings row:

1. Select the required default project manager.
2. Under "مسیرهای اختصاصی مدیر فروش", add any number of rows.
3. Select the source sales manager and destination project manager for each row.
4. Save the settings.

The form rejects incomplete rows, inactive/invalid users and duplicate routes
for the same source sales manager within one content product.

## Verification

The package static audit checks schema uniqueness, original-seller attribution,
the default fallback, presence-marker protection, immutable historical rows and
JavaScript/PHP wiring. A staging WordPress/MySQL smoke test is still required to
exercise real HR data and `dbDelta` on the target server.
