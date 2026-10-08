TASK
SALES DEPUTY REDESIGN

MODE
DESIGN / PROTOTYPE ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

SUCCESS:
SALES DEPUTY DESIGN V1 FROZEN — LEADERSHIP CRM DESIGN SYSTEM EXTENDED

Then STOP.
Do not start MIS.

--------------------------------------------------
1. SOURCES OF TRUTH
--------------------------------------------------

PRODUCT:
- SALES-DEPUTY-PRODUCT-SPEC.md
- GATE-0-PRODUCT-INVARIANTS.md

VISUAL:
- SELLER DESIGN V1 FROZEN
- SUPERVISOR DESIGN V1 FROZEN
- SENIOR SUPERVISOR DESIGN V1 FROZEN
- SALES MANAGER DESIGN V1 FROZEN if available

Product Spec = WHAT / WHY.
Frozen Design System = HOW.

Do not derive business rules from live UI or mock data.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT re-audit the Deputy role.

Codex already verified:
- Deputy #2
- version 2.0.123
- current destinations
- F02/F03
- current invoice readonly behavior
- archive/extra-number absence
- conditional direct allocation
- HR eligibility
- wallet risk

Use live CRM only for targeted visual/content verification.

Do NOT:
- reread unrelated plugin code
- dump DOM
- re-audit lower roles
- make redundant screenshots
- narrate every implementation step

Prefer:
Product Spec → existing components → focused design → focused QA.

--------------------------------------------------
3. CORE DESIGN PRINCIPLE
--------------------------------------------------

Sales Deputy = LEADERSHIP + CROSS-MANAGER COORDINATION

Not:
- Manager with more rows
- executive vanity dashboard
- Finance
- HR admin
- MIS console
- universal queue operator

Deputy should quickly understand:

- which Manager branch needs attention?
- where workload/capacity is uneven?
- which exception lacks resolution?
- which metric is unreliable?
- who owns next action?
- where direct allocation may be conditionally needed?

--------------------------------------------------
4. TARGET IA
--------------------------------------------------

PRIMARY:
- Leadership Overview
- Hierarchy & Performance
- Exceptions
- Reports

CONDITIONAL:
- Direct Allocation
- HR Request Context
- Archive / Extra-number Monitor if authorized

SECONDARY:
- Scoped Invoice Context
- Own Account / Wallet

ADVANCED:
- Metric / Case Lineage Diagnostics

Do not preserve old level-specific tabs as separate primary destinations.

--------------------------------------------------
5. REUSE EXISTING SHARED SYSTEM
--------------------------------------------------

Reuse:

- AppShell
- RoleNavigation
- ScopeBreadcrumb
- PerformanceExplorer
- DimensionSwitcher
- AttributionBasis
- MetricCoverage
- LinkConfidence
- AttentionList
- OperationsExceptionConsole if already added
- DataTable
- Drawer / Sheet
- Reports
- ApprovalChain
- Help / Tour
- CommandPalette
- shared state patterns
- Theme / Density / Focus Mode

Do not duplicate components.

--------------------------------------------------
6. LEADERSHIP OVERVIEW
--------------------------------------------------

This is not an executive KPI wall.

It should answer:

- which Manager branch needs attention?
- which branch has incomplete/stale data?
- where is workload concentrated?
- where is distribution constrained?
- which exception is unresolved?
- who owns the next action?

Avoid:

- ranking
- leaderboard
- performance score
- target
- conversion rate
- fake “best/worst team”

Use data trust indicators.

--------------------------------------------------
7. HIERARCHY & PERFORMANCE
--------------------------------------------------

Expected drill:

Manager
→ Senior
→ Supervisor
→ Seller
→ Case / Invoice

Clearly distinguish:

- Current Hierarchy
- Historical Attribution
- Credit Owner
- Event Actor

Do not let current parent imply historical credit.

Do not double-count ancestor + descendant facts.

Use one hierarchy-aware explorer instead of separate repeated screens.

--------------------------------------------------
8. KPI / METRIC TRUST
--------------------------------------------------

Approved concepts include:

- Invoice Created
- Open Pre-invoices
- Pre-invoices Issued
- Sales Completed
- Total Cases
- Legacy Leads Only

F02 and F03 remain unresolved implementation issues.

F04 is not proven on Deputy today, but the system-wide discrepancy risk remains.

Where trust is incomplete show:

- داده ناقص
- پوشش ناقص
- نیازمند تطبیق
- تعریف نهایی نشده

Never fake 0.

--------------------------------------------------
9. DIRECT ALLOCATION
--------------------------------------------------

This is CONDITIONAL.

Technical recipient levels may include:

- Manager
- Senior
- Supervisor
- Seller

Do not make all four look equally normal.

Primary hierarchy path should be clear.
Bypass / skip-level should look exceptional.

Show:

- current custody
- recipient relationship
- scope
- source
- eligibility
- financial dependency
- preview
- block reason
- apply recheck expectation

Do NOT invent business approval policy.

--------------------------------------------------
10. RETURN
--------------------------------------------------

Reuse safe return patterns.

Deputy cannot visually appear able to recall everything in subtree.

Represent:

- Eligible
- Blocked
- Conflict
- Unknown / Reconciliation Needed

Financial dependency rules from Gate 0 remain binding.

Cancelled/rejected does not auto-release.

--------------------------------------------------
11. EXCEPTIONS
--------------------------------------------------

Design cross-Manager coordination around Product-Spec-supported categories:

