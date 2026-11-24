---
id: PLAN-081
title: 'Project Auto-Mode: autonomous end-to-end implementation loop'
status: completed
createdAt: 2026-11-24
tasks: [TASK-662, TASK-663, TASK-664, TASK-665]
---

# PLAN-081 — Project Auto-Mode: autonomous end-to-end implementation loop

Turn nexusOrchestrator into a full autonomous implementation engine for any project:
given a goal (or backlog), Nexus syncs the project brain (RAG), decomposes work into
dependency-ordered tasks, executes them via workspace-aware agents (or delegates to
active external sessions), verifies and self-heals each step, runs a final project check
with automatic repair, and digests all learnings back into project memory.

| Task     | Title                                                         | Role         | Status |
| -------- | ------------------------------------------------------------- | ------------ | ------ |
| TASK-662 | Workspace discovery tools in the agentic loop (list/search)   | coder        | done   |
| TASK-663 | Task dependencies (DependsOn) and dependency-aware scheduling | architecture | done   |
| TASK-664 | Project Auto-Mode engine (Brain -> Plan -> Execute -> Verify) | architecture | done   |
| TASK-665 | Auto-Mode REST, CLI (`nexus auto`), UI panel, docs, and E2E   | coder        | done   |
