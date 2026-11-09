# R3 — Deployment runbook (DRAFT, staging rehearsal only)

**DRAFT. Not approved.** There is no production deployment command and no automated promotion from
staging to production. Every step below targets the disposable `crm-staging` project.

## Lifecycle

| Step | Command | Output / gate |
|---|---|---|
| A. Build | `scripts/release/build.sh [dir]` | `dist/<commit>/`: 7 static binaries, `web-dist.tar.gz`, `SHA256SUMS`, `manifest.json`. A dirty tree is refused. Runs `go mod verify` and `npm ci` (lockfile). |
| B. Validate | `scripts/release/verify.sh <dir>` | Checksums, manifest binding, each binary's embedded `vcs.revision` equal to the manifest commit, `vcs.modified=false`. |
| C. Package | `verify.sh <dir> --mark-eligible` | Writes `ELIGIBLE` (staging rehearsal only) only if every check passed. A failed check never produces it. |
| D. Rehearse deployment | `deploy/staging/stage.sh init-secrets`, then `stage.sh up <dir>` | Images are built from the verified artifact. The previous release tag is retained. `compose up --wait` must reach healthy, else FAILED. |
| E. Verify deployment | `stage.sh verify` | JSON with: unhealthy services; TLS readiness; HSTS; internal outbound traffic blocked; worker jobs paused or blocked; API metric series. |
| F. Rehearse code rollback | `stage.sh rollback` | Restarts the previous release's images (api, worker, web). **The database, committed business effects and evidence are NOT reverted.** |
| G. Produce evidence | `stage.sh evidence` | `deployment-evidence-<tag>.json`: image ids, revision label, timed events. |

Preflight (`stage.sh preflight <dir>`) refuses in any of these cases:
- the artifact is not eligible, or is dirty, or its checksums don't match;
- a secret is missing or not mode `0400`;
- the compose configuration is invalid;
- a production-looking endpoint appears in the configuration;
- the internal network is not `internal: true`;
- any service other than web publishes a port, or a port binds to anything but `127.0.0.1`;
- a credential is passed as a plain environment variable.

## Migration compatibility

- `crm-migrate up` runs as a one-shot service before api and worker start.
- A failed migration exits non-zero, and the dependent services never start (drill #13).
- There are **no automatic down-migrations**. Recovery from a bad migration is: restore into a new
  database (crm-backup), then fix forward.
- Code rollback is safe only when the previous release understands the current schema. Migrations are
  additive (R1 / R2 practice). Confirm this per release in its PR's Database Impact section.

## Measured (local rehearsal, 2026-11-08)

| Event | Seconds | Evidence |
|---|---|---|
| First `up` (cold) | 22 | `docs/remediation/evidence/r3-deploy-events-20261108.jsonl` |
| Upgrade to the next release | 16 | same |
| Executable rollback | 10 | same; `crm_build_info` revision confirmed |
| Misconfigured release | `--wait` fails; recovered by redeploy in ~6 | drill #14 |

## Secrets and access

| Secret | Injection | Owner uid | Rotation (procedure; not automated) |
|---|---|---|---|
| PostgreSQL admin password / DSN | `/run/secrets/pg_admin_password` (`POSTGRES_PASSWORD_FILE`), `pg_admin_dsn` | 70 / 65532 | `ALTER ROLE … PASSWORD`, rewrite the file, restart migrate / bootstrap consumers |
| Per-module logins (flags, payments, integration, cases) | `dsn_*` via `*_DATABASE_URL_FILE` | 65532 | New password with `ALTER ROLE`, rewrite the file, restart the worker; revoke by `ALTER ROLE … NOLOGIN` |
| Backup login (`pg_read_all_data WITH INHERIT TRUE`, read-only) | `dsn_backup` | 65532 | as above |
| Session secret | none: R1 sessions are server-side; tokens are hashed, and no signing key exists | — | Revoke all sessions: flag `identity.session_auth` OFF |
| TLS private key | `tls_key` (root, `0400`), mounted into nginx | 0 | Reissue the certificate, replace the file, reload nginx |
| Backup encryption | age recipients (public) for backups; the identity (private) only where verify / restore runs | 65532 | New key pair; keep old identities until no backup needs them |
| Monitoring credentials | none (private network scrape) | — | — |
| CI credentials | none used by any workflow (`permissions: contents: read`); forked PRs get no secrets | — | — |

Further rules:
- `crm-healthcheck`, `crm-worker check` and `crm-backup` never print credentials. Their redaction is tested
  (`TestRefusalsAndRedaction`, drill #13 credential scan, drill #21 credential scan).
- No legacy WordPress credential is used or copied.
- No user access, grant or privilege is created by R3. The bootstrap provisions technical owner logins
  only.

## Not provided (blocked)

- A production orchestrator, hosts, DNS, certificates or secret manager (infrastructure owner decision).
- Image registry and signing (no push anywhere).
- Approval workflow for promotion (governance decision).
