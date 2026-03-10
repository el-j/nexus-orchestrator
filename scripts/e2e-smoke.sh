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
DAEMON_PID=""

PASS_COUNT=0
FAIL_COUNT=0

# ── Helpers ───────────────────────────────────────────────────────────────────

cleanup() {
    if [ -n "$DAEMON_PID" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
        kill "$DAEMON_PID" 2>/dev/null || true
        wait "$DAEMON_PID" 2>/dev/null || true
    fi
    rm -f "$DAEMON_BIN" "$DB_PATH"
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

# ── Build ─────────────────────────────────────────────────────────────────────

echo "==> Building daemon…"
CGO_ENABLED=1 go build -o "$DAEMON_BIN" ./cmd/nexus-daemon
echo "    Built: $DAEMON_BIN"

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
    -d '{"projectPath":"/tmp","targetFile":"test.go","instruction":"test","contextFiles":[]}')"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
if [ "$HTTP_CODE" = "201" ] && echo "$BODY" | grep -q '"task_id"'; then
    pass "Test 3 — Submit task (201 Created)"
    TASK_ID="$(echo "$BODY" | grep -o '"task_id":"[^"]*"' | head -1 | cut -d'"' -f4)"
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

# Test 6 — Cancel task
if [ -n "$TASK_ID" ]; then
    RESP="$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "${HTTP_BASE}/api/tasks/${TASK_ID}")"
    if [ "$RESP" = "204" ]; then
        pass "Test 6 — Cancel task (204 No Content)"
    else
        fail "Test 6 — Cancel task: expected 204, got $RESP"
    fi
else
    fail "Test 6 — Cancel task: skipped (no task_id)"
fi

# Test 7 — MCP health (tools/call → health)
BODY="$(curl -sf -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"health","arguments":{}}}')"
assert_contains "Test 7 — MCP health" "$BODY" '"ok"'

# Test 8 — MCP initialize
BODY="$(curl -sf -X POST "${MCP_BASE}/mcp" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e-smoke","version":"1.0"}}}')"
assert_contains "Test 8 — MCP initialize" "$BODY" '2024-11-05'

# Test 9 — Dashboard HTML
RESP="$(curl -s -w '\n%{http_code}' "${HTTP_BASE}/ui")"
HTTP_CODE="$(echo "$RESP" | tail -1)"
BODY="$(echo "$RESP" | sed '$d')"
if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -qi 'html'; then
    pass "Test 9 — Dashboard returns HTML"
else
    fail "Test 9 — Dashboard: HTTP $HTTP_CODE, body snippet: ${BODY:0:80}"
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
