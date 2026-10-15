package services

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// PlanFile is one plan or task markdown file an agent wrote, reduced to what the onboarding
// shows. Where such files live is not fixed: see ScanPlanFiles. The reading and linking rules
// are the same as in the VS Code extension (vscode-extension/src/planFiles.ts); both are tested
// against testdata/planfiles/cases.json.
type PlanFile struct {
	// ID is the front matter id, else PLAN-n / TASK-n from the file name, else the file name.
	ID, Title, Status string
	// Rel is the project-relative path with forward slashes.
	Rel string
	// PlanRef is the plan a task names itself (front matter planId / plan / plan_id).
	PlanRef string
	// Listed are the task ids a plan lists in its front matter.
	Listed []string
	// BodyPlan is the first PLAN-n a task's text mentions; Mentioned are the TASK-n a plan's text mentions.
	BodyPlan  string
	Mentioned []string
	stem      string
}

// PlanNode is a plan with the tasks linked to it.
type PlanNode struct {
	PlanFile
	Tasks []PlanFile
}

// PlanScan is everything ScanPlanFiles found in one project.
type PlanScan struct {
	// Plans in natural id order (PLAN-2 before PLAN-10), each with its linked tasks.
	Plans []PlanNode
	// Orphans are tasks that no plan claims.
	Orphans []PlanFile
	// Folders are the project-relative folders the files came from, sorted.
	Folders []string
}

var (
	planIDInName = regexp.MustCompile(`(?i)^((?:PLAN|TASK)-\d+)`)
	planHeading  = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	planTitleID  = regexp.MustCompile(`(?i)^(?:PLAN|TASK)-\d+\s*[—–:-]\s*`)
	planStatusLn = regexp.MustCompile(`(?im)^\s*[*_]*status[*_]*\s*:\s*[*_]*\s*([^*_\n][^\n]*?)\s*[*_]*\s*$`)
	planMention  = regexp.MustCompile(`(?i)PLAN-\d+`)
	taskMention  = regexp.MustCompile(`(?i)TASK-\d+`)
	planDoneRe   = regexp.MustCompile(`(?i)^(done|completed?|closed|merged|shipped|cancell?ed)$`)
	planActiveRe = regexp.MustCompile(`(?i)^(active|in[-_ ]?progress|doing|started|wip)$`)
	planNumEnd   = regexp.MustCompile(`(\d+)\s*$`)
)

const (
	// maxPlanFileBytes: a bigger file is not a plan or task document.
	maxPlanFileBytes = 256 << 10
	// maxPlanWalkDepth and maxPlanWalkEntries bound the search of a whole project.
	maxPlanWalkDepth   = 6
	maxPlanWalkEntries = 20000
	// maxPlanFolderDepth is how deep below a plan or task folder files are read.
	maxPlanFolderDepth = 3
)

// planSkipDirs are never searched for plan files.
var planSkipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "build": true, "dist": true, "target": true,
	"__pycache__": true, "coverage": true, "out": true, "tmp": true,
}

// planFrontMatter returns the simple "key: value" pairs of a leading --- block (a list may be
// inline [a, b] or a "- item" block) and the text after it.
func planFrontMatter(text string) (map[string]string, map[string][]string, string) {
	meta, lists := map[string]string{}, map[string][]string{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return meta, lists, text
	}
	unquote := func(v string) string {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
			v = strings.TrimSpace(v[1 : len(v)-1])
		}
		return v
	}
	key := ""
	i := 1
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		line := lines[i]
		if t := strings.TrimLeft(line, " \t"); key != "" && t != line && strings.HasPrefix(t, "- ") {
			lists[key] = append(lists[key], unquote(t[2:]))
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" || strings.ContainsAny(strings.TrimSpace(k), " \t") {
			continue
		}
		key = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
			for _, item := range strings.Split(v[1:len(v)-1], ",") {
				if item = unquote(item); item != "" {
					lists[key] = append(lists[key], item)
				}
			}
			continue
		}
		meta[key] = unquote(v)
	}
	if i < len(lines) {
		i++
	}
	return meta, lists, strings.Join(lines[min(i, len(lines)):], "\n")
}

