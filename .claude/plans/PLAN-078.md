---
id: PLAN-078
title: 'Cross-project & natural-language RAG upgrade for the Project Brain'
status: done
createdAt: 2026-11-18
tasks: [TASK-647, TASK-648, TASK-649, TASK-650, TASK-651]
---

# PLAN-078 — Cross-Project & Natural-Language RAG Upgrade

Upgrade the Nexus Project Brain from a single-project literal phrase lookup into a full RAG system across all projects:

1. Tokenized natural-language FTS5 BM25 search with ranked query building.
2. Cross-project search across all registered workspaces (`*` or empty project path).
3. Auto-ingestion of `README.md` and `docs/**/*.md` during project initialization.
4. Optional hybrid vector embeddings & reciprocal rank fusion (RRF) when embedding providers are reachable.
5. End-to-end tests, docscheck enforcement, surface matrix update, and performance checks.

| Task     | Title                                              | Role    | Status |
| -------- | -------------------------------------------------- | ------- | ------ |
| TASK-647 | Tokenized natural-language FTS5 BM25 search        | search  | done   |
| TASK-648 | Cross-project search across all repositories       | search  | done   |
| TASK-649 | Auto-ingest README and docs directory markdown     | ingest  | done   |
| TASK-650 | Optional hybrid vector search & RRF ranking        | search  | done   |
| TASK-651 | Surface matrix, docs, E2E tests and coverage gates | testing | done   |
