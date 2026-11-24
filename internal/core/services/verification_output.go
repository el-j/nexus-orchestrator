package services

import "regexp"

var terminalEscapes = regexp.MustCompile("\x1b(?:\\[[0-9;?]*[\x20-\x2f]*[\x40-\x7e]|\\([A-Za-z0-9])")

// stripTerminalEscapes removes colour and cursor codes from the output of a verification command; they
// are noise in the task log and in the prompt of the next correction turn.
func stripTerminalEscapes(s string) string { return terminalEscapes.ReplaceAllString(s, "") }
