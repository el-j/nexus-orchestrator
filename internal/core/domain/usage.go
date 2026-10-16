package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Usage is what one call to a model consumed, as the provider reported it.
type Usage struct {
	// Model is the model that answered, when the provider says.
	Model string `json:"model,omitempty"`
	// InputTokens and OutputTokens are the plain token counts of the call.
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	// CachedTokens are input tokens read from or written to the provider's prompt cache.
	CachedTokens int `json:"cachedTokens,omitempty"`
	// CostUSD is the list-price cost the provider reported; zero when it does not say.
	CostUSD float64 `json:"costUSD,omitempty"`
}

// QuotaReading is how much of a plan's allowance is used, read from the provider's own client.
type QuotaReading struct {
	// Window names the allowance, for example "five_hour" or "seven_day".
	Window string `json:"window"`
	// Status is the client's verdict: "allowed", "allowed_warning" or "rejected".
	Status string `json:"status"`
	// Utilization is the used share of the window, 0 to 1; negative when not reported.
	Utilization float64 `json:"utilization"`
	// ResetsAt is when the window resets; zero when not reported.
	ResetsAt time.Time `json:"resetsAt,omitzero"`
}

// Rejected reports whether the client refused the call because the allowance is used up.
func (q QuotaReading) Rejected() bool { return q.Status == "rejected" }

// UsageRecord is one model call as stored in the usage ledger.
type UsageRecord struct {
	// At is when the call finished.
	At time.Time `json:"at"`
	// TaskID is the task the call belonged to; empty for calls outside a task.
	TaskID string `json:"taskId,omitempty"`
	// Provider is the provider's name and Role the task's role (may be empty).
	Provider string `json:"provider"`
	Role     string `json:"role,omitempty"`
	// Usage is what the provider reported.
	Usage
}

// UsageSummary adds up the ledger for one provider, model and role.
type UsageSummary struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model,omitempty"`
	Role         string  `json:"role,omitempty"`
	Calls        int     `json:"calls"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CachedTokens int     `json:"cachedTokens,omitempty"`
	CostUSD      float64 `json:"costUSD,omitempty"`
}

// UsageTotals adds a list of summaries up.
func UsageTotals(sums []UsageSummary) UsageSummary {
	var t UsageSummary
	for _, s := range sums {
		t.Calls += s.Calls
		t.InputTokens += s.InputTokens
		t.OutputTokens += s.OutputTokens
		t.CachedTokens += s.CachedTokens
		t.CostUSD += s.CostUSD
	}
	return t
}

// ParseSince reads the start of a usage window: a duration back from now ("90m", "24h", "7d"), an
// RFC 3339 time, or "all" (the zero time). An empty value means the last 24 hours.
func ParseSince(v string, now time.Time) (time.Time, error) {
	v = strings.TrimSpace(v)
	switch {
	case v == "":
		return now.Add(-24 * time.Hour), nil
	case strings.EqualFold(v, "all"):
		return time.Time{}, nil
	}
	if days, ok := strings.CutSuffix(v, "d"); ok {
		if n, err := strconv.Atoi(days); err == nil && n > 0 {
			return now.Add(-time.Duration(n) * 24 * time.Hour), nil
		}
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return now.Add(-d), nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid since %q: use a duration like 24h or 7d, an RFC 3339 time, or all", v)
}
