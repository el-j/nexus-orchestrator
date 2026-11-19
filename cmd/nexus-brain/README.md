# nexus-brain

`nexus-brain` is a standalone CLI and Model Context Protocol (MCP) server for the Nexus Brain RAG engine.

It provides semantic documentation indexing, natural-language full-text search (SQLite FTS5 BM25), context synthesis for LLM prompts, and workspace structural mapping.

---

## Operating Modes

### 1. Standalone Local Mode (Default)

Operates directly against a local SQLite database (default `~/.nexus/nexus.db` or custom `--db <path>`). No background daemon required:

```sh
# Search knowledge in local database
nexus-brain search "database migrations" --limit 5

# Initialize project documentation into local database
nexus-brain init --project /path/to/repo
```

### 2. Remote Mode

Delegates queries to an existing `nexus-daemon` instance via `--addr` or `NEXUS_ADDR`:

```sh
nexus-brain --addr http://localhost:63987 search "rate limit recovery"
```

---

## Dedicated MCP Server for IDEs

Run `nexus-brain` as a dedicated, lightweight Model Context Protocol (MCP) server for Claude Code, Cursor, Antigravity, or VS Code:

```sh
nexus-brain serve-mcp
```

### MCP Configuration Example (`.mcp.json`)

```json
{
  "mcpServers": {
    "nexus-brain": {
      "command": "nexus-brain",
      "args": ["serve-mcp"]
    }
  }
}
```

---

## Command Reference

| Command     | Usage                                              | Description                                                           |
| ----------- | -------------------------------------------------- | --------------------------------------------------------------------- |
| `init`      | `nexus-brain init --project <dir>`                 | Ingests project guidelines (`CLAUDE.md`, `README.md`, `docs/**/*.md`) |
| `ingest`    | `nexus-brain ingest --file <path> --project <dir>` | Ingests an individual markdown file into the knowledge repository     |
| `search`    | `nexus-brain search "<query>" [--limit N]`         | Performs BM25 natural-language search with relevance scoring          |
| `context`   | `nexus-brain context --query "<prompt>"`           | Assembles focused context snippets within a token budget              |
| `map`       | `nexus-brain map [--dir <dir>] [--max-depth N]`    | Generates a structural file map of the repository                     |
| `status`    | `nexus-brain status`                               | Displays knowledge database stats and indexed document counts         |
| `serve-mcp` | `nexus-brain serve-mcp`                            | Runs the dedicated JSON-RPC stdio MCP server                          |
