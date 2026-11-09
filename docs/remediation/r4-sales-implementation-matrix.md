# R4 — Sales implementation matrix (invoice issuance, claim, return)

**Baseline:** `4c7c1352bb10aa0751c14d524f0e2f1867e9460f` (R3 merge).

**Scope:**
- Slice 15 blocker **P0-5**: ADR-01 / F01 inventory safety, plus invoice issue, claim and return.
- The policy-independent foundations only.

**Not authorized:** no decision is approved and no command is activated. Legacy 2.0.160 remains the
system of record.

## 1. Inventory before R4

| Capability | Before R4 | Evidence |
|---|---|---|
| Invoice issuance (`billing.IssueInvoice`) | BLOCKED: no command, no route | Slice 11 / 12 matrices; `s11-forbidden-command` |
| Invoice claim (`cases.ClaimCaseForInvoice`) | BLOCKED: no command, no route | Slice 11 matrix |
| Return / reset / release (`custody.ReturnCase` …) | BLOCKED: no command, no route | Slice 11 matrix (ADR-01, F01, OPD-03) |
| Financial-dependency guard (Cross-Role §8) | Contract only (prose); no code | Cross-Role §8 |
| Race safety of the ADR-01 candidate protocol | Analysis only; no executable proof | Architecture V1 §6 |

## 2. Delivered in R4

None of these is a command. Each answers BLOCKED while its decisions are open.

| Component | Package | Kind | What it decides | Never SUCCESS because |
|---|---|---|---|---|
| Union-of-links dependency classifier | `internal/cases/api` (`ClassifyDependencies`) | Published pure contract | §8 eligibility class of the union of all valid links | It is a guard, not consent: actor / row / scope / handoff / reset policy still apply |
| Shared gate result | `internal/cases/api` (`GateEvaluation`, `GateBuilder`) | Published pure contract | Status precedence and decision naming | — |
| Invoice claim gate | `internal/cases/claimgate` | Evaluator (pure) | Actor = verified holder; version; one open claim; no return in progress; dependencies; customer identity; lineage | ADR-01 |
| Invoice issuance gate | `internal/billing/issuance` | Evaluator (pure) | Active, unexpired claim of the same seller and case; idempotency; duplicate issuance; immutable attribution; exact positive money; D-28 | ADR-01 (+ D-28 when partial assessment) |
| Return / reset / release gate | `internal/custody/release` | Evaluator (pure) | Actor = verified holder; version; no active claim (no issuance race); union dependencies; lineage | ADR-01, F01 (+ OPD-03 for reset / release) |
| Race harness of the ADR-01 candidate | `internal/integration/legacy/shadow/r4_sales_race_test.go` | Test only, synthetic schemas | 15 concurrency / failure scenarios ([r4-inventory-race-safety.md](r4-inventory-race-safety.md)) | It is evidence for the decision, not approval |

**Architecture guard:** rule `r4-evaluator-wiring` keeps `claimgate`, `issuance`, `release`, `finance/posting`
and `finance/refund` importable only by tests. The CI step "R4 boundary" repeats the check.

## 3. Invariants and where they are enforced

| Invariant | Enforcement |
|---|---|
| One authoritative writer per operation | Each gate belongs to its owner module (cases claim, billing issue, custody return); no cross-module transaction is assumed (harness: one transaction per module) |
| No issuance while a return / reset / claim is in progress | `claimgate` (return in progress → CONFLICT); `release` (active claim → CONFLICT); harness #2 |
| No double assignment or duplicate invoice | `claimgate` (open claim → CONFLICT); `issuance` (`IssuedForClaim` → CONFLICT); harness #1, #3 (unique constraints proven necessary by mutation) |
| Original seller attribution immutable | `issuance` (snapshot ≠ proposed → CONFLICT); harness #13 + global invariant |
| Historical links not erased | Gates never write; harness history is append-only (trigger), #13 |
| Cancelled unpaid / paid cases never auto-released | `ClassifyDependencies` (CANCELLED → BLOCKED, OPD-03 / OPD-04); harness #10 |
| Customer identity never from similarity alone | `GateBuilder.Identity` (DISCOVERY_ONLY → UNKNOWN); `ClassifyDependencies` ignores similarity links as proof |
| Ambiguous lineage stays UNKNOWN | Lineage AMBIGUOUS / UNKNOWN → UNKNOWN, CONFLICT → CONFLICT; harness #7 |
| All linked records considered before release | Union classification; incomplete coverage → FAIL_CLOSED (UNKNOWN); harness #8, #9 |

## 4. Gate outcome vocabulary

Status precedence:

1. **UNAUTHORIZED**: actor or row scope.
2. **CONFLICT**: fresh facts contradict the intent.
3. **UNKNOWN**: facts cannot be established.
4. **FAILED**: the request itself is invalid.
5. **BLOCKED**: policy.

`PreconditionsMet = true` means only decision blockers remain; the gate still never executes. Reason
codes are the frozen foundation-v1 codes. Decisions are named in `BlockedBy`.

## 5. What remains BLOCKED

- `IssueInvoice`, `ClaimCaseForInvoice`, `ReturnCase`, `Reset`, `Release`: no command, no route, no flag.
- Approval needs ADR-01 to be accepted (or a specific multi-module transaction ADR), plus F01 and OPD-03.
- After approval, still required:
  - commands that reuse these gates inside owner transactions;
  - owner-schema migrations for claims and reservations;
  - outbox / inbox wiring;
  - write-back for legacy coexistence;
  - real-data parity.
- **P0-5: PARTIAL.** See [r4-decision-gates.md](r4-decision-gates.md).
