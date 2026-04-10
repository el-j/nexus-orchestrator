# PLAN-065 — Desktop Launch UX, Branding, and Download Detection

**Status:** active
**Created:** 2026-04-10
**Author:** copilot

## Summary

This plan hardens desktop release UX for end users by fixing Wails shutdown semantics (no invisible background process and no forced hard-exit paths), replacing default Wails branding with project-specific icon assets, and improving GitHub Pages download recommendation accuracy for Apple Silicon and mixed user-agent environments.

## Task Map

| Task     | Layer          | Priority | Description |
| -------- | -------------- | -------- | ----------- |
| TASK-505 | Desktop / Wails | Critical | Ensure close-to-quit lifecycle and graceful shutdown (no hidden process, no os.Exit quit path) |
| TASK-506 | Branding / Assets | High | Replace default app logo and tray icon placeholders with branded assets wired into builds |
| TASK-507 | Docs / Web      | High | Fix download page architecture detection and recommendation highlighting behavior |
| TASK-508 | Validation       | Critical | Verify desktop build + docs build + targeted tests, then document outcomes |

## Waves

### Wave 1 — Lifecycle correctness (TASK-505)

Fix Wails close/quit behavior so app exits cleanly and all embedded services are stopped.

### Wave 2 — Branding assets (TASK-506)

Replace default icon assets and wire them into Wails + tray icon code paths.

### Wave 3 — Download detection (TASK-507)

Improve platform and architecture detection logic for macOS Apple Silicon and uncertain states.

### Wave 4 — Validation and closure (TASK-508)

Run compile/build checks and confirm user-visible outcomes before closing tasks.
