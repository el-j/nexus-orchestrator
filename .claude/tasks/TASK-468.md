---
id: TASK-468
plan: PLAN-061
status: done
wave: 5
priority: 5
---

# TASK-468: Remove dead wailsbind package

**Problem:** `internal/adapters/inbound/wailsbind/bind.go` contains 8 methods that duplicate Wails binding logic. The file is never imported by any other package — it is orphaned dead code that creates confusion about whether it or `app.go` is the canonical Wails binding surface. Having two apparent binding layers makes the codebase harder to navigate and maintain.

**Fix:** Verify there are zero imports of the `wailsbind` package across the codebase. If confirmed, delete the package directory. If any import is found, either wire it properly (replacing the duplicate in `app.go`) or merge its content into `app.go` and then delete the package.

**Files:**

- `internal/adapters/inbound/wailsbind/bind.go` (to be deleted)
- `internal/adapters/inbound/wailsbind/` directory (to be removed)
- `app.go` (reference; authoritative Wails binding — no changes expected)

## Checklist

- [ ] Run `grep -r "wailsbind" --include="*.go" .` to confirm zero imports of the package
- [ ] Run `grep -r "wailsbind" --include="*.go" --include="*.ts" .` to check for any frontend codegen references
- [ ] If zero references: delete `internal/adapters/inbound/wailsbind/bind.go` and the directory
- [ ] If references exist: assess whether to wire or merge; document the decision here before proceeding
- [ ] Run `go build ./...` after deletion to confirm nothing breaks
- [ ] Run `CGO_ENABLED=1 go test -race ./...` to confirm no tests referenced the package

## Acceptance Criteria

- `internal/adapters/inbound/wailsbind/` directory does not exist
- `go build ./...` and all tests pass after removal
- No compile errors or missing symbol references
