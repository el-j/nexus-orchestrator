# TASK-324: Fix Vite proxy config

**Plan:** PLAN-049
**Status:** DONE

## Description

Fix `frontend/vite.config.ts` proxy configuration:

1. Change `/mcp` proxy target from `http://127.0.0.1:63987` to `http://127.0.0.1:63988`
2. Add `/.well-known` proxy rule pointing to `http://127.0.0.1:63987`

## Acceptance

- `/mcp` proxy target is `:63988`
- `/.well-known/` proxy target is `:63987`
- Vite config passes TypeScript check
