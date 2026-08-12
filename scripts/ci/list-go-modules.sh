#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
grep '^\s*\./' "$root/go.work" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
