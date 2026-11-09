# R4 — Fault-injection and test evidence

**What these tests are:**
- **Data:** synthetic only, on disposable PostgreSQL databases.
- **Environment:** no production, legacy, gateway, wallet or Biawin system is reached.
- **Pass status:** a passing test is engineering evidence, not approval of any decision.

| Area | Test | Fault / condition | Expected | Result |
|---|---|---|---|---|
| Concurrency | `TestR4SalesRaceHarness/01` | 10 concurrent issuances on one claim, forced interleaving | 1 invoice | PASS |
| Concurrency | `/02` | Claim vs return, 25 rounds; issuance vs return | Exactly one winner | PASS |
| Concurrency | `/03` | Two sellers; double claim | One claim | PASS |
| Idempotency | `/04`, `/05` | Replay; key reuse with a different payload | Same result; CONFLICT | PASS |
| Concurrency | `/06` | Stale version | CONFLICT | PASS |
| Linkage | `/07`, `/08` | Ambiguous / conflicting / missing links | UNKNOWN / CONFLICT; nothing written | PASS |
| Financial dependency | `/09`, `/10` | Paid / cancelled invoice | BLOCKED (OPD-03 / OPD-04) | PASS |
| Crash | `/11` | Failure after the invoice write, before commit | Full rollback; retry once | PASS |
| Replay | `/12` | Relay crash after the cases commit; redelivery ×2 | Confirmed once | PASS |
| History | `/13` | History UPDATE / DELETE | Refused (append-only) | PASS |
| Authorization | `/14` | Unverified actor; other seller | UNAUTHORIZED | PASS |
| Cross-module failure | `/15` | Billing never commits; late confirmation after expiry | Reconciliation, never release | PASS |
| Mutation | harness without the row lock and unique constraints | Protections removed | Harness FAILS (10 invoices, double winners, invariant broken) | DETECTED |
| Maker / checker | `TestR4FinanceDecisionContractOverReviewRead` | Submitter as reviewer | UNAUTHORIZED | PASS |
| Evidence integrity | same | Wrong receipt digest; stale version | CONFLICT | PASS |
| Read purity | same | Evaluate then re-read | Attempt unchanged | PASS |
| Duplicate economic effect | `TestPostingContractOnPostgres` | 20 concurrent identical posting requests | 1 row, 19 replays, net once | PASS |
| Lost ack | same | Commit acknowledgement lost | UNKNOWN, then reconciled replay | PASS |
| Reversal | same | Reversal; second reversal; UPDATE / DELETE | Separate row; CONFLICT; refused | PASS |
| Test sensitivity | posting store defect during development | Line written before the key check | Detected (`net 2000`) | DETECTED, fixed |
| Refund evidence | `TestRefundGuards` | Cancellation / HTTP-success as proof; amount mismatch | Refused | PASS |
| Commission duplicates | `TestReconcileEngines` | Two engines / one engine twice / shadow claims posted | Classified | PASS |
| Wiring | `TestViolatingFixtureFails` (rule `r4-evaluator-wiring`) | Production import of an evaluator; stray `EvaluateDecision` | Lint fires (2) | PASS |
| Feature flags | existing Slice 4–14 flag tests; R4 adds no flag | — | All write flags OFF by default | PASS (unchanged) |
| Production adapters | lint `s12-gateway-wiring`, `biawin-provider-call`, `r2-network-call`; R4 boundary CI | Any real adapter wiring | None | PASS |

## Commands

```sh
go test -race ./internal/cases/api/ ./internal/cases/claimgate/ ./internal/billing/issuance/ ./internal/custody/release/ \
  ./internal/finance/review/ ./internal/finance/posting/ ./internal/finance/refund/ ./internal/commission/domain/
CRM_TEST_DATABASE_URL=... go test -race -run 'TestR4' ./internal/integration/legacy/shadow/
CRM_TEST_DATABASE_URL=... go test -race ./internal/finance/posting/
```

## Not verified

- **Real legacy coexistence and real-data parity:** not verified. No real customer data was used
  (ADR #18).
- **Production runtime:** not verified.
- **Real roles:** HR and Finance role behaviour is not live-verified (AGENTS.md runtime rule).
