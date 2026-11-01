-- Integration raw staging, Slice 7 extension: invoice, invoice item, invoice
-- payment stage and payment sources (Architecture V1 §10, §24.3). Generated
-- from the published Slice 7 specs; regenerate with the package -update flag.

-- +goose Up

-- legacy.invoice.v2: raw sn_invoices (33 columns; classification NONE)
CREATE TABLE integration.stg_invoice_v2 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_code text NULL,
    raw_seller_id text NULL,
    raw_issued_by_user_id text NULL,
    raw_lead_id text NULL,
    raw_product_id text NULL,
    raw_product_price text NULL,
    raw_pay_method text NULL,
    raw_status text NULL,
    raw_invoice_status text NULL,
    raw_payment_status text NULL,
    raw_receipt_url text NULL,
    raw_receipt_file text NULL,
    raw_receipt_urls text NULL,
    raw_receipt_source text NULL,
    raw_payment_source text NULL,
    raw_paid_at text NULL,
    raw_approved_at text NULL,
    raw_financial_reviewed_at text NULL,
    raw_original_total text NULL,
    raw_discount_total text NULL,
    raw_final_total text NULL,
    raw_payment_plan text NULL,
    raw_payment_total_amount text NULL,
    raw_current_due_amount text NULL,
    raw_paid_total_amount text NULL,
    raw_remaining_amount text NULL,
    raw_current_payment_stage text NULL,
    raw_payment_workflow_status text NULL,
    raw_payment_completed_at text NULL,
    raw_financial_return_state text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_invoice_v2_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_invoice_v2_spec CHECK (spec_id = 'legacy.invoice.v2'),
    CONSTRAINT stg_invoice_v2_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_invoice_v2_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_invoice_v2_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_invoice_v2 TO crm_owner_integration;

-- legacy.invoice_item.v1: raw sn_invoice_items (10 columns; classification NONE)
CREATE TABLE integration.stg_invoice_item_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_id text NULL,
    raw_product_id text NULL,
    raw_product_name text NULL,
    raw_product_type text NULL,
    raw_qty text NULL,
    raw_unit_price text NULL,
    raw_total_price text NULL,
    raw_is_free text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_invoice_item_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_invoice_item_v1_spec CHECK (spec_id = 'legacy.invoice_item.v1'),
    CONSTRAINT stg_invoice_item_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_invoice_item_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_invoice_item_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_invoice_item_v1 TO crm_owner_integration;

-- legacy.invoice_payment_stage.v1: raw sn_invoice_payment_stages (15 columns; classification NONE)
CREATE TABLE integration.stg_invoice_payment_stage_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_id text NULL,
    raw_stage_no text NULL,
    raw_stage_type text NULL,
    raw_requested_amount text NULL,
    raw_status text NULL,
    raw_pay_method text NULL,
    raw_payment_source text NULL,
    raw_receipt_urls text NULL,
    raw_issued_by_user_id text NULL,
    raw_approved_by_user_id text NULL,
    raw_approved_at text NULL,
    raw_paid_at text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_invoice_payment_stage_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_invoice_payment_stage_v1_spec CHECK (spec_id = 'legacy.invoice_payment_stage.v1'),
    CONSTRAINT stg_invoice_payment_stage_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_invoice_payment_stage_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_invoice_payment_stage_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_invoice_payment_stage_v1 TO crm_owner_integration;

-- legacy.payment.v1: raw sn_payments (13 columns; classification SENSITIVE)
CREATE TABLE integration.stg_payment_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_id text NULL,
    raw_payment_stage_no text NULL,
    raw_authority text NULL,
    raw_ref_id text NULL,
    raw_amount text NULL,
    raw_status text NULL,
    raw_pay_method text NULL,
    raw_payment_source text NULL,
    raw_uploaded_by_type text NULL,
    raw_uploaded_by_user_id text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_payment_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_payment_v1_spec CHECK (spec_id = 'legacy.payment.v1'),
    CONSTRAINT stg_payment_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_payment_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_payment_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_payment_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_invoice_v2;
DROP TABLE integration.stg_invoice_item_v1;
DROP TABLE integration.stg_invoice_payment_stage_v1;
DROP TABLE integration.stg_payment_v1;
