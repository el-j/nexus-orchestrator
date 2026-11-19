package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type mockGenerator struct {
	name      string
	responses []string
	errors    []error
	calls     int
	prompts   []string
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	idx := m.calls
	m.calls++
	m.prompts = append(m.prompts, prompt)
	if idx < len(m.errors) && m.errors[idx] != nil {
		return "", m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "default code", nil
}

func (m *mockGenerator) ProviderName() string {
	if m.name != "" {
		return m.name
	}
	return "mock-provider"
}

func TestPolicy_DefaultsAndLoad(t *testing.T) {
	def := DefaultPolicy()
	if def.MaxSourceLines != 500 || def.MaxTestLines != 800 {
		t.Fatalf("unexpected defaults: %+v", def)
	}

	// Empty path returns defaults
	pol, err := LoadPolicy("")
	if err != nil || pol.MaxSourceLines != 500 {
		t.Fatalf("expected default policy for empty path, got %+v, err: %v", pol, err)
	}

	// Non-existent path returns error
	_, err = LoadPolicy("non-existent-policy.json")
	if err == nil {
		t.Fatal("expected error reading non-existent file")
	}

	// Invalid JSON returns error
	tmpDir := t.TempDir()
	badJSON := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadPolicy(badJSON)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}

	// Valid JSON with zeroes should restore defaults
	zeroJSON := filepath.Join(tmpDir, "zero.json")
	if err := os.WriteFile(zeroJSON, []byte(`{"max_source_lines":0,"max_test_lines":-1}`), 0644); err != nil {
		t.Fatal(err)
	}
	pol, err = LoadPolicy(zeroJSON)
	if err != nil || pol.MaxSourceLines != 500 || pol.MaxTestLines != 800 {
		t.Fatalf("expected defaults for zero limits, got %+v", pol)
	}

	// Custom valid policy
	customJSON := filepath.Join(tmpDir, "custom.json")
	content := `{"max_source_lines":100,"max_test_lines":200,"enforce_test_first":true,"skip_patterns":["vendor/"],"file_extensions":[".go"]}`
	if err := os.WriteFile(customJSON, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pol, err = LoadPolicy(customJSON)
	if err != nil || pol.MaxSourceLines != 100 || pol.MaxTestLines != 200 || !pol.EnforceTestFirst {
		t.Fatalf("unexpected loaded policy: %+v, err: %v", pol, err)
	}
}

func TestIsTestFileAndShouldSkip(t *testing.T) {
	testFiles := []string{
		"foo_test.go",
		"components/Button.spec.ts",
		"src/index.test.ts",
		"internal/test/helper.go",
		"pkg/testutil/fakes.go",
	}
	for _, f := range testFiles {
		if !IsTestFile(f) {
			t.Errorf("expected %s to be recognized as test file", f)
		}
	}

	srcFiles := []string{
		"main.go",
		"services/orchestrator.go",
		"app.vue",
	}
	for _, f := range srcFiles {
		if IsTestFile(f) {
			t.Errorf("expected %s not to be test file", f)
		}
	}

	skips := []string{"node_modules/", "^dist/", `\.d\.ts$`}
	if !ShouldSkip("node_modules/pkg/index.js", skips) {
		t.Error("expected node_modules to be skipped")
	}
	if !ShouldSkip("dist/bundle.js", skips) {
		t.Error("expected dist to be skipped")
	}
	if !ShouldSkip("types/index.d.ts", skips) {
		t.Error("expected d.ts to be skipped")
	}
	if ShouldSkip("src/main.go", skips) {
		t.Error("expected src/main.go not to be skipped")
	}
}

func TestCountFileLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing file error
	_, err := countFileLines(filepath.Join(tmpDir, "none.txt"))
	if err == nil {
		t.Fatal("expected error on missing file")
	}

	// File ending with newline
	f1 := filepath.Join(tmpDir, "f1.txt")
	if err := os.WriteFile(f1, []byte("line 1\nline 2\nline 3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c1, err := countFileLines(f1)
	if err != nil || c1 != 3 {
		t.Fatalf("expected 3 lines, got %d, err: %v", c1, err)
	}

	// File ending without newline
	f2 := filepath.Join(tmpDir, "f2.txt")
	if err := os.WriteFile(f2, []byte("line 1\nline 2"), 0644); err != nil {
		t.Fatal(err)
	}
	c2, err := countFileLines(f2)
	if err != nil || c2 != 2 {
		t.Fatalf("expected 2 lines, got %d, err: %v", c2, err)
	}
}

func TestCheckFileSizes(t *testing.T) {
	tmpDir := t.TempDir()
	pol := DefaultPolicy()
	pol.MaxSourceLines = 3
	pol.MaxTestLines = 4

	// Create oversized source file (4 lines)
	src := filepath.Join(tmpDir, "app.go")
	if err := os.WriteFile(src, []byte("1\n2\n3\n4\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create oversized test file (5 lines)
	tst := filepath.Join(tmpDir, "app_test.go")
	if err := os.WriteFile(tst, []byte("1\n2\n3\n4\n5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compliant files
	okSrc := filepath.Join(tmpDir, "ok.go")
	if err := os.WriteFile(okSrc, []byte("1\n2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	violations, err := CheckFileSizes(ctx, tmpDir, pol)
	if err != nil {
		t.Fatalf("CheckFileSizes error: %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %+v", len(violations), violations)
	}

	// Check specific file paths (relative and absolute)
	relViolations, err := CheckFileSizes(ctx, tmpDir, pol, "app.go", tst)
	if err != nil || len(relViolations) != 2 {
		t.Fatalf("expected 2 violations on specific paths, got %d", len(relViolations))
	}

	// Context cancellation
	cancCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = CheckFileSizes(cancCtx, tmpDir, pol)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCheckTestFirst(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// 1. Empty targets and not a git repo: returns nil without error
	v, err := CheckTestFirst(ctx, tmpDir, nil)
	if err != nil || len(v) != 0 {
		t.Fatalf("expected nil violations for empty git directory, got: %v, err: %v", v, err)
	}

	// 2. Only test files provided: compliant
	v, err = CheckTestFirst(ctx, tmpDir, []string{"pkg/foo_test.go"})
	if err != nil || len(v) != 0 {
		t.Fatalf("expected nil violations when test files modified, got: %v", v)
	}

	// 3. Source file provided, corresponding test exists on disk: compliant
	src := filepath.Join(tmpDir, "foo.go")
	tst := filepath.Join(tmpDir, "foo_test.go")
	_ = os.WriteFile(src, []byte("package foo"), 0644)
	_ = os.WriteFile(tst, []byte("package foo"), 0644)

	v, err = CheckTestFirst(ctx, tmpDir, []string{"foo.go"})
	if err != nil || len(v) != 0 {
		t.Fatalf("expected 0 violations when test exists on disk, got: %+v", v)
	}

	// 4. Source file provided, test missing on disk: violation
	srcMissing := filepath.Join(tmpDir, "bar.go")
	_ = os.WriteFile(srcMissing, []byte("package bar"), 0644)
	v, err = CheckTestFirst(ctx, tmpDir, []string{"bar.go"})
	if err != nil || len(v) != 1 {
		t.Fatalf("expected 1 violation for missing test, got: %+v", v)
	}
	if v[0].Rule != "test_first_required" {
		t.Fatalf("unexpected rule: %s", v[0].Rule)
	}

	// 5. TypeScript test convention: bar.test.ts missing
	v, err = CheckTestFirst(ctx, tmpDir, []string{"bar.ts"})
	if err != nil || len(v) != 1 {
		t.Fatalf("expected 1 violation for bar.ts, got: %+v", v)
	}
}

func TestVerificationCommandAndVerify(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Empty command passes immediately
	res, err := RunVerificationCommand(ctx, tmpDir, "")
	if err != nil || !res.Passed {
		t.Fatalf("expected empty command to pass, got: %+v", res)
	}

	// Successful command
	successCmd := "echo test-ok"
	res, err = RunVerificationCommand(ctx, tmpDir, successCmd)
	if err != nil || !res.Passed || !strings.Contains(res.Output, "test-ok") {
		t.Fatalf("expected success command to pass, got: %+v", res)
	}

	// Failing command
	failCmd := "exit 42"
	if runtime.GOOS == "windows" {
		failCmd = "exit /b 42"
	}
	res, err = RunVerificationCommand(ctx, tmpDir, failCmd)
	if err != nil || res.Passed || res.ExitCode != 42 {
		t.Fatalf("expected exit code 42, got: %+v, err: %v", res, err)
	}

	// Integrated Verify call
	pol := DefaultPolicy()
	pol.MaxSourceLines = 2
	src := filepath.Join(tmpDir, "viol.go")
	_ = os.WriteFile(src, []byte("1\n2\n3\n4\n"), 0644)
	pol.VerificationCmd = "echo verify-ran"

	vRes, err := Verify(ctx, tmpDir, pol)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if vRes.Passed || len(vRes.Violations) == 0 || !strings.Contains(vRes.Output, "verify-ran") {
		t.Fatalf("expected Verify to fail due to file size violation, got: %+v", vRes)
	}

	// Test-first enabled in Verify
	pol2 := DefaultPolicy()
	pol2.EnforceTestFirst = true
	vRes2, err := Verify(ctx, tmpDir, pol2, "untested.go")
	if err != nil || vRes2.Passed || len(vRes2.Violations) == 0 {
		t.Fatalf("expected test-first violation in Verify, got: %+v", vRes2)
	}
}

func TestExecutor_Helpers(t *testing.T) {
	// ExtractCode
	fenced := "```go\npackage main\nfunc main() {}\n```"
	if ExtractCode(fenced) != "package main\nfunc main() {}" {
		t.Fatalf("unexpected extract: %q", ExtractCode(fenced))
	}
	unclosed := "```go\nline1\nline2"
	if ExtractCode(unclosed) != "line1\nline2" {
		t.Fatalf("unexpected unclosed extract: %q", ExtractCode(unclosed))
	}
	plain := "package main"
	if ExtractCode(plain) != plain {
		t.Fatalf("unexpected plain extract: %q", ExtractCode(plain))
	}

	// Truncate observation
	short := "short log"
	if truncateObservation(short) != short {
		t.Fatalf("expected short log unchanged")
	}
	long := strings.Repeat("x", 5000)
	if !strings.HasSuffix(truncateObservation(long), "... [truncated]") {
		t.Fatalf("expected long log to be truncated")
	}

	// FormatCorrectionPrompt
	prompt := FormatCorrectionPrompt("make app", "main.go", "code", "go test", "fail error")
	if !strings.Contains(prompt, "Original task instruction") || !strings.Contains(prompt, "main.go") {
		t.Fatalf("unexpected prompt: %s", prompt)
	}

	// readTaskFile
	tmpDir := t.TempDir()
	taskPath := filepath.Join(tmpDir, "task.md")
	content := "title: Build\ntarget: internal/foo/bar.go\nverification: go test ./...\nInstruction body"
	if err := os.WriteFile(taskPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	inst, tgt, vcmd, err := readTaskFile(taskPath)
	if err != nil || tgt != "internal/foo/bar.go" || vcmd != "go test ./..." || !strings.Contains(inst, "Instruction body") {
		t.Fatalf("readTaskFile unexpected: inst=%q, tgt=%q, vcmd=%q, err=%v", inst, tgt, vcmd, err)
	}

	_, _, _, err = readTaskFile(filepath.Join(tmpDir, "none.md"))
	if err == nil {
		t.Fatal("expected error reading non-existent task file")
	}
}

func TestExecuteWithHealing(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// 1. Task file execution without generator, valid verification
	taskPath := filepath.Join(tmpDir, "task.md")
	_ = os.WriteFile(taskPath, []byte("target: out.txt\nverify: echo ok\nBody"), 0644)
	res, err := ExecuteWithHealing(ctx, nil, ExecuteConfig{
		TaskFilePath: taskPath,
		ProjectPath:  tmpDir,
	})
	if err != nil || !res.Success {
		t.Fatalf("expected execution without generator to succeed: %+v, err: %v", res, err)
	}

	// 2. Successful turn 1 generation and verification
	mock := &mockGenerator{
		name:      "mock-llm",
		responses: []string{"```txt\nhello world\n```"},
	}
	res, err = ExecuteWithHealing(ctx, mock, ExecuteConfig{
		Instruction:     "write hello",
		TargetFile:      "hello.txt",
		VerificationCmd: "echo pass",
		ProjectPath:     tmpDir,
	})
	if err != nil || !res.Success || res.Turns != 1 {
		t.Fatalf("expected turn 1 success, got %+v, err: %v", res, err)
	}
	saved, _ := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
	if string(saved) != "hello world" {
		t.Fatalf("unexpected file content: %q", string(saved))
	}

	// 3. Generation failure on turn 1
	failMock := &mockGenerator{
		errors: []error{errors.New("quota exceeded")},
	}
	res, err = ExecuteWithHealing(ctx, failMock, ExecuteConfig{
		Instruction: "fail now",
	})
	if err == nil || res.Success {
		t.Fatalf("expected error on generation failure, got res: %+v", res)
	}

	// 4. Verification fails on turn 1, healed on turn 2
	healMock := &mockGenerator{
		responses: []string{"bad code", "good code"},
	}
	// On turn 1 check fails, on turn 2 file contains "good code" and passes
	verifyCheck := "grep -q 'good code' " + filepath.Join(tmpDir, "target.txt")
	res, err = ExecuteWithHealing(ctx, healMock, ExecuteConfig{
		Instruction:     "write good code",
		TargetFile:      "target.txt",
		VerificationCmd: verifyCheck,
		Heal:            true,
		MaxTurns:        2,
		ProjectPath:     tmpDir,
	})
	if err != nil || !res.Success || res.Turns != 2 {
		t.Fatalf("expected healed on turn 2, got: %+v, err: %v", res, err)
	}

	// 5. Verification fails and correction turns exhausted
	alwaysFailMock := &mockGenerator{
		responses: []string{"bad 1", "bad 2"},
	}
	res, err = ExecuteWithHealing(ctx, alwaysFailMock, ExecuteConfig{
		Instruction:     "fail healing",
		TargetFile:      "bad.txt",
		VerificationCmd: "exit 1",
		Heal:            true,
		MaxTurns:        2,
		ProjectPath:     tmpDir,
	})
	if err != nil || res.Success || res.Turns != 2 {
		t.Fatalf("expected failed after 2 turns, got: %+v, err: %v", res, err)
	}

	// 6. Healing turn generation error
	turn2ErrMock := &mockGenerator{
		responses: []string{"bad 1"},
		errors:    []error{nil, errors.New("mid-heal error")},
	}
	res, err = ExecuteWithHealing(ctx, turn2ErrMock, ExecuteConfig{
		Instruction:     "fail turn 2",
		TargetFile:      "bad2.txt",
		VerificationCmd: "exit 1",
		Heal:            true,
		MaxTurns:        2,
		ProjectPath:     tmpDir,
	})
	if err == nil || res.Success {
		t.Fatalf("expected error on turn 2 generation, got res: %+v", res)
	}

	// 7. Invalid task file path error in ExecuteWithHealing
	_, err = ExecuteWithHealing(ctx, nil, ExecuteConfig{
		TaskFilePath: filepath.Join(tmpDir, "missing-task.md"),
	})
	if err == nil {
		t.Fatal("expected error on missing task file")
	}

	// 8. Write target file failure on turn 1
	origMkdir := osMkdirAll
	osMkdirAll = func(path string, perm os.FileMode) error {
		return errors.New("mkdir failed")
	}
	_, err = ExecuteWithHealing(ctx, mock, ExecuteConfig{
		Instruction: "test mkdir fail",
		TargetFile:  "fail.txt",
		ProjectPath: tmpDir,
	})
	osMkdirAll = origMkdir
	if err == nil {
		t.Fatal("expected write target file failure on turn 1")
	}

	// 9. Write target file failure on healing turn
	origWriteFile := osWriteFile
	healWriteFailMock := &mockGenerator{
		responses: []string{"bad code", "heal code"},
	}
	turnCounter := 0
	osWriteFile = func(path string, data []byte, perm os.FileMode) error {
		turnCounter++
		if turnCounter == 2 {
			return errors.New("write fail turn 2")
		}
		return origWriteFile(path, data, perm)
	}
	_, err = ExecuteWithHealing(ctx, healWriteFailMock, ExecuteConfig{
		Instruction:     "test write fail turn 2",
		TargetFile:      "fail2.txt",
		VerificationCmd: "exit 1",
		Heal:            true,
		MaxTurns:        2,
		ProjectPath:     tmpDir,
	})
	osWriteFile = origWriteFile
	if err == nil {
		t.Fatal("expected write failure on turn 2")
	}

	// 10. Context cancelled during healing turn
	cancCtx, cancel := context.WithCancel(ctx)
	healCancelMock := &mockGenerator{
		responses: []string{"bad code"},
	}
	go func() {
		cancel()
	}()
	_, _ = ExecuteWithHealing(cancCtx, healCancelMock, ExecuteConfig{
		Instruction:     "cancel healing",
		TargetFile:      "canc.txt",
		VerificationCmd: "exit 1",
		Heal:            true,
		MaxTurns:        2,
		ProjectPath:     tmpDir,
	})
}

func TestInspect(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create a source file and a test file
	_ = os.WriteFile(filepath.Join(tmpDir, "core.go"), []byte("package core\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "core_test.go"), []byte("package core\n"), 0644)

	pol := DefaultPolicy()
	report, err := Inspect(ctx, tmpDir, pol)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if !report.Passed || report.SourceFiles != 1 || report.TestFiles != 1 || report.TotalFiles != 2 {
		t.Fatalf("unexpected inspection report: %+v", report)
	}

	// Violation in inspect
	polViol := DefaultPolicy()
	polViol.MaxSourceLines = 1
	_ = os.WriteFile(filepath.Join(tmpDir, "core.go"), []byte("line1\nline2\nline3\n"), 0644)
	repViol, err := Inspect(ctx, tmpDir, polViol)
	if err != nil || repViol.Passed || len(repViol.Violations) == 0 {
		t.Fatalf("expected inspect to report violation, got: %+v", repViol)
	}

	// inspect with EnforceTestFirst
	polTestFirst := DefaultPolicy()
	polTestFirst.EnforceTestFirst = true
	_, err = Inspect(ctx, tmpDir, polTestFirst)
	if err != nil {
		t.Fatalf("Inspect with EnforceTestFirst failed: %v", err)
	}

	// inspect scanFiles error
	origScan := scanFilesFn
	scanFilesFn = func(root string, pol Policy) ([]string, error) {
		return nil, errors.New("scan error")
	}
	_, err = Inspect(ctx, tmpDir, pol)
	scanFilesFn = origScan
	if err == nil {
		t.Fatal("expected inspect scan error")
	}

	// inspect checkFileSizes error
	origSizes := checkFileSizesFn
	checkFileSizesFn = func(ctx context.Context, root string, pol Policy, filePaths ...string) ([]Violation, error) {
		return nil, errors.New("check sizes error")
	}
	_, err = Inspect(ctx, tmpDir, pol)
	checkFileSizesFn = origSizes
	if err == nil {
		t.Fatal("expected inspect check sizes error")
	}

	// inspect checkTestFirst error
	origTestFirst := checkTestFirstFn
	checkTestFirstFn = func(ctx context.Context, root string, files []string) ([]Violation, error) {
		return nil, errors.New("check test first error")
	}
	_, err = Inspect(ctx, tmpDir, polTestFirst)
	checkTestFirstFn = origTestFirst
	if err == nil {
		t.Fatal("expected inspect check test first error")
	}
}

func TestVerifier_MoreEdges(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// buildShellCmd windows vs linux
	cmdWin := buildShellCmd(ctx, tmpDir, "dir", "windows")
	if !strings.Contains(cmdWin.Path, "cmd") {
		t.Errorf("expected cmd on windows: %s", cmdWin.Path)
	}
	cmdUnix := buildShellCmd(ctx, tmpDir, "ls", "linux")
	if !strings.Contains(cmdUnix.Path, "sh") {
		t.Errorf("expected sh on unix: %s", cmdUnix.Path)
	}

	// Non-exit error in RunVerificationCommand (e.g. invalid command binary)
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "non-existent-binary-123456")
	}
	res, _ := RunVerificationCommand(ctx, tmpDir, "some cmd")
	execCommandContext = origExec
	if res.Passed || res.ExitCode != 1 {
		t.Fatalf("expected failure and exit code 1, got %+v", res)
	}

	// Verify error branches
	pol := DefaultPolicy()
	origSizes := checkFileSizesFn
	checkFileSizesFn = func(ctx context.Context, root string, pol Policy, filePaths ...string) ([]Violation, error) {
		return nil, errors.New("sizes err")
	}
	_, err := Verify(ctx, tmpDir, pol)
	checkFileSizesFn = origSizes
	if err == nil {
		t.Fatal("expected verify error on checkFileSizes")
	}

	origTestFirst := checkTestFirstFn
	checkTestFirstFn = func(ctx context.Context, root string, files []string) ([]Violation, error) {
		return nil, errors.New("test first err")
	}
	pol.EnforceTestFirst = true
	_, err = Verify(ctx, tmpDir, pol)
	checkTestFirstFn = origTestFirst
	if err == nil {
		t.Fatal("expected verify error on checkTestFirst")
	}

	origCmd := runVerificationCmdFn
	runVerificationCmdFn = func(ctx context.Context, dir string, cmdStr string) (VerificationResult, error) {
		return VerificationResult{}, errors.New("cmd run err")
	}
	pol.EnforceTestFirst = false
	pol.VerificationCmd = "echo test"
	_, err = Verify(ctx, tmpDir, pol)
	runVerificationCmdFn = origCmd
	if err == nil {
		t.Fatal("expected verify error on runVerificationCmd")
	}
}

func TestPolicy_MoreEdges(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Empty arrays in JSON policy trigger default resets
	emptyArrJSON := filepath.Join(tmpDir, "empty.json")
	_ = os.WriteFile(emptyArrJSON, []byte(`{"skip_patterns":[], "file_extensions":[]}`), 0644)
	pol, err := LoadPolicy(emptyArrJSON)
	if err != nil || len(pol.SkipPatterns) == 0 || len(pol.FileExtensions) == 0 {
		t.Fatalf("expected defaults for empty arrays: %+v", pol)
	}

	// scanFiles with directory skipped and file skipped
	skipDir := filepath.Join(tmpDir, "node_modules", "sub")
	_ = os.MkdirAll(skipDir, 0755)
	_ = os.WriteFile(filepath.Join(skipDir, "a.go"), []byte("package a"), 0644)
	skipFile := filepath.Join(tmpDir, "test.d.ts")
	_ = os.WriteFile(skipFile, []byte("type X = string;"), 0644)
	validFile := filepath.Join(tmpDir, "valid.go")
	_ = os.WriteFile(validFile, []byte("package valid"), 0644)

	files, err := scanFiles(tmpDir, pol)
	if err != nil || len(files) != 1 || !strings.HasSuffix(files[0], "valid.go") {
		t.Fatalf("unexpected scanned files: %v, err: %v", files, err)
	}

	// scanFiles with filepathRel error
	origRel := filepathRel
	filepathRel = func(basepath, targpath string) (string, error) {
		return "", errors.New("rel err")
	}
	filesRel, _ := scanFiles(tmpDir, pol)
	if len(filesRel) == 0 {
		t.Fatal("expected scanFiles to fallback to path on rel err")
	}

	// CheckFileSizes with filepathRel error and scanFilesFn error
	origScan := scanFilesFn
	scanFilesFn = func(root string, pol Policy) ([]string, error) {
		return nil, errors.New("scan err")
	}
	_, err = CheckFileSizes(ctx, tmpDir, pol)
	scanFilesFn = origScan
	if err == nil {
		t.Fatal("expected CheckFileSizes error on scan failure")
	}

	// CheckFileSizes with target file that fails countFileLines
	_, err = CheckFileSizes(ctx, tmpDir, pol, filepath.Join(tmpDir, "non-existent-target.go"))
	filepathRel = origRel
	if err != nil {
		t.Fatalf("expected non-existent file to be skipped, got error: %v", err)
	}

	// gitStatusFiles in a real git repository
	gitDir := filepath.Join(tmpDir, "gitrepo")
	_ = os.MkdirAll(gitDir, 0755)
	initCmd := exec.CommandContext(ctx, "git", "init")
	initCmd.Dir = gitDir
	_ = initCmd.Run()

	testGitFile := filepath.Join(gitDir, "hello.go")
	_ = os.WriteFile(testGitFile, []byte("package gitrepo"), 0644)

	stFiles, err := gitStatusFiles(ctx, gitDir)
	if err != nil || len(stFiles) == 0 || !strings.Contains(stFiles[0], "hello.go") {
		t.Fatalf("unexpected git status files: %v, err: %v", stFiles, err)
	}

	// gitStatusFiles in non-git directory returns error
	_, err = gitStatusFiles(ctx, filepath.Join(tmpDir, "not-a-repo"))
	if err == nil {
		t.Fatal("expected error running git status in non-repo")
	}

	// CheckTestFirst using gitStatusRunner
	origGitRunner := gitStatusRunner
	gitStatusRunner = func(ctx context.Context, root string) ([]string, error) {
		return []string{"skip/node_modules/pkg.go", "doc.txt"}, nil
	}
	tfViolations, err := CheckTestFirst(ctx, tmpDir, nil)
	gitStatusRunner = origGitRunner
	if err != nil || len(tfViolations) != 0 {
		t.Fatalf("expected no violations for non-source git files: %+v", tfViolations)
	}
}
