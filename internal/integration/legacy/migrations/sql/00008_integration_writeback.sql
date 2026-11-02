-- Integration legacy write-back (Slice 11 controlled strangler).
--
-- Integration consumes the cases and organization outbox events (inbox
-- deduplicated: one write-back command per (source module, event)), writes
-- back to the legacy master through the write-back port, and records every
-- attempt, reconciliation and parity observation. Legacy table names and
-- request shapes live only in the integration legacy tree; these tables keep
-- the intent and the outcome evidence.
--
-- writeback_command is the current delivery state of one command (mutable,
-- versioned; never deleted). writeback_attempt, writeback_reconciliation and
-- writeback_parity are append-only evidence (no UPDATE / DELETE grant; a
-- trigger rejects them even for the owner).
--
-- An UNKNOWN attempt is never turned into FAILED: the next step for it is an
-- OBSERVE (reconcile) attempt, never a blind APPLY. A REJECTED write-back never
-- rewrites target history; it opens a reconciliation record instead.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION integration.reject_writeback_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'integration.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TABLE integration.writeback_command (
    command_id             uuid        PRIMARY KEY,
    source_module          text        NOT NULL,
    event_id               uuid        NOT NULL,
    event_type             text        NOT NULL,
    subject_kind           text        NOT NULL,
    subject_id             uuid        NOT NULL,
    idempotency_key        text        NOT NULL,
    correlation_id         text        NOT NULL,
    rollout_mode           text        NOT NULL,
    intent                 jsonb       NOT NULL,
    depends_on_command_id  uuid        NULL,
    state                  text        NOT NULL,
    attempts               integer     NOT NULL,
    legacy_ref             text        NULL,
    parity                 text        NULL,
    version                bigint      NOT NULL,
    received_at            timestamptz NOT NULL,
    updated_at             timestamptz NOT NULL,
    CONSTRAINT writeback_command_event UNIQUE (source_module, event_id),
    CONSTRAINT writeback_command_module CHECK (source_module IN ('cases', 'organization')),
    CONSTRAINT writeback_command_subject CHECK (subject_kind IN ('case', 'hr_request')),
    CONSTRAINT writeback_command_mode CHECK (rollout_mode IN ('SHADOW_COMPARE', 'CONTROLLED_WRITEBACK')),
    CONSTRAINT writeback_command_state CHECK (state IN ('PENDING', 'WAITING_DEPENDENCY', 'APPLIED', 'UNKNOWN', 'REJECTED',
        'RECONCILIATION_REQUIRED', 'COMPARED')),
    CONSTRAINT writeback_command_shadow CHECK (rollout_mode = 'CONTROLLED_WRITEBACK' OR state IN ('PENDING', 'COMPARED', 'RECONCILIATION_REQUIRED')),
    CONSTRAINT writeback_command_parity CHECK (parity IS NULL OR parity IN ('MATCH', 'MISMATCH', 'UNKNOWN', 'CONFLICT', 'INCOMPLETE')),
    CONSTRAINT writeback_command_intent CHECK (jsonb_typeof(intent) = 'object' AND pg_column_size(intent) <= 16384),
    CONSTRAINT writeback_command_attempts CHECK (attempts >= 0),
    CONSTRAINT writeback_command_legacy_ref CHECK (legacy_ref IS NULL OR length(legacy_ref) BETWEEN 1 AND 128),
    CONSTRAINT writeback_command_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128),
    CONSTRAINT writeback_command_version CHECK (version > 0)
);
CREATE INDEX writeback_command_state_idx ON integration.writeback_command (state, received_at);
CREATE INDEX writeback_command_subject_idx ON integration.writeback_command (subject_kind, subject_id, received_at);

CREATE TABLE integration.writeback_attempt (
    attempt_id      uuid        PRIMARY KEY,
    command_id      uuid        NOT NULL REFERENCES integration.writeback_command (command_id),
    attempt_no      integer     NOT NULL,
    kind            text        NOT NULL,
    outcome         text        NOT NULL,
    observation     text        NULL,
    reasons         jsonb       NOT NULL,
    legacy_ref      text        NULL,
    detail          text        NULL,
    service_actor   text        NOT NULL,
    correlation_id  text        NOT NULL,
    started_at      timestamptz NOT NULL,
    finished_at     timestamptz NOT NULL,
    CONSTRAINT writeback_attempt_no UNIQUE (command_id, attempt_no),
    CONSTRAINT writeback_attempt_kind CHECK (kind IN ('APPLY', 'OBSERVE')),
    CONSTRAINT writeback_attempt_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED', 'UNAUTHORIZED')),
    CONSTRAINT writeback_attempt_observation CHECK ((kind = 'OBSERVE') = (observation IS NOT NULL)
        AND (observation IS NULL OR observation IN ('COMMITTED', 'ABSENT', 'AMBIGUOUS', 'UNKNOWN'))),
    CONSTRAINT writeback_attempt_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT writeback_attempt_detail CHECK (detail IS NULL OR length(detail) <= 300),
    CONSTRAINT writeback_attempt_order CHECK (started_at <= finished_at AND attempt_no > 0)
);

