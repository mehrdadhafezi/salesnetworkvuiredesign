# R3 — Infrastructure readiness matrix (reconciled with the Slice 15 checklist)

This reconciles `docs/cutover/infrastructure-operations-checklist.md` (column A = repository capability)
with R3.

**Column B (operator verification in a real target environment) is NOT signed.** No infrastructure,
security, operations or business owner approval is implied.

| # | Area | Status after R3 | Implementation reference | Test evidence | Remaining gap | Owner role | Required approval | Closure criteria |
|---|---|---|---|---|---|---|---|---|
| 1 | Application runtime processes | IMPL+TESTED (staging reference) | `deploy/docker/*`, `deploy/staging/compose.yaml`, `stage.sh` | `stage.sh up/verify`; drills 1–2 | Production orchestrator / hosts not chosen | Infrastructure owner | Topology decision | Approved target topology deployed and verified (column B) |
| 2 | Database schema versions | IMPL+TESTED | `crm-migrate status`; backup manifest records every version table | Restore rehearsal (migration diff empty) | Read-only status login per environment | Database operations | — | Status attached per environment |
| 3 | Migration ordering | IMPL+TESTED | One-shot migrate service gates services | Drill 13 | Lock / duration measurement on realistic volumes | Database operations | — | Rehearsal on a restored realistic copy |
| 4 | Backups and restoration | IMPL+TESTED (logical, staging) | `cmd/crm-backup`, `internal/platform/backup` | `TestR3*`; drill 21 | Scheduling, off-site copy, PITR, realistic-volume timing | Database operations | Retention, RPO / RTO | Scheduled encrypted backups + periodic restore rehearsal with alerts delivered |
| 5 | Object storage durability | PARTIAL | Filesystem store; backup and restore of objects with consistency checks | `TestR3BackupIntegrityAndEvidenceConsistency` | Replication, encryption at rest, capacity monitoring, antivirus policy | Infrastructure owner | Storage class | Storage evidence + restore rehearsal |
| 6 | TLS and networking | IMPL+TESTED (staging self-signed) | nginx config, internal network | `verify` (TLS 200, HSTS, outbound blocked) | Real certificates, DNS, segmentation of database logins | Infrastructure owner / security | Certificate process | Verified in the target environment |
| 7 | Secrets and credentials | IMPL+TESTED (file injection) | `envfile`, compose secrets, `0400` files | Drill 16; redaction tests | Secret manager, automated rotation | Security | Secret-management decision | Secret store integrated; rotation rehearsed |
| 8 | Session and cookie security | R1 implemented (flag OFF) | R1 | R1 tests | Real accounts BLOCKED (D-34 / D-42) | Security / business | D-34, D-42 | Per R1 |
| 9 | RBAC / row authorization | Unchanged (R1) | — | — | Real role matrix tests | Security / business | D-42 | Per R1 |
| 10 | Scheduled jobs and workers | IMPL+TESTED (R2 + staging) | `crm-worker` service | R2 tests; drills 2–3 | Job switches OFF; delivery BLOCKED (E-3) | Platform operations | Per-job enablement | Owner-approved enablement per job |
| 11 | Outbox relays | IMPL+TESTED (R2) | R2 | R2 | — | Platform operations | — | — |
| 12 | Inbox consumers | IMPL+TESTED (R2) | R2 | R2 | — | — | — | — |
| 13 | Monitoring and alerting | Metrics IMPL+TESTED; rules / dashboard PROPOSED | `apiserver/metrics.go`, `deploy/observability/*` | `alerts_test.go`; drills 17–19 | Monitoring stack, Alertmanager, paging, promtool validation | Platform operations | Thresholds, on-call model | Alerts delivered end to end in the target environment |
| 14 | Error logging and correlation | Unchanged + nginx log format without query / cookies | `nginx.conf` | — | Log shipping, retention, PII review | Platform operations / security | Retention | — |
| 15 | Security audit logs | Unchanged (R1 auth events) | — | — | SIEM forwarding | Security | — | — |
| 16 | Failure ownership | DOC (roles per alert) | `r3-observability-and-alerts.md` | — | Named owners and on-call | Operations leadership | On-call model | Approved roster (not invented here) |
| 17 | Storage retention | BLOCKED-DECISION | Retention planner is a dry run only; purge BLOCKED | — | D-44 | Legal / business | D-44 | Approved policy implemented |
| 18 | Disaster recovery | DOC (DRAFT) + rehearsed components | `r3-disaster-recovery-DRAFT.md` | Drills | Production DR site / procedure | Infrastructure owner | DR policy | Approved plan rehearsed in the target environment |
| 19 | RPO / RTO | BLOCKED-DECISION (matrix proposed) | `r3-disaster-recovery-DRAFT.md` | — | Targets | Business / operations | RPO / RTO | Approved targets met in rehearsal |
| 20 | Capacity and performance | MISSING | — | — | Load test | Platform operations | — | Load test on realistic volumes |
| 21 | CI on main | IMPL (push to main + release workflow) | `go-ci.yml`, `release-ci.yml` | PR CI | — | Engineering | — | Green post-merge runs |
| 22 | Repository hygiene | FIXED | Binary removed; `.gitignore`; CI guard | R3 boundary step | — | Engineering | — | — |

## E-6 and P0-4 status

- **E-6 (infrastructure engineering prerequisites): technically implemented for the verified
  capabilities only:**
  - reproducible artifact and provenance;
  - staging reference deployment, upgrade and rollback;
  - logical backup and restore of the database and evidence;
  - metrics;
  - DR drills.

  Not closed for: production topology, scheduling, monitoring delivery, PITR, or capacity.
- **P0-4: NOT closed.**
  - Real infrastructure, monitoring delivery and backup operations remain unverified (column B).
  - Approvals remain open: topology, secrets, retention (D-44), RPO / RTO, on-call.

**Production remains NO-GO.**
