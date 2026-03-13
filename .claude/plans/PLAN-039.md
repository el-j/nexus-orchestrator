# PLAN-039 — Fix CI Failures from Build-Path Unification

**Status**: active  
**Created**: 2026-03-13  
**Triggered by**: CI run #53 ("refactor: remove unused frontend assets and update build paths")

## Root Cause Analysis

PLAN-038 unified all build output under `build/`. Two issues surfaced in CI:

### Issue A — `go vet ./...` + `golangci-lint` fail
`main.go:35:12: pattern all:build/frontend: no matching files found`

The `//go:embed all:build/frontend` directive requires the directory to exist at compile time.
`build/frontend/` was gitignored so it never exists on a fresh CI checkout.
Affects: **Vet** job, **Lint** job.

### Issue B — Build VS Code Extension fails
`npm error Missing: esbuild@0.27.4 from lock file`

`vscode-extension/package-lock.json` was generated with an older resolution tree.
`vitest@^4.1.0` transitively requires `esbuild@^0.27.0`; the lock file pre-dates this constraint.
Affects: **Build VS Code Extension** job.

## Fix Strategy

| Task | Fix | Scope |
|------|-----|-------|
| TASK-259 | Create `build/frontend/.gitkeep`, update `.gitignore` exception | Local + CI |
| TASK-260 | Regenerate `vscode-extension/package-lock.json` via `npm install` | CI |

## Success Criteria
- `go vet ./...` passes on fresh clone with no prior `make build-frontend`  
- `golangci-lint` passes  
- `npm ci` in `vscode-extension/` passes  

## Task IDs
- TASK-259
- TASK-260
