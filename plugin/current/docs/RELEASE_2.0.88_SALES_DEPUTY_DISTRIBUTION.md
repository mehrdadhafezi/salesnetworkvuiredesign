# Sales Network 2.0.88 — Sales Deputy number allocation

## Request

Expose the existing `تخصیص شماره` workflow inside the Sales Deputy panel without creating a second allocation engine or weakening hierarchy rules.

## Implementation

- Added a new `sd-distribution` / `تخصیص شماره` tab to `sn_sales_deputy_panel`.
- The tab reuses the guarded progressive V4 allocation renderer already used by Sales Manager / Senior Supervisor / Supervisor panels.
- Sales Deputy recipient scope remains constrained by the existing HR hierarchy contract: only active descendant Sales Managers, Senior Supervisors, Supervisors and Sellers are offered.
- Transfers still require the current user to own the distribution item; administrator bypass remains unchanged.
- Existing case/batch filter, per-recipient count allocation, multi-recipient selection, level filter, duplicate hard-guard and safe return workflow are reused unchanged.
- The return-note default was made role-neutral (`برگشت تخصیص به پنل من`) because the shared component is now rendered in multiple sales roles.
- No database migration and no destructive data rewrite are included.

## Safety

The change does **not** add a new direct SQL allocation path. It only exposes the existing allocation workflow in the Sales Deputy panel. Backend recipient resolution already recognizes `sales_deputy` and limits targets to HR descendants at allowed sales levels.

## Runtime acceptance

On staging, log in as a Sales Deputy with at least one owned V4 distribution item and active HR descendants. Confirm: the new tab opens; only descendants are listed; level filtering works; allocating N records moves only owned records; the destination user receives them; and safe return can return eligible unconverted records to the deputy.
