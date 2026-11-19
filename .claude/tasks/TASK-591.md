---
id: TASK-591
planId: PLAN-074
title: 'Ladder in the UI and CLI'
role: polish
wave: 3
status: done
createdAt: 2026-10-16
dependsOn: [TASK-590]
---

# TASK-591 — Ladder in the UI and CLI

Show and edit the ladder (Providers/Settings, `nexus-cli`), surface matrix.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Scope note: the ladder lives in `.nexus/policy.json`, which has no API (it is a file in the repository), so "edit" stays editing that file. What was missing was seeing the effect, so this task adds the routing preview (REST, MCP, CLI); the Providers view shows it with TASK-601.
