TASK
CRM CROSS-ROLE DESIGN SYSTEM INTEGRATION

MODE
DESIGN SYSTEM / UX INTEGRATION ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

INPUTS:

- CRM-CROSS-ROLE-ARCHITECTURE-V1.md
- GATE-0-PRODUCT-INVARIANTS.md
- all FROZEN role designs:
  Seller
  Supervisor
  Senior Supervisor
  Sales Manager
  Sales Deputy
  MIS
  HR
  Finance

GOAL:

Turn all frozen role designs into ONE coherent CRM Design System.

Do NOT redesign the roles from scratch.

--------------------------------------------------
1. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do not re-audit business logic.

Do not revisit already-frozen role decisions unless:
- there is a true cross-role visual contradiction,
- a shared component has diverged,
- or an accessibility/responsive issue blocks reuse.

Use the Cross-Role Architecture as authority for:
- terminology
- shared states
- ownership concepts
- permission semantics
- request model
- temporal model
- financial vocabulary

Avoid redundant screenshots.

--------------------------------------------------
2. PRIMARY GOAL
--------------------------------------------------

Verify all 8 role designs feel like ONE product.

Check consistency of:

- AppShell
- Header
- RoleNavigation
- PageHeader
- Search
- Filters
- DataTable
- Drawer / Sheet
- Status
- Metric metadata
- Scope
- Ownership
- Timeline
- Request / Review
- Bulk actions
- Sensitive actions
- Help / Tour
- Theme
- Density
- Focus Mode
- Responsive behavior
- Accessibility

--------------------------------------------------
3. GLOBAL TERMINOLOGY
--------------------------------------------------

Standardize the design language for:

Customer
Case
Source
Current Custody
Original Owner
Next Actor
Event Actor
Credit Owner
Invoice
Payment Stage
Finance Review
Refund
Ledger
Request
Reviewer
Applied
Outcome Unknown
Incomplete
Conflict

Do not rename stored backend statuses.

This is presentation terminology only.

--------------------------------------------------
4. GLOBAL STATE SYSTEM
--------------------------------------------------

Standardize presentation for:

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

Also standardize:

Known Failed
Blocked
Conditional
Restricted
Needs Validation

Do not make every role invent its own visual pattern.

--------------------------------------------------
5. GLOBAL OWNERSHIP CONTEXT
--------------------------------------------------

Create/reconcile shared design patterns for:

Current Custody
Original Owner
Next Actor
Event Actor
Credit Owner
Source Owner
Reviewer

The component must allow roles to see only relevant fields.

Do not collapse them into “Owner”.

--------------------------------------------------
6. GLOBAL IDENTITY / LINEAGE CONTEXT
--------------------------------------------------

Standardize:

CaseRef
SourceRef
Native ID
Alias
Verified linkage
Incomplete linkage
Conflict
Unknown

Phone must not look canonical.

Potential shared patterns:

IdentityContext
SourceContext
LineageConfidence

Only keep components that have real multi-role reuse.

--------------------------------------------------
7. REQUEST / REVIEW PATTERN
--------------------------------------------------

Create shared visual grammar for:

Requester
Subject
Current Reviewer
State
Reason
History
Result
Next Actor

But allow domain-specific:

HR
Extra-number
Finance Review
Reconciliation

Do not force one interaction flow across all domains.

--------------------------------------------------
8. SENSITIVE ACTION PATTERN
--------------------------------------------------

Standardize impact review for:

Financial action
HR change
Credential action
Maintenance
Destructive source action
Bulk high-impact action

Shared confirmation should support:

Target
Current state
Proposed change
Impact
Dependencies
Reason
Expected next state
Audit context

Avoid generic “Are you sure?”

--------------------------------------------------
9. BULK ACTION PATTERN
--------------------------------------------------

Standardize:

Requested
Eligible
Applied
Skipped
Rejected
Failed
Unknown
Unprocessed

Reuse one shared bulk outcome language.

Bulk UI must not imply permissions.

--------------------------------------------------
10. TEMPORAL UX
--------------------------------------------------

Standardize time concepts:

Current
Historical
Effective From/To
Applied At
Event Time
Posting Time
Gather Time

Create common visual language for:
Current vs Historical
Requested vs Applied

Do not imply Historical As-Of when data does not support it.

--------------------------------------------------
11. REPORT / KPI TRUST UX
--------------------------------------------------

Standardize:

