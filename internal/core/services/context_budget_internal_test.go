package services

import (
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestWithNote(t *testing.T) {
	if got := withNote("// --- a.go ---", "outline"); got != "// --- a.go (outline) ---" {
		t.Errorf("%q", got)
	}
	if got := withNote("// a.go", "outline"); got != "// a.go (outline)" {
		t.Errorf("a header without the closing marker: %q", got)
	}
}

func TestIsDeclaration(t *testing.T) {
	yes := []string{"func main() {", "type T struct {", "class A:", "export const x = 1", "\tfunc (t T) M() {", "    def method(self):", "  public void run() {", "  async load() {"}
	no := []string{"", "   ", "}", ")", "// comment", "# comment", "\tx := 1", "    return y", "  }"}
	for _, l := range yes {
		if !isDeclaration(l) {
			t.Errorf("%q is a declaration", l)
		}
	}
	for _, l := range no {
		if isDeclaration(l) {
			t.Errorf("%q is not", l)
		}
	}
}

func TestShortenBlock(t *testing.T) {
	block := "// --- f ---\n" + strings.Repeat("line of text\n", 100)
	short := shortenBlock(block, 20)
	if !strings.Contains(short, "shortened") || len(short) > 200 || !strings.HasSuffix(short, "line of text\n") {
		t.Errorf("cut at a line end: %q", short)
	}
	if got := shortenBlock(block, 100000); got != block {
		t.Error("a block that fits is not touched")
	}
	oneLine := "// --- f ---\n" + strings.Repeat("x", 500)
	if got := shortenBlock(oneLine, 10); !strings.Contains(got, "shortened") || len(got) > 120 {
		t.Errorf("no newline to cut at: %q", got)
	}
}

func TestOutlineKeepsTheHeadAndDeclarations(t *testing.T) {
	block := "// --- f.go ---\n" + strings.Repeat("// head\n", 3) + "package p\n" + strings.Repeat("\tbody()\n", 30) + "func After() {\n\tx()\n}\n"
	out := outlineBlock(block)
	if !strings.Contains(out, "outline:") || !strings.Contains(out, "func After() {") || strings.Contains(out, "x()") {
		t.Errorf("%s", out)
	}
}

func TestTrimHistoryTokens(t *testing.T) {
	msgs := []domain.Message{
		{Role: domain.RoleAssistant, Content: "orphan reply"},
		{Role: domain.RoleUser, Content: "q1"},
		{Role: domain.RoleAssistant, Content: "a1"},
		{Role: domain.RoleUser, Content: "q2"},
		{Role: domain.RoleAssistant, Content: "a2"},
	}
	if got := trimHistoryTokens(msgs, 100000); len(got) != 4 || got[0].Content != "q1" {
		t.Errorf("everything fits, a leading assistant turn is dropped: %+v", got)
	}
	if got := trimHistoryTokens(msgs, 11); len(got) != 2 || got[0].Content != "q2" {
		t.Errorf("only the newest pair fits: %+v", got)
	}
	if got := trimHistoryTokens(msgs, 1); len(got) != 0 {
		t.Errorf("nothing fits: %+v", got)
	}
	if got := trimHistoryTokens(nil, 100); len(got) != 0 {
		t.Errorf("empty: %+v", got)
	}
}

func TestTrimHistoryTokens_RollingConsolidation(t *testing.T) {
	msgs := []domain.Message{
		{Role: domain.RoleUser, Content: "implement authentication middleware\nwith jwt validation"},
		{Role: domain.RoleAssistant, Content: "auth.go generated with jwt validation and token checks"},
		{Role: domain.RoleUser, Content: "now add rate limiting"},
		{Role: domain.RoleAssistant, Content: "rate limiting added"},
	}

	// Budget that fits rolling summary + newest user/assistant turn
	got := trimHistoryTokens(msgs, 50)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(got))
	}
	if !strings.Contains(got[0].Content, "[Previous Context Summary:") {
		t.Fatalf("expected summary in first message, got: %s", got[0].Content)
	}
	if !strings.Contains(got[0].Content, "implement authentication middleware") {
		t.Fatalf("expected original prompt summary, got: %s", got[0].Content)
	}

	// Test roll-forward of previous summary
	msgs2 := append(got,
		domain.Message{Role: domain.RoleUser, Content: "now add prometheus metrics"},
		domain.Message{Role: domain.RoleAssistant, Content: "metrics added"},
	)
	got2 := trimHistoryTokens(msgs2, 60)
	if len(got2) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(got2))
	}
	if !strings.Contains(got2[0].Content, "[Previous Context Summary:") {
		t.Fatalf("expected rolled-forward summary, got: %s", got2[0].Content)
	}
}

