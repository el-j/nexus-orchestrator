---
id: TASK-648
planId: PLAN-078
title: 'Cross-project search across all repositories'
role: search
status: done
createdAt: 2026-11-18
---

# TASK-648 — Cross-project search across all repositories

Source: PLAN-078.

Problem: Knowledge search always filters by single `project_path = ?`, preventing an agent from querying patterns, conventions, or decisions from other projects.

Goal:

1. Support empty `projectPath` or `"*"` to query across all projects.
2. Include `projectPath` on `domain.ContextSection`.
3. Support across REST, MCP (`search_knowledge`, `get_focused_context`), CLI, and Web UI.
