# PLAN-047: Comprehensive Codebase Hardening

**Status:** active
**Created:** 2026-03-25
**Goal:** Fix all bugs, stubs, missing implementations, code duplication, MCP gaps, and test coverage issues found in the 2026-03-25 5-agent audit swarm.

---

## Audit Summary

5 specialized agents ran in parallel on 2026-03-25 and produced the following findings:

| Category                          | Count | Severity             |
| --------------------------------- | ----- | -------------------- |
| Critical bugs                     | 3     | HIGH                 |
| Silent stubs / missing HTTP calls | 3     | HIGH                 |
| Error handling silenced           | 4     | MEDIUM               |
| MCP tools missing                 | 9     | MEDIUM               |
| Code duplication sites            | 5     | MEDIUM               |
| Untested routes/methods           | 14+   | MEDIUM               |
| Dead code                         | 2     | LOW                  |
| God object (orchestrator.go)      | 1     | LOW (stability risk) |

---

## Wave Map

### Wave 1 — Bugs & Stubs (fully parallel, 5 independent files)

| Task     | File(s)               | Description                                                                 |
| -------- | --------------------- | --------------------------------------------------------------------------- |
| TASK-301 | orchestrator.go       | double processNext bug, PromoteProvider error swallow, hardcoded port 63987 |
| TASK-302 | cmd/nexus-cli/main.go | wire GetDiscoveredAgents + DelegateToNexus + force flag                     |
| TASK-303 | httpapi/server.go     | 400 on bad JSON body, writeJSON helper                                      |
| TASK-304 | repo_sqlite/\*.go     | tags JSON, RowsAffected, LastSeen silenced errors                           |
| TASK-305 | mcp/server.go         | 9 missing tools                                                             |

### Wave 2 — Deduplication (parallel, after wave 1)

| Task     | File(s)                            | Description                                                  |
| -------- | ---------------------------------- | ------------------------------------------------------------ |
| TASK-306 | llm\_\*/adapter.go                 | DefaultBaseURL, merge Once guards, message-conversion helper |
| TASK-307 | main.go + cmd/nexus-daemon/main.go | extract buildProviders to internal/bootstrap                 |

### Wave 3 — Test Coverage (parallel, after wave 1)

| Task     | File(s)                 | Description                                |
| -------- | ----------------------- | ------------------------------------------ |
| TASK-308 | httpapi/server_test.go  | 8 untested routes                          |
| TASK-309 | repo_sqlite/\*\_test.go | 6 untested methods + discovered_agent_repo |
| TASK-310 | mcp/server_test.go      | 9 new tool tests                           |

### Wave 4 — Major Refactoring (after waves 1–3, execute with care)

| Task     | File(s)                     | Description                    |
| -------- | --------------------------- | ------------------------------ |
| TASK-311 | orchestrator.go → 4 files   | split god object               |
| TASK-312 | cmd/nexus-cli/main.go       | extract httpapi_client adapter |
| TASK-313 | mcp/server.go → 2 files     | split protocol from tools      |
| TASK-314 | httpapi/server.go → 4 files | split by handler group         |

---

## Acceptance Criteria

- `go vet ./...` exits 0
- `go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- `go test ./... -race -count=1` exits 0
- All 14 tasks marked done in orchestrator.json