// parsePlanFile reads one file; name supplies the id and title when the file has none. kind is
// "plan" or "task" when the caller knows it, "" to decide from the file (name, then front matter
// type); ok is false when it is neither.
func parsePlanFile(rel, text, kind string) (pf PlanFile, gotKind string, ok bool) {
	meta, lists, body := planFrontMatter(text)
	name := path.Base(rel)
	stem := strings.TrimSuffix(name, path.Ext(name))
	idFromName := ""
	if m := planIDInName.FindStringSubmatch(stem); m != nil {
		idFromName = strings.ToUpper(m[1])
	}
	if kind == "" {
		switch t := strings.ToLower(meta["type"]); {
		case t == "plan" || t == "task":
			kind = t
		case strings.HasPrefix(idFromName, "PLAN"):
			kind = "plan"
		case strings.HasPrefix(idFromName, "TASK"):
			kind = "task"
		case isPlanDir(path.Base(path.Dir(rel))):
			kind = "plan"
		case isTaskDir(path.Base(path.Dir(rel))):
			kind = "task"
		default:
			return PlanFile{}, "", false
		}
	}
	pf = PlanFile{Rel: rel, stem: stem}
	if pf.ID = meta["id"]; pf.ID == "" {
		if pf.ID = idFromName; pf.ID == "" {
			pf.ID = stem
		}
	}
	if pf.Title = meta["title"]; pf.Title == "" {
		if m := planHeading.FindStringSubmatch(body); m != nil {
			pf.Title = strings.TrimSpace(planTitleID.ReplaceAllString(m[1], ""))
		}
	}
	if pf.Title == "" {
		pf.Title = stem
	}
	if pf.Status = meta["status"]; pf.Status == "" {
		if m := planStatusLn.FindStringSubmatch(body); m != nil {
			pf.Status = m[1]
		}
	}
	if pf.Status == "" {
		pf.Status = "unknown"
	}
	if kind == "plan" {
		pf.Listed = lists["tasks"]
		seen := map[string]bool{}
		for _, m := range taskMention.FindAllString(body, -1) {
			if m = strings.ToUpper(m); !seen[m] {
				seen[m] = true
				pf.Mentioned = append(pf.Mentioned, m)
			}
		}
	} else {
		for _, k := range []string{"planId", "plan", "plan_id"} {
			if pf.PlanRef == "" {
				pf.PlanRef = meta[k]
			}
		}
		pf.BodyPlan = strings.ToUpper(planMention.FindString(body))
	}
	return pf, kind, true
}

func isPlanDir(n string) bool { n = strings.ToLower(n); return n == "plans" || n == "plan" }
func isTaskDir(n string) bool { n = strings.ToLower(n); return n == "tasks" || n == "task" }

// normalizePlanFolders turns configured folders into clean project-relative slash paths.
func normalizePlanFolders(dirs []string) []string {
	var out []string
	for _, d := range dirs {
		d = strings.Trim(strings.ReplaceAll(strings.TrimSpace(d), "\\", "/"), "/")
		if d = path.Clean(d); d != "." && d != "" {
			out = append(out, d)
		}
	}
	return out
}

// ScanPlanFiles finds the plan and task files of a project and links tasks to plans.
//
// Where files are searched, in this order (the first file with an id wins):
//  1. the folders named in the policy (cfg.Plans are plans, cfg.Tasks are tasks, whatever the
//     files are called),
//  2. .claude/{plans,tasks} and .nexus/{plans,tasks},
//  3. unless cfg.Discover is false: any markdown file in the project whose name is PLAN-n or
//     TASK-n, whose folder is called plans/plan or tasks/task, or whose front matter says
//     "type: plan" or "type: task". node_modules, vendor, build output and hidden folders other
//     than .claude, .nexus and .github are not searched.
//
// With cfg.Discover false and folders configured, only step 1 applies.
//
// A task belongs to the plan it names (planId / plan), else to the plan that lists it in its
// front matter, else to the plan whose name its folder carries (tasks/PLAN-2/TASK-5.md), else to
// the first plan its text mentions, else to a plan whose text mentions the task.
func ScanPlanFiles(project string, cfg *domain.PlanFilesPolicy) PlanScan {
	var plansDirs, tasksDirs []string
	discover := true
	if cfg != nil {
		plansDirs, tasksDirs = normalizePlanFolders(cfg.Plans), normalizePlanFolders(cfg.Tasks)
		if cfg.Discover != nil {
			discover = *cfg.Discover
		}
	}
	configured := len(plansDirs)+len(tasksDirs) > 0

	var plans, tasks []PlanFile
	seen := map[string]bool{} // rel paths already read
	ids := map[string]bool{}  // "kind:id" already taken
	add := func(rel, kind string) {
		if seen[rel] {
			return
		}
		seen[rel] = true
		abs := filepath.Join(project, filepath.FromSlash(rel))
		if info, err := os.Stat(abs); err != nil || info.Size() > maxPlanFileBytes {
			return
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return
		}
		pf, k, ok := parsePlanFile(rel, string(b), kind)
		if !ok || ids[k+":"+pf.ID] {
			return
		}
		ids[k+":"+pf.ID] = true
		if k == "plan" {
			plans = append(plans, pf)
		} else {
			tasks = append(tasks, pf)
		}
	}
	addFolder := func(dir, kind string) {
		for _, rel := range markdownUnder(project, dir, maxPlanFolderDepth) {
			add(rel, kind)
		}
	}
	for _, d := range plansDirs {
		addFolder(d, "plan")
	}
	for _, d := range tasksDirs {
		addFolder(d, "task")
	}
	if discover || !configured {
		for _, base := range []string{".claude", ".nexus"} {
			addFolder(base+"/plans", "plan")
			addFolder(base+"/tasks", "task")
		}
	}
	if discover {
		for _, rel := range discoverMarkdown(project) {
			add(rel, "")
		}
	}
	return linkPlanFiles(plans, tasks)
}

