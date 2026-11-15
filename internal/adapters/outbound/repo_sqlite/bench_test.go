package repo_sqlite_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// BenchmarkGetAll lists a history of two thousand tasks with a log each, as the History view does.
func BenchmarkGetAll(b *testing.B) {
	r := newBenchRepo(b)
	ctx := context.Background()
	for i := 0; i < 2000; i++ {
		t := newTask(fmt.Sprintf("t%04d", i), domain.StatusCompleted, time.Unix(int64(1_700_000_000+i), 0))
		t.Logs = "completed via a provider\n"
		if err := r.Save(ctx, t); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.GetAll(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearchFTS searches the knowledge of a project that holds a thousand entries.
func BenchmarkSearchFTS(b *testing.B) {
	k := repo_sqlite.NewKnowledgeRepo(newBenchRepo(b))
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		entry := domain.ProjectKnowledge{
			ID: fmt.Sprintf("k%04d", i), ProjectPath: "/p", Kind: domain.KnowledgeLearning,
			Topic: fmt.Sprintf("topic %d", i), Content: fmt.Sprintf("entry %d about the scheduler and zebra retries", i),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := k.SaveKnowledge(ctx, entry); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := k.SearchFTS(ctx, "/p", "zebra", 10); err != nil {
			b.Fatal(err)
		}
	}
}
