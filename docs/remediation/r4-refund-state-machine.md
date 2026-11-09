# R4 — Refund state machine

**No refund can be executed or confirmed.** `internal/finance/refund` models the refund lifecycle
(Gate-0 INV-015; Cross-Role §9) as a *contract*.
- **Always BLOCKED:** every structurally sound transition is BLOCKED by OPD-04, `NO_REFUND_AUTHORITY`
  and `FINANCIAL_COMPENSATION_UNAPPROVED`.
- **Not wired:** there is no store, route or command; lint `r4-evaluator-wiring` keeps the package
  test-only.

## States (all distinct; none implies another)

`REQUESTED`, `REVIEWED`, `AUTHORIZED`, `EXTERNALLY_EXECUTED`, `EVIDENCE_RECEIVED`, `RECONCILED`,
`COMPLETED`, `REJECTED`, `UNKNOWN`.

```
REQUESTED ──▶ REVIEWED ──▶ AUTHORIZED ──▶ EXTERNALLY_EXECUTED ──▶ EVIDENCE_RECEIVED ──▶ RECONCILED ──▶ COMPLETED
    │             │             │                  │                       ▲
    ▼             ▼             ▼                  ▼                       │
 REJECTED      REJECTED      UNKNOWN ◀─────────── UNKNOWN ─────────────────┘  (resolved only by evidence)
```

- There are 11 structural transitions.
- COMPLETED and REJECTED are terminal.
- There is **no path from UNKNOWN back to AUTHORIZED**: no blind re-execution.

## Evidence rules

| Transition | Requires | Refused |
|---|---|---|
| any | A verified actor; the preserved original invoice and payment references; a positive exact amount ≤ proven refundable (paid − previously refunded, same unit) | Above refundable → CONFLICT; zero, missing unit or unit mismatch → FAILED |
| → REVIEWED | Reviewer ≠ requester | Requester → UNAUTHORIZED |
| → AUTHORIZED | Authorizer ≠ requester and ≠ reviewer | → UNAUTHORIZED |
| → EXTERNALLY_EXECUTED / EVIDENCE_RECEIVED | External proof (bank statement or provider confirmation) with a reference, document SHA-256 and an amount exactly equal | **HTTP success** → FAILED; **invoice cancellation** → FAILED; amount or unit differs → CONFLICT |
| → RECONCILED | A reconciliation record with a document digest | → FAILED |
| → UNKNOWN | — | reported as `OUTCOME_UNKNOWN` |

**Never inferred:**
- an invoice cancellation is not money returned;
- an HTTP 2xx is not a refund;
- compensation is never done by deleting or rewriting ledger, invoice or payment records.

## Read projection

`Project` derives the current state from the append-only history, ordered by sequence:
- the history must start with REQUESTED;
- an illegal step stops the projection at the last valid state with a finding;
- events of another refund are findings;
- nothing is repaired or skipped.

## Tests (`refund_test.go`)

- all 11 legal transitions are BLOCKED by OPD-04;
- cancellation and HTTP-success evidence are refused;
- amount or unit mismatch → CONFLICT;
- skipping states → CONFLICT;
- no UNKNOWN → AUTHORIZED;
- maker / checker for review and authorization;
- an unverified actor is refused;
- amounts above refundable, zero, and a missing original are refused;
- reconciliation without a record is refused;
- terminal states have no transitions;
- projection: ordering, illegal steps, wrong start and foreign events.

## Open (OPD-04)

- the refund object (whole invoice, stage or payment);
- the proof standard;
- who requests, reviews and authorizes;
- the execution channel;
- the reconciliation owner;
- the partial-refund policy;
- how refunds post (D-41).

Financial compensation needs its own formal approval.
