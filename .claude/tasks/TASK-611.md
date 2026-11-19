---
id: TASK-611
planId: PLAN-075
title: 'Fix Wails desktop getUsage() relative fetch and complete attached-daemon parity in app.go and httpapi_client'
role: coder
wave: 1
status: done
createdAt: 2026-10-19
---

# TASK-611 — Fix Wails desktop getUsage() relative fetch and complete attached-daemon parity in app.go and httpapi_client

1. **`frontend/src/types/wails.ts`**:
   - Prepend `await resolveServerUrl()` in `request<T>(method, path, body)` when `path` starts with `'/'` so HTTP fallback calls (`getUsage`, etc.) reach `http://127.0.0.1:63987` when running inside the Wails desktop WebView (`wails://wails` / `http://wails.localhost`).
2. **`internal/adapters/outbound/httpapi_client`**:
   - Add `IngestContent(ctx, projectPath, name, content)` (`POST /api/brain/ingest-content`) to `BrainClient`.
   - Add `GetRecentActivities`, `GetActivityTimeline`, `GetUsage`, and `GetRouting` to `httpapi_client.Client`.
3. **`app.go` & `main.go`**:
   - Change `App.activitySvc` from concrete `*services.ActivityService` to an interface satisfied by both `*services.ActivityService` and `*httpapi_client.Client`, and wire `.withActivityService(client)` in `attachOrStart` (`main.go`) so attached-daemon desktop mode supports Activity, Brain content upload, Usage, and Routing.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated.
