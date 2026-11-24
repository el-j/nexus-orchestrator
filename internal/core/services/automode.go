package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

const (
	defaultAutoMaxTasks     = 10
	hardAutoMaxTasks        = 25
	defaultAutoMaxTurns     = 8
	defaultAutoRepairLimit  = 2
	defaultAutoPollInterval = 150 * time.Millisecond
)

// ErrAutoModeActive is returned when a project already has an active Auto-Mode run.
var ErrAutoModeActive = errors.New("auto-mode is already active for this project")

// autoPollInterval is a test seam for how often Auto-Mode checks task settlement.
var autoPollInterval = defaultAutoPollInterval

// autoRunState holds the mutable state and cancellation handle of one Auto-Mode run.
type autoRunState struct {
	mu     sync.Mutex
	run    domain.AutoModeRun
	ctx    context.Context
	cancel context.CancelFunc
}

// snapshot returns a deep-enough copy of the run state safe to return to callers.
func (s *autoRunState) snapshot() domain.AutoModeRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := s.run
	cp.TaskIDs = append([]string(nil), s.run.TaskIDs...)
	return cp
}

// update mutates the run under lock and bumps UpdatedAt.
func (s *autoRunState) update(fn func(*domain.AutoModeRun)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.run)
	s.run.UpdatedAt = time.Now()
	if s.run.Phase.IsTerminal() && s.run.FinishedAt.IsZero() {
		s.run.FinishedAt = s.run.UpdatedAt
	}
}

// StartAutoMode launches an autonomous end-to-end loop (Brain sync -> Plan -> Execute -> Verify/Repair -> Learn)
// for req.ProjectPath and returns the initial run snapshot.
func (o *OrchestratorService) StartAutoMode(_ context.Context, req domain.AutoModeRequest) (domain.AutoModeRun, error) {
	proj := strings.TrimSpace(req.ProjectPath)
	if proj == "" {
		return domain.AutoModeRun{}, fmt.Errorf("automode: projectPath is required")
	}
	proj = filepath.Clean(proj)
	req.ProjectPath = proj
	if req.MaxTasks <= 0 {
		req.MaxTasks = defaultAutoMaxTasks
	} else if req.MaxTasks > hardAutoMaxTasks {
		req.MaxTasks = hardAutoMaxTasks
	}
	if req.MaxAgentTurns <= 0 {
		req.MaxAgentTurns = defaultAutoMaxTurns
	}
	if req.MaxRepairAttempts <= 0 {
		req.MaxRepairAttempts = defaultAutoRepairLimit
	}

	o.autoMu.Lock()
	if o.autoRuns == nil {
		o.autoRuns = make(map[string]*autoRunState)
	}
	for _, existing := range o.autoRuns {
		snap := existing.snapshot()
		if snap.ProjectPath == proj && !snap.Phase.IsTerminal() {
			o.autoMu.Unlock()
			return domain.AutoModeRun{}, fmt.Errorf("automode: %w (%s)", ErrAutoModeActive, snap.ID)
		}
	}
	now := time.Now()
	ctx, cancel := context.WithCancel(o.baseCtx)
	state := &autoRunState{
		ctx:    ctx,
		cancel: cancel,
		run: domain.AutoModeRun{
			ID:          uuid.NewString(),
			ProjectPath: proj,
			Goal:        strings.TrimSpace(req.Goal),
			Phase:       domain.AutoPhaseSyncing,
			TaskIDs:     []string{},
			StartedAt:   now,
			UpdatedAt:   now,
		},
	}
	o.autoRuns[state.run.ID] = state
	o.autoMu.Unlock()

	o.workerWg.Add(1)
	go func() {
		defer o.workerWg.Done()
		defer cancel()
		o.runAutoMode(state, req)
	}()

	return state.snapshot(), nil
}

// GetAutoModeRun returns the current snapshot of the auto-mode run with id.
func (o *OrchestratorService) GetAutoModeRun(id string) (domain.AutoModeRun, error) {
	o.autoMu.Lock()
	state, ok := o.autoRuns[id]
	o.autoMu.Unlock()
	if !ok {
		return domain.AutoModeRun{}, fmt.Errorf("automode: %w", domain.ErrNotFound)
	}
	return state.snapshot(), nil
}

