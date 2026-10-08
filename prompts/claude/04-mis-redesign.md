TASK
MIS REDESIGN

MODE
DESIGN / PROTOTYPE ONLY.

NO PLUGIN / BACKEND / DATABASE / PERMISSION / BUSINESS-LOGIC CHANGES.

SUCCESS:
MIS DESIGN V1 FROZEN — DATA OPERATIONS CRM DESIGN SYSTEM EXTENDED

Then STOP.
Do not start HR.

--------------------------------------------------
1. SOURCES OF TRUTH
--------------------------------------------------

PRODUCT:
- MIS-PRODUCT-SPEC.md
- GATE-0-PRODUCT-INVARIANTS.md

VISUAL:
- SELLER DESIGN V1 FROZEN
- SUPERVISOR DESIGN V1 FROZEN
- SENIOR SUPERVISOR DESIGN V1 FROZEN
- SALES MANAGER DESIGN V1 FROZEN
- SALES DEPUTY DESIGN V1 FROZEN if available

Product Spec = WHAT / WHY.
Frozen Design System = HOW.

Do not derive business logic from live UI or mock data.

--------------------------------------------------
2. TOKEN / CONTEXT EFFICIENCY
--------------------------------------------------

Do NOT re-run a full MIS audit.

Codex already verified:
- MIS #7
- version 2.0.123
- 11 current destinations
- import/source/custody/report structure
- F01/F02/F03/F04/F09
- planning boundaries
- maintenance boundaries

Use live CRM only for targeted visual/content checks.

Do NOT:
- reread unrelated plugin files
- dump DOM
- re-audit Sales roles
- create screenshots after every small change
- narrate every implementation step

Prefer:
Product Spec → existing components → focused MIS patterns → focused QA.

--------------------------------------------------
3. CORE DESIGN PRINCIPLE
--------------------------------------------------

MIS = DATA OPERATIONS + SOURCE QUALITY + TRACEABILITY + REPORT TRUST

Not:
- Sales operator
- generic admin panel
- Finance console
- repair-all toolbox
- spreadsheet dump

The interface should help MIS answer:

- what source/import needs attention?
- which rows are valid/duplicate/invalid/unknown?
- where did this Case/source come from?
- who currently has custody?
- what was the transfer history?
- which report can be trusted?
- what discrepancy needs reconciliation?
- what operation is ordinary vs maintenance?

--------------------------------------------------
4. TARGET IA
--------------------------------------------------

PRIMARY:
- Today / Work Queue
- Import & Data Quality
- Source Cases
- Custody / Delivery
- Reports & Reconciliation

SECONDARY / CONDITIONAL:
- Planning
- Quick Delivery

ADVANCED:
- Maintenance
- Diagnostics

Do not preserve all 11 old tabs as equal top-level destinations.

Maintenance must be visually separated from routine work.

--------------------------------------------------
5. REUSE EXISTING DESIGN SYSTEM
--------------------------------------------------

Reuse shared patterns:

- AppShell
- RoleNavigation
- PageHeader
- ScopeBadge
- FreshnessIndicator
- MetricCoverage
- LinkConfidence
- DataTable
- SelectableDataTable
- BulkActionBar / BulkReview / BulkResult
- Drawer / Sheet
- Timeline
- OperationLog
- ReportCatalog
- Error/Empty/Stale/Conflict/Partial states
- Help/Tour
- Theme / Density / Focus Mode

Also reuse hierarchy/source context patterns where relevant.

Do not duplicate shared components.

--------------------------------------------------
6. TODAY / WORK QUEUE
--------------------------------------------------

Do not create a KPI wall.

Show operationally useful items:

- source/import health
- batches needing attention
- rows waiting for action
- custody/delivery issues
- blocked return cases
- report trust issues
- unresolved identity/linkage
- next responsible actor

Important:
current MIS counters may overlap.

Do not visually imply:
waiting + delivered + returnable = partition of total.

Missing/unavailable != zero.

