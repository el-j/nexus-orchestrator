# PLAN-048: Middlepoint Sniffer — Universal Agent & Plan Discovery

**Goal:** Expand nexusOrchestrator into a true "middlepoint-sniffer": detect ALL plan/task files (orchestrator.json + markdown + cursor + MCP configs + crewai), enrich discovered agents with sub-agent trees, model IDs, and identities, and surface everything in the UI.

## Waves

| Wave | Tasks                        | Description                              |
| ---- | ---------------------------- | ---------------------------------------- |
| 1    | TASK-315                     | Domain types only (no deps)              |
| 2    | TASK-316, TASK-317, TASK-322 | Scanner + VS Code (parallel, dep on 315) |
| 3    | TASK-318                     | Backend wiring (dep on 316+317)          |
| 4    | TASK-319, TASK-320, TASK-321 | MCP + Frontend (parallel, dep on 318)    |

## Tasks

| ID       | Title                                                                             | Role     | Deps        | Status |
| -------- | --------------------------------------------------------------------------------- | -------- | ----------- | ------ |
| TASK-315 | Domain types: extend DiscoveredAgent + new DiscoveredPlanFile + AISession.ModelID | backend  | —           | todo   |
| TASK-316 | sys_scanner: ScanPlanFiles — detect 8+ plan file patterns                         | backend  | 315         | todo   |
| TASK-317 | sys_scanner: sub-agent enrichment from ~/.claude/projects JSONL                   | backend  | 315         | todo   |
| TASK-318 | Ports + OrchestratorService + SQLite + HTTP API for plan-file discovery           | backend  | 315,316,317 | todo   |
| TASK-319 | MCP tools: get_discovered_plans + enrich agent tools                              | mcp      | 318         | todo   |
| TASK-320 | Frontend: DiscoveredPlansView with kind/format badges                             | frontend | 318         | todo   |
| TASK-321 | Frontend: AIAgentsView sub-agent tree + model badge                               | frontend | 318         | todo   |
| TASK-322 | VS Code extension: modelId in registerSession + display                           | vscode   | 315         | todo   |
