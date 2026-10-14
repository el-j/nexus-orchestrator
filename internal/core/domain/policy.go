package domain

import (
	"fmt"
	"path"
	"strings"
)

// Limits that keep a policy cheap to put in front of a model on every task.
const (
	// PolicyVersion is the only policy file format version understood today.
	PolicyVersion = 1
	// MaxPolicyRules caps how many free-text rules a policy may carry.
	MaxPolicyRules = 30
	// MaxPolicyRuleLen caps the length of one rule in characters.
	MaxPolicyRuleLen = 300
	// MaxPolicyVerifyLen caps the verification command length.
	MaxPolicyVerifyLen = 500
)

// ProjectPolicy is the per-project rule set kept in <project>/.nexus/policy.json.
// It lives in the repository so every agent, model and machine that opens the
// project gets the same rules, including a fresh session with no memory.
//
// The engine enforces the structured fields (Verify, MaxFileLines, TestsFirst)
// on agentic tasks and states all of it, with Rules, in the prompt of every task.
type ProjectPolicy struct {
	// Version is the file format version; 0 is read as PolicyVersion.
	Version int `json:"version,omitempty"`
	// Verify is the default verification command for tasks that do not set one.
	Verify string `json:"verify,omitempty"`
	// MaxFileLines rejects an agent write that would leave a file longer than this; 0 disables it.
	MaxFileLines int `json:"maxFileLines,omitempty"`
	// TestsFirst makes an agent write a test file before it may write source code.
	TestsFirst bool `json:"testsFirst,omitempty"`
	// Remember stores a short outcome entry in the project knowledge after each finished task.
	Remember bool `json:"remember,omitempty"`
	// Rules are free-text rules shown to the model verbatim.
	Rules []string `json:"rules,omitempty"`
}

// Validate reports the first problem that would make the policy unsafe or too large to use.
func (p ProjectPolicy) Validate() error {
	if p.Version != 0 && p.Version != PolicyVersion {
		return fmt.Errorf("policy: unsupported version %d (this daemon reads version %d)", p.Version, PolicyVersion)
	}
	if p.MaxFileLines < 0 {
		return fmt.Errorf("policy: maxFileLines must not be negative")
	}
	if strings.ContainsAny(p.Verify, "\r\n") || len(p.Verify) > MaxPolicyVerifyLen {
		return fmt.Errorf("policy: verify must be a single line of at most %d characters", MaxPolicyVerifyLen)
	}
	if len(p.Rules) > MaxPolicyRules {
		return fmt.Errorf("policy: at most %d rules are allowed, got %d", MaxPolicyRules, len(p.Rules))
	}
	for i, r := range p.Rules {
		switch {
		case strings.TrimSpace(r) == "":
			return fmt.Errorf("policy: rule %d is blank", i+1)
		case len(r) > MaxPolicyRuleLen:
			return fmt.Errorf("policy: rule %d is longer than %d characters", i+1, MaxPolicyRuleLen)
		}
	}
	return nil
}

// FileKind says what role a file plays for the tests-first rule.
type FileKind int

// The three kinds of file ClassifyFile distinguishes.
const (
	// FileOther is documentation, configuration, fixtures and anything not code.
	FileOther FileKind = iota
	// FileSource is production source code.
	FileSource
	// FileTest is an automated test.
	FileTest
)

var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".vue": true, ".py": true, ".rs": true, ".java": true, ".kt": true, ".cs": true, ".rb": true,
	".php": true, ".swift": true, ".c": true, ".cc": true, ".cpp": true, ".h": true, ".hpp": true,
}

// ClassifyFile sorts a project-relative path into test, source or other by its name and directory.
func ClassifyFile(p string) FileKind {
	lower := strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
	ext := path.Ext(lower)
	if !sourceExts[ext] {
		return FileOther
	}
	for _, seg := range strings.Split(path.Dir(lower), "/") {
		switch seg {
		case "testdata", "fixtures", "__fixtures__":
			return FileOther
		case "test", "tests", "__tests__", "e2e":
			return FileTest
		}
	}
	base := strings.TrimSuffix(path.Base(lower), ext)
	switch {
	case strings.HasSuffix(base, "_test"), strings.HasSuffix(base, ".test"), strings.HasSuffix(base, ".spec"),
		strings.HasPrefix(base, "test_"):
		return FileTest
	}
	return FileSource
}
