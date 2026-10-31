---
id: PLAN-076
title: 'Audit round 2 (2026-10-31): functionality and performance of the released daemon'
status: active
createdAt: 2026-10-31
tasks: [TASK-626, TASK-627, TASK-628]
---

# PLAN-076 — Audit round 2

Goal: after the three subscription routes (Claude CLI, Antigravity CLI, Copilot via the editor bridge)
were verified end to end, audit the running daemon for functional and performance flaws and fix
each one with a test. Findings are added to this plan as tasks.

| Task     | Finding                                                                                          | Status |
| -------- | ------------------------------------------------------------------------------------------------ | ------ |
| TASK-626 | `daemon.log` grew 20 MB in 11 h: 65 % heartbeats, the rest polls; no rotation                    | done   |
| TASK-627 | `npm audit`: esbuild 0.27.3-0.28.0 (low, Windows dev server) in `frontend` and `docs`            | done   |
| TASK-628 | `-race -count=3`: flaky `TestWaitForTask_StreamsProgress…` (test raced the handler's first read) | done   |
