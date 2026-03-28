package mcp

// SseKeepaliveInterval is a pointer to the package-level keepalive interval.
// Tests may override it to speed up keepalive-related assertions.
var SseKeepaliveInterval = &sseKeepaliveInterval
