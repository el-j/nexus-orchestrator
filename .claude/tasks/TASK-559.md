---
id: TASK-559
planId: PLAN-072
title: 'Decompose monolithic mcp/tools.go into domain modules'
role: backend
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:11:30Z
---

# TASK-559 — Decompose monolithic mcp/tools.go into domain modules

## Context

`internal/adapters/inbound/mcp/tools.go` is 1,241 lines long and handles 40 tools covering task queues, backlog, providers, system discovery, sessions, and project brain. It is the single largest production source file in the repository.

## Work Required

1. Split `tools.go` into domain-specific tool files within `package mcp`:
   - `tools_tasks.go`: `submit_task`, `get_task`, `cancel_task`, `get_queue`, `create_draft`, `get_backlog`, `promote_task`, `update_task`, `heartbeat_task`, `claim_task`, `update_task_status`.
   - `tools_brain.go`: `ingest_knowledge`, `get_project_context`, `get_focused_context`, `search_knowledge`, `get_brain_status`, `init_project`, `list_knowledge`, `delete_knowledge`, `get_file_map`.
   - `tools_providers.go`: `get_providers`, `get_provider_models`, `add_provider_config`, `update_provider_config`, `remove_provider_config`, `list_provider_configs`, `get_discovered_providers`, `trigger_scan`, `promote_provider`.
   - `tools_sessions.go`: `register_session`, `get_ai_sessions`, `deregister_session`, `terminate_ai_session`, `heartbeat_ai_session`, `purge_disconnected_sessions`, `get_discovered_agents`, `delegate_to_nexus`, `get_discovered_plan_files`.
   - `tools.go`: Retain core `toolRegistry`, dispatch map, `executeToolCall` dispatcher, and helper methods.
2. Verify all existing tests pass:
   - `go test -race -count=1 ./internal/adapters/inbound/mcp/...`
