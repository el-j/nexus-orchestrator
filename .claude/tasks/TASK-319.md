# TASK-319: MCP tools — get_discovered_plans + enrich agent tools

**Plan:** PLAN-048
**Role:** mcp
**Dependencies:** TASK-318

## Goal

Add MCP tool `get_discovered_plans` so AI agents can query what plan files exist in a project. Also update `get_discovered_agents` to return the enriched `modelId`, `subAgentIds`, `parentAgentId`, and `workingDir` fields.

## Changes

### `internal/adapters/inbound/mcp/tools.go`

**Add `case "get_discovered_plans":` in `handleToolCall`:**

```go
case "get_discovered_plans":
    return s.toolGetDiscoveredPlans(params)
```

**Add handler:**

```go
func (s *Server) toolGetDiscoveredPlans(params callToolParams) (callToolResult, error) {
    projectPath, _ := params.Args["projectPath"].(string)
    files, err := s.orch.GetDiscoveredPlanFiles(context.Background(), projectPath)
    if err != nil {
        return callToolResult{}, fmt.Errorf("mcp: get_discovered_plans: %w", err)
    }
    data, _ := json.Marshal(files)
    return textResult(string(data)), nil
}
```

**Add to `toolList()`:**

```go
{
    Name: "get_discovered_plans",
    Description: "Scan for plan/task/orchestration files in a project directory. Returns nexus orchestrator.json, markdown task files, Cursor rules, MCP configs, and more.",
    InputSchema: inputSchema{
        Type: "object",
        Properties: map[string]property{
            "projectPath": {Type: "string", Description: "Absolute path to the project root to scan"},
        },
    },
},
```

Tool count goes **29 → 30**.

**Update `TestMCP_ToolsList_Returns29Tools`** → `TestMCP_ToolsList_Returns30Tools` (update test name and assertion count).

**Update `toolHowto()`** inline guide to add `get_discovered_plans` to the tool list.

## Acceptance Criteria

- [ ] `get_discovered_plans` tool exists in `toolList()` (30 total)
- [ ] `handleToolCall` routes it correctly
- [ ] `TestMCP_ToolsList_Returns30Tools` passes
- [ ] `go build ./...` and `go test ./internal/adapters/inbound/mcp/ -race -count=1` pass
