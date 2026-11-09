#!/usr/bin/env bash
# R3 disaster-recovery drill against the RUNNING disposable staging stack
# (deploy/staging/stage.sh up). Every scenario records expected behaviour,
# actual result, measured seconds, pass/fail and residual risk as one JSON
# line in .state/drill-<timestamp>.jsonl. A simulated alert condition is
# NOT a delivered page. Never run against anything but crm-staging.
#
#   drill.sh <artifact-dir>
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"
project=crm-staging
tag="$(cat .state/current)"
artifact="$(cd "${1:?artifact dir}" && pwd)"
export CRM_RELEASE="$tag"
compose() { docker compose -p "$project" -f compose.yaml "$@"; }
out=".state/drill-$(date -u +%Y%m%dT%H%M%SZ).jsonl"
net="${project}_internal"
probe() { docker run --rm --network "$net" alpine:3.20 wget -q -T 10 -O - "$1" 2>/dev/null; }
probe_code() { docker run --rm --network "$net" alpine:3.20 sh -c "wget -T 10 -S -O /dev/null $1 2>&1 | grep -o 'HTTP/1.[01] [0-9][0-9][0-9]' | tail -1 | cut -d' ' -f2"; }
health() { docker inspect --format '{{.State.Health.Status}}' "$(compose ps -q "$1")" 2>/dev/null; }
wait_healthy() { local s="$1" t=0; while [ "$(health "$s")" != healthy ] && [ $t -lt 120 ]; do sleep 1; t=$((t+1)); done; [ "$(health "$s")" = healthy ]; }
rec() { # id name expected actual pass seconds residual
  jq -cn --arg id "$1" --arg n "$2" --arg e "$3" --arg a "$4" --arg p "$5" --arg s "$6" --arg r "$7" --arg rel "$tag" \
    '{scenario:($id|tonumber), name:$n, expected:$e, actual:$a, result:$p, seconds:($s|tonumber), residual_risk:$r, release:$rel, at:(now|todate)}' | tee -a "$out"
}
psql_admin() { compose exec -T postgres psql -U crm_staging_admin -d crm_staging -tA -c "$1"; }
t0() { date +%s.%N; }
dt() { echo "$(date +%s.%N) - $1" | bc | awk '{printf "%.2f", $1}'; }

# A crash is a process dying on its own: kill the container's main process
# from the host PID namespace (docker kill counts as a manual stop and
# would not exercise the restart policy).
crash() { kill -9 "$(docker inspect --format '{{.State.Pid}}' "$(compose ps -q "$1")")"; }

# 1. API process termination.
s=$(t0); crash api
sleep 1; if wait_healthy api; then r=PASS; else r=FAIL; fi
rec 1 api_termination "SIGKILL: restart policy restarts crm-api; healthy again" "health=$(health api)" $r "$(dt $s)" "no in-flight request survives a kill; clients retry idempotently"

# 2. Worker termination.
s=$(t0); crash worker
sleep 1; if wait_healthy worker; then r=PASS; else r=FAIL; fi
rec 2 worker_termination "SIGKILL: worker restarts; durable claims recovered (R2 tests)" "health=$(health worker)" $r "$(dt $s)" "passes in flight are redone; covered by R2 claim/lease tests"

# 3. PostgreSQL unavailable.
s=$(t0); compose stop postgres >/dev/null 2>&1; sleep 4
down_ready="$(probe_code http://worker:9091/health/ready)"
check_up="$(probe http://worker:9091/metrics | awk '/^crm_worker_check_up/{print $2}' | sort -u | tr '\n' ',')"
api_live="$(probe_code http://api:8080/health/live)"
compose start postgres >/dev/null 2>&1; wait_healthy postgres
for _ in $(seq 1 30); do [ "$(probe_code http://worker:9091/health/ready)" = 200 ] && break; sleep 1; done
up_ready="$(probe_code http://worker:9091/health/ready)"
[ "$down_ready" = 503 ] && [ "$up_ready" = 200 ] && [[ "$check_up" == *0* ]] && r=PASS || r=FAIL
rec 3 postgres_unavailable "worker unready (503, check_up=0) while the database is down; ready again after" \
  "during=$down_ready check_up=$check_up api_live=$api_live after=$up_ready" $r "$(dt $s)" "crm-api readiness has no database check in this configuration (no surfaces opened); documented gap"

# 4. Incorrect database environment.
s=$(t0)
res="$(compose run --rm --no-deps -e CRM_WORKER_ENV=production --entrypoint /opt/crm/bin/crm-worker worker check 2>&1)"; code=$?
[ $code -ne 0 ] && grep -q "attestation failed" <<< "$res" && r=PASS || r=FAIL
rec 4 wrong_environment "worker declared production on a staging-attested database refuses to start" "exit=$code $(grep -o 'attestation failed[^"]*' <<< "$res" | head -1)" $r "$(dt $s)" "relies on operators never attesting a disposable database as production"

