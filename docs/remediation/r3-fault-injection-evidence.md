# R3 — Fault-injection evidence (disposable local staging and tests)

**Where these ran:**
- **Container drills:** `deploy/staging/drill.sh` against the running `crm-staging` project, release
  `efb76c8a6ad1`, on 2026-11-08. Raw JSON lines: `docs/remediation/evidence/r3-drill-20261108.jsonl`.
- **Go tests:** run locally against PostgreSQL 16, and in CI against PostgreSQL 18.

All data is synthetic. No production, legacy, gateway, wallet or Biawin system was contacted: the
internal network has no outbound route, and the adapters are disabled or synthetic.

**A simulated alert condition is not a delivered page.**

| # | Scenario | Expected | Actual | Time (s) | Database / object state | Audit / recovery evidence | Result | Residual risk |
|---|---|---|---|---|---|---|---|---|
| 1 | API process termination | Host-side SIGKILL; restart policy restarts it; healthy | healthy | 3.79 | unchanged | drill line 1 | PASS | In-flight requests are lost; clients retry idempotently |
| 2 | Worker termination | Restart; durable claims resume | healthy | 3.80 | unchanged; R2 claims durable | drill line 2; R2 `TestStaleClaimIsRecoverable` | PASS | Interrupted passes are redone |
| 3 | PostgreSQL unavailable | Worker ready 503 and `check_up=0` while down; recovers | during=503, check_up=0, api_live=200, after=200 | 9.66 | no writes while down | drill line 3 | PASS | `crm-api` readiness has no database check in this configuration (no surfaces opened) |
| 4 | Incorrect database environment | Worker declaring production on a staging-attested database refuses to start | exit 1 "attestation failed" | 0.66 | unchanged | drill line 4; `TestModuleLoginsBoundToAttestedDatabase` | PASS | Relies on attestation never being set wrongly by an operator |
| 5 | Backup interrupted | A `.partial` directory has no manifest, never verifies and blocks the name | refused | test | source unchanged | `TestR3BackupIntegrityAndEvidenceConsistency` | PASS | — |
| 6 | Backup corrupted | Flipped dump, truncated archive, tampered manifest or wrong key are refused | `ErrIntegrity` for each | test | — | same | PASS | — |
| 7 | Restore into an existing / non-disposable database | Refused | refused (Go test; live drill: re-restore exit 2; discard of the live database exit 2) | test / 5.13 | existing databases untouched | `TestR3BackupRestoreRehearsal`; drill line 21 | PASS | — |
| 8 | Missing evidence object | Restore FAILED with MISSING; never READY | READY + STORAGE_ERROR objects MISSING; the finaliser on the copy → NOT_RECOVERABLE | test | receipt states unchanged | `TestR3BackupIntegrityAndEvidenceConsistency` | PASS | Business re-collection needed |
| 9 | Corrupted evidence object | CORRUPT finding | UNBOUND upload CORRUPT | test | unchanged | same | PASS | — |
| 10 | Queue backlog after restart | Backlog persists across restart; delivered after backoff, once | delivered once | test | outbox dispatched | R2 `TestR2WorkerFaultInjection`; drill #2 | PASS | — |
| 11 | Duplicate event replay | No duplicate effect | inbox duplicate; effect count 1 | test | — | R2 relay tests | PASS | — |
| 12 | Unresolved UNKNOWN operation | Survives restore; never retried blindly | UNKNOWN count preserved; quarantined | test | UNKNOWN rows equal | `TestR3BackupRestoreRehearsal`; R2 | PASS | Needs owner evidence to resolve |
| 13 | Failed migration | Stops at the first failure; non-zero exit; no credential in output; gated services don't start | exit 1; 1 identity migration applied; 0 credential matches | 1.70 | drill-only scratch database dropped | drill line 13 | PASS | No down-migrations: restore + fix forward |
| 14 | Deployment health-check failure | `--wait` fails; the previous configuration restored healthy | wait exit 1; api restarting/1; after=healthy | 5.83 | unchanged | drill line 14 | PASS | Operator decides on rollback |
| 15 | Previous executable restored | `stage.sh rollback` to the retained images; healthy | rolled back to c01626b64fd4; revision label confirmed | 10 | **database not reverted** (by design) | `docs/remediation/evidence/r3-deploy-events-20261108.jsonl` | PASS | Business effects are not reverted |
| 16 | Missing secret | Startup refused, naming the variable only | exit 1 | 0.54 | — | drill line 16; `envfile` tests | PASS | — |
| 17 | Monitoring unavailable | Application unaffected; detection via the `CRMWatchdog` dead-man's switch | api ready 200; no scraper deployed | 0.26 | — | drill line 17 | **PARTIAL** | The monitoring pipeline and watchdog delivery are not deployed |
| 18 | Alert condition generated | `crm_worker_feature_flags_enabled` 1 → 2 → 1 when an unapproved flag is toggled | 1 → 2 → 1 | 1.13 | drill flag left disabled | drill line 18 | PASS (condition only, not delivered) | No Alertmanager or paging |
| 19 | Disk capacity threshold simulated | free/size < 0.15 on a 1 MiB tmpfs | 0.121, condition true | 0.28 | — | drill line 19 | PASS (simulated) | Node exporter not deployed |
| 20 | Safe environment cleanup | `destroy` refused without `CRM_STAGING_DESTROY=crm-staging`; unrelated volume kept | exit 2; sentinel kept; api healthy | 0.37 | unchanged | drill line 20 | PASS | Full `destroy` runs in `release-ci.yml` (verifies no project volume remains) |
| 21 | Live staging backup → verify → restore → discard | Encrypted backup of the live staging database and evidence volume; restore COMPLETE and quarantined; refusals | all exit 0; COMPLETE; 231 tables; recovery 2.5 s; 0 credential matches | 5.13 | `crm_restore_drill_*` created, then discarded | drill line 21; `r3-live-staging-restore-20261108.json` | PASS | Host `age` binary mounted; tiny dataset (not an RTO) |

## Worker and queue recovery (section 8)

These are covered by R2 tests on PostgreSQL. They prove that recovery creates no duplicate external
effect: legacy and provider call counts are asserted.

| Case | Test |
|---|---|
| Worker killed during outbox processing | `TestStaleClaimIsRecoverable`, `TestConcurrentRelaysDeliverEachEventOnce` |
| Worker stopped after consumer commit | `TestCrashAfterConsumerCommitIsSafeOnReplay` |
| In-flight receipt finalisation | `TestEvidenceFinalisation` (lease), `TestR2WorkerFaultInjection` (lost reply) |
| Pending / UNKNOWN write-back | `TestR2WorkerFaultInjection` (write-ahead claim, crash before request, observation budget) |
| Provider reconciliation backlog | `TestProviderReconcileJob` |
| Exhausted delivery | `TestFailuresVisibleBoundedAndNoSilentSkip`, `TestR2EvidenceFaults` |
| Queue backlog growth | `TestR2WorkerFaultInjection` (consumer down) |
| Database outage | `TestR2EvidenceFaults`, drill #3 |
| Object-store outage | `TestR2WorkerFaultInjection` (STORAGE_ERROR), `TestR2EvidenceFaults` |

## Reproduce

```sh
scripts/release/build.sh /tmp/a && scripts/release/verify.sh /tmp/a --mark-eligible
cd deploy/staging && ./stage.sh init-secrets && ./stage.sh up /tmp/a && ./stage.sh verify && ./drill.sh /tmp/a
CRM_TEST_BACKUP=1 go test -v -run TestR3 ./internal/integration/legacy/shadow/   # PostgreSQL + pg_dump of the same major + age
```
