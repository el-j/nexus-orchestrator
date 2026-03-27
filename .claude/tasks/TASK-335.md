# TASK-335: ActivityService — periodic polling + session bridge

**Plan:** PLAN-050 · Wave 3
**Status:** DONE
**Agent:** Senior Developer

## Description

Create `internal/core/services/activity_service.go` with `ActivityService` struct.

Responsibilities:

1. Hold a slice of `ports.ActivityReader` implementations
2. Run a background goroutine polling all readers every 5 seconds
3. Save new activities to `AIActivityRepository`
4. Auto-create/update `AISession` records when activity is detected from a source without a registered session
5. Update AISession.CurrentActivity, MessageCount, TokensUsed from aggregated activities
6. Broadcast `ai_activity_new` SSE events for each new activity
7. Run hourly retention purge via AIActivityRepository.PurgeOlderThan

Constructor: `NewActivityService(repo AIActivityRepository, sessionRepo AISessionRepository, readers ...ActivityReader)`
Methods: `Start()`, `Stop()`, `GetRecentActivities(since time.Time, filters) []AIActivity`, `GetTimeline(since, limit) []AIActivity`

## Acceptance

- Background polling works with graceful shutdown (stopCh)
- Auto-creates sessions from discovered activity
- SSE events broadcast
- Retention purge runs
- Tests with mock readers

## Completed

Implemented `ActivityService` with 5s background poll of all readers, session auto-create/update bridge, `ai_activity_new` SSE broadcast, and hourly retention purge.
