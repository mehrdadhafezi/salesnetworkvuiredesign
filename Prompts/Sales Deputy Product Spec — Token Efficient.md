TASK
SALES DEPUTY PRODUCT SPEC

MODE
PRODUCT / WHAT-WHY ONLY.

NO UI DESIGN.
NO CODE / PLUGIN / DATABASE / PERMISSION / LIVE MUTATION.

OUTPUT:
SALES-DEPUTY-PRODUCT-SPEC.md

SUCCESS:

SALES DEPUTY PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN
NO CODE CHANGED.

If blocked:
SALES DEPUTY DESIGN BLOCKED BY: <ID>

--------------------------------------------------
1. AUTHORITATIVE SOURCES
--------------------------------------------------

Use existing accepted project evidence first:

- CRM PRODUCT & ROLE ARCHITECTURE AUDIT
- GATE-0-PRODUCT-INVARIANTS.md
- SENIOR-SUPERVISOR-PRODUCT-SPEC.md
- SALES-MANAGER-PRODUCT-SPEC.md
- existing Code Evidence / Status / Decisions docs

Claude prototypes are visual references only.

Do not derive permissions, KPI formulas or workflows from prototypes.

Gate0 INV-001…INV-036 remain binding.

KEEP + ADAPT is default.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT restart the full CRM audit.

Reuse accepted evidence.

Live verification only for:
- current Deputy inventory
- Deputy-specific permissions/behavior
- stale/contradictory evidence
- role-boundary questions that cannot be resolved from existing docs

Do NOT:
- re-audit lower roles
- dump full DOM
- reread unrelated plugin files
- repeatedly inspect the same page
- capture unnecessary screenshots
- produce long progress narration

Use targeted code inspection only for exact Deputy guards/predicates.

--------------------------------------------------
3. LIVE VERIFICATION
--------------------------------------------------

Use:
https://crm.maximumclub.ir/

Enter the real Sales Deputy account identified by the Audit.

READ ONLY.

Never execute:
distribution,
assignment,
return,
invoice mutation,
payment action,
HR decision,
archive decision,
export,
or any other write.

Verify current version once.

Mark:
LIVE VERIFIED
CODE VERIFIED
BOTH
INFERENCE
NOT VERIFIED

UI presence != successful write authority.

--------------------------------------------------
4. ROLE MISSION
--------------------------------------------------

Define:

Role Mission
Daily Jobs
Occasional Jobs
Decision Responsibilities
Data Needed
Data Not Needed
Allowed Responsibilities
Explicit Non-Responsibilities

Accepted direction:

Deputy = leadership / cross-Manager coordination / hierarchy and performance oversight / exception management.

Deputy is NOT automatically:

- Manager operator
- Finance
- HR final authority
- MIS
- Admin
- Seller operator

--------------------------------------------------
5. DEPUTY vs SALES MANAGER
--------------------------------------------------

Mandatory concise comparison:

Capability
Sales Manager
Sales Deputy
Why Different?

Cover:

- organizational scope
- hierarchy level
- distribution
- direct allocation
- archive responsibility
- extra-number requests
- HR requests
- invoice visibility
- performance
- exceptions
- reports
- wallet
- bulk actions
- operational authority

Do not turn Deputy into “Manager with more rows”.

--------------------------------------------------
6. DEPUTY vs MIS / EXECUTIVE MANAGEMENT
--------------------------------------------------

Clarify boundaries.

Deputy may need broad visibility,
but must not absorb:

- raw data repair
- import ownership
- report engineering
- Finance approval
- HR administration
- system administration

Also distinguish:
leadership oversight
from
hands-on Manager queue operation.

--------------------------------------------------
7. TARGET IA
--------------------------------------------------

Accepted audit direction:

- Leadership Overview
- Hierarchy & Performance
- Exceptions
- Reports
- Direct Allocation — conditional
- Own Account / Wallet

Validate against live/code evidence.

Define:

PRIMARY
SECONDARY
CONDITIONAL
ADVANCED / DIAGNOSTIC

Do not design visually.

--------------------------------------------------
8. LEADERSHIP OVERVIEW
--------------------------------------------------

Deputy overview should answer:

- which Managers / territories need attention?
- where is capacity/workload uneven?
- which exceptions remain unresolved?
- where is metric data incomplete?
- where is distribution bottleneck?
- what requires Manager action versus Deputy attention?
- who owns next action?

Avoid:

- generic executive vanity dashboard
- rankings
- leaderboard
- unapproved targets
- raw User IDs as KPI

