# pkg/runner

Package `runner` provides standalone task verification, architectural policy enforcement, invariant audits, and self-healing code execution loops.

It is designed as a reusable library with zero dependencies on background daemons, databases, or specific web frameworks. Both `cmd/nexus-runner` and the central `nexus-daemon` execution engine consume this package.

---

## Capabilities

1. **Policy Enforcement (`policy.go`)**:
   - Source code file-size limits (default $\le 500$ lines).
   - Test file-size limits (default $\le 800$ lines).
   - Test-first file structure checks.
   - Configurable ignore patterns and regex overrides (`.nexus/policy.json`).

2. **Verification Gates (`verifier.go`)**:
   - Executes shell commands (e.g. `go test -v ./...`, `npm test`, `golangci-lint run`).
   - Captures output, execution duration, and exit status.
   - Evaluates policy violations alongside command exit codes.

3. **Self-Healing Execution (`executor.go`)**:
   - Runs an iterative test/fix loop with a language model generator.
   - Extracts code blocks from LLM markdown responses (`ExtractCode`).
   - Formats targeted compiler and test error correction prompts (`FormatCorrectionPrompt`).
   - Re-applies fixes and re-verifies until passing or reaching maximum attempts (`ExecuteWithHealing`).

4. **Invariant Repository Audits (`inspect.go`)**:
   - Scans workspaces for line-count compliance across all tracked source and test files.
   - Audits git working tree status for unstaged or uncommitted modifications.

---

## Quick Example: Using `pkg/runner` in Go

```go
package main

import (
	"context"
	"fmt"
	"os"

	"nexus-orchestrator/pkg/runner"
)

func main() {
	ctx := context.Background()

	// 1. Load policy or use defaults
	policy, err := runner.LoadPolicy(".nexus/policy.json")
	if err != nil {
		policy = runner.DefaultPolicy()
	}

	// 2. Run verification command with policy checks
	result, err := runner.Verify(ctx, ".", policy, "go test ./...")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
	}

	if result.Passed {
		fmt.Println("All verification checks and policies passed!")
	} else {
		fmt.Printf("Command failed (exit code %d):\n%s\n", result.ExitCode, result.Output)
		for _, v := range result.Violations {
			fmt.Printf("Policy violation: %s (%s)\n", v.Rule, v.File)
		}
	}
}
```

---

## Invariant Inspection Example

```go
report, err := runner.Inspect(ctx, ".", policy)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Scanned %d files (%d source, %d test)\n",
	report.TotalFiles, report.SourceFiles, report.TestFiles)

if len(report.Violations) > 0 {
	for _, v := range report.Violations {
		fmt.Printf("VIOLATION: %s - %s\n", v.File, v.Message)
	}
}
```
