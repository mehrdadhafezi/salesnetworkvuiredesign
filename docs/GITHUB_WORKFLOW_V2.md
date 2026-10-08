# GitHub Workflow V2 — Sales Network CRM Redesign

## Repository contract
GitHub is the single source of truth for accepted project work.

Current meaning:
- `main` = latest accepted stable baseline.
- `epic/SN-redesign-v1` = integration branch for CRM Redesign V1.
- Task branches merge into the epic branch through Pull Requests.
- The epic branch merges into `main` only after final cross-role review.

## Required flow
Every change follows:

Issue → dedicated branch → implementation/design → focused QA → commit → push → Pull Request → AI review → human review → merge

No direct task work on `main` or `epic/SN-redesign-v1`.

## Issue identifiers
Use stable task IDs in title, branch, commits, and PRs.

Current redesign epic:
- SN-200 — CRM Redesign V1
- SN-201 — Senior Supervisor Design V1
- SN-202 — Sales Manager Design V1
- SN-203 — Sales Deputy Design V1
- SN-204 — MIS Design V1
- SN-205 — HR Design V1
- SN-206 — Finance Design V1
- SN-207 — Cross-Role Design System V1

## Branch naming
- `design/SN-<id>-<slug>`
- `feature/SN-<id>-<slug>`
- `bugfix/SN-<id>-<slug>`
- `hotfix/SN-<id>-<slug>`
- `docs/SN-<id>-<slug>`
- `chore/SN-<id>-<slug>`

Examples:
- `design/SN-201-senior-supervisor`
- `feature/SN-310-refund-register`
- `bugfix/SN-311-assigned-search`

## Commit format
Prefer:
- `design(SN-201): build senior supervisor workspace`
- `feat(SN-310): add refund register import`
- `fix(SN-311): resolve assigned search 404`
- `test(SN-201): add responsive QA evidence`
- `docs(SN-199): add GitHub workflow V2`

One concern per commit. Avoid unrelated refactors.

## Pull Requests
Task PR target for redesign work:
`design/SN-20x-...` → `epic/SN-redesign-v1`

Final integration:
`epic/SN-redesign-v1` → `main`

PRs must include:
- linked Issue
- scope
- authoritative product sources
- files changed
- shared components reused/changed
- QA/tests
- database/API/permission impact
- known risks
- deferred product gaps
- rollback notes where relevant

Do not self-merge unless the user explicitly approves.

## Claude / Codex collaboration
Never have Claude and Codex make changes on the same branch simultaneously.

Recommended:
- one agent = implementer/designer
- another agent = reviewer
- human = final merge authority

Claude owns UX/UI/design-system work unless explicitly assigned otherwise.
Codex owns implementation engineering and code-focused tasks unless explicitly assigned otherwise.
Frozen product architecture and role specs remain authoritative.

## Dependency order for CRM Redesign V1
SN-201 → SN-202 → SN-203 → SN-204 → SN-205 → SN-206 → SN-207

SN-207 must not begin until SN-201 through SN-206 are accepted into the epic branch.

## Labels
Recommended taxonomy:
- type:design, type:feature, type:bug, type:hotfix, type:docs, type:architecture, type:qa
- role:seller, role:supervisor, role:senior-supervisor, role:sales-manager, role:sales-deputy, role:mis, role:hr, role:finance, role:cross-role
- agent:claude, agent:codex, agent:human
- risk:low, risk:medium, risk:high, risk:critical
- status:blocked, status:ready, status:review, status:frozen
- area:frontend, area:backend, area:database, area:permissions, area:finance, area:hr, area:reporting

## Milestone
CRM Redesign V1

## Board states
Backlog → Ready → In Progress → Review → Changes Requested / Approved → Done
Blocked is a separate state.

## Release rule
Once implementation reaches production:
merge → version bump → CHANGELOG → tag → ZIP/release artifact → production.

Production tags are immutable rollback points.
