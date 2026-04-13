---
id: TASK-517
title: Add brain CLI commands
role: cli
planId: PLAN-066
status: done
dependencies: [TASK-513]
createdAt: 2026-04-13T00:00:00Z
---

## Context

A `nexus brain` cobra subcommand tree provides operator access to the brain from the terminal. The CLI talks to the daemon via HTTP (same as other CLI commands) rather than calling BrainService directly. This keeps the CLI stateless. The `brain` subcommand is registered only when the HTTP base URL is available.

## Files to Read

- `internal/adapters/inbound/cli/root.go` — how NewRootCmd works, how existing subcommands (queue, providers, etc.) are added, HTTP client setup
- `internal/adapters/inbound/cli/` — read 2-3 existing command files to understand the pattern: HTTP call, JSON decode, table/JSON output
- `cmd/nexus-cli/main.go` — how root command is constructed

## Implementation Steps

1. Create `internal/adapters/inbound/cli/brain.go`, package `cli`

2. `newBrainCmd(baseURL string) *cobra.Command` — parent command:

   ```go
   &cobra.Command{
       Use:   "brain",
       Short: "Project context intelligence (brain) management",
   }
   ```

   Attach all subcommands via `cmd.AddCommand(...)`

3. Subcommand `brain init <project-path>`:
   - Flag: `--claude-md string` (path to CLAUDE.md, optional)
   - HTTP: `POST {baseURL}/api/brain/init` with body `{"projectPath": args[0], "claudeMdPath": flagValue}`
   - Print: `Brain initialized: {EntryCount} entries, {TotalTokens} total tokens`

4. Subcommand `brain context <project-path>`:
   - Flags: `--focus string` (default "all"), `--max-tokens int` (default 400)
   - HTTP: `POST {baseURL}/api/brain/context` with body `{"projectPath": args[0], "focus": focus, "maxTokens": maxTokens}`
   - Print: for each section in response, print `[kind/topic] content` separator lines
   - Print footer: `Tokens used: {TokensUsed}/{TokenBudget}{" (truncated)" if Truncated}`

5. Subcommand `brain search <project-path>`:
   - Flags: `--query string` (required), `--max-tokens int` (default 400)
   - HTTP: `POST {baseURL}/api/brain/search` with body `{"projectPath": args[0], "query": query, "maxTokens": maxTokens}`
   - Print: numbered list of sections with kind, topic, content preview (first 100 chars)

6. Subcommand `brain list <project-path>`:
   - Flag: `--kind string` (optional)
   - HTTP: `GET {baseURL}/api/brain/knowledge?project={projectPath}&kind={kind}`
   - Print: table with columns: ID (short 8 chars), Kind, Topic, Tokens, Source

7. Subcommand `brain status <project-path>`:
   - HTTP: `GET {baseURL}/api/brain/status?project={projectPath}`
   - Print: `Project: {projectPath}`, `Initialized: yes/no`, `Entries: {EntryCount}`, `Total tokens: {TotalTokens}`, kind breakdown table

8. Subcommand `brain ingest <project-path>`:
   - Flags: `--file string` (path, if provided calls IngestFromFile), `--kind string`, `--topic string`, `--content string`
   - If `--file` given: HTTP `POST /api/brain/init` with the file path as `claudeMdPath` (reuse init endpoint for file ingest, or add a separate ingest endpoint)
   - If `--kind/--topic/--content` given: HTTP `POST /api/brain/knowledge` with full body
   - Print: confirmation message

9. Register `newBrainCmd(baseURL)` in `root.go` (or wherever other commands are added):
   - Only register if baseURL is available (follow how other commands are conditionally registered)

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `nexus brain --help` shows all 6 subcommands
- [ ] `brain init`, `brain status`, `brain context`, `brain search`, `brain list`, `brain ingest` all compile and have correct flags
- [ ] All HTTP calls use the existing HTTP client pattern (not raw net/http directly)

## Anti-patterns to Avoid

- NEVER call BrainService directly from CLI — always go via HTTP to the daemon
- NEVER hardcode localhost URL — use the baseURL from root command config
- NEVER use fmt.Println for errors — use cobra's error return pattern
