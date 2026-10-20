package docscheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const root = "../.."

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// documents returns the repository's human-facing text files.
func documents(t *testing.T) []string {
	t.Helper()
	var out []string
	skip := map[string]bool{"node_modules": true, ".git": true, "dist": true, "build": true, ".claude": true, "vendor": true}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".md", ".vue", ".html":
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// The repository is el-j/nexus-orchestrator. GitHub redirects the old name, but
// badges (goreportcard) and tooling that does not follow redirects break.
func TestNoDocumentUsesTheOldRepositoryName(t *testing.T) {
	for _, path := range documents(t) {
		b, _ := os.ReadFile(path)
		if strings.Contains(string(b), "el-j/nexusOrchestrator") {
			t.Errorf("%s still links to el-j/nexusOrchestrator; the repository is el-j/nexus-orchestrator", path)
		}
	}
}

func TestReadmeVersionBadgeIsNotHardCoded(t *testing.T) {
	if strings.Contains(read(t, "README.md"), "img.shields.io/badge/version-") {
		t.Error("README pins a version badge by hand; use the dynamic github/v/release badge")
	}
}

// docs/index.html is the Vite entry point of the docs site; every other .html in
// docs/ would be an orphaned copy of a page the Vue app already renders, drifting
// silently (the old downloads.html linked to release files that never existed).
func TestDocsFolderHoldsNoOrphanedHTMLPages(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(root, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".html") && e.Name() != "index.html" {
			t.Errorf("docs/%s is an orphaned static page; the site is built from docs/src", e.Name())
		}
	}
}

// releaseFacts extracts, from the release workflow, the names it can publish.
type releaseFacts struct {
	oses, arches map[string]bool
	desktop      map[string]bool // archive_name values
	vsix         string
}