--------------------------------------------------
7. IMPORT & DATA QUALITY
--------------------------------------------------

Design around distinct concepts:

- File
- Batch
- Import Run
- Source Row

Do not collapse them into one “پرونده”.

Show:

- filename/source
- batch/run
- parsed rows
- imported
- skipped
- duplicate
- invalid
- failed
- unknown
- parser/quality reasons
- provenance
- checked_at / freshness

Do not imply import preview is always side-effect free.

Do not silently merge duplicate phone numbers.

--------------------------------------------------
8. SOURCE CASES / LINEAGE
--------------------------------------------------

Design a Case Lineage Inspector.

Must distinguish:

- native source ID
- MIS row
- legacy lead
- V4 item
- Dot case/invoice
- logical Case
- alias relationship
- confidence
- unresolved/contradictory linkage

Phone must never look like canonical Case identity.

Use operator-friendly statuses such as:

Verified
Partial
Conflicting
Unknown

Do not imply unified resolver is fully implemented.

--------------------------------------------------
9. CUSTODY / DELIVERY
--------------------------------------------------

Design clearly separate concepts:

- Source Owner
- Current Custody
- Original Owner
- Next Actor
- Credit Owner
- Event Actor
- Recipient

Do not visually merge them.

Flows may include:

- source assign
- pool preparation
- delivery
- quick delivery
- return

Quick Delivery is CONDITIONAL / ADVANCED.

Do not present five-level bypass as ordinary default.

--------------------------------------------------
10. RETURN SAFETY
--------------------------------------------------

This is a critical safety UX.

Return must expose:

- current custody
- transfer proof
- financial dependency
- source linkage
- blocked reason
- unknown dependency
- preview state
- apply recheck warning

States:

Eligible
Blocked
Conflict
Unknown / Reconciliation Required

Do not rely on one invoice field.

Cancelled/rejected does NOT automatically mean safe to return.

No destructive-looking “reset” affordance for ambiguous cases.

--------------------------------------------------
11. RECONCILIATION WORKSPACE
--------------------------------------------------

Design a diagnostic workspace, not unrestricted repair.

Possible issue types:

- identity mismatch
- source coverage mismatch
- duplicate source row
- unresolved alias
- missing custody
- hierarchy inconsistency
- invoice/report mismatch
- stale report
- partial import
- unknown write outcome
- orphan-looking relation
- health contradiction

Each issue should show:

- evidence
- affected entity
- owner/domain
- confidence
- allowed MIS action
- action not allowed
- resolution proof

Do not invent “Fix All”.

--------------------------------------------------
12. REPORT TRUST
--------------------------------------------------

Reports must visibly communicate:

- grain
- source coverage
- cohort
- scope
- time basis
- freshness
- gather window
- bounded/partial result
- metric version

Keep distinct:

Current Snapshot
Event Range + Current Gathered State
Historical As-Of

Do not present Event Range as Historical As-Of.

F02/F03/F04/F09 may appear as trust/reconciliation issues.

Do not fake parity.

--------------------------------------------------
13. PLANNING
--------------------------------------------------

Planning is occasional, not daily core.

Separate:

Rule Draft
Plan Preview
Plan Save
Plan Approval
Materialization

Do not visually imply:

Approve Plan = Leads Created

Show state and consequences clearly.

No forecasting, quota or capacity scoring unless Product Spec explicitly supports it.

--------------------------------------------------
14. ADVANCED / MAINTENANCE
--------------------------------------------------

Create a clear visual boundary.

Examples:

NORMAL:
- view source
- import
- ordinary assign
- report
- reconcile read

ADVANCED:
- quick delivery
- planning
- controlled materialization

MAINTENANCE:
- delete file
- delete full batch
- cleanup
- rebuild
- repair linkage
- exceptional reassignment

Maintenance actions need:

- impact summary
- affected objects
- protected dependencies
- reason
- confirmation
- audit context
- rollback warning where appropriate

