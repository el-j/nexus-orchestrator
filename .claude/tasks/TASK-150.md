---
id: TASK-150
title: Domain — AISession, AISessionStatus, AISessionSource types
role: architecture
planId: PLAN-022
status: todo
dependencies: []
priority: critical
estimated_effort: S
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Add the `AISession`, `AISessionStatus`, and `AISessionSource` domain types to `internal/core/domain/` so the rest of the system has a pure, framework-free foundation for tracking external AI agent sessions.

## Context
nexusOrchestrator already has `domain.Task` (unit of LLM work) and `domain.Session` (per-project conversation history). A third entity is needed: `AISession` represents an external AI agent running somewhere in the developer's environment (VS Code Copilot, Claude Desktop, any MCP client) that nexusOrchestrator has detected or been told about. This is the central new entity for PLAN-022.

The type MUST follow the exact same Go struct conventions as `internal/core/domain/task.go`:
- No framework imports (pure `time` + builtin only)
- `json:"..."` tags on every exported field (camelCase JSON)
- `ErrNotFound` sentinel already exists in task.go — reuse it
- Provide an `IsTerminal()` convenience method on `AISessionStatus`

## Scope

### Files to create
- `internal/core/domain/ai_session.go`

### Tests
- `internal/core/domain/ai_session_test.go` — table-driven tests for `IsTerminal()` and that zero-value struct is valid JSON-serialisable

## Implementation Steps
1. Create `internal/core/domain/ai_session.go` in `package domain`.
2. Define `AISessionSource` string type with constants:
   - `SessionSourceMCP = "mcp"` — registered via MCP `register_session` tool
   - `SessionSourceVSCode = "vscode"` — pushed by the nexus VS Code extension
   - `SessionSourceHTTP = "http"` — posted to `POST /api/ai-sessions`
3. Define `AISessionStatus` string type with constants:
   - `SessionStatusActive = "active"`
   - `SessionStatusIdle = "idle"`
   - `SessionStatusDisconnected = "disconnected"`
   - Add `func (s AISessionStatus) IsTerminal() bool` — returns true for `Disconnected`
4. Define `AISession` struct:
   - `ID string` `json:"id"`
   - `Source AISessionSource` `json:"source"`
   - `ExternalID string` `json:"externalId,omitempty"` — caller-provided correlation token
   - `AgentName string` `json:"agentName"` — human label e.g. "GitHub Copilot", "Claude Desktop"
   - `ProjectPath string` `json:"projectPath,omitempty"` — may be empty until user confirms
   - `Status AISessionStatus` `json:"status"`
   - `LastActivity time.Time` `json:"lastActivity"`
   - `RoutedTaskIDs []string` `json:"routedTaskIds,omitempty"` — task IDs submitted from this session
   - `CreatedAt time.Time` `json:"createdAt"`
   - `UpdatedAt time.Time` `json:"updatedAt"`
5. Write `ai_session_test.go` with at minimum:
   - `TestAISessionStatus_IsTerminal` table test covering all three statuses
   - `TestAISession_JSONRoundTrip` confirming json.Marshal → json.Unmarshal roundtrip produces equal struct

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/domain/...` exits 0
- [ ] `AISession`, `AISessionStatus`, `AISessionSource` are exported from `package domain`
- [ ] `AISessionStatus.IsTerminal()` returns true only for `SessionStatusDisconnected`
- [ ] JSON tags use camelCase (matching existing `task.go` convention)
- [ ] Zero new imports beyond `time` in the domain package

## Anti-patterns to Avoid
- NEVER import anything from `internal/adapters/` or framework packages in domain
- NEVER use `interface{}` or `any` — every field is typed
- NEVER add business logic to domain types beyond simple state predicates
- NEVER skip JSON tags — the API layer depends on them
