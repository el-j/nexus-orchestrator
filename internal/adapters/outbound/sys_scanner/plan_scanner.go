package sys_scanner

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"nexus-orchestrator/internal/core/domain"
)

// ScanPlanFiles scans rootPaths and their well-known subdirectories for plan/task/orchestration files.
func (s *Scanner) ScanPlanFiles(_ context.Context, rootPaths []string) ([]domain.DiscoveredPlanFile, error) {
	home := os.Getenv("HOME")
	seen := map[string]bool{}
	var results []domain.DiscoveredPlanFile

	for _, root := range rootPaths {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		// Scan the root dir itself and immediate subdirs of interest.
		scanDirs := []string{
			absRoot,
			filepath.Join(absRoot, ".claude"),
			filepath.Join(absRoot, ".cursor"),
			filepath.Join(absRoot, ".continue"),
			filepath.Join(absRoot, ".github"),
			filepath.Join(absRoot, ".github", "agents"),
		}
		for _, dir := range scanDirs {
			found, err := scanDir(dir, absRoot, home, seen)
			if err != nil {
				continue
			}
			results = append(results, found...)
		}

		// Recursively discover bounded markdown instruction/prompt artifacts.
		recursiveRoots := []string{
			absRoot,
			filepath.Join(absRoot, ".github"),
		}
		for _, base := range recursiveRoots {
			found, err := scanRecursiveInstructionFiles(base, home, seen, 3)
			if err != nil {
				continue
			}
			results = append(results, found...)
		}
	}
	return results, nil
}

func scanRecursiveInstructionFiles(base, home string, seen map[string]bool, maxDepth int) ([]domain.DiscoveredPlanFile, error) {
	if _, err := os.Stat(base); err != nil {
		return nil, nil
	}

	var results []domain.DiscoveredPlanFile
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == base {
			return nil
		}

		rel, err := filepath.Rel(base, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))

		if d.IsDir() {
			if depth >= maxDepth {
				return filepath.SkipDir
			}
			name := d.Name()
			switch name {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxDepth {
			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".instructions.md") && !strings.HasSuffix(name, ".prompt.md") {
			return nil
		}
		if seen[path] {
			return nil
		}
		seen[path] = true

		pf, err := buildPlanFile(path, domain.PlanFileKindMarkdown, "md", home)
		if err != nil {
			return nil
		}
		pf.ProjectPath = findProjectPath(filepath.Dir(path), home)
		results = append(results, pf)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// scanDir inspects dir for all recognised plan-file patterns.
func scanDir(dir, root, home string, seen map[string]bool) ([]domain.DiscoveredPlanFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []domain.DiscoveredPlanFile

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		fullPath := filepath.Join(dir, name)

		kind, format := classifyFile(fullPath, dir, name)
		if kind == "" {
			continue
		}
		if seen[fullPath] {
			continue
		}
		seen[fullPath] = true

		pf, err := buildPlanFile(fullPath, kind, format, home)
		if err != nil {
			continue
		}
		results = append(results, pf)
	}

	// Handle glob subdirs: .claude/tasks/*.md and .cursor/rules/*.mdc

	// .claude/tasks/*.md
	tasksDir := filepath.Join(dir, "tasks")
	if filepath.Base(dir) == ".claude" {
		taskEntries, err := os.ReadDir(tasksDir)
		if err == nil {
			for _, e := range taskEntries {
				if e.IsDir() {
					continue
				}
				if strings.HasSuffix(e.Name(), ".md") {
					fullPath := filepath.Join(tasksDir, e.Name())
					if seen[fullPath] {
						continue
					}
					seen[fullPath] = true
					pf, err := buildPlanFile(fullPath, domain.PlanFileKindClaudeTask, "md", home)
					if err == nil {
						results = append(results, pf)
					}
				}
			}
		}
	}

	// .cursor/rules/*.mdc
	rulesDir := filepath.Join(dir, "rules")
	if filepath.Base(dir) == ".cursor" {
		ruleEntries, err := os.ReadDir(rulesDir)
		if err == nil {
			for _, e := range ruleEntries {
				if e.IsDir() {
					continue
				}
				if strings.HasSuffix(e.Name(), ".mdc") {
					fullPath := filepath.Join(rulesDir, e.Name())
					if seen[fullPath] {
						continue
					}
					seen[fullPath] = true
					pf, err := buildPlanFile(fullPath, domain.PlanFileKindCursor, "md", home)
					if err == nil {
						results = append(results, pf)
					}
				}
			}
		}
	}

	// .github/agents/*.agent.md
	if filepath.Base(dir) == "agents" && filepath.Base(filepath.Dir(dir)) == ".github" {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(e.Name(), ".agent.md") {
				fullPath := filepath.Join(dir, e.Name())
				if seen[fullPath] {
					continue
				}
				seen[fullPath] = true
				pf, err := buildPlanFile(fullPath, domain.PlanFileKindClaude, "md", home)
				if err == nil {
					results = append(results, pf)
				}
			}
		}
	}

	return results, nil
}

