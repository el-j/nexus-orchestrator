---
id: TASK-654
planId: PLAN-079
title: 'Standalone verification gate & execution engine (nexus-runner)'
role: runner
status: done
createdAt: 2026-11-19
---

# TASK-654 — Standalone verification gate & execution engine (nexus-runner)

Source: PLAN-079 / GitHub Issue #145.

Goals:

1. Provide a standalone binary `cmd/nexus-runner` that executes commands and validates changes against project quality invariants.
2. Invariant verification: file size limits (500 lines source / 800 lines test), test coverage, test-first rule.
3. Self-healing execution loop over tasks with auto-correction turns.
4. Ensure 100% statement and doc-comment test coverage.
