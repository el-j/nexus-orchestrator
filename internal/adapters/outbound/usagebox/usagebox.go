// Package usagebox holds the token usage the last call of an LLM adapter reported, so the adapter can
// satisfy ports.UsageReporter by embedding a Box.
package usagebox

import (
	"sync"

	"nexus-orchestrator/internal/core/domain"
)

// Box remembers the usage of the most recent call. The zero value is ready to use and every method is
// safe for concurrent use.
type Box struct {
	mu sync.Mutex
	u  domain.Usage
	ok bool
}

// Set records the usage of a finished call.
func (b *Box) Set(u domain.Usage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.u, b.ok = u, true
}

// LastUsage returns the usage of the most recent call; ok is false before the first call that
// reported any.
func (b *Box) LastUsage() (domain.Usage, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.u, b.ok
}
