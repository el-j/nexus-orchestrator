---
id: TASK-640
planId: PLAN-077
title: 'Split the boss files'
role: refactor
status: todo
createdAt: 2026-10-31
---

# TASK-640 — Split the boss files

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Tests: orchestrator_test.go 1 931, mcp/server_test.go 1 338, httpapi_client/client_test.go 1 188,
httpapi/server_test.go 1 184 into one file per feature. docs/assets/css/style.css 1 913 into
partials; docs/api-reference.md 1 000 into one page per area; publish.yml 747 into reusable
workflows; .claude/orchestrator.json session notes (5 676 lines) archived to .claude/archive. Extend
check-file-size.sh to tests (limit 800) after the split.
