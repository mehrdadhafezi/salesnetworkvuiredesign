-- Per-module infrastructure for module "customers" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_customers NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA customers;
REVOKE ALL ON SCHEMA customers FROM PUBLIC;
GRANT USAGE ON SCHEMA customers TO crm_owner_customers;

CREATE TABLE customers.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON customers.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE customers.inbox_message (
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

CREATE TABLE customers.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA customers FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON customers.outbox_event, customers.inbox_message, customers.idempotency_record TO crm_owner_customers;

-- Module-owned tables follow. Grants on them go to crm_owner_customers only.
-- Customers shadow model (Architecture V1 §5, §7, §10, §25; Slice 5).
-- Materialised offline by the customers import command only; never written
-- by a read path. One customer is minted per legacy issued-to subject; no
-- merge is ever executed: there is no merge command, no "merged into"
-- column, and the owner role holds no DELETE grant on any table, so a
-- customer cannot be collapsed into another even by its owner.
--
-- Phone is NEVER identity (§10, D-29/D-30/D-31): there is no unique
-- constraint and no key on a normalized contact value; the same phone may
-- belong to several customers. Normalized values are comparison evidence
-- only. Identifiers (legacy account ids, external ids) are never keys.
-- Evidence content is append-only; only its presence markers change.

CREATE TABLE customers.customer (
    customer_id        uuid        PRIMARY KEY,
    subject_kind       text        NOT NULL,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    source_row_sha256  bytea       NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    name_evidence      text        NULL,
    observed_raw       text        NULL,
    recoverable        boolean     NOT NULL,
    resolution_state   text        NOT NULL,
    resolution_reason  text        NOT NULL,
    version            bigint      NOT NULL,
    CONSTRAINT customer_source_unique UNIQUE (source_system, subject_kind, source_key),
    CONSTRAINT customer_subject_kind CHECK (subject_kind IN ('INVOICE_ISSUED_TO', 'SYNTHETIC_SUBJECT')),
    CONSTRAINT customer_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT customer_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT customer_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT customer_resolution_state CHECK (resolution_state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT customer_resolution_reason_length CHECK (length(resolution_reason) BETWEEN 1 AND 300),
    CONSTRAINT customer_version CHECK (version > 0)
);
CREATE INDEX customer_state_idx ON customers.customer (resolution_state, customer_id);

-- Contact points: many per customer, raw value always kept. The normalized
-- value is a NON-UNIQUE comparison index; equal values never imply one owner.
CREATE TABLE customers.contact_point (
    contact_point_id   uuid        PRIMARY KEY,
    customer_id        uuid        NOT NULL REFERENCES customers.customer (customer_id),
    kind               text        NOT NULL,
    role               text        NOT NULL,
    raw_value          text        NOT NULL,
    normalized_value   text        NULL,
    normalization      text        NOT NULL,
    valid_from_raw     text        NULL,
    valid_to_raw       text        NULL,
    observed_raw       text        NULL,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    CONSTRAINT contact_point_identity UNIQUE (customer_id, kind, role, raw_value),
    CONSTRAINT contact_point_kind CHECK (kind IN ('PHONE', 'EMAIL')),
    CONSTRAINT contact_point_role CHECK (role IN ('PRIMARY', 'SECONDARY')),
    CONSTRAINT contact_point_normalization CHECK (normalization IN ('VALID', 'INVALID', 'EMPTY')),
    CONSTRAINT contact_point_normalized CHECK ((normalization = 'VALID') = (normalized_value IS NOT NULL)),
    CONSTRAINT contact_point_raw_length CHECK (length(raw_value) <= 200),
    CONSTRAINT contact_point_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX contact_point_customer_idx ON customers.contact_point (customer_id);
CREATE INDEX contact_point_lookup_idx ON customers.contact_point (kind, normalized_value) WHERE normalized_value IS NOT NULL;

-- External / legacy identifiers: evidence attached to a customer, never a
-- key. The same (scheme, value) on several customers is detected and
-- reported (CONFLICT for exclusive schemes), never silently reassigned.
CREATE TABLE customers.customer_identifier (
    identifier_id      uuid        PRIMARY KEY,
    customer_id        uuid        NOT NULL REFERENCES customers.customer (customer_id),
    scheme             text        NOT NULL,
    external_value     text        NOT NULL,
    derivation         text        NOT NULL,
    status             text        NOT NULL,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    CONSTRAINT customer_identifier_identity UNIQUE (customer_id, scheme, external_value),
    CONSTRAINT customer_identifier_scheme CHECK (scheme IN ('LEGACY_WP_ACCOUNT', 'WOO_CUSTOMER', 'BIAWIN', 'LEGACY_AGGREGATE_V1', 'LEGACY_MAP_V2', 'CRM_CODE')),
    CONSTRAINT customer_identifier_derivation CHECK (derivation IN ('DIRECT', 'PHONE_DERIVED')),
    CONSTRAINT customer_identifier_status CHECK (status IN ('ACTIVE', 'CONFLICT')),
    CONSTRAINT customer_identifier_value_length CHECK (length(external_value) BETWEEN 1 AND 120),
    CONSTRAINT customer_identifier_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX customer_identifier_lookup_idx ON customers.customer_identifier (scheme, external_value);

-- Identity evidence: append-only content (deduplicated by a deterministic
-- key). Only presence/last-seen markers may change.
CREATE TABLE customers.identity_evidence (
    evidence_id        uuid        PRIMARY KEY,
    evidence_key       bytea       NOT NULL UNIQUE,
    kind               text        NOT NULL,
    strength           text        NOT NULL,
    polarity           text        NOT NULL,
    origin             text        NOT NULL,
    reference          text        NOT NULL,
    observed_raw       text        NULL,
    -- Normalized contact a supplied OTP_CONTACT_CONTROL record is about, so
    -- the case projection can be rebuilt from stored evidence. Internal
    -- only: never exposed by any read model; never a key.
    contact_normalized text        NULL,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NULL,
    first_batch_id     uuid        NOT NULL,
    recorded_at        timestamptz NOT NULL,
    last_seen_batch_id uuid        NOT NULL,
    last_seen_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    CONSTRAINT identity_evidence_key_length CHECK (octet_length(evidence_key) = 32),
    CONSTRAINT identity_evidence_kind CHECK (kind IN ('PHONE_MATCH', 'ACCOUNT_REF_MATCH', 'SEQUENTIAL_PHONE_REUSE',
        'OVERLAPPING_PHONE_OWNERSHIP', 'EXCLUSIVE_IDENTIFIER_SHARED', 'OTP_CONTACT_CONTROL', 'VERIFIED_EQUALITY',
        'VERIFIED_DISTINCT', 'SOURCE_NOT_RECOVERABLE')),
    CONSTRAINT identity_evidence_strength CHECK (strength IN ('STRONG', 'MEDIUM', 'WEAK')),
    CONSTRAINT identity_evidence_polarity CHECK (polarity IN ('EQUALITY', 'RELATION', 'DISCOVERY', 'CONTRADICTION', 'MISSING')),
    CONSTRAINT identity_evidence_origin CHECK (origin IN ('DERIVED', 'SUPPLIED')),
    CONSTRAINT identity_evidence_reference_length CHECK (length(reference) BETWEEN 1 AND 200),
    CONSTRAINT identity_evidence_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);

CREATE TABLE customers.evidence_subject (
    evidence_id        uuid        NOT NULL REFERENCES customers.identity_evidence (evidence_id),
    customer_id        uuid        NOT NULL REFERENCES customers.customer (customer_id),
    PRIMARY KEY (evidence_id, customer_id)
);
CREATE INDEX evidence_subject_customer_idx ON customers.evidence_subject (customer_id);

-- Merge cases: read/queue candidates only. A state is a resolver projection
-- of the case's evidence; RESOLVED never executes anything. There is no
-- winner, canonical or "merged into" column.
CREATE TABLE customers.merge_case (
    merge_case_id      uuid        PRIMARY KEY,
    case_key           bytea       NOT NULL UNIQUE,
    reason             text        NOT NULL,
    state              text        NOT NULL,
    state_reason       text        NOT NULL,
    max_strength       text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    created_at         timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    version            bigint      NOT NULL,
    CONSTRAINT merge_case_key_length CHECK (octet_length(case_key) = 32),
    CONSTRAINT merge_case_reason CHECK (reason IN ('SHARED_PHONE', 'SHARED_ACCOUNT_REF', 'SHARED_EXCLUSIVE_IDENTIFIER', 'SUPPLIED_EVIDENCE')),
    CONSTRAINT merge_case_state CHECK (state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT merge_case_strength CHECK (max_strength IN ('STRONG', 'MEDIUM', 'WEAK')),
    CONSTRAINT merge_case_state_reason_length CHECK (length(state_reason) BETWEEN 1 AND 300),
    CONSTRAINT merge_case_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT merge_case_version CHECK (version > 0)
);
CREATE INDEX merge_case_state_idx ON customers.merge_case (state, merge_case_id);

CREATE TABLE customers.merge_case_member (
    merge_case_id      uuid        NOT NULL REFERENCES customers.merge_case (merge_case_id),
    customer_id        uuid        NOT NULL REFERENCES customers.customer (customer_id),
    presence           text        NOT NULL,
    PRIMARY KEY (merge_case_id, customer_id),
    CONSTRAINT merge_case_member_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX merge_case_member_customer_idx ON customers.merge_case_member (customer_id);

CREATE TABLE customers.merge_case_evidence (
    merge_case_id      uuid        NOT NULL REFERENCES customers.merge_case (merge_case_id),
    evidence_id        uuid        NOT NULL REFERENCES customers.identity_evidence (evidence_id),
    presence           text        NOT NULL,
    PRIMARY KEY (merge_case_id, evidence_id),
    CONSTRAINT merge_case_evidence_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);

-- Append-only projection history: a future merge/unmerge appends here and
-- to the evidence; nothing is rewritten.
CREATE TABLE customers.merge_case_state_history (
    merge_case_id      uuid        NOT NULL REFERENCES customers.merge_case (merge_case_id),
    run_id             uuid        NOT NULL,
    state              text        NOT NULL,
    state_reason       text        NOT NULL,
    recorded_at        timestamptz NOT NULL,
    PRIMARY KEY (merge_case_id, run_id)
);

CREATE TABLE customers.customer_resolution_history (
    customer_id        uuid        NOT NULL REFERENCES customers.customer (customer_id),
    run_id             uuid        NOT NULL,
    state              text        NOT NULL,
    reason             text        NOT NULL,
    recorded_at        timestamptz NOT NULL,
    PRIMARY KEY (customer_id, run_id)
);

CREATE TABLE customers.import_run (
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

-- No DELETE grant anywhere: absence is a presence marker, never a delete.
GRANT SELECT, INSERT, UPDATE ON customers.customer, customers.contact_point, customers.customer_identifier,
    customers.merge_case, customers.merge_case_member, customers.merge_case_evidence TO crm_owner_customers;
GRANT SELECT, INSERT ON customers.identity_evidence, customers.evidence_subject, customers.merge_case_state_history,
    customers.customer_resolution_history, customers.import_run TO crm_owner_customers;
GRANT UPDATE (last_seen_batch_id, last_seen_at, presence) ON customers.identity_evidence TO crm_owner_customers;

-- +goose Down
DROP SCHEMA customers CASCADE;
