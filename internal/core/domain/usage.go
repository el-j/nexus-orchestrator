package domain

import "time"

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
