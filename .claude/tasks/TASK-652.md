---
id: TASK-652
planId: PLAN-079
title: 'Standalone RAG CLI & dedicated Brain MCP server (nexus-brain)'
role: architecture
status: done
createdAt: 2026-11-19
---

# TASK-652 — Standalone RAG CLI & dedicated Brain MCP server (nexus-brain)

Source: PLAN-079 / GitHub Issue #143.

Goals:

1. Provide a standalone binary `cmd/nexus-brain` that executes RAG search, ingestion, and watching without requiring `nexus-daemon`.
2. Support `nexus-brain search <query> [--all] [--hybrid] [--json]`.
3. Support `nexus-brain ingest <path>... [--topic <topic>] [--watch]`.
4. Support dedicated Brain MCP server `nexus-brain mcp [--stdio] [--port <port>]` exposing only the knowledge tools (`search_knowledge`, `ingest_knowledge`, `get_focused_context`, `list_knowledge`).
5. Ensure 100% statement and doc-comment test coverage.
