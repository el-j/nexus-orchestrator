# TASK-338: Bridge discovered agents → sessions via activity

**Plan:** PLAN-050 · Wave 3
**Status:** DONE
**Agent:** Senior Developer

## Description

In ActivityService, when a new activity arrives from a reader:

1. Check if an AISession exists for this agent+project combination
2. If not, auto-create one with: source="discovered", agentName from activity, projectPath from activity, status="active"
3. If exists, update: CurrentActivity, LastActivity, MessageCount, TokensUsed, status="active"
4. If no activity for 5 minutes, mark session status="idle"
5. If no activity for 2 hours, mark session status="disconnected"

This bridges the gap between "discovered agents" (backend scanner) and "AI sessions" (frontend display).

## Acceptance

- New activities auto-create sessions
- Existing sessions updated with activity metadata
- Idle/disconnect timeouts work

## Completed

Implemented activity→session bridge in `ActivityService` with auto-create, 5-minute idle timeout, and 2-hour disconnect timeout.

- No duplicate sessions (key by agentName + projectPath)
