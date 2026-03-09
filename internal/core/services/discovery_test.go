package services_test

import (
	"testing"

	"nexus-ai/internal/core/services"
)

// mockLLMClient is a test double for ports.LLMClient.
type mockLLMClient struct {
	alive    bool
	name     string
	code     string
	codeErr  error
}

func (m *mockLLMClient) Ping() bool                        { return m.alive }
func (m *mockLLMClient) ProviderName() string              { return m.name }
func (m *mockLLMClient) GetAvailableModels() ([]string, error) { return []string{"test-model"}, nil }
func (m *mockLLMClient) GenerateCode(prompt string) (string, error) {
	return m.code, m.codeErr
}

func TestDiscoveryService_DetectActive_ReturnsFirstAlive(t *testing.T) {
	dead := &mockLLMClient{alive: false, name: "dead"}
	alive := &mockLLMClient{alive: true, name: "alive"}

	svc := services.NewDiscoveryService(dead, alive)
	got := svc.DetectActive()

	if got == nil {
		t.Fatal("expected an active client, got nil")
	}
	if got.ProviderName() != "alive" {
		t.Errorf("expected provider %q, got %q", "alive", got.ProviderName())
	}
}

func TestDiscoveryService_DetectActive_ReturnsNilWhenNoneAlive(t *testing.T) {
	d1 := &mockLLMClient{alive: false, name: "d1"}
	d2 := &mockLLMClient{alive: false, name: "d2"}

	svc := services.NewDiscoveryService(d1, d2)
	if got := svc.DetectActive(); got != nil {
		t.Errorf("expected nil, got %q", got.ProviderName())
	}
}

func TestDiscoveryService_DetectActive_EmptyList(t *testing.T) {
	svc := services.NewDiscoveryService()
	if got := svc.DetectActive(); got != nil {
		t.Errorf("expected nil for empty client list, got %v", got)
	}
}
