package repo_sqlite

import (
	"testing"
)

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "punctuation only",
			input:    "??? !!! -- :()",
			expected: "",
		},
		{
			name:     "single word",
			input:    "interface",
			expected: `"interface"*`,
		},
		{
			name:     "single word with quotes",
			input:    `"interface"`,
			expected: `"interface"*`,
		},
		{
			name:     "multi word natural language",
			input:    "how do we handle rate limits",
			expected: `("handle"* AND "rate"* AND "limits"*) OR ("handle"* OR "rate"* OR "limits"*)`,
		},
		{
			name:     "only stop words",
			input:    "what is this",
			expected: `("what"* AND "is"* AND "this"*) OR ("what"* OR "is"* OR "this"*)`,
		},
		{
			name:     "underscores and numbers preserved",
			input:    "sqlite3_fts5 v2",
			expected: `("sqlite3_fts5"* AND "v2"*) OR ("sqlite3_fts5"* OR "v2"*)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFTSQuery(tt.input)
			if got != tt.expected {
				t.Errorf("buildFTSQuery(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
