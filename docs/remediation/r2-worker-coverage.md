# R2 — Worker coverage matrix (E-2)

Baseline `5b66b03` (R1 merge). This matrix lists every background operation the Slice 15 audit found without deployable scheduling (U6 in `docs/cutover/module-readiness-matrix.md`).

Each row is classified as one of:

- **domain service**: the business logic exists as a library function.
- **durable state machine**: the states are persisted in PostgreSQL.
- **worker**: a scheduled, runnable process.
- **synthetic adapter**: a deterministic test counterpart.
- **real connector**: a production external client.
- **missing**: the capability does not exist.

All values below are proposed engineering defaults, not approved SLA values.

## Before → after

| # | Operation | Before R2 | After R2 |
|---|---|---|---|
| A | Transactional outbox relay | Domain service only (`relay.Drain`). No process ran it. Claims were released before delivery. `ErrSkip` silently dispatched any event a consumer chose to skip. | **Worker:** `fulfillment_intake` and `writeback_intake_{cases,organization,payments}`. Each event is claimed under a lock held through delivery. There is an explicit route per source outbox, and no silent skip. |
| B | Receipt / evidence finalisation | Domain service plus durable state machine (`Finalise`, lease 5 min). Retries were unbounded. No process ran it. | **Worker:** `receipt_finalise`. Retries are bounded (`MaxFinaliseAttempts` = 20). Backlog metrics. |
| C | Unbound-upload expiry | Domain service (`ExpireUnbound`: claim, then **delete**). No process ran it. | **Worker:** `upload_expiry` in **report-only** mode. Purge is **BLOCKED by D-44** outside a disposable development database. |
| D | Legacy write-back delivery | Domain service plus durable state machine (`writeback.Worker.Step`). Synthetic legacy only. Observation of UNKNOWN commands was unbounded. | **Worker:** `writeback_deliver`, bounded by an observe interval and an observation budget, with `writeback_backlog` alongside. **Delivery is BLOCKED:** no approved legacy contract (E-3). |
| E | Provider-operation reconciliation | Domain service plus durable state machine (`providerops.Reconcile`). A crashed dispatch stayed DISPATCHING forever. | **Worker:** `provider_reconcile`, with stale-dispatch recovery to UNKNOWN. Reconciliation is **BLOCKED:** no verified provider adapter. `provider_backlog` runs alongside. |
| F | Inbox consumers | `fulfillment.OnItemActivated` and `writeback.Receive` were inbox-deduplicated, but nothing ran them. | Composed through A. |

## Per-job contract

| Job | Event source | Business owner | DB owner (login) | Input contract | Switch / gate | Scheduling |
|---|---|---|---|---|---|---|
| `fulfillment_intake` | `biawin.outbox_event` | Fulfillment | biawin (source), fulfillment, catalog, integration | `biawin.subscription_item_activated` v1 (handled)<br>`biawin.subscription_activated` (declared non-actionable, with reason) | `worker.job.fulfillment_intake` | Interval 30 s ± 20 %, batch 50 |
| `writeback_intake_cases` | `cases.outbox_event` | Cases | cases (source), integration | `cases.contact_outcome_recorded` | `worker.job.writeback_intake_cases` | same |
| `writeback_intake_organization` | `organization.outbox_event` | Organization (HR) | organization (source), integration | `organization.hr_request_created`, `organization.hr_request_decided` | `worker.job.writeback_intake_organization` | same |
| `writeback_intake_payments` | `payments.outbox_event` | Payments | payments (source), integration | `payments.attempt_evidence_ready`<br>`payments.batch_submitted` (consumed without a command) | `worker.job.writeback_intake_payments` | same |
| `receipt_finalise` | `payments.receipt_evidence` (QUARANTINED / STORAGE_ERROR / lease-expired FINALIZING) | Payments (Finance owns the reconciliation records) | payments | Content-addressed copy `evidence/<sha256>` | `worker.job.receipt_finalise` | same |
| `upload_expiry` | `payments.upload_object` | Payments (retention: D-44) | payments | Read-only report | `worker.job.upload_expiry`; purge BLOCKED by D-44 | same |
| `writeback_deliver` | `integration.writeback_command` | Integration (domain owners review) | integration | Neutral intents (cases / HR only; payments excluded) | BLOCKED (`WRITEBACK_ADAPTER_DISABLED`) | same |
| `writeback_backlog` | `integration.writeback_*` | Integration | integration | Read only | `worker.job.writeback_backlog` | same |
| `provider_reconcile` | `integration.provider_operation` | Integration | integration | UNKNOWN → provider lookup | BLOCKED (`PROVIDER_ADAPTER_NOT_VERIFIED`) | same |
| `provider_backlog` | `integration.provider_*` | Integration | integration | Read only, plus stale DISPATCHING → UNKNOWN | `worker.job.provider_backlog` | same |

## Guarantees

### Relays (`*_intake`)

| Aspect | Guarantee |
|---|---|
| Idempotency | Consumer inbox `(consumer, message_id)`. Stable event ids. |
| Delivery | At least once to the consumer, at most one effect. |
| Timeout | Per-pass deadline (2 min). Optional per-event `DeliveryTimeout`. |
| Retry | Backoff 30 s · 2^(n−1), capped at 30 min. At most 10 attempts. |
| UNKNOWN handling | n/a (internal). |
| Failure state | Outbox row `failed`, with `last_error`. After 10 attempts it is `exhausted` (never claimed, never deleted). |
| Recovery | A crash rolls the claim back and the event is redelivered. The inbox turns the redelivery into a no-op. |
| Audit evidence | Outbox row plus inbox row, in their owners' schemas. |
| Metrics | `relay_dispatched_total`, `relay_ignored_total`, `relay_delivery_failed_total`, `relay_no_route_total`, `outbox_pending`, `outbox_failed_retryable`, `outbox_exhausted`, `outbox_oldest_pending_age_seconds` |
| Status | **Implemented** |

