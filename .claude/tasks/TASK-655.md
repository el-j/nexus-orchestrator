---
id: TASK-655
planId: PLAN-079
title: 'Wire control plane and unified CLI to decoupled packages'
role: orchestrator
status: open
createdAt: 2026-11-19
---

# TASK-655 — Wire control plane and unified CLI to decoupled packages

Source: PLAN-079 / GitHub Issue #146.

Goals:

1. Wire `nexus-daemon` to import and compose decoupled packages without duplication.
2. Wire `nexus` unified CLI to seamlessly dispatch between local standalone operations and daemon proxying.
3. Ensure backwards compatibility with Wails desktop app, REST API, and full MCP server.
4. Ensure 100% statement and doc-comment test coverage.
