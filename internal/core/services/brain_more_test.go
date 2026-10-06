package services_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// failingKnowledgeRepo wraps mockKnowledgeRepo and fails the named methods.
type failingKnowledgeRepo struct {
	mockKnowledgeRepo
	fail map[string]bool
}

func (f *failingKnowledgeRepo) SaveKnowledge(ctx context.Context, k domain.ProjectKnowledge) error {
	if f.fail["Save"] {
		return errInjected
	}
	return f.mockKnowledgeRepo.SaveKnowledge(ctx, k)
}

func (f *failingKnowledgeRepo) GetByProject(ctx context.Context, p string) ([]domain.ProjectKnowledge, error) {
	if f.fail["GetByProject"] {
		return nil, errInjected
	}
	return f.mockKnowledgeRepo.GetByProject(ctx, p)
}

func (f *failingKnowledgeRepo) GetByProjectAndKind(ctx context.Context, p string, k domain.KnowledgeKind) ([]domain.ProjectKnowledge, error) {
	if f.fail["GetByProjectAndKind"] {
		return nil, errInjected
	}
	return f.mockKnowledgeRepo.GetByProjectAndKind(ctx, p, k)
}

func (f *failingKnowledgeRepo) SearchFTS(ctx context.Context, p, q string, n int) ([]domain.ProjectKnowledge, error) {
	if f.fail["SearchFTS"] {
		return nil, errInjected
	}
	return f.mockKnowledgeRepo.SearchFTS(ctx, p, q, n)
}

