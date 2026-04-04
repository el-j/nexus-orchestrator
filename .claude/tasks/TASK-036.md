---
id: TASK-036
title: Dogfood scripts and .claude/commands/dogfood-plan002.md
role: planning
planId: PLAN-003
status: todo
dependencies: [TASK-033]
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

The dogfood loop requires a reproducible script that:
1. Builds the daemon and nexus-submit
2. Starts the daemon with a fresh DB
3. Submits each PLAN-002 implementation task as a nexus task (pointing to the correct Go source file)
4. Prints task IDs and the dashboard URL for tracking

This is both a `scripts/dogfood-plan002.sh` bash script and a `.claude/commands/dogfood-plan002.md` Claude command for interactive use.

## Files to Create

- `scripts/dogfood-plan002.sh`
- `.claude/commands/dogfood-plan002.md`

## scripts/dogfood-plan002.sh

Key design points:
- Must work on macOS (zsh/bash compatible)
- Uses `CGO_ENABLED=1 go build` — requires gcc/clang
- Starts daemon in background, captures PID, traps EXIT to kill it
- Submits these PLAN-002 tasks with their target files and context files:

| Task File | Target Go File | Context Files |
|-----------|---------------|---------------|
| TASK-013.md | internal/core/services/orchestrator.go | internal/core/services/orchestrator.go,internal/core/ports/ports.go |
| TASK-014.md | internal/core/services/orchestrator.go | internal/core/services/orchestrator.go,internal/core/domain/task.go |
| TASK-015.md | internal/adapters/inbound/httpapi/server.go | internal/adapters/inbound/httpapi/server.go,internal/core/ports/ports.go |
| TASK-016.md | internal/adapters/inbound/cli/root.go | internal/adapters/inbound/cli/root.go |
| TASK-026.md | internal/core/domain/task.go | internal/core/domain/task.go,internal/adapters/outbound/repo_sqlite/repo.go |
| TASK-027.md | internal/adapters/outbound/fs_writeback/writeback.go | internal/core/ports/ports.go |
| TASK-028.md | internal/core/services/orchestrator.go | internal/core/services/orchestrator.go,internal/core/ports/ports.go |
| TASK-029.md | internal/adapters/inbound/httpapi/server.go | internal/adapters/inbound/httpapi/server.go,internal/adapters/inbound/mcp/server.go |

