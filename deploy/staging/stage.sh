#!/usr/bin/env bash
# R3 DISPOSABLE staging driver (local rehearsal only). It never targets a
# production host, database, volume, DNS name or credential.
#
#   stage.sh init-secrets                  generate per-environment secrets (.secrets/, 0400)
#   stage.sh preflight <artifact-dir>      validate artifact, config, secrets and isolation
#   stage.sh up <artifact-dir>             build images from the verified artifact and start
#   stage.sh verify                        health / readiness / isolation checks (JSON evidence)
#   stage.sh rollback                      restart the previous artifact's images (code only)
#   stage.sh evidence                      write the deployment evidence manifest
#   stage.sh down                          stop containers (volumes kept)
#   stage.sh destroy                       remove THIS project's containers AND volumes
#                                          (requires CRM_STAGING_DESTROY=crm-staging)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"
project=crm-staging
state="$here/.state"
secrets="$here/.secrets"
compose() { docker compose -p "$project" -f "$here/compose.yaml" "$@"; }
die() { echo "stage: $*" >&2; exit 2; }
now() { date -u +%FT%TZ; }

# Anything that looks like a production endpoint must never appear in the
# runnable configuration (defence in depth; nothing here uses one).
forbidden='(sn-crm\.ir|crm\.[a-z0-9-]+\.(ir|com)|wp-admin|admin-ajax|woocommerce|biawin\.(ir|com)|amazonaws\.com)'

artifact_ok() {
  local a="$1"
  [ -f "$a/ELIGIBLE" ] || die "artifact $a is not marked ELIGIBLE (run scripts/release/verify.sh --mark-eligible)"
  ( cd "$a" && sha256sum --quiet --strict -c SHA256SUMS ) || die "artifact checksums do not match"
  grep -q '"dirty": false' "$a/manifest.json" || die "a dirty artifact is never deployed"
}

commit_of() { sed -n 's/.*"commit": "\([0-9a-f]\{40\}\)".*/\1/p' "$1/manifest.json"; }

cmd="${1:-}"; shift || true
case "$cmd" in
init-secrets)
  [ ! -e "$secrets" ] || die ".secrets exists (never overwritten; destroy first)"
  install -d -m 0700 "$secrets"
  rnd() { openssl rand -hex 24; }
  admin="$(rnd)"; printf '%s' "$admin" > "$secrets/pg_admin_password"
  printf 'postgres://crm_staging_admin:%s@postgres:5432/crm_staging?sslmode=disable' "$admin" > "$secrets/pg_admin_dsn"
  : > "$secrets/login_passwords.sql"
  for m in platform_featureflags payments integration cases; do
    pw="$(rnd)"; role="crm_stg_${m}"
    cat >> "$secrets/login_passwords.sql" <<SQL
DO \$\$ BEGIN CREATE ROLE ${role} LOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS; EXCEPTION WHEN duplicate_object THEN NULL; END \$\$;
ALTER ROLE ${role} PASSWORD '${pw}';
GRANT crm_owner_${m} TO ${role};
SQL
    short="$m"; [ "$m" = platform_featureflags ] && short=flags
    printf 'postgres://%s:%s@postgres:5432/crm_staging?sslmode=disable' "$role" "$pw" > "$secrets/dsn_${short}"
  done
  bpw="$(rnd)"
  cat >> "$secrets/login_passwords.sql" <<SQL
