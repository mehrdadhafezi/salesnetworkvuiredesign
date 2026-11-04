-- Catalog append-only guards (Slice 14): definitions, components and scope
-- events are never updated, deleted or truncated.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION catalog.reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'catalog.%: append-only record; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER service_version_append_only BEFORE UPDATE OR DELETE ON catalog.service_version
    FOR EACH ROW EXECUTE FUNCTION catalog.reject_change();
CREATE TRIGGER service_version_no_truncate BEFORE TRUNCATE ON catalog.service_version
    FOR EACH STATEMENT EXECUTE FUNCTION catalog.reject_change();
CREATE TRIGGER subscription_component_append_only BEFORE UPDATE OR DELETE ON catalog.subscription_component
    FOR EACH ROW EXECUTE FUNCTION catalog.reject_change();
CREATE TRIGGER subscription_component_no_truncate BEFORE TRUNCATE ON catalog.subscription_component
    FOR EACH STATEMENT EXECUTE FUNCTION catalog.reject_change();
CREATE TRIGGER service_scope_event_append_only BEFORE UPDATE OR DELETE ON catalog.service_scope_event
    FOR EACH ROW EXECUTE FUNCTION catalog.reject_change();
CREATE TRIGGER service_scope_event_no_truncate BEFORE TRUNCATE ON catalog.service_scope_event
    FOR EACH STATEMENT EXECUTE FUNCTION catalog.reject_change();

-- +goose Down
DROP TRIGGER service_version_append_only ON catalog.service_version;
DROP TRIGGER service_version_no_truncate ON catalog.service_version;
DROP TRIGGER subscription_component_append_only ON catalog.subscription_component;
DROP TRIGGER subscription_component_no_truncate ON catalog.subscription_component;
DROP TRIGGER service_scope_event_append_only ON catalog.service_scope_event;
DROP TRIGGER service_scope_event_no_truncate ON catalog.service_scope_event;
DROP FUNCTION catalog.reject_change();
