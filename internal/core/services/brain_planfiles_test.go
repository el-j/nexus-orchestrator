package services_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/services"
)

func writePlanFile(t *testing.T, dir, rel, text string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func onboarding(t *testing.T, dir string) string {
	t.Helper()
	out, err := services.NewBrainService(&mockKnowledgeRepo{}, nil).GetOnboardingContext(bg, dir, 5000)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// An agent that does not maintain .claude/orchestrator.json still leaves plan and task files
// behind; the onboarding must find the orchestration from those.
func TestOnboarding_FindsTheActivePlanInPlanFilesWithoutOrchestratorJSON(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, ".claude/plans/PLAN-001.md", "---\nid: PLAN-001\ntitle: Setup\nstatus: done\n---\n")
	writePlanFile(t, dir, ".claude/plans/PLAN-002.md", "---\nid: PLAN-002\ntitle: 'Quality audit'\nstatus: in-progress\n---\n")
	writePlanFile(t, dir, ".claude/tasks/TASK-001.md", "---\nstatus: done\n---\n# TASK-001 — a\n")
	writePlanFile(t, dir, ".claude/tasks/TASK-002.md", "# TASK-002 — b\n\n**Status:** In Progress\n")
	writePlanFile(t, dir, ".claude/tasks/TASK-003.md", "# TASK-003 — c\n")
	writePlanFile(t, dir, ".claude/tasks/notes.txt", "not a task")

	out := onboarding(t, dir)
	for _, want := range []string{"**Active Plan:** PLAN-002 — Quality audit", "3 task files (1 done, 1 in progress, 1 to do)"} {
		if !strings.Contains(out, want) {
			t.Errorf("onboarding lacks %q:\n%s", want, out)
		}
	}
}

func TestOnboarding_OrchestratorJSONStillWinsAndBrokenOnesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, ".claude/orchestrator.json", `{"activePlanId":"PLAN-009"}`)
	writePlanFile(t, dir, ".claude/plans/PLAN-002.md", "---\nstatus: active\n---\n")
	if out := onboarding(t, dir); !strings.Contains(out, "**Active Plan:** PLAN-009") {
		t.Errorf("the named plan must win:\n%s", out)
	}

	broken := t.TempDir()
	writePlanFile(t, broken, ".claude/orchestrator.json", "{not json")
	writePlanFile(t, broken, ".nexus/plans/PLAN-004.md", "# Launch\n")
	if out := onboarding(t, broken); !strings.Contains(out, "**Active Plan:** PLAN-004 — Launch") {
		t.Errorf("an unreadable JSON must not hide the plan files:\n%s", out)
	}
}

func TestOnboarding_NothingFinishedMeansNoActivePlan(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, ".claude/plans/PLAN-001.md", "---\nstatus: completed\n---\n")
	out := onboarding(t, dir)
	if !strings.Contains(out, "**Active Plan:** None") || strings.Contains(out, "task files") {
		t.Errorf("out = %s", out)
	}
}

func TestOnboarding_ReadsOddlyWrittenPlanFiles(t *testing.T) {
	dir := t.TempDir()
	// no number in the name, CRLF front matter, an unterminated header, a duplicate under .nexus,
	// a directory and a file that cannot be read.
	writePlanFile(t, dir, ".claude/plans/quality-audit.md", "---\r\nid: audit\r\nstatus: active\r\n---\r\n# x\r\n")
	writePlanFile(t, dir, ".claude/plans/PLAN-007.md", "---\nstatus: todo\nnever closed\n")
	writePlanFile(t, dir, ".nexus/plans/PLAN-007.md", "# duplicate, ignored\n")
	writePlanFile(t, dir, ".claude/plans/sub/ignored.md", "# in a directory\n")
	writePlanFile(t, dir, ".claude/plans/PLAN-008.md", "# unreadable\n")
	if err := os.Chmod(filepath.Join(dir, ".claude", "plans", "PLAN-008.md"), 0); err != nil {
		t.Fatal(err)
	}
	out := onboarding(t, dir)
	if !strings.Contains(out, "**Active Plan:** audit — x") {
		t.Errorf("an active plan with no number in its name must be found:\n%s", out)
	}
}

