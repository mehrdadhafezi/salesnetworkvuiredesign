-- Finance shadow ledger guards (Slice 9): the double-entry invariant and the
-- append-only rule.
--
-- Append-only: runs, sources, accounts,
-- entries, lines, findings, legacy balances and reconciliations are never
-- updated, deleted or truncated -- not even by the owner role, which holds
-- no such grant anyway. A correction is a new compensating entry; a new
-- source state is a new run.

-- +goose Up
-- Double-entry invariant, enforced at COMMIT (deferred): every entry has at
-- least one debit and one credit line and DEBITS == CREDITS (summed in
-- numeric: no overflow). Lines carry the entry's single unit (foreign key).
-- +goose StatementBegin
CREATE FUNCTION finance.assert_entry_balanced() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    d numeric;
    c numeric;
    nd integer;
    nc integer;
BEGIN
    SELECT coalesce(sum(amount) FILTER (WHERE side = 'DEBIT'), 0), coalesce(sum(amount) FILTER (WHERE side = 'CREDIT'), 0),
           count(*) FILTER (WHERE side = 'DEBIT'), count(*) FILTER (WHERE side = 'CREDIT')
      INTO d, c, nd, nc
      FROM finance.ledger_line WHERE run_id = NEW.run_id AND entry_id = NEW.entry_id;
    IF nd = 0 OR nc = 0 OR d <> c THEN
        RAISE EXCEPTION 'finance.ledger_entry %: unbalanced (debits % / credits %)', NEW.entry_id, d, c
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER ledger_entry_balanced AFTER INSERT ON finance.ledger_entry
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION finance.assert_entry_balanced();
CREATE CONSTRAINT TRIGGER ledger_line_balanced AFTER INSERT ON finance.ledger_line
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION finance.assert_entry_balanced();

-- +goose StatementBegin
CREATE FUNCTION finance.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'finance.%: append-only shadow fact; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER shadow_run_append_only BEFORE UPDATE OR DELETE ON finance.shadow_run
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER shadow_run_no_truncate BEFORE TRUNCATE ON finance.shadow_run
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER run_source_append_only BEFORE UPDATE OR DELETE ON finance.run_source
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER run_source_no_truncate BEFORE TRUNCATE ON finance.run_source
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_account_append_only BEFORE UPDATE OR DELETE ON finance.ledger_account
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_account_no_truncate BEFORE TRUNCATE ON finance.ledger_account
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_entry_append_only BEFORE UPDATE OR DELETE ON finance.ledger_entry
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_entry_no_truncate BEFORE TRUNCATE ON finance.ledger_entry
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_line_append_only BEFORE UPDATE OR DELETE ON finance.ledger_line
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER ledger_line_no_truncate BEFORE TRUNCATE ON finance.ledger_line
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER finding_append_only BEFORE UPDATE OR DELETE ON finance.finding
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER finding_no_truncate BEFORE TRUNCATE ON finance.finding
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER legacy_balance_append_only BEFORE UPDATE OR DELETE ON finance.legacy_balance
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER legacy_balance_no_truncate BEFORE TRUNCATE ON finance.legacy_balance
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER reconciliation_append_only BEFORE UPDATE OR DELETE ON finance.reconciliation
    FOR EACH ROW EXECUTE FUNCTION finance.reject_change();
CREATE TRIGGER reconciliation_no_truncate BEFORE TRUNCATE ON finance.reconciliation
    FOR EACH STATEMENT EXECUTE FUNCTION finance.reject_change();

-- +goose Down
DROP TRIGGER ledger_line_balanced ON finance.ledger_line;
DROP TRIGGER ledger_entry_balanced ON finance.ledger_entry;
DROP FUNCTION finance.assert_entry_balanced();
DROP TRIGGER shadow_run_append_only ON finance.shadow_run;
DROP TRIGGER shadow_run_no_truncate ON finance.shadow_run;
DROP TRIGGER run_source_append_only ON finance.run_source;
DROP TRIGGER run_source_no_truncate ON finance.run_source;
DROP TRIGGER ledger_account_append_only ON finance.ledger_account;
DROP TRIGGER ledger_account_no_truncate ON finance.ledger_account;
DROP TRIGGER ledger_entry_append_only ON finance.ledger_entry;
DROP TRIGGER ledger_entry_no_truncate ON finance.ledger_entry;
DROP TRIGGER ledger_line_append_only ON finance.ledger_line;
DROP TRIGGER ledger_line_no_truncate ON finance.ledger_line;
DROP TRIGGER finding_append_only ON finance.finding;
DROP TRIGGER finding_no_truncate ON finance.finding;
DROP TRIGGER legacy_balance_append_only ON finance.legacy_balance;
DROP TRIGGER legacy_balance_no_truncate ON finance.legacy_balance;
DROP TRIGGER reconciliation_append_only ON finance.reconciliation;
DROP TRIGGER reconciliation_no_truncate ON finance.reconciliation;
DROP FUNCTION finance.reject_change();
