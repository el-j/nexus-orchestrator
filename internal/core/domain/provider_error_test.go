package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestClassifyHTTP(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		status     int
		retryAfter string
		body       string
		want       ProviderErrorKind
		msg        string
		until      time.Time
	}{
		{"429 plain", 429, "", `{}`, ErrKindRateLimited, "p: rate limited (429)", time.Time{}},
		{"429 seconds", 429, "30", `{}`, ErrKindRateLimited, "p: rate limited (429)", now.Add(30 * time.Second)},
		{"429 http date", 429, "Fri, 16 Oct 2026 12:05:00 GMT", ``, ErrKindRateLimited, "p: rate limited (429)", now.Add(5 * time.Minute)},
		{"429 junk header", 429, "soon", ``, ErrKindRateLimited, "p: rate limited (429)", time.Time{}},
		{"429 negative", 429, "-5", ``, ErrKindRateLimited, "p: rate limited (429)", time.Time{}},
		{"429 quota body", 429, "", `{"error":{"status":"RESOURCE_EXHAUSTED","message":"You exceeded your current quota"}}`, ErrKindQuotaExhausted, "p: rate limited (429): quota exhausted", time.Time{}},
		{"402", 402, "", ``, ErrKindQuotaExhausted, "p: quota exhausted (402)", time.Time{}},
		{"529", 529, "10", ``, ErrKindOverloaded, "p: overloaded (529)", now.Add(10 * time.Second)},
		{"503", 503, "", ``, ErrKindOverloaded, "p: overloaded (503)", time.Time{}},
		{"401", 401, "", ``, ErrKindAuth, "p: not authorised (401)", time.Time{}},
		{"403", 403, "", ``, ErrKindAuth, "p: not authorised (403)", time.Time{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pe := ClassifyHTTP("p", c.status, c.retryAfter, c.body, now)
			if pe == nil {
				t.Fatal("nil")
			}
			if pe.Kind != c.want || pe.Error() != c.msg || !pe.ResetAt.Equal(c.until) || pe.Status != c.status || pe.Provider != "p" {
				t.Errorf("got %+v msg %q", pe, pe.Error())
			}
		})
	}
	for _, status := range []int{200, 400, 404, 500, 502} {
		if ClassifyHTTP("p", status, "", "", now) != nil {
			t.Errorf("status %d must not be classified", status)
		}
	}
}

func TestProviderErrorHelpers(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	pe := &ProviderError{Provider: "x", Kind: ErrKindRateLimited, Status: 429}
	wrapped := fmt.Errorf("task: %w", pe)
	got, ok := AsProviderError(wrapped)
	if !ok || got != pe {
		t.Fatal("AsProviderError must see through wrapping")
	}
	if _, ok := AsProviderError(errors.New("plain")); ok {
		t.Error("plain error is not a ProviderError")
	}
	if pe.Unwrap() != nil {
		t.Error("no cause: Unwrap is nil")
	}
	cause := errors.New("boom")
	if !errors.Is((&ProviderError{Err: cause}).Unwrap(), cause) {
		t.Error("Unwrap must return the cause")
	}
	// CooldownUntil: the reset time when known, otherwise a default per kind.
	if until := (&ProviderError{Kind: ErrKindRateLimited, ResetAt: now.Add(time.Minute)}).CooldownUntil(now); !until.Equal(now.Add(time.Minute)) {
		t.Errorf("known reset: %v", until)
	}
	for kind, d := range map[ProviderErrorKind]time.Duration{
		ErrKindRateLimited: 60 * time.Second, ErrKindOverloaded: 30 * time.Second, ErrKindQuotaExhausted: time.Hour,
	} {
		if until := (&ProviderError{Kind: kind}).CooldownUntil(now); !until.Equal(now.Add(d)) {
			t.Errorf("%s default: %v", kind, until)
		}
	}
	for _, kind := range []ProviderErrorKind{ErrKindAuth, ErrKindNetwork} {
		if !(&ProviderError{Kind: kind}).CooldownUntil(now).IsZero() {
			t.Errorf("%s must not cool down by itself", kind)
		}
	}
	// A reset time in the past is ignored.
	if until := (&ProviderError{Kind: ErrKindRateLimited, ResetAt: now.Add(-time.Minute)}).CooldownUntil(now); !until.Equal(now.Add(60 * time.Second)) {
		t.Errorf("past reset: %v", until)
	}
}
