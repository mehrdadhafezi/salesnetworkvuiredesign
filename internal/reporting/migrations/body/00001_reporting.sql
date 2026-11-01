-- Reporting projections (Architecture V1 §20; Slice 8). NOT source of truth.
-- Built only by the offline projection rebuild from owner PUBLISHED read
-- APIs (no cross-schema read, no event exists yet). Every table is
-- INSERT-only: a run writes its facts and its run row in one transaction and
-- is never rewritten, so an older cutoff result never changes silently and
-- projection versions coexist. No PII: no name, phone, address, postcode,
-- national id, URL, token or gateway reference column exists.

-- Versioned metric contracts. A version is immutable: changing formula,
-- grain, cohort, scope or time basis requires a new version.
CREATE TABLE reporting.metric_contract (
    metric_id       text        NOT NULL,
    version         integer     NOT NULL,
    name            text        NOT NULL,
    grain           text        NOT NULL,
    source          text        NOT NULL,
    cohort          text        NOT NULL,
    time_basis      text        NOT NULL,
    scope           text        NOT NULL,
    formula         text        NOT NULL,
    unit            text        NOT NULL,
    freshness_sla_seconds integer NOT NULL,
    coverage        text        NOT NULL,
    status          text        NOT NULL,
    status_reason   text        NULL,
    contract_sha256 bytea       NOT NULL,
    registered_at   timestamptz NOT NULL,
    PRIMARY KEY (metric_id, version),
    CONSTRAINT metric_contract_time_basis CHECK (time_basis IN ('CURRENT_SNAPSHOT', 'EVENT_RANGE', 'HISTORICAL_AS_OF', 'NOT_DEFINED')),
    CONSTRAINT metric_contract_status CHECK (status IN ('FINAL', 'NOT_FINAL')),
    CONSTRAINT metric_contract_version CHECK (version > 0),
    CONSTRAINT metric_contract_sha CHECK (octet_length(contract_sha256) = 32)
);

-- One projection run (rebuild). Inserted with its facts in one transaction.
CREATE TABLE reporting.projection_run (
    run_id              uuid        PRIMARY KEY,
    run_seq             bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    projection_version  integer     NOT NULL,
    source_cutoff       timestamptz NOT NULL,
    started_at          timestamptz NOT NULL,
    completed_at        timestamptz NOT NULL,
    outcome             text        NOT NULL,
    reasons             jsonb       NOT NULL,
    fact_counts         jsonb       NOT NULL,
    facts_sha256        bytea       NOT NULL,
    CONSTRAINT projection_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL')),
    CONSTRAINT projection_run_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT projection_run_counts CHECK (jsonb_typeof(fact_counts) = 'object'),
    CONSTRAINT projection_run_sha CHECK (octet_length(facts_sha256) = 32),
    CONSTRAINT projection_run_order CHECK (source_cutoff <= completed_at AND started_at <= completed_at)
);

