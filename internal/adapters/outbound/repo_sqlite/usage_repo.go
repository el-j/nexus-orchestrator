package repo_sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.UsageRepository = (*UsageRepo)(nil)

// UsageRepo implements ports.UsageRepository: the usage ledger in the shared SQLite database.
type UsageRepo struct {
	db *sql.DB
}

// NewUsageRepo returns a UsageRepo backed by the same database as r.
func NewUsageRepo(r *Repository) *UsageRepo { return &UsageRepo{db: r.db} }

// Record stores one call. A zero At is replaced by the current time.
func (u *UsageRepo) Record(ctx context.Context, r domain.UsageRecord) error {
	if r.At.IsZero() {
		r.At = time.Now()
	}
	_, err := u.db.ExecContext(ctx, `
		INSERT INTO llm_usage (at, task_id, provider, model, role, input_tokens, output_tokens, cached_tokens, cost_usd)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.At.UnixMilli(), r.TaskID, r.Provider, r.Model, r.Role, r.InputTokens, r.OutputTokens, r.CachedTokens, r.CostUSD)
	if err != nil {
		return fmt.Errorf("sqlite: record usage: %w", err)
	}
	return nil
}

// Summarize adds the calls at or after since up per provider, model and role, largest consumers first.
func (u *UsageRepo) Summarize(ctx context.Context, since time.Time) ([]domain.UsageSummary, error) {
	var from int64
	if !since.IsZero() {
		from = since.UnixMilli()
	}
	rows, err := u.db.QueryContext(ctx, `
		SELECT provider, model, role, COUNT(*), SUM(input_tokens), SUM(output_tokens), SUM(cached_tokens), SUM(cost_usd)
		FROM llm_usage WHERE at >= ?
		GROUP BY provider, model, role
		ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC, provider, model, role`, from)
	if err != nil {
		return nil, fmt.Errorf("sqlite: summarize usage: %w", err)
	}
	defer rows.Close()
	out := []domain.UsageSummary{}
	for rows.Next() {
		var s domain.UsageSummary
		if err := rows.Scan(&s.Provider, &s.Model, &s.Role, &s.Calls, &s.InputTokens, &s.OutputTokens, &s.CachedTokens, &s.CostUSD); err != nil {
			return nil, fmt.Errorf("sqlite: scan usage: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: read usage: %w", err)
	}
	return out, nil
}
