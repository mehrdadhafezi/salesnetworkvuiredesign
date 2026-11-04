#!/usr/bin/env bash
# Slice 15 cutover-readiness evidence collector. READ-ONLY. NOT a cutover tool.
#
#   scripts/cutover/readiness-evidence.sh static
#       Repository-only checks. No network, no database, no writes.
#
#   CRM_CUTOVER_ENV=disposable CRM_CUTOVER_DB_URL=postgres://...@localhost/... \
#   scripts/cutover/readiness-evidence.sh db
#       Read-only queries against a DISPOSABLE local database that crm-migrate
#       has already prepared. Refuses unless CRM_CUTOVER_ENV=disposable, the
#       host is a loopback address, and CRM_ENV is not production. Every
#       session runs with default_transaction_read_only=on.
#
# This script never deploys, migrates, changes a flag, grants a capability,
# imports data, or calls a legacy, gateway or Biawin endpoint. Its output is
# evidence for docs/cutover/*; it is not proof of any production setting.
#
# An inspection that cannot run (missing toolchain, git error, unreadable
# file) is reported as UNKNOWN, never as VERIFIED absence. Exit status:
# 0 = every inspection ran (findings may exist), 3 = at least one UNKNOWN,
# 2 = refused, 64 = usage.
set -euo pipefail

mode="${1:-static}"
root="$(git rev-parse --show-toplevel)"
cd "$root"
pass=0 fail=0 info=0 unknown=0
ok()   { printf 'VERIFIED  %s\n' "$*"; pass=$((pass + 1)); }
bad()  { printf 'FINDING   %s\n' "$*"; fail=$((fail + 1)); }
note() { printf 'INFO      %s\n' "$*"; info=$((info + 1)); }
unk()  { printf 'UNKNOWN   %s\n' "$*"; unknown=$((unknown + 1)); }

# gg <git-grep args>: prints match | none | error. git grep exits 0 on a
# match, 1 on no match and >1 on failure; only 1 proves absence.
gg() {
  local rc=0
  git grep -q "$@" >/dev/null 2>&1 || rc=$?
  case "$rc" in 0) echo match ;; 1) echo none ;; *) echo error ;; esac
}

refuse() { printf 'REFUSED   %s\n' "$*" >&2; exit 2; }

