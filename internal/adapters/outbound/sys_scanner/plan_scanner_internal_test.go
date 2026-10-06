package sys_scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"nexus-orchestrator/internal/core/domain"
)

// planTree builds a project exercising every recognised plan-file pattern.
func planTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	w := func(rel, content string) { writeFile(t, filepath.Join(root, rel), content) }
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	w("CLAUDE.md", "# Claude\nrules")
	w("AGENTS.md", "# Agents")
	w("TASKS.md", "- [ ] one\n- [x] two\n")
	w("CONVENTIONS.md", "# Conv")
	w(".cursorrules", "rules")
	w(".windsurfrules", "rules")
	w(".aider.conf.yml", "model: x")
	w("mcp.json", `{"mcpServers":{}}`)
	w("tasks.json", `[]`)
	w("agent.yaml", "name: x")
	w("crew.py", "from crewai import Agent\n")
	w("agents.py", "print('not a crew file')\n")
	w("notes.md", "just some prose without structure")
	w("ideas.md", "---\ntitle: t\n---\nbody")
	w("sprint.md", "# Sprint 1\nship it")
	w("steps.md", "1. first\n2. second\n3. third\n")
	w("many-headings.md", "# a\n# b\n# c\n# d\n# e\n")
	w(".claude/orchestrator.json", `{"counters":{"nextTaskId":4},"plans":{"p1":{"status":"completed"},"p2":{"status":"active"},"p3":{"status":"done"}},"updatedAt":"2026-01-01"}`)
	w(".claude/tasks/T-1.md", "# task")
	w(".claude/tasks/ignore.txt", "not md")
	w(".cursor/rules/a.mdc", "rule a")
	w(".cursor/rules/b.txt", "not mdc")
	w(".continue/config.json", `{}`)
	w(".continue/config.yaml", "x: 1")
	w(".github/copilot-instructions.md", "# Copilot")
	w(".github/agents/review.agent.md", "# Reviewer")
	w(".github/agents/AGENTS.md", "# nested agents")
	w("docs/guide.instructions.md", "# guide")
	w("docs/deep/er/est/too-deep.prompt.md", "# deep")
	w("node_modules/pkg/x.prompt.md", "# skipped")
	w("vendor/pkg/y.instructions.md", "# skipped")
	return root
}

func TestScanPlanFiles_ClassifiesEveryKnownPattern(t *testing.T) {
	root := planTree(t)
	got, err := newTestScanner().ScanPlanFiles(context.Background(), []string{root, root, "/definitely/not/here"})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]domain.DiscoveredPlanFile{}
	for _, f := range got {
		rel, _ := filepath.Rel(root, f.Path)
		if _, dup := byName[rel]; dup {
			t.Errorf("%s reported twice (the same root was scanned twice)", rel)
		}
		byName[rel] = f
	}
	want := map[string]domain.PlanFileKind{
		"CLAUDE.md": domain.PlanFileKindClaude, "AGENTS.md": domain.PlanFileKindMarkdown,
		"TASKS.md": domain.PlanFileKindMarkdown, "CONVENTIONS.md": domain.PlanFileKindMarkdown,
		".cursorrules": domain.PlanFileKindCursor, ".windsurfrules": domain.PlanFileKindCursor,
		".aider.conf.yml": domain.PlanFileKindMCPConfig, "mcp.json": domain.PlanFileKindMCPConfig,
		"tasks.json": domain.PlanFileKindMarkdown, "agent.yaml": domain.PlanFileKindMarkdown,
		"crew.py": domain.PlanFileKindCrewAI, "ideas.md": domain.PlanFileKindMarkdown,
		"sprint.md": domain.PlanFileKindMarkdown, "steps.md": domain.PlanFileKindMarkdown,
		"many-headings.md":                domain.PlanFileKindMarkdown,
		".claude/orchestrator.json":       domain.PlanFileKindNexus,
		".claude/tasks/T-1.md":            domain.PlanFileKindClaudeTask,
		".cursor/rules/a.mdc":             domain.PlanFileKindCursor,
		".continue/config.json":           domain.PlanFileKindMCPConfig,
		".continue/config.yaml":           domain.PlanFileKindMCPConfig,
		".github/copilot-instructions.md": domain.PlanFileKindClaude,
		".github/agents/review.agent.md":  domain.PlanFileKindClaude,
		".github/agents/AGENTS.md":        domain.PlanFileKindClaude,
		"docs/guide.instructions.md":      domain.PlanFileKindMarkdown,
	}
	for rel, kind := range want {
		f, ok := byName[rel]
		if !ok {
			t.Errorf("%s was not discovered", rel)
			continue
		}
		if f.Kind != kind {
			t.Errorf("%s: kind %q, want %q", rel, f.Kind, kind)
		}
		if f.ProjectPath != root {
			t.Errorf("%s: project path %q, want the git root %q", rel, f.ProjectPath, root)
		}
		if len(f.ID) != 12 || !f.IsActive {
			t.Errorf("%s: id=%q active=%v", rel, f.ID, f.IsActive)
		}
	}
	for _, rel := range []string{"notes.md", "agents.py", ".claude/tasks/ignore.txt", ".cursor/rules/b.txt",
		"docs/deep/er/est/too-deep.prompt.md", "node_modules/pkg/x.prompt.md", "vendor/pkg/y.instructions.md"} {
		if _, bad := byName[rel]; bad {
			t.Errorf("%s must not be reported", rel)
		}
	}
	if s := byName[".claude/orchestrator.json"].Summary; !strings.Contains(s, "plans: 3 (2 completed)") || !strings.Contains(s, "tasks: 3") {
		t.Errorf("orchestrator summary = %q", s)
	}
}

