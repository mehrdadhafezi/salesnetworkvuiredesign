-- Platform migration: boolean feature-flag store (E0 §6 #17; default-off).
-- Owner: module "platform_featureflags" (NOLOGIN role crm_owner_platform_featureflags).

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE crm_owner_platform_featureflags NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
-- +goose StatementEnd

CREATE SCHEMA platform_featureflags;
REVOKE ALL ON SCHEMA platform_featureflags FROM PUBLIC;
GRANT USAGE ON SCHEMA platform_featureflags TO crm_owner_platform_featureflags;

CREATE TABLE platform_featureflags.feature_flag (
    flag_key     text        PRIMARY KEY,
    enabled      boolean     NOT NULL,
    description  text        NULL,
    updated_at   timestamptz NOT NULL,
    CONSTRAINT feature_flag_key_format CHECK (flag_key ~ '^[a-z][a-z0-9_.-]{2,63}$'),
    CONSTRAINT feature_flag_description_length CHECK (description IS NULL OR length(description) BETWEEN 1 AND 500)
);

REVOKE ALL ON platform_featureflags.feature_flag FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON platform_featureflags.feature_flag TO crm_owner_platform_featureflags;

-- +goose Down
DROP SCHEMA platform_featureflags CASCADE;
