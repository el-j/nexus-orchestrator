package repo_sqlite

import (
	"context"
	"database/sql"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// DiscoveredAgentRepo persists domain.DiscoveredAgent records using the shared SQLite database.
type DiscoveredAgentRepo struct {
	db *sql.DB
}

// NewDiscoveredAgentRepo returns a DiscoveredAgentRepo backed by the same database as r.
func NewDiscoveredAgentRepo(r *Repository) *DiscoveredAgentRepo {
	return &DiscoveredAgentRepo{db: r.db}
}

// UpsertDiscoveredAgent inserts or replaces a discovered agent record.
func (d *DiscoveredAgentRepo) UpsertDiscoveredAgent(ctx context.Context, a domain.DiscoveredAgent) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO discovered_agents
			(id, kind, name, detection_method, process_name, cli_path, config_path, mcp_endpoint, is_running, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, string(a.Kind), a.Name, a.DetectionMethod,
		a.ProcessName, a.CLIPath, a.ConfigPath, a.MCPEndpoint,
		boolToInt(a.IsRunning), a.LastSeen.UTC().Format(time.RFC3339),
	)
	return err
}

// ListDiscoveredAgents returns all discovered agent records ordered by last_seen descending.
func (d *DiscoveredAgentRepo) ListDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, kind, name, detection_method, process_name, cli_path, config_path, mcp_endpoint, is_running, last_seen
		FROM discovered_agents ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []domain.DiscoveredAgent
	for rows.Next() {
		var a domain.DiscoveredAgent
		var lastSeenStr string
		var isRunningInt int
		if err := rows.Scan(&a.ID, (*string)(&a.Kind), &a.Name, &a.DetectionMethod,
			&a.ProcessName, &a.CLIPath, &a.ConfigPath, &a.MCPEndpoint,
			&isRunningInt, &lastSeenStr); err != nil {
			return nil, err
		}
		a.IsRunning = isRunningInt != 0
		a.LastSeen, _ = time.Parse(time.RFC3339, lastSeenStr)
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// boolToInt converts a bool to 1 (true) or 0 (false) for SQLite INTEGER storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