static_checks() {
  echo "== environment: repository only (no database, no network)"
  note "HEAD $(git rev-parse HEAD)"
  [ -z "$(git status --porcelain)" ] && ok "working tree clean" || bad "working tree has uncommitted changes"

  # Legacy behavioural baseline: 2.0.160 synced at 86859d7.
  if [ -z "$(git diff --name-only 86859d7 HEAD -- plugin/current)" ]; then ok "plugin/current identical to the 2.0.160 sync (86859d7)"; else bad "plugin/current differs from 86859d7"; fi
  grep -q "2.0.160" plugin/current/sales-network.php && ok "legacy SN_VERSION 2.0.160" || bad "legacy version is not 2.0.160"

  # Every migration file is tracked.
  untracked="$(comm -23 <(find internal -path '*/migrations/sql/*.sql' | sort) <(git ls-files 'internal/*migrations/sql/*.sql' | sort) | wc -l)"
  [ "$untracked" = "0" ] && ok "every migration .sql file is tracked ($(git ls-files 'internal/*migrations/sql/*.sql' | wc -l) files)" || bad "$untracked migration .sql files are not tracked"

  # No migration seeds a feature flag (flags must default OFF).
  case "$(gg -iE "INSERT INTO platform_featureflags" -- 'internal/*.sql')" in
    match) bad "a migration seeds a feature flag" ;;
    none) ok "no migration seeds a feature flag (all flags absent = OFF)" ;;
    *) unk "could not inspect migrations for flag seeds" ;;
  esac

  # Role → capability grants in migrations: only the Slice 4 HR read grants.
  local out rc=0
  out="$(git grep -hE "^\s*\('[a-z_]+', '[a-z_.]+'\)" -- 'internal/identity/migrations/sql/*.sql' 2>&1)" || rc=$?
  if [ "$rc" -gt 1 ]; then
    unk "could not inspect identity migrations for role grants"
  else
    grants="$( [ -n "$out" ] && printf '%s\n' "$out" | wc -l || echo 0)"
    [ "$grants" = "6" ] && ok "migrations grant exactly 6 role capabilities (hr → identity/org reads)" || bad "migrations grant $grants role capabilities (expected 6)"
  fi

  # Production authentication.
  case "$(gg -E "authorization.Anonymous\{\}" -- cmd/crm-api)" in
    match) bad "crm-api resolves every non-dev request as authorization.Anonymous: no production authentication exists" ;;
    none) note "crm-api does not use authorization.Anonymous (re-check how production subjects are resolved)" ;;
    *) unk "could not inspect crm-api subject resolution" ;;
  esac
  case "$(gg "ErrProductionLockout" -- internal/platform/authstub)" in
    match) ok "dev auth stub is locked out of production" ;;
    none) bad "dev auth stub production lockout not found" ;;
    *) unk "could not inspect the dev auth stub" ;;
  esac

  # Runtime workers: is any cmd wired to the background steps?
  # (payments receipt finaliser, unbound-upload expiry, outbox relay,
  # legacy write-back worker, provider-operation dispatch / reconcile)
  for sym in "Finalise(" "ExpireUnbound(" "relay.Drain(" "writeback.New(" "providerops" ; do
    case "$(gg -F "$sym" -- 'cmd/*.go' ':!*_test.go' ':!cmd/crm-api/*')" in
      match) note "an offline command references $sym" ;;
      none) bad "no offline command runs $sym (worker not deployable)" ;;
      *) unk "could not inspect commands for $sym" ;;
    esac
  done

  # Provider packages unreachable from every command.
  # go list omits packages it cannot load, so absence counts only when the
  # enumeration itself succeeded.
  local deps lrc=0
  deps="$(go list -deps ./cmd/... 2>&1)" || lrc=$?
  if [ "$lrc" -ne 0 ]; then
    unk "dependency enumeration failed (go list exit $lrc: $(printf '%s' "$deps" | head -1)); provider reachability not inspected"
  elif printf '%s\n' "$deps" | grep -qE "/internal/integration/(biawin|gateway/synthetic)$"; then
    bad "a command depends on a provider adapter package"
  else
    ok "no command depends on the Biawin or synthetic gateway adapter"
  fi

  # Deployment artefacts.
  if ls Dockerfile* docker-compose* deploy k8s helm .github/workflows/*deploy* >/dev/null 2>&1; then note "deployment artefacts present"; else bad "no deployment, container or infrastructure definition in the repository"; fi

  # Flags referenced by crm-api.
  note "feature flags referenced by crm-api: $(git grep -hoE '(Flag|Family) *= *"[a-z_.]+"' -- cmd/crm-api | grep -oE '"[a-z_.]+"' | tr -d '"' | sort -u | tr '\n' ' ')"
}

db_checks() {
  [ "${CRM_CUTOVER_ENV:-}" = "disposable" ] || refuse "set CRM_CUTOVER_ENV=disposable to confirm the target is a throwaway database"
  [ "${CRM_ENV:-}" != "production" ] || refuse "CRM_ENV=production"
  url="${CRM_CUTOVER_DB_URL:-}"
  [ -n "$url" ] || refuse "CRM_CUTOVER_DB_URL is not set"
  host="$(sed -E 's#^[a-z]+://([^@/]*@)?(\[[^]]+\]|[^:/?]+).*#\2#' <<<"$url")"
  case "$host" in localhost|127.0.0.1|"[::1]"|::1) ;; *) refuse "host '$host' is not a loopback address";; esac
  command -v psql >/dev/null || refuse "psql not found"
  echo "== environment: disposable local database ($host), read-only session"
  q() { PGOPTIONS='-c default_transaction_read_only=on' psql "$url" -v ON_ERROR_STOP=1 -tAc "$1"; }
  [ "$(q 'SHOW default_transaction_read_only')" = "on" ] && ok "session is read-only" || refuse "could not force a read-only session"
  vt="$(q "SELECT count(*) FROM information_schema.tables WHERE table_name LIKE '%schema_migrations%'")"
  note "migration version tables: $vt (crm-migrate status is the authoritative applied list)"
  flags="$(q 'SELECT count(*) FROM platform_featureflags.feature_flag WHERE enabled')"
  [ "$flags" = "0" ] && ok "no feature flag is enabled in this database" || bad "$flags feature flags are enabled"
  grants="$(q 'SELECT count(*) FROM identity.role_capability')"
  note "role_capability rows: $grants ($(q "SELECT string_agg(role_key||'→'||capability_key, ', ' ORDER BY 1) FROM identity.role_capability"))"
  for s in identity organization customers cases custody billing payments reporting finance biawin fulfillment commission catalog; do
    note "$s tables: $(q "SELECT count(*) FROM information_schema.tables WHERE table_schema = '$s'")"
  done
  note "staging tables: $(q "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'integration' AND table_name LIKE 'stg\_%'")"
  note "staged snapshots: $(q 'SELECT count(*) FROM integration.import_batch' 2>/dev/null || echo n/a)"
}

case "$mode" in
  static) static_checks ;;
  db) db_checks ;;
  *) echo "usage: $0 static|db" >&2; exit 64 ;;
esac
printf '\nsummary: %d verified, %d findings, %d unknown, %d info\n' "$pass" "$fail" "$unknown" "$info"
[ "$unknown" -eq 0 ] || exit 3
