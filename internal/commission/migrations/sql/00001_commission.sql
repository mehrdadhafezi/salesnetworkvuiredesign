-- Per-module infrastructure for module "commission" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_commission NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA commission;
REVOKE ALL ON SCHEMA commission FROM PUBLIC;
GRANT USAGE ON SCHEMA commission TO crm_owner_commission;

CREATE TABLE commission.outbox_event (
    event_id         uuid        PRIMARY KEY,
    event_type       text        NOT NULL,
    event_version    integer     NOT NULL,
    subject_domain   text        NULL,
    subject_id       text        NULL,
    payload          jsonb       NOT NULL,
    occurred_at      timestamptz NOT NULL,
    recorded_at      timestamptz NOT NULL,
    correlation_id   text        NULL,
    dispatch_state   text        NOT NULL DEFAULT 'pending',
    attempts         integer     NOT NULL DEFAULT 0,
    last_attempt_at  timestamptz NULL,
    last_error       text        NULL,
    dispatched_at    timestamptz NULL,
    CONSTRAINT outbox_event_type_format CHECK (event_type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(event_type) <= 128),
    CONSTRAINT outbox_event_version_positive CHECK (event_version > 0),
    CONSTRAINT outbox_event_subject_pair CHECK ((subject_domain IS NULL) = (subject_id IS NULL)),
    CONSTRAINT outbox_event_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT outbox_event_correlation_length CHECK (correlation_id IS NULL OR length(correlation_id) BETWEEN 1 AND 128),
    CONSTRAINT outbox_event_state CHECK (dispatch_state IN ('pending', 'dispatched', 'failed')),
    CONSTRAINT outbox_event_attempts CHECK (attempts >= 0),
    CONSTRAINT outbox_event_error_length CHECK (last_error IS NULL OR length(last_error) <= 1000),
    CONSTRAINT outbox_event_dispatched CHECK ((dispatch_state = 'dispatched') = (dispatched_at IS NOT NULL))
);
CREATE INDEX outbox_event_undispatched_idx ON commission.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE commission.inbox_message (
    consumer         text        NOT NULL,
    message_id       uuid        NOT NULL,
    state            text        NOT NULL,
    first_seen_at    timestamptz NOT NULL,
    last_attempt_at  timestamptz NOT NULL,
    processed_at     timestamptz NULL,
    attempts         integer     NOT NULL,
    last_error       text        NULL,
    PRIMARY KEY (consumer, message_id),
    CONSTRAINT inbox_message_consumer_format CHECK (consumer ~ '^[a-z][a-z0-9_.]{1,127}$'),
    CONSTRAINT inbox_message_state CHECK (state IN ('processing', 'processed', 'failed')),
    CONSTRAINT inbox_message_processed CHECK ((state = 'processed') = (processed_at IS NOT NULL)),
    CONSTRAINT inbox_message_attempts CHECK (attempts >= 1),
    CONSTRAINT inbox_message_error_length CHECK (last_error IS NULL OR length(last_error) <= 1000)
);

CREATE TABLE commission.idempotency_record (
    scope            text        NOT NULL,
    idem_key         text        NOT NULL,
    request_hash     bytea       NOT NULL,
    state            text        NOT NULL,
    result           jsonb       NULL,
    created_at       timestamptz NOT NULL,
    completed_at     timestamptz NULL,
    PRIMARY KEY (scope, idem_key),
    CONSTRAINT idempotency_scope_format CHECK (scope ~ '^[a-z][a-z0-9_.]{1,63}$'),
    CONSTRAINT idempotency_key_format CHECK (idem_key ~ '^[A-Za-z0-9_-]{8,128}$'),
    CONSTRAINT idempotency_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT idempotency_state CHECK (state IN ('in_progress', 'completed')),
    CONSTRAINT idempotency_completed CHECK ((state = 'completed') = (completed_at IS NOT NULL)),
    CONSTRAINT idempotency_result_size CHECK (result IS NULL OR pg_column_size(result) <= 65536)
);

REVOKE ALL ON ALL TABLES IN SCHEMA commission FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON commission.outbox_event, commission.inbox_message, commission.idempotency_record TO crm_owner_commission;

