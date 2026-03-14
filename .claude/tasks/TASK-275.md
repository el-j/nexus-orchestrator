# TASK-275 — Start daemon + end-to-end validation

**Plan**: PLAN-043  
**Status**: done  
**Role**: qa

## What

Build and start the nexus-daemon, verify the HTTP API + MCP are reachable, then confirm
the VS Code extension (new VSIX) connects and shows a live task after submission.

## Steps

```sh
# Build daemon binary
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/

# Start daemon (background, logs to /tmp/nexus-daemon.log)
./nexus-daemon &> /tmp/nexus-daemon.log &

# Verify endpoints
curl -s http://127.0.0.1:63987/api/health
curl -s http://127.0.0.1:63987/.well-known/nexus.json | python3 -m json.tool | head -20
curl -s http://127.0.0.1:63987/api/howto | python3 -m json.tool | head -10

# Submit a test task
curl -s -X POST http://127.0.0.1:63987/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"instruction":"hello world test","projectPath":"/tmp/test-proj"}'

# Verify task in queue
curl -s http://127.0.0.1:63987/api/tasks | python3 -m json.tool
```

## Acceptance
- `health` returns `{"status":"ok"}`
- `/.well-known/nexus.json` returns schema_version + name
- Task submission returns a task_id
- Task Queue in VS Code extension shows the submitted task
