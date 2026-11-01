-- Per-module infrastructure for module "billing" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_billing NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA billing;
REVOKE ALL ON SCHEMA billing FROM PUBLIC;
GRANT USAGE ON SCHEMA billing TO crm_owner_billing;

CREATE TABLE billing.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON billing.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE billing.inbox_message (
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

CREATE TABLE billing.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA billing FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON billing.outbox_event, billing.inbox_message, billing.idempotency_record TO crm_owner_billing;

-- Module-owned tables follow. Grants on them go to crm_owner_billing only.
-- Billing shadow model (Architecture V1 §5, §7, §10; Slice 7). Materialised
-- offline by the billing import command only; never written by a read path.
-- One Invoice per staged legacy invoice row. No issue/next-stage/cancel
-- command exists and no DELETE is granted.
--
-- States are never collapsed: the five legacy invoice status fields are
-- kept as separate evidence rows (raw value + namespace + mapping), stage
-- state lives on the stage, and payment truth lives in the payments schema.
-- The target invoice lifecycle is UNKNOWN until the canonical main-status
-- resolver exists (TAD #20 deferred); the canonical total resolver (TAD #19)
-- is deferred too, so amounts are evidence, never a canonical total.
--
-- Money: amount bigint + explicit unit (IRR | IRT); the raw decimal text is
-- kept; a non-integral or unparseable raw value is UNREPRESENTABLE with a
-- NULL amount (never rounded, never converted, never zero). No phone, name,
-- address, card data, token or file path is stored.

CREATE TABLE billing.invoice (
    invoice_id            uuid        PRIMARY KEY,
    source_system         text        NOT NULL,
    source_spec           text        NOT NULL,
    source_key            text        NOT NULL,
    first_batch_id        uuid        NOT NULL,
    last_batch_id         uuid        NOT NULL,
    row_sha256            bytea       NOT NULL,
    first_seen_at         timestamptz NOT NULL,
    last_seen_at          timestamptz NOT NULL,
    presence              text        NOT NULL,
    invoice_code_raw      text        NULL,
    seller_legacy_key     text        NULL,
    seller_user_id        uuid        NULL,
    issued_by_legacy_key  text        NULL,
    issued_by_user_id     uuid        NULL,
    product_ref_raw       text        NULL,
    pay_method_raw        text        NULL,
    payment_plan_raw      text        NULL,
    current_stage_raw     text        NULL,
    issued_at_raw         text        NULL,
    updated_at_raw        text        NULL,
    lifecycle_state       text        NOT NULL,
    customer_id           uuid        NULL,
    customer_link_state   text        NOT NULL,
    customer_link_basis   text        NOT NULL,
    case_id               uuid        NULL,
    case_link_state       text        NOT NULL,
    case_link_basis       text        NOT NULL,
    items_source_complete  boolean     NOT NULL DEFAULT false,
    stages_source_complete boolean     NOT NULL DEFAULT false,
    version               bigint      NOT NULL DEFAULT 1,
    CONSTRAINT invoice_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT invoice_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT invoice_sha_length CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT invoice_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT invoice_lifecycle_unknown CHECK (lifecycle_state = 'UNKNOWN'),
    CONSTRAINT invoice_customer_link CHECK (customer_link_state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT invoice_case_link CHECK (case_link_state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT invoice_customer_resolved_ref CHECK (customer_link_state <> 'RESOLVED' OR customer_id IS NOT NULL),
    CONSTRAINT invoice_case_resolved_ref CHECK (case_link_state <> 'RESOLVED' OR case_id IS NOT NULL),
    CONSTRAINT invoice_version CHECK (version > 0)
);

-- One row per legacy status field; never merged into one status.
CREATE TABLE billing.invoice_state_evidence (
    invoice_id   uuid NOT NULL REFERENCES billing.invoice (invoice_id),
    field        text NOT NULL,
    namespace    text NOT NULL,
    raw_value    text NULL,
    mapping      text NOT NULL,
    PRIMARY KEY (invoice_id, field),
    CONSTRAINT state_evidence_field CHECK (field IN ('MAIN', 'INVOICE', 'PAYMENT', 'WORKFLOW', 'RETURN', 'LEGACY_DISPLAY')),
    CONSTRAINT state_evidence_mapping CHECK (mapping IN ('KNOWN_VOCABULARY', 'UNKNOWN', 'EMPTY', 'LEGACY_DERIVED')),
    CONSTRAINT state_evidence_derived_only_display CHECK ((mapping = 'LEGACY_DERIVED') = (field = 'LEGACY_DISPLAY'))
);

-- Declared amount evidence per invoice (each legacy amount field separately).
CREATE TABLE billing.amount_evidence (
    invoice_id     uuid   NOT NULL REFERENCES billing.invoice (invoice_id),
    kind           text   NOT NULL,
    raw_value      text   NULL,
    amount         bigint NULL,
    unit           text   NULL,
    representation text   NOT NULL,
    PRIMARY KEY (invoice_id, kind),
    CONSTRAINT amount_evidence_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT')),
    CONSTRAINT amount_evidence_representation CHECK (representation IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')),
    CONSTRAINT amount_evidence_amount_iff CHECK ((representation = 'REPRESENTABLE') = (amount IS NOT NULL AND unit IS NOT NULL))
);

CREATE TABLE billing.invoice_item (
    item_id             uuid        PRIMARY KEY,
    invoice_id          uuid        NOT NULL REFERENCES billing.invoice (invoice_id),
    source_system       text        NOT NULL,
    source_spec         text        NOT NULL,
    source_key          text        NOT NULL,
    first_batch_id      uuid        NOT NULL,
    last_batch_id       uuid        NOT NULL,
    row_sha256          bytea       NOT NULL,
    first_seen_at       timestamptz NOT NULL,
    last_seen_at        timestamptz NOT NULL,
    presence            text        NOT NULL,
    product_ref_raw     text        NULL,
    title_snapshot      text        NULL,
    kind_raw            text        NULL,
    qty_raw             text        NULL,
    unit_price_raw      text        NULL,
    unit_price          bigint      NULL,
    line_total_raw      text        NULL,
    line_total          bigint      NULL,
    unit                text        NULL,
    line_total_repr     text        NOT NULL,
    is_free_raw         text        NULL,
    created_at_raw      text        NULL,
    CONSTRAINT item_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT item_sha_length CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT item_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT item_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT')),
    CONSTRAINT item_repr CHECK (line_total_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')),
    CONSTRAINT item_amount_iff CHECK ((line_total_repr = 'REPRESENTABLE') = (line_total IS NOT NULL AND unit IS NOT NULL))
);
CREATE INDEX invoice_item_invoice_idx ON billing.invoice_item (invoice_id, item_id);

CREATE TABLE billing.payment_stage (
    stage_id               uuid        PRIMARY KEY,
    invoice_id             uuid        NOT NULL REFERENCES billing.invoice (invoice_id),
    source_system          text        NOT NULL,
    source_spec            text        NOT NULL,
    source_key             text        NOT NULL,
    first_batch_id         uuid        NOT NULL,
    last_batch_id          uuid        NOT NULL,
    row_sha256             bytea       NOT NULL,
    first_seen_at          timestamptz NOT NULL,
    last_seen_at           timestamptz NOT NULL,
    presence               text        NOT NULL,
    stage_no_raw           text        NULL,
    stage_type_raw         text        NULL,
    requested_raw          text        NULL,
    requested_amount       bigint      NULL,
    unit                   text        NULL,
    requested_repr         text        NOT NULL,
    state_namespace        text        NOT NULL,
    state_raw              text        NULL,
    state_mapping          text        NOT NULL,
    state_target           text        NULL,
    pay_method_raw         text        NULL,
    payment_source_raw     text        NULL,
    issued_by_legacy_key   text        NULL,
    issued_by_user_id      uuid        NULL,
    approved_by_legacy_key text        NULL,
    approved_by_user_id    uuid        NULL,
    approved_at_raw        text        NULL,
    paid_at_raw            text        NULL,
    created_at_raw         text        NULL,
    CONSTRAINT stage_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT stage_sha_length CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stage_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT stage_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT')),
    CONSTRAINT stage_repr CHECK (requested_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')),
    CONSTRAINT stage_amount_iff CHECK ((requested_repr = 'REPRESENTABLE') = (requested_amount IS NOT NULL AND unit IS NOT NULL)),
    CONSTRAINT stage_state_mapping CHECK (state_mapping IN ('MAPPED', 'UNKNOWN', 'EMPTY')),
    CONSTRAINT stage_state_target CHECK ((state_mapping = 'MAPPED') = (state_target IS NOT NULL)),
    CONSTRAINT stage_state_vocab CHECK (state_target IS NULL OR state_target IN ('PENDING', 'AWAITING_REVIEW', 'ACCEPTED', 'REJECTED', 'CANCELLED'))
);
CREATE INDEX payment_stage_invoice_idx ON billing.payment_stage (invoice_id, stage_id);

CREATE TABLE billing.import_run (
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
GRANT SELECT, INSERT, UPDATE ON billing.invoice, billing.invoice_state_evidence, billing.amount_evidence,
    billing.invoice_item, billing.payment_stage TO crm_owner_billing;
GRANT SELECT, INSERT ON billing.import_run TO crm_owner_billing;

-- +goose Down
DROP SCHEMA billing CASCADE;
