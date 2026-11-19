package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxObservationLength limits the length of error logs in self-healing prompts.
const maxObservationLength = 4000

// ExtractCode strips markdown code blocks if present, returning raw source code.
func ExtractCode(s string) string {
	lines := strings.Split(s, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			start = i
			break
		}
	}
	if start == -1 {
		return s
	}
	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			end = i
			break
		}
	}
	if end == -1 {
		return strings.Join(lines[start+1:], "\n")
	}
	return strings.Join(lines[start+1:end], "\n")
}

// truncateObservation truncates error output if it exceeds maxObservationLength.
func truncateObservation(s string) string {
	if len(s) <= maxObservationLength {
		return s
	}
	return s[:maxObservationLength] + "\n... [truncated]"
}

// FormatCorrectionPrompt constructs a self-healing prompt containing the prior failure and instructions.
func FormatCorrectionPrompt(instruction, targetFile, latestOutput, verificationCmd, verificationOutput string) string {
	return fmt.Sprintf(
		"Original task instruction:\n%s\n\nYour latest output for %s:\n%s\n\n"+
			"It failed verification with command %q:\n\n%s\n\n"+
			"Please fix the errors and output the entire corrected file content.",
		instruction, targetFile, ExtractCode(latestOutput), verificationCmd, truncateObservation(verificationOutput),
	)
}

// readTaskFile parses instruction, target file, and verify command from a task file if present.
func readTaskFile(path string) (instruction, target, verifyCmd string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	content := string(data)
	instruction = content

	lines := strings.Split(content, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "target:") {
			target = strings.TrimSpace(strings.TrimPrefix(trimmed, "target:"))
		}
		if strings.HasPrefix(trimmed, "verification:") || strings.HasPrefix(trimmed, "verify:") {
			idx := strings.Index(trimmed, ":")
			verifyCmd = strings.TrimSpace(trimmed[idx+1:])
		}
	}

	return instruction, target, verifyCmd, nil
}

var (
	osMkdirAll  = os.MkdirAll
	osWriteFile = os.WriteFile
)

// writeTargetFile writes generated content to disk under projectPath.
func writeTargetFile(projectPath, targetFile, content string) error {
	fullPath := targetFile
	if !filepath.IsAbs(fullPath) && projectPath != "" {
		fullPath = filepath.Join(projectPath, targetFile)
	}
	dir := filepath.Dir(fullPath)
	if err := osMkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	return osWriteFile(fullPath, []byte(content), 0644)
}

// runHealingLoop conducts auto-correction iterations with the generator until passing or turns exhaust.
func runHealingLoop(ctx context.Context, gen Generator, cfg ExecuteConfig, instruction, targetFile, verifyCmd, initialCode string, res *ExecuteResult) error {
	latest := initialCode
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 2
	}
	for turn := 2; turn <= maxTurns; turn++ {
		if ctx.Err() != nil {
			res.Success = false
			return ctx.Err()
		}

		res.Turns = turn
		res.Logs = append(res.Logs, fmt.Sprintf("turn %d/%d: healing correction via %s", turn, maxTurns, gen.ProviderName()))

		healPrompt := FormatCorrectionPrompt(instruction, targetFile, latest, verifyCmd, res.Verification.Output)
		corrected, cErr := gen.Generate(ctx, healPrompt)
		if cErr != nil {
			res.Success = false
			res.Logs = append(res.Logs, fmt.Sprintf("healing turn %d failed: %v", turn, cErr))
			return cErr
		}

		latest = corrected
		res.Output = latest

		if targetFile != "" {
			if err := writeTargetFile(cfg.ProjectPath, targetFile, ExtractCode(latest)); err != nil {
				res.Success = false
				res.Logs = append(res.Logs, fmt.Sprintf("failed writing target file on turn %d: %v", turn, err))
				return err
			}
		}

		vRes, vErr := runVerificationCmdFn(ctx, cfg.ProjectPath, verifyCmd)
		if vErr != nil {
			res.Success = false
			return vErr
		}
		res.Verification = vRes

		if vRes.Passed {
			res.Success = true
			res.Logs = append(res.Logs, fmt.Sprintf("healing passed on turn %d/%d at %s", turn, maxTurns, time.Now().UTC().Format(time.RFC3339)))
			return nil
		}
	}
	res.Success = false
	return nil
}

// ExecuteWithHealing coordinates task execution with automated verification and correction turns.
func ExecuteWithHealing(ctx context.Context, gen Generator, cfg ExecuteConfig) (ExecuteResult, error) {
	res := ExecuteResult{
		Success: true,
	}

	instruction := cfg.Instruction
	targetFile := cfg.TargetFile
	verifyCmd := cfg.VerificationCmd

	if cfg.TaskFilePath != "" {
		fileInst, fileTarget, fileVerify, err := readTaskFile(cfg.TaskFilePath)
		if err != nil {
			return res, fmt.Errorf("read task file %s: %w", cfg.TaskFilePath, err)
		}
		if instruction == "" {
			instruction = fileInst
		}
		if targetFile == "" {
			targetFile = fileTarget
		}
		if verifyCmd == "" {
			verifyCmd = fileVerify
		}
	}

	res.Turns = 1
	var latest string
	var err error

	if gen != nil && strings.TrimSpace(instruction) != "" {
		res.Logs = append(res.Logs, fmt.Sprintf("generating code via %s", gen.ProviderName()))
		latest, err = gen.Generate(ctx, instruction)
		if err != nil {
			res.Success = false
			res.Logs = append(res.Logs, fmt.Sprintf("generation failed: %v", err))
			return res, err
		}
		res.Output = latest

		if targetFile != "" {
			if err := writeTargetFile(cfg.ProjectPath, targetFile, ExtractCode(latest)); err != nil {
				res.Success = false
				res.Logs = append(res.Logs, fmt.Sprintf("failed writing target file %s: %v", targetFile, err))
				return res, err
			}
		}
	}

	if verifyCmd != "" {
		vRes, vErr := runVerificationCmdFn(ctx, cfg.ProjectPath, verifyCmd)
		if vErr != nil {
			res.Success = false
			return res, vErr
		}
		res.Verification = vRes

		if !vRes.Passed {
			res.Logs = append(res.Logs, fmt.Sprintf("verification failed (exit code %d)", vRes.ExitCode))
			if cfg.Heal && gen != nil {
				err := runHealingLoop(ctx, gen, cfg, instruction, targetFile, verifyCmd, latest, &res)
				return res, err
			}
			res.Success = false
			return res, nil
		}
	}

	return res, nil
}
