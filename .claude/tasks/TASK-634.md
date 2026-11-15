---
id: TASK-634
planId: PLAN-077
title: 'Go coverage to 100 %: storage and watcher'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-634 — Go coverage to 100 %: storage and watcher

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Uncovered statements (merged profile): repo_sqlite repo_repair.go 15, repo.go 8, ai_session_repo 4,
knowledge_repo 4, model_capability_repo 4, session_repo 3, activity_repo; fs_watcher/watcher.go 12;
fs_writer 3. Failure-path tests first (injected I/O errors, read-only dirs, closed DB).

Done: `repo_sqlite`, `fs_watcher` and `fs_writer` are at 100 % statement coverage (own tests, with `-race`). `fault_injection_test.go` and `fault_injection_writes_test.go` break one table at a time (a view that adds an unscannable row or one that fails halfway, triggers that abort or skip a statement, a deferred foreign key that refuses the commit). Dead branches were removed instead of tested: the second `Abs` and the root-reached case in `fs_writer`, the `WalkDir` error in `Watch` (its callback has none to return), the time-parse branch in `ListDiscoveredAgents` (the driver already returns a time) and the impossible `json.Marshal` error in `SaveRuntimeConfig`. `RepairForeignRows` reads each row once (it used to read twice) and now checks `rows.Err()`. `New` closes the handle when a pragma or migration fails. Failures of `sql.Open`, `fsnotify.NewWatcher` and `filepath.Abs` are injectable for tests.
