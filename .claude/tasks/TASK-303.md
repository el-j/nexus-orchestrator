---
id: TASK-303
title: Fix httpapi — return 400 on bad JSON in handleTerminateAISession + add writeJSON success helper
role: backend
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

Two issues in `internal/adapters/inbound/httpapi/server.go`:

**Issue 1 — Silent JSON error in handleTerminateAISession**
Line ~744: `_ = json.NewDecoder(r.Body).Decode(&body)` silently ignores decode errors. A malformed JSON body is treated as `{force: false}` with no 400 response to the caller.

**Issue 2 — Missing writeJSON success helper**
`writeJSONError` is extracted as a helper but the 27+ success-path handlers each manually call `w.Header().Set("Content-Type", "application/json")` followed by `json.NewEncoder(w).Encode(...)`. A missing Content-Type header on any new handler would go unnoticed.

## Files to Read

- `internal/adapters/inbound/httpapi/server.go` — full file, especially `writeJSONError`, `handleTerminateAISession`, and 3–4 representative success handlers

## Implementation Steps

### Fix 1 — handleTerminateAISession JSON error

Replace:

```go
_ = json.NewDecoder(r.Body).Decode(&body)
```

With:

```go
if r.ContentLength > 0 {
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }
}
```

Only reject if a body is actually present (Content-Length > 0 or r.Body != http.NoBody) so clients that send no body at all still get `force=false` default.

### Fix 2 — writeJSON helper

Add after `writeJSONError`:

```go
func writeJSON(w http.ResponseWriter, code int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    _ = json.NewEncoder(w).Encode(v)
}
```

Then replace every occurrence of the two-line pattern:

```go
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(someValue)
```

With:

```go
writeJSON(w, http.StatusOK, someValue)
```

And for handlers that call `w.WriteHeader(...)` explicitly before encoding, use `writeJSON(w, code, value)`.

Do NOT change any handler's HTTP status codes — only refactor the boilerplate.

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/inbound/httpapi/...` clean
- [ ] `go test ./internal/adapters/inbound/httpapi/... -race -count=1` all pass
- [ ] `handleTerminateAISession` returns 400 when body is present but malformed JSON
- [ ] `writeJSON(w, code, v)` helper exists and is used throughout
- [ ] No bare `w.Header().Set("Content-Type", "application/json")` + `json.NewEncoder` pairs remain
