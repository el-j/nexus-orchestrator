# TASK-340: AIActivityCard component

**Plan:** PLAN-050 · Wave 4
**Status:** DONE
**Agent:** UI Designer

## Description

Create `frontend/src/components/AIActivityCard.vue` — a compact card for displaying a single AI activity in the timeline.

Visual treatment by activity type:

- message: 💬 chat bubble style, blue accent
- tool_use: 🔧 terminal/code style, green accent
- thinking: 🧠 subtle pulse animation, purple accent
- file_edit: 📄 file icon, orange accent
- generation: ⚡ lightning, yellow accent

Card shows: agent name (bold) + model (light), activity summary, project path (abbreviated), relative timestamp ("2s ago"), token count badge if > 0.

Hover: show full metadata in tooltip.

## Acceptance

- Renders all 5 activity types with distinct visual treatment
- Responsive layout
- Tooltips work
- Follows existing component patterns

## Completed

Created `AIActivityCard.vue` with per-type icons/accents (message/tool_use/thinking/file_edit/generation), token badge, and metadata tooltip.
