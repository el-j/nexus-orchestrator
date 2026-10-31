---
id: TASK-634
planId: PLAN-077
title: 'Go coverage to 100 %: storage and watcher'
role: testing
status: todo
createdAt: 2026-10-31
---

# TASK-634 — Go coverage to 100 %: storage and watcher

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Uncovered statements (merged profile): repo_sqlite repo_repair.go 15, repo.go 8, ai_session_repo 4,
knowledge_repo 4, model_capability_repo 4, session_repo 3, activity_repo; fs_watcher/watcher.go 12;
fs_writer 3. Failure-path tests first (injected I/O errors, read-only dirs, closed DB).
