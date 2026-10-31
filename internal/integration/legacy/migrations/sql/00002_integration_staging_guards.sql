-- Integration staging guards (Slice 3). A batch may become LOADED only when
-- every one of its entities is LOADED and declared COMPLETE, and
-- LOADED_INCOMPLETE only when every entity is LOADED and at least one is not
-- COMPLETE. Failure can therefore never look like success at batch level.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION integration.import_batch_guard_loaded() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    total      bigint;
    not_loaded bigint;
    incomplete bigint;
BEGIN
    IF NEW.state NOT IN ('LOADED', 'LOADED_INCOMPLETE') THEN
        RETURN NEW;
    END IF;
    SELECT count(*),
           count(*) FILTER (WHERE state <> 'LOADED'),
           count(*) FILTER (WHERE completeness <> 'COMPLETE')
      INTO total, not_loaded, incomplete
      FROM integration.import_entity
     WHERE batch_id = NEW.batch_id;
    IF total = 0 OR not_loaded > 0 THEN
        RAISE EXCEPTION 'integration.import_batch %: % of % entities not LOADED', NEW.batch_id, not_loaded, total
            USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.state = 'LOADED' AND incomplete > 0 THEN
        RAISE EXCEPTION 'integration.import_batch %: % entities are not COMPLETE; use LOADED_INCOMPLETE', NEW.batch_id, incomplete
            USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.state = 'LOADED_INCOMPLETE' AND incomplete = 0 THEN
        RAISE EXCEPTION 'integration.import_batch %: every entity is COMPLETE; use LOADED', NEW.batch_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER import_batch_guard_loaded
    BEFORE INSERT OR UPDATE OF state ON integration.import_batch
    FOR EACH ROW EXECUTE FUNCTION integration.import_batch_guard_loaded();

-- +goose Down
DROP TRIGGER IF EXISTS import_batch_guard_loaded ON integration.import_batch;
DROP FUNCTION IF EXISTS integration.import_batch_guard_loaded();
