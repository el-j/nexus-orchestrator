package domain_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// Every limit of a policy is tested on both sides: the value at the limit is accepted and the next one is
// refused. A comparison written with the wrong boundary then fails a test.
func TestProjectPolicyLimitsAreExact(t *testing.T) {
	budgets := func(n int) map[string]domain.ProviderBudget {
		m := map[string]domain.ProviderBudget{}
		for i := 0; i < n; i++ {
			m[fmt.Sprintf("p%d", i)] = domain.ProviderBudget{DailyTokens: 1}
		}
		return m
	}
	ladders := func(n int) map[string][]string {
		m := map[string][]string{}
		for i := 0; i < n; i++ {
			m[fmt.Sprintf("r%d", i)] = []string{"a/b"}
		}
		return m
	}
	rungs := func(n int) []string {
		var r []string
		for i := 0; i < n; i++ {
			r = append(r, fmt.Sprintf("a/m%d", i))
		}
		return r
	}
	folders := func(n int) []string {
		var f []string
		for i := 0; i < n; i++ {
			f = append(f, fmt.Sprintf("d%d", i))
		}
		return f
	}
	rule := func(n int) string { return strings.Repeat("x", n) }
	cases := []struct {
		name   string
		policy domain.ProjectPolicy
		ok     bool
	}{
		{"budgets at the limit", domain.ProjectPolicy{Budgets: budgets(domain.MaxBudgets)}, true},
		{"budgets over the limit", domain.ProjectPolicy{Budgets: budgets(domain.MaxBudgets + 1)}, false},
		{"daily tokens of 1", domain.ProjectPolicy{Budgets: map[string]domain.ProviderBudget{"p": {DailyTokens: 1}}}, true},
		{"daily tokens of 0", domain.ProjectPolicy{Budgets: map[string]domain.ProviderBudget{"p": {DailyTokens: 0}}}, false},
		{"daily tokens at the limit", domain.ProjectPolicy{Budgets: map[string]domain.ProviderBudget{"p": {DailyTokens: domain.MaxDailyTokens}}}, true},
		{"daily tokens over the limit", domain.ProjectPolicy{Budgets: map[string]domain.ProviderBudget{"p": {DailyTokens: domain.MaxDailyTokens + 1}}}, false},
		{"blank budget name", domain.ProjectPolicy{Budgets: map[string]domain.ProviderBudget{" ": {DailyTokens: 1}}}, false},
		{"ladder roles at the limit", domain.ProjectPolicy{Ladder: ladders(domain.MaxLadderRoles)}, true},
		{"ladder roles over the limit", domain.ProjectPolicy{Ladder: ladders(domain.MaxLadderRoles + 1)}, false},
		{"rungs at the limit", domain.ProjectPolicy{Ladder: map[string][]string{"r": rungs(domain.MaxLadderRungs)}}, true},
		{"rungs over the limit", domain.ProjectPolicy{Ladder: map[string][]string{"r": rungs(domain.MaxLadderRungs + 1)}}, false},
		{"one rung", domain.ProjectPolicy{Ladder: map[string][]string{"r": {"a/b"}}}, true},
		{"no rung", domain.ProjectPolicy{Ladder: map[string][]string{"r": {}}}, false},
		{"rung length at the limit", domain.ProjectPolicy{Ladder: map[string][]string{"r": {"a/" + strings.Repeat("m", domain.MaxRungLen-2)}}}, true},
		{"rung length over the limit", domain.ProjectPolicy{Ladder: map[string][]string{"r": {"a/" + strings.Repeat("m", domain.MaxRungLen-1)}}}, false},
		{"rung without a model", domain.ProjectPolicy{Ladder: map[string][]string{"r": {"a/ "}}}, false},
		{"rung without a provider", domain.ProjectPolicy{Ladder: map[string][]string{"r": {" /b"}}}, false},
		{"rung twice", domain.ProjectPolicy{Ladder: map[string][]string{"r": {"a/b", "A/B"}}}, false},
		{"blank ladder role", domain.ProjectPolicy{Ladder: map[string][]string{" ": {"a/b"}}}, false},
		{"context budget at the limit", domain.ProjectPolicy{ContextBudget: domain.MaxTokenBudget}, true},
		{"context budget over the limit", domain.ProjectPolicy{ContextBudget: domain.MaxTokenBudget + 1}, false},
		{"history budget at the limit", domain.ProjectPolicy{HistoryBudget: domain.MaxTokenBudget}, true},
		{"history budget over the limit", domain.ProjectPolicy{HistoryBudget: domain.MaxTokenBudget + 1}, false},
		{"negative context budget", domain.ProjectPolicy{ContextBudget: -1}, false},
		{"negative history budget", domain.ProjectPolicy{HistoryBudget: -1}, false},
		{"quota threshold of 1", domain.ProjectPolicy{QuotaThreshold: 1}, true},
		{"quota threshold above 1", domain.ProjectPolicy{QuotaThreshold: 1.01}, false},
		{"negative quota threshold", domain.ProjectPolicy{QuotaThreshold: -0.01}, false},
		{"verify at the limit", domain.ProjectPolicy{Verify: rule(domain.MaxPolicyVerifyLen)}, true},
		{"verify over the limit", domain.ProjectPolicy{Verify: rule(domain.MaxPolicyVerifyLen + 1)}, false},
		{"rules at the limit", domain.ProjectPolicy{Rules: strings.Split(strings.Repeat("r,", domain.MaxPolicyRules-1)+"r", ",")}, true},
		{"rules over the limit", domain.ProjectPolicy{Rules: strings.Split(strings.Repeat("r,", domain.MaxPolicyRules)+"r", ",")}, false},
		{"rule length at the limit", domain.ProjectPolicy{Rules: []string{rule(domain.MaxPolicyRuleLen)}}, true},
		{"rule length over the limit", domain.ProjectPolicy{Rules: []string{rule(domain.MaxPolicyRuleLen + 1)}}, false},
		{"blank rule", domain.ProjectPolicy{Rules: []string{"  "}}, false},
		{"plan folders at the limit", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: folders(domain.MaxPlanFolders)}}, true},
		{"plan folders over the limit", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: folders(domain.MaxPlanFolders + 1)}}, false},
		{"task folders over the limit", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Tasks: folders(domain.MaxPlanFolders + 1)}}, false},
		{"folder length at the limit", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{rule(256)}}}, true},
		{"folder length over the limit", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{rule(257)}}}, false},
		{"blank folder", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{" / "}}}, false},
		{"absolute folder", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{"/etc"}}}, false},
		{"backslash-absolute folder", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{"\\etc"}}}, false},
		{"drive folder", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{"C:/x"}}}, false},
		{"two-letter folder is not a drive", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{"ab"}}}, true},
		{"folder that leaves the project", domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: []string{"a/../.."}}}, false},
		{"unsupported version", domain.ProjectPolicy{Version: domain.PolicyVersion + 1}, false},
		{"negative maxFileLines", domain.ProjectPolicy{MaxFileLines: -1}, false},
		{"zero maxFileLines", domain.ProjectPolicy{MaxFileLines: 0}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.policy.Validate(); (err == nil) != c.ok {
				t.Errorf("Validate() = %v, want ok=%v", err, c.ok)
			}
		})
	}
}

