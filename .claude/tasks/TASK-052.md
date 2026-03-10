---
id: TASK-052
title: "QA: tests for model routing, FindForModel, StatusNoProvider, new adapters"
role: qa
planId: PLAN-005
status: todo
dependencies: [TASK-044, TASK-045, TASK-046, TASK-047, TASK-048, TASK-049, TASK-050, TASK-051]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
PLAN-005 adds model-aware routing, StatusNoProvider, and new external provider adapters.
These must be covered by tests.  Extend the existing mock infrastructure and add targeted tests.

## Files to Read
- `internal/core/services/discovery_test.go`
- `internal/core/services/orchestrator_test.go`
- `internal/core/services/discovery.go` (after TASK-049)

## Implementation Steps

### 1. Extend `mockLLMClient` in `discovery_test.go`

Read the current struct first.  Add `activeModel string` field and update `ActiveModel()`:
```go
type mockLLMClient struct {
    alive        bool
    name         string
    code         string
    codeErr      error
    contextLimit int
    activeModel  string   // NEW
    models       []string // NEW — list of models this mock "supports"
}
func (m *mockLLMClient) ActiveModel() string                        { return m.activeModel }
func (m *mockLLMClient) GetAvailableModels() ([]string, error)      { return m.models, nil }
```
Keep the existing `ContextLimit()` method.

### 2. `TestDiscovery_FindForModel_ActiveModelMatch`
```go
func TestDiscovery_FindForModel_ActiveModelMatch(t *testing.T) {
    a := &mockLLMClient{alive: true, name: "A", activeModel: "qwen3-coder"}
    b := &mockLLMClient{alive: true, name: "B", activeModel: "codellama"}
    d := services.NewDiscoveryService(a, b)

    got, err := d.FindForModel("qwen3-coder", "")
    if err != nil {
        t.Fatalf("FindForModel: %v", err)
    }
    if got.ProviderName() != "A" {
        t.Errorf("expected provider A, got %s", got.ProviderName())
    }
}
```

### 3. `TestDiscovery_FindForModel_ModelListFallback`
```go
func TestDiscovery_FindForModel_ModelListFallback(t *testing.T) {
    a := &mockLLMClient{alive: true, name: "A", activeModel: "other", models: []string{"gpt-4o", "gpt-4"}}
    d := services.NewDiscoveryService(a)

    got, err := d.FindForModel("gpt-4o", "")
    if err != nil {
        t.Fatalf("FindForModel: %v", err)
    }
    if got.ProviderName() != "A" {
        t.Errorf("expected provider A, got %s", got.ProviderName())
    }
}
```

### 4. `TestDiscovery_FindForModel_NoProvider`
```go
func TestDiscovery_FindForModel_NoProvider(t *testing.T) {
    a := &mockLLMClient{alive: true, name: "A", activeModel: "codellama", models: []string{"codellama"}}
    d := services.NewDiscoveryService(a)

    _, err := d.FindForModel("gpt-4o", "")
    if err == nil {
        t.Fatal("expected error when model not available")
    }
}
```

### 5. `TestDiscovery_FindForModel_ProviderHintFirst`
```go
func TestDiscovery_FindForModel_ProviderHintFirst(t *testing.T) {
    a := &mockLLMClient{alive: true, name: "OpenAI", activeModel: "gpt-4o"}
    b := &mockLLMClient{alive: true, name: "Ollama", models: []string{"gpt-4o"}}
    d := services.NewDiscoveryService(b, a) // Ollama registered first

    // With hint "OpenAI" → OpenAI should be selected even though Ollama is first
    got, err := d.FindForModel("gpt-4o", "OpenAI")
    if err != nil {
        t.Fatalf("FindForModel: %v", err)
    }
    if got.ProviderName() != "OpenAI" {
        t.Errorf("expected OpenAI (hinted), got %s", got.ProviderName())
    }
}
```

### 6. `TestOrchestrator_StatusNoProvider`
```go
func TestOrchestrator_StatusNoProvider(t *testing.T) {
    repo := newMemRepo()
    llm := &mockLLMClient{alive: true, name: "mock", activeModel: "codellama", models: []string{"codellama"}}
    discovery := services.NewDiscoveryService(llm)
    orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
    defer orch.Stop()

    // Request a model that no provider has
    id, err := orch.SubmitTask(domain.Task{
        Instruction: "do work",
        ModelID:     "gpt-4o", // not available anywhere
    })
    if err != nil {
        t.Fatalf("SubmitTask: %v", err)
    }

    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        time.Sleep(200 * time.Millisecond)
        saved, _ := repo.GetByID(id)
        if saved.Status == domain.StatusNoProvider {
            return // pass
        }
        if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
            t.Fatalf("expected StatusNoProvider, got %s", saved.Status)
        }
    }
    t.Fatal("task did not reach StatusNoProvider within timeout")
}
```

### 7. `TestListProviders_IncludesActiveModel`
```go
func TestListProviders_IncludesActiveModel(t *testing.T) {
    a := &mockLLMClient{alive: true, name: "LM Studio", activeModel: "qwen3-coder", models: []string{"qwen3-coder"}}
    d := services.NewDiscoveryService(a)

    infos := d.ListProviders()
    if len(infos) != 1 {
        t.Fatalf("expected 1 provider, got %d", len(infos))
    }
    if infos[0].ActiveModel != "qwen3-coder" {
        t.Errorf("expected ActiveModel=qwen3-coder, got %q", infos[0].ActiveModel)
    }
    if len(infos[0].Models) == 0 {
        t.Error("expected non-empty Models list")
    }
}
```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/...` exits 0
- [ ] `TestDiscovery_FindForModel_ActiveModelMatch` passes
- [ ] `TestDiscovery_FindForModel_ModelListFallback` passes
- [ ] `TestDiscovery_FindForModel_NoProvider` passes
- [ ] `TestDiscovery_FindForModel_ProviderHintFirst` passes
- [ ] `TestOrchestrator_StatusNoProvider` passes
- [ ] `TestListProviders_IncludesActiveModel` passes
- [ ] All pre-existing tests continue to pass (zero regressions)

## Anti-patterns to Avoid
- NEVER add tests that depend on wall-clock time > 10s
- NEVER share mutable mock state without sync.Mutex
- Field `models []string` in mockLLMClient is only ever read (set at construction), so no mutex needed
