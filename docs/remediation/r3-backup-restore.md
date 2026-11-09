# R3 — Backup and restore (PostgreSQL + evidence store)

**What exists (implemented and tested):**
- a **logical** backup (pg_dump custom format) of a development or staging database, taken from **one
  exported snapshot**;
- an encrypted (age) archive of the evidence store;
- a manifest, verification, restore into a **new** disposable database, reconciliation, quarantine and
  discard.

**What does not exist (not claimed):**
- physical base backups, WAL archiving or point-in-time recovery;
- scheduling, off-site copies, retention deletion;
- production use. `crm-backup` refuses any environment other than development or staging.

## Toolchain

| Command | Behaviour |
|---|---|
| `crm-backup backup` | See below. |
| `crm-backup verify <dir>` | Manifest checksum; file digests; decryption; `pg_restore --list` structure; plaintext digest over EVERY decrypted byte (including the bytes `pg_restore --list` does not read); every archived object's SHA-256. Any drain, close or decryption error fails. |
| `crm-backup restore <dir>` | See below. |
| `crm-backup discard <db>` | Drops only a database carrying the tool's marker comment, and the evidence directory carrying the tool's marker file. |
| `crm-backup retention-plan <root>` | **Dry run.** Lists candidates beyond keep N and older than a minimum age. Deletion is not implemented (approved retention decision required). |

`crm-backup backup`:
1. The database must attest exactly the declared environment.
2. The tool opens a REPEATABLE READ transaction, captures the inventory and exports its snapshot.
3. `pg_dump --snapshot` dumps exactly that snapshot.
4. The evidence objects are copied after the snapshot.
5. Everything is written to `<dest>.partial`, which is renamed only when the manifest is complete.

`crm-backup restore <dir>`:
1. Verify the backup first.
2. `CREATE DATABASE crm_restore_*`. The target must not exist; the database is marked INCOMPLETE.
3. `pg_restore --single-transaction --exit-on-error`.
4. Re-capture the inventory and diff it against the manifest.
5. Extract the evidence into a new directory and check every receipt or upload row against its bytes.
6. **Quarantine:** set every flag OFF.
7. Mark the database COMPLETE or FAILED.

The manifest records:
- format and id, environment, tool revision, pg_dump version;
- the consistency statement;
- encryption and recipients fingerprint, file digests;
- the inventory, which holds counts and digests only (no row content):
  - per table: row count, an order-independent checksum, owner;
  - per migration version table: applied count and max version;
  - constraints per schema and type, and not-validated constraints;
  - grants per `crm_*` role;
  - lifecycle state counts (write-back, provider, receipts, uploads, submitted attempts; undispatched
    outbox per schema);
  - feature flags;
- per evidence object: key, SHA-256 and size.

No DSN, host, password or row content appears in it (asserted in the tests).

## Safety

- Destinations must lie under an absolute allowlist (`CRM_BACKUP_ALLOWED_ROOTS`) and must not exist.
  `.partial` is never valid, and a leftover `.partial` blocks reuse of its name.
- A restore never overwrites: an existing target, or a name outside `crm_restore_*`, is refused. The
  source is proven unchanged by comparing inventories after the backup.
- Credentials reach `pg_dump` and `pg_restore` only via `PG*` child environment variables. Errors are
  redacted.
- Encryption is mandatory outside development. The recipients file is public; the identity is needed
  only where `verify` and `restore` run.
- A restored copy is quarantined. All flags are set OFF, including `worker.environment.*`, so a worker
  pointed at it refuses to start (attestation) until an operator re-validates and re-attests it.

## Rehearsal results

The rehearsal is `TestR3BackupRestoreRehearsal` and `TestR3BackupIntegrityAndEvidenceConsistency`
(`internal/integration/legacy/shadow/r3_backup_e2e_test.go`). They run on PostgreSQL 16 locally and on
PostgreSQL 18 in CI (`CRM_TEST_BACKUP=1`).

