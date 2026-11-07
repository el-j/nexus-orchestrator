#!/usr/bin/env bash
# Local-only mutation testing. Deliberately NOT wired into any GitHub workflow: a full run takes
# many minutes per package and would burn workflow minutes.
#
#   scripts/mutation.sh                 # every Go package with logic + every TypeScript package (frontend, docs, extension, action)
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
  # Every package that holds logic: the core, then the adapters. A full run takes hours; choose
  # directories to run less (see below).
  go_paths=(
    internal/core/domain
    internal/core/services
    internal/adapters/inbound/cli
    internal/adapters/inbound/httpapi
    internal/adapters/inbound/httpguard
    internal/adapters/inbound/mcp
    internal/adapters/outbound/activity_claude
    internal/adapters/outbound/activity_network
    internal/adapters/outbound/cmd_runner
    internal/adapters/outbound/fs_watcher
    internal/adapters/outbound/fs_writer
    internal/adapters/outbound/httpapi_client
    internal/adapters/outbound/httperr
    internal/adapters/outbound/llm_anthropic
    internal/adapters/outbound/llm_cli
    internal/adapters/outbound/llm_gemini
    internal/adapters/outbound/llm_lmstudio
    internal/adapters/outbound/llm_ollama
    internal/adapters/outbound/llm_openaicompat
    internal/adapters/outbound/repo_sqlite
    internal/adapters/outbound/sse
    internal/adapters/outbound/sys_scanner
    internal/adapters/outbound/usagebox
  )
fi
efficacy="${MUTATION_MIN_EFFICACY:-70}"
timeout_coeff="${MUTATION_TIMEOUT_COEFF:-3}"

run_go() {
  command -v gremlins >/dev/null || {
    echo "gremlins not found: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest" >&2
    exit 1
  }
  for path in "${go_paths[@]}"; do
    echo "==> gremlins ./$path (min efficacy ${efficacy}%, timeout-coefficient ${timeout_coeff})"
    CGO_ENABLED=1 CGO_CFLAGS="${CGO_CFLAGS:--DSQLITE_ENABLE_FTS5}" \
      gremlins unleash "./$path" \
      --integration \
      --timeout-coefficient "$timeout_coeff" \
      --threshold-efficacy "$efficacy"
  done
}

run_ts() {
  for pkg in frontend docs vscode-extension github-action; do
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
