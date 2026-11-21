# RM-P04 — HR operational workspace (read-first, controlled requests unchanged)

Status: implemented on `claude/rm-p04-hr-operational-workspace`. Base `main` = `d8f78de`.
Authority: AGENTS.md, `HR-PRODUCT-SPEC.md` (§5, §6, §11, §12, §17), `GATE-0-PRODUCT-INVARIANTS.md`, `prototype/hr/HR-DESIGN-SPEC.md`
(**READY_FOR_REVIEW — not frozen; this work does not change that status**).
No permission, capability, role, flag, status, workflow transition, financial rule or identity semantic is added or changed.
Group B, R5-SEC-01, D-34, D-42 and production readiness are untouched (PRODUCTION NO-GO).

## Workspace

One section navigation (`web/src/modules/hr/workspace/SectionNav.tsx`) on every HR page, in HR IA order:

| Section | Route | Read (own authorization) |
|---|---|---|
| نیروها (personnel directory) | `/hr` (existing) | `GET /organization/persons` (+ new level / active-flag filters, `user_id` shared-account view) |
| پرونده نیرو | `/hr/people/:id` (existing) | person detail + **new** account card (`GET /identity/users/{id}`, only with `identity.user.read`) + **new** caller-scoped requests (`GET /hr/requests?subject_user_id=`, only with `org.hr_request.read` and the request feature on) |
| ساختار سازمانی | `/hr/structure` (existing) | org units; unit member count links to the filtered directory |
| انتساب‌ها و گزارش‌دهی | `/hr/assignments` (**new page**) | `GET /organization/assignments` (person scope, CHILD/PARENT, current flag, kind, findings, cursor) |
| سمت‌ها و سطوح | `/hr/positions` (**new page**) | `GET /organization/positions`, `GET /organization/levels` |
| درخواست‌ها | `/hr/requests`, `/hr/requests/:id` (existing) | status queues (**new server filter**), cursor pager, people resolved from account ids, lifecycle, requested-vs-applied, history actors |
| استثناها | `/hr/exceptions` (existing) | hierarchy findings |
| دسترسی من | `/hr/access` (**new page**) | the shell's existing `GET /authorization/effective-access` evaluation (no second request) + own position via `persons?user_id=` |

A section is listed only when the shell's fail-closed resolver lists its destination (feature switch AND ALLOWED decision);
the current section and "دسترسی من" always stay visible. Hiding is presentation: every page renders the server's answer.

## Separations kept explicit

* Person ≠ Employment ≠ User account (account card: not a role, capability or credential; no password/reset/provisioning control).
* Position ≠ Role ≠ Capability ≠ Credential ≠ Grant (access page shows all five apart; "position grants nothing").
* Reporting line ≠ permission ("being a manager grants nothing"); current manager ≠ historical attribution.
* A shared account lists every linked person as `SHARED_USER` conflict — never one merged person.
* Request "step approval ≠ applied"; final apply is shown blocked (`HR_APPLY_IMPLEMENTATION_BLOCKED`, OPD-05, SD-04), never as a control.

## Backend (narrow, read only)

1. `GET /api/v1/organization/persons?user_id=` — exact employment user link. Refused (403) without `org.employment.read`
   (the link is a masked employment field). Returns every linked person.
2. `GET /api/v1/hr/requests?status=&subject_user_id=` — ANDed with the existing row scope (requester / current reviewer /
   history actor); never widens it. Invalid values are 400.

Both: OpenAPI + generated Go/TS clients, query allowlist in `apiserver`, READ ONLY transactions, proven by
`internal/integration/legacy/shadow/rm_p04_hr_workspace_test.go` (shared user, masking refusal, no scope widening, flag off, no writes).

## Not done (approval-dependent, not invented)

HR final apply, onboarding, transfer/termination execution, account provisioning, credential issue/reset, impersonation,
capability/role grants, position-role mapping, compensation, other people's effective access. HR runtime remains
NOT LIVE VERIFIED (overlay evidence is dev-stub only).
