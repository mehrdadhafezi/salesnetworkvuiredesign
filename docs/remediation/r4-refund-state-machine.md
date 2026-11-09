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
| any | A verified actor; the preserved original invoice and payment references; money facts validated by the evaluator itself (see **Monetary invariants** below) | See below |
| → REVIEWED | Reviewer account ≠ requester account, **and** person-level independence | Same account or same human → UNAUTHORIZED; no person linkage → BLOCKED by `IDENTITY_PERSON_LINK_UNMODELLED` |
| → AUTHORIZED | Authorizer account ≠ requester and reviewer accounts, **and** person-level independence | Same account or same human → UNAUTHORIZED; no person linkage → BLOCKED by `IDENTITY_PERSON_LINK_UNMODELLED` |
| → EXTERNALLY_EXECUTED / EVIDENCE_RECEIVED | External proof (bank statement or provider confirmation) with a reference, document SHA-256 and an amount exactly equal | **HTTP success** → FAILED; **invoice cancellation** → FAILED; amount or unit differs → CONFLICT |
| → RECONCILED | A reconciliation record with a document digest | → FAILED |
| → UNKNOWN | — | reported as `OUTCOME_UNKNOWN` |

**Separation of duties.** Distinct account ids are necessary but never prove different people: one
person may hold several accounts. Identity has no authoritative person linkage today, so review and
authorization fail closed with `IDENTITY_PERSON_LINK_UNMODELLED`. The finance decision contract does the
same. A same-human link is UNAUTHORIZED.

## Monetary invariants (PR #44 review, P2-A)

| Fact | Rule | Violation |
|---|---|---|
| `Amount` | Valid, positive, explicit unit | FAILED |
| `OriginalPaid` | Valid, nonnegative, explicit unit | FAILED |
| `PreviouslyRefunded` | Valid, nonnegative, explicit unit | FAILED (a negative value can no longer inflate the refundable amount) |
| Units | `OriginalPaid`, `PreviouslyRefunded` and `Amount` share one unit; no conversion | FAILED |
| `PreviouslyRefunded` ≤ `OriginalPaid` | Consistent facts | CONFLICT |
| `Amount` ≤ `OriginalPaid` − `PreviouslyRefunded` | Remaining refundable, exact int64 | CONFLICT |
| Overflow | Cannot arise once the rules above hold; any arithmetic error fails closed | FAILED |

**Never inferred:**
- an invoice cancellation is not money returned;
- an HTTP 2xx is not a refund;
- compensation is never done by deleting or rewriting ledger, invoice or payment records.

## Read projection

`Project` derives the current state from the append-only history (PR #44 review, P2-B).

**Sequence contract.** This is an engineering contract of the event log, not a business policy. No
authority specified one, so it is an explicit choice:
- the events of one refund carry a per-refund sequence starting at 1 and increasing by exactly 1 (1..n);
- this mirrors the per-aggregate versions every target aggregate already uses;
- strictly increasing but gapped sequences are rejected, because a gap cannot be told apart from a lost
  or hidden event, so it must fail closed;
- delivery order does not matter: events are ordered by their sequence.

**Invalid history:** any of the following makes the history invalid, and an invalid history yields **no
current state** (`State` empty):
- empty history;
- an event of another refund;
- a sequence below 1;
- a duplicate or missing sequence;
- a first event other than REQUESTED;
- an illegal transition.

Nothing is skipped, repaired or manufactured, and the input is never modified. `VerifiedState` /
`VerifiedThrough` report the longest valid prefix for diagnosis only.

## Tests (`refund_test.go`, `refund_integrity_test.go`)

- all 11 legal transitions are BLOCKED by OPD-04;
- cancellation and HTTP-success evidence are refused;
- amount or unit mismatch → CONFLICT;
- skipping states → CONFLICT;
- no UNKNOWN → AUTHORIZED;
- maker / checker for review and authorization (account level), person-level independence (same human →
  UNAUTHORIZED; no linkage → BLOCKED);
- monetary invariants: negative, invalid or mismatched paid / refunded facts, refunded > paid, amount >
  remaining, boundaries (`TestRefundMonetaryInvariants`);
- history integrity: empty, duplicate, gap, zero / negative sequence, wrong start, foreign event,
  illegal step, input not mutated, verified prefix (`TestProjectionHistoryIntegrity`,
  `TestProjectionVerifiedPrefix`);
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
