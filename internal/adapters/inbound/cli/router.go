package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/gateway"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"

	"github.com/spf13/cobra"
)

// netListen aliases net.Listen for testing listener failure scenarios.
var netListen = net.Listen

// NewRouterStandaloneCmd returns the root cobra command for the standalone nexus-router CLI.
func NewRouterStandaloneCmd(gw *gateway.Gateway, discovery *services.DiscoveryService, orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nexus-router",
		Short: "nexus-router — Standalone LLM Gateway & OpenAI-compatible proxy",
		Long:  "Run a high-performance OpenAI-compatible reverse proxy or dispatch prompts using Nexus's smart model ladder.",
	}

	cmd.AddCommand(newRouterServeCmd(gw, discovery))
	cmd.AddCommand(newRouterPromptCmd(gw))
	cmd.AddCommand(newRouterStatusCmd(discovery, orch))

	if orch != nil {
		cmd.AddCommand(NewRoutingCmd(orch))
		cmd.AddCommand(NewProvidersCmd(orch))
	} else {
		cmd.AddCommand(newRouterLocalRoutingCmd(gw))
		cmd.AddCommand(newRouterLocalProvidersCmd(discovery))
	}

	return cmd
}

// newRouterServeCmd returns the `serve` command running the OpenAI-compatible HTTP server.
func newRouterServeCmd(gw *gateway.Gateway, discovery *services.DiscoveryService) *cobra.Command {
	var (
		port    int
		host    string
		apiKey  string
		project string
		role    string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the OpenAI-compatible reverse proxy HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := gateway.Config{
				APIKey:      apiKey,
				DefaultRole: role,
				ProjectPath: project,
			}
			engine := gw
			if engine == nil {
				engine = gateway.NewGateway(cfg, discovery)
			}

			server := gateway.NewServer(cfg, engine)
			addr := fmt.Sprintf("%s:%d", host, port)
			ln, err := netListen("tcp", addr)
			if err != nil {
				return fmt.Errorf("router: listen on %s: %w", addr, err)
			}

			httpServer := &http.Server{
				Handler:           server.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "nexus-router listening on http://%s/v1 (OpenAI-compatible)\n", addr)
			fmt.Fprintln(out, "Routes available: /v1/chat/completions, /v1/models, /v1/health")

			errCh := make(chan error, 1)
			go func() {
				if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errCh <- err
				}
			}()

			sigCtx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			select {
			case <-sigCtx.Done():
				fmt.Fprintln(out, "nexus-router shutting down...")
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return httpServer.Shutdown(shutCtx)
			case err := <-errCh:
				return fmt.Errorf("router: server error: %w", err)
			}
		},
	}

	cmd.Flags().IntVar(&port, "port", 63991, "Port to listen on")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface to bind to")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Optional bearer API key for authentication")
	cmd.Flags().StringVar(&project, "project", ".", "Project directory for policy and ladder lookup")
	cmd.Flags().StringVar(&role, "role", "coder", "Default routing role")

	return cmd
}

// newRouterPromptCmd returns the `prompt` command for sending prompts directly.
func newRouterPromptCmd(gw *gateway.Gateway) *cobra.Command {
	var (
		role    string
		model   string
		stream  bool
		project string
	)

	cmd := &cobra.Command{
		Use:   "prompt [text]",
		Short: "Send a prompt directly through the optimal provider ladder",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("prompt text required")
			}
			promptText := strings.Join(args, " ")
			reqModel := model
			if reqModel == "" {
				reqModel = role
			}

			req := gateway.ChatCompletionRequest{
				Model: reqModel,
				Messages: []gateway.ChatMessage{
					{
						Role:    "user",
						Content: promptText,
					},
				},
				Stream: stream,
			}

			out := cmd.OutOrStdout()
			if stream {
				_, err := gw.StreamChat(cmd.Context(), req, project, func(piece string) error {
					_, _ = fmt.Fprint(out, piece)
					return nil
				})
				if err != nil {
					return fmt.Errorf("prompt stream: %w", err)
				}
				fmt.Fprintln(out)
				return nil
			}

			resp, err := gw.ExecuteChat(cmd.Context(), req, project)
			if err != nil {
				return fmt.Errorf("prompt execute: %w", err)
			}
			if len(resp.Choices) > 0 {
				fmt.Fprintln(out, resp.Choices[0].Message.Content)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&role, "role", "coder", "Task role for ladder routing (coder, architect, reviewer, etc.)")
	cmd.Flags().StringVar(&model, "model", "", "Explicit model ID or provider name")
	cmd.Flags().BoolVar(&stream, "stream", false, "Stream output chunk by chunk")
	cmd.Flags().StringVar(&project, "project", ".", "Project directory for policy lookup")

	return cmd
}

// newRouterStatusCmd returns the `status` command listing providers and cooldowns.
func newRouterStatusCmd(discovery *services.DiscoveryService, orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of LLM providers and active cooldowns",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			var providers []ports.ProviderInfo
			if orch != nil {
				var err error
				providers, err = orch.GetProviders()
				if err != nil {
					return fmt.Errorf("status: %w", err)
				}
			} else if discovery != nil {
				providers = discovery.ListProviders()
			}

			if len(providers) == 0 {
				fmt.Fprintln(out, "No LLM providers registered or reachable.")
				return nil
			}

			fmt.Fprintf(out, "%-20s  %-26s  %-8s  %s\n", "Provider", "Model", "Status", "Endpoint")
			fmt.Fprintln(out, strings.Repeat("-", 80))
			for _, p := range providers {
				st := "offline"
				if p.Active {
					st = "active"
				}
				fmt.Fprintf(out, "%-20s  %-26s  %-8s  %s\n", p.Name, p.ActiveModel, st, p.BaseURL)
			}
			return nil
		},
	}
}

// newRouterLocalRoutingCmd previews the provider chain without requiring a daemon.
func newRouterLocalRoutingCmd(gw *gateway.Gateway) *cobra.Command {
	var (
		role    string
		project string
	)

	cmd := &cobra.Command{
		Use:   "routing",
		Short: "Preview provider routing chain and ladder",
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ladder, err := gw.ResolveChain(cmd.Context(), role, project)
			if err != nil {
				return fmt.Errorf("routing preview: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Role: %s\n", role)
			if len(ladder) == 0 {
				fmt.Fprintln(out, "Ladder: none (default role order)")
			} else {
				fmt.Fprintf(out, "Ladder: %v\n", ladder)
			}
			fmt.Fprintf(out, "%3s  %-22s  %-28s\n", "#", "Provider", "Model")
			for i, c := range chain {
				fmt.Fprintf(out, "%3d  %-22s  %-28s\n", i+1, c.ProviderName(), c.ActiveModel())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&role, "role", "coder", "Role to preview")
	cmd.Flags().StringVar(&project, "project", ".", "Project directory")
	return cmd
}

// newRouterLocalProvidersCmd prints provider JSON directly from DiscoveryService.
func newRouterLocalProvidersCmd(discovery *services.DiscoveryService) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List registered providers in JSON format",
		RunE: func(cmd *cobra.Command, args []string) error {
			if discovery == nil {
				return errors.New("discovery service not available")
			}
			providers := discovery.ListProviders()
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(providers)
		},
	}
}
