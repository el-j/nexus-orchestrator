// Package docscheck holds tests, and only tests, that keep documentation honest.
//
// Prose drifts from code: install instructions name release files that do not
// exist, links keep an old repository name, hand-written counts go stale. These
// tests compare documents against the sources of truth (the release workflow, the
// tool catalog in internal/adapters/inbound/mcp) so a mismatch fails CI instead of
// reaching a reader.
package docscheck
