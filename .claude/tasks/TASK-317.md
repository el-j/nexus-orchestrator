# TASK-317: sys_scanner sub-agent enrichment from ~/.claude/projects JSONL

**Plan:** PLAN-048
**Role:** backend
**Dependencies:** TASK-315

## Goal

Add a 6th probe to `ScanAgents` that reads Claude Code's session JSONL files from `~/.claude/projects/` to discover active Claude CLI / sub-agent sessions, their models, working directories, parent-child relationships, and whether they are currently running (recently modified JSONL).

## Background

Claude Code stores each conversation as a `.jsonl` file under:

```
~/.claude/projects/<base64-encoded-path>/<session-uuid>.jsonl
```

Each line is a JSON message. Key fields:

- `"agentId"` — unique ID for this agent instance
- `"model"` — e.g. `"claude-sonnet-4-6"` (present on assistant messages)
- `"isSidechain": true` — marks sub-agent messages (launched via Agent tool)
- `"parentUuid"` — links sub-agent message to parent session message
- `"cwd"` — working directory from user messages
- File `mtime` — if modified within last 5 minutes, session is "active"

## Changes

### New probe in `internal/adapters/outbound/sys_scanner/agent_scanner.go`

Add `probeClaudeSubAgents` as a 6th probe in `ScanAgents`:

```go
func (s *Scanner) probeClaudeSubAgents(ctx context.Context) []domain.DiscoveredAgent
```

**Logic:**

1. Find `~/.claude/projects/` — skip if not present
2. Walk all `*.jsonl` files recursively
3. For each file:
   - Check `mtime`: if older than 2 hours, skip (stale session)
   - Read lines (limit 500 lines to avoid huge files)
   - Extract: first `agentId` seen, first `model` seen, `cwd` from first user message, `isSidechain` flag, `parentUuid`
   - Decode the base64 directory name to get `workingDir` (the project path)
4. Build `DiscoveredAgent`:
   - `ID`: agentId (or file basename if missing)
   - `Kind`: `AgentKindClaudeCLI`
   - `Name`: `"Claude Code"` + working dir basename
   - `ModelID`: extracted model string
   - `WorkingDir`: decoded project path
   - `IsRunning`: mtime < 5 minutes
   - `ParentAgentID`: if `isSidechain=true` and parentUuid found, set to parent's agentId
   - `DetectionMethod`: `"claude-session-file"`

**Merge logic** in `ScanAgents`: after all probes complete, for each agent with `ParentAgentID` set, find the parent in the merged map and append this agent's ID to `parent.SubAgentIDs`.

### Also enrich `probeClaudeConfig` and `probeAgentProcesses`

When finding a Claude CLI process via `pgrep`, also try to read its `WorkingDir` from `/proc/<pid>/cwd` (Linux) or `lsof -p <pid> -d cwd` (macOS) and set `ModelID` from the process command line if `--model` flag is present.

## Acceptance Criteria

- [ ] `probeClaudeSubAgents` added and wired into `ScanAgents` probes slice
- [ ] Correctly identifies sub-agents (`isSidechain=true`) and sets `ParentAgentID`
- [ ] Parent agents have `SubAgentIDs` populated after merge
- [ ] `ModelID` populated from JSONL when present
- [ ] `IsRunning=true` only for sessions with mtime < 5 minutes
- [ ] Handles missing `~/.claude/projects/` gracefully (no error, empty result)
- [ ] `go vet ./...` clean, all existing tests pass
