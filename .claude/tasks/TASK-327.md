# TASK-327: Validate `make dev` full-stack startup

**Plan:** PLAN-049
**Status:** done

## Description

Run `make dev` and verify:

1. Daemon starts and becomes healthy
2. Health-check wait loop works
3. Frontend starts after daemon is ready
4. Frontend can reach `/api/health` through proxy
5. Ctrl+C stops both

## Acceptance

- No connection errors on startup
- All services reachable from frontend
