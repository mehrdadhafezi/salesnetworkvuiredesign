# R2 — Implementation matrix

Branch `feat/readiness-r2-workers-writeback` over `5b66b03`. Addresses Slice 15 E-2 and E-3 (U6 / U7).

**Production remains NO-GO.**

## Deliverables

| # | Deliverable | Status | Where |
|---|---|---|---|
| 1 | Worker runtime and executable composition | DONE | `internal/platform/worker` (runtime, metrics, health, relay job); `cmd/crm-worker` (`check` / `run` / `once`) |
| 2 | Worker configuration validation | DONE | `cmd/crm-worker/config.go`: explicit environment, DB attestation, job allowlist, per-owner logins, bounds, adapter gates |
| 3 | Safe outbox / inbox processing | DONE | `relay.Drain` rewritten: lock-held claim, backoff, one attempt per pass, explicit routes, `ErrSkip` removed; `outbox.ClaimNext`, `outbox.ReadStats` |
| 4 | Receipt finalisation scheduling | DONE | `payments/command/jobs.go` `FinaliseJob`; `MaxFinaliseAttempts` bound |
| 5 | Upload-expiry scheduling | PARTIAL (by design) | `ExpiryReportJob` (no mutation). Purge job exists but is gated to development: **BLOCKED by D-44**. |
| 6 | Write-back adapter interface and synthetic implementation | DONE | Existing `lwb.Legacy` port plus `legacytest.SyntheticLegacy`; new `legacy/compat` client (PROPOSED contract, unconstructable without approval) and `writeback_exchange` evidence |
| 7 | Guarded write-back worker composition | DONE (delivery BLOCKED) | `writeback.IntakeRoute`, `Options` (types, observe interval, observation budget), `DeliverJob`, `BacklogJob`; delivery gated `WRITEBACK_ADAPTER_DISABLED` |
| 8 | Evidence-based reconciliation orchestration | DONE (provider BLOCKED) | `providerops.ReconcileJob` (lookup only, never dispatch), `ExpireStaleDispatches`, `BacklogJob`; write-back observation budget → RECONCILIATION_REQUIRED |
| 9 | Operational metrics and diagnostics | DONE | `/metrics` (job label only), `/health/ready`, per-job backlog gauges |
| 10 | Integration and fault-injection tests | DONE | See `r2-failure-recovery.md` (25 scenarios) |
| 11–15 | Documents | DONE | `docs/remediation/r2-*.md` |

## Safety corrections to existing behaviour

| Finding | Risk | Correction | Regression test |
|---|---|---|---|
| `relay.ErrSkip` let a consumer mark ANY event dispatched without delivery | A missing route silently discarded a business effect | Removed. Routes declare handled types and documented non-actionable types. Anything else fails visibly and is bounded. | `TestFailuresVisibleBoundedAndNoSilentSkip`; CI grep |
| Relay claim committed **before** delivery | Two workers could deliver the same event concurrently | Per-event source transaction holds the row lock through delivery | `TestConcurrentRelaysDeliverEachEventOnce`, `TestStaleClaimIsRecoverable` |
| Failed events reclaimed immediately | A poison event burned all attempts in one pass | Exponential backoff plus one attempt per pass | `TestBackoffAndOneAttemptPerPass` |
| `writeback.Drain` ignored `MarkFailed` errors and duplicated relay logic | Lost failure evidence | Uses `relay.Drain` over `IntakeRoute` | Existing Slice 11 / 12 suites, unchanged and green |
| UNKNOWN write-back observed forever | Unbounded loop | Observe interval plus observation budget → RECONCILIATION_REQUIRED (opt-in; the worker sets it) | `TestR2WorkerFaultInjection` |
| STORAGE_ERROR receipts retried forever | Unbounded loop | `MaxFinaliseAttempts` = 20, then visible `receipt_retry_exhausted` | `TestR2EvidenceFaults` |
| A crashed provider dispatch stayed DISPATCHING | Stuck operation, invisible | `ExpireStaleDispatches` (15 min) → UNKNOWN (lookup only) | `TestProviderReconcileJob` |

## Adapter inventory

| Adapter | Kind | In crm-worker? | State |
|---|---|---|---|
| `legacytest.SyntheticLegacy` | Synthetic legacy store | No (test-only; lint) | Tests only |
| `legacy/compat.Client` | Real transport for a PROPOSED contract | Linked; construction refused | **BLOCKED (E-3)** |
| `integration/biawin.SyntheticPort` | Synthetic provider | No (lint, CI grep) | Tests only |
| `integration/biawin.DisabledPort` | Disabled provider | Not needed (job gated) | Default |
| Real Biawin adapter | — | — | **Missing** (protocol not evidenced) |
| `objectstore.FS` | Real filesystem store | Yes (`CRM_WORKER_EVIDENCE_ROOT`) | Available |
| `gateway/synthetic` | Synthetic gateway | No | Unchanged; not a worker concern |

## Guards

- **Lint:**
  - `r2-network-call`: client network calls only in `legacy/compat`.
  - `r2-worker-sql`: no SQL in `cmd/crm-worker`.
  - `cmd/crm-worker` added as a legacy importer (it is the write-back composition root).
- **CI step "R2 background workers"**: verbose and `-race` runs.
- **CI step "R2 boundary"**:
  - no `ErrSkip`;
  - empty contract pin;
  - no synthetic adapter or provider port in the binary;
  - no financial delivery type;
  - D-44 gate present;
  - no `Dispatch` in workers.
- **CI migrate block**: `00012` applied; `writeback_exchange` append-only, SELECT / INSERT only, digest-only columns, empty.
- **`readiness-evidence.sh static`**: reports the composed jobs and the three BLOCKED capabilities.

## Security

- **Owner logins.** One owner login per pool, enforced by `db.Runner`. The feature-flag login must differ from every owner login. No superuser.
- **System actors.** Every system action carries its system actor:
  - `service:integration.writeback`
  - `service:payments.evidence`
  - `service:integration`

  No approver is manufactured; human resolution still requires an authorised user and evidence.
- **No grants.** No capability, role grant or flag is created or enabled by R2. Job switches and the environment attestation default to off.
- **No secrets in output.** No DSN, key, token, payload or PII in logs, metrics or exchange evidence. Compat credentials never print key material.
- **No credentials from legacy.** Nothing is copied from `plugin/current`. `plugin/current` is unchanged.

## Unresolved decisions (not decided here)

| Decision | Blocks |
|---|---|
| E-3 legacy write-back contract (Product and Security) | Real write-back delivery |
| Financial write-back contract | Manual payment write-back |
| D-44 retention | Upload purge |
| Biawin protocol evidence (D-35 / D-38 context) | Provider reconciliation |
| P-SEC / ops | Alert thresholds, SLA and RTO; monitoring stack; evidence storage durability |
