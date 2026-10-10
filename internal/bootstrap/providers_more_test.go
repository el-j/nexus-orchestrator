package bootstrap

import (
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func clearProviderEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"NEXUS_LMSTUDIO_URL", "NEXUS_OLLAMA_URL", "NEXUS_OPENAI_API_KEY", "NEXUS_OPENAI_MODEL",
		"NEXUS_GITHUBCOPILOT_TOKEN", "NEXUS_GITHUBCOPILOT_MODEL", "NEXUS_ANTHROPIC_API_KEY", "NEXUS_ANTHROPIC_MODEL",
		"NEXUS_GEMINI_API_KEY", "GEMINI_API_KEY", "NEXUS_GEMINI_MODEL", "GEMINI_MODEL", "NEXUS_ANTIGRAVITY_URL"} {
		t.Setenv(k, "")
	}
}

func names(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, p := range BuildProviders() {
		out[p.ProviderName()] = true
	}
	return out
}

func TestBuildProviders_LocalAlwaysCloudOnlyWithCredentials(t *testing.T) {
	clearProviderEnv(t)
	got := names(t)
	for _, want := range []string{"LM Studio", "Ollama", "Antigravity"} {
		if !got[want] {
			t.Errorf("local provider %q must always be present: %v", want, got)
		}
	}
	for _, cloud := range []string{"OpenAI", "GitHub Copilot", "Anthropic", "Google Gemini"} {
		if got[cloud] {
			t.Errorf("%s must not be registered without credentials", cloud)
		}
	}

	t.Setenv("NEXUS_OPENAI_API_KEY", "k")
	t.Setenv("NEXUS_GITHUBCOPILOT_TOKEN", "t")
	t.Setenv("NEXUS_ANTHROPIC_API_KEY", "a")
	t.Setenv("GEMINI_API_KEY", "g") // the unprefixed fallback variable
	t.Setenv("GEMINI_MODEL", "gemini-2.5-flash")
	got = names(t)
	for _, cloud := range []string{"OpenAI", "GitHub Copilot", "Anthropic", "Google Gemini"} {
		if !got[cloud] {
			t.Errorf("%s should be registered: %v", cloud, got)
		}
	}
	for _, p := range BuildProviders() {
		if p.ProviderName() == "Google Gemini" && p.ActiveModel() != "gemini-2.5-flash" {
			t.Errorf("GEMINI_MODEL fallback ignored: %q", p.ActiveModel())
		}
	}
}

func TestBuildProviders_HonoursURLAndModelOverrides(t *testing.T) {
	clearProviderEnv(t)
	t.Setenv("NEXUS_LMSTUDIO_URL", "http://lm.example/v1")
	t.Setenv("NEXUS_OLLAMA_URL", "http://ol.example")
	t.Setenv("NEXUS_OPENAI_API_KEY", "k")
	t.Setenv("NEXUS_OPENAI_MODEL", "gpt-x")
	t.Setenv("NEXUS_ANTHROPIC_API_KEY", "a")
	t.Setenv("NEXUS_ANTHROPIC_MODEL", "claude-x")
	t.Setenv("NEXUS_GEMINI_API_KEY", "g")
	t.Setenv("NEXUS_GEMINI_MODEL", "gemini-x")
	t.Setenv("NEXUS_ANTIGRAVITY_URL", "http://ag.example/v1")
	want := map[string][2]string{
		"LM Studio": {"http://lm.example/v1", ""}, "Ollama": {"http://ol.example", ""},
		"OpenAI": {"", "gpt-x"}, "Anthropic": {"", "claude-x"}, "Google Gemini": {"", "gemini-x"},
		"Antigravity": {"http://ag.example/v1", ""},
	}
	for _, p := range BuildProviders() {
		w, ok := want[p.ProviderName()]
		if !ok {
			continue
		}
		if w[0] != "" && p.BaseURL() != w[0] {
			t.Errorf("%s base URL %q, want %q", p.ProviderName(), p.BaseURL(), w[0])
		}
		if w[1] != "" && p.ActiveModel() != w[1] {
			t.Errorf("%s model %q, want %q", p.ProviderName(), p.ActiveModel(), w[1])
		}
	}
}

