package repo_sqlite

// addedColumn is a column later versions added to a table that may already exist.
type addedColumn struct{ table, name, def string }

// addedColumns are applied after the tables exist; adding one that is already there is ignored.
var addedColumns = []addedColumn{
	{"tasks", "model_id", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "provider_hint", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "command", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "provider_name", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "priority", "INTEGER NOT NULL DEFAULT 2"},
	{"tasks", "tags", "TEXT NOT NULL DEFAULT '[]'"},
	{"tasks", "retry_count", "INTEGER NOT NULL DEFAULT 0"},
	{"tasks", "ai_session_id", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "verification_command", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "max_correction_turns", "INTEGER NOT NULL DEFAULT 0"},
	{"tasks", "verification_output", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "role", "TEXT NOT NULL DEFAULT ''"},
	{"tasks", "agentic", "INTEGER NOT NULL DEFAULT 0"},
	{"tasks", "max_agent_turns", "INTEGER NOT NULL DEFAULT 0"},
	{"tasks", "wait_until", "INTEGER NOT NULL DEFAULT 0"},
	{"ai_sessions", "delegated_to_nexus", "INTEGER NOT NULL DEFAULT 0"},
	{"ai_sessions", "delegation_timestamp", "DATETIME"},
	{"ai_sessions", "agent_capabilities", "TEXT NOT NULL DEFAULT '[]'"},
	{"ai_sessions", "detection_method", "TEXT NOT NULL DEFAULT ''"},
	{"ai_sessions", "model_id", "TEXT NOT NULL DEFAULT ''"},
	{"ai_sessions", "current_activity", "TEXT NOT NULL DEFAULT ''"},
	{"ai_sessions", "message_count", "INTEGER NOT NULL DEFAULT 0"},
	{"ai_sessions", "tokens_used", "INTEGER NOT NULL DEFAULT 0"},
	{"ai_sessions", "last_message", "TEXT NOT NULL DEFAULT ''"},
	{"ai_sessions", "pid", "INTEGER NOT NULL DEFAULT 0"},
	{"discovered_agents", "pid", "INTEGER NOT NULL DEFAULT 0"},
}
