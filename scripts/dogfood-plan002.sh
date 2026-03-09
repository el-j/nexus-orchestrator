#!/usr/bin/env bash
# nexusOrchestrator PLAN-002 dogfood runner
# Builds daemon + nexus-submit, starts daemon, submits PLAN-002 implementation tasks.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NEXUS_ADDR="${NEXUS_ADDR:-http://127.0.0.1:9999}"
DB_PATH="${NEXUS_DB_PATH:-/tmp/nexus-dogfood-$(date +%s).db}"
DAEMON_BINARY="/tmp/nexus-daemon-dogfood"
SUBMIT_BINARY="/tmp/nexus-submit-dogfood"

cd "$PROJECT_ROOT"

echo "=== nexusOrchestrator PLAN-002 Dogfood Runner ==="
echo "Project: $PROJECT_ROOT"
echo "DB:      $DB_PATH"
echo "Addr:    $NEXUS_ADDR"
echo ""

# 1. Build binaries
echo "[1/4] Building nexus-daemon..."
CGO_ENABLED=1 go build -o "$DAEMON_BINARY" ./cmd/nexus-daemon

echo "[2/4] Building nexus-submit..."
CGO_ENABLED=1 go build -o "$SUBMIT_BINARY" ./cmd/nexus-submit

# 2. Start daemon
echo "[3/4] Starting daemon..."
NEXUS_DB_PATH="$DB_PATH" "$DAEMON_BINARY" &
DAEMON_PID=$!
trap 'echo ""; echo "Stopping daemon (PID $DAEMON_PID)..."; kill "$DAEMON_PID" 2>/dev/null || true; rm -f "$DB_PATH"' EXIT

# Wait for daemon to be ready (up to 10 seconds)
echo -n "Waiting for daemon..."
READY=0
for i in $(seq 1 20); do
    if curl -sf "$NEXUS_ADDR/api/health" > /dev/null 2>&1; then
        READY=1
        break
    fi
    sleep 0.5
    echo -n "."
done
echo ""

if [[ "$READY" -ne 1 ]]; then
    echo "ERROR: Daemon did not become ready in 10 seconds." >&2
    exit 1
fi
echo "Daemon ready ✓"

echo ""
echo "Dashboard: $NEXUS_ADDR/ui"
echo ""

# 3. Submit PLAN-002 tasks
echo "[4/4] Submitting PLAN-002 tasks..."

# Map: task_id -> "target_file|context_files"
submit_task() {
    local TASK_ID="$1"
    local TARGET="$2"
    local CONTEXT="$3"
    local TASK_FILE=".claude/tasks/${TASK_ID}.md"

    if [[ ! -f "$TASK_FILE" ]]; then
        echo "  SKIP $TASK_ID — $TASK_FILE not found"
        return
    fi

    local RESULT
    RESULT=$("$SUBMIT_BINARY" \
        --task-file "$TASK_FILE" \
        --project "$PROJECT_ROOT" \
        --target "$TARGET" \
        --context "$CONTEXT" \
        --addr "$NEXUS_ADDR" 2>&1) || true

    local TASK_UUID
    TASK_UUID=$(echo "$RESULT" | grep 'task_id=' | sed 's/.*task_id=//' | awk '{print $1}')
    if [[ -n "$TASK_UUID" ]]; then
        echo "  ✓ $TASK_ID → $TASK_UUID"
    else
        echo "  ✗ $TASK_ID FAILED"
        echo "    $RESULT"
    fi
}

# Wave 1: orchestrator hardening (sequential — same target file)
submit_task "TASK-013" \
    "internal/core/services/orchestrator.go" \
    "internal/core/services/orchestrator.go,internal/core/ports/ports.go"

submit_task "TASK-014" \
    "internal/core/services/orchestrator.go" \
    "internal/core/services/orchestrator.go,internal/core/domain/task.go"

# Wave 2: API + CLI (different files — could be parallel but sequential for simplicity)
submit_task "TASK-015" \
    "internal/adapters/inbound/httpapi/server.go" \
    "internal/adapters/inbound/httpapi/server.go,internal/core/ports/ports.go"

submit_task "TASK-016" \
    "internal/adapters/inbound/cli/root.go" \
    "internal/adapters/inbound/cli/root.go"

# Wave 3: writeback domain
submit_task "TASK-026" \
    "internal/core/domain/task.go" \
    "internal/core/domain/task.go,internal/adapters/outbound/repo_sqlite/repo.go"

# Wave 4: writeback adapter
submit_task "TASK-027" \
    "internal/adapters/outbound/fs_writeback/writeback.go" \
    "internal/core/ports/ports.go"

submit_task "TASK-028" \
    "internal/core/services/orchestrator.go" \
    "internal/core/services/orchestrator.go,internal/core/ports/ports.go"

submit_task "TASK-029" \
    "internal/adapters/inbound/httpapi/server.go" \
    "internal/adapters/inbound/httpapi/server.go,internal/adapters/inbound/mcp/server.go"

echo ""
echo "=== Submission complete ==="
echo "Track progress at: $NEXUS_ADDR/ui"
echo ""
echo "Daemon running (PID $DAEMON_PID). Press Ctrl+C to stop and clean up."
wait "$DAEMON_PID" || true
