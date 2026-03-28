// Package mcp — tool handlers and schema definitions.
// This file contains the handleToolCall dispatcher, every tool* method on
// Server, the textResult helper, and the toolList/toolDef schema catalogue.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"nexus-orchestrator/internal/core/domain"
)

// ----- Tool dispatch -----

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	var p callToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeError(w, req.ID, codeInvalidParams, "invalid params")
		return
	}

	var (
		result callToolResult
		err    error
	)

	switch p.Name {
	case "submit_task":
		result, err = s.toolSubmitTask(p.Arguments)
	case "get_task":
		result, err = s.toolGetTask(p.Arguments)
	case "get_queue":
		result, err = s.toolGetQueue()
	case "get_all_tasks":
		result, err = s.toolGetAllTasks()
	case "cancel_task":
		result, err = s.toolCancelTask(p.Arguments)
	case "get_providers":
		result, err = s.toolGetProviders()
	case "health":
		result, err = s.toolHealth()
	case "create_draft":
		result, err = s.toolCreateDraft(p.Arguments)
	case "get_backlog":
		result, err = s.toolGetBacklog(p.Arguments)
	case "promote_task":
		result, err = s.toolPromoteTask(p.Arguments)
	case "update_task":
		result, err = s.toolUpdateTask(p.Arguments)
	case "discover_providers":
		result, err = s.toolDiscoverProviders(r.Context())
	case "promote_provider":
		result, err = s.toolPromoteProvider(r.Context(), p.Arguments)
	case "register_session":
		result, err = s.toolRegisterSession(r.Context(), p.Arguments)
	case "get_ai_sessions":
		result, err = s.toolGetAISessions(r.Context())
	case "claim_task":
		result, err = s.toolClaimTask(r.Context(), p.Arguments)
	case "update_task_status":
		result, err = s.toolUpdateTaskStatus(r.Context(), p.Arguments)
	case "heartbeat_task":
		result, err = s.toolHeartbeatTask(r.Context(), p.Arguments)
	case "terminate_ai_session":
		result, err = s.toolTerminateAISession(r.Context(), p.Arguments)
	case "list_provider_configs":
		result, err = s.toolListProviderConfigs(r.Context())
	case "add_provider_config":
		result, err = s.toolAddProviderConfig(r.Context(), p.Arguments)
	case "update_provider_config":
		result, err = s.toolUpdateProviderConfig(r.Context(), p.Arguments)
	case "remove_provider_config":
		result, err = s.toolRemoveProviderConfig(r.Context(), p.Arguments)
	case "deregister_ai_session":
		result, err = s.toolDeregisterAISession(r.Context(), p.Arguments)
	case "heartbeat_ai_session":
		result, err = s.toolHeartbeatAISession(r.Context(), p.Arguments)
	case "purge_disconnected_sessions":
		result, err = s.toolPurgeDisconnectedSessions(r.Context())
	case "get_discovered_agents":
		result, err = s.toolGetDiscoveredAgents(r.Context())
	case "get_discovered_plans":
		result, err = s.toolGetDiscoveredPlans(r.Context(), p.Arguments)
	case "delegate_to_nexus":
		result, err = s.toolDelegateToNexus(r.Context(), p.Arguments)
	case "howto":
		result, err = s.toolHowto()
	case "howto_brief":
		result, err = s.toolHowtoBrief()
	default:
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("unknown tool: %s", p.Name))
		return
	}

	if err != nil {
		var me *mcpError
		if errors.As(err, &me) {
			writeError(w, req.ID, me.code, me.msg)
			return
		}
		writeError(w, req.ID, codeInternalError, err.Error())
		return
	}

	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	_ = json.NewEncoder(w).Encode(resp)
}

// ----- Individual tool handlers -----

