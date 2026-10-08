TASK
HR REDESIGN

MODE
DESIGN / PROTOTYPE ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

SUCCESS:
HR DESIGN V1 FROZEN — WORKFORCE & ACCESS CRM DESIGN SYSTEM EXTENDED

Then STOP.
Do not start Finance.

--------------------------------------------------
1. SOURCES OF TRUTH
--------------------------------------------------

PRODUCT:
- HR-PRODUCT-SPEC.md
- GATE-0-PRODUCT-INVARIANTS.md

VISUAL:
- all previously frozen CRM role designs
- shared CRM Design System

IMPORTANT:
HR ROLE RUNTIME IS NOT LIVE VERIFIED.

Do not visually imply that HR behavior was live-confirmed.
Use Product Spec + code-verified contracts as authority.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT re-run a full HR audit.

Codex already:
- verified HR code inventory
- confirmed no real HR runtime account
- documented workforce/structure/access/request/compensation/credential boundaries
- documented temporal and handover gaps

Use live/admin pages only for targeted content reference if needed.
Do not treat Admin behavior as HR evidence.

Do NOT:
- reread unrelated Sales/MIS code
- dump DOM
- capture redundant screenshots
- narrate every step
- retest lower-role frozen flows

Prefer:
Product Spec → shared components → HR-specific design → focused QA.

--------------------------------------------------
3. CORE DESIGN PRINCIPLE
--------------------------------------------------

HR = WORKFORCE LIFECYCLE + STRUCTURE + ACCESS + TEMPORAL HISTORY

Not:
- unrestricted WordPress Admin
- Sales operator
- Finance workspace
- MIS repair tool
- generic user-management screen

Every sensitive change should make clear:

- who is affected?
- what is changing?
- current state
- requested state
- effective time
- actual applied result
- what downstream impact exists?
- what remains unresolved?

--------------------------------------------------
4. TARGET IA
--------------------------------------------------

PRIMARY:
- Workforce
- Onboarding / Transfer
- Requests
- Structure & Access
- Compensation
- Exceptions / Audit

ADVANCED / RESTRICTED:
- Credentials / Impersonation

SECONDARY / CONDITIONAL:
- Import / Bulk / Export
- Legacy Mapping / Diagnostics

Do not expose every old HR tool as equal navigation item.

--------------------------------------------------
5. REUSE EXISTING SHARED SYSTEM
--------------------------------------------------

Reuse:

- AppShell
- RoleNavigation
- PageHeader
- Scope / identity context
- DataTable
- Drawer / Sheet
- Timeline
- ApprovalChain
- OperationLog
- BulkActionBar / BulkResult
- shared system states
- Help / Tour
- Toast / Tooltip
- Theme / Density / Focus Mode

Do not duplicate shared components.

--------------------------------------------------
6. WORKFORCE EXPLORER
--------------------------------------------------

Design around distinct entities:

- Person
- Workforce Profile
- WordPress User
- Position
- Level
- Hierarchy Assignment
- Unit / Department / Team
- Employment State
- Access State

Do not visually collapse them into one profile object.

Show linkage clearly:
Person ↔ Workforce Profile ↔ User Account

Conflicts should be visible:
- missing user
- duplicate profile
- role-position mismatch
- hierarchy gap
- access mismatch

--------------------------------------------------
7. TEMPORAL HISTORY
--------------------------------------------------

Design an Employment / Structure Timeline.

Clearly distinguish:

- Current Hierarchy
- Historical Manager
- Historical Position
- Effective From / To
- Applied At
- Event Actor
- Credit / historical attribution boundary

Missing history must appear as:
UNKNOWN / INCOMPLETE

Never infer past manager from today's hierarchy.

--------------------------------------------------
8. ONBOARDING
--------------------------------------------------

Design onboarding as staged but do not invent unsupported workflow.

Potential stages:

- Identity discovery
- Profile creation / linkage
- Position / level
- Hierarchy placement
- Access activation
- Completion / unresolved effects

Credentials should NOT automatically be embedded as ordinary onboarding if policy is restricted.

Do not invent:
- training workflow
- document approval workflow
- legal onboarding stages
unless Product Spec supports them.

--------------------------------------------------
9. TRANSFER
--------------------------------------------------

Design a Hierarchy Change Preview.

Show:

- subject
- current manager
- target manager
- position context
- reason
- reviewer chain
- requested effective date if available
- actual apply time
- affected access
- affected operational responsibilities
- historical preservation

Intermediate approval must not visually equal final transfer.

OPD-05 handover remains conditional.

--------------------------------------------------
10. TERMINATION
--------------------------------------------------

Termination is NOT a simple “Disable User”.

Design an impact review showing:

- employment state
- access state
- current manager
- open work
- current custody
- future responsibility
- invoice/customer responsibility
- compensation boundary
- next responsible actor
- unresolved handover items

Do not invent automatic reassignment.

If handover is unresolved:
show incomplete/blocked/conditional state.

--------------------------------------------------
11. HR REQUESTS
--------------------------------------------------

Design request lifecycle:

- Requested
- Pending Review
- Intermediate Approval / Rejection
- Pending HR
- Apply
- Applied
- Failed

Show:

- requested change
- actual applied change
- current reviewer
- next reviewer
- history
- reason
- effective/applied time

Do not collapse:
approved step
and
applied change.

--------------------------------------------------
12. STRUCTURE & ACCESS
--------------------------------------------------

Design clear separation between:

- Business Position
- Product Permission
- WP Role / Capability
- Credential
- Direct Override / Inherit

