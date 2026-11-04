-- Fulfillment work model (Architecture V1 §13; Slice 14). Target-owned,
-- separate from the run-pinned Slice 10 shadow facts (legacy stays master
-- for every imported case; no write-back to legacy exists).
--
-- One work case per SubscriptionItem (never one case for a multi-service
-- subscription): its own identity, state, assignee, workflow and history.
-- work_case is the only mutable row (state, assignee, version) and is
-- changed only under its version (optimistic concurrency). Intake,
-- events, assignments, evidence metadata and command records are
-- append-only. Evidence is METADATA only (type, SHA-256, label): no file,
-- URL or path is stored. No money, wallet, credit or commission column.

-- +goose Up
CREATE TABLE fulfillment.work_intake (
    message_id      uuid        PRIMARY KEY,
    item_id         uuid        NOT NULL,
    subscription_id uuid        NOT NULL,
    service_key     text        NOT NULL,
    service_version integer     NOT NULL,
    decision        text        NOT NULL,
    reasons         jsonb       NOT NULL,
    case_id         uuid        NULL,
    correlation_id  text        NOT NULL,
    causation_id    text        NOT NULL,
    received_at     timestamptz NOT NULL,
    CONSTRAINT work_intake_decision CHECK (decision IN ('CASE_CREATED', 'BLOCKED', 'EXISTING_CASE')),
    CONSTRAINT work_intake_case CHECK ((decision = 'BLOCKED') = (case_id IS NULL)),
    CONSTRAINT work_intake_reasons CHECK (jsonb_typeof(reasons) = 'array')
);

