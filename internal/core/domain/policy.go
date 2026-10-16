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
	// QuotaThreshold is the share (0 to 1) of a provider's plan allowance at which routing prefers the
	// other providers of the chain; 0 means DefaultQuotaThreshold. 1 only avoids a provider whose
	// allowance is used up.
	QuotaThreshold float64 `json:"quotaThreshold,omitempty"`
	// Ladder orders the models a role may use, cheapest first, as "provider/model" patterns where *
	// matches any text: {"coder": ["Ollama/*", "Anthropic/claude-sonnet*", "Anthropic/claude-opus*"]}.
	// A task starts on the first rung that is available (not cooling down, under the quota threshold)
	// and a failed verification asks the next rung up. The key "default" serves roles without an entry.
	Ladder map[string][]string `json:"ladder,omitempty"`
	// ContextBudget caps the tokens a task's context files may take in the prompt (0: no cap). Over
	// the cap the largest files are sent as outlines (their top-level declarations) and, if that is
	// not enough, shortened, each with a note saying so.
	ContextBudget int `json:"contextBudget,omitempty"`
	// HistoryBudget caps the tokens of session history sent with a task (0: only the newest
	// MaxHistoryMessages messages are sent). Older messages are dropped first.
	HistoryBudget int `json:"historyBudget,omitempty"`
	// PlanFiles says where the project keeps its plan and task files; unset, they are found by themselves.
	PlanFiles *PlanFilesPolicy `json:"planFiles,omitempty"`
}

// DefaultQuotaThreshold is the allowance share at which routing starts to prefer other providers.
const DefaultQuotaThreshold = 0.9

// QuotaLimit returns the quota threshold routing uses: the policy's, or DefaultQuotaThreshold.
func (p ProjectPolicy) QuotaLimit() float64 {
	if p.QuotaThreshold > 0 {
		return p.QuotaThreshold
	}
	return DefaultQuotaThreshold
}

// MaxTokenBudget is the largest ContextBudget or HistoryBudget a policy may set.
const MaxTokenBudget = 200000

// Limits of a ladder.
const (
	// MaxLadderRoles is how many roles a policy may give a ladder.
	MaxLadderRoles = 20
	// MaxLadderRungs is how many rungs one ladder may have.
	MaxLadderRungs = 10
	// MaxRungLen is the length of one rung pattern in characters.
	MaxRungLen = 128
)

// LadderFor returns the ladder of role (case-insensitive), the "default" ladder when the role has
// none, or nil.
func (p ProjectPolicy) LadderFor(role string) []string {
	for _, key := range []string{strings.ToLower(strings.TrimSpace(role)), "default"} {
		for k, rungs := range p.Ladder {
			if key != "" && strings.EqualFold(k, key) {
				return rungs
			}
		}
	}
	return nil
}

// RungMatches reports whether a ladder rung ("provider/model", * matches any text, case-insensitive)
// fits the provider and model.
func RungMatches(rung, provider, model string) bool {
	rung = strings.ToLower(rung)
	if !strings.Contains(rung, "/") {
		return false
	}
	return globMatch(rung, strings.ToLower(provider)+"/"+strings.ToLower(model))
}

// globMatch matches s against a pattern in which * stands for any run of characters.
func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	last := parts[len(parts)-1]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(s, mid)
		if i < 0 {
			return false
		}
		s = s[i+len(mid):]
	}
	return strings.HasSuffix(s, last) && len(s) >= len(last)
}

