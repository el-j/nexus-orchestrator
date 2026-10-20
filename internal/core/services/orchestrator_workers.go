package services

import (
	"context"
	"fmt"
	"log"
	"nexus-orchestrator/internal/core/domain"
	"time"
)

func (o *OrchestratorService) signalWorker() {
	select {
	case o.workCh <- struct{}{}:
	default:
	}
}

func (o *OrchestratorService) validateQueueAdmission(task domain.Task) error {
	o.mu.Lock()
	stopped := o.stopped
	queueCap := o.queueCap
	o.mu.Unlock()
	if stopped {
		return fmt.Errorf("orchestrator: queue task: service is stopped")
	}
	if queueCap <= 0 {
		queueCap = 50
	}

	pending, err := o.repo.GetPending(o.dbCtx())
	if err != nil {
		return fmt.Errorf("orchestrator: queue task: check queue cap: %w", err)
	}
	if len(pending) >= queueCap {
		return fmt.Errorf("orchestrator: queue task: %w", ErrQueueFull)
	}

	if task.Command == domain.CommandExecute {
		existing, err := o.repo.GetByProjectPath(o.dbCtx(), task.ProjectPath)
		if err != nil {
			return fmt.Errorf("orchestrator: queue task: %w", err)
		}
		hasPlan := false
		for _, existingTask := range existing {
			if existingTask.Command == domain.CommandPlan && existingTask.Status == domain.StatusCompleted {
				hasPlan = true
				break
			}
		}
		if !hasPlan {
			return fmt.Errorf("orchestrator: queue task: %w", domain.ErrNoPlan)
		}
	}

	return nil
}

// recoverStuckTasks re-queues any tasks that were in PROCESSING state when the
// previous service instance crashed. Called from NewOrchestrator before the
// worker goroutine starts, so no locking is needed on o.queue.
func (o *OrchestratorService) recoverStuckTasks() {
	pending, err := o.repo.GetPending(o.dbCtx())
	if err != nil {
		log.Printf("orchestrator: startup recovery: get pending: %v", err)
		return
	}
	requeued := 0
	hasQueued := false
	for _, t := range pending {
		if t.Status == domain.StatusQueued {
			hasQueued = true
		}
		if t.Status == domain.StatusProcessing {
			ok, err := o.repo.UpdateStatusIfCurrent(o.dbCtx(), t.ID, domain.StatusProcessing, domain.StatusQueued)
			if err != nil {
				log.Printf("orchestrator: startup recovery: re-queue task %s: %v", t.ID, err)
				continue
			}
			if ok {
				requeued++
				hasQueued = true
			}
		}
	}
	if requeued > 0 {
		log.Printf("orchestrator: startup recovery: re-queued %d stuck tasks", requeued)
	}
	if hasQueued {
		o.signalWorker()
	}
}

// requeueForRetry increments the task's RetryCount, persists it with StatusQueued,
// re-adds it to the in-memory queue, and signals the worker.
// Returns true when the task was successfully re-queued; false when maxRetries is
// exhausted or the repo update fails (caller should then mark the task FAILED).
func (o *OrchestratorService) requeueForRetry(task domain.Task) bool {
	if task.RetryCount >= o.maxRetries {
		return false
	}
	task.RetryCount++
	task.Status = domain.StatusQueued
	task.UpdatedAt = time.Now()
	if err := o.repo.Update(o.dbCtx(), task); err != nil {
		log.Printf("orchestrator: requeue task %s: update: %v", task.ID, err)
		return false
	}
	log.Printf("orchestrator: task %s: retry %d/%d", task.ID, task.RetryCount, o.maxRetries)
	o.signalWorker()
	o.emit(task.ID, domain.StatusQueued)
	return true
}

// runWorker is the background loop that processes QUEUED tasks sequentially.
// It blocks on workCh until a task is submitted, then drains the entire queue
// before waiting again — guaranteeing only one LLM call is ever in flight.
func (o *OrchestratorService) runWorker() {
	defer o.workerWg.Done()
	for {
		select {
		case <-o.stopCh:
			return
		case <-o.workCh:
			for {
				select {
				case <-o.stopCh:
					return
				default:
				}
				if !o.processNext() {
					break
				}
			}
		}
	}
}

// runSessionCleanup periodically marks AI sessions as disconnected when they
// have not sent a heartbeat within the stale threshold (5 × heartbeat interval = 5 min).
// It runs until stopCh is closed and shares the workerWg lifecycle.
func (o *OrchestratorService) runSessionCleanup() {
	defer o.workerWg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-o.stopCh
		cancel()
	}()
	defer cancel()
	ticker := time.NewTicker(o.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.mu.Lock()
			repo := o.aiSessionRepo
			b := o.broadcaster
			o.mu.Unlock()
			if repo == nil {
				continue
			}
			sessions, err := repo.ListAISessions(ctx)
			if err != nil {
				log.Printf("orchestrator: session cleanup: list: %v", err)
				continue
			}
			cutoff := time.Now().Add(-o.staleThreshold)
			for _, s := range sessions {
				if s.Status != domain.SessionStatusDisconnected && s.LastActivity.Before(cutoff) {
					if markErr := repo.UpdateAISessionStatus(ctx, s.ID, domain.SessionStatusDisconnected, s.LastActivity); markErr != nil {
						log.Printf("orchestrator: session cleanup: mark disconnected %s: %v", s.ID, markErr)
						continue
					}
					if b != nil {
						b.BroadcastAISessionEvent(domain.AISessionEvent{
							Type:        "ai_session_changed",
							AISessionID: s.ID,
							Status:      domain.SessionStatusDisconnected,
							Timestamp:   time.Now(),
						})
					}
				}
			}
			// Purge disconnected sessions older than 2 hours to prevent unbounded growth.
			if n, purgeErr := repo.PurgeDisconnected(ctx, 2*time.Hour); purgeErr != nil {
				log.Printf("orchestrator: session cleanup: purge: %v", purgeErr)
			} else if n > 0 {
				log.Printf("orchestrator: session cleanup: purged %d stale disconnected sessions", n)
			}
		}
	}
}

func (o *OrchestratorService) runTaskWatchdog() {
	defer o.workerWg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-o.stopCh
		cancel()
	}()
	defer cancel()
	watchdogInterval := o.watchdogInterval
	if watchdogInterval <= 0 {
		watchdogInterval = time.Minute
	}
	ticker := time.NewTicker(watchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			if _, err := o.ResumeWaitingTasks(ctx, time.Now()); err != nil {
				log.Printf("orchestrator: task watchdog: %v", err)
			}
			// 5 minutes timeout for processing tasks
			tasks, err := o.repo.GetStaleProcessing(ctx, 5*time.Minute)
			if err != nil {
				log.Printf("orchestrator: task watchdog: get stale tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				o.runMu.Lock()
				_, inProcess := o.running[t.ID]
				o.runMu.Unlock()
				if inProcess {
					continue
				}
				log.Printf("orchestrator: task watchdog: task %s stuck in PROCESSING, failing", t.ID)
				t.Status = domain.StatusFailed
				t.Logs += "\n\n[System] Task timed out while PROCESSING. Marked as FAILED by watchdog."
				if err := o.repo.Update(o.dbCtx(), t); err != nil {
					log.Printf("orchestrator: task watchdog: update task %s fail: %v", t.ID, err)
					continue
				}
				_ = o.repo.UpdateLogs(o.dbCtx(), t.ID, t.Logs)
				// emit already publishes the task.failed event (under the lock that
				// guards o.broadcaster); broadcasting again would duplicate it.
				o.emit(t.ID, domain.StatusFailed)
			}
		}
	}
}
