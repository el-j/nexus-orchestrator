---
id: TASK-465
plan: PLAN-061
status: done
wave: 3
priority: 3
---

# TASK-465: Investigate and implement fs_writeback adapter

**Problem:** `internal/adapters/outbound/fs_writeback/` contains zero Go source files. The architecture documentation lists it as an outbound adapter responsible for writing AI-generated results back to task files on disk. It is unclear whether any code path actually references this port or whether the directory is a placeholder that was never implemented.

**Fix:** Investigate whether any port interface, orchestrator option, or wiring code references `fs_writeback`. If no code depends on it: remove the empty directory and delete any port or architecture reference. If a port interface (`FileWriteback` or similar) exists and is referenced: implement the adapter as a minimal `Write(taskID, content string) error` that appends or overwrites the `output` section of the task's `.claude/tasks/TASK-NNN.md` file.

**Files:**

- `internal/adapters/outbound/fs_writeback/` (to be implemented or removed)
- `internal/core/ports/ports.go` (audit for orphaned interface)
- `main.go`, `cmd/nexus-daemon/main.go` (audit for orphaned wiring)
- Architecture docs if reference needs updating

## Checklist

- [ ] Search codebase for all references to `fs_writeback`, `FileWriter`, `FileWriteback`, and `writeback` across `*.go` files
- [ ] Check `internal/core/ports/ports.go` for any port interface that names file writeback
- [ ] Check all wiring points (`main.go`, `cmd/nexus-daemon/main.go`, `bootstrap/`) for any `WithFileWriter` or equivalent option
- [ ] **If no references exist:** delete `internal/adapters/outbound/fs_writeback/` directory; confirm build still passes
- [ ] **If references exist:** implement `internal/adapters/outbound/fs_writeback/adapter.go` with a struct that satisfies the port interface; write the output to the appropriate task file, appending under a `## Output` heading if not present
- [ ] Add at least one test (using a temp directory) if the adapter is implemented
- [ ] Run `go build ./...` and `CGO_ENABLED=1 go test -race ./...` clean

## Acceptance Criteria

- No empty directories remain in `internal/adapters/outbound/`
- Either: `fs_writeback` is implemented and wired, tested, and documented; OR: directory and all orphaned references are removed
- Build and tests pass