func parseRelease(t *testing.T) releaseFacts {
	t.Helper()
	wf := read(t, ".github/workflows/publish.yml")
	f := releaseFacts{oses: map[string]bool{}, arches: map[string]bool{}, desktop: map[string]bool{}}
	for _, m := range regexp.MustCompile(`(?m)^\s*(?:-\s+)?os:\s*(\w+)\s*$`).FindAllStringSubmatch(wf, -1) {
		f.oses[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`(?m)^\s*(?:-\s+)?arch:\s*(\w+)\s*$`).FindAllStringSubmatch(wf, -1) {
		f.arches[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`(?m)^\s+archive_name:\s*(\S+)\s*$`).FindAllStringSubmatch(wf, -1) {
		f.desktop[m[1]] = true
	}
	if m := regexp.MustCompile(`--out \.\./build/vscode/([\w.-]+\.vsix)`).FindStringSubmatch(wf); m != nil {
		f.vsix = m[1]
	}
	return f
}

func TestDownloadsPageNamesTheFilesTheReleaseWorkflowPublishes(t *testing.T) {
	facts := parseRelease(t)
	if len(facts.oses) == 0 || len(facts.arches) == 0 || len(facts.desktop) == 0 || facts.vsix == "" {
		t.Fatalf("could not read the release naming from publish.yml: %+v", facts)
	}
	// The page is its template plus the static asset tables it imports.
	page := read(t, "docs/src/views/DownloadsView.vue") + read(t, "docs/src/data/downloads.ts")

	files := regexp.MustCompile(`file:\s*'([^']+)'`).FindAllStringSubmatch(page, -1)
	if len(files) == 0 {
		t.Fatal("no download files found in DownloadsView.vue")
	}
	ext := regexp.MustCompile(`\.(tar\.gz|zip)$`)
	cli := regexp.MustCompile(`^nexus-orchestrator-(\w+)-(\w+)$`)
	for _, m := range files {
		name := m[1]
		base := ext.ReplaceAllString(name, "")
		switch {
		case facts.desktop[base]:
		case cli.MatchString(base) && !strings.Contains(base, "desktop"):
			g := cli.FindStringSubmatch(base)
			if !facts.oses[g[1]] || !facts.arches[g[2]] {
				t.Errorf("DownloadsView lists %s but publish.yml never builds %s/%s", name, g[1], g[2])
			}
		default:
			t.Errorf("DownloadsView lists %s, which publish.yml does not produce", name)
		}
	}

	if !strings.Contains(page, "/"+facts.vsix) || !strings.Contains(page, "install-extension "+facts.vsix) {
		t.Errorf("DownloadsView must link and install %s (the file the release publishes)", facts.vsix)
	}
	if !strings.Contains(page, "SHA256SUMS.txt") {
		t.Error("DownloadsView must point at SHA256SUMS.txt")
	}
}

func TestReadmeNamesTheReleaseFilesThatExist(t *testing.T) {
	facts := parseRelease(t)
	readme := read(t, "README.md")
	if !strings.Contains(readme, facts.vsix) {
		t.Errorf("README should tell people to install %s", facts.vsix)
	}
	for _, m := range regexp.MustCompile("`(nexus-orchestrator-[\\w.-]+\\.(?:zip|tar\\.gz))`").FindAllStringSubmatch(readme, -1) {
		base := regexp.MustCompile(`\.(tar\.gz|zip)$`).ReplaceAllString(m[1], "")
		if facts.desktop[base] {
			continue
		}
		g := regexp.MustCompile(`^nexus-orchestrator-(\w+)-(\w+)$`).FindStringSubmatch(base)
		if len(g) < 3 || !facts.oses[g[1]] || !facts.arches[g[2]] {
			t.Errorf("README names %s, which publish.yml does not produce", m[1])
		}
	}
}

func TestReleaseWorkflowEmbedsFullUIInCLIDaemonAndEnforcesCoverageGate(t *testing.T) {
	wf := read(t, ".github/workflows/publish.yml")
	buildCLIIdx := strings.Index(wf, "  build-cli:")
	if buildCLIIdx < 0 {
		t.Fatal("publish.yml has no build-cli job")
	}
	buildSection := wf[buildCLIIdx:]
	if nextJob := strings.Index(buildSection, "\n  build-desktop:"); nextJob > 0 {
		buildSection = buildSection[:nextJob]
	}
	if !strings.Contains(buildSection, "internal/adapters/inbound/webui/dist") {
		t.Error("publish.yml build-cli job must copy built frontend into internal/adapters/inbound/webui/dist before building nexus-daemon")
	}
	if !strings.Contains(wf, "MIN_COVERAGE=96 ./scripts/go-coverage.sh") {
		t.Error("publish.yml must enforce MIN_COVERAGE=96 to match ci.yml")
	}

	ci := read(t, ".github/workflows/ci.yml")
	for _, want := range []string{"http://wails.localhost", "wails://wails", `cd "$HOME"`, `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go build ./cmd/nexus-daemon/...`} {
		if !strings.Contains(ci, want) {
			t.Errorf("ci.yml must verify %s", want)
		}
	}
}

func TestDockerfileAndReleaseConfigInvariants(t *testing.T) {
	dockerfile := read(t, "Dockerfile")
	for _, want := range []string{"-DSQLITE_ENABLE_FTS5", "internal/adapters/inbound/webui/dist"} {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("Dockerfile must include %s", want)
		}
	}

	rpCfg := read(t, "release-please-config.json")
	if strings.Contains(rpCfg, `"release-as"`) {
		t.Error("release-please-config.json must not pin release-as")
	}
	for _, want := range []string{"frontend/package.json", "vscode-extension/package.json", "github-action/package.json", "docs/package.json", "wails.json"} {
		if !strings.Contains(rpCfg, want) {
			t.Errorf("release-please-config.json must include %s in extra-files", want)
		}
	}

	vsync := read(t, "scripts/version-sync.sh")
	for _, want := range []string{"frontend/package.json", "vscode-extension/package.json", "github-action/package.json", "docs/package.json", "wails.json"} {
		if !strings.Contains(vsync, want) {
			t.Errorf("scripts/version-sync.sh must stamp %s", want)
		}
	}

	wf := read(t, ".github/workflows/publish.yml")
	desktopIdx := strings.Index(wf, "  build-desktop:")
	vscodeIdx := strings.Index(wf, "\n  build-vscode:")
	smokeIdx := strings.Index(wf, "\n  smoke-test:")
	if desktopIdx < 0 || vscodeIdx < 0 || smokeIdx < 0 {
		t.Fatal("publish.yml missing build-desktop, build-vscode, or smoke-test job")
	}
	desktopJob := wf[desktopIdx:vscodeIdx]
	if !strings.Contains(desktopJob, "scripts/version-sync.sh") {
		t.Error("publish.yml build-desktop must run scripts/version-sync.sh before wails build")
	}
	vscodeJob := wf[vscodeIdx:smokeIdx]
	syncPos := strings.Index(vscodeJob, "scripts/version-sync.sh")
	buildPos := strings.Index(vscodeJob, "npm run build")
	if syncPos < 0 || buildPos < 0 || syncPos > buildPos {
		t.Error("publish.yml build-vscode must run scripts/version-sync.sh before npm run build")
	}
}
