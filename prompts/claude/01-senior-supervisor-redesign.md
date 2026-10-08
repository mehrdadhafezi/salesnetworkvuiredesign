TASK
SENIOR SUPERVISOR REDESIGN

MODE
DESIGN / PROTOTYPE ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

SUCCESS:
SENIOR SUPERVISOR DESIGN V1 FROZEN — MULTI-TEAM CRM DESIGN SYSTEM EXTENDED

Then STOP.
Do not start Sales Manager.

--------------------------------------------------
1. SOURCES OF TRUTH
--------------------------------------------------

VISUAL:
- SELLER DESIGN V1 FROZEN
- SUPERVISOR DESIGN V1 FROZEN

PRODUCT:
- SENIOR-SUPERVISOR-PRODUCT-SPEC.md
- GATE-0-PRODUCT-INVARIANTS.md

Product Spec defines WHAT/WHY.
Frozen Design System defines HOW.

Never derive business logic from current live visuals or mock data.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT perform a new full product audit.

Codex already completed the Senior live audit and froze the Product Spec.

Use live CRM only for targeted visual/content verification when:
- the Product Spec references an existing control/content pattern,
- a current screen is needed to understand density,
- or a contradiction must be resolved.

Do NOT:
- re-audit every business rule,
- reread unrelated plugin code,
- dump DOM,
- produce screenshots after every minor change,
- repeatedly test unchanged Seller/Supervisor screens,
- narrate every implementation step in chat.

Reuse frozen shared assets directly.

Prefer:
Product Spec → existing shared components → targeted prototype change → focused QA.

Keep progress messages brief.

--------------------------------------------------
3. HARD DESIGN BOUNDARY
--------------------------------------------------

Senior = MULTI-TEAM COORDINATION.

Not:
Supervisor with more rows.
Not:
generic executive dashboard.

Do NOT invent:

permissions,
statuses,
transitions,
KPI formulas,
targets,
ranking,
SLA,
skip-level policy,
Finance authority,
HR final authority,
conversion assignment rights,
escalation command,
refund behavior,
commission behavior.

Deferred Product decisions stay conditional/disabled/informational.

--------------------------------------------------
4. TARGET IA
--------------------------------------------------

Use Product Spec as final WHAT:

PRIMARY
- Multi-Team Overview
- Structure & Performance Explorer
- Distribution & Eligible Return
- Ready Conversion — Team View
- Invoices & Exceptions
- HR Requests
- Shared Reports

SECONDARY
- Case Explorer / Source History
- Own Account / Wallet

CONDITIONAL
- My Assigned Conversions

ADVANCED
- metric/source/link diagnostics

Do not make every destination a top-level tab.

Reuse Supervisor RoleNavigation grouping/overflow.

--------------------------------------------------
5. REUSE BEFORE CREATE
--------------------------------------------------

Before making any new component:

check Seller + Supervisor shared library.

Reuse existing:

AppShell
Header
RoleNavigation
PageHeader
Metric/Stat components
MetricMeta
FreshnessIndicator
ScopeBadge
Toolbar/Search/Filters
DataTable
SelectableDataTable
BulkActionBar/Review/Result
Drawer/Sheet
Timeline
Status system
EligibilityPanel
OwnershipGrid
FinancialFacets
ApprovalChain
AttentionList
Reports
OperationLog
Help/Tour
Toast/Tooltip
CommandPalette
system states
Theme
Density
Focus Mode

Do NOT duplicate shared components.

Only create a new component if Senior has a proven unmet need.

--------------------------------------------------
6. REQUIRED SENIOR EXPERIENCE
--------------------------------------------------

A. MULTI-TEAM OVERVIEW

Answer quickly:

- which teams are in scope?
- responsible Supervisor?
- workload concentration?
- important exceptions?
- stale/incomplete metrics?
- invoice/ready/HR issue?
- next responsible actor?

No leaderboard, ranking or target wall.

B. STRUCTURE & PERFORMANCE EXPLORER

Support clear modes such as:

Team
Supervisor
Seller
Conversion
Invoice
Case Workload

Never mix grains.

Distinguish visibly:

Current Hierarchy
Historical Attribution
Event Actor

Do not infer historical credit from today's parent.

C. DIRECT / INDIRECT RELATIONSHIPS

Clearly distinguish:

direct Supervisors
direct Sellers if any
indirect Sellers

Visibility != write authority.

D. METRIC TRUST

Use only Product-Spec-approved metrics.

Incomplete definitions must display:
تعریف نشده
داده ناکافی
پوشش ناقص
نیازمند تطبیق

Never fake zero.

Show operator-friendly:
freshness
coverage
basis
discrepancy

without exposing unnecessary technical internals.

--------------------------------------------------
7. DISTRIBUTION / RETURN
--------------------------------------------------

Primary Senior distribution path:
Senior → Supervisor.

Direct-to-Seller:
CONDITIONAL / EXCEPTION PATH.

Do not present skip-level as normal primary behavior.

If represented, show:

relationship
bypassed level
reason
current custody
eligibility
dependency conflict

without inventing approval policy.

RETURN:
reuse Supervisor safe-return pattern.

Senior must NOT visually appear able to recall everything below them.

Represent:

Eligible
Blocked
Conflict
Unknown / Reconciliation Needed

Gate0 financial dependency rules remain binding.

Preview is not a guarantee of Apply.

--------------------------------------------------
8. READY CONVERSION
--------------------------------------------------

This is SUPERVISORY / READONLY in baseline.

Show:

team
responsible Supervisor
Seller
current owner
next actor
status
source
allowed financial context
freshness if defined

