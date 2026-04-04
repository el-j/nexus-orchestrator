# TASK-259 — Fix go embed: create build/frontend placeholder

**Plan**: PLAN-039  
**Status**: done  
**Agent**: Senior Developer

## Problem
`//go:embed all:build/frontend` in `main.go` fails on any clean checkout because
`build/frontend/` is gitignored and doesn't exist. Breaks `go vet ./...` and `golangci-lint`.

## Fix
1. Change `.gitignore` entry `build/frontend/` → `build/frontend/*` with exception `!build/frontend/.gitkeep`  
   (the directory-level ignore blocks even explicit exceptions, so it must become a glob)
2. Create `build/frontend/.gitkeep`

## Files Changed
- `.gitignore` — updated embed directory ignore pattern
- `build/frontend/.gitkeep` — new placeholder file (empty)

## Result
`go vet ./...` and `golangci-lint` pass on fresh clone. Actual built frontend content remains gitignored.