| # | Step | Result |
|---|---|---|
| 1 | Synthetic data across modules | identity, organization, billing, payments (2 READY, 1 STORAGE_ERROR receipt, 1 UNBOUND upload, real objects), integration write-back incl. UNKNOWN, outbox / inbox, flags |
| 2 | Known state recorded | Source inventory captured before the backup |
| 3 | Backup | Encrypted; at least 50 tables; at least 10 migration tables; at least 4 objects |
| 4 | Integrity verified | `verify`: database archive and every object |
| 5 | Restore into a different disposable database | `crm_restore_<random>` |
| 6 | Migrations and schema | Migration states identical (`Diff` empty) |
| 7 | Owner-scoped permissions | Table owners and per-role grant counts identical; payments tables have a single owner |
| 8 | Table counts reconciled | All tables: rows equal |
| 9 | Deterministic checksums | All tables: checksums equal |
| 10 | Constraints / foreign keys | Constraint counts per schema and type equal; not-validated constraints equal |
| 11 | Audit / outbox / inbox | Included in 8–9; undispatched outbox counts equal |
| 12 | UNKNOWN survives | Write-back UNKNOWN count equal (> 0) |
| 13 | Restored services start safely | The payments finaliser on the restored copy finalises the restored STORAGE_ERROR receipt from its restored bytes; workers stay quarantined |
| 14 | Recovery time measured | Report fields `timings_seconds` and `measured_recovery_seconds` |
| 15 | Nothing overwritten | Source inventory unchanged; re-restore into the same target refused; discard of the source refused |

The live staging drill (#21) restored the running staging database (231 tables) into `crm_restore_drill_*`:
- status COMPLETE, measured recovery **2.5 s**;
- the attestation flag was quarantined;
- re-restore was refused, discard of the live database was refused, and the drill copy was discarded.

Evidence: `docs/remediation/evidence/r3-live-staging-restore-20261108.json`.

**Not an RTO.** These timings come from tiny synthetic datasets on one host. They are evidence that the
procedure works, not recovery targets. See the decision matrix in `r3-disaster-recovery-DRAFT.md`.

## Database and object-store consistency

**Strategy:**
- Snapshot the database first, then copy the objects.
- Evidence objects are content-addressed (`evidence/<sha256>`), immutable and never deleted while
  referenced. So every object the snapshot references is present unless storage lost it.
- Objects created after the snapshot are harmless extras.
- Quarantine objects of unbound uploads may expire. Purge is BLOCKED by D-44, so this does not happen
  today.

| Check | Result (`TestR3BackupIntegrityAndEvidenceConsistency`) |
|---|---|
| Object existence / expected SHA-256 | Every READY, QUARANTINED, STORAGE_ERROR or FINALIZING receipt and every UNBOUND or BOUND upload is checked against its bytes |
| Missing object | Missing READY and STORAGE_ERROR objects reported MISSING; restore FAILED |
| Corrupt object | Corrupt UNBOUND upload reported CORRUPT |
| Incomplete copy | Truncated archive refused (`ErrIntegrity`) |
| Database restored without matching objects | FAILED status, no state changed; the finaliser on that copy declares the receipt NOT_RECOVERABLE and never READY |
| Objects without matching database references | Harmless extras (not referenced); not a finding |
| Interrupted backup | A `.partial` directory has no manifest, never verifies and blocks the name |
| Failed dump or encryption | A failing `pg_dump` (encrypted and plaintext) or a failing `age` returns an error, publishes nothing and writes no manifest; the name stays blocked (`TestR3FailedBackupPublishesNoManifest`) |
| Large dump, tampered tail | ~9.6 MB of incompressible synthetic data: a valid backup verifies and restores (encrypted and development plaintext); one byte changed near the end of the decrypted dump, with file digests resealed, is rejected by the content digest (`TestR3LargeEncryptedBackupVerifiesRestoresAndRejectsTailTamper`, `TestR3DevelopmentPlaintextLargeBackupRoundTrip`) |
| Interrupted restore | The database stays marked INCOMPLETE (or FAILED); `discard` removes only tool-marked copies |
| Wrong key / tampered manifest / corrupted dump | Refused (`ErrIntegrity`) |
| Replay / reconciliation after restore | Finaliser behaviour above |

No receipt is ever marked READY because its row was restored. A restore without the bytes is a FAILED
restore.

## Not verified / blocked

- **Production-grade storage durability:** no infrastructure evidence (replication, off-site, encryption at
  rest).
- **Scheduled backups, alerting on backup age:** rules are proposed; no scheduler or monitoring is
  deployed.
- **PITR:** not implemented.
- **Role provisioning on a different cluster:** owner roles must exist before `pg_restore` in a new
  cluster. Run `crm-migrate` there first, or provision the roles; this is documented, not automated.
- **`Dockerfile.backup` image build:** NOT VERIFIED locally (no package DNS in the build sandbox); built
  in `release-ci.yml`.
