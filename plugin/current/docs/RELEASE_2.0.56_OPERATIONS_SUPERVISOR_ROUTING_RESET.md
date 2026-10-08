# Release 2.0.56 — Operations supervisor routing and protected reset

## Scope

- Operations manager and supervisor panels now expose an explicit Select All checkbox for visible, assignable cards.
- HR workforce settings provide a per-supervisor invoice source policy:
  - `manual`: manual issuance plus assigned MIS numbers.
  - `mis_only`: assigned MIS numbers only, inherited by descendants.
- Biavin card settings now map `content card + source Sales Manager` directly to an Operations supervisor under the selected Operations manager.
- New operation rows snapshot both destination IDs. Existing operation rows are not reassigned.
- Customer-requested and inactivity-routed cards enter `sales_supervisor` directly when a matching route exists; otherwise they fall back to the Operations manager queue.
- A protected reset tool deletes operational data and non-admin users while preserving administrators, pages, products, WordPress options and structural configuration.

## Safety boundaries

- Route save validates the source Sales Manager, Operations manager, Operations supervisor and direct HR parent relation.
- The reset is never automatic. It requires `manage_options`, a dedicated nonce, exact text `حذف کامل داده‌ها`, a checkbox and a browser confirmation.
- Posts and pages owned by deleted users are reassigned to the administrator executing the reset.
- The reset records aggregate counts in `sn_last_full_data_reset_report` and never exposes deleted record contents.

## Staging acceptance

1. Configure a card with an Operations manager and two source-manager-to-supervisor routes.
2. Create a sale from each Sales Manager branch and verify the new card snapshots the expected manager/supervisor.
3. Trigger customer expert activation and inactivity routing; verify the card appears directly in the mapped supervisor panel.
4. Verify unmatched branches appear in the selected Operations manager queue.
5. Filter Operations cards, use Select All and verify only visible assignable rows are selected.
6. Set a supervisor to `mis_only`; verify the supervisor and a descendant cannot create a no-lead invoice but can issue from an assigned MIS number.
7. Set the policy to `manual`; verify manual issuance is restored, subject to the existing per-user access switch.
8. On a disposable Staging copy, run the reset and verify non-admin users and operational rows are removed while pages, products and settings remain.

## Rollout note

Run database upgrade and all destructive-reset checks only on a disposable Staging copy before Production. PHP/MySQL runtime behavior cannot be proven by local static analysis alone.
