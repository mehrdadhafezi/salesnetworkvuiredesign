TASK
FINANCE REDESIGN

MODE
DESIGN / PROTOTYPE ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

SUCCESS:
FINANCE DESIGN V1 FROZEN — FINANCIAL OPERATIONS CRM DESIGN SYSTEM EXTENDED

Then STOP.
Do not start implementation.

--------------------------------------------------
1. SOURCES OF TRUTH
--------------------------------------------------

PRODUCT:
- FINANCE-PRODUCT-SPEC.md
- GATE-0-PRODUCT-INVARIANTS.md

VISUAL:
- all previously frozen CRM role designs
- Shared CRM Design System

IMPORTANT:
FINANCE ROLE RUNTIME NOT LIVE VERIFIED.

Do not visually imply current Finance workflows were successfully runtime-tested.

Product Spec = WHAT / WHY.
Shared Design System = HOW.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT re-run a full Finance audit.

Codex already verified:

- Finance code inventory
- Review Queue
- financial state distinctions
- Approve / Reject guards
- Refund uncertainty
- Ledger
- Rules / Runs / Posting
- entitlement boundaries
- F06 / F07
- posting/idempotency risks
- Finance runtime absence

Use current CRM only for targeted visual/content reference if truly necessary.

Do NOT:
- re-audit Sales/MIS/HR
- reread unrelated plugin code
- dump DOM
- create redundant screenshots
- narrate every implementation detail

Prefer:
Product Spec → Shared Components → Finance design → focused QA.

--------------------------------------------------
3. CORE DESIGN PRINCIPLE
--------------------------------------------------

Finance = FINANCIAL TRUTH + CONTROLLED DECISION + AUDITABILITY

Not:
- generic invoice table
- Sales workspace
- accounting spreadsheet
- unrestricted ledger editor
- auto-fix console

Finance must clearly distinguish:

Payment Evidence
Finance Review
Payment Stage
Invoice State
Paid Amount
Remaining
Refund State
Ledger Posting
Entitlement
Settlement

Never collapse these into one status.

--------------------------------------------------
4. TARGET IA
--------------------------------------------------

PRIMARY:

- Review Queue
- Reconciliation / Refund
- Ledger
- Rules / Runs / Posting
- Reports

SECONDARY:

- Audit / History
- authorized Configuration

ADVANCED:

- Maintenance / Recovery

Do not make diagnostic / maintenance controls part of normal daily review.

--------------------------------------------------
5. REUSE EXISTING DESIGN SYSTEM
--------------------------------------------------

Reuse:

AppShell
RoleNavigation
PageHeader
ScopeBadge
MetricMeta
FreshnessIndicator
MetricCoverage
LinkConfidence
DataTable
SelectableDataTable
Drawer / Sheet
Timeline
FinancialFacets
ApprovalChain
OperationLog
BulkReview
BulkResult
Attention / Exception patterns
ReportCatalog
Toast / Tooltip
Help / Tour
CommandPalette
shared system states
Theme / Density / Focus Mode

Do not fork the visual language.

--------------------------------------------------
6. REVIEW QUEUE
--------------------------------------------------

This is Finance's main operational screen.

Primary grain:
reviewable payment stage / evidence context linked to invoice.

Must make visible:

- Invoice
- Case / Source
- customer minimum context
- Seller / team
- payment stage
- submitted evidence
- evidence version/source/time
- claimed amount
- validated amount
- currency/unit
- total / paid / remaining
- prior review
- reason
- current reviewer
- next actor
- linkage confidence
- freshness

Do not present overlapping facets as exclusive totals.

--------------------------------------------------
7. FINANCIAL STATE UX
--------------------------------------------------

Design clearly independent dimensions:

Invoice Status
Payment Stage
Finance Review
Payment Evidence
Paid Amount
Remaining
Refund State
Ledger Posting State

Use layered/contextual state UI rather than one giant status badge.

Important:

receipt uploaded != payment validated
payment validated != stage approved
stage approved != invoice fully paid
invoice fully paid != entitlement posted
wallet credit != settlement

Do not fake simplification at the cost of truth.

