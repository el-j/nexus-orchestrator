package services_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// headerFS reads files like the real writer: a header line, the content and a newline, per file.
type headerFS struct {
	noopWriter
	mu    sync.Mutex
	files map[string]string
	reads [][]string
}

func (h *headerFS) ReadContextFiles(_ string, files []string) (string, error) {
	h.mu.Lock()
	h.reads = append(h.reads, files)
	h.mu.Unlock()
	var sb strings.Builder
	for _, f := range files {
		c, ok := h.files[f]
		if !ok {
			return "", fmt.Errorf("no such file %q", f)
		}
		fmt.Fprintf(&sb, "// --- %s ---\n%s\n", f, c)
	}
	return sb.String(), nil
}

// goFile builds a Go source file with n methods of 8 lines each.
func goFile(pkg string, n int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "package %s\n\nimport \"fmt\"\n\ntype Thing struct{ A, B int }\n\n", pkg)
	for i := range n {
		fmt.Fprintf(&sb, "// Method%d does thing %d.\nfunc (t Thing) Method%d() int {\n\tx := t.A + %d\n\tfmt.Println(x)\n\ty := x * t.B\n\treturn y + %d\n}\n\n", i, i, i, i, i)
	}
	return sb.String()
}

func budgetProject(t *testing.T, policy string) string { return ladderProject(t, policy) }

func TestContextBudget_UnderTheBudgetNothingChanges(t *testing.T) {
	fs := &headerFS{files: map[string]string{"a.go": goFile("a", 2), "b.go": goFile("b", 2)}}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, fs, nil, nil, llm)
	dir := budgetProject(t, `{"contextBudget":50000}`)
	e.runOne(e.queue(domain.Task{Instruction: "use them", ContextFiles: []string{"a.go", "b.go"}, ProjectPath: dir}), domain.StatusCompleted)
	p := llm.prompts[0]
	if !strings.Contains(p, "func (t Thing) Method1() int {\n\tx := t.A + 1") || strings.Contains(p, "outline") {
		t.Errorf("both files are sent whole:\n%s", p)
	}
}

func TestContextBudget_TheLargestFileIsSentAsAnOutlineWhenOverTheBudget(t *testing.T) {
	big, small := goFile("big", 60), goFile("small", 1)
	fs := &headerFS{files: map[string]string{"big.go": big, "small.go": small}}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, fs, nil, nil, llm)
	dir := budgetProject(t, `{"contextBudget":1200}`)
	e.runOne(e.queue(domain.Task{Instruction: "use them", ContextFiles: []string{"big.go", "small.go"}, ProjectPath: dir}), domain.StatusCompleted)
	p := llm.prompts[0]
	if !strings.Contains(p, "// --- big.go (outline") {
		t.Fatalf("the big file must be an outline:\n%s", p[:min(len(p), 600)])
	}
	if !strings.Contains(p, "func (t Thing) Method59() int {") || strings.Contains(p, "x := t.A + 59") {
		t.Errorf("an outline keeps the declarations and drops the bodies")
	}
	if !strings.Contains(p, "x := t.A + 0") || strings.Contains(p, "// --- small.go (outline") {
		t.Errorf("the small file stays whole")
	}
	if got := len(p) / 4; got > 1600 {
		t.Errorf("the prompt is ~%d tokens, the budget is 1200 (plus the task)", got)
	}
}

func TestContextBudget_WhenOutlinesAreNotEnoughFilesAreShortened(t *testing.T) {
	fs := &headerFS{files: map[string]string{"x.go": goFile("x", 200)}}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, fs, nil, nil, llm)
	dir := budgetProject(t, `{"contextBudget":150}`)
	e.runOne(e.queue(domain.Task{Instruction: "tiny budget", ContextFiles: []string{"x.go"}, ProjectPath: dir}), domain.StatusCompleted)
	p := llm.prompts[0]
	if !strings.Contains(p, "shortened") || len(p)/4 > 400 {
		t.Errorf("shortened to the budget (~%d tokens):\n%s", len(p)/4, p[:min(len(p), 400)])
	}
}

func TestContextBudget_NoBudgetMeansNoChangeAndUnreadableFilesStillFail(t *testing.T) {
	fs := &headerFS{files: map[string]string{"a.go": goFile("a", 80)}}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, fs, nil, nil, llm)
	e.runOne(e.queue(domain.Task{Instruction: "no policy", ContextFiles: []string{"a.go"}, ProjectPath: t.TempDir()}), domain.StatusCompleted)
	if strings.Contains(llm.prompts[0], "outline") || len(fs.reads) != 1 || len(fs.reads[0]) != 1 {
		t.Error("without a budget the context is read and sent as before")
	}

	dir := budgetProject(t, `{"contextBudget":100}`)
	e2 := newEngine(t, fs, nil, nil, llm)
	id := e2.queue(domain.Task{Instruction: "missing", ContextFiles: []string{"nope.go"}, ProjectPath: dir})
	got := e2.runOne(id, domain.StatusFailed)
	if !strings.Contains(got.Logs, "failed reading context files") {
		t.Errorf("logs: %s", got.Logs)
	}
}

func TestHistoryBudget_OldMessagesAreDroppedFirst(t *testing.T) {
	long := strings.Repeat("x", 400) // 100 tokens plus overhead
	var msgs []domain.Message
	for i := range 8 {
		msgs = append(msgs, domain.Message{Role: domain.RoleUser, Content: fmt.Sprintf("u%d %s", i, long)},
			domain.Message{Role: domain.RoleAssistant, Content: fmt.Sprintf("a%d %s", i, long)})
	}
	sessions := &fixedSessions{msgs: msgs}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, &noopWriter{}, sessions, nil, llm)
	dir := budgetProject(t, `{"historyBudget":450}`)
	e.runOne(e.queue(domain.Task{Instruction: "now", ProjectPath: dir}), domain.StatusCompleted)
	h := llm.histories[0]
	// the newest messages that fit in ~450 tokens, starting with a user turn, then the new prompt
	if len(h) < 3 || len(h) > 6 || h[0].Role != domain.RoleUser {
		t.Fatalf("history sent: %d messages, first role %s", len(h), h[0].Role)
	}
	if !strings.HasPrefix(h[len(h)-2].Content, "a7") {
		t.Errorf("the newest message is kept: %q", h[len(h)-2].Content[:4])
	}
	if strings.HasPrefix(h[0].Content, "u0") {
		t.Error("the oldest message must be gone")
	}

	// Without a budget the message cap applies as before.
	e2 := newEngine(t, &noopWriter{}, &fixedSessions{msgs: msgs}, nil, llm)
	e2.runOne(e2.queue(domain.Task{Instruction: "no budget", ProjectPath: t.TempDir()}), domain.StatusCompleted)
	if got := len(llm.histories[len(llm.histories)-1]); got != 17 {
		t.Errorf("all 16 messages (under the cap of 20) and the prompt: %d", got)
	}
}

// fixedSessions is a session store with a fixed history.
type fixedSessions struct{ msgs []domain.Message }

func (f *fixedSessions) Save(context.Context, domain.Session) error { return nil }
func (f *fixedSessions) GetByProjectPath(_ context.Context, p string) (domain.Session, error) {
	return domain.Session{ProjectPath: p, Messages: f.msgs}, nil
}
func (f *fixedSessions) AppendMessage(context.Context, string, domain.Message) error { return nil }
