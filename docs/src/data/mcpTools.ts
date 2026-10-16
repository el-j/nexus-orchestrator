/**
 * The MCP tool catalogue shown on the MCP Integration page.
 */

export const tools = [
  // Tasks
  {
    category: 'Tasks',
    name: 'submit_task',
    desc: 'Submit a new code-generation task to the orchestrator',
  },
  {
    category: 'Tasks',
    name: 'get_task',
    desc: 'Get the current status and output of a task by ID',
  },
  {
    category: 'Tasks',
    name: 'wait_for_task',
    desc: 'Wait until a task has finished and return it; streams the reply as progress notifications when asked',
  },
  { category: 'Tasks', name: 'get_queue', desc: 'List all tasks currently in the queue' },
  {
    category: 'Tasks',
    name: 'get_all_tasks',
    desc: 'Return every task regardless of status (QUEUED, PROCESSING, DRAFT, BACKLOG, COMPLETED, FAILED, CANCELLED)',
  },
  { category: 'Tasks', name: 'cancel_task', desc: 'Cancel a pending task by ID' },
  {
    category: 'Tasks',
    name: 'update_task',
    desc: 'Update mutable fields on an existing task (instruction, priority, provider, tags, status)',
  },
  {
    category: 'Tasks',
    name: 'create_draft',
    desc: 'Create a draft idea for a project without entering the execution queue',
  },
  {
    category: 'Tasks',
    name: 'get_backlog',
    desc: 'List draft and backlog items for a project, ordered by priority',
  },
  {
    category: 'Tasks',
    name: 'promote_task',
    desc: 'Promote a draft or backlog task to the execution queue',
  },
  {
    category: 'Tasks',
    name: 'claim_task',
    desc: 'Claim a QUEUED task for execution by the specified AI session, transitioning it to PROCESSING',
  },
  {
    category: 'Tasks',
    name: 'update_task_status',
    desc: 'Report task completion or failure from the executing AI session',
  },
  {
    category: 'Tasks',
    name: 'heartbeat_task',
    desc: 'Keep a PROCESSING task alive — prevents the watchdog from marking it failed',
  },
  {
    category: 'Tasks',
    name: 'terminate_ai_session',
    desc: 'Terminate an external AI agent session (SIGTERM by default, SIGKILL when force=true)',
  },
  // AI Sessions
  {
    category: 'AI Sessions',
    name: 'register_session',
    desc: 'Announce this AI agent session to nexusOrchestrator for visualisation and orchestration',
  },
  {
    category: 'AI Sessions',
    name: 'get_ai_sessions',
    desc: 'Return the list of all known external AI agent sessions registered with this instance',
  },
  {
    category: 'AI Sessions',
    name: 'deregister_ai_session',
    desc: 'Soft-disconnect an AI agent session, marking it as disconnected without killing the process',
  },
  {
    category: 'AI Sessions',
    name: 'heartbeat_ai_session',
    desc: 'Refresh the last-activity timestamp of an AI session to keep it alive',
  },
  {
    category: 'AI Sessions',
    name: 'purge_disconnected_sessions',
    desc: 'Delete all AI sessions with status "disconnected" inactive for more than 2 hours',
  },
  // Providers
  {
    category: 'Providers',
    name: 'get_providers',
    desc: 'List available LLM providers and their models',
  },
  {
    category: 'Providers',
    name: 'discover_providers',
    desc: 'Scan the local system for installed AI providers/agents and return discovered results',
  },
  {
    category: 'Providers',
    name: 'promote_provider',
    desc: 'Promote a discovered provider to an active LLM backend',
  },
  {
    category: 'Providers',
    name: 'list_provider_configs',
    desc: 'List all persisted LLM provider configuration records',
  },
  {
    category: 'Providers',
    name: 'add_provider_config',
    desc: 'Add a new LLM provider configuration and register it when enabled=true',
  },
  {
    category: 'Providers',
    name: 'update_provider_config',
    desc: 'Update an existing LLM provider configuration by ID',
  },
  {
    category: 'Providers',
    name: 'remove_provider_config',
    desc: 'Delete a persisted provider configuration and deregister its adapter',
  },
  // Brain / Knowledge
  {
    category: 'Brain / Knowledge',
    name: 'get_brain_status',
    desc: 'Check the knowledge repository status for a project',
  },
  {
    category: 'Brain / Knowledge',
    name: 'ingest_knowledge',
    desc: 'Parse and ingest a markdown file (often CLAUDE.md) into the project knowledge repository',
  },
  {
    category: 'Brain / Knowledge',
    name: 'get_project_context',
    desc: 'Get macro context for a project (Architectures, Conventions, File Maps) bounded by a token budget',
  },
  {
    category: 'Brain / Knowledge',
    name: 'get_onboarding_context',
    desc: 'Get a token-budgeted (< 800 tokens) onboarding summary for agent bootstrap: stack, invariants, test commands, active plan and tasks',
  },
  {
    category: 'Brain / Knowledge',
    name: 'get_focused_context',
    desc: 'Get task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget',
  },
  {
    category: 'Brain / Knowledge',
    name: 'search_knowledge',
    desc: "Perform full-text search across the project's knowledge base",
  },
  {
    category: 'Brain / Knowledge',
    name: 'init_project',
    desc: "Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step",
  },
  {
    category: 'Brain / Knowledge',
    name: 'list_knowledge',
    desc: 'List all knowledge documents stored for a project, optionally filtered by kind',
  },
  {
    category: 'Brain / Knowledge',
    name: 'delete_knowledge',
    desc: 'Delete a knowledge document by ID from the project repository',
  },
  {
    category: 'Brain / Knowledge',
    name: 'get_file_map',
    desc: 'Retrieve the cached or structured file map knowledge document for a project',
  },
  // Discovery & System
  {
    category: 'Discovery & System',
    name: 'get_discovered_agents',
    desc: 'Return AI agent tools detected on the local system (Claude CLI, VS Code Copilot, etc.)',
  },
  {
    category: 'Discovery & System',
    name: 'delegate_to_nexus',
    desc: 'Delegate an AI agent session to the nexus task queue and return the workflow instruction string',
  },
  {
    category: 'Discovery & System',
    name: 'get_discovered_plans',
    desc: 'Scan for plan/task/orchestration files in a project directory',
  },
  {
    category: 'Discovery & System',
    name: 'howto',
    desc: 'Return a complete integration guide — all tools, workflow patterns, and HTTP endpoint reference',
  },
  {
    category: 'Discovery & System',
    name: 'howto_brief',
    desc: 'Get the ultra-compact integration guide (~200 tokens), recommended for small-context models',
  },
  {
    category: 'Discovery & System',
    name: 'health',
    desc: 'Check that the nexusOrchestrator daemon is reachable',
  },
  {
    category: 'Discovery & System',
    name: 'get_routing',
    desc: 'How providers would be ordered for a role in a project: model ladder, cooldowns and quota readings',
  },
  {
    category: 'Discovery & System',
    name: 'get_usage',
    desc: 'Tokens and cost per provider, model and role from the usage ledger (since: 24h, 7d, an RFC 3339 time or all)',
  },
];
