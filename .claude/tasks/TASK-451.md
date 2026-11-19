---
id: TASK-451
plan: PLAN-059
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 5
priority: 5
---

# TASK-451: Fix silent JSON encoding errors in HTTP handlers

**Problem:** 44+ instances of `_ = json.NewEncoder(w).Encode(...)` across all HTTP handlers. If encoding fails, client receives incomplete data with no error indication.

**Fix:** Create a helper `writeJSON(w http.ResponseWriter, status int, v any)` that:

1. Sets Content-Type
2. Sets status code
3. Encodes v
4. On error: logs and writes 500 if headers not yet sent

Replace all `_ = json.NewEncoder(w).Encode(...)` calls with `writeJSON(w, statusCode, data)`.

**Files:** `internal/adapters/inbound/httpapi/*.go`
