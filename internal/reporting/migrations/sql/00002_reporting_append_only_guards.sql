-- Reporting append-only guards (Slice 8). Contracts, runs and facts are never
-- updated, deleted or truncated -- not even by the owner role, which holds
-- no such grant anyway. A new cutoff is a new run; a new meaning is a new
-- contract version.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION reporting.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'reporting.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER metric_contract_append_only BEFORE UPDATE OR DELETE ON reporting.metric_contract
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER metric_contract_no_truncate BEFORE TRUNCATE ON reporting.metric_contract
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER projection_run_append_only BEFORE UPDATE OR DELETE ON reporting.projection_run
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER projection_run_no_truncate BEFORE TRUNCATE ON reporting.projection_run
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER projection_source_append_only BEFORE UPDATE OR DELETE ON reporting.projection_source
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER projection_source_no_truncate BEFORE TRUNCATE ON reporting.projection_source
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER source_aggregate_append_only BEFORE UPDATE OR DELETE ON reporting.source_aggregate
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER source_aggregate_no_truncate BEFORE TRUNCATE ON reporting.source_aggregate
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER case_fact_append_only BEFORE UPDATE OR DELETE ON reporting.case_fact
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER case_fact_no_truncate BEFORE TRUNCATE ON reporting.case_fact
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER custody_fact_append_only BEFORE UPDATE OR DELETE ON reporting.custody_fact
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER custody_fact_no_truncate BEFORE TRUNCATE ON reporting.custody_fact
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER invoice_fact_append_only BEFORE UPDATE OR DELETE ON reporting.invoice_fact
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER invoice_fact_no_truncate BEFORE TRUNCATE ON reporting.invoice_fact
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER invoice_unit_fact_append_only BEFORE UPDATE OR DELETE ON reporting.invoice_unit_fact
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER invoice_unit_fact_no_truncate BEFORE TRUNCATE ON reporting.invoice_unit_fact
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER payment_fact_append_only BEFORE UPDATE OR DELETE ON reporting.payment_fact
    FOR EACH ROW EXECUTE FUNCTION reporting.reject_change();
CREATE TRIGGER payment_fact_no_truncate BEFORE TRUNCATE ON reporting.payment_fact
    FOR EACH STATEMENT EXECUTE FUNCTION reporting.reject_change();

-- +goose Down
DROP TRIGGER IF EXISTS payment_fact_no_truncate ON reporting.payment_fact;
DROP TRIGGER IF EXISTS payment_fact_append_only ON reporting.payment_fact;
DROP TRIGGER IF EXISTS invoice_unit_fact_no_truncate ON reporting.invoice_unit_fact;
DROP TRIGGER IF EXISTS invoice_unit_fact_append_only ON reporting.invoice_unit_fact;
DROP TRIGGER IF EXISTS invoice_fact_no_truncate ON reporting.invoice_fact;
DROP TRIGGER IF EXISTS invoice_fact_append_only ON reporting.invoice_fact;
DROP TRIGGER IF EXISTS custody_fact_no_truncate ON reporting.custody_fact;
DROP TRIGGER IF EXISTS custody_fact_append_only ON reporting.custody_fact;
DROP TRIGGER IF EXISTS case_fact_no_truncate ON reporting.case_fact;
DROP TRIGGER IF EXISTS case_fact_append_only ON reporting.case_fact;
DROP TRIGGER IF EXISTS source_aggregate_no_truncate ON reporting.source_aggregate;
DROP TRIGGER IF EXISTS source_aggregate_append_only ON reporting.source_aggregate;
DROP TRIGGER IF EXISTS projection_source_no_truncate ON reporting.projection_source;
DROP TRIGGER IF EXISTS projection_source_append_only ON reporting.projection_source;
DROP TRIGGER IF EXISTS projection_run_no_truncate ON reporting.projection_run;
DROP TRIGGER IF EXISTS projection_run_append_only ON reporting.projection_run;
DROP TRIGGER IF EXISTS metric_contract_no_truncate ON reporting.metric_contract;
DROP TRIGGER IF EXISTS metric_contract_append_only ON reporting.metric_contract;
DROP FUNCTION IF EXISTS reporting.reject_change();