# 13. Failed migration (on a drill-only scratch database).
s=$(t0); db="crm_drill_mig_$(date +%s)"
psql_admin "CREATE DATABASE $db" >/dev/null
compose exec -T postgres psql -U crm_staging_admin -d "$db" -qc "CREATE SCHEMA identity" >/dev/null
pw="$(cat .secrets/pg_admin_password 2>/dev/null)"
res="$(compose run --rm --no-deps -e CRM_MIGRATE_DATABASE_URL="postgres://crm_staging_admin:${pw}@postgres:5432/${db}?sslmode=disable" -e CRM_MIGRATE_DATABASE_URL_FILE= migrate 2>&1)"; code=$?
applied="$(compose exec -T postgres psql -U crm_staging_admin -d "$db" -tA -c "SELECT count(*) FROM crm_schema_migrations_identity WHERE is_applied" 2>/dev/null || echo 0)"
leak="$(grep -c "$pw" <<< "$res" || true)"
psql_admin "DROP DATABASE $db" >/dev/null
unset pw
[ $code -ne 0 ] && [ "${applied:-0}" -le 1 ] && [ "$leak" = 0 ] && r=PASS || r=FAIL
rec 13 failed_migration "crm-migrate stops at the first failing migration, non-zero exit, no credential in output; services gated on migrate never start" \
  "exit=$code identity_applied=${applied:-0} credential_in_output=$leak" $r "$(dt $s)" "no automatic down-migration; recovery is restore + fix-forward"

# 14. Deployment health-check failure.
s=$(t0)
compose -f compose.yaml -f faults/api-misconfig.yaml up -d --no-deps --wait --wait-timeout 25 api >/dev/null 2>&1; code=$?
st="$(docker inspect --format '{{.State.Status}}/{{.State.ExitCode}}' "$(compose ps -aq api)")"
compose up -d --no-deps --wait --wait-timeout 60 api >/dev/null 2>&1; recovered="$(health api)"
[ $code -ne 0 ] && [ "$recovered" = healthy ] && r=PASS || r=FAIL
rec 14 deployment_health_failure "a misconfigured release fails --wait (non-zero); the previous configuration is restored healthy" \
  "wait_exit=$code faulty_state=$st after=$recovered" $r "$(dt $s)" "stage.sh up reports FAILED; an operator decides to roll back"

# 16. Missing secret.
s=$(t0)
res="$(compose run --rm --no-deps -e CRM_WORKER_PAYMENTS_DATABASE_URL_FILE=/run/secrets/does-not-exist --entrypoint /opt/crm/bin/crm-worker worker check 2>&1)"; code=$?
[ $code -ne 0 ] && grep -q "CRM_WORKER_PAYMENTS_DATABASE_URL" <<< "$res" && r=PASS || r=FAIL
rec 16 missing_secret "a missing credential file refuses startup with the variable name only" "exit=$code" $r "$(dt $s)" "none for startup; rotation procedure is documented, not automated"

# 18. Alert condition generated (NOT delivered): enable an unapproved flag.
s=$(t0)
before="$(probe http://worker:9091/metrics | awk '/^crm_worker_feature_flags_enabled/{print $2}')"
psql_admin "INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, description, updated_at) VALUES ('drill.unapproved_flag', true, 'R3 drill', now()) ON CONFLICT (flag_key) DO UPDATE SET enabled = true, updated_at = now()" >/dev/null
during="$(probe http://worker:9091/metrics | awk '/^crm_worker_feature_flags_enabled/{print $2}')"
psql_admin "UPDATE platform_featureflags.feature_flag SET enabled = false, updated_at = now() WHERE flag_key = 'drill.unapproved_flag'" >/dev/null
after="$(probe http://worker:9091/metrics | awk '/^crm_worker_feature_flags_enabled/{print $2}')"
[ "$before" = 1 ] && [ "$during" = 2 ] && [ "$after" = 1 ] && r=PASS || r=FAIL
rec 18 alert_condition_generated "CRMUnexpectedFlagActivation condition (crm_worker_feature_flags_enabled > 1) becomes true, then clears" \
  "before=$before during=$during after=$after (no Alertmanager: condition only, not a delivered page)" $r "$(dt $s)" "alert delivery and paging are NOT configured or verified"

