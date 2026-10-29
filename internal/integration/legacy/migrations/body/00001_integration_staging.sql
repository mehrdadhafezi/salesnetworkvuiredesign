-- Integration raw legacy staging (Architecture V1 §24.3, Slice 3).
-- Raw means raw: staged values are the exact legacy text; NULL stays NULL;
-- nothing is normalised, merged, converted or reinterpreted here.

CREATE TABLE integration.import_batch (
    batch_id                 uuid        PRIMARY KEY,
    snapshot_id              text        NOT NULL,
    content_sha256           bytea       NOT NULL,
    manifest_sha256          bytea       NOT NULL,
    manifest_format_version  integer     NOT NULL,
    manifest_json            text        NOT NULL,
    legacy_application       text        NOT NULL,
    legacy_version           text        NOT NULL,
    legacy_build             text        NULL,
    source_id                text        NOT NULL,
    source_timezone          text        NULL,
    source_table_prefix      text        NULL,
    data_origin              text        NOT NULL,
    extracted_at             timestamptz NOT NULL,
    extraction_tool          text        NOT NULL,
    extraction_tool_version  text        NOT NULL,
    import_tool_version      text        NOT NULL,
    state                    text        NOT NULL,
    failure_reason           text        NULL,
    created_at               timestamptz NOT NULL,
    updated_at               timestamptz NOT NULL,
    loaded_at                timestamptz NULL,
    CONSTRAINT import_batch_snapshot_id_unique UNIQUE (snapshot_id),
    CONSTRAINT import_batch_content_unique UNIQUE (content_sha256),
    CONSTRAINT import_batch_snapshot_id_format CHECK (snapshot_id ~ '^[A-Za-z0-9][A-Za-z0-9._-]{7,127}$'),
    CONSTRAINT import_batch_content_sha_length CHECK (octet_length(content_sha256) = 32),
    CONSTRAINT import_batch_manifest_sha_length CHECK (octet_length(manifest_sha256) = 32),
    CONSTRAINT import_batch_format_version CHECK (manifest_format_version = 1),
    CONSTRAINT import_batch_data_origin CHECK (data_origin IN ('synthetic', 'unclassified')),
    CONSTRAINT import_batch_state CHECK (state IN ('VALIDATED', 'LOADING', 'LOADED', 'LOADED_INCOMPLETE', 'FAILED')),
    CONSTRAINT import_batch_loaded CHECK ((state IN ('LOADED', 'LOADED_INCOMPLETE')) = (loaded_at IS NOT NULL)),
    CONSTRAINT import_batch_failure CHECK ((state = 'FAILED') = (failure_reason IS NOT NULL)),
    CONSTRAINT import_batch_failure_length CHECK (failure_reason IS NULL OR length(failure_reason) <= 1000)
);

CREATE TABLE integration.import_entity (
    batch_id                uuid        NOT NULL REFERENCES integration.import_batch (batch_id),
    spec_id                 text        NOT NULL,
    source_table            text        NOT NULL,
    staging_table           text        NOT NULL,
    file_path               text        NOT NULL,
    completeness            text        NOT NULL,
    expected_rows           bigint      NOT NULL,
    expected_bytes          bigint      NOT NULL,
    expected_file_sha256    bytea       NOT NULL,
    expected_content_sha256 bytea       NOT NULL,
    parsed_rows             bigint      NULL,
    actual_bytes            bigint      NULL,
    staged_rows             bigint      NULL,
    actual_file_sha256      bytea       NULL,
    staging_sha256          bytea       NULL,
    state                   text        NOT NULL,
    reason_code             text        NULL,
    failure_reason          text        NULL,
    warnings                jsonb       NOT NULL DEFAULT '[]'::jsonb,
    updated_at              timestamptz NOT NULL,
    loaded_at               timestamptz NULL,
    PRIMARY KEY (batch_id, spec_id),
    CONSTRAINT import_entity_spec_id_format CHECK (spec_id ~ '^legacy\.[a-z][a-z0-9_]{1,48}\.v[1-9][0-9]{0,2}$'),
    CONSTRAINT import_entity_completeness CHECK (completeness IN ('COMPLETE', 'INCOMPLETE', 'UNKNOWN', 'NOT_RECOVERABLE')),
    CONSTRAINT import_entity_counts CHECK (expected_rows >= 0 AND expected_bytes >= 0),
    CONSTRAINT import_entity_sha_lengths CHECK (octet_length(expected_file_sha256) = 32 AND octet_length(expected_content_sha256) = 32
        AND (actual_file_sha256 IS NULL OR octet_length(actual_file_sha256) = 32)
        AND (staging_sha256 IS NULL OR octet_length(staging_sha256) = 32)),
    CONSTRAINT import_entity_state CHECK (state IN ('PENDING', 'LOADED', 'FAILED')),
    -- An entity is LOADED only when source, file checksum, staged rows and the
    -- staging fidelity checksum all reconcile with the manifest.
    CONSTRAINT import_entity_loaded_reconciled CHECK (state <> 'LOADED' OR (
        loaded_at IS NOT NULL
        AND parsed_rows = expected_rows AND staged_rows = expected_rows
        AND actual_bytes = expected_bytes
        AND actual_file_sha256 = expected_file_sha256
        AND staging_sha256 = expected_content_sha256)),
    CONSTRAINT import_entity_not_loaded CHECK (state = 'LOADED' OR loaded_at IS NULL),
    CONSTRAINT import_entity_failure CHECK ((state = 'FAILED') = (failure_reason IS NOT NULL)),
    CONSTRAINT import_entity_failure_length CHECK (failure_reason IS NULL OR length(failure_reason) <= 1000),
    CONSTRAINT import_entity_warnings_array CHECK (jsonb_typeof(warnings) = 'array')
);

