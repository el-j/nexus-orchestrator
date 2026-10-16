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
