package services

// ProcessNext runs exactly one iteration of the worker loop (claim the next
// QUEUED task and execute it) and reports whether a task was claimed. It exists
// so tests can drive the execution engine deterministically with
// WithDisableBackgroundWorkers instead of polling a background goroutine.
func (o *OrchestratorService) ProcessNext() bool { return o.processNext() }
