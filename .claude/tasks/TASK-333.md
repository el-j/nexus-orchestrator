# TASK-333: Continue sessions activity reader

**Plan:** PLAN-050 · Wave 2
**Status:** DONE
**Agent:** Senior Developer

## Description

Create `internal/adapters/outbound/activity_continue/reader.go` implementing `ports.ActivityReader`.

Scans `~/.continue/sessions/sessions.json` for session index, then reads individual `<sessionId>.json` files for chat history. Extract: session title, message count, workspace directory, last message timestamp. Generate one AIActivity per session with summary = session title, type = "message", metadata includes messageCount.

Only process sessions modified since `since` parameter.

## Acceptance

- Implements ActivityReader
- Reads Continue session index + individual files
- Extracts session titles and message counts
- Tests with sample data

## Completed

Implemented `activity_continue` reader parsing `~/.continue/sessions/` session index and per-session JSON chat files.
