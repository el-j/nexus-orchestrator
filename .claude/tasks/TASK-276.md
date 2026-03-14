# TASK-276 — Domain: `DiscoveredAgent` type + `AISession` field extensions

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · domain (`internal/core/domain/`)  
**Depends on:** —  

## Objective

Extend the domain layer with the types needed for PLAN-044. No framework imports anywhere in this package.

## Changes

### New file: `internal/core/domain/discovered_agent.go`

```go
package domain

import "time"

type AgentKind string

const (
    AgentKindClaudeCLI     AgentKind = "claude-cli"
    AgentKindClaudeDesktop AgentKind = "claude-desktop"
    AgentKindAntigravity   AgentKind = "antigravity"
    AgentKindCline         AgentKind = "cline"
    AgentKindContinue      AgentKind = "continue"
    AgentKindCodeGPT       AgentKind = "codegpt"
    AgentKindCursor        AgentKind = "cursor"
    AgentKindCopilot       AgentKind = "copilot"
    AgentKindAichat        AgentKind = "aichat"
    AgentKindGeneric       AgentKind = "generic"
)

type DiscoveredAgent struct {
    ID              string    `json:"id"`
    Kind            AgentKind `json:"kind"`
    Name            string    `json:"name"`
    DetectionMethod string    `json:"detectionMethod"`
    ProcessName     string    `json:"processName,omitempty"`
    CLIPath         string    `json:"cliPath,omitempty"`
    ConfigPath      string    `json:"configPath,omitempty"`
    MCPEndpoint     string    `json:"mcpEndpoint,omitempty"`
    IsRunning       bool      `json:"isRunning"`
    LastSeen        time.Time `json:"lastSeen"`
}
```

### Modifications to `internal/core/domain/ai_session.go`

Add to source constants:
```go
SessionSourceVSCodeDiscovered AISessionSource = "vscode-discovered"
```

Add to `AISession` struct:
```go
DelegatedToNexus    bool       `json:"delegatedToNexus"`
DelegationTimestamp *time.Time `json:"delegationTimestamp,omitempty"`
AgentCapabilities   []string   `json:"agentCapabilities,omitempty"`
DetectionMethod     string     `json:"detectionMethod,omitempty"`
```

## Acceptance Criteria

- `go build ./internal/core/domain/...` succeeds
- `go vet ./internal/core/domain/...` clean
- Existing `ai_session_test.go` passes (JSON round-trip) — extend the test to cover new fields
