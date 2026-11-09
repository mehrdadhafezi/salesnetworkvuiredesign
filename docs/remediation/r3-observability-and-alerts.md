# R3 — Observability and alerts

**Status:** the metrics are implemented and tested; the alert rules and dashboard are PROPOSED. No
monitoring stack, Alertmanager or paging route is deployed or verified. A firing rule, or a condition the
drill simulates, is not a delivered page. All thresholds are proposed engineering values awaiting
operations-owner approval. Roles are named, never people.

## 1. Signals (implemented)

| Area | Metric(s) | Source | Labels |
|---|---|---|---|
| API volume, errors, latency | `crm_api_requests_total`, `crm_api_request_duration_seconds_*` | `internal/platform/apiserver/metrics.go` | `method`, `route` (OpenAPI template, never a concrete path), `class` |
| Authentication failures | `crm_api_login_outcomes_total` | same | `outcome` (status class only; no login, password, token or cookie) |
| Database pool saturation | `crm_db_pool_*`, `crm_worker_db_pool_*` | `internal/platform/db/stats.go` | `owner` (module) |
| Build provenance | `crm_build_info` | API and worker | `component`, `version`, `revision` |
| Worker liveness and readiness | `crm_worker_ready`, `crm_worker_check_up` | `internal/platform/worker/metrics.go` (evaluated at scrape, checks concurrent, each ≤ 2 s) | `job` = `runtime` / check name |
| Per-job state and last success | `crm_worker_job_state_*`, `crm_worker_job_last_success_timestamp_seconds`, `crm_worker_job_runs_*_total`, `crm_worker_job_timeouts_total` | R2 runtime | `job` |
| Queue backlog and oldest age | `crm_worker_outbox_pending`, `crm_worker_outbox_oldest_pending_age_seconds`, `crm_worker_outbox_exhausted` | R2 relay jobs | `job` |
| UNKNOWN / reconciliation / parity | `crm_worker_writeback_*`, `crm_worker_provider_*` | R2 jobs | `job` |
| Write-back / provider blocked | `crm_worker_job_state_blocked{job="writeback_deliver"\|"provider_reconcile"}` | R2 gates | `job` |
| Receipt STORAGE_ERROR / NOT_RECOVERABLE | `crm_worker_receipt_storage_error`, `crm_worker_receipt_not_recoverable`, `crm_worker_receipt_retry_exhausted` | R2 finaliser | `job` |
| Flag activation / attestation | `crm_worker_feature_flags_enabled`, `crm_worker_environment_attested` | `cmd/crm-worker` | `environment` |
| Backup age / failure / restore rehearsal | `crm_backup_*`, `crm_backup_verify_*`, `crm_restore_rehearsal_*` | `cmd/crm-backup` Prometheus textfile (`CRM_BACKUP_METRICS_FILE`) | none |
| Disk / volume capacity | `node_filesystem_*` | node exporter (operator component, NOT deployed) | mountpoint |
| Certificate expiry | `probe_ssl_earliest_cert_expiry` | blackbox exporter (operator component, NOT deployed) | target |
| Migration status | `crm-migrate status`; the backup manifest records every version table | CLI | — |

**Telemetry hygiene (tested).**
- `TestRequestMetricsLowCardinalityAndRedacted` proves that no concrete id, query value, login name or
  password reaches a label or a sample.
- The worker metric registry drops invalid names and labels (`TestHealthReadinessAndMetrics`).
- The nginx access-log format records method, path and status only.

## 2. Configuration

- `deploy/observability/prometheus.yml`: scrape configuration for the staging stack.
- `deploy/observability/alerts.yaml`: 23 proposed rules.
- `deploy/observability/dashboard.json`: 16 panels.
- `deploy/observability/alerts_test.go` checks that every `crm_*` metric referenced by a rule or a panel
  is emitted by the code, and that every rule has `proposed="true"`, a severity, an owner role, a
  containment step and a runbook section in this document.
- **`promtool check rules` NOT VERIFIED.** The tool could not be fetched in this environment (Docker Hub
  rate limit; Quay and the GitHub release download were blocked by policy). It is the first operator step
  before loading the rules.

## 3. Alerts and runbooks

### CRMApiDown

| | |
|---|---|
| Detection signal | `up{job="crm-api"} == 0` for 2m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | critical |
| Responsible role | platform-operations |
| Immediate containment | check process / health; do not fail over to legacy writes |
| Recovery | Restart the process (restart policy does this automatically); if a new release caused it, `stage.sh rollback` (executables only). |
| Escalation route | Platform operations lead after 15 min unresolved. |
| Evidence to retain | Container exit status, logs around the stop, `/health/ready` responses. |

### CRMApiErrorRate