-- Module-owned tables follow. Grants on them go to crm_owner_commission only.
-- Commission controlled SHADOW model (Architecture V1 §19; Slice 13). Built
-- only by the offline shadow job (cmd/crm-commission-shadow) from normalized,
-- STAGED legacy evidence; no other schema is read or written. Every table is
-- INSERT-only (grants + 00002 guards): a run writes all its facts in one
-- transaction and is never rewritten; an approved historical snapshot is
-- therefore immutable and history is never overwritten.
--
-- ZERO financial effect: there is no approval, posting, crediting,
-- settlement or reversal command and no table that represents a CRM-side
-- approval or posting. Amounts are exact integers in hundredths of an
-- explicit unit (never floating point); a missing amount is NULL, never 0.
-- No PII column: no phone, name, address or contact data.

-- One shadow run: one staged snapshot under one normalization version and
-- one rounding mode. A CRM run is only ever CALCULATED (approval / posting
-- are decision-blocked: OPD-09, D-41).
CREATE TABLE commission.shadow_run (
    run_id                uuid        PRIMARY KEY,
    run_seq               bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    record_kind           text        NOT NULL DEFAULT 'SHADOW',
    state                 text        NOT NULL DEFAULT 'CALCULATED',
    normalization_version integer     NOT NULL,
    rounding              text        NOT NULL,
    snapshot_id           text        NOT NULL,
    batch_id              text        NOT NULL,
    source_cutoff         timestamptz NOT NULL,
    started_at            timestamptz NOT NULL,
    completed_at          timestamptz NOT NULL,
    outcome               text        NOT NULL,
    reasons               jsonb       NOT NULL,
    counts                jsonb       NOT NULL,
    input_sha256          bytea       NOT NULL,
    facts_sha256          bytea       NOT NULL,
    CONSTRAINT shadow_run_kind CHECK (record_kind = 'SHADOW'),
    CONSTRAINT shadow_run_state CHECK (state = 'CALCULATED'),
    CONSTRAINT shadow_run_version CHECK (normalization_version > 0),
    CONSTRAINT shadow_run_rounding CHECK (rounding IN ('LEGACY_PHP_ROUND_HALF_AWAY_2DP', 'NONE')),
    CONSTRAINT shadow_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL')),
    CONSTRAINT shadow_run_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT shadow_run_counts CHECK (jsonb_typeof(counts) = 'object'),
    CONSTRAINT shadow_run_shas CHECK (octet_length(input_sha256) = 32 AND octet_length(facts_sha256) = 32),
    CONSTRAINT shadow_run_order CHECK (started_at <= completed_at),
    -- The same snapshot under the same version and rounding is one run.
    CONSTRAINT shadow_run_replay UNIQUE (normalization_version, rounding, input_sha256)
);

CREATE TABLE commission.run_source (
    run_id    uuid   NOT NULL REFERENCES commission.shadow_run (run_id),
    source    text   NOT NULL,
    status    text   NOT NULL,
    rows_read bigint NOT NULL,
    PRIMARY KEY (run_id, source),
    CONSTRAINT run_source_status CHECK (status IN ('AVAILABLE', 'PARTIAL', 'MISSING')),
    CONSTRAINT run_source_rows CHECK (rows_read >= 0)
);

