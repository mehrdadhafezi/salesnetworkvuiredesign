-- External provider operations (Architecture V1 §13, §24; Slice 14). Owned
-- by integration: the only place an external provider (Biawin) is reached,
-- through a BiawinPort. A domain never calls a provider.
--
-- provider_operation is one external operation identity (business key
-- unique, our own reference sent to the provider, request fingerprint,
-- correlation / causation). Its state is written AHEAD of every network
-- call (DISPATCHING), so a crash or timeout leaves UNKNOWN, never a
-- presumed success or failure. UNKNOWN is never retried blindly: it is
-- reconciled by a provider lookup or by recorded human evidence; a retry is
-- permitted only after absence is proven (CONFIRMED_ABSENT). Wallet charges
-- are representable only as BLOCKED (D-38). Events and callbacks are
-- append-only. No credential, URL, phone, name or money column.

-- +goose Up
CREATE TABLE integration.provider_operation (
    operation_id          uuid        PRIMARY KEY,
    operation_key         text        NOT NULL UNIQUE,
    provider              text        NOT NULL,
    op_type               text        NOT NULL,
    subject_case_id       uuid        NULL,
    reference             text        NOT NULL UNIQUE,
    request_fingerprint   bytea       NOT NULL,
    correlation_id        text        NOT NULL,
    causation_id          text        NOT NULL,
    state                 text        NOT NULL,
    blocked_by            text        NULL,
    attempts              integer     NOT NULL,
    external_ref          text        NULL,
    reconciliation_owner  text        NOT NULL,
    version               bigint      NOT NULL,
    created_at            timestamptz NOT NULL,
    updated_at            timestamptz NOT NULL,
    CONSTRAINT provider_operation_provider CHECK (provider IN ('BIAWIN')),
    CONSTRAINT provider_operation_type CHECK (op_type IN ('SERVICE_ORDER', 'WALLET_CHARGE')),
    CONSTRAINT provider_operation_state CHECK (state IN ('PREPARED', 'DISPATCHING', 'ACKNOWLEDGED', 'CONFIRMED', 'REJECTED', 'UNKNOWN',
        'RECONCILIATION_REQUIRED', 'CONFIRMED_ABSENT', 'BLOCKED')),
    CONSTRAINT provider_operation_wallet CHECK (op_type <> 'WALLET_CHARGE' OR (state = 'BLOCKED' AND blocked_by = 'D-38')),
    CONSTRAINT provider_operation_blocked CHECK ((state = 'BLOCKED') = (blocked_by IS NOT NULL)),
    CONSTRAINT provider_operation_fingerprint CHECK (octet_length(request_fingerprint) = 32),
    CONSTRAINT provider_operation_attempts CHECK (attempts >= 0 AND version > 0),
    CONSTRAINT provider_operation_reference CHECK (length(reference) BETWEEN 8 AND 191)
);
CREATE INDEX provider_operation_state_idx ON integration.provider_operation (state, updated_at);

CREATE TABLE integration.provider_operation_event (
    event_id        uuid        PRIMARY KEY,
    operation_id    uuid        NOT NULL REFERENCES integration.provider_operation (operation_id),
    seq             bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    from_state      text        NULL,
    to_state        text        NOT NULL,
    kind            text        NOT NULL,
    evidence        jsonb       NOT NULL,
    actor           text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT provider_operation_event_kind CHECK (kind IN ('REQUESTED', 'DISPATCH_STARTED', 'ACKNOWLEDGED', 'REJECTED', 'OUTCOME_UNKNOWN',
        'LOOKUP_FOUND', 'LOOKUP_ABSENT', 'LOOKUP_UNAVAILABLE', 'CALLBACK_CONFIRMED', 'CALLBACK_REJECTED', 'REFERENCE_CONFLICT', 'RESOLVED_PRESENT',
        'RESOLVED_ABSENT', 'BLOCKED')),
    CONSTRAINT provider_operation_event_evidence CHECK (jsonb_typeof(evidence) = 'object' AND pg_column_size(evidence) <= 8192)
);

CREATE TABLE integration.provider_callback (
    callback_id     text        PRIMARY KEY,
    provider        text        NOT NULL,
    reference       text        NOT NULL,
    operation_id    uuid        NULL REFERENCES integration.provider_operation (operation_id),
    external_ref    text        NULL,
    status          text        NOT NULL,
    disposition     text        NOT NULL,
    body_sha256     bytea       NOT NULL,
    received_at     timestamptz NOT NULL,
    CONSTRAINT provider_callback_status CHECK (status IN ('CONFIRMED', 'REJECTED')),
    CONSTRAINT provider_callback_disposition CHECK (disposition IN ('APPLIED', 'EARLY_APPLIED', 'ALREADY_FINAL', 'CONFLICT', 'UNMATCHED')),
    CONSTRAINT provider_callback_sha CHECK (octet_length(body_sha256) = 32)
);

GRANT SELECT, INSERT, UPDATE ON integration.provider_operation TO crm_owner_integration;
GRANT SELECT, INSERT ON integration.provider_operation_event, integration.provider_callback TO crm_owner_integration;

-- +goose StatementBegin
CREATE FUNCTION integration.reject_provider_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'integration.%: append-only provider evidence; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER provider_operation_no_delete BEFORE DELETE ON integration.provider_operation FOR EACH ROW EXECUTE FUNCTION integration.reject_provider_history_change();
CREATE TRIGGER provider_operation_no_truncate BEFORE TRUNCATE ON integration.provider_operation FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_provider_history_change();
CREATE TRIGGER provider_operation_event_append_only BEFORE UPDATE OR DELETE ON integration.provider_operation_event FOR EACH ROW EXECUTE FUNCTION integration.reject_provider_history_change();
CREATE TRIGGER provider_operation_event_no_truncate BEFORE TRUNCATE ON integration.provider_operation_event FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_provider_history_change();
CREATE TRIGGER provider_callback_append_only BEFORE UPDATE OR DELETE ON integration.provider_callback FOR EACH ROW EXECUTE FUNCTION integration.reject_provider_history_change();
CREATE TRIGGER provider_callback_no_truncate BEFORE TRUNCATE ON integration.provider_callback FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_provider_history_change();

-- +goose Down
DROP TABLE IF EXISTS integration.provider_callback;
DROP TABLE IF EXISTS integration.provider_operation_event;
DROP TABLE IF EXISTS integration.provider_operation;
DROP FUNCTION IF EXISTS integration.reject_provider_history_change();
