package repo_sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"nexus-ai/internal/adapters/outbound/repo_sqlite"
	"nexus-ai/internal/core/domain"
)

func newTestRepoForSession(t *testing.T) *repo_sqlite.Repository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	r, err := repo_sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestSessionRepo_GetByProjectPath_NotFound(t *testing.T) {
	r := newTestRepoForSession(t)
	sr := repo_sqlite.NewSessionRepo(r)

	_, err := sr.GetByProjectPath("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing session, got nil")
	}
}

func TestSessionRepo_AppendMessage_CreatesSession(t *testing.T) {
	r := newTestRepoForSession(t)
	sr := repo_sqlite.NewSessionRepo(r)

	msg := domain.Message{Role: "user", Content: "hello", CreatedAt: time.Now()}
	if err := sr.AppendMessage("/proj/foo", msg); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	sess, err := sr.GetByProjectPath("/proj/foo")
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if len(sess.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sess.Messages))
	}
	if sess.Messages[0].Content != "hello" {
		t.Errorf("unexpected message content: %q", sess.Messages[0].Content)
	}
}

func TestSessionRepo_AppendMessage_AccumulatesMessages(t *testing.T) {
	r := newTestRepoForSession(t)
	sr := repo_sqlite.NewSessionRepo(r)

	path := "/proj/accumulate"
	messages := []domain.Message{
		{Role: "user", Content: "msg1", CreatedAt: time.Now()},
		{Role: "assistant", Content: "reply1", CreatedAt: time.Now()},
		{Role: "user", Content: "msg2", CreatedAt: time.Now()},
	}
	for _, m := range messages {
		if err := sr.AppendMessage(path, m); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
	}

	sess, err := sr.GetByProjectPath(path)
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if len(sess.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(sess.Messages))
	}
	for i, m := range messages {
		if sess.Messages[i].Content != m.Content {
			t.Errorf("message[%d] content: want %q, got %q", i, m.Content, sess.Messages[i].Content)
		}
	}
}

func TestSessionRepo_IsolatesByProjectPath(t *testing.T) {
	r := newTestRepoForSession(t)
	sr := repo_sqlite.NewSessionRepo(r)

	if err := sr.AppendMessage("/proj/alpha", domain.Message{Role: "user", Content: "alpha-msg", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("AppendMessage alpha: %v", err)
	}
	if err := sr.AppendMessage("/proj/beta", domain.Message{Role: "user", Content: "beta-msg", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("AppendMessage beta: %v", err)
	}

	alpha, err := sr.GetByProjectPath("/proj/alpha")
	if err != nil {
		t.Fatalf("GetByProjectPath alpha: %v", err)
	}
	beta, err := sr.GetByProjectPath("/proj/beta")
	if err != nil {
		t.Fatalf("GetByProjectPath beta: %v", err)
	}

	if len(alpha.Messages) != 1 || alpha.Messages[0].Content != "alpha-msg" {
		t.Errorf("alpha session contaminated: %+v", alpha.Messages)
	}
	if len(beta.Messages) != 1 || beta.Messages[0].Content != "beta-msg" {
		t.Errorf("beta session contaminated: %+v", beta.Messages)
	}
}

func TestSessionRepo_Save_RoundTrip(t *testing.T) {
	r := newTestRepoForSession(t)
	sr := repo_sqlite.NewSessionRepo(r)

	now := time.Now().Truncate(time.Millisecond)
	sess := domain.Session{
		ID:          "session-001",
		ProjectPath: "/proj/roundtrip",
		Messages: []domain.Message{
			{Role: "user", Content: "ping", CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := sr.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := sr.GetByProjectPath("/proj/roundtrip")
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("ID: want %q, got %q", sess.ID, got.ID)
	}
	if len(got.Messages) != 1 || got.Messages[0].Role != "user" {
		t.Errorf("unexpected messages: %+v", got.Messages)
	}
}
