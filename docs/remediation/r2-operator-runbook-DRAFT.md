# R2 — crm-worker operator runbook (DRAFT)

**DRAFT. Disposable staging or development only. Never run these steps against Production.**

Production remains NO-GO. Nothing here connects to live WordPress, WooCommerce, gateway, wallet or Biawin systems.

## 1. Build

```sh
go build -o crm-worker ./cmd/crm-worker
./crm-worker check   # validates config + attestation, prints the redacted plan, exits
```

## 2. Prepare a disposable staging database

1. Create the database and apply migrations with `crm-migrate up`. Then run `crm-migrate status`: it must show 53 applied and none pending, including `00012_integration_writeback_exchange.sql`.
2. Create one LOGIN role per owner the enabled jobs need. Use NOINHERIT, no superuser or CREATEROLE, and make each a member of exactly one `crm_owner_<module>` role. `db.Runner` refuses any other shape. Never share a login across owners or with the feature-flag login.
3. Attest the environment deliberately. Set only `worker.environment.staging`; `worker.environment.development` and `worker.environment.production` must stay unset or false:

   ```sql
   -- as the platform_featureflags owner, using the featureflags tooling
   worker.environment.staging = true
   ```

   The worker refuses to start if the declared `CRM_WORKER_ENV` and the attestation differ, or if more than one environment is attested. A host name (including `localhost`) is never used as evidence.

   Every owner login must also reach the **same database** as the feature-flag login. Each connection reports `pg_control_system().system_identifier` and `current_database()`, and any mismatch refuses startup. One attested database therefore cannot authorise jobs against another.

## 3. Configure

| Variable | Value |
|---|---|
| `CRM_WORKER_ENV` | `staging` (required, explicit) |
| `CRM_WORKER_JOBS` | e.g. `receipt_finalise,upload_expiry,writeback_intake_cases,writeback_intake_organization,writeback_intake_payments,fulfillment_intake,writeback_deliver,provider_reconcile` |
| `CRM_WORKER_FEATUREFLAGS_DATABASE_URL` | feature-flag login (required) |
| `CRM_WORKER_<MODULE>_DATABASE_URL` | one per owner: `PAYMENTS`, `INTEGRATION`, `CASES`, `ORGANIZATION`, `BIAWIN`, `FULFILLMENT`, `CATALOG` |
| `CRM_WORKER_EVIDENCE_ROOT` | absolute path of the evidence volume (finalisation) |
| `CRM_WORKER_ADDR` | health / metrics listener (default `127.0.0.1:9091`; private network only) |
| `CRM_WORKER_INTERVAL` / `_TIMEOUT` / `_MAX_BACKOFF` / `_BATCH` / `_MAX_PARALLEL` / `_GRACE` | defaults 30s / 2m / 10m / 50 / 4 / 30s |
| `CRM_WORKER_UPLOAD_EXPIRY_MODE` | `report` (default). `purge` is refused outside development (D-44). |
| `CRM_WORKER_WRITEBACK_ADAPTER` | `disabled` (default). `compat` refuses to start: no approved contract (E-3). |
| `CRM_WORKER_PROVIDER_ADAPTER` | `disabled` only |

DSNs are never logged or printed. `check` lists only the owner names.

## 4. Start, stop and switch jobs

- **Start:** `./crm-worker run`. Every job starts **paused** until its switch `worker.job.<job>` is set to true.
- **Switch a job on or off:** set or clear `worker.job.<job>`. The change takes effect on the next pass. A paused job leaves its work queued and visible.
- **Stop:** send SIGTERM or SIGINT. In-flight passes get `CRM_WORKER_GRACE` to finish, then they are cancelled. All claims live in PostgreSQL, so any interrupted pass is redone safely.
- **One pass:** `./crm-worker once <job>`. Exit 0 means the pass succeeded; non-zero means paused, blocked or failed.

## 5. Monitor

| What | Where |
|---|---|
| Liveness | `GET /health/live` |
| Readiness | `GET /health/ready`: database pings, runtime running, no stale enabled job |
| Metrics | `GET /metrics`: label `job` only (see `r2-worker-coverage.md`) |
| Logs | JSON on stderr, keyed by `job`, `instance` and `run_id` |

Expected on staging today:

- `job_state_blocked{job="writeback_deliver"} 1`
- `job_state_blocked{job="provider_reconcile"} 1`
- `writeback_pending` grows as commands are recorded (delivery is blocked)

## 6. Rehearse a timeout (staging, synthetic only)

There is no staging legacy endpoint, so real write-back cannot be rehearsed. The rehearsal is the test suite:

```sh
go test -count=1 -v -run 'TestR2' ./internal/integration/legacy/shadow/
go test -count=1 -v ./internal/integration/legacy/compat/
```

## 7. Recover a crashed worker

1. Restart it. Nothing else is needed:
   - relay claims are released with the connection;
   - FINALIZING receipts are re-claimed after the 5-minute lease;
   - write-back claims are already UNKNOWN (observed, never re-applied);
   - DISPATCHING provider operations become UNKNOWN after 15 minutes.
2. Check `outbox_failed_retryable`, `receipt_storage_error`, `writeback_unknown` and `provider_unknown`.

## 8. Review UNKNOWN operations

- **Write-back:**

  ```sql
  SELECT * FROM integration.writeback_command WHERE state IN ('UNKNOWN', 'RECONCILIATION_REQUIRED');
  ```

  The evidence is in `writeback_attempt`, `writeback_reconciliation` and `writeback_exchange`. Never re-apply by hand.
- **Provider:** use the read API `GET /api/v1/integration/provider-operations?state=UNKNOWN`. Resolve through the authorised resolution route with evidence.

## 9. Check audit evidence

| Area | Tables |
|---|---|
| Receipts | `payments.receipt_state_history`, `payments.evidence_reconciliation` |
| Write-back | `integration.writeback_attempt`, `writeback_reconciliation`, `writeback_exchange` |
| Provider | `integration.provider_operation_event`, `provider_callback` |
| Delivery | `<module>.outbox_event`, `<module>.inbox_message` |

## 10. Disable safely

Turn off `worker.job.<job>`, or remove the job from `CRM_WORKER_JOBS` and restart. Queued work stays queued. Nothing is marked done and nothing is deleted.

## 11. Staging rollback rehearsal

1. Stop the worker.
2. Clear all `worker.job.*` switches.
3. Verify the backlogs are unchanged.
4. Restore the database from the pre-rehearsal snapshot.
5. Confirm `crm-migrate status`.

Migration `00012` is additive and empty, so code rollback does not require a schema rollback.

## Dependency matrix

| Capability | Runs with what exists today | Needs before it can operate for real |
|---|---|---|
| Outbox relays (intake) | Yes: PostgreSQL only | Staging DB, owner logins, switches |
| Receipt finalisation | Yes: shared evidence volume | Durable evidence storage (infra), backup policy |
| Upload expiry report | Yes | — |
| Upload purge | **No** | D-44 retention decision |
| Write-back delivery | **No** | Approved compatibility contract, legacy endpoint on a staging copy, keys (E-3) |
| Financial write-back | **No** | Separately approved financial contract |
| Provider reconciliation | **No** | Verified Biawin lookup protocol and adapter |
| Alerting | Metrics exist | Monitoring stack and approved thresholds |