func writeMD(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func topics(entries []domain.ProjectKnowledge) map[string]domain.KnowledgeKind {
	out := map[string]domain.KnowledgeKind{}
	for _, e := range entries {
		out[e.Topic] = e.Kind
	}
	return out
}

func TestIngestFromFile_FileStartingWithSectionHeadingKeepsItsHeading(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	md := writeMD(t, "## Architecture\nThe system uses a hexagonal layout with ports.\n\n## Conventions\nAlways wrap errors with context using fmt.Errorf.\n")

	n, err := b.IngestFromFile(bg, "/proj", md)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got := topics(repo.entries)
	if got["Architecture"] != domain.KnowledgeArchitecture || got["Conventions"] != domain.KnowledgeConvention {
		t.Errorf("the first heading was lost or misclassified: %v", got)
	}
	if _, bogus := got["Overview"]; bogus {
		t.Errorf("first section must not be stored as the generic 'Overview': %v", got)
	}
}

func TestIngestFromFile_PreambleSectionsAndShortContent(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	md := writeMD(t, "# Title\nThis project does many interesting things worth knowing.\n"+
		"## File Structure\ncmd/ internal/ docs/ and the other top-level directories\n"+
		"## Lessons Learned\nAvoid global state because tests become order dependent\n"+
		"## Glossary\nTerm one means this and term two means that\n"+
		"## Tiny\nshort\n"+
		"## NoBody\n")
	n, err := b.IngestFromFile(bg, "/proj", md)
	if err != nil || n != 4 {
		t.Fatalf("n=%d err=%v (sections under 20 chars and body-less headings are skipped)", n, err)
	}
	got := topics(repo.entries)
	want := map[string]domain.KnowledgeKind{
		"Overview":        domain.KnowledgeConvention,
		"File Structure":  domain.KnowledgeFileMap,
		"Lessons Learned": domain.KnowledgeLearning,
		"Glossary":        domain.KnowledgeGlossary,
	}
	for topic, kind := range want {
		if got[topic] != kind {
			t.Errorf("topic %q: kind %q, want %q (all: %v)", topic, got[topic], kind, got)
		}
	}
}

func TestIngestFromFile_TitleOnlyOrTrivialPreambleIsSkipped(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	if n, err := b.IngestFromFile(bg, "/p", writeMD(t, "# Only a title\n")); err != nil || n != 0 {
		t.Errorf("title only: n=%d err=%v", n, err)
	}
	if n, err := b.IngestFromFile(bg, "/p", writeMD(t, "# T\ntiny\n")); err != nil || n != 0 {
		t.Errorf("trivial preamble: n=%d err=%v", n, err)
	}
}

func TestIngestFromFile_StorageFailures(t *testing.T) {
	failing := &failingKnowledgeRepo{fail: map[string]bool{"Save": true}}
	b := services.NewBrainService(failing, nil)
	md := writeMD(t, "# T\nA sufficiently long preamble paragraph here.\n## A\nSection content that is long enough to keep.\n")
	if n, err := b.IngestFromFile(bg, "/p", md); err == nil || n != 0 || !strings.Contains(err.Error(), "all 2 sections failed") {
		t.Errorf("every save failing must be an error: n=%d err=%v", n, err)
	}
}

func TestClassifySection_ViaIngestedKinds(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	body := "# T\n"
	headings := map[string]domain.KnowledgeKind{
		"System Design": domain.KnowledgeArchitecture, "Directory Layout": domain.KnowledgeFileMap,
		"Pitfalls": domain.KnowledgeLearning, "Vocabulary": domain.KnowledgeGlossary,
		"Code Style": domain.KnowledgeConvention, "Something Else Entirely": domain.KnowledgeConvention,
	}
	for h := range headings {
		body += "## " + h + "\nThis body is comfortably longer than twenty characters.\n"
	}
	if _, err := b.IngestFromFile(bg, "/p", writeMD(t, body)); err != nil {
		t.Fatal(err)
	}
	got := topics(repo.entries)
	for h, kind := range headings {
		if got[h] != kind {
			t.Errorf("heading %q classified %q, want %q", h, got[h], kind)
		}
	}
}

func TestGetContext_FocusFilterBudgetAndSuggestedFiles(t *testing.T) {
	repo := &mockKnowledgeRepo{entries: []domain.ProjectKnowledge{
		{Kind: domain.KnowledgeArchitecture, Topic: "Arch", Content: "A", TokenCount: 10, RelevanceScore: 0.9, ProjectPath: "/p"},
		{Kind: domain.KnowledgeConvention, Topic: "auth rules", Content: "C1", TokenCount: 10, RelevanceScore: 0.5, ProjectPath: "/p"},
		{Kind: domain.KnowledgeConvention, Topic: "db rules", Content: "C2", TokenCount: 10, RelevanceScore: 0.9, ProjectPath: "/p"},
		{Kind: domain.KnowledgeFileMap, Topic: "auth files", Content: "auth.go", TokenCount: 5, RelevanceScore: 0.9, ProjectPath: "/p"},
		{Kind: domain.KnowledgeLearning, Topic: "learn", Content: "L", TokenCount: 10, RelevanceScore: 0.9, ProjectPath: "/p"},
	}}
	b := services.NewBrainService(repo, nil)

	resp, err := b.GetContext(bg, domain.ContextQuery{ProjectPath: "/p", FocusArea: "AUTH", MaxTokens: 100})
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, s := range resp.Sections {
		seen = append(seen, s.Topic)
	}
	if strings.Join(seen, ",") != "Arch,auth rules,auth files,learn" {
		t.Errorf("focus must filter conventions/file maps only (kept, ordered by kind priority): %v", seen)
	}
	if len(resp.SuggestedFiles) != 1 || resp.SuggestedFiles[0] != "auth.go" || resp.Truncated {
		t.Errorf("resp = %+v", resp)
	}

	// "all" disables the filter; the default budget is 400.
	resp, _ = b.GetContext(bg, domain.ContextQuery{ProjectPath: "/p", FocusArea: "all"})
	if len(resp.Sections) != 5 || resp.TokenBudget != 400 {
		t.Errorf("focus=all: %+v", resp)
	}

	// A tight budget truncates and stops at the first entry that does not fit.
	resp, _ = b.GetContext(bg, domain.ContextQuery{ProjectPath: "/p", MaxTokens: 15})
	if !resp.Truncated || resp.TokensUsed > 15 {
		t.Errorf("budget not enforced: %+v", resp)
	}

	failing := &failingKnowledgeRepo{fail: map[string]bool{"GetByProject": true}}
	if _, err := services.NewBrainService(failing, nil).GetContext(bg, domain.ContextQuery{ProjectPath: "/p"}); err == nil {
		t.Error("repo failure must be returned")
	}
}

func TestGetFocusedContext_BudgetAndErrors(t *testing.T) {
	repo := &mockKnowledgeRepo{entries: []domain.ProjectKnowledge{
		{Topic: "one", TokenCount: 30, ProjectPath: "/p"},
		{Topic: "two", TokenCount: 30, ProjectPath: "/p"},
	}}
	b := services.NewBrainService(repo, nil)
	resp, err := b.GetFocusedContext(bg, domain.ContextQuery{ProjectPath: "/p", Question: "q", MaxTokens: 40})
	if err != nil || len(resp.Sections) != 1 || !resp.Truncated || resp.TokensUsed != 30 {
		t.Errorf("resp=%+v err=%v", resp, err)
	}
	failing := &failingKnowledgeRepo{fail: map[string]bool{"SearchFTS": true}}
	if _, err := services.NewBrainService(failing, nil).GetFocusedContext(bg, domain.ContextQuery{ProjectPath: "/p"}); err == nil {
		t.Error("search failure must be returned")
	}
}

func TestSearchKnowledge_ItemLimitVersusTokenBudget(t *testing.T) {
	var entries []domain.ProjectKnowledge
	for _, name := range []string{"a", "b", "c", "d"} {
		entries = append(entries, domain.ProjectKnowledge{Topic: name, TokenCount: 100, ProjectPath: "/p"})
	}
	b := services.NewBrainService(&mockKnowledgeRepo{entries: entries}, nil)

	// A small number (<=20) is an item limit, not a token budget.
	if got, _ := b.SearchKnowledge(bg, "/p", "q", 2); len(got) != 2 {
		t.Errorf("item limit 2: got %d", len(got))
	}
	// Default budget (400 tokens) fits all four 100-token entries.
	if got, _ := b.SearchKnowledge(bg, "/p", "q", 0); len(got) != 4 {
		t.Errorf("default budget: got %d", len(got))
	}
	// An explicit token budget cuts the result.
	if got, _ := b.SearchKnowledge(bg, "/p", "q", 250); len(got) != 2 {
		t.Errorf("250-token budget: got %d", len(got))
	}
	// Negative budgets fall back to the default.
	if got, _ := b.SearchKnowledge(bg, "/p", "q", -1); len(got) != 4 {
		t.Errorf("negative budget: got %d", len(got))
	}
	failing := &failingKnowledgeRepo{fail: map[string]bool{"SearchFTS": true}}
	if _, err := services.NewBrainService(failing, nil).SearchKnowledge(bg, "/p", "q", 5); err == nil {
		t.Error("search failure must be returned")
	}
}

func TestBrain_SimpleDelegationsAndErrors(t *testing.T) {
	failing := &failingKnowledgeRepo{fail: map[string]bool{"GetByProjectAndKind": true, "Save": true, "GetByProject": true}}
	b := services.NewBrainService(failing, nil)
	if _, err := b.GetFileMap(bg, "/p", ""); err == nil {
		t.Error("GetFileMap must surface repo errors")
	}
	if _, err := b.IngestKnowledge(bg, domain.ProjectKnowledge{ProjectPath: "/p", Content: "x"}); err == nil {
		t.Error("IngestKnowledge must surface repo errors")
	}
	if _, err := b.ListKnowledge(bg, "/p", "learning"); err == nil {
		t.Error("ListKnowledge(kind) must surface repo errors")
	}
	if _, err := b.ListKnowledge(bg, "/p", ""); err == nil {
		t.Error("ListKnowledge(all) must surface repo errors")
	}
	if err := b.DeleteKnowledge(bg, "missing"); err == nil {
		t.Error("deleting an unknown id must fail")
	}

	// IngestKnowledge defaults: id, score, timestamps, token estimate (min 1).
	ok := services.NewBrainService(&mockKnowledgeRepo{}, nil)
	got, err := ok.IngestKnowledge(bg, domain.ProjectKnowledge{ProjectPath: "/p/", Content: "x"})
	if err != nil || got.ID == "" || got.RelevanceScore != 0.5 || got.TokenCount != 1 ||
		got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() || got.ProjectPath != "/p" {
		t.Errorf("defaults not applied: %+v %v", got, err)
	}
}

func TestInitProject_ExplicitPathAndErrors(t *testing.T) {
	repo := &mockKnowledgeRepo{status: domain.BrainStatus{ProjectPath: "/p", EntryCount: 1}}
	b := services.NewBrainService(repo, nil)
	md := writeMD(t, "# T\nA sufficiently long preamble paragraph here.\n")
	st, err := b.InitProject(bg, t.TempDir(), md)
	if err != nil || st.EntryCount != 1 || len(repo.entries) != 1 {
		t.Errorf("explicit path: %+v %v", st, err)
	}
	// An unreadable explicit CLAUDE.md is an error.
	if _, err := b.InitProject(bg, t.TempDir(), "/no/such/CLAUDE.md"); err == nil {
		t.Error("missing explicit file must fail")
	}
	// No CLAUDE.md anywhere: just returns the status.
	if st, err := b.InitProject(bg, t.TempDir(), ""); err != nil || st.EntryCount != 1 {
		t.Errorf("no file: %+v %v", st, err)
	}
	// .claude/CLAUDE.md is auto-detected.
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".claude"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, ".claude", "CLAUDE.md"), []byte("# T\nA sufficiently long preamble paragraph here.\n"), 0o600)
	before := len(repo.entries)
	if _, err := b.InitProject(bg, dir, ""); err != nil || len(repo.entries) != before+1 {
		t.Errorf(".claude/CLAUDE.md not auto-detected: %v", err)
	}
}

