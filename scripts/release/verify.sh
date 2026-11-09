#!/usr/bin/env bash
# R3 artifact verification: checksums, provenance and embedded VCS revision.
#
#   scripts/release/verify.sh <artifact-dir> [--mark-eligible]
#
# --mark-eligible writes ELIGIBLE only when every check passes and the
# artifact is clean; a failed check never produces it. Eligible means
# "may be used for a STAGING rehearsal"; nothing here deploys anything.
set -euo pipefail
dir="${1:?usage: verify.sh <artifact-dir> [--mark-eligible]}"
mark="${2:-}"
fail() { echo "verify: FAILED: $*" >&2; exit 1; }
[ -f "$dir/manifest.json" ] && [ -f "$dir/SHA256SUMS" ] || fail "manifest or SHA256SUMS missing"
[ ! -e "$dir/ELIGIBLE" ] || fail "ELIGIBLE already present (artifacts are verified once, never re-marked)"
( cd "$dir" && sha256sum --quiet --strict -c SHA256SUMS ) || fail "checksum mismatch"
want="$(sha256sum "$dir/SHA256SUMS" | cut -d' ' -f1)"
grep -q "\"checksums_sha256\": \"$want\"" "$dir/manifest.json" || fail "manifest does not bind SHA256SUMS"
commit="$(sed -n 's/.*"commit": "\([0-9a-f]\{40\}\)".*/\1/p' "$dir/manifest.json")"
[ -n "$commit" ] || fail "no commit in manifest"
for b in "$dir"/bin/*; do
  rev="$(go version -m "$b" | awk '$1=="build" && $2 ~ /^vcs.revision=/ {sub("vcs.revision=","",$2); print $2}')"
  [ "$rev" = "$commit" ] || fail "$(basename "$b") was built from ${rev:-unknown}, manifest says $commit"
  mod="$(go version -m "$b" | awk '$1=="build" && $2 ~ /^vcs.modified=/ {sub("vcs.modified=","",$2); print $2}')"
  if grep -q '"dirty": false' "$dir/manifest.json" && [ "$mod" != "false" ]; then fail "$(basename "$b") built from a modified tree"; fi
done
echo "verify: OK commit=$commit"
if [ "$mark" = "--mark-eligible" ]; then
  grep -q '"dirty": false' "$dir/manifest.json" || fail "a dirty build is never eligible"
  printf 'eligible-for=staging-rehearsal\ncommit=%s\nchecksums_sha256=%s\nverified_at=%s\n' "$commit" "$want" "$(date -u +%FT%TZ)" > "$dir/ELIGIBLE"
  echo "verify: marked eligible for a STAGING rehearsal only"
fi