Full script:
```bash
#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NEXUS_ADDR="${NEXUS_ADDR:-http://127.0.0.1:63987}"
DB_PATH="${NEXUS_DB_PATH:-/tmp/nexus-dogfood-$(date +%s).db}"
DAEMON_BINARY="/tmp/nexus-daemon-dogfood"
SUBMIT_BINARY="/tmp/nexus-submit-dogfood"

cd "$PROJECT_ROOT"

echo "=== nexusOrchestrator Dogfood Runner ==="
echo "Project: $PROJECT_ROOT"
echo "DB:      $DB_PATH"
echo "Addr:    $NEXUS_ADDR"
echo ""

# 1. Build binaries
echo "Building daemon..."
CGO_ENABLED=1 go build -o "$DAEMON_BINARY" ./cmd/nexus-daemon

echo "Building nexus-submit..."
CGO_ENABLED=1 go build -o "$SUBMIT_BINARY" ./cmd/nexus-submit

# 2. Start daemon
echo "Starting daemon (DB: $DB_PATH)..."
NEXUS_DB_PATH="$DB_PATH" "$DAEMON_BINARY" &
DAEMON_PID=$!
trap 'echo "Stopping daemon (PID $DAEMON_PID)..."; kill "$DAEMON_PID" 2>/dev/null; rm -f "$DB_PATH"' EXIT

# Wait for daemon to be ready
for i in $(seq 1 10); do
    if curl -sf "$NEXUS_ADDR/api/health" > /dev/null 2>&1; then
        echo "Daemon ready ✓"
        break
    fi
    sleep 0.5
done

echo ""
echo "Dashboard: $NEXUS_ADDR/ui"
echo ""

# 3. Submit tasks
declare -A TASKS=(
    ["TASK-013"]="internal/core/services/orchestrator.go|internal/core/services/orchestrator.go,internal/core/ports/ports.go"
    ["TASK-014"]="internal/core/services/orchestrator.go|internal/core/services/orchestrator.go,internal/core/domain/task.go"
    ["TASK-015"]="internal/adapters/inbound/httpapi/server.go|internal/adapters/inbound/httpapi/server.go,internal/core/ports/ports.go"
    ["TASK-016"]="internal/adapters/inbound/cli/root.go|internal/adapters/inbound/cli/root.go"
    ["TASK-026"]="internal/core/domain/task.go|internal/core/domain/task.go,internal/adapters/outbound/repo_sqlite/repo.go"
    ["TASK-027"]="internal/adapters/outbound/fs_writeback/writeback.go|internal/core/ports/ports.go"
    ["TASK-028"]="internal/core/services/orchestrator.go|internal/core/services/orchestrator.go,internal/core/ports/ports.go"
    ["TASK-029"]="internal/adapters/inbound/httpapi/server.go|internal/adapters/inbound/httpapi/server.go,internal/adapters/inbound/mcp/server.go"
)

SUBMITTED=()
for TASK_ID in TASK-013 TASK-014 TASK-015 TASK-016 TASK-026 TASK-027 TASK-028 TASK-029; do
    TASK_FILE=".claude/tasks/${TASK_ID}.md"
    if [[ ! -f "$TASK_FILE" ]]; then
        echo "WARN: $TASK_FILE not found — skipping"
        continue
    fi
    ENTRY="${TASKS[$TASK_ID]}"
    TARGET="${ENTRY%%|*}"
    CONTEXT="${ENTRY##*|}"

    echo -n "Submitting $TASK_ID (target: $TARGET)... "
    RESULT=$("$SUBMIT_BINARY" \
        --task-file "$TASK_FILE" \
        --project "$PROJECT_ROOT" \
        --target "$TARGET" \
        --context "$CONTEXT" \
        --addr "$NEXUS_ADDR" 2>&1) || true

    TASK_UUID=$(echo "$RESULT" | grep 'task_id=' | sed 's/.*task_id=//' | awk '{print $1}')
    if [[ -n "$TASK_UUID" ]]; then
        echo "✓ $TASK_UUID"
        SUBMITTED+=("$TASK_ID:$TASK_UUID")
    else
        echo "✗ FAILED"
        echo "  $RESULT"
    fi
done

echo ""
echo "=== Submitted ${#SUBMITTED[@]} tasks ==="
for entry in "${SUBMITTED[@]}"; do
    TASK_ID="${entry%%:*}"
    UUID="${entry##*:}"
    echo "  $TASK_ID → $UUID"
done
echo ""
echo "Track progress: $NEXUS_ADDR/ui"
echo "API:            $NEXUS_ADDR/api/tasks"
echo ""
echo "Daemon is running (PID $DAEMON_PID). Press Ctrl+C to stop."
wait "$DAEMON_PID" || true
```

The script is designed to be run from the project root even though it lives in `scripts/`.

## .claude/commands/dogfood-plan002.md

The Claude command is an interactive version that:
1. Asks which PLAN-002 tasks to submit (default: all todo tasks)
2. Checks if daemon is running (tries GET /api/health)
3. Submits each task via HTTP POST
4. Tracks progress by polling GET /api/tasks/{id}
5. Opens /ui in browser when done

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `scripts/dogfood-plan002.sh` exists and is executable (`chmod +x`)
- [ ] Script builds daemon + nexus-submit before submitting
- [ ] Script waits for daemon health check before submitting
- [ ] Script traps EXIT to kill daemon and clean up DB file
- [ ] `.claude/commands/dogfood-plan002.md` exists
- [ ] Running the script with a running LM Studio or Ollama submits ≥1 task successfully

## Anti-patterns to Avoid

- NEVER use `kill -9` — use `kill` (SIGTERM) for daemon graceful shutdown
- NEVER commit the DB file to git — it's ephemeral in /tmp
- NEVER hardcode task target files in the task MD itself — specify them at submit time
- NEVER submit tasks that overlap on the same output file simultaneously (sequential per file group)