// A ladder rung matches with * for any run of characters, and only when both sides are present.
func TestRungMatchesAndLadderLookup(t *testing.T) {
	cases := []struct {
		rung, provider, model string
		want                  bool
	}{
		{"a/b", "A", "B", true},
		{"a/*", "a", "anything", true},
		{"*/big", "x", "big", true},
		{"a/b*d", "a", "bcd", true},
		{"a/b*d", "a", "bd", true},
		{"a/b*b", "a", "b", false}, // the prefix and the suffix may not share a character
		{"a/b*c*d", "a", "bxcxd", true},
		{"a/b*c*d", "a", "bxd", false},
		{"nosplit", "nosplit", "", false},
		{"a/b", "a", "c", false},
	}
	for _, c := range cases {
		if got := domain.RungMatches(c.rung, c.provider, c.model); got != c.want {
			t.Errorf("RungMatches(%q, %q, %q) = %v, want %v", c.rung, c.provider, c.model, got, c.want)
		}
	}
	p := domain.ProjectPolicy{Ladder: map[string][]string{"default": {"d/1"}, "Coder": {"c/1"}}}
	if got := p.LadderFor(" CODER "); len(got) != 1 || got[0] != "c/1" {
		t.Errorf("LadderFor(coder) = %v", got)
	}
	if got := p.LadderFor("other"); len(got) != 1 || got[0] != "d/1" {
		t.Errorf("LadderFor(other) = %v, want the default ladder", got)
	}
	if got := p.LadderFor(""); len(got) != 1 || got[0] != "d/1" {
		t.Errorf("LadderFor(\"\") = %v, want the default ladder", got)
	}
	if got := (domain.ProjectPolicy{}).LadderFor("x"); got != nil {
		t.Errorf("no ladder must give nil, got %v", got)
	}
}

