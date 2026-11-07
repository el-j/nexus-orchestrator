---
id: PLAN-077
title: 'Definition of done: close every gap the 2026-10-31 audit found'
status: active
createdAt: 2026-10-31
tasks:
  [
    TASK-632,
    TASK-633,
    TASK-634,
    TASK-635,
    TASK-636,
    TASK-637,
    TASK-638,
    TASK-639,
    TASK-640,
    TASK-641,
    TASK-642,
    TASK-643,
    TASK-644,
    TASK-645,
    TASK-646,
    TASK-578,
  ]
---

# PLAN-077 — Definition of done

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md). Goal: 100 % Go coverage, every
function documented, no dead code, no oversized function or file, every route and the CLI-driver path
under e2e, mutation testing over the whole core, no `as any`, bounded history.

Order: dead code and types first (632, 633), then tests (634-637, 641, 642), documentation and
structure (638-640, 646), then the gates that keep it (643-645). Raise each CI gate in the task that
reaches it. TASK-578 (progress rendering in three MCP clients) stays with the owner.

| Task     | Title                                                | Role        | Status |
| -------- | ---------------------------------------------------- | ----------- | ------ |
| TASK-632 | Remove dead frontend code                            | frontend    | done   |
| TASK-633 | Replace the five `as any` escapes with real types    | frontend    | done   |
| TASK-634 | Go coverage to 100 %: storage and watcher            | testing     | done   |
| TASK-635 | Go coverage to 100 %: inbound adapters and bootstrap | testing     | done   |
| TASK-636 | Go coverage to 100 %: core services                  | testing     | done   |
| TASK-637 | Go coverage to 100 %: scanners and LLM adapters      | testing     | done   |
| TASK-638 | Document every Go function, enforce it               | docs        | done   |
| TASK-639 | Split the oversized functions                        | refactor    | done   |
| TASK-640 | Split the boss files                                 | refactor    | done   |
| TASK-641 | e2e for the 16 untouched REST routes                 | testing     | done   |
| TASK-642 | e2e for the CLI-driver path and the browser views    | testing     | done   |
| TASK-643 | Mutation testing to the whole core and tighter gates | testing     | todo   |
| TASK-644 | Frontend lint and the nilerr design                  | quality     | todo   |
| TASK-645 | Bound the history and add benchmarks                 | performance | todo   |
| TASK-646 | Mark the historical audits superseded                | docs        | todo   |
