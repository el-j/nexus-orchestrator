---
id: TASK-617
planId: PLAN-075
title: 'Add structured IngestKnowledge, CLI focused-context, and full UpdateTask fields across MCP and CLI'
role: coder
wave: 3
status: done
createdAt: 2026-10-19
---

# TASK-617 — Add structured IngestKnowledge, CLI focused-context, and full UpdateTask fields across MCP and CLI

1. **`internal/adapters/inbound/mcp/brain_tools.go` & `internal/adapters/inbound/cli/brain.go`**:
   - Allow MCP `ingest_knowledge` to accept either `filePath` (`IngestFromFile`) or structured `{kind, topic, content}` (`BrainService.IngestKnowledge`), so agents following `baselineRules` can record single knowledge entries directly without writing a file first.
   - Allow CLI `nexus brain ingest` to accept `--kind`, `--topic`, and `--content` in addition to `--file`.
   - Add `--question` / `-q` to CLI `nexus brain context` so it calls `brain.GetFocusedContext` when a question is supplied (matching `surface_matrix_test.go:85`).
2. **`internal/adapters/inbound/mcp/tools_tasks.go` & `internal/adapters/inbound/cli/root.go`**:
   - Support `targetFile`, `providerHint`, and `status` (`DRAFT` ↔ `BACKLOG`) in MCP `update_task` and CLI `nexus update <task-id>`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated.
