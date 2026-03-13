#!/usr/bin/env bash
set -euo pipefail

# ── E2E Smoke Test for nexusOrchestrator ──────────────────────────────────────
# Builds the daemon, starts it, exercises HTTP + MCP endpoints, and reports.
# No external dependencies beyond curl and grep.
#
# Usage:
#   ./scripts/e2e-smoke.sh
#   NEXUS_HTTP_PORT=29999 NEXUS_MCP_PORT=29998 ./scripts/e2e-smoke.sh

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

HTTP_PORT="${NEXUS_HTTP_PORT:-19999}"
MCP_PORT="${NEXUS_MCP_PORT:-19998}"
HTTP_BASE="http://127.0.0.1:${HTTP_PORT}"
MCP_BASE="http://127.0.0.1:${MCP_PORT}"
DAEMON_BIN="/tmp/nexus-daemon-e2e-$$"
DB_PATH="/tmp/nexus-e2e-$$.db"
PROJECT_PATH="/tmp/nexus-e2e-project-$$"
DAEMON_PID=""

PASS_COUNT=0
FAIL_COUNT=0

# ── Helpers ───────────────────────────────────────────────────────────────────

cleanup() {
    if [ -n "$DAEMON_PID" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
        kill "$DAEMON_PID" 2>/dev/null || true
        wait "$DAEMON_PID" 2>/dev/null || true
    fi
    rm -rf "$DAEMON_BIN" "$DB_PATH" "$PROJECT_PATH"
}
trap cleanup EXIT

pass() {
    echo "  PASS: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo "  FAIL: $1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

assert_contains() {
    local label="$1" body="$2" expected="$3"
    if echo "$body" | grep -q "$expected"; then
        pass "$label"
    else
        fail "$label — expected body to contain '$expected'"
    fi
}

extract_json_field() {
    local body="$1" field="$2"
    echo "$body" | grep -o '"'"$field"'":"[^"]*"' | head -1 | cut -d'"' -f4
}

extract_rpc_text_field() {
    local body="$1" field="$2"
    echo "$body" | sed -n 's/.*\\"'"$field"'\\":\\"\([^\\"]*\)\\".*/\1/p' | head -1
}

# ── Build ─────────────────────────────────────────────────────────────────────

echo "==> Building daemon…"
CGO_ENABLED=1 go build -o "$DAEMON_BIN" ./cmd/nexus-daemon
echo "    Built: $DAEMON_BIN"
mkdir -p "$PROJECT_PATH"

# ── Start daemon ──────────────────────────────────────────────────────────────

echo "==> Starting daemon (HTTP :${HTTP_PORT}, MCP :${MCP_PORT})…"
NEXUS_DB_PATH="$DB_PATH" \
NEXUS_LISTEN_ADDR="127.0.0.1:${HTTP_PORT}" \
NEXUS_MCP_ADDR="127.0.0.1:${MCP_PORT}" \
    "$DAEMON_BIN" &
DAEMON_PID=$!

# Wait for health endpoint (up to 10 seconds)
echo "==> Waiting for daemon to become healthy…"
for i in $(seq 1 20); do
    if curl -sf "${HTTP_BASE}/api/health" >/dev/null 2>&1; then
        echo "    Daemon ready (after ~$((i / 2))s)"
        break
    fi
    if [ "$i" -eq 20 ]; then
        echo "FATAL: Daemon did not become healthy within 10s"
        exit 1
    fi
    sleep 0.5
done

# ── Tests ─────────────────────────────────────────────────────────────────────

echo ""
echo "==> Running smoke tests…"
echo ""

# Test 1 — Health
BODY="$(curl -sf "${HTTP_BASE}/api/health")"
assert_contains "Test 1 — Health" "$BODY" '"ok"'

# Test 2 — Providers (JSON array)
BODY="$(curl -sf "${HTTP_BASE}/api/providers")"
if echo "$BODY" | grep -q '^\['; then
    pass "Test 2 — Providers returns JSON array"
else
    fail "Test 2 — Providers: expected JSON array, got: ${BODY:0:80}"
fi

# Test 3 — Submit task (POST /api/tasks)
RESP="$(curl -s -w '\n%{http_code}' -X POST "${HTTP_BASE}/api/tasks" \
    -H 'Content-Type: application/json' \
    -d '{"projectPath":"'"${PROJECT_PATH}"'","targetFile":"test.go","instruction":"test","contextFiles":[]}')"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
if [ "$HTTP_CODE" = "201" ] && echo "$BODY" | grep -q '"task_id"'; then
    pass "Test 3 — Submit task (201 Created)"
    TASK_ID="$(extract_json_field "$BODY" "task_id")"
else
    fail "Test 3 — Submit task: HTTP $HTTP_CODE, body: ${BODY:0:120}"
    TASK_ID=""
fi

# Test 4 — Get task
if [ -n "$TASK_ID" ]; then
    BODY="$(curl -sf "${HTTP_BASE}/api/tasks/${TASK_ID}")"
    assert_contains "Test 4 — Get task" "$BODY" "$TASK_ID"
else
    fail "Test 4 — Get task: skipped (no task_id)"
fi

# Test 5 — List tasks
BODY="$(curl -sf "${HTTP_BASE}/api/tasks")"
if echo "$BODY" | grep -q '^\['; then
    pass "Test 5 — List tasks returns JSON array"
else
    fail "Test 5 — List tasks: expected JSON array, got: ${BODY:0:80}"
fi

# Test 6 — Create draft over HTTP
RESP="$(curl -s -w '\n%{http_code}' -X POST "${HTTP_BASE}/api/tasks/draft" \
    -H 'Content-Type: application/json' \
    -d '{"projectPath":"'"${PROJECT_PATH}"'","targetFile":"draft.go","instruction":"draft item","priority":2,"tags":["e2e"]}')"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
if [ "$HTTP_CODE" = "201" ] && echo "$BODY" | grep -q '"id"'; then
    pass "Test 6 — Create draft"
    DRAFT_ID="$(extract_json_field "$BODY" "id")"
else
    fail "Test 6 — Create draft: HTTP $HTTP_CODE, body: ${BODY:0:120}"
    DRAFT_ID=""
fi

# Test 7 — Get backlog scoped to project
BODY="$(curl -sf "${HTTP_BASE}/api/tasks/backlog?project=${PROJECT_PATH}")"
if [ -n "$DRAFT_ID" ] && echo "$BODY" | grep -q "$DRAFT_ID"; then
    pass "Test 7 — Project backlog contains draft"
else
    fail "Test 7 — Project backlog missing draft"
fi

# Test 8 — Update draft to BACKLOG
if [ -n "$DRAFT_ID" ]; then
    RESP="$(curl -s -w '\n%{http_code}' -X PUT "${HTTP_BASE}/api/tasks/${DRAFT_ID}" \
        -H 'Content-Type: application/json' \
        -d '{"status":"BACKLOG","priority":1}')"
    HTTP_CODE="$(echo "$RESP" | tail -1)"
    BODY="$(echo "$RESP" | sed '$d')"
    if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q '"status":"BACKLOG"'; then
        pass "Test 8 — Update draft to BACKLOG"
    else
        fail "Test 8 — Update draft to BACKLOG: HTTP $HTTP_CODE, body: ${BODY:0:120}"
    fi
else
    fail "Test 8 — Update draft to BACKLOG: skipped (no draft id)"
fi

# Test 9 — Global backlog works without project param
BODY="$(curl -sf "${HTTP_BASE}/api/tasks/backlog")"
if [ -n "$DRAFT_ID" ] && echo "$BODY" | grep -q "$DRAFT_ID"; then
    pass "Test 9 — Global backlog contains draft"
else
    fail "Test 9 — Global backlog missing draft"
fi

# Test 10 — Create an additional queued item deterministically
RESP="$(curl -s -w '\n%{http_code}' -X POST "${HTTP_BASE}/api/tasks/draft" \
    -H 'Content-Type: application/json' \
    -d '{"projectPath":"'"${PROJECT_PATH}"'","targetFile":"queued.go","instruction":"queued item"}')"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
QUEUED_ID="$(extract_json_field "$BODY" "id")"
if [ "$HTTP_CODE" = "201" ] && [ -n "$QUEUED_ID" ]; then
    RESP="$(curl -s -w '\n%{http_code}' -X PUT "${HTTP_BASE}/api/tasks/${QUEUED_ID}" \
        -H 'Content-Type: application/json' \
        -d '{"status":"QUEUED"}')"
    HTTP_CODE="$(echo "$RESP" | tail -1)"
    BODY="$(echo "$RESP" | sed '$d')"
    if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q '"status":"QUEUED"'; then
        pass "Test 10 — Create queued item via update"
    else
        fail "Test 10 — Create queued item: HTTP $HTTP_CODE, body: ${BODY:0:120}"
    fi
else
    fail "Test 10 — Create queued item failed"
fi

# Test 11 — Get all tasks includes backlog and queued states
BODY="$(curl -sf "${HTTP_BASE}/api/tasks/all")"
if echo "$BODY" | grep -q '"status":"BACKLOG"' && echo "$BODY" | grep -q '"status":"QUEUED"'; then
    pass "Test 11 — Get all tasks includes backlog and queued"
else
    fail "Test 11 — Get all tasks missing expected states"
fi

# Test 12 — Promote backlog task
if [ -n "$DRAFT_ID" ]; then
    RESP="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${HTTP_BASE}/api/tasks/${DRAFT_ID}/promote")"
    if [ "$RESP" = "204" ]; then
        pass "Test 12 — Promote backlog task"
    else
        fail "Test 12 — Promote backlog task: expected 204, got $RESP"
    fi
else
    fail "Test 12 — Promote backlog task: skipped (no draft id)"
fi

# Test 13 — Register AI session over HTTP
RESP="$(curl -s -w '\n%{http_code}' -X POST "${HTTP_BASE}/api/ai-sessions" \
    -H 'Content-Type: application/json' \
    -d '{"agentName":"E2E Agent","source":"http","projectPath":"'"${PROJECT_PATH}"'","externalId":"e2e-http-'"$$"'"}')"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
SESSION_ID="$(extract_json_field "$BODY" "id")"
if [ "$HTTP_CODE" = "201" ] && [ -n "$SESSION_ID" ]; then
    pass "Test 13 — Register AI session"
else
    fail "Test 13 — Register AI session: HTTP $HTTP_CODE, body: ${BODY:0:120}"
fi

# Test 14 — List AI sessions contains registered session
BODY="$(curl -sf "${HTTP_BASE}/api/ai-sessions")"
if [ -n "$SESSION_ID" ] && echo "$BODY" | grep -q "$SESSION_ID"; then
    pass "Test 14 — List AI sessions"
else
    fail "Test 14 — List AI sessions missing registered session"
fi

# Test 15 — Heartbeat AI session
if [ -n "$SESSION_ID" ]; then
    RESP="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${HTTP_BASE}/api/ai-sessions/${SESSION_ID}/heartbeat")"
    if [ "$RESP" = "204" ]; then
        pass "Test 15 — Heartbeat AI session"
    else
        fail "Test 15 — Heartbeat AI session: expected 204, got $RESP"
    fi
