# HR — Access Discovery and Targeted Code Review

Date: 2026-10-06, Asia/Tehran. Environment: https://crm.maximumclub.ir/.

**HR ROLE RUNTIME NOT LIVE VERIFIED.** This note is not a runtime HR audit. Admin access is not substituted for HR.

## Existing safe access paths

The session was Admin #1. `/crm-hr/` workforce showed six total profiles, six displayed, zero hidden by filter/limit and no paging controls in that view. Selected positions were Deputy, Manager, Senior Supervisor, Supervisor, Seller and MIS. No real HR account was available there. Admin Integration had HR rebuild/profile-from-users controls, not a separate HR-account login control in the inspected HR-related entries. No rebuild, profile creation or settings action was invoked.

These are LIVE VERIFIED access-discovery observations only. They do not prove absence of every WordPress HR user or HR runtime behavior. Earlier accepted access discovery remains the broader source; this stage did not restart all account/role discovery.

## Targeted source facts

- `sn_can_manage_hr_panel:9112`: Admin branch, or logged-in `sn_hr` role, or HR position `hr` through direct/scope service checks.
- `sn_hr_apply_user_access_for_position:9158`: position maps to plugin-managed operational roles/capabilities; workforce edits can affect access. Unrelated roles/history are not intentionally reset by that mapping function.
- `handle_hr_view_staff_panel:2287`: HR-management gate, no nested viewing, target-specific nonce, internal non-Admin/non-self target and resolved non-placeholder route; actor/target transient and session switch for 30 minutes. Handler does not establish a universal readonly sandbox or a durable impersonation audit trail.
- `handle_hr_set_user_password:19060`: HR-management gate and nonce, existing WP user and password checks; password reset then optional SMS; audit payload explicitly excludes password. The single-target handler inspected does not show the same internal-user/non-Admin target restriction as the impersonation handler. No exploit or runtime enforcement test performed.
- `SN_Hierarchy_Service::assign_parent:29`: profile existence/self-parent/cycle checks, closes previous assignment, inserts new current assignment with application-time effective_from and history. Atomic close+insert failure behavior is not established.
- `sn_hr_change_request_can_review:10397` / apply / review `10439`: HR authority can review pending_review or pending_hr; Sales actors only their assigned pending_review step. HR transfer calls assign_parent; termination updates inactive/employment/termination date and user activity meta. Request successful branch records approved/applied_at, failure branch records failed. Complete handover and robust write-result checks are not established.
- Compensation timeline `19614`: APPLY, date/reason/profile checks, closes previous interval before final overlap/insert; no encompassing transaction in inspected handler. Existing compensation update helper and history paths differ; runtime payroll locks/rollback behavior not established.

## Required future HR access

A non-Admin test user with existing WordPress role **sn_hr**, base **read**, and an active HR profile with position slug **hr**, routed to `/crm-hr/`, is suitable for verification across the distinct frontend/request gates. `sn_manage_hr` is not an invented required capability. No account or role was created/changed in this stage.

## Limits

No workforce, hierarchy, onboarding, access, credential, compensation, request, export or bulk mutation was performed. No impersonation beyond inspecting the available access paths was used. All HR role-specific reads/writes, negative enforcement, concurrency, failed-apply recovery and sensitive-field/export behavior remain NOT LIVE VERIFIED. Target requirements are distinct from successful implementation claims.

## Local validation

The HR specification has 20 ordered top-level sections and no broken local document links. All 248 baseline plugin file hashes remained unchanged. Input ZIP SHA256 remained `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. Only this evidence note and the HR specification were created in this stage; no implementation files were edited.
