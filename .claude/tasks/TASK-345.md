# TASK-345: VS Code status bar — activity summary

**Plan:** PLAN-050 · Wave 5
**Status:** DONE
**Agent:** Senior Developer

## Description

Add a VS Code status bar item showing compact activity summary:

- Format: "🤖 N agents active · M generating" (or "nexus: offline" when daemon down)
- Click action: open the Nexus Orchestrator activity panel/tree view
- Update every 10s by polling GET /api/activities (count unique agent names in last 5 min)

## Acceptance

- Status bar item shows in VS Code
- Updates periodically
- Click opens activity view
- Shows "offline" when daemon unreachable

## Completed

Added VS Code status bar item showing active-agent count summary with click-to-open panel and daemon-offline state.
