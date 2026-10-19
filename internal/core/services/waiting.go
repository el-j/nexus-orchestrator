package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// waitGrace is added to the reset time a provider gave, so the first retry does not land a moment too
// early and run into the same limit.
const waitGrace = time.Second

// chainKeys returns the cooldown keys of every candidate in the chain.
func chainKeys(chain []ports.LLMClient) []string {
	keys := make([]string, len(chain))
	for i, c := range chain {
		keys[i] = cooldownKey(c)
	}
	return keys
}

// holdUntilReset parks a task whose every candidate provider is cooling down: it becomes WAITING until
// the earliest reset and is queued again by ResumeWaitingTasks. It reports whether the task was held;
// false means at least one candidate is available (or the hold could not be saved) and the normal
// retry rules apply.
func (o *OrchestratorService) holdUntilReset(task domain.Task, chain []ports.LLMClient) bool {
	until, ok := o.discovery.cooldowns.earliest(chainKeys(chain), time.Now())
	if !ok {
		return false
	}
	task.Status = domain.StatusWaiting
	task.WaitUntil = until.Add(waitGrace)
	if err := o.repo.Update(o.dbCtx(), task); err != nil {
		log.Printf("orchestrator: hold task %s: %v", task.ID, err)
		return false
	}
	entry := fmt.Sprintf("all providers are cooling down: waiting until %s", task.WaitUntil.Local().Format("Mon 15:04:05"))
	log.Printf("orchestrator: task %s: %s", task.ID, entry)
	o.appendTaskLog(task.ID, entry)
	o.emit(task.ID, domain.StatusWaiting)
	if d := time.Until(task.WaitUntil); d > 0 {
		// One timer per task: a task held again replaces its old timer, and a fired timer removes itself,
		// so the map holds only the timers still pending (Stop cancels those).
		o.waitTimerMu.Lock()
		if old := o.waitTimers[task.ID]; old != nil {
			old.Stop()
		}
		if o.waitTimers == nil {
			o.waitTimers = map[string]*time.Timer{}
		}
		o.waitTimers[task.ID] = time.AfterFunc(d, func() {
			o.waitTimerMu.Lock()
			delete(o.waitTimers, task.ID)
			o.waitTimerMu.Unlock()
			select {
			case <-o.stopCh:
				return
			default:
			}
			if _, err := o.ResumeWaitingTasks(o.dbCtx(), time.Now()); err != nil {
				log.Printf("orchestrator: resume waiting tasks: %v", err)
			}
		})
		o.waitTimerMu.Unlock()
	}
	return true
}

// ResumeWaitingTasks queues the WAITING tasks whose wait time is up at now (or that have none) and
// returns how many it resumed; a task whose providers are no longer cooling down (a budget was raised) is
// resumed before its wait time. The task watchdog calls it on every tick and a timer calls it at the
// reset time, so a restart of the daemon does not strand a waiting task.
func (o *OrchestratorService) ResumeWaitingTasks(ctx context.Context, now time.Time) (int, error) {
	tasks, err := o.repo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("orchestrator: resume waiting tasks: %w", err)
	}
	resumed := 0
	for _, t := range tasks {
		if t.Status != domain.StatusWaiting || (t.WaitUntil.After(now) && !o.holdReleased(ctx, t)) {
			continue
		}
		moved, err := o.repo.UpdateStatusIfCurrent(ctx, t.ID, domain.StatusWaiting, domain.StatusQueued)
		if err != nil || !moved {
			continue // cancelled in the meantime, or another resumer was faster
		}
		t.Status, t.WaitUntil = domain.StatusQueued, time.Time{}
		if err := o.repo.Update(ctx, t); err != nil {
			log.Printf("orchestrator: resume task %s: clear wait time: %v", t.ID, err)
		}
		o.appendTaskLog(t.ID, "wait is over: queued again")
		o.emit(t.ID, domain.StatusQueued)
		resumed++
	}
	if resumed > 0 {
		o.signalWorker()
	}
	return resumed, nil
}
