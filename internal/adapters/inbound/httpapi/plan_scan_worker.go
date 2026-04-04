package httpapi

import (
	"context"
	"log"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

// StartPlanScanWorker runs a background goroutine that periodically triggers a plan
// discovery scan. It stops when ctx is cancelled.
// interval is the time between scans (recommended: 5 minutes for production).
func StartPlanScanWorker(ctx context.Context, orch ports.Orchestrator, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := orch.TriggerScan(ctx); err != nil {
					log.Printf("httpapi: background plan scan: %v", err)
				}
			}
		}
	}()
}
