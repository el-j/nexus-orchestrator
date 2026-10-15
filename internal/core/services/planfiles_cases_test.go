package services_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type planCase struct {
	Name   string                  `json:"name"`
	Config *domain.PlanFilesPolicy `json:"config"`
	Files  map[string]string       `json:"files"`
	Expect struct {
		Plans []struct {
			ID     string   `json:"id"`
			Title  string   `json:"title"`
			Status string   `json:"status"`
			Tasks  []string `json:"tasks"`
		} `json:"plans"`
		OrphanTasks []string `json:"orphanTasks"`
		Folders     []string `json:"folders"`
	} `json:"expect"`
}

// The same cases run against the VS Code extension (vscode-extension/src/planFiles.cases.test.ts):
// both must find the same plans, tasks and links in the same files.
func TestScanPlanFiles_SharedCases(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "planfiles", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []planCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := t.TempDir()
			for rel, text := range c.Files {
				p := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got := services.ScanPlanFiles(dir, c.Config)
			type plan struct {
				ID, Title, Status string
				Tasks             []string
			}
			var gotPlans, wantPlans []plan
			for _, n := range got.Plans {
				p := plan{n.ID, n.Title, n.Status, []string{}}
				for _, tk := range n.Tasks {
					p.Tasks = append(p.Tasks, tk.ID)
				}
				gotPlans = append(gotPlans, p)
			}
			for _, w := range c.Expect.Plans {
				wantPlans = append(wantPlans, plan{w.ID, w.Title, w.Status, append([]string{}, w.Tasks...)})
			}
			if !reflect.DeepEqual(gotPlans, wantPlans) {
				t.Errorf("plans:\n got %+v\nwant %+v", gotPlans, wantPlans)
			}
			orphans := []string{}
			for _, o := range got.Orphans {
				orphans = append(orphans, o.ID)
			}
			if !reflect.DeepEqual(orphans, append([]string{}, c.Expect.OrphanTasks...)) {
				t.Errorf("orphan tasks: got %v want %v", orphans, c.Expect.OrphanTasks)
			}
			if folders := append([]string{}, got.Folders...); !reflect.DeepEqual(folders, append([]string{}, c.Expect.Folders...)) {
				t.Errorf("folders: got %v want %v", folders, c.Expect.Folders)
			}
		})
	}
}
