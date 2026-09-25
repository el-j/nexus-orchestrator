package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"nexus-orchestrator/internal/core/domain"
)

// ----- Session & Agent Handlers -----

func (s *Server) toolRegisterSession(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		AgentName   string `json:"agent_name"`
		ProjectPath string `json:"project_path"`
		ExternalID  string `json:"external_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: register_session: invalid arguments: %w", err)
	}
	if p.AgentName == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "agent_name is required"}
	}
	session := domain.AISession{
		AgentName:   p.AgentName,
		Source:      domain.SessionSourceMCP,
		ProjectPath: p.ProjectPath,
		ExternalID:  p.ExternalID,
	}
	registered, err := s.orch.RegisterAISession(ctx, session)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: register_session: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"session_id": registered.ID, "status": "registered"})
	return textResult(string(b)), nil
}

func (s *Server) toolGetAISessions(ctx context.Context) (callToolResult, error) {
	sessions, err := s.orch.ListAISessions(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_ai_sessions: %w", err)
	}
	b, err := json.Marshal(sessions)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_ai_sessions: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolClaimTask(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		TaskID    string `json:"task_id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid claim_task params"}
	}
	if p.TaskID == "" || p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "task_id and session_id are required"}
	}
	task, err := s.orch.ClaimTask(ctx, p.TaskID, p.SessionID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: claim_task: %w", err)
	}
	b, _ := json.Marshal(task)
	return textResult(string(b)), nil
}

func (s *Server) toolUpdateTaskStatus(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		TaskID    string `json:"task_id"`
		SessionID string `json:"session_id"`
		Status    string `json:"status"`
		Logs      string `json:"logs"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid update_task_status params"}
	}
	if p.TaskID == "" || p.SessionID == "" || p.Status == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "task_id, session_id, and status are required"}
	}
	if p.Status != "COMPLETED" && p.Status != "FAILED" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "status must be COMPLETED or FAILED"}
	}
	task, err := s.orch.UpdateTaskStatus(ctx, p.TaskID, p.SessionID, domain.TaskStatus(p.Status), p.Logs)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task_status: %w", err)
	}
	b, _ := json.Marshal(task)
	return textResult(string(b)), nil
}

func (s *Server) toolHeartbeatTask(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		TaskID    string `json:"task_id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid heartbeat_task params"}
	}
	if p.TaskID == "" || p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "task_id and session_id are required"}
	}
	if err := s.orch.HeartbeatTask(ctx, p.TaskID, p.SessionID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: heartbeat_task: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"ok": true})
	return textResult(string(b)), nil
}