--------------------------------------------------
9. HIERARCHY & PERFORMANCE EXPLORER
--------------------------------------------------

Expected drill:

Manager
→ Senior
→ Supervisor
→ Seller
→ Case / Invoice

Keep:

Current Hierarchy
Historical Attribution
Credit Owner
Event Actor

separate.

Visibility of entire subtree must not imply write authority.

Do not mix grains.

Reuse common Performance Explorer contracts where valid.

--------------------------------------------------
10. F02 / F03 / F04 / METRIC TRUST
--------------------------------------------------

Deputy inherits visibility of the same metric risks.

Explicitly address:

F02 pre-invoice predicate inconsistency
F03 legacy-only case counts
F04 report/invoice mismatch

Do not assume they are fixed.

Define trusted KPI contract using:

name
grain
source
cohort
time_basis
scope
formula/predicate
freshness

If undefined:
NOT FINAL.

Missing data != zero.

No invented:
conversion rate
target
efficiency score
ranking
SLA

--------------------------------------------------
11. DIRECT ALLOCATION
--------------------------------------------------

Audit indicated Deputy may have distribution/direct allocation capability across:

Manager
Senior
Supervisor
Seller

depending on actual scope.

Do NOT infer all descendants writable.

Define:

- ownership/custody requirement
- recipient eligibility
- direct vs skip-level path
- reason for bypass
- source
- financial dependency
- preview
- apply recheck
- history
- partial results

If business authority remains unclear:
CONDITIONAL / NEEDS POLICY.

Do not silently remove existing technical paths.

--------------------------------------------------
12. RETURN / RECALL
--------------------------------------------------

Gate0 F01 remains binding.

Deputy rank does NOT mean:
recall everything in hierarchy.

Define:

- allowed return basis
- current custody
- transfer proof
- financial dependency
- source ambiguity
- race/recheck
- cancelled/rejected behavior
- historical preservation

Unknown financial linkage:
fail closed.

--------------------------------------------------
13. EXCEPTIONS
--------------------------------------------------

Define Deputy-level exceptions.

Possible categories only where evidence supports:

- Manager-level unresolved operational issue
- cross-team capacity issue
- distribution conflict
- stale/incomplete metric
- unresolved F04 mismatch
- inactive hierarchy actor
- consumed-return conflict
- unresolved Case lineage
- invoice exception requiring correct owner
- HR chain waiting
- organizational scope conflict

For each:

owner
Deputy responsibility
allowed action
prohibited action
next actor
resolution signal

Do not invent escalation command/SLA.

--------------------------------------------------
14. INVOICES
--------------------------------------------------

Deputy likely needs oversight, not Finance operation.

Specify:

Must See:
invoice
Manager/Senior/Seller context
Case/source
invoice state
payment stage
Finance review
paid/remaining/unit
next actor
reason
freshness/completeness

Classify observed Deputy controls:

READ
ASSIST
WRITE
RESTRICTED
NEEDS VALIDATION

Do not inherit Manager invoice-assist rights automatically.

Do not grant Finance authority.

--------------------------------------------------
15. ARCHIVE / EXTRA-NUMBER
--------------------------------------------------

Explicitly determine whether Deputy should:

- see Manager archive aggregates
- drill into archive records
- act on archive records
- see extra-number request backlog
- review Manager-owned extra-number requests

Do NOT infer review authority from hierarchy.

If no Deputy-specific handler/policy exists:
READ / MONITOR only
or
NOT IN ROLE

Record exact evidence.

--------------------------------------------------
16. HR REQUESTS
--------------------------------------------------

Define Deputy role in HR chain.

Potential:
submit scoped
review when current reviewer
observe chain

But:
Deputy != HR final apply.

Define:

subject scope
current reviewer
next reviewer
history
applied/failed distinction

Intermediate approval != applied change.

--------------------------------------------------
17. REPORTS
--------------------------------------------------

Deputy probably has broader reporting scope than Manager.

Define:

- Manager-level grouping
- Senior/Supervisor/Seller drilldown
- source coverage
- time basis
- freshness
- export permission
- sensitive fields

Keep separate:

Current Snapshot
Event Range
Historical As-Of

Do not imply entire organization access unless evidence supports it.

--------------------------------------------------
18. OWN ACCOUNT / WALLET
--------------------------------------------------

Personal only.

Not:
organization revenue
team commission
Finance ledger

Respect pure-read / F07.

Use previous evidence if safer than opening risky renderer.

