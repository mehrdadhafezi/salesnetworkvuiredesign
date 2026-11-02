-- Per-module infrastructure for module "finance" (Architecture V1 §5, §8.8):
-- the module's own schema, NOLOGIN owner role and its own outbox, inbox and
-- idempotency tables. Rendered by migrations.RenderModuleInfra; never edited
-- by hand in a rendered migration.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_finance NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA finance;
REVOKE ALL ON SCHEMA finance FROM PUBLIC;
GRANT USAGE ON SCHEMA finance TO crm_owner_finance;

CREATE TABLE finance.outbox_event (
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
CREATE INDEX outbox_event_undispatched_idx ON finance.outbox_event (recorded_at, event_id)
    WHERE dispatch_state <> 'dispatched';

CREATE TABLE finance.inbox_message (
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

CREATE TABLE finance.idempotency_record (
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

REVOKE ALL ON ALL TABLES IN SCHEMA finance FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON finance.outbox_event, finance.inbox_message, finance.idempotency_record TO crm_owner_finance;

-- Module-owned tables follow. Grants on them go to crm_owner_finance only.
-- Finance SHADOW ledger (Architecture V1 §16; Slice 9). NOT the financial
-- system of record: nothing here is settled, bank-confirmed, Finance-approved
-- or a posted production ledger. Built only by the offline shadow rebuild
-- from billing / payments PUBLISHED facts and normalized legacy wallet
-- evidence; no other schema is read or written. Every table is INSERT-only
-- (grants + 00002 guards): a run writes all its facts in one transaction and
-- is never rewritten, so runs (and derivation versions) coexist and a prior
-- reconciliation never changes. Balances are derived from ledger lines at
-- read time; no balance is stored as mutable state. No PII column exists.

-- One shadow derivation run.
CREATE TABLE finance.shadow_run (
    run_id              uuid        PRIMARY KEY,
    run_seq             bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    record_kind         text        NOT NULL DEFAULT 'SHADOW',
    derivation_version  integer     NOT NULL,
    bases               text[]      NOT NULL,
    source_cutoff       timestamptz NOT NULL,
    started_at          timestamptz NOT NULL,
    completed_at        timestamptz NOT NULL,
    outcome             text        NOT NULL,
    reasons             jsonb       NOT NULL,
    fact_counts         jsonb       NOT NULL,
    input_sha256        bytea       NOT NULL,
    facts_sha256        bytea       NOT NULL,
    CONSTRAINT shadow_run_kind CHECK (record_kind = 'SHADOW'),
    CONSTRAINT shadow_run_version CHECK (derivation_version > 0),
    CONSTRAINT shadow_run_bases CHECK (bases = ARRAY['ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW']::text[]),
    CONSTRAINT shadow_run_outcome CHECK (outcome IN ('SUCCESS', 'PARTIAL')),
    CONSTRAINT shadow_run_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT shadow_run_counts CHECK (jsonb_typeof(fact_counts) = 'object'),
    CONSTRAINT shadow_run_input_sha CHECK (octet_length(input_sha256) = 32),
    CONSTRAINT shadow_run_facts_sha CHECK (octet_length(facts_sha256) = 32),
    CONSTRAINT shadow_run_order CHECK (started_at <= completed_at),
    -- Replay of the same inputs under the same derivation version is the
    -- same run: it can never be derived twice.
    CONSTRAINT shadow_run_replay UNIQUE (derivation_version, input_sha256)
);

-- Source manifest of a run: which source, at which snapshot / cutoff, with
-- which coverage.
CREATE TABLE finance.run_source (
    run_id        uuid        NOT NULL REFERENCES finance.shadow_run (run_id),
    source        text        NOT NULL,
    status        text        NOT NULL,
    rows_read     bigint      NULL,
    snapshot_id   text        NULL,
    source_cutoff timestamptz NULL,
    read_at       timestamptz NOT NULL,
    reason        text        NOT NULL,
    PRIMARY KEY (run_id, source),
    CONSTRAINT run_source_name CHECK (source IN ('billing', 'payments', 'legacy_wallets', 'legacy_wallet_transactions')),
    CONSTRAINT run_source_status CHECK (status IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE')),
    CONSTRAINT run_source_rows CHECK (rows_read IS NULL OR rows_read >= 0)
);

-- Ledger accounts of a run. Identity is a category plus a stable target /
-- native reference; never a phone, name or email. An unresolved customer
-- link stays explicit (customer_id only when RESOLVED).
CREATE TABLE finance.ledger_account (
    run_id          uuid  NOT NULL REFERENCES finance.shadow_run (run_id),
    account_id      uuid  NOT NULL,
    account_key     text  NOT NULL,
    category        text  NOT NULL,
    normal_side     text  NOT NULL,
    subject_domain  text  NOT NULL,
    subject_ref     text  NOT NULL,
    customer_link   text  NULL,
    customer_id     uuid  NULL,
    holder_key      text  NULL,
    holder_user_id  uuid  NULL,
    wallet_type_raw text  NULL,
    provenance      text  NULL,
    route           text  NULL,
    PRIMARY KEY (run_id, account_id),
    CONSTRAINT ledger_account_key UNIQUE (run_id, account_key),
    CONSTRAINT ledger_account_category CHECK (category IN ('customer_receivable', 'revenue_by_product_kind', 'cash_bank_clearing',
        'commission_payable', 'legacy_wallet_unclassified', 'legacy_wallet_contra')),
    CONSTRAINT ledger_account_side CHECK (normal_side IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ledger_account_customer CHECK (customer_id IS NULL OR customer_link = 'RESOLVED'),
    CONSTRAINT ledger_account_key_format CHECK (account_key ~ '^[a-z_]+\|[a-z_]+:[A-Za-z0-9_.:-]+$' AND length(account_key) <= 200)
);

-- Ledger entries. business_key is deterministic and UNIQUE per run: replay
-- cannot duplicate a ledger effect. A correction is a new entry pointing to
-- an EARLIER entry of the same run (the foreign key makes a cycle
-- impossible: the reversed entry must already exist); an entry is reversed
-- at most once and never by itself. The unit is the entry's only unit.
CREATE TABLE finance.ledger_entry (
    run_id                uuid        NOT NULL REFERENCES finance.shadow_run (run_id),
    entry_id              uuid        NOT NULL,
    entry_seq             integer     NOT NULL,
    record_kind           text        NOT NULL DEFAULT 'SHADOW',
    business_key          text        NOT NULL,
    basis                 text        NOT NULL,
    unit                  text        NOT NULL,
    effect_kind           text        NOT NULL,
    provenance            text        NOT NULL,
    finance_review        text        NOT NULL DEFAULT 'NOT_FINANCE_REVIEWED',
    source_refs           jsonb       NOT NULL,
    effective_at          timestamptz NULL,
    posted_at             timestamptz NULL,
    source_time_raw       text        NULL,
    time_evidence         text        NOT NULL,
    reverses_entry_id     uuid        NULL,
    reconciliation_status text        NOT NULL,
    derivation_version    integer     NOT NULL,
    PRIMARY KEY (run_id, entry_id),
    CONSTRAINT ledger_entry_unit_key UNIQUE (run_id, entry_id, unit),
    CONSTRAINT ledger_entry_business_key UNIQUE (run_id, business_key),
    CONSTRAINT ledger_entry_seq UNIQUE (run_id, entry_seq),
    CONSTRAINT ledger_entry_reversed_once UNIQUE (run_id, reverses_entry_id),
    CONSTRAINT ledger_entry_reverses FOREIGN KEY (run_id, reverses_entry_id) REFERENCES finance.ledger_entry (run_id, entry_id),
    CONSTRAINT ledger_entry_not_self CHECK (reverses_entry_id IS NULL OR reverses_entry_id <> entry_id),
    CONSTRAINT ledger_entry_kind CHECK (record_kind = 'SHADOW'),
    CONSTRAINT ledger_entry_key_format CHECK (business_key ~ '^[a-z_]+(/[A-Za-z0-9_.:-]+){4,}/v[1-9][0-9]*$' AND length(business_key) <= 300),
    CONSTRAINT ledger_entry_basis CHECK (basis IN ('ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW')),
    CONSTRAINT ledger_entry_unit CHECK (unit IN ('IRR', 'IRT')),
    CONSTRAINT ledger_entry_review CHECK (finance_review = 'NOT_FINANCE_REVIEWED'),
    CONSTRAINT ledger_entry_refs CHECK (jsonb_typeof(source_refs) = 'array' AND jsonb_array_length(source_refs) > 0),
    CONSTRAINT ledger_entry_time CHECK (time_evidence IN ('BUSINESS_TIME_ABSENT', 'SOURCE_TIME_ZONE_UNASSERTED', 'SOURCE_TIME_UTC')),
    CONSTRAINT ledger_entry_status CHECK (reconciliation_status IN ('CONSISTENT', 'CONFLICT', 'UNRESOLVED_CORRECTION')),
    CONSTRAINT ledger_entry_version CHECK (derivation_version > 0),
    CONSTRAINT ledger_entry_seq_positive CHECK (entry_seq > 0)
);

-- Ledger lines. Amount is a positive int64 in the entry's unit; the side
-- carries the sign. No float, no conversion, no rounding.
CREATE TABLE finance.ledger_line (
    run_id      uuid    NOT NULL,
    entry_id    uuid    NOT NULL,
    line_no     integer NOT NULL,
    account_id  uuid    NOT NULL,
    side        text    NOT NULL,
    amount      bigint  NOT NULL,
    unit        text    NOT NULL,
    source_ref  text    NULL,
    PRIMARY KEY (run_id, entry_id, line_no),
    CONSTRAINT ledger_line_entry FOREIGN KEY (run_id, entry_id, unit) REFERENCES finance.ledger_entry (run_id, entry_id, unit),
    CONSTRAINT ledger_line_account FOREIGN KEY (run_id, account_id) REFERENCES finance.ledger_account (run_id, account_id),
    CONSTRAINT ledger_line_side CHECK (side IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ledger_line_amount CHECK (amount > 0),
    CONSTRAINT ledger_line_no CHECK (line_no > 0)
);
CREATE INDEX ledger_line_account_idx ON finance.ledger_line (run_id, account_id);

-- Explained derivation gaps: evidence that produced NO entry (never a
-- fabricated posting), or a notice about it.
CREATE TABLE finance.finding (
    run_id          uuid    NOT NULL REFERENCES finance.shadow_run (run_id),
    finding_no      integer NOT NULL,
    subject_domain  text    NOT NULL,
    subject_ref     text    NOT NULL,
    basis           text    NULL,
    reason          text    NOT NULL,
    state           text    NOT NULL,
    detail          text    NOT NULL,
    PRIMARY KEY (run_id, finding_no),
    CONSTRAINT finding_state CHECK (state IN ('UNKNOWN', 'INCOMPLETE', 'CONFLICT', 'NOT_RECOVERABLE', 'NOTICE')),
    CONSTRAINT finding_basis CHECK (basis IS NULL OR basis IN ('ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW'))
);

-- Legacy wallet balances captured at the run's source snapshot: evidence to
-- reconcile against, never canonical truth.
CREATE TABLE finance.legacy_balance (
    run_id             uuid    NOT NULL REFERENCES finance.shadow_run (run_id),
    wallet_key         text    NOT NULL,
    holder_key         text    NOT NULL,
    holder_user_id     uuid    NULL,
    wallet_type_raw    text    NOT NULL,
    category           text    NOT NULL,
    unit               text    NULL,
    balance_raw        text    NULL,
    balance            bigint  NULL,
    balance_repr       text    NOT NULL,
    total_credit       bigint  NULL,
    total_credit_repr  text    NOT NULL,
    total_debit        bigint  NULL,
    total_debit_repr   text    NOT NULL,
    snapshot_id        text    NOT NULL,
    PRIMARY KEY (run_id, wallet_key),
    CONSTRAINT legacy_balance_unit CHECK (unit IS NULL OR unit IN ('IRR', 'IRT')),
    CONSTRAINT legacy_balance_category CHECK (category IN ('COMMISSION_PAYABLE', 'UNCLASSIFIED')),
    CONSTRAINT legacy_balance_repr CHECK (balance_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')
        AND total_credit_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')
        AND total_debit_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT')),
    CONSTRAINT legacy_balance_value CHECK ((balance IS NOT NULL) = (balance_repr = 'REPRESENTABLE'))
);

-- Reconciliation: one legacy figure vs one shadow amount, per subject,
-- basis and unit, tied to one run and its source snapshot. MATCH / MISMATCH
-- always carry both amounts and the exact difference (shadow - legacy).
CREATE TABLE finance.reconciliation (
    run_id          uuid    NOT NULL REFERENCES finance.shadow_run (run_id),
    recon_no        integer NOT NULL,
    subject_domain  text    NOT NULL,
    subject_ref     text    NOT NULL,
    account_id      uuid    NULL,
    basis           text    NOT NULL,
    unit            text    NOT NULL,
    legacy_figure   text    NOT NULL,
    legacy_raw      text    NULL,
    legacy_amount   bigint  NULL,
    legacy_repr     text    NOT NULL,
    shadow_amount   bigint  NULL,
    difference      bigint  NULL,
    state           text    NOT NULL,
    reasons         jsonb   NOT NULL,
    coverage        text    NOT NULL,
    source_snapshot text    NULL,
    PRIMARY KEY (run_id, recon_no),
    CONSTRAINT reconciliation_subject UNIQUE (run_id, subject_domain, subject_ref, basis, legacy_figure, unit),
    CONSTRAINT reconciliation_account FOREIGN KEY (run_id, account_id) REFERENCES finance.ledger_account (run_id, account_id),
    CONSTRAINT reconciliation_domain CHECK (subject_domain IN ('legacy_wallet', 'invoice')),
    CONSTRAINT reconciliation_basis CHECK (basis IN ('ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW')),
    CONSTRAINT reconciliation_unit CHECK (unit IN ('IRR', 'IRT', 'UNKNOWN')),
    CONSTRAINT reconciliation_figure CHECK (legacy_figure IN ('WALLET_BALANCE', 'INVOICE_REMAINING', 'INVOICE_PAID')),
    CONSTRAINT reconciliation_legacy_repr CHECK (legacy_repr IN ('REPRESENTABLE', 'UNREPRESENTABLE', 'ABSENT', 'OTHER_UNIT')),
    CONSTRAINT reconciliation_state CHECK (state IN ('MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE', 'CONFLICT', 'NOT_RECOVERABLE')),
    CONSTRAINT reconciliation_reasons CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT reconciliation_coverage CHECK (coverage IN ('AVAILABLE', 'PARTIAL', 'MISSING', 'UNKNOWN', 'NOT_RECOVERABLE', 'STALE')),
    CONSTRAINT reconciliation_compared CHECK (state NOT IN ('MATCH', 'MISMATCH') OR
        (legacy_amount IS NOT NULL AND shadow_amount IS NOT NULL AND difference IS NOT NULL AND unit <> 'UNKNOWN')),
    CONSTRAINT reconciliation_match CHECK (state <> 'MATCH' OR difference = 0),
    CONSTRAINT reconciliation_mismatch CHECK (state <> 'MISMATCH' OR difference <> 0)
);

GRANT SELECT, INSERT ON finance.shadow_run, finance.run_source, finance.ledger_account, finance.ledger_entry, finance.ledger_line,
    finance.finding, finance.legacy_balance, finance.reconciliation TO crm_owner_finance;

-- +goose Down
DROP SCHEMA finance CASCADE;
