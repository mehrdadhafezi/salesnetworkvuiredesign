-- Cases append-only guards (Slice 6). History, snapshot and event rows are
-- never updated, deleted or truncated -- not even by the owner role, which
-- holds no such grant anyway. Corrections append; they never rewrite.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION cases.reject_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'cases.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER case_state_history_append_only BEFORE UPDATE OR DELETE ON cases.case_state_history
    FOR EACH ROW EXECUTE FUNCTION cases.reject_history_change();
CREATE TRIGGER case_state_history_no_truncate BEFORE TRUNCATE ON cases.case_state_history
    FOR EACH STATEMENT EXECUTE FUNCTION cases.reject_history_change();

CREATE TRIGGER attribution_snapshot_append_only BEFORE UPDATE OR DELETE ON cases.attribution_snapshot
    FOR EACH ROW EXECUTE FUNCTION cases.reject_history_change();
CREATE TRIGGER attribution_snapshot_no_truncate BEFORE TRUNCATE ON cases.attribution_snapshot
    FOR EACH STATEMENT EXECUTE FUNCTION cases.reject_history_change();

-- +goose Down
DROP TRIGGER IF EXISTS case_state_history_no_truncate ON cases.case_state_history;
DROP TRIGGER IF EXISTS case_state_history_append_only ON cases.case_state_history;
DROP TRIGGER IF EXISTS attribution_snapshot_no_truncate ON cases.attribution_snapshot;
DROP TRIGGER IF EXISTS attribution_snapshot_append_only ON cases.attribution_snapshot;
DROP FUNCTION IF EXISTS cases.reject_history_change();
