# SALES MANAGER REDESIGN — TOKEN / CONTEXT EFFICIENT

MODE:
DESIGN SYSTEM / UX / PROTOTYPE ONLY

DO NOT implement:
- backend logic
- WordPress/PHP business behavior
- database changes
- migrations
- permissions
- capabilities
- role mappings
- financial logic
- workflow/status changes

==================================================
EFFICIENCY CONTRACT
==================================================

Use frozen artifacts first.

Do NOT restart a broad CRM audit.

Do NOT:
- reread unrelated plugin files
- dump large source files or DOM
- repeat previously verified evidence
- reopen frozen Seller/Supervisor/Senior decisions without a true blocker
- generate huge screenshot matrices before structural QA is complete
- narrate every small step

Use targeted code/evidence lookup only when:
- the Product Spec references behavior that is ambiguous,
- frozen artifacts contradict each other,
- or implementation reality is necessary to avoid designing an impossible interaction.

Verification quality must NOT be reduced to save tokens.

==================================================
AUTHORITATIVE INPUTS
==================================================

Read first:

1. AGENTS.md
2. README.md
3. docs/GITHUB_HANDOFF.md
4. docs/source-of-truth/SALES-MANAGER-PRODUCT-SPEC.md
5. docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md
6. docs/source-of-truth/GATE-0-PRODUCT-INVARIANTS.md

Then inspect only relevant existing prototype/shared design files.

Frozen previous role designs are visual/design-system inputs.

Product Specs and frozen architecture override prototype assumptions.

==================================================
ROLE CONTRACT
==================================================

Sales Manager is NOT:

- Senior Supervisor with more rows
- Finance
- MIS
- HR
- unrestricted CRM admin
- a replacement for operational Supervisors

Sales Manager owns a management/coordination layer centered on:

- Sales Operations overview
- Exceptions requiring Manager attention
- Distribution / allocation oversight
- Team and hierarchy performance
- invoice visibility appropriate to the frozen Product Spec
- Archive Explorer
- Extra-number Requests
- HR Requests at the Manager's legitimate workflow position
- Shared Reports
- Wallet where supported
- contextual Customer / Case exploration

Preserve all conditional/deferred behavior from the frozen Product Spec.

Do NOT turn Product Gaps into capabilities.

==================================================
LATEST CROSS-ROLE ARCHITECTURE ADDENDUM
==================================================

Use CRM-CROSS-ROLE-ARCHITECTURE-V1 only for shared/global contracts:

- terminology
- identity / lineage
- ownership dimensions
- handoffs
- permission semantics
- KPI/report metadata
- system result states
- bulk outcome language
- temporal concepts
- audit/history concepts
- shared component boundaries

Priority:

1. SALES-MANAGER-PRODUCT-SPEC.md = Manager-specific WHAT / WHY
2. CRM-CROSS-ROLE-ARCHITECTURE-V1.md = shared/global contract
3. GATE-0-PRODUCT-INVARIANTS.md = binding invariants
4. Frozen prior role designs = visual/design foundation

If an older prototype conflicts with the frozen Cross-Role contract,
standardize presentation without inventing new behavior.

Hierarchy does NOT automatically grant permission.

View does NOT mean Write.

==================================================
DESIGN SYSTEM CONTRACT
==================================================

Reuse and extend the frozen CRM Design System.

Preserve:

- Shared Shell
- compact header
- RoleNavigation language
- Light / Dim / Dark
- Comfortable / Compact density
- Focus Mode
- Command Palette
- keyboard accessibility
- RTL behavior
- responsive/mobile patterns
- Drawer / Sheet language
- table language
- state banners
- loading/error/empty/no-result/offline states
- toast/tooltip/help/tour system
- muted status color + icon + text
- reduced motion
- accessible contrast

Reuse shared components before creating Manager-specific ones.

A component MAY create a reusable presentation pattern.

A component MUST NOT create a new business capability.

==================================================
MANAGER UX GOAL
==================================================

Design the Sales Manager workspace around:

1. What requires management attention now?
2. Which teams / Supervisors are healthy or at risk?
3. Where are allocation/distribution exceptions?
4. What requires Manager decision vs simple observation?
5. Which metrics are trustworthy, scoped, and fresh?
6. What can the Manager act on without crossing into Finance/MIS/HR authority?

