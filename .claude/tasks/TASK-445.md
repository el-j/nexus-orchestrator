---
id: TASK-445
plan: PLAN-059
status: todo
wave: 3
priority: 3
---

# TASK-445: wails.ts HeartbeatAISession return type fix

**Problem:** `HeartbeatAISession(id: string): Promise<Error>` — wrong return type. Should be `Promise<void>`. The wrapper catches errors internally and warns.

**Fix:** Change return type from `Promise<Error>` to `Promise<void>` in the wails.ts type declaration.

**Files:** `frontend/src/wailsjs/wails.ts` (or wherever HeartbeatAISession is declared)
