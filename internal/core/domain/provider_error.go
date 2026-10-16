package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ProviderErrorKind says why a provider call failed, in the terms routing needs: whether to try the
// next provider, and whether to leave this one alone for a while.
type ProviderErrorKind string

// The kinds of provider failure the engine distinguishes.
const (
	// ErrKindRateLimited: too many requests for now; retrying later works.
	ErrKindRateLimited ProviderErrorKind = "rate_limited"
	// ErrKindQuotaExhausted: the plan's allowance is used up until it resets.
	ErrKindQuotaExhausted ProviderErrorKind = "quota_exhausted"
	// ErrKindOverloaded: the provider is busy or down for a moment.
	ErrKindOverloaded ProviderErrorKind = "overloaded"
	// ErrKindAuth: the login or key is not accepted; waiting does not help.
	ErrKindAuth ProviderErrorKind = "auth"
	// ErrKindNetwork: the provider could not be reached.
	ErrKindNetwork ProviderErrorKind = "network"
)

// Default cooldowns when the provider does not say when to come back.
const (
	defaultRateLimitCooldown = 60 * time.Second
	defaultOverloadCooldown  = 30 * time.Second
	defaultQuotaCooldown     = time.Hour
)

// ProviderError is a failed provider call with what routing needs to know about it. Error keeps the
// text the adapters always produced, so logs and existing checks do not change.
type ProviderError struct {
	// Provider is the provider's name.
	Provider string
	// Kind classifies the failure.
	Kind ProviderErrorKind
	// Status is the HTTP status, or 0 when there was none.
	Status int
	// ResetAt is when the provider said to come back (Retry-After or a reset header); zero when unknown.
	ResetAt time.Time
	// Msg is the error text.
	Msg string
	// Err is the underlying error, when there is one.
	Err error
}

// Error returns the message.
func (e *ProviderError) Error() string { return e.Msg }

// Unwrap returns the underlying error.
func (e *ProviderError) Unwrap() error { return e.Err }

// CooldownUntil returns when this provider should be tried again: the reset time the provider gave, or
// a default for the kind, or the zero time when waiting would not help (Auth, Network).
func (e *ProviderError) CooldownUntil(now time.Time) time.Time {
	if e.ResetAt.After(now) {
		return e.ResetAt
	}
	switch e.Kind {
	case ErrKindRateLimited:
		return now.Add(defaultRateLimitCooldown)
	case ErrKindOverloaded:
		return now.Add(defaultOverloadCooldown)
	case ErrKindQuotaExhausted:
		return now.Add(defaultQuotaCooldown)
	}
	return time.Time{}
}

// AsProviderError finds a ProviderError anywhere in err's chain.
func AsProviderError(err error) (*ProviderError, bool) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// ClassifyHTTP turns an HTTP failure into a ProviderError, or returns nil for a status that is not one
// of the kinds above (the caller reports those as before). retryAfter is the Retry-After header (seconds
// or an HTTP date); body is the start of the response body, used to tell an exhausted quota from a short
// rate limit.
func ClassifyHTTP(provider string, status int, retryAfter, body string, now time.Time) *ProviderError {
	pe := &ProviderError{Provider: provider, Status: status, ResetAt: parseRetryAfter(retryAfter, now)}
	switch {
	case status == 429 && looksLikeQuota(body):
		pe.Kind = ErrKindQuotaExhausted
		pe.Msg = fmt.Sprintf("%s: rate limited (429): quota exhausted", provider)
	case status == 429:
		pe.Kind = ErrKindRateLimited
		pe.Msg = fmt.Sprintf("%s: rate limited (429)", provider)
	case status == 402:
		pe.Kind = ErrKindQuotaExhausted
		pe.Msg = fmt.Sprintf("%s: quota exhausted (402)", provider)
	case status == 529 || status == 503:
		pe.Kind = ErrKindOverloaded
		pe.Msg = fmt.Sprintf("%s: overloaded (%d)", provider, status)
	case status == 401 || status == 403:
		pe.Kind = ErrKindAuth
		pe.Msg = fmt.Sprintf("%s: not authorised (%d)", provider, status)
	default:
		return nil
	}
	return pe
}

func looksLikeQuota(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "quota") || strings.Contains(b, "resource_exhausted") ||
		strings.Contains(b, "credit balance") || strings.Contains(b, "billing")
}

// parseRetryAfter reads a Retry-After value: whole seconds or an HTTP date. Junk and values in the
// past give the zero time.
func parseRetryAfter(v string, now time.Time) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return time.Time{}
		}
		return now.Add(time.Duration(secs) * time.Second)
	}
	if t, err := time.Parse(time.RFC1123, v); err == nil && t.After(now) {
		return t
	}
	return time.Time{}
}
