package sys_scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestScanPlanFiles_NexusSummary(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `{
		"counters": {"nextPlanId": 6, "nextTaskId": 373},
		"plans": {
			"PLAN-001": {"status": "completed"},
			"PLAN-002": {"status": "done"},
			"PLAN-003": {"status": "in-progress"}
		},
		"updatedAt": "2026-03-28"
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "orchestrator.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	var nexus *domain.DiscoveredPlanFile
	for i := range results {
		if results[i].Kind == domain.PlanFileKindNexus {
			nexus = &results[i]
			break
		}
	}
	if nexus == nil {
		t.Fatal("expected a nexus result, got none")
	}
	if !strings.Contains(nexus.Summary, "plans:") {
		t.Errorf("summary missing 'plans:': %q", nexus.Summary)
	}
	if !strings.Contains(nexus.Summary, "tasks:") {
		t.Errorf("summary missing 'tasks:': %q", nexus.Summary)
	}
	// 3 plans total, 2 completed (completed + done)
	if !strings.Contains(nexus.Summary, "3 (2 completed)") {
		t.Errorf("summary has wrong plan counts: %q", nexus.Summary)
	}
	// nextTaskId=373 → totalTasks=372
	if !strings.Contains(nexus.Summary, "tasks: 372") {
		t.Errorf("summary has wrong task count: %q", nexus.Summary)
	}
}

func TestScanPlanFiles_ExtendedDiscoveryPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".windsurfrules"), []byte("rules"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "tasks.json"), []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "tasks.yaml"), []byte("version: 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "agent.yaml"), []byte("name: test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".aider.conf.yml"), []byte("model: sonnet"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "CONVENTIONS.md"), []byte("# Conventions"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".continue"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".continue", "config.json"), []byte(`{"models":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".continue", "config.yaml"), []byte("models: []"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".github", "copilot-instructions.md"), []byte("# Copilot"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	foundByPath := map[string]domain.DiscoveredPlanFile{}
	for _, r := range results {
		foundByPath[r.Path] = r
	}

	mustFind := []string{
		filepath.Join(tmpDir, ".windsurfrules"),
		filepath.Join(tmpDir, "tasks.json"),
		filepath.Join(tmpDir, "tasks.yaml"),
		filepath.Join(tmpDir, "agent.yaml"),
		filepath.Join(tmpDir, ".aider.conf.yml"),
		filepath.Join(tmpDir, "CONVENTIONS.md"),
		filepath.Join(tmpDir, ".continue", "config.json"),
		filepath.Join(tmpDir, ".continue", "config.yaml"),
		filepath.Join(tmpDir, ".github", "copilot-instructions.md"),
	}
	for _, path := range mustFind {
		if _, ok := foundByPath[path]; !ok {
			t.Fatalf("expected %s to be discovered", path)
		}
	}
}

func TestScanPlanFiles_RecursiveInstructionPromptDepthBound(t *testing.T) {
	tmpDir := t.TempDir()

	depth3 := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(depth3, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depth3, "agent.instructions.md"), []byte("# Instructions"), 0644); err != nil {
		t.Fatal(err)
	}

	depth4 := filepath.Join(tmpDir, "d", "e", "f", "g")
	if err := os.MkdirAll(depth4, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depth4, "deep.prompt.md"), []byte("# Too Deep"), 0644); err != nil {
		t.Fatal(err)
	}

	githubDepth2 := filepath.Join(tmpDir, ".github", "prompts")
	if err := os.MkdirAll(githubDepth2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(githubDepth2, "review.prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	paths := map[string]bool{}
	for _, r := range results {
		paths[r.Path] = true
	}

	if !paths[filepath.Join(depth3, "agent.instructions.md")] {
		t.Fatalf("expected depth<=3 instructions file to be discovered")
	}
	if !paths[filepath.Join(githubDepth2, "review.prompt.md")] {
		t.Fatalf("expected .github recursive prompt file to be discovered")
	}
	if paths[filepath.Join(depth4, "deep.prompt.md")] {
		t.Fatalf("expected depth>3 prompt file to be excluded")
	}
}

func TestScanPlanFiles_MarkdownHeuristicSummaryAndKind(t *testing.T) {
	tmpDir := t.TempDir()

	content := `---
id: TASK-999
title: Heuristic Test
---
# Plan
- [ ] item one
- [x] item two
## Steps
`
	path := filepath.Join(tmpDir, "workflow-notes.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.ScanPlanFiles(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("ScanPlanFiles returned error: %v", err)
	}

	var got *domain.DiscoveredPlanFile
	for i := range results {
		if results[i].Path == path {
			got = &results[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("expected heuristic markdown file to be discovered")
	}
	if got.Kind != domain.PlanFileKindMarkdown {
		t.Fatalf("expected kind markdown, got %s", got.Kind)
	}
	if !strings.Contains(got.Summary, "frontmatter") {
		t.Fatalf("expected summary to include frontmatter heuristic, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "checklist") {
		t.Fatalf("expected summary to include checklist heuristic, got %q", got.Summary)
	}
}
