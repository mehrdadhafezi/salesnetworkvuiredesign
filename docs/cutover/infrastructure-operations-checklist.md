# Slice 15 — Infrastructure and operations checklist

**Column A** records what the repository actually provides (VERIFIED at `b3bf31b`). **Column B** records
what an operator must verify in each target environment. Nothing in column B was run.

**Not provided here.** No RPO, RTO, SLA or topology figure is stated, because none exists in any
authoritative artefact. Each one is an owner decision to record.

| # | Area | A. Repository capability (VERIFIED) | B. Operator verification procedure (NOT RUN) | Status |
|---|---|---|---|---|
| 1 | Application runtime processes | `crm-api`: plain HTTP `ListenAndServe` with timeouts and graceful shutdown (`internal/platform/apiserver/server.go:876-900`, `config.go:31-46`). Offline commands: `crm-migrate`, `crm-legacy-import`, `crm-finance-shadow`, `crm-biawin-shadow`, `crm-commission-shadow`, `crm-reporting`. **No Dockerfile, compose file, unit file, Helm chart or deploy workflow.** | Document the host/orchestrator, process supervisor, binary provenance (build from a tagged commit) and environment variables per process. | NO-GO (no artefacts) |
| 2 | Database schema versions | `crm-migrate status` lists each applied migration per source (`cmd/crm-migrate/main.go`). 51 migrations across 15 version tables. | Run `crm-migrate status` with a read-only login and attach the output. | NOT VERIFIED |
| 3 | Migration ordering | Fixed source order (`cmd/crm-migrate/main.go:47-51`). Each source has its own version table. The run stops at the first failure. No app process migrates on startup. **No down migrations are used operationally.** | Rehearse on a restored copy; record duration and locks. | NOT VERIFIED |
| 4 | Backups and restoration | none | Backup schedule, retention, encryption, and a restore test with timing. | NOT VERIFIED |
| 5 | Object storage durability | Filesystem object store with fsync, hash verification and atomic rename (`objectstore.go:1-7`). Root is `CRM_PAYMENTS_OBJECT_ROOT`. | Storage class, replication, backup, capacity, path permissions, antivirus policy. | NOT VERIFIED |
| 6 | TLS and networking | App serves HTTP only. TLS is expected at a terminating proxy (not defined). | Proxy config, certificate management, HSTS, allowed origins, network segmentation of DB logins. | NOT VERIFIED |
| 7 | Secrets and credentials | Per-module DB URLs in environment variables (37 `CRM_*_DATABASE_URL` names). Distinct logins are enforced per module owner role. No secret store integration. | Secret store, rotation, least-privilege login mapping per process. | NOT VERIFIED |
| 8 | Session and cookie security | **No sessions or cookies exist** (no production auth, U1). | — | NO-GO |
| 9 | RBAC and row-level authorization | Route→module→action→row evaluator, capability catalogue, per-command row scope (VERIFIED by tests). Production principals are anonymous (U1). | Real-account matrix tests per role. | NO-GO (U1) |
| 10 | Scheduled jobs and workers | Library steps only (U6). Offline jobs are manual commands. | Scheduler definition, concurrency limits, failure ownership. | NO-GO |
| 11 | Outbox relays | `relay.Drain` library; no process. | Relay process, backlog metric, alert threshold. | NO-GO |
| 12 | Inbox consumers | Inbox dedupe library and consumers (fulfillment, activation, write-back) callable from code only. | as above | NO-GO |
| 13 | Monitoring and alerting | `/health/live` and `/health/ready` only. **No metrics endpoint.** | Metrics backend, dashboards, alert rules (5xx, latency, DB pool, outbox age, UNKNOWN counts). | NOT VERIFIED |
| 14 | Error logging and correlation | Structured `slog` logs; `X-Correlation-ID` validated and echoed (`internal/platform/httpx/correlation.go:16-24`); RFC 7807 problems. | Log shipping, retention, PII scrubbing review. | NOT VERIFIED |
| 15 | Security audit logs | Append-only `platform audit` schema (`internal/platform/audit`); sensitive-read audit for Invoice Review (D-45 fail-closed). No auth events, because there is no auth. | SIEM forwarding, review cadence. | NOT VERIFIED |
| 16 | Failure ownership | Provider operations carry `reconciliation_owner`; write-back and gateway carry UNKNOWN states. No on-call model. | Named owners per failure class (see the rollback DRAFT). | NOT VERIFIED |
| 17 | Storage retention | D-44 safe default in code: unbound uploads expire after 7 days; bound evidence is kept indefinitely. Expiry worker not deployed. | Legal retention approval (D-44). | NOT VERIFIED |
| 18 | Disaster recovery | none | DR site/procedure, failover test. | NOT VERIFIED |
| 19 | RPO / RTO | **Not defined anywhere.** | Owner decision. | NOT VERIFIED |
| 20 | Capacity and performance | **No load or performance test exists.** | Load test against realistic volumes from a real snapshot. | NOT VERIFIED |
| 21 | CI on main | CI workflows trigger on `pull_request` only (`.github/workflows/go-ci.yml:3-11`). `main` has no post-merge run; `b3bf31b` is covered only because its tree equals `26f92b1` (PR #39 green). | Decide whether to add push-to-main and release-tag pipelines. | REPORTED |
| 22 | Repository hygiene | A 16 MB compiled `crm-biawin-shadow` binary has been tracked at the repo root since Slice 10. | Remove it, and ignore build outputs. | P2 finding |

## Read-only evidence collector

`scripts/cutover/readiness-evidence.sh` runs in two modes:
- **`static`** checks the repository only.
- **`db`** runs read-only queries. It refuses unless `CRM_CUTOVER_ENV=disposable`, the host is a loopback
  address and `CRM_ENV` is not production. It forces `default_transaction_read_only=on`.

**Local results at `b3bf31b`:**
- `static`: 8 VERIFIED, 7 FINDINGS:
  - no production authentication
  - five worker steps not run by any command
  - no deployment artefacts
- `db` (disposable local database): read-only session, 0 flags enabled, 6 HR role grants, 37 staging
  tables, 0 staged snapshots.
- The script refused a non-loopback host and a missing `CRM_CUTOVER_ENV`; both refusals were tested.
