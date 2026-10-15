package mcp_test

import (
	"encoding/json"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestSubmitTaskForwardsRoutingFields(t *testing.T) {
	orch := &mockOrch{submitID: "t1", createDraftID: "d1"}
	srv := newServer(t, orch)
	args := map[string]any{
		"projectPath": "/p", "targetFile": "a.go", "instruction": "do it",
		"providerName": "GitHub Copilot", "providerHint": "Ollama", "modelId": "gpt-4o",
		"role": "architect", "priority": 1, "tags": []string{"x", "y"},
	}
	for _, name := range []string{"submit_task", "create_draft"} {
		r := postRPC(t, srv, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": name, "arguments": args}})
		if r.Error != nil {
			t.Fatalf("%s: %+v", name, r.Error)
		}
	}
	for name, got := range map[string]domain.Task{"submit_task": orch.submitTask, "create_draft": orch.draftTask} {
		b, _ := json.Marshal(got)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if got.ProviderName != "GitHub Copilot" || got.ProviderHint != "Ollama" || got.ModelID != "gpt-4o" ||
			got.Role != "architect" || got.Priority != 1 || len(got.Tags) != 2 {
			t.Errorf("%s dropped routing fields: %s", name, b)
		}
	}
}
