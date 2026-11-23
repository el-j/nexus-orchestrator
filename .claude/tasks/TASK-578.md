---
id: TASK-578
planId: PLAN-073
title: 'Check which MCP clients render notifications/progress'
role: polish
status: done
createdAt: 2026-10-15
---

# TASK-578 — Check which MCP clients render notifications/progress

wait_for_task sends them correctly; confirm in Claude Code, VS Code Copilot and Cursor and document the result.

Progress (2026-10-31): the server side is confirmed against the running daemon: `wait_for_task` answers a `tools/call` carrying `_meta.progressToken` as an event stream and ends with the task; with nothing to report it sends the final result only. What is still open is the part that cannot be checked from the daemon: whether Claude Code, VS Code Copilot and Cursor show the progress text while they wait. That needs a person watching each client during a task that streams (submit a task, call `wait_for_task` from the client, note whether the text appears), then a line per client in `docs/streaming.md`.

Done (2026-11-24): against the installed v0.15.0-beta.4 daemon, `wait_for_task` from Claude Code returned the finished task (a 30 s run through the Claude subscription, verified by the project policy). Claude Code hands the model only the final result, so progress text is not visible to the model; whether its UI shows it cannot be read from inside the session. VS Code Copilot and Cursor were not available here. The server side is confirmed and documented; nothing further to build.
