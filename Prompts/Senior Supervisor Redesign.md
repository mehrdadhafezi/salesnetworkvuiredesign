We are now starting:

SENIOR SUPERVISOR REDESIGN — MULTI-TEAM MANAGEMENT EXTENSION

This is NOT a greenfield redesign.

You are extending the already frozen CRM Design System.

Authoritative visual foundations:

1. SELLER DESIGN V1 FROZEN
2. SUPERVISOR DESIGN V1 FROZEN

Authoritative product foundations:

3. SENIOR SUPERVISOR PRODUCT SPEC FROZEN
4. GATE 0 PRODUCT INVARIANTS

Your job is DESIGN ONLY.

Do not modify:
- plugin code
- live CRM
- backend
- database
- permissions
- business logic
- role hierarchy
- statuses
- KPI formulas
- Gate 0 contracts

Do not invent product behavior.

---

# CORE DESIGN PRINCIPLE

Senior Supervisor is:

MULTI-TEAM COORDINATION

not:

SUPERVISOR WITH MORE ROWS

The design must emphasize:

- several Supervisor-led teams
- team responsibility
- capacity / workload comparison
- scope-aware drilldown
- multi-team exceptions
- read-only Ready Conversion supervision
- scoped distribution
- invoice exception monitoring
- HR step-review context
- reports

Do not create a generic executive dashboard.

---

# FIRST STEP — READ-ONLY LIVE AUDIT

Open:

https://crm.maximumclub.ir/

Use the existing role switching path to enter the real Senior Supervisor account.

Audit read-only.

Do not execute:

- distribution
- return
- receipt submission
- invoice mutation
- payment action
- HR review
- export
- personal conversion action

Verify:

- current nav
- overview
- team structure
- distribution
- seller performance
- invoices
- ready conversion
- converter stats
- personal conversions
- HR requests
- legacy leads
- reports
- wallet presence

Do not use current UI as visual target.

---

# SECOND STEP — CURRENT → TARGET MAPPING

Before designing create:

Current Section
→ Product Purpose
→ KEEP / MOVE / MERGE / CONDITIONAL
→ Target Destination
→ Reused Component
→ New Shared / Senior-specific Component

Do not design until this mapping is complete.

---

# TARGET SENIOR IA

Use the frozen Product Spec:

Primary:
- Multi-Team Overview
- Structure & Performance Explorer
- Distribution & Eligible Return
- Ready Conversion — Team View
- Invoices & Exceptions
- HR Requests
- Shared Reports

Secondary:
- Case Explorer & Source History
- Own Account / Wallet

Conditional:
- My Assigned Conversions

Advanced / Diagnostic:
- metric/source/link discrepancy detail

Do not mechanically turn each item into a top-level tab.

Use the shared role navigation language already extended in Supervisor.

---

# REUSE EXISTING SHARED SYSTEM

Reuse Seller + Supervisor design foundations:

- AppShell
- Header
- RoleNavigation
- PageHeader
- StatStrip
- KPI / MetricMeta
- FreshnessIndicator
- ScopeBadge
- Toolbar
- Search
- FilterChip
- DataTable
- SelectableDataTable
- BulkActionBar
- BulkReview
- BulkResult
- Pagination
- StatusPill
- Drawer / Sheet
- Timeline
- EligibilityPanel
- OwnershipGrid
- FinancialFacets
- ApprovalChain
- AttentionList
- ReportCatalog
- OperationLog
- Toast
- Tooltip
- CommandPalette
- Help Center
- Tour Engine
- Empty/Error/Loading/Stale/Conflict/Partial states
- Theme
- Density
- Focus Mode

Do not fork the visual language.

---

# MULTI-TEAM OVERVIEW

This is one of the most important Senior screens.

It must answer:

- Which teams are under scope?
- Who is the responsible Supervisor?
- Where is workload concentrated?
- Which team needs attention?
- Which metrics are stale/incomplete?
- Which invoice / ready / HR exception needs coordination?
- What is the next responsible actor?

Avoid:
- leaderboard
- sales pressure
- ranks
- targets
- gamification

Operational coordination > competition.

---

# STRUCTURE & PERFORMANCE EXPLORER

This is likely the most important new pattern for Senior.

It should support multiple modes without mixing their data grains:

- Team
- Supervisor
- Seller
- Conversion
- Invoice
- Case workload

Do not create several disconnected tables if one structured Explorer can handle the modes.

Possible shared pattern:

PerformanceExplorer
+
DimensionSwitcher
+
ScopeBreadcrumb
+
MetricDefinition
+
Freshness / Coverage
+
Drilldown

But do not force this exact implementation if a better design fits the frozen system.

Important:
- current hierarchy
- historical attribution
must be visually distinguishable.

Do not let current manager relationships imply historical credit.

---

# DIRECT vs INDIRECT

Senior must clearly understand:

- direct Supervisors
- direct Sellers if any
- indirect Sellers

Visibility does NOT mean write authority.

Use a visual relationship hierarchy that makes this clear.

Do not use hierarchy depth alone to infer available actions.

