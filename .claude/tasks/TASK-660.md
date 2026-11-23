---
id: TASK-660
planId: PLAN-080
title: 'Close TASK-578 and make the MCP shutdown test deterministic'
role: polish
status: done
createdAt: 2026-11-24
---

# TASK-660 — Close TASK-578 and make the MCP shutdown test deterministic

TASK-578: wait_for_task verified through Claude Code against the v0.15.0-beta.4 daemon. The MCP shutdown-grace test raced the server's accept loop, so the 100 % coverage gate failed one run in a few; it now waits until the server has accepted the silent connection.
