# TASK-325: Rewrite `make dev` with health-check and correct banner

**Plan:** PLAN-049
**Status:** DONE

## Description

Rewrite the `make dev` Makefile target to:

1. Start `air` (daemon hot-reload) in background
2. Wait for daemon health-check (`curl -sf http://127.0.0.1:63987/api/health`) with retry loop (max 30 attempts, 1s apart)
3. Print clear banner showing all service URLs once daemon is healthy
4. Start `cd frontend && npm run dev` in foreground
5. Trap SIGINT to kill both processes cleanly

## Banner should show:

```
  Daemon   → http://127.0.0.1:63987  (API)
  MCP      → http://127.0.0.1:63988  (JSON-RPC)
  Frontend → http://127.0.0.1:63989  (Vite HMR → proxies /api → :63987, /mcp → :63988)
```

## Acceptance

- `make dev` starts daemon, waits for health, then starts frontend
- Ctrl+C stops both processes cleanly
- Banner text is accurate
