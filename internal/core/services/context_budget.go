package services

import (
	"bufio"
	"fmt"
	"sort"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// outlineHeadLines is how many leading lines (package, imports, headers) an outline keeps.
const outlineHeadLines = 12

// tokensOf approximates the tokens of s the way estimateTokens does: four characters per token.
func tokensOf(s string) int { return (len(s) + 3) / 4 }

// readContext returns the context files of a task for the prompt. Without a budget that is what the
// file writer returns. With one, files are read one by one and, if together they exceed the budget,
// the largest are sent as outlines (their top-level declarations) and then shortened until they fit,
// each marked so the model knows it does not see the whole file.
func (o *OrchestratorService) readContext(task domain.Task, budget int) (string, error) {
	if budget <= 0 {
		return o.fileWriter.ReadContextFiles(task.ProjectPath, task.ContextFiles)
	}
	blocks := make([]string, len(task.ContextFiles))
	total := 0
	for i, f := range task.ContextFiles {
		b, err := o.fileWriter.ReadContextFiles(task.ProjectPath, []string{f})
		if err != nil {
			return "", err
		}
		blocks[i] = b
		total += tokensOf(b)
	}
	if total <= budget {
		return strings.Join(blocks, ""), nil
	}
	order := make([]int, len(blocks))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return len(blocks[order[a]]) > len(blocks[order[b]]) })
	for _, i := range order { // first the outlines, largest file first
		if total <= budget {
			break
		}
		if short := outlineBlock(blocks[i]); len(short) < len(blocks[i]) {
			total += tokensOf(short) - tokensOf(blocks[i])
			blocks[i] = short
		}
	}
	for _, i := range order { // then shorten what is still too much, largest first
		if total <= budget {
			break
		}
		keep := max(tokensOf(blocks[i])-(total-budget), 20)
		short := shortenBlock(blocks[i], keep)
		total += tokensOf(short) - tokensOf(blocks[i])
		blocks[i] = short
	}
	return strings.Join(blocks, ""), nil
}

// splitBlock separates the header line the file writer puts before a file from its content.
func splitBlock(block string) (header, body string) {
	header, body, _ = strings.Cut(block, "\n")
	return header, body
}

// withNote adds a note in front of the closing " ---" of a header such as "// --- a.go ---".
func withNote(header, note string) string {
	if h, ok := strings.CutSuffix(header, " ---"); ok {
		return h + " (" + note + ") ---"
	}
	return header + " (" + note + ")"
}

// isDeclaration reports whether a line of source starts a top-level or method declaration in the common
// languages: an unindented line that is not only a closing bracket or comment, or an indented method.
func isDeclaration(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "}") || strings.HasPrefix(t, ")") || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") {
		return false
	}
	if line[0] != ' ' && line[0] != '\t' {
		return true
	}
	for _, kw := range []string{"func ", "def ", "public ", "private ", "protected ", "async ", "static ", "fn "} {
		if strings.HasPrefix(t, kw) {
			return true
		}
	}
	return false
}

// outlineBlock replaces a file's content by its first lines and its declarations.
func outlineBlock(block string) string {
	header, body := splitBlock(block)
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	var kept []string
	for i, l := range lines {
		if i < outlineHeadLines || isDeclaration(l) {
			kept = append(kept, l)
		}
	}
	note := fmt.Sprintf("outline: %d of %d lines; bodies left out", len(kept), len(lines))
	return withNote(header, note) + "\n" + strings.Join(kept, "\n") + "\n"
}

// shortenBlock keeps the start of a block within about keepTokens tokens.
func shortenBlock(block string, keepTokens int) string {
	header, body := splitBlock(block)
	limit := keepTokens * 4
	if limit >= len(body) {
		return block
	}
	cut := strings.LastIndex(body[:limit], "\n")
	if cut < 0 {
		cut = limit
	}
	return withNote(header, "shortened: the rest of the file is left out") + "\n" + body[:cut] + "\n"
}

// trimHistoryTokens keeps the newest messages that fit in budget tokens, starting with a user turn.
// When older messages are pruned to satisfy the budget, it distills them into a rolling memory block.
func trimHistoryTokens(msgs []domain.Message, budget int) []domain.Message {
	if len(msgs) == 0 || budget <= 0 {
		return nil
	}
	used, start := 0, len(msgs)
	for i := len(msgs) - 1; i >= 0; i-- {
		t := tokensOf(msgs[i].Content) + 4
		if used+t > budget {
			break
		}
		used += t
		start = i
	}
	kept := msgs[start:]
	for len(kept) > 0 && kept[0].Role != domain.RoleUser {
		used -= tokensOf(kept[0].Content) + 4
		kept = kept[1:]
	}
	if start == 0 {
		return kept
	}

	pruned := msgs[:start]
	summary := distillPrunedHistory(pruned)
	if summary == "" {
		return kept
	}
	summaryBlock := "[Previous Context Summary: " + summary + "]"
	summaryTokens := tokensOf(summaryBlock) + 2

	if len(kept) > 0 {
		if used+summaryTokens <= budget {
			cloned := make([]domain.Message, len(kept))
			copy(cloned, kept)
			cloned[0].Content = summaryBlock + "\n\n" + cloned[0].Content
			return cloned
		}
		// Try pruning an older user-assistant pair if multiple pairs exist to fit the rolling memory
		temp := kept
		tempUsed := used
		for len(temp) > 2 && tempUsed+summaryTokens > budget {
			tempUsed -= tokensOf(temp[0].Content) + 4
			temp = temp[1:]
			for len(temp) > 0 && temp[0].Role != domain.RoleUser {
				tempUsed -= tokensOf(temp[0].Content) + 4
				temp = temp[1:]
			}
		}
		if len(temp) > 0 && temp[0].Role == domain.RoleUser && tempUsed+summaryTokens <= budget {
			cloned := make([]domain.Message, len(temp))
			copy(cloned, temp)
			cloned[0].Content = summaryBlock + "\n\n" + cloned[0].Content
			return cloned
		}
		return kept
	}

	if summaryTokens <= budget {
		return []domain.Message{
			{
				Role:    domain.RoleUser,
				Content: summaryBlock,
			},
		}
	}
	return nil
}

// distillPrunedHistory condenses pruned messages into a rolling memory string.
func distillPrunedHistory(pruned []domain.Message) string {
	var parts []string
	for _, m := range pruned {
		text := strings.TrimSpace(m.Content)
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "[Previous Context Summary: ") {
			end := strings.Index(text, "]")
			if end > len("[Previous Context Summary: ") {
				prev := text[len("[Previous Context Summary: "):end]
				parts = append(parts, prev)
				text = strings.TrimSpace(text[end+1:])
			}
		}
		if text == "" {
			continue
		}
		snippet := firstNonEmptyLine(text, 60)
		switch m.Role {
		case domain.RoleUser:
			parts = append(parts, "User: "+snippet)
		case domain.RoleAssistant:
			parts = append(parts, "Assistant: "+snippet)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 4 {
		parts = parts[len(parts)-4:]
	}
	return strings.Join(parts, "; ")
}

// firstNonEmptyLine returns the first non-empty line truncated to maxLen characters.
func firstNonEmptyLine(s string, maxLen int) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if len(line) > maxLen {
				return line[:maxLen] + "..."
			}
			return line
		}
	}
	return ""
}
