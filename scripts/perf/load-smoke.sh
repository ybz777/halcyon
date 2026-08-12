#!/usr/bin/env bash
# Local load smoke test — not a production capacity measurement.
set -euo pipefail

URL="${1:-http://localhost:8080/health}"
CONCURRENCY="${2:-20}"
REQUESTS="${3:-500}"

echo "Load smoke: ${REQUESTS} requests, concurrency ${CONCURRENCY}, target ${URL}"
echo "Environment: local test only"
echo ""

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

per_worker=$(( (REQUESTS + CONCURRENCY - 1) / CONCURRENCY ))
start=$(date +%s.%N)

for ((c=0; c<CONCURRENCY; c++)); do
  (
    outfile="${tmpdir}/worker-${c}.log"
    for ((i=0; i<per_worker; i++)); do
      code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$URL" || echo "000")
      echo "$code" >> "$outfile"
    done
  ) &
done
wait

end=$(date +%s.%N)
elapsed=$(echo "$end - $start" | bc)
total=$(cat "${tmpdir}"/*.log 2>/dev/null | wc -l | tr -d ' ')
ok=$(grep -E '^2[0-9]{2}$' "${tmpdir}"/*.log 2>/dev/null | wc -l | tr -d ' ')
rps=$(echo "scale=2; $total / $elapsed" | bc)

echo "Completed: ${total} requests in ${elapsed}s"
echo "Throughput: ${rps} req/s"
echo "2xx responses: ${ok} / ${total}"
echo "Status distribution:"
cat "${tmpdir}"/*.log 2>/dev/null | sort | uniq -c | sort -k2
