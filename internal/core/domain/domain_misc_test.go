package domain

import "testing"

func TestLookupBuiltInProfile(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string // expected ModelID, "" for nil
	}{
		{"exact", "qwen3-coder-next", "qwen3-coder-next"},
		{"case-insensitive", "LLaMA3.2", "llama3.2"},
		{"key longer than profile id", "qwen3-coder-next-q4_K_M", "qwen3-coder-next"},
		{"key shorter than profile id", "codest", "codestral"},
		{"surrounding whitespace", "  gemma-2b  ", "gemma"},
		{"unknown", "gpt-9000", ""},
		{"empty must not match everything", "", ""},
		{"blank must not match everything", "   \t", ""},
	}
	for _, tc := range cases {
		got := LookupBuiltInProfile(tc.key)
		switch {
		case tc.want == "" && got != nil:
			t.Errorf("%s: LookupBuiltInProfile(%q) = %q, want nil", tc.name, tc.key, got.ModelID)
		case tc.want != "" && (got == nil || got.ModelID != tc.want):
			t.Errorf("%s: LookupBuiltInProfile(%q) = %v, want %q", tc.name, tc.key, got, tc.want)
		}
	}
}

func TestBuiltInModelProfiles_AreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range BuiltInModelProfiles {
		if p.ModelID == "" || seen[p.ModelID] {
			t.Errorf("profile id %q is empty or duplicated", p.ModelID)
		}
		seen[p.ModelID] = true
		if !p.BuiltIn {
			t.Errorf("%s: BuiltIn must be true", p.ModelID)
		}
		if p.ContextWindow <= 0 || p.RecommendedMaxOutput <= 0 || p.RecommendedMaxOutput >= p.ContextWindow {
			t.Errorf("%s: implausible limits ctx=%d out=%d", p.ModelID, p.ContextWindow, p.RecommendedMaxOutput)
		}
	}
}

func TestStringers(t *testing.T) {
	if ProviderKindOllama.String() != "ollama" || ProviderKindGemini.String() != "gemini" {
		t.Error("ProviderKind.String must return the raw value")
	}
	if StatusQueued.String() != "QUEUED" || StatusNoProvider.String() != "NO_PROVIDER" {
		t.Error("TaskStatus.String must return the raw value")
	}
	if CommandExecute.String() != "execute" || CommandType("").String() != "" {
		t.Error("CommandType.String must return the raw value")
	}
}

func TestCommandType_IsValid(t *testing.T) {
	for _, c := range []CommandType{CommandPlan, CommandExecute, CommandAuto, ""} {
		if !c.IsValid() {
			t.Errorf("%q must be valid", c)
		}
	}
	for _, c := range []CommandType{"deploy", "PLAN", "Execute", " "} {
		if c.IsValid() {
			t.Errorf("%q must be invalid (matching is exact)", c)
		}
	}
}

func TestIsMaskedSecret(t *testing.T) {
	for _, s := range []string{"****", "****abcd", MaskedSecretPrefix + "x"} {
		if !IsMaskedSecret(s) {
			t.Errorf("%q is a masked placeholder", s)
		}
	}
	for _, s := range []string{"", "sk-real", "***abcd", "abcd****"} {
		if IsMaskedSecret(s) {
			t.Errorf("%q is not masked", s)
		}
	}
}

func TestMaskSecretAndMasked(t *testing.T) {
	cases := map[string]string{"": "", "a": "****", "abcd": "****", "abcde": "****bcde", "sk-real-key-wxyz": "****wxyz"}
	for in, want := range cases {
		if got := MaskSecret(in); got != want {
			t.Errorf("MaskSecret(%q) = %q, want %q", in, got, want)
		}
	}
	cfg := ProviderConfig{Name: "n", APIKey: "sk-secret-9999"}
	m := cfg.Masked()
	if m.APIKey != "****9999" || cfg.APIKey != "sk-secret-9999" || m.Name != "n" {
		t.Errorf("Masked must copy, not mutate: %+v / %+v", m, cfg)
	}
}
