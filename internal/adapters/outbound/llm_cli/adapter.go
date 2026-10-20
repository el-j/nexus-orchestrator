// Package llm_cli implements the LLMClient port on top of an AI command-line tool that is
// logged in on this machine, so a subscription (Claude Pro, for example) can be delegated to
// without an API key. The prompt goes to the tool's stdin and its stdout is the reply; it
// runs in an empty temporary directory so an agentic CLI cannot touch a project by accident.
package llm_cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"nexus-orchestrator/internal/core/domain"
)

// knownCLIs are the executables a provider config may name. A provider config can be written
// through the REST API and MCP, so it must not be a way to run arbitrary programs; add more
// with NEXUS_CLI_PROVIDERS (comma-separated executable names).
var knownCLIs = map[string]bool{"claude": true, "copilot": true, "aichat": true, "llm": true, "gemini": true, "codex": true}

// runTimeout bounds one invocation; a CLI that never answers must not hold a task forever.
const runTimeout = 10 * time.Minute

// Adapter runs one invocation of the CLI per request.
type Adapter struct {
	name    string
	command string // as configured, shown as the base URL
	exe     string
	args    []string
	model   string

	mu        sync.Mutex // guards the last reading below
	lastUsage domain.Usage
	hasUsage  bool
	lastQuota domain.QuotaReading
	hasQuota  bool
}

// NewAdapter builds an adapter for a command line such as "claude -p --model sonnet". The first
// word must be a known CLI (by file name, any directory). model is informational: pass it to the
// tool in the command line if the tool supports choosing one.
func NewAdapter(name, commandLine, model string) (*Adapter, error) {
	fields, err := splitCommand(commandLine)
	if err != nil {
		return nil, fmt.Errorf("cli provider %q: %w", name, err)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("cli provider %q: a command is required (for example \"claude -p\")", name)
	}
	if !allowed(filepath.Base(fields[0])) {
		return nil, fmt.Errorf("cli provider %q: %q is not a known AI CLI (claude, copilot, aichat, llm, gemini, codex; extend with NEXUS_CLI_PROVIDERS)", name, filepath.Base(fields[0]))
	}
	return &Adapter{name: name, command: commandLine, exe: fields[0], args: fields[1:], model: model}, nil
}

func allowed(exe string) bool {
	exe = strings.TrimSuffix(strings.ToLower(exe), ".exe")
	if knownCLIs[exe] {
		return true
	}
	for _, extra := range strings.Split(os.Getenv("NEXUS_CLI_PROVIDERS"), ",") {
		if strings.EqualFold(strings.TrimSpace(extra), exe) && exe != "" {
			return true
		}
	}
	return false
}

// splitCommand splits a command line into words, honouring single and double quotes.
func splitCommand(s string) ([]string, error) {
	var (
		out   []string
		cur   strings.Builder
		quote rune
		has   bool
	)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, has = r, true
		case r == ' ' || r == '\t':
			if has || cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote in the command")
	}
	if has || cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out, nil
}

// ProviderName returns the name shown in the UI and used for routing.
func (a *Adapter) ProviderName() string { return a.name }

// ActiveModel returns the configured model name (informational).
func (a *Adapter) ActiveModel() string { return a.model }

// BaseURL returns the command line this adapter runs.
func (a *Adapter) BaseURL() string { return a.command }

// ContextLimit is 200000 for claude, 128000 for copilot, and 0 (unknown) for other tools.
func (a *Adapter) ContextLimit(context.Context) int {
	switch strings.TrimSuffix(strings.ToLower(filepath.Base(a.exe)), ".exe") {
	case "claude":
		return 200000
	case "copilot":
		return 128000
	}
	return 0
}

// Ping reports whether the executable can be found.
func (a *Adapter) Ping(context.Context) bool {
	_, err := exec.LookPath(a.exe)
	return err == nil
}

