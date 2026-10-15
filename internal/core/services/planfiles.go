package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// planFile is one plan or task markdown file an agent wrote (.claude/plans/PLAN-*.md,
// .claude/tasks/TASK-*.md, and the same under .nexus), reduced to what the onboarding shows.
// The reading rules match the VS Code extension's (vscode-extension/src/planFiles.ts): front
// matter when there is any, otherwise the first heading and a "Status:" line.
type planFile struct {
	ID, Title, Status string
}

var (
	planIDInName  = regexp.MustCompile(`(?i)^((?:PLAN|TASK)-\d+)`)
	planHeading   = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	planTitleID   = regexp.MustCompile(`(?i)^(?:PLAN|TASK)-\d+\s*[—–:-]\s*`)
	planStatusLn  = regexp.MustCompile(`(?im)^\s*[*_]*status[*_]*\s*:\s*[*_]*\s*([^*_\n][^\n]*?)\s*[*_]*\s*$`)
	planDoneRe    = regexp.MustCompile(`(?i)^(done|completed?|closed|merged|shipped|cancell?ed)$`)
	planActiveRe  = regexp.MustCompile(`(?i)^(active|in[-_ ]?progress|doing|started|wip)$`)
	planNumberEnd = regexp.MustCompile(`(\d+)\s*$`)
)

// parsePlanFile reads one file's text; name supplies the id and title when the file has none.
func parsePlanFile(name, text string) planFile {
	meta, body := planFrontMatter(text)
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	id := meta["id"]
	if id == "" {
		if m := planIDInName.FindStringSubmatch(stem); m != nil {
			id = strings.ToUpper(m[1])
		} else {
			id = stem
		}
	}
	title := meta["title"]
	if title == "" {
		if m := planHeading.FindStringSubmatch(body); m != nil {
			title = strings.TrimSpace(planTitleID.ReplaceAllString(m[1], ""))
		}
	}
	if title == "" {
		title = stem
	}
	status := meta["status"]
	if status == "" {
		if m := planStatusLn.FindStringSubmatch(body); m != nil {
			status = m[1]
		}
	}
	if status == "" {
		status = "unknown"
	}
	return planFile{ID: id, Title: title, Status: status}
}

// planFrontMatter returns the simple "key: value" pairs of a leading --- block and the rest.
func planFrontMatter(text string) (map[string]string, string) {
	meta := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return meta, text
	}
	i := 1
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		k, v, ok := strings.Cut(lines[i], ":")
		if !ok || strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t") {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		meta[strings.TrimSpace(k)] = v
	}
	if i < len(lines) {
		i++
	}
	return meta, strings.Join(lines[min(i, len(lines)):], "\n")
}

// readPlanFiles lists the plan and task files of a project. Unreadable files are skipped.
func readPlanFiles(project string) (plans, tasks []planFile) {
	for _, kind := range []string{"plans", "tasks"} {
		seen := map[string]bool{}
		for _, base := range []string{".claude", ".nexus"} {
			dir := filepath.Join(project, base, kind)
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
					continue
				}
				b, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil {
					continue
				}
				pf := parsePlanFile(e.Name(), string(b))
				if seen[pf.ID] {
					continue
				}
				seen[pf.ID] = true
				if kind == "plans" {
					plans = append(plans, pf)
				} else {
					tasks = append(tasks, pf)
				}
			}
		}
	}
	sort.SliceStable(plans, func(i, j int) bool { return planOrder(plans[i].ID) < planOrder(plans[j].ID) })
	return plans, tasks
}

// planOrder sorts PLAN-2 before PLAN-10; ids without a number sort last.
func planOrder(id string) int {
	if m := planNumberEnd.FindStringSubmatch(id); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 1 << 30
}

// activePlanFromFiles picks the plan one is working on: one marked active, else the newest that
// is not finished. ok is false when there is none.
func activePlanFromFiles(plans []planFile) (p planFile, ok bool) {
	for _, c := range plans {
		if planActiveRe.MatchString(strings.TrimSpace(c.Status)) {
			return c, true
		}
	}
	for i := len(plans) - 1; i >= 0; i-- {
		if !planDoneRe.MatchString(strings.TrimSpace(plans[i].Status)) {
			return plans[i], true
		}
	}
	return planFile{}, false
}

// taskFileSummary counts the task files by state, or "" when there are none.
func taskFileSummary(tasks []planFile) string {
	if len(tasks) == 0 {
		return ""
	}
	var done, doing int
	for _, t := range tasks {
		s := strings.TrimSpace(t.Status)
		switch {
		case planDoneRe.MatchString(s):
			done++
		case planActiveRe.MatchString(s):
			doing++
		}
	}
	return fmt.Sprintf("%d task files (%d done, %d in progress, %d to do)", len(tasks), done, doing, len(tasks)-done-doing)
}
