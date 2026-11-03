-- Payments upload lifecycle (Slice 12 review fixes).
--
-- Expiry becomes a durable, retryable two-step lifecycle:
--   UNBOUND --(claim: expired, never bound)--> EXPIRING --(bytes deleted)--> PURGED
-- An EXPIRING upload can never be bound (binding requires UNBOUND and an
-- unexpired upload under the same row lock). A failed or interrupted
-- deletion leaves the upload EXPIRING, so the next run retries it;
-- deletion of a missing object is idempotent. purge_attempts and
-- last_purge_error record failures.
--
-- upload_event is the append-only history of every upload: UPLOADED,
-- RENEWED (an expired, unbound upload re-sent while its bytes are intact),
-- RESTAGED (an expired upload re-sent whose bytes were missing: fresh bytes,
-- same identity), REVIVED (a PURGED / EXPIRING upload re-sent: fresh bytes,
-- same identity), EXPIRING, PURGE_FAILED and PURGED. Each event names the
-- quarantine object the upload referenced at that moment.

-- +goose Up
ALTER TABLE payments.upload_object DROP CONSTRAINT upload_state;
ALTER TABLE payments.upload_object ADD CONSTRAINT upload_state CHECK (state IN ('UNBOUND', 'BOUND', 'EXPIRING', 'PURGED'));
ALTER TABLE payments.upload_object ADD COLUMN expiring_at timestamptz NULL;
ALTER TABLE payments.upload_object ADD COLUMN purge_attempts integer NOT NULL DEFAULT 0;
ALTER TABLE payments.upload_object ADD COLUMN last_purge_error text NULL;
ALTER TABLE payments.upload_object ADD CONSTRAINT upload_expiring CHECK (state <> 'EXPIRING' OR expiring_at IS NOT NULL);
ALTER TABLE payments.upload_object ADD CONSTRAINT upload_purge_attempts CHECK (purge_attempts >= 0);
ALTER TABLE payments.upload_object ADD CONSTRAINT upload_purge_error CHECK (last_purge_error IS NULL OR length(last_purge_error) <= 300);
DROP INDEX payments.upload_object_unbound_idx;
CREATE INDEX upload_object_expiry_idx ON payments.upload_object (expires_at) WHERE state IN ('UNBOUND', 'EXPIRING');

CREATE TABLE payments.upload_event (
    event_id        uuid        PRIMARY KEY,
    upload_id       uuid        NOT NULL REFERENCES payments.upload_object (upload_id),
    kind            text        NOT NULL,
    quarantine_ref  text        NOT NULL,
    detail          text        NULL,
    actor_principal text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT upload_event_kind CHECK (kind IN ('UPLOADED', 'RENEWED', 'RESTAGED', 'REVIVED', 'EXPIRING', 'PURGE_FAILED', 'PURGED')),
    CONSTRAINT upload_event_ref CHECK (quarantine_ref ~ '^quarantine/[0-9a-f-]{36}$'),
    CONSTRAINT upload_event_detail CHECK (detail IS NULL OR length(detail) <= 300)
);
CREATE INDEX upload_event_upload_idx ON payments.upload_event (upload_id, occurred_at);

GRANT SELECT, INSERT ON payments.upload_event TO crm_owner_payments;

CREATE TRIGGER upload_event_append_only BEFORE UPDATE OR DELETE ON payments.upload_event
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER upload_event_no_truncate BEFORE TRUNCATE ON payments.upload_event
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();

-- +goose Down
DROP TABLE IF EXISTS payments.upload_event;
DROP INDEX IF EXISTS payments.upload_object_expiry_idx;
CREATE INDEX upload_object_unbound_idx ON payments.upload_object (expires_at) WHERE state = 'UNBOUND';
ALTER TABLE payments.upload_object DROP CONSTRAINT IF EXISTS upload_purge_error;
ALTER TABLE payments.upload_object DROP CONSTRAINT IF EXISTS upload_purge_attempts;
ALTER TABLE payments.upload_object DROP CONSTRAINT IF EXISTS upload_expiring;
ALTER TABLE payments.upload_object DROP COLUMN IF EXISTS last_purge_error;
ALTER TABLE payments.upload_object DROP COLUMN IF EXISTS purge_attempts;
ALTER TABLE payments.upload_object DROP COLUMN IF EXISTS expiring_at;
ALTER TABLE payments.upload_object DROP CONSTRAINT upload_state;
ALTER TABLE payments.upload_object ADD CONSTRAINT upload_state CHECK (state IN ('UNBOUND', 'BOUND', 'PURGED'));
