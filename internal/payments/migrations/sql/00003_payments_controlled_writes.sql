-- Payments controlled financial writes (Slice 12; Architecture V1 §15).
--
-- Evidence: upload_object (bytes durable in quarantine BEFORE this row is
-- written), receipt (one per bound upload) and the append-only
-- receipt_state_history. A bound upload is never purged; an unbound upload
-- may be purged after expires_at (D-44 safe default: 7 days). The content
-- hash of an upload and of a receipt is immutable.
--
-- Submission: payment_batch, batch_item and submitted_attempt are written in
-- ONE payments transaction (all-or-nothing, §15.2) with the receipts, the
-- idempotency record, the command record and the outbox event. A target
-- attempt is only ever PENDING_REVIEW here: there is no target Finance
-- review (D-41, D-43, OPD-08), so ACCEPTED / REJECTED cannot be stored.
-- Imported legacy attempts stay in payment_attempt (Slice 7) untouched.
--
-- collection_account is the per (invoice, stage) serialisation row
-- (SELECT ... FOR UPDATE) and the optimistic-concurrency version a client
-- sends as If-Match.
--
-- gateway_request is the synthetic-only gateway attempt (no route). D-25 and
-- D-26 are schema guards: a gateway result is never Finance-reviewed and a
-- callback never issues a session.
--
-- No billing, finance, custody, cases, biawin or fulfillment table is
-- referenced or written. No money is converted or rounded (OPD-06).

-- +goose Up
CREATE TABLE payments.upload_object (
    upload_id          uuid        PRIMARY KEY,
    uploader_user_id   uuid        NOT NULL,
    content_sha256     bytea       NOT NULL,
    size_bytes         integer     NOT NULL,
    media_type         text        NOT NULL,
    quarantine_ref     text        NOT NULL UNIQUE,
    state              text        NOT NULL,
    created_at         timestamptz NOT NULL,
    expires_at         timestamptz NOT NULL,
    bound_at           timestamptz NULL,
    purged_at          timestamptz NULL,
    version            bigint      NOT NULL DEFAULT 1,
    CONSTRAINT upload_once_per_uploader UNIQUE (uploader_user_id, content_sha256),
    CONSTRAINT upload_sha_length CHECK (octet_length(content_sha256) = 32),
    CONSTRAINT upload_size CHECK (size_bytes BETWEEN 1 AND 5242880),
    CONSTRAINT upload_media CHECK (media_type IN ('image/jpeg', 'image/png', 'image/webp', 'application/pdf')),
    CONSTRAINT upload_ref CHECK (quarantine_ref ~ '^quarantine/[0-9a-f-]{36}$'),
    CONSTRAINT upload_state CHECK (state IN ('UNBOUND', 'BOUND', 'PURGED')),
    CONSTRAINT upload_bound CHECK ((state = 'BOUND') = (bound_at IS NOT NULL)),
    CONSTRAINT upload_purged CHECK ((state = 'PURGED') = (purged_at IS NOT NULL)),
    CONSTRAINT upload_expiry CHECK (expires_at > created_at),
    CONSTRAINT upload_version CHECK (version > 0)
);
CREATE INDEX upload_object_unbound_idx ON payments.upload_object (expires_at) WHERE state = 'UNBOUND';

CREATE TABLE payments.collection_account (
    invoice_id   uuid        NOT NULL,
    stage_no     integer     NOT NULL,
    version      bigint      NOT NULL,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    PRIMARY KEY (invoice_id, stage_no),
    CONSTRAINT collection_stage CHECK (stage_no > 0),
    CONSTRAINT collection_version CHECK (version >= 0)
);

