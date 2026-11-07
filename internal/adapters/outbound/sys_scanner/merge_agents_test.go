package sys_scanner

import (
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestMergeAgents_KeepsWhatEveryProbeLearnedAndLinksSubAgentsToTheirParent(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	found := [][]domain.DiscoveredAgent{
		{{ID: "claude", Name: "Claude", ConfigPath: "/c/settings.json"}},
		{{ID: "claude", IsRunning: true, MCPEndpoint: "http://x/mcp", CLIPath: "/bin/claude", PID: 42, ProcessName: "claude"}},
		{{ID: "sub-1", ParentAgentID: "claude"}, {ID: "orphan", ParentAgentID: "missing"}},
	}
	byID := map[string]domain.DiscoveredAgent{}
	for _, a := range mergeAgents(found, now) {
		byID[a.ID] = a
	}
	c := byID["claude"]
	if !c.IsRunning || c.MCPEndpoint != "http://x/mcp" || c.ConfigPath != "/c/settings.json" || c.CLIPath != "/bin/claude" || c.PID != 42 || c.ProcessName != "claude" {
		t.Errorf("merged agent lost something: %+v", c)
	}
	if !c.LastSeen.Equal(now) {
		t.Errorf("LastSeen = %v, want the scan time", c.LastSeen)
	}
	if len(c.SubAgentIDs) != 1 || c.SubAgentIDs[0] != "sub-1" {
		t.Errorf("SubAgentIDs = %v", c.SubAgentIDs)
	}
	if _, ok := byID["orphan"]; !ok {
		t.Error("an agent whose parent was not found is still listed")
	}
	if len(byID) != 3 {
		t.Errorf("want 3 agents, got %d", len(byID))
	}
}
