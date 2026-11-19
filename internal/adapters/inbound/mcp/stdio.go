// Package mcp — stdio transport implementation for MCP clients.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// stdioResponseWriter writes HTTP response bytes to an io.Writer for stdio MCP.
type stdioResponseWriter struct {
	out    io.Writer
	header http.Header
}

// Header returns the response HTTP header map.
func (w *stdioResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

// Write writes response payload bytes to the underlying writer.
func (w *stdioResponseWriter) Write(b []byte) (int, error) {
	return w.out.Write(b)
}

// WriteHeader records the HTTP status code (no-op in stdio transport).
func (w *stdioResponseWriter) WriteHeader(int) {}

// ServeStdio reads JSON-RPC requests from in and writes JSON-RPC responses to out,
// processing MCP protocol messages line by line until EOF or context cancellation.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	const maxScanToken = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanToken)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeError(&stdioResponseWriter{out: out}, nil, codeParseError, "parse error")
			_, _ = fmt.Fprintln(out)
			continue
		}
		if req.JSONRPC != "2.0" {
			writeError(&stdioResponseWriter{out: out}, req.ID, codeInvalidRequest, `invalid request: jsonrpc must be "2.0"`)
			_, _ = fmt.Fprintln(out)
			continue
		}

		httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/mcp", bytes.NewReader(line))
		var respBuf bytes.Buffer
		s.processRPC(&stdioResponseWriter{out: &respBuf}, httpReq, req)
		respBytes := append(bytes.TrimSpace(respBuf.Bytes()), '\n')
		if len(respBytes) > 1 {
			if _, err := out.Write(respBytes); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
