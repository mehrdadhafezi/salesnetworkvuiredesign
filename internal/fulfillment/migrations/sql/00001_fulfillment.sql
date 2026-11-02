-- Per-module infrastructure for module "fulfillment" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_fulfillment NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA fulfillment;
REVOKE ALL ON SCHEMA fulfillment FROM PUBLIC;
GRANT USAGE ON SCHEMA fulfillment TO crm_owner_fulfillment;

CREATE TABLE fulfillment.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON fulfillment.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE fulfillment.inbox_message (
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

CREATE TABLE fulfillment.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA fulfillment FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON fulfillment.outbox_event, fulfillment.inbox_message, fulfillment.idempotency_record TO crm_owner_fulfillment;

-- Module-owned tables follow. Grants on them go to crm_owner_fulfillment only.
-- Fulfillment READ / SHADOW model (Architecture V1 §13; Slice 10). Built
-- only by the offline shadow import from normalized legacy evidence; no
-- other schema is read or written (subscriptions / items / invoices are
-- referenced by stable ids, never joined). Every table is INSERT-only
-- (grants + 00002 guards): runs are never rewritten, list and detail always
-- come from one run, and history events are never overwritten. No
-- assignment, completion, shipping, upgrade or evidence command exists.
-- No PII column: no phone, name, address, postal code, tracking or
-- activation code (external references only as SHA-256).

CREATE TABLE fulfillment.import_run (
    run_id                uuid        PRIMARY KEY,
    run_seq               bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    record_kind           text        NOT NULL DEFAULT 'SHADOW',
    normalization_version integer     NOT NULL,
    snapshot_id           text        NOT NULL,
    batch_id              uuid        NOT NULL,
    source_cutoff         timestamptz NOT NULL,
    started_at            timestamptz NOT NULL,
    completed_at          timestamptz NOT NULL,
    outcome               text        NOT NULL,
    reasons               jsonb       NOT NULL,
    counts                jsonb       NOT NULL,
    input_sha256          bytea       NOT NULL,
    facts_sha256          bytea       NOT NULL,
    CONSTRAINT import_run_kind CHECK (record_kind = 'SHADOW'),
    CONSTRAINT import_run_version CHECK (normalization_version > 0),
    CONSTRAINT import_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL')),
    CONSTRAINT import_run_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT import_run_counts CHECK (jsonb_typeof(counts) = 'object'),
    CONSTRAINT import_run_shas CHECK (octet_length(input_sha256) = 32 AND octet_length(facts_sha256) = 32),
    CONSTRAINT import_run_order CHECK (started_at <= completed_at),
    CONSTRAINT import_run_replay UNIQUE (normalization_version, input_sha256)
);

CREATE TABLE fulfillment.run_source (
    run_id     uuid    NOT NULL REFERENCES fulfillment.import_run (run_id),
    source     text    NOT NULL,
    status     text    NOT NULL,
    rows_read  bigint  NOT NULL,
    PRIMARY KEY (run_id, source),
    CONSTRAINT run_source_status CHECK (status IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE')),
    CONSTRAINT run_source_rows CHECK (rows_read >= 0)
);

-- One fulfillment case: operational execution only (never financial
-- truth). Kind explicit; raw status kept; mapped only where proven.
CREATE TABLE fulfillment.fulfillment_case (
    run_id                uuid   NOT NULL REFERENCES fulfillment.import_run (run_id),
    case_id               uuid   NOT NULL,
    kind                  text   NOT NULL,
    source_system         text   NOT NULL,
    source_spec           text   NOT NULL,
    source_key            text   NOT NULL,
    subtype_raw           text   NULL,
    subscription_id       uuid   NULL,
    item_id               uuid   NULL,
    invoice_link_state    text   NOT NULL,
    invoice_id            uuid   NULL,
    invoice_key           text   NULL,
    status_namespace      text   NOT NULL,
    status_raw            text   NULL,
    status_mapped         text   NOT NULL,
    shipping_status_raw   text   NULL,
    provider_status_raw   text   NULL,
    contact_status_raw    text   NULL,
    completed_at_raw      text   NULL,
    completed_by_key      text   NULL,
    completed_by_user_id  uuid   NULL,
    cancelled_at_raw      text   NULL,
    shipped_at_raw        text   NULL,
    delivered_at_raw      text   NULL,
    status_updated_at_raw text   NULL,
    carrier_raw           text   NULL,
    tracking_ref_sha256   bytea  NULL,
    provider_ref_sha256   bytea  NULL,
    provider_ref_scheme   text   NULL,
    created_at_raw        text   NULL,
    updated_at_raw        text   NULL,
    PRIMARY KEY (run_id, case_id),
    CONSTRAINT fulfillment_case_source UNIQUE (run_id, source_system, source_spec, source_key),
    CONSTRAINT fulfillment_case_kind CHECK (kind IN ('SERVICE_FOLLOWUP', 'EXECUTION', 'UPGRADE_EXECUTION', 'SHIPPING')),
    CONSTRAINT fulfillment_case_status CHECK (status_mapped IN ('NOT_STARTED', 'IN_PROGRESS', 'AWAITING_PAYMENT', 'AWAITING_PROVIDER',
        'PROVIDER_ACTION_RECORDED', 'OUTCOME_RECORDED', 'SHIPPED', 'DELIVERED', 'RETURNED', 'NOT_EXECUTED', 'CANCELLED', 'COMPLETED',
        'FAILED_AT_PROVIDER', 'UNKNOWN', 'EMPTY')),
    CONSTRAINT fulfillment_case_unknown_keeps_raw CHECK (status_mapped <> 'UNKNOWN' OR status_raw IS NOT NULL),
    CONSTRAINT fulfillment_case_invoice CHECK (invoice_link_state IN ('RESOLVED', 'UNKNOWN') AND (invoice_id IS NOT NULL) = (invoice_link_state = 'RESOLVED')),
    CONSTRAINT fulfillment_case_refs CHECK ((tracking_ref_sha256 IS NULL OR octet_length(tracking_ref_sha256) = 32)
        AND (provider_ref_sha256 IS NULL OR octet_length(provider_ref_sha256) = 32)
        AND ((provider_ref_sha256 IS NULL) = (provider_ref_scheme IS NULL))),
    CONSTRAINT fulfillment_case_item_needs_subscription CHECK (item_id IS NULL OR subscription_id IS NOT NULL)
);
CREATE INDEX fulfillment_case_subscription_idx ON fulfillment.fulfillment_case (run_id, subscription_id);

-- Current assignees as evidenced at the run (one row per role; never an
-- owner, credit owner, customer or subscription owner).
CREATE TABLE fulfillment.assignment (
    run_id           uuid  NOT NULL,
    case_id          uuid  NOT NULL,
    role             text  NOT NULL,
    user_key         text  NOT NULL,
    user_id          uuid  NULL,
    assigned_at_raw  text  NULL,
    PRIMARY KEY (run_id, case_id, role),
    CONSTRAINT assignment_case FOREIGN KEY (run_id, case_id) REFERENCES fulfillment.fulfillment_case (run_id, case_id),
    CONSTRAINT assignment_role CHECK (role IN ('MANAGER', 'EXPERT', 'HANDLER'))
);

-- Service evidence METADATA (no content, no file) and its outcome.
CREATE TABLE fulfillment.service_evidence (
    run_id               uuid  NOT NULL,
    source_key           text  NOT NULL,
    case_id              uuid  NOT NULL,
    form_definition_key  text  NULL,
    submitted_by_key     text  NULL,
    submitted_by_user_id uuid  NULL,
    submitted_at_raw     text  NULL,
    outcome_raw          text  NULL,
    outcome_mapped       text  NOT NULL,
    decided_by_key       text  NULL,
    decided_at_raw       text  NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT service_evidence_case FOREIGN KEY (run_id, case_id) REFERENCES fulfillment.fulfillment_case (run_id, case_id),
    CONSTRAINT service_evidence_outcome CHECK (outcome_mapped IN ('EXECUTED', 'NOT_EXECUTED', 'UNKNOWN', 'NONE')),
    CONSTRAINT service_evidence_unknown_keeps_raw CHECK (outcome_mapped <> 'UNKNOWN' OR outcome_raw IS NOT NULL)
);

-- Case history (append-only facts of the run; never rewritten).
CREATE TABLE fulfillment.case_event (
    run_id           uuid  NOT NULL,
    source_spec      text  NOT NULL,
    source_key       text  NOT NULL,
    case_id          uuid  NOT NULL,
    actor_key        text  NULL,
    actor_user_id    uuid  NULL,
    event_key_raw    text  NOT NULL,
    old_status_raw   text  NULL,
    new_status_raw   text  NULL,
    occurred_at_raw  text  NULL,
    PRIMARY KEY (run_id, source_spec, source_key),
    CONSTRAINT case_event_case FOREIGN KEY (run_id, case_id) REFERENCES fulfillment.fulfillment_case (run_id, case_id)
);
CREATE INDEX case_event_case_idx ON fulfillment.case_event (run_id, case_id);

GRANT SELECT, INSERT ON fulfillment.import_run, fulfillment.run_source, fulfillment.fulfillment_case, fulfillment.assignment,
    fulfillment.service_evidence, fulfillment.case_event TO crm_owner_fulfillment;

-- +goose Down
DROP SCHEMA fulfillment CASCADE;
