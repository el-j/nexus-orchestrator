# TASK-331: Claude JSONL activity reader

**Plan:** PLAN-050 · Wave 2
**Status:** DONE
**Agent:** Senior Developer

## Description

Create `internal/adapters/outbound/activity_claude/reader.go` implementing `ports.ActivityReader`.

Scans `~/.claude/projects/**/*.jsonl` for session activity. For each JSONL file modified within the retention window:

1. Parse each line for `type` field: "user", "assistant", "tool_use"
2. Extract: timestamp, message role, model, cwd (project path), agentId, token usage (input_tokens, output_tokens)
3. Generate `AIActivity` per significant event (user message, assistant response, tool call)
4. Track file read position per-file (in-memory) for incremental reads — don't re-parse completed lines
5. Detect sidechain relationships via `parentUuid` + `isSidechain` fields
6. Summary generation: for tool_use → "Using <tool_name>", for assistant → "Responding (N tokens)", for user → "User prompt"

Privacy: Do NOT store full message content — only extract metadata, tool names, file paths from tool_use blocks.

## Acceptance

- Implements ActivityReader interface
- Parses real Claude JSONL files from ~/.claude/projects/
- Incremental read (doesn't re-parse old lines)
- Tests with sample JSONL data
- No full message content in AIActivity.Summary

## Completed

Implemented `activity_claude` reader parsing `~/.claude/projects/**/*.jsonl` with incremental file-position tracking and privacy-safe metadata-only summaries.