func TestGetOnboardingContext_StackPlanAndTaskSummary(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"go.mod", "package.json", "Cargo.toml", "requirements.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.MkdirAll(filepath.Join(dir, ".claude"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, ".claude", "orchestrator.json"), []byte(`{"activePlanId":"PLAN-099"}`), 0o600)

	tasks := newMemRepo()
	for i, st := range []domain.TaskStatus{domain.StatusQueued, domain.StatusProcessing, domain.StatusProcessing, domain.StatusCompleted, domain.StatusFailed} {
		_ = tasks.Save(domain.Task{ID: string(rune('a' + i)), ProjectPath: dir, Instruction: "work", Status: st})
	}
	repo := &mockKnowledgeRepo{}
	var many []domain.ProjectKnowledge
	for i := 0; i < 7; i++ {
		many = append(many, domain.ProjectKnowledge{Kind: domain.KnowledgeConvention, Topic: "rule" + string(rune('A'+i)), Content: "do the thing", ProjectPath: dir})
	}
	repo.entries = many
	b := services.NewBrainService(repo, tasks)

	out, err := b.GetOnboardingContext(bg, dir, 5000)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Go, TypeScript/JavaScript (Node), Rust, Python", "PLAN-099",
		"1 queued, 2 processing, 1 completed, 1 failed", "Current Task:",
		"go test -race", "npm test", "cargo test", "pytest",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("onboarding missing %q:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "- **rule"); n != 5 {
		t.Errorf("only the top 5 conventions are listed, got %d", n)
	}

	// pyproject.toml also counts as Python; an unreadable orchestrator.json is ignored.
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "pyproject.toml"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(dir2, ".claude"), 0o750)
	_ = os.WriteFile(filepath.Join(dir2, ".claude", "orchestrator.json"), []byte("{not json"), 0o600)
	out, _ = services.NewBrainService(&mockKnowledgeRepo{}, nil).GetOnboardingContext(bg, dir2, 0)
	if !strings.Contains(out, "**Stack:** Python") || !strings.Contains(out, "Active Plan:** None") {
		t.Errorf("out = %s", out)
	}
}

func TestGetOnboardingContext_TruncationNeverSplitsMultibyteCharacters(t *testing.T) {
	dir := t.TempDir()
	// Long multi-byte (3 bytes/char) content in the architecture section.
	long := strings.Repeat("架", 400)
	repo := &mockKnowledgeRepo{entries: []domain.ProjectKnowledge{
		{Kind: domain.KnowledgeArchitecture, Topic: "t", Content: long, ProjectPath: dir},
		{Kind: domain.KnowledgeConvention, Topic: "c", Content: long, ProjectPath: dir},
	}}
	b := services.NewBrainService(repo, nil)

	full, _ := b.GetOnboardingContext(bg, dir, 5000)
	if !utf8.ValidString(full) {
		t.Fatalf("per-entry truncation produced invalid UTF-8")
	}
	if !strings.Contains(full, "...") {
		t.Error("over-long entries must be marked as truncated")
	}
	// Try many budgets so at least some land inside a 3-byte character.
	for tokens := 40; tokens < 80; tokens++ {
		out, _ := b.GetOnboardingContext(bg, dir, tokens)
		if !utf8.ValidString(out) {
			t.Fatalf("budget %d produced invalid UTF-8", tokens)
		}
	}
}