else
    fail "Test 15 — Heartbeat AI session: skipped (no session id)"
fi

# Test 16 — Deregister AI session
if [ -n "$SESSION_ID" ]; then
    RESP="$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "${HTTP_BASE}/api/ai-sessions/${SESSION_ID}")"
    if [ "$RESP" = "204" ]; then
        pass "Test 16 — Deregister AI session"
    else
        fail "Test 16 — Deregister AI session: expected 204, got $RESP"
    fi
else
    fail "Test 16 — Deregister AI session: skipped (no session id)"
fi

# Test 17 — Cancel task (accept 204 or 404 — task may already be processing/completed)
if [ -n "$TASK_ID" ]; then
    RESP="$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "${HTTP_BASE}/api/tasks/${TASK_ID}")"
    if [ "$RESP" = "204" ] || [ "$RESP" = "404" ]; then
        pass "Test 17 — Cancel task ($RESP accepted)"
    else
        fail "Test 17 — Cancel task: expected 204 or 404, got $RESP"
    fi
else
    fail "Test 17 — Cancel task: skipped (no task_id)"
fi

# Test 18 — MCP health (tools/call → health)
BODY="$(curl -sf -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"health","arguments":{}}}')"
assert_contains "Test 18 — MCP health" "$BODY" 'ok'

