#!/bin/sh
# R3 staging bootstrap (runs once per deployment, after migrations):
#   - per-module LOGIN roles: NOINHERIT, not superuser, member of exactly one
#     crm_owner_<module> role (db.Runner refuses anything else);
#   - a read-all backup login (pg_read_all_data) for crm-backup;
#   - the deliberate STAGING attestation (worker.environment.staging) on this
#     fresh disposable database; refused if another environment is attested.
# No job switch or feature flag is enabled.
set -eu
umask 077
# The evidence volume is written only by the application uid.
if [ -d /evidence ]; then chown 65532:65532 /evidence && chmod 0700 /evidence; fi
printf 'postgres:5432:crm_staging:crm_staging_admin:%s\n' "$(cat /run/secrets/pg_admin_password)" > /tmp/.pgpass
psql -q -v ON_ERROR_STOP=1 -f /run/secrets/login_passwords >/dev/null
psql -q -v ON_ERROR_STOP=1 <<'SQL'
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM platform_featureflags.feature_flag
             WHERE enabled AND flag_key LIKE 'worker.environment.%' AND flag_key <> 'worker.environment.staging') THEN
    RAISE EXCEPTION 'refusing: this database attests another environment';
  END IF;
END $$;
INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, description, updated_at)
VALUES ('worker.environment.staging', true, 'R3 disposable staging attestation (bootstrap)', now())
ON CONFLICT (flag_key) DO UPDATE SET enabled = true, updated_at = now();
SQL
n="$(psql -tA -c "SELECT count(*) FROM platform_featureflags.feature_flag WHERE enabled AND flag_key <> 'worker.environment.staging'")"
if [ "$n" != "0" ]; then echo "bootstrap: refusing: $n feature flags are enabled on a fresh staging database" >&2; exit 1; fi
echo "bootstrap: staging logins provisioned; attestation staging; 0 other flags enabled"
