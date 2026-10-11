#!/usr/bin/env bash
# Fails when a hand-written source file grows past MAX_LINES (default 500), so
# files keep a single responsibility (#45). Tests, generated bindings, vendored
# and built output are exempt.
set -euo pipefail

max="${MAX_LINES:-500}"
cd "$(git rev-parse --show-toplevel)"

status=0
while IFS= read -r f; do
  n=$(wc -l <"$f")
  if [ "$n" -gt "$max" ]; then
    echo "$f: $n lines (limit $max)"
    status=1
  fi
done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.ts' '*.vue' |
  grep -vE 'node_modules/|/dist/|^dist/|wailsjs/|_test\.go$|\.spec\.ts$|\.test\.ts$|/test/|testutil/|\.d\.ts$')

[ "$status" -eq 0 ] && echo "all source files are within $max lines"
exit "$status"
