# TASK-316: sys_scanner ScanPlanFiles — detect 8+ plan file patterns

**Plan:** PLAN-048
**Role:** backend
**Dependencies:** TASK-315

## Goal

Implement `ScanPlanFiles` in the `sys_scanner` package so nexusOrchestrator can discover plan/task/orchestration files of any format across project directories — not just `orchestrator.json`.

## Changes

### `internal/adapters/outbound/sys_scanner/plan_scanner.go` (new file)

```go
package sys_scanner

// ScanPlanFiles implements ports.AgentScanner.ScanPlanFiles.
// It scans rootPaths (and their parents up to home dir) for known plan file patterns.
func (s *Scanner) ScanPlanFiles(ctx context.Context, rootPaths []string) ([]domain.DiscoveredPlanFile, error)
```

**Detection patterns** (scan each rootPath and its `.claude/`, `.cursor/`, `.github/agents/` subdirs):

| Pattern                                                                 | Kind        | Format | Notes                           |
| ----------------------------------------------------------------------- | ----------- | ------ | ------------------------------- |
| `**/orchestrator.json` inside `.claude/`                                | nexus       | json   | Check `activePlanId` key exists |
| `.claude/tasks/*.md`                                                    | claude-task | md     | Each file is a task             |
| `AGENTS.md`, `CLAUDE.md`, `copilot-instructions.md`                     | claude      | md     |                                 |
| `.github/agents/*.agent.md`                                             | claude      | md     |                                 |
| `.cursorrules`                                                          | cursor      | md     |                                 |
| `.cursor/rules/*.mdc`                                                   | cursor      | md     |                                 |
| `TASKS.md`, `tasks.md`, `TODO.md`, `PLAN.md`, `ROADMAP.md`, `AGENTS.md` | markdown    | md     |                                 |
| `mcp.json`, `.mcp.json`                                                 | mcp-config  | json   |                                 |
| `claude_desktop_config.json` (anywhere in project)                      | mcp-config  | json   |                                 |
| `crew.py`, `agents.py` containing `from crewai` or `import crewai`      | crewai      | py     | Grep first 20 lines             |

**ID generation:** `fmt.Sprintf("%x", sha1.Sum([]byte(path)))[:12]`

**Summary:** read first 300 bytes of file, strip non-printable chars, trim to 200 chars.

**IsActive:** `time.Since(fi.ModTime()) < 24*time.Hour`

**ProjectPath:** walk up from file path until finding `.git` dir or home dir, use that.

### `internal/adapters/outbound/sys_scanner/scanner.go`

Ensure `Scanner` struct satisfies the updated `ports.AgentScanner` interface (add `_ ports.AgentScanner = (*Scanner)(nil)` assertion).

### Tests

Add `TestScanPlanFiles` in `internal/adapters/outbound/sys_scanner/agent_scanner_test.go` (or new `plan_scanner_test.go`):

- Create a temp dir with `.claude/orchestrator.json`, `TASKS.md`, `.cursorrules`
- Call `ScanPlanFiles` with that temp dir as rootPath
- Assert 3 results returned with correct kinds/formats

## Acceptance Criteria

- [ ] `ScanPlanFiles` implemented and compiles
- [ ] Detects all 10+ file patterns listed above
- [ ] Returns correct `Kind`, `Format`, `Summary`, `IsActive`, `ProjectPath`
- [ ] `go vet ./...` clean
- [ ] `go test ./internal/adapters/outbound/sys_scanner/ -race -count=1` passes including new test