-- Policy versions OBSERVED from legacy evidence. A version is a distinct
-- content (rate / fixed / interval / function) of one policy key; the first
-- run that observed it is recorded. Nothing is approved (approval_state).
CREATE TABLE commission.policy_version (
    policy_key        text    NOT NULL,
    version           integer NOT NULL,
    content_sha256    bytea   NOT NULL,
    purpose           text    NOT NULL,
    share_facet       text    NOT NULL,
    eligibility_ref   text    NOT NULL,
    function_id       text    NOT NULL,
    base_kind         text    NOT NULL,
    rate_e4           bigint  NULL,
    fixed_hundredths  bigint  NULL,
    unit              text    NOT NULL,
    effective_from    date    NULL,
    effective_to      date    NULL,
    attribution_facet text    NOT NULL,
    scope             text    NOT NULL,
    provenance        text    NOT NULL,
    evidence          text    NOT NULL,
    approval_state    text    NOT NULL DEFAULT 'NOT_APPROVED',
    source_ref        text    NOT NULL,
    first_run_id      uuid    NOT NULL REFERENCES commission.shadow_run (run_id),
    PRIMARY KEY (policy_key, version),
    CONSTRAINT policy_version_content UNIQUE (policy_key, content_sha256),
    CONSTRAINT policy_version_positive CHECK (version > 0),
    CONSTRAINT policy_version_sha CHECK (octet_length(content_sha256) = 32),
    CONSTRAINT policy_version_function CHECK (function_id IN ('purpose_matrix_stage_percent_v1', 'rule_invoice_percent_fixed_v1')),
    CONSTRAINT policy_version_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT policy_version_rate CHECK (rate_e4 IS NULL OR rate_e4 BETWEEN 0 AND 1000000),
    CONSTRAINT policy_version_interval CHECK (effective_from IS NULL OR effective_to IS NULL OR effective_from <= effective_to),
    CONSTRAINT policy_version_evidence CHECK (evidence IN ('OBSERVED_UNVERSIONED', 'EVENT_SNAPSHOT', 'INVALID')),
    CONSTRAINT policy_version_not_approved CHECK (approval_state = 'NOT_APPROVED'),
    CONSTRAINT policy_version_facet CHECK (attribution_facet IN ('SOURCE_OWNER', 'ORIGINAL_SELLER', 'CREDIT_OWNER', 'CURRENT_CUSTODY',
        'CURRENT_ORG_ASSIGNMENT', 'EVENT_ACTOR'))
);

-- The global entitlement identity: invoice + stage + purpose + share facet +
-- generation context. The recipient is NOT part of the key (first observed
-- recipient snapshot kept as an attribute only), so a recipient or
-- hierarchy change can never mint a second entitlement.
CREATE TABLE commission.entitlement_identity (
    entitlement_key       text  PRIMARY KEY,
    kind                  text  NOT NULL,
    invoice_key           text  NOT NULL,
    stage                 integer NOT NULL,
    purpose               text  NOT NULL,
    share_facet           text  NOT NULL,
    generation_context    text  NOT NULL,
    first_run_id          uuid  NOT NULL REFERENCES commission.shadow_run (run_id),
    first_recipient_key   text  NULL,
    CONSTRAINT entitlement_identity_kind CHECK (kind IN ('INVOICE_STAGE', 'INVOICE')),
    CONSTRAINT entitlement_identity_stage CHECK (stage >= 0)
);

-- One historical legacy commission wallet transaction (credit or reversal
-- debit) of a run, with the writer's event-time snapshot. Rewritten-in-place
-- legacy types are preserved raw; the reversal link names the original.
CREATE TABLE commission.historical_entitlement (
    run_id               uuid    NOT NULL REFERENCES commission.shadow_run (run_id),
    source_key           text    NOT NULL,
    engine               text    NOT NULL,
    engine_basis         text    NOT NULL,
    type_raw             text    NOT NULL,
    direction            text    NOT NULL,
    status_raw           text    NULL,
    rewritten_in_place   boolean NOT NULL,
    holder_key           text    NULL,
    holder_user_id       uuid    NULL,
    invoice_key          text    NULL,
    invoice_id           uuid    NULL,
    amount_raw           text    NULL,
    amount_hundredths    bigint  NULL,
    unit                 text    NOT NULL,
    stage_raw            text    NULL,
    purpose_raw          text    NULL,
    share_facet_raw      text    NULL,
    rate_raw             text    NULL,
    base_raw             text    NULL,
    recipient_snapshot_key text  NULL,
    snapshot_version_raw text    NULL,
    reverses_source_key  text    NULL,
    dry_run_key          text    NULL,
    dry_run_item_key     text    NULL,
    entitlement_key      text    NULL,
    occurred_at_raw      text    NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT historical_entitlement_engine CHECK (engine IN ('PURPOSE_MATRIX', 'RULES_AUTO', 'DRY_RUN_POSTING', 'PROJECT_CARD', 'ENGINE_ROUTING_UNKNOWN')),
    CONSTRAINT historical_entitlement_direction CHECK (direction IN ('CREDIT', 'DEBIT', 'UNKNOWN')),
    CONSTRAINT historical_entitlement_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT historical_entitlement_not_self CHECK (reverses_source_key IS NULL OR reverses_source_key <> source_key)
);
CREATE INDEX historical_entitlement_holder ON commission.historical_entitlement (holder_user_id, run_id);

