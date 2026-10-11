package repo_sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

func migrate(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), `
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
			last_seen        DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_discovered_agents_kind ON discovered_agents(kind);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate: %w", err)
	}
	_, err = db.ExecContext(context.Background(), `
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_project_path ON tasks(project_path);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate: %w", err)
	}
	// Additive column migrations — safe to re-run; errors are ignored if columns already exist.
	for _, col := range []struct{ name, def string }{
		{"model_id", "TEXT NOT NULL DEFAULT ''"},
		{"provider_hint", "TEXT NOT NULL DEFAULT ''"},
		{"command", "TEXT NOT NULL DEFAULT ''"},
		{"provider_name", "TEXT NOT NULL DEFAULT ''"},
		{"priority", "INTEGER NOT NULL DEFAULT 2"},
		{"tags", "TEXT NOT NULL DEFAULT '[]'"},
		{"retry_count", "INTEGER NOT NULL DEFAULT 0"},
		{"ai_session_id", "TEXT NOT NULL DEFAULT ''"},
		{"verification_command", "TEXT NOT NULL DEFAULT ''"},
		{"max_correction_turns", "INTEGER NOT NULL DEFAULT 0"},
		{"verification_output", "TEXT NOT NULL DEFAULT ''"},
		{"role", "TEXT NOT NULL DEFAULT ''"},
		{"agentic", "INTEGER NOT NULL DEFAULT 0"},
		{"max_agent_turns", "INTEGER NOT NULL DEFAULT 0"},
	} {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col.name, col.def))
	}
	for _, col := range []struct{ table, name, def string }{
		{"ai_sessions", "delegated_to_nexus", "INTEGER NOT NULL DEFAULT 0"},
		{"ai_sessions", "delegation_timestamp", "DATETIME"},
		{"ai_sessions", "agent_capabilities", "TEXT NOT NULL DEFAULT '[]'"},
		{"ai_sessions", "detection_method", "TEXT NOT NULL DEFAULT ''"},
		{"ai_sessions", "model_id", "TEXT NOT NULL DEFAULT ''"},
		{"ai_sessions", "current_activity", "TEXT NOT NULL DEFAULT ''"},
		{"ai_sessions", "message_count", "INTEGER NOT NULL DEFAULT 0"},
		{"ai_sessions", "tokens_used", "INTEGER NOT NULL DEFAULT 0"},
		{"ai_sessions", "last_message", "TEXT NOT NULL DEFAULT ''"},
	} {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", col.table, col.name, col.def))
	}
	_, err = db.ExecContext(context.Background(), `
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
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate discovered_plan_files: %w", err)
	}
	_, err = db.ExecContext(context.Background(), `
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
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate ai_activities: %w", err)
	}
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS model_capabilities (
			model_id               TEXT    PRIMARY KEY,
			context_window         INTEGER NOT NULL DEFAULT 0,
			recommended_max_output INTEGER NOT NULL DEFAULT 0,
			notes                  TEXT    NOT NULL DEFAULT '',
			built_in               INTEGER NOT NULL DEFAULT 0,
			created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate model_capabilities: %w", err)
	}
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS runtime_config (
			id         INTEGER PRIMARY KEY CHECK(id = 1),
			json       TEXT    NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		);
		INSERT OR IGNORE INTO runtime_config (id, updated_at) VALUES (1, CAST(strftime('%s','now') AS INTEGER) * 1000);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate runtime_config: %w", err)
	}
	if err := migrateKnowledge(db); err != nil {
		return err
	}
	return nil
}
