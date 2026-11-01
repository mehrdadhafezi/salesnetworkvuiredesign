-- Integration raw staging, Slice 9 extension: legacy wallet balance and wallet
-- transaction sources for the finance SHADOW ledger (Architecture V1 §16, §24.3).
-- Generated from the published Slice 9 specs; regenerate with the package
-- -update flag.

-- +goose Up

-- legacy.wallet.v1: raw sn_wallets (8 columns; classification NONE)
CREATE TABLE integration.stg_wallet_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_user_id text NULL,
    raw_wallet_type text NULL,
    raw_balance text NULL,
    raw_total_credit text NULL,
    raw_total_debit text NULL,
    raw_created_at text NULL,
    raw_updated_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_wallet_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_wallet_v1_spec CHECK (spec_id = 'legacy.wallet.v1'),
    CONSTRAINT stg_wallet_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_wallet_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_wallet_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_wallet_v1 TO crm_owner_integration;

-- legacy.wallet_transaction.v1: raw sn_wallet_transactions (13 columns; classification SENSITIVE)
CREATE TABLE integration.stg_wallet_transaction_v1 (
    batch_id       uuid        NOT NULL,
    spec_id        text        NOT NULL,
    source_ordinal bigint      NOT NULL,
    source_key     text        NOT NULL,
    row_sha256     bytea       NOT NULL,
    imported_at    timestamptz NOT NULL,
    raw_id text NULL,
    raw_wallet_id text NULL,
    raw_user_id text NULL,
    raw_wallet_type text NULL,
    raw_invoice_id text NULL,
    raw_lead_id text NULL,
    raw_amount text NULL,
    raw_direction text NULL,
    raw_type text NULL,
    raw_status text NULL,
    raw_meta text NULL,
    raw_idempotency_key text NULL,
    raw_created_at text NULL,
    PRIMARY KEY (batch_id, source_ordinal),
    CONSTRAINT stg_wallet_transaction_v1_key_unique UNIQUE (batch_id, source_key),
    CONSTRAINT stg_wallet_transaction_v1_spec CHECK (spec_id = 'legacy.wallet_transaction.v1'),
    CONSTRAINT stg_wallet_transaction_v1_ordinal CHECK (source_ordinal > 0),
    CONSTRAINT stg_wallet_transaction_v1_row_sha CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT stg_wallet_transaction_v1_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)
);
GRANT SELECT, INSERT ON integration.stg_wallet_transaction_v1 TO crm_owner_integration;

-- +goose Down
DROP TABLE integration.stg_wallet_v1;
DROP TABLE integration.stg_wallet_transaction_v1;
