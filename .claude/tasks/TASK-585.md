---
id: TASK-585
planId: PLAN-074
title: 'Usage ledger: store, REST, MCP, CLI'
role: coder
wave: 1
status: todo
createdAt: 2026-10-16
dependsOn: [TASK-584]
---

# TASK-585 — Usage ledger: store, REST, MCP, CLI

Table per task/provider/model/role with a migration; `GET /api/usage?since=`, MCP `get_usage`, `nexus-cli usage`; surface matrix, API reference and changelog updated.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
