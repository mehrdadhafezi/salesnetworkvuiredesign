#!/usr/bin/env bash
# R3 release build: deterministic binaries + web bundle, checksums and a
# provenance manifest for ONE commit. It builds; it never deploys.
#
#   scripts/release/build.sh [OUT_DIR]      default OUT_DIR: dist/<commit>
#
# Refuses a dirty working tree (provenance would be ambiguous) unless
# CRM_RELEASE_ALLOW_DIRTY=1 (then marked "dirty": true, never eligible).
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

commit="$(git rev-parse HEAD)"
dirty=false
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  if [ "${CRM_RELEASE_ALLOW_DIRTY:-}" != "1" ]; then
    echo "build: refusing a dirty working tree (commit or stash first)" >&2; exit 2
  fi
  dirty=true
fi
out="${1:-dist/$commit}"
if [ -e "$out" ]; then echo "build: $out already exists (never overwritten)" >&2; exit 2; fi
commit_time="$(git show -s --format=%ct "$commit")"
mkdir -p "$out/bin"

# Dependency integrity: module hashes against go.sum; npm lockfile install.
go mod verify >/dev/null
gomod_sha="$(sha256sum go.sum | cut -d' ' -f1)"

# Deterministic Go builds: static, trimmed paths, no build id, VCS stamping
# (vcs.revision / vcs.time come from the commit, so they are reproducible).
export CGO_ENABLED=0 GOOS="${GOOS:-linux}" GOARCH="${GOARCH:-amd64}" GOFLAGS=-mod=readonly
cmds=(crm-api crm-worker crm-migrate crm-backup crm-healthcheck crm-legacy-import crm-reporting)
for c in "${cmds[@]}"; do
  go build -trimpath -buildvcs=true -ldflags='-s -w -buildid=' -o "$out/bin/$c" "./cmd/$c"
done

web_sha="none"; lock_sha="none"
if [ "${CRM_RELEASE_SKIP_WEB:-}" != "1" ]; then
  lock_sha="$(sha256sum web/package-lock.json | cut -d' ' -f1)"
  npm ci --prefix web --no-audit --no-fund --ignore-scripts >/dev/null
  npm --prefix web run build >/dev/null
  # Reproducible archive: sorted, fixed mtime / owner, gzip without name+time.
  tar --sort=name --mtime="@$commit_time" --owner=0 --group=0 --numeric-owner -C web/dist -cf - . | gzip -n -9 > "$out/web-dist.tar.gz"
  web_sha="$(sha256sum "$out/web-dist.tar.gz" | cut -d' ' -f1)"
fi

( cd "$out" && find . -type f ! -name SHA256SUMS ! -name manifest.json -printf '%P\n' | LC_ALL=C sort | xargs sha256sum > SHA256SUMS )

go_version="$(go env GOVERSION)"
node_version="$(node --version 2>/dev/null || echo none)"
{
  printf '{\n'
  printf '  "format": 1,\n'
  printf '  "commit": "%s",\n' "$commit"
  printf '  "commit_time_unix": %s,\n' "$commit_time"
  printf '  "dirty": %s,\n' "$dirty"
  printf '  "eligible": false,\n'
  printf '  "eligibility": "set only by scripts/release/verify.sh after every gate passed",\n'
  printf '  "target": "%s/%s",\n' "$GOOS" "$GOARCH"
  printf '  "go_version": "%s",\n' "$go_version"
  printf '  "node_version": "%s",\n' "$node_version"
  printf '  "go_sum_sha256": "%s",\n' "$gomod_sha"
  printf '  "web_lock_sha256": "%s",\n' "$lock_sha"
  printf '  "web_bundle_sha256": "%s",\n' "$web_sha"
  printf '  "checksums_sha256": "%s",\n' "$(sha256sum "$out/SHA256SUMS" | cut -d' ' -f1)"
  printf '  "binaries": ['
  sep=""
  for c in "${cmds[@]}"; do printf '%s"%s"' "$sep" "$c"; sep=", "; done
  printf ']\n}\n'
} > "$out/manifest.json"
echo "$out"
