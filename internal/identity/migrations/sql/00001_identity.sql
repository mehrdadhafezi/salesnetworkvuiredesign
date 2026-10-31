-- Per-module infrastructure for module "identity" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_identity NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA identity;
REVOKE ALL ON SCHEMA identity FROM PUBLIC;
GRANT USAGE ON SCHEMA identity TO crm_owner_identity;

CREATE TABLE identity.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON identity.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE identity.inbox_message (
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

CREATE TABLE identity.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA identity FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON identity.outbox_event, identity.inbox_message, identity.idempotency_record TO crm_owner_identity;

-- Module-owned tables follow. Grants on them go to crm_owner_identity only.
-- Identity read/shadow model (Architecture V1 §5, §7, §18, §25; Slice 4).
-- Read-only target model materialised offline from Slice 3 staging by the
-- identity import command. Nothing here is operational authentication:
-- there is no credential, password hash or session table, accounts cannot
-- log in, and every role grant is a SHADOW grant (evidence for the
-- capability diff), never runtime authority. Legacy keys are provenance
-- only: user_id is the native identifier; phone, e-mail and login are
-- never keys.

-- Capability vocabulary and role bundles. Code-defined (identity/api) and
-- mirrored here as reference data; a test proves both are identical.
CREATE TABLE identity.capability (
    capability_key text PRIMARY KEY,
    description    text NOT NULL,
    CONSTRAINT capability_key_format CHECK (capability_key ~ '^[a-z][a-z_]*(\.[a-z][a-z_]*){2}$'),
    CONSTRAINT capability_description_length CHECK (length(description) BETWEEN 1 AND 200)
);

CREATE TABLE identity.role (
    role_key text PRIMARY KEY,
    label    text NOT NULL,
    CONSTRAINT role_key_format CHECK (role_key ~ '^[a-z][a-z_]{1,40}$'),
    CONSTRAINT role_label_length CHECK (length(label) BETWEEN 1 AND 100)
);

CREATE TABLE identity.role_capability (
    role_key       text NOT NULL REFERENCES identity.role (role_key),
    capability_key text NOT NULL REFERENCES identity.capability (capability_key),
    PRIMARY KEY (role_key, capability_key)
);

INSERT INTO identity.capability (capability_key, description) VALUES
    ('identity.user.read',     'Read user accounts (shadow identity model)'),
    ('org.person.read',        'Read persons (HR identity)'),
    ('org.employment.read',    'Read employment lifecycle attributes'),
    ('org.position.read',      'Read positions and levels'),
    ('org.unit.read',          'Read the organisation unit tree'),
    ('org.assignment.read',    'Read reporting assignments, their history and hierarchy findings');

INSERT INTO identity.role (role_key, label) VALUES
    ('seller',            'Seller'),
    ('supervisor',        'Supervisor'),
    ('senior_supervisor', 'Senior Supervisor'),
    ('sales_manager',     'Sales Manager'),
    ('sales_deputy',      'Sales Deputy'),
    ('mis',               'MIS'),
    ('hr',                'HR'),
    ('finance',           'Finance'),
    ('invoice_reviewer',  'Invoice Reviewer');

INSERT INTO identity.role_capability (role_key, capability_key) VALUES
    ('hr', 'identity.user.read'),
    ('hr', 'org.person.read'),
    ('hr', 'org.employment.read'),
    ('hr', 'org.position.read'),
    ('hr', 'org.unit.read'),
    ('hr', 'org.assignment.read');

-- User accounts (shadow). One row per legacy account key.
CREATE TABLE identity.user_account (
    user_id           uuid        PRIMARY KEY,
    source_system     text        NOT NULL,
    source_spec       text        NOT NULL,
    source_key        text        NOT NULL,
    first_batch_id    uuid        NOT NULL,
    source_batch_id   uuid        NOT NULL,
    source_row_sha256 bytea       NOT NULL,
    imported_at       timestamptz NOT NULL,
    refreshed_at      timestamptz NOT NULL,
    presence          text        NOT NULL,
    login_evidence    text        NULL,
    display_name      text        NULL,
    registered_raw    text        NULL,
    account_state     text        NOT NULL,
    login_enabled     boolean     NOT NULL,
    version           bigint      NOT NULL,
    CONSTRAINT user_account_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT user_account_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT user_account_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT user_account_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT user_account_state CHECK (account_state = 'SHADOW_IMPORTED'),
    CONSTRAINT user_account_no_login CHECK (login_enabled = false),
    CONSTRAINT user_account_version CHECK (version > 0)
);
CREATE INDEX user_account_display_name_idx ON identity.user_account (display_name, user_id);

-- Role grants. Slice 4 materialises only SHADOW evidence from the legacy
-- profile role label; a SHADOW grant never authorises a runtime request.
CREATE TABLE identity.role_grant (
    grant_id            uuid        PRIMARY KEY,
    user_id             uuid        NOT NULL REFERENCES identity.user_account (user_id),
    role_key            text        NULL REFERENCES identity.role (role_key),
    legacy_role_value   text        NOT NULL,
    mapping_state       text        NOT NULL,
    basis               text        NOT NULL,
    status              text        NOT NULL,
    evidence_spec       text        NOT NULL,
    evidence_key        text        NOT NULL,
    first_batch_id      uuid        NOT NULL,
    source_batch_id     uuid        NOT NULL,
    imported_at         timestamptz NOT NULL,
    refreshed_at        timestamptz NOT NULL,
    presence            text        NOT NULL,
    version             bigint      NOT NULL,
    CONSTRAINT role_grant_evidence_unique UNIQUE (basis, evidence_key),
    CONSTRAINT role_grant_mapping CHECK (mapping_state IN ('MAPPED', 'UNMAPPED')),
    CONSTRAINT role_grant_mapping_role CHECK ((mapping_state = 'MAPPED') = (role_key IS NOT NULL)),
    CONSTRAINT role_grant_basis CHECK (basis = 'LEGACY_PROFILE_ROLE'),
    CONSTRAINT role_grant_shadow_only CHECK (status = 'SHADOW'),
    CONSTRAINT role_grant_legacy_value_length CHECK (length(legacy_role_value) BETWEEN 1 AND 120),
    CONSTRAINT role_grant_evidence_key_format CHECK (evidence_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT role_grant_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT role_grant_version CHECK (version > 0)
);
CREATE INDEX role_grant_user_idx ON identity.role_grant (user_id);

-- One row per import command run (provenance of the shadow refresh).
CREATE TABLE identity.import_run (
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

GRANT SELECT ON identity.capability, identity.role, identity.role_capability TO crm_owner_identity;
GRANT SELECT, INSERT, UPDATE ON identity.user_account, identity.role_grant TO crm_owner_identity;
GRANT SELECT, INSERT ON identity.import_run TO crm_owner_identity;

-- +goose Down
DROP SCHEMA identity CASCADE;
