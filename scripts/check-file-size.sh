#!/usr/bin/env bash
# Fails when a hand-written file grows past a limit, so files keep a single
# responsibility (#45): MAX_LINES (default 500) for source, MAX_TEST_LINES
# (default 800) for tests. Generated bindings, vendored and built output are exempt.
set -euo pipefail

max="${MAX_LINES:-500}"
max_test="${MAX_TEST_LINES:-800}"
cd "$(git rev-parse --show-toplevel)"

status=0
check() { # check LIMIT FILE...: print every file over LIMIT lines
  local limit="$1" f n
  shift
  for f in "$@"; do
    n=$(wc -l <"$f")
    if [ "$n" -gt "$limit" ]; then
      echo "$f: $n lines (limit $limit)"
      status=1
    fi
  done
}

files() { git ls-files --cached --others --exclude-standard -- '*.go' '*.ts' '*.vue'; }
is_test='_test\.go$|\.spec\.ts$|\.test\.ts$|/test/|testutil/'
skip='node_modules/|/dist/|^dist/|wailsjs/|\.d\.ts$'

# shellcheck disable=SC2046
check "$max" $(files | grep -vE "$skip" | grep -vE "$is_test" || true)
# shellcheck disable=SC2046
check "$max_test" $(files | grep -vE "$skip" | grep -E "$is_test" || true)

[ "$status" -eq 0 ] && echo "all source files are within $max lines, tests within $max_test"
exit "$status"