-- Legacy E3 runs with their approval provenance (raw kept; a rejected then
-- approved run keeps both the rejection and the approval).
CREATE TABLE commission.historical_run (
    run_id                 uuid    NOT NULL REFERENCES commission.shadow_run (run_id),
    source_key             text    NOT NULL,
    run_code               text    NULL,
    window_from_raw        text    NULL,
    window_to_raw          text    NULL,
    status_raw             text    NULL,
    approval_raw           text    NULL,
    state                  text    NOT NULL,
    created_by_key         text    NULL,
    approved_by_key        text    NULL,
    approved_at_raw        text    NULL,
    rejected_by_key        text    NULL,
    rejected_at_raw        text    NULL,
    locked_at_raw          text    NULL,
    rejected_then_approved boolean NOT NULL,
    created_at_raw         text    NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT historical_run_state CHECK (state IN ('GENERATED', 'APPROVED', 'REJECTED', 'UNKNOWN'))
);

CREATE TABLE commission.historical_run_item (
    run_id             uuid    NOT NULL REFERENCES commission.shadow_run (run_id),
    source_key         text    NOT NULL,
    run_key            text    NULL,
    invoice_key        text    NULL,
    invoice_id         uuid    NULL,
    seller_key         text    NULL,
    seller_user_id     uuid    NULL,
    position_raw       text    NULL,
    level_raw          text    NULL,
    invoice_amount_raw text    NULL,
    matched_rule_key   text    NULL,
    calculated_raw     text    NULL,
    item_status_raw    text    NULL,
    posting_status_raw text    NULL,
    posting_state      text    NOT NULL,
    wallet_tx_key      text    NULL,
    posted_at_raw      text    NULL,
    compensation_key   text    NULL,
    calculation_date_raw text  NULL,
    non_invoice        boolean NOT NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT historical_run_item_posting CHECK (posting_state IN ('GENERATED', 'POSTED', 'POSTING_FAILED', 'UNKNOWN'))
);

CREATE TABLE commission.historical_posting_log (
    run_id         uuid NOT NULL REFERENCES commission.shadow_run (run_id),
    source_key     text NOT NULL,
    run_key        text NULL,
    item_key       text NULL,
    wallet_tx_key  text NULL,
    actor_key      text NULL,
    action_raw     text NULL,
    status_raw     text NULL,
    created_at_raw text NULL,
    PRIMARY KEY (run_id, source_key)
);

