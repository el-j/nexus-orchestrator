package repo_sqlite

import (
	"strings"
	"unicode"
)

// stopWords contains common English words filtered from multi-word queries to improve BM25 relevance.
var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "by": true, "can": true, "did": true, "do": true, "does": true,
	"for": true, "from": true, "how": true, "i": true, "in": true, "is": true,
	"it": true, "my": true, "of": true, "on": true, "or": true, "our": true,
	"should": true, "that": true, "the": true, "they": true, "this": true,
	"to": true, "was": true, "we": true, "what": true, "when": true,
	"where": true, "which": true, "who": true, "will": true, "with": true,
	"you": true,
}

// tokenizeFTS splits a query into alphanumeric and underscore word tokens in lowercase.
func tokenizeFTS(raw string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			current.WriteRune(unicode.ToLower(r))
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// buildFTSQuery turns a natural-language query into a safe, ranked FTS5 MATCH expression.
// If the input produces no alphanumeric tokens, it returns an empty string.
func buildFTSQuery(raw string) string {
	tokens := tokenizeFTS(raw)
	if len(tokens) == 0 {
		return ""
	}

	var significant []string
	for _, t := range tokens {
		if !stopWords[t] {
			significant = append(significant, t)
		}
	}
	if len(significant) == 0 {
		significant = tokens
	}

	if len(significant) == 1 {
		return `"` + significant[0] + `"*`
	}

	var andParts, orParts []string
	for _, t := range significant {
		quoted := `"` + t + `"*`
		andParts = append(andParts, quoted)
		orParts = append(orParts, quoted)
	}

	andClause := "(" + strings.Join(andParts, " AND ") + ")"
	orClause := "(" + strings.Join(orParts, " OR ") + ")"
	return andClause + " OR " + orClause
}