--------------------------------------------------
8. APPROVE / REJECT
--------------------------------------------------

Finance decisions are sensitive actions.

Before approval show:

- current stage
- evidence
- amount/unit
- prior decisions
- Case/link confidence
- resulting financial impact
- resulting next actor

Reject must require visible reason context.

Reject must NOT visually imply:
delete
cancel
refund
history reset

Stale or changed stage:
Conflict / reload required.

Use Sensitive Financial Action Confirmation if appropriate.

--------------------------------------------------
9. REFUND
--------------------------------------------------

Refund remains CONDITIONAL because OPD-04 is unresolved.

Design separate concepts:

Requested
Reported
Proof Pending
Confirmed
Partial
Full
Unknown

Show:

- original invoice/payment
- requested amount/unit
- already refunded amount
- remaining eligible context
- execution source
- proof
- reviewer
- history

Do NOT imply:
Cancelled Invoice = Refunded Money.

Do not invent bank workflow or proof policy.

--------------------------------------------------
10. LEDGER EXPLORER
--------------------------------------------------

Ledger must look audit-oriented, not editable balance UI.

Show:

- Transaction ID
- Business Key
- domain/account
- debit / credit
- amount/unit
- source
- invoice/refund/commission/run relation
- actor/system
- effective time
- posting time
- posting state
- reversal/correction relation

Corrections should visually appear as linked events.

Do not design destructive edit/delete of financial history.

--------------------------------------------------
11. RULES / RUNS / POSTING
--------------------------------------------------

Design lifecycle:

Draft
Preview
Approved
Posting
Posted
Partial
Failed
Outcome Unknown
Reconciled

Important distinction:

Generate Preview may itself create run metadata.

Preview != pure read.

Approved != Posted.

Posted summary must show coverage:

- total intended
- processed
- posted
- existing/skipped
- failed
- unprocessed
- unknown

Do not visually claim full-run success from a bounded result.

--------------------------------------------------
12. OUTCOME UNKNOWN / IDEMPOTENCY
--------------------------------------------------

This needs strong Finance UX.

When outcome may have committed:

do NOT offer immediate blind Retry.

Show:

- original intent
- transaction/business key
- possible committed state
- lookup/reconciliation requirement
- retry eligibility
- existing transaction result
- next safe action

Potential shared component:

OutcomeUnknownContext

or

IdempotencyContext

Only create if it generalizes cleanly.

--------------------------------------------------
13. ENTITLEMENT / COMMISSION BOUNDARY
--------------------------------------------------

Clearly distinguish:

Earned Entitlement
Calculated Preview
Approved Posting
Wallet Credit
Settlement
Reversal

Show source policy/engine where relevant.

Do not make:
current owner
original seller
next actor
commission recipient

look synonymous.

No HR compensation editing in Finance.

--------------------------------------------------
14. RECONCILIATION WORKSPACE
--------------------------------------------------

Supported issue categories:

- amount mismatch
- duplicate payment
- missing mirror
- stage/review mismatch
- invoice/report mismatch
- refund mismatch
- ledger mismatch
- unknown posting outcome
- currency/unit ambiguity
- orphan-looking relation

Each issue should show:

- evidence
- financial impact
- owner/domain
- allowed Finance action
- restricted action
- next actor
- resolution proof

No Fix All.

--------------------------------------------------
15. REPORTS
--------------------------------------------------

Keep grains distinct:

- Review workload
- Current invoice financial state
- Transaction events
- Ledger activity
- Runs / Posting
- Historical As-Of
- Settlement readiness

Every report should expose when needed:

scope
grain
time basis
unit
source
freshness
coverage
reconciliation state

Do not sum incompatible monetary units.

Settlement readiness != settlement paid.

--------------------------------------------------
16. F06 / PERMISSION AWARENESS
--------------------------------------------------

Current F06 exists:
view-level Finance permission may be used by sensitive writes.

Design Target Contract with action-aware permissions.

UI must never imply:

Can View = Can Execute

Restricted/Unavailable action states should be explicit.

Do not invent actual WP capabilities.

--------------------------------------------------
17. F07 / WALLET PURE READ
--------------------------------------------------

