-- Integration raw staging, Slice 13 extension: purpose commission matrix,
-- commission rules, dry runs, dry run items and posting logs for the
-- commission READ / SHADOW parity (Architecture V1 §19, §24.3). Generated from
-- the published Slice 13 specs; regenerate with the package -update flag.

-- +goose Up

-- legacy.commission_dry_run.v1: raw sn_commission_dry_runs (13 columns; classification NONE)
CREATE TABLE integration.stg_commission_dry_run_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_run_code text NULL,
    raw_date_from text NULL,
    raw_date_to text NULL,
    raw_status text NULL,
    raw_created_by text NULL,
    raw_approval_status text NULL,
    raw_approved_by text NULL,
    raw_approved_at text NULL,
    raw_rejected_by text NULL,
    raw_rejected_at text NULL,
    raw_locked_at text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_commission_dry_run_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_commission_dry_run_v1_spec CHECK (spec_id = 'legacy.commission_dry_run.v1'),
    CONSTRAINT stg_commission_dry_run_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_commission_dry_run_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_commission_dry_run_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_commission_dry_run_v1 TO crm_owner_integration;

-- legacy.commission_dry_run_item.v1: raw sn_commission_dry_run_items (24 columns; classification NONE)
CREATE TABLE integration.stg_commission_dry_run_item_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_run_id text NULL,
    raw_invoice_id text NULL,
    raw_seller_user_id text NULL,
    raw_seller_position text NULL,
    raw_seller_level text NULL,
    raw_invoice_amount text NULL,
    raw_payment_method text NULL,
    raw_invoice_status text NULL,
    raw_matched_rule_id text NULL,
    raw_calculated_commission text NULL,
    raw_legacy_commission text NULL,
    raw_difference_amount text NULL,
    raw_item_status text NULL,
    raw_wallet_transaction_id text NULL,
    raw_posted_at text NULL,
    raw_posting_status text NULL,
    raw_compensation_history_id text NULL,
    raw_compensation_effective_from text NULL,
    raw_compensation_effective_to text NULL,
    raw_rule_source text NULL,
    raw_warning_code text NULL,
    raw_calculation_date text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_commission_dry_run_item_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_commission_dry_run_item_v1_spec CHECK (spec_id = 'legacy.commission_dry_run_item.v1'),
    CONSTRAINT stg_commission_dry_run_item_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_commission_dry_run_item_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_commission_dry_run_item_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_commission_dry_run_item_v1 TO crm_owner_integration;

-- legacy.commission_posting_log.v1: raw sn_commission_wallet_posting_logs (8 columns; classification NONE)
CREATE TABLE integration.stg_commission_posting_log_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_run_id text NULL,
    raw_item_id text NULL,
    raw_wallet_transaction_id text NULL,
    raw_actor_user_id text NULL,
    raw_action text NULL,
    raw_status text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_commission_posting_log_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_commission_posting_log_v1_spec CHECK (spec_id = 'legacy.commission_posting_log.v1'),
    CONSTRAINT stg_commission_posting_log_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_commission_posting_log_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_commission_posting_log_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_commission_posting_log_v1 TO crm_owner_integration;

-- legacy.commission_rule.v1: raw sn_commission_rules (22 columns; classification NONE)
CREATE TABLE integration.stg_commission_rule_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_rule_code text NULL,
    raw_applies_to_position text NULL,
    raw_applies_to_level text NULL,
    raw_payment_method text NULL,
    raw_invoice_status text NULL,
    raw_employment_status text NULL,
    raw_employment_type text NULL,
    raw_calculation_type text NULL,
    raw_rate_percent text NULL,
    raw_fixed_amount text NULL,
    raw_daily_base_amount text NULL,
    raw_attendance_first_before text NULL,
    raw_attendance_last_after text NULL,
    raw_min_invoice_amount text NULL,
    raw_max_invoice_amount text NULL,
    raw_effective_from text NULL,
    raw_effective_to text NULL,
    raw_is_active text NULL,
    raw_priority text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_commission_rule_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_commission_rule_v1_spec CHECK (spec_id = 'legacy.commission_rule.v1'),
    CONSTRAINT stg_commission_rule_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_commission_rule_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_commission_rule_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_commission_rule_v1 TO crm_owner_integration;

-- legacy.purpose_commission_rate.v1: raw sn_payment_commission_matrix (7 columns; classification NONE)
CREATE TABLE integration.stg_purpose_commission_rate_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_payment_purpose text NULL,
    raw_role_key text NULL,
    raw_rate_percent text NULL,
    raw_updated_by text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_purpose_commission_rate_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_purpose_commission_rate_v1_spec CHECK (spec_id = 'legacy.purpose_commission_rate.v1'),
    CONSTRAINT stg_purpose_commission_rate_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_purpose_commission_rate_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_purpose_commission_rate_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_purpose_commission_rate_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_commission_dry_run_v1;
DROP TABLE integration.stg_commission_dry_run_item_v1;
DROP TABLE integration.stg_commission_posting_log_v1;
DROP TABLE integration.stg_commission_rule_v1;
DROP TABLE integration.stg_purpose_commission_rate_v1;
