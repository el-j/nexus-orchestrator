package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExecutor_VerificationCmdErrors(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Turn 1 verification command returns error
	origCmd := runVerificationCmdFn
	runVerificationCmdFn = func(ctx context.Context, dir string, cmdStr string) (VerificationResult, error) {
		return VerificationResult{}, errors.New("turn 1 vErr")
	}
	res, err := ExecuteWithHealing(ctx, nil, ExecuteConfig{
		VerificationCmd: "test cmd",
		ProjectPath:     tmpDir,
	})
	if err == nil || res.Success {
		t.Fatal("expected error on turn 1 verification failure")
	}

	// Turn 2 healing verification command returns error
	healMock := &mockGenerator{
		responses: []string{"bad code", "heal code"},
	}
	callCount := 0
	runVerificationCmdFn = func(ctx context.Context, dir string, cmdStr string) (VerificationResult, error) {
		callCount++
		if callCount == 1 {
			return VerificationResult{Passed: false, ExitCode: 1}, nil
		}
		return VerificationResult{}, errors.New("turn 2 vErr")
	}
	res, err = ExecuteWithHealing(ctx, healMock, ExecuteConfig{
		Instruction:     "write code",
		TargetFile:      "heal.txt",
		VerificationCmd: "test cmd",
		Heal:            true,
		MaxTurns:        0, // tests default to 2
		ProjectPath:     tmpDir,
	})
	runVerificationCmdFn = origCmd
	if err == nil || res.Success {
		t.Fatal("expected error on turn 2 verification failure")
	}

	// Verification fails when heal is false (or generator is nil)
	failWithoutHealRes, err := ExecuteWithHealing(ctx, nil, ExecuteConfig{
		VerificationCmd: "exit 1",
		ProjectPath:     tmpDir,
	})
	if err != nil || failWithoutHealRes.Success {
		t.Fatal("expected failure without heal when verification fails")
	}
}

func TestPolicy_GitStatusAndFileGaps(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// countFileLines on directory triggers non-EOF error
	_, err := countFileLines(tmpDir)
	if err == nil {
		t.Fatal("expected error calling countFileLines on a directory")
	}

	// scanFiles with non-existent root
	_, err = scanFiles(filepath.Join(tmpDir, "missing-dir"), DefaultPolicy())
	if err == nil {
		t.Fatal("expected scanFiles error on non-existent root")
	}

	// CheckFileSizes with explicit skipped path in filePaths
	pol := DefaultPolicy()
	violations, err := CheckFileSizes(ctx, tmpDir, pol, "node_modules/skipped.go")
	if err != nil || len(violations) != 0 {
		t.Fatalf("expected 0 violations for skipped path: %v", violations)
	}

	// CheckTestFirst with gitStatusRunner returning empty list
	origGit := gitStatusRunner
	gitStatusRunner = func(ctx context.Context, root string) ([]string, error) {
		return []string{}, nil
	}
	v, err := CheckTestFirst(ctx, tmpDir, nil)
	gitStatusRunner = origGit
	if err != nil || len(v) != 0 {
		t.Fatalf("expected 0 violations for empty git status: %v", v)
	}

	// Git status in a real repo with short line and rename
	gitDir := filepath.Join(tmpDir, "git-rename-repo")
	_ = os.MkdirAll(gitDir, 0755)
	c1 := exec.CommandContext(ctx, "git", "init")
	c1.Dir = gitDir
	_ = c1.Run()

	c2 := exec.CommandContext(ctx, "git", "config", "user.name", "Test")
	c2.Dir = gitDir
	_ = c2.Run()

	c3 := exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	c3.Dir = gitDir
	_ = c3.Run()

	file1 := filepath.Join(gitDir, "f1.go")
	_ = os.WriteFile(file1, []byte("package gitrepo"), 0644)
	c4 := exec.CommandContext(ctx, "git", "add", "f1.go")
	c4.Dir = gitDir
	_ = c4.Run()

	c5 := exec.CommandContext(ctx, "git", "commit", "-m", "init")
	c5.Dir = gitDir
	_ = c5.Run()

	c6 := exec.CommandContext(ctx, "git", "mv", "f1.go", "f2.go")
	c6.Dir = gitDir
	_ = c6.Run()

	stFiles, err := gitStatusFiles(ctx, gitDir)
	if err != nil || len(stFiles) == 0 {
		t.Fatalf("expected git status to find renamed file, got: %v, err: %v", stFiles, err)
	}

	// Inspect in git repo sets GitStatus
	rep, err := Inspect(ctx, gitDir, pol)
	if err != nil || rep.GitStatus == "" {
		t.Fatalf("expected inspect to capture GitStatus: %+v, err: %v", rep, err)
	}

	// gitStatusFiles with short lines (line < 4)
	origStatusCmd := gitStatusCmd
	gitStatusCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "ok") // "ok\n" has len 2 (< 4)
	}
	shortLineFiles, err := gitStatusFiles(ctx, gitDir)
	gitStatusCmd = origStatusCmd
	if err != nil || len(shortLineFiles) != 0 {
		t.Fatalf("expected 0 files for short status output: %v", shortLineFiles)
	}
}

func TestVerify_FailingVerificationCommand(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	pol := DefaultPolicy()
	pol.VerificationCmd = "exit 1"

	res, err := Verify(ctx, tmpDir, pol)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if res.Passed {
		t.Fatal("expected Verify to report passed=false when verification cmd fails")
	}
}
