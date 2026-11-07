---
id: TASK-638
planId: PLAN-077
title: 'Document every Go function, enforce it'
role: docs
status: todo
createdAt: 2026-10-31
---

# TASK-638 — Document every Go function, enforce it

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

About 225 unexported functions (mcp 67, httpapi 49, sys_scanner 23, services 22, cli 13, llm_cli 8,
repo_sqlite 7) have no doc comment, plus the interface-method implementations. Write them (what and
why, not a restatement of the name) and add a check (a go/ast test in internal/docscheck or revive
`exported` with unexported checks) so it stays at 100 %.