func (s *Server) toolSubmitTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath  string   `json:"projectPath"`
		TargetFile   string   `json:"targetFile"`
		Instruction  string   `json:"instruction"`
		ContextFiles []string `json:"contextFiles"`
		Command      string   `json:"command"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: invalid arguments: %w", err)
	}
	t := domain.Task{
		ProjectPath:  p.ProjectPath,
		TargetFile:   p.TargetFile,
		Instruction:  p.Instruction,
		ContextFiles: p.ContextFiles,
		Command:      domain.CommandType(p.Command),
	}
	id, err := s.orch.SubmitTask(t)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"id": id})
	return textResult(string(b)), nil
}

func (s *Server) toolGetTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: invalid arguments: %w", err)
	}
	task, err := s.orch.GetTask(p.ID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: %w", err)
	}
	b, err := json.Marshal(task)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolGetQueue() (callToolResult, error) {
	tasks, err := s.orch.GetQueue()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolGetAllTasks() (callToolResult, error) {
	tasks, err := s.orch.GetAllTasks()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_all_tasks: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_all_tasks: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolCancelTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: invalid arguments: %w", err)
	}
	if err := s.orch.CancelTask(p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"cancelled": true})
	return textResult(string(b)), nil
}

func (s *Server) toolGetProviders() (callToolResult, error) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: %w", err)
	}
	b, err := json.Marshal(providers)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolHealth() (callToolResult, error) {
	b, _ := json.Marshal(map[string]string{"status": "ok"})
	return textResult(string(b)), nil
}

func (s *Server) toolCreateDraft(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath  string   `json:"projectPath"`
		Instruction  string   `json:"instruction"`
		TargetFile   string   `json:"targetFile"`
		ProviderName string   `json:"providerName"`
		ModelID      string   `json:"modelId"`
		Priority     int      `json:"priority"`
		Tags         []string `json:"tags"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: create_draft: invalid arguments: %w", err)
	}
	t := domain.Task{
		ProjectPath:  p.ProjectPath,
		Instruction:  p.Instruction,
		TargetFile:   p.TargetFile,
		ProviderName: p.ProviderName,
		ModelID:      p.ModelID,
		Priority:     p.Priority,
		Tags:         p.Tags,
	}
	id, err := s.orch.CreateDraft(t)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: create_draft: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"id": id, "status": string(domain.StatusDraft)})
	return textResult(string(b)), nil
}

func (s *Server) toolGetBacklog(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: invalid arguments: %w", err)
	}
	tasks, err := s.orch.GetBacklog(p.ProjectPath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolPromoteTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_task: invalid arguments: %w", err)
	}
	result, err := s.orch.PromoteTask(p.ID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_task: %w", err)
	}
	resp := map[string]any{"promoted": result.Promoted}
	if result.Warning != "" {
		resp["warning"] = result.Warning
	}
	b, _ := json.Marshal(resp)
	return textResult(string(b)), nil
}

