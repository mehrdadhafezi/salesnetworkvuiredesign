-- Platform migration: append-only audit foundation (Architecture V1 §7, §8.7).
-- Owner: module "audit" (schema audit, NOLOGIN role crm_owner_audit).

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_audit NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL; -- roles are cluster-wide; another database may have created it
END
$$;
-- +goose StatementEnd

CREATE SCHEMA audit;
REVOKE ALL ON SCHEMA audit FROM PUBLIC;
GRANT USAGE ON SCHEMA audit TO crm_owner_audit;

CREATE TABLE audit.audit_event (
    event_id        uuid        PRIMARY KEY,
    occurred_at     timestamptz NOT NULL,
    recorded_at     timestamptz NOT NULL DEFAULT now(),
    actor_domain    text        NULL,
    actor_id        text        NULL,
    subject_domain  text        NOT NULL,
    subject_id      text        NOT NULL,
    action          text        NOT NULL,
    reason          text        NULL,
    correlation_id  text        NULL,
    metadata        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT audit_event_actor_pair CHECK ((actor_domain IS NULL) = (actor_id IS NULL)),
    CONSTRAINT audit_event_action_format CHECK (action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(action) <= 128),
    CONSTRAINT audit_event_reason_length CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 1000),
    CONSTRAINT audit_event_correlation_length CHECK (correlation_id IS NULL OR length(correlation_id) BETWEEN 1 AND 128),
    CONSTRAINT audit_event_metadata_object CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT audit_event_metadata_size CHECK (pg_column_size(metadata) <= 8192)
);

CREATE INDEX audit_event_subject_idx ON audit.audit_event (subject_domain, subject_id, occurred_at);

-- +goose StatementBegin
CREATE FUNCTION audit.reject_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit.audit_event is append-only (% rejected)', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_event_no_update_delete
    BEFORE UPDATE OR DELETE ON audit.audit_event
    FOR EACH ROW EXECUTE FUNCTION audit.reject_mutation();

CREATE TRIGGER audit_event_no_truncate
    BEFORE TRUNCATE ON audit.audit_event
    FOR EACH STATEMENT EXECUTE FUNCTION audit.reject_mutation();

REVOKE ALL ON audit.audit_event FROM PUBLIC;
GRANT SELECT, INSERT ON audit.audit_event TO crm_owner_audit;

-- +goose Down
DROP SCHEMA audit CASCADE;
