---
id: TASK-439
plan: PLAN-058
status: done
---

# TASK-439: Plans collapsible tree with overview stats

**Plan:** PLAN-058 (Wave 4)

## Problem

- DiscoveredPlansView renders projects as flat sections — cannot collapse individual projects
- No overview stats visible without scrolling through all plan files
- With 401 plans across 2 projects, too much information at once

## Solution

1. Add `collapsedProjects` reactive Set to track collapsed project paths
2. Make project header clickable to toggle collapse
3. When collapsed, show stats inline: plan count, task count, active plan indicator, last modified
4. Add "Collapse All" / "Expand All" toggle button in the header
5. Default: all projects collapsed except the one with active plans
6. Show chevron icon (▶/▼) before project name

## Files

- `frontend/src/views/DiscoveredPlansView.vue`

## Acceptance

- Clicking project header toggles collapse/expand
- Collapsed state shows plan count, task count, active indicator
- Collapse All / Expand All button works
- Projects with active plans are expanded by default
