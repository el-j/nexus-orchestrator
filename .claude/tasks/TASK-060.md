---
id: TASK-060
title: "Security: path traversal guard in fs_writer"
role: backend
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
The `fs_writer` package has a CRITICAL path traversal vulnerability. `filepath.Join(projectPath, targetFile)` does not prevent `targetFile` values like `../../etc/passwd` from escaping the project directory. Both `WriteCodeToFile` and `ReadContextFiles` are affected. This allows arbitrary file read/write on the host.

## Files to Read
- `internal/adapters/outbound/fs_writer/writer.go`
- `internal/adapters/outbound/fs_writer/writer_test.go`

## Implementation Steps
1. In `WriteCodeToFile`, after `filepath.Join`, resolve the absolute path with `filepath.Abs()`, then verify the result starts with the resolved `projectPath` using `strings.HasPrefix`. Return a descriptive error if the path escapes.
2. In `ReadContextFiles`, apply the same containment check for each file before reading.
3. Use `filepath.Clean` on both `projectPath` and the joined result before comparison.
4. Add test cases for path traversal attempts: `../../etc/passwd`, `../sibling/file`, absolute paths like `/etc/passwd`, and paths with `..` in the middle.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `WriteCodeToFile("project", "../../etc/passwd", "x")` returns an error
- [ ] `ReadContextFiles("project", []string{"../../etc/passwd"})` returns an error
- [ ] Tests cover traversal via `..`, absolute paths, and normal valid paths still work

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
