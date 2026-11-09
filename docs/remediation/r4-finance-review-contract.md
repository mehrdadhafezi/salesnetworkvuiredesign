# R4 — Finance review contract

**No Finance approval or rejection exists or is exposed.** R4 adds the decision *contract*,
`review.EvaluateDecision` in `internal/finance/review/contract.go`. It extends the Slice 12 gate
(`review.Evaluate`), and the Slice 12 gate is always evaluated first, never bypassed.

OPD-08, D-43, D-41 and OPD-09 remain OPEN, so the answer is never SUCCESS.

## Checks (all evaluated; status precedence UNAUTHORIZED > CONFLICT > UNKNOWN > FAILED > BLOCKED)

| # | Requirement | Check | Outcome when violated |
|---|---|---|---|
| 1 | Slice 12 guards | `Evaluate`: maker ≠ checker (fail closed, empty reviewer = maker); D-43 evidence READY; reviewable state | UNAUTHORIZED (maker); BLOCKED by D-43 |
| 2 | Verified actor identity | `Reviewer.Verified` must come from a verified session (server-derived) | UNAUTHORIZED |
| 3 | Independence by actual human | Account ids differ (Slice 12), and `PersonLink`: SAME_HUMAN → UNAUTHORIZED; NOT_MODELLED → BLOCKED by `IDENTITY_PERSON_LINK_UNMODELLED` | UNAUTHORIZED / BLOCKED |
| 4 | Explicit capability | The reviewer must hold exactly the action capability. **None is defined or granted** (no new grants). | BLOCKED (`NO_DECISION_CAPABILITY`) |
| 5 | Record scope | `InRecordScope` | UNAUTHORIZED |
| 6 | Immutable evidence references | Every receipt cited by its recorded SHA-256; nothing uncited or foreign | CONFLICT (digest or foreign); FAILED (uncited, none) |
| 7 | READY receipts | D-43 safe default via `Evaluate` | BLOCKED by D-43 |
| 8 | Stale version | `ExpectedVersion` = attempt `Version` | CONFLICT |
| 9 | Not reviewable | attempt not `PENDING_REVIEW` | CONFLICT |
| 10 | Duplicate review | `PriorDecision` present | CONFLICT |
| 11 | Structured rejection reason | Code + bounded detail (≤ 1000 runes). The approved code vocabulary is **empty**, so any code → BLOCKED by `REJECTION_REASON_VOCABULARY_UNDEFINED` | FAILED / BLOCKED |
| 12 | Append-only audit evidence | `EvaluationSHA256`: canonical digest of the request and the answer, for an append-only audit record of the evaluation (not a decision) | — |
| 13 | Policy dependencies | Approve: D-41, OPD-08, OPD-09. Reject: OPD-08, `BILLING_REJECTION_TRANSITION_UNDEFINED`. Both: `NO_DECISION_CAPABILITY` | BLOCKED |

## Published read extension (additive, read-only)

The payments review read now also carries:
- the attempt `Version`;
- each receipt's `ContentSHA256` and `EvidenceRef`.

Details:
- These fields exist in the payments schema already; no migration was needed.
- The HTTP / OpenAPI response is unchanged: the fields are used by the contract only.

## Tests

- **Unit** (`contract_test.go`):
  - a hypothetical fully eligible reviewer is still BLOCKED;
  - all guards in the table above;
  - rejection structure;
  - digest determinism and sensitivity.
- **PostgreSQL** (`TestR4FinanceDecisionContractOverReviewRead`): the contract runs over the real
  payments review read, using a synthetic submission finalised to READY.
  - the read carries the version and the uploaded digest;
  - a READY receipt carries its content-addressed reference;
  - an independent reviewer gets BLOCKED with `PreconditionsMet`;
  - a maker gets UNAUTHORIZED;
  - a stale version or a wrong digest gets CONFLICT;
  - the attempt is unchanged afterwards.

## Known gaps (not invented here)

- **Person-level identity:** no model links several accounts of one human. Independence is proven only
  at the account level; activation stays BLOCKED by `IDENTITY_PERSON_LINK_UNMODELLED`.
- **Rejection:** no rejection vocabulary and no billing rejection transition exist. OPD-08 and the
  billing owner must define them.
- **Approval effects:** approval posts to the ledger in the same transaction (Arch §16). That is blocked
  by D-41 and the posting contract in [r4-ledger-posting-contract.md](r4-ledger-posting-contract.md).
- **No decision route:** none exists. Lint `s12-forbidden-command`, `finance-write-route` and
  `r4-evaluator-wiring` keep it that way, and so does the "R4 boundary" CI step.
