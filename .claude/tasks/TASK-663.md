---
id: TASK-663
planId: PLAN-081
title: 'Task dependencies (DependsOn) and dependency-aware scheduling'
role: architecture
status: done
---

Add `DependsOn []string` to `domain.Task`, persist `depends_on` in SQLite, skip queued tasks
whose dependencies have not yet completed, fail dependents cleanly when a dependency fails or
is cancelled, and wake workers when tasks complete. Expose `dependsOn` on REST and MCP.
