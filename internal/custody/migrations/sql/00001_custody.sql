-- Per-module infrastructure for module "custody" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_custody NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA custody;
REVOKE ALL ON SCHEMA custody FROM PUBLIC;
GRANT USAGE ON SCHEMA custody TO crm_owner_custody;

CREATE TABLE custody.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON custody.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE custody.inbox_message (
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

CREATE TABLE custody.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA custody FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON custody.outbox_event, custody.inbox_message, custody.idempotency_record TO crm_owner_custody;

-- Module-owned tables follow. Grants on them go to crm_owner_custody only.
-- Custody shadow model (Architecture V1 §5, §7, §10, §25; Slice 6).
-- CustodyEvent is append-only (FPR; E0 §10): one row per legacy log row,
-- never updated or deleted (no grant; a trigger rejects either even for the
-- owner). The legacy action is preserved raw; its kind comes only from an
-- explicit, evidence-backed mapping, otherwise UNKNOWN.
--
-- Assignment is a projection DERIVED from the events, stored next to the
-- legacy current-holder evidence and their parity class. It is refreshed by
-- the import only and never the sole source of truth. Findings report gaps,
-- contradictions and mismatches; nothing is auto-repaired.
--
-- Case references are target case ids resolved through the published cases
-- API; there is no cross-schema foreign key and no phone anywhere.

CREATE TABLE custody.custody_event (
    event_id          uuid        PRIMARY KEY,
    case_id           uuid        NOT NULL,
    source_system     text        NOT NULL,
    source_spec       text        NOT NULL,
    source_key        text        NOT NULL,
    source_ordinal    numeric(20) NOT NULL,
    legacy_action     text        NOT NULL,
    kind              text        NOT NULL,
    context           text        NOT NULL,
    actor_legacy_key  text        NULL,
    actor_user_id     uuid        NULL,
    from_legacy_key   text        NULL,
    from_user_id      uuid        NULL,
    to_legacy_key     text        NULL,
    to_user_id        uuid        NULL,
    from_position_raw text        NULL,
    to_position_raw   text        NULL,
    step_order_raw    text        NULL,
    occurred_raw      text        NULL,
    row_sha256        bytea       NOT NULL,
    first_batch_id    uuid        NOT NULL,
    recorded_at       timestamptz NOT NULL,
    CONSTRAINT custody_event_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT custody_event_kind CHECK (kind IN ('HOLDER_ASSIGNED', 'HOLDER_RELEASED', 'NO_HOLDER_CHANGE', 'UNKNOWN')),
    CONSTRAINT custody_event_context CHECK (context IN ('DISTRIBUTION', 'MIS', 'PAYMENT_WORKFLOW', 'REFERRAL', 'CONVERSION', 'SELLER_UPDATE', 'UNKNOWN')),
    CONSTRAINT custody_event_action_length CHECK (length(legacy_action) BETWEEN 1 AND 100),
    CONSTRAINT custody_event_sha_length CHECK (octet_length(row_sha256) = 32)
);
CREATE INDEX custody_event_case_idx ON custody.custody_event (case_id, source_ordinal);

-- Derived, reconcilable projection: one row per Case with custody evidence.
CREATE TABLE custody.assignment (
    assignment_id          uuid        PRIMARY KEY,
    case_id                uuid        NOT NULL UNIQUE,
    derivation             text        NOT NULL,
    derived_legacy_key     text        NULL,
    derived_user_id        uuid        NULL,
    derived_position_raw   text        NULL,
    legacy_holder_key      text        NULL,
    legacy_holder_user_id  uuid        NULL,
    legacy_position_raw    text        NULL,
    legacy_source_spec     text        NOT NULL,
    parity                 text        NOT NULL,
    event_count            integer     NOT NULL,
    last_event_id          uuid        NULL,
    presence               text        NOT NULL,
    refreshed_batch_id     uuid        NOT NULL,
    refreshed_at           timestamptz NOT NULL,
    CONSTRAINT assignment_derivation CHECK (derivation IN ('DERIVED', 'RELEASED', 'UNKNOWN', 'MULTIPLE', 'NONE')),
    CONSTRAINT assignment_derived_has_holder CHECK (derivation <> 'DERIVED' OR derived_legacy_key IS NOT NULL),
    CONSTRAINT assignment_parity CHECK (parity IN ('MATCH', 'TARGET_UNKNOWN', 'LEGACY_UNKNOWN', 'CONFLICT', 'MULTIPLE_CURRENT', 'HISTORY_GAP', 'NOT_RECOVERABLE')),
    CONSTRAINT assignment_event_count CHECK (event_count >= 0),
    CONSTRAINT assignment_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX assignment_parity_idx ON custody.assignment (parity, case_id);

CREATE TABLE custody.finding (
    finding_id         uuid        PRIMARY KEY,
    finding_key        bytea       NOT NULL UNIQUE,
    case_id            uuid        NOT NULL,
    kind               text        NOT NULL,
    detail             text        NOT NULL,
    event_id           uuid        NULL REFERENCES custody.custody_event (event_id),
    presence           text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    last_seen_batch_id uuid        NOT NULL,
    recorded_at        timestamptz NOT NULL,
    CONSTRAINT finding_key_length CHECK (octet_length(finding_key) = 32),
    CONSTRAINT finding_kind CHECK (kind IN ('MISSING_PRIOR_OWNER', 'MISSING_TARGET_OWNER', 'HISTORY_GAP', 'MULTIPLE_CURRENT',
        'ORDERING_CONTRADICTION', 'MISSING_EVENT_HISTORY', 'SOURCE_CURRENT_MISMATCH', 'UNKNOWN_ACTION')),
    CONSTRAINT finding_detail_length CHECK (length(detail) BETWEEN 1 AND 300),
    CONSTRAINT finding_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX finding_case_idx ON custody.finding (case_id);

CREATE TABLE custody.import_run (
    run_id       uuid        PRIMARY KEY,
    batch_id     uuid        NOT NULL,
    snapshot_id  text        NOT NULL,
    outcome      text        NOT NULL,
    replay       boolean     NOT NULL,
    counts       jsonb       NOT NULL,
    recorded_at  timestamptz NOT NULL,
    CONSTRAINT import_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED')),
    CONSTRAINT import_run_counts_object CHECK (jsonb_typeof(counts) = 'object')
);

-- Append-only guards (UPDATE / DELETE / TRUNCATE rejected even for the
-- owner) are installed by 00002_custody_append_only_guards.sql.

-- No DELETE grant anywhere; events are INSERT-only.
GRANT SELECT, INSERT ON custody.custody_event, custody.import_run TO crm_owner_custody;
GRANT SELECT, INSERT, UPDATE ON custody.assignment TO crm_owner_custody;
GRANT SELECT, INSERT ON custody.finding TO crm_owner_custody;
GRANT UPDATE (presence, last_seen_batch_id) ON custody.finding TO crm_owner_custody;

-- +goose Down
DROP SCHEMA custody CASCADE;
