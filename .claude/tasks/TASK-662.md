---
id: TASK-662
planId: PLAN-081
title: 'Workspace discovery tools in the agentic loop (list_dir, search_files)'
role: coder
status: done
---

Add an optional `ports.ProjectExplorer` interface implemented by `fs_writer.Writer`
(sandboxed with `safePath`) and wire `list_dir` and `search_files` into `agent_loop.go`
so autonomous agents can inspect and navigate repositories without pre-specified target files.
