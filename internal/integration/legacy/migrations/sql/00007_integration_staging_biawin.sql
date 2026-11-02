-- Integration raw staging, Slice 10 extension: project membership, item, event,
-- invoice link and action, operations execution case / event / form submission
-- and shipping order / log sources for the Biawin / fulfillment READ shadow
-- (Architecture V1 §13, §24.3). Generated from the published Slice 10 specs;
-- regenerate with the package -update flag.

-- +goose Up

-- legacy.operations_execution_case.v1: raw sn_operations_execution_cases (22 columns; classification SENSITIVE)
CREATE TABLE integration.stg_operations_execution_case_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_operation_id text NULL,
    raw_membership_item_id text NULL,
    raw_content_product_id text NULL,
    raw_fulfillment_type text NULL,
    raw_manager_user_id text NULL,
    raw_expert_user_id text NULL,
    raw_assigned_by_user_id text NULL,
    raw_assigned_at text NULL,
    raw_status text NULL,
    raw_wallet_status text NULL,
    raw_wallet_external_id text NULL,
    raw_shipping_status text NULL,
    raw_shipping_carrier text NULL,
    raw_shipping_tracking_code text NULL,
    raw_form_submission_id text NULL,
    raw_final_decision text NULL,
    raw_completed_at text NULL,
    raw_completed_by_user_id text NULL,
    raw_cancelled_at text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_operations_execution_case_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_operations_execution_case_v1_spec CHECK (spec_id = 'legacy.operations_execution_case.v1'),
    CONSTRAINT stg_operations_execution_case_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_operations_execution_case_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_operations_execution_case_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_operations_execution_case_v1 TO crm_owner_integration;

-- legacy.operations_execution_event.v1: raw sn_operations_execution_events (7 columns; classification NONE)
CREATE TABLE integration.stg_operations_execution_event_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_execution_case_id text NULL,
    raw_operation_id text NULL,
    raw_membership_item_id text NULL,
    raw_actor_user_id text NULL,
    raw_event_key text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_operations_execution_event_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_operations_execution_event_v1_spec CHECK (spec_id = 'legacy.operations_execution_event.v1'),
    CONSTRAINT stg_operations_execution_event_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_operations_execution_event_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_operations_execution_event_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_operations_execution_event_v1 TO crm_owner_integration;

-- legacy.operations_form_submission.v1: raw sn_operations_form_submissions (8 columns; classification NONE)
CREATE TABLE integration.stg_operations_form_submission_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_execution_case_id text NULL,
    raw_form_definition_id text NULL,
    raw_submitted_by_user_id text NULL,
    raw_submitted_at text NULL,
    raw_final_decision text NULL,
    raw_decided_by_user_id text NULL,
    raw_decided_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_operations_form_submission_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_operations_form_submission_v1_spec CHECK (spec_id = 'legacy.operations_form_submission.v1'),
    CONSTRAINT stg_operations_form_submission_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_operations_form_submission_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_operations_form_submission_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_operations_form_submission_v1 TO crm_owner_integration;

-- legacy.project_action.v1: raw sn_project_actions (10 columns; classification NONE)
CREATE TABLE integration.stg_project_action_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_membership_item_id text NULL,
    raw_actor_user_id text NULL,
    raw_action_type text NULL,
    raw_fulfillment_mode text NULL,
    raw_target_product_id text NULL,
    raw_status text NULL,
    raw_invoice_id text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_project_action_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_project_action_v1_spec CHECK (spec_id = 'legacy.project_action.v1'),
    CONSTRAINT stg_project_action_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_project_action_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_project_action_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_project_action_v1 TO crm_owner_integration;

-- legacy.project_event.v1: raw sn_project_events (6 columns; classification NONE)
CREATE TABLE integration.stg_project_event_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_membership_id text NULL,
    raw_membership_item_id text NULL,
    raw_actor_user_id text NULL,
    raw_event_key text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_project_event_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_project_event_v1_spec CHECK (spec_id = 'legacy.project_event.v1'),
    CONSTRAINT stg_project_event_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_project_event_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_project_event_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_project_event_v1 TO crm_owner_integration;

