package runner

import (
	"strings"
	"testing"
)

func TestParseDiagnostics(t *testing.T) {
	output := `
# some build comments
main.go:42:15: undefined: ProcessData
service/handler.go:10: syntax error
src/index.ts(25,8): error TS2304: Cannot find name 'foo'.
--- FAIL: TestCalculation (0.01s)
    calc_test.go:55: failed assertion in method DoMath
`
	diags := ParseDiagnostics(output)
	if len(diags) < 4 {
		t.Fatalf("expected at least 4 diagnostics, got %d: %+v", len(diags), diags)
	}

	byFile := make(map[string]Diagnostic)
	for _, d := range diags {
		byFile[d.File] = d
	}

	d0, ok := byFile["main.go"]
	if !ok || d0.Line != 42 || d0.Column != 15 || d0.Symbol != "ProcessData" {
		t.Fatalf("unexpected d0: %+v", d0)
	}

	d1, ok := byFile["service/handler.go"]
	if !ok || d1.Line != 10 {
		t.Fatalf("unexpected d1: %+v", d1)
	}

	d2, ok := byFile["src/index.ts"]
	if !ok || d2.Line != 25 || d2.Column != 8 {
		t.Fatalf("unexpected d2: %+v", d2)
	}

	d3, ok := byFile["calc_test.go"]
	if !ok || d3.Line != 55 || d3.Symbol != "DoMath" {
		t.Fatalf("unexpected d3: %+v", d3)
	}

	// Deduplication and invalid line handling
	empty := ParseDiagnostics("no line numbers here")
	if len(empty) != 0 {
		t.Fatalf("expected 0 diagnostics for clean output, got %d", len(empty))
	}
}

func TestExtractASTDeclaration_GoFunctionsAndTypes(t *testing.T) {
	code := `package sample

import "fmt"

type ServiceConfig struct {
	Port int
	Host string
}

func NewService(cfg ServiceConfig) *Service {
	return &Service{cfg: cfg}
}

type Service struct {
	cfg ServiceConfig
}

func (s *Service) Start() error {
	fmt.Println("starting")
	return nil
}

func (s Service) Stop() {
}
`

	// 1. Target inside NewService (line 11)
	sym, decl, ok := ExtractASTDeclaration(code, 11)
	if !ok || sym != "NewService" || !strings.Contains(decl, "func NewService") {
		t.Fatalf("expected NewService, got ok=%v sym=%s decl=%s", ok, sym, decl)
	}

	// 2. Target inside ServiceConfig type (line 6)
	sym, decl, ok = ExtractASTDeclaration(code, 6)
	if !ok || sym != "ServiceConfig" || !strings.Contains(decl, "type ServiceConfig struct") {
		t.Fatalf("expected ServiceConfig, got ok=%v sym=%s decl=%s", ok, sym, decl)
	}

	// 3. Target inside pointer receiver method Start (line 19)
	sym, decl, ok = ExtractASTDeclaration(code, 19)
	if !ok || sym != "(*Service).Start" || !strings.Contains(decl, "func (s *Service) Start()") {
		t.Fatalf("expected (*Service).Start, got ok=%v sym=%s decl=%s", ok, sym, decl)
	}

	// 4. Target inside value receiver method Stop (line 24)
	sym, decl, ok = ExtractASTDeclaration(code, 24)
	if !ok || sym != "(Service).Stop" || !strings.Contains(decl, "func (s Service) Stop()") {
		t.Fatalf("expected (Service).Stop, got ok=%v sym=%s decl=%s", ok, sym, decl)
	}

	// 5. Line outside any declaration (e.g. package or imports, line 1 or 3)
	_, _, ok = ExtractASTDeclaration(code, 1)
	if ok {
		t.Fatalf("expected false for line 1, got ok=true")
	}

	// 6. Invalid inputs
	if _, _, ok := ExtractASTDeclaration("", 10); ok {
		t.Fatal("expected false for empty code")
	}
	if _, _, ok := ExtractASTDeclaration("invalid syntax {{{", 1); ok {
		t.Fatal("expected false for invalid go syntax")
	}
}

func TestExtractFocusedSnippet(t *testing.T) {
	code := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7"

	snippet, start, end := ExtractFocusedSnippet(code, 4, 1)
	if start != 3 || end != 5 {
		t.Fatalf("expected start=3 end=5, got %d, %d", start, end)
	}
	if !strings.Contains(snippet, "   4 | line 4") {
		t.Fatalf("expected line 4 in snippet, got: %s", snippet)
	}

	// Edge cases: start < 1, end > total, empty code
	snippetFirst, startF, _ := ExtractFocusedSnippet(code, 1, 5)
	if startF != 1 || !strings.Contains(snippetFirst, "   1 | line 1") {
		t.Fatalf("expected start at 1, got %d in %s", startF, snippetFirst)
	}

	snippetEmpty, sE, eE := ExtractFocusedSnippet("", 0, 5)
	if snippetEmpty != "" || sE != 1 || eE != 0 {
		t.Fatalf("unexpected snippetEmpty: %q (%d-%d)", snippetEmpty, sE, eE)
	}
}

func TestFormatCorrectionPrompt_ASTAndFallbacks(t *testing.T) {
	goCode := `package main

func Execute() {
	x := 10
	fmt.Println(x)
}
`

	// 1. AST declaration extracted successfully
	promptAST := FormatCorrectionPrompt("fix execute", "main.go", goCode, "go test", "main.go:5:2: undefined: fmt")
	if !strings.Contains(promptAST, "Target file main.go failed verification at line 5 (in Execute)") {
		t.Fatalf("expected AST line and symbol in prompt, got:\n%s", promptAST)
	}
	if !strings.Contains(promptAST, "func Execute()") {
		t.Fatalf("expected failing declaration in prompt, got:\n%s", promptAST)
	}

	// 2. Syntax error in code -> falls back to focused line snippet
	badCode := "package main\n\nfunc Execute() {\n{{{\n"
	promptSnippet := FormatCorrectionPrompt("fix syntax", "main.go", badCode, "go test", "main.go:4:1: syntax error")
	if !strings.Contains(promptSnippet, "Context lines") || !strings.Contains(promptSnippet, "Target file main.go failed verification at line 4:") {
		t.Fatalf("expected focused snippet fallback, got:\n%s", promptSnippet)
	}

	// 3. No matching diagnostic file -> falls back to full output
	promptFull := FormatCorrectionPrompt("do something", "other.go", goCode, "make test", "different.go:1: error")
	if !strings.Contains(promptFull, "Your latest output for other.go:") {
		t.Fatalf("expected full output fallback, got:\n%s", promptFull)
	}
}

func TestAST_EdgeCases(t *testing.T) {
	// 1. recvTypeName fallback
	if got := recvTypeName(nil); got != "recv" {
		t.Fatalf("expected 'recv' for nil expr, got %q", got)
	}

	// 2. ParseDiagnostics with line <= 0 and duplicates
	out := "file.go:0: invalid line\nfile.go:10: err\nfile.go:10: err\n"
	diags := ParseDiagnostics(out)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (filtered line 0 and duplicate), got %d", len(diags))
	}

	// 3. ExtractASTDeclaration boundary clamp
	singleLine := "package p; func F() {}"
	_, _, ok := ExtractASTDeclaration(singleLine, 1)
	if !ok {
		t.Fatal("expected ok for singleLine")
	}
}