CREATE TABLE fulfillment.work_case (
    case_id          uuid        PRIMARY KEY,
    item_id          uuid        NOT NULL UNIQUE,
    subscription_id  uuid        NOT NULL,
    customer_id      uuid        NOT NULL,
    service_key      text        NOT NULL,
    service_version  integer     NOT NULL,
    workflow_key     text        NOT NULL,
    origin_mode      text        NOT NULL,
    state            text        NOT NULL,
    assignee_user_id uuid        NULL,
    version          bigint      NOT NULL,
    source_message_id uuid       NOT NULL,
    correlation_id   text        NOT NULL,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    CONSTRAINT work_case_origin CHECK (origin_mode = 'SHADOW_COMPARE'),
    CONSTRAINT work_case_workflow CHECK (workflow_key = 'operational_task_v1'),
    CONSTRAINT work_case_state CHECK (state IN ('QUEUED', 'ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'NOT_EXECUTED')),
    CONSTRAINT work_case_assignee CHECK ((state = 'QUEUED') = (assignee_user_id IS NULL)),
    CONSTRAINT work_case_version CHECK (version > 0)
);
CREATE INDEX work_case_service_idx ON fulfillment.work_case (service_key, state, case_id);
CREATE INDEX work_case_subscription_idx ON fulfillment.work_case (subscription_id);

CREATE TABLE fulfillment.work_event (
    event_id          uuid        PRIMARY KEY,
    case_id           uuid        NOT NULL REFERENCES fulfillment.work_case (case_id),
    command_id        uuid        NULL,
    kind              text        NOT NULL,
    detail            jsonb       NOT NULL,
    actor_user_id     uuid        NULL,
    supervisors       jsonb       NOT NULL,
    before_version    bigint      NULL,
    after_version     bigint      NOT NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT work_event_kind CHECK (kind IN ('CREATED', 'ASSIGNED', 'ACTION_RECORDED', 'EVIDENCE_ATTACHED', 'COMPLETED')),
    CONSTRAINT work_event_detail CHECK (jsonb_typeof(detail) = 'object' AND jsonb_typeof(supervisors) = 'array')
);
CREATE INDEX work_event_case_idx ON fulfillment.work_event (case_id, occurred_at, event_id);

CREATE TABLE fulfillment.work_assignment (
    assignment_id         uuid        PRIMARY KEY,
    case_id               uuid        NOT NULL REFERENCES fulfillment.work_case (case_id),
    command_id            uuid        NOT NULL UNIQUE,
    assignee_user_id      uuid        NOT NULL,
    previous_assignee     uuid        NULL,
    assigned_by_user_id   uuid        NOT NULL,
    occurred_at           timestamptz NOT NULL
);

CREATE TABLE fulfillment.work_evidence (
    evidence_id     uuid        PRIMARY KEY,
    case_id         uuid        NOT NULL REFERENCES fulfillment.work_case (case_id),
    command_id      uuid        NOT NULL UNIQUE,
    evidence_type   text        NOT NULL,
    content_sha256  bytea       NOT NULL,
    label           text        NOT NULL,
    submitted_by    uuid        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT work_evidence_unique UNIQUE (case_id, evidence_type, content_sha256),
    CONSTRAINT work_evidence_sha CHECK (octet_length(content_sha256) = 32),
    CONSTRAINT work_evidence_label CHECK (length(label) BETWEEN 1 AND 200)
);

CREATE TABLE fulfillment.work_command_record (
    command_id       uuid        PRIMARY KEY,
    command_type     text        NOT NULL,
    case_id          uuid        NOT NULL REFERENCES fulfillment.work_case (case_id),
    idempotency_key  text        NOT NULL,
    request_sha256   bytea       NOT NULL,
    rollout_mode     text        NOT NULL,
    applied          boolean     NOT NULL,
    intent           jsonb       NOT NULL,
    before_version   bigint      NOT NULL,
    after_version    bigint      NULL,
    actor_principal  text        NOT NULL,
    actor_user_id    uuid        NOT NULL,
    correlation_id   text        NOT NULL,
    occurred_at      timestamptz NOT NULL,
    CONSTRAINT work_command_type CHECK (command_type IN ('fulfillment.assign_case', 'fulfillment.record_action', 'fulfillment.attach_evidence',
        'fulfillment.complete_case')),
    CONSTRAINT work_command_key UNIQUE (command_type, idempotency_key),
    CONSTRAINT work_command_sha CHECK (octet_length(request_sha256) = 32),
    CONSTRAINT work_command_mode CHECK (rollout_mode IN ('SHADOW_COMPARE', 'CONTROLLED_WRITEBACK')),
    CONSTRAINT work_command_applied CHECK (applied = (rollout_mode = 'CONTROLLED_WRITEBACK')),
    CONSTRAINT work_command_after CHECK ((after_version IS NOT NULL) = applied),
    CONSTRAINT work_command_intent CHECK (jsonb_typeof(intent) = 'object' AND pg_column_size(intent) <= 8192)
);

GRANT SELECT, INSERT, UPDATE ON fulfillment.work_case TO crm_owner_fulfillment;
GRANT SELECT, INSERT ON fulfillment.work_intake, fulfillment.work_event, fulfillment.work_assignment, fulfillment.work_evidence,
    fulfillment.work_command_record TO crm_owner_fulfillment;

-- work_case is never deleted (state, assignee and version change only under
-- the version guard; history lives in work_event).
CREATE TRIGGER work_case_no_delete BEFORE DELETE ON fulfillment.work_case FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_case_no_truncate BEFORE TRUNCATE ON fulfillment.work_case FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_intake_append_only BEFORE UPDATE OR DELETE ON fulfillment.work_intake FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_intake_no_truncate BEFORE TRUNCATE ON fulfillment.work_intake FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_event_append_only BEFORE UPDATE OR DELETE ON fulfillment.work_event FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_event_no_truncate BEFORE TRUNCATE ON fulfillment.work_event FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_assignment_append_only BEFORE UPDATE OR DELETE ON fulfillment.work_assignment FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_assignment_no_truncate BEFORE TRUNCATE ON fulfillment.work_assignment FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_evidence_append_only BEFORE UPDATE OR DELETE ON fulfillment.work_evidence FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_evidence_no_truncate BEFORE TRUNCATE ON fulfillment.work_evidence FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_command_record_append_only BEFORE UPDATE OR DELETE ON fulfillment.work_command_record FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER work_command_record_no_truncate BEFORE TRUNCATE ON fulfillment.work_command_record FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();

-- +goose Down
DROP TABLE IF EXISTS fulfillment.work_command_record;
DROP TABLE IF EXISTS fulfillment.work_evidence;
DROP TABLE IF EXISTS fulfillment.work_assignment;
DROP TABLE IF EXISTS fulfillment.work_event;
DROP TABLE IF EXISTS fulfillment.work_case;
DROP TABLE IF EXISTS fulfillment.work_intake;
