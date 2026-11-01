-- Per-module infrastructure for module "payments" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_payments NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA payments;
REVOKE ALL ON SCHEMA payments FROM PUBLIC;
GRANT USAGE ON SCHEMA payments TO crm_owner_payments;

CREATE TABLE payments.outbox_event (
    event_id         uuid        PRIMARY KEY,
    event_type       text        NOT NULL,
    event_version    integer     NOT NULL,
    subject_domain   text        NULL,
    subject_id       text        NULL,
    payload          jsonb       NOT NULL,
    occurred_at      timestamptz NOT NULL,
    recorded_at      timestamptz NOT NULL,
    correlation_id   text        NULL,
    dispatch_state   text        NOT NULL DEFAULT 'pending',
    attempts         integer     NOT NULL DEFAULT 0,
    last_attempt_at  timestamptz NULL,
    last_error       text        NULL,
    dispatched_at    timestamptz NULL,
    CONSTRAINT outbox_event_type_format CHECK (event_type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(event_type) <= 128),
    CONSTRAINT outbox_event_version_positive CHECK (event_version > 0),
    CONSTRAINT outbox_event_subject_pair CHECK ((subject_domain IS NULL) = (subject_id IS NULL)),
    CONSTRAINT outbox_event_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT outbox_event_correlation_length CHECK (correlation_id IS NULL OR length(correlation_id) BETWEEN 1 AND 128),
    CONSTRAINT outbox_event_state CHECK (dispatch_state IN ('pending', 'dispatched', 'failed')),
    CONSTRAINT outbox_event_attempts CHECK (attempts >= 0),
    CONSTRAINT outbox_event_error_length CHECK (last_error IS NULL OR length(last_error) <= 1000),
    CONSTRAINT outbox_event_dispatched CHECK ((dispatch_state = 'dispatched') = (dispatched_at IS NOT NULL))
);
CREATE INDEX outbox_event_undispatched_idx ON payments.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE payments.inbox_message (
    consumer         text        NOT NULL,
    message_id       uuid        NOT NULL,
    state            text        NOT NULL,
    first_seen_at    timestamptz NOT NULL,
    last_attempt_at  timestamptz NOT NULL,
    processed_at     timestamptz NULL,
    attempts         integer     NOT NULL,
    last_error       text        NULL,
    PRIMARY KEY (consumer, message_id),
    CONSTRAINT inbox_message_consumer_format CHECK (consumer ~ '^[a-z][a-z0-9_.]{1,127}$'),
    CONSTRAINT inbox_message_state CHECK (state IN ('processing', 'processed', 'failed')),
    CONSTRAINT inbox_message_processed CHECK ((state = 'processed') = (processed_at IS NOT NULL)),
    CONSTRAINT inbox_message_attempts CHECK (attempts >= 1),
    CONSTRAINT inbox_message_error_length CHECK (last_error IS NULL OR length(last_error) <= 1000)
);

CREATE TABLE payments.idempotency_record (
    scope            text        NOT NULL,
    idem_key         text        NOT NULL,
    request_hash     bytea       NOT NULL,
    state            text        NOT NULL,
    result           jsonb       NULL,
    created_at       timestamptz NOT NULL,
    completed_at     timestamptz NULL,
    PRIMARY KEY (scope, idem_key),
    CONSTRAINT idempotency_scope_format CHECK (scope ~ '^[a-z][a-z0-9_.]{1,63}$'),
    CONSTRAINT idempotency_key_format CHECK (idem_key ~ '^[A-Za-z0-9_-]{8,128}$'),
    CONSTRAINT idempotency_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT idempotency_state CHECK (state IN ('in_progress', 'completed')),
    CONSTRAINT idempotency_completed CHECK ((state = 'completed') = (completed_at IS NOT NULL)),
    CONSTRAINT idempotency_result_size CHECK (result IS NULL OR pg_column_size(result) <= 65536)
);

REVOKE ALL ON ALL TABLES IN SCHEMA payments FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON payments.outbox_event, payments.inbox_message, payments.idempotency_record TO crm_owner_payments;

