package services_test

import (
	"fmt"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// mockLLMClient is a test double for ports.LLMClient.
type mockLLMClient struct {
	alive        bool
	name         string
	code         string
	codeErr      error
	contextLimit int
	activeModel  string
	models       []string
}

func (m *mockLLMClient) Ping() bool                                 { return m.alive }
func (m *mockLLMClient) ProviderName() string                       { return m.name }
func (m *mockLLMClient) ActiveModel() string                        { return m.activeModel }
func (m *mockLLMClient) GetAvailableModels() ([]string, error)      { return m.models, nil }
func (m *mockLLMClient) ContextLimit() int                          { return m.contextLimit }
func (m *mockLLMClient) GenerateCode(prompt string) (string, error) { return m.code, m.codeErr }
func (m *mockLLMClient) Chat(_ []domain.Message) (string, error)    { return m.code, m.codeErr }

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

func TestDiscovery_FindForModel_ActiveModelMatch(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "p1", activeModel: "llama3"}
	p2 := &mockLLMClient{alive: true, name: "p2", activeModel: "codellama"}
	svc := services.NewDiscoveryService(p1, p2)

	got, err := svc.FindForModel("codellama", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderName() != "p2" {
		t.Errorf("expected p2, got %q", got.ProviderName())
	}
}

func TestDiscovery_FindForModel_ModelListFallback(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "p1", activeModel: "other", models: []string{"mistral", "codellama"}}
	svc := services.NewDiscoveryService(p1)

	got, err := svc.FindForModel("codellama", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderName() != "p1" {
		t.Errorf("expected p1, got %q", got.ProviderName())
	}
}

func TestDiscovery_FindForModel_NoProvider(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "p1", activeModel: "llama3", models: []string{"llama3"}}
	svc := services.NewDiscoveryService(p1)

	_, err := svc.FindForModel("gpt-4o", "")
	if err == nil {
		t.Fatal("expected error for unavailable model, got nil")
	}
}

func TestDiscovery_FindForModel_ProviderHintFirst(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "OpenAI", activeModel: "gpt-4o", models: []string{"gpt-4o"}}
	p2 := &mockLLMClient{alive: true, name: "Ollama", activeModel: "gpt-4o", models: []string{"gpt-4o"}}
	svc := services.NewDiscoveryService(p2, p1) // p2 is first but hint targets p1

	got, err := svc.FindForModel("gpt-4o", "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderName() != "OpenAI" {
		t.Errorf("expected OpenAI (hint match first), got %q", got.ProviderName())
	}
}

func TestDiscovery_FindForModel_EmptyModelIDUsesDetectActive(t *testing.T) {
	p1 := &mockLLMClient{alive: false, name: "dead"}
	p2 := &mockLLMClient{alive: true, name: "alive"}
	svc := services.NewDiscoveryService(p1, p2)

	got, err := svc.FindForModel("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderName() != "alive" {
		t.Errorf("expected alive, got %q", got.ProviderName())
	}
}

func TestListProviders_IncludesActiveModel(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "LM Studio", activeModel: "llama3", models: []string{"llama3", "codellama"}}
	p2 := &mockLLMClient{alive: false, name: "Ollama"}
	svc := services.NewDiscoveryService(p1, p2)

	infos := svc.ListProviders()
	if len(infos) != 2 {
		t.Fatalf("expected 2 provider infos, got %d", len(infos))
	}
	if infos[0].ActiveModel != "llama3" {
		t.Errorf("expected ActiveModel=llama3, got %q", infos[0].ActiveModel)
	}
	if len(infos[0].Models) != 2 {
		t.Errorf("expected 2 models for p1, got %d", len(infos[0].Models))
	}
	if infos[1].ActiveModel != "" {
		t.Errorf("expected empty ActiveModel for offline provider, got %q", infos[1].ActiveModel)
	}
}

// --- RemoveProvider / GetClientByName / concurrent safety ---

func TestDiscovery_RemoveProvider_Found(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "alpha"}
	p2 := &mockLLMClient{alive: true, name: "beta"}
	svc := services.NewDiscoveryService(p1, p2)

	if !svc.RemoveProvider("alpha") {
		t.Fatal("expected RemoveProvider to return true for a known provider")
	}
	infos := svc.ListProviders()
	if len(infos) != 1 || infos[0].Name != "beta" {
		t.Errorf("expected only 'beta' remaining, got %+v", infos)
	}
}

func TestDiscovery_RemoveProvider_NotFound(t *testing.T) {
	p1 := &mockLLMClient{alive: true, name: "alpha"}
	svc := services.NewDiscoveryService(p1)

	if svc.RemoveProvider("nonexistent") {
		t.Fatal("expected RemoveProvider to return false for unknown provider")
	}
	if len(svc.ListProviders()) != 1 {
		t.Error("provider list should be unchanged after failed removal")
	}
}

func TestDiscovery_RemoveProvider_CaseInsensitive(t *testing.T) {
	p := &mockLLMClient{alive: true, name: "LMStudio"}
	svc := services.NewDiscoveryService(p)

	if !svc.RemoveProvider("lmstudio") {
		t.Fatal("expected case-insensitive match to succeed")
	}
	if len(svc.ListProviders()) != 0 {
		t.Error("provider should have been removed")
	}
}

func TestDiscovery_GetClientByName_Found(t *testing.T) {
	p := &mockLLMClient{alive: true, name: "mysvc"}
	svc := services.NewDiscoveryService(p)

	got, ok := svc.GetClientByName("mysvc")
	if !ok {
		t.Fatal("expected GetClientByName to find 'mysvc'")
	}
	if got.ProviderName() != "mysvc" {
		t.Errorf("unexpected name: %q", got.ProviderName())
	}
}

func TestDiscovery_GetClientByName_NotFound(t *testing.T) {
	svc := services.NewDiscoveryService()

	_, ok := svc.GetClientByName("ghost")
	if ok {
		t.Fatal("expected GetClientByName to return false for unknown provider")
	}
}

func TestDiscovery_RegisterProvider_ConcurrentSafe(t *testing.T) {
	svc := services.NewDiscoveryService()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			svc.RegisterProvider(&mockLLMClient{alive: true, name: fmt.Sprintf("p%d", n)})
		}(i)
	}
	wg.Wait()
	infos := svc.ListProviders()
	if len(infos) != 20 {
		t.Errorf("expected 20 providers after concurrent registration, got %d", len(infos))
	}
}