func TestSummarizeOrchestratorJSON_EdgeCases(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		writeFile(t, p, body)
		return p
	}
	if got := summarizeOrchestratorJSON(filepath.Join(dir, "missing.json")); got != "" {
		t.Errorf("missing: %q", got)
	}
	if got := summarizeOrchestratorJSON(write("bad.json", "{")); got != "" {
		t.Errorf("invalid: %q", got)
	}
	if got := summarizeOrchestratorJSON(write("empty.json", `{"counters":{"nextTaskId":0}}`)); !strings.Contains(got, "tasks: 0") {
		t.Errorf("negative task count must clamp to 0: %q", got)
	}
}

func TestReadSummary_NeverSplitsMultibyteCharacters(t *testing.T) {
	dir := t.TempDir()
	// 3-byte characters: the 300-byte read window and the 200-byte cut both land mid-character for some offsets.
	for pad := 0; pad < 3; pad++ {
		body := strings.Repeat("a", pad) + strings.Repeat("架", 200)
		p := filepath.Join(dir, "doc.txt")
		writeFile(t, p, body)
		got := readSummary(p)
		if !utf8.ValidString(got) || strings.ContainsRune(got, utf8.RuneError) {
			t.Fatalf("pad %d: summary contains a broken character: %q", pad, got)
		}
		if len(got) > 200 {
			t.Errorf("pad %d: summary is %d bytes, want <= 200", pad, len(got))
		}
	}
	md := filepath.Join(dir, "plan.md")
	writeFile(t, md, "---\nt: 1\n---\n# 架架架架\n## 架架架架\n- [ ] 架\n- [x] 架\n"+strings.Repeat("架", 150))
	got := readSummary(md)
	if !utf8.ValidString(got) || len(got) > 200 || !strings.HasPrefix(got, "[frontmatter, checklist, headings] ") {
		t.Errorf("markdown summary = %q (len %d)", got, len(got))
	}
	if readSummary(filepath.Join(dir, "missing")) != "" {
		t.Error("an unreadable file has no summary")
	}
	// Non-printable control characters are stripped.
	ctl := filepath.Join(dir, "ctl.txt")
	writeFile(t, ctl, "a\x00b\x07c\n\td")
	if got := readSummary(ctl); got != "abc\n\td" {
		t.Errorf("control chars: %q", got)
	}
}

func TestFindProjectPath(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := findProjectPath(deep, root); got != deep {
		t.Errorf("no .git: want the starting dir %q, got %q", deep, got)
	}
	if err := os.MkdirAll(filepath.Join(root, "a", ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if got := findProjectPath(deep, root); got != filepath.Join(root, "a") {
		t.Errorf("want the nearest git root, got %q", got)
	}
	if got := findProjectPath(deep, ""); got != filepath.Join(root, "a") {
		t.Errorf("empty home must not break the walk: %q", got)
	}
}

func TestPlanScan_TolerantOfUnreadableEntries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "TASKS.md"), "- [ ] a\n- [ ] b\n")
	if err := os.Symlink(filepath.Join(root, "nope"), filepath.Join(root, "PLAN.md")); err == nil {
		// A dangling symlink named like a plan file is skipped, not fatal.
		got, err := newTestScanner().ScanPlanFiles(context.Background(), []string{root})
		if err != nil || len(got) != 1 || filepath.Base(got[0].Path) != "TASKS.md" {
			t.Errorf("got %v %v", got, err)
		}
	}
	if _, err := scanDir(filepath.Join(root, "missing"), "", map[string]bool{}); err == nil {
		t.Error("scanDir on a missing dir must report the error")
	}
	if recs, err := scanRecursiveInstructionFiles(filepath.Join(root, "missing"), "", map[string]bool{}, 3); err != nil || recs != nil {
		t.Errorf("missing base: %v %v", recs, err)
	}
	// Files already seen are not reported twice.
	seen := map[string]bool{filepath.Join(root, "TASKS.md"): true}
	if recs, _ := scanDir(root, "", seen); len(recs) != 0 {
		t.Errorf("seen file reported again: %v", recs)
	}
}

func TestContainsCrewAI_OnlyInspectsTheHeadOfTheFile(t *testing.T) {
	dir := t.TempDir()
	late := filepath.Join(dir, "late.py")
	writeFile(t, late, strings.Repeat("x = 1\n", 25)+"import crewai\n")
	if containsCrewAI(late) {
		t.Error("an import after line 20 must not count")
	}
	if containsCrewAI(filepath.Join(dir, "missing.py")) {
		t.Error("missing file")
	}
}

func TestIsActiveReflectsRecentModification(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "TASKS.md")
	writeFile(t, p, "- [ ] a\n- [ ] b\n")
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	pf, err := buildPlanFile(p, domain.PlanFileKindMarkdown, "md", "")
	if err != nil || pf.IsActive {
		t.Errorf("a 48h-old file is not active: %+v %v", pf, err)
	}
	if _, err := buildPlanFile(filepath.Join(root, "missing"), domain.PlanFileKindMarkdown, "md", ""); err == nil {
		t.Error("missing file must error")
	}
}
