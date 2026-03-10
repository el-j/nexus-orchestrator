---
id: TASK-042
title: "Orchestrator: pre-flight token guard + estimateTokens() helper"
role: backend
planId: PLAN-004
status: todo
dependencies: [TASK-039]
createdAt: 2026-03-10T00:00:00.000Z
---

## Context
Once the port and adapters expose `ContextLimit()`, the orchestrator's `processNext()` must use it to abort early with a clear `StatusTooLarge` state before making any LLM call.  A response-headroom constant of 512 tokens is reserved for the assistant's reply.

## Files to Read
- `internal/core/services/orchestrator.go`
- `internal/core/domain/task.go` (for `StatusTooLarge`)

## Implementation Steps

### 1. Add the `estimateTokens` helper
Append to the bottom of `orchestrator.go` (after `extractCode`):
```go
// estimateTokens approximates the total token count of a message slice using
// the widely-accepted heuristic of 4 characters per token, plus 4 overhead
// tokens per message (role + formatting separators).
func estimateTokens(messages []domain.Message) int {
    total := 0
    for _, m := range messages {
        total += (len(m.Content)+3)/4 + 4
    }
    return total
}
```

### 2. Add the pre-flight check in `processNext()`
Insert AFTER the `llm := o.discovery.DetectActive()` nil-check block and BEFORE the `_ = o.repo.UpdateStatus(task.ID, domain.StatusProcessing)` line:

```go
// Pre-flight: guard against context-window overflow BEFORE spending LLM time.
const maxResponseTokens = 512
if limit := llm.ContextLimit(); limit > 0 {
    // Build a representative history for estimation (mirrors what Chat() will receive).
    histEst := append(existingSession.Messages, domain.Message{Role: "user", Content: prompt})
    if estimated := estimateTokens(histEst); estimated > limit-maxResponseTokens {
        logEntry := fmt.Sprintf(
            "context too large: ~%d tokens estimated, model limit is %d (headroom %d) — shorten the instruction or reduce context files",
            estimated, limit, maxResponseTokens,
        )
        log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
        _ = o.repo.UpdateLogs(task.ID, logEntry)
        _ = o.repo.UpdateStatus(task.ID, domain.StatusTooLarge)
        o.emit(task.ID, domain.StatusTooLarge)
        return
    }
}
```

**NOTE**: The estimation must happen AFTER the prompt is fully assembled (including context files) but BEFORE `UpdateStatus(Processing)`.  The session history needed for the estimate is loaded inside the `if o.sessionRepo != nil` block later, so a clean approach is:

a. Load the session history ONCE before the pre-flight block (reuse in Chat later):
```go
var sessionHistory []domain.Message
if o.sessionRepo != nil {
    sess, _ := o.sessionRepo.GetByProjectPath(task.ProjectPath)
    sessionHistory = sess.Messages
}
```
b. Pre-flight check using `sessionHistory` + the assembled `prompt`.
c. Inside the `if o.sessionRepo != nil` block, use the already-loaded `sessionHistory` instead of calling `GetByProjectPath` again.

Refactor `processNext()` accordingly to avoid the double `GetByProjectPath` call.

### 3. Ensure the new `StatusTooLarge` emit type resolves correctly
`"task." + strings.ToLower(string(domain.StatusTooLarge))` = `"task.too_large"` — this is fine; the event bus handles arbitrary types.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/...` exits 0
- [ ] A task whose estimated token count exceeds `contextLimit - 512` reaches `StatusTooLarge` (not `StatusFailed`)
- [ ] The `Logs` field on such a task contains the estimated token count, the model limit, and a "shorten the instruction" hint
- [ ] When `ContextLimit()` returns 0, no pre-flight check is performed and processing continues normally

## Anti-patterns to Avoid
- NEVER call `GetByProjectPath` twice — load history once, reuse
- NEVER block the worker goroutine on the limit fetch (sync.Once in adapters handles it)
- NEVER emit `StatusFailed` for a too-large task — use `StatusTooLarge` so UI can show a distinct message
