#!/usr/bin/env bash
# Post-deployment smoke test — read-only, safe for local/staging.
set -euo pipefail

BASE_GATEWAY="${1:-http://localhost:8088}"
declare -a ENDPOINTS=(
  "http://localhost:8080/api/v1/health"
  "http://localhost:8080/api/v1/ready"
  "http://localhost:8081/api/v1/health"
  "http://localhost:8081/api/v1/ready"
  "http://localhost:8082/api/v1/health"
  "http://localhost:8083/api/v1/health"
  "http://localhost:8084/api/v1/health"
  "http://localhost:8084/api/v1/ready"
  "http://localhost:8085/api/v1/health"
  "http://localhost:8085/api/v1/ready"
  "http://localhost:8086/api/v1/health"
  "http://localhost:8086/api/v1/ready"
  "${BASE_GATEWAY}/api/v1/health"
  "${BASE_GATEWAY}/api/v1/ready"
)

echo "Halcyon deployment smoke test"
echo "Environment: local/staging only"
echo ""

failed=0
for url in "${ENDPOINTS[@]}"; do
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" || echo "000")
  if [[ "$code" =~ ^2 ]]; then
    echo "OK  $code  $url"
  else
    echo "FAIL $code  $url"
    failed=$((failed + 1))
  fi
done

if [ "$failed" -gt 0 ]; then
  echo ""
  echo "$failed endpoint(s) failed"
  exit 1
fi

echo ""
echo "All smoke checks passed."