// markdownUnder lists the markdown files below dir (project-relative) down to depth levels.
func markdownUnder(project, dir string, depth int) []string {
	var out []string
	root := filepath.Join(project, filepath.FromSlash(dir))
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			if rel != "." && strings.Count(filepath.ToSlash(rel), "/") >= depth {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			out = append(out, path.Join(dir, filepath.ToSlash(rel)))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// discoverMarkdown walks the project for markdown files that might be plans or tasks.
func discoverMarkdown(project string) []string {
	var out []string
	visited := 0
	_ = filepath.WalkDir(project, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if visited++; visited > maxPlanWalkEntries {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(project, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			name := d.Name()
			if rel == "." {
				return nil
			}
			hidden := strings.HasPrefix(name, ".") && name != ".claude" && name != ".nexus" && name != ".github"
			if hidden || planSkipDirs[name] || strings.Count(rel, "/") >= maxPlanWalkDepth {
				return filepath.SkipDir
			}
			return nil
		}
		low := strings.ToLower(d.Name())
		if strings.HasSuffix(low, ".md") && low != "readme.md" && low != "index.md" && low != "changelog.md" {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func planOrder(id string) int {
	if m := planNumEnd.FindStringSubmatch(id); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 1 << 30
}

// linkPlanFiles assigns every task to at most one plan.
func linkPlanFiles(plans, tasks []PlanFile) PlanScan {
	sort.SliceStable(plans, func(i, j int) bool {
		a, b := planOrder(plans[i].ID), planOrder(plans[j].ID)
		if a != b {
			return a < b
		}
		return plans[i].ID < plans[j].ID
	})
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := planOrder(tasks[i].ID), planOrder(tasks[j].ID)
		if a != b {
			return a < b
		}
		return tasks[i].ID < tasks[j].ID
	})
	nodes := make([]PlanNode, len(plans))
	for i, p := range plans {
		nodes[i].PlanFile = p
	}
	byRef := func(ref string) int {
		for i, p := range plans {
			if ref != "" && (strings.EqualFold(p.ID, ref) || strings.EqualFold(p.stem, ref)) {
				return i
			}
		}
		return -1
	}
	listedBy := func(id string) int {
		for i, p := range plans {
			for _, l := range p.Listed {
				if strings.EqualFold(l, id) {
					return i
				}
			}
		}
		return -1
	}
	mentionedBy := func(id string) int {
		for i, p := range plans {
			for _, m := range p.Mentioned {
				if strings.EqualFold(m, id) {
					return i
				}
			}
		}
		return -1
	}
	var orphans []PlanFile
	for _, t := range tasks {
		at := byRef(t.PlanRef)
		if at < 0 {
			at = listedBy(t.ID)
		}
		if at < 0 {
			at = byRef(path.Base(path.Dir(t.Rel)))
		}
		if at < 0 {
			at = byRef(t.BodyPlan)
		}
		if at < 0 {
			at = mentionedBy(t.ID)
		}
		if at < 0 {
			orphans = append(orphans, t)
			continue
		}
		nodes[at].Tasks = append(nodes[at].Tasks, t)
	}
	// A task a plan lists or mentions but that has no file is still part of the plan.
	for i := range nodes {
		have := map[string]bool{}
		for _, t := range nodes[i].Tasks {
			have[strings.ToUpper(t.ID)] = true
		}
		missing := append(append([]string{}, nodes[i].Listed...), nodes[i].Mentioned...)
		for _, id := range missing {
			if key := strings.ToUpper(id); !have[key] && !taskHasFile(tasks, id) {
				have[key] = true
				nodes[i].Tasks = append(nodes[i].Tasks, PlanFile{ID: id, Title: id, Status: "unknown"})
			}
		}
		sort.SliceStable(nodes[i].Tasks, func(a, b int) bool { return planOrder(nodes[i].Tasks[a].ID) < planOrder(nodes[i].Tasks[b].ID) })
	}
	folders := map[string]bool{}
	for _, f := range append(append([]PlanFile{}, plans...), tasks...) {
		folders[path.Dir(f.Rel)] = true
	}
	var dirs []string
	for d := range folders {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return PlanScan{Plans: nodes, Orphans: orphans, Folders: dirs}
}

func taskHasFile(tasks []PlanFile, id string) bool {
	for _, t := range tasks {
		if strings.EqualFold(t.ID, id) {
			return true
		}
	}
	return false
}

// flat returns the plans and all tasks (linked or not) as plain lists.
func (s PlanScan) flat() (plans, tasks []PlanFile) {
	for _, n := range s.Plans {
		plans = append(plans, n.PlanFile)
		tasks = append(tasks, n.Tasks...)
	}
	return plans, append(tasks, s.Orphans...)
}

// activePlanFromFiles picks the plan one is working on: one marked active, else the newest that
// is not finished. ok is false when there is none.
func activePlanFromFiles(plans []PlanFile) (p PlanFile, ok bool) {
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
	return PlanFile{}, false
}

// taskFileSummary counts the task files by state, or "" when there are none.
func taskFileSummary(tasks []PlanFile) string {
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
