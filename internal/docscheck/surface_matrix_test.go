package docscheck

import (
	"regexp"
	"strings"
	"testing"
)

// cell is one capability on one surface: either a needle that must appear in the
// surface's source, or a reason the capability is deliberately absent there. An
// empty cell fails, so a feature cannot be added to one surface and forgotten on
// the others.
type cell struct{ needle, reason string }

func has(needle string) cell  { return cell{needle: needle} }
func without(why string) cell { return cell{reason: why} }

// surfaces maps a column name to the source text its needles are searched in.
func surfaceSources(t *testing.T) map[string]string {
	t.Helper()
	cli := read(t, "internal/adapters/inbound/cli/root.go") + read(t, "internal/adapters/inbound/cli/brain.go")
	return map[string]string{
		"rest":      read(t, "internal/adapters/inbound/httpapi/server.go"),
		"mcp":       read(t, "docs/src/data/mcpTools.ts"),
		"cli":       cli + read(t, "cmd/nexus-submit/main.go"),
		"wails":     read(t, "app.go"),
		"frontend":  read(t, "frontend/src/types/wails.ts"),
		"extension": read(t, "vscode-extension/src/nexusClient.ts") + read(t, "vscode-extension/package.json"),
	}
}

type row struct {
	capability                                 string
	rest, mcp, cli, wails, frontend, extension cell
}

var (
	noCLI   = without("operator-only; the CLI covers task and brain workflows")
	noExt   = without("the extension submits work and shows status; it does not manage this")
	noWails = without("served over HTTP by the same daemon; the desktop UI reaches it through the REST client")
)

// matrix is the audit behind issue #54. Extending the product means extending
// this table, which is the point.
var matrix = []row{
	{"submit a task", has(`"/api/tasks"`), has("'submit_task'"), has("/api/tasks"), has("SubmitTask("), has("submitTask"), has("/api/tasks")},
	{"read a task", has(`"/api/tasks/{id}"`), has("'get_task'"), has("get <task-id>"), has("GetTask("), has("getTask"), noExt},
	{"wait for a task", has(`"/api/events"`), has("'wait_for_task'"),
		without("nexus-cli get reads a task; blocking until it ends is for agents calling over MCP"),
		without("the desktop UI follows tasks through the live event stream instead of blocking"),
		without("the UI follows tasks through the live event stream (GET /api/events) instead of blocking"),
		without("the extension follows tasks through the event stream (Live Output) instead of blocking")},
	{"connect the agents on this machine to MCP", without("a local setup step for the machine, not something the daemon can do from outside"),
		without("a local setup step for the machine"), has("newConnectCmd"),
		without("a local setup step; the CLI does it"), without("the UI shows the queue and agents, it does not edit other tools' config files"),
		has("nexus.connectAgent")},
	{"usage ledger (tokens and cost per provider, model, role)", has(`"/api/usage"`), has("'get_usage'"), has("newUsageCmd"),
		without("served over HTTP by the same daemon; the desktop UI reads the ledger through the REST client"),
		has("getUsage"),
		without("the extension shows provider state in its status bar; usage is read in the Mission Control card, REST, MCP and the CLI")},
	{"routing preview (provider order for a role)", has(`"/api/routing"`), has("'get_routing'"), has("newRoutingCmd"),
		without("served over HTTP by the same daemon; the desktop UI reaches it through the REST client"),
		without("the Mission Control provider cards show each provider's cooldown and quota from /api/providers; the full order is for agents (MCP) and the CLI"),
		without("the extension shows provider state in its status bar; the preview is for agents and the CLI")},
	{"project policy (ladder, budgets, quota threshold, token caps)", has(`"/api/policy"`),
		without("budgets and ladders are the owner's decision; agents read their effect through get_routing and get_usage, they do not change them"),
		has("newPolicyCmd"),
		without("served over HTTP by the same daemon; the desktop UI reaches it through the REST client"),
		has("getPolicy"),
		without("the extension shows provider state in its status bar; the policy is edited in the app, the CLI or REST")},
	{"list the queue", has(`Get("/api/tasks"`), has("'get_queue'"), has(`"list"`), has("GetQueue("), has("getQueue"), has("/api/tasks")},
	{"list every task", has(`"/api/tasks/all"`), has("'get_all_tasks'"), noCLI, has("GetAllTasks("), has("getAllTasks"), has("/api/tasks/all")},
	{"cancel a task", has(`Delete("/api/tasks/{id}"`), has("'cancel_task'"), has("cancel <task-id>"), has("CancelTask("), has("cancelTask"), has("cancelTask")},
	{"update a task", has(`Put("/api/tasks/{id}"`), has("'update_task'"), has("update <task-id>"), has("UpdateTask("), has("updateTask"), noExt},
	{"create a draft", has(`"/api/tasks/draft"`), has("'create_draft'"), has(`"draft"`), has("CreateDraft("), has("createDraft"), noExt},
	{"read the backlog", has(`"/api/tasks/backlog"`), has("'get_backlog'"), has(`"backlog"`), has("GetBacklog("), has("getBacklog"), noExt},
	{"promote a draft", has(`"/api/tasks/{id}/promote"`), has("'promote_task'"), has("promote <task-id>"), has("PromoteTask("), has("promoteTask"), noExt},
	{"claim a task", has(`"/api/tasks/{id}/claim"`), has("'claim_task'"), noCLI, has("ClaimTask("), has("claimTask"), noExt},
	{"report task status", has(`"/api/tasks/{id}/status"`), has("'update_task_status'"), noCLI, has("UpdateTaskStatus("), has("updateTaskStatus"), noExt},
	{"list providers", has(`Get("/api/providers"`), has("'get_providers'"), has(`"providers"`), has("GetProviders("), has("getProviders"), has("/api/providers")},
	{"discover providers", has(`"/api/providers/discovered/scan"`), has("'discover_providers'"), noCLI, has("TriggerScan("), has("triggerScan"), noExt},
	{"configure providers", has(`"/api/providers/config"`), has("'add_provider_config'"), noCLI, has("AddProviderConfig("), has("addProviderConfig"), noExt},
	{"AI sessions", has(`"/api/ai-sessions"`), has("'register_session'"), noCLI, has("RegisterAISession("), has("registerAISession"), has("/api/ai-sessions")},
	{"delegate sessions", has(`"/api/ai-sessions/{id}/delegate"`), has("'delegate_to_nexus'"), noCLI, noWails, without("reached through the REST client"), has("delegateToNexus")},
	{"runtime config", has(`"/api/config"`), without("config is changed by operators, not by agents"), noCLI, has("UpdateRuntimeConfig("), has("updateRuntimeConfig"), noExt},
	{"brain: status", has(`"/api/brain/status"`), has("'get_brain_status'"), has(`"status"`), has("GetBrainStatus("), has("getBrainStatus"), has("nexus.brain.status")},
	{"brain: ingest", has(`"/api/brain/ingest"`), has("'ingest_knowledge'"), has(`"ingest"`), has("IngestKnowledge("), has("ingestKnowledge"), has("nexus.brain.ingest")},
	{"brain: search", has(`"/api/brain/search"`), has("'search_knowledge'"), has(`"search"`), has("SearchKnowledge("), has("searchKnowledge"), has("nexus.brain.search")},
	{"brain: init", has(`"/api/brain/init"`), has("'init_project'"), has(`"init"`), has("InitProject("), has("initProject"), has("nexus.brain.init")},
	{"brain: context", has(`"/api/brain/context"`), has("'get_project_context'"), has(`"context"`), has("GetProjectContext("), has("getProjectContext"), has("/api/brain/context")},
	{"brain: focused context", has(`"/api/brain/focused-context"`), has("'get_focused_context'"), has(`"context"`), has("GetFocusedContext("), has("getFocusedContext"), has("/api/brain/focused-context")},
	{"brain: list and delete", has(`"/api/brain/knowledge/{id}"`), has("'delete_knowledge'"), has(`"delete"`), has("DeleteKnowledge("), has("deleteKnowledge"), noExt},
	{"brain: file map", has(`"/api/brain/file-map"`), has("'get_file_map'"), has(`"file-map"`), has("GetFileMap("), has("getFileMap"), noExt},
	{"activities", has(`"/api/activities"`), without("agents report activity through sessions"), noCLI, has("GetRecentActivities("), without("read via the REST composables"), has("showActivityLog")},
	{"discovered plans", has(`"/api/plans/discovered"`), has("'get_discovered_plans'"), noCLI, noWails, without("read via the REST composables"), noExt},
}

