-- Commission shadow append-only guards (Slice 13). Runs, sources, observed
-- policy versions, entitlement identities, historical evidence, calculations,
-- parity and findings are never updated, deleted or truncated -- not even by
-- the owner role, which holds no such grant anyway. An approved historical
-- snapshot therefore stays immutable; a new snapshot is a new run.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION commission.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'commission.%: append-only shadow fact; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER shadow_run_append_only BEFORE UPDATE OR DELETE ON commission.shadow_run
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER shadow_run_no_truncate BEFORE TRUNCATE ON commission.shadow_run
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER run_source_append_only BEFORE UPDATE OR DELETE ON commission.run_source
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER run_source_no_truncate BEFORE TRUNCATE ON commission.run_source
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER policy_version_append_only BEFORE UPDATE OR DELETE ON commission.policy_version
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER policy_version_no_truncate BEFORE TRUNCATE ON commission.policy_version
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER entitlement_identity_append_only BEFORE UPDATE OR DELETE ON commission.entitlement_identity
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER entitlement_identity_no_truncate BEFORE TRUNCATE ON commission.entitlement_identity
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_entitlement_append_only BEFORE UPDATE OR DELETE ON commission.historical_entitlement
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_entitlement_no_truncate BEFORE TRUNCATE ON commission.historical_entitlement
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_run_append_only BEFORE UPDATE OR DELETE ON commission.historical_run
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_run_no_truncate BEFORE TRUNCATE ON commission.historical_run
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_run_item_append_only BEFORE UPDATE OR DELETE ON commission.historical_run_item
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_run_item_no_truncate BEFORE TRUNCATE ON commission.historical_run_item
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_posting_log_append_only BEFORE UPDATE OR DELETE ON commission.historical_posting_log
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER historical_posting_log_no_truncate BEFORE TRUNCATE ON commission.historical_posting_log
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER calculation_append_only BEFORE UPDATE OR DELETE ON commission.calculation
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER calculation_no_truncate BEFORE TRUNCATE ON commission.calculation
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER parity_summary_append_only BEFORE UPDATE OR DELETE ON commission.parity_summary
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER parity_summary_no_truncate BEFORE TRUNCATE ON commission.parity_summary
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER reconciliation_finding_append_only BEFORE UPDATE OR DELETE ON commission.reconciliation_finding
    FOR EACH ROW EXECUTE FUNCTION commission.reject_change();
CREATE TRIGGER reconciliation_finding_no_truncate BEFORE TRUNCATE ON commission.reconciliation_finding
    FOR EACH STATEMENT EXECUTE FUNCTION commission.reject_change();

-- +goose Down
DROP TRIGGER shadow_run_append_only ON commission.shadow_run;
DROP TRIGGER shadow_run_no_truncate ON commission.shadow_run;
DROP TRIGGER run_source_append_only ON commission.run_source;
DROP TRIGGER run_source_no_truncate ON commission.run_source;
DROP TRIGGER policy_version_append_only ON commission.policy_version;
DROP TRIGGER policy_version_no_truncate ON commission.policy_version;
DROP TRIGGER entitlement_identity_append_only ON commission.entitlement_identity;
DROP TRIGGER entitlement_identity_no_truncate ON commission.entitlement_identity;
DROP TRIGGER historical_entitlement_append_only ON commission.historical_entitlement;
DROP TRIGGER historical_entitlement_no_truncate ON commission.historical_entitlement;
DROP TRIGGER historical_run_append_only ON commission.historical_run;
DROP TRIGGER historical_run_no_truncate ON commission.historical_run;
DROP TRIGGER historical_run_item_append_only ON commission.historical_run_item;
DROP TRIGGER historical_run_item_no_truncate ON commission.historical_run_item;
DROP TRIGGER historical_posting_log_append_only ON commission.historical_posting_log;
DROP TRIGGER historical_posting_log_no_truncate ON commission.historical_posting_log;
DROP TRIGGER calculation_append_only ON commission.calculation;
DROP TRIGGER calculation_no_truncate ON commission.calculation;
DROP TRIGGER parity_summary_append_only ON commission.parity_summary;
DROP TRIGGER parity_summary_no_truncate ON commission.parity_summary;
DROP TRIGGER reconciliation_finding_append_only ON commission.reconciliation_finding;
DROP TRIGGER reconciliation_finding_no_truncate ON commission.reconciliation_finding;
DROP FUNCTION commission.reject_change();
