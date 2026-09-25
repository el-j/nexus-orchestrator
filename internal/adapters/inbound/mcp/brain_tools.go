package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"nexus-orchestrator/internal/core/domain"
)

func (s *Server) toolGetProjectContext(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		MaxTokens   int    `json:"maxTokens,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_project_context: invalid args: %w", err)
	}

	maxT := p.MaxTokens
	if maxT <= 0 {
		maxT = 400
	}

	q := domain.ContextQuery{
		ProjectPath: p.ProjectPath,
		MaxTokens:   maxT,
	}

	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}

	resp, err := s.brain.GetContext(ctx, q)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_project_context: %w", err)
	}

	b, _ := json.Marshal(resp)
	return textResult(string(b)), nil
}

func (s *Server) toolGetFocusedContext(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		Question    string `json:"question"`
		MaxTokens   int    `json:"maxTokens,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_focused_context: invalid args: %w", err)
	}

	maxT := p.MaxTokens
	if maxT <= 0 {
		maxT = 400
	}

	q := domain.ContextQuery{
		ProjectPath: p.ProjectPath,
		Question:    p.Question,
		MaxTokens:   maxT,
	}

	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}

	resp, err := s.brain.GetFocusedContext(ctx, q)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_focused_context: %w", err)
	}

	b, _ := json.Marshal(resp)
	return textResult(string(b)), nil
}

func (s *Server) toolSearchKnowledge(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		Query       string `json:"query"`
		Limit       int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: search_knowledge: invalid args: %w", err)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 5
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	results, err := s.brain.SearchKnowledge(ctx, p.ProjectPath, p.Query, limit)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: search_knowledge: %w", err)
	}
	b, _ := json.Marshal(results)
	return textResult(string(b)), nil
}

func (s *Server) toolGetBrainStatus(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_brain_status: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	status, err := s.brain.GetStatus(ctx, p.ProjectPath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_brain_status: %w", err)
	}
	b, _ := json.Marshal(status)
	return textResult(string(b)), nil
}

func (s *Server) toolIngestKnowledge(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		FilePath    string `json:"filePath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: ingest_knowledge: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	count, err := s.brain.IngestFromFile(ctx, p.ProjectPath, p.FilePath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: ingest_knowledge: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"ingestedSections": count})
	return textResult(string(b)), nil
}

func (s *Server) toolInitProject(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath  string `json:"projectPath"`
		ClaudeMDPath string `json:"claudeMDPath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: init_project: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	status, err := s.brain.InitProject(ctx, p.ProjectPath, p.ClaudeMDPath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: init_project: %w", err)
	}
	b, _ := json.Marshal(status)
	return textResult(string(b)), nil
}

func (s *Server) toolListKnowledge(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		Kind        string `json:"kind"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: list_knowledge: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	entries, err := s.brain.ListKnowledge(ctx, p.ProjectPath, p.Kind)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: list_knowledge: %w", err)
	}
	b, _ := json.Marshal(entries)
	return textResult(string(b)), nil
}

func (s *Server) toolDeleteKnowledge(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: delete_knowledge: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	if err := s.brain.DeleteKnowledge(ctx, p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: delete_knowledge: %w", err)
	}
	return textResult(`{"deleted":true}`), nil
}

func (s *Server) toolGetFileMap(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		FocusArea   string `json:"focusArea"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_file_map: invalid args: %w", err)
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	paths, err := s.brain.GetFileMap(ctx, p.ProjectPath, p.FocusArea)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_file_map: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"filePaths": paths})
	return textResult(string(b)), nil
}

func (s *Server) toolGetOnboardingContext(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
		MaxTokens   int    `json:"maxTokens,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_onboarding_context: invalid args: %w", err)
	}
	if p.ProjectPath == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "projectPath is required"}
	}
	if s.brain == nil {
		return callToolResult{}, fmt.Errorf("brain service not configured")
	}
	summary, err := s.brain.GetOnboardingContext(ctx, p.ProjectPath, p.MaxTokens)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_onboarding_context: %w", err)
	}
	return textResult(summary), nil
}

// ----- Brain Schema Definitions -----

func brainToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "get_onboarding_context",
			Description: "Get a token-budgeted (< 800 tokens) Tier 0/1 onboarding summary of the project for agent bootstrap (stack, invariants, test commands, active plan/tasks).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"maxTokens":   {Type: "number", Description: "Maximum tokens to return (default: 800)"},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "get_project_context",
			Description: "Get the macro context for a project (Architectures, Conventions, File Maps) bounded by a token budget.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"maxTokens":   {Type: "number", Description: "Maximum tokens to return (default: 400)"},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "get_focused_context",
			Description: "Get task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"question":    {Type: "string", Description: "Semantic search query to match against knowledge"},
					"maxTokens":   {Type: "number", Description: "Maximum tokens to return (default: 400)"},
				},
				Required: []string{"projectPath", "question"},
			},
		},
		{
			Name:        "search_knowledge",
			Description: "Perform full-text search across the project's knowledge base.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"query":       {Type: "string", Description: "The FTS query"},
					"limit":       {Type: "number", Description: "Max results to return"},
				},
				Required: []string{"projectPath", "query"},
			},
		},
		{
			Name:        "get_brain_status",
			Description: "Check the knowledge repository status for a project.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "ingest_knowledge",
			Description: "Parse and ingest a markdown file (often CLAUDE.md) into the project knowledge repository.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root"},
					"filePath":    {Type: "string", Description: "Path to the markdown file to ingest"},
				},
				Required: []string{"projectPath", "filePath"},
			},
		},
		{
			Name:        "init_project",
			Description: "Auto-ingest CLAUDE.md and initialize a project's knowledge base",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":  {Type: "string", Description: "Absolute path to the project"},
					"claudeMDPath": {Type: "string", Description: "Path to CLAUDE.md (optional, auto-detected if empty)"},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "list_knowledge",
			Description: "List all knowledge entries for a project, optionally filtered by kind",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project"},
					"kind":        {Type: "string", Description: "Knowledge kind filter (optional)"},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "delete_knowledge",
			Description: "Delete a knowledge entry by ID",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Knowledge entry ID"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "get_file_map",
			Description: "Get the file path map for a project from the knowledge base",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project"},
					"focusArea":   {Type: "string", Description: "Optional focus area filter"},
				},
				Required: []string{"projectPath"},
			},
		},
	}
}
