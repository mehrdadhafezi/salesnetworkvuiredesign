# R4 — Inventory race safety (synthetic harness of the ADR-01 candidate)

**This is evidence for a decision, not the decision.** The harness simulates the ADR-01 *candidate*
reservation protocol (Architecture V1 §6). Passing it does not approve ADR-01 or F01, and it activates
nothing.

## Setup

- **Test:** `TestR4SalesRaceHarness` (`internal/integration/legacy/shadow/r4_sales_race_test.go`).
- **Database:** a disposable PostgreSQL database per run (`pgtest.CreateDatabase`, dropped at cleanup).
- **Synthetic schemas:** `syn_claim` (stands for cases), `syn_invoice` (billing), `syn_hold` (custody).
  - No target module table is touched.
  - No migration is added.
- **Transactions:** one per module. There is no cross-module transaction (none is approved).
- **Decisions:** every step asks the real R4 evaluators (`claimgate`, `issuance`, `release`) and proceeds
  only when `PreconditionsMet`, i.e. only the open decisions block it.
- **Forced interleaving:** a hook pauses after evaluation and before any write. Every concurrent
  transaction therefore evaluates before any of them writes, so the database constraints and locks, not
  timing luck, must prevent the race.
- **Data:** synthetic only. No production database, customer data, gateway or legacy write.

**Protocol under test (candidate):**
1. **Claim** (cases transaction): lock the case row `FOR UPDATE`, run the claim gate, insert the ACTIVE
   claim, bump the case version, append history. A partial unique index allows at most one ACTIVE /
   RECONCILIATION_REQUIRED claim per case.
2. **Issue** (billing transaction):
   - insert-first idempotency record;
   - synchronous read of the claim;
   - issuance gate;
   - invoice with `UNIQUE (claim_id)`;
   - outbox `InvoiceIssued`.
3. **Confirm** (cases transaction): an inbox-deduplicated relay sets ACTIVE → CONFIRMED. An already
   expired claim records `LATE_CONFIRMATION` and stays in reconciliation.
4. **Expiry:** ACTIVE past TTL → RECONCILIATION_REQUIRED. It is never released or deleted.
5. **Return reservation** (cases transaction): the same row lock, then the release gate, `return_state`
   IN_PROGRESS and a version bump. The custody move follows in custody's own transaction.

## Results (local, PostgreSQL 16, race detector on; CI runs the same on PostgreSQL 18)

| # | Scenario | Expected | Actual | Result |
|---|---|---|---|---|
| 1 | Simultaneous invoice issuance (10 concurrent, distinct keys, one claim) | 1 SUCCESS, 9 CONFLICT, 1 invoice | as expected | PASS |
| 2 | Issuance racing case return (25 rounds), then return vs issuance on a claimed case | Exactly one of claim / return wins; return during issuance → CONFLICT | as expected | PASS |
| 3 | Two sellers claiming; one seller, two concurrent claims | Holder SUCCESS, other UNAUTHORIZED; one claim recorded | as expected | PASS |
| 4 | Duplicate command replay | Same invoice id, one invoice | as expected | PASS |
| 5 | Reused idempotency key with a different payload | CONFLICT, original invoice only | as expected | PASS |
| 6 | Stale aggregate version (claim and return) | CONFLICT | as expected | PASS |
| 7 | Ambiguous / conflicting case-source links | UNKNOWN / CONFLICT; nothing written | as expected | PASS |
| 8 | Missing invoice dependency (link target missing) | Return UNKNOWN (fail closed) | as expected | PASS |
| 9 | Paid invoice return attempt | BLOCKED; holder unchanged | as expected | PASS |
| 10 | Cancelled invoice release / reset / return | BLOCKED (OPD-03 / OPD-04); nothing changed | as expected | PASS |
| 11 | Interrupted transaction (failure after the invoice write, before commit) | No invoice, outbox or idempotency row; claim ACTIVE; retry succeeds once | as expected | PASS |
| 12 | Restart and replay (relay crashes after the cases commit, then redelivers twice) | CONFIRMED exactly once; the return is then BLOCKED by the committed invoice | as expected | PASS |
| 13 | Historic attribution preservation | Invoice carries the original seller; history UPDATE / DELETE refused | as expected | PASS |
| 14 | Incorrect actor scope (unverified, other seller issuing or returning) | UNAUTHORIZED | as expected | PASS |
| 15 | Cross-module consistency under failure | See below | as expected | PASS |

Scenario 15 in detail:
- Billing never commits: the claim expires into RECONCILIATION_REQUIRED, the return CONFLICTs and
  issuance on the expired claim CONFLICTs.
- Billing commits, then the claim expires before delivery: `LATE_CONFIRMATION` is recorded, the claim
  stays in reconciliation, and nothing is released.

Global invariants are checked after all scenarios: no orphan invoice; never an open claim together with
a return; attribution intact.

## Mutation check (the harness detects real races)

Removing the protections makes the harness fail, which proves it is not vacuous:

```
mutation: no case-row FOR UPDATE, claim index not unique, invoice.claim_id not unique
  01 simultaneous issuance: map[SUCCESS:10]
  02 exactly one of claim (SUCCESS) and return (SUCCESS) must win
  03 double claim: SUCCESS / SUCCESS
  invariants: 1 cases with both a return and an invoice claim
```

## What the harness does NOT prove

- **Real legacy coexistence:** the legacy writers still run, and there is no legacy-side disable (P1-8).
- **Real-data link coverage:** the union-of-links sources are not yet produced from real staging data
  for every proof source of §8 (link adapters are future work).
- **TTL value, reconciliation owner and late-confirmation handling:** these are policy, decided with
  ADR-01 and F01.
- **Production runtime:** NOT VERIFIED.
