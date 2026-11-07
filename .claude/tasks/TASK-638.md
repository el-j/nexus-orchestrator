---
id: TASK-638
planId: PLAN-077
title: 'Document every Go function, enforce it'
role: docs
status: done
createdAt: 2026-10-31
---

# TASK-638 — Document every Go function, enforce it

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

About 225 unexported functions (mcp 67, httpapi 49, sys_scanner 23, services 22, cli 13, llm_cli 8,
repo_sqlite 7) have no doc comment, plus the interface-method implementations. Write them (what and
why, not a restatement of the name) and add a check (a go/ast test in internal/docscheck or revive
`exported` with unexported checks) so it stays at 100 %.

Done: 225 functions and methods now carry a doc comment (what and why). `internal/docscheck/go_docs_test.go` parses every non-test Go file and fails with the list when a function or method has none, so the in-code documentation stays at 100 %.