// ListAutoModeRuns returns all recorded auto-mode runs (newest first), optionally filtered by projectPath.
func (o *OrchestratorService) ListAutoModeRuns(projectPath string) []domain.AutoModeRun {
	filter := ""
	if strings.TrimSpace(projectPath) != "" {
		filter = filepath.Clean(projectPath)
	}
	o.autoMu.Lock()
	out := make([]domain.AutoModeRun, 0, len(o.autoRuns))
	for _, s := range o.autoRuns {
		snap := s.snapshot()
		if filter != "" && snap.ProjectPath != filter {
			continue
		}
		out = append(out, snap)
	}
	o.autoMu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

// StopAutoMode cancels an active auto-mode run and its non-terminal tasks.
func (o *OrchestratorService) StopAutoMode(id string) error {
	o.autoMu.Lock()
	state, ok := o.autoRuns[id]
	o.autoMu.Unlock()
	if !ok {
		return fmt.Errorf("automode: %w", domain.ErrNotFound)
	}
	state.cancel()
	snap := state.snapshot()
	for _, tid := range snap.TaskIDs {
		_ = o.CancelTask(tid)
	}
	state.update(func(r *domain.AutoModeRun) {
		if !r.Phase.IsTerminal() {
			r.Phase = domain.AutoPhaseStopped
			r.Summary = "Stopped by user"
		}
	})
	return nil
}

// runAutoMode executes the full autonomous loop: Brain sync -> Plan -> Execute -> Verify/Repair -> Learn.
func (o *OrchestratorService) runAutoMode(state *autoRunState, req domain.AutoModeRequest) {
	ctx := state.ctx
	proj := req.ProjectPath

	// Phase 1: Sync project brain and gather RAG context.
	ragContext := o.autoSyncBrain(ctx, proj, req.Goal)
	if ctx.Err() != nil {
		o.markAutoStopped(state)
		return
	}

	// Phase 2: Plan (decompose goal or promote existing backlog).
	state.update(func(r *domain.AutoModeRun) { r.Phase = domain.AutoPhasePlanning })
	policy, _, _ := LoadProjectPolicy(proj)
	verifyCmd := strings.TrimSpace(req.VerifyCommand)
	if verifyCmd == "" {
		verifyCmd = strings.TrimSpace(policy.Verify)
	}

	taskIDs, err := o.autoPlanTasks(ctx, req, ragContext, verifyCmd)
	if ctx.Err() != nil {
		o.markAutoStopped(state)
		return
	}
	if err != nil {
		state.update(func(r *domain.AutoModeRun) {
			r.Phase = domain.AutoPhaseFailed
			r.Error = err.Error()
		})
		return
	}
	if len(taskIDs) == 0 {
		state.update(func(r *domain.AutoModeRun) {
			r.Phase = domain.AutoPhaseCompleted
			r.Summary = "No backlog tasks to execute"
		})
		return
	}
	state.update(func(r *domain.AutoModeRun) {
		r.TaskIDs = taskIDs
		r.Phase = domain.AutoPhaseExecuting
	})

	// Phase 3: Wait for all tasks to settle.
	completed, failed, stopped := o.autoWaitTasks(ctx, state, taskIDs)
	if stopped {
		o.markAutoStopped(state)
		return
	}

	// Phase 4: Final project verification + automatic repair loop.
	verifyErr := ""
	if failed == 0 && verifyCmd != "" && o.commandRunner != nil {
		verifyErr = o.autoVerifyAndRepair(ctx, state, req, verifyCmd)
		if ctx.Err() != nil {
			o.markAutoStopped(state)
			return
		}
	}

	// Phase 5: Continuous learning digestion into project brain.
	state.update(func(r *domain.AutoModeRun) { r.Phase = domain.AutoPhaseLearning })
	learnings := o.autoDigestLearnings(ctx, state.snapshot().TaskIDs)

	// Settle terminal status.
	state.update(func(r *domain.AutoModeRun) {
		r.LearningsSaved = learnings
		if r.FailedTasks > 0 || verifyErr != "" {
			r.Phase = domain.AutoPhaseFailed
			if verifyErr != "" {
				r.Error = verifyErr
			} else {
				r.Error = fmt.Sprintf("%d of %d tasks failed", r.FailedTasks, len(r.TaskIDs))
			}
			r.Summary = fmt.Sprintf("Auto-Mode finished with failures (%d completed, %d failed)", completed, r.FailedTasks)
			return
		}
		r.Phase = domain.AutoPhaseCompleted
		r.Summary = fmt.Sprintf("Auto-Mode completed %d task(s) cleanly (%d learnings saved)", len(r.TaskIDs), learnings)
	})
}

// markAutoStopped sets the run to AutoPhaseStopped unless already terminal.
func (o *OrchestratorService) markAutoStopped(state *autoRunState) {
	state.update(func(r *domain.AutoModeRun) {
		if !r.Phase.IsTerminal() {
			r.Phase = domain.AutoPhaseStopped
			r.Summary = "Stopped"
		}
	})
}

// autoSyncBrain initializes the project brain if needed and returns concise RAG context for planning.
func (o *OrchestratorService) autoSyncBrain(ctx context.Context, projectPath, goal string) string {
	o.mu.Lock()
	brain := o.brain
	o.mu.Unlock()
	if brain == nil {
		return ""
	}
	if st, err := brain.GetStatus(ctx, projectPath); err == nil && !st.Initialized {
		_, _ = brain.InitProject(ctx, projectPath, "")
	}
	q := domain.ContextQuery{ProjectPath: projectPath, Question: goal, MaxTokens: 800}
	resp, err := brain.GetContext(ctx, q)
	if err != nil || len(resp.Sections) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, sec := range resp.Sections {
		fmt.Fprintf(&sb, "## %s (%s)\n%s\n\n", sec.Topic, sec.Kind, sec.Content)
	}
	return sb.String()
}

// autoWaitTasks polls until every task in taskIDs reaches a terminal state or ctx is cancelled.
func (o *OrchestratorService) autoWaitTasks(ctx context.Context, state *autoRunState, taskIDs []string) (completed, failed int, stopped bool) {
	ticker := time.NewTicker(autoPollInterval)
	defer ticker.Stop()
	for {
		c, f, pending := 0, 0, 0
		for _, id := range taskIDs {
			t, err := o.GetTask(id)
			if err != nil {
				f++
				continue
			}
			switch t.Status {
			case domain.StatusCompleted:
				c++
			case domain.StatusFailed, domain.StatusCancelled, domain.StatusTooLarge, domain.StatusNoProvider:
				f++
			default:
				pending++
			}
		}
		state.update(func(r *domain.AutoModeRun) {
			r.CompletedTasks = c
			r.FailedTasks = f
		})
		if pending == 0 {
			return c, f, false
		}
		select {
		case <-ctx.Done():
			return c, f, true
		case <-ticker.C:
		}
	}
}

// autoVerifyAndRepair runs verifyCmd on the project and spawns up to MaxRepairAttempts agentic repair tasks.
func (o *OrchestratorService) autoVerifyAndRepair(ctx context.Context, state *autoRunState, req domain.AutoModeRequest, verifyCmd string) string {
	for attempt := 0; ; attempt++ {
		state.update(func(r *domain.AutoModeRun) { r.Phase = domain.AutoPhaseVerifying })
		out, err := o.commandRunner.Run(ctx, req.ProjectPath, verifyCmd)
		if err == nil {
			return ""
		}
		cleanOut := stripTerminalEscapes(out)
		if attempt >= req.MaxRepairAttempts {
			return fmt.Sprintf("final verification failed after %d repair attempt(s): %v", attempt, err)
		}
		state.update(func(r *domain.AutoModeRun) {
			r.Phase = domain.AutoPhaseRepairing
			r.RepairAttempts = attempt + 1
		})
		repairID, subErr := o.SubmitTask(domain.Task{
			ProjectPath:         req.ProjectPath,
			Instruction:         fmt.Sprintf("Fix the failing project verification command %q.\n\nFailure output:\n%s", verifyCmd, truncateObservation(cleanOut)),
			Role:                "coder",
			Agentic:             true,
			MaxAgentTurns:       req.MaxAgentTurns,
			VerificationCommand: verifyCmd,
			Tags:                []string{"auto-mode", "repair"},
		})
		if subErr != nil {
			return fmt.Sprintf("automode: submit repair task: %v", subErr)
		}
		state.update(func(r *domain.AutoModeRun) {
			r.TaskIDs = append(r.TaskIDs, repairID)
		})
		_, _, stopped := o.autoWaitTasks(ctx, state, state.snapshot().TaskIDs)
		if stopped {
			return "stopped"
		}
	}
}

// autoDigestLearnings feeds settled tasks into the project brain for continuous memory.
func (o *OrchestratorService) autoDigestLearnings(ctx context.Context, taskIDs []string) int {
	o.mu.Lock()
	brain := o.brain
	o.mu.Unlock()
	if brain == nil {
		return 0
	}
	saved := 0
	for _, id := range taskIDs {
		t, err := o.GetTask(id)
		if err != nil {
			continue
		}
		saved += len(DigestTaskOutcome(ctx, brain, t, t.Status))
	}
	return saved
}

// Compile-time check that OrchestratorService implements ports.AutoModeManager.
var _ ports.AutoModeManager = (*OrchestratorService)(nil)
