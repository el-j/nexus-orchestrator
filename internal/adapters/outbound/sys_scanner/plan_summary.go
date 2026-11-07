package sys_scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

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

// readSummary reads the first 300 bytes, strips non-printable chars, and trims to 200 chars.
func readSummary(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 300)
	n, _ := f.Read(buf)
	// The 300-byte window may end inside a multi-byte character; drop the fragment.
	raw := strings.ToValidUTF8(string(buf[:n]), "")

	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	summary := cutUTF8(strings.TrimSpace(sb.String()), 200)
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		summary = summarizeMarkdownHeuristics(raw, summary)
	}
	return summary
}

// cutUTF8 returns the longest prefix of s that is at most maxBytes long and does
// not end in the middle of a multi-byte character.
func cutUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

// summarizeMarkdownHeuristics adds what the markdown contains (frontmatter, checklist, headings,
// numbered steps) to the summary of a plan.
func summarizeMarkdownHeuristics(raw, summary string) string {
	features := markdownFeatures(raw)
	if len(features) == 0 {
		return summary
	}
	prefix := "[" + strings.Join(features, ", ") + "] "
	return cutUTF8(prefix+summary, 200)
}

// markdownFeatures names the plan-like parts a markdown text has.
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

// hasYAMLFrontmatter reports whether the text starts with a closed YAML frontmatter block.
func hasYAMLFrontmatter(text string) bool {
	if !strings.HasPrefix(text, "---\n") {
		return false
	}
	rest := text[4:]
	return strings.Contains(rest, "\n---\n")
}

// checklistCount counts the "- [ ]" and "- [x]" items.
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

// headingCount counts headings of the first three levels.
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

// numberedListCount counts the lines that start a numbered list item.
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

// looksLikePlanMarkdown reports whether the first 2 KiB of the file read like a plan: a frontmatter block,
// a checklist, many headings or a numbered list, or a heading such as "# Plan", "# Roadmap" or "# Backlog".
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
		if hasPlanHeading(strings.TrimSpace(strings.ToUpper(line))) {
			return true
		}
	}
	return false
}

// planHeadingWords are the words after "# " or "## " that mark a document as a plan.
var planHeadingWords = []string{"PLAN", "TASK", "ROADMAP", "TODO", "WORKFLOW", "SPRINT", "EPIC", "MILESTONE", "BACKLOG", "OBJECTIVES", "IMPLEMENTATION"}

// hasPlanHeading reports whether an upper-cased, trimmed line is a level 1 or 2 heading that starts with
// one of planHeadingWords.
func hasPlanHeading(line string) bool {
	for _, w := range planHeadingWords {
		if strings.HasPrefix(line, "# "+w) || strings.HasPrefix(line, "## "+w) {
			return true
		}
	}
	return false
}
