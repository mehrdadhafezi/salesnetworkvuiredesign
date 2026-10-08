# GitHub Handoff

## Branch Policy

`main` is the accepted integration baseline.

All future work is performed on task branches.

Examples:

- `claude/senior-supervisor-redesign`
- `claude/sales-manager-redesign`
- `codex/foundation-result-contract`
- `fix/f01-return-guard`

## Pull Requests

Every PR must include:

- Summary
- Scope
- Files changed
- Product / invariant references
- QA or tests performed
- Known limitations
- Deferred issues
- Rollback notes

## Merge Rules

Do not merge automatically.

Design PR:
requires design/product review.

Sensitive implementation PR:
requires engineering review plus applicable domain review.

## No-Go

Do not combine in one PR:

- UI redesign
- permission rewrite
- DB migration
- financial logic change

unless explicitly approved as a coordinated change.

## Claude Cloud Sessions

Claude must:

1. create/use a dedicated `claude/*` branch
2. read `AGENTS.md`
3. read only relevant frozen sources
4. perform the requested redesign
5. run focused QA
6. commit
7. push
8. create PR to `main`
9. never merge
10. return PR URL

## Codex

Codex follows the same PR contract using `codex/*` branches.

## Rollback

Keep each PR small enough for package/code rollback.

Database/history rollback must never be assumed from Git revert.
