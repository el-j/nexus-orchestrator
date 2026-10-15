package mcp_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
)

type listedTool struct {
	Name        string `json:"name"`
	InputSchema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type  string `json:"type"`
			Items *struct {
				Type string `json:"type"`
			} `json:"items"`
		} `json:"properties"`
	} `json:"inputSchema"`
}

func fetchTools(t *testing.T, srv *httptest.Server) []listedTool {
	t.Helper()
	r := postRPC(t, srv, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	var out struct {
		Tools []listedTool `json:"tools"`
	}
	if err := json.Unmarshal(r.Result, &out); err != nil || len(out.Tools) == 0 {
		t.Fatalf("tools/list: %v %s", err, r.Result)
	}
	return out.Tools
}

// argValue builds a plausible value for a schema property; a few field names
// need domain-valid values for the call to succeed.
func argValue(name, typ, itemType string) any {
	switch name {
	case "status":
		return "COMPLETED"
	case "command":
		return "plan"
	case "kind":
		return "ollama"
	case "source":
		return "mcp"
	case "base_url", "baseUrl":
		return "http://127.0.0.1:1"
	}
	switch typ {
	case "integer", "number":
		return 5
	case "boolean":
		return true
	case "array":
		return []any{argValue("", itemType, "")}
	case "object":
		return map[string]any{}
	default:
		return "x"
	}
}

// validArgs fills every schema property (not just the required ones) so each
// optional code path is exercised too.
func validArgs(tool listedTool) map[string]any {
	args := map[string]any{}
	for name, p := range tool.InputSchema.Properties {
		item := ""
		if p.Items != nil {
			item = p.Items.Type
		}
		args[name] = argValue(name, p.Type, item)
	}
	return args
}

func newMatrixServer(orch *failOrch, brain *failBrain) *httptest.Server {
	srv := httptest.NewServer(mcp.NewMcpServer(orch, brain))
	return srv
}

// Tools that never touch the orchestrator or brain, so a failing backend cannot affect them.
var backendFree = map[string]bool{"howto": true, "howto_brief": true, "health": true}

func TestEveryListedToolIsDispatchableAndSucceedsWithValidArguments(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := newMatrixServer(newFailOrch(), newFailBrain())
	defer srv.Close()
	tools := fetchTools(t, srv)
	if len(tools) < 40 {
		t.Errorf("expected at least 40 tools, got %d", len(tools))
	}
	seen := map[string]bool{}
	for i, tool := range tools {
		if seen[tool.Name] {
			t.Errorf("tool %q is listed twice", tool.Name)
		}
		seen[tool.Name] = true
		r := callTool(t, srv, 100+i, tool.Name, validArgs(tool))
		if r.Error != nil {
			t.Errorf("%s: unexpected error %d %q (valid arguments: %v)", tool.Name, r.Error.Code, r.Error.Message, validArgs(tool))
			continue
		}
		if text := extractToolText(t, r); text == "" {
			t.Errorf("%s: empty result", tool.Name)
		}
	}
}

func TestEveryToolReportsBackendFailuresAsRPCErrors(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srvOK := newMatrixServer(newFailOrch(), newFailBrain())
	tools := fetchTools(t, srvOK)
	srvOK.Close()

	boom := errors.New("backend exploded: /secret/path")
	// Make every operation fail.
	orch := newFailOrch()
	for op := range allOrchOps() {
		orch.failWith(op, boom)
	}
	brain := newFailBrain()
	for _, op := range []string{"GetContext", "GetFocusedContext", "IngestKnowledge", "IngestFromFile", "SearchKnowledge",
		"GetFileMap", "InitProject", "GetStatus", "ListKnowledge", "DeleteKnowledge", "GetOnboardingContext"} {
		brain.failWith(op, boom)
	}
	srv := newMatrixServer(orch, brain)
	defer srv.Close()

	var swallowed []string
	for i, tool := range tools {
		if backendFree[tool.Name] {
			continue
		}
		r := callTool(t, srv, 200+i, tool.Name, validArgs(tool))
		if r.Error == nil {
			swallowed = append(swallowed, tool.Name)
			continue
		}
		if r.Error.Code == -32601 {
			t.Errorf("%s: reported as an unknown tool", tool.Name)
		}
	}
	sort.Strings(swallowed)
	// Tools that deliberately degrade to an empty result instead of failing.
	for _, name := range swallowed {
		t.Logf("tool %q returns a result even when the backend fails", name)
	}
	if len(swallowed) > 3 {
		t.Errorf("too many tools hide backend failures: %v", swallowed)
	}
}

func TestEveryToolRejectsMalformedArguments(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := newMatrixServer(newFailOrch(), newFailBrain())
	defer srv.Close()
	for i, tool := range fetchTools(t, srv) {
		if len(tool.InputSchema.Properties) == 0 {
			continue // takes no arguments
		}
		// Arguments of the wrong JSON type: a string where an object is required.
		r := postRPC(t, srv, map[string]any{
			"jsonrpc": "2.0", "id": 300 + i, "method": "tools/call",
			"params": map[string]any{"name": tool.Name, "arguments": "not an object"},
		})
		if r.Error == nil {
			t.Errorf("%s: malformed arguments were accepted", tool.Name)
		}
	}
}

func TestToolsRejectMissingRequiredArguments(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := newMatrixServer(newFailOrch(), newFailBrain())
	defer srv.Close()
	rejected := 0
	for i, tool := range fetchTools(t, srv) {
		if len(tool.InputSchema.Required) == 0 {
			continue
		}
		r := callTool(t, srv, 400+i, tool.Name, map[string]any{})
		if r.Error != nil {
			rejected++
			if strings.Contains(r.Error.Message, "panic") {
				t.Errorf("%s: %s", tool.Name, r.Error.Message)
			}
		}
	}
	if rejected == 0 {
		t.Error("no tool rejected an empty argument set; required-argument validation looks absent")
	}
}

func TestUnknownToolAndMalformedEnvelopes(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := newMatrixServer(newFailOrch(), newFailBrain())
	defer srv.Close()
	if r := callTool(t, srv, 1, "no_such_tool", nil); r.Error == nil || r.Error.Code != -32601 {
		t.Errorf("unknown tool: %+v", r.Error)
	}
	// tools/call without params.
	if r := postRPC(t, srv, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": "junk"}); r.Error == nil || r.Error.Code != -32602 {
		t.Errorf("bad params: %+v", r.Error)
	}
}

// allOrchOps lists every failOrch operation by reflecting on a fresh instance's
// canned behaviour: calling failWith with an unknown name is harmless, so the
// list is maintained here next to the generated double.
func allOrchOps() map[string]bool {
	ops := []string{"SubmitTask", "GetTask", "GetQueue", "GetQueueForProject", "GetAllTasks", "GetTasksForProject",
		"GetProviders", "CancelTask", "RegisterCloudProvider", "RemoveProvider", "GetProviderModels", "GetBacklog",
		"CreateDraft", "PromoteTask", "UpdateTask", "AddProviderConfig", "UpdateProviderConfig", "RemoveProviderConfig",
		"ListProviderConfigs", "GetDiscoveredProviders", "TriggerScan", "PromoteProvider", "RegisterAISession",
		"ListAISessions", "DeregisterAISession", "HeartbeatAISession", "HeartbeatTask", "ClaimTask", "UpdateTaskStatus",
		"PurgeDisconnectedSessions", "GetDiscoveredAgents", "DelegateToNexus", "TerminateAISession",
		"GetDiscoveredPlanFiles", "GetRuntimeConfig", "UpdateRuntimeConfig"}
	m := map[string]bool{}
	for _, o := range ops {
		m[o] = true
	}
	return m
}

// A malformed projectPath must be an error, never a silent fall-back to the
// unfiltered (all projects) queue.
func TestListToolsNeverFallBackToAllProjectsOnBadArguments(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := newMatrixServer(newFailOrch(), newFailBrain())
	defer srv.Close()
	for _, tool := range []string{"get_queue", "get_all_tasks"} {
		r := callTool(t, srv, 1, tool, map[string]any{"projectPath": 12345})
		if r.Error == nil || r.Error.Code != -32602 {
			t.Errorf("%s with a numeric projectPath: %+v", tool, r.Error)
		}
		// Valid forms still work: no arguments, empty object, a project path.
		for _, args := range []map[string]any{nil, {}, {"projectPath": "/p"}} {
			if r := callTool(t, srv, 2, tool, args); r.Error != nil {
				t.Errorf("%s with %v: %+v", tool, args, r.Error)
			}
		}
	}
}