---

# PERFORMANCE METRICS

Only use approved metric contracts.

Examples with approved semantics:

- Invoice Created
- Open Pre-invoices
- Pre-invoices Issued
- Sales Completed
- Verified Collected Amount
- Total Cases
- Legacy Leads Only
- Team Workload
- Direct Supervisors
- Active / All Sellers

Do not use:
- Conversion Rate
- Overdue %
- Performance Score
- Targets
- Ranking

unless Product Spec marks them final.

Where metric definition is incomplete:
display:
- تعریف نشده
- داده ناکافی
- پوشش ناقص
- نیازمند تطبیق

Never fake 0.

---

# METRIC TRUST UX

Senior needs to know whether a number is trustworthy.

Extend shared MetricMeta / Freshness patterns to support:

- source coverage
- current vs historical basis
- freshness
- incomplete dataset
- reconciliation discrepancy

Do not expose raw technical internals by default.

Use operator-readable explanations.

---

# DISTRIBUTION & CAPACITY

Senior distribution differs from Supervisor.

Primary route:
Senior → Supervisor

Possible direct-to-Seller path:
CONDITIONAL / NEEDS POLICY

Do not present skip-level assignment as normal/default.

If shown, it must clearly communicate:
- exceptional path
- bypassed level
- reason
- target relationship
- current custody
- eligibility
- dependency conflicts

Do not invent approval policy.

---

# SKIP-LEVEL UX

If current functionality exists but business policy is unresolved:

show it as:
Conditional / Exception Path

not:
normal primary CTA

Do not silently remove it.
Do not normalize it.

Use language that indicates exceptional routing.

---

# RETURN

Reuse the safe return design from Supervisor,
but adjust it to Senior responsibility.

Senior can only initiate return where:
- current policy permits
- transfer proof is valid
- scope is valid
- no protected dependency exists

Do not imply:
“Senior can recall everything below them.”

Blocked reasons must include product-readable cases such as:
- financial dependency
- current custody changed
- source ownership mismatch
- unresolved linkage
- protected downstream operation

Cancelled-case release remains Deferred.

---

# READY CONVERSION — READONLY SUPERVISION

This is a major difference from Supervisor.

Senior should see:
- team
- responsible Supervisor
- seller
- current owner
- next actor
- conversion status
- source
- payment context if allowed
- freshness / waiting context if defined

But Senior must NOT get a direct assignment action in the baseline.

This view is supervisory.

If a formal escalation command does not exist:
do not invent one.

You may show:
- responsible actor
- escalation needed
- unresolved owner

but no new business action.

---

# PERSONAL CONVERSIONS

Conditional module.

Show only if:
- Senior can actually receive personal assignment
or
- current user already has assigned personal cases

Do not keep a permanent empty destination.

Do not infer receiving rights from nav presence.

---

# INVOICES & EXCEPTIONS

Senior needs more context than Supervisor but not Finance authority.

Show:
- Invoice
- Seller
- Supervisor / Team
- Case source
- Invoice Status
- Payment Stage
- Paid
- Remaining
- Finance Review
- Next Actor
- Rejection reason
- linkage / data confidence if necessary

Actions must be capability-aware.

Classify visually:
- Read
- Assist
- Write
- Restricted

Do NOT show Finance approve / reject / refund / post actions.

If an assist action exists but is policy-dependent:
show it only when allowed by fixture/permission state.

---

# HR REQUESTS

Senior can participate in request chain,
but is not HR final authority.

Clearly distinguish:

"تأیید این مرحله"

from:

"اعمال نهایی توسط منابع انسانی"

Show:
- subject
- request type
- reason
- current reviewer
- previous steps
- next reviewer
- final HR pending state
- failed apply state

Do not visually imply that Senior changes employment directly.

---

# CASE EXPLORER

Legacy lead page should not remain conceptually misleading.

Design toward:
Case Explorer

with:
- Source Badge
- Legacy facet
- Source history
- CaseRef context
- linkage confidence
- explicit incomplete coverage

Do NOT imply that unified Case adapter already exists.

If mock data uses unified cases,
label demo clearly.

Legacy-only data must remain distinguishable.

---

# REPORTS

Reuse Shared Reports.

Senior enters Reports with:
- multi-team scope
- team subset
- person subset
- metric definition
- time basis
- source coverage
- freshness
- export permission

Do not allow a dropdown to visually imply access outside Senior scope.

Current Snapshot
Event Range
Historical As-Of

must not be mixed.

---

# OWN WALLET

Personal only.

Reuse shared wallet visual system.

Do not mix:
- team sales
- team commission
- organizational revenue
with:
Senior own wallet.

Opening wallet must be represented as read-only in the prototype.

Do not simulate automatic posting.

---

# EXCEPTION COORDINATION

Senior should see exceptions across teams.

Possible categories supported by Product Spec:

- workload variation
- assignment conflict
- stale / incomplete metrics
- consumed-return conflict
- invoice rejected / pending
- ready waiting for Supervisor
- inactive recipient
- unresolved source / linkage
- HR request waiting in chain

