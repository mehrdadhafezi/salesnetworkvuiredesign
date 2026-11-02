-- Fulfillment shadow append-only guards (Slice 10). Runs, sources, shadow facts,
-- and history are never updated, deleted or truncated -- not even by the
-- owner role, which holds no such grant anyway. A new snapshot is a new run.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION fulfillment.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'fulfillment.%: append-only shadow fact; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER import_run_append_only BEFORE UPDATE OR DELETE ON fulfillment.import_run
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER import_run_no_truncate BEFORE TRUNCATE ON fulfillment.import_run
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER run_source_append_only BEFORE UPDATE OR DELETE ON fulfillment.run_source
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER run_source_no_truncate BEFORE TRUNCATE ON fulfillment.run_source
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER fulfillment_case_append_only BEFORE UPDATE OR DELETE ON fulfillment.fulfillment_case
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER fulfillment_case_no_truncate BEFORE TRUNCATE ON fulfillment.fulfillment_case
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER assignment_append_only BEFORE UPDATE OR DELETE ON fulfillment.assignment
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER assignment_no_truncate BEFORE TRUNCATE ON fulfillment.assignment
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER service_evidence_append_only BEFORE UPDATE OR DELETE ON fulfillment.service_evidence
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER service_evidence_no_truncate BEFORE TRUNCATE ON fulfillment.service_evidence
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER case_event_append_only BEFORE UPDATE OR DELETE ON fulfillment.case_event
    FOR EACH ROW EXECUTE FUNCTION fulfillment.reject_change();
CREATE TRIGGER case_event_no_truncate BEFORE TRUNCATE ON fulfillment.case_event
    FOR EACH STATEMENT EXECUTE FUNCTION fulfillment.reject_change();

-- +goose Down
DROP TRIGGER import_run_append_only ON fulfillment.import_run;
DROP TRIGGER import_run_no_truncate ON fulfillment.import_run;
DROP TRIGGER run_source_append_only ON fulfillment.run_source;
DROP TRIGGER run_source_no_truncate ON fulfillment.run_source;
DROP TRIGGER fulfillment_case_append_only ON fulfillment.fulfillment_case;
DROP TRIGGER fulfillment_case_no_truncate ON fulfillment.fulfillment_case;
DROP TRIGGER assignment_append_only ON fulfillment.assignment;
DROP TRIGGER assignment_no_truncate ON fulfillment.assignment;
DROP TRIGGER service_evidence_append_only ON fulfillment.service_evidence;
DROP TRIGGER service_evidence_no_truncate ON fulfillment.service_evidence;
DROP TRIGGER case_event_append_only ON fulfillment.case_event;
DROP TRIGGER case_event_no_truncate ON fulfillment.case_event;
DROP FUNCTION fulfillment.reject_change();
