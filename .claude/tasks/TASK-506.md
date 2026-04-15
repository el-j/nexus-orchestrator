---
id: TASK-506
plan: PLAN-065
status: done
wave: 2
priority: 2
---

# TASK-506: Replace default Wails logo with branded icon assets

## Description

The desktop app still ships with the default Wails icon and a placeholder tray icon byte-array. This task replaces those assets with project-branded icons and wires icon usage into build/runtime paths.

## Checklist

- [ ] Generate and commit a branded app icon PNG replacing `build/appicon.png`
- [ ] Replace tray placeholder bytes with embedded PNG asset
- [ ] Ensure icon-related config remains compatible with `wails build`

## Files

- `build/appicon.png`
- `internal/adapters/inbound/tray/icon.go`
- `internal/adapters/inbound/tray/icon.png` (new)

## Acceptance Criteria

- Wails desktop builds no longer use the default Wails icon
- Tray icon bytes come from a committed image asset via `go:embed`
- `go build ./cmd/nexus-daemon/...` still passes after icon refactor