Do not simulate or depend on wallet autopost.

Any wallet/ledger read representation in prototype is read-only.

If engine effect is relevant:
show status/context,
not a fake background posting behavior.

--------------------------------------------------
18. BULK ACTIONS
--------------------------------------------------

Baseline:

Bulk Review → required
Bulk Approve → conditional
Bulk Reject → needs validation
Bulk Refund → not allowed baseline
Bulk Reconcile → diagnosis only
Bulk Posting → conditional
Bulk Export → conditional

For every financial bulk action:

- show item count
- per-item eligibility
- impact review
- per-item result
- partial outcome
- unknown outcome

Never show fake “100 succeeded” from request submission alone.

--------------------------------------------------
19. SENSITIVE FINANCIAL ACTIONS
--------------------------------------------------

For:
approve
reject
refund
posting
reversal
recovery

confirmation should show:

- target
- current state
- amount/unit
- affected records
- resulting state
- irreversible/protected history
- actor
- reason if needed

Avoid generic confirmation dialogs.

--------------------------------------------------
20. STATES
--------------------------------------------------

Shared:

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

Finance-specific examples:

- stale review
- duplicate payment
- amount mismatch
- missing evidence
- refund proof missing
- posting conflict
- partially posted run
- existing transaction found
- unprocessed run tail
- currency ambiguity

Do not convert unknown/missing to zero.

--------------------------------------------------
21. RESPONSIVE / ACCESSIBILITY
--------------------------------------------------

Validate:

1920×1080
1366×768
1024×768
768×1024
390×844

Finance has dense evidence/history.

Use:
priority columns
drawer/details
summary/detail
sticky decision area where appropriate

Avoid page-level horizontal scroll.

Preserve:
Light / Dim / Dark
Comfortable / Compact
Focus Mode

Accessibility:
keyboard
focus
semantic tables/forms
dialog focus
status not color-only
contrast >= 4.5:1
reduced motion

--------------------------------------------------
22. HELP / TOUR
--------------------------------------------------

Reuse shared Help/Tour.

Finance-specific:

1. Review Queue
2. Financial States
3. Evidence
4. Approve / Reject
5. Reconciliation
6. Ledger
7. Runs / Posting
8. Reports
9. Advanced / Maintenance

Tour must not execute financial actions.

--------------------------------------------------
23. QA — TOKEN EFFICIENT
--------------------------------------------------

PASS A
1366 + 390
one theme

Check:
layout
navigation
financial state readability
dense tables
drawer/dialog
overflow

PASS B
critical flows:

- evidence review
- approve
- reject
- stale conflict
- conditional refund
- ledger drill
- run approval
- partial posting
- outcome unknown
- reconciliation

PASS C
all required widths/themes/density/focus
representative states only.

Capture minimal final screenshots.

--------------------------------------------------
24. ZERO BREAKAGE
--------------------------------------------------

Do not assume:

Finance runtime verified
F06 fixed
F07 fixed
OPD-04 resolved
OPD-06 resolved
OPD-08 resolved
OPD-09 resolved
exact-once posting solved
run coverage solved
refund execution exists
bank import exists

KEEP + ADAPT.

Do not destructively modify earlier frozen designs.

--------------------------------------------------
25. DELIVERABLES
--------------------------------------------------

Keep compact:

1. Current→Target mapping
2. Finance IA
3. Review Queue
4. Financial State Model
5. Approve / Reject
6. Refund
7. Ledger
8. Rules / Runs / Posting
9. Outcome Unknown / Idempotency
10. Reconciliation
11. Reports
12. Shared Component Delta
13. Help / Tour
14. Final QA evidence
15. Known Limitations
16. Product Gaps / Deferred
17. Freeze Review

--------------------------------------------------
26. FREEZE REVIEW
--------------------------------------------------

Report:

Fixed During QA
Verified
Shared Components Added
Finance-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role Consistency
Ready For Integration?

If no true design blocker remains:

FINANCE DESIGN V1 FROZEN — FINANCIAL OPERATIONS CRM DESIGN SYSTEM EXTENDED

Then STOP.

Do not begin implementation.