### `receipt_finalise`

| Aspect | Guarantee |
|---|---|
| Idempotency | Content-addressed key; `CopyIfAbsent` verifies the hash. |
| Delivery | Exactly one READY transition per receipt (version check). |
| Timeout | Per-pass deadline. A FINALIZING lease of 5 min recovers a crashed pass. |
| Retry | STORAGE_ERROR is retried on later passes, at most 20 claims. |
| UNKNOWN handling | A lost copy reply becomes STORAGE_ERROR. The next pass verifies the object instead of copying again. |
| Failure state | STORAGE_ERROR (quarantine kept, reconciliation opened). NOT_RECOVERABLE (missing or corrupt bytes, Finance reconciliation). |
| Recovery | Automatic retry from quarantine. NOT_RECOVERABLE requires domain-owner review. |
| Audit evidence | `receipt_state_history`, `evidence_reconciliation` (OPENED / RESOLVED), `submitted_attempt_event`. |
| Metrics | `receipt_ready_total`, `receipt_storage_error`, `receipt_not_recoverable`, `receipt_retry_exhausted`, `receipt_open_reconciliations`, `receipt_oldest_unfinalised_age_seconds` |
| Status | **Implemented** |

### `upload_expiry`

| Aspect | Guarantee |
|---|---|
| Idempotency | Report: n/a. Purge (development only): claim `UNBOUND → EXPIRING` with SKIP LOCKED, and a binding check before deletion. |
| Delivery | n/a. |
| Timeout | Per-pass deadline. |
| Retry | Purge failure is recorded (`purge_attempts`, `PURGE_FAILED`). |
| UNKNOWN handling | n/a. |
| Failure state | Report: none (no mutation). |
| Recovery | n/a. |
| Audit evidence | `upload_event` (purge mode). |
| Metrics | `upload_unbound_expired`, `upload_expiring`, `upload_purge_failures`, `upload_oldest_unbound_expired_age_seconds` |
| Status | **Report implemented. Purge BLOCKED (D-44).** |

### `writeback_deliver`

| Aspect | Guarantee |
|---|---|
| Idempotency | Command id = legacy request token / reconciliation key. Write-ahead UNKNOWN claim before every apply. |
| Delivery | At most one apply per proven-absent window. |
| Timeout | Transport timeout → UNKNOWN. |
| Retry | Re-apply only after an observation proves ABSENT. |
| UNKNOWN handling | Observed no sooner than every 5 min. After 12 consecutive unreadable observations: RECONCILIATION_REQUIRED (`WRITEBACK_UNKNOWN`, `DOMAIN_OWNER_REVIEW`). |
| Failure state | REJECTED (proven refusal), RECONCILIATION_REQUIRED. |
| Recovery | Evidence-based owner review. |
| Audit evidence | `writeback_attempt`, `writeback_reconciliation`, `writeback_parity`, `writeback_exchange` (new). |
| Metrics | `writeback_applied_total`, `writeback_outcome_unknown_total`, `writeback_rejected_total`, `writeback_reconciliation_opened_total`, `writeback_pending`, `writeback_unknown`, `writeback_reconciliation_required`, `writeback_parity_conflicts`, `writeback_oldest_pending_age_seconds` |
| Status | **Worker implemented. Real delivery BLOCKED (E-3).** |

### `provider_reconcile`

| Aspect | Guarantee |
|---|---|
| Idempotency | Operation key; the reference is our own deterministic id. |
| Delivery | Never dispatches. |
| Timeout | Lookup error → RECONCILIATION_REQUIRED. |
| Retry | No blind redispatch. CONFIRMED_ABSENT permits a later explicit dispatch command only. |
| UNKNOWN handling | Lookup only. DISPATCHING older than 15 min → UNKNOWN. |
| Failure state | RECONCILIATION_REQUIRED. |
| Recovery | Human `Resolve`, with evidence and actor. |
| Audit evidence | `provider_operation_event`, `provider_callback`. |
| Metrics | `provider_unknown`, `provider_reconciliation_required`, `provider_dispatching`, `provider_conflicting_callbacks`, `provider_unmatched_callbacks`, `provider_stale_dispatch_unknown_total`, `provider_lookup_*_total`, `provider_oldest_unresolved_age_seconds` |
| Status | **Worker implemented. Real reconciliation BLOCKED (no verified adapter).** |

### Every job (runtime metrics)

| Aspect | Guarantee |
|---|---|
| Timeout | Grace 30 s on shutdown. |
| Retry | Pass error backoff: interval · 2^n, capped at 10 min. |
| Failure state | Job state `failing`, `blocked` or `paused` (gauge). |
| Recovery | Restart is safe: all claims live in PostgreSQL. |
| Audit evidence | Structured JSON logs: job, instance, `run_id`. No DSN, token or payload. |
| Metrics | `job_state_*`, `job_running`, `job_runs_succeeded_total`, `job_runs_failed_total`, `job_timeouts_total`, `job_last_success_timestamp_seconds`, `job_last_duration_seconds` |
| Status | **Implemented** |

## Not scheduled by design

- **Provider dispatch.** A PREPARED or CONFIRMED_ABSENT operation is dispatched only by an explicit, authorised command. A worker never places a provider order, activates a subscription or charges a wallet (D-35, D-38).
- **Fulfillment work-case events** (`fulfillment.work_case_*`). No consumer is defined. They remain undispatched and visible in the fulfillment outbox. They are not silently marked dispatched.
- **Manual payment write-back** (`payments.manual_payment`). This is financial: no worker delivers it without a separately approved contract. The commands stay PENDING.
