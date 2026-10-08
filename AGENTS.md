# AGENTS.md

## Purpose

This repository is a controlled CRM redesign and migration-safe implementation project.

Always read this file before doing work.

---

## Global Contract

KEEP + ADAPT.

Preserve:
- native IDs
- database history
- invoice/payment relationships
- source provenance
- historical ownership
- existing routes
- existing statuses
- current permissions until explicitly approved

Never silently migrate or rewrite historical data.

---

## Sources of Truth

Product / Business authority:

1. `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md`
2. `docs/source-of-truth/GATE-0-PRODUCT-INVARIANTS.md`
3. relevant frozen Role Product Spec
4. accepted Audit / Evidence

Design authority:

1. Frozen shared CRM Design System
2. previously frozen role designs
3. current approved role redesign task

Prototype/mock data is NOT business authority.

---

## Agent Ownership

### Claude

Owns:

HOW
UX
UI
Design System
Prototype
Responsive behavior
Accessibility
Visual QA

Claude MUST NOT invent:

permissions
business statuses
workflow transitions
financial rules
role authority
KPI formulas
database behavior

### Codex

Owns:

WHAT / WHY
Product Architecture
Role Architecture
Workflow contracts
Engineering architecture
Implementation
Tests
Migration-safe adapters

Codex must not redesign approved UX without a design blocker.

---

## Context Efficiency

Reuse frozen evidence first.

Do NOT:

- restart full audits without a real reason
- reread unrelated plugin files
- dump full DOM/source/logs
- make repetitive screenshots
- reopen frozen role decisions
- repeatedly inspect unchanged screens
- produce long progress narration

Prefer:

frozen contract
→ targeted evidence
→ bounded change
→ focused QA.

Saving context must never reduce verification quality.

---

## Safety

Unless the task explicitly authorizes implementation:

NO:
- database mutation
- production/staging mutation
- permission mutation
- role mutation
- customer mutation
- invoice/payment mutation
- financial posting
- HR change
- migration
- destructive cleanup

---

## Identity

Phone is discovery/search data only.

Phone must NEVER be used as canonical Case identity or financial join proof.

Unknown or conflicting lineage remains UNKNOWN / CONFLICT.

---

## Ownership

Never collapse:

Current Custody
Original Owner
Next Actor
Event Actor
Credit Owner

Current hierarchy must not rewrite historical attribution.

---

## Permissions

Permission model:

Route
→ Module
→ Action
→ Row / Field

View != Write.

Hierarchy != Permission.

Position != Role != Capability != Credential.

---

## Git Workflow

After baseline initialization:

NEVER work directly on `main`.

Use branches:

`claude/<task>`
`codex/<task>`
`fix/<issue>`
`docs/<scope>`

Every task must:

1. inspect current branch
2. make bounded changes
3. run relevant QA/tests
4. review diff
5. commit
6. push branch
7. create Pull Request into main
8. report PR URL
9. STOP

Do NOT merge your own Pull Request unless explicitly instructed.

---

## Frozen Work

Do not reopen frozen role/product/design contracts unless:

- a true blocker is demonstrated
- frozen sources directly contradict each other
- implementation evidence proves the target impossible

Record blockers instead of silently changing contracts.

---

## Runtime Verification

HR and Finance runtime remain NOT LIVE VERIFIED until dedicated real role accounts are tested.

Admin behavior is not proof of HR or Finance role behavior.

---

## Sensitive Outcomes

Do not convert missing/unknown data into zero.

Distinguish:

SUCCESS
PARTIAL
FAILED
UNKNOWN
CONFLICT
BLOCKED
UNAUTHORIZED

Outcome Unknown must be reconciled before blind retry.

---

## Final Rule

A shared component may introduce a new reusable presentation pattern.

It must NOT introduce a new business capability.
