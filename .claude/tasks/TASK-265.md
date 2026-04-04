# TASK-265 — Call PurgeDisconnected in cleanup goroutine

**Plan**: PLAN-041  
**Status**: done

## What
After marking stale sessions as disconnected in `runSessionCleanup()`, also call
`aiSessionRepo.PurgeDisconnected(ctx, 2*time.Hour)` to delete sessions that have
been disconnected for more than 2 hours.
