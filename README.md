# Sales Network CRM — Product & UX Redesign

This repository is the working source of truth for the Maximum Sales Network CRM redesign.

## Status

Product Architecture: FROZEN
Cross-Role Architecture V1: FROZEN
Implementation Foundation Plan V1: READY
Phase 0 Implementation Spec V1: READY

Design integration is still in progress.

## Roles

1. Seller
2. Supervisor
3. Senior Supervisor
4. Sales Manager
5. Sales Deputy
6. MIS
7. HR
8. Finance

## Repository Structure

- `plugin/` — accepted current plugin source baseline
- `prototype/` — CRM redesign prototype and shared UI system
- `docs/source-of-truth/` — frozen product/architecture specifications
- `docs/evidence/` — audit and validation evidence
- `prompts/claude/` — approved Claude redesign tasks

## Core Rule

KEEP + ADAPT.

Do not destructively rewrite:
- database IDs
- historical ownership
- invoice relationships
- statuses
- permissions
- financial history

## Development Workflow

After the baseline commit:

main is protected conceptually.

All work must use:

Branch → Commit → Push → Pull Request → Review → Merge

Never directly push implementation/design changes to main.

## Current Design Sequence

Senior Supervisor
→ Sales Manager
→ Sales Deputy
→ MIS
→ HR
→ Finance
→ Cross-Role Design System Integration

## Important

Claude owns HOW / UX / Design.

Codex owns WHAT-WHY product architecture and implementation engineering work.

Frozen Product Specs always override prototype assumptions.