DO \$\$ BEGIN CREATE ROLE crm_stg_backup LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS; EXCEPTION WHEN duplicate_object THEN NULL; END \$\$;
ALTER ROLE crm_stg_backup INHERIT PASSWORD '${bpw}';
GRANT pg_read_all_data TO crm_stg_backup WITH INHERIT TRUE;
SQL
  printf 'postgres://crm_stg_backup:%s@postgres:5432/crm_staging?sslmode=disable' "$bpw" > "$secrets/dsn_backup"
  openssl req -x509 -newkey rsa:2048 -nodes -days 30 -subj "/CN=crm-staging.local" -addext "subjectAltName=DNS:crm-staging.local,IP:127.0.0.1" \
    -keyout "$secrets/tls_key.pem" -out "$secrets/tls_cert.pem" >/dev/null 2>&1
  # Least-privilege file access: app secrets readable only by the app uid
  # (65532), the database password only by postgres (70); TLS and the
  # provisioning SQL only by root.
  chmod 0400 "$secrets"/*
  chown 65532:65532 "$secrets"/dsn_* "$secrets/pg_admin_dsn"
  chown 70:70 "$secrets/pg_admin_password"
  echo "stage: secrets generated in .secrets (0400; values never printed)"
  ;;
preflight)
  a="$(cd "${1:?artifact dir}" && pwd)"
  artifact_ok "$a"
  for f in pg_admin_password pg_admin_dsn dsn_flags dsn_payments dsn_integration dsn_cases login_passwords.sql tls_cert.pem tls_key.pem; do
    [ -s "$secrets/$f" ] || die "missing secret $f (run init-secrets)"
    [ "$(stat -c %a "$secrets/$f")" = "400" ] || die "secret $f must be mode 0400"
  done
  CRM_RELEASE=x compose config -q || die "compose configuration invalid"
  cfg="$(CRM_RELEASE=x compose config --format json)"
  if grep -Eiq "$forbidden" <<< "$cfg" || grep -Eiq "$forbidden" "$here/nginx/nginx.conf"; then die "a production-looking endpoint appears in the staging configuration"; fi
  [ "$(jq -r '.networks.internal.internal' <<< "$cfg")" = "true" ] || die "the internal network must be internal: true"
  [ "$(jq -r '[.services | to_entries[] | select(.value.ports != null) | .key] | join(",")' <<< "$cfg")" = "web" ] || die "only the web proxy may publish a port"
  [ "$(jq -r '[.services.web.ports[] | select(.host_ip != "127.0.0.1")] | length' <<< "$cfg")" = "0" ] || die "published ports must bind 127.0.0.1 only"
  [ "$(jq -r '[.services[] | .networks // {} | keys[] | select(. == "edge")] | length' <<< "$cfg")" = "1" ] || die "only the web proxy may join the edge network"
  if jq -r '.services[].environment // {} | to_entries[] | select((.key | test("PASSWORD$|_DSN$|DATABASE_URL$")) and (.key | test("_FILE$") | not)) | .key' <<< "$cfg" | grep -q .; then
    die "a credential is configured as a plain environment variable"
  fi
  echo "stage: preflight OK (artifact $(commit_of "$a"))"
  ;;
up)
  a="$(cd "${1:?artifact dir}" && pwd)"
  "$0" preflight "$a"
  c="$(commit_of "$a")"; tag="${c:0:12}"
  docker build -q -f "$here/../docker/Dockerfile.app" --build-arg CRM_COMMIT="$c" -t "crm-app:$tag" "$a" >/dev/null
  docker build -q -f "$here/../docker/Dockerfile.web" --build-arg CRM_COMMIT="$c" -t "crm-web:$tag" "$a" >/dev/null
  install -d -m 0700 "$state"
  if [ -f "$state/current" ] && [ "$(cat "$state/current")" != "$tag" ]; then cp "$state/current" "$state/previous"; fi
  echo "$tag" > "$state/current"
  start=$(date +%s)
  CRM_RELEASE="$tag" compose up -d --wait --wait-timeout 180 || { echo "stage: deployment FAILED health checks (release $tag)" >&2; CRM_RELEASE="$tag" compose ps >&2; exit 1; }
  echo "{\"event\":\"up\",\"release\":\"$tag\",\"commit\":\"$c\",\"seconds\":$(( $(date +%s) - start )),\"at\":\"$(now)\"}" >> "$state/events.jsonl"
  echo "stage: release $tag healthy"
  ;;
verify)
  tag="$(cat "$state/current")"
  out="$(CRM_RELEASE="$tag" compose ps --format json)"
  unhealthy="$(jq -s -r '[.[] | (if type=="array" then .[] else . end) | select(.Service=="api" or .Service=="worker" or .Service=="web" or .Service=="postgres") | select(.Health!="healthy")] | length' <<< "$out")"
  ready="$(curl -sk -o /dev/null -w '%{http_code}' https://127.0.0.1:${CRM_STAGING_HTTPS_PORT:-8443}/health/ready || true)"
  hdr="$(curl -skI https://127.0.0.1:${CRM_STAGING_HTTPS_PORT:-8443}/ | tr -d '\r' | grep -ci 'strict-transport-security' || true)"
  egress="blocked"
  if docker run --rm --network "${project}_internal" alpine:3.20 wget -q -T 3 -O /dev/null http://1.1.1.1 >/dev/null 2>&1; then egress="OPEN"; fi
  wstate="$(docker run --rm --network "${project}_internal" alpine:3.20 wget -q -T 3 -O - http://worker:9091/health/ready 2>/dev/null || true)"
  paused="$(grep -o '"state":"paused"' <<< "$wstate" | wc -l | tr -d ' ')"
  blocked="$(grep -o '"state":"blocked"' <<< "$wstate" | wc -l | tr -d ' ')"
  amet="$(docker run --rm --network "${project}_internal" alpine:3.20 wget -q -T 3 -O - http://api:9090/metrics 2>/dev/null | grep -c '^crm_api_requests_total' || true)"
  printf '{"release":"%s","unhealthy_services":%s,"tls_ready_status":"%s","hsts":%s,"internal_egress":"%s","worker_jobs_paused":%s,"worker_jobs_blocked":%s,"api_metric_series":%s,"at":"%s"}\n' \
    "$tag" "$unhealthy" "$ready" "$hdr" "$egress" "$paused" "$blocked" "${amet:-0}" "$(now)" | tee -a "$state/events.jsonl"
  [ "$unhealthy" = "0" ] && [ "$ready" = "200" ] && [ "$egress" = "blocked" ] || exit 1
  ;;
rollback)
  [ -f "$state/previous" ] || die "no previous release recorded"
  prev="$(cat "$state/previous")"; cur="$(cat "$state/current")"
  docker image inspect "crm-app:$prev" "crm-web:$prev" >/dev/null || die "previous images crm-*:$prev are not retained"
  echo "stage: rolling back executables $cur → $prev. Database state, committed business effects and evidence are NOT reverted." >&2
  start=$(date +%s)
  CRM_RELEASE="$prev" compose up -d --wait --wait-timeout 180 api worker web || { echo "stage: rollback FAILED health checks" >&2; exit 1; }
  echo "$prev" > "$state/current"; echo "$cur" > "$state/previous"
  echo "{\"event\":\"rollback\",\"from\":\"$cur\",\"to\":\"$prev\",\"seconds\":$(( $(date +%s) - start )),\"at\":\"$(now)\"}" >> "$state/events.jsonl"
  echo "stage: rolled back to $prev"
  ;;
evidence)
  tag="$(cat "$state/current")"
  {
    printf '{"release":"%s","generated_at":"%s",' "$tag" "$(now)"
    printf '"images":{"app":"%s","web":"%s"},' "$(docker image inspect --format '{{.Id}}' "crm-app:$tag")" "$(docker image inspect --format '{{.Id}}' "crm-web:$tag")"
    printf '"image_revision":"%s",' "$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "crm-app:$tag")"
    printf '"events":['; paste -sd, "$state/events.jsonl"; printf ']}\n'
  } > "$state/deployment-evidence-$tag.json"
  echo "$state/deployment-evidence-$tag.json"
  ;;
down)
  tag="$(cat "$state/current" 2>/dev/null || echo x)"
  CRM_RELEASE="$tag" compose down
  ;;
destroy)
  [ "${CRM_STAGING_DESTROY:-}" = "$project" ] || die "set CRM_STAGING_DESTROY=$project to remove this project's containers and volumes"
  for v in $(docker volume ls -q --filter "label=com.docker.compose.project=$project"); do
    [ "$(docker volume inspect --format '{{index .Labels "crm.disposable"}}' "$v")" = "true" ] || die "volume $v is not marked disposable; refusing"
  done
  tag="$(cat "$state/current" 2>/dev/null || echo x)"
  CRM_RELEASE="$tag" compose down --volumes --remove-orphans
  rm -rf "$state" "$secrets"
  echo "stage: disposable staging project $project removed"
  ;;
*) sed -n '2,15p' "$0"; exit 2 ;;
esac
