# TASK-361: Fix .continue MCP config file

**Plan:** PLAN-052 | **Wave:** 6 | **Status:** done

## Description

Fix `.continue/mcpServers/new-mcp-server.yaml` to work correctly with the
new SSE transport endpoint.

## Current (broken)

```yaml
- name: 'Nexus Orchestrator MCP Server'
  type: sse
  url: 'http://127.0.0.1:63988/sse' # /sse doesn't exist
```

## Fixed

```yaml
- name: 'Nexus Orchestrator MCP Server'
  type: sse
  url: 'http://127.0.0.1:63988/sse'
```

After TASK-357 adds the SSE endpoint, this URL will work.

Also add a streamable-http variant and a stdio variant as commented examples.

## Files to modify

- `.continue/mcpServers/new-mcp-server.yaml`
