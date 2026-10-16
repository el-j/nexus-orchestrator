package llm_cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// The Claude driver. A command line that runs `claude` with `--output-format stream-json` gets more
// than the text: the tokens and cost of the call, and the plan's own reading of how much of the
// allowance is used (`rate_limit_event`: window, utilization, reset time). It also gets true streaming
// and typed limit errors. Every other command line keeps the plain text-pipe behaviour.

// claudeLeanFlags make a call to the claude client cheap: it is used as a completion engine (Nexus runs
// its own tool loop), so no tools, no saved session, no slash commands, no settings (hooks, plugins,
// memory files and skills of the user's setup) and no MCP servers are loaded into its prompt. Measured
// with claude 2.1.301: a one-word call read about 15 500 input tokens with a typical setup and about
// 3 700 with these flags. `--verbose` and `--include-partial-messages` are what stream-json needs to
// stream text. The login is not a setting, so the subscription still applies.
var claudeLeanFlags = [][]string{
	{"--include-partial-messages"},
	{"--verbose"},
	{"--tools", ""},
	{"--no-session-persistence"},
	{"--disable-slash-commands"},
	{"--setting-sources", ""},
	{"--strict-mcp-config"},
}

// childEnvDrop are removed from the environment of claude: with an API key set it bills the API, not
// the subscription the user means to use.
var childEnvDrop = []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"}

// isClaudeStream reports whether the command is claude in stream-json mode.
func isClaudeStream(exe string, args []string) bool {
	if strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe") != "claude" {
		return false
	}
	for i, a := range args {
		if a == "--output-format=stream-json" || (a == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json") {
			return true
		}
	}
	return false
}

// withLeanFlags appends the flags of claudeLeanFlags the command does not already have.
func withLeanFlags(args []string) []string {
	out := slices.Clone(args)
	for _, f := range claudeLeanFlags {
		if !slices.Contains(out, f[0]) {
			out = append(out, f...)
		}
	}
	return out
}

// withoutEnv returns env without the named variables.
func withoutEnv(env []string, names ...string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !slices.Contains(names, name) {
			out = append(out, kv)
		}
	}
	return out
}

// claudeOutcome is what one stream-json run said.
type claudeOutcome struct {
	text      string // the streamed text
	result    string // the final result text
	isError   bool
	apiStatus int
	usage     domain.Usage
	hasUsage  bool
	quota     domain.QuotaReading
	hasQuota  bool
}

// parseClaudeStream reads stream-json lines, hands text deltas to onChunk and collects the rest. A line
// that is not JSON is taken as plain text, so an older client or a changed format still gives an answer.
func parseClaudeStream(r io.Reader, onChunk func(string)) (claudeOutcome, error) {
	var o claudeOutcome
	var text strings.Builder
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, rerr := br.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			var ev claudeEvent
			if err := json.Unmarshal(trimmed, &ev); err != nil || ev.Type == "" {
				plain := string(trimmed)
				text.WriteString(plain)
				onChunk(plain)
			} else {
				o.apply(ev, &text, onChunk)
			}
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				return o, rerr
			}
			break
		}
	}
	o.text = text.String()
	return o, nil
}

// claudeEvent is the part of a stream-json line the driver reads.
type claudeEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Model   string `json:"model"`
	Event   struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	RateLimit struct {
		Status      string   `json:"status"`
		ResetsAt    float64  `json:"resetsAt"`
		Window      string   `json:"rateLimitType"`
		Utilization *float64 `json:"utilization"`
	} `json:"rate_limit_info"`
	IsError   bool    `json:"is_error"`
	Result    string  `json:"result"`
	APIStatus int     `json:"api_error_status"`
	Cost      float64 `json:"total_cost_usd"`
	Usage     struct {
		Input        int `json:"input_tokens"`
		Output       int `json:"output_tokens"`
		CacheCreated int `json:"cache_creation_input_tokens"`
		CacheRead    int `json:"cache_read_input_tokens"`
	} `json:"usage"`
	ModelUsage map[string]json.RawMessage `json:"modelUsage"`
}