// The status codes of a failed call map to a kind, and a quota is told from a rate limit by the body.
func TestClassifyHTTPKindsAndCooldowns(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		status   int
		body     string
		kind     domain.ProviderErrorKind
		cooldown time.Duration
	}{
		{429, "slow down", domain.ErrKindRateLimited, 60 * time.Second},
		{429, "You exceeded your current QUOTA", domain.ErrKindQuotaExhausted, time.Hour},
		{429, "RESOURCE_EXHAUSTED", domain.ErrKindQuotaExhausted, time.Hour},
		{429, "Your credit balance is too low", domain.ErrKindQuotaExhausted, time.Hour},
		{429, "check your Billing details", domain.ErrKindQuotaExhausted, time.Hour},
		{402, "", domain.ErrKindQuotaExhausted, time.Hour},
		{503, "", domain.ErrKindOverloaded, 30 * time.Second},
		{529, "", domain.ErrKindOverloaded, 30 * time.Second},
		{401, "", domain.ErrKindAuth, 0},
		{403, "", domain.ErrKindAuth, 0},
	}
	for _, c := range cases {
		pe := domain.ClassifyHTTP("p", c.status, "", c.body, now)
		if pe == nil || pe.Kind != c.kind {
			t.Fatalf("status %d body %q: %+v, want kind %s", c.status, c.body, pe, c.kind)
		}
		got := pe.CooldownUntil(now)
		if c.cooldown == 0 && !got.IsZero() || c.cooldown > 0 && !got.Equal(now.Add(c.cooldown)) {
			t.Errorf("status %d: cooldown until %v, want %v after now", c.status, got, c.cooldown)
		}
	}
	for _, status := range []int{200, 400, 404, 500, 502} {
		if pe := domain.ClassifyHTTP("p", status, "", "", now); pe != nil {
			t.Errorf("status %d must not be classified, got %+v", status, pe)
		}
	}
}

// Retry-After counts in whole seconds or is an HTTP date; zero, negative and past values are ignored.
func TestRetryAfterBoundaries(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	at := func(v string) time.Time { return domain.ClassifyHTTP("p", 429, v, "", now).ResetAt }
	if got := at("1"); !got.Equal(now.Add(time.Second)) {
		t.Errorf("1 second: %v", got)
	}
	for _, v := range []string{"0", "-5", "", "junk"} {
		if got := at(v); !got.IsZero() {
			t.Errorf("Retry-After %q gives %v, want the zero time", v, got)
		}
	}
	future := now.Add(time.Hour).Format(time.RFC1123)
	if got := at(future); !got.Equal(now.Add(time.Hour)) {
		t.Errorf("date in the future: %v", got)
	}
	if got := at(now.Format(time.RFC1123)); !got.IsZero() {
		t.Errorf("a date that is now is not in the future: %v", got)
	}
	if got := at(now.Add(-time.Hour).Format(time.RFC1123)); !got.IsZero() {
		t.Errorf("a date in the past: %v", got)
	}
}

// A secret keeps its last four characters in the masked form only when there are more than four.
func TestMaskSecretBoundary(t *testing.T) {
	for in, want := range map[string]string{"": "", "abcd": "****", "abcde": "****bcde", "a": "****"} {
		if got := domain.MaskSecret(in); got != want {
			t.Errorf("MaskSecret(%q) = %q, want %q", in, got, want)
		}
	}
}

// The usage window: empty is a day, a number of days or a Go duration counts back, nothing else.
func TestParseSinceForms(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	for in, want := range map[string]time.Time{
		"":     now.Add(-24 * time.Hour),
		"1d":   now.Add(-24 * time.Hour),
		"7d":   now.Add(-7 * 24 * time.Hour),
		"90m":  now.Add(-90 * time.Minute),
		"all":  {},
		"ALL":  {},
		" 2h ": now.Add(-2 * time.Hour),
	} {
		got, err := domain.ParseSince(in, now)
		if err != nil || !got.Equal(want) {
			t.Errorf("ParseSince(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"0d", "-1d", "0h", "-3h", "soon"} {
		if _, err := domain.ParseSince(bad, now); err == nil {
			t.Errorf("ParseSince(%q) must fail", bad)
		}
	}
}
