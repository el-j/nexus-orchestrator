package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// MCP clients are AI agents: a credential in a tool result lands in the model's
// context window and the agent's logs. Provider API keys must never be returned.
func TestMCPProviderConfigTools_NeverReturnAPIKeysAndKeepStoredOnOmission(t *testing.T) {
	ts := newMCPTestStack(t)
	cfgRepo := repo_sqlite.NewProviderConfigRepo(ts.repo)
	ts.orch.WithProviderConfigRepo(cfgRepo)
	ctx := context.Background()

	added := extractToolText(t, callTool(t, ts.srv, 1, "add_provider_config", map[string]any{
		"kind": "openaicompat", "name": "cloud", "base_url": "https://api.example.com/v1",
		"api_key": "sk-real-key-wxyz", "enabled": false,
	}))
	if strings.Contains(added, "sk-real-key") || !strings.Contains(added, "****wxyz") {
		t.Fatalf("add_provider_config leaked or failed to mask the key: %s", added)
	}
	var created domain.ProviderConfig
	if err := json.Unmarshal([]byte(added), &created); err != nil || created.ID == "" {
		t.Fatalf("decode: %v %s", err, added)
	}
	storedKey := func() string {
		t.Helper()
		got, err := cfgRepo.GetProviderConfig(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got.APIKey
	}
	if storedKey() != "sk-real-key-wxyz" {
		t.Fatalf("the real key must still be stored, got %q", storedKey())
	}

	listed := extractToolText(t, callTool(t, ts.srv, 2, "list_provider_configs", map[string]any{}))
	if strings.Contains(listed, "sk-real-key") || !strings.Contains(listed, "****wxyz") {
		t.Errorf("list_provider_configs leaked or failed to mask the key: %s", listed)
	}

	update := func(id int, args map[string]any) string {
		args["id"], args["kind"], args["name"] = created.ID, "openaicompat", "cloud"
		return extractToolText(t, callTool(t, ts.srv, id, "update_provider_config", args))
	}

	// Omitting api_key must NOT clear the stored credential.
	out := update(3, map[string]any{"base_url": "https://api.example.com/v2"})
	if strings.Contains(out, "sk-real-key") || storedKey() != "sk-real-key-wxyz" {
		t.Errorf("an omitted api_key destroyed the stored key: stored=%q resp=%s", storedKey(), out)
	}
	// Echoing the masked value back also keeps it.
	update(4, map[string]any{"api_key": "****wxyz"})
	if storedKey() != "sk-real-key-wxyz" {
		t.Errorf("masked placeholder replaced the key: %q", storedKey())
	}
	// A new key replaces it, and the response still shows only the mask.
	out = update(5, map[string]any{"api_key": "sk-new-key-1234"})
	if storedKey() != "sk-new-key-1234" || strings.Contains(out, "sk-new-key") || !strings.Contains(out, "****1234") {
		t.Errorf("rotation: stored=%q resp=%s", storedKey(), out)
	}
	// Explicit empty clears.
	update(6, map[string]any{"api_key": ""})
	if storedKey() != "" {
		t.Errorf("explicit empty must clear the key, got %q", storedKey())
	}
}
