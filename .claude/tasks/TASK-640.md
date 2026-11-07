---
id: TASK-640
planId: PLAN-077
title: 'Split the boss files'
role: refactor
status: done
createdAt: 2026-10-31
---

# TASK-640 — Split the boss files

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Tests: orchestrator_test.go 1 931, mcp/server_test.go 1 338, httpapi_client/client_test.go 1 188,
httpapi/server_test.go 1 184 into one file per feature. docs/assets/css/style.css 1 913 into
partials; docs/api-reference.md 1 000 into one page per area; publish.yml 747 into reusable
workflows; .claude/orchestrator.json session notes (5 676 lines) archived to .claude/archive. Extend
check-file-size.sh to tests (limit 800) after the split.

Done: the four Go test files are split by feature (`orchestrator_{fakes,queue,session,providers,claim}_test.go`, `server_{helpers,protocol,tasks,providers,sessions}_test.go` for mcp and `server_{helpers,tasks,providers,sessions,system}_test.go` for httpapi, `client_*_test.go`, and `repo_sqlite/repo_test.go` into four), none over 800 lines; `scripts/check-file-size.sh` now also checks tests (limit 800, `MAX_TEST_LINES`). `docs/assets` (the old static site's 1 913-line `style.css`, `custom.scss` and `main.js`, loaded by nothing) is deleted, with a docscheck test that keeps it gone. `.claude/orchestrator.json` keeps the counters and the open entries (107 lines); finished plans and tasks, the session notes and the stray top-level entries moved to `.claude/archive/orchestrator-until-PLAN-074.json`.

Decided against: splitting `docs/api-reference.md` (the docs site renders it as one page and three drift tests read it, so a split changes the site and the tests for a reference page that is read by search, not top to bottom) and `publish.yml` into reusable workflows (the release and the beta publish cannot be exercised without publishing, and docscheck parses the file; a mistake would be found by a failed release). Both can be done as their own task if the size becomes a problem.
