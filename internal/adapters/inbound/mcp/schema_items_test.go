package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestToolSchemaArrayItems verifies every array-type property has items defined
// (required by JSON Schema / MCP client validators).
func TestToolSchemaArrayItems(t *testing.T) {
	srv := NewServer(nil)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp", "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}

	var rpcResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		t.Fatal(err)
	}

	tools := rpcResp["result"].(map[string]any)["tools"].([]any)
	for _, tool := range tools {
		tm := tool.(map[string]any)
		name := tm["name"].(string)
		schema := tm["inputSchema"].(map[string]any)
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			continue
		}
		for pname, pv := range props {
			pm, ok := pv.(map[string]any)
			if !ok {
				continue
			}
			if pm["type"] == "array" {
				if _, hasItems := pm["items"]; !hasItems {
					t.Errorf("tool %s property %s: type=array but missing items", name, pname)
				} else {
					t.Logf("OK: %s.%s has items", name, pname)
				}
			}
		}
	}
}
