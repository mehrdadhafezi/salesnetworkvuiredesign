-- Per-module infrastructure for module "organization" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_organization NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA organization;
REVOKE ALL ON SCHEMA organization FROM PUBLIC;
GRANT USAGE ON SCHEMA organization TO crm_owner_organization;

CREATE TABLE organization.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON organization.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE organization.inbox_message (
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

CREATE TABLE organization.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA organization FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON organization.outbox_event, organization.inbox_message, organization.idempotency_record TO crm_owner_organization;

-- Module-owned tables follow. Grants on them go to crm_owner_organization only.
-- Organization read/shadow model (Architecture V1 §5, §7, §25; Slice 4).
-- Materialised offline from Slice 3 staging by the organization import
-- command; never written by a read path. The legacy mixed profile is SPLIT
-- into Person (human / HR identity) and Employment (organisation
-- relationship); the three legacy structure tables MERGE into one typed
-- OrgUnit tree; reporting assignments keep their raw interval evidence and
-- their history is append-only.
--
-- Every row keeps its provenance (source system, extract spec, legacy key,
-- batch, row checksum, import time, presence). Raw legacy values that the
-- target does not interpret (dates without an asserted zone, unfamiliar
-- statuses) are kept verbatim next to the typed column, which is NULL or
-- UNMAPPED when the raw value is not understood. Sensitive legacy profile
-- columns (national id, salary, phones, notes, free text) are NOT imported.

CREATE TABLE organization.position (
    position_id        uuid        PRIMARY KEY,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    source_row_sha256  bytea       NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    slug               text        NULL,
    label              text        NULL,
    description        text        NULL,
    sort_order         integer     NULL,
    is_active          boolean     NULL,
    is_system          boolean     NULL,
    source_created_raw text        NULL,
    source_updated_raw text        NULL,
    version            bigint      NOT NULL,
    CONSTRAINT position_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT position_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT position_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT position_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT position_version CHECK (version > 0)
);

CREATE TABLE organization.level (
    level_id           uuid        PRIMARY KEY,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    source_row_sha256  bytea       NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    slug               text        NULL,
    label              text        NULL,
    description        text        NULL,
    sort_order         integer     NULL,
    is_active          boolean     NULL,
    is_system          boolean     NULL,
    source_created_raw text        NULL,
    source_updated_raw text        NULL,
    version            bigint      NOT NULL,
    CONSTRAINT level_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT level_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT level_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT level_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT level_version CHECK (version > 0)
);

-- One tree, typed by kind. Legacy keys are unique per kind only.
CREATE TABLE organization.org_unit (
    org_unit_id        uuid        PRIMARY KEY,
    kind               text        NOT NULL,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    source_row_sha256  bytea       NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    parent_org_unit_id uuid        NULL REFERENCES organization.org_unit (org_unit_id),
    parent_source_key  text        NULL,
    parent_state       text        NOT NULL,
    slug               text        NULL,
    label              text        NULL,
    description        text        NULL,
    sort_order         integer     NULL,
    is_active          boolean     NULL,
    source_created_raw text        NULL,
    source_updated_raw text        NULL,
    version            bigint      NOT NULL,
    CONSTRAINT org_unit_source_unique UNIQUE (source_system, kind, source_key),
    CONSTRAINT org_unit_kind CHECK (kind IN ('DEPARTMENT', 'UNIT', 'TEAM')),
    CONSTRAINT org_unit_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT org_unit_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT org_unit_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT org_unit_parent_state CHECK (parent_state IN ('NOT_APPLICABLE', 'NOT_RECORDED', 'RESOLVED', 'MISSING')),
    CONSTRAINT org_unit_department_root CHECK ((kind = 'DEPARTMENT') = (parent_state = 'NOT_APPLICABLE')),
    CONSTRAINT org_unit_parent_resolved CHECK ((parent_state = 'RESOLVED') = (parent_org_unit_id IS NOT NULL)),
    CONSTRAINT org_unit_parent_key CHECK ((parent_state IN ('RESOLVED', 'MISSING')) = (parent_source_key IS NOT NULL)),
    CONSTRAINT org_unit_not_self CHECK (parent_org_unit_id IS NULL OR parent_org_unit_id <> org_unit_id),
    CONSTRAINT org_unit_version CHECK (version > 0)
);
CREATE INDEX org_unit_parent_idx ON organization.org_unit (parent_org_unit_id);

CREATE TABLE organization.person (
    person_id          uuid        PRIMARY KEY,
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    source_batch_id    uuid        NOT NULL,
    source_row_sha256  bytea       NOT NULL,
    imported_at        timestamptz NOT NULL,
    refreshed_at       timestamptz NOT NULL,
    presence           text        NOT NULL,
    display_name       text        NULL,
    version            bigint      NOT NULL,
    CONSTRAINT person_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT person_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT person_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT person_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT person_version CHECK (version > 0)
);
CREATE INDEX person_display_name_idx ON organization.person (display_name, person_id);

-- Employment: the organisation relationship of a person. user_id is an
-- identity reference (published identity API), not a foreign key.
CREATE TABLE organization.employment (
    employment_id          uuid        PRIMARY KEY,
    person_id              uuid        NOT NULL UNIQUE REFERENCES organization.person (person_id),
    source_system          text        NOT NULL,
    source_spec            text        NOT NULL,
    source_key             text        NOT NULL,
    first_batch_id         uuid        NOT NULL,
    source_batch_id        uuid        NOT NULL,
    source_row_sha256      bytea       NOT NULL,
    imported_at            timestamptz NOT NULL,
    refreshed_at           timestamptz NOT NULL,
    presence               text        NOT NULL,
    user_id                uuid        NULL,
    user_source_key        text        NULL,
    user_link_state        text        NOT NULL,
    employee_code          text        NULL,
    employment_status      text        NOT NULL,
    employment_status_raw  text        NULL,
    employment_type_raw    text        NULL,
    hire_date              date        NULL,
    hire_date_raw          text        NULL,
    termination_date       date        NULL,
    termination_date_raw   text        NULL,
    is_active              boolean     NULL,
    position_id            uuid        NULL REFERENCES organization.position (position_id),
    position_source_key    text        NULL,
    position_ref_state     text        NOT NULL,
    level_id               uuid        NULL REFERENCES organization.level (level_id),
    level_source_key       text        NULL,
    level_ref_state        text        NOT NULL,
    department_id          uuid        NULL REFERENCES organization.org_unit (org_unit_id),
    department_source_key  text        NULL,
    department_ref_state   text        NOT NULL,
    unit_id                uuid        NULL REFERENCES organization.org_unit (org_unit_id),
    unit_source_key        text        NULL,
    unit_ref_state         text        NOT NULL,
    team_id                uuid        NULL REFERENCES organization.org_unit (org_unit_id),
    team_source_key        text        NULL,
    team_ref_state         text        NOT NULL,
    source_created_raw     text        NULL,
    source_updated_raw     text        NULL,
    version                bigint      NOT NULL,
    CONSTRAINT employment_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT employment_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT employment_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT employment_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT employment_user_link CHECK (user_link_state IN ('LINKED', 'USER_NOT_IMPORTED', 'USER_NOT_RECORDED', 'SHARED_USER')),
    CONSTRAINT employment_user_linked CHECK ((user_id IS NOT NULL) = (user_link_state IN ('LINKED', 'SHARED_USER'))),
    CONSTRAINT employment_status CHECK (employment_status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'RESIGNED', 'TERMINATED', 'PROBATION', 'UNMAPPED', 'NOT_RECORDED')),
    CONSTRAINT employment_position_ref CHECK (position_ref_state IN ('RESOLVED', 'NOT_RECORDED', 'MISSING')),
    CONSTRAINT employment_level_ref CHECK (level_ref_state IN ('RESOLVED', 'NOT_RECORDED', 'MISSING')),
    CONSTRAINT employment_department_ref CHECK (department_ref_state IN ('RESOLVED', 'NOT_RECORDED', 'MISSING')),
    CONSTRAINT employment_unit_ref CHECK (unit_ref_state IN ('RESOLVED', 'NOT_RECORDED', 'MISSING')),
    CONSTRAINT employment_team_ref CHECK (team_ref_state IN ('RESOLVED', 'NOT_RECORDED', 'MISSING')),
    CONSTRAINT employment_position_resolved CHECK ((position_ref_state = 'RESOLVED') = (position_id IS NOT NULL)),
    CONSTRAINT employment_level_resolved CHECK ((level_ref_state = 'RESOLVED') = (level_id IS NOT NULL)),
    CONSTRAINT employment_department_resolved CHECK ((department_ref_state = 'RESOLVED') = (department_id IS NOT NULL)),
    CONSTRAINT employment_unit_resolved CHECK ((unit_ref_state = 'RESOLVED') = (unit_id IS NOT NULL)),
    CONSTRAINT employment_team_resolved CHECK ((team_ref_state = 'RESOLVED') = (team_id IS NOT NULL)),
    CONSTRAINT employment_version CHECK (version > 0)
);
CREATE INDEX employment_user_idx ON organization.employment (user_id);
CREATE INDEX employment_position_idx ON organization.employment (position_id);

