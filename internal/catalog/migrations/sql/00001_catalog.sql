-- Per-module infrastructure for module "catalog" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_catalog NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA catalog;
REVOKE ALL ON SCHEMA catalog FROM PUBLIC;
GRANT USAGE ON SCHEMA catalog TO crm_owner_catalog;

CREATE TABLE catalog.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON catalog.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE catalog.inbox_message (
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

CREATE TABLE catalog.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA catalog FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON catalog.outbox_event, catalog.inbox_message, catalog.idempotency_record TO crm_owner_catalog;

-- Module-owned tables follow. Grants on them go to crm_owner_catalog only.
-- Catalog service model (Architecture V1 §13; Slice 14). Written only by
-- the offline administrative recorder with an external Human Owner approval
-- reference; read by biawin and fulfillment through the catalog published
-- API. Every table is INSERT-only (grants + 00002 guards): a definition
-- change is a new version, an ownership change is a new scope event, and
-- history (who was responsible when) is never rewritten.
-- No PII column; no price, credit or money column.

CREATE TABLE catalog.service_version (
    service_key          text        NOT NULL,
    version              integer     NOT NULL,
    title                text        NOT NULL,
    workflow_key         text        NOT NULL,
    evidence_requirements jsonb      NOT NULL,
    completion_outcomes  text[]      NOT NULL,
    financial_behaviour  text        NOT NULL,
    integration_provider text        NOT NULL,
    status               text        NOT NULL,
    approval_ref         text        NOT NULL,
    recorded_by          text        NOT NULL,
    recorded_at          timestamptz NOT NULL,
    PRIMARY KEY (service_key, version),
    CONSTRAINT service_version_key CHECK (service_key ~ '^[a-z][a-z0-9_.-]{1,80}$'),
    CONSTRAINT service_version_positive CHECK (version > 0),
    CONSTRAINT service_version_workflow CHECK (workflow_key IN ('operational_task_v1')),
    CONSTRAINT service_version_evidence CHECK (jsonb_typeof(evidence_requirements) = 'array'),
    CONSTRAINT service_version_outcomes CHECK (cardinality(completion_outcomes) > 0
        AND completion_outcomes <@ ARRAY['EXECUTED', 'NOT_EXECUTED']::text[]),
    CONSTRAINT service_version_financial CHECK (financial_behaviour IN ('NONE', 'CONSUMES_CREDIT', 'CHARGES_LEDGER')),
    CONSTRAINT service_version_provider CHECK (integration_provider IN ('NONE', 'BIAWIN')),
    CONSTRAINT service_version_status CHECK (status IN ('ACTIVE', 'RETIRED')),
    CONSTRAINT service_version_approval CHECK (length(approval_ref) BETWEEN 3 AND 200),
    CONSTRAINT service_version_recorder CHECK (length(recorded_by) BETWEEN 1 AND 128)
);

-- The service components of a subscription product version (one product
-- may carry 1 … 30+ services; each becomes its own SubscriptionItem).
CREATE TABLE catalog.subscription_component (
    product_key      text        NOT NULL,
    product_version  integer     NOT NULL,
    seq              integer     NOT NULL,
    service_key      text        NOT NULL,
    service_version  integer     NOT NULL,
    approval_ref     text        NOT NULL,
    recorded_by      text        NOT NULL,
    recorded_at      timestamptz NOT NULL,
    PRIMARY KEY (product_key, product_version, seq),
    CONSTRAINT subscription_component_service FOREIGN KEY (service_key, service_version) REFERENCES catalog.service_version (service_key, version),
    CONSTRAINT subscription_component_seq CHECK (seq > 0 AND product_version > 0),
    CONSTRAINT subscription_component_approval CHECK (length(approval_ref) BETWEEN 3 AND 200)
);

-- Per-service responsibility scope events. A GRANT makes a user the
-- supervisor (or an executor) of one service; a REVOKE ends it. The current
-- scope is derived; past events are never rewritten, so a changed
-- supervisor never rewrites historical assignments or outcomes. A scope is
-- row-level authority for ONE service only, never a role or hierarchy tier.
CREATE TABLE catalog.service_scope_event (
    event_id      uuid        PRIMARY KEY,
    seq           bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    service_key   text        NOT NULL,
    user_id       uuid        NOT NULL,
    scope_kind    text        NOT NULL,
    action        text        NOT NULL,
    approval_ref  text        NOT NULL,
    recorded_by   text        NOT NULL,
    recorded_at   timestamptz NOT NULL,
    CONSTRAINT service_scope_kind CHECK (scope_kind IN ('SUPERVISOR', 'EXECUTOR')),
    CONSTRAINT service_scope_action CHECK (action IN ('GRANT', 'REVOKE')),
    CONSTRAINT service_scope_approval CHECK (length(approval_ref) BETWEEN 3 AND 200)
);
CREATE INDEX service_scope_event_lookup ON catalog.service_scope_event (service_key, scope_kind, user_id, seq);

GRANT SELECT, INSERT ON catalog.service_version, catalog.subscription_component, catalog.service_scope_event TO crm_owner_catalog;

-- +goose Down
DROP SCHEMA catalog CASCADE;