// The agent that works on a project may not be Claude: its instructions live in GEMINI.md,
// AGENTS.md or the Copilot instructions file, and init must learn from them as from CLAUDE.md.
func TestInitProject_AutoDetectsInstructionFilesOfOtherAgents(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	dir := t.TempDir()
	para := "# Title\nA sufficiently long preamble paragraph here.\n"
	writePlanFile(t, dir, "GEMINI.md", para)
	writePlanFile(t, dir, "AGENTS.md", para)
	writePlanFile(t, dir, ".github/copilot-instructions.md", para)
	if _, err := b.InitProject(bg, dir, ""); err != nil {
		t.Fatal(err)
	}
	sources := map[string]bool{}
	for _, e := range repo.entries {
		sources[e.Source] = true
	}
	for _, want := range []string{"GEMINI.md", "AGENTS.md", "copilot-instructions.md"} {
		found := false
		for s := range sources {
			if strings.Contains(s, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s was not ingested; sources: %v", want, sources)
		}
	}
}

// The project says where its plans and tasks live; the onboarding follows.
func TestOnboarding_FollowsTheFoldersNamedInThePolicy(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, ".nexus/policy.json", `{"planFiles":{"plans":["design/specs"],"tasks":["design/work"]}}`)
	writePlanFile(t, dir, "design/specs/launch.md", "# Launch\nstatus: in-progress\n")
	writePlanFile(t, dir, "design/work/a.md", "# A\nstatus: done\n")
	writePlanFile(t, dir, "design/work/b.md", "# B\n")
	out := onboarding(t, dir)
	for _, want := range []string{"**Active Plan:** launch — Launch", "2 task files (1 done, 0 in progress, 1 to do), read from design/specs, design/work"} {
		if !strings.Contains(out, want) {
			t.Errorf("onboarding lacks %q:\n%s", want, out)
		}
	}
}

func TestScanPlanFiles_Limits(t *testing.T) {
	dir := t.TempDir()
	deep := ".claude/tasks/a/b/c/d"
	writePlanFile(t, dir, deep+"/too-deep.md", "# too deep in a task folder, and not named like a task\n")
	writePlanFile(t, dir, ".claude/tasks/a/b/deep-enough.md", "# deep enough\n")
	writePlanFile(t, dir, "x/y/z/w/v/u/t/PLAN-9.md", "# beyond the search depth\n")
	writePlanFile(t, dir, "docs/PLAN-1.md", "# found\n")
	writePlanFile(t, dir, ".hidden/PLAN-2.md", "# hidden folders are skipped\n")
	writePlanFile(t, dir, "docs/huge.md", "---\ntype: plan\n---\n"+strings.Repeat("x", 300<<10))
	got := services.ScanPlanFiles(dir, nil)
	ids := map[string]bool{}
	for _, p := range got.Plans {
		ids[p.ID] = true
		for _, tk := range p.Tasks {
			ids[tk.ID] = true
		}
	}
	for _, o := range got.Orphans {
		ids[o.ID] = true
	}
	if !ids["PLAN-1"] || !ids["deep-enough"] {
		t.Errorf("expected PLAN-1 and deep-enough, got %v", ids)
	}
	for _, not := range []string{"too-deep", "PLAN-9", "PLAN-2", "huge"} {
		if ids[not] {
			t.Errorf("%s must not be found: %v", not, ids)
		}
	}
}

func TestScanPlanFiles_ListedTaskWithoutAFileAndPlanLinksByStem(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, ".claude/plans/launch-plan.md", "---\ntasks:\n  - TASK-5\n---\n# Launch\n")
	writePlanFile(t, dir, ".claude/tasks/TASK-1.md", "---\nplan: launch-plan\n---\n# One\n")
	got := services.ScanPlanFiles(dir, nil)
	if len(got.Plans) != 1 || len(got.Plans[0].Tasks) != 2 || got.Plans[0].Tasks[1].ID != "TASK-5" || got.Plans[0].Tasks[1].Status != "unknown" {
		t.Errorf("plans = %+v", got.Plans)
	}
}