CREATE TABLE payments.payment_batch (
    batch_id          uuid        PRIMARY KEY,
    invoice_id        uuid        NOT NULL,
    stage_no          integer     NOT NULL,
    stage_id          uuid        NOT NULL,
    invoice_source_system text    NOT NULL,
    invoice_source_spec   text    NOT NULL,
    invoice_source_key    text    NOT NULL,
    idempotency_key   text        NOT NULL,
    request_sha256    bytea       NOT NULL,
    item_count        integer     NOT NULL,
    total_amount      bigint      NOT NULL,
    unit              text        NOT NULL,
    before_version    bigint      NOT NULL,
    after_version     bigint      NOT NULL,
    actor_principal   text        NOT NULL,
    actor_user_id     uuid        NOT NULL,
    correlation_id    text        NOT NULL,
    created_at        timestamptz NOT NULL,
    CONSTRAINT batch_key UNIQUE (idempotency_key, actor_user_id),
    CONSTRAINT batch_invoice_source CHECK (invoice_source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT batch_sha_length CHECK (octet_length(request_sha256) = 32),
    CONSTRAINT batch_items CHECK (item_count BETWEEN 1 AND 10),
    CONSTRAINT batch_total CHECK (total_amount > 0),
    CONSTRAINT batch_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT batch_versions CHECK (after_version = before_version + 1),
    CONSTRAINT batch_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX payment_batch_invoice_idx ON payments.payment_batch (invoice_id, stage_no, created_at);

CREATE TABLE payments.submitted_attempt (
    attempt_id          uuid        PRIMARY KEY,
    batch_id            uuid        NOT NULL REFERENCES payments.payment_batch (batch_id),
    item_key            text        NOT NULL,
    item_no             integer     NOT NULL,
    invoice_id          uuid        NOT NULL,
    stage_no            integer     NOT NULL,
    channel             text        NOT NULL,
    entry_mode          text        NOT NULL,
    amount              bigint      NOT NULL,
    unit                text        NOT NULL,
    state               text        NOT NULL,
    review_provenance   text        NOT NULL,
    evidence_state      text        NOT NULL,
    submitted_by_user_id uuid       NOT NULL,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL,
    version             bigint      NOT NULL,
    CONSTRAINT attempt_item_unique UNIQUE (batch_id, item_key),
    CONSTRAINT attempt_item_no UNIQUE (batch_id, item_no),
    CONSTRAINT submitted_item_key CHECK (item_key ~ '^[A-Za-z0-9_-]{1,64}$'),
    CONSTRAINT submitted_channel CHECK (channel = 'MANUAL_CARD'),
    CONSTRAINT submitted_entry_mode CHECK (entry_mode = 'RECEIPT'),
    CONSTRAINT submitted_amount CHECK (amount > 0),
    CONSTRAINT submitted_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT submitted_state CHECK (state = 'PENDING_REVIEW'),
    CONSTRAINT submitted_provenance CHECK (review_provenance = 'NOT_REVIEWED'),
    CONSTRAINT submitted_evidence CHECK (evidence_state IN ('PENDING_FINALISATION', 'READY', 'NOT_RECOVERABLE')),
    CONSTRAINT submitted_version CHECK (version > 0)
);
CREATE INDEX submitted_attempt_stage_idx ON payments.submitted_attempt (invoice_id, stage_no);
CREATE INDEX submitted_attempt_review_idx ON payments.submitted_attempt (state, created_at, attempt_id);

CREATE TABLE payments.submitted_attempt_event (
    event_id        uuid        PRIMARY KEY,
    attempt_id      uuid        NOT NULL REFERENCES payments.submitted_attempt (attempt_id),
    kind            text        NOT NULL,
    detail          jsonb       NOT NULL,
    actor_principal text        NOT NULL,
    correlation_id  text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT attempt_event_kind CHECK (kind IN ('SUBMITTED', 'EVIDENCE_READY', 'EVIDENCE_NOT_RECOVERABLE')),
    CONSTRAINT attempt_event_detail CHECK (jsonb_typeof(detail) = 'object' AND pg_column_size(detail) <= 8192)
);
CREATE INDEX submitted_attempt_event_idx ON payments.submitted_attempt_event (attempt_id, occurred_at);

CREATE TABLE payments.receipt_evidence (
    receipt_id        uuid        PRIMARY KEY,
    attempt_id        uuid        NOT NULL REFERENCES payments.submitted_attempt (attempt_id),
    batch_id          uuid        NOT NULL REFERENCES payments.payment_batch (batch_id),
    invoice_id        uuid        NOT NULL,
    upload_id         uuid        NOT NULL UNIQUE REFERENCES payments.upload_object (upload_id),
    content_sha256    bytea       NOT NULL,
    size_bytes        integer     NOT NULL,
    media_type        text        NOT NULL,
    quarantine_ref    text        NOT NULL,
    evidence_ref      text        NULL,
    state             text        NOT NULL,
    finalise_attempts integer     NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    version           bigint      NOT NULL,
    CONSTRAINT receipt_sha_length CHECK (octet_length(content_sha256) = 32),
    CONSTRAINT receipt_state CHECK (state IN ('QUARANTINED', 'FINALIZING', 'READY', 'STORAGE_ERROR', 'NOT_RECOVERABLE')),
    CONSTRAINT receipt_ready_ref CHECK ((state = 'READY') = (evidence_ref IS NOT NULL)),
    CONSTRAINT receipt_evidence_ref CHECK (evidence_ref IS NULL OR evidence_ref ~ '^evidence/[0-9a-f]{64}$'),
    CONSTRAINT receipt_attempts CHECK (finalise_attempts >= 0),
    CONSTRAINT receipt_version CHECK (version > 0)
);
CREATE INDEX receipt_evidence_attempt_idx ON payments.receipt_evidence (attempt_id);
CREATE INDEX receipt_evidence_work_idx ON payments.receipt_evidence (state, updated_at) WHERE state IN ('QUARANTINED', 'FINALIZING', 'STORAGE_ERROR');

CREATE TABLE payments.receipt_state_history (
    history_id      uuid        PRIMARY KEY,
    receipt_id      uuid        NOT NULL REFERENCES payments.receipt_evidence (receipt_id),
    from_state      text        NULL,
    to_state        text        NOT NULL,
    reason          text        NOT NULL,
    actor_principal text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT receipt_history_reason CHECK (length(reason) BETWEEN 1 AND 300)
);
CREATE INDEX receipt_state_history_idx ON payments.receipt_state_history (receipt_id, occurred_at);

CREATE TABLE payments.evidence_reconciliation (
    reconciliation_id uuid        PRIMARY KEY,
    receipt_id        uuid        NOT NULL REFERENCES payments.receipt_evidence (receipt_id),
    attempt_id        uuid        NOT NULL,
    kind              text        NOT NULL,
    reason            text        NOT NULL,
    domain_owner      text        NOT NULL,
    next_safe_action  text        NOT NULL,
    evidence          jsonb       NOT NULL,
    opened_at         timestamptz NOT NULL,
    CONSTRAINT evidence_recon_kind CHECK (kind IN ('OPENED', 'RESOLVED')),
    CONSTRAINT evidence_recon_reason CHECK (reason IN ('STORAGE_ERROR', 'NOT_RECOVERABLE', 'RECOVERED')),
    CONSTRAINT evidence_recon_owner CHECK (domain_owner = 'finance'),
    CONSTRAINT evidence_recon_action CHECK (next_safe_action IN ('RETRY_FROM_QUARANTINE', 'DOMAIN_OWNER_REVIEW', 'NO_AUTOMATIC_ACTION')),
    CONSTRAINT evidence_recon_json CHECK (jsonb_typeof(evidence) = 'object' AND pg_column_size(evidence) <= 8192)
);
CREATE INDEX evidence_reconciliation_idx ON payments.evidence_reconciliation (opened_at, reconciliation_id);

CREATE TABLE payments.command_record (
    command_id        uuid        PRIMARY KEY,
    command_type      text        NOT NULL,
    subject_id        uuid        NOT NULL,
    idempotency_key   text        NOT NULL,
    request_sha256    bytea       NOT NULL,
    rollout_mode      text        NOT NULL,
    applied           boolean     NOT NULL,
    intent            jsonb       NOT NULL,
    actor_principal   text        NOT NULL,
    actor_user_id     uuid        NOT NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT command_type_known CHECK (command_type IN ('payments.submit_batch')),
    CONSTRAINT command_mode CHECK (rollout_mode = 'CONTROLLED_WRITEBACK'),
    CONSTRAINT command_sha_length CHECK (octet_length(request_sha256) = 32),
    CONSTRAINT command_intent CHECK (jsonb_typeof(intent) = 'object' AND pg_column_size(intent) <= 16384)
);

CREATE TABLE payments.gateway_request (
    request_id          uuid        PRIMARY KEY,
    idempotency_key     text        NOT NULL,
    invoice_id          uuid        NOT NULL,
    stage_no            integer     NOT NULL,
    amount              bigint      NOT NULL,
    unit                text        NOT NULL,
    provider            text        NOT NULL,
    state               text        NOT NULL,
    external_ref        text        NULL,
    verified_ref_sha256 bytea       NULL,
    approval_provenance text        NOT NULL,
    finance_reviewed    boolean     NOT NULL DEFAULT false,
    session_issued      boolean     NOT NULL DEFAULT false,
    requested_by_user_id uuid       NOT NULL,
    correlation_id      text        NOT NULL,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL,
    version             bigint      NOT NULL,
    CONSTRAINT gateway_key UNIQUE (idempotency_key, requested_by_user_id),
    CONSTRAINT gateway_external UNIQUE (provider, external_ref),
    CONSTRAINT gateway_provider CHECK (provider ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT gateway_amount CHECK (amount > 0),
    CONSTRAINT gateway_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT gateway_state CHECK (state IN ('UNKNOWN', 'REDIRECT_ISSUED', 'VERIFIED', 'FAILED_AT_GATEWAY', 'CANCELLED_AT_GATEWAY',
        'RECONCILIATION_REQUIRED')),
    CONSTRAINT gateway_external_ref CHECK (external_ref IS NULL OR external_ref ~ '^[A-Za-z0-9_.:-]{1,128}$'),
    CONSTRAINT gateway_verified CHECK ((state = 'VERIFIED') = (verified_ref_sha256 IS NOT NULL)),
    CONSTRAINT gateway_provenance CHECK (approval_provenance IN ('NOT_APPROVED', 'GATEWAY_AUTO')
        AND ((approval_provenance = 'GATEWAY_AUTO') = (state = 'VERIFIED'))),
    CONSTRAINT gateway_never_finance_review CHECK (finance_reviewed = false),
    CONSTRAINT gateway_never_session CHECK (session_issued = false),
    CONSTRAINT gateway_version CHECK (version > 0)
);

CREATE TABLE payments.gateway_event (
    event_id        uuid        PRIMARY KEY,
    request_id      uuid        NULL REFERENCES payments.gateway_request (request_id),
    provider        text        NOT NULL,
    kind            text        NOT NULL,
    outcome         text        NOT NULL,
    external_ref    text        NULL,
    detail          text        NULL,
    actor_principal text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    CONSTRAINT gateway_event_kind CHECK (kind IN ('REQUEST', 'CALLBACK', 'VERIFY', 'INQUIRY', 'ORPHAN_CALLBACK', 'STALE_CALLBACK')),
    CONSTRAINT gateway_event_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED', 'UNAUTHORIZED')),
    CONSTRAINT gateway_event_detail CHECK (detail IS NULL OR length(detail) <= 300)
);
CREATE INDEX gateway_event_request_idx ON payments.gateway_event (request_id, occurred_at);

GRANT SELECT, INSERT, UPDATE ON payments.upload_object, payments.collection_account, payments.receipt_evidence, payments.gateway_request
    TO crm_owner_payments;
-- A target attempt is never updated through SQL other than its evidence
-- summary; the owner holds UPDATE only on that column set.
GRANT SELECT, INSERT ON payments.submitted_attempt TO crm_owner_payments;
GRANT UPDATE (evidence_state, updated_at, version) ON payments.submitted_attempt TO crm_owner_payments;
GRANT SELECT, INSERT ON payments.payment_batch, payments.submitted_attempt_event, payments.receipt_state_history,
    payments.evidence_reconciliation, payments.command_record, payments.gateway_event TO crm_owner_payments;

CREATE TRIGGER payment_batch_append_only BEFORE UPDATE OR DELETE ON payments.payment_batch
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER payment_batch_no_truncate BEFORE TRUNCATE ON payments.payment_batch
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER submitted_attempt_event_append_only BEFORE UPDATE OR DELETE ON payments.submitted_attempt_event
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER submitted_attempt_event_no_truncate BEFORE TRUNCATE ON payments.submitted_attempt_event
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER receipt_state_history_append_only BEFORE UPDATE OR DELETE ON payments.receipt_state_history
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER receipt_state_history_no_truncate BEFORE TRUNCATE ON payments.receipt_state_history
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER evidence_reconciliation_append_only BEFORE UPDATE OR DELETE ON payments.evidence_reconciliation
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER evidence_reconciliation_no_truncate BEFORE TRUNCATE ON payments.evidence_reconciliation
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER payments_command_record_append_only BEFORE UPDATE OR DELETE ON payments.command_record
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER payments_command_record_no_truncate BEFORE TRUNCATE ON payments.command_record
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER gateway_event_append_only BEFORE UPDATE OR DELETE ON payments.gateway_event
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER gateway_event_no_truncate BEFORE TRUNCATE ON payments.gateway_event
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
-- Evidence rows are never deleted (bound evidence is retained, D-44).
CREATE TRIGGER receipt_evidence_no_delete BEFORE DELETE ON payments.receipt_evidence
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER receipt_evidence_no_truncate BEFORE TRUNCATE ON payments.receipt_evidence
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER upload_object_no_delete BEFORE DELETE ON payments.upload_object
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER upload_object_no_truncate BEFORE TRUNCATE ON payments.upload_object
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER submitted_attempt_no_delete BEFORE DELETE ON payments.submitted_attempt
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER submitted_attempt_no_truncate BEFORE TRUNCATE ON payments.submitted_attempt
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();

-- +goose Down
DROP TABLE IF EXISTS payments.gateway_event;
DROP TABLE IF EXISTS payments.gateway_request;
DROP TABLE IF EXISTS payments.command_record;
DROP TABLE IF EXISTS payments.evidence_reconciliation;
DROP TABLE IF EXISTS payments.receipt_state_history;
DROP TABLE IF EXISTS payments.receipt_evidence;
DROP TABLE IF EXISTS payments.submitted_attempt_event;
DROP TABLE IF EXISTS payments.submitted_attempt;
DROP TABLE IF EXISTS payments.payment_batch;
DROP TABLE IF EXISTS payments.collection_account;
DROP TABLE IF EXISTS payments.upload_object;
