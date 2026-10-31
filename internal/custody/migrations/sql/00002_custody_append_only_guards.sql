-- Custody append-only guards (Slice 6). History, snapshot and event rows are
-- never updated, deleted or truncated -- not even by the owner role, which
-- holds no such grant anyway. Corrections append; they never rewrite.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION custody.reject_event_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'custody.%: append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER custody_event_append_only BEFORE UPDATE OR DELETE ON custody.custody_event
    FOR EACH ROW EXECUTE FUNCTION custody.reject_event_change();
CREATE TRIGGER custody_event_no_truncate BEFORE TRUNCATE ON custody.custody_event
    FOR EACH STATEMENT EXECUTE FUNCTION custody.reject_event_change();

-- +goose Down
DROP TRIGGER IF EXISTS custody_event_no_truncate ON custody.custody_event;
DROP TRIGGER IF EXISTS custody_event_append_only ON custody.custody_event;
DROP FUNCTION IF EXISTS custody.reject_event_change();
