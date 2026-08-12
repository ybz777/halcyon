#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

dirs=()
while IFS= read -r mod; do
  dirs+=("$mod")
done < <("$root/scripts/ci/list-go-modules.sh")

unformatted="$(gofmt -l "${dirs[@]}")"
if [ -n "$unformatted" ]; then
  echo "The following files are not gofmt-formatted:"
  echo "$unformatted"
  exit 1
fi

echo "gofmt check passed"
