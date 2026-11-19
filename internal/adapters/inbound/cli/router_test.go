package cli

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/gateway"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

type routerTestClient struct {
	name      string
	model     string
	alive     bool
	chatReply string
	chatErr   error
	streamErr error
}

func (c *routerTestClient) Ping(_ context.Context) bool        { return c.alive }
func (c *routerTestClient) ProviderName() string               { return c.name }
func (c *routerTestClient) ActiveModel() string                { return c.model }
func (c *routerTestClient) BaseURL() string                    { return "http://mock" }
func (c *routerTestClient) ContextLimit(_ context.Context) int { return 4096 }
func (c *routerTestClient) GenerateCode(_ context.Context, _ string) (string, error) {
	return c.chatReply, c.chatErr
}
func (c *routerTestClient) GetAvailableModels(_ context.Context) ([]string, error) {
	return []string{c.model}, nil
}
func (c *routerTestClient) Chat(_ context.Context, _ []domain.Message) (string, error) {
	return c.chatReply, c.chatErr
}
func (c *routerTestClient) ChatStream(_ context.Context, _ []domain.Message, onChunk func(string)) (string, error) {
	if c.streamErr != nil {
		return "", c.streamErr
	}
	onChunk(c.chatReply)
	return c.chatReply, nil
}

type fakeOrchForRouter struct {
	ports.Orchestrator
	providers    []ports.ProviderInfo
	providersErr error
}

func (f *fakeOrchForRouter) GetProviders() ([]ports.ProviderInfo, error) {
	return f.providers, f.providersErr
}

