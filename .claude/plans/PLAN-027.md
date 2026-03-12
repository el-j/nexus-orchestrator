# PLAN-027: Full Build Consolidation & Runtime Bug Fixes

**Status:** Completed  
**Completed:** 2026-03-12T12:00:00Z

## Tasks

| ID | Title | Role | Completed |
|----|-------|------|-----------|
| TASK-193 | Centralize dist paths + make build-all completeness | devops | 2026-03-12T12:00:00Z |
| TASK-194 | Update GitHub workflows for new dist output structure | devops | 2026-03-12T12:00:00Z |
| TASK-195 | Fix AI Sessions: CORS headers + Wails app.go bindings + wails.ts | backend | 2026-03-12T12:00:00Z |
| TASK-196 | Fix Dashboard layout: TaskSubmitForm visibility + TaskQueue empty state | frontend | 2026-03-12T12:00:00Z |
| TASK-197 | Fix VS Code SessionMonitor: Copilot detection retry with backoff | vscode | 2026-03-12T12:00:00Z |
| TASK-198 | QA: verify all fixes end-to-end | qa | 2026-03-12T12:00:00Z |

## Summary

Consolidated the entire build system and resolved a set of runtime bugs discovered when running the v0.9.x desktop app:

- **Dist path centralization**: unified where CLI/daemon/VSIX artifacts land so Makefile `build-all` reliably produces every deliverable without manual path hunting.
- **GitHub workflows**: updated `ci.yml` and `publish.yml` to reference the new consolidated dist paths for upload/release artifact steps; fixed the Makefile `vsce` invocation to use `npx @vscode/vsce`.
- **CORS on AI-session endpoints**: `POST/GET/DELETE /api/ai-sessions` handlers were missing CORS headers — added via chi middleware configuration so the Wails embedded webview can call them.
- **Wails `app.go` + `wails.ts`**: wired `RegisterAISession`, `ListAISessions`, `DeregisterAISession` through the Wails Go-to-JS bridge and added matching TypeScript helpers in `frontend/src/types/wails.ts`.
- **Dashboard layout**: `TaskSubmitForm` was invisible because `expanded` defaulted to `false`; corrected and verified `TaskQueue` empty-state rendering.
- **SessionMonitor backoff**: VS Code extension retry logic for Copilot model detection was tight-looping on failure; replaced with exponential backoff so the extension does not spam error logs if Copilot is slow to initialise.
