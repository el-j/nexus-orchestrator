package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// baseURLGetter allows the logs command to construct endpoint URLs at runtime.
type baseURLGetter interface {
	GetBaseURL() string
}

func newLogsCmd(orch ports.Orchestrator) *cobra.Command {
	var follow bool
	var count int
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Print recent daemon log entries (use --follow for live stream)",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := defaultDaemonURL
			if g, ok := orch.(baseURLGetter); ok {
				baseURL = g.GetBaseURL()
			}
			if follow {
				return streamSSELogs(cmd.Context(), baseURL)
			}
			return fetchLogs(cmd.Context(), baseURL, count)
		},
	}
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream live log events via SSE")
	cmd.Flags().IntVar(&count, "n", 50, "Number of log entries to show")
	return cmd
}

func fetchLogs(ctx context.Context, baseURL string, n int) error {
	url := fmt.Sprintf("%s/api/logs?n=%d", strings.TrimRight(baseURL, "/"), n)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("logs: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("logs: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logs: unexpected status %d", resp.StatusCode)
	}
	var entries []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("logs: decode: %w", err)
	}
	for _, e := range entries {
		fmt.Println(string(e))
	}
	return nil
}

func streamSSELogs(ctx context.Context, baseURL string) error {
	url := strings.TrimRight(baseURL, "/") + "/api/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("logs follow: build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("logs follow: connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logs follow: unexpected status %d", resp.StatusCode)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				fmt.Println(strings.TrimPrefix(line, "data:"))
			}
		}
	}()

	select {
	case <-sigCh:
		return nil
	case <-doneCh:
		return nil
	}
}