# 17. Monitoring unavailable.
s=$(t0)
ready="$(probe_code http://api:8080/health/ready)"
[ "$ready" = 200 ] && r=PARTIAL || r=FAIL
rec 17 monitoring_unavailable "no metrics scraper is deployed: the application is unaffected; detection relies on the CRMWatchdog dead-man's switch" \
  "api_ready=$ready scraper=absent" $r "$(dt $s)" "monitoring pipeline and dead-man's switch delivery are NOT deployed (operator infrastructure)"

# 19. Disk capacity threshold (simulated on a 1 MiB tmpfs).
s=$(t0)
pct="$(docker run --rm --mount type=tmpfs,dst=/v,tmpfs-size=1048576 alpine:3.20 sh -c 'dd if=/dev/zero of=/v/f bs=1024 count=900 2>/dev/null; df -P /v | awk "NR==2{print \$4/\$2}"')"
cond="$(awk -v p="$pct" 'BEGIN{print (p < 0.15) ? "true" : "false"}')"
[ "$cond" = true ] && r=PASS || r=FAIL
rec 19 disk_threshold_simulated "free/size below 0.15 makes the CRMVolumeSpaceLow condition true" "free_ratio=$pct condition=$cond" $r "$(dt $s)" "node exporter not deployed; condition evaluated by the drill, not by Prometheus"

# Live staging backup → verify → restore rehearsal → discard (5/6/7 live
# complements of the Go tests). crm-backup runs on the internal network in
# the PostgreSQL 16 client image, as uid 65532, with a drill age key.
s=$(t0)
install -d -m 0700 .state/backups .state/keys
chown 65532:65532 .state/backups .state/keys
key=".state/keys/drill-key.txt"
if [ ! -f "$key" ]; then
  age-keygen -o "$key" 2>/dev/null || { echo "drill: age-keygen failed" >&2; exit 1; }
  age-keygen -y "$key" > .state/keys/recipients.txt
  chown 65532:65532 "$key" .state/keys/recipients.txt; chmod 0400 "$key"
fi
bname="staging-$(date -u +%Y%m%dT%H%M%SZ)"
img=postgres@sha256:5264710be20f409750ac6dc4e5706dcef929c87e241605463f0f9aefa932a8d9
docker run --rm --network "$net" --user 65532:65532 \
    -v /usr/bin/age:/usr/local/bin/age:ro -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" \
    -v "$here/.state/backups:/backups" -v "$here/.state/keys:/keys:ro" -v "$here/.secrets/dsn_backup:/secrets/dsn_backup:ro" -v "$here/.secrets/pg_admin_dsn:/secrets/pg_admin_dsn:ro" \
    -v "${project}_evidence:/evidence:ro" -e CRM_BACKUP_ENV=staging -e CRM_BACKUP_ALLOWED_ROOTS=/backups \
    -e CRM_BACKUP_SOURCE_DATABASE_URL_FILE=/secrets/dsn_backup -e CRM_BACKUP_AGE_RECIPIENTS=/keys/recipients.txt \
    -e CRM_BACKUP_EVIDENCE_ROOT=/evidence -e CRM_BACKUP_METRICS_FILE=/backups/crm_backup.prom -e CRM_BACKUP_DEST="/backups/$bname" \
    "$img" crm-backup backup > ".state/$bname.backup.json" 2> ".state/$bname.backup.err"; bcode=$?