func (o *claudeOutcome) apply(ev claudeEvent, text *strings.Builder, onChunk func(string)) {
	switch ev.Type {
	case "system":
		if ev.Subtype == "init" && ev.Model != "" {
			o.usage.Model = ev.Model
		}
	case "stream_event":
		if ev.Event.Type == "content_block_delta" && ev.Event.Delta.Type == "text_delta" && ev.Event.Delta.Text != "" {
			text.WriteString(ev.Event.Delta.Text)
			onChunk(ev.Event.Delta.Text)
		}
	case "rate_limit_event":
		q := domain.QuotaReading{Window: ev.RateLimit.Window, Status: ev.RateLimit.Status, Utilization: -1}
		if ev.RateLimit.Utilization != nil {
			q.Utilization = *ev.RateLimit.Utilization
		}
		if ev.RateLimit.ResetsAt > 0 {
			q.ResetsAt = time.Unix(int64(ev.RateLimit.ResetsAt), 0).UTC()
		}
		o.quota, o.hasQuota = q, true
	case "result":
		o.result, o.isError, o.apiStatus = strings.TrimSpace(ev.Result), ev.IsError, ev.APIStatus
		o.usage.InputTokens, o.usage.OutputTokens = ev.Usage.Input, ev.Usage.Output
		o.usage.CachedTokens = ev.Usage.CacheCreated + ev.Usage.CacheRead
		o.usage.CostUSD = ev.Cost
		for name := range ev.ModelUsage { // one model per call; the init event named it already
			if o.usage.Model == "" {
				o.usage.Model = name
			}
		}
		o.hasUsage = true
	}
}

// failure turns an unsuccessful outcome into the error routing understands, or nil.
func (o claudeOutcome) failure(provider string, now time.Time) error {
	if o.hasQuota && o.quota.Rejected() {
		kind := domain.ErrKindQuotaExhausted
		if o.quota.Window == "five_hour" {
			kind = domain.ErrKindRateLimited
		}
		return &domain.ProviderError{
			Provider: provider, Kind: kind, Status: 429, ResetAt: o.quota.ResetsAt,
			Msg: fmt.Sprintf("%s: %s limit reached, resets %s", provider, o.quota.Window, o.quota.ResetsAt.Local().Format("Mon 15:04")),
		}
	}
	if !o.isError {
		return nil
	}
	if pe := domain.ClassifyHTTP(provider, o.apiStatus, "", o.result, now); pe != nil {
		return pe
	}
	return fmt.Errorf("%s: %s", provider, firstNonEmpty(o.result, "the client reported an error"))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// runClaude runs claude in stream-json mode for one request.
func (a *Adapter) runClaude(ctx context.Context, prompt string, onChunk func(string)) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.exe, withLeanFlags(a.args)...)
	cmd.Dir = os.TempDir()
	cmd.Env = withoutEnv(os.Environ(), childEnvDrop...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.WaitDelay = 2 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, left: 8192}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("%s: %w", a.name, err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: start %s: %w", a.name, a.exe, err)
	}
	o, perr := parseClaudeStream(out, onChunk)
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return o.text, fmt.Errorf("%s: %w", a.name, ctx.Err())
	}
	a.remember(o)
	if ferr := o.failure(a.name, time.Now()); ferr != nil {
		return "", ferr
	}
	if perr == nil {
		perr = waitErr
	}
	if perr != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return o.text, fmt.Errorf("%s: %w: %s", a.name, perr, msg)
		}
		return o.text, fmt.Errorf("%s: %w", a.name, perr)
	}
	reply := firstNonEmpty(o.result, strings.TrimSpace(o.text))
	if reply == "" {
		return "", fmt.Errorf("%s: the command printed nothing", a.name)
	}
	return reply, nil
}

// remember keeps the usage and quota the last call reported.
func (a *Adapter) remember(o claudeOutcome) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if o.hasUsage {
		a.lastUsage, a.hasUsage = o.usage, true
	}
	if o.hasQuota {
		a.lastQuota, a.hasQuota = o.quota, true
	}
}

// LastUsage returns the tokens and cost the last call reported; ok is false when the command line
// does not report them (anything but claude in stream-json mode) or nothing was called yet.
func (a *Adapter) LastUsage() (domain.Usage, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastUsage, a.hasUsage
}

// LastQuota returns the plan's own reading of its allowance from the last call; ok is false when none
// was reported.
func (a *Adapter) LastQuota() (domain.QuotaReading, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastQuota, a.hasQuota
}