-- legacy.project_invoice_link.v1: raw sn_project_invoice_links (7 columns; classification NONE)
CREATE TABLE integration.stg_project_invoice_link_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_membership_item_id text NULL,
    raw_invoice_id text NULL,
    raw_content_product_id text NULL,
    raw_status text NULL,
    raw_verified_at text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_project_invoice_link_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_project_invoice_link_v1_spec CHECK (spec_id = 'legacy.project_invoice_link.v1'),
    CONSTRAINT stg_project_invoice_link_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_project_invoice_link_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_project_invoice_link_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_project_invoice_link_v1 TO crm_owner_integration;

-- legacy.project_membership.v1: raw sn_project_memberships (14 columns; classification NONE)
CREATE TABLE integration.stg_project_membership_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_source_key text NULL,
    raw_source_type text NULL,
    raw_source_invoice_id text NULL,
    raw_dot_case_id text NULL,
    raw_subscription_product_id text NULL,
    raw_subscription_name_snapshot text NULL,
    raw_customer_wp_id text NULL,
    raw_origin_user_id text NULL,
    raw_first_payment_invoice_id text NULL,
    raw_status text NULL,
    raw_activated_at text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_project_membership_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_project_membership_v1_spec CHECK (spec_id = 'legacy.project_membership.v1'),
    CONSTRAINT stg_project_membership_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_project_membership_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_project_membership_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_project_membership_v1 TO crm_owner_integration;

-- legacy.project_membership_item.v1: raw sn_project_membership_items (11 columns; classification NONE)
CREATE TABLE integration.stg_project_membership_item_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_membership_id text NULL,
    raw_content_product_id text NULL,
    raw_content_name_snapshot text NULL,
    raw_manager_user_id text NULL,
    raw_expert_user_id text NULL,
    raw_workflow_status text NULL,
    raw_assigned_at text NULL,
    raw_status_updated_at text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_project_membership_item_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_project_membership_item_v1_spec CHECK (spec_id = 'legacy.project_membership_item.v1'),
    CONSTRAINT stg_project_membership_item_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_project_membership_item_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_project_membership_item_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_project_membership_item_v1 TO crm_owner_integration;

-- legacy.shipping_order.v1: raw sn_shipping_orders (14 columns; classification SENSITIVE)
CREATE TABLE integration.stg_shipping_order_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_id text NULL,
    raw_seller_id text NULL,
    raw_customer_wp_id text NULL,
    raw_contact_status text NULL,
    raw_fulfillment_status text NULL,
    raw_shipping_method text NULL,
    raw_carrier text NULL,
    raw_tracking_code text NULL,
    raw_handled_by text NULL,
    raw_shipped_at text NULL,
    raw_delivered_at text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_shipping_order_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_shipping_order_v1_spec CHECK (spec_id = 'legacy.shipping_order.v1'),
    CONSTRAINT stg_shipping_order_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_shipping_order_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_shipping_order_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_shipping_order_v1 TO crm_owner_integration;

-- legacy.shipping_order_log.v1: raw sn_shipping_order_logs (8 columns; classification NONE)
CREATE TABLE integration.stg_shipping_order_log_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_shipping_order_id text NULL,
    raw_invoice_id text NULL,
    raw_actor_user_id text NULL,
    raw_event_key text NULL,
    raw_old_status text NULL,
    raw_new_status text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_shipping_order_log_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_shipping_order_log_v1_spec CHECK (spec_id = 'legacy.shipping_order_log.v1'),
    CONSTRAINT stg_shipping_order_log_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_shipping_order_log_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_shipping_order_log_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_shipping_order_log_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_operations_execution_case_v1;
DROP TABLE integration.stg_operations_execution_event_v1;
DROP TABLE integration.stg_operations_form_submission_v1;
DROP TABLE integration.stg_project_action_v1;
DROP TABLE integration.stg_project_event_v1;
DROP TABLE integration.stg_project_invoice_link_v1;
DROP TABLE integration.stg_project_membership_v1;
DROP TABLE integration.stg_project_membership_item_v1;
DROP TABLE integration.stg_shipping_order_v1;
DROP TABLE integration.stg_shipping_order_log_v1;