-- Reporting assignments (reports-to lines), one row per legacy assignment.
-- Interval and current-flag evidence is raw: the legacy zone is not asserted.
CREATE TABLE organization.org_assignment (
    assignment_id          uuid        PRIMARY KEY,
    source_system          text        NOT NULL,
    source_spec            text        NOT NULL,
    source_key             text        NOT NULL,
    first_batch_id         uuid        NOT NULL,
    source_batch_id        uuid        NOT NULL,
    source_row_sha256      bytea       NOT NULL,
    imported_at            timestamptz NOT NULL,
    refreshed_at           timestamptz NOT NULL,
    presence               text        NOT NULL,
    child_employment_id    uuid        NULL REFERENCES organization.employment (employment_id),
    child_source_key       text        NOT NULL,
    child_ref_state        text        NOT NULL,
    parent_employment_id   uuid        NULL REFERENCES organization.employment (employment_id),
    parent_source_key      text        NULL,
    parent_ref_state       text        NOT NULL,
    relationship_kind      text        NOT NULL,
    relationship_raw       text        NULL,
    effective_from_raw     text        NULL,
    effective_to_raw       text        NULL,
    is_current             boolean     NULL,
    current_raw            text        NULL,
    origin_raw             text        NULL,
    recorded_by_source_key text        NULL,
    source_created_raw     text        NULL,
    source_updated_raw     text        NULL,
    version                bigint      NOT NULL,
    CONSTRAINT org_assignment_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT org_assignment_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT org_assignment_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT org_assignment_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT org_assignment_child_ref CHECK (child_ref_state IN ('RESOLVED', 'MISSING')),
    CONSTRAINT org_assignment_child_resolved CHECK ((child_ref_state = 'RESOLVED') = (child_employment_id IS NOT NULL)),
    CONSTRAINT org_assignment_parent_ref CHECK (parent_ref_state IN ('RESOLVED', 'MISSING', 'NOT_RECORDED')),
    CONSTRAINT org_assignment_parent_resolved CHECK ((parent_ref_state = 'RESOLVED') = (parent_employment_id IS NOT NULL)),
    CONSTRAINT org_assignment_parent_key CHECK ((parent_ref_state = 'NOT_RECORDED') = (parent_source_key IS NULL)),
    CONSTRAINT org_assignment_kind CHECK (relationship_kind IN ('REPORTS_TO', 'UNMAPPED')),
    CONSTRAINT org_assignment_version CHECK (version > 0)
);
CREATE INDEX org_assignment_child_idx ON organization.org_assignment (child_employment_id);
CREATE INDEX org_assignment_parent_idx ON organization.org_assignment (parent_employment_id);

