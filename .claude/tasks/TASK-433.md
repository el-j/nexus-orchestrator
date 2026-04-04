---
id: TASK-433
plan: PLAN-058
status: done
---

# TASK-433: LogPanel flex layout fix

**Plan:** PLAN-058 (Wave 1)

## Problem

- `LogPanel.vue` uses `position: fixed; bottom: 0` — it overlays content instead of participating in flex flow
- `App.vue` has `<main style="padding-bottom: 208px">` hardcoded — when log panel is collapsed, the 208px gap remains wasted

## Solution

1. **LogPanel.vue**: Remove `position: fixed; bottom: 0; left: 0; right: 0; z-index: 100` from `.log-panel` CSS
2. **LogPanel.vue**: Make `.log-panel` a normal flex child: `flex-shrink: 0` with dynamic height
3. **LogPanel.vue**: When collapsed, panel shows only the header bar (~28px). When expanded, uses `panelHeight` ref
4. **App.vue**: Remove `style="padding-bottom: 208px"` from `<main>`. The flex layout will naturally allocate space

## Files

- `frontend/src/App.vue` — remove hardcoded padding-bottom
- `frontend/src/components/LogPanel.vue` — convert from fixed to flex-based

## Acceptance

- Collapsing log panel reclaims all space for main content
- Expanding log panel pushes main content up
- Drag resize still works
- No layout jump on toggle
