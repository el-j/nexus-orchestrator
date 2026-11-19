# Standalone CLIs & Microservices

nexus-orchestrator provides both a unified control plane and modular, standalone CLI binaries. Each core subsystem can run completely standalone without background daemon dependencies, or connect seamlessly to a shared `nexus-daemon`.

---

## Overview of Standalone Binaries

| Binary                   | Subsystem                | Modes                            | Typical Use Case                                                                             |
| ------------------------ | ------------------------ | -------------------------------- | -------------------------------------------------------------------------------------------- |
| `nexus-brain`            | Knowledge & RAG Engine   | Local SQLite FTS5 or Remote API  | Dedicated MCP server, repository indexing, semantic doc retrieval                            |
| `nexus-router`           | LLM Gateway & Proxy      | Standalone Gateway or Remote API | OpenAI-compatible proxy (`/v1/chat/completions`), prompt execution, multi-provider fallbacks |
| `nexus-runner`           | Execution & Policy Gates | Standalone Policy Engine         | CI/CD verification gates, self-healing code generation, invariant audits                     |
| `nexus-cli` (or `nexus`) | Unified Control Plane    | Remote Daemon Client             | All-in-one management of tasks, brain, runner, and sessions                                  |

---

## 1. nexus-brain: Knowledge & RAG CLI

`nexus-brain` provides standalone semantic indexing, full-text search (SQLite FTS5), and context synthesis for codebases and project documentation.

### Operating Modes

1. **Standalone Local Mode** (default): Operates directly against an embedded SQLite database (defaults to `~/.nexus/nexus.db` or custom `--db <path>`). No daemon required.
2. **Remote Mode**: Connects to an existing daemon when `--addr <url>` or `NEXUS_ADDR` is provided.

### Key Commands

```sh
# Initialize project knowledge from documentation and guidelines
nexus-brain init --project /path/to/repo

# Ingest an individual markdown file into the knowledge store
nexus-brain ingest --file docs/architecture.md --project /path/to/repo

# Query relevant knowledge snippets with relevance scoring
nexus-brain search "database schema migrations" --limit 5

# Synthesize focused context for an AI prompt
nexus-brain context --query "how to add a new provider"

# Generate a structural file map of the repository
nexus-brain map --dir . --max-depth 3

# Check knowledge database status and indexed document counts
nexus-brain status
```

### Dedicated MCP Server for IDEs

Run `nexus-brain` as a dedicated, lightweight Model Context Protocol (MCP) server for agents such as Claude Code, Cursor, or Antigravity without running the full task orchestrator daemon:

```sh
nexus-brain serve-mcp
```

Configure your editor's MCP configuration (`.mcp.json` or `claude_desktop_config.json`):

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

## 2. nexus-router: LLM Gateway & OpenAI-Compatible Proxy

`nexus-router` provides a standalone model gateway with automatic multi-provider discovery, fallback ladder execution, provider rate-limit cooldowns, and real-time streaming.

### Standalone Proxy Server

Start an OpenAI-compatible HTTP proxy on a dedicated port:

```sh
nexus-router serve --port 8080 --default-role coder
```

Once running, any OpenAI-compatible client (curl, Python `openai` SDK, LiteLLM, LangChain, or custom agents) can use Nexus as its backend:

```sh
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Explain Go concurrency patterns"}
    ]
  }'
```

### Direct CLI Prompt Execution

Test prompt generation and verify model routing from the terminal:

```sh
# Run a quick prompt using the default role ladder
nexus-router prompt "Write a Go function to compute Fibonacci numbers"

# Target a specific agent role profile
nexus-router prompt "Refactor this function for maximum throughput" --role architect
```

### Provider Inspection

List all discovered local and cloud providers with their status and supported model tiers:

```sh
nexus-router providers
```

---

## 3. nexus-runner: Verification Gates & Self-Healing Execution

`nexus-runner` enforces architectural policies, runs verification checks, and executes automated self-healing loops for code generation in local development and CI/CD pipelines.

### Verification Gate (`verify`)

Execute verification commands (such as tests, linting, or coverage gates) against repository policies:

```sh
# Run tests with verification checks
nexus-runner verify --command "go test -v ./..."

# Apply a custom policy file
nexus-runner verify --policy .nexus/policy.json --command "make test"
```

### Self-Healing Execution (`execute`)

Run a command in a loop. When a failure occurs, the runner captures compiler/test errors, constructs a targeted correction prompt, delegates repair to an available LLM provider, applies extracted patch code, and re-runs verification:

```sh
nexus-runner execute \
  --command "go test -v ./internal/core/..." \
  --max-attempts 3 \
  --prompt "Fix failing unit tests in core services"
```

### Invariant Policy Audit (`inspect`)

Audit the repository against clean architecture constraints, line-count limits (500 lines source / 800 lines tests), test-first adherence, and uncommitted git modifications:

```sh
# Inspect current working tree
nexus-runner inspect

# Inspect a specific target directory with custom policy
nexus-runner inspect --dir ./pkg --policy .nexus/policy.json
```

---

## 4. Unified Access via `nexus-cli`

For convenience, the root `nexus-cli` binary (aliased as `nexus`) embeds all subcommands from the modular micro-CLIs:

```sh
# Brain operations
nexus brain init --project .
nexus brain search "verification"

# Runner operations
nexus verify --command "npm test"
nexus execute --command "make test" --max-attempts 3
nexus inspect

# Task orchestration & monitoring
nexus list
nexus submit --title "Implement feature X"
nexus status
```

This architecture ensures complete flexibility: use standalone binaries in focused microservices or CI pipelines, and use the unified CLI for interactive day-to-day development.