vcode=1; rcode=1; dcode=1
if [ $bcode = 0 ]; then
  docker run --rm --network "$net" --user 65532:65532 -v /usr/bin/age:/usr/local/bin/age:ro -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" \
    -v "$here/.state/backups:/backups" -v "$here/.state/keys:/keys:ro" -e CRM_BACKUP_AGE_IDENTITY=/keys/drill-key.txt \
    "$img" crm-backup verify "/backups/$bname" > ".state/$bname.verify.json" 2>&1; vcode=$?
  target="crm_restore_drill_$(date +%s)"
  docker run --rm --network "$net" --user 65532:65532 -v /usr/bin/age:/usr/local/bin/age:ro -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" \
    -v "$here/.state/backups:/backups" -v "$here/.state/keys:/keys:ro" -v "$here/.secrets/dsn_backup:/secrets/dsn_backup:ro" -v "$here/.secrets/pg_admin_dsn:/secrets/pg_admin_dsn:ro" -e CRM_BACKUP_AGE_IDENTITY=/keys/drill-key.txt \
    -e CRM_BACKUP_ALLOWED_ROOTS=/backups -e CRM_RESTORE_ADMIN_DATABASE_URL_FILE=/secrets/pg_admin_dsn -e CRM_RESTORE_TARGET="$target" \
    -e CRM_RESTORE_EVIDENCE_DEST="/backups/$target-evidence" -e CRM_BACKUP_METRICS_FILE=/backups/crm_backup.prom \
    "$img" crm-backup restore "/backups/$bname" > ".state/$bname.restore.json" 2>&1; rcode=$?
  # Refusal: restoring again into the same (now existing) database.
  docker run --rm --network "$net" --user 65532:65532 -v /usr/bin/age:/usr/local/bin/age:ro -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" \
    -v "$here/.state/backups:/backups" -v "$here/.state/keys:/keys:ro" -v "$here/.secrets/dsn_backup:/secrets/dsn_backup:ro" -v "$here/.secrets/pg_admin_dsn:/secrets/pg_admin_dsn:ro" -e CRM_BACKUP_AGE_IDENTITY=/keys/drill-key.txt \
    -e CRM_BACKUP_ALLOWED_ROOTS=/backups -e CRM_RESTORE_ADMIN_DATABASE_URL_FILE=/secrets/pg_admin_dsn -e CRM_RESTORE_TARGET="$target" \
    -e CRM_RESTORE_EVIDENCE_DEST="/backups/$target-evidence-2" "$img" crm-backup restore "/backups/$bname" >/dev/null 2>&1; again=$?
  # Refusal: the live staging database can never be discarded.
  docker run --rm --network "$net" --user 65532:65532 -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" -v "$here/.secrets/dsn_backup:/secrets/dsn_backup:ro" -v "$here/.secrets/pg_admin_dsn:/secrets/pg_admin_dsn:ro" \
    -e CRM_RESTORE_ADMIN_DATABASE_URL_FILE=/secrets/pg_admin_dsn "$img" crm-backup discard crm_staging >/dev/null 2>&1; nodiscard=$?
  docker run --rm --network "$net" --user 65532:65532 -v "$artifact/bin/crm-backup:/usr/local/bin/crm-backup:ro" -v "$here/.secrets/dsn_backup:/secrets/dsn_backup:ro" -v "$here/.secrets/pg_admin_dsn:/secrets/pg_admin_dsn:ro" \
    -v "$here/.state/backups:/backups" -e CRM_BACKUP_ALLOWED_ROOTS=/backups -e CRM_RESTORE_EVIDENCE_DEST="/backups/$target-evidence" \
    -e CRM_RESTORE_ADMIN_DATABASE_URL_FILE=/secrets/pg_admin_dsn "$img" crm-backup discard "$target" >/dev/null 2>&1; dcode=$?
fi
status="$(jq -r '.status // "none"' ".state/$bname.restore.json" 2>/dev/null)"
recovery="$(jq -r '.measured_recovery_seconds // 0' ".state/$bname.restore.json" 2>/dev/null)"
leak="$(cat .state/$bname.*.json .state/$bname.backup.err 2>/dev/null | grep -c -f <(cut -d: -f3 .secrets/dsn_backup | cut -d@ -f1) || true)"
[ $bcode = 0 ] && [ $vcode = 0 ] && [ $rcode = 0 ] && [ "$status" = COMPLETE ] && [ "${again:-0}" = 2 ] && [ "${nodiscard:-0}" = 2 ] && [ $dcode = 0 ] && [ "$leak" = 0 ] && r=PASS || r=FAIL
rec 21 live_staging_backup_restore "encrypted backup of the live staging database + evidence volume; verify; restore into a NEW crm_restore_* (quarantined); re-restore refused; live DB discard refused; drill copy discarded" \
  "backup=$bcode verify=$vcode restore=$rcode status=$status recovery_s=$recovery rerestore_exit=${again:-x} discard_live_exit=${nodiscard:-x} discard_copy=$dcode credential_in_output=$leak" \
  $r "$(dt $s)" "synthetic, tiny dataset: recovery time is NOT an RTO; host age binary mounted (Dockerfile.backup build NOT VERIFIED in this sandbox)"

# 20. Safe environment cleanup (refusal only; destroy is run by the operator).
s=$(t0)
docker volume create crm-drill-sentinel >/dev/null
./stage.sh destroy >/dev/null 2>&1; code=$?
alive="$(docker volume inspect crm-drill-sentinel >/dev/null 2>&1 && echo kept || echo removed)"
docker volume rm crm-drill-sentinel >/dev/null
[ $code = 2 ] && [ "$alive" = kept ] && [ "$(health api)" = healthy ] && r=PASS || r=FAIL
rec 20 safe_cleanup "destroy without CRM_STAGING_DESTROY=crm-staging is refused; unrelated volumes are never touched" "destroy_exit=$code sentinel=$alive api=$(health api)" $r "$(dt $s)" "destroy removes only this project's disposable-labelled volumes"

echo "drill evidence: $here/$out"
