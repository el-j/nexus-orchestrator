package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAISessionStatus_IsTerminal(t *testing.T) {
	cases := []struct {
		status AISessionStatus
		want   bool
	}{
		{SessionStatusActive, false},
		{SessionStatusIdle, false},
		{SessionStatusDisconnected, true},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			if got := tc.status.IsTerminal(); got != tc.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAISession_JSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := AISession{
		ID:            "sess-001",
		Source:        SessionSourceMCP,
		ExternalID:    "ext-abc",
		AgentName:     "CopilotAgent",
		ProjectPath:   "/projects/foo",
		Status:        SessionStatusActive,
		LastActivity:  now,
		RoutedTaskIDs: []string{"task-1", "task-2"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got AISession
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID: got %q, want %q", got.ID, original.ID)
	}
	if got.Source != original.Source {
		t.Errorf("Source: got %q, want %q", got.Source, original.Source)
	}
	if got.ExternalID != original.ExternalID {
		t.Errorf("ExternalID: got %q, want %q", got.ExternalID, original.ExternalID)
	}
	if got.AgentName != original.AgentName {
		t.Errorf("AgentName: got %q, want %q", got.AgentName, original.AgentName)
	}
	if got.ProjectPath != original.ProjectPath {
		t.Errorf("ProjectPath: got %q, want %q", got.ProjectPath, original.ProjectPath)
	}
	if got.Status != original.Status {
		t.Errorf("Status: got %q, want %q", got.Status, original.Status)
	}
	if !got.LastActivity.Equal(original.LastActivity) {
		t.Errorf("LastActivity: got %v, want %v", got.LastActivity, original.LastActivity)
	}
	if len(got.RoutedTaskIDs) != len(original.RoutedTaskIDs) {
		t.Errorf("RoutedTaskIDs len: got %d, want %d", len(got.RoutedTaskIDs), len(original.RoutedTaskIDs))
	}
	if !got.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", got.CreatedAt, original.CreatedAt)
	}
	if !got.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", got.UpdatedAt, original.UpdatedAt)
	}
}
