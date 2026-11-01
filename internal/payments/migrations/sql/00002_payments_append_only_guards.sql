-- Payments append-only guards (Slice 7). Attempt state history is never
-- updated, deleted or truncated -- not even by the owner role, which holds
-- no such grant anyway.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION payments.reject_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'payments.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER attempt_state_history_append_only BEFORE UPDATE OR DELETE ON payments.attempt_state_history
    FOR EACH ROW EXECUTE FUNCTION payments.reject_history_change();
CREATE TRIGGER attempt_state_history_no_truncate BEFORE TRUNCATE ON payments.attempt_state_history
    FOR EACH STATEMENT EXECUTE FUNCTION payments.reject_history_change();

-- +goose Down
DROP TRIGGER IF EXISTS attempt_state_history_no_truncate ON payments.attempt_state_history;
DROP TRIGGER IF EXISTS attempt_state_history_append_only ON payments.attempt_state_history;
DROP FUNCTION IF EXISTS payments.reject_history_change();
