-- Integration raw staging, Slice 6 extension: lead, lead status history,
-- distribution item and distribution item log sources (Architecture V1 §10,
-- §24.3). Generated from the published Slice 6 specs; regenerate with the
-- package -update flag.

-- +goose Up

-- legacy.distribution_item.v1: raw sn_distribution_items (13 columns; classification NONE)
CREATE TABLE integration.stg_distribution_item_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_source_type text NULL,
    raw_source_id text NULL,
    raw_batch_id text NULL,
    raw_current_owner_user_id text NULL,
    raw_current_owner_position text NULL,
    raw_current_step_order text NULL,
    raw_final_seller_user_id text NULL,
    raw_live_lead_id text NULL,
    raw_conversion_status text NULL,
    raw_status text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_distribution_item_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_distribution_item_v1_spec CHECK (spec_id = 'legacy.distribution_item.v1'),
    CONSTRAINT stg_distribution_item_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_distribution_item_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_distribution_item_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_distribution_item_v1 TO crm_owner_integration;

-- legacy.distribution_item_log.v1: raw sn_distribution_item_logs (10 columns; classification NONE)
CREATE TABLE integration.stg_distribution_item_log_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_item_id text NULL,
    raw_action text NULL,
    raw_actor_user_id text NULL,
    raw_from_user_id text NULL,
    raw_to_user_id text NULL,
    raw_from_position text NULL,
    raw_to_position text NULL,
    raw_step_order text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_distribution_item_log_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_distribution_item_log_v1_spec CHECK (spec_id = 'legacy.distribution_item_log.v1'),
    CONSTRAINT stg_distribution_item_log_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_distribution_item_log_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_distribution_item_log_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_distribution_item_log_v1 TO crm_owner_integration;

-- legacy.lead.v1: raw sn_leads (11 columns; classification NONE)
CREATE TABLE integration.stg_lead_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_marketing_form_id text NULL,
    raw_import_code text NULL,
    raw_status text NULL,
    raw_lead_status text NULL,
    raw_destination_panel text NULL,
    raw_seller_id text NULL,
    raw_supervisor_id text NULL,
    raw_imported_at text NULL,
    raw_assigned_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_lead_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_lead_v1_spec CHECK (spec_id = 'legacy.lead.v1'),
    CONSTRAINT stg_lead_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_lead_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_lead_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_lead_v1 TO crm_owner_integration;

-- legacy.lead_status_history.v1: raw sn_lead_status_history (6 columns; classification NONE)
CREATE TABLE integration.stg_lead_status_history_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_lead_id text NULL,
    raw_user_id text NULL,
    raw_old_status text NULL,
    raw_new_status text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_lead_status_history_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_lead_status_history_v1_spec CHECK (spec_id = 'legacy.lead_status_history.v1'),
    CONSTRAINT stg_lead_status_history_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_lead_status_history_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_lead_status_history_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_lead_status_history_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_distribution_item_v1;
DROP TABLE integration.stg_distribution_item_log_v1;
DROP TABLE integration.stg_lead_v1;
DROP TABLE integration.stg_lead_status_history_v1;
