# R2 — Failure and recovery

All results come from deterministic tests on PostgreSQL with:

- a temporary-directory object store;
- the synthetic legacy store;
- the synthetic Biawin port;
- an in-process TLS fake implementing the proposed compat contract.

No real legacy, financial, Biawin or gateway system was contacted.

## Fault-injection matrix

| # | Scenario | Test | Database state asserted | Outcome / recovery |
|---|---|---|---|---|
| 1 | Restart while idle | `worker.TestGracefulShutdownAndConcurrency`; `crm-worker.TestRunLifecycle` | Jobs `stopped`; nothing claimed | Restart resumes from PostgreSQL |
| 2 | Restart during claim | `relay.TestStaleClaimIsRecoverable` | Held row skipped by a second relay; after the crash the row is claimable | Claim lease = transaction; released on disconnect |
| 3 | Crash after external success, before acknowledgement | `TestR2WorkerFaultInjection` (`TimeoutAfterCommit`) | Command UNKNOWN, then APPLIED:MATCH after observation | Exactly one legacy effect (calls asserted) |
| 4 | Crash before the external request | `TestR2WorkerFaultInjection` (`panicConn`) | Write-ahead UNKNOWN; legacy calls unchanged | Observe ABSENT → PENDING → applied once |
| 5 | Same event delivered twice | `relay.TestAtomicAppendAndDispatchAfterAcceptance`; `TestR2WorkerFaultInjection` | Inbox returns duplicate; effect counter 1 | No duplicate effect |
| 6 | Two workers polling concurrently | `relay.TestConcurrentRelaysDeliverEachEventOnce` (4 relays, 12 events, non-idempotent effect) | No overlapping delivery; each effect applied once | — |
| 7 | Database unavailable | `TestR2EvidenceFaults` (closed pool) | Pass `failing`; no success timestamp; readiness 503 | Nothing reported done |
| 8 | Consumer unavailable | `TestR2WorkerFaultInjection` | Source event `failed`, visible; redelivered after backoff | — |
| 9 | Legacy timeout before acknowledgement | `compat.TestUnprovenAnswersAreUnknown/timeout` | Exchange verdict TIMEOUT | `ErrOutcomeUnknown` → UNKNOWN |
| 10 | Invalid response schema | `compat` cases: unknown field, trailing data, applied without ref, unknown outcome / reason | Verdict INVALID_SCHEMA | UNKNOWN, never success or rejection |
| 11 | Business rejection | `compat.TestAppliedAndRejected` (`STATE_CONFLICT`) | Verdict VALID, outcome REJECTED | Proven non-commit → REJECTED with reconciliation |
| 12 | Network recovery | `compat.TestNetworkRecoveryAndIdempotentToken` | Two exchanges recorded | Lost answer UNKNOWN; observation finds the single effect |
| 13 | Provider UNKNOWN | `biawin.TestProviderReconcileJob` | UNKNOWN → CONFIRMED / CONFIRMED_ABSENT / RECONCILIATION_REQUIRED | Never redispatched (`Calls` asserted) |
| 14 | Provider duplicate callback | `biawin.TestProviderReconcileJob` | Dispositions ALREADY_FINAL, DUPLICATE, CONFLICT | Conflict visible in metrics |
| 15 | Receipt copied but state not updated | `TestR2WorkerFaultInjection` (`lostReply`) | STORAGE_ERROR → READY; 2 evidence objects; RECOVERED reconciliation | No second copy |
| 16 | Receipt hash mismatch | `TestR2EvidenceFaults` | NOT_RECOVERABLE; attempt evidence NOT_RECOVERABLE; attempt still PENDING_REVIEW | Finance review; payment not approved |
| 17 | Upload expiry racing binding | Library: `TestBindingVersusExpiry`, `TestUploadRevivalRace`. Worker: `TestR2EvidenceFaults` (report mode) | Report mode changes nothing; bytes kept | Purge BLOCKED by D-44 |
| 18 | Exhausted retry budget | `relay.TestFailuresVisibleBoundedAndNoSilentSkip` (10); `TestR2EvidenceFaults` (20 claims); write-back observation budget (3 in test, 12 default) | `failed:10` exhausted; STORAGE_ERROR at 20; RECONCILIATION_REQUIRED | Visible; no further automatic attempt |
| 19 | Feature flag off while jobs are queued | `TestR2WorkerFaultInjection`; `crm-worker.TestRunLifecycle` | Commands stay PENDING; legacy calls 0; job `paused` | Resumes when switched on |
| 20 | Cancellation and graceful shutdown | `worker.TestGracefulShutdownAndConcurrency` | In-flight pass completes within grace, else is cancelled | — |
| 21 | Malformed or missing adapter credentials | `compat.TestNotApprovedAndConfig` | Construction refused | — |
| 22 | Wrong environment configuration | `crm-worker.TestConfigFailsClosed`, `TestAttestationAndCheck`, `TestModuleLoginsBoundToAttestedDatabase` | Startup refused: undeclared or inferred environment, attestation mismatch, ambiguous attestation, or a module login reaching a database other than the attested one | — |
| 23 | No cross-module direct DB writes | Lint `foreign-schema-write`, `r2-worker-sql`; `db.Runner` owner-role isolation | Worker binary holds no SQL; each pool logs in to one owner | — |
| 24 | No forbidden financial side effects | `TestR2WorkerFaultInjection` | No attempt leaves PENDING_REVIEW; `finance.ledger_entry` empty; legacy payments stay `pending`; financial type excluded from delivery | — |
| 25 | Evidence retained across recovery | `TestR2WorkerFaultInjection` | Attempt rows per command (2 and 4); service actor only; every outbox event dispatched | Append-only triggers prevent deletion |

