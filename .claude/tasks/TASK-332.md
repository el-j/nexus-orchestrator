# TASK-332: Claude global history reader

**Plan:** PLAN-050 · Wave 2
**Status:** DONE
**Agent:** Senior Developer

## Description

Create `internal/adapters/outbound/activity_claude/history_reader.go` implementing `ports.ActivityReader`.

Parses `~/.claude/history.jsonl` — the global CLI command log. Each line has: `display` (command text), `timestamp` (unix ms), `project` (absolute path), `sessionId`. Generate AIActivity with type "message", summary from display text (truncated), project path.

Track file position for incremental reads.

## Acceptance

- Implements ActivityReader
- Parses ~/.claude/history.jsonl
- Incremental read
- Links activities to projects

## Completed

Implemented Claude global history reader parsing `~/.claude/history.jsonl` for cross-project CLI command activity with incremental reads.
