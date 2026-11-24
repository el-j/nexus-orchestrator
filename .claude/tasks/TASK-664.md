---
id: TASK-664
planId: PLAN-081
title: 'Project Auto-Mode engine (Brain -> Plan -> Execute -> Verify -> Learn)'
role: architecture
status: done
---

Implement the core Auto-Mode orchestrator loop: sync project brain, decompose a goal into
dependency-ordered agentic tasks (or promote existing backlog), monitor execution and
external session delegation, run a final project verification with automatic repair tasks,
and digest learnings into project memory.
