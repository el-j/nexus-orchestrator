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
