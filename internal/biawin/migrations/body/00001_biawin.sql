-- Biawin READ / SHADOW model (Architecture V1 §13; Slice 10). Built only by
-- the offline shadow import from normalized legacy evidence (integration);
-- no other schema is read or written. Every table is INSERT-only (grants +
-- 00002 guards): a run writes all its facts in one transaction and is never
-- rewritten, so a list and its detail always come from the same run (same
-- cutoff, same normalization version) and history is never overwritten.
-- No activation, suspension, order or wallet charge exists (D-35, D-38).
-- No PII column: no phone, name, address or contact data.

CREATE TABLE biawin.import_run (
    run_id                uuid        PRIMARY KEY,
    run_seq               bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    record_kind           text        NOT NULL DEFAULT 'SHADOW',
    normalization_version integer     NOT NULL,
    snapshot_id           text        NOT NULL,
    batch_id              uuid        NOT NULL,
    source_cutoff         timestamptz NOT NULL,
    started_at            timestamptz NOT NULL,
    completed_at          timestamptz NOT NULL,
    outcome               text        NOT NULL,
    reasons               jsonb       NOT NULL,
    counts                jsonb       NOT NULL,
    input_sha256          bytea       NOT NULL,
    facts_sha256          bytea       NOT NULL,
    CONSTRAINT import_run_kind CHECK (record_kind = 'SHADOW'),
    CONSTRAINT import_run_version CHECK (normalization_version > 0),
    CONSTRAINT import_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL')),
    CONSTRAINT import_run_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT import_run_counts CHECK (jsonb_typeof(counts) = 'object'),
    CONSTRAINT import_run_shas CHECK (octet_length(input_sha256) = 32 AND octet_length(facts_sha256) = 32),
    CONSTRAINT import_run_order CHECK (started_at <= completed_at),
    -- The same snapshot under the same normalization version is one run.
    CONSTRAINT import_run_replay UNIQUE (normalization_version, input_sha256)
);

