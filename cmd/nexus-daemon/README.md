# nexus-daemon

`nexus-daemon` is the headless core orchestrator service of Nexus.

It manages the persistent task queue, runs background AI tasks, monitors local and cloud providers, serves the REST API (`:63987`), serves the Model Context Protocol (MCP) server (`:63988`), and embeds the Vue web UI.

---

## Running the Daemon

```sh
# Run with default settings (database in ~/.nexus/nexus.db)
nexus-daemon

# Override database or listening ports
NEXUS_DB_PATH=/path/to/custom.db NEXUS_LISTEN_ADDR=0.0.0.0:63987 nexus-daemon
```

---

## Embedded Services

- **HTTP REST API**: `http://localhost:63987/api/`
- **Web UI Dashboard**: `http://localhost:63987/app/` (full interface) and `http://localhost:63987/ui` (light dashboard)
- **MCP Server**: `http://localhost:63988/mcp` (Streamable JSON-RPC 2.0 with 44 tools)
- **Server-Sent Events (SSE)**: `http://localhost:63987/api/events` (Live task streaming and state updates)