// GetAvailableModels lists the aliases claude or copilot accepts, and "default" for other tools.
func (a *Adapter) GetAvailableModels(context.Context) ([]string, error) {
	switch strings.TrimSuffix(strings.ToLower(filepath.Base(a.exe)), ".exe") {
	case "claude":
		return []string{"sonnet", "opus", "haiku"}, nil
	case "copilot":
		return []string{"auto"}, nil // the CLI lists no models non-interactively; "auto" lets Copilot pick
	}
	return []string{"default"}, nil
}

// GenerateCode sends one prompt.
func (a *Adapter) GenerateCode(ctx context.Context, prompt string) (string, error) {
	return a.run(ctx, prompt, func(string) {})
}

// Chat sends the conversation as a transcript and returns the reply.
func (a *Adapter) Chat(ctx context.Context, messages []domain.Message) (string, error) {
	return a.run(ctx, transcript(messages), func(string) {})
}

// ChatStream is Chat that hands the tool's output to onChunk as it is written.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	return a.run(ctx, transcript(messages), onChunk)
}

// transcript renders a conversation as the single text prompt a CLI takes.
func transcript(messages []domain.Message) string {
	var b strings.Builder
	b.WriteString("Below is a conversation. Reply only with the assistant's next message.\n\n")
	for _, m := range messages {
		role := "User"
		switch m.Role {
		case domain.RoleSystem:
			role = "System"
		case domain.RoleAssistant:
			role = "Assistant"
		}
		fmt.Fprintf(&b, "%s: %s\n\n", role, m.Content)
	}
	b.WriteString("Assistant:")
	return b.String()
}

func (a *Adapter) run(ctx context.Context, prompt string, onChunk func(string)) (string, error) {
	if isClaudeStream(a.exe, a.args) {
		return a.runClaude(ctx, prompt, onChunk)
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	args := a.args
	if isCopilot(a.exe) {
		args = withCopilotFlags(args, a.model)
	}
	cmd := exec.CommandContext(ctx, a.exe, args...)
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(prompt)
	cmd.WaitDelay = 2 * time.Second // children that keep the pipe open must not hang Wait
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, left: 8192}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("%s: %w", a.name, err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: start %s: %w", a.name, a.exe, err)
	}

	var reply strings.Builder
	var carry []byte
	buf := make([]byte, 4096)
	for {
		n, rerr := out.Read(buf)
		carry = append(carry, buf[:n]...)
		cut := completePrefix(carry) // whole characters only; keep an incomplete tail for the next read
		if cut > 0 {
			piece := string(carry[:cut])
			reply.WriteString(piece)
			onChunk(piece)
			carry = append(carry[:0], carry[cut:]...)
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) && ctx.Err() == nil {
				err = rerr
			}
			break
		}
	}
	if len(carry) > 0 { // a stray byte sequence at the end: pass it on as it is
		reply.Write(carry)
		onChunk(string(carry))
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return reply.String(), fmt.Errorf("%s: %w", a.name, ctx.Err())
	}
	if err == nil {
		err = waitErr
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(reply.String())
		}
		if msg != "" {
			if isCopilot(a.exe) {
				if pe := classifyCLIFailure(a.name, fmt.Sprintf("%v: %s", err, msg)); pe != nil {
					return reply.String(), pe
				}
			}
			return reply.String(), fmt.Errorf("%s: %w: %s", a.name, err, msg)
		}
		return reply.String(), fmt.Errorf("%s: %w", a.name, err)
	}
	if strings.TrimSpace(reply.String()) == "" {
		return "", fmt.Errorf("%s: the command printed nothing", a.name)
	}
	return strings.TrimSpace(reply.String()), nil
}

// completePrefix returns how many leading bytes of b end on a character boundary: a multibyte
// character cut in half by a read belongs to the next piece.
func completePrefix(b []byte) int {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if utf8.FullRune(b[i:]) {
				return len(b)
			}
			return i
		}
	}
	return len(b)
}

// limitedWriter keeps the first left bytes written and drops the rest.
type limitedWriter struct {
	w    io.Writer
	left int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	n := len(p)
	if l.left > 0 {
		k := min(len(p), l.left)
		_, _ = l.w.Write(p[:k])
		l.left -= k
	}
	return n, nil
}
