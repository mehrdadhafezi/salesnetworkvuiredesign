-- Custody shadow model (Architecture V1 §5, §7, §10, §25; Slice 6).
-- CustodyEvent is append-only (FPR; E0 §10): one row per legacy log row,
-- never updated or deleted (no grant; a trigger rejects either even for the
-- owner). The legacy action is preserved raw; its kind comes only from an
-- explicit, evidence-backed mapping, otherwise UNKNOWN.
--
-- Assignment is a projection DERIVED from the events, stored next to the
-- legacy current-holder evidence and their parity class. It is refreshed by
-- the import only and never the sole source of truth. Findings report gaps,
-- contradictions and mismatches; nothing is auto-repaired.
--
-- Case references are target case ids resolved through the published cases
-- API; there is no cross-schema foreign key and no phone anywhere.

CREATE TABLE custody.custody_event (
    event_id          uuid        PRIMARY KEY,
    case_id           uuid        NOT NULL,
    source_system     text        NOT NULL,
    source_spec       text        NOT NULL,
    source_key        text        NOT NULL,
    source_ordinal    numeric(20) NOT NULL,
    legacy_action     text        NOT NULL,
    kind              text        NOT NULL,
    context           text        NOT NULL,
    actor_legacy_key  text        NULL,
    actor_user_id     uuid        NULL,
    from_legacy_key   text        NULL,
    from_user_id      uuid        NULL,
    to_legacy_key     text        NULL,
    to_user_id        uuid        NULL,
    from_position_raw text        NULL,
    to_position_raw   text        NULL,
    step_order_raw    text        NULL,
    occurred_raw      text        NULL,
    row_sha256        bytea       NOT NULL,
    first_batch_id    uuid        NOT NULL,
    recorded_at       timestamptz NOT NULL,
    CONSTRAINT custody_event_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT custody_event_kind CHECK (kind IN ('HOLDER_ASSIGNED', 'HOLDER_RELEASED', 'NO_HOLDER_CHANGE', 'UNKNOWN')),
    CONSTRAINT custody_event_context CHECK (context IN ('DISTRIBUTION', 'MIS', 'PAYMENT_WORKFLOW', 'REFERRAL', 'CONVERSION', 'SELLER_UPDATE', 'UNKNOWN')),
    CONSTRAINT custody_event_action_length CHECK (length(legacy_action) BETWEEN 1 AND 100),
    CONSTRAINT custody_event_sha_length CHECK (octet_length(row_sha256) = 32)
);
CREATE INDEX custody_event_case_idx ON custody.custody_event (case_id, source_ordinal);

-- Derived, reconcilable projection: one row per Case with custody evidence.
CREATE TABLE custody.assignment (
    assignment_id          uuid        PRIMARY KEY,
    case_id                uuid        NOT NULL UNIQUE,
    derivation             text        NOT NULL,
    derived_legacy_key     text        NULL,
    derived_user_id        uuid        NULL,
    derived_position_raw   text        NULL,
    legacy_holder_key      text        NULL,
    legacy_holder_user_id  uuid        NULL,
    legacy_position_raw    text        NULL,
    legacy_source_spec     text        NOT NULL,
    parity                 text        NOT NULL,
    event_count            integer     NOT NULL,
    last_event_id          uuid        NULL,
    presence               text        NOT NULL,
    refreshed_batch_id     uuid        NOT NULL,
    refreshed_at           timestamptz NOT NULL,
    CONSTRAINT assignment_derivation CHECK (derivation IN ('DERIVED', 'RELEASED', 'UNKNOWN', 'MULTIPLE', 'NONE')),
    CONSTRAINT assignment_derived_has_holder CHECK (derivation <> 'DERIVED' OR derived_legacy_key IS NOT NULL),
    CONSTRAINT assignment_parity CHECK (parity IN ('MATCH', 'TARGET_UNKNOWN', 'LEGACY_UNKNOWN', 'CONFLICT', 'MULTIPLE_CURRENT', 'HISTORY_GAP', 'NOT_RECOVERABLE')),
    CONSTRAINT assignment_event_count CHECK (event_count >= 0),
    CONSTRAINT assignment_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX assignment_parity_idx ON custody.assignment (parity, case_id);

CREATE TABLE custody.finding (
    finding_id         uuid        PRIMARY KEY,
    finding_key        bytea       NOT NULL UNIQUE,
    case_id            uuid        NOT NULL,
    kind               text        NOT NULL,
    detail             text        NOT NULL,
    event_id           uuid        NULL REFERENCES custody.custody_event (event_id),
    presence           text        NOT NULL,
    first_batch_id     uuid        NOT NULL,
    last_seen_batch_id uuid        NOT NULL,
    recorded_at        timestamptz NOT NULL,
    CONSTRAINT finding_key_length CHECK (octet_length(finding_key) = 32),
    CONSTRAINT finding_kind CHECK (kind IN ('MISSING_PRIOR_OWNER', 'MISSING_TARGET_OWNER', 'HISTORY_GAP', 'MULTIPLE_CURRENT',
        'ORDERING_CONTRADICTION', 'MISSING_EVENT_HISTORY', 'SOURCE_CURRENT_MISMATCH', 'UNKNOWN_ACTION')),
    CONSTRAINT finding_detail_length CHECK (length(detail) BETWEEN 1 AND 300),
    CONSTRAINT finding_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);
CREATE INDEX finding_case_idx ON custody.finding (case_id);

CREATE TABLE custody.import_run (
    run_id       uuid        PRIMARY KEY,
    batch_id     uuid        NOT NULL,
    snapshot_id  text        NOT NULL,
    outcome      text        NOT NULL,
    replay       boolean     NOT NULL,
    counts       jsonb       NOT NULL,
    recorded_at  timestamptz NOT NULL,
    CONSTRAINT import_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED')),
    CONSTRAINT import_run_counts_object CHECK (jsonb_typeof(counts) = 'object')
);

-- Append-only guards (UPDATE / DELETE / TRUNCATE rejected even for the
-- owner) are installed by 00002_custody_append_only_guards.sql.

-- No DELETE grant anywhere; events are INSERT-only.
GRANT SELECT, INSERT ON custody.custody_event, custody.import_run TO crm_owner_custody;
GRANT SELECT, INSERT, UPDATE ON custody.assignment TO crm_owner_custody;
GRANT SELECT, INSERT ON custody.finding TO crm_owner_custody;
GRANT UPDATE (presence, last_seen_batch_id) ON custody.finding TO crm_owner_custody;