-- One stored shadow calculation (the pure engine's output) with its
-- immutable input / attribution snapshot refs and its parity.
CREATE TABLE commission.calculation (
    run_id              uuid    NOT NULL REFERENCES commission.shadow_run (run_id),
    calc_id             uuid    NOT NULL,
    source_key          text    NOT NULL,
    entitlement_key     text    NULL,
    engine              text    NOT NULL,
    function_id         text    NOT NULL,
    policy_key          text    NULL,
    policy_version      integer NULL,
    event_ref           text    NOT NULL,
    input_sha256        text    NOT NULL,
    attribution         jsonb   NOT NULL,
    share_facet         text    NULL,
    recipient_state     text    NOT NULL,
    recipient_key       text    NULL,
    recipient_user_id   uuid    NULL,
    base_hundredths     bigint  NULL,
    rate_e4             bigint  NULL,
    amount_hundredths   bigint  NULL,
    legacy_amount_hundredths bigint NULL,
    unit                text    NOT NULL,
    eligibility         text    NOT NULL,
    coverage            text    NOT NULL,
    blocks              jsonb   NOT NULL,
    reasons             jsonb   NOT NULL,
    parity_class        text    NOT NULL,
    parity_fields       jsonb   NOT NULL,
    PRIMARY KEY (run_id, calc_id),
    CONSTRAINT calculation_source UNIQUE (run_id, source_key),
    -- One calculation per entitlement per run: a duplicate is a finding.
    CONSTRAINT calculation_entitlement UNIQUE (run_id, entitlement_key),
    CONSTRAINT calculation_policy FOREIGN KEY (policy_key, policy_version) REFERENCES commission.policy_version (policy_key, version),
    CONSTRAINT calculation_entitlement_fk FOREIGN KEY (entitlement_key) REFERENCES commission.entitlement_identity (entitlement_key),
    CONSTRAINT calculation_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT calculation_eligibility CHECK (eligibility IN ('ELIGIBLE', 'INELIGIBLE', 'BLOCKED')),
    CONSTRAINT calculation_coverage CHECK (coverage IN ('FULL', 'PARTIAL', 'NONE')),
    CONSTRAINT calculation_blocked_no_amount CHECK (eligibility <> 'BLOCKED' OR amount_hundredths IS NULL),
    CONSTRAINT calculation_recipient CHECK (recipient_state IN ('SNAPSHOT', 'UNKNOWN', 'CONFLICT')
        AND (recipient_state = 'SNAPSHOT' OR recipient_key IS NULL)),
    CONSTRAINT calculation_parity CHECK (parity_class IN ('MATCH', 'AMOUNT_MISMATCH', 'RECIPIENT_MISMATCH', 'POLICY_MISMATCH',
        'ATTRIBUTION_UNKNOWN', 'DUPLICATE_RISK', 'ENGINE_ROUTING_UNKNOWN', 'SOURCE_INCOMPLETE', 'NOT_COMPARABLE')),
    CONSTRAINT calculation_arrays CHECK (jsonb_typeof(blocks) = 'array' AND jsonb_typeof(reasons) = 'array'
        AND jsonb_typeof(parity_fields) = 'array' AND jsonb_typeof(attribution) = 'object')
);
CREATE INDEX calculation_recipient_user ON commission.calculation (recipient_user_id, run_id);

-- The parity report of a run (honest coverage; totals per unit as exact text).
CREATE TABLE commission.parity_summary (
    run_id         uuid   PRIMARY KEY REFERENCES commission.shadow_run (run_id),
    cohort         text   NOT NULL,
    window_from_raw text  NULL,
    window_to_raw  text   NULL,
    legacy_rows    bigint NOT NULL,
    compared_rows  bigint NOT NULL,
    coverage       text   NOT NULL,
    classes        jsonb  NOT NULL,
    engines        jsonb  NOT NULL,
    legacy_totals  jsonb  NOT NULL,
    target_totals  jsonb  NOT NULL,
    legacy_missing bigint NOT NULL,
    target_missing bigint NOT NULL,
    reasons        jsonb  NOT NULL,
    CONSTRAINT parity_summary_coverage CHECK (coverage IN ('FULL', 'PARTIAL', 'NONE')),
    CONSTRAINT parity_summary_rows CHECK (compared_rows >= 0 AND compared_rows <= legacy_rows)
);

-- Reconciliation findings (duplicate risk, rewritten history, broken
-- reversal links, engine routing, posting linkage). Never auto-resolved.
CREATE TABLE commission.reconciliation_finding (
    run_id          uuid NOT NULL REFERENCES commission.shadow_run (run_id),
    finding_id      uuid NOT NULL,
    kind            text NOT NULL,
    severity        text NOT NULL,
    source_key      text NOT NULL,
    entitlement_key text NULL,
    detail          text NOT NULL,
    PRIMARY KEY (run_id, finding_id),
    CONSTRAINT reconciliation_finding_kind CHECK (kind IN ('DUPLICATE_RISK', 'REWRITTEN_IN_PLACE', 'REVERSAL_LINK_MISSING', 'ENGINE_ROUTING_UNKNOWN',
        'POSTING_LINK_MISMATCH', 'RECIPIENT_MISMATCH', 'REJECTED_THEN_APPROVED', 'NON_INVOICE_BLOCKED', 'COMPONENT_ALLOCATION_BLOCKED')),
    CONSTRAINT reconciliation_finding_severity CHECK (severity IN ('INFO', 'WARNING', 'BLOCKING'))
);

GRANT SELECT, INSERT ON commission.shadow_run, commission.run_source, commission.policy_version, commission.entitlement_identity,
    commission.historical_entitlement, commission.historical_run, commission.historical_run_item, commission.historical_posting_log,
    commission.calculation, commission.parity_summary, commission.reconciliation_finding TO crm_owner_commission;

-- +goose Down
DROP SCHEMA commission CASCADE;
