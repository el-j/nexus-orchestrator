# TASK-337: Wire activity service into entry points

**Plan:** PLAN-050 · Wave 3
**Status:** DONE
**Agent:** Senior Developer

## Description

Wire the ActivityService and all activity readers into both entry points:

1. `cmd/nexus-daemon/main.go`: create activity readers (claude, continue, network), instantiate ActivityService, call Start(), defer Stop()
2. `app.go` (Wails): same wiring, expose `GetRecentActivities()` and `GetTimeline()` as Wails bindings

Register activity readers only when their data source directories exist (e.g., skip Continue reader if ~/.continue/ doesn't exist).

## Acceptance

- Daemon starts with activity readers
- Wails app starts with activity readers
- Graceful shutdown stops activity service
- Missing directories handled gracefully (skip reader, log warning)

## Completed

Wired `ActivityService` and all four readers into daemon and Wails entry points with graceful shutdown and missing-directory guards.
