-- R1: production authentication and server-side sessions (Architecture V1
-- §5 / §6 / §7: identity owns User, Credential and Session).
--
-- What this migration deliberately does NOT do:
--   * no credential is created, imported or reused: legacy WordPress
--     password hashes, phone numbers and legacy sessions are never copied
--     (provisioning / initial password policy D-34 is open). The application
--     role gets SELECT only on identity.credential; enrolment needs a later,
--     approved migration and command;
--   * no identity.user_account row is activated: Slice 4's
--     user_account_no_login / SHADOW_IMPORTED constraints stay. A credential
--     row is the only target login enablement, and it cannot be written by
--     the application;
--   * no role grant becomes ACTIVE (role_grant_shadow_only stays): an
--     authenticated user has identity, not authority (D-42);
--   * no plaintext secret is stored: credential secrets are PHC-encoded
--     Argon2id hashes; a session token is stored as the SHA-256 digest of a
--     256-bit random value, never the value itself. The CSRF token
--     is derived from the session token (HMAC) and is never stored.

-- +goose Up
CREATE TABLE identity.credential (
    credential_id   uuid        PRIMARY KEY,
    user_id         uuid        NOT NULL UNIQUE REFERENCES identity.user_account (user_id),
    login_name      text        NOT NULL UNIQUE,
    kind            text        NOT NULL,
    secret_hash     text        NOT NULL,
    status          text        NOT NULL,
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL,
    version         bigint      NOT NULL,
    -- Normalised login names: lowercase, no surrounding space.
    CONSTRAINT credential_login_name_format CHECK (login_name = lower(btrim(login_name)) AND length(login_name) BETWEEN 3 AND 128),
    -- AGENTS.md / INV-001: a phone number is never an identity key.
    CONSTRAINT credential_login_name_not_phone CHECK (login_name !~ '^\+?[0-9][0-9 ()-]{6,}$'),
    CONSTRAINT credential_kind CHECK (kind = 'PASSWORD_ARGON2ID'),
    CONSTRAINT credential_secret_phc CHECK (secret_hash ~ '^\$argon2id\$v=19\$m=[0-9]+,t=[0-9]+,p=[0-9]+\$[A-Za-z0-9+/]+\$[A-Za-z0-9+/]+$'),
    CONSTRAINT credential_status CHECK (status IN ('ACTIVE', 'DISABLED')),
    CONSTRAINT credential_version CHECK (version > 0)
);

CREATE TABLE identity.session (
    session_id           uuid        PRIMARY KEY,
    token_sha256         bytea       NOT NULL UNIQUE,
    user_id              uuid        NOT NULL REFERENCES identity.user_account (user_id),
    credential_id        uuid        NOT NULL REFERENCES identity.credential (credential_id),
    -- Architecture §7: a session is never issued by a payment callback (D-26).
    issuance_reason      text        NOT NULL,
    issued_at            timestamptz NOT NULL,
    last_seen_at         timestamptz NOT NULL,
    idle_expires_at      timestamptz NOT NULL,
    absolute_expires_at  timestamptz NOT NULL,
    revoked_at           timestamptz NULL,
    revoke_reason        text        NULL,
    correlation_id       text        NOT NULL,
    CONSTRAINT session_token_sha CHECK (octet_length(token_sha256) = 32),
    CONSTRAINT session_issuance CHECK (issuance_reason = 'PASSWORD_LOGIN'),
    CONSTRAINT session_window CHECK (issued_at <= last_seen_at AND idle_expires_at <= absolute_expires_at AND issued_at < absolute_expires_at),
    CONSTRAINT session_revoked CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL)),
    CONSTRAINT session_revoke_reason CHECK (revoke_reason IS NULL OR revoke_reason IN ('LOGOUT', 'LOGOUT_ALL', 'REPLACED_AT_LOGIN', 'CREDENTIAL_DISABLED', 'ACCOUNT_NOT_PRESENT')),
    CONSTRAINT session_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX session_user_idx ON identity.session (user_id) WHERE revoked_at IS NULL;

-- Append-only authentication evidence. Never a credential, a token, a raw
-- login name, an IP address or a user agent (privacy decisions D-45 / ADR
-- #18 open): an unknown login is identified only by the SHA-256 of its
-- normalised form, which also keys the per-login throttle.
CREATE TABLE identity.auth_event (
    event_id          uuid        PRIMARY KEY,
    seq               bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    kind              text        NOT NULL,
    user_id           uuid        NULL,
    session_id        uuid        NULL,
    login_key_sha256  bytea       NULL,
    reason            text        NULL,
    correlation_id    text        NOT NULL,
    occurred_at       timestamptz NOT NULL,
    CONSTRAINT auth_event_kind CHECK (kind IN ('LOGIN_SUCCEEDED', 'LOGIN_FAILED', 'LOGIN_THROTTLED', 'LOGIN_BLOCKED',
        'LOGOUT', 'LOGOUT_ALL', 'SESSION_REPLACED', 'SESSION_REVOKED', 'CSRF_REJECTED')),
    CONSTRAINT auth_event_login_key CHECK (login_key_sha256 IS NULL OR octet_length(login_key_sha256) = 32),
    CONSTRAINT auth_event_reason CHECK (reason IS NULL OR reason ~ '^[A-Z][A-Z_]{1,63}$'),
    CONSTRAINT auth_event_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);
CREATE INDEX auth_event_throttle_idx ON identity.auth_event (login_key_sha256, occurred_at) WHERE kind = 'LOGIN_FAILED';

GRANT SELECT ON identity.credential TO crm_owner_identity;
GRANT SELECT, INSERT ON identity.session TO crm_owner_identity;
GRANT UPDATE (last_seen_at, idle_expires_at, revoked_at, revoke_reason) ON identity.session TO crm_owner_identity;
GRANT SELECT, INSERT ON identity.auth_event TO crm_owner_identity;

-- +goose StatementBegin
CREATE FUNCTION identity.reject_auth_history_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'identity.%: append-only authentication evidence; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER auth_event_append_only BEFORE UPDATE OR DELETE ON identity.auth_event
    FOR EACH ROW EXECUTE FUNCTION identity.reject_auth_history_change();
CREATE TRIGGER auth_event_no_truncate BEFORE TRUNCATE ON identity.auth_event
    FOR EACH STATEMENT EXECUTE FUNCTION identity.reject_auth_history_change();
CREATE TRIGGER session_no_delete BEFORE DELETE ON identity.session
    FOR EACH ROW EXECUTE FUNCTION identity.reject_auth_history_change();
CREATE TRIGGER session_no_truncate BEFORE TRUNCATE ON identity.session
    FOR EACH STATEMENT EXECUTE FUNCTION identity.reject_auth_history_change();
CREATE TRIGGER credential_no_delete BEFORE DELETE ON identity.credential
    FOR EACH ROW EXECUTE FUNCTION identity.reject_auth_history_change();

-- +goose Down
DROP TABLE IF EXISTS identity.auth_event;
DROP TABLE IF EXISTS identity.session;
DROP TABLE IF EXISTS identity.credential;
DROP FUNCTION IF EXISTS identity.reject_auth_history_change();
