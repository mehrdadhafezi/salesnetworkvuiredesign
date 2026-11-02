-- Biawin shadow append-only guards (Slice 10). Runs, sources, shadow facts,
-- history and parity are never updated, deleted or truncated -- not even by the
-- owner role, which holds no such grant anyway. A new snapshot is a new run.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION biawin.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'biawin.%: append-only shadow fact; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER import_run_append_only BEFORE UPDATE OR DELETE ON biawin.import_run
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER import_run_no_truncate BEFORE TRUNCATE ON biawin.import_run
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER run_source_append_only BEFORE UPDATE OR DELETE ON biawin.run_source
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER run_source_no_truncate BEFORE TRUNCATE ON biawin.run_source
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_append_only BEFORE UPDATE OR DELETE ON biawin.subscription
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_no_truncate BEFORE TRUNCATE ON biawin.subscription
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_item_append_only BEFORE UPDATE OR DELETE ON biawin.subscription_item
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_item_no_truncate BEFORE TRUNCATE ON biawin.subscription_item
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER item_invoice_link_append_only BEFORE UPDATE OR DELETE ON biawin.item_invoice_link
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER item_invoice_link_no_truncate BEFORE TRUNCATE ON biawin.item_invoice_link
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_event_append_only BEFORE UPDATE OR DELETE ON biawin.subscription_event
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER subscription_event_no_truncate BEFORE TRUNCATE ON biawin.subscription_event
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER parity_check_append_only BEFORE UPDATE OR DELETE ON biawin.parity_check
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER parity_check_no_truncate BEFORE TRUNCATE ON biawin.parity_check
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER parity_row_append_only BEFORE UPDATE OR DELETE ON biawin.parity_row
    FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER parity_row_no_truncate BEFORE TRUNCATE ON biawin.parity_row
    FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();

-- +goose Down
DROP TRIGGER import_run_append_only ON biawin.import_run;
DROP TRIGGER import_run_no_truncate ON biawin.import_run;
DROP TRIGGER run_source_append_only ON biawin.run_source;
DROP TRIGGER run_source_no_truncate ON biawin.run_source;
DROP TRIGGER subscription_append_only ON biawin.subscription;
DROP TRIGGER subscription_no_truncate ON biawin.subscription;
DROP TRIGGER subscription_item_append_only ON biawin.subscription_item;
DROP TRIGGER subscription_item_no_truncate ON biawin.subscription_item;
DROP TRIGGER item_invoice_link_append_only ON biawin.item_invoice_link;
DROP TRIGGER item_invoice_link_no_truncate ON biawin.item_invoice_link;
DROP TRIGGER subscription_event_append_only ON biawin.subscription_event;
DROP TRIGGER subscription_event_no_truncate ON biawin.subscription_event;
DROP TRIGGER parity_check_append_only ON biawin.parity_check;
DROP TRIGGER parity_check_no_truncate ON biawin.parity_check;
DROP TRIGGER parity_row_append_only ON biawin.parity_row;
DROP TRIGGER parity_row_no_truncate ON biawin.parity_row;
DROP FUNCTION biawin.reject_change();
