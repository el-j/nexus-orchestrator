package runner

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	execCommandContext   = exec.CommandContext
	checkFileSizesFn     = CheckFileSizes
	checkTestFirstFn     = CheckTestFirst
	runVerificationCmdFn = RunVerificationCommand
)

// buildShellCmd constructs an exec.Cmd with the appropriate shell for the platform.
func buildShellCmd(ctx context.Context, dir, cmdStr, goos string) *exec.Cmd {
	var cmd *exec.Cmd
	if goos == "windows" {
		cmd = execCommandContext(ctx, "cmd", "/C", cmdStr)
	} else {
		cmd = execCommandContext(ctx, "sh", "-c", cmdStr)
	}
	cmd.Dir = dir
	return cmd
}

// RunVerificationCommand executes a verification shell command in dir and records the outcome.
func RunVerificationCommand(ctx context.Context, dir string, cmdStr string) (VerificationResult, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return VerificationResult{Passed: true}, nil
	}

	start := time.Now()
	cmd := buildShellCmd(ctx, dir, cmdStr, runtime.GOOS)

	outputBytes, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := VerificationResult{
		Command:    cmdStr,
		Output:     string(outputBytes),
		DurationMs: duration.Milliseconds(),
		ExitCode:   0,
		Passed:     true,
	}

	if err != nil {
		result.Passed = false
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result, nil
}

// Verify runs all configured policy invariant checks (file sizes, test-first) and verification command.
func Verify(ctx context.Context, dir string, pol Policy, targets ...string) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	sizeViolations, err := checkFileSizesFn(ctx, dir, pol, targets...)
	if err != nil {
		return result, err
	}
	result.Violations = append(result.Violations, sizeViolations...)

	if pol.EnforceTestFirst {
		testViolations, err := checkTestFirstFn(ctx, dir, targets)
		if err != nil {
			return result, err
		}
		result.Violations = append(result.Violations, testViolations...)
	}

	if len(result.Violations) > 0 {
		result.Passed = false
	}

	if pol.VerificationCmd != "" {
		cmdResult, err := runVerificationCmdFn(ctx, dir, pol.VerificationCmd)
		if err != nil {
			return result, err
		}
		result.Command = cmdResult.Command
		result.Output = cmdResult.Output
		result.ExitCode = cmdResult.ExitCode
		result.DurationMs = cmdResult.DurationMs
		if !cmdResult.Passed {
			result.Passed = false
		}
	}

	return result, nil
}
