---
id: PLAN-079
title: 'Monorepo Modularization: Standalone CLIs, Dedicated Micro-Services, and Decoupled Packages'
status: in_progress
createdAt: 2026-11-19
tasks: [TASK-652, TASK-653, TASK-654, TASK-655, TASK-656]
---

# PLAN-079 — Monorepo Modularization & Standalone CLIs

Decompose Nexus Orchestrator into standalone, reusable CLIs and micro-services while retaining a unified, zero-friction developer experience in a clean monorepo.

1. **TASK-652**: Standalone RAG CLI & dedicated Brain MCP server (`nexus-brain`).
2. **TASK-653**: Standalone LLM Gateway & OpenAI-compatible reverse proxy (`nexus-router`).
3. **TASK-654**: Standalone verification gate & task execution engine (`nexus-runner`).
4. **TASK-655**: Wire control plane (`nexus-daemon`) and unified CLI (`nexus`) to decoupled packages.
5. **TASK-656**: Build matrix, documentation, and 100% quality gate coverage.

| Task     | Title                                                | Role         | Status |
| -------- | ---------------------------------------------------- | ------------ | ------ |
| TASK-652 | Standalone RAG CLI & dedicated Brain MCP server      | architecture | done   |
| TASK-653 | Standalone LLM Gateway & OpenAI-compatible proxy     | gateway      | done   |
| TASK-654 | Standalone verification gate & execution engine      | runner       | done   |
| TASK-655 | Wire control plane and unified CLI to packages       | orchestrator | open   |
| TASK-656 | Build matrix, documentation, and 100% coverage gates | devops       | open   |