| | |
|---|---|
| Detection signal | `sum(rate(crm_api_requests_total{class="5xx"}[5m])) / clamp_min(sum(rate(crm_api_requests_total[5m])), 1e-9) > 0.05` for 10m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | inspect logs by correlation id; roll back executables if a release caused it |
| Recovery | Correlate 5xx with `X-Correlation-ID` in logs; roll back executables if release-related. |
| Escalation route | Platform operations lead; owning module engineer. |
| Evidence to retain | Error logs by correlation id; request metrics snapshot. |

### CRMApiLatency

| | |
|---|---|
| Detection signal | `histogram_quantile(0.95, sum by (le) (rate(crm_api_request_duration_seconds_bucket[5m]))) > 2` for 15m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | warning |
| Responsible role | platform-operations |
| Immediate containment | check database pool saturation |
| Recovery | Check `crm_db_pool_*` saturation and slow queries; scale or roll back. |
| Escalation route | Platform operations lead. |
| Evidence to retain | Latency histogram snapshot; pg_stat_activity summary (no query parameters). |

### CRMLoginFailuresSpike

| | |
|---|---|
| Detection signal | `sum(increase(crm_api_login_outcomes_total{outcome=~"denied|throttled"}[10m])) > 50` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | warning |
| Responsible role | security |
| Immediate containment | review auth_event aggregates; consider the identity.session_auth kill switch |
| Recovery | Review `identity.auth_event` aggregates (no credentials stored); the `identity.session_auth` kill switch stops all logins. |
| Escalation route | Security role. |
| Evidence to retain | auth_event counts by kind and window; never raw logins. |

### CRMDatabasePoolSaturated

| | |
|---|---|
| Detection signal | `max by (owner) (crm_db_pool_acquired_conns / clamp_min(crm_db_pool_max_conns, 1)) > 0.9` for 10m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | warning |
| Responsible role | platform-operations |
| Immediate containment | check slow queries / locks |
| Recovery | Find long transactions / locks; do not raise pool limits blindly. |
| Escalation route | Database operations. |
| Evidence to retain | Pool metrics; lock/activity summary. |

### CRMWorkerDown

| | |
|---|---|
| Detection signal | `up{job="crm-worker"} == 0` for 2m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | critical |
| Responsible role | platform-operations |
| Immediate containment | restart; all claims are durable |
| Recovery | Restart; claims are durable (R2). Verify readiness afterwards. |
| Escalation route | Platform operations lead. |
| Evidence to retain | Exit status; worker logs by run_id. |

### CRMDatabaseUnavailable

| | |
|---|---|
| Detection signal | `min(crm_worker_check_up) == 0 or crm_worker_ready == 0` for 2m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | critical |
| Responsible role | database-operations |
| Immediate containment | verify PostgreSQL; jobs fail closed |
| Recovery | Restore database availability; jobs fail closed and resume. |
| Escalation route | Database operations → platform lead. |
| Evidence to retain | check_up series; database logs. |

### CRMWatchdog

| | |
|---|---|
| Detection signal | `vector(1)` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | none |
| Responsible role | platform-operations |
| Immediate containment | restore the monitoring pipeline |
| Recovery | If the watchdog stops arriving, the monitoring pipeline is broken: restore it before trusting any silence. |
| Escalation route | Platform operations lead. |
| Evidence to retain | Pipeline component status. |

### CRMJobNoRecentSuccess

| | |
|---|---|
| Detection signal | `(time() - crm_worker_job_last_success_timestamp_seconds) > 3600 and on (job) crm_worker_job_state_paused == 0 and on (job) crm_worker_job_state_blocked == 0` for 5m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | read the job's last_error in logs by run_id |
| Recovery | Read the job's `last_error`; fix the cause; passes resume with backoff. |
| Escalation route | Platform operations → owning module. |
| Evidence to retain | Job status, last_error (sanitised), run_id logs. |

### CRMQueueBacklog

| | |
|---|---|
| Detection signal | `crm_worker_outbox_oldest_pending_age_seconds > 900` for 15m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | check the consumer and the relay job state |
| Recovery | Check the consumer route and the relay job's gate state. |
| Escalation route | Platform operations → owning module. |
| Evidence to retain | outbox_pending / oldest age; last_error of failing events. |

### CRMDeliveryExhausted

| | |
|---|---|
| Detection signal | `crm_worker_outbox_exhausted > 0` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | root-cause before any reviewed re-arm |
| Recovery | Root-cause; a re-arm is a reviewed manual procedure (no tool in R2/R3). |
| Escalation route | Owning module engineer + platform lead. |
| Evidence to retain | Exhausted event ids, last_error, attempts. |

### CRMUnknownIncreasing

| | |
|---|---|
| Detection signal | `delta(crm_worker_writeback_unknown[1h]) > 0 or crm_worker_provider_unknown > 0` for 1h |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | integration-owner |
| Immediate containment | never retry blindly; wait for observation / lookup |
| Recovery | Never retry blindly; observation / lookup resolves UNKNOWN; otherwise owner review. |
| Escalation route | Integration owner. |
| Evidence to retain | writeback_attempt / provider_operation_event rows. |

