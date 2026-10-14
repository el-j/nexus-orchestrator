#!/usr/bin/env bash
# Local-only mutation testing. Deliberately NOT wired into any GitHub workflow: a full run takes
# many minutes per package and would burn workflow minutes.
#
#   scripts/mutation.sh                 # Go core packages + every TypeScript package
#   scripts/mutation.sh go              # Go only (gremlins)
#   scripts/mutation.sh ts              # frontend, vscode-extension, github-action (StrykerJS)
#   scripts/mutation.sh go internal/core/services   # Go only, chosen package directories
#
# Go: gremlins (go install github.com/go-gremlins/gremlins/cmd/gremlins@latest).
# TS: StrykerJS, configured per package in stryker.config.json; its `break` threshold fails the run.
set -euo pipefail
cd "$(dirname "$0")/.."

scope="${1:-all}"
shift || true
go_paths=("$@")
if [ "${#go_paths[@]}" -eq 0 ]; then
  go_paths=(internal/core/domain internal/core/services)
fi
efficacy="${MUTATION_MIN_EFFICACY:-70}"

run_go() {
  command -v gremlins >/dev/null || {
    echo "gremlins not found: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest" >&2
    exit 1
  }
  for path in "${go_paths[@]}"; do
    echo "==> gremlins ./$path (min efficacy ${efficacy}%)"
    CGO_ENABLED=1 CGO_CFLAGS="${CGO_CFLAGS:--DSQLITE_ENABLE_FTS5}" \
      gremlins unleash "./$path" --threshold-efficacy "$efficacy"
  done
}

run_ts() {
  for pkg in frontend vscode-extension github-action; do
    echo "==> stryker $pkg"
    (cd "$pkg" && npm run test:mutation)
  done
}

case "$scope" in
  go) run_go ;;
  ts) run_ts ;;
  all) run_go; run_ts ;;
  *) echo "usage: $0 [go|ts|all] [go package dirs...]" >&2; exit 2 ;;
esac
