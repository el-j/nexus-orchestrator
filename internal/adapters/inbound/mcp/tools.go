// Package mcp — tool handlers and schema definitions.
// This file contains the handleToolCall dispatcher, the textResult helper,
// and the composite toolList() function. Domain-specific tool handlers and
// schemas are organized into tools_tasks.go, tools_providers.go,
// tools_sessions.go, tools_howto.go, and brain_tools.go.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	// Task tools
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
	case "create_draft":
		result, err = s.toolCreateDraft(p.Arguments)
	case "get_backlog":
		result, err = s.toolGetBacklog(p.Arguments)
	case "promote_task":
		result, err = s.toolPromoteTask(p.Arguments)
	case "update_task":
		result, err = s.toolUpdateTask(p.Arguments)

	// Provider tools
	case "get_providers":
		result, err = s.toolGetProviders()
	case "health":
		result, err = s.toolHealth()
	case "discover_providers":
		result, err = s.toolDiscoverProviders(r.Context())
	case "promote_provider":
		result, err = s.toolPromoteProvider(r.Context(), p.Arguments)
	case "list_provider_configs":
		result, err = s.toolListProviderConfigs(r.Context())
	case "add_provider_config":
		result, err = s.toolAddProviderConfig(r.Context(), p.Arguments)
	case "update_provider_config":
		result, err = s.toolUpdateProviderConfig(r.Context(), p.Arguments)
	case "remove_provider_config":
		result, err = s.toolRemoveProviderConfig(r.Context(), p.Arguments)

	// Session & Agent tools
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

	// Howto tools
	case "howto":
		result, err = s.toolHowto()
	case "howto_brief":
		result, err = s.toolHowtoBrief()

	// Brain & Knowledge tools
	case "get_onboarding_context":
		result, err = s.toolGetOnboardingContext(r.Context(), p.Arguments)
	case "get_project_context":
		result, err = s.toolGetProjectContext(r.Context(), p.Arguments)
	case "get_focused_context":
		result, err = s.toolGetFocusedContext(r.Context(), p.Arguments)
	case "search_knowledge":
		result, err = s.toolSearchKnowledge(r.Context(), p.Arguments)
	case "get_brain_status":
		result, err = s.toolGetBrainStatus(r.Context(), p.Arguments)
	case "ingest_knowledge":
		result, err = s.toolIngestKnowledge(r.Context(), p.Arguments)
	case "init_project":
		result, err = s.toolInitProject(r.Context(), p.Arguments)
	case "list_knowledge":
		result, err = s.toolListKnowledge(r.Context(), p.Arguments)
	case "delete_knowledge":
		result, err = s.toolDeleteKnowledge(r.Context(), p.Arguments)
	case "get_file_map":
		result, err = s.toolGetFileMap(r.Context(), p.Arguments)

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

// ----- Helpers -----

func textResult(text string) callToolResult {
	return callToolResult{Content: []contentItem{{Type: "text", Text: text}}}
}

// ----- Composite Tool Catalogue -----

func toolList() []toolDef {
	var list []toolDef
	list = append(list, taskToolDefs()...)
	list = append(list, providerToolDefs()...)
	list = append(list, sessionToolDefs()...)
	list = append(list, howtoToolDefs()...)
	list = append(list, brainToolDefs()...)
	return list
}
