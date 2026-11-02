-- Cases controlled writes (Slice 11): RecordContactOutcome only.
--
-- contact_outcome_current is the case's current sales-workflow contact
-- outcome, written only by the command under the case version (optimistic
-- concurrency on cases.work_case.version). It is separate from the imported
-- case state, the ready state, the invoice lifecycle and every payment
-- state: none of them is changed or implied by an outcome.
--
-- contact_outcome_event and command_record are append-only evidence: actor,
-- before / after, reason, correlation and time. No UPDATE, DELETE or
-- TRUNCATE (no grant; a trigger rejects them even for the owner).
-- command_record also keeps SHADOW_COMPARE evaluations (applied = false).
-- Phone, names and legacy table names never appear here.

-- +goose Up
CREATE TABLE cases.contact_outcome_current (
    case_id          uuid        PRIMARY KEY REFERENCES cases.work_case (case_id),
    outcome          text        NOT NULL,
    reason           text        NULL,
    callback_at      timestamptz NULL,
    last_command_id  uuid        NOT NULL,
    recorded_at      timestamptz NOT NULL,
    CONSTRAINT contact_outcome_current_outcome CHECK (outcome IN ('NO_ANSWER', 'CALLBACK', 'NOT_PURCHASED')),
    CONSTRAINT contact_outcome_current_reason CHECK ((outcome = 'NOT_PURCHASED') = (reason IS NOT NULL) AND (reason IS NULL OR length(reason) BETWEEN 1 AND 1000)),
    CONSTRAINT contact_outcome_current_callback CHECK ((outcome = 'CALLBACK') = (callback_at IS NOT NULL))
);

CREATE TABLE cases.contact_outcome_event (
    event_id          uuid        PRIMARY KEY,
    case_id           uuid        NOT NULL REFERENCES cases.work_case (case_id),
    command_id        uuid        NOT NULL UNIQUE,
    outcome           text        NOT NULL,
    reason            text        NULL,
    callback_at       timestamptz NULL,
    previous_outcome  text        NULL,
    before_version    bigint      NOT NULL,
    after_version     bigint      NOT NULL,
    actor_principal   text        NOT NULL,
    actor_user_id     uuid        NOT NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT contact_outcome_event_outcome CHECK (outcome IN ('NO_ANSWER', 'CALLBACK', 'NOT_PURCHASED')),
    CONSTRAINT contact_outcome_event_previous CHECK (previous_outcome IS NULL OR previous_outcome IN ('NO_ANSWER', 'CALLBACK', 'NOT_PURCHASED')),
    CONSTRAINT contact_outcome_event_reason CHECK ((outcome = 'NOT_PURCHASED') = (reason IS NOT NULL) AND (reason IS NULL OR length(reason) BETWEEN 1 AND 1000)),
    CONSTRAINT contact_outcome_event_callback CHECK ((outcome = 'CALLBACK') = (callback_at IS NOT NULL)),
    CONSTRAINT contact_outcome_event_versions CHECK (after_version = before_version + 1 AND before_version > 0),
    CONSTRAINT contact_outcome_event_actor CHECK (length(actor_principal) BETWEEN 1 AND 128),
    CONSTRAINT contact_outcome_event_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX contact_outcome_event_case_idx ON cases.contact_outcome_event (case_id, occurred_at, event_id);

CREATE TABLE cases.command_record (
    command_id        uuid        PRIMARY KEY,
    command_type      text        NOT NULL,
    case_id           uuid        NOT NULL REFERENCES cases.work_case (case_id),
    idempotency_key   text        NOT NULL,
    request_sha256    bytea       NOT NULL,
    rollout_mode      text        NOT NULL,
    applied           boolean     NOT NULL,
    intent            jsonb       NOT NULL,
    before_version    bigint      NOT NULL,
    after_version     bigint      NULL,
    actor_principal   text        NOT NULL,
    actor_user_id     uuid        NOT NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT command_record_type CHECK (command_type IN ('cases.record_contact_outcome')),
    CONSTRAINT command_record_key UNIQUE (command_type, idempotency_key),
    CONSTRAINT command_record_sha CHECK (octet_length(request_sha256) = 32),
    CONSTRAINT command_record_mode CHECK (rollout_mode IN ('SHADOW_COMPARE', 'CONTROLLED_WRITEBACK')),
    CONSTRAINT command_record_applied CHECK (applied = (rollout_mode = 'CONTROLLED_WRITEBACK')),
    CONSTRAINT command_record_after CHECK ((after_version IS NOT NULL) = applied),
    CONSTRAINT command_record_intent CHECK (jsonb_typeof(intent) = 'object' AND pg_column_size(intent) <= 8192),
    CONSTRAINT command_record_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX command_record_case_idx ON cases.command_record (case_id, occurred_at);

GRANT SELECT, INSERT, UPDATE ON cases.contact_outcome_current TO crm_owner_cases;
GRANT SELECT, INSERT ON cases.contact_outcome_event, cases.command_record TO crm_owner_cases;

CREATE TRIGGER contact_outcome_event_append_only BEFORE UPDATE OR DELETE ON cases.contact_outcome_event
    FOR EACH ROW EXECUTE FUNCTION cases.reject_history_change();
CREATE TRIGGER contact_outcome_event_no_truncate BEFORE TRUNCATE ON cases.contact_outcome_event
    FOR EACH STATEMENT EXECUTE FUNCTION cases.reject_history_change();
CREATE TRIGGER command_record_append_only BEFORE UPDATE OR DELETE ON cases.command_record
    FOR EACH ROW EXECUTE FUNCTION cases.reject_history_change();
CREATE TRIGGER command_record_no_truncate BEFORE TRUNCATE ON cases.command_record
    FOR EACH STATEMENT EXECUTE FUNCTION cases.reject_history_change();

-- +goose Down
DROP TABLE IF EXISTS cases.command_record;
DROP TABLE IF EXISTS cases.contact_outcome_event;
DROP TABLE IF EXISTS cases.contact_outcome_current;