### CRMReconciliationRequired

| | |
|---|---|
| Detection signal | `crm_worker_writeback_reconciliation_required > 0 or crm_worker_provider_reconciliation_required > 0` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | domain-owner |
| Immediate containment | review evidence; record resolution through the authorised route |
| Recovery | Evidence-based human resolution through authorised routes only. |
| Escalation route | Domain owner (cases / HR / finance) per record. |
| Evidence to retain | Reconciliation records and the evidence used. |

### CRMWritebackParityMismatch

| | |
|---|---|
| Detection signal | `crm_worker_writeback_parity_conflicts > 0` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | domain-owner |
| Immediate containment | no automatic correction; owner review |
| Recovery | No automatic correction; compare expected vs observed legacy state. |
| Escalation route | Domain owner. |
| Evidence to retain | writeback_parity rows. |

### CRMReceiptStorageError

| | |
|---|---|
| Detection signal | `crm_worker_receipt_storage_error > 0` for 30m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | check the evidence volume (capacity, permissions) |
| Recovery | Fix the evidence volume; finalisation retries (≤ 20 claims). |
| Escalation route | Platform operations. |
| Evidence to retain | receipt_state_history; volume metrics. |

### CRMReceiptNotRecoverable

| | |
|---|---|
| Detection signal | `crm_worker_receipt_not_recoverable > 0` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | finance-owner |
| Immediate containment | never mark READY; review with the uploader |
| Recovery | Never mark READY; finance review with the uploader. |
| Escalation route | Finance owner. |
| Evidence to retain | evidence_reconciliation records. |

### CRMUnexpectedFlagActivation

| | |
|---|---|
| Detection signal | `crm_worker_feature_flags_enabled > 1` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | critical |
| Responsible role | release-manager |
| Immediate containment | verify the change against an approval record; switch OFF if unapproved |
| Recovery | Compare with an approval record; switch OFF if unapproved; investigate who changed it. |
| Escalation route | Release manager → security role. |
| Evidence to retain | feature_flag rows and updated_at; change record. |

### CRMAttestationMismatch

| | |
|---|---|
| Detection signal | `crm_worker_environment_attested != 1` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | critical |
| Responsible role | platform-operations |
| Immediate containment | stop the worker; verify which database it reaches |
| Recovery | Stop the worker; verify which database each login reaches (R2 binding). |
| Escalation route | Platform operations → security role. |
| Evidence to retain | Flag rows; worker startup refusal logs. |

### CRMBackupFailed

| | |
|---|---|
| Detection signal | `crm_backup_last_run_success == 0` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | database-operations |
| Immediate containment | read crm-backup stderr; never reuse a .partial |
| Recovery | Read crm-backup stderr (redacted); never reuse a `.partial` directory; rerun to a new destination. |
| Escalation route | Database operations. |
| Evidence to retain | crm-backup output; textfile metrics. |

### CRMBackupTooOld

| | |
|---|---|
| Detection signal | `(time() - crm_backup_last_success_timestamp_seconds) > 93600` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | database-operations |
| Immediate containment | run a backup; check the schedule |
| Recovery | Run a backup; check the scheduler (none is deployed in R3). |
| Escalation route | Database operations. |
| Evidence to retain | Backup manifests (ids, times). |

### CRMRestoreRehearsalFailed

| | |
|---|---|
| Detection signal | `crm_restore_rehearsal_last_run_success == 0 or (time() - crm_restore_rehearsal_last_success_timestamp_seconds) > 3024000` |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | database-operations |
| Immediate containment | investigate the report; backups are unproven until a rehearsal passes |
| Recovery | Read the restore report differences / evidence findings; backups are unproven until a rehearsal passes. |
| Escalation route | Database operations → platform lead. |
| Evidence to retain | Restore report JSON. |

### CRMVolumeSpaceLow

| | |
|---|---|
| Detection signal | `node_filesystem_avail_bytes{mountpoint=~"/var/lib/(crm|postgresql).*|/srv/backups.*"} / node_filesystem_size_bytes < 0.15` for 15m |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | high |
| Responsible role | platform-operations |
| Immediate containment | expand the volume; never delete evidence or backups without an approved retention decision |
| Recovery | Expand capacity; never delete evidence or backups without an approved retention decision (D-44). |
| Escalation route | Platform operations. |
| Evidence to retain | Volume metrics. |

### CRMCertificateExpiry

| | |
|---|---|
| Detection signal | `probe_ssl_earliest_cert_expiry - time() < 14 * 86400` for 1h |
| Proposed threshold | as in the expression (proposed, not approved) |
| Severity | warning |
| Responsible role | platform-operations |
| Immediate containment | renew through the certificate process |
| Recovery | Renew via the certificate process (none defined for production). |
| Escalation route | Platform operations. |
| Evidence to retain | Certificate metadata. |

