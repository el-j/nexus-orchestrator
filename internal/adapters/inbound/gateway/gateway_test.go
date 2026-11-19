package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type fakeClient struct {
	name            string
	model           string
	baseURL         string
	availableModels []string
	alive           bool
	chatReply       string
	chatErr         error
	streamChunks    []string
	streamErr       error
	usage           *domain.Usage
}

type fakeNonStreamClient struct {
	name      string
	model     string
	alive     bool
	chatReply string
	chatErr   error
}

func (f *fakeNonStreamClient) Ping(_ context.Context) bool        { return f.alive }
func (f *fakeNonStreamClient) ProviderName() string               { return f.name }
func (f *fakeNonStreamClient) ActiveModel() string                { return f.model }
func (f *fakeNonStreamClient) BaseURL() string                    { return "http://local" }
func (f *fakeNonStreamClient) ContextLimit(_ context.Context) int { return 2048 }
func (f *fakeNonStreamClient) GenerateCode(_ context.Context, _ string) (string, error) {
	return f.chatReply, f.chatErr
}
func (f *fakeNonStreamClient) GetAvailableModels(_ context.Context) ([]string, error) {
	return nil, nil
}
func (f *fakeNonStreamClient) Chat(_ context.Context, _ []domain.Message) (string, error) {
	return f.chatReply, f.chatErr
}

func (f *fakeClient) Ping(_ context.Context) bool        { return f.alive }
func (f *fakeClient) ProviderName() string               { return f.name }
func (f *fakeClient) ActiveModel() string                { return f.model }
func (f *fakeClient) BaseURL() string                    { return f.baseURL }
func (f *fakeClient) ContextLimit(_ context.Context) int { return 4096 }
func (f *fakeClient) GenerateCode(_ context.Context, _ string) (string, error) {
	return f.chatReply, f.chatErr
}
func (f *fakeClient) GetAvailableModels(_ context.Context) ([]string, error) {
	return f.availableModels, nil
}
func (f *fakeClient) Chat(_ context.Context, _ []domain.Message) (string, error) {
	return f.chatReply, f.chatErr
}
func (f *fakeClient) ChatStream(_ context.Context, _ []domain.Message, onChunk func(string)) (string, error) {
	for _, chunk := range f.streamChunks {
		onChunk(chunk)
	}
	return f.chatReply, f.streamErr
}
func (f *fakeClient) LastUsage() (domain.Usage, bool) {
	if f.usage != nil {
		return *f.usage, true
	}
	return domain.Usage{}, false
}

func TestNewGateway_Defaults(t *testing.T) {
	gw := NewGateway(Config{}, nil)
	if gw.cfg.DefaultRole != "coder" {
		t.Fatalf("expected default role coder, got %q", gw.cfg.DefaultRole)
	}
}

func TestResolveChain_Errors(t *testing.T) {
	gwNil := NewGateway(Config{}, nil)
	if _, _, err := gwNil.ResolveChain(context.Background(), "coder", "."); err == nil {
		t.Fatal("expected error with nil discovery")
	}

	discEmpty := services.NewDiscoveryService()
	gwEmpty := NewGateway(Config{}, discEmpty)
	if _, _, err := gwEmpty.ResolveChain(context.Background(), "coder", "."); err == nil {
		t.Fatal("expected error with empty discovery")
	}

	dead := &fakeClient{name: "ollama", model: "qwen", alive: false}
	discDead := services.NewDiscoveryService(dead)
	gwDead := NewGateway(Config{}, discDead)
	chain, _, err := gwDead.ResolveChain(context.Background(), "coder", ".")
	if err != nil || len(chain) == 0 {
		t.Fatalf("expected fallback to all candidates when none alive, got err: %v", err)
	}
}

