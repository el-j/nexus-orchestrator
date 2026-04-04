# TASK-328: AIActivity domain type + ActivityReader port

**Plan:** PLAN-050 · Wave 1
**Status:** DONE
**Agent:** Senior Developer

## Description

Add new domain type `AIActivity` to `internal/core/domain/` representing a single observable action by an AI agent. Fields: ID, SessionID, AgentName, ActivityType (message/tool_use/thinking/file_edit/generation), Summary, ProjectPath, Model, TokensIn, TokensOut, Timestamp, Metadata map.

Add `ActivityReader` port interface to `internal/core/ports/ports.go`: `ReadActivities(ctx, since time.Time) ([]AIActivity, error)` + `SourceName() string`.

## Acceptance

- `domain.AIActivity` struct exists with all fields
- `ActivityType` constants defined
- `ports.ActivityReader` interface defined
- Compiles clean, `go vet` passes

## Completed

Added `AIActivity` domain type with activity type constants and `ActivityReader` port interface to `ports.go`.