-- Per run, which published owner source contributed and how.
CREATE TABLE reporting.projection_source (
    run_id      uuid        NOT NULL REFERENCES reporting.projection_run (run_id),
    source      text        NOT NULL,
    status      text        NOT NULL,
    rows_read   bigint      NULL,
    read_at     timestamptz NULL,
    reason      text        NOT NULL,
    PRIMARY KEY (run_id, source),
    CONSTRAINT projection_source_status CHECK (status IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE'))
);

-- Owner aggregate evidence captured at the run (e.g. customer resolution).
CREATE TABLE reporting.source_aggregate (
    run_id   uuid   NOT NULL REFERENCES reporting.projection_run (run_id),
    source   text   NOT NULL,
    key      text   NOT NULL,
    value    bigint NOT NULL,
    PRIMARY KEY (run_id, source, key)
);

CREATE TABLE reporting.case_fact (
    run_id              uuid        NOT NULL REFERENCES reporting.projection_run (run_id),
    case_id             uuid        NOT NULL,
    workflow_type       text        NOT NULL,
    state_namespace     text        NOT NULL,
    state_raw           text        NULL,
    state_mapping       text        NOT NULL,
    customer_link_state text        NOT NULL,
    presence            text        NOT NULL,
    source_spec         text        NOT NULL,
    source_key          text        NOT NULL,
    alias_count         bigint      NOT NULL,
    first_seen_at       timestamptz NOT NULL,
    original_owner_resolution  text NOT NULL,
    original_owner_key         text NULL,
    original_seller_resolution text NOT NULL,
    original_seller_key        text NULL,
    credit_owner_resolution    text NOT NULL,
    PRIMARY KEY (run_id, case_id)
);

CREATE TABLE reporting.custody_fact (
    run_id        uuid   NOT NULL REFERENCES reporting.projection_run (run_id),
    case_id       uuid   NOT NULL,
    derivation    text   NOT NULL,
    parity        text   NOT NULL,
    derived_key   text   NULL,
    legacy_key    text   NULL,
    event_count   integer NOT NULL,
    finding_count bigint NOT NULL,
    presence      text   NOT NULL,
    PRIMARY KEY (run_id, case_id)
);

-- Every legacy invoice state field stays a separate column (TAD #20).
CREATE TABLE reporting.invoice_fact (
    run_id              uuid        NOT NULL REFERENCES reporting.projection_run (run_id),
    invoice_id          uuid        NOT NULL,
    source_spec         text        NOT NULL,
    source_key          text        NOT NULL,
    code_raw            text        NULL,
    presence            text        NOT NULL,
    customer_link_state text        NOT NULL,
    case_link_state     text        NOT NULL,
    seller_key          text        NULL,
    main_raw     text NULL, main_mapping     text NOT NULL,
    invoice_raw  text NULL, invoice_mapping  text NOT NULL,
    payment_raw  text NULL, payment_mapping  text NOT NULL,
    workflow_raw text NULL, workflow_mapping text NOT NULL,
    return_raw   text NULL, return_mapping   text NOT NULL,
    unknown_state_count bigint      NOT NULL,
    item_count          bigint      NOT NULL,
    stage_count         bigint      NOT NULL,
    items_complete      boolean     NOT NULL,
    stages_complete     boolean     NOT NULL,
    first_seen_at       timestamptz NOT NULL,
    PRIMARY KEY (run_id, invoice_id)
);

-- Per invoice and unit total evidence; no canonical total (TAD #19).
CREATE TABLE reporting.invoice_unit_fact (
    run_id           uuid    NOT NULL REFERENCES reporting.projection_run (run_id),
    invoice_id       uuid    NOT NULL,
    unit             text    NOT NULL,
    declared_final   bigint  NULL,
    declared_payment bigint  NULL,
    item_sum         bigint  NULL, item_state    text NOT NULL,
    stage_sum        bigint  NULL, stage_state   text NOT NULL,
    payment_sum      bigint  NULL, payment_state text NOT NULL,
    total_class      text    NOT NULL,
    PRIMARY KEY (run_id, invoice_id, unit),
    CONSTRAINT invoice_unit_fact_unit CHECK (unit IN ('IRR', 'IRT', 'UNATTRIBUTED'))
);

CREATE TABLE reporting.payment_fact (
    run_id             uuid        NOT NULL REFERENCES reporting.projection_run (run_id),
    attempt_id         uuid        NOT NULL,
    invoice_id         uuid        NULL,
    invoice_link_state text        NOT NULL,
    channel            text        NOT NULL,
    state_raw          text        NULL,
    state_mapping      text        NOT NULL,
    state_target       text        NULL,
    approval_provenance text       NOT NULL,
    amount_raw         text        NULL,
    amount             bigint      NULL,
    unit               text        NULL,
    amount_repr        text        NOT NULL,
    gateway_state      text        NULL,
    presence           text        NOT NULL,
    first_seen_at      timestamptz NOT NULL,
    PRIMARY KEY (run_id, attempt_id),
    CONSTRAINT payment_fact_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT'))
);

CREATE INDEX case_fact_first_seen_idx ON reporting.case_fact (run_id, first_seen_at);
CREATE INDEX invoice_fact_first_seen_idx ON reporting.invoice_fact (run_id, first_seen_at);
CREATE INDEX payment_fact_first_seen_idx ON reporting.payment_fact (run_id, first_seen_at);

-- Append-only guards are installed by 00002_reporting_append_only_guards.sql.

-- INSERT and SELECT only: no UPDATE, DELETE or TRUNCATE anywhere.
GRANT SELECT, INSERT ON reporting.metric_contract, reporting.projection_run, reporting.projection_source, reporting.source_aggregate,
    reporting.case_fact, reporting.custody_fact, reporting.invoice_fact, reporting.invoice_unit_fact, reporting.payment_fact
    TO crm_owner_reporting;