func TestResolveChain_Targets(t *testing.T) {
	c1 := &fakeClient{name: "Anthropic", model: "claude-3-5-sonnet", alive: true}
	c2 := &fakeClient{name: "Ollama", model: "codellama", availableModels: []string{"deepseek-coder"}, alive: true}
	disc := services.NewDiscoveryService(c1, c2)
	gw := NewGateway(Config{}, disc)

	// Target matching provider name
	chain, _, err := gw.ResolveChain(context.Background(), "ollama", ".")
	if err != nil || len(chain) == 0 || chain[0].ProviderName() != "Ollama" {
		t.Fatalf("expected Ollama preferred, got chain: %v, err: %v", chain, err)
	}

	// Target matching active model
	chain, _, err = gw.ResolveChain(context.Background(), "claude-3-5-sonnet", ".")
	if err != nil || len(chain) == 0 || chain[0].ActiveModel() != "claude-3-5-sonnet" {
		t.Fatalf("expected Anthropic preferred, got %v", chain)
	}

	// Target matching available model
	chain, _, err = gw.ResolveChain(context.Background(), "deepseek-coder", ".")
	if err != nil || len(chain) == 0 || chain[0].ProviderName() != "Ollama" {
		t.Fatalf("expected Ollama preferred for available model, got %v", chain)
	}
}

func TestResolveChain_RolesAndLadder(t *testing.T) {
	tmpDir := t.TempDir()
	nexusDir := filepath.Join(tmpDir, ".nexus")
	if err := os.MkdirAll(nexusDir, 0755); err != nil {
		t.Fatal(err)
	}
	policyJSON := `{
		"ladder": {
			"coder": ["Ollama/*", "Anthropic/*"]
		}
	}`
	if err := os.WriteFile(filepath.Join(nexusDir, "policy.json"), []byte(policyJSON), 0644); err != nil {
		t.Fatal(err)
	}

	c1 := &fakeClient{name: "Anthropic", model: "claude-3-5-sonnet", alive: true}
	c2 := &fakeClient{name: "Ollama", model: "codellama", alive: true}
	disc := services.NewDiscoveryService(c1, c2)
	gw := NewGateway(Config{}, disc)

	chain, ladder, err := gw.ResolveChain(context.Background(), "coder", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ladder) != 2 || chain[0].ProviderName() != "Ollama" {
		t.Fatalf("expected ladder ordering with Ollama first, got ladder=%v, chain[0]=%s", ladder, chain[0].ProviderName())
	}

	// Partition cooling providers
	disc.MarkCooldown(c2, time.Now().Add(1*time.Minute), domain.ErrKindRateLimited, "rate limited")
	chain, _, err = gw.ResolveChain(context.Background(), "coder", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if chain[0].ProviderName() != "Anthropic" {
		t.Fatalf("expected non-cooling Anthropic first, got %s", chain[0].ProviderName())
	}

	// Test role preference without ladder
	chainRole, _, err := gw.ResolveChain(context.Background(), "architect", ".")
	if err != nil {
		t.Fatal(err)
	}
	if chainRole[0].ProviderName() != "Anthropic" {
		t.Fatalf("expected frontier Anthropic first for architect, got %s", chainRole[0].ProviderName())
	}
}

func TestExecuteChat_SuccessAndFailover(t *testing.T) {
	cFail := &fakeClient{
		name:    "Ollama",
		model:   "broken",
		alive:   true,
		chatErr: errors.New("429 rate limit exceeded"),
	}
	cSuccess := &fakeClient{
		name:      "Anthropic",
		model:     "claude-3-5-sonnet",
		alive:     true,
		chatReply: "Quicksort implementation",
		usage:     &domain.Usage{InputTokens: 10, OutputTokens: 20},
	}

	disc := services.NewDiscoveryService(cFail, cSuccess)
	gw := NewGateway(Config{DefaultRole: "coder"}, disc)

	req := ChatCompletionRequest{
		Model: "coder",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are an assistant"},
			{Role: "user", Content: "Write quicksort"},
			{Role: "other", Content: "extra"},
		},
	}

	resp, err := gw.ExecuteChat(context.Background(), req, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Quicksort implementation" {
		t.Fatalf("unexpected content: %+v", resp.Choices)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 30 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}

	// Verify cFail was marked for cooldown
	if _, cooling := disc.Cooldown(cFail, time.Now()); !cooling {
		t.Fatal("expected cFail to be cooling down")
	}

	// Verify all fail
	cSuccess.chatErr = errors.New("service down")
	_, err = gw.ExecuteChat(context.Background(), req, ".")
	if err == nil {
		t.Fatal("expected error when all fail")
	}

	// ResolveChain error
	gwNil := NewGateway(Config{}, nil)
	if _, err := gwNil.ExecuteChat(context.Background(), req, "."); err == nil {
		t.Fatal("expected resolve chain error")
	}
}

