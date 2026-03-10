---
id: TASK-043
title: "QA: tests for token estimation, pre-flight guard, and adapter ContextLimit()"
role: qa
planId: PLAN-004
status: todo
dependencies: [TASK-039, TASK-040, TASK-041, TASK-042]
createdAt: 2026-03-10T00:00:00.000Z
---

## Context
The context-window guard is a critical safety mechanism.  It must be exhaustively tested: correct token estimation, correct abort behaviour (StatusTooLarge), bypass behaviour when limit=0, and session-history accumulation still works after the guard passes.

## Files to Read
- `internal/core/services/orchestrator_test.go` (existing stubs: memRepo, noopWriter, mockLLMClient, chatTrackingLLM)
- `internal/core/services/discovery_test.go` (mockLLMClient definition)
- `internal/core/services/orchestrator.go` (estimateTokens, processNext)

## Implementation Steps

### 1. Extend `mockLLMClient` in `discovery_test.go`
Add a `contextLimit` field and implement `ContextLimit()`:
```go
type mockLLMClient struct {
    alive        bool
    name         string
    code         string
    codeErr      error
    contextLimit int // 0 = no limit (default); > 0 = enforced
}
func (m *mockLLMClient) ContextLimit() int { return m.contextLimit }
```
This is a non-breaking extension — existing tests construct `mockLLMClient{alive: true, name: "mock", code: "..."}` and the zero-value `contextLimit: 0` means "no limit enforced", so all existing tests continue to pass.

### 2. Add table-driven unit test for `estimateTokens` (in `orchestrator_test.go`)
```go
func TestEstimateTokens(t *testing.T) {
    cases := []struct {
        name     string
        messages []domain.Message
        wantMin  int // estimated >= wantMin
        wantMax  int // estimated <= wantMax
    }{
        {"empty", nil, 0, 0},
        {"single short", []domain.Message{{Role: "user", Content: "hi"}}, 1, 10},
        {"4096-char content", []domain.Message{{Role: "user", Content: strings.Repeat("a", 4096)}}, 1020, 1030},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := services.EstimateTokens(tc.messages)
            if got < tc.wantMin || got > tc.wantMax {
                t.Errorf("EstimateTokens() = %d; want [%d, %d]", got, tc.wantMin, tc.wantMax)
            }
        })
    }
}
```
**NOTE**: Export `EstimateTokens` from the `services` package (rename `estimateTokens` → `EstimateTokens`) so the `_test` package can call it.  Or keep it unexported and use the `package services` test variant — choose whichever the existing tests already use.

Actually — since tests are in `package services_test` (external test package), you must either:  
a. Export `EstimateTokens` (preferred — it's a useful utility), OR  
b. Write tests for the observable side-effect (StatusTooLarge) only.

Use approach **b** (black-box via orchestrator) to avoid changing the public API unnecessarily.

### 3. Add `TestOrchestrator_PreFlight_TooLarge` test
```go
func TestOrchestrator_PreFlight_TooLarge(t *testing.T) {
    repo := newMemRepo()
    // LLM with a tiny context limit of 10 tokens — any real instruction will exceed it
    llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 10}
    discovery := services.NewDiscoveryService(llm)
    orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
    defer orch.Stop()

    id, err := orch.SubmitTask(domain.Task{
        Instruction: strings.Repeat("x", 200), // ~50 tokens >> 10-512 (clamped to 0, check skipped? No: limit-512 < 0 means always block)
    })
    if err != nil {
        t.Fatalf("SubmitTask: %v", err)
    }

    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        time.Sleep(200 * time.Millisecond)
        saved, _ := repo.GetByID(id)
        if saved.Status == domain.StatusTooLarge {
            if !strings.Contains(saved.Logs, "context too large") {
                t.Errorf("expected 'context too large' in Logs, got: %s", saved.Logs)
            }
            return // pass
        }
        if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
            t.Fatalf("expected StatusTooLarge but got %s", saved.Status)
        }
    }
    t.Fatal("task did not reach StatusTooLarge within timeout")
}
```

### 4. Add `TestOrchestrator_PreFlight_NoLimitSkipsCheck` test
When `ContextLimit()` returns 0, the task should process normally:
```go
func TestOrchestrator_PreFlight_NoLimitSkipsCheck(t *testing.T) {
    repo := newMemRepo()
    llm := &mockLLMClient{alive: true, name: "mock", code: "package main", contextLimit: 0}
    discovery := services.NewDiscoveryService(llm)
    orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
    defer orch.Stop()

    id, _ := orch.SubmitTask(domain.Task{Instruction: strings.Repeat("x", 10000)})
    waitCompleted(t, repo, id, 10*time.Second) // must reach COMPLETED, not TOO_LARGE
}
```

### 5. Add `TestOrchestrator_PreFlight_WithinLimit_Completes` test
When estimated tokens ≤ limit - 512, normal completion should occur:
```go
func TestOrchestrator_PreFlight_WithinLimit_Completes(t *testing.T) {
    repo := newMemRepo()
    llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 8192}
    discovery := services.NewDiscoveryService(llm)
    orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
    defer orch.Stop()

    id, _ := orch.SubmitTask(domain.Task{Instruction: "short instruction"})
    waitCompleted(t, repo, id, 10*time.Second)
}
```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/...` exits 0
- [ ] `TestOrchestrator_PreFlight_TooLarge` passes (StatusTooLarge reached, Logs contain "context too large")
- [ ] `TestOrchestrator_PreFlight_NoLimitSkipsCheck` passes (StatusCompleted when limit=0)
- [ ] `TestOrchestrator_PreFlight_WithinLimit_Completes` passes (StatusCompleted when within budget)
- [ ] All pre-existing tests continue to pass (no regressions)

## Anti-patterns to Avoid
- NEVER use `time.Sleep` longer than necessary — poll with 200ms intervals, timeout at 10s
- NEVER share `mock` state without `sync.Mutex` — the `contextLimit` field read from the worker goroutine must be safe (it is okay if it is read-only after construction, but verify)
- NEVER add tests that depend on timing — use the waitCompleted helper pattern