GRANT SELECT, INSERT, UPDATE ON integration.import_batch, integration.import_entity TO crm_owner_integration;

-- Raw staging tables: one per published extract spec version. Every source
-- column is a nullable text column raw_<column> holding the exact legacy
-- value. Rows are append-only for the owner (SELECT, INSERT only).
-- BEGIN GENERATED STAGING TABLES (do not edit; see package doc)

-- legacy.hr_assignment.v1: raw sn_hr_assignments (11 columns; classification NONE)
CREATE TABLE integration.stg_hr_assignment_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_child_profile_id text NULL,
    raw_parent_profile_id text NULL,
    raw_relationship_type text NULL,
    raw_effective_from text NULL,
    raw_effective_to text NULL,
    raw_is_current text NULL,
    raw_source text NULL,
    raw_created_by text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_assignment_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_assignment_v1_spec CHECK (spec_id = 'legacy.hr_assignment.v1'),
    CONSTRAINT stg_hr_assignment_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_assignment_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_assignment_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_assignment_v1 TO crm_owner_integration;

-- legacy.hr_assignment_history.v1: raw sn_hr_assignment_history (10 columns; classification PERSONAL)
CREATE TABLE integration.stg_hr_assignment_history_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_assignment_id text NULL,
    raw_child_profile_id text NULL,
    raw_old_parent_profile_id text NULL,
    raw_new_parent_profile_id text NULL,
    raw_actor_user_id text NULL,
    raw_action text NULL,
    raw_reason text NULL,
    raw_context text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_assignment_history_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_assignment_history_v1_spec CHECK (spec_id = 'legacy.hr_assignment_history.v1'),
    CONSTRAINT stg_hr_assignment_history_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_assignment_history_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_assignment_history_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_assignment_history_v1 TO crm_owner_integration;

-- legacy.hr_department.v1: raw sn_hr_departments (8 columns; classification NONE)
CREATE TABLE integration.stg_hr_department_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_slug text NULL,
    raw_label text NULL,
    raw_description text NULL,
    raw_is_active text NULL,
    raw_sort_order text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_department_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_department_v1_spec CHECK (spec_id = 'legacy.hr_department.v1'),
    CONSTRAINT stg_hr_department_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_department_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_department_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_department_v1 TO crm_owner_integration;

-- legacy.hr_level.v1: raw sn_hr_levels (9 columns; classification NONE)
CREATE TABLE integration.stg_hr_level_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_slug text NULL,
    raw_label text NULL,
    raw_description text NULL,
    raw_sort_order text NULL,
    raw_is_active text NULL,
    raw_is_system text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_level_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_level_v1_spec CHECK (spec_id = 'legacy.hr_level.v1'),
    CONSTRAINT stg_hr_level_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_level_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_level_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_level_v1 TO crm_owner_integration;

