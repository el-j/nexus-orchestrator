package services_test

import (
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/services"
)

// planProject writes the given files (relative path to text) into a fresh project directory.
func planProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScanPlanFiles_ReadsTheKindFromTheFolderWhenTheNameAndTheFrontMatterSayNothing(t *testing.T) {
	dir := planProject(t, map[string]string{
		"plans/overview.md": "# Overview\n\nWhat we build.\n",
		"tasks/do-it.md":    "# Do it\n\nOne small change.\n",
	})
	scan := services.ScanPlanFiles(dir, nil)
	if len(scan.Plans) != 1 || scan.Plans[0].Rel != "plans/overview.md" {
		t.Fatalf("plans = %+v", scan.Plans)
	}
	if len(scan.Orphans) != 1 || scan.Orphans[0].Rel != "tasks/do-it.md" {
		t.Errorf("orphans = %+v", scan.Orphans)
	}
}

func TestScanPlanFiles_ATaskListedByOnePlanButFiledUnderAnotherGetsNoPlaceholder(t *testing.T) {
	dir := planProject(t, map[string]string{
		".claude/plans/PLAN-1.md": "---\nid: PLAN-1\ntasks: [TASK-1]\n---\n# One\n",
		".claude/plans/PLAN-2.md": "---\nid: PLAN-2\n---\n# Two\n",
		".claude/tasks/TASK-1.md": "---\nid: TASK-1\nplan: PLAN-2\n---\n# Task\n",
	})
	scan := services.ScanPlanFiles(dir, nil)
	for _, p := range scan.Plans {
		switch p.ID {
		case "PLAN-1":
			if len(p.Tasks) != 0 {
				t.Errorf("PLAN-1 lists TASK-1, whose file belongs to PLAN-2: %+v", p.Tasks)
			}
		case "PLAN-2":
			if len(p.Tasks) != 1 || p.Tasks[0].ID != "TASK-1" {
				t.Errorf("PLAN-2 tasks = %+v", p.Tasks)
			}
		}
	}
}

func TestScanPlanFiles_StopsSearchingAProjectAfterTheEntryLimit(t *testing.T) {
	defer services.SetMaxPlanWalkEntries(3)()
	files := map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e", "f"} {
		files["docs/"+n+"/PLAN-"+n+".md"] = "# Plan " + n + "\n"
	}
	scan := services.ScanPlanFiles(planProject(t, files), nil)
	if len(scan.Plans) >= len(files) {
		t.Errorf("found %d of %d plans although the search should have stopped early", len(scan.Plans), len(files))
	}
}
