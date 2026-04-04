---
id: TASK-438
plan: PLAN-058
status: done
---

# TASK-438: AI Coding Tools grouping by kind

**Plan:** PLAN-058 (Wave 4)

## Problem

- AI Coding Tools in ProvidersView renders flat grid — all Claude CLI cards together with Copilot, Continue, etc
- No grouping by provider kind, hard to see what each offers

## Solution

1. Create `groupedAgents` computed that groups agents by `kind` field
2. Render each group with a header showing: kind label, count, and what the tool offers (description)
3. Within each group, show compact cards with: name, status badge, configPath/workingDir, detection method
4. Use denser card layout: `p-3` padding, `gap-2` spacing, smaller text
5. Add kind descriptions: copilot="AI pair programmer", claude-cli="Terminal AI assistant", continue="Open-source AI code assistant", cline="VS Code AI agent"

## Files

- `frontend/src/views/ProvidersView.vue`

## Acceptance

- Tools grouped under kind headers (e.g., "Claude CLI (3)", "GitHub Copilot (1)")
- Each group header has a brief description of what the tool does
- Cards are more compact than current layout
