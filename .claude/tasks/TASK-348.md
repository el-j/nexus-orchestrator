---
id: TASK-348
title: Continue reader — extract model field from session files + populate AIActivity.Model
role: backend
planId: PLAN-051
status: done
dependencies: [TASK-346]
createdAt: 2026-03-28T00:00:00Z
---

## Context

The `.continue` session files contain rich metadata including the model used in each chat. Currently the `ContinueSessionReader` only extracts session title, message count, and workspace directory. We need to extract the model name so that `AIActivity.Model` is populated and the nexusOrchestrator knows which qwen/llama/etc. model is active in the .continue chat.

## Files to Read

- `internal/adapters/outbound/activity_continue/reader.go` — current reader implementation
- `internal/core/domain/activity.go` — `AIActivity` struct (confirm `Model` field exists)

## Implementation Steps

1. In `reader.go`, extend the `sessionFile` struct to capture model metadata:

```go
type sessionFile struct {
    SessionID string           `json:"sessionId"`
    Title     string           `json:"title"`
    History   []historyMessage `json:"history"`
    // ModelID is the model used in this session (populated by Continue >= 0.9.x).
    ModelID   string           `json:"model,omitempty"`
}
```

2. The `historyMessage` type may also carry a per-message model. Check the actual JSON format — Continue stores model info in message content blocks or at the session level. Add:

```go
type historyMessage struct {
    Role      string      `json:"role"`
    Content   interface{} `json:"content"`
    Timestamp int64       `json:"timestamp"`
    // Model is optionally present on assistant messages in newer Continue versions.
    Model     string      `json:"model,omitempty"`
}
```

3. In `buildActivity`, extract the model ID:

```go
// Extract model: prefer session-level, fall back to last assistant message.
modelID := sf.ModelID
if modelID == "" {
    for i := len(sf.History) - 1; i >= 0; i-- {
        if sf.History[i].Role == "assistant" && sf.History[i].Model != "" {
            modelID = sf.History[i].Model
            break
        }
    }
}
```

4. Set it on the activity:

```go
a := &domain.AIActivity{
    // ... existing fields ...
    Model: modelID,  // <-- new
    Metadata: map[string]string{
        "messageCount": strconv.Itoa(messageCount),
        "source":       "continue-sessions",
        "model":        modelID,  // also in metadata for easy access
    },
}
```

5. Verify `domain.AIActivity` has a `Model string` field. If not (check `internal/core/domain/activity.go`), add it.

## Acceptance Criteria

- When a `.continue` session file contains `"model": "qwen3.5-35b-a3b"`, `AIActivity.Model` is populated with that value
- Existing tests in `activity_continue` package still pass
- `go vet ./internal/adapters/outbound/activity_continue/...` clean
