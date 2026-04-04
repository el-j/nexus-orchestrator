// nexus-mcp-stdio is a thin proxy that bridges MCP's stdio transport to the
// running nexusOrchestrator daemon's HTTP-based MCP endpoint.
//
// Usage:
//
//	nexus-mcp-stdio                                        # default http://127.0.0.1:63988/mcp
//	NEXUS_MCP_URL=http://host:port/mcp nexus-mcp-stdio     # custom endpoint
//
// It reads newline-delimited JSON-RPC messages from stdin, forwards each one
// to the daemon via HTTP POST, and writes the response to stdout.
// Diagnostic messages go to stderr only.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	mcpURL := os.Getenv("NEXUS_MCP_URL")
	if mcpURL == "" {
		mcpURL = "http://127.0.0.1:63988/mcp"
	}
	mcpToken := strings.TrimSpace(os.Getenv("NEXUS_MCP_TOKEN"))

	client := &http.Client{Timeout: 120 * time.Second}

	fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: forwarding to %s\n", mcpURL)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		req, err := http.NewRequest(http.MethodPost, mcpURL, bytes.NewReader(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: new request error: %v\n", err)
			writeErr("proxy request error")
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if mcpToken != "" {
			req.Header.Set("Authorization", "Bearer "+mcpToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: POST error: %v\n", err)
			writeErr(err.Error())
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: read response: %v\n", err)
			writeErr("proxy read error")
			continue
		}

		if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
			continue
		}

		if _, err := fmt.Fprint(os.Stdout, string(body)); err != nil {
			fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: stdout write error: %v\n", err)
			os.Exit(1)
		}
		if body[len(body)-1] != '\n' {
			if _, err := fmt.Fprint(os.Stdout, "\n"); err != nil {
				fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: stdout newline error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "nexus-mcp-stdio: stdin read error: %v\n", err)
		os.Exit(1)
	}
}

func writeErr(msg string) {
	fmt.Fprintf(os.Stdout, "{\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32603,\"message\":\"%s\"}}\n", msg)
}
