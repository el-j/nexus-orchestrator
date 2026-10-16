package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"nexus-orchestrator/internal/core/ports"
)

// routingToolDefs lists the routing preview tool.
func routingToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "get_routing",
			Description: "Show how Nexus would order the providers for a task of a role in a project right now: the project's model ladder (cheapest first), which providers are cooling down after a limit, and the plan's own quota readings. Changes nothing.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"role":        {Type: "string", Description: "The task role, for example coder, tester, architect"},
				},
				Required: []string{"projectPath"},
			},
		},
	}
}

func (s *Server) toolGetRouting(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		Role        string `json:"role"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(p.ProjectPath) == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "projectPath is required"}
	}
	rp, ok := s.orch.(ports.RoutingPreviewer)
	if !ok {
		return callToolResult{}, fmt.Errorf("mcp: get_routing: routing preview is not available")
	}
	preview, err := rp.PreviewRouting(ctx, p.ProjectPath, strings.TrimSpace(p.Role))
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_routing: %w", err)
	}
	b, err := json.Marshal(preview)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_routing: marshal: %w", err)
	}
	return textResult(string(b)), nil
}