## Recovery procedures (operator view)

| Symptom (metric) | Meaning | Action |
|---|---|---|
| `outbox_failed_retryable` > 0 | Consumer failing | Read `last_error` in the source outbox. Fix the consumer. Events retry with backoff. |
| `outbox_exhausted` > 0 | 10 failures | Root-cause first. A re-arm is a reviewed manual operation; no tool is provided in R2. |
| `receipt_storage_error` > 0 | Storage failing | Check the evidence volume. Retries continue up to 20 claims. |
| `receipt_retry_exhausted` > 0 | Storage failed 20 times | Finance and ops review the reconciliation record. |
| `receipt_not_recoverable` > 0 | Bytes missing or corrupt | Finance review with the uploader. Never auto-resolved. |
| `writeback_unknown` > 0 | Outcome unproven | Wait for observations. Never re-apply by hand. |
| `writeback_reconciliation_required` > 0 | Owner review | Domain owner compares legacy evidence. Resolution is recorded outside the worker (no tool in R2). |
| `provider_reconciliation_required` > 0 | Lookup unavailable | Human `Resolve` with evidence through the existing authorised HTTP route. |
| `job_state_blocked` = 1 | Prerequisite missing | Expected for `writeback_deliver` and `provider_reconcile` (E-3, adapter). |
| `job_state_failing` = 1 | Pass errors | Read logs by `run_id`. Fix the cause. Backoff caps at 10 min. |

## Proposed alert conditions (engineering defaults; not approved SLA / RTO)

| Condition | Trigger |
|---|---|
| Relay lag | `outbox_oldest_pending_age_seconds` > 900 for 15 min |
| Delivery failing | `outbox_exhausted` > 0, or `relay_no_route_total` increasing |
| Evidence | `receipt_oldest_unfinalised_age_seconds` > 3600; `receipt_not_recoverable` > 0 |
| Write-back | `writeback_unknown` > 0 for 1 h; `writeback_reconciliation_required` > 0; `writeback_parity_conflicts` > 0 |
| Provider | `provider_unknown` > 0 for 1 h; `provider_conflicting_callbacks` > 0 |
| Process health | readiness 503 for 5 min; `job_state_failing` = 1 for 3 passes; `job_timeouts_total` increasing |
