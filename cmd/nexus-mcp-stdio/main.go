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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Getenv))
}

// run resolves the endpoint and token via getenv, proxies stdin to the daemon
// and returns the process exit code.
func run(in io.Reader, out, errOut io.Writer, getenv func(string) string) int {
	mcpURL := getenv("NEXUS_MCP_URL")
	if mcpURL == "" {
		mcpURL = "http://127.0.0.1:63988/mcp"
	}
	mcpToken := strings.TrimSpace(getenv("NEXUS_MCP_TOKEN"))

	if err := proxy(in, out, errOut, mcpURL, mcpToken); err != nil {
		fmt.Fprintf(errOut, "nexus-mcp-stdio: %v\n", err)
		return 1
	}
	return 0
}

// proxy reads JSON-RPC messages from in, forwards them via HTTP POST to mcpURL,
// and writes responses to out. Diagnostic logs go to errOut.
func proxy(in io.Reader, out, errOut io.Writer, mcpURL, mcpToken string) error {
	client := &http.Client{Timeout: 120 * time.Second}

	fmt.Fprintf(errOut, "nexus-mcp-stdio: forwarding to %s\n", mcpURL)

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		req, err := http.NewRequest(http.MethodPost, mcpURL, bytes.NewReader(line))
		if err != nil {
			fmt.Fprintf(errOut, "nexus-mcp-stdio: new request error: %v\n", err)
			writeErrTo(out, "proxy request error")
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if mcpToken != "" {
			req.Header.Set("Authorization", "Bearer "+mcpToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(errOut, "nexus-mcp-stdio: POST error: %v\n", err)
			writeErrTo(out, err.Error())
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(errOut, "nexus-mcp-stdio: read response: %v\n", err)
			writeErrTo(out, "proxy read error")
			continue
		}

		if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
			continue
		}

		if _, err := fmt.Fprint(out, string(body)); err != nil {
			return fmt.Errorf("stdout write error: %w", err)
		}
		if body[len(body)-1] != '\n' {
			if _, err := fmt.Fprint(out, "\n"); err != nil {
				return fmt.Errorf("stdout newline error: %w", err)
			}
		}
	}

	return scanner.Err()
}

// writeErrTo emits a JSON-RPC internal-error response. The message is marshalled
// (not interpolated) because Go's HTTP errors contain double quotes, which would
// otherwise produce invalid JSON on the wire.
func writeErrTo(out io.Writer, msg string) {
	resp := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{JSONRPC: "2.0"}
	resp.Error.Code = -32603
	resp.Error.Message = msg
	b, _ := json.Marshal(resp) // cannot fail: only strings, ints and nil
	fmt.Fprintf(out, "%s\n", b)
}