CREATE TABLE biawin.run_source (
    run_id     uuid    NOT NULL REFERENCES biawin.import_run (run_id),
    source     text    NOT NULL,
    status     text    NOT NULL,
    rows_read  bigint  NOT NULL,
    PRIMARY KEY (run_id, source),
    CONSTRAINT run_source_status CHECK (status IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE')),
    CONSTRAINT run_source_rows CHECK (rows_read >= 0)
);

-- One subscription (legacy membership) of a run. Identity is a stable,
-- deterministic id from source system + kind + native key (never a phone).
-- Customer / invoice / case links keep their resolution state; ids only
-- when RESOLVED. Product is historical evidence (version never guessed).
CREATE TABLE biawin.subscription (
    run_id                  uuid  NOT NULL REFERENCES biawin.import_run (run_id),
    subscription_id         uuid  NOT NULL,
    source_system           text  NOT NULL,
    source_spec             text  NOT NULL,
    source_key              text  NOT NULL,
    source_ref_raw          text  NULL,
    source_type_raw         text  NULL,
    provider                text  NOT NULL DEFAULT 'BIAWIN_LEGACY_EVIDENCE',
    product_key             text  NULL,
    product_title_snapshot  text  NULL,
    product_version_state   text  NOT NULL DEFAULT 'UNKNOWN',
    customer_link_state     text  NOT NULL,
    customer_link_basis     text  NOT NULL,
    customer_id             uuid  NULL,
    customer_account_key    text  NULL,
    invoice_link_state      text  NOT NULL,
    invoice_id              uuid  NULL,
    invoice_key             text  NULL,
    first_payment_link_state text NOT NULL,
    first_payment_invoice_id uuid NULL,
    first_payment_invoice_key text NULL,
    case_link_state         text  NOT NULL,
    case_link_basis         text  NOT NULL,
    case_id                 uuid  NULL,
    dot_case_key            text  NULL,
    origin_user_key         text  NULL,
    origin_user_id          uuid  NULL,
    state_namespace         text  NOT NULL,
    state_raw               text  NULL,
    state_mapped            text  NOT NULL,
    activation_evidence     text  NOT NULL,
    activated_at_raw        text  NULL,
    created_at_raw          text  NULL,
    updated_at_raw          text  NULL,
    item_count              integer NOT NULL,
    PRIMARY KEY (run_id, subscription_id),
    CONSTRAINT subscription_source UNIQUE (run_id, source_system, source_spec, source_key),
    CONSTRAINT subscription_state CHECK (state_mapped IN ('ACTIVE', 'SUSPENDED', 'UNKNOWN', 'EMPTY')),
    CONSTRAINT subscription_unknown_keeps_raw CHECK (state_mapped <> 'UNKNOWN' OR state_raw IS NOT NULL),
    CONSTRAINT subscription_links CHECK (customer_link_state IN ('RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE')
        AND invoice_link_state IN ('RESOLVED', 'UNKNOWN') AND first_payment_link_state IN ('RESOLVED', 'UNKNOWN')
        AND case_link_state IN ('RESOLVED', 'UNKNOWN')),
    CONSTRAINT subscription_customer CHECK (customer_id IS NULL OR customer_link_state = 'RESOLVED'),
    CONSTRAINT subscription_invoice CHECK ((invoice_id IS NOT NULL) = (invoice_link_state = 'RESOLVED')),
    CONSTRAINT subscription_activation CHECK (activation_evidence IN ('LEGACY_ACTIVATION_RECORDED', 'NOT_EVIDENCED')),
    CONSTRAINT subscription_product_version CHECK (product_version_state IN ('UNKNOWN'))
);

-- One subscription item: one included service instance (1 .. many per
-- subscription; never collapsed by title). No item-level lifecycle state
-- exists in the source: NOT_EVIDENCED (its follow-up is fulfillment's).
CREATE TABLE biawin.subscription_item (
    run_id                  uuid    NOT NULL,
    item_id                 uuid    NOT NULL,
    subscription_id         uuid    NOT NULL,
    source_key              text    NOT NULL,
    service_key             text    NULL,
    service_title_snapshot  text    NULL,
    state_mapped            text    NOT NULL DEFAULT 'NOT_EVIDENCED',
    created_at_raw          text    NULL,
    PRIMARY KEY (run_id, item_id),
    CONSTRAINT subscription_item_parent FOREIGN KEY (run_id, subscription_id) REFERENCES biawin.subscription (run_id, subscription_id),
    CONSTRAINT subscription_item_source UNIQUE (run_id, source_key),
    CONSTRAINT subscription_item_state CHECK (state_mapped = 'NOT_EVIDENCED')
);
CREATE INDEX subscription_item_parent_idx ON biawin.subscription_item (run_id, subscription_id);

-- Invoice links of an item (billing refs only; billing owns invoices).
CREATE TABLE biawin.item_invoice_link (
    run_id             uuid  NOT NULL,
    item_id            uuid  NOT NULL,
    source_key         text  NOT NULL,
    invoice_link_state text  NOT NULL,
    invoice_id         uuid  NULL,
    invoice_key        text  NULL,
    status_raw         text  NULL,
    verified_at_raw    text  NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT item_invoice_link_item FOREIGN KEY (run_id, item_id) REFERENCES biawin.subscription_item (run_id, item_id),
    CONSTRAINT item_invoice_link_state CHECK (invoice_link_state IN ('RESOLVED', 'UNKNOWN') AND (invoice_id IS NOT NULL) = (invoice_link_state = 'RESOLVED'))
);

-- Legacy history (append-only facts of the run; never rewritten).
CREATE TABLE biawin.subscription_event (
    run_id           uuid  NOT NULL,
    source_key       text  NOT NULL,
    subscription_id  uuid  NOT NULL,
    item_id          uuid  NULL,
    event_class      text  NOT NULL,
    event_key_raw    text  NOT NULL,
    actor_key        text  NULL,
    actor_user_id    uuid  NULL,
    occurred_at_raw  text  NULL,
    PRIMARY KEY (run_id, source_key),
    CONSTRAINT subscription_event_parent FOREIGN KEY (run_id, subscription_id) REFERENCES biawin.subscription (run_id, subscription_id),
    CONSTRAINT subscription_event_item FOREIGN KEY (run_id, item_id) REFERENCES biawin.subscription_item (run_id, item_id),
    CONSTRAINT subscription_event_class CHECK (event_class IN ('SUBSCRIPTION_EVENT', 'ITEM_EVENT', 'PROVIDER_EVENT_EVIDENCE'))
);
CREATE INDEX subscription_event_parent_idx ON biawin.subscription_event (run_id, subscription_id);

-- Active subscription parity: the INDEPENDENT legacy active classification
-- of one staged snapshot vs the ACTIVE set of one target run, by stable
-- source key. Computed and recorded only by the offline job.
CREATE TABLE biawin.parity_check (
    check_id             uuid        PRIMARY KEY,
    check_seq            bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    run_id               uuid        NOT NULL REFERENCES biawin.import_run (run_id),
    evidence_snapshot_id text        NOT NULL,
    evidence_cutoff      timestamptz NOT NULL,
    evidence_status      text        NOT NULL,
    checked_at           timestamptz NOT NULL,
    legacy_count         bigint      NOT NULL,
    target_count         bigint      NOT NULL,
    matched              bigint      NOT NULL,
    missing_target       bigint      NOT NULL,
    extra_target         bigint      NOT NULL,
    unknown              bigint      NOT NULL,
    conflict             bigint      NOT NULL,
    not_recoverable      bigint      NOT NULL,
    outcome              text        NOT NULL,
    reasons              jsonb       NOT NULL,
    CONSTRAINT parity_check_outcome CHECK (outcome IN ('MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE')),
    CONSTRAINT parity_check_status CHECK (evidence_status IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE')),
    CONSTRAINT parity_check_counts CHECK (legacy_count >= 0 AND target_count >= 0 AND matched >= 0 AND missing_target >= 0 AND extra_target >= 0
        AND unknown >= 0 AND conflict >= 0 AND not_recoverable >= 0),
    CONSTRAINT parity_check_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT parity_check_match CHECK (outcome <> 'MATCH' OR (missing_target = 0 AND extra_target = 0 AND unknown = 0 AND conflict = 0
        AND not_recoverable = 0 AND evidence_status = 'AVAILABLE'))
);

CREATE TABLE biawin.parity_row (
    check_id      uuid  NOT NULL REFERENCES biawin.parity_check (check_id),
    source_key    text  NOT NULL,
    legacy_class  text  NULL,
    target_state  text  NULL,
    result        text  NOT NULL,
    PRIMARY KEY (check_id, source_key),
    CONSTRAINT parity_row_result CHECK (result IN ('MATCHED', 'MISSING_TARGET', 'EXTRA_TARGET', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE', 'INACTIVE_BOTH'))
);

GRANT SELECT, INSERT ON biawin.import_run, biawin.run_source, biawin.subscription, biawin.subscription_item, biawin.item_invoice_link,
    biawin.subscription_event, biawin.parity_check, biawin.parity_row TO crm_owner_biawin;
