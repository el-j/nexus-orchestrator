---
id: TASK-563
planId: PLAN-072
title: 'Real-Time Workspace Change Invalidation Watcher'
role: backend
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:35:00Z
---

# TASK-563 — Real-Time Workspace Change Invalidation Watcher

## Context

Static knowledge bases rapidly rot when developers or external agents (Aider, Claude Code) edit files directly on disk. A filesystem watcher (`fsnotify`) monitoring active workspace roots ensures modified files are automatically re-indexed into SQLite FTS5 without human intervention.

## Completed Work

1. **Created `internal/adapters/outbound/fs_watcher/watcher.go`**:
   - Built recursive filesystem watcher using `github.com/fsnotify/fsnotify v1.10.1`.
   - Filters out non-source/ignored paths (`.git`, `node_modules`, `vendor`, `build`, `dist`, `.claude`, `target`, `__pycache__`).
   - Supports source extensions (`.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.rs`, `.py`, `.md`, `.json`) and knowledge files (`CLAUDE.md`, `README.md`, `GEMINI.md`, `AGENTS.md`).
   - Implements thread-safe 300ms debounce timer window to prevent thrashing during fast git checkouts or bulk edits.
   - Automatically auto-ingests Markdown changes via `brain.IngestFromFile`.
   - Supports dynamic detection and watching of newly created subdirectories.
2. **Wired Watcher into Entry Points**:
   - `cmd/nexus-daemon/main.go`: instantiates `fs_watcher` with `brainSvc` and watches current working directory on startup.
   - `app.go` & `main.go`: wired `fs_watcher` into GUI `App` with `withFsWatcher`, `WatchWorkspace`, and `UnwatchWorkspace` bindings.
3. **Tests**:
   - `internal/adapters/outbound/fs_watcher/watcher_test.go` with 100% pass on `ShouldProcess` filtering, rapid write debouncing, brain auto-ingestion, and unwatching.
