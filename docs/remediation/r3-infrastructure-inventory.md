# R3 — Infrastructure inventory (before / after)

**Baseline:** `e3aa49a` (R2 merge). **Branch:** `feat/readiness-r3-infrastructure`. **Production
remains NO-GO.**

**Classification:**
- IMPL+TESTED: implemented and tested
- IMPL: implemented, not tested
- DOC: documentation only
- MISSING
- BLOCKED-INFRA: blocked by external infrastructure
- BLOCKED-DECISION: blocked by an unresolved decision

## Before R3 (at `e3aa49a`)

| Asset | State | Evidence |
|---|---|---|
| Dockerfile / Compose / IaC / Helm / unit files | MISSING | Slice 15 checklist #1 |
| Nginx / TLS termination config | MISSING | checklist #6 |
| Deployment scripts / release packaging | MISSING | — |
| Artifact provenance / checksums | MISSING | — |
| Committed build output | **Defect:** a 16 MB debug binary `crm-biawin-shadow` tracked at the root since Slice 10 | checklist #22 |
| CI on `main` | MISSING (pull_request only) | checklist #21 |
| Migration runner | IMPL+TESTED (`crm-migrate up/status`, 53 migrations) | Slice 1 / R2 |
| Backups / restore | MISSING | checklist #4 |
| Evidence store durability / backup | IMPL (fsync, atomic rename, hash verify); backup MISSING | checklist #5 |
| API metrics | MISSING (`/health/live`, `/health/ready` only) | checklist #13 |
| Worker metrics / health | IMPL+TESTED (R2 `/metrics`, `/health/*`) | R2 |
| DB pool metrics | MISSING | — |
| Secret injection | Environment variables only | checklist #7 |
| Alert rules / dashboards | MISSING | — |
| DR plan | DOC (DRAFT) | `docs/cutover/rollback-and-recovery-DRAFT.md` |
| RPO / RTO | BLOCKED-DECISION | checklist #19 |

## After R3

| Capability | State | Where | Evidence |
|---|---|---|---|
| Deterministic release build (7 binaries + web bundle), checksums, provenance manifest | IMPL+TESTED | `scripts/release/build.sh` | Two independent builds produced identical `SHA256SUMS` (local; also the CI job `release-ci.yml`) |
| Artifact verification (checksums, embedded `vcs.revision`, clean tree) and eligibility marking | IMPL+TESTED | `scripts/release/verify.sh` | CI: a tampered artifact is refused and never marked |
| Runtime images built from the verified artifact (distroless, uid 65532, pinned digests) | IMPL+TESTED (app, web) | `deploy/docker/Dockerfile.app`, `.web` | Local builds and staging rehearsal |
| Backup-runner image | IMPL | `deploy/docker/Dockerfile.backup` | **NOT VERIFIED locally:** the build sandbox has no package-repository DNS; built in `release-ci.yml` |
| Disposable staging stack: PostgreSQL, migrate, bootstrap, api, worker, TLS proxy | IMPL+TESTED | `deploy/staging/compose.yaml`, `stage.sh` | Healthy in 22 s; `verify` evidence |
| Staging isolation: internal network, loopback-only TLS port, file secrets, no production endpoint | IMPL+TESTED | `stage.sh preflight` / `verify`; CI R3 boundary | Outbound traffic from the internal network blocked (`verify`) |
| Upgrade and executable rollback | IMPL+TESTED | `stage.sh up` / `rollback` | Upgrade 16 s; rollback 10 s; `crm_build_info` revision confirmed |
| Deployment evidence manifest | IMPL+TESTED | `stage.sh evidence` | `docs/remediation/evidence/r3-deployment-evidence-*.json` |
| Logical backup / verify / restore / discard (PostgreSQL + evidence), encrypted, consistent snapshot | IMPL+TESTED | `internal/platform/backup`, `cmd/crm-backup` | `TestR3BackupRestoreRehearsal`, `TestR3BackupIntegrityAndEvidenceConsistency`; live staging drill #21 |
| Physical backup / PITR / WAL archiving | MISSING (not claimed) | — | — |
| Backup scheduling and off-site copies | BLOCKED-INFRA | — | No scheduler or off-site store exists |
| Backup retention / deletion | BLOCKED-DECISION (dry-run planner only) | `crm-backup retention-plan` | D-44 / ops retention decision |
| API request / latency / login-outcome / pool metrics (private listener) | IMPL+TESTED | `apiserver/metrics.go`, `CRM_API_METRICS_ADDR` | `TestRequestMetricsLowCardinalityAndRedacted` |
| Worker readiness, dependency checks, flag and attestation gauges | IMPL+TESTED | `worker/metrics.go`, `cmd/crm-worker` | Drills #3, #18 |
| Backup / restore textfile metrics | IMPL+TESTED | `cmd/crm-backup` | `TestMetricsTextfile`; drill #21 |
| Alert rules and dashboard | DOC / proposed, consistency-tested | `deploy/observability/*` | `alerts_test.go`; promtool + real staging scrape in `release-ci.yml` |
| Monitoring stack, Alertmanager, paging | BLOCKED-INFRA | — | — |
| File-based secret injection (`*_FILE`) | IMPL+TESTED | `internal/platform/envfile` | Unit tests; drill #16 |
| Secret store, rotation automation | BLOCKED-INFRA (procedure documented) | `r3-deployment-runbook-DRAFT.md` | — |
| TLS for production, DNS, certificates | BLOCKED-INFRA (staging self-signed only) | — | — |
| CI on `main` (post-merge), release and staging rehearsal workflow | IMPL (runs on GitHub) | `go-ci.yml` (push), `release-ci.yml` | CI results on the PR |
| Committed binary | **Removed**, ignored, CI-guarded | `.gitignore`, R3 boundary step | — |
| RPO / RTO targets | BLOCKED-DECISION | `r3-disaster-recovery-DRAFT.md` (proposed decision matrix) | — |

Existing assets are reused rather than duplicated:
- `crm-migrate` remains the only migration path.
- R2 worker health and metrics are extended, not replaced.
- `scripts/cutover/readiness-evidence.sh` is unchanged.
