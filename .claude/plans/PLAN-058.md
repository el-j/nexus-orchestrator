# PLAN-058: UX Polish — Sorting, Filtering, Grouping & Layout Fixes

**Status:** completed  
**Created:** 2026-03-29  
**Goal:** Address remaining UX issues from post-PLAN-057 review: sort active items first, add filtering, group AI coding tools by kind, make Plans view collapsible, fix LogPanel collapse space reclamation, and tighten card density.

## Waves

### Wave 1 — Critical Layout Fix (TASK-433)

- LogPanel `position: fixed` → flex-based layout; remove hardcoded `padding-bottom: 208px`

### Wave 2 — Sorting (TASK-434, TASK-435)

- AgentsView: sort sessions active-first, discovered agents running-first, add sort toggle
- ProvidersView: sort active providers first, sort AI tools active-first

### Wave 3 — Filtering (TASK-436, TASK-437)

- AgentsView: search bar + status filter for sessions and discovered agents
- ProvidersView: search + status filter for providers and AI coding tools

### Wave 4 — Grouping & Tree (TASK-438, TASK-439)

- ProvidersView AI Coding Tools: group by kind, show capabilities, denser cards
- DiscoveredPlansView: collapsible projects with overview stats

### Wave 5 — Density (TASK-440)

- Tighten padding/gaps across AgentsView, ProvidersView, DiscoveredPlansView cards

## Tasks

- TASK-433: LogPanel flex layout fix
- TASK-434: AgentsView sorting (active first + toggle)
- TASK-435: ProvidersView sorting (active first + toggle)
- TASK-436: AgentsView filtering (search + status)
- TASK-437: ProvidersView filtering (search + status)
- TASK-438: AI Coding Tools grouping by kind
- TASK-439: Plans collapsible tree with stats
- TASK-440: Dense card layout across views
