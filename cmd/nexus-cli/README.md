# nexus-cli

`nexus-cli` (aliased as `nexus`) is the unified command-line interface for `nexus-orchestrator`.

It connects to a running `nexus-daemon` (defaults to `http://127.0.0.1:63987` or `$NEXUS_ADDR`), while also integrating the commands of all modular sub-CLIs (`brain`, `verify`, `execute`, `inspect`).

---

## Command Reference

### Task Management

- `nexus list [--status <status>]`: List current queue and task statuses.
- `nexus submit --title "<title>" --prompt "<prompt>"`: Submit a new AI task.
- `nexus status <task-id>`: Inspect task details, streaming output, and verification results.
- `nexus cancel <task-id>`: Cancel a queued or running task.

### Brain & Knowledge Commands

- `nexus brain init --project <dir>`: Ingest project guidelines and documentation.
- `nexus brain search "<query>"`: Search indexed knowledge across projects.
- `nexus brain status`: View knowledge store document counts and status.

### Runner & Verification Gates

- `nexus verify --command "<cmd>"`: Run verification command and check policies.
- `nexus execute --command "<cmd>" --prompt "<prompt>"`: Run self-healing execution loop.
- `nexus inspect`: Audit repository line-count invariants and git cleanliness.

### Agent Connection

- `nexus connect`: Check which local agents can communicate with Nexus.
- `nexus connect --write`: Automatically inject the Nexus MCP server into agent configurations (Antigravity, Cursor, Claude Code).
