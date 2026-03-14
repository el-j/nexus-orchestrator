package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// howToDoc is the machine-readable "how to work with nexusOrchestrator" guide.
// Served at GET /api/howto and consumed by AIs, CLIs, and clients on first contact.
type howToDoc struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	QuickStart  string          `json:"quick_start"`
	Connection  howToConnection `json:"connection"`
	AIWorkflows howToWorkflows  `json:"ai_workflows"`
	Endpoints   []howToEndpoint `json:"http_endpoints"`
	Examples    []howToExample  `json:"examples"`
}

type howToConnection struct {
	HTTPAPI   string `json:"http_api"`
	MCP       string `json:"mcp_endpoint"`
	Dashboard string `json:"dashboard"`
	Discovery string `json:"discovery"`
}

type howToWorkflows struct {
	AsWorker       []string `json:"as_worker"`
	AsPlanner      []string `json:"as_planner"`
	AsOrchestrator []string `json:"as_orchestrator"`
}

type howToEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type howToExample struct {
	Description string `json:"description"`
	Request     string `json:"request"`
}

// nexusDiscoveryDoc is served at /.well-known/nexus.json.
// It is the lightweight "here I am" beacon — a single well-known URL any tool or
// AI can GET immediately after discovering the server's address.
type nexusDiscoveryDoc struct {
	SchemaVersion string            `json:"schema_version"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	API           nexusDiscoveryAPI `json:"api"`
	MCP           nexusDiscoveryMCP `json:"mcp"`
	Capabilities  []string          `json:"capabilities"`
}

type nexusDiscoveryAPI struct {
	BaseURL string `json:"base_url"`
	HowTo   string `json:"howto"`
	Health  string `json:"health"`
	Events  string `json:"events"`
}

type nexusDiscoveryMCP struct {
	Protocol string `json:"protocol"`
	Version  string `json:"version"`
	Note     string `json:"note"`
}

// buildHowToDoc constructs the full integration guide using the request's base URL
// so every link in the document points at the actual live server address.
func buildHowToDoc(baseURL string) howToDoc {
	return howToDoc{
		Name:    "nexusOrchestrator",
		Version: "1.0.0",
		Description: "Multi-LLM AI task orchestration server. A human (or another AI) submits " +
			"coding, research, or analysis tasks. AI agents discover the queue, claim tasks, " +
			"execute them using any registered LLM provider, and report results back. " +
			"Multiple AI agents of different types coordinate through a shared task queue " +
			"with session isolation and per-project conversation history.",
		QuickStart: "1. GET /api/howto  (you are here — read this document). " +
			"2. GET /api/providers  to see which LLM backends are available. " +
			"3. POST /api/ai-sessions  to register yourself as a named agent. " +
			"4. GET /api/tasks  to see the active work queue. " +
			"5. POST /api/tasks/{id}/claim  to take ownership of a task. " +
			"6. PUT /api/tasks/{id}/status  to report progress and final results. " +
			"7. POST /api/ai-sessions/{id}/heartbeat  every 60 s to stay registered. " +
			"8. DELETE /api/ai-sessions/{id}  when your session ends.",
		Connection: howToConnection{
			HTTPAPI:   baseURL,
			MCP:       "JSON-RPC 2.0 — see " + baseURL + "/.well-known/nexus.json for the MCP port",
			Dashboard: baseURL + "/ui",
			Discovery: baseURL + "/.well-known/nexus.json",
		},
		AIWorkflows: howToWorkflows{
			AsWorker: []string{
				"POST /api/ai-sessions  — register your session with name, role, and model",
				"GET /api/tasks  — list queued tasks available to claim",
				"POST /api/tasks/{id}/claim  — claim a task (prevents duplicate work by other agents)",
				"PUT /api/tasks/{id}/status  — update progress: body {status, result, sessionId}",
				"  status values: in_progress | completed | failed",
				"POST /api/ai-sessions/{id}/heartbeat  — keep your session alive (TTL: 5 min)",
				"DELETE /api/ai-sessions/{id}  — deregister when your session ends",
			},
			AsPlanner: []string{
				"POST /api/tasks/draft  — create a backlog draft (title, description, projectPath)",
				"GET /api/tasks/backlog  — review all draft tasks",
				"POST /api/tasks/{id}/promote  — move a draft into the active queue",
				"PUT /api/tasks/{id}  — update task title, description, or priority",
				"GET /api/tasks/all  — list every task including drafts and cancelled",
			},
			AsOrchestrator: []string{
				"POST /api/tasks  — submit a task directly to the active queue",
				"GET /api/tasks  — monitor the queue (filter by status in query params)",
				"GET /api/tasks/{id}  — inspect a single task",
				"DELETE /api/tasks/{id}  — cancel a task",
				"GET /api/providers  — check which LLM backends are currently active",
				"POST /api/providers/discovered/scan  — trigger a fresh provider discovery scan",
				"GET /api/events  — subscribe to SSE stream for real-time task + log events",
				"GET /api/ai-sessions  — list all registered AI agents",
			},
		},
		Endpoints: []howToEndpoint{
			// Tasks
			{"POST", "/api/tasks", "Submit a task directly to the active queue"},
			{"GET", "/api/tasks", "List active queue tasks (queued, in_progress, completed)"},
			{"GET", "/api/tasks/all", "List every task including drafts and cancelled"},
			{"GET", "/api/tasks/{id}", "Get a single task by ID"},
			{"DELETE", "/api/tasks/{id}", "Cancel a task"},
			{"PUT", "/api/tasks/{id}", "Update task title, description, or priority"},
			{"POST", "/api/tasks/{id}/promote", "Promote a backlog draft into the active queue"},
			{"POST", "/api/tasks/{id}/claim", "Claim a task (AI worker marks it as taken)"},
			{"PUT", "/api/tasks/{id}/status", "Update task status with optional result payload"},
			{"POST", "/api/tasks/draft", "Create a backlog draft task"},
			{"GET", "/api/tasks/backlog", "List all backlog drafts"},
			// Providers
			{"GET", "/api/providers", "List active LLM providers"},
			{"POST", "/api/providers", "Register a new provider"},
			{"DELETE", "/api/providers/{name}", "Remove a provider"},
			{"GET", "/api/providers/{name}/models", "List models available from a provider"},
			{"GET", "/api/providers/discovered", "List providers found by the system scanner"},
			{"POST", "/api/providers/discovered/scan", "Trigger a fresh provider discovery scan"},
			{"POST", "/api/providers/config", "Persist a provider configuration (with API-key masking)"},
			{"GET", "/api/providers/config", "List persisted provider configurations"},
			{"PUT", "/api/providers/config/{id}", "Update a persisted provider configuration"},
			{"DELETE", "/api/providers/config/{id}", "Remove a persisted provider configuration"},
			// AI Sessions
			{"POST", "/api/ai-sessions", "Register an AI session (name, role, model)"},
			{"GET", "/api/ai-sessions", "List all registered AI sessions"},
			{"DELETE", "/api/ai-sessions", "Purge all disconnected sessions"},
			{"DELETE", "/api/ai-sessions/{id}", "Deregister a specific AI session"},
			{"POST", "/api/ai-sessions/{id}/heartbeat", "Renew session TTL (call at least every 5 min)"},
			{"GET", "/api/ai-sessions/{id}/tasks", "List tasks claimed by a specific session"},
			// System
			{"GET", "/api/health", "Health check — returns {status:ok}"},
			{"GET", "/api/events", "SSE stream — real-time task lifecycle and log events"},
			{"GET", "/api/howto", "This document — machine-readable integration guide"},
			{"GET", "/.well-known/nexus.json", "Service discovery beacon — identity, capabilities, endpoints"},
		},
		Examples: []howToExample{
			{
				Description: "Submit a coding task",
				Request: fmt.Sprintf(
					`curl -s -X POST %s/api/tasks \`+"\n"+
						`  -H 'Content-Type: application/json' \`+"\n"+
						`  -d '{"title":"Refactor auth module","description":"Extract JWT logic into a separate package","projectPath":"/my/project"}'`,
					baseURL),
			},
			{
				Description: "Register as an AI worker agent",
				Request: fmt.Sprintf(
					`curl -s -X POST %s/api/ai-sessions \`+"\n"+
						`  -H 'Content-Type: application/json' \`+"\n"+
						`  -d '{"name":"ClaudeWorker","role":"worker","model":"claude-opus-4-5"}'`,
					baseURL),
			},
			{
				Description: "Claim and begin working on the first queued task",
				Request: fmt.Sprintf(
					"# Step 1: find a queued task\n"+
						`curl -s '%s/api/tasks?status=queued' | jq '.[0].id'`+"\n\n"+
						"# Step 2: claim it\n"+
						`curl -s -X POST %s/api/tasks/TASK_ID/claim \`+"\n"+
						`  -H 'Content-Type: application/json' \`+"\n"+
						`  -d '{"sessionId":"YOUR_SESSION_ID"}'`,
					baseURL, baseURL),
			},
			{
				Description: "Report task completion with a result",
				Request: fmt.Sprintf(
					`curl -s -X PUT %s/api/tasks/TASK_ID/status \`+"\n"+
						`  -H 'Content-Type: application/json' \`+"\n"+
						`  -d '{"status":"completed","result":"Refactored — new package: internal/auth/jwt","sessionId":"YOUR_SESSION_ID"}'`,
					baseURL),
			},
			{
				Description: "Subscribe to real-time task and log events",
				Request:     fmt.Sprintf("curl -N -H 'Accept: text/event-stream' %s/api/events", baseURL),
			},
		},
	}
}

// handleHowto serves GET /api/howto — the full machine-readable integration guide.
// The base URL is derived from the incoming request so links always point at the
// actual live server, whether it's localhost or a remote host.
func (s *Server) handleHowto(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)
	doc := buildHowToDoc(baseURL)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(doc)
}

// handleWellKnownNexus serves GET /.well-known/nexus.json — the lightweight
// service-discovery beacon. Any tool, agent, or AI can GET this URL immediately
// after connecting to learn the server's identity, capabilities, and where to
// find the full how-to guide and the MCP endpoint.
func (s *Server) handleWellKnownNexus(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	base := fmt.Sprintf("%s://%s", scheme, r.Host)

	doc := nexusDiscoveryDoc{
		SchemaVersion: "1",
		Name:          "nexusOrchestrator",
		Description:   "Multi-LLM AI task orchestration server — submit tasks, coordinate agents, track results",
		API: nexusDiscoveryAPI{
			BaseURL: base + "/api",
			HowTo:   base + "/api/howto",
			Health:  base + "/api/health",
			Events:  base + "/api/events",
		},
		MCP: nexusDiscoveryMCP{
			Protocol: "json-rpc-2.0",
			Version:  "2024-11-05",
			Note:     "MCP server runs on a separate port — default :63988. Use NEXUS_MCP_ADDR to override.",
		},
		Capabilities: []string{
			"task-queue",
			"ai-session-registry",
			"provider-discovery",
			"llm-dispatch",
			"sse-events",
			"mcp-tools",
			"session-isolation",
			"backlog-drafts",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(doc)
}
