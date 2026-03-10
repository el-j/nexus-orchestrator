---
id: TASK-044
title: "Architecture: domain + port changes for model-aware routing"
role: architecture
planId: PLAN-005
status: todo
dependencies: []
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
Enable per-task model/provider assignment and smart failover routing.
Three small changes: (1) two new fields on `domain.Task`, (2) `StatusNoProvider` constant,
(3) `ActiveModel() string` method on `LLMClient` port, (4) extend `ProviderInfo` with model metadata.

## Files to Read
- `internal/core/domain/task.go`
- `internal/core/ports/ports.go`

## Implementation Steps

### 1. `internal/core/domain/task.go`

Add after `StatusTooLarge`:
```go
// StatusNoProvider is set when no registered provider has the requested model
// available or all are unreachable.  The task is never sent to any LLM.
StatusNoProvider TaskStatus = "NO_PROVIDER"
```

Add two fields to `Task` struct after `ContextFiles`:
```go
// ModelID constrains which model must be used.  Empty means "use any active provider".
ModelID      string `json:"modelId,omitempty"`
// ProviderHint is a hint (provider name) to prefer when multiple providers carry the
// same model.  Ignored when empty.
ProviderHint string `json:"providerHint,omitempty"`
```

### 2. `internal/core/ports/ports.go`

Add `ActiveModel() string` to the `LLMClient` interface between `ProviderName()` and `GetAvailableModels()`:
```go
// ActiveModel returns the model ID currently loaded / configured on this provider.
// Returns empty string when unknown.
ActiveModel() string
```

Extend `ProviderInfo` struct:
```go
type ProviderInfo struct {
    Name        string   `json:"name"`
    Active      bool     `json:"active"`
    ActiveModel string   `json:"activeModel,omitempty"`
    Models      []string `json:"models,omitempty"`
}
```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `StatusNoProvider` constant exists with value `"NO_PROVIDER"`
- [ ] `Task.ModelID` and `Task.ProviderHint` fields exist with correct json tags
- [ ] `LLMClient.ActiveModel() string` is declared in ports.go
- [ ] `ProviderInfo` has `ActiveModel string` and `Models []string` fields

## Anti-patterns to Avoid
- NEVER add logic or imports to domain/ports layers — types and interfaces only
- NEVER use pointer receivers in interface declarations