-- Assignment history: append-only evidence; never updated or deleted.
-- Free-text reason/context columns are not imported.
CREATE TABLE organization.org_assignment_history (
    history_id               uuid        PRIMARY KEY,
    source_system            text        NOT NULL,
    source_spec              text        NOT NULL,
    source_key               text        NOT NULL,
    source_batch_id          uuid        NOT NULL,
    source_row_sha256        bytea       NOT NULL,
    imported_at              timestamptz NOT NULL,
    assignment_source_key    text        NULL,
    child_source_key         text        NULL,
    child_employment_id      uuid        NULL REFERENCES organization.employment (employment_id),
    old_parent_source_key    text        NULL,
    old_parent_employment_id uuid        NULL REFERENCES organization.employment (employment_id),
    new_parent_source_key    text        NULL,
    new_parent_employment_id uuid        NULL REFERENCES organization.employment (employment_id),
    actor_user_source_key    text        NULL,
    action_raw               text        NULL,
    recorded_raw             text        NULL,
    CONSTRAINT org_assignment_history_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT org_assignment_history_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT org_assignment_history_row_sha_length CHECK (octet_length(source_row_sha256) = 32)
);
CREATE INDEX org_assignment_history_child_idx ON organization.org_assignment_history (child_employment_id);

-- Default role suggested for a position (legacy evidence). Never a grant:
-- role_key is an identity role reference applied by no command in Slice 4.
CREATE TABLE organization.position_role_default (
    default_id               uuid        PRIMARY KEY,
    source_system            text        NOT NULL,
    source_spec              text        NOT NULL,
    source_key               text        NOT NULL,
    first_batch_id           uuid        NOT NULL,
    source_batch_id          uuid        NOT NULL,
    source_row_sha256        bytea       NOT NULL,
    imported_at              timestamptz NOT NULL,
    refreshed_at             timestamptz NOT NULL,
    presence                 text        NOT NULL,
    position_id              uuid        NULL REFERENCES organization.position (position_id),
    position_source_key      text        NULL,
    role_key                 text        NULL,
    legacy_role_value        text        NULL,
    mapping_state            text        NOT NULL,
    default_level_id         uuid        NULL REFERENCES organization.level (level_id),
    default_level_source_key text        NULL,
    is_active                boolean     NULL,
    version                  bigint      NOT NULL,
    CONSTRAINT position_role_default_source_unique UNIQUE (source_system, source_key),
    CONSTRAINT position_role_default_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT position_role_default_row_sha_length CHECK (octet_length(source_row_sha256) = 32),
    CONSTRAINT position_role_default_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT position_role_default_mapping CHECK (mapping_state IN ('MAPPED', 'UNMAPPED')),
    CONSTRAINT position_role_default_mapping_role CHECK ((mapping_state = 'MAPPED') = (role_key IS NOT NULL)),
    CONSTRAINT position_role_default_version CHECK (version > 0)
);