func (s *Server) toolUpdateTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID           string   `json:"id"`
		Instruction  string   `json:"instruction"`
		Priority     int      `json:"priority"`
		ProviderName string   `json:"providerName"`
		ModelID      string   `json:"modelId"`
		Tags         []string `json:"tags"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: invalid arguments: %w", err)
	}
	updates := domain.Task{
		Instruction:  p.Instruction,
		Priority:     p.Priority,
		ProviderName: p.ProviderName,
		ModelID:      p.ModelID,
		Tags:         p.Tags,
	}
	updated, err := s.orch.UpdateTask(p.ID, updates)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: %w", err)
	}
	b, err := json.Marshal(updated)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolDiscoverProviders(ctx context.Context) (callToolResult, error) {
	providers, err := s.orch.TriggerScan(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: discover_providers: %w", err)
	}
	b, err := json.Marshal(providers)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: discover_providers: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolPromoteProvider(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_provider: invalid arguments: %w", err)
	}
	if err := s.orch.PromoteProvider(ctx, p.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return callToolResult{}, &mcpError{code: codeInvalidParams, msg: fmt.Sprintf("provider not found: %s", p.ID)}
		}
		return callToolResult{}, fmt.Errorf("mcp: promote_provider: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"promoted": true, "id": p.ID})
	return textResult(string(b)), nil
}

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

func (s *Server) toolListProviderConfigs(ctx context.Context) (callToolResult, error) {
	cfgs, err := s.orch.ListProviderConfigs(ctx)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: list_provider_configs: %w", err)
	}
	b, _ := json.Marshal(cfgs)
	return textResult(string(b)), nil
}

func (s *Server) toolAddProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid add_provider_config params"}
	}
	if p.Kind == "" || p.Name == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "kind and name are required"}
	}
	cfg := domain.ProviderConfig{Kind: domain.ProviderKind(p.Kind), Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Enabled: p.Enabled}
	saved, err := s.orch.AddProviderConfig(ctx, cfg)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: add_provider_config: %w", err)
	}
	b, _ := json.Marshal(saved)
	return textResult(string(b)), nil
}

func (s *Server) toolUpdateProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid update_provider_config params"}
	}
	if p.ID == "" || p.Kind == "" || p.Name == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "id, kind and name are required"}
	}
	cfg := domain.ProviderConfig{ID: p.ID, Kind: domain.ProviderKind(p.Kind), Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Enabled: p.Enabled}
	updated, err := s.orch.UpdateProviderConfig(ctx, cfg)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_provider_config: %w", err)
	}
	b, _ := json.Marshal(updated)
	return textResult(string(b)), nil
}

func (s *Server) toolRemoveProviderConfig(ctx context.Context, args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid remove_provider_config params"}
	}
	if p.ID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "id is required"}
	}
	if err := s.orch.RemoveProviderConfig(ctx, p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: remove_provider_config: %w", err)
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
	data, _ := json.Marshal(files)
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

// toolHowto returns the complete integration guide as a text block.
// This is the in-protocol equivalent of GET /api/howto — useful when an AI
// agent has only MCP access and no direct HTTP connectivity.
func (s *Server) toolHowto() (callToolResult, error) {
	guide := `nexusOrchestrator — Integration Guide
======================================

WHAT IS THIS?
A multi-LLM AI task orchestration server. Humans (or AIs) submit tasks;
AI agents discover the queue, claim tasks, execute them, and report results.

QUICK START (AI WORKER)
1. call register_session   — identify yourself with a name, role, and model
2. call get_queue          — see tasks available to claim
3. call claim_task         — take ownership of a task (prevents duplicate work)
4. execute the task using your own LLM / reasoning capabilities
5. call update_task_status — report status: COMPLETED or FAILED with logs
6. repeat from step 2
7. call register_session periodically as a heartbeat to stay visible

QUICK START (AI PLANNER)
1. call create_draft       — create backlog items without queuing them
2. call get_backlog        — review drafts
3. call promote_task       — move a draft into the active execution queue
4. call update_task        — adjust instruction, priority, provider, or tags

QUICK START (AI ORCHESTRATOR)
1. call submit_task        — queue a task directly for LLM execution
2. call get_queue          — monitor progress
3. call get_task           — inspect a specific task
4. call cancel_task        — abort if needed
5. call get_providers      — see which LLM backends are available

ALL AVAILABLE TOOLS
- howto              this guide
- health             ping the daemon
- get_providers      list active LLM backends
- discover_providers scan the local system for AI providers
- promote_provider   activate a discovered provider
- submit_task        queue a task for LLM execution
- get_task           get task status and output
- get_queue          list queued/processing tasks
- get_all_tasks      list every task regardless of status
- cancel_task        cancel a pending task
- create_draft       create a backlog draft
- get_backlog        list backlog drafts
- promote_task       move draft → execution queue
- update_task        update task fields
- register_session   announce this AI session (call on startup + as heartbeat)
- get_ai_sessions    list all registered AI sessions
- claim_task                   claim a queued task for execution
- update_task_status           report completion or failure
- heartbeat_task               keep a claimed task alive
- terminate_ai_session         force-stop or gracefully end an AI session
- list_provider_configs        list persisted provider configurations
- add_provider_config          add a new provider configuration
- update_provider_config       update an existing provider configuration
- remove_provider_config       remove a provider configuration
- deregister_ai_session        deregister an AI session
- heartbeat_ai_session         send a keepalive for an AI session
- purge_disconnected_sessions  remove all stale/disconnected sessions
- get_discovered_agents        list discovered AI agents on this machine
- delegate_to_nexus            get delegation instruction for an AI session
- get_discovered_plans         scan for plan/task/orchestration files in a project directory

