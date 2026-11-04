# Slice 15 — Feature flag and capability audit

**Scope.** This audit covers the repository at `b3bf31b`. It describes repository defaults only.

**Production flag state is NOT VERIFIED.** This environment has no production database or
configuration access, and repository defaults do not prove production settings.

## 1. Flag mechanism (VERIFIED)

- **Storage.** Flags live in `platform_featureflags.feature_flag(flag_key, enabled, description,
  updated_at)` (`internal/platform/migrations/sql/00002_platform_featureflags.sql`).
- **Absent means OFF.** An absent flag is disabled (`internal/platform/featureflags/featureflags.go:108-115`).
- **No seeding.** No migration seeds a flag (`readiness-evidence.sh static`; CI Slice 14 grep). On a fresh
  database every flag is OFF. This was verified with `readiness-evidence.sh db` on a freshly migrated
  local database: 0 flags enabled.
- **Who reads flags.** crm-api reads flags through the `CRM_FEATUREFLAGS_DATABASE_URL` login
  (`cmd/crm-api/read.go:47,120`). Each request re-reads the flag, so no restart is needed.
- **Who writes flags.** No API, CLI or UI writes a flag. A flag can only be changed by direct SQL by
  whoever holds the flag-store owner login. The procedure, its audit trail and the authorised operator
  are **UNKNOWN** and need an operator procedure (see the runbook).
- **Rollout modes.** Command families take a mode from two flags (`internal/platform/rollout/rollout.go:55-77`):

  | `<family>_write` | `<family>_writeback` | Mode | Effect |
  |---|---|---|---|
  | OFF | any | `TARGET_WRITE_DISABLED` | Command answers BLOCKED. Nothing is written. |
  | ON | OFF | `SHADOW_COMPARE` | Command is evaluated and the intent recorded. The aggregate is not applied. |
  | ON | ON | `CONTROLLED_WRITEBACK` | Target aggregate committed, history recorded, write-back intent queued. |

  A read error yields `TARGET_WRITE_DISABLED` plus an error. A failure never enables anything.

## 2. Flag inventory

All flags default **OFF**. "Current state" is **NOT VERIFIED** for every flag.

| Flag | Location | Controls | Depends on | Permissions required | DISABLED | SHADOW | CONTROLLED | Bypass risk | Rollback consequence |
|---|---|---|---|---|---|---|---|---|---|
| `organization.hr_read_shadow` | `cmd/crm-api/read.go:53` | Slice 4 identity/org/HR reads | 3 read logins + flag store | `identity.user.read`, `org.*.read` | BLOCKED / FEATURE_DISABLED | n/a | n/a | none | reads stop |
| `customers.identity_shadow_read` | `read.go:114` | customer reads, merge-case queue | customers login | `customers.*.read` | BLOCKED | n/a | n/a | none | reads stop |
| `sales.read_shadow` | `read.go:192` | cases, custody, sales reports; also the read side of the contact-outcome panel | cases + custody logins | `cases.case.read`, `custody.custody.read`, `sales.report.read` | BLOCKED | n/a | n/a | none | reads stop |
| `billing_payments.read_shadow` | `read.go:271` | invoices, stages, payments reads | billing + payments logins | `billing.*`, `payments.attempt.read` | BLOCKED | n/a | n/a | none | reads stop |
| `reporting.read_shadow` | `read.go:353` | MIS, reports, metrics, Invoice Review | reporting (+ audit for Invoice Review) | `reporting.*` | BLOCKED | n/a | n/a | Invoice Review fails closed if the audit write fails (D-45 safe default) | reads stop |
| `finance.shadow_ledger_read` | `finance.go:25` | shadow ledger reads | finance login | `finance.*.read` | BLOCKED | n/a | n/a | none | reads stop |
| `biawin_fulfillment.read_shadow` | `biawin.go:29` | Slice 10 subscriptions, cases, parity | biawin + fulfillment logins | `biawin.subscription.read`, `fulfillment.case.read`, `biawin.parity.read` | BLOCKED | n/a | n/a | none | reads stop |
| `commission.read_shadow` | `commission.go:24` | commission shadow reads | commission login | `commission.*.read` | BLOCKED | n/a | n/a | none | reads stop |
| `integration.writeback_read` | `writes.go` | write-back diagnostics | writeback login | `integration.writeback.read` | BLOCKED | n/a | n/a | none | reads stop |
| `cases.nonfinancial_write` / `_writeback` | `writes.go` | `RecordContactOutcome` | `sales.read_shadow` for the panel read; cases + custody holder logins | `cases.contact_outcome.record` | BLOCKED, no write | intent recorded, not applied | case outcome committed; outbox → write-back | **CONTROLLED without a running write-back worker and production connector (U6/U7) commits target outcomes that never reach legacy: divergence** | OFF stops new commands; committed outcomes and queued intents remain (C/D) |
| `organization.hr_request_write` / `_writeback` | `writes.go` | HR request create / step decision | organization write login | `org.hr_request.create/review/read` | BLOCKED | intent | request committed; write-back queued | same divergence risk as above | same |
| `payments.manual_write` / `_writeback` | `payments.go:36,63-71` | upload staging, receipt-mode batch | payments + billing read logins, object root | `payments.upload.stage`, `payments.batch.submit` | BLOCKED | **BLOCKED for payments** (Slice 12 §7) | attempt and evidence committed; write-back queued | **Uploads are written to the object store when the write flag is ON. Without the finaliser (U6), receipts stay QUARANTINED and attempts can never be reviewed. Without write-back, legacy never sees the payment: divergence on money.** | OFF stops new submissions; committed attempts, receipts and objects remain and need formal compensation (C) |
| `finance.review_read` | `payments.go:38,67` | read-only Finance review queue | payments login | `finance.review.read` | BLOCKED | n/a | n/a | No approve/reject exists | reads stop |
| `fulfillment.work_read` | `operations.go:42` | work queue, activation decisions, provider operations | 4 ops logins + flag store | `fulfillment.work.read`, `biawin.activation.read`, `integration.provider_operation.read` | BLOCKED | n/a | n/a | none | reads stop |
| `fulfillment.work_write` / `_writeback` | `operations.go:45` | assign, action, evidence, outcome | `fulfillment.work_read` (UI) | `fulfillment.work.assign/execute` plus catalog scope | BLOCKED | evaluated, not applied | applied to the target-native SHADOW case; no legacy write-back implemented | Cases exist only if the activation consumer ran; no production producer exists | OFF stops new commands |
| `integration.provider_reconcile` | `operations.go:47` | human resolution of UNKNOWN provider operations | `fulfillment.work_read` | `integration.provider_operation.reconcile` | BLOCKED | n/a | n/a | No dispatch path exists in crm-api | OFF stops resolutions; recorded resolutions remain |

