---
id: TASK-032
title: Add JSON struct tags to domain.Task, domain.Message, domain.Session
role: backend
planId: PLAN-003
status: todo
dependencies: []
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

`domain.Task`, `domain.Message`, and `domain.Session` have **no `json:` struct tags**. JSON encoding therefore uses Go FieldName casing (PascalCase). This means:
- The HTTP API returns `{"ID":"...","ProjectPath":"..."}` — non-standard for REST APIs
- The web dashboard (TASK-034) cannot use standard camelCase field access in JS
- `nexus-submit` (TASK-033) must hardcode PascalCase keys in POST bodies

This task adds proper `json:"camelCase"` tags to all domain types. No behavioural changes — it's purely a serialisation fix.

## Files to Modify

- `internal/core/domain/task.go`
- `internal/core/domain/session.go`

## Current domain.Task (no tags):

```go
type Task struct {
    ID           string
    ProjectPath  string
    TargetFile   string
    Instruction  string
    ContextFiles []string
    Status       TaskStatus
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Logs         string
}
```

## After TASK-032 — domain.Task:

```go
type Task struct {
    ID           string     `json:"id"`
    ProjectPath  string     `json:"projectPath"`
    TargetFile   string     `json:"targetFile"`
    Instruction  string     `json:"instruction"`
    ContextFiles []string   `json:"contextFiles"`
    Status       TaskStatus `json:"status"`
    CreatedAt    time.Time  `json:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt"`
    Logs         string     `json:"logs,omitempty"`
}
```

## After TASK-032 — domain.Message:

```go
type Message struct {
    Role      string    `json:"role"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"createdAt"`
}
```

## After TASK-032 — domain.Session:

```go
type Session struct {
    ID          string    `json:"id"`
    ProjectPath string    `json:"projectPath"`
    Messages    []Message `json:"messages"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}
```

## Risk Analysis

| Risk | Impact | Mitigation |
|------|--------|-----------|
| HTTP API request body format changes (camelCase) | HIGH | All callers updated |
| CLI `nexus queue list` prints camelCase | LOW | Cosmetic only |
| SQLite repo not affected (uses column scan) | NONE | Repo uses named column binding |
| MCP server uses `json.Unmarshal` into typed structs | NONE | MCP uses its own input parsing |

## Callers to Update

The HTTP API `POST /api/tasks` handler decodes directly into `domain.Task`:
```go
var req domain.Task
json.NewDecoder(r.Body).Decode(&req)
```
After adding json tags, POST bodies must use camelCase keys. All callers:
1. MCP `submit_task` handler — builds `domain.Task` directly from parsed fields, NOT from JSON decode → **no change needed**
2. `app.go` Wails SubmitTask — JS calls binding directly → **no JSON decode**, Wails handles marshalling → **no change needed**
3. `httpapi` POST handler — decodes `domain.Task` from body → callers must use camelCase (they should already since this is fixing the API)
4. `orchestrator_test.go` — uses struct literals, not JSON → **no change needed**

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `domain.Task` has `json:"camelCase"` tags on all fields
- [ ] `domain.Message` has `json:"role"`, `json:"content"`, `json:"createdAt"` tags
- [ ] `domain.Session` has `json:"id"`, `json:"projectPath"`, etc. tags
- [ ] `Logs` field has `json:"logs,omitempty"` (omit empty string from JSON responses)
- [ ] `GET /api/tasks` returns `[{"id":"...","projectPath":"...",...}]` (camelCase)
- [ ] No changes to repo_sqlite — it uses column scanning, not JSON

## Anti-patterns to Avoid

- NEVER add `json:"-"` tags — all fields must be visible in API output
- NEVER change field names or types — only add json tags
- NEVER update SQL queries — this is a serialisation-only change
- NEVER add `omitempty` to required fields like `id`, `projectPath`, `status`