func (s *Server) toolTerminateAISession(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		SessionID string `json:"session_id"`
		Force     bool   `json:"force"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid terminate_ai_session params"}
	}
	if p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "session_id is required"}
	}
	if err := s.orch.TerminateAISession(ctx, p.SessionID, p.Force); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: terminate_ai_session: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"ok": true})
	return textResult(string(b)), nil
}

func (s *Server) toolDeregisterAISession(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid deregister_ai_session params"}
	}
	if p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "session_id is required"}
	}
	if err := s.orch.DeregisterAISession(ctx, p.SessionID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: deregister_ai_session: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"ok": true})
	return textResult(string(b)), nil
}

func (s *Server) toolHeartbeatAISession(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid heartbeat_ai_session params"}
	}
	if p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "session_id is required"}
	}
	if err := s.orch.HeartbeatAISession(ctx, p.SessionID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: heartbeat_ai_session: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"ok": true})
	return textResult(string(b)), nil
}

func (s *Server) toolPurgeDisconnectedSessions(ctx context.Context) (callToolResult, error) {
	n, err := s.orch.PurgeDisconnectedSessions(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: purge_disconnected_sessions: %w", err)
	}
	b, _ := json.Marshal(map[string]int{"purged": n})
	return textResult(string(b)), nil
}

func (s *Server) toolGetDiscoveredAgents(ctx context.Context) (callToolResult, error) {
	agents, err := s.orch.GetDiscoveredAgents(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_discovered_agents: %w", err)
	}
	b, _ := json.Marshal(agents)
	return textResult(string(b)), nil
}

func (s *Server) toolGetDiscoveredPlans(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_discovered_plans: invalid arguments: %w", err)
	}
	files, err := s.orch.GetDiscoveredPlanFiles(ctx, p.ProjectPath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_discovered_plans: %w", err)
	}

	byKind := map[string]int{}
	for _, f := range files {
		byKind[string(f.Kind)]++
	}

	activeTool := "unknown"
	if len(files) > 0 {
		if byKind[string(domain.PlanFileKindNexus)] > 0 {
			activeTool = string(domain.PlanFileKindNexus)
		} else {
			maxCount := 0
			for k, c := range byKind {
				if c > maxCount {
					maxCount = c
					activeTool = k
				}
			}
		}
	}

	response := struct {
		ProjectPath string                      `json:"projectPath"`
		FileCount   int                         `json:"fileCount"`
		ByKind      map[string]int              `json:"byKind"`
		ActiveTool  string                      `json:"activeTool"`
		Files       []domain.DiscoveredPlanFile `json:"files"`
	}{
		ProjectPath: p.ProjectPath,
		FileCount:   len(files),
		ByKind:      byKind,
		ActiveTool:  activeTool,
		Files:       files,
	}
	data, _ := json.Marshal(response)
	return textResult(string(data)), nil
}

func (s *Server) toolDelegateToNexus(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid delegate_to_nexus params"}
	}
	if p.SessionID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "session_id is required"}
	}
	instruction, err := s.orch.DelegateToNexus(ctx, p.SessionID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: delegate_to_nexus: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"instruction": instruction})
	return textResult(string(b)), nil
}

// ----- Session & Agent Schema Definitions -----

func sessionToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "register_session",
			Description: "Announce this AI agent session to nexusOrchestrator for visualisation and orchestration. Call once when starting, and periodically as a heartbeat to update last_activity.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"agent_name":   {Type: "string", Description: "Human-readable name of this AI agent (e.g. 'Claude Desktop', 'GitHub Copilot')"},
					"project_path": {Type: "string", Description: "Absolute path of the project this agent is working on (optional)"},
					"external_id":  {Type: "string", Description: "Caller-provided correlation token for deduplication (optional)"},
				},
				Required: []string{"agent_name"},
			},
		},
		{
			Name:        "get_ai_sessions",
			Description: "Return the list of all known external AI agent sessions registered with this nexusOrchestrator instance.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "claim_task",
			Description: "Claim a QUEUED task for execution by the specified AI session, transitioning it to PROCESSING.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"task_id":    {Type: "string", Description: "ID of the task to claim."},
					"session_id": {Type: "string", Description: "ID of the AI session claiming the task."},
				},
				Required: []string{"task_id", "session_id"},
			},
		},
		{
			Name:        "update_task_status",
			Description: "Report task completion or failure from the executing AI session. Only the session that claimed the task may update it.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"task_id":    {Type: "string", Description: "ID of the task to update."},
					"session_id": {Type: "string", Description: "ID of the AI session that claimed the task."},
					"status":     {Type: "string", Description: "New status: COMPLETED or FAILED."},
					"logs":       {Type: "string", Description: "Optional output or log message."},
				},
				Required: []string{"task_id", "session_id", "status"},
			},
		},
		{
			Name:        "heartbeat_task",
			Description: "Keep a PROCESSING task alive — prevents the watchdog from marking it failed. Call periodically while the task is being worked on.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"task_id":    {Type: "string", Description: "ID of the PROCESSING task to keep alive."},
					"session_id": {Type: "string", Description: "ID of the AI session that claimed the task."},
				},
				Required: []string{"task_id", "session_id"},
			},
		},
		{
			Name:        "terminate_ai_session",
			Description: "Terminate an external AI agent session. Sends SIGTERM by default, or SIGKILL when force=true.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"session_id": {Type: "string", Description: "ID of the AI session to terminate."},
					"force":      {Type: "boolean", Description: "Send SIGKILL instead of SIGTERM (default false)."},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "deregister_ai_session",
			Description: "Soft-disconnect an AI agent session, marking it as disconnected without killing the process.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"session_id": {Type: "string", Description: "ID of the AI session to deregister."},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "heartbeat_ai_session",
			Description: "Refresh the last-activity timestamp of an AI session to keep it alive.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"session_id": {Type: "string", Description: "ID of the AI session to heartbeat."},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "purge_disconnected_sessions",
			Description: "Delete all AI sessions with status 'disconnected' that have been inactive for more than 2 hours. Returns the number purged.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "get_discovered_agents",
			Description: "Return AI agent tools detected on the local system (Claude CLI, VS Code Copilot, etc.).",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "delegate_to_nexus",
			Description: "Delegate an AI agent session to the nexus task queue and return the workflow instruction string.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"session_id": {Type: "string", Description: "ID of the AI session to delegate."},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "get_discovered_plans",
			Description: "Scan for plan/task/orchestration files in a project directory. Returns nexus orchestrator.json, markdown task files, Cursor rules, MCP configs, and more. The response includes projectPath, fileCount, byKind (counts per tool kind), activeTool (dominant AI tool detected), and files (full list).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root to scan"},
				},
			},
		},
	}
}
