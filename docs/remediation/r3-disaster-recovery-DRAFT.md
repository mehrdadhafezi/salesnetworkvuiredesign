# R3 — Disaster recovery plan (DRAFT, not approved)

This updates `docs/cutover/rollback-and-recovery-DRAFT.md` (Slice 15) with what R1–R3 implemented and
rehearsed in a **disposable local staging** environment. It is not approved, it does not define
organisational RPO or RTO, and it authorises nothing.

**The central limitation.** Turning a feature flag OFF, or rolling back executables, stops NEW
operations. It does **not** undo committed business operations:
- target rows, outbox events and write-back commands remain;
- evidence remains;
- anything already sent to an external system remains.

There is no financial compensation command and no external cancellation mechanism. Neither is invented
here.

## Recovery classes

| Class | Meaning | Prerequisites | Operator actions | Evidence | Known limitations |
|---|---|---|---|---|---|
| **A. Application rollback** | Restart the previous release's executables | Previous images retained (`stage.sh` keeps the previous tag); schema compatible (additive migrations) | `stage.sh rollback` → `stage.sh verify` → `stage.sh evidence` | Events JSONL; `crm_build_info` revision | Does not revert the database, evidence or business effects. Rehearsed (10 s). |
| **B. Infrastructure rollback** | Restore container / network / config definitions | Versioned `deploy/` in git; secrets regenerated per environment | Redeploy the previous commit's `deploy/` with `stage.sh up` | Preflight output | No IaC for real hosts (BLOCKED-INFRA). |
| **C. Database restore** | Logical restore into a NEW database, then repoint services | A verified backup; age identity; a cluster with the owner roles | `crm-backup verify` → `restore` (new `crm_restore_*`, quarantined) → review report → re-attest → repoint logins → enable jobs one by one | Restore report (differences, evidence findings, timings) | Logical only, so data after the snapshot is lost; no PITR. Repointing the live service at a restored database is NOT automated and NOT rehearsed. |
| **D. Evidence-store recovery** | Restore objects with the same database snapshot | The backup includes the evidence archive | Part of `restore`; every receipt and upload checked against its bytes | Evidence findings | MISSING or CORRUPT objects fail the restore; affected receipts become NOT_RECOVERABLE and need business re-collection. |
| **E. Business-effect compensation** | Correct committed business outcomes | Owner decision per case | Through existing authorised domain commands only | Domain history | No compensation, refund, reversal or correction command exists (not invented). |
| **F. External-provider reconciliation** | Resolve UNKNOWN external outcomes | Verified adapter (none) or human evidence | Lookup (blocked) or the authorised human resolution route with evidence | `provider_operation_event`; write-back attempts | Real write-back (E-3) and Biawin lookups are BLOCKED; UNKNOWN stays visible until resolved by evidence. |

## Failure classes (updated)

| Failure | Detection (R3) | Containment | Recovery | Class | Rehearsed |
|---|---|---|---|---|---|
| API process crash | `CRMApiDown`; container health | Restart policy | Automatic restart | A | Drill #1 (3.8 s) |
| Worker crash | `CRMWorkerDown`; health | Restart policy | Durable claims resume (R2) | A | Drill #2 (3.8 s); R2 tests |
| Database outage | `CRMDatabaseUnavailable` (`check_up=0`, ready 503) | Jobs fail closed | Restore availability; jobs resume | B / C | Drill #3 (9.7 s) |
| Wrong environment / database | `CRMAttestationMismatch`; startup refusal | Worker refuses to start | Fix configuration / attestation | B | Drill #4; binding test |
| Failed migration | `crm-migrate` non-zero; services not started | Deployment stops | Restore into a new database, then fix forward (no down-migrations) | B / C | Drill #13 |
| Bad release | `stage.sh up` fails the health wait | Previous containers keep running where possible | `stage.sh rollback` | A | Drills #14, #15 |
| Missing secret | Startup refusal naming the variable | — | Provide the secret file | B | Drill #16 |
| Backup failure / corruption | `CRMBackupFailed`; `verify` refuses | Keep the previous backups | New backup to a new destination | C | Go tests |
| Evidence loss | Restore FAILED with MISSING / CORRUPT; `CRMReceiptNotRecoverable` | Never mark READY | Business re-collection | D / E | Go tests |
| Queue backlog / exhausted delivery | `CRMQueueBacklog`, `CRMDeliveryExhausted` | Fix the consumer | Backoff redelivery; reviewed re-arm (no tool) | A | R2 tests |
| UNKNOWN external outcome | `CRMUnknownIncreasing`, `CRMReconciliationRequired` | No blind retry | Observation / lookup / human evidence | F | R2 tests; restore preserves UNKNOWN |
| Unexpected flag activation | `CRMUnexpectedFlagActivation` | Switch OFF if unapproved | Reconcile commands executed in the window | A + E | Drill #18 (condition only) |
| Monitoring down | `CRMWatchdog` absence (requires a pipeline) | — | Restore the pipeline | B | PARTIAL (drill #17) |

## Proposed RPO / RTO decision matrix (targets OPEN)

| Option | RPO (data loss bound) | RTO (time to service) | Needs | Measured basis today |
|---|---|---|---|---|
| Logical backup every 24 h | ≤ 24 h | Restore time + repoint + validation | Scheduler, off-site copy, monitoring | Restore of a small synthetic database: 2.5 s (not representative) |
| Logical backup every 1 h | ≤ 1 h | as above | as above, plus capacity | — |
| Physical backup + WAL archiving (PITR) | Minutes | Base-backup restore + WAL replay | WAL archive storage, PITR tooling, rehearsal | **Not implemented** |
| Synchronous replica + failover | ≈ 0 | Minutes | Replica infrastructure, failover procedure | **Not implemented** |

Organisational RPO and RTO remain **OPEN**: decisions for the business, operations and infrastructure
owners. No figure here is a target.

## Exercise record

See `r3-fault-injection-evidence.md` for all 20 scenarios: expected behaviour, actual result, measured
time, state, evidence, pass/fail and residual risk.