CREATE TABLE integration.writeback_reconciliation (
    reconciliation_id  uuid        PRIMARY KEY,
    command_id         uuid        NOT NULL REFERENCES integration.writeback_command (command_id),
    domain_owner       text        NOT NULL,
    subject_kind       text        NOT NULL,
    subject_id         uuid        NOT NULL,
    command_type       text        NOT NULL,
    reason             text        NOT NULL,
    attempt_no         integer     NOT NULL,
    evidence           jsonb       NOT NULL,
    next_safe_action   text        NOT NULL,
    correlation_id     text        NOT NULL,
    opened_at          timestamptz NOT NULL,
    CONSTRAINT writeback_reconciliation_once UNIQUE (command_id, reason, attempt_no),
    CONSTRAINT writeback_reconciliation_owner CHECK (domain_owner IN ('cases', 'organization')),
    CONSTRAINT writeback_reconciliation_reason CHECK (reason IN ('WRITEBACK_UNKNOWN', 'WRITEBACK_REJECTED', 'PARITY_MISMATCH', 'LEGACY_AMBIGUOUS')),
    CONSTRAINT writeback_reconciliation_action CHECK (next_safe_action IN ('OBSERVE_LEGACY_BEFORE_RETRY', 'DOMAIN_OWNER_REVIEW', 'NO_AUTOMATIC_ACTION')),
    CONSTRAINT writeback_reconciliation_evidence CHECK (jsonb_typeof(evidence) = 'object' AND pg_column_size(evidence) <= 8192)
);
CREATE INDEX writeback_reconciliation_reason_idx ON integration.writeback_reconciliation (reason, opened_at);

CREATE TABLE integration.writeback_parity (
    parity_id       uuid        PRIMARY KEY,
    command_id      uuid        NOT NULL REFERENCES integration.writeback_command (command_id),
    rollout_mode    text        NOT NULL,
    result          text        NOT NULL,
    expected        jsonb       NOT NULL,
    observed        jsonb       NOT NULL,
    correlation_id  text        NOT NULL,
    observed_at     timestamptz NOT NULL,
    CONSTRAINT writeback_parity_once UNIQUE (command_id),
    CONSTRAINT writeback_parity_mode CHECK (rollout_mode IN ('SHADOW_COMPARE', 'CONTROLLED_WRITEBACK')),
    CONSTRAINT writeback_parity_result CHECK (result IN ('MATCH', 'MISMATCH', 'UNKNOWN', 'CONFLICT', 'INCOMPLETE')),
    CONSTRAINT writeback_parity_json CHECK (jsonb_typeof(expected) = 'object' AND jsonb_typeof(observed) = 'object')
);

GRANT SELECT, INSERT ON integration.writeback_command TO crm_owner_integration;
GRANT UPDATE (state, attempts, legacy_ref, parity, depends_on_command_id, version, updated_at) ON integration.writeback_command TO crm_owner_integration;
GRANT SELECT, INSERT ON integration.writeback_attempt, integration.writeback_reconciliation, integration.writeback_parity TO crm_owner_integration;

CREATE TRIGGER writeback_command_no_delete BEFORE DELETE ON integration.writeback_command
    FOR EACH ROW EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_command_no_truncate BEFORE TRUNCATE ON integration.writeback_command
    FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_attempt_append_only BEFORE UPDATE OR DELETE ON integration.writeback_attempt
    FOR EACH ROW EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_attempt_no_truncate BEFORE TRUNCATE ON integration.writeback_attempt
    FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_reconciliation_append_only BEFORE UPDATE OR DELETE ON integration.writeback_reconciliation
    FOR EACH ROW EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_reconciliation_no_truncate BEFORE TRUNCATE ON integration.writeback_reconciliation
    FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_parity_append_only BEFORE UPDATE OR DELETE ON integration.writeback_parity
    FOR EACH ROW EXECUTE FUNCTION integration.reject_writeback_history_change();
CREATE TRIGGER writeback_parity_no_truncate BEFORE TRUNCATE ON integration.writeback_parity
    FOR EACH STATEMENT EXECUTE FUNCTION integration.reject_writeback_history_change();

-- +goose Down
DROP TABLE IF EXISTS integration.writeback_parity;
DROP TABLE IF EXISTS integration.writeback_reconciliation;
DROP TABLE IF EXISTS integration.writeback_attempt;
DROP TABLE IF EXISTS integration.writeback_command;
DROP FUNCTION IF EXISTS integration.reject_writeback_history_change();
