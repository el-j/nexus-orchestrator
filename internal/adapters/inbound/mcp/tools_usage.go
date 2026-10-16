package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// usageToolDefs lists the usage ledger tools.
func usageToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "get_usage",
			Description: "Tokens and cost the models consumed, added up per provider, model and role, from the usage ledger: input, output and cached tokens, and the list-price cost when the provider reports it. Use it to see which role or model burns the quota.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"since": {Type: "string", Description: "Start of the window: a duration back from now (90m, 24h, 7d), an RFC 3339 time, or all. Default 24h."},
				},
			},
		},
	}
}

func (s *Server) toolGetUsage(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		Since string `json:"since"`
	}
	if err := decodeOptionalArgs(args, &p); err != nil {
		return callToolResult{}, err
	}
	q, ok := s.orch.(ports.UsageQuerier)
	if !ok {
		return callToolResult{}, fmt.Errorf("mcp: get_usage: the usage ledger is not available")
	}
	since, err := domain.ParseSince(p.Since, time.Now())
	if err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: err.Error()}
	}
	sums, err := q.GetUsage(ctx, since)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_usage: %w", err)
	}
	b, err := json.Marshal(map[string]any{"since": since, "totals": domain.UsageTotals(sums), "usage": sums})
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_usage: marshal: %w", err)
	}
	return textResult(string(b)), nil
}
