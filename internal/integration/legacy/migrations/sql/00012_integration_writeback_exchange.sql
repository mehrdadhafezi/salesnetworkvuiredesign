-- R2: evidence of every request / response exchanged with a legacy
-- write-back transport (the PROPOSED compatibility contract client,
-- internal/integration/legacy/compat). Append-only. It stores digests and
-- validation verdicts, never request or response bodies, credentials,
-- signatures, names, phones or amounts: the body digests let a later
-- reconciliation prove what was sent and received without retaining PII.
--
-- A row records what happened on the wire, not a business outcome: an
-- exchange whose response failed validation is evidence of an UNKNOWN
-- outcome, never of success or rejection.

-- +goose Up
CREATE TABLE integration.writeback_exchange (
    exchange_id      uuid        PRIMARY KEY,
    request_token    text        NOT NULL,
    operation        text        NOT NULL,
    contract         text        NOT NULL,
    correlation_id   text        NULL,
    request_sha256   bytea       NOT NULL,
    response_sha256  bytea       NULL,
    http_status      integer     NULL,
    verdict          text        NOT NULL,
    outcome          text        NULL,
    started_at       timestamptz NOT NULL,
    finished_at      timestamptz NOT NULL,
    CONSTRAINT writeback_exchange_token CHECK (request_token ~ '^[A-Za-z0-9_-]{8,64}$'),
    CONSTRAINT writeback_exchange_operation CHECK (operation IN ('lead_outcome.apply', 'lead_outcome.observe', 'hr_request.create',
        'hr_request.create.observe', 'hr_request.decide', 'hr_request.decide.observe')),
    CONSTRAINT writeback_exchange_contract CHECK (length(contract) BETWEEN 1 AND 64),
    CONSTRAINT writeback_exchange_correlation CHECK (correlation_id IS NULL OR length(correlation_id) BETWEEN 1 AND 128),
    CONSTRAINT writeback_exchange_digests CHECK (octet_length(request_sha256) = 32 AND (response_sha256 IS NULL OR octet_length(response_sha256) = 32)),
    CONSTRAINT writeback_exchange_status CHECK (http_status IS NULL OR http_status BETWEEN 100 AND 599),
    CONSTRAINT writeback_exchange_verdict CHECK (verdict IN ('VALID', 'TRANSPORT_ERROR', 'TIMEOUT', 'HTTP_STATUS', 'BAD_SIGNATURE',
        'INVALID_SCHEMA', 'MISMATCH')),
    CONSTRAINT writeback_exchange_outcome CHECK (outcome IS NULL OR outcome IN ('APPLIED', 'REJECTED', 'OBSERVED')),
    CONSTRAINT writeback_exchange_valid CHECK ((verdict = 'VALID') = (outcome IS NOT NULL)),
    CONSTRAINT writeback_exchange_order CHECK (started_at <= finished_at)
);
CREATE INDEX writeback_exchange_token_idx ON integration.writeback_exchange (request_token, started_at);

GRANT SELECT, INSERT ON integration.writeback_exchange TO crm_owner_integration;

-- +goose StatementBegin
CREATE FUNCTION integration.reject_writeback_exchange_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'integration.writeback_exchange: append-only write-back evidence; % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd
CREATE TRIGGER writeback_exchange_append_only BEFORE UPDATE OR DELETE ON integration.writeback_exchange
    FOR EACH ROW EXECUTE FUNCTION integration.reject_writeback_exchange_change();
CREATE TRIGGER writeback_exchange_no_truncate BEFORE TRUNCATE ON integration.writeback_exchange
    FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_writeback_exchange_change();

-- +goose Down
DROP TABLE IF EXISTS integration.writeback_exchange;
DROP FUNCTION IF EXISTS integration.reject_writeback_exchange_change();