// classifyFile returns the Kind and Format for a given file, or empty strings if not recognised.
func classifyFile(fullPath, dir, name string) (domain.PlanFileKind, string) {
	dirBase := filepath.Base(dir)
	parentBase := filepath.Base(filepath.Dir(dir))

	switch name {
	// .claude/orchestrator.json — only when inside a .claude dir
	case "orchestrator.json":
		if dirBase == ".claude" {
			return domain.PlanFileKindNexus, "json"
		}
	// Claude markdown files
	case "AGENTS.md":
		// AGENTS.md is listed under both claude and markdown; use claude per spec primary,
		// but spec table lists it under markdown too — we use markdown for plain root files,
		// claude for .github/agents context. At root level treat as markdown.
		if dirBase == "agents" && parentBase == ".github" {
			return domain.PlanFileKindClaude, "md"
		}
		return domain.PlanFileKindMarkdown, "md"
	case "CLAUDE.md", "copilot-instructions.md":
		return domain.PlanFileKindClaude, "md"
	case ".windsurfrules":
		return domain.PlanFileKindCursor, "md"
	case ".aider.conf.yml":
		return domain.PlanFileKindMCPConfig, "yaml"
	case "CONVENTIONS.md":
		return domain.PlanFileKindMarkdown, "md"
	case "config.json":
		if dirBase == ".continue" {
			return domain.PlanFileKindMCPConfig, "json"
		}
	case "config.yaml", "config.yml":
		if dirBase == ".continue" {
			return domain.PlanFileKindMCPConfig, "yaml"
		}
	case "tasks.json", "agent.json":
		return domain.PlanFileKindMarkdown, "json"
	case "tasks.yaml", "tasks.yml", "agent.yaml", "agent.yml":
		return domain.PlanFileKindMarkdown, "yaml"
	// Cursor rules
	case ".cursorrules":
		return domain.PlanFileKindCursor, "md"
	// Markdown plan files
	case "TASKS.md", "tasks.md", "TODO.md", "PLAN.md", "ROADMAP.md":
		return domain.PlanFileKindMarkdown, "md"
	// MCP configs
	case "mcp.json", ".mcp.json", "claude_desktop_config.json":
		return domain.PlanFileKindMCPConfig, "json"
	// CrewAI python files
	case "crew.py", "agents.py":
		if containsCrewAI(fullPath) {
			return domain.PlanFileKindCrewAI, "py"
		}
	}
	if strings.HasSuffix(name, ".instructions.md") || strings.HasSuffix(name, ".prompt.md") {
		return domain.PlanFileKindMarkdown, "md"
	}
	if strings.HasSuffix(name, ".md") && looksLikePlanMarkdown(fullPath) {
		return domain.PlanFileKindMarkdown, "md"
	}
	return "", ""
}

// containsCrewAI checks whether the first 20 lines of a Python file contain a crewai import.
func containsCrewAI(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() && lineNum < 20 {
		line := scanner.Text()
		if strings.Contains(line, "crewai") {
			return true
		}
		lineNum++
	}
	return false
}