# Test 19 — MCP initialize
BODY="$(curl -sf -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e-smoke","version":"1.0"}}}')"
assert_contains "Test 19 — MCP initialize" "$BODY" '2024-11-05'

# Test 20 — MCP tools/list includes new parity tools
BODY="$(curl -sf -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":3,"method":"tools/list"}')"
if echo "$BODY" | grep -q '"get_all_tasks"' && echo "$BODY" | grep -q '"create_draft"' && echo "$BODY" | grep -q '"get_ai_sessions"'; then
    pass "Test 20 — MCP tools/list includes parity tools"
else
    fail "Test 20 — MCP tools/list missing expected tools"
fi

# Test 21 — MCP create_draft
BODY="$(curl -s -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"create_draft","arguments":{"projectPath":"'"${PROJECT_PATH}"'","instruction":"mcp draft","targetFile":"mcp.go"}}}')"
MCP_DRAFT_ID="$(extract_rpc_text_field "$BODY" "id")"
if [ -n "$MCP_DRAFT_ID" ]; then
    pass "Test 21 — MCP create_draft"
else
    fail "Test 21 — MCP create_draft missing id"
fi

# Test 22 — MCP get_backlog returns project draft
BODY="$(curl -s -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_backlog","arguments":{"projectPath":"'"${PROJECT_PATH}"'"}}}')"
if [ -n "$MCP_DRAFT_ID" ] && echo "$BODY" | grep -q "$MCP_DRAFT_ID"; then
    pass "Test 22 — MCP get_backlog"
