#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

fail=0
while IFS= read -r dir; do
  [ -d "$dir" ] || continue
  name="$(basename "$(dirname "$dir")")/$(basename "$dir")"
  declare -A versions=()
  for up in "$dir"/*.up.sql; do
    [ -f "$up" ] || continue
    base="$(basename "$up")"
    version="${base%%_*}"
    if [ -n "${versions[$version]:-}" ]; then
      echo "duplicate migration version $version in $name"
      fail=1
    fi
    versions[$version]=1
    down="${dir}/${version}_$(basename "$up" | sed 's/\.up\.sql$//').down.sql"
    down_alt="${up%.up.sql}.down.sql"
    if [ ! -f "$down_alt" ]; then
      echo "missing down migration for $up"
      fail=1
    fi
    if ! grep -qE '(CREATE|ALTER|DROP|INSERT|UPDATE|DELETE)' "$up"; then
      echo "suspicious empty migration: $up"
      fail=1
    fi
  done
done < <(find services pkg/platform/cmd -type d -name migrations 2>/dev/null)

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "migration validation passed"
