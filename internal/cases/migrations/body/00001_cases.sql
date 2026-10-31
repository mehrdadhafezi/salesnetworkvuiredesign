-- Cases shadow model (Architecture V1 §5, §7, §10, §25; Slice 6).
-- Materialised offline by the cases import command only; never written by a
-- read path. One Case is minted per staged legacy source row (a legacy lead
-- or a distribution item). Cases are never merged: there is no "merged into"
-- column and no DELETE grant. Links between representations (CaseAlias)
-- record explicit native references only; continuation versus
-- related-but-distinct stays unresolved (OPD-01), so an alias is never
-- RESOLVED in Slice 6.
--
-- Phone, names and seller usernames are never keys and are not stored.
-- Customer linkage uses only an explicit customers reference; without one it
-- is UNKNOWN. State history and attribution snapshots are append-only: no
-- UPDATE or DELETE grant, and a trigger rejects either even for the owner.
-- The table is named work_case because CASE is an SQL keyword.

CREATE TABLE cases.work_case (
    case_id              uuid        PRIMARY KEY,
    workflow_type        text        NOT NULL,
    state_namespace      text        NOT NULL,
    state_raw            text        NULL,
    state_mapping        text        NOT NULL,
    customer_id          uuid        NULL,
    customer_link_state  text        NOT NULL,
    customer_link_reason text        NOT NULL,
    presence             text        NOT NULL,
    imported_at          timestamptz NOT NULL,
    refreshed_at         timestamptz NOT NULL,
    version              bigint      NOT NULL,
    CONSTRAINT work_case_workflow_type CHECK (workflow_type IN ('LEGACY_LEAD', 'DISTRIBUTION_ITEM')),
    CONSTRAINT work_case_state_mapping CHECK (state_mapping IN ('KNOWN', 'UNKNOWN')),
    CONSTRAINT work_case_customer_link_state CHECK (customer_link_state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT work_case_resolved_needs_ref CHECK (customer_link_state <> 'RESOLVED' OR customer_id IS NOT NULL),
    CONSTRAINT work_case_reason_length CHECK (length(customer_link_reason) BETWEEN 1 AND 300),
    CONSTRAINT work_case_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST')),
    CONSTRAINT work_case_version CHECK (version > 0)
);
CREATE INDEX work_case_workflow_idx ON cases.work_case (workflow_type, case_id);

-- Lineage: exactly one source row per Case in Slice 6. No source-less Case.
CREATE TABLE cases.case_source (
    case_source_id     uuid        PRIMARY KEY,
    case_id            uuid        NOT NULL UNIQUE REFERENCES cases.work_case (case_id),
    source_system      text        NOT NULL,
    source_spec        text        NOT NULL,
    source_key         text        NOT NULL,
    channel_raw        text        NULL,
    origin_ref_raw     text        NULL,
    first_batch_id     uuid        NOT NULL,
    last_batch_id      uuid        NOT NULL,
    row_sha256         bytea       NOT NULL,
    presence           text        NOT NULL,
    first_seen_at      timestamptz NOT NULL,
    last_seen_at       timestamptz NOT NULL,
    CONSTRAINT case_source_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT case_source_key_format CHECK (source_key ~ '^[1-9][0-9]{0,19}$'),
    CONSTRAINT case_source_sha_length CHECK (octet_length(row_sha256) = 32),
    CONSTRAINT case_source_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);

-- Legacy lead facts (the "Legacy Leads Only" grain). Raw values only.
CREATE TABLE cases.lead (
    lead_id               uuid        PRIMARY KEY,
    case_id               uuid        NOT NULL UNIQUE REFERENCES cases.work_case (case_id),
    status_raw            text        NULL,
    lead_status_raw       text        NULL,
    destination_panel_raw text        NULL,
    imported_at_raw       text        NULL,
    assigned_at_raw       text        NULL,
    updated_at_raw        text        NULL,
    presence              text        NOT NULL,
    CONSTRAINT lead_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);

-- Explicit native links between representations. Never a merge.
CREATE TABLE cases.case_alias (
    alias_id            uuid        PRIMARY KEY,
    case_id             uuid        NOT NULL REFERENCES cases.work_case (case_id),
    related_case_id     uuid        NULL REFERENCES cases.work_case (case_id),
    related_source_spec text        NOT NULL,
    related_source_key  text        NOT NULL,
    relation            text        NOT NULL,
    resolution          text        NOT NULL,
    basis               text        NOT NULL,
    first_batch_id      uuid        NOT NULL,
    recorded_at         timestamptz NOT NULL,
    presence            text        NOT NULL,
    CONSTRAINT case_alias_unique UNIQUE (case_id, relation, related_source_spec, related_source_key),
    CONSTRAINT case_alias_relation CHECK (relation IN ('CONVERSION_LINK')),
    CONSTRAINT case_alias_resolution CHECK (resolution IN ('RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT case_alias_basis_length CHECK (length(basis) BETWEEN 1 AND 300),
    CONSTRAINT case_alias_not_self CHECK (related_case_id IS NULL OR related_case_id <> case_id),
    CONSTRAINT case_alias_presence CHECK (presence IN ('PRESENT', 'ABSENT_IN_LATEST'))
);

-- Append-only state history (one row per legacy history row; never updated).
CREATE TABLE cases.case_state_history (
    history_id      uuid        PRIMARY KEY,
    case_id         uuid        NOT NULL REFERENCES cases.work_case (case_id),
    source_system   text        NOT NULL,
    source_spec     text        NOT NULL,
    source_key      text        NOT NULL,
    source_ordinal  numeric(20) NOT NULL,
    old_state_raw   text        NULL,
    new_state_raw   text        NULL,
    actor_legacy_key text       NULL,
    actor_user_id   uuid        NULL,
    occurred_raw    text        NULL,
    row_sha256      bytea       NOT NULL,
    first_batch_id  uuid        NOT NULL,
    recorded_at     timestamptz NOT NULL,
    CONSTRAINT case_state_history_unique UNIQUE (source_system, source_spec, source_key),
    CONSTRAINT case_state_history_sha_length CHECK (octet_length(row_sha256) = 32)
);
CREATE INDEX case_state_history_case_idx ON cases.case_state_history (case_id, source_ordinal);

-- Immutable attribution snapshots (FPR INV-029). A later, different evidence
-- produces a NEW snapshot; an existing snapshot is never updated, and the
-- current organisation hierarchy is never consulted.
CREATE TABLE cases.attribution_snapshot (
    snapshot_id         uuid        PRIMARY KEY,
    snapshot_key        bytea       NOT NULL UNIQUE,
    case_id             uuid        NOT NULL REFERENCES cases.work_case (case_id),
    facet               text        NOT NULL,
    resolution          text        NOT NULL,
    subject_legacy_key  text        NULL,
    subject_user_id     uuid        NULL,
    position_raw        text        NULL,
    basis               text        NOT NULL,
    source_time_raw     text        NULL,
    evidence_spec       text        NULL,
    evidence_key        text        NULL,
    captured_batch_id   uuid        NOT NULL,
    captured_at         timestamptz NOT NULL,
    CONSTRAINT attribution_snapshot_key_length CHECK (octet_length(snapshot_key) = 32),
    CONSTRAINT attribution_snapshot_facet CHECK (facet IN ('ORIGINAL_OWNER', 'ORIGINAL_SELLER', 'CREDIT_OWNER')),
    CONSTRAINT attribution_snapshot_resolution CHECK (resolution IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT attribution_snapshot_resolved_needs_subject CHECK (resolution <> 'RESOLVED' OR subject_legacy_key IS NOT NULL),
    CONSTRAINT attribution_snapshot_basis_length CHECK (length(basis) BETWEEN 1 AND 300)
);
CREATE INDEX attribution_snapshot_case_idx ON cases.attribution_snapshot (case_id, facet, captured_at);

CREATE TABLE cases.import_run (
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
-- owner) are installed by 00002_cases_append_only_guards.sql.

-- No DELETE grant anywhere: absence is a presence marker, never a delete.
GRANT SELECT, INSERT, UPDATE ON cases.work_case, cases.case_source, cases.lead TO crm_owner_cases;
GRANT SELECT, INSERT ON cases.case_state_history, cases.attribution_snapshot, cases.import_run TO crm_owner_cases;
GRANT SELECT, INSERT ON cases.case_alias TO crm_owner_cases;
GRANT UPDATE (related_case_id, resolution, basis, presence) ON cases.case_alias TO crm_owner_cases;