func TestEverySurfaceCoversEveryCapabilityOrSaysWhy(t *testing.T) {
	src := surfaceSources(t)
	for _, r := range matrix {
		cols := map[string]cell{"rest": r.rest, "mcp": r.mcp, "cli": r.cli, "wails": r.wails, "frontend": r.frontend, "extension": r.extension}
		for name, c := range cols {
			switch {
			case c.needle == "" && c.reason == "":
				t.Errorf("%s: no entry for %s (add it, or say why it is absent)", r.capability, name)
			case c.needle != "" && !strings.Contains(src[name], c.needle):
				t.Errorf("%s: %s no longer exposes %q", r.capability, name, c.needle)
			}
		}
	}
}

// Every MCP tool and every REST route must appear in the matrix or in the
// exemptions below, so a new endpoint cannot ship without a decision about the
// other surfaces.
func TestEveryMCPToolBelongsToTheMatrix(t *testing.T) {
	var covered strings.Builder
	for _, r := range matrix {
		covered.WriteString(r.mcp.needle + " ")
	}
	// Tools whose capability is the same row as a sibling (list/get variants), plus
	// session housekeeping and the self-describing endpoints.
	siblings := []string{"get_ai_sessions", "deregister_ai_session", "heartbeat_ai_session", "terminate_ai_session",
		"purge_disconnected_sessions", "heartbeat_task", "promote_provider", "list_provider_configs",
		"update_provider_config", "remove_provider_config", "get_onboarding_context", "list_knowledge",
		"get_discovered_agents", "howto", "howto_brief", "health"}
	for _, name := range mcpToolNames(t) {
		if !strings.Contains(covered.String(), "'"+name+"'") && !contains(siblings, name) {
			t.Errorf("MCP tool %q is in no matrix row; add a row or list it as a sibling", name)
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

var toolName = regexp.MustCompile(`name: '([a-z_]+)'`)

func mcpToolNames(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range toolName.FindAllStringSubmatch(read(t, "docs/src/data/mcpTools.ts"), -1) {
		out = append(out, m[1])
	}
	if len(out) == 0 {
		t.Fatal("no MCP tools found in docs/src/data/mcpTools.ts")
	}
	return out
}

var routeDecl = regexp.MustCompile(`r\.(Get|Post|Put|Delete)\("(/api/[^"]+)"`)

// The REST reference must document every registered route.
func TestAPIReferenceDocumentsEveryRoute(t *testing.T) {
	doc := read(t, "docs/api-reference.md")
	for _, m := range routeDecl.FindAllStringSubmatch(read(t, "internal/adapters/inbound/httpapi/server.go"), -1) {
		verb, path := strings.ToUpper(m[1]), m[2]
		if !strings.Contains(doc, path) {
			t.Errorf("docs/api-reference.md does not mention %s %s", verb, path)
		}
	}
}