--------------------------------------------------
19. BULK ACTIONS
--------------------------------------------------

Classify:

Bulk Direct Allocation
Bulk Return
Bulk Export
Bulk Exception Action
Bulk HR Review
Bulk Archive Review
Bulk Extra-number Review

as:

REQUIRED
SUPPORTED
CONDITIONAL
NOT ALLOWED
NEEDS VALIDATION

Do not copy Manager bulk rights upward automatically.

--------------------------------------------------
20. PERMISSIONS
--------------------------------------------------

Use conceptual permissions only.

Possible concepts:

view_deputy_overview
view_manager_hierarchy
view_scoped_cases
view_cross_manager_performance
assign_scoped_cases
return_eligible_cases
view_scoped_invoices
view_exceptions
submit_hr_request
review_assigned_hr_request
view_reports
export_reports
view_own_wallet

Do not invent WP capability names.

For WRITE/APPROVAL actions specify:

scope
state guard
row guard
field restriction.

--------------------------------------------------
21. HANDOFFS
--------------------------------------------------

Document only relevant paths:

upstream → Deputy custody if real
Deputy → Manager
Deputy → Senior/Supervisor/Seller if conditional
Manager → Deputy unresolved exception if existing policy supports
Finance → Sales-visible outcome
HR → Deputy review → next reviewer/HR
Reports → read-only

Preserve separately:

Current Custody
Original Owner
Next Actor
Credit Owner

No generic ACK/escalation invented.

--------------------------------------------------
22. DATA VISIBILITY
--------------------------------------------------

Define:

MUST SEE
NICE TO HAVE
SHOULD HIDE / RESTRICT

Should Hide/Restrict:

Finance approval/posting
commission rules
HR credentials/compensation
raw MIS repair
out-of-scope customers
sensitive bank data
technical statuses by default
User ID as KPI

--------------------------------------------------
23. STATES
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
- partial hierarchy load
- incomplete report coverage
- metric mismatch
- out-of-scope direct allocation
- recipient changed before apply
- exception owner unresolved

0 must not represent unavailable data.

--------------------------------------------------
24. PRODUCT GAPS / DEFERRED
--------------------------------------------------

Record unresolved items as:

PRODUCT GAP — DO NOT IMPLEMENT

or

DEFERRED / CONDITIONAL

Expected areas:

- Deputy direct-allocation policy
- write authority across hierarchy
- unresolved KPI/cohort definitions
- F04 parity
- formal escalation ownership
- archive/extra-number visibility vs action
- HR reviewer position
- bulk operation semantics

Do not resolve by assumption.

--------------------------------------------------
25. COMPATIBILITY
--------------------------------------------------

KEEP + ADAPT.

Preserve:

IDs
routes
history
legacy sources
ownership relationships
current workflows
permissions until rollout

No destructive:

migration
status rename
ID rewrite
invoice relink
historical attribution rewrite

Risky future change:
FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK.

--------------------------------------------------
26. FINAL FILE STRUCTURE
--------------------------------------------------

Keep output concise.

1. Executive Summary
2. Role Mission & Boundaries
3. Current Verified Inventory
4. Deputy vs Manager / MIS
5. Target IA
6. Leadership Overview
7. Hierarchy & Performance
8. KPI Contract
9. Direct Allocation / Return
10. Exceptions
11. Invoices
12. Archive / Extra-number Boundary
13. HR Requests
14. Reports / Wallet
15. Permissions / Bulk Actions
16. Handoffs
17. Data Visibility / States
18. Product Gaps / Deferred
19. Compatibility / Safe Adoption
20. Claude Design Handoff

Avoid repeating Gate0 rules.
Cross-reference instead.

--------------------------------------------------
27. CLAUDE HANDOFF
--------------------------------------------------

End with:

WHAT CLAUDE MAY ASSUME
WHAT CLAUDE MUST NOT INVENT
SHARED COMPONENTS TO REUSE
DEPUTY-SPECIFIC DESIGN NEEDS
LIKELY NEW SHARED COMPONENTS
DEFERRED / CONDITIONAL ITEMS

Potential design needs only if supported:

- Leadership Overview
- Manager Hierarchy Explorer
- Cross-Manager Performance Explorer
- Exception Coordination
- Conditional Direct Allocation
- Metric Trust / Reconciliation

Do NOT design them.

--------------------------------------------------
28. STOP CONDITION
--------------------------------------------------

After final markdown:
STOP.

Do not implement.
Do not start Claude work.
Do not start MIS audit/spec.