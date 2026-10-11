package sys_scanner

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: non-cryptographic stable id, see the id derivation below
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// ScanPlanFiles scans rootPaths and their well-known subdirectories for plan/task/orchestration files.
func (s *Scanner) ScanPlanFiles(_ context.Context, rootPaths []string) ([]domain.DiscoveredPlanFile, error) {
	home := userHome()
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
			found, err := scanDir(dir, home, seen)
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
func scanDir(dir, home string, seen map[string]bool) ([]domain.DiscoveredPlanFile, error) {
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

// buildPlanFile creates a DiscoveredPlanFile from a path and its classification.
func buildPlanFile(absPath string, kind domain.PlanFileKind, format, home string) (domain.DiscoveredPlanFile, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return domain.DiscoveredPlanFile{}, err
	}

	// Non-cryptographic: a short stable id for a path. Changing the hash would
	// orphan ids already persisted for discovered plan files.
	id := fmt.Sprintf("%x", sha1.Sum([]byte(absPath)))[:12] //nolint:gosec // G401
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
