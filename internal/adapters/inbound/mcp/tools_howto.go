package mcp

// ----- Howto Handlers -----

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
- get_project_context          get macro context for a project bounded by a token budget
- get_focused_context          get task-specific micro context using semantic search
- search_knowledge             full-text search across the project's knowledge base
- get_brain_status             check the knowledge repository status for a project
- ingest_knowledge             parse and ingest a markdown file into the knowledge repository
- init_project                 auto-ingest CLAUDE.md and init project knowledge base
- list_knowledge               list all knowledge entries for a project
- delete_knowledge             delete a knowledge entry by ID
- get_file_map                 get the project file path map

CLIENT SETUP
VS Code (GitHub Copilot / Copilot Chat):
  mcp.json → servers → Nexus Orchestrator:
  { "type": "http", "url": "http://127.0.0.1:63988/mcp" }
  Use type:"http" (Streamable HTTP) NOT type:"sse" — avoids 400 errors on
  daemon restart. Streamable HTTP is stateless; SSE holds session in memory.

Legacy SSE (Continue IDE, Cursor, older clients):
  { "type": "sse", "url": "http://127.0.0.1:63988/sse" }
  Sessions invalidate on restart — client will auto-reconnect after ~1 ping cycle.

stdio (Claude Desktop, any stdio-only client):
  Use nexus-mcp-stdio subprocess bridge. See docs/mcp-integration.md.

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
  1. get_project_context {"projectPath": "/path/to/project"}
     → Returns active plan, task counts, guidance.
  2. get_focused_context {"projectPath": "/path/to/project", "question": "TASK-NNN implementation"}
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
  get_brain_status    — get project knowledge base status
  ingest_knowledge    — ingest a markdown file into project brain
  search_knowledge    — full-text search the project knowledge base
  register_model_capabilities — store your context window size
  get_model_capabilities      — look up known model profiles

VS CODE SETUP:
  mcp.json → { "type": "http", "url": "http://127.0.0.1:63988/mcp" }
  Use type:"http" (Streamable HTTP) NOT type:"sse" — reconnects cleanly after daemon restart.

SMALL-CONTEXT TIPS:
  Use get_project_context first, then get_focused_context for ONE task at a time.
  Do NOT call get_all_tasks (response too large). Use get_queue instead.
  Register: register_model_capabilities {"model_id": "...", "context_window": 32768}
  Tool responses show [~N tokens] budget estimate where applicable.
`
	return textResult(guide), nil
}

// ----- Howto Schema Definitions -----

func howtoToolDefs() []toolDef {
	return []toolDef{
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
	}
}
