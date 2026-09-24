---
id: TASK-555
planId: PLAN-071
title: 'SQLite Persistence & Additive Migrations for Task Verification Fields'
role: backend
status: done
createdAt: 2026-09-24T15:58:00Z
---

# TASK-555 — SQLite Persistence & Additive Migrations

## Context

Tasks now have `verification_command`, `max_correction_turns`, and `verification_output`. The SQLite repository in `internal/adapters/outbound/repo_sqlite/repo.go` must apply additive schema migrations and correctly read/write these fields across all query and persistence methods.

## Work Required

1. In `internal/adapters/outbound/repo_sqlite/repo.go`:
   - Add columns to `migrate()`:
     - `verification_command TEXT NOT NULL DEFAULT ''`
     - `max_correction_turns INTEGER NOT NULL DEFAULT 0`
     - `verification_output TEXT NOT NULL DEFAULT ''`
   - Update `Save`, `GetByID`, `scanTask`, `GetPending`, `GetAll`, `GetByProjectPath`, `GetByProjectPathAndStatus`, `Update` to handle the new columns.
2. In `internal/adapters/outbound/repo_sqlite/repo_test.go`:
   - Add test verifying that tasks with verification command, max turns, and output persist and reload accurately.