- unresolved Manager operational issue
- workload/capacity issue
- distribution conflict
- stale/incomplete metric
- F04 reconciliation need
- inactive hierarchy actor
- consumed-return conflict
- unresolved lineage
- invoice needing correct Sales actor
- HR chain waiting
- organizational scope conflict

Each exception should show:

- subject
- responsible owner
- current state
- why attention is needed
- allowed Deputy action
- next actor
- resolution signal

If no escalation command exists:
do not invent one.

--------------------------------------------------
12. INVOICES
--------------------------------------------------

Deputy baseline is READ-oriented.

Show:

- invoice
- Manager/Senior/Supervisor/Seller context
- Case/source
- invoice lifecycle
- payment stage
- Finance review
- total / paid / remaining / unit
- next actor
- reason
- freshness / completeness

Do not inherit Manager assist/write controls.

No:
- Finance approve/reject
- refund
- posting
- commission controls

--------------------------------------------------
13. ARCHIVE / EXTRA-NUMBER
--------------------------------------------------

Baseline:

Archive aggregates → CONDITIONAL READ
Archive record drill → CONDITIONAL / NEEDS VALIDATION
Archive mutation → NOT IN ROLE

Extra-number backlog → CONDITIONAL READ
Extra-number review → NOT IN ROLE

Do not visually copy Manager queues into Deputy.

If data is not authorized:
show unavailable / unauthorized state,
not fake data.

--------------------------------------------------
14. HR REQUESTS
--------------------------------------------------

Deputy may participate only when actually assigned.

Show:

- request
- subject
- current reviewer
- prior reviewers
- next reviewer
- HR final state
- failed apply

Differentiate clearly:

"تأیید این مرحله"

from:

"اعمال نهایی توسط منابع انسانی"

No HR admin tools.

--------------------------------------------------
15. REPORTS / WALLET
--------------------------------------------------

REPORTS:
broader scoped hierarchy than Manager, but not all-org by default.

Support:

- Manager grouping
- Senior/Supervisor/Seller drill
- source coverage
- time basis
- freshness
- export permission
- sensitive-field restrictions

Keep:
Current Snapshot
Event Range
Historical As-Of
separate.

WALLET:
personal, read-only representation.

--------------------------------------------------
16. LIKELY SHARED EXTENSIONS
--------------------------------------------------

Evaluate only if actually needed:

- BranchCoverage
- LeadershipScopeSummary
- MetricReconciliation
- OwnerAwareException
- CrossManagerPerformance
- ConditionalAllocationContext

Classify:
SHARED CRM
or
DEPUTY-SPECIFIC

Avoid speculative components.

--------------------------------------------------
17. BULK ACTIONS
--------------------------------------------------

Baseline:

Bulk Direct Allocation → supported technically / conditional policy
Bulk Return → conditional
Bulk Export → conditional
Bulk Exception Action → needs validation
Bulk HR Review → not allowed
Bulk Archive Review → not allowed for mutation
Bulk Extra-number Review → not allowed

Do not invent bulk leadership powers.

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

Deputy-specific examples:

- one Manager branch failed
- partial hierarchy
- incomplete report coverage
- recipient changed before apply
- out-of-scope direct allocation
- unknown owner
- metric mismatch

Reuse shared state patterns.

--------------------------------------------------
19. RESPONSIVE / ACCESSIBILITY
--------------------------------------------------

Validate:

1920×1080
1366×768
1024×768
768×1024
390×844

Preserve:

Light
Dim
Dark
Comfortable
Compact
Focus Mode

Avoid page-level horizontal scroll.

Accessibility:
keyboard
focus
semantic navigation
table/grid semantics
drawer focus
status not color-only
contrast >= 4.5:1
reduced motion

--------------------------------------------------
20. QA — TOKEN EFFICIENT
--------------------------------------------------

PASS A
1366 + 390
one theme

Check:
layout
nav
hierarchy drill
runtime
overflow

PASS B
critical flows only:

- branch drilldown
- metric trust
- direct allocation preview
- blocked return
- exception coordination
- invoice context
- HR assigned step
- report scope

PASS C
final widths/themes/density/Focus Mode
representative states only.

Capture minimal final screenshots.

--------------------------------------------------
21. ZERO BREAKAGE
--------------------------------------------------

Do not assume:

- F01 fixed
- F02 fixed
- F03 fixed
- F04 fixed
- direct-allocation policy finalized
- archive visibility deployed
- extra-number visibility deployed
- HR entry deployed
- DB migration
- permission rollout

KEEP + ADAPT.

Do not destructively modify prior frozen role designs.

--------------------------------------------------
22. DELIVERABLES
--------------------------------------------------

Keep compact:

1. Current→Target mapping
2. Deputy IA
3. Leadership Overview
4. Hierarchy & Performance
5. Exceptions
6. Conditional Direct Allocation
7. Scoped Invoice Context
8. HR Context
9. Reports / Wallet
10. Shared Component Delta
11. System States
12. Help / Tour
13. Final QA evidence
14. Known Limitations
15. Product Gaps / Deferred
16. Freeze Review

--------------------------------------------------
23. FREEZE REVIEW
--------------------------------------------------

Report:

Fixed During QA
Verified
Shared Components Added
Deputy-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role Consistency
Ready For Next Role?

If no true design blocker remains:

SALES DEPUTY DESIGN V1 FROZEN — LEADERSHIP CRM DESIGN SYSTEM EXTENDED

Then STOP.