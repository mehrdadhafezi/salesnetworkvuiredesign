-- Organization controlled writes (Slice 11): HR change requests.
--
-- Only the evidenced legacy request types (transfer_seller, terminate_seller)
-- and the evidenced request / step-review / rejection transitions exist.
-- Final apply is not implemented (OPD-05, SD-04; HR apply is IMPLEMENTATION
-- BLOCKED): no request reaches 'approved' or 'failed' in Slice 11, so no
-- hierarchy, employment, credential or grant is ever changed by a request.
--
-- Requester, subject current parent, target and current reviewer are
-- snapshots taken at the transition; they are never re-derived from today's
-- hierarchy. hr_request_event and command_record are append-only (no UPDATE
-- or DELETE grant; a trigger rejects them even for the owner). A partial
-- unique index enforces the legacy one-open-request-per-subject-and-type
-- guard under concurrency.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION organization.reject_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'organization.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TABLE organization.hr_request (
    request_id                    uuid        PRIMARY KEY,
    request_type                  text        NOT NULL,
    subject_user_id               uuid        NOT NULL,
    subject_employment_id         uuid        NOT NULL REFERENCES organization.employment (employment_id),
    subject_parent_employment_id  uuid        NULL REFERENCES organization.employment (employment_id),
    subject_parent_user_id        uuid        NULL,
    target_parent_employment_id   uuid        NULL REFERENCES organization.employment (employment_id),
    target_parent_user_id         uuid        NULL,
    requested_by_user_id          uuid        NOT NULL,
    requested_by_employment_id    uuid        NOT NULL REFERENCES organization.employment (employment_id),
    current_reviewer_user_id      uuid        NULL,
    status                        text        NOT NULL,
    reason                        text        NOT NULL,
    reviewer_notes                text        NULL,
    created_command_id            uuid        NOT NULL UNIQUE,
    version                       bigint      NOT NULL,
    created_at                    timestamptz NOT NULL,
    updated_at                    timestamptz NOT NULL,
    CONSTRAINT hr_request_type CHECK (request_type IN ('transfer_seller', 'terminate_seller')),
    CONSTRAINT hr_request_status CHECK (status IN ('pending_review', 'pending_hr', 'rejected')),
    CONSTRAINT hr_request_target CHECK ((request_type = 'transfer_seller') = (target_parent_user_id IS NOT NULL AND target_parent_employment_id IS NOT NULL)),
    CONSTRAINT hr_request_reviewer CHECK ((status = 'pending_review') = (current_reviewer_user_id IS NOT NULL)),
    CONSTRAINT hr_request_reason CHECK (length(reason) BETWEEN 1 AND 1000),
    CONSTRAINT hr_request_notes CHECK (reviewer_notes IS NULL OR length(reviewer_notes) BETWEEN 1 AND 1000),
    CONSTRAINT hr_request_not_self CHECK (subject_user_id <> requested_by_user_id),
    CONSTRAINT hr_request_version CHECK (version > 0)
);
CREATE UNIQUE INDEX hr_request_one_open ON organization.hr_request (subject_user_id, request_type)
    WHERE status IN ('pending_review', 'pending_hr');
CREATE INDEX hr_request_reviewer_idx ON organization.hr_request (current_reviewer_user_id, status);
CREATE INDEX hr_request_requester_idx ON organization.hr_request (requested_by_user_id, created_at);

CREATE TABLE organization.hr_request_event (
    event_id               uuid        PRIMARY KEY,
    request_id             uuid        NOT NULL REFERENCES organization.hr_request (request_id),
    command_id             uuid        NOT NULL UNIQUE,
    action                 text        NOT NULL,
    from_status            text        NULL,
    to_status              text        NOT NULL,
    next_reviewer_user_id  uuid        NULL,
    notes                  text        NULL,
    before_version         bigint      NULL,
    after_version          bigint      NOT NULL,
    actor_principal        text        NOT NULL,
    actor_user_id          uuid        NOT NULL,
    correlation_id         text        NOT NULL,
    occurred_at            timestamptz NOT NULL,
    CONSTRAINT hr_request_event_action CHECK (action IN ('created', 'approved_step', 'rejected')),
    CONSTRAINT hr_request_event_to CHECK (to_status IN ('pending_review', 'pending_hr', 'rejected')),
    CONSTRAINT hr_request_event_from CHECK ((action = 'created') = (from_status IS NULL AND before_version IS NULL)),
    CONSTRAINT hr_request_event_notes CHECK (notes IS NULL OR length(notes) BETWEEN 1 AND 1000),
    CONSTRAINT hr_request_event_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX hr_request_event_request_idx ON organization.hr_request_event (request_id, occurred_at, event_id);

CREATE TABLE organization.command_record (
    command_id        uuid        PRIMARY KEY,
    command_type      text        NOT NULL,
    subject_ref       uuid        NOT NULL,
    idempotency_key   text        NOT NULL,
    request_sha256    bytea       NOT NULL,
    rollout_mode      text        NOT NULL,
    applied           boolean     NOT NULL,
    intent            jsonb       NOT NULL,
    actor_principal   text        NOT NULL,
    actor_user_id     uuid        NOT NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT command_record_type CHECK (command_type IN ('organization.create_hr_request', 'organization.decide_hr_request')),
    CONSTRAINT command_record_key UNIQUE (command_type, idempotency_key),
    CONSTRAINT command_record_sha CHECK (octet_length(request_sha256) = 32),
    CONSTRAINT command_record_mode CHECK (rollout_mode IN ('SHADOW_COMPARE', 'CONTROLLED_WRITEBACK')),
    CONSTRAINT command_record_applied CHECK (applied = (rollout_mode = 'CONTROLLED_WRITEBACK')),
    CONSTRAINT command_record_intent CHECK (jsonb_typeof(intent) = 'object' AND pg_column_size(intent) <= 8192),
    CONSTRAINT command_record_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);

GRANT SELECT, INSERT ON organization.hr_request_event, organization.command_record TO crm_owner_organization;
GRANT SELECT, INSERT ON organization.hr_request TO crm_owner_organization;
GRANT UPDATE (current_reviewer_user_id, status, reviewer_notes, version, updated_at) ON organization.hr_request TO crm_owner_organization;

CREATE TRIGGER hr_request_event_append_only BEFORE UPDATE OR DELETE ON organization.hr_request_event
    FOR EACH ROW EXECUTE FUNCTION organization.reject_history_change();
CREATE TRIGGER hr_request_event_no_truncate BEFORE TRUNCATE ON organization.hr_request_event
    FOR EACH STATEMENT EXECUTE FUNCTION organization.reject_history_change();
CREATE TRIGGER command_record_append_only BEFORE UPDATE OR DELETE ON organization.command_record
    FOR EACH ROW EXECUTE FUNCTION organization.reject_history_change();
CREATE TRIGGER command_record_no_truncate BEFORE TRUNCATE ON organization.command_record
    FOR EACH STATEMENT EXECUTE FUNCTION organization.reject_history_change();
CREATE TRIGGER hr_request_no_delete BEFORE DELETE ON organization.hr_request
    FOR EACH ROW EXECUTE FUNCTION organization.reject_history_change();
CREATE TRIGGER hr_request_no_truncate BEFORE TRUNCATE ON organization.hr_request
    FOR EACH STATEMENT EXECUTE FUNCTION organization.reject_history_change();

-- +goose Down
DROP TABLE IF EXISTS organization.command_record;
DROP TABLE IF EXISTS organization.hr_request_event;
DROP TABLE IF EXISTS organization.hr_request;
DROP FUNCTION IF EXISTS organization.reject_history_change();
