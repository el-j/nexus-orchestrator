package mcp

import (
	"context"
	"encoding/json"
)

// toolCall is what a tool handler is given for one tools/call.
type toolCall struct {
	ctx      context.Context
	args     json.RawMessage
	progress progressFunc // set only for wait_for_task
}

// toolHandler runs one tool.
type toolHandler func(s *Server, c toolCall) (callToolResult, error)

// argsOnly adapts a tool that takes only its arguments.
func argsOnly(f func(*Server, json.RawMessage) (callToolResult, error)) toolHandler {
	return func(s *Server, c toolCall) (callToolResult, error) { return f(s, c.args) }
}

// ctxArgs adapts a tool that takes the request context and its arguments.
func ctxArgs(f func(*Server, context.Context, json.RawMessage) (callToolResult, error)) toolHandler {
	return func(s *Server, c toolCall) (callToolResult, error) { return f(s, c.ctx, c.args) }
}

// ctxOnly adapts a tool that takes only the request context.
func ctxOnly(f func(*Server, context.Context) (callToolResult, error)) toolHandler {
	return func(s *Server, c toolCall) (callToolResult, error) { return f(s, c.ctx) }
}

// noArgs adapts a tool that takes nothing.
func noArgs(f func(*Server) (callToolResult, error)) toolHandler {
	return func(s *Server, _ toolCall) (callToolResult, error) { return f(s) }
}

// toolHandlers maps every tool name of toolList to its handler; handleToolCall looks the call up here.
var toolHandlers = map[string]toolHandler{
	"submit_task":                 argsOnly((*Server).toolSubmitTask),
	"get_task":                    argsOnly((*Server).toolGetTask),
	"get_queue":                   argsOnly((*Server).toolGetQueue),
	"get_all_tasks":               argsOnly((*Server).toolGetAllTasks),
	"cancel_task":                 argsOnly((*Server).toolCancelTask),
	"create_draft":                argsOnly((*Server).toolCreateDraft),
	"get_backlog":                 argsOnly((*Server).toolGetBacklog),
	"promote_task":                argsOnly((*Server).toolPromoteTask),
	"update_task":                 argsOnly((*Server).toolUpdateTask),
	"get_providers":               noArgs((*Server).toolGetProviders),
	"health":                      noArgs((*Server).toolHealth),
	"discover_providers":          ctxOnly((*Server).toolDiscoverProviders),
	"promote_provider":            ctxArgs((*Server).toolPromoteProvider),
	"list_provider_configs":       ctxOnly((*Server).toolListProviderConfigs),
	"add_provider_config":         ctxArgs((*Server).toolAddProviderConfig),
	"update_provider_config":      ctxArgs((*Server).toolUpdateProviderConfig),
	"remove_provider_config":      ctxArgs((*Server).toolRemoveProviderConfig),
	"register_session":            ctxArgs((*Server).toolRegisterSession),
	"get_ai_sessions":             ctxOnly((*Server).toolGetAISessions),
	"claim_task":                  ctxArgs((*Server).toolClaimTask),
	"update_task_status":          ctxArgs((*Server).toolUpdateTaskStatus),
	"heartbeat_task":              ctxArgs((*Server).toolHeartbeatTask),
	"terminate_ai_session":        ctxArgs((*Server).toolTerminateAISession),
	"deregister_ai_session":       ctxArgs((*Server).toolDeregisterAISession),
	"heartbeat_ai_session":        ctxArgs((*Server).toolHeartbeatAISession),
	"purge_disconnected_sessions": ctxOnly((*Server).toolPurgeDisconnectedSessions),
	"get_discovered_agents":       ctxOnly((*Server).toolGetDiscoveredAgents),
	"get_discovered_plans":        ctxArgs((*Server).toolGetDiscoveredPlans),
	"delegate_to_nexus":           ctxArgs((*Server).toolDelegateToNexus),
	"get_routing":                 ctxArgs((*Server).toolGetRouting),
	"get_usage":                   ctxArgs((*Server).toolGetUsage),
	"howto":                       noArgs((*Server).toolHowto),
	"howto_brief":                 noArgs((*Server).toolHowtoBrief),
	"get_onboarding_context":      ctxArgs((*Server).toolGetOnboardingContext),
	"get_project_context":         ctxArgs((*Server).toolGetProjectContext),
	"get_focused_context":         ctxArgs((*Server).toolGetFocusedContext),
	"search_knowledge":            ctxArgs((*Server).toolSearchKnowledge),
	"get_brain_status":            ctxArgs((*Server).toolGetBrainStatus),
	"ingest_knowledge":            ctxArgs((*Server).toolIngestKnowledge),
	"init_project":                ctxArgs((*Server).toolInitProject),
	"list_knowledge":              ctxArgs((*Server).toolListKnowledge),
	"delete_knowledge":            ctxArgs((*Server).toolDeleteKnowledge),
	"get_file_map":                ctxArgs((*Server).toolGetFileMap),
	"wait_for_task": func(s *Server, c toolCall) (callToolResult, error) {
		return s.toolWaitForTask(c.ctx, c.args, c.progress)
	},
}