Do NOT provide direct Senior assignment CTA.

If formal escalation action is not defined:
show responsible actor / needs attention,
not a functional “ارجاع” command.

--------------------------------------------------
9. INVOICES / HR / CASE / REPORTS
--------------------------------------------------

INVOICES
Show distinct:
Invoice Status
Payment Stage
Finance Review
Paid / Remaining
Next Actor
Reason
data/link confidence where needed.

No Finance approve/reject/refund/post.

HR
Clearly separate:
تأیید این مرحله
from
اعمال نهایی توسط منابع انسانی.

CASE EXPLORER
Represent target unified view cautiously:
SourceBadge
Legacy facet
Source history
Case context
Link confidence
Incomplete coverage

Do not imply adapter is already implemented.

REPORTS
Reuse Shared Reports with:
multi-team scope
team/person subset
metric definition
time basis
coverage
freshness
export permission.

Do not visually imply access outside scope.

WALLET
personal + read-only representation.
No simulated posting.

--------------------------------------------------
10. EXCEPTIONS
--------------------------------------------------

Design coordination around Product-Spec-supported exceptions such as:

workload variation
assignment conflict
stale/incomplete metric
consumed-return conflict
invoice pending/rejected
ready waiting for Supervisor
inactive recipient
unresolved source/link
HR request waiting in chain

For each, prioritize:

subject
responsible actor
state
reason
allowed next action

Only show escalation target/action if Product Spec supports it.

--------------------------------------------------
11. LIKELY SHARED EXTENSIONS
--------------------------------------------------

Evaluate whether these are genuinely needed:

PerformanceExplorer
DimensionSwitcher
ScopeBreadcrumb
AttributionBasis
MetricCoverage
LinkConfidence

Prefer generic names because Manager/Deputy may reuse them.

Classify every new component:

SHARED CRM
or
SENIOR-SPECIFIC

Do not create speculative components.

--------------------------------------------------
12. BULK / STATES
--------------------------------------------------

Baseline:

Bulk Assignment → supported
Bulk Return → conditional
Bulk Export → conditional
Bulk Escalation → not verified
Bulk HR Review → not allowed

Required system states:

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

Multi-team additions:

partial source coverage
one team failed / others loaded
stale team metric
unknown historical attribution

Reuse existing state patterns instead of creating unique ones per screen.

--------------------------------------------------
13. RESPONSIVE / ACCESSIBILITY / COMFORT
--------------------------------------------------

Support:

1920×1080
1366×768
1024×768
768×1024
390×844

Avoid solving dense Senior views with page-level horizontal scroll.

Preserve:

Light / Dim / Dark
Comfortable / Compact
Focus Mode

Accessibility:
keyboard,
visible focus,
semantic controls,
drawer/dialog focus,
ARIA where appropriate,
status not color-only,
contrast ≥4.5:1,
reduced motion.

Do not add visual noise, rankings or oversized dashboard cards.

--------------------------------------------------
14. HELP
--------------------------------------------------

Reuse Shared Help/Tour engine.

Senior-specific content only.

Suggested coverage:
Overview
Structure/Scope
Performance Explorer
Distribution
Ready
Invoices
HR
Reports
Help

Tour must never perform business actions.

--------------------------------------------------
15. QA — TOKEN EFFICIENT STRATEGY
--------------------------------------------------

Do not render a huge screenshot matrix after every change.

Use three QA passes:

PASS A — STRUCTURAL
Use 1366 desktop + 390 mobile in one theme.
Catch layout/runtime/navigation problems first.

PASS B — INTERACTION
Test only major flows:
performance drilldown,
distribution preview,
blocked return,
readonly ready,
invoice exception,
HR review context,
reports,
conditional personal module.

PASS C — FINAL MATRIX
Only after A/B pass:
validate all required widths,
themes,
densities,
Focus Mode,
key system states.

Screenshots:
capture only representative/final evidence.

Do not create dozens of redundant screenshots unless a failure requires them.

Automated checks are preferred for:
syntax,
horizontal overflow,
contrast,
basic accessibility,
keyboard guards.

Do not repeatedly re-render unchanged Seller/Supervisor pages.

--------------------------------------------------
16. ZERO BREAKAGE
--------------------------------------------------

Do not destructively modify Seller or Supervisor frozen work.

Do not assume:

DB migration
status rename
ID rewrite
legacy route removal
historical merge
permission rollout
backend fixes

Prototype represents Target UX only.

KEEP + ADAPT remains assumed.

--------------------------------------------------
17. DELIVERABLES
--------------------------------------------------

Keep deliverables compact:

1. Current→Target mapping
2. Senior IA
3. Working responsive prototype
4. Performance Explorer
5. Distribution / Return
6. Ready readonly
7. Invoice / HR / Reports
8. Conditional Personal Conversion
9. Shared Component Delta
10. Help/Tour
11. System States
12. Final QA evidence
13. Known Limitations
14. Product Gaps / Deferred
15. Freeze Review

Do not duplicate the same information across multiple documents.

Prefer one main Senior design spec + prototype + small final QA evidence set.

--------------------------------------------------
18. FREEZE REVIEW
--------------------------------------------------

Report only:

Fixed During QA
Verified
Shared Components Added
Senior-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role Consistency
Ready For Next Role?

If there is a true design blocker, do not freeze.

If clear:

SENIOR SUPERVISOR DESIGN V1 FROZEN — MULTI-TEAM CRM DESIGN SYSTEM EXTENDED

Then STOP.

Do not begin Sales Manager.