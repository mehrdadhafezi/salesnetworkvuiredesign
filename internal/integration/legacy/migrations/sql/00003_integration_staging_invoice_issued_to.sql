-- Integration raw staging, Slice 5 extension: invoice ISSUED-TO identity
-- evidence only (Architecture V1 §10, §24.3). Generated from the published
-- Slice 5 specs; regenerate with the package -update flag.

-- +goose Up

-- legacy.invoice_issued_to.v1: raw sn_invoices (7 columns; classification PERSONAL)
CREATE TABLE integration.stg_invoice_issued_to_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_invoice_code text NULL,
    raw_customer_wp_id text NULL,
    raw_customer_name text NULL,
    raw_customer_phone text NULL,
    raw_customer_phone_secondary text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_invoice_issued_to_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_invoice_issued_to_v1_spec CHECK (spec_id = 'legacy.invoice_issued_to.v1'),
    CONSTRAINT stg_invoice_issued_to_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_invoice_issued_to_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_invoice_issued_to_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_invoice_issued_to_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_invoice_issued_to_v1;
