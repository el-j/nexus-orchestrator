#!/usr/bin/env bash
# Merged Go statement coverage across every package, with a minimum gate.
#
#   scripts/go-coverage.sh            # report only
#   MIN_COVERAGE=95 scripts/go-coverage.sh   # fail below 95%
#
# Each package's tests also exercise code in other packages (-coverpkg), and a
# block counts as covered when ANY test binary executed it. Excluded from the
# denominator: internal/testutil (test doubles) and the repository-root main.go
# (its only uncovered code is the call into the Wails GUI runtime).
set -euo pipefail

profile="$(mktemp)"
log="$(mktemp)"
trap 'rm -f "$profile" "$log"' EXIT
# A failing test must say which one: the output is kept and its failures are printed.
default_ldflags="-lm"
if [ "$(uname -s 2>/dev/null)" = "Darwin" ]; then
  default_ldflags="-lm -Wl,-no_warn_duplicate_libraries"
fi
if ! CGO_ENABLED=1 CGO_CFLAGS="${CGO_CFLAGS:--DSQLITE_ENABLE_FTS5}" CGO_LDFLAGS="${CGO_LDFLAGS:-$default_ldflags}" \
  go test -count=1 -coverpkg=./... -coverprofile="$profile" ./... >"$log" 2>&1; then
  grep -E '^(--- FAIL|FAIL|panic:)|_test\.go:[0-9]+:' "$log" | head -40 >&2
  echo "go test failed (see above)" >&2
  exit 1
fi

python3 - "$profile" "${MIN_COVERAGE:-0}" <<'PY'
import re, sys, collections
profile, minimum = sys.argv[1], float(sys.argv[2])
blocks = {}
for line in open(profile).read().splitlines()[1:]:
    m = re.match(r'(.+?:\d+\.\d+,\d+\.\d+) (\d+) (\d+)', line)
    key, stmts, count = m.group(1), int(m.group(2)), int(m.group(3))
    prev = blocks.get(key, (stmts, 0))
    blocks[key] = (stmts, max(prev[1], count))

def excluded(path):
    return '/internal/testutil/' in path or path.endswith('nexus-orchestrator/main.go')

total = covered = 0
missed = collections.Counter()
for key, (stmts, count) in blocks.items():
    path = key.split(':')[0]
    if excluded(path):
        continue
    total += stmts
    if count > 0:
        covered += stmts
    else:
        missed[path] += stmts
pct = 100.0 * covered / total
print(f"Go merged statement coverage: {pct:.1f}% ({total - covered} of {total} statements uncovered)")
for path, n in missed.most_common(10):
    print(f"  {n:4d} uncovered  {path.replace('nexus-orchestrator/', '')}")
if pct + 1e-9 < minimum:
    print(f"FAIL: coverage {pct:.1f}% is below the required {minimum:.1f}%")
    sys.exit(1)
PY
