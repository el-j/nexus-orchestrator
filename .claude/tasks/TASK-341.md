# TASK-341: Agent detail panel

**Plan:** PLAN-050 · Wave 4
**Status:** DONE
**Agent:** UI Designer

## Description

Create agent detail view accessible from sidebar or by clicking an agent card. Shows:

- Agent name, model, status (active/idle/disconnected) with colored badge
- Current activity summary (live-updating)
- Active project path
- Session stats: uptime, message count, total tokens used
- Last 10 activities as a mini-timeline
- Subagent tree: if this agent has spawned subagents (parentUuid relationship), show them nested

Can be a drawer/panel overlay or a dedicated route (`/agents/:id`).

## Acceptance

- Shows agent details with all fields
- Mini-timeline updates in real-time
- Subagent relationships displayed if present

## Completed

Created agent detail panel (`/agents/:id` route) with live status, session stats, mini-timeline, and subagent parent→child tree view.