For every exception display:

- Subject
- Responsible Actor
- Current State
- Why it needs attention
- Allowed next action
- Escalation target if defined

If escalation command is not defined:
show informational responsibility only.

Do not invent “ارجاع” behavior.

---

# SHARED COMPONENT EXTENSIONS

Potential new shared components:

## PerformanceExplorer
Likely reusable later by Manager / Deputy.

## DimensionSwitcher
Team / Supervisor / Seller / Conversion.

## ScopeBreadcrumb
Shows:
current role scope
selected team
selected supervisor
selected seller

## AttributionBasis
Shows:
Current Hierarchy
Historical Attribution
Event Actor

## MetricCoverage
Displays:
Complete
Incomplete
Stale
Reconciliation Needed

## LinkConfidence
Displays:
Verified
Legacy Fallback
Unknown

These are conceptual candidates.

Classify final implementation as:
SHARED CRM COMPONENT
or
SENIOR-SPECIFIC COMPONENT

Do not add components without a real role need.

---

# BULK ACTIONS

Do not copy Supervisor bulk actions blindly.

Supported / conditional baseline:

- Bulk Assignment: supported in distribution
- Bulk Return: conditional eligibility
- Bulk Export: conditional
- Bulk Escalation: NOT VERIFIED
- Bulk HR Review: NOT ALLOWED baseline

Design only supported / conditional states.

Do not create UI for unsupported business actions.

---

# STATES

Every main Senior page must support:

- Loading
- Empty
- No Result
- Unauthorized
- Stale
- Incomplete
- Conflict
- Partial Success
- Retryable Failure
- Outcome Unknown
- Offline

For multi-team views also support:

- partial source coverage
- one team failed, others loaded
- stale team metric
- unknown historical attribution

Do not collapse all of these into generic error.

---

# RESPONSIVE

Validate at:

1920×1080
1366×768
1024×768
768×1024
390×844

Senior has denser multi-team content.

Do not solve density only with horizontal scrolling.

Use:
- priority columns
- grouped cards
- drilldown
- drawer
- expandable hierarchy
- summary modes

where appropriate.

---

# ACCESSIBILITY

Review:

- keyboard
- focus
- semantic navigation
- table/grid semantics
- aria-expanded
- aria-selected
- aria-invalid
- drawer/dialog focus
- readable hierarchy relationships
- status not color-only
- 4.5:1 contrast
- reduced motion

---

# OPERATOR COMFORT

Senior may use the system for long periods.

Avoid:
- huge dashboard cards
- bright KPI walls
- excessive color
- unnecessary charts
- leaderboard visuals
- decorative animation

Preserve:
- Light
- Dim
- Dark
- Comfortable
- Compact
- Focus Mode

---

# HELP / TOUR

Reuse Shared Help Engine.

Create Senior-specific registry.

Potential tour:

1. Multi-Team Overview
2. Structure / Scope
3. Performance Explorer
4. Distribution
5. Ready Team View
6. Invoice Exceptions
7. HR Requests
8. Reports
9. Help

Tour must not perform actions.

---

# PRODUCT GAPS

Respect:

SG-01…SG-08
SD-01…SD-05
Gate 0 Deferred decisions

Do not implement them.

When needed in design:

PRODUCT GAP — DO NOT IMPLEMENT

or

DEFERRED — CONDITIONAL

---

# ZERO BREAKAGE DESIGN ASSUMPTION

Do not assume:
- database migration
- renamed statuses
- removed legacy routes
- rewritten IDs
- merged historical records
- new permission model already deployed

The design must remain compatible with:

KEEP + ADAPT

The prototype may visually represent the Target Contract,
but must not imply underlying migration already happened.

---

# DELIVERABLES

Create the Senior prototype in the existing design workspace.

Do not destructively modify:

Seller Frozen
Supervisor Frozen

Reuse shared assets.

Deliver:

1. Current → Target mapping
2. Senior IA
3. Desktop prototype
4. Tablet/mobile
5. Light/Dim/Dark
6. Comfortable/Compact
7. Focus Mode
8. Performance Explorer
9. Distribution / Return flow
10. Ready readonly flow
11. Invoice exception flow
12. HR request flow
13. Reports
14. Conditional Personal Conversion
15. Help / Tour
16. All system states
17. QA screenshots
18. Design QA report
19. Shared Design System delta
20. Known limitations
21. Product gaps
22. Deferred items

---

# DESIGN FREEZE REVIEW

After implementation run full QA.

Report:

Fixed During QA
Verified
Shared Components Added
Senior-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role consistency
Ready For Next Role?

Do NOT freeze if a real design blocker remains.

---

# SUCCESS CONDITION

If successful:

SENIOR SUPERVISOR DESIGN V1 FROZEN — MULTI-TEAM CRM DESIGN SYSTEM EXTENDED

This means only visual/product presentation is frozen.

It does NOT mean:
- backend changed
- database changed
- Gate 0 implemented
- permissions rolled out
- metrics repaired in production

Stop after Freeze Review.

Do not begin Sales Manager.