### Side effects that bypass, or survive, a flag

| # | Side effect | Verification | Status |
|---|---|---|---|
| 1 | Offline jobs (`crm-legacy-import`, `crm-finance-shadow`, `crm-biawin-shadow`, `crm-commission-shadow`, `crm-reporting`) do not read flags. They write only their own target schemas and never touch legacy (lint `legacy-import`; CI boundary steps). | VERIFIED | Running them against production staging requires a separate operator authorisation. |
| 2 | Sensitive-read audit rows are written on Invoice Review reads only. | VERIFIED | Writes audit rows only, never business state. |
| 3 | GET endpoints never write business state. | VERIFIED by tests (`TestOperationsHTTP` fingerprint and earlier slices' HTTP e2e). Lint rules `read-path-import`, `s13-read-write`, `biawin-read-write`. | No GET writes business state. |

## 3. Capability audit (VERIFIED in repository; production NOT VERIFIED)

- **Catalogue.** The capability list is `internal/identity/api/api.go:31-112`: 43 capabilities. The dev
  stub grants all of them to synthetic `dev-user-001` (`internal/authorization/guard.go:42-55`). That stub
  refuses production (`authstub.go:35,46`).
- **Role grants in migrations.** The only role→capability rows are the six HR read grants
  (`00001_identity.sql:137-143`): `hr → identity.user.read, org.person.read, org.employment.read,
  org.position.read, org.unit.read, org.assignment.read`. Every later slice's CI asserts that its new
  capabilities are granted to no role.
- **Fresh-database result.** `readiness-evidence.sh db` reports exactly those 6 rows.
- **No silent grants.** No migration since Slice 4 inserts into `identity.role_capability` (CI DB
  invariants for Slices 5–14). Shadow imports store legacy role evidence as `role_grant` rows with
  `LegacyValue`, not as new capabilities.
- **Effective access in production is nil.** Production requests are anonymous (U1), so no capability can
  be exercised regardless of grants.

## 4. Findings

| ID | Finding | Severity |
|---|---|---|
| FF-1 | No authenticated principal exists in production, so flags cannot be exercised by real users. | P0 (inherited from U1) |
| FF-2 | Setting a write family to CONTROLLED has no supporting worker or connector, so target writes diverge from legacy. Payments CONTROLLED is the money case. | P0 |
| FF-3 | No audited operator procedure for flag changes. Direct SQL only; history keeps only `updated_at`. | P1 |
| FF-4 | No grant-management path. Capabilities can only be granted by SQL; D-42 is open. | P1 |
| FF-5 | Production flag values: **NOT VERIFIED**. | — |

**Operator verification (do not run without authorisation).** Run against each environment with a
read-only login:

```sql
SELECT flag_key, enabled, updated_at FROM platform_featureflags.feature_flag ORDER BY 1;
SELECT role_key, capability_key FROM identity.role_capability ORDER BY 1, 2;
```

Expected for any environment not yet approved for a pilot: zero enabled flags and exactly the six HR
read grants. Record the output with environment name, timestamp and operator.