func validateLadder(l map[string][]string) error {
	if len(l) > MaxLadderRoles {
		return fmt.Errorf("policy: ladders for at most %d roles are allowed, got %d", MaxLadderRoles, len(l))
	}
	for role, rungs := range l {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("policy: a ladder has a blank role")
		}
		if len(rungs) == 0 || len(rungs) > MaxLadderRungs {
			return fmt.Errorf("policy: the ladder of %q needs 1 to %d rungs, got %d", role, MaxLadderRungs, len(rungs))
		}
		seen := map[string]bool{}
		for _, r := range rungs {
			provider, model, ok := strings.Cut(strings.TrimSpace(r), "/")
			switch {
			case !ok || strings.TrimSpace(provider) == "" || strings.TrimSpace(model) == "":
				return fmt.Errorf("policy: rung %q of %q must look like provider/model (* matches anything)", r, role)
			case len(r) > MaxRungLen:
				return fmt.Errorf("policy: a rung of %q is longer than %d characters", role, MaxRungLen)
			case seen[strings.ToLower(r)]:
				return fmt.Errorf("policy: rung %q appears twice in the ladder of %q", r, role)
			}
			seen[strings.ToLower(r)] = true
		}
	}
	return nil
}

// MaxPlanFolders bounds how many folders a policy may name for plans and for tasks each.
const MaxPlanFolders = 20

// PlanFilesPolicy configures where plan and task markdown files live. Everything is optional.
// The defaults (.claude and .nexus plans/tasks folders, and any PLAN-n / TASK-n file anywhere in
// the project) need no configuration; this block adds folders for projects with another layout
// or switches the search of the whole project off.
type PlanFilesPolicy struct {
	// Plans are project-relative folders whose markdown files are plans.
	Plans []string `json:"plans,omitempty"`
	// Tasks are project-relative folders whose markdown files are tasks.
	Tasks []string `json:"tasks,omitempty"`
	// Discover is true (the default when unset) to also find plan and task files anywhere in the
	// project; false limits the search to the listed folders, or to the defaults when none are listed.
	Discover *bool `json:"discover,omitempty"`
}

// validatePlanFolders checks that every entry is a project-relative path that stays inside the project.
func validatePlanFolders(kind string, dirs []string) error {
	if len(dirs) > MaxPlanFolders {
		return fmt.Errorf("policy: at most %d %s folders are allowed, got %d", MaxPlanFolders, kind, len(dirs))
	}
	for _, d := range dirs {
		clean := strings.Trim(strings.ReplaceAll(strings.TrimSpace(d), "\\", "/"), "/")
		switch {
		case clean == "":
			return fmt.Errorf("policy: a %s folder is blank", kind)
		case len(clean) > 256:
			return fmt.Errorf("policy: a %s folder is longer than 256 characters", kind)
		case strings.HasPrefix(strings.TrimSpace(d), "/") || strings.HasPrefix(strings.TrimSpace(d), "\\") || (len(clean) > 1 && clean[1] == ':'):
			return fmt.Errorf("policy: %s folder %q must be relative to the project", kind, d)
		}
		for _, part := range strings.Split(clean, "/") {
			if part == ".." {
				return fmt.Errorf("policy: %s folder %q must stay inside the project", kind, d)
			}
		}
	}
	return nil
}

// Validate reports the first problem that would make the policy unsafe or too large to use.
func (p ProjectPolicy) Validate() error {
	if p.Version != 0 && p.Version != PolicyVersion {
		return fmt.Errorf("policy: unsupported version %d (this daemon reads version %d)", p.Version, PolicyVersion)
	}
	if p.MaxFileLines < 0 {
		return fmt.Errorf("policy: maxFileLines must not be negative")
	}
	if err := validateLadder(p.Ladder); err != nil {
		return err
	}
	if p.ContextBudget < 0 || p.ContextBudget > MaxTokenBudget || p.HistoryBudget < 0 || p.HistoryBudget > MaxTokenBudget {
		return fmt.Errorf("policy: contextBudget and historyBudget must be between 0 and %d tokens", MaxTokenBudget)
	}
	if p.QuotaThreshold < 0 || p.QuotaThreshold > 1 {
		return fmt.Errorf("policy: quotaThreshold must be between 0 and 1")
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
	if p.PlanFiles != nil {
		if err := validatePlanFolders("plan", p.PlanFiles.Plans); err != nil {
			return err
		}
		if err := validatePlanFolders("task", p.PlanFiles.Tasks); err != nil {
			return err
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