func TestStreamChat_SuccessAndFailover(t *testing.T) {
	cFail := &fakeClient{
		name:      "Ollama",
		model:     "broken",
		alive:     true,
		streamErr: errors.New("initial connection failed"),
	}
	cSuccess := &fakeClient{
		name:         "Anthropic",
		model:        "claude-3-5-sonnet",
		alive:        true,
		streamChunks: []string{"func ", "quicksort() {}"},
	}

	disc := services.NewDiscoveryService(cFail, cSuccess)
	gw := NewGateway(Config{}, disc)

	var chunks []string
	req := ChatCompletionRequest{
		Model: "coder",
		Messages: []ChatMessage{
			{Role: "user", Content: "hello"},
		},
		Stream: true,
	}

	resp, err := gw.StreamChat(context.Background(), req, ".", func(piece string) error {
		chunks = append(chunks, piece)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 2 || chunks[0] != "func " || chunks[1] != "quicksort() {}" {
		t.Fatalf("unexpected stream chunks: %v", chunks)
	}
	if resp.Choices[0].Message.Content != "func quicksort() {}" {
		t.Fatalf("unexpected final content: %s", resp.Choices[0].Message.Content)
	}

	// Test non-streaming fallback
	cNonStream := &fakeNonStreamClient{name: "Local", model: "simple", alive: true, chatReply: "simple reply"}
	disc2 := services.NewDiscoveryService(cNonStream)
	gw2 := NewGateway(Config{}, disc2)

	var chunks2 []string
	resp2, err := gw2.StreamChat(context.Background(), req, ".", func(piece string) error {
		chunks2 = append(chunks2, piece)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks2) != 1 || chunks2[0] != "simple reply" || resp2.Choices[0].Message.Content != "simple reply" {
		t.Fatalf("unexpected non-streaming chunk: %v", chunks2)
	}

	// Test mid-stream error
	cMidStreamFail := &fakeClient{
		name:         "MidStreamFail",
		model:        "fail",
		alive:        true,
		streamChunks: []string{"start of text"},
		streamErr:    errors.New("connection reset by peer"),
	}
	disc3 := services.NewDiscoveryService(cMidStreamFail)
	gw3 := NewGateway(Config{}, disc3)

	_, err = gw3.StreamChat(context.Background(), req, ".", func(piece string) error { return nil })
	if err == nil {
		t.Fatal("expected mid-stream error to be returned")
	}

	// Test all fail
	cFail2 := &fakeClient{name: "Fail2", model: "fail2", alive: true, streamErr: errors.New("err2")}
	disc4 := services.NewDiscoveryService(cFail2)
	gw4 := NewGateway(Config{}, disc4)
	if _, err := gw4.StreamChat(context.Background(), req, ".", func(string) error { return nil }); err == nil {
		t.Fatal("expected error when all fail")
	}

	// Test ResolveChain error
	gwNil := NewGateway(Config{}, nil)
	if _, err := gwNil.StreamChat(context.Background(), req, ".", func(string) error { return nil }); err == nil {
		t.Fatal("expected resolve chain error")
	}
}

func TestListModels(t *testing.T) {
	c := &fakeClient{
		name:            "Ollama",
		model:           "codellama",
		availableModels: []string{"qwen:coder", "codellama"},
		alive:           true,
	}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{}, disc)

	items := gw.ListModels(context.Background())
	if len(items) < 7 {
		t.Fatalf("expected at least 7 items (roles + models), got %d", len(items))
	}

	// Nil discovery
	gwNil := NewGateway(Config{}, nil)
	nilItems := gwNil.ListModels(context.Background())
	if len(nilItems) != 6 {
		t.Fatalf("expected 6 virtual roles with nil discovery, got %d", len(nilItems))
	}
}

func TestGateway_CoverageGaps(t *testing.T) {
	c := &fakeClient{
		name:         "Anthropic",
		model:        "claude-3-5-sonnet",
		alive:        true,
		streamChunks: []string{"", "hello"},
	}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{DefaultRole: "coder"}, disc)

	// Empty role defaults to DefaultRole
	chain, _, err := gw.ResolveChain(context.Background(), "", ".")
	if err != nil || len(chain) == 0 {
		t.Fatalf("expected default role resolution, got %v", err)
	}

	// Stream chat with empty piece in streamChunks
	var chunks []string
	req := ChatCompletionRequest{
		Model:    "coder",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
		Stream:   true,
	}
	_, err = gw.StreamChat(context.Background(), req, ".", func(piece string) error {
		chunks = append(chunks, piece)
		return nil
	})
	if err != nil || len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("unexpected stream chunks: %v, err: %v", chunks, err)
	}

	// Single cooling provider in ExecuteChat
	disc.MarkCooldown(c, time.Now().Add(1*time.Minute), domain.ErrKindRateLimited, "cooldown")
	c.chatErr = errors.New("cannot execute")
	reqNoStream := ChatCompletionRequest{
		Model:    "coder",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	_, err = gw.ExecuteChat(context.Background(), reqNoStream, ".")
	if err == nil {
		t.Fatal("expected error with single cooling failing provider")
	}

	// Single cooling provider in StreamChat
	c.streamErr = errors.New("cannot stream")
	c.streamChunks = nil
	_, err = gw.StreamChat(context.Background(), req, ".", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected error with single cooling failing streamer")
	}
}

func TestGateway_AllCooling_And_LadderUnmatched(t *testing.T) {
	tmpDir := t.TempDir()
	nexusDir := filepath.Join(tmpDir, ".nexus")
	_ = os.MkdirAll(nexusDir, 0755)
	policyJSON := `{
		"ladder": {
			"coder": ["Ollama/*"]
		}
	}`
	_ = os.WriteFile(filepath.Join(nexusDir, "policy.json"), []byte(policyJSON), 0644)

	c1 := &fakeClient{name: "Ollama", model: "codellama", alive: true}
	c2 := &fakeClient{name: "Custom", model: "custom-model", alive: true}
	disc := services.NewDiscoveryService(c1, c2)
	gw := NewGateway(Config{}, disc)

	// One matches ladder, one does not
	chain, _, err := gw.ResolveChain(context.Background(), "coder", tmpDir)
	if err != nil || len(chain) != 2 {
		t.Fatalf("unexpected chain: %v", chain)
	}

	// Both cooling down: len(chain) > 1 and all cooling
	disc.MarkCooldown(c1, time.Now().Add(time.Minute), domain.ErrKindRateLimited, "cooling")
	disc.MarkCooldown(c2, time.Now().Add(time.Minute), domain.ErrKindRateLimited, "cooling")

	req := ChatCompletionRequest{Model: "coder", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	_, err = gw.ExecuteChat(context.Background(), req, tmpDir)
	if err == nil || !strings.Contains(err.Error(), "no candidate provider available") {
		t.Fatalf("expected 'no candidate provider available', got %v", err)
	}

	_, err = gw.StreamChat(context.Background(), req, tmpDir, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "no candidate provider available") {
		t.Fatalf("expected 'no candidate provider available' for stream, got %v", err)
	}
}
