package mcp

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// These tests keep the prose honest: the tool catalog is the source of truth, and
// documents that quote it must not drift (they did: README, AUDIT and the API
// reference disagreed, and one tool was undocumented entirely).

func readDoc(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile("../../../../" + rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func catalogNames() map[string]bool {
	names := map[string]bool{}
	for _, tool := range toolList() {
		names[tool.Name] = true
	}
	return names
}

// mcpSection returns the "MCP tools" part of the API reference, from its first
// group heading to the end of the last table.
func mcpSection(t *testing.T) string {
	doc := readDoc(t, "docs/api-reference.md")
	i := strings.Index(doc, "#### Tasks (")
	if i < 0 {
		t.Fatal("api-reference.md has no MCP tool groups")
	}
	rest := doc[i:]
	// the tool tables end at the next level-2/3 heading
	if j := regexp.MustCompile(`(?m)^#{1,3} `).FindStringIndex(rest); j != nil {
		rest = rest[:j[0]]
	}
	return rest
}

func TestAPIReferenceDocumentsExactlyTheRealTools(t *testing.T) {
	documented := map[string]bool{}
	for _, m := range regexp.MustCompile("(?m)^\\| `([a-z_]+)`").FindAllStringSubmatch(mcpSection(t), -1) {
		documented[m[1]] = true
	}
	real := catalogNames()
	for name := range real {
		if !documented[name] {
			t.Errorf("tool %q exists but is missing from docs/api-reference.md", name)
		}
	}
	for name := range documented {
		if !real[name] {
			t.Errorf("docs/api-reference.md documents %q, which is not a tool", name)
		}
	}
}

func TestAPIReferenceGroupCountsAddUpToTheCatalog(t *testing.T) {
	section := mcpSection(t)
	heading := regexp.MustCompile(`(?m)^#### (.+) \((\d+)\)$`)
	groups := heading.FindAllStringSubmatchIndex(section, -1)
	if len(groups) == 0 {
		t.Fatal("no tool groups found")
	}
	total := 0
	for i, g := range groups {
		name := section[g[2]:g[3]]
		claimed, _ := strconv.Atoi(section[g[4]:g[5]])
		end := len(section)
		if i+1 < len(groups) {
			end = groups[i+1][0]
		}
		rows := len(regexp.MustCompile("(?m)^\\| `[a-z_]+`").FindAllString(section[g[1]:end], -1))
		if rows != claimed {
			t.Errorf("group %q says %d tools but lists %d", name, claimed, rows)
		}
		total += rows
	}
	if total != len(toolList()) {
		t.Errorf("groups list %d tools, the catalog has %d", total, len(toolList()))
	}
}

func TestDocsQuotingAToolCountMatchTheCatalog(t *testing.T) {
	want := len(toolList())
	pattern := regexp.MustCompile(`\b(\d+) (?:MCP )?tools\b`)
	for _, doc := range []string{"README.md", "docs/src/views/HomeView.vue", "docs/getting-started.md", "docs/mcp-integration.md", "docs/architecture.md"} {
		for _, m := range pattern.FindAllStringSubmatch(readDoc(t, doc), -1) {
			if n, _ := strconv.Atoi(m[1]); n != want {
				t.Errorf("%s says %q but the catalog has %d tools", doc, m[0], want)
			}
		}
	}
}

// The docs site's MCP page keeps its own list of tools; it must name exactly the
// tools the server offers (it was missing get_onboarding_context).
func TestMCPIntegrationPageListsExactlyTheRealTools(t *testing.T) {
	page := readDoc(t, "docs/src/views/McpIntegrationView.vue")
	start := strings.Index(page, "const tools = [")
	end := strings.Index(page, "const examples = [")
	if start < 0 || end < start {
		t.Fatal("could not find the tools array in McpIntegrationView.vue")
	}
	listed := map[string]bool{}
	for _, m := range regexp.MustCompile(`name:\s*'([a-z_]+)'`).FindAllStringSubmatch(page[start:end], -1) {
		listed[m[1]] = true
	}
	real := catalogNames()
	for name := range real {
		if !listed[name] {
			t.Errorf("tool %q exists but is missing from the MCP integration page", name)
		}
	}
	for name := range listed {
		if !real[name] {
			t.Errorf("the MCP integration page lists %q, which is not a tool", name)
		}
	}
}