Useful shared pattern:
Access State Matrix / Access Source Inspector

Must communicate:

- source of access
- inherited vs direct
- effective state
- override reason
- sensitive delta

Do not expose unrestricted WP capability editing.

--------------------------------------------------
13. CREDENTIALS / IMPERSONATION
--------------------------------------------------

This area is RESTRICTED.

Design separately from Workforce CRUD.

Credential actions may include:

- reset password
- send credential
- account linkage

Impersonation should clearly show:

- actor
- target
- purpose
- start state
- active session warning
- return action
- audit requirement

Do NOT visually imply impersonation is read-only.

Do not expose password values or secrets.

--------------------------------------------------
14. COMPENSATION
--------------------------------------------------

Design separate Compensation workspace/context.

Show, when authorized:

- current compensation
- effective period
- currency/unit
- history
- commission eligibility/reference
- reason
- before / after

Do not merge with:
- Wallet
- Sales Performance
- Finance posting
- commission engine

Historical earned credit must not look recalculated by current HR edit.

--------------------------------------------------
15. EXCEPTIONS / AUDIT
--------------------------------------------------

Design around supported exception classes:

- invalid hierarchy
- inactive manager with active staff
- failed HR apply
- duplicate workforce profile
- missing user linkage
- role-position mismatch
- termination with open work
- out-of-scope request
- access mismatch
- historical attribution gap

Each item should show:

- subject
- owner
- current state
- impact
- allowed HR action
- restricted action
- resolution proof

No invented SLA.

--------------------------------------------------
16. SENSITIVE ACTION UX
--------------------------------------------------

Sensitive actions should use explicit impact confirmation.

Examples:
- access change
- credential reset
- impersonation
- hierarchy change
- termination
- compensation change

Confirmation should show:
what changes
what does not change
affected entities
effective time
audit reason

Do not use generic “Are you sure?” only.

--------------------------------------------------
17. BULK ACTIONS
--------------------------------------------------

Baseline:

Bulk Workforce Update → conditional
Bulk Transfer → conditional
Bulk Termination → not allowed baseline
Bulk Access Change → needs validation
Bulk Credential Reset → restricted conditional
Bulk Compensation Update → conditional
Bulk Export → conditional

Do not make destructive bulk operations primary.

All results need:
applied
skipped
failed
unknown

--------------------------------------------------
18. STATES
--------------------------------------------------

Required:

Loading
Empty
No Result
Unauthorized
Stale
Incomplete
Conflict
Partial Success
Retryable Failure
Outcome Unknown
Offline

HR-specific:

- request changed before review
- target manager inactive
- duplicate profile
- access apply partial
- termination handover incomplete
- effective date conflict
- historical attribution unavailable
- credential reset success / notification failure

Reuse shared state components.

--------------------------------------------------
19. RESPONSIVE / ACCESSIBILITY
--------------------------------------------------

Validate:

1920×1080
1366×768
1024×768
768×1024
390×844

HR can contain long forms and history.

Use:
- progressive disclosure
- drawers
- step sections
- summary / details
- sticky save/review zones where appropriate

Avoid page-level horizontal scroll.

Preserve:
Light / Dim / Dark
Comfortable / Compact
Focus Mode

Accessibility:
keyboard
focus
semantic forms
aria-invalid
dialog focus
status not color-only
contrast >= 4.5:1
reduced motion

--------------------------------------------------
20. HELP / TOUR
--------------------------------------------------

Reuse Shared Help/Tour.

HR-specific coverage:

1. Workforce
2. Onboarding
3. Transfer
4. Requests
5. Structure & Access
6. Compensation
7. Exceptions
8. Restricted Credentials

Tour must not execute actions.

--------------------------------------------------
21. QA — TOKEN EFFICIENT
--------------------------------------------------

PASS A
1366 + 390
one theme

Check:
layout
long forms
timeline
drawers
navigation
overflow

PASS B
critical flows only:

- workforce profile
- onboarding conflict
- transfer preview
- termination handover
- HR request review
- access-state comparison
- compensation history
- restricted credential/impersonation

PASS C
required widths/themes/density/focus
representative states only.

Capture minimal final screenshots.

--------------------------------------------------
22. ZERO BREAKAGE
--------------------------------------------------

Do not assume:

- HR runtime verified
- OPD-05 solved
- scheduled transfer exists
- access policy finalized
- credential authority finalized
- impersonation is safe/read-only
- compensation locking solved
- failed apply recovery solved

KEEP + ADAPT.

Do not destructively modify prior frozen role designs.

--------------------------------------------------
23. DELIVERABLES
--------------------------------------------------

Keep compact:

1. Current→Target mapping
2. HR IA
3. Workforce Explorer
4. Employment Timeline
5. Onboarding / Transfer
6. Termination / Handover
7. Requests
8. Structure & Access
9. Compensation
10. Credentials / Impersonation
11. Exceptions / Audit
12. Sensitive Action Confirmation
13. Shared Component Delta
14. Help / Tour
15. Final QA evidence
16. Known Limitations
17. Product Gaps / Deferred
18. Freeze Review

--------------------------------------------------
24. FREEZE REVIEW
--------------------------------------------------

Report:

Fixed During QA
Verified
Shared Components Added
HR-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role Consistency
Ready For Next Role?

If no true design blocker remains:

HR DESIGN V1 FROZEN — WORKFORCE & ACCESS CRM DESIGN SYSTEM EXTENDED

Then STOP.