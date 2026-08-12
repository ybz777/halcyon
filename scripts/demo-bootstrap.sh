#!/usr/bin/env bash
# Unix wrapper — runs the PowerShell bootstrap on Windows or invokes equivalent steps.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if command -v pwsh >/dev/null 2>&1; then
  exec pwsh -File "$ROOT/scripts/demo-bootstrap.ps1" "$@"
fi
echo "Install PowerShell (pwsh) or run scripts/demo-bootstrap.ps1 on Windows." >&2
exit 1
