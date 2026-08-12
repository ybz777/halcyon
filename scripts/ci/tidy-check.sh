#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

go work sync
while IFS= read -r mod; do
  (cd "$mod" && go mod tidy)
done < <("$root/scripts/ci/list-go-modules.sh")

if ! git diff --quiet go.work go.work.sum $(git ls-files '**/go.mod' '**/go.sum'); then
  echo "go.mod or go.sum is out of date. Run: make tidy"
  git diff go.work go.work.sum $(git ls-files '**/go.mod' '**/go.sum') || true
  exit 1
fi

echo "go mod tidy check passed"