grain
scope
source
cohort
time basis
unit
freshness
coverage
version
reconciliation state

Shared MetricMeta / MetricCoverage / Freshness patterns should work across roles.

Do not fake zero for unavailable data.

--------------------------------------------------
12. NAVIGATION SYSTEM
--------------------------------------------------

Review role navigation together.

Do not force identical nav depth.

Allowed:
- Seller simpler top tabs
- Supervisor/Senior grouped role nav
- Manager/Deputy broader grouped nav
- MIS/HR/Finance domain-based nav

But interaction language should remain consistent.

--------------------------------------------------
13. COMPONENT INVENTORY
--------------------------------------------------

Create a canonical component registry.

For each component classify:

GLOBAL SHARED
DOMAIN SHARED
ROLE-SPECIFIC
DEPRECATED / DUPLICATE
CONDITIONAL

Expected examples:

AppShell
RoleNavigation
ScopeBreadcrumb
MetricMeta
MetricCoverage
FreshnessIndicator
IdentityContext
OwnershipContext
PerformanceExplorer
RequestReview
ApprovalChain
BulkActionBar
BulkResult
SensitiveActionConfirmation
OutcomeUnknownContext
Timeline
AuditContext
ReportCatalog

Do not create speculative components.

--------------------------------------------------
14. DESIGN TOKEN REVIEW
--------------------------------------------------

Verify all roles use the same:

Typography
Spacing
Radius
Borders
Elevation
Semantic colors
Status colors
Focus styles
Density
Motion

Do not introduce new token families without proven need.

Preserve:
Light / Dim / Dark
Comfortable / Compact
Focus Mode

--------------------------------------------------
15. RESPONSIVE SYSTEM
--------------------------------------------------

Review shared behavior at:

1920
1366
1024
768
390

Define global rules for:

table → priority columns/cards
drawer → sheet
navigation collapse
sticky actions
dense analytics
long forms
hierarchy
financial tables

Avoid page-level horizontal scrolling.

--------------------------------------------------
16. ACCESSIBILITY
--------------------------------------------------

Run cross-role accessibility consistency review:

keyboard
focus
semantic nav
table/grid semantics
dialogs
drawers
forms
ARIA
status not color-only
contrast >=4.5:1
reduced motion
hit targets

Fix shared problems once, not role by role.

--------------------------------------------------
17. HELP / TOUR
--------------------------------------------------

Keep one shared Help architecture.

Standardize:

Help Center
Tour registry
Shortcut help
Glossary
Context Help

Role content stays role-specific.

--------------------------------------------------
18. CROSS-ROLE DESIGN CONFLICT REVIEW
--------------------------------------------------

Find:

duplicate components
same concept with different visuals
different words for same concept
same color meaning different things
inconsistent bulk outcomes
different permission presentation
different stale/incomplete handling
navigation language divergence
drawer/form/table inconsistencies

Classify:

KEEP
MERGE
STANDARDIZE
ROLE-SPECIFIC
DEFERRED

--------------------------------------------------
19. OUTPUT
--------------------------------------------------

Create:

CRM-DESIGN-SYSTEM-V1.md

and update prototype shared component registry.

Keep documentation concise.

Suggested sections:

1. Design Principles
2. Navigation
3. Tokens
4. Typography
5. Layout
6. Identity / Ownership
7. Status / System States
8. Tables / Lists
9. Drawers / Sheets
10. Forms
11. Request / Review
12. Bulk Actions
13. Sensitive Actions
14. Temporal UI
15. KPI / Report Trust
16. Help
17. Accessibility
18. Responsive Rules
19. Component Registry
20. Role-specific Exceptions
21. Deferred Design Items
22. Cross-role QA

--------------------------------------------------
20. QA STRATEGY
--------------------------------------------------

Do not run every role × every viewport × every state blindly.

PASS A
shared components and shell

PASS B
representative role families:
- Seller/Supervisor
- Senior/Manager/Deputy
- MIS
- HR
- Finance

PASS C
final regression matrix only after shared fixes.

Capture minimal representative screenshots.

--------------------------------------------------
21. FREEZE CONDITION
--------------------------------------------------

If cross-role visual language is coherent:

CRM DESIGN SYSTEM V1 FROZEN
ALL 8 ROLE DESIGNS INTEGRATED
READY FOR IMPLEMENTATION FOUNDATION

Then STOP.

Do not implement plugin changes.