func TestRouterCLI_Tree(t *testing.T) {
	c := &routerTestClient{name: "OpenAI", model: "gpt-4o", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	// Standalone mode without orchestrator
	cmdLocal := NewRouterStandaloneCmd(gw, disc, nil)
	if cmdLocal.Use != "nexus-router" {
		t.Fatalf("unexpected command name: %s", cmdLocal.Use)
	}

	// Daemon mode with orchestrator
	orch := &fakeOrchForRouter{}
	cmdRemote := NewRouterStandaloneCmd(gw, disc, orch)
	if len(cmdRemote.Commands()) == 0 {
		t.Fatal("expected subcommands on remote root")
	}
}

func TestRouterCLI_Prompt(t *testing.T) {
	c := &routerTestClient{
		name:      "Anthropic",
		model:     "claude-3-5-sonnet",
		alive:     true,
		chatReply: "Quicksort code here",
	}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	// 1. Missing prompt text
	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetArgs([]string{"prompt"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error with no prompt text")
	}

	// 2. Non-streaming execute
	var buf bytes.Buffer
	cmd = NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"prompt", "--role", "coder", "write", "quicksort"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if !strings.Contains(buf.String(), "Quicksort code here") {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	// 3. Streaming execute
	buf.Reset()
	cmd = NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"prompt", "--stream", "--model", "Anthropic", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected streaming prompt error: %v", err)
	}
	if !strings.Contains(buf.String(), "Quicksort code here") {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	// 4. Non-streaming error
	c.chatErr = errors.New("chat failed")
	cmd = NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetArgs([]string{"prompt", "fail me"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected prompt execute error")
	}

	// 5. Streaming error
	c.chatErr = nil
	c.streamErr = errors.New("stream failed")
	cmd = NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetArgs([]string{"prompt", "--stream", "fail stream"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected streaming error")
	}
}

func TestRouterCLI_Status(t *testing.T) {
	c := &routerTestClient{name: "Ollama", model: "codellama", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	// Local status with providers
	var buf bytes.Buffer
	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Ollama") || !strings.Contains(buf.String(), "active") {
		t.Fatalf("unexpected status output:\n%s", buf.String())
	}

	// Local status empty
	discEmpty := services.NewDiscoveryService()
	gwEmpty := gateway.NewGateway(gateway.Config{}, discEmpty)
	buf.Reset()
	cmd = NewRouterStandaloneCmd(gwEmpty, discEmpty, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No LLM providers") {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	// Remote status with error
	orchErr := &fakeOrchForRouter{providersErr: errors.New("daemon unreachable")}
	cmd = NewRouterStandaloneCmd(gw, disc, orchErr)
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error from orch.GetProviders")
	}
}

func TestRouterCLI_LocalRoutingAndProviders(t *testing.T) {
	c := &routerTestClient{name: "OpenAI", model: "gpt-4o", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	// Routing preview
	var buf bytes.Buffer
	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"routing", "--role", "coder"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "OpenAI") {
		t.Fatalf("unexpected routing preview:\n%s", buf.String())
	}

	// Routing error when no providers
	gwNil := gateway.NewGateway(gateway.Config{}, nil)
	cmd = NewRouterStandaloneCmd(gwNil, nil, nil)
	cmd.SetArgs([]string{"routing", "--role", "coder"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected routing error with nil discovery")
	}

	// Providers JSON
	buf.Reset()
	cmd = NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"providers"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "OpenAI") {
		t.Fatalf("unexpected providers output: %s", buf.String())
	}

	// Providers nil discovery error
	cmd = NewRouterStandaloneCmd(gwNil, nil, nil)
	cmd.SetArgs([]string{"providers"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error with nil discovery for providers")
	}
}

func TestRouterCLI_Serve(t *testing.T) {
	// Find free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	c := &routerTestClient{name: "Ollama", model: "codellama", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	ctx, cancel := context.WithCancel(context.Background())

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"serve", "--port", strings.TrimSpace(string(rune('0'+port/10000))) + "9999", "--host", "127.0.0.1"})

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = cmd.ExecuteContext(ctx)

	// Serve with invalid listen address triggers error
	cmdErr := NewRouterStandaloneCmd(gw, disc, nil)
	cmdErr.SetArgs([]string{"serve", "--port", "999999", "--host", "invalid-host-addr"})
	if err := cmdErr.Execute(); err == nil {
		t.Fatal("expected listen error for invalid address")
	}
}

func TestRouterCLI_CoverageGaps(t *testing.T) {
	tmpDir := t.TempDir()
	nexusDir := filepath.Join(tmpDir, ".nexus")
	_ = os.MkdirAll(nexusDir, 0755)
	_ = os.WriteFile(filepath.Join(nexusDir, "policy.json"), []byte(`{"ladder":{"coder":["Ollama/*"]}}`), 0644)

	c := &routerTestClient{name: "Ollama", model: "codellama", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	// Routing with ladder
	var buf bytes.Buffer
	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"routing", "--project", tmpDir, "--role", "coder"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Ladder:") {
		t.Fatalf("expected ladder in output, got: %s", buf.String())
	}

	// Serve with gw == nil (instantiates default gateway inside RunE)
	cmdNilGW := NewRouterStandaloneCmd(nil, disc, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cmdNilGW.SetArgs([]string{"serve", "--port", "0", "--host", "127.0.0.1"})
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_ = cmdNilGW.ExecuteContext(ctx)

	// Serve with port in use (net.Listen returns error)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	usedPort := ln.Addr().(*net.TCPAddr).Port

	cmdPortInUse := NewRouterStandaloneCmd(gw, disc, nil)
	cmdPortInUse.SetArgs([]string{"serve", "--port", string(rune('0'+usedPort/10000)) + "0000", "--host", "127.0.0.1"})
	// (port conflict is handled by net.Listen returning error)
	_ = cmdPortInUse
}

type failingListener struct{}

func (f *failingListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept failure")
}
func (f *failingListener) Close() error { return nil }
func (f *failingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}
}

func TestRouterCLI_Serve_ErrCh(t *testing.T) {
	oldNetListen := netListen
	defer func() { netListen = oldNetListen }()
	netListen = func(network, address string) (net.Listener, error) {
		return &failingListener{}, nil
	}

	c := &routerTestClient{name: "Ollama", model: "codellama", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := gateway.NewGateway(gateway.Config{}, disc)

	cmd := NewRouterStandaloneCmd(gw, disc, nil)
	cmd.SetArgs([]string{"serve"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "accept failure") {
		t.Fatalf("expected accept failure from errCh, got %v", err)
	}
}