func TestDistillPrunedHistory_EdgeCases(t *testing.T) {
	if got := distillPrunedHistory(nil); got != "" {
		t.Fatalf("expected empty for nil, got %q", got)
	}
	emptyMsgs := []domain.Message{
		{Role: domain.RoleUser, Content: "  "},
		{Role: domain.RoleAssistant, Content: "\n\t"},
	}
	if got := distillPrunedHistory(emptyMsgs); got != "" {
		t.Fatalf("expected empty for blank messages, got %q", got)
	}

	// Test line truncation and > 4 parts capping
	many := []domain.Message{
		{Role: domain.RoleUser, Content: "1: " + strings.Repeat("a", 100)},
		{Role: domain.RoleAssistant, Content: "2: ok"},
		{Role: domain.RoleUser, Content: "3: ok"},
		{Role: domain.RoleAssistant, Content: "4: ok"},
		{Role: domain.RoleUser, Content: "5: ok"},
		{Role: domain.RoleAssistant, Content: "6: " + strings.Repeat("b", 100)},
	}
	distilled := distillPrunedHistory(many)
	if !strings.Contains(distilled, "...") {
		t.Fatalf("expected truncation in %s", distilled)
	}
	if strings.Contains(distilled, "1: ") {
		t.Fatalf("expected oldest part to be dropped when > 4 parts, got %s", distilled)
	}

	// Test firstNonEmptyLine with all newlines
	if got := firstNonEmptyLine("\n\n\n", 50); got != "" {
		t.Fatalf("expected empty for newlines, got %q", got)
	}
}

func TestTrimHistoryTokens_SummaryOnlyWhenMessagesTooLarge(t *testing.T) {
	msgs := []domain.Message{
		{Role: domain.RoleUser, Content: "small user instruction"},
		{Role: domain.RoleAssistant, Content: "very long response " + strings.Repeat("data ", 500)},
	}
	// Budget is too small for the 500-word response, but large enough for the compact summary
	got := trimHistoryTokens(msgs, 60)
	if len(got) != 1 || got[0].Role != domain.RoleUser || !strings.Contains(got[0].Content, "[Previous Context Summary:") {
		t.Fatalf("expected summary-only user message, got: %+v", got)
	}
}

func TestTrimHistoryTokens_AdditionalBranches(t *testing.T) {
	// 1. Pruned messages that distill to empty string
	msgsEmptyPruned := []domain.Message{
		{Role: domain.RoleUser, Content: "   \n\t"},
		{Role: domain.RoleUser, Content: "valid question"},
		{Role: domain.RoleAssistant, Content: "valid answer"},
	}
	// budget that only fits valid question and answer
	got := trimHistoryTokens(msgsEmptyPruned, 15)
	if len(got) != 2 || got[0].Content != "valid question" {
		t.Fatalf("expected kept without summary, got %+v", got)
	}

	// 2. Prune older kept user-assistant pair to make room for rolling memory
	msgsMultiplePairs := []domain.Message{
		{Role: domain.RoleUser, Content: "early step 1"},
		{Role: domain.RoleAssistant, Content: "early answer 1"},
		{Role: domain.RoleUser, Content: "middle step 2"},
		{Role: domain.RoleAssistant, Content: "middle answer 2"},
		{Role: domain.RoleUser, Content: "recent step 3"},
		{Role: domain.RoleAssistant, Content: "recent answer 3"},
	}
	// Budget where 2 pairs fit initially, but adding summary requires pruning the older kept pair
	// so that only recent step 3 + recent answer 3 + summary are returned
	got2 := trimHistoryTokens(msgsMultiplePairs, 40)
	if len(got2) < 2 || !strings.Contains(got2[0].Content, "[Previous Context Summary:") {
		t.Fatalf("expected summary prepended to pruned pair, got %+v", got2)
	}

	// 3. distillPrunedHistory with system role or unknown role
	msgsUnknown := []domain.Message{
		{Role: domain.RoleSystem, Content: "system instructions"},
	}
	if got := distillPrunedHistory(msgsUnknown); got != "" {
		t.Fatalf("expected empty summary for system message, got %q", got)
	}

	// 4. distillPrunedHistory with empty text after previous summary extraction
	msgsPrevOnly := []domain.Message{
		{Role: domain.RoleUser, Content: "[Previous Context Summary: prior task completed]"},
	}
	if got := distillPrunedHistory(msgsPrevOnly); got != "prior task completed" {
		t.Fatalf("expected prior summary preserved, got %q", got)
	}
}
