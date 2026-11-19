//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// Plans and tasks kept in folders the project names in its policy are what the daemon tells a new session in
// its onboarding context: the active plan, the task counts and the folders they were read from.
func TestDaemonReadsPlanAndTaskFilesFromTheFoldersThePolicyNamesIntoTheOnboarding(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	mustWrite(t, filepath.Join(project, ".nexus", "policy.json"), `{"planFiles":{"plans":["design/specs"],"tasks":["design/work"],"discover":false}}`)
	mustWrite(t, filepath.Join(project, "design", "specs", "launch.md"), "# Launch\nstatus: in-progress\n")
	mustWrite(t, filepath.Join(project, "design", "work", "TASK-1.md"), "---\nid: TASK-1\ntitle: first\nstatus: done\n---\n# Task\n")
	mustWrite(t, filepath.Join(project, "design", "work", "TASK-2.md"), "---\nid: TASK-2\ntitle: second\nstatus: todo\n---\n# Task\n")
	mustWrite(t, filepath.Join(project, "elsewhere", "TASK-9.md"), "---\nid: TASK-9\nstatus: todo\n---\n# not in a named folder\n")

	var onboarding struct {
		Content string `json:"content"`
	}
	if code := d.do(http.MethodGet, "/api/brain/onboarding?projectPath="+url.QueryEscape(project), nil, &onboarding); code != http.StatusOK {
		t.Fatalf("GET /api/brain/onboarding: HTTP %d\n%s", code, d.logs())
	}
	if !contains(onboarding.Content, "launch", "design/specs") {
		t.Errorf("the plan from the named folder is the active plan:\n%s", onboarding.Content)
	}
	if !strings.Contains(onboarding.Content, "2 task files") {
		t.Errorf("only the two task files of the named folder are counted (discover is off):\n%s", onboarding.Content)
	}
}

// An agent connected over the API registers, is delegated to Nexus, claims a task the daemon had no model
// for, and reports it done: the task ends COMPLETED with what the agent reported.
func TestDaemonLetsARegisteredAgentClaimAndFinishAPromotedTask(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()

	var session struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodPost, "/api/ai-sessions", map[string]any{"agentName": "e2e-agent", "source": "http", "projectPath": project}, &session); code != http.StatusCreated || session.ID == "" {
		t.Fatalf("register: HTTP %d %+v\n%s", code, session, d.logs())
	}
	var instruction struct {
		Instruction string `json:"instruction"`
	}
	if code := d.do(http.MethodPost, "/api/ai-sessions/"+session.ID+"/delegate", nil, &instruction); code != http.StatusOK || !strings.Contains(instruction.Instruction, "nexusOrchestrator") {
		t.Errorf("delegate: HTTP %d %q", code, instruction.Instruction)
	}

	var draft map[string]string
	if code := d.do(http.MethodPost, "/api/tasks/draft", map[string]any{"projectPath": project, "instruction": "agent work", "role": "coder"}, &draft); code != http.StatusCreated {
		t.Fatalf("draft: HTTP %d\n%s", code, d.logs())
	}
	id := draft["id"]
	if id == "" {
		id = draft["task_id"]
	}
	if code := d.do(http.MethodPost, "/api/tasks/"+id+"/promote", nil, nil); code != http.StatusOK {
		t.Fatalf("promote: HTTP %d\n%s", code, d.logs())
	}
	// The daemon has no model, so its engine settles the task as NO_PROVIDER; the agent takes it from there.
	d.waitStatus(id, "NO_PROVIDER")
	var claimed apiTask
	if code := d.do(http.MethodPost, "/api/tasks/"+id+"/claim", map[string]string{"sessionId": session.ID}, &claimed); code != http.StatusOK || claimed.Status != "PROCESSING" {
		t.Fatalf("claim: HTTP %d %+v\n%s", code, claimed, d.logs())
	}
	var done apiTask
	if code := d.do(http.MethodPut, "/api/tasks/"+id+"/status", map[string]string{"sessionId": session.ID, "status": "COMPLETED", "logs": "agent says done"}, &done); code != http.StatusOK || done.Status != "COMPLETED" {
		t.Fatalf("report: HTTP %d %+v\n%s", code, done, d.logs())
	}
	var final apiTask
	d.do(http.MethodGet, "/api/tasks/"+id, nil, &final)
	if final.Status != "COMPLETED" || !strings.Contains(final.Logs, "agent says done") {
		t.Errorf("final task: %+v", final)
	}
}