-- Hierarchy findings of the latest import run (derived read model). They
-- report conflicts; nothing is auto-fixed.
CREATE TABLE organization.hierarchy_finding (
    finding_id          uuid        PRIMARY KEY,
    run_id              uuid        NOT NULL,
    kind                text        NOT NULL,
    severity            text        NOT NULL,
    subject_type        text        NOT NULL,
    subject_id          uuid        NULL,
    subject_source_key  text        NULL,
    related_id          uuid        NULL,
    related_source_key  text        NULL,
    detail              text        NOT NULL,
    CONSTRAINT hierarchy_finding_kind CHECK (kind IN (
        'MISSING_PARENT', 'SELF_PARENT', 'CYCLE', 'DUPLICATE_CURRENT', 'MULTIPLE_CURRENT_PARENTS',
        'INTERVAL_OVERLAP', 'HISTORY_GAP', 'UNEXPLAINED_GAP', 'MISSING_CHILD', 'ORG_UNIT_PARENT_MISSING',
        'ORG_UNIT_PARENT_NOT_RECORDED', 'MEMBERSHIP_MISMATCH', 'CURRENT_FLAG_INCONSISTENT',
        'HISTORY_CURRENT_DISAGREE', 'UNREADABLE_DATE', 'UNMAPPED_RELATIONSHIP', 'MISSING_REFERENCE',
        'SHARED_USER', 'USER_NOT_IMPORTED', 'HISTORY_CHANGED')),
    CONSTRAINT hierarchy_finding_severity CHECK (severity IN ('CONFLICT', 'UNKNOWN', 'INFO')),
    CONSTRAINT hierarchy_finding_subject CHECK (subject_type IN ('ASSIGNMENT', 'EMPLOYMENT', 'ORG_UNIT', 'HISTORY')),
    CONSTRAINT hierarchy_finding_detail_length CHECK (length(detail) BETWEEN 1 AND 300)
);
CREATE INDEX hierarchy_finding_subject_idx ON organization.hierarchy_finding (subject_id);

CREATE TABLE organization.import_run (
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

GRANT SELECT, INSERT, UPDATE ON organization.position, organization.level, organization.org_unit,
    organization.person, organization.employment, organization.org_assignment,
    organization.position_role_default TO crm_owner_organization;
GRANT SELECT, INSERT ON organization.org_assignment_history, organization.import_run TO crm_owner_organization;
GRANT SELECT, INSERT, DELETE ON organization.hierarchy_finding TO crm_owner_organization;

-- +goose Down
DROP SCHEMA organization CASCADE;
