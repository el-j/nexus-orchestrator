---
id: TASK-038
title: PLAN-003 verification + README update + task completion report
role: planning
planId: PLAN-003
status: todo
dependencies: [TASK-034, TASK-036]
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

Final verification task for PLAN-003. Confirms all deliverables work together and documents the dogfood workflow in the README so future contributors can use nexusOrchestrator for its own development.

## Files to Modify

- `README.md` — add Dogfood section
- `.claude/orchestrator.json` — mark PLAN-003 tasks as done after verification

## Verification Checklist

Run all of these before marking PLAN-003 complete:

```bash
# 1. Full build
CGO_ENABLED=1 go build ./...

# 2. Tests with race detector
CGO_ENABLED=1 go test -race -count=1 -timeout=60s ./...

# 3. Vet
go vet ./...

# 4. Start daemon + check dashboard
NEXUS_DB_PATH=/tmp/verify.db nexus-daemon &
sleep 1

curl -sf http://localhost:63987/api/health | jq .
curl -sf http://localhost:63987/ui | grep -q "nexusOrchestrator"

# 5. Test nexus-submit
nexus-submit --task-file .claude/tasks/TASK-013.md \
             --project "$PWD" \
             --target internal/core/services/orchestrator.go \
             --context internal/core/services/orchestrator.go | grep "task_id"

# 6. Check SSE
curl -N --max-time 3 http://localhost:63987/api/events | head -1

# 7. Run dogfood script (requires LLM provider online)
# ./scripts/dogfood-plan002.sh

kill $(lsof -ti:63987) 2>/dev/null
```

## README Section to Add

Add under a `## Dogfooding` heading after the Usage section:

```markdown
## Dogfood — use nexusOrchestrator for its own development

nexusOrchestrator can implement its own backlog tasks. This is the primary validation of the pipeline.

### Prerequisites

- LM Studio running at `http://127.0.0.1:1234/v1` OR Ollama at `http://127.0.0.1:11434`
- `CGO_ENABLED=1 go build ./...` passes

### Start the daemon and dashboard

```bash
# Build and run the daemon
CGO_ENABLED=1 go build -o /tmp/nexus-daemon ./cmd/nexus-daemon
NEXUS_DB_PATH=/tmp/nexus-local.db /tmp/nexus-daemon &

# Open the live dashboard
open http://localhost:63987/ui
```

### Submit a PLAN-002 task for LLM implementation

```bash
# Build nexus-submit
CGO_ENABLED=1 go build -o /tmp/nexus-submit ./cmd/nexus-submit

# Submit TASK-013 (orchestrator hardening)
/tmp/nexus-submit \
  --task-file .claude/tasks/TASK-013.md \
  --project "$PWD" \
  --target internal/core/services/orchestrator.go \
  --context internal/core/services/orchestrator.go,internal/core/ports/ports.go \
  --wait
```

### Run all PLAN-002 tasks at once

```bash
./scripts/dogfood-plan002.sh
```

### Track via MCP (for AI editors)

Use any MCP-compatible agent:
```json
{"method": "tools/call", "params": {"name": "get_queue"}}
```
```

## Acceptance Criteria

- [ ] All items in the verification checklist pass
- [ ] `README.md` has a `## Dogfooding` section with copy-pasteable commands
- [ ] `scripts/dogfood-plan002.sh` is committed and executable
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` passes
- [ ] `cmd/nexus-submit` binary builds from clean source

## Anti-patterns to Avoid

- NEVER mark PLAN-003 complete before running the full verification checklist
- NEVER commit DB files or binary outputs
