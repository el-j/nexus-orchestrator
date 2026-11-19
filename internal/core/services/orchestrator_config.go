package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"nexus-orchestrator/internal/core/domain"
	"os"
	"time"
)

// GetRuntimeConfig returns the current runtime configuration.
func (o *OrchestratorService) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	o.mu.Lock()
	queueCap := o.queueCap
	repo := o.runtimeCfg
	o.mu.Unlock()

	effectiveCap := queueCap
	if effectiveCap <= 0 {
		effectiveCap = 50
	}

	cfg := domain.RuntimeConfig{
		QueueCap: effectiveCap,
		APIToken: os.Getenv("NEXUS_API_TOKEN"),
		MCPToken: os.Getenv("NEXUS_MCP_TOKEN"),
	}

	if repo == nil {
		return cfg, nil
	}
	stored, err := repo.GetRuntimeConfig(ctx)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: get runtime config: %w", err)
	}
	if err == nil {
		// Prefer persisted values when env vars are not set.
		if cfg.APIToken == "" {
			cfg.APIToken = stored.APIToken
		}
		if cfg.MCPToken == "" {
			cfg.MCPToken = stored.MCPToken
		}
		if stored.QueueCap > 0 {
			cfg.QueueCap = stored.QueueCap
		}
		cfg.UpdatedAt = stored.UpdatedAt
	}

	return cfg, nil
}

// UpdateRuntimeConfig validates, applies and persists a new runtime configuration.
func (o *OrchestratorService) UpdateRuntimeConfig(ctx context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	o.mu.Lock()
	repo := o.runtimeCfg
	o.mu.Unlock()

	current, err := o.GetRuntimeConfig(ctx)
	if err != nil {
		return domain.RuntimeConfig{}, err
	}

	if update.QueueCap != nil {
		if *update.QueueCap <= 0 {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: update runtime config: queueCap must be > 0")
		}
		current.QueueCap = *update.QueueCap
	}

	if update.RotateAPIToken {
		current.APIToken = generateToken()
	}
	if update.RotateMCPToken {
		current.MCPToken = generateToken()
	}
	if update.APIToken != nil {
		current.APIToken = *update.APIToken
	}
	if update.MCPToken != nil {
		current.MCPToken = *update.MCPToken
	}

	current.UpdatedAt = time.Now()
	if repo != nil {
		if err := repo.SaveRuntimeConfig(ctx, current); err != nil {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: save runtime config: %w", err)
		}
	}

	// Apply the in-memory cap only after it has been persisted, so a failed save
	// cannot leave the running service and the stored config disagreeing.
	if update.QueueCap != nil {
		o.WithQueueCap(current.QueueCap)
	}

	return current, nil
}

// generateToken returns 32 random bytes as URL-safe text. crypto/rand.Read cannot fail: it ends the
// program when the system has no randomness.
func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
