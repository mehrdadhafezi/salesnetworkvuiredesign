# Release 2.0.49 — Operations callback scheduling and retained case history

## Scope

- Customer-selected Jalali callback date and Tehran time for expert activation.
- Manager/Supervisor callback scheduling for inactivity-routed cards.
- Customer-facing callback visibility.
- Retained case ownership across later workflow states.
- Logout, status filtering and four overall KPIs in Operations Sales panels.
- Originating Sales Manager visibility on every Operations case.
- Supervisor self-assignment with expert-equivalent case actions.

## Workflow contract

1. Manual customer expert activation requires a valid future date and time.
2. Automatic inactivity routing keeps the callback empty until the assigned Operations Manager or Supervisor schedules it.
3. Manager ownership is retained by `operations_sales_manager_user_id`; Supervisor and Expert history is retained by their assignment ids rather than the current stage alone.
4. Assignment controls are shown only while the case is in the matching assignment stage.
5. Completed and cancelled cases remain visible as read-only history and can be filtered by status.
6. A Supervisor using «تخصیص پرونده به خودم» becomes the case worker and can use the same outcome controls as an Expert.

## Staging acceptance

- Create one customer-selected expert case and confirm the submit button remains disabled until date and time are selected.
- Verify the stored Jalali time appears in the customer profile and all assigned Operations panels.
- Let a fresh card auto-route, schedule its callback as Manager, edit it as Supervisor, and verify the customer view after each change.
- Assign one case to an Expert and one to the Supervisor themself; complete one and cancel the other.
- Confirm both records stay in Manager/Supervisor/Expert history as applicable.
- Confirm search and status filters combine correctly and no-result state appears correctly.
- Confirm every Manager card shows the originating Sales Manager.
- Confirm logout returns to the site home page.
- Run PHP lint on PHP 8+, WordPress smoke tests and the repository static audit before Production.

## Safety

- No destructive database migration is used; the existing `follow_up_at` field stores callback time.
- Server-side capability, ownership, stage and nonce checks remain authoritative.
- Payment, OTP, wallet and commission pathways are unchanged.