func TestBuildProviderFromConfig(t *testing.T) {
	ok := []struct {
		cfg     domain.ProviderConfig
		name    string
		baseURL string
	}{
		{domain.ProviderConfig{Kind: domain.ProviderKindLMStudio}, "LM Studio", "http://127.0.0.1:1234/v1"},
		{domain.ProviderConfig{Kind: domain.ProviderKindOllama, Model: "m"}, "Ollama", "http://127.0.0.1:11434"},
		{domain.ProviderConfig{Kind: domain.ProviderKindOpenAICompat, Name: "Mine", BaseURL: "http://x/v1", Model: "m"}, "Mine", "http://x/v1"},
		{domain.ProviderConfig{Kind: domain.ProviderKindAnthropic, Name: "A", APIKey: "k", Model: "m"}, "Anthropic", ""},
		{domain.ProviderConfig{Kind: domain.ProviderKindGemini, Name: "G", APIKey: "k"}, "Google Gemini", ""},
		{domain.ProviderConfig{Kind: domain.ProviderKindVLLM, Name: "V"}, "V", "http://127.0.0.1:4315/v1"},
		{domain.ProviderConfig{Kind: domain.ProviderKindLocalAI, Name: "L", BaseURL: "http://h:8080"}, "L", "http://h:8080/v1"},
		{domain.ProviderConfig{Kind: domain.ProviderKindTextGenUI, Name: "T", BaseURL: "http://h:5000/v1"}, "T", "http://h:5000/v1"},
		{domain.ProviderConfig{Kind: domain.ProviderKindDesktopApp, Name: "D", BaseURL: "http://h:1///"}, "D", "http://h:1/v1"},
	}
	for _, tc := range ok {
		c, err := BuildProviderFromConfig(tc.cfg)
		if err != nil || c == nil {
			t.Errorf("%s: %v", tc.cfg.Kind, err)
			continue
		}
		if c.ProviderName() != tc.name {
			t.Errorf("%s: name %q, want %q", tc.cfg.Kind, c.ProviderName(), tc.name)
		}
		if tc.baseURL != "" && c.BaseURL() != tc.baseURL {
			t.Errorf("%s: base URL %q, want %q", tc.cfg.Kind, c.BaseURL(), tc.baseURL)
		}
	}
}

// A provider that can never work must be rejected up front instead of being
// registered and then failing every request.
func TestOpenAICompatWithoutBaseURLDefaultsToOpenAI(t *testing.T) {
	for _, base := range []string{"", "   "} {
		c, err := BuildProviderFromConfig(domain.ProviderConfig{Kind: domain.ProviderKindOpenAICompat, Name: "OpenAI", BaseURL: base, APIKey: "k"})
		if err != nil || c.BaseURL() != "https://api.openai.com/v1" {
			t.Errorf("base %q: %v %v", base, c, err)
		}
	}
}

func TestBuildProviderFromConfig_RejectsUnusableConfigs(t *testing.T) {
	bad := []struct {
		cfg  domain.ProviderConfig
		want string
	}{
		{domain.ProviderConfig{Kind: domain.ProviderKindAnthropic, Name: "NoKey"}, "apiKey is required"},
		{domain.ProviderConfig{Kind: domain.ProviderKindGemini, Name: "NoKey"}, "apiKey is required"},
		{domain.ProviderConfig{Kind: domain.ProviderKind("nope"), Name: "X"}, "unknown provider kind"},
		{domain.ProviderConfig{Kind: domain.ProviderKindCLI, Name: "X"}, "unknown provider kind"},
	}
	for _, tc := range bad {
		if c, err := BuildProviderFromConfig(tc.cfg); err == nil || !strings.Contains(err.Error(), tc.want) || c != nil {
			t.Errorf("%s/%s: got %v %v, want error containing %q", tc.cfg.Kind, tc.cfg.Name, c, err, tc.want)
		}
	}
}
