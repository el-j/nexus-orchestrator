package mcp

// SseKeepaliveInterval is a pointer to the package-level keepalive interval.
// Tests may override it to speed up keepalive-related assertions.
var SseKeepaliveInterval = &sseKeepaliveInterval

// WaitPollInterval is a pointer to the interval at which wait_for_task re-reads the task, so
// tests can shorten it.
var WaitPollInterval = &waitPollInterval

// ShutdownGrace is a pointer to the time in-flight requests get on shutdown, so tests can shorten it.
var ShutdownGrace = &shutdownGrace

// JSONResult exposes the shared JSON answer of the tools to the external tests.
var JSONResult = jsonResult
