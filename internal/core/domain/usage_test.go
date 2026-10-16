package domain

import (
	"testing"
	"time"
)

func TestQuotaReadingRejected(t *testing.T) {
	if !(QuotaReading{Status: "rejected"}).Rejected() {
		t.Error("rejected")
	}
	for _, s := range []string{"allowed", "allowed_warning", ""} {
		if (QuotaReading{Status: s}).Rejected() {
			t.Errorf("%q is not rejected", s)
		}
	}
}

func TestUsageTotals(t *testing.T) {
	got := UsageTotals([]UsageSummary{
		{Calls: 1, InputTokens: 10, OutputTokens: 2, CachedTokens: 3, CostUSD: 0.5},
		{Calls: 2, InputTokens: 5, OutputTokens: 1, CostUSD: 0.25},
	})
	if got.Calls != 3 || got.InputTokens != 15 || got.OutputTokens != 3 || got.CachedTokens != 3 || got.CostUSD != 0.75 {
		t.Errorf("%+v", got)
	}
	if UsageTotals(nil).Calls != 0 {
		t.Error("empty")
	}
}

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	ok := map[string]time.Time{
		"":                     now.Add(-24 * time.Hour),
		"  ":                   now.Add(-24 * time.Hour),
		"90m":                  now.Add(-90 * time.Minute),
		"36h":                  now.Add(-36 * time.Hour),
		"7d":                   now.Add(-7 * 24 * time.Hour),
		"all":                  {},
		"ALL":                  {},
		"2026-10-01T00:00:00Z": time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
	for in, want := range ok {
		got, err := ParseSince(in, now)
		if err != nil || !got.Equal(want) {
			t.Errorf("%q: %v %v, want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"yesterday", "0d", "-5h", "d", "0h", "2026-10-01"} {
		if _, err := ParseSince(bad, now); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}