Avoid vanity dashboards.

Prioritize:
- attention
- exception handling
- coordination
- scoped decision support
- traceability

==================================================
TARGET INFORMATION ARCHITECTURE
==================================================

Use the frozen Product Spec to finalize exact labels and visibility.

Expected Manager domains include:

- Overview / Attention
- Sales Operations
- Exceptions
- Distribution / Allocation
- Team & Hierarchy Performance
- Invoices / Financial visibility
- Archive Explorer
- Extra-number Requests
- HR Requests
- Reports
- Wallet
- contextual Customer / Case Explorer

Do NOT expose an area merely because a lower/higher role has it.

==================================================
KPI / REPORT TRUST
==================================================

Every important metric presentation must support the shared metric trust model:

- scope
- cohort
- time basis
- freshness
- coverage
- unit / formula meaning where needed

Do not display unknown/incomplete data as zero.

Differentiate:

- Current Snapshot
- Event Range
- Historical As-Of

When data is incomplete/stale/ambiguous, design the truthful state.

==================================================
OWNERSHIP / CASE LANGUAGE
==================================================

Never visually collapse:

- Current Custody
- Original Owner
- Next Actor
- Event Actor
- Credit Owner

Phone is discovery/search information only.

Never present phone as canonical Case identity.

Conflicting lineage must remain:

UNKNOWN / CONFLICT

where applicable.

==================================================
SENSITIVE ACTIONS
==================================================

For any legitimate Manager action:

- make action authority clear
- show scope
- show affected records
- show eligibility
- require confirmation where risk warrants it
- show partial/failed/unknown outcomes truthfully

Bulk actions must support:

requested
eligible
applied
skipped
rejected
failed
unknown

Do not imply all-or-nothing success when backend behavior may be per-item.

==================================================
PROTOTYPE WORK
==================================================

Extend the existing prototype architecture.

Do NOT create a separate visual product.

Prefer:

prototype/
  shared/
  sales-manager/

Use current shared assets/components where possible.

Manager-specific files may be introduced only when justified.

Do not destructively rewrite frozen Seller/Supervisor designs.

==================================================
RESPONSIVE EXPECTATIONS
==================================================

Validate Manager flows at minimum against representative:

- 1920 desktop
- 1366 desktop
- 1024
- 768 tablet
- 390 mobile

Use focused QA.

Do not produce repetitive screenshots for unchanged states.

Check:
- Light / Dim / Dark
- Comfortable / Compact where meaningful
- keyboard/focus behavior
- responsive action hierarchy
- tables → cards/sheets where appropriate
- drawers/sheets
- long Persian text
- empty/error/stale/incomplete states

==================================================
IMPORTANT PRODUCT GAPS
==================================================

Preserve unresolved Manager Product Gaps as unresolved.

Do not design them as confirmed capabilities.

In particular:

- report/invoice parity issues remain product/implementation concerns
- customer behavior loading concerns must not be hidden by mock data
- legacy archive semantics must be preserved
- extra-number approval side effects must not be silently normalized
- Manager does not inherit Finance/HR/MIS powers

If a design depends on a deferred decision:
show it as conditional / blocked / pending policy.

==================================================
DELIVERABLE
==================================================

Produce:

1. Sales Manager prototype/redesign
2. Manager design documentation
3. component reuse/additions summary
4. focused QA evidence
5. unresolved product/design blockers

Freeze the result as:

SALES MANAGER DESIGN V1 FROZEN

Do not reopen the Product Spec.

==================================================
GIT / PR CONTRACT
==================================================

Before editing:

- confirm current branch is NOT main
- use branch:

  claude/sales-manager-redesign

At completion:

1. run relevant focused QA
2. review git diff
3. ensure only this task's files changed
4. commit the work
5. push the branch
6. open a Pull Request into main

PR description must include:

- Summary
- Files changed
- Shared components reused
- Shared components added/changed
- Manager-specific components
- QA performed
- Known limitations
- Deferred Product Gaps

DO NOT merge the PR.

Return the PR URL.

Then print:

SALES MANAGER DESIGN V1 FROZEN
READY FOR REVIEW

Then STOP.