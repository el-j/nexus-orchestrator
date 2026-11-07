---
id: PLAN-076
title: 'Audit round 2 (2026-10-31): functionality and performance of the released daemon'
status: completed
createdAt: 2026-10-31
tasks: [TASK-626, TASK-627, TASK-628, TASK-629, TASK-630, TASK-631]
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
| TASK-629 | Provider probes held a global lock and ran one after another (0.95 s cold, blocks routing)       | done   |
| TASK-630 | `agy models` (2 s, network) ran every 30 s                                                       | done   |
| TASK-631 | `/api/tasks/all` is unbounded (20 k tasks = 93 MB, 3.2 s); needs a product decision on paging    | todo   |
