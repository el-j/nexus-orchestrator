package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func newAISession(id string) domain.AISession {
	now := time.Now().UTC().Truncate(time.Second)
	return domain.AISession{
		ID:            id,
		Source:        domain.SessionSourceMCP,
		ExternalID:    "ext-" + id,
		AgentName:     "GitHub Copilot",
		ProjectPath:   "/projects/foo",
		Status:        domain.SessionStatusActive,
		LastActivity:  now,
		RoutedTaskIDs: []string{"task-1", "task-2"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestAISessionRepo_SaveAndGetByID(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	sess := newAISession("ai-sess-1")

	if err := ar.SaveAISession(ctx, sess); err != nil {
		t.Fatalf("SaveAISession: %v", err)
	}

	got, err := ar.GetAISessionByID(ctx, "ai-sess-1")
	if err != nil {
		t.Fatalf("GetAISessionByID: %v", err)
	}

	if got.ID != sess.ID {
		t.Errorf("ID: got %q, want %q", got.ID, sess.ID)
	}
	if got.Source != sess.Source {
		t.Errorf("Source: got %q, want %q", got.Source, sess.Source)
	}
	if got.ExternalID != sess.ExternalID {
		t.Errorf("ExternalID: got %q, want %q", got.ExternalID, sess.ExternalID)
	}
	if got.AgentName != sess.AgentName {
		t.Errorf("AgentName: got %q, want %q", got.AgentName, sess.AgentName)
	}
	if got.ProjectPath != sess.ProjectPath {
		t.Errorf("ProjectPath: got %q, want %q", got.ProjectPath, sess.ProjectPath)
	}
	if got.Status != sess.Status {
		t.Errorf("Status: got %q, want %q", got.Status, sess.Status)
	}
	if !got.LastActivity.Equal(sess.LastActivity) {
		t.Errorf("LastActivity: got %v, want %v", got.LastActivity, sess.LastActivity)
	}
	if !got.CreatedAt.Equal(sess.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", got.CreatedAt, sess.CreatedAt)
	}
	if len(got.RoutedTaskIDs) != len(sess.RoutedTaskIDs) {
		t.Fatalf("RoutedTaskIDs length: got %d, want %d", len(got.RoutedTaskIDs), len(sess.RoutedTaskIDs))
	}
	for i, id := range sess.RoutedTaskIDs {
		if got.RoutedTaskIDs[i] != id {
			t.Errorf("RoutedTaskIDs[%d]: got %q, want %q", i, got.RoutedTaskIDs[i], id)
		}
	}
}

func TestAISessionRepo_SaveAndGetByID_EmptyRoutedTaskIDs(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	sess := domain.AISession{
		ID:            "ai-sess-empty",
		Source:        domain.SessionSourceHTTP,
		AgentName:     "Claude Desktop",
		Status:        domain.SessionStatusIdle,
		LastActivity:  now,
		RoutedTaskIDs: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := ar.SaveAISession(ctx, sess); err != nil {
		t.Fatalf("SaveAISession: %v", err)
	}

	got, err := ar.GetAISessionByID(ctx, "ai-sess-empty")
	if err != nil {
		t.Fatalf("GetAISessionByID: %v", err)
	}
	// nil marshals to "null" by default — but we default to '[]' in the schema.
	// SaveAISession marshals nil as "null", so we expect the slice to be nil or empty after unmarshal.
	if len(got.RoutedTaskIDs) != 0 {
		t.Errorf("expected empty RoutedTaskIDs, got %v", got.RoutedTaskIDs)
	}
}

func TestAISessionRepo_GetByID_NotFound(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	_, err := ar.GetAISessionByID(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestAISessionRepo_ListAISessions(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	// Empty list initially
	list, err := ar.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions (empty): %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(list))
	}

	now := time.Now().UTC().Truncate(time.Second)
	s1 := domain.AISession{
		ID: "list-1", Source: domain.SessionSourceMCP, AgentName: "Agent A",
		Status: domain.SessionStatusActive, LastActivity: now,
		CreatedAt: now, UpdatedAt: now, RoutedTaskIDs: []string{},
	}
	s2 := domain.AISession{
		ID: "list-2", Source: domain.SessionSourceVSCode, AgentName: "Agent B",
		Status: domain.SessionStatusIdle, LastActivity: now.Add(-time.Minute),
		CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute), RoutedTaskIDs: []string{},
	}

	if err := ar.SaveAISession(ctx, s1); err != nil {
		t.Fatalf("SaveAISession s1: %v", err)
	}
	if err := ar.SaveAISession(ctx, s2); err != nil {
		t.Fatalf("SaveAISession s2: %v", err)
	}

	list, err = ar.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
	// Ordered by last_activity DESC — s1 is more recent
	if list[0].ID != "list-1" {
		t.Errorf("first session: got %q, want %q", list[0].ID, "list-1")
	}
	if list[1].ID != "list-2" {
		t.Errorf("second session: got %q, want %q", list[1].ID, "list-2")
	}
}

func TestAISessionRepo_UpdateAISessionStatus(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	sess := newAISession("update-sess-1")
	if err := ar.SaveAISession(ctx, sess); err != nil {
		t.Fatalf("SaveAISession: %v", err)
	}

	newActivity := sess.LastActivity.Add(5 * time.Minute)
	if err := ar.UpdateAISessionStatus(ctx, "update-sess-1", domain.SessionStatusDisconnected, newActivity); err != nil {
		t.Fatalf("UpdateAISessionStatus: %v", err)
	}

	got, err := ar.GetAISessionByID(ctx, "update-sess-1")
	if err != nil {
		t.Fatalf("GetAISessionByID after update: %v", err)
	}
	if got.Status != domain.SessionStatusDisconnected {
		t.Errorf("Status: got %q, want %q", got.Status, domain.SessionStatusDisconnected)
	}
	if !got.LastActivity.Equal(newActivity) {
		t.Errorf("LastActivity: got %v, want %v", got.LastActivity, newActivity)
	}
}

func TestAISessionRepo_DeleteAISession(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	sess := newAISession("delete-sess-1")
	if err := ar.SaveAISession(ctx, sess); err != nil {
		t.Fatalf("SaveAISession: %v", err)
	}

	if err := ar.DeleteAISession(ctx, "delete-sess-1"); err != nil {
		t.Fatalf("DeleteAISession: %v", err)
	}

	_, err := ar.GetAISessionByID(ctx, "delete-sess-1")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound after delete, got %v", err)
	}
}

func TestAISessionRepo_DeleteAISession_Idempotent(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	ar := repo_sqlite.NewAISessionRepo(r)
	ctx := context.Background()

	// Delete a non-existent session — must not error
	if err := ar.DeleteAISession(ctx, "ghost-sess"); err != nil {
		t.Errorf("DeleteAISession on non-existent id: expected nil, got %v", err)
	}
}
