#!/usr/bin/env bash
# Local performance check, not part of any workflow: runs the Go benchmarks and fails when one is more
# than BENCH_FACTOR (default 3) times slower than the baseline in scripts/bench-baseline.txt.
#
#   scripts/bench.sh            # compare with the baseline
#   scripts/bench.sh --record   # measure on this machine and write the baseline
set -euo pipefail
cd "$(dirname "$0")/.."

factor="${BENCH_FACTOR:-3}"
baseline="scripts/bench-baseline.txt"
export CGO_CFLAGS="${CGO_CFLAGS:--DSQLITE_ENABLE_FTS5}"

measured="$(go test -run '^$' -bench . -benchtime 200x ./internal/core/services ./internal/adapters/outbound/repo_sqlite |
  awk '/^Benchmark/ { sub(/-[0-9]+$/, "", $1); print $1, $3 }')"

if [ "${1:-}" = "--record" ]; then
  printf '%s\n' "$measured" >"$baseline"
  echo "recorded $(wc -l <"$baseline" | tr -d ' ') benchmarks in $baseline"
  exit 0
fi

status=0
while read -r name ns; do
  base="$(awk -v n="$name" '$1 == n { print $2 }' "$baseline")"
  if [ -z "$base" ]; then
    echo "$name: no baseline (run scripts/bench.sh --record)"
    status=1
  elif awk -v ns="$ns" -v base="$base" -v f="$factor" 'BEGIN { exit !(ns > base * f) }'; then
    echo "$name: $ns ns/op is more than ${factor}x the baseline $base"
    status=1
  else
    echo "$name: $ns ns/op (baseline $base)"
  fi
done <<<"$measured"
exit "$status"