-- Module-owned tables follow. Grants on them go to crm_owner_payments only.
-- Payments shadow model (Architecture V1 §5, §10, §15; Slice 7).
-- Materialised offline by the payments import command only; never written
-- by a read path. Invoice references are target invoice ids resolved through
-- the published billing API (no cross-schema foreign key); payments never
-- writes billing and billing never writes payments.
--
-- Attempt, gateway and receipt states are separate. Legacy 'approved' and
-- 'reconcile_error' stay raw; approval provenance is evidence
-- (GATEWAY_AUTO = gateway auto-approval, D-25; never Finance-reviewed).
-- A state change of an existing attempt appends to attempt_state_history
-- (append-only); historical attempts are never rewritten silently.
-- Gateway references are stored as SHA-256 only; receipts are reference
-- metadata only (host, extension, reference hash): no URL, path or file.

CREATE TABLE payments.payment_attempt (
    attempt_id            uuid        PRIMARY KEY,
    invoice_id            uuid        NULL,
    invoice_link_state    text        NOT NULL,
    source_system         text        NOT NULL,
    source_spec           text        NOT NULL,
    source_key            text        NOT NULL,
    first_batch_id        uuid        NOT NULL,
    last_batch_id         uuid        NOT NULL,
    row_sha256            bytea       NOT NULL,
    first_seen_at         timestamptz NOT NULL,
    last_seen_at          timestamptz NOT NULL,
    presence              text        NOT NULL,
    stage_no_raw          text        NULL,
    channel               text        NOT NULL,
    pay_method_raw        text        NULL,
    payment_source_raw    text        NULL,
    amount_raw            text        NULL,
    amount                bigint      NULL,
    unit                  text        NULL,
    amount_repr           text        NOT NULL,
    state_namespace       text        NOT NULL,
    state_raw             text        NULL,
    state_mapping         text        NOT NULL,
    state_target          text        NULL,
    approval_provenance   text        NOT NULL,
    uploaded_by_type_raw  text        NULL,
    uploaded_by_legacy_key text       NULL,
    uploaded_by_user_id   uuid        NULL,
    created_at_raw        text        NULL,
    updated_at_raw        text        NULL,
    version               bigint      NOT NULL,
    CONSTRAINT attempt_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT attempt_sha_length CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT attempt_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT attempt_invoice_link CHECK (invoice_link_state IN ('RESOLVED', 'UNKNOWN')),
    CONSTRAINT attempt_invoice_resolved CHECK ((invoice_link_state = 'RESOLVED') = (invoice_id IS NOT NULL)),
    CONSTRAINT attempt_channel CHECK (channel IN ('GATEWAY', 'MANUAL_CARD', 'UNKNOWN')),
    CONSTRAINT attempt_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT')),
    CONSTRAINT attempt_repr CHECK (amount_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')),
    CONSTRAINT attempt_amount_iff CHECK ((amount_repr = 'REPRESENTABLE') = (amount IS NOT NULL AND unit IS NOT NULL)),
    CONSTRAINT attempt_state_mapping CHECK (state_mapping IN ('MAPPED', 'UNKNOWN', 'EMPTY')),
    CONSTRAINT attempt_state_target CHECK ((state_mapping = 'MAPPED') = (state_target IS NOT NULL)),
    CONSTRAINT attempt_state_vocab CHECK (state_target IS NULL OR state_target IN ('PENDING', 'ACCEPTED', 'REJECTED', 'CANCELLED', 'RECONCILIATION_REQUIRED')),
    CONSTRAINT attempt_provenance CHECK (approval_provenance IN ('GATEWAY_AUTO', 'LEGACY_MANUAL_APPROVAL', 'NOT_APPROVED', 'UNKNOWN')),
    CONSTRAINT attempt_version CHECK (version > 0)
);
CREATE INDEX payment_attempt_invoice_idx ON payments.payment_attempt (invoice_id, attempt_id);

CREATE TABLE payments.gateway_attempt (
    gateway_attempt_id  uuid  PRIMARY KEY,
    attempt_id          uuid  NOT NULL UNIQUE REFERENCES payments.payment_attempt (attempt_id),
    provider_raw        text  NULL,
    authority_sha256    bytea NULL,
    ref_sha256          bytea NULL,
    state_raw           text  NULL,
    state_mapping       text  NOT NULL,
    state_target        text  NULL,
    CONSTRAINT gateway_authority_len CHECK (authority_sha256 IS NULL OR octet_length(authority_sha256) = 32),
    CONSTRAINT gateway_ref_len CHECK (ref_sha256 IS NULL OR octet_length(ref_sha256) = 32),
    CONSTRAINT gateway_state_mapping CHECK (state_mapping IN ('MAPPED', 'UNKNOWN', 'EMPTY')),
    CONSTRAINT gateway_state_target CHECK ((state_mapping = 'MAPPED') = (state_target IS NOT NULL)),
    CONSTRAINT gateway_state_vocab CHECK (state_target IS NULL OR state_target IN ('INITIATED', 'VERIFIED', 'CANCELLED_AT_GATEWAY'))
);

-- Append-only: every observed raw state change of an existing attempt.
CREATE TABLE payments.attempt_state_history (
    history_id     uuid        PRIMARY KEY,
    attempt_id     uuid        NOT NULL REFERENCES payments.payment_attempt (attempt_id),
    old_state_raw  text        NULL,
    new_state_raw  text        NULL,
    regression     boolean     NOT NULL,
    batch_id       uuid        NOT NULL,
    recorded_at    timestamptz NOT NULL
);
CREATE INDEX attempt_state_history_idx ON payments.attempt_state_history (attempt_id, recorded_at);

CREATE TABLE payments.receipt (
    receipt_id      uuid        PRIMARY KEY,
    receipt_key     bytea       NOT NULL UNIQUE,
    invoice_id      uuid        NULL,
    stage_no_raw    text        NULL,
    owner_kind      text        NOT NULL,
    source_system   text        NOT NULL,
    source_spec     text        NOT NULL,
    source_key      text        NOT NULL,
    ref_index       integer     NOT NULL,
    ref_sha256      bytea       NULL,
    host            text        NULL,
    extension       text        NULL,
    state           text        NOT NULL,
    first_batch_id  uuid        NOT NULL,
    last_batch_id   uuid        NOT NULL,
    first_seen_at   timestamptz NOT NULL,
    last_seen_at    timestamptz NOT NULL,
    presence        text        NOT NULL,
    CONSTRAINT receipt_key_len CHECK (octet_length(receipt_key) = 32),
    CONSTRAINT receipt_owner_kind CHECK (owner_kind IN ('STAGE', 'INVOICE')),
    CONSTRAINT receipt_state CHECK (state IN ('REFERENCE_ONLY', 'AVAILABLE', 'MISSING', 'NOT_RECOVERABLE', 'UNKNOWN')),
    CONSTRAINT receipt_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT receipt_ref_len CHECK (ref_sha256 IS NULL OR octet_length(ref_sha256) = 32)
);
CREATE INDEX receipt_invoice_idx ON payments.receipt (invoice_id, receipt_id);

-- run_seq orders runs deterministically (recorded_at may tie): the latest
-- run decides whether the attempt source was COMPLETE.
CREATE TABLE payments.import_run (
    run_id       uuid        PRIMARY KEY,
    run_seq      bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    batch_id     uuid        NOT NULL,
    snapshot_id  text        NOT NULL,
    outcome      text        NOT NULL,
    replay       boolean     NOT NULL,
    counts       jsonb       NOT NULL,
    recorded_at  timestamptz NOT NULL,
    CONSTRAINT import_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED')),
    CONSTRAINT import_run_counts_object CHECK (jsonb_typeof(counts) = 'object')
);

-- Append-only guards for attempt_state_history are installed by
-- 00002_payments_append_only_guards.sql.

-- No DELETE grant anywhere.
GRANT SELECT, INSERT, UPDATE ON payments.payment_attempt, payments.gateway_attempt, payments.receipt TO crm_owner_payments;
GRANT SELECT, INSERT ON payments.attempt_state_history, payments.import_run TO crm_owner_payments;

-- +goose Down
DROP SCHEMA payments CASCADE;
