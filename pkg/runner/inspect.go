package runner

import (
	"context"
	"fmt"
	"strings"
)

// Inspect performs an audit of project invariants, file size boundaries, and git state.
func Inspect(ctx context.Context, root string, pol Policy) (InspectionReport, error) {
	report := InspectionReport{
		ProjectPath: root,
		Passed:      true,
	}

	files, err := scanFilesFn(root, pol)
	if err != nil {
		return report, fmt.Errorf("inspect scan files in %s: %w", root, err)
	}

	report.TotalFiles = len(files)
	for _, f := range files {
		if IsTestFile(f) {
			report.TestFiles++
		} else {
			report.SourceFiles++
		}
	}

	violations, err := checkFileSizesFn(ctx, root, pol, files...)
	if err != nil {
		return report, fmt.Errorf("check file sizes: %w", err)
	}
	report.Violations = append(report.Violations, violations...)

	if pol.EnforceTestFirst {
		testViolations, err := checkTestFirstFn(ctx, root, nil)
		if err != nil {
			return report, fmt.Errorf("check test first: %w", err)
		}
		report.Violations = append(report.Violations, testViolations...)
	}

	if len(report.Violations) > 0 {
		report.Passed = false
	}

	// Capture git status summary if available
	cmd := execCommandContext(ctx, "git", "status", "--short")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil {
		report.GitStatus = strings.TrimSpace(string(out))
	}

	return report, nil
}
