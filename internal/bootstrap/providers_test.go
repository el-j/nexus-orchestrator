package bootstrap_test

import (
	"os"
	"testing"

	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/domain"
)

func TestBuildProviders_WithGemini(t *testing.T) {
	origKey := os.Getenv("GEMINI_API_KEY")
	origModel := os.Getenv("GEMINI_MODEL")
	defer func() {
		os.Setenv("GEMINI_API_KEY", origKey)
		os.Setenv("GEMINI_MODEL", origModel)
	}()

	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	os.Setenv("GEMINI_MODEL", "gemini-2.5-flash")

	providers := bootstrap.BuildProviders()
	found := false
	for _, p := range providers {
		if p.ProviderName() == "Google Gemini" {
			found = true
			if p.ActiveModel() != "gemini-2.5-flash" {
				t.Errorf("expected model gemini-2.5-flash, got %s", p.ActiveModel())
			}
		}
	}
	if !found {
		t.Error("expected Google Gemini to be instantiated when GEMINI_API_KEY is set")
	}
}

func TestBuildProviderFromConfig_Gemini(t *testing.T) {
	cfg := domain.ProviderConfig{
		Kind:    domain.ProviderKindGemini,
		Name:    "MyGemini",
		APIKey:  "my-key",
		Model:   "gemini-2.5-pro",
		BaseURL: "https://custom.gemini.endpoint",
	}

	p, err := bootstrap.BuildProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ProviderName() != "Google Gemini" {
		t.Errorf("expected 'Google Gemini', got %s", p.ProviderName())
	}
	if p.ActiveModel() != "gemini-2.5-pro" {
		t.Errorf("expected 'gemini-2.5-pro', got %s", p.ActiveModel())
	}
	if p.BaseURL() != "https://custom.gemini.endpoint" {
		t.Errorf("expected 'https://custom.gemini.endpoint', got %s", p.BaseURL())
	}
}

func TestBuildProviderFromConfig_AllKinds(t *testing.T) {
	cases := []struct {
		kind domain.ProviderKind
		name string
	}{
		{domain.ProviderKindLMStudio, "LM Studio"},
		{domain.ProviderKindOllama, "Ollama"},
		{domain.ProviderKindAnthropic, "Anthropic"},
		{domain.ProviderKindOpenAICompat, "OpenAI"},
		{domain.ProviderKindLocalAI, "LocalAI"},
		{domain.ProviderKindGemini, "Google Gemini"},
	}

	for _, tc := range cases {
		cfg := domain.ProviderConfig{
			Kind:   tc.kind,
			Name:   tc.name,
			APIKey: "key",
			Model:  "model",
		}
		p, err := bootstrap.BuildProviderFromConfig(cfg)
		if err != nil {
			t.Errorf("failed to build provider for kind %s: %v", tc.kind, err)
			continue
		}
		if p == nil {
			t.Errorf("expected non-nil provider for kind %s", tc.kind)
		}
	}

	// Unknown kind
	_, err := bootstrap.BuildProviderFromConfig(domain.ProviderConfig{Kind: "unknown_vendor"})
	if err == nil {
		t.Error("expected error for unknown kind, got nil")
	}
}
