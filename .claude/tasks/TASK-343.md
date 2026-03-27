# TASK-343: Dashboard provider cards with generating state

**Plan:** PLAN-050 · Wave 4
**Status:** DONE
**Agent:** UI Designer

## Description

Update the provider cards on the Task Queue dashboard (top bar showing LM Studio, Ollama, Antigravity) to display:

- When a model is actively loaded: green badge + model name
- When generating: pulsing animation + "Generating..." text
- Active model count: "2 models loaded"

Data source: network probe activity reader output via useDiscovery composable (extend to include model details from activities).

## Acceptance

- Provider cards show model state when reachable
- Visual distinction between idle and active providers
- No false positives when provider unreachable

## Completed

Updated dashboard provider cards with generating-state pulse animation, active model name badge, and model-loaded indicator from network probe activity data.