else
    fail "Test 22 — MCP get_backlog missing draft"
fi

# Test 23 — MCP get_all_tasks returns deterministic all-task state
BODY="$(curl -s -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"get_all_tasks","arguments":{}}}')"
if [ -n "$MCP_DRAFT_ID" ] && [ -n "$QUEUED_ID" ] && echo "$BODY" | grep -q "$MCP_DRAFT_ID" && echo "$BODY" | grep -q "$QUEUED_ID" && echo "$BODY" | grep -q 'DRAFT' && echo "$BODY" | grep -q 'QUEUED'; then
    pass "Test 23 — MCP get_all_tasks"
else
    fail "Test 23 — MCP get_all_tasks missing expected states"
fi

# Test 24 — MCP register_session and get_ai_sessions
BODY="$(curl -s -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"register_session","arguments":{"agent_name":"E2E MCP Agent","project_path":"'"${PROJECT_PATH}"'","external_id":"e2e-mcp-'"$$"'"}}}')"
MCP_SESSION_ID="$(extract_rpc_text_field "$BODY" "session_id")"
if [ -n "$MCP_SESSION_ID" ]; then
    BODY="$(curl -s -X POST "${MCP_BASE}/mcp" \
        -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"get_ai_sessions","arguments":{}}}')"
    if echo "$BODY" | grep -q "$MCP_SESSION_ID"; then
        pass "Test 24 — MCP AI session lifecycle"
    else
        fail "Test 24 — MCP get_ai_sessions missing registered session"
    fi
else
    fail "Test 24 — MCP register_session missing id"
fi

# Test 25 — Dashboard HTML
RESP="$(curl -s -w '\n%{http_code}' "${HTTP_BASE}/ui")"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -qi 'html'; then
    pass "Test 25 — Dashboard returns HTML"
else
    fail "Test 25 — Dashboard: HTTP $HTTP_CODE, body snippet: ${BODY:0:80}"
fi

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo "Results: $PASS_COUNT/$TOTAL passed"

if [ "$FAIL_COUNT" -eq 0 ]; then
    echo "All tests passed ✓"
    exit 0
else
    echo "$FAIL_COUNT tests failed ✗"
    exit 1
fi