Do not normalize destructive actions.

--------------------------------------------------
15. BULK OPERATIONS
--------------------------------------------------

Baseline:

Bulk Import → supported
Bulk Assignment → supported
Bulk Return → conditional
Bulk Reconcile → conditional read
Bulk Repair → maintenance only
Bulk Export → conditional
Bulk Delete/Cleanup → maintenance only

Every bulk result supports:

- applied
- skipped
- rejected
- failed
- unknown

No fake all-success.

--------------------------------------------------
16. AUDITABILITY
--------------------------------------------------

Design should expose traceability when relevant:

- actor
- action
- source
- batch/run
- CaseRef / SourceRef
- timestamp
- reason
- result
- correlation/reference
- per-item outcome

Do not overwhelm daily views with raw technical detail.

Use drilldown/details for deep evidence.

--------------------------------------------------
17. STATES
--------------------------------------------------

Required shared states:

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

MIS-specific:

- partially parsed import
- row rejected
- duplicate
- unresolved identity
- lineage conflict
- report bounded
- reconcile mismatch
- maintenance blocked
- health contradiction

Reuse shared state components.

--------------------------------------------------
18. RESPONSIVE / ACCESSIBILITY
--------------------------------------------------

Validate:

1920×1080
1366×768
1024×768
768×1024
390×844

MIS can be dense.

Avoid solving everything with horizontal scroll.

Use:
- priority columns
- drawers
- expandable evidence
- summary/detail split
- grouped bulk results

Preserve:
Light / Dim / Dark
Comfortable / Compact
Focus Mode

Accessibility:
keyboard
focus
semantic tables
dialogs
ARIA
status not color-only
contrast >= 4.5:1
reduced motion

--------------------------------------------------
19. HELP / TOUR
--------------------------------------------------

Reuse shared Help/Tour.

MIS-specific tour:

1. Today
2. Import
3. Source Quality
4. Case Lineage
5. Custody
6. Reconciliation
7. Reports
8. Planning
9. Maintenance Boundary

Tour must never execute business actions.

--------------------------------------------------
20. QA — TOKEN EFFICIENT
--------------------------------------------------

PASS A
1366 + 390
one theme

Check:
layout
nav
dense tables
drawers
overflow
runtime

PASS B
critical flows only:

- import result
- duplicate/invalid row
- lineage inspection
- custody timeline
- blocked return
- reconciliation issue
- report trust
- planning approval vs materialization
- maintenance impact review

PASS C
all required widths/themes/density/focus
representative states only.

Capture minimal final screenshots.

--------------------------------------------------
21. ZERO BREAKAGE
--------------------------------------------------

Do not assume:

- F01 fixed
- resolver complete
- F02/F03/F04 fixed
- Historical As-Of exists
- repair authority finalized
- maintenance safety implemented
- large-batch idempotency solved
- export security fully verified

KEEP + ADAPT.

Do not destructively modify frozen Sales designs.

--------------------------------------------------
22. DELIVERABLES
--------------------------------------------------

Keep compact:

1. Current→Target mapping
2. MIS IA
3. Today / Work Queue
4. Import & Data Quality
5. Source Cases / Lineage
6. Custody / Delivery
7. Return Safety
8. Reconciliation Workspace
9. Reports
10. Planning
11. Maintenance Boundary
12. Bulk Result patterns
13. Shared Component Delta
14. Help/Tour
15. Final QA evidence
16. Known Limitations
17. Product Gaps / Deferred
18. Freeze Review

--------------------------------------------------
23. FREEZE REVIEW
--------------------------------------------------

Report:

Fixed During QA
Verified
Shared Components Added
MIS-specific Components
Known Limitations
Deferred Product Decisions
Product Gaps
Cross-role Consistency
Ready For Next Role?

If no true design blocker remains:

MIS DESIGN V1 FROZEN — DATA OPERATIONS CRM DESIGN SYSTEM EXTENDED

Then STOP. 