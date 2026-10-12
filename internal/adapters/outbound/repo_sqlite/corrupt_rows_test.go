package repo_sqlite_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
)

// insertAISession writes a row straight into the table so a column can hold a
// value the repository itself would never write.
func insertAISession(t *testing.T, r *repo_sqlite.Repository, id string, overrides map[string]string) {
	t.Helper()
	cols := map[string]string{
		"id": id, "source": "mcp", "agent_name": "a", "last_activity": "2026-01-01T00:00:00Z",
		"routed_task_ids": "[]", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	}
	for k, v := range overrides {
		cols[k] = v
	}
	var names, marks []string
	var args []any
	for k, v := range cols {
		names = append(names, k)
		marks = append(marks, "?")
		args = append(args, v)
	}
	if _, err := r.DB().Exec(fmt.Sprintf("INSERT INTO ai_sessions (%s) VALUES (%s)", strings.Join(names, ","), strings.Join(marks, ",")), args...); err != nil {
		t.Fatal(err)
	}
}

func TestAISessionRepo_ReportsRowsItCannotDecode(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		col, val, want string
	}{
		"routed_task_ids": {"routed_task_ids", "{not json", "unmarshal routed_task_ids"},
		"message_count":   {"message_count", "many", "message_count"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestRepo(t)
			defer r.Close()
			insertAISession(t, r, "bad", map[string]string{tc.col: tc.val})
			ar := repo_sqlite.NewAISessionRepo(r)
			if _, err := ar.GetAISessionByID(ctx, "bad"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("get: %v", err)
			}
			if _, err := ar.ListAISessions(ctx); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("list: %v", err)
			}
		})
	}
}

func TestAISessionRepo_ToleratesDamagedOptionalColumns(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	defer r.Close()
	insertAISession(t, r, "odd", map[string]string{
		"delegation_timestamp": "sometime", "agent_capabilities": "{broken", "last_activity": "yesterday",
	})
	insertAISession(t, r, "good", map[string]string{"delegation_timestamp": "2026-02-03T04:05:06Z"})
	ar := repo_sqlite.NewAISessionRepo(r)

	odd, err := ar.GetAISessionByID(ctx, "odd")
	// The driver reads a timestamp it cannot parse as the zero time.
	if err != nil || odd.DelegationTimestamp != nil || odd.AgentCapabilities != nil || !odd.LastActivity.IsZero() {
		t.Errorf("odd = %+v, err = %v", odd, err)
	}
	good, err := ar.GetAISessionByID(ctx, "good")
	if err != nil || good.DelegationTimestamp == nil || good.DelegationTimestamp.Year() != 2026 {
		t.Errorf("good = %+v, err = %v", good, err)
	}
}

func TestAISessionRepo_AppendRoutedTaskIDRepairsDamagedLists(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	defer r.Close()
	insertAISession(t, r, "s", map[string]string{"routed_task_ids": "{not json"})
	ar := repo_sqlite.NewAISessionRepo(r)
	if err := ar.AppendRoutedTaskID(ctx, "s", "t1"); err != nil {
		t.Fatal(err)
	}
	got, err := ar.GetAISessionByID(ctx, "s")
	if err != nil || len(got.RoutedTaskIDs) != 1 || got.RoutedTaskIDs[0] != "t1" {
		t.Errorf("got %+v, err = %v", got.RoutedTaskIDs, err)
	}
}

func TestAISessionRepo_RoundTripsADelegationTimestamp(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	defer r.Close()
	ar := repo_sqlite.NewAISessionRepo(r)
	s := newAISession("deleg")
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	s.DelegationTimestamp = &at
	if err := ar.SaveAISession(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, err := ar.GetAISessionByID(ctx, "deleg")
	if err != nil || got.DelegationTimestamp == nil || !got.DelegationTimestamp.Equal(at) {
		t.Errorf("got %v, err = %v", got.DelegationTimestamp, err)
	}
}
