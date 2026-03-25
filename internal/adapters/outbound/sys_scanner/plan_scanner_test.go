package sys_scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestScanPlanFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude/orchestrator.json
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "orchestrator.json"), []byte(`{"activePlanId":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create TASKS.md
	if err := os.WriteFile(filepath.Join(tmpDir, "TASKS.md"), []byte("# Tasks"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .cursorrules
	if err := os.WriteFile(filepath.Join(tmpDir, ".cursorrules"), []byte("rules here"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}

	kinds := map[domain.PlanFileKind]bool{}
	for _, r := range results {
		kinds[r.Kind] = true
		if r.ID == "" {
			t.Errorf("result has empty ID: %+v", r)
		}
		if r.Path == "" {
			t.Errorf("result has empty Path: %+v", r)
		}
		if r.Format == "" {
			t.Errorf("result has empty Format: %+v", r)
		}
	}

	if !kinds[domain.PlanFileKindNexus] {
		t.Errorf("expected PlanFileKindNexus in results, got kinds: %v", kinds)
	}
	if !kinds[domain.PlanFileKindMarkdown] {
		t.Errorf("expected PlanFileKindMarkdown in results, got kinds: %v", kinds)
	}
	if !kinds[domain.PlanFileKindCursor] {
		t.Errorf("expected PlanFileKindCursor in results, got kinds: %v", kinds)
	}
}

func TestScanPlanFiles_ClaudeTaskSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	tasksDir := filepath.Join(tmpDir, ".claude", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "TASK-001.md"), []byte("# Task 001\nDo something."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "TASK-002.md"), []byte("# Task 002\nDo another thing."), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.Kind == domain.PlanFileKindClaudeTask {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 claude-task results, got %d (total results: %d)", count, len(results))
	}
}

func TestScanPlanFiles_CrewAI(t *testing.T) {
	tmpDir := t.TempDir()

	crewContent := "from crewai import Agent, Task, Crew\n\nagent = Agent()"
	if err := os.WriteFile(filepath.Join(tmpDir, "crew.py"), []byte(crewContent), 0644); err != nil {
		t.Fatal(err)
	}
	nonCrewContent := "import something_else\n\ndef main(): pass"
	if err := os.WriteFile(filepath.Join(tmpDir, "agents.py"), []byte(nonCrewContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.Kind == domain.PlanFileKindCrewAI {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 crewai result (crew.py only), got %d", count)
	}
}

func TestScanPlanFiles_MCPConfig(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "mcp.json"), []byte(`{"servers":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.Kind == domain.PlanFileKindMCPConfig {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 mcp-config result, got %d", count)
	}
}

func TestScanPlanFiles_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "orchestrator.json"), []byte(`{"activePlanId":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir, tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	paths := map[string]int{}
	for _, r := range results {
		paths[r.Path]++
	}
	for p, count := range paths {
		if count > 1 {
			t.Errorf("path %s appears %d times (expected 1)", p, count)
		}
	}
}

func TestScanPlanFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty dir, got %d", len(results))
	}
}
