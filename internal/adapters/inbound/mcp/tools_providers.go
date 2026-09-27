package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"nexus-orchestrator/internal/core/domain"
)

// ----- Provider Handlers -----

func (s *Server) toolGetProviders() (callToolResult, error) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: %w", err)
	}
	b, err := json.Marshal(providers)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolHealth() (callToolResult, error) {
	b, _ := json.Marshal(map[string]string{"status": "ok"})
	return textResult(string(b)), nil
}

func (s *Server) toolDiscoverProviders(ctx context.Context) (callToolResult, error) {
	providers, err := s.orch.TriggerScan(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: discover_providers: %w", err)
	}
	b, err := json.Marshal(providers)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: discover_providers: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolPromoteProvider(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_provider: invalid arguments: %w", err)
	}
	if err := s.orch.PromoteProvider(ctx, p.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return callToolResult{}, &mcpError{code: codeInvalidParams, msg: fmt.Sprintf("provider not found: %s", p.ID)}
		}
		return callToolResult{}, fmt.Errorf("mcp: promote_provider: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"promoted": true, "id": p.ID})
	return textResult(string(b)), nil
}

func (s *Server) toolListProviderConfigs(ctx context.Context) (callToolResult, error) {
	cfgs, err := s.orch.ListProviderConfigs(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: list_provider_configs: %w", err)
	}
	b, _ := json.Marshal(cfgs)
	return textResult(string(b)), nil
}

func (s *Server) toolAddProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid add_provider_config params"}
	}
	if p.Kind == "" || p.Name == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "kind and name are required"}
	}
	cfg := domain.ProviderConfig{Kind: domain.ProviderKind(p.Kind), Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Enabled: p.Enabled}
	saved, err := s.orch.AddProviderConfig(ctx, cfg)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: add_provider_config: %w", err)
	}
	b, _ := json.Marshal(saved)
	return textResult(string(b)), nil
}

func (s *Server) toolUpdateProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid update_provider_config params"}
	}
	if p.ID == "" || p.Kind == "" || p.Name == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "id, kind and name are required"}
	}
	cfg := domain.ProviderConfig{ID: p.ID, Kind: domain.ProviderKind(p.Kind), Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Enabled: p.Enabled}
	updated, err := s.orch.UpdateProviderConfig(ctx, cfg)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_provider_config: %w", err)
	}
	b, _ := json.Marshal(updated)
	return textResult(string(b)), nil
}

func (s *Server) toolRemoveProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid remove_provider_config params"}
	}
	if p.ID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "id is required"}
	}
	if err := s.orch.RemoveProviderConfig(ctx, p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: remove_provider_config: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"ok": true})
	return textResult(string(b)), nil
}

// ----- Provider Schema Definitions -----

func providerToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "get_providers",
			Description: "List available LLM providers and their models.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "health",
			Description: "Check that the nexusOrchestrator daemon is reachable.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "discover_providers",
			Description: "Scan the local system for installed AI providers/agents and return discovered results",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "promote_provider",
			Description: "Promote a discovered provider to an active LLM backend",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "ID of the discovered provider to promote"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "list_provider_configs",
			Description: "List all persisted LLM provider configuration records.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "add_provider_config",
			Description: "Add a new LLM provider configuration and register it when enabled=true.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"kind":     {Type: "string", Description: "Provider kind (e.g. lmstudio, ollama, openai, anthropic)."},
					"name":     {Type: "string", Description: "Display name for the provider."},
					"base_url": {Type: "string", Description: "API base URL (e.g. http://127.0.0.1:1234/v1)."},
					"api_key":  {Type: "string", Description: "API key if required."},
					"enabled":  {Type: "boolean", Description: "Whether to activate the provider immediately."},
				},
				Required: []string{"kind", "name"},
			},
		},
		{
			Name:        "update_provider_config",
			Description: "Update an existing LLM provider configuration by ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":       {Type: "string", Description: "Provider config ID to update."},
					"kind":     {Type: "string", Description: "Provider kind."},
					"name":     {Type: "string", Description: "Display name."},
					"base_url": {Type: "string", Description: "API base URL."},
					"api_key":  {Type: "string", Description: "API key."},
					"enabled":  {Type: "boolean", Description: "Whether the provider is active."},
				},
				Required: []string{"id", "kind", "name"},
			},
		},
		{
			Name:        "remove_provider_config",
			Description: "Delete a persisted provider configuration and deregister its adapter.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Provider config ID to remove."},
				},
				Required: []string{"id"},
			},
		},
	}
}
