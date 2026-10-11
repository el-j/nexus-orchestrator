package services_test

import (
	"reflect"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func chainFor(t *testing.T, task domain.Task, names ...string) []string {
	t.Helper()
	var llms []ports.LLMClient
	for _, n := range names {
		llms = append(llms, &scriptLLM{name: n, alive: true, limit: 100000})
	}
	orch := newEngine(t, &noopWriter{}, nil, nil, llms...).orch
	got, err := orch.ResolveChain(task)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return got
}

// Issue #53: the fallback order is deterministic - frontier cloud, then other
// cloud, then local - and a role can flip it. GitHub Copilot is a cloud
// frontier backend; "LM Studio" (with the space the adapter really reports) is local.
func TestProviderOrderIsDeterministicFrontierThenCloudThenLocal(t *testing.T) {
	names := []string{"LM Studio", "Ollama", "Acme Gateway", "GitHub Copilot", "Anthropic"}
	cases := []struct {
		role string
		want []string
	}{
		{"", []string{"GitHub Copilot", "Anthropic", "Acme Gateway", "LM Studio", "Ollama"}},
		{"architect", []string{"GitHub Copilot", "Anthropic", "Acme Gateway", "LM Studio", "Ollama"}},
		{"linter", []string{"LM Studio", "Ollama", "Acme Gateway", "GitHub Copilot", "Anthropic"}},
	}
	for _, c := range cases {
		t.Run("role="+c.role, func(t *testing.T) {
			got := chainFor(t, domain.Task{Role: c.role}, names...)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("chain = %v, want %v", got, c.want)
			}
		})
	}
}
