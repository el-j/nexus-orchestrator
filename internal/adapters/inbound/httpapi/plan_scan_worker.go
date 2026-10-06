package httpapi

import (
	"context"
	"log"
	"path/filepath"
	"sort"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

// StartPlanScanWorker runs a background goroutine that periodically refreshes
// the discovered plan files (TASKS.md, CLAUDE.md, .claude/tasks/*, ...) of every
// project the daemon knows about, so the plan list stays current without a
// client having to request a scan. It stops when ctx is cancelled.
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
				scanKnownProjectPlans(ctx, orch)
			}
		}
	}()
}

// scanKnownProjectPlans scans the plan files of each distinct project that has
// tasks or AI sessions. Failures are logged per project and never stop the sweep.
func scanKnownProjectPlans(ctx context.Context, orch ports.Orchestrator) {
	for _, project := range knownProjectPaths(ctx, orch) {
		if _, err := orch.GetDiscoveredPlanFiles(ctx, project); err != nil {
			log.Printf("httpapi: background plan scan %s: %v", project, err)
		}
	}
}

// knownProjectPaths returns the sorted, de-duplicated project paths referenced by
// tasks and AI sessions.
func knownProjectPaths(ctx context.Context, orch ports.Orchestrator) []string {
	seen := map[string]bool{}
	add := func(p string) {
		if p != "" {
			seen[filepath.Clean(p)] = true
		}
	}
	if tasks, err := orch.GetAllTasks(); err == nil {
		for _, t := range tasks {
			add(t.ProjectPath)
		}
	}
	if sessions, err := orch.ListAISessions(ctx); err == nil {
		for _, s := range sessions {
			add(s.ProjectPath)
		}
	}
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
