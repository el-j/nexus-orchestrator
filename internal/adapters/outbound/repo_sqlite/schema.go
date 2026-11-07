package repo_sqlite

// The schema, as data: each step is idempotent (CREATE ... IF NOT EXISTS), so migrate can run on every start.

// schemaStep is one named group of statements.
type schemaStep struct{ name, sql string }

// schemaBase creates the core tables and the indexes of the task table; the additive columns follow it.
var schemaBase = []schemaStep{
	{"core tables", `
		CREATE TABLE IF NOT EXISTS tasks (
			id            TEXT    PRIMARY KEY,
			project_path  TEXT    NOT NULL,
			target_file   TEXT    NOT NULL,
			instruction   TEXT    NOT NULL,
			context_files TEXT    NOT NULL DEFAULT '[]',
			status        TEXT    NOT NULL DEFAULT 'QUEUED',
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL,
			logs          TEXT    NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id           TEXT    PRIMARY KEY,
			project_path TEXT    NOT NULL UNIQUE,
			messages     TEXT    NOT NULL DEFAULT '[]',
			created_at   INTEGER NOT NULL,
			updated_at   INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS provider_configs (
			id            TEXT    PRIMARY KEY,
			name          TEXT    NOT NULL,
			kind          TEXT    NOT NULL,
			base_url      TEXT    NOT NULL DEFAULT '',
			api_key       TEXT    NOT NULL DEFAULT '',
			model         TEXT    NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS ai_sessions (
			id               TEXT PRIMARY KEY,
			source           TEXT NOT NULL,
			external_id      TEXT NOT NULL DEFAULT '',
			agent_name       TEXT NOT NULL,
			project_path     TEXT NOT NULL DEFAULT '',
			status           TEXT NOT NULL DEFAULT 'active',
			last_activity    DATETIME NOT NULL,
			routed_task_ids  TEXT NOT NULL DEFAULT '[]',
			pid              INTEGER NOT NULL DEFAULT 0,
			created_at       DATETIME NOT NULL,
			updated_at       DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS discovered_agents (
			id               TEXT PRIMARY KEY,
			kind             TEXT NOT NULL,
			name             TEXT NOT NULL,
			detection_method TEXT NOT NULL DEFAULT '',
			process_name     TEXT NOT NULL DEFAULT '',
			cli_path         TEXT NOT NULL DEFAULT '',
			config_path      TEXT NOT NULL DEFAULT '',
			mcp_endpoint     TEXT NOT NULL DEFAULT '',
			is_running       INTEGER NOT NULL DEFAULT 0,
			pid              INTEGER NOT NULL DEFAULT 0,
			last_seen        DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_discovered_agents_kind ON discovered_agents(kind);
	`},
	{"task indexes", `
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_project_path ON tasks(project_path);
	`},
}

// schemaMore creates the tables that were added later; the knowledge tables are in knowledge_repo.go.
var schemaMore = []schemaStep{
	{"discovered_plan_files", `
		CREATE TABLE IF NOT EXISTS discovered_plan_files (
			id            TEXT    PRIMARY KEY,
			path          TEXT    NOT NULL UNIQUE,
			kind          TEXT    NOT NULL,
			format        TEXT    NOT NULL,
			project_path  TEXT    NOT NULL,
			summary       TEXT,
			last_modified INTEGER NOT NULL,
			is_active     INTEGER NOT NULL DEFAULT 0,
			updated_at    INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_discovered_plan_files_project_path ON discovered_plan_files(project_path);
	`},
	{"ai_activities", `
		CREATE TABLE IF NOT EXISTS ai_activities (
			id            TEXT PRIMARY KEY,
			session_id    TEXT NOT NULL DEFAULT '',
			agent_name    TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			summary       TEXT NOT NULL DEFAULT '',
			project_path  TEXT NOT NULL DEFAULT '',
			model         TEXT NOT NULL DEFAULT '',
			tokens_in     INTEGER NOT NULL DEFAULT 0,
			tokens_out    INTEGER NOT NULL DEFAULT 0,
			timestamp     DATETIME NOT NULL,
			metadata      TEXT NOT NULL DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_ai_activities_timestamp ON ai_activities(timestamp);
		CREATE INDEX IF NOT EXISTS idx_ai_activities_agent ON ai_activities(agent_name);
		CREATE INDEX IF NOT EXISTS idx_ai_activities_project ON ai_activities(project_path);
	`},
	{"model_capabilities", `
		CREATE TABLE IF NOT EXISTS model_capabilities (
			model_id               TEXT    PRIMARY KEY,
			context_window         INTEGER NOT NULL DEFAULT 0,
			recommended_max_output INTEGER NOT NULL DEFAULT 0,
			notes                  TEXT    NOT NULL DEFAULT '',
			built_in               INTEGER NOT NULL DEFAULT 0,
			created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`},
	{"runtime_config", `
		CREATE TABLE IF NOT EXISTS runtime_config (
			id         INTEGER PRIMARY KEY CHECK(id = 1),
			json       TEXT    NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		);
		INSERT OR IGNORE INTO runtime_config (id, updated_at) VALUES (1, CAST(strftime('%s','now') AS INTEGER) * 1000);
	`},
	{"llm_usage", `
		CREATE TABLE IF NOT EXISTS llm_usage (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			at            INTEGER NOT NULL,
			task_id       TEXT    NOT NULL DEFAULT '',
			provider      TEXT    NOT NULL,
			model         TEXT    NOT NULL DEFAULT '',
			role          TEXT    NOT NULL DEFAULT '',
			input_tokens  INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cached_tokens INTEGER NOT NULL DEFAULT 0,
			cost_usd      REAL    NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_llm_usage_at ON llm_usage(at);
	`},
}