HTTP ENDPOINTS (all at :63987 by default)
GET  /.well-known/nexus.json  service discovery beacon
GET  /api/howto               this guide in JSON form
GET  /api/health              health check
GET  /api/events              SSE real-time stream
POST /api/tasks               submit a task
GET  /api/tasks               list tasks
.. and more — see /api/howto for the full endpoint list
`
	return textResult(guide), nil
}

// toolHowtoBrief returns an ultra-compact integration guide for small-context models.
func (s *Server) toolHowtoBrief() (callToolResult, error) {
	guide := `nexusOrchestrator — Quick Start (compact edition)
==================================================
You are connected to nexusOrchestrator, an AI task orchestration server.

FIRST STEPS (run in order):
  1. get_project_context {"project_path": "/path/to/project"}
     → Returns active plan, task counts, guidance.
  2. get_focused_context {"task_id": "TASK-NNN"}
     → Returns implementation steps + files to read for one task.
  3. claim_task {"task_id": "TASK-NNN", "session_id": "your-session-id"}
     → Marks the task as yours (PROCESSING).
  4. update_task_status {"task_id": "TASK-NNN", "status": "COMPLETED", "logs": "summary"}
     → Marks done. Use "FAILED" if it failed.

KEY TOOLS:
  howto              — full guide (large context only)
  howto_brief        — this guide
  get_project_context — compact project snapshot
  get_focused_context — task implementation bundle
  get_queue          — list queued tasks (compact, prefer over get_all_tasks)
  submit_task        — queue a new task for an LLM
  health             — ping daemon
  register_model_capabilities — store your context window size
  get_model_capabilities      — look up known model profiles

SMALL-CONTEXT TIPS:
  Use get_project_context first, then get_focused_context for ONE task at a time.
  Do NOT call get_all_tasks (response too large). Use get_queue instead.
  Register: register_model_capabilities {"model_id": "...", "context_window": 32768}
  Tool responses show [~N tokens] budget estimate where applicable.
