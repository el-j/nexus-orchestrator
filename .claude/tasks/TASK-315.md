# TASK-315: Domain types — extend DiscoveredAgent + new DiscoveredPlanFile + AISession.ModelID

**Plan:** PLAN-048
**Role:** backend
**Dependencies:** none

## Goal

Extend existing domain types and add new ones to support universal plan-file discovery and sub-agent/model enrichment. This is a pure domain layer change — no logic, no adapters, no HTTP. Everything else in PLAN-048 depends on this task.

## Changes

### 1. `internal/core/domain/discovered_agent.go`

Extend `DiscoveredAgent` with new fields (JSON-tagged, zero-value safe):

```go
type DiscoveredAgent struct {
    // ... existing fields unchanged ...
    ModelID       string    `json:"modelId,omitempty"`       // e.g. "claude-sonnet-4-6", "gpt-4o"
    WorkingDir    string    `json:"workingDir,omitempty"`    // project/cwd of the agent process
    ParentAgentID string    `json:"parentAgentId,omitempty"` // ID of parent if this is a sub-agent
    SubAgentIDs   []string  `json:"subAgentIds,omitempty"`   // IDs of spawned sub-agents
}
```

### 2. `internal/core/domain/discovered_agent.go` — new types

Add after `DiscoveredAgent`:

```go
// PlanFileKind identifies the format/tool of a discovered plan file.
type PlanFileKind string

const (
    PlanFileKindNexus      PlanFileKind = "nexus"        // .claude/orchestrator.json
    PlanFileKindClaudeTask PlanFileKind = "claude-task"  // .claude/tasks/*.md
    PlanFileKindMarkdown   PlanFileKind = "markdown"     // TASKS.md / TODO.md / PLAN.md / ROADMAP.md / AGENTS.md
    PlanFileKindCursor     PlanFileKind = "cursor"       // .cursorrules / .cursor/rules/*.mdc
    PlanFileKindMCPConfig  PlanFileKind = "mcp-config"   // mcp.json / .mcp.json / claude_desktop_config.json
    PlanFileKindCrewAI     PlanFileKind = "crewai"       // crew.py / agents.py containing crewai imports
    PlanFileKindClaude     PlanFileKind = "claude"       // CLAUDE.md / copilot-instructions.md / .github/agents/*.agent.md
)

// DiscoveredPlanFile represents a detected orchestration/plan/task file.
type DiscoveredPlanFile struct {
    ID          string       `json:"id"`                    // stable: sha1 of path
    Path        string       `json:"path"`                  // absolute filesystem path
    Kind        PlanFileKind `json:"kind"`
    Format      string       `json:"format"`                // "json", "md", "yaml", "py"
    ProjectPath string       `json:"projectPath"`           // nearest git root or parent dir
    Summary     string       `json:"summary,omitempty"`     // first ~200 chars of content
    LastModified time.Time   `json:"lastModified"`
    IsActive    bool         `json:"isActive"`              // modified within last 24h
}
```

### 3. `internal/core/domain/ai_session.go`

Add `ModelID string` field to `AISession` (between existing fields, json-tagged `modelId,omitempty`).

### 4. `internal/core/ports/ports.go`

Add to `AgentScanner` interface:

```go
// ScanPlanFiles scans the provided root directories for plan/task/orchestration files.
ScanPlanFiles(ctx context.Context, rootPaths []string) ([]domain.DiscoveredPlanFile, error)
```

Add to `Orchestrator` interface (inbound port):

```go
// GetDiscoveredPlanFiles returns plan/task/orchestration files found near projectPath.
GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error)
```

## Acceptance Criteria

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] All existing tests still pass (no logic changes, only additive struct fields)
- [ ] `DiscoveredPlanFile` and `PlanFileKind` are in the `domain` package
- [ ] `AISession.ModelID` field exists with correct json tag
- [ ] `AgentScanner` interface has `ScanPlanFiles`
- [ ] `Orchestrator` interface has `GetDiscoveredPlanFiles`
- [ ] No adapter or service files are modified in this task (domain + ports only)
