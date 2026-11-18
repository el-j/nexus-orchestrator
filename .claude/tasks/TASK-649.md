---
id: TASK-649
planId: PLAN-078
title: 'Auto-ingest README and docs directory markdown'
role: ingest
status: done
createdAt: 2026-11-18
---

# TASK-649 — Auto-ingest README and docs directory markdown

Source: PLAN-078.

Problem: `InitProject` only reads single instruction files (`CLAUDE.md`, etc.), leaving `README.md` and `docs/**/*.md` unindexed.

Goal:

1. Discover and ingest `README.md` and markdown files under `docs/` during `InitProject`.
2. Skip archive folders (`docs/archive/`), node_modules, and hidden files.
3. Clean up stale entries from previously ingested files upon re-ingest.
