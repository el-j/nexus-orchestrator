package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// PolicyFile is the project-relative location of the rule set for a project.
const PolicyFile = ".nexus/policy.json"

// maxPolicyBytes bounds the file so a huge or hostile policy cannot flood a prompt.
const maxPolicyBytes = 64 << 10

// LoadProjectPolicy reads <project>/.nexus/policy.json. The bool is false when the
// project has no policy file. A file that exists but cannot be used (unreadable,
// not JSON, unknown field, invalid value) is an error: silently ignoring a broken
// rule file would let agents run without the rules the project asked for.
func LoadProjectPolicy(projectPath string) (domain.ProjectPolicy, bool, error) {
	p := filepath.Join(projectPath, filepath.FromSlash(PolicyFile))
	info, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return domain.ProjectPolicy{}, false, nil
	}
	if err != nil {
		return domain.ProjectPolicy{}, false, fmt.Errorf("policy: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxPolicyBytes {
		return domain.ProjectPolicy{}, false, fmt.Errorf("policy: %s must be a regular file of at most %d bytes", PolicyFile, maxPolicyBytes)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return domain.ProjectPolicy{}, false, fmt.Errorf("policy: %w", err)
	}
	var pol domain.ProjectPolicy
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pol); err != nil {
		return domain.ProjectPolicy{}, false, fmt.Errorf("policy: %s: %w", PolicyFile, err)
	}
	if dec.More() {
		return domain.ProjectPolicy{}, false, fmt.Errorf("policy: %s: unexpected data after the JSON object", PolicyFile)
	}
	if err := pol.Validate(); err != nil {
		return domain.ProjectPolicy{}, false, fmt.Errorf("%s: %w", PolicyFile, err)
	}
	return pol, true, nil
}

// baselineRules are the working rules written into a new policy. Each is short on
// purpose: they are repeated in front of the model on every task.
var baselineRules = []string{
	"Read the code you will change before editing it; follow the existing style and layering.",
	"Plan first: state the small steps, then do one step at a time.",
	"Write the failing test before the code that makes it pass.",
	"Test behaviour from the outside (black-box) as well as units.",
	"Keep files small and single-purpose; split instead of growing a file.",
	"Document every exported/public item in the code.",
	"Do not change anything the task does not ask for.",
	"Finish only when the verification command passes.",
	"Report progress through the Nexus tools (claim_task, update_task_status, ingest_knowledge); never edit the Nexus database or task states by hand.",
}

// DefaultPolicy returns a starting policy for the stack found in projectPath.
func DefaultPolicy(projectPath string) domain.ProjectPolicy {
	has := func(name string) bool { _, err := os.Stat(filepath.Join(projectPath, name)); return err == nil }
	var verify []string
	if has("go.mod") {
		verify = append(verify, "go vet ./... && go test ./...")
	}
	if has("package.json") {
		verify = append(verify, "npm test")
	}
	if has("Cargo.toml") {
		verify = append(verify, "cargo test")
	}
	if has("pyproject.toml") || has("requirements.txt") {
		verify = append(verify, "pytest")
	}
	return domain.ProjectPolicy{
		Version:      domain.PolicyVersion,
		Verify:       strings.Join(verify, " && "),
		MaxFileLines: 500,
		TestsFirst:   true,
		Remember:     true,
		Rules:        append([]string(nil), baselineRules...),
	}
}

// EnsureProjectPolicy writes DefaultPolicy to <project>/.nexus/policy.json unless a
// policy file already exists. It reports whether it created the file.
func EnsureProjectPolicy(projectPath string) (bool, error) {
	p := filepath.Join(projectPath, filepath.FromSlash(PolicyFile))
	if _, err := os.Lstat(p); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("policy: %w", err)
	}
	data, err := json.MarshalIndent(DefaultPolicy(projectPath), "", "  ")
	if err != nil {
		return false, fmt.Errorf("policy: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return false, fmt.Errorf("policy: %w", err)
	}
	if err := os.WriteFile(p, append(data, '\n'), 0o644); err != nil {
		return false, fmt.Errorf("policy: %w", err)
	}
	return true, nil
}

// renderPolicy states a policy as a short block of text for a prompt.
func renderPolicy(p domain.ProjectPolicy) string {
	var sb strings.Builder
	sb.WriteString("## Workflow rules for this project\n")
	if p.Verify != "" {
		fmt.Fprintf(&sb, "- Verification command (must pass before you finish): `%s`\n", p.Verify)
	}
	if p.MaxFileLines > 0 {
		fmt.Fprintf(&sb, "- Source files must stay at or under %d lines; a write that exceeds it is refused. Split the code.\n", p.MaxFileLines)
	}
	if p.TestsFirst {
		sb.WriteString("- Tests first: write or extend a test file before you write any source file; source writes are refused until you have.\n")
	}
	for _, r := range p.Rules {
		fmt.Fprintf(&sb, "- %s\n", strings.TrimSpace(r))
	}
	return sb.String()
}

// countLines counts the lines of text the way an editor does: a final line without a newline counts.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// policyWriteViolation returns why an agent write of content to path breaks the
// policy, or "" when it is allowed. wroteTest says whether this run already wrote a test file.
func policyWriteViolation(p domain.ProjectPolicy, path, content string, wroteTest bool) string {
	kind := domain.ClassifyFile(path)
	if kind == domain.FileSource {
		if p.TestsFirst && !wroteTest {
			return fmt.Sprintf("policy: tests first. Write or extend a test file before changing source file %s.", path)
		}
		if p.MaxFileLines > 0 {
			if n := countLines(content); n > p.MaxFileLines {
				return fmt.Sprintf("policy: %s would have %d lines, the limit is %d. Split it into smaller files and write again.", path, n, p.MaxFileLines)
			}
		}
	}
	return ""
}
