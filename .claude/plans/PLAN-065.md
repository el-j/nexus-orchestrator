# PLAN-065: Desktop Launch UX, Branding, and Download Detection

**Status:** Completed
**Completed:** 2026-04-13T14:00:00Z

## Tasks

| ID       | Title                                            | Role    | Status |
| -------- | ------------------------------------------------ | ------- | ------ |
| TASK-505 | Make Wails close semantics fully graceful        | backend | done   |
| TASK-506 | Replace default Wails logo with branded icons    | backend | done   |
| TASK-507 | Fix GitHub Pages download architecture detection | devops  | done   |
| TASK-508 | Validate and close PLAN-065 changes              | qa      | done   |

## Summary

**TASK-505 — Wails close semantics** (verified already implemented):

- Tray adapter `Enabled()` returns `false` (stub) → `trayEnabled = false`
- `HideWindowOnClose: trayEnabled` → `false` → window close actually quits
- `OnBeforeClose` returns `false` when tray disabled, allowing process exit
- `OnShutdown` calls `cancelHTTP()` → HTTP + MCP servers stop via context cancellation
- No `os.Exit` anywhere in `tray.go`

**TASK-506 — Branded icon assets** (verified already in place):

- `internal/adapters/inbound/tray/icon.go` embeds `icon.png` via `//go:embed` (28 KB)
- `build/appicon.png` replaced with project branding (1.1 MB)
- `go build ./cmd/nexus-daemon/...` passes cleanly

**TASK-507 — macOS architecture detection fix** (implemented):

- `docs/src/views/DownloadsView.vue` `onMounted` handler patched
- When all ARM heuristics fail (userAgentData, UA string, WebGL GPU renderer), `detectedKey` is now set to `''` instead of `'mac-intel'`
- Apple Silicon users on Safari/Firefox no longer see the Intel build falsely highlighted
- Confirmed ARM detection still works (userAgentData → UA string → WebGL `Apple M…` renderer)

**TASK-508 — Validation** (passed):

- `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/...` → clean
- Pre-existing `go vet` test failures are from PLAN-066 BrainService signature changes, not from PLAN-065 edits
