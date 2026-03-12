---
id: TASK-145
title: Rebuild Wails GUI and dogfood smoke test
role: devops
planId: PLAN-020
status: todo
dependencies: [TASK-143, TASK-144]
createdAt: 2026-03-11T19:00:00.000Z
---

## Context

After TASK-143 (type fix) and TASK-144 (routing), the frontend needs to be compiled
into the Wails desktop app binary. Then validate that:
1. Provider cards display correctly (LM Studio active, Ollama unreachable)
2. The Providers page is reachable
3. A real task can be submitted end-to-end (dogfooding)

## Files to Read

- `Makefile` (to confirm `build-gui` target)

## Implementation Steps

1. **Run frontend TypeScript build** to confirm zero errors:
   ```bash
   cd frontend && npm run build 2>&1
   ```
   If any TypeScript errors, stop and report them — do NOT proceed to Wails build.

2. **Build the Wails desktop app**:
   ```bash
   cd /path/to/project && make build-gui
   ```
   Expected output: `build/bin/nexusOrchestrator.app` updated.

3. **Smoke-test the API** (app must be running — check if it is, then query):
   ```bash
   curl -s http://127.0.0.1:9999/api/providers | python3 -m json.tool
   ```
   Confirm LM Studio shows `"active": true` with at least 1 model.

4. **Submit a dogfood task** via the HTTP API to verify end-to-end processing:
   ```bash
   curl -s -X POST http://127.0.0.1:9999/api/tasks \
     -H "Content-Type: application/json" \
     -d '{
       "projectPath": "/Users/rex-fab-alt/Documents/code/playground/nexusOrchestrator",
       "targetFile": "DOGFOOD_TEST.md",
       "instruction": "Write a one-paragraph summary of what nexus-orchestrator does",
       "command": "execute"
     }'
   ```
   Poll until status is COMPLETED or FAILED:
   ```bash
   TASK_ID=$(curl -s http://127.0.0.1:9999/api/tasks | python3 -c "import sys,json; tasks=json.load(sys.stdin); print(tasks[0]['ID'] if tasks else '')")
   curl -s http://127.0.0.1:9999/api/tasks/$TASK_ID | python3 -m json.tool
   ```

5. Report:
   - Frontend build success/failure
   - Wails build success/failure
   - Provider API response
   - Task final status and any error logs

## Acceptance Criteria

- [ ] `make build-gui` exits 0
- [ ] API returns LM Studio as active with models listed
- [ ] Dogfood task reaches COMPLETED or a meaningful error is reported

## Note

If the app is not running (port 9999 not responsive), start it:
```bash
open build/bin/nexusOrchestrator.app
sleep 3
```