// summarizeOrchestratorJSON parses an orchestrator.json file and returns a human-readable summary.
func summarizeOrchestratorJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Counters struct {
			NextPlanID int `json:"nextPlanId"`
			NextTaskID int `json:"nextTaskId"`
		} `json:"counters"`
		Plans map[string]struct {
			Status string `json:"status"`
		} `json:"plans"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	totalPlans := len(doc.Plans)
	completedPlans := 0
	for _, p := range doc.Plans {
		if p.Status == "completed" || p.Status == "done" {
			completedPlans++
		}
	}
	totalTasks := doc.Counters.NextTaskID - 1
	if totalTasks < 0 {
		totalTasks = 0
	}
	return fmt.Sprintf("plans: %d (%d completed) · tasks: %d · updated: %s",
		totalPlans, completedPlans, totalTasks, doc.UpdatedAt)
}

// buildPlanFile creates a DiscoveredPlanFile from a path and its classification.
func buildPlanFile(absPath string, kind domain.PlanFileKind, format, home string) (domain.DiscoveredPlanFile, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return domain.DiscoveredPlanFile{}, err
	}

	id := fmt.Sprintf("%x", sha1.Sum([]byte(absPath)))[:12]
	var summary string
	if kind == domain.PlanFileKindNexus {
		summary = summarizeOrchestratorJSON(absPath)
	} else {
		summary = readSummary(absPath)
	}
	isActive := time.Since(fi.ModTime()) < 24*time.Hour
	projectPath := findProjectPath(filepath.Dir(absPath), home)

	return domain.DiscoveredPlanFile{
		ID:           id,
		Path:         absPath,
		Kind:         kind,
		Format:       format,
		ProjectPath:  projectPath,
		Summary:      summary,
		LastModified: fi.ModTime(),
		IsActive:     isActive,
	}, nil
}

// readSummary reads the first 300 bytes, strips non-printable chars, and trims to 200 chars.
func readSummary(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 300)
	n, _ := f.Read(buf)
	raw := string(buf[:n])

	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	summary := strings.TrimSpace(sb.String())
	if len(summary) > 200 {
		summary = summary[:200]
	}
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		summary = summarizeMarkdownHeuristics(raw, summary)
	}
	return summary
}

func summarizeMarkdownHeuristics(raw, summary string) string {
	features := markdownFeatures(raw)
	if len(features) == 0 {
		return summary
	}
	prefix := "[" + strings.Join(features, ", ") + "] "
	out := prefix + summary
	if len(out) > 200 {
		return out[:200]
	}
	return out
}

func markdownFeatures(raw string) []string {
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	features := make([]string, 0, 3)
	if hasYAMLFrontmatter(text) {
		features = append(features, "frontmatter")
	}
	if checklistCount(text) >= 2 {
		features = append(features, "checklist")
	}
	if headingCount(text) >= 2 {
		features = append(features, "headings")
	}
	return features
}

func hasYAMLFrontmatter(text string) bool {
	if !strings.HasPrefix(text, "---\n") {
		return false
	}
	rest := text[4:]
	return strings.Contains(rest, "\n---\n")
}

func checklistCount(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(strings.ToLower(trimmed), "- [x] ") {
			count++
		}
	}
	return count
}

func headingCount(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			count++
		}
	}
	return count
}

func numberedListCount(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) == 0 {
			continue
		}
		// Find end of leading digits.
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		// Need at least one digit, then ". ", then non-whitespace.
		if i > 0 && i+2 <= len(trimmed) && trimmed[i] == '.' && trimmed[i+1] == ' ' {
			if strings.TrimSpace(trimmed[i+2:]) != "" {
				count++
			}
		}
	}
	return count
}

func looksLikePlanMarkdown(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	sample := string(data)
	if len(sample) > 2048 {
		sample = sample[:2048]
	}
	if hasYAMLFrontmatter(sample) || checklistCount(sample) >= 2 || headingCount(sample) >= 5 || numberedListCount(sample) >= 3 {
		return true
	}
	for _, line := range strings.Split(sample, "\n") {
		trimmed := strings.TrimSpace(strings.ToUpper(line))
		if strings.HasPrefix(trimmed, "# PLAN") ||
			strings.HasPrefix(trimmed, "## PLAN") ||
			strings.HasPrefix(trimmed, "# TASK") ||
			strings.HasPrefix(trimmed, "## TASK") ||
			strings.HasPrefix(trimmed, "# ROADMAP") ||
			strings.HasPrefix(trimmed, "## ROADMAP") ||
			strings.HasPrefix(trimmed, "# TODO") ||
			strings.HasPrefix(trimmed, "## TODO") ||
			strings.HasPrefix(trimmed, "# WORKFLOW") ||
			strings.HasPrefix(trimmed, "## WORKFLOW") ||
			strings.HasPrefix(trimmed, "# SPRINT") ||
			strings.HasPrefix(trimmed, "## SPRINT") ||
			strings.HasPrefix(trimmed, "# EPIC") ||
			strings.HasPrefix(trimmed, "## EPIC") ||
			strings.HasPrefix(trimmed, "# MILESTONE") ||
			strings.HasPrefix(trimmed, "## MILESTONE") ||
			strings.HasPrefix(trimmed, "# BACKLOG") ||
			strings.HasPrefix(trimmed, "## BACKLOG") ||
			strings.HasPrefix(trimmed, "# OBJECTIVES") ||
			strings.HasPrefix(trimmed, "## OBJECTIVES") ||
			strings.HasPrefix(trimmed, "# IMPLEMENTATION") ||
			strings.HasPrefix(trimmed, "## IMPLEMENTATION") {
			return true
		}
	}
	return false
}

// findProjectPath walks up from dir until it finds a .git directory or reaches the home dir.
func findProjectPath(dir, home string) string {
	current := dir
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current
		}
		if current == home || current == "/" {
			return dir
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}
