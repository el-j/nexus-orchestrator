# TASK-270 — LiveActivityView.vue

**Plan**: PLAN-041  
**Status**: done

## What
Create `frontend/src/views/LiveActivityView.vue` — a unified "what's happening now on this machine" view.
Uses both `useDiscovery()` and `useAISessions()` composables.
Shows two sections:
1. "AI Tools Detected" — from sys_scanner (ports + processes + CLIs) with rich badges
2. "Active AI Sessions" — from Nexus AI session registry
Auto-refreshes via existing SSE/polling from both composables.