-- legacy.hr_position.v1: raw sn_hr_positions (10 columns; classification NONE)
CREATE TABLE integration.stg_hr_position_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_slug text NULL,
    raw_label text NULL,
    raw_description text NULL,
    raw_panel_key text NULL,
    raw_sort_order text NULL,
    raw_is_active text NULL,
    raw_is_system text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_position_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_position_v1_spec CHECK (spec_id = 'legacy.hr_position.v1'),
    CONSTRAINT stg_hr_position_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_position_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_position_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_position_v1 TO crm_owner_integration;

-- legacy.hr_position_role_mapping.v1: raw sn_hr_position_role_mappings (7 columns; classification NONE)
CREATE TABLE integration.stg_hr_position_role_mapping_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_legacy_role text NULL,
    raw_position_id text NULL,
    raw_default_level_id text NULL,
    raw_is_active text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_position_role_mapping_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_position_role_mapping_v1_spec CHECK (spec_id = 'legacy.hr_position_role_mapping.v1'),
    CONSTRAINT stg_hr_position_role_mapping_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_position_role_mapping_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_position_role_mapping_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_position_role_mapping_v1 TO crm_owner_integration;

-- legacy.hr_profile.v1: raw sn_hr_profiles (26 columns; classification SENSITIVE)
CREATE TABLE integration.stg_hr_profile_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_user_id text NULL,
    raw_hr_display_name text NULL,
    raw_position_id text NULL,
    raw_level_id text NULL,
    raw_employee_code text NULL,
    raw_employment_status text NULL,
    raw_employment_type text NULL,
    raw_contract_status text NULL,
    raw_training_status text NULL,
    raw_hire_date text NULL,
    raw_termination_date text NULL,
    raw_base_salary text NULL,
    raw_salary_currency text NULL,
    raw_national_id text NULL,
    raw_work_phone text NULL,
    raw_emergency_phone text NULL,
    raw_department_id text NULL,
    raw_unit_id text NULL,
    raw_team_id text NULL,
    raw_notes text NULL,
    raw_legacy_role text NULL,
    raw_legacy_source text NULL,
    raw_is_active text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_profile_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_profile_v1_spec CHECK (spec_id = 'legacy.hr_profile.v1'),
    CONSTRAINT stg_hr_profile_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_profile_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_profile_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_profile_v1 TO crm_owner_integration;

-- legacy.hr_team.v1: raw sn_hr_teams (9 columns; classification NONE)
CREATE TABLE integration.stg_hr_team_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_unit_id text NULL,
    raw_slug text NULL,
    raw_label text NULL,
    raw_description text NULL,
    raw_is_active text NULL,
    raw_sort_order text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_team_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_team_v1_spec CHECK (spec_id = 'legacy.hr_team.v1'),
    CONSTRAINT stg_hr_team_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_team_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_team_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_team_v1 TO crm_owner_integration;

-- legacy.hr_unit.v1: raw sn_hr_units (9 columns; classification NONE)
CREATE TABLE integration.stg_hr_unit_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_department_id text NULL,
    raw_slug text NULL,
    raw_label text NULL,
    raw_description text NULL,
    raw_is_active text NULL,
    raw_sort_order text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_hr_unit_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_hr_unit_v1_spec CHECK (spec_id = 'legacy.hr_unit.v1'),
    CONSTRAINT stg_hr_unit_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_hr_unit_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_hr_unit_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_hr_unit_v1 TO crm_owner_integration;

-- legacy.wp_user.v1: raw users (5 columns; classification PERSONAL)
CREATE TABLE integration.stg_wp_user_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_user_login text NULL,
    raw_user_email text NULL,
    raw_display_name text NULL,
    raw_user_registered text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_wp_user_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_wp_user_v1_spec CHECK (spec_id = 'legacy.wp_user.v1'),
    CONSTRAINT stg_wp_user_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_wp_user_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_wp_user_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_wp_user_v1 TO crm_owner_integration;

-- legacy.wp_usermeta.v1: raw usermeta (4 columns; classification SENSITIVE)
CREATE TABLE integration.stg_wp_usermeta_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_umeta_id text NULL,
    raw_user_id text NULL,
    raw_meta_key text NULL,
    raw_meta_value text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_wp_usermeta_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_wp_usermeta_v1_spec CHECK (spec_id = 'legacy.wp_usermeta.v1'),
    CONSTRAINT stg_wp_usermeta_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_wp_usermeta_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_wp_usermeta_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_wp_usermeta_v1 TO crm_owner_integration;
-- END GENERATED STAGING TABLES