`
	return textResult(guide), nil
}

// ----- Helpers -----

func textResult(text string) callToolResult {
	return callToolResult{Content: []contentItem{{Type: "text", Text: text}}}
}

// ----- Tool schema catalogue -----

func toolList() []toolDef {
	return []toolDef{
		{
			Name:        "submit_task",
			Description: "Submit a new code-generation task to the orchestrator.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":  {Type: "string", Description: "Absolute path to the project root."},
					"targetFile":   {Type: "string", Description: "Relative path of the file to generate or modify."},
					"instruction":  {Type: "string", Description: "Natural-language instruction for the LLM."},
					"contextFiles": {Type: "array", Description: "Optional list of relative file paths to include as context.", Items: &propertyItems{Type: "string"}},
					"command":      {Type: "string", Description: "Task type: plan, execute, or auto (default: auto)."},
				},
				Required: []string{"projectPath", "targetFile", "instruction"},
			},
		},
		{
			Name:        "get_task",
			Description: "Get the current status and output of a task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID returned by submit_task."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "get_queue",
			Description: "List all tasks currently in the queue.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "get_all_tasks",
			Description: "Return every task regardless of status (QUEUED, PROCESSING, DRAFT, BACKLOG, COMPLETED, FAILED, CANCELLED).",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "cancel_task",
			Description: "Cancel a pending task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID to cancel."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "get_providers",
			Description: "List available LLM providers and their models.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "health",
			Description: "Check that the nexusOrchestrator daemon is reachable.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "create_draft",
			Description: "Create a draft idea for a project without entering the execution queue.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":  {Type: "string", Description: "Absolute path to the project."},
					"instruction":  {Type: "string", Description: "What the task should do."},
					"targetFile":   {Type: "string", Description: "File to write output to (optional)."},
					"providerName": {Type: "string", Description: "Exact provider name for routing (optional)."},
					"modelId":      {Type: "string", Description: "Model to use (optional)."},
					"priority":     {Type: "integer", Description: "Priority 1=high, 2=medium, 3=low (default 2)."},
					"tags":         {Type: "array", Description: "Labels (optional).", Items: &propertyItems{Type: "string"}},
				},
				Required: []string{"projectPath", "instruction"},
			},
		},
		{
			Name:        "get_backlog",
			Description: "List draft and backlog items for a project, ordered by priority.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project."},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "promote_task",
			Description: "Promote a draft or backlog task to the execution queue.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Task ID to promote."},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "update_task",
			Description: "Update mutable fields on an existing task (instruction, priority, provider, tags, status).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":           {Type: "string", Description: "Task ID to update."},
					"instruction":  {Type: "string", Description: "Updated instruction."},
					"priority":     {Type: "integer", Description: "Updated priority."},
					"providerName": {Type: "string", Description: "Updated provider name."},
					"modelId":      {Type: "string", Description: "Updated model ID."},
					"tags":         {Type: "array", Description: "Updated tags.", Items: &propertyItems{Type: "string"}},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "discover_providers",
			Description: "Scan the local system for installed AI providers/agents and return discovered results",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "promote_provider",
			Description: "Promote a discovered provider to an active LLM backend",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "ID of the discovered provider to promote"},
				},
				Required: []string{"id"},
			},
		},
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
			Name:        "howto",
			Description: "Return a complete integration guide — what nexusOrchestrator does, all tools, workflow patterns for worker/planner/orchestrator roles, and HTTP endpoint reference. Call this first when you connect.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "howto_brief",
			Description: "Get the ultra-compact integration guide (~200 tokens). RECOMMENDED as first call for small-context models (< 64K token context window). Use howto for the full guide.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
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
			Name:        "list_provider_configs",
			Description: "List all persisted LLM provider configuration records.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "add_provider_config",
			Description: "Add a new LLM provider configuration and register it when enabled=true.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"kind":     {Type: "string", Description: "Provider kind (e.g. lmstudio, ollama, openai, anthropic)."},
					"name":     {Type: "string", Description: "Display name for the provider."},
					"base_url": {Type: "string", Description: "API base URL (e.g. http://127.0.0.1:1234/v1)."},
					"api_key":  {Type: "string", Description: "API key if required."},
					"enabled":  {Type: "boolean", Description: "Whether to activate the provider immediately."},
				},
				Required: []string{"kind", "name"},
			},
		},
		{
			Name:        "update_provider_config",
			Description: "Update an existing LLM provider configuration by ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":       {Type: "string", Description: "Provider config ID to update."},
					"kind":     {Type: "string", Description: "Provider kind."},
					"name":     {Type: "string", Description: "Display name."},
					"base_url": {Type: "string", Description: "API base URL."},
					"api_key":  {Type: "string", Description: "API key."},
					"enabled":  {Type: "boolean", Description: "Whether the provider is active."},
				},
				Required: []string{"id", "kind", "name"},
			},
		},
		{
			Name:        "remove_provider_config",
			Description: "Delete a persisted provider configuration and deregister its adapter.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Provider config ID to remove."},
				},
				Required: []string{"id"},
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
			Description: "Scan for plan/task/orchestration files in a project directory. Returns nexus orchestrator.json, markdown task files, Cursor rules, MCP configs, and more.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project root to scan"},
				},
			},
		},
	}
}
