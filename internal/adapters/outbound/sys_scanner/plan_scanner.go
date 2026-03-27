package sys_scanner

import (
	"bufio"
	"context"
	"crypto/sha1"
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
			filepath.Join(absRoot, ".github", "agents"),
		}
		for _, dir := range scanDirs {
			found, err := scanDir(dir, absRoot, home, seen)
			if err != nil {
				continue
			}
			results = append(results, found...)
		}
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

// buildPlanFile creates a DiscoveredPlanFile from a path and its classification.
func buildPlanFile(absPath string, kind domain.PlanFileKind, format, home string) (domain.DiscoveredPlanFile, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return domain.DiscoveredPlanFile{}, err
	}

	id := fmt.Sprintf("%x", sha1.Sum([]byte(absPath)))[:12]
	summary := readSummary(absPath)
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
	return summary